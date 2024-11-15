package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// EffectiveMenuService computes the effective menu for a restaurant.
type EffectiveMenuService interface {
	// GetEffectiveMenu is the actor-scoped entry point (HTTP, humans).
	// Performs RBAC + scope checks before computing.
	GetEffectiveMenu(ctx context.Context, actor *domain.User, restaurantID uuid.UUID) (*domain.EffectiveMenu, error)

	// GetEffectiveMenuInternal is the service-scoped entry point (gRPC,
	// internal callers such as kfc-userapi). No RBAC — access control is
	// enforced by the transport layer (service token / mTLS).
	GetEffectiveMenuInternal(ctx context.Context, restaurantID uuid.UUID) (*domain.EffectiveMenu, error)
}

type effectiveMenuService struct {
	effectiveRepo  repository.EffectiveMenuRepository
	restaurantRepo repository.RestaurantRepository
	rbacService    RBACService
	logger         *logger.Logger
}

// NewEffectiveMenuService creates an EffectiveMenuService.
func NewEffectiveMenuService(
	effectiveRepo repository.EffectiveMenuRepository,
	restaurantRepo repository.RestaurantRepository,
	rbacService RBACService,
	log *logger.Logger,
) EffectiveMenuService {
	return &effectiveMenuService{
		effectiveRepo:  effectiveRepo,
		restaurantRepo: restaurantRepo,
		rbacService:    rbacService,
		logger:         log,
	}
}

// GetEffectiveMenu computes the menu for a restaurant.
//
// Algorithm:
//  1. Verify the actor can read the menu for this restaurant
//  2. Load categories, items, variations, overrides, and active promotions
//  3. Merge: master → apply overrides → apply promotions
//  4. Return the assembled tree
func (s *effectiveMenuService) GetEffectiveMenu(
	ctx context.Context,
	actor *domain.User,
	restaurantID uuid.UUID,
) (*domain.EffectiveMenu, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermMenuItemRead); err != nil {
		log.Warn("effective menu denied: permission",
			"actor_id", actor.ID,
			"error", err,
		)
		return nil, err
	}

	// 2. Scope — can the actor see this restaurant?
	if err := s.rbacService.CanAccessRestaurant(ctx, actor, restaurantID); err != nil {
		log.Warn("effective menu denied: scope",
			"actor_id", actor.ID,
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}

	return s.computeEffectiveMenu(ctx, restaurantID)
}

func (s *effectiveMenuService) GetEffectiveMenuInternal(
	ctx context.Context,
	restaurantID uuid.UUID,
) (*domain.EffectiveMenu, error) {
	return s.computeEffectiveMenu(ctx, restaurantID)
}

func (s *effectiveMenuService) computeEffectiveMenu(
	ctx context.Context,
	restaurantID uuid.UUID,
) (*domain.EffectiveMenu, error) {
	start := time.Now()
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("computing effective menu",
		"restaurant_id", restaurantID,
	)

	// 3. Load restaurant (need region for promotion scoping)
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		log.Error("effective menu: load restaurant",
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}

	// 4. Load all inputs
	categories, err := s.effectiveRepo.LoadCategories(ctx)
	if err != nil {
		log.Error("effective menu: load categories", "error", err)
		return nil, err
	}

	items, err := s.effectiveRepo.LoadMenuItems(ctx)
	if err != nil {
		log.Error("effective menu: load items", "error", err)
		return nil, err
	}

	itemIDs := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		itemIDs = append(itemIDs, it.ID)
	}

	variations, err := s.effectiveRepo.LoadVariationsByItemIDs(ctx, itemIDs)
	if err != nil {
		log.Error("effective menu: load variations", "error", err)
		return nil, err
	}

	overrides, err := s.effectiveRepo.LoadOverridesForRestaurant(ctx, restaurantID)
	if err != nil {
		log.Error("effective menu: load overrides", "error", err)
		return nil, err
	}

	promotions, err := s.effectiveRepo.LoadActivePromotions(ctx, restaurantID, restaurant.RegionID)
	if err != nil {
		log.Error("effective menu: load promotions", "error", err)
		return nil, err
	}

	log.Debug("effective menu inputs loaded",
		"categories", len(categories),
		"items", len(items),
		"variations", len(variations),
		"overrides", len(overrides),
		"promotions", len(promotions),
	)

	// 5. Index for fast lookup
	overrideByItem := make(map[uuid.UUID]*domain.RestaurantMenuItemOverride, len(overrides))
	for _, o := range overrides {
		overrideByItem[o.MenuItemID] = o
	}

	variationsByItem := make(map[uuid.UUID][]*domain.MenuItemVariation)
	for _, v := range variations {
		variationsByItem[v.MenuItemID] = append(variationsByItem[v.MenuItemID], v)
	}

	promotionByItem := buildPromotionIndex(promotions, items)

	// 6. Assemble
	effective := s.assembleEffectiveMenu(
		restaurantID,
		categories,
		items,
		variationsByItem,
		overrideByItem,
		promotionByItem,
	)

	log.Info("effective menu computed",
		"restaurant_id", restaurantID,
		"categories", len(effective.Categories),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return effective, nil
}

// ═══════════════════════════════════════════════════════════════════
// ASSEMBLY
// ═══════════════════════════════════════════════════════════════════

