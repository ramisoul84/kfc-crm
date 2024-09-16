package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// EffectiveMenuRepository loads the raw data needed to compute an effective menu.
type EffectiveMenuRepository interface {
	// LoadCategories returns all active categories ordered by sort_order.
	LoadCategories(ctx context.Context) ([]*domain.MenuCategory, error)

	// LoadMenuItems returns all active menu items.
	LoadMenuItems(ctx context.Context) ([]*domain.MenuItem, error)

	// LoadVariationsByItemIDs returns active variations for the given items.
	LoadVariationsByItemIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*domain.MenuItemVariation, error)

	// LoadOverridesForRestaurant returns all overrides for the given restaurant.
	LoadOverridesForRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]*domain.RestaurantMenuItemOverride, error)

	// LoadActivePromotions returns promotions active for a restaurant at the given time.
	LoadActivePromotions(ctx context.Context, restaurantID, regionID uuid.UUID) ([]*domain.Promotion, error)
}

type effectiveMenuRepo struct {
	db *sqlx.DB
}

// NewEffectiveMenuRepository creates an EffectiveMenuRepository.
func NewEffectiveMenuRepository(db *sqlx.DB) EffectiveMenuRepository {
	return &effectiveMenuRepo{db: db}
}

func (r *effectiveMenuRepo) LoadCategories(ctx context.Context) ([]*domain.MenuCategory, error) {
	query := `
		SELECT id, name, code,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       sort_order, is_active, created_at, updated_at
		FROM menu_categories
		WHERE deleted_at IS NULL AND is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	categories := make([]*domain.MenuCategory, 0)
	if err := r.db.SelectContext(ctx, &categories, query); err != nil {
		return nil, fmt.Errorf("load categories: %w", err)
	}
	return categories, nil
}

func (r *effectiveMenuRepo) LoadMenuItems(ctx context.Context) ([]*domain.MenuItem, error) {
	query := `
		SELECT id, category_id, product_code, name,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       base_price, sort_order, is_active, created_at, updated_at
		FROM menu_items
		WHERE deleted_at IS NULL AND is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	items := make([]*domain.MenuItem, 0)
	if err := r.db.SelectContext(ctx, &items, query); err != nil {
		return nil, fmt.Errorf("load menu items: %w", err)
	}
	return items, nil
}

func (r *effectiveMenuRepo) LoadVariationsByItemIDs(ctx context.Context, itemIDs []uuid.UUID) ([]*domain.MenuItemVariation, error) {
	if len(itemIDs) == 0 {
		return []*domain.MenuItemVariation{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, menu_item_id, name, price_delta, is_default,
		       sort_order, is_active, created_at, updated_at
		FROM menu_item_variations
		WHERE menu_item_id IN (?) AND is_active = true
		ORDER BY menu_item_id, sort_order ASC, name ASC
	`, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("build variations query: %w", err)
	}
	query = r.db.Rebind(query)

	variations := make([]*domain.MenuItemVariation, 0)
	if err := r.db.SelectContext(ctx, &variations, query, args...); err != nil {
		return nil, fmt.Errorf("load variations: %w", err)
	}
	return variations, nil
}

func (r *effectiveMenuRepo) LoadOverridesForRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]*domain.RestaurantMenuItemOverride, error) {
	query := `
		SELECT id, restaurant_id, menu_item_id,
		       price_override, is_available_override,
		       COALESCE(reason, '') AS reason,
		       created_at, updated_at
		FROM restaurant_menu_overrides
		WHERE restaurant_id = $1
	`

	overrides := make([]*domain.RestaurantMenuItemOverride, 0)
	if err := r.db.SelectContext(ctx, &overrides, query, restaurantID); err != nil {
		return nil, fmt.Errorf("load overrides: %w", err)
	}
	return overrides, nil
}

func (r *effectiveMenuRepo) LoadActivePromotions(ctx context.Context, restaurantID, regionID uuid.UUID) ([]*domain.Promotion, error) {
	query := `
		SELECT id, name,
		       COALESCE(description, '') AS description,
		       scope, region_id, restaurant_id,
		       discount_type, discount_value, starts_at, ends_at,
		       is_active, priority, created_at, updated_at
		FROM menu_promotions
		WHERE deleted_at IS NULL
		  AND is_active = true
		  AND starts_at <= $1
		  AND ends_at >= $1
		  AND (
		    scope = 'global'
		    OR (scope = 'region'     AND region_id = $2)
		    OR (scope = 'restaurant' AND restaurant_id = $3)
		  )
		ORDER BY priority DESC, starts_at ASC
	`

	promotions := make([]*domain.Promotion, 0)
	if err := r.db.SelectContext(ctx, &promotions, query, time.Now(), regionID, restaurantID); err != nil {
		return nil, fmt.Errorf("load active promotions: %w", err)
	}

	// Attach item mappings
	for _, p := range promotions {
		items, err := r.loadPromotionItems(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.MenuItemIDs = items
	}
	return promotions, nil
}

func (r *effectiveMenuRepo) loadPromotionItems(ctx context.Context, promotionID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT menu_item_id FROM menu_promotion_items WHERE promotion_id = $1`

	items := make([]uuid.UUID, 0)
	if err := r.db.SelectContext(ctx, &items, query, promotionID); err != nil {
		return nil, fmt.Errorf("load promotion items: %w", err)
	}
	return items, nil
}
