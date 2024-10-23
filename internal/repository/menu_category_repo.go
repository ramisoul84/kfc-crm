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

// MenuCategoryRepository defines data access for menu categories.
type MenuCategoryRepository interface {
	Create(ctx context.Context, category *domain.MenuCategory) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuCategory, error)
	GetByCode(ctx context.Context, code string) (*domain.MenuCategory, error)
	List(ctx context.Context, filter domain.MenuCategoryFilter) ([]*domain.MenuCategory, int, error)
	Update(ctx context.Context, category *domain.MenuCategory) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type menuCategoryRepo struct {
	db *sqlx.DB
}

// NewMenuCategoryRepository creates a MenuCategoryRepository.
func NewMenuCategoryRepository(db *sqlx.DB) MenuCategoryRepository {
	return &menuCategoryRepo{db: db}
}

func (r *menuCategoryRepo) Create(ctx context.Context, category *domain.MenuCategory) error {
	query := `
		INSERT INTO menu_categories (id, name, code, description, image_url, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		category.ID,
		category.Name,
		category.Code,
		nullableString(category.Description),
		nullableString(category.ImageURL),
		category.SortOrder,
		category.IsActive,
	).Scan(&category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("category with this code already exists")
		}
		return fmt.Errorf("create menu category: %w", err)
	}
	return nil
}

func (r *menuCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuCategory, error) {
	query := `
		SELECT id, name, code,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       sort_order, is_active, created_at, updated_at, deleted_at
		FROM menu_categories
		WHERE id = $1 AND deleted_at IS NULL
	`

	category := &domain.MenuCategory{}
	if err := r.db.GetContext(ctx, category, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu category not found")
		}
		return nil, fmt.Errorf("get menu category: %w", err)
	}
	return category, nil
}

func (r *menuCategoryRepo) GetByCode(ctx context.Context, code string) (*domain.MenuCategory, error) {
	query := `
		SELECT id, name, code,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       sort_order, is_active, created_at, updated_at, deleted_at
		FROM menu_categories
		WHERE code = $1 AND deleted_at IS NULL
	`

	category := &domain.MenuCategory{}
	if err := r.db.GetContext(ctx, category, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu category not found")
		}
		return nil, fmt.Errorf("get menu category by code: %w", err)
	}
	return category, nil
}

func (r *menuCategoryRepo) List(ctx context.Context, filter domain.MenuCategoryFilter) ([]*domain.MenuCategory, int, error) {
	query := `
		SELECT id, name, code,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       sort_order, is_active, created_at, updated_at
		FROM menu_categories
		WHERE deleted_at IS NULL
	`
	countQuery := `SELECT COUNT(*) FROM menu_categories WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions,
			fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", argCount, argCount))
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
		return nil, 0, fmt.Errorf("count menu categories: %w", err)
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

	categories := make([]*domain.MenuCategory, 0)
	if err := r.db.SelectContext(ctx, &categories, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list menu categories: %w", err)
	}
	return categories, total, nil
}

func (r *menuCategoryRepo) Update(ctx context.Context, category *domain.MenuCategory) error {
	query := `
		UPDATE menu_categories
		SET name = $2, description = $3, image_url = $4,
		    sort_order = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		category.ID,
		category.Name,
		nullableString(category.Description),
		nullableString(category.ImageURL),
		category.SortOrder,
		category.IsActive,
	).Scan(&category.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("menu category not found")
		}
		return fmt.Errorf("update menu category: %w", err)
	}
	return nil
}

func (r *menuCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE menu_categories
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete menu category: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("menu category not found")
	}
	return nil
}
