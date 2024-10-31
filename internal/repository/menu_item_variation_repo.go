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

// MenuItemVariationRepository defines data access for menu item variations.
type MenuItemVariationRepository interface {
	Create(ctx context.Context, variation *domain.MenuItemVariation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuItemVariation, error)
	ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.MenuItemVariation, error)
	List(ctx context.Context, filter domain.MenuItemVariationFilter) ([]*domain.MenuItemVariation, int, error)
	Update(ctx context.Context, variation *domain.MenuItemVariation) error
	Delete(ctx context.Context, id uuid.UUID) error
}
type menuItemVariationRepo struct {
	db *sqlx.DB
}

// NewMenuItemVariationRepository creates a MenuItemVariationRepository.
func NewMenuItemVariationRepository(db *sqlx.DB) MenuItemVariationRepository {
	return &menuItemVariationRepo{db: db}
}

func (r *menuItemVariationRepo) Create(ctx context.Context, v *domain.MenuItemVariation) error {
	query := `
		INSERT INTO menu_item_variations
			(id, menu_item_id, name, price_delta, is_default, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		v.ID,
		v.MenuItemID,
		v.Name,
		v.PriceDelta,
		v.IsDefault,
		v.SortOrder,
		v.IsActive,
	).Scan(&v.CreatedAt, &v.UpdatedAt)

	if err != nil {
		if isForeignKeyError(err) {
			return domain.NewValidationError("menu item does not exist")
		}
		return fmt.Errorf("create menu item variation: %w", err)
	}
	return nil
}

func (r *menuItemVariationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuItemVariation, error) {
	query := `
		SELECT id, menu_item_id, name, price_delta, is_default,
		       sort_order, is_active, created_at, updated_at
		FROM menu_item_variations
		WHERE id = $1
	`

	v := &domain.MenuItemVariation{}
	if err := r.db.GetContext(ctx, v, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item variation not found")
		}
		return nil, fmt.Errorf("get menu item variation: %w", err)
	}
	return v, nil
}

func (r *menuItemVariationRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.MenuItemVariation, error) {
	query := `
		SELECT id, menu_item_id, name, price_delta, is_default,
		       sort_order, is_active, created_at, updated_at
		FROM menu_item_variations
		WHERE menu_item_id = $1 AND is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	variations := make([]*domain.MenuItemVariation, 0)
	if err := r.db.SelectContext(ctx, &variations, query, itemID); err != nil {
		return nil, fmt.Errorf("list variations by item: %w", err)
	}
	return variations, nil
}

func (r *menuItemVariationRepo) List(ctx context.Context, filter domain.MenuItemVariationFilter) ([]*domain.MenuItemVariation, int, error) {
	query := `
		SELECT id, menu_item_id, name, price_delta, is_default,
		       sort_order, is_active, created_at, updated_at
		FROM menu_item_variations
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM menu_item_variations WHERE 1=1`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.MenuItemID != nil {
		args = append(args, *filter.MenuItemID)
		conditions = append(conditions, fmt.Sprintf("menu_item_id = $%d", argCount))
		argCount++
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argCount))
		argCount++
	}

	if len(conditions) > 0 {
		where := " AND " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count variations: %w", err)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query += " ORDER BY sort_order ASC, name ASC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	variations := make([]*domain.MenuItemVariation, 0)
	if err := r.db.SelectContext(ctx, &variations, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list variations: %w", err)
	}
	return variations, total, nil
}

func (r *menuItemVariationRepo) Update(ctx context.Context, v *domain.MenuItemVariation) error {
	query := `
		UPDATE menu_item_variations
		SET name = $2, price_delta = $3, is_default = $4,
		    sort_order = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		v.ID,
		v.Name,
		v.PriceDelta,
		v.IsDefault,
		v.SortOrder,
		v.IsActive,
	).Scan(&v.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("menu item variation not found")
		}
		return fmt.Errorf("update menu item variation: %w", err)
	}
	return nil
}

func (r *menuItemVariationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE menu_item_variations
		SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND is_active = true
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete menu item variation: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("menu item variation not found")
	}
	return nil
}
