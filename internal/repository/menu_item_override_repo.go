package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// MenuItemOverrideRepository defines data access for restaurant menu overrides.
type MenuItemOverrideRepository interface {
	Upsert(ctx context.Context, override *domain.RestaurantMenuItemOverride) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RestaurantMenuItemOverride, error)
	Get(ctx context.Context, restaurantID, menuItemID uuid.UUID) (*domain.RestaurantMenuItemOverride, error)
	ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]*domain.RestaurantMenuItemOverride, error)
	List(ctx context.Context, filter domain.MenuItemOverrideFilter) ([]*domain.RestaurantMenuItemOverride, int, error)
	Update(ctx context.Context, override *domain.RestaurantMenuItemOverride) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type menuItemOverrideRepo struct {
	db *sqlx.DB
}

// NewMenuItemOverrideRepository creates a MenuItemOverrideRepository.
func NewMenuItemOverrideRepository(db *sqlx.DB) MenuItemOverrideRepository {
	return &menuItemOverrideRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// UPSERT
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemOverrideRepo) Upsert(ctx context.Context, o *domain.RestaurantMenuItemOverride) error {
	query := `
		INSERT INTO restaurant_menu_overrides
			(id, restaurant_id, menu_item_id, price_override, is_available_override, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (restaurant_id, menu_item_id) DO UPDATE SET
			price_override = EXCLUDED.price_override,
			is_available_override = EXCLUDED.is_available_override,
			reason = EXCLUDED.reason,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		o.ID,
		o.RestaurantID,
		o.MenuItemID,
		o.PriceOverride,
		o.IsAvailableOverride,
		nullableString(o.Reason),
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)

	if err != nil {
		if isForeignKeyError(err) {
			return domain.NewValidationError("restaurant or menu item does not exist")
		}
		if isCheckConstraintError(err) {
			return domain.NewValidationError("override must set price or availability")
		}
		return fmt.Errorf("upsert menu item override: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemOverrideRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RestaurantMenuItemOverride, error) {
	query := `
		SELECT id, restaurant_id, menu_item_id,
		       price_override, is_available_override,
		       COALESCE(reason, '') AS reason,
		       created_at, updated_at
		FROM restaurant_menu_overrides
		WHERE id = $1
	`

	o := &domain.RestaurantMenuItemOverride{}
	if err := r.db.GetContext(ctx, o, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item override not found")
		}
		return nil, fmt.Errorf("get menu item override: %w", err)
	}
	return o, nil
}

func (r *menuItemOverrideRepo) Get(ctx context.Context, restaurantID, menuItemID uuid.UUID) (*domain.RestaurantMenuItemOverride, error) {
	query := `
		SELECT id, restaurant_id, menu_item_id,
		       price_override, is_available_override,
		       COALESCE(reason, '') AS reason,
		       created_at, updated_at
		FROM restaurant_menu_overrides
		WHERE restaurant_id = $1 AND menu_item_id = $2
	`

	o := &domain.RestaurantMenuItemOverride{}
	if err := r.db.GetContext(ctx, o, query, restaurantID, menuItemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item override not found")
		}
		return nil, fmt.Errorf("get menu item override: %w", err)
	}
	return o, nil
}

func (r *menuItemOverrideRepo) ListByRestaurant(ctx context.Context, restaurantID uuid.UUID) ([]*domain.RestaurantMenuItemOverride, error) {
	query := `
		SELECT id, restaurant_id, menu_item_id,
		       price_override, is_available_override,
		       COALESCE(reason, '') AS reason,
		       created_at, updated_at
		FROM restaurant_menu_overrides
		WHERE restaurant_id = $1
		ORDER BY created_at DESC
	`

	overrides := make([]*domain.RestaurantMenuItemOverride, 0)
	if err := r.db.SelectContext(ctx, &overrides, query, restaurantID); err != nil {
		return nil, fmt.Errorf("list overrides by restaurant: %w", err)
	}
	return overrides, nil
}

func (r *menuItemOverrideRepo) List(ctx context.Context, filter domain.MenuItemOverrideFilter) ([]*domain.RestaurantMenuItemOverride, int, error) {
	query := `
		SELECT id, restaurant_id, menu_item_id,
		       price_override, is_available_override,
		       COALESCE(reason, '') AS reason,
		       created_at, updated_at
		FROM restaurant_menu_overrides
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM restaurant_menu_overrides WHERE 1=1`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.RestaurantID != nil {
		args = append(args, *filter.RestaurantID)
		conditions = append(conditions, fmt.Sprintf("restaurant_id = $%d", argCount))
		argCount++
	}
	if filter.MenuItemID != nil {
		args = append(args, *filter.MenuItemID)
		conditions = append(conditions, fmt.Sprintf("menu_item_id = $%d", argCount))
		argCount++
	}

	if len(conditions) > 0 {
		where := " AND " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count overrides: %w", err)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query += " ORDER BY created_at DESC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	overrides := make([]*domain.RestaurantMenuItemOverride, 0)
	if err := r.db.SelectContext(ctx, &overrides, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list overrides: %w", err)
	}
	return overrides, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemOverrideRepo) Update(ctx context.Context, o *domain.RestaurantMenuItemOverride) error {
	query := `
		UPDATE restaurant_menu_overrides
		SET price_override = $2,
		    is_available_override = $3,
		    reason = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		o.ID,
		o.PriceOverride,
		o.IsAvailableOverride,
		nullableString(o.Reason),
	).Scan(&o.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("menu item override not found")
		}
		if isCheckConstraintError(err) {
			return domain.NewValidationError("override must set price or availability")
		}
		return fmt.Errorf("update menu item override: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemOverrideRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM restaurant_menu_overrides WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete menu item override: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("menu item override not found")
	}
	return nil
}
