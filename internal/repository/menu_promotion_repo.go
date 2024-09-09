package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// MenuPromotionRepository defines data access for promotions.
type MenuPromotionRepository interface {
	Create(ctx context.Context, promotion *domain.Promotion) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Promotion, error)
	List(ctx context.Context, filter domain.PromotionFilter) ([]*domain.Promotion, int, error)
	Update(ctx context.Context, promotion *domain.Promotion) error
	Delete(ctx context.Context, id uuid.UUID) error

	// ListActiveForRestaurant returns promotions active at the given time
	// that apply to the given restaurant (global + region + restaurant scope).
	ListActiveForRestaurant(
		ctx context.Context,
		restaurantID uuid.UUID,
		regionID uuid.UUID,
		at time.Time,
	) ([]*domain.Promotion, error)
}

type menuPromotionRepo struct {
	db *sqlx.DB
}

// NewMenuPromotionRepository creates a MenuPromotionRepository.
func NewMenuPromotionRepository(db *sqlx.DB) MenuPromotionRepository {
	return &menuPromotionRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *menuPromotionRepo) Create(ctx context.Context, p *domain.Promotion) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO menu_promotions (
			id, name, description, scope, region_id, restaurant_id,
			discount_type, discount_value, starts_at, ends_at,
			is_active, priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		p.ID,
		p.Name,
		nullableString(p.Description),
		p.Scope,
		p.RegionID,
		p.RestaurantID,
		p.DiscountType,
		p.DiscountValue,
		p.StartsAt,
		p.EndsAt,
		p.IsActive,
		p.Priority,
	).Scan(&p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		if isCheckConstraintError(err) {
			return domain.NewValidationError("invalid promotion scope, discount, or date range")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("region or restaurant does not exist")
		}
		return fmt.Errorf("create promotion: %w", err)
	}

	// Insert item mappings
	if err := r.replaceItems(ctx, tx, p.ID, p.MenuItemIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (r *menuPromotionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Promotion, error) {
	query := `
		SELECT id, name,
		       COALESCE(description, '') AS description,
		       scope, region_id, restaurant_id,
		       discount_type, discount_value, starts_at, ends_at,
		       is_active, priority, created_at, updated_at, deleted_at
		FROM menu_promotions
		WHERE id = $1 AND deleted_at IS NULL
	`

	p := &domain.Promotion{}
	if err := r.db.GetContext(ctx, p, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("promotion not found")
		}
		return nil, fmt.Errorf("get promotion: %w", err)
	}

	items, err := r.loadItems(ctx, id)
	if err != nil {
		return nil, err
	}
	p.MenuItemIDs = items
	return p, nil
}

func (r *menuPromotionRepo) List(ctx context.Context, filter domain.PromotionFilter) ([]*domain.Promotion, int, error) {
	query := `
		SELECT id, name,
		       COALESCE(description, '') AS description,
		       scope, region_id, restaurant_id,
		       discount_type, discount_value, starts_at, ends_at,
		       is_active, priority, created_at, updated_at
		FROM menu_promotions
		WHERE deleted_at IS NULL
	`
	countQuery := `SELECT COUNT(*) FROM menu_promotions WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.Scope != "" {
		args = append(args, filter.Scope)
		conditions = append(conditions, fmt.Sprintf("scope = $%d", argCount))
		argCount++
	}
	if filter.RegionID != nil {
		args = append(args, *filter.RegionID)
		conditions = append(conditions, fmt.Sprintf("region_id = $%d", argCount))
		argCount++
	}
	if filter.RestaurantID != nil {
		args = append(args, *filter.RestaurantID)
		conditions = append(conditions, fmt.Sprintf("restaurant_id = $%d", argCount))
		argCount++
	}
	if filter.ActiveNow {
		conditions = append(conditions, "is_active = true AND starts_at <= NOW() AND ends_at >= NOW()")
	}

	if len(conditions) > 0 {
		where := " AND " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count promotions: %w", err)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query += " ORDER BY priority DESC, starts_at DESC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	promotions := make([]*domain.Promotion, 0)
	if err := r.db.SelectContext(ctx, &promotions, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list promotions: %w", err)
	}

	// Load item mappings for each promotion
	for _, p := range promotions {
		items, err := r.loadItems(ctx, p.ID)
		if err != nil {
			return nil, 0, err
		}
		p.MenuItemIDs = items
	}
	return promotions, total, nil
}

func (r *menuPromotionRepo) ListActiveForRestaurant(
	ctx context.Context,
	restaurantID, regionID uuid.UUID,
	at time.Time,
) ([]*domain.Promotion, error) {
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
	if err := r.db.SelectContext(ctx, &promotions, query, at, regionID, restaurantID); err != nil {
		return nil, fmt.Errorf("list active promotions: %w", err)
	}

	for _, p := range promotions {
		items, err := r.loadItems(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.MenuItemIDs = items
	}
	return promotions, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *menuPromotionRepo) Update(ctx context.Context, p *domain.Promotion) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE menu_promotions
		SET name = $2, description = $3,
		    discount_type = $4, discount_value = $5,
		    starts_at = $6, ends_at = $7,
		    is_active = $8, priority = $9, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err = tx.QueryRowContext(ctx, query,
		p.ID,
		p.Name,
		nullableString(p.Description),
		p.DiscountType,
		p.DiscountValue,
		p.StartsAt,
		p.EndsAt,
		p.IsActive,
		p.Priority,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("promotion not found")
		}
		if isCheckConstraintError(err) {
			return domain.NewValidationError("invalid promotion discount or date range")
		}
		return fmt.Errorf("update promotion: %w", err)
	}

	// Replace item mappings
	if err := r.replaceItems(ctx, tx, p.ID, p.MenuItemIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (r *menuPromotionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE menu_promotions
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete promotion: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("promotion not found")
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// ITEM MAPPINGS
// ═══════════════════════════════════════════════════════════════════

func (r *menuPromotionRepo) loadItems(ctx context.Context, promotionID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT menu_item_id FROM menu_promotion_items WHERE promotion_id = $1`

	items := make([]uuid.UUID, 0)
	if err := r.db.SelectContext(ctx, &items, query, promotionID); err != nil {
		return nil, fmt.Errorf("load promotion items: %w", err)
	}
	return items, nil
}

func (r *menuPromotionRepo) replaceItems(
	ctx context.Context,
	tx *sqlx.Tx,
	promotionID uuid.UUID,
	itemIDs []uuid.UUID,
) error {
	// Delete existing
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM menu_promotion_items WHERE promotion_id = $1`, promotionID); err != nil {
		return fmt.Errorf("clear promotion items: %w", err)
	}

	// Insert new
	for _, itemID := range itemIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO menu_promotion_items (promotion_id, menu_item_id) VALUES ($1, $2)`,
			promotionID, itemID,
		); err != nil {
			if isForeignKeyError(err) {
				return domain.NewValidationError("menu item does not exist: " + itemID.String())
			}
			return fmt.Errorf("insert promotion item: %w", err)
		}
	}
	return nil
}
