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

// MenuItemRepository defines data access for menu items.
type MenuItemRepository interface {
	Create(ctx context.Context, item *domain.MenuItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error)
	GetByIDWithCategory(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error)
	GetByProductCode(ctx context.Context, code string) (*domain.MenuItem, error)
	List(ctx context.Context, filter domain.MenuItemFilter) ([]*domain.MenuItem, int, error)
	Update(ctx context.Context, item *domain.MenuItem) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type menuItemRepo struct {
	db *sqlx.DB
}

// NewMenuItemRepository creates a MenuItemRepository.
func NewMenuItemRepository(db *sqlx.DB) MenuItemRepository {
	return &menuItemRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// JOIN ROW HELPERS
// ═══════════════════════════════════════════════════════════════════

type menuItemRow struct {
	ID          uuid.UUID `db:"id"`
	CategoryID  uuid.UUID `db:"category_id"`
	ProductCode string    `db:"product_code"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	ImageURL    string    `db:"image_url"`
	BasePrice   float64   `db:"base_price"`
	SortOrder   int       `db:"sort_order"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	CategoryName string `db:"category_name"`
	CategoryCode string `db:"category_code"`
}

func (row *menuItemRow) toDomain() *domain.MenuItem {
	return &domain.MenuItem{
		ID:          row.ID,
		CategoryID:  row.CategoryID,
		ProductCode: row.ProductCode,
		Name:        row.Name,
		Description: row.Description,
		ImageURL:    row.ImageURL,
		BasePrice:   row.BasePrice,
		SortOrder:   row.SortOrder,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Category: &domain.MenuCategory{
			ID:   row.CategoryID,
			Name: row.CategoryName,
			Code: row.CategoryCode,
		},
	}
}

const joinedMenuItemColumns = `
	mi.id, mi.category_id, mi.product_code, mi.name,
	COALESCE(mi.description, '') AS description,
	COALESCE(mi.image_url, '')   AS image_url,
	mi.base_price, mi.sort_order, mi.is_active,
	mi.created_at, mi.updated_at,
	mc.name AS category_name,
	mc.code AS category_code
`

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemRepo) Create(ctx context.Context, item *domain.MenuItem) error {
	query := `
		INSERT INTO menu_items (id, category_id, product_code, name, description, image_url, base_price, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		item.ID,
		item.CategoryID,
		item.ProductCode,
		item.Name,
		nullableString(item.Description),
		nullableString(item.ImageURL),
		item.BasePrice,
		item.SortOrder,
		item.IsActive,
	).Scan(&item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("menu item with this product code already exists")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("category does not exist")
		}
		return fmt.Errorf("create menu item: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	query := `
		SELECT id, category_id, product_code, name,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       base_price, sort_order, is_active, created_at, updated_at, deleted_at
		FROM menu_items
		WHERE id = $1 AND deleted_at IS NULL
	`

	item := &domain.MenuItem{}
	if err := r.db.GetContext(ctx, item, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item not found")
		}
		return nil, fmt.Errorf("get menu item: %w", err)
	}
	return item, nil
}

func (r *menuItemRepo) GetByIDWithCategory(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	query := `
		SELECT ` + joinedMenuItemColumns + `
		FROM menu_items mi
		JOIN menu_categories mc ON mc.id = mi.category_id
		WHERE mi.id = $1 AND mi.deleted_at IS NULL
	`

	var row menuItemRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item not found")
		}
		return nil, fmt.Errorf("get menu item with category: %w", err)
	}
	return row.toDomain(), nil
}

func (r *menuItemRepo) GetByProductCode(ctx context.Context, code string) (*domain.MenuItem, error) {
	query := `
		SELECT id, category_id, product_code, name,
		       COALESCE(description, '') AS description,
		       COALESCE(image_url, '')   AS image_url,
		       base_price, sort_order, is_active, created_at, updated_at, deleted_at
		FROM menu_items
		WHERE product_code = $1 AND deleted_at IS NULL
	`

	item := &domain.MenuItem{}
	if err := r.db.GetContext(ctx, item, query, code); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("menu item not found")
		}
		return nil, fmt.Errorf("get menu item by code: %w", err)
	}
	return item, nil
}

func (r *menuItemRepo) List(ctx context.Context, filter domain.MenuItemFilter) ([]*domain.MenuItem, int, error) {
	query := `
		SELECT ` + joinedMenuItemColumns + `
		FROM menu_items mi
		JOIN menu_categories mc ON mc.id = mi.category_id
		WHERE mi.deleted_at IS NULL
	`

	countQuery := `SELECT COUNT(*) FROM menu_items mi WHERE mi.deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions,
			fmt.Sprintf("(mi.name ILIKE $%d OR mi.product_code ILIKE $%d)", argCount, argCount))
		argCount++
	}
	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		conditions = append(conditions, fmt.Sprintf("mi.category_id = $%d", argCount))
		argCount++
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("mi.is_active = $%d", argCount))
		argCount++
	}

	if len(conditions) > 0 {
		where := " AND " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count menu items: %w", err)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query += " ORDER BY mi.sort_order ASC, mi.name ASC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	rows := make([]menuItemRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list menu items: %w", err)
	}

	items := make([]*domain.MenuItem, 0, len(rows))
	for i := range rows {
		items = append(items, rows[i].toDomain())
	}
	return items, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemRepo) Update(ctx context.Context, item *domain.MenuItem) error {
	query := `
		UPDATE menu_items
		SET category_id = $2, name = $3, description = $4, image_url = $5,
		    base_price = $6, sort_order = $7, is_active = $8, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		item.ID,
		item.CategoryID,
		item.Name,
		nullableString(item.Description),
		nullableString(item.ImageURL),
		item.BasePrice,
		item.SortOrder,
		item.IsActive,
	).Scan(&item.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("menu item not found")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("category does not exist")
		}
		return fmt.Errorf("update menu item: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (r *menuItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE menu_items
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete menu item: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("menu item not found")
	}
	return nil
}