func (s *effectiveMenuService) assembleEffectiveMenu(
	restaurantID uuid.UUID,
	categories []*domain.MenuCategory,
	items []*domain.MenuItem,
	variationsByItem map[uuid.UUID][]*domain.MenuItemVariation,
	overridesByItem map[uuid.UUID]*domain.RestaurantMenuItemOverride,
	promotionByItem map[uuid.UUID]*domain.Promotion,
) *domain.EffectiveMenu {
	// Group items by category
	itemsByCategory := make(map[uuid.UUID][]*domain.MenuItem)
	for _, it := range items {
		itemsByCategory[it.CategoryID] = append(itemsByCategory[it.CategoryID], it)
	}

	effectiveCategories := make([]*domain.EffectiveCategory, 0, len(categories))

	for _, cat := range categories {
		catItems := itemsByCategory[cat.ID]
		if len(catItems) == 0 {
			continue
		}

		effectiveItems := make([]*domain.EffectiveMenuItem, 0, len(catItems))
		for _, it := range catItems {
			item := buildEffectiveItem(
				it,
				variationsByItem[it.ID],
				overridesByItem[it.ID],
				promotionByItem[it.ID],
			)

			// Skip unavailable items
			if !item.IsAvailable {
				continue
			}
			effectiveItems = append(effectiveItems, item)
		}

		if len(effectiveItems) == 0 {
			continue
		}

		effectiveCategories = append(effectiveCategories, &domain.EffectiveCategory{
			ID:        cat.ID,
			Name:      cat.Name,
			Code:      cat.Code,
			ImageURL:  cat.ImageURL,
			SortOrder: cat.SortOrder,
			Items:     effectiveItems,
		})
	}

	return &domain.EffectiveMenu{
		RestaurantID: restaurantID,
		Categories:   effectiveCategories,
		GeneratedAt:  time.Now(),
	}
}

// buildEffectiveItem applies override + promotion to a single item.
func buildEffectiveItem(
	item *domain.MenuItem,
	variations []*domain.MenuItemVariation,
	override *domain.RestaurantMenuItemOverride,
	promotion *domain.Promotion,
) *domain.EffectiveMenuItem {
	// Effective price: master → override
	price := item.BasePrice
	if override != nil && override.PriceOverride != nil {
		price = *override.PriceOverride
	}

	// Effective availability: master → override
	isAvailable := item.IsActive
	if override != nil && override.IsAvailableOverride != nil {
		isAvailable = *override.IsAvailableOverride
	}

	effective := &domain.EffectiveMenuItem{
		ID:          item.ID,
		ProductCode: item.ProductCode,
		Name:        item.Name,
		Description: item.Description,
		ImageURL:    item.ImageURL,
		BasePrice:   item.BasePrice,
		FinalPrice:  price,
		Currency:    "RUB",
		IsAvailable: isAvailable,
		SortOrder:   item.SortOrder,
	}

	// Apply promotion
	if promotion != nil {
		discounted := applyDiscount(price, promotion)
		effective.FinalPrice = discounted
		effective.Promotion = &domain.EffectivePromotion{
			ID:              promotion.ID,
			Name:            promotion.Name,
			DiscountType:    promotion.DiscountType,
			DiscountValue:   promotion.DiscountValue,
			OriginalPrice:   price,
			DiscountedPrice: discounted,
			EndsAt:          promotion.EndsAt,
		}
	}

	// Attach variations with effective prices
	for _, v := range variations {
		effective.Variations = append(effective.Variations, &domain.EffectiveVariation{
			ID:         v.ID,
			Name:       v.Name,
			PriceDelta: v.PriceDelta,
			FinalPrice: effective.FinalPrice + v.PriceDelta,
			IsDefault:  v.IsDefault,
			SortOrder:  v.SortOrder,
		})
	}

	return effective
}

// applyDiscount returns the discounted price.
func applyDiscount(price float64, p *domain.Promotion) float64 {
	switch p.DiscountType {
	case domain.DiscountTypePercentage:
		return price * (1 - p.DiscountValue/100)
	case domain.DiscountTypeFixed:
		result := price - p.DiscountValue
		if result < 0 {
			return 0
		}
		return result
	}
	return price
}

// buildPromotionIndex maps each item to its best active promotion.
// Empty MenuItemIDs on a promotion means it applies to all items.
// When multiple promotions apply, the one with the highest priority wins;
// ties are broken by the larger discount value.
func buildPromotionIndex(
	promotions []*domain.Promotion,
	items []*domain.MenuItem,
) map[uuid.UUID]*domain.Promotion {
	index := make(map[uuid.UUID]*domain.Promotion)

	for _, p := range promotions {
		targets := p.MenuItemIDs
		if len(targets) == 0 {
			targets = make([]uuid.UUID, 0, len(items))
			for _, it := range items {
				targets = append(targets, it.ID)
			}
		}

		for _, itemID := range targets {
			existing, ok := index[itemID]
			if !ok {
				index[itemID] = p
				continue
			}
			if p.Priority > existing.Priority {
				index[itemID] = p
				continue
			}
			if p.Priority == existing.Priority && p.DiscountValue > existing.DiscountValue {
				index[itemID] = p
			}
		}
	}

	return index
}
