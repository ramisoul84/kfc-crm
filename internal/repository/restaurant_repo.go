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

// RestaurantRepository defines data access for restaurants.
type RestaurantRepository interface {
	Create(ctx context.Context, restaurant *domain.Restaurant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Restaurant, error)
	GetByIDWithRegion(ctx context.Context, id uuid.UUID) (*domain.Restaurant, error)
	GetByCode(ctx context.Context, code string) (*domain.Restaurant, error)
	List(ctx context.Context, filter domain.RestaurantFilter) ([]*domain.Restaurant, int, error)
	Update(ctx context.Context, restaurant *domain.Restaurant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type restaurantRepo struct {
	db *sqlx.DB
}

// NewRestaurantRepository creates a new RestaurantRepository.
func NewRestaurantRepository(db *sqlx.DB) RestaurantRepository {
	return &restaurantRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// JOIN ROW HELPER
// ═══════════════════════════════════════════════════════════════════

// restaurantRow is a flat representation of a restaurant joined with its region.
// Used only for scanning — converted to *domain.Restaurant after.
type restaurantRow struct {
	ID         uuid.UUID `db:"id"`
	Name       string    `db:"name"`
	Code       string    `db:"code"`
	RegionID   uuid.UUID `db:"region_id"`
	Address    string    `db:"address"`
	City       string    `db:"city"`
	PostalCode string    `db:"postal_code"`
	Phone      string    `db:"phone"`
	IsActive   bool      `db:"is_active"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`

	RegionName string `db:"region_name"`
	RegionCode string `db:"region_code"`
}

func (row *restaurantRow) toDomain() *domain.Restaurant {
	return &domain.Restaurant{
		ID:         row.ID,
		Name:       row.Name,
		Code:       row.Code,
		RegionID:   row.RegionID,
		Address:    row.Address,
		City:       row.City,
		PostalCode: row.PostalCode,
		Phone:      row.Phone,
		IsActive:   row.IsActive,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		Region: &domain.Region{
			ID:   row.RegionID,
			Name: row.RegionName,
			Code: row.RegionCode,
		},
	}
}

// Shared SELECT column list for joined queries.
const joinedRestaurantColumns = `
	res.id, res.name, res.code, res.region_id,
	COALESCE(res.address, '')      AS address,
	COALESCE(res.city, '')         AS city,
	COALESCE(res.postal_code, '')  AS postal_code,
	COALESCE(res.phone, '')        AS phone,
	res.is_active, res.created_at, res.updated_at,
	reg.name AS region_name,
	reg.code AS region_code
`

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *restaurantRepo) Create(ctx context.Context, restaurant *domain.Restaurant) error {
	query := `
		INSERT INTO restaurants (
			id, name, code, region_id,
			address, city, postal_code, phone, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		restaurant.ID,
		restaurant.Name,
		restaurant.Code,
		restaurant.RegionID,
		nullableString(restaurant.Address),
		nullableString(restaurant.City),
		nullableString(restaurant.PostalCode),
		nullableString(restaurant.Phone),
		restaurant.IsActive,
	).Scan(&restaurant.CreatedAt, &restaurant.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("restaurant with this code already exists")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("region does not exist")
		}
		return fmt.Errorf("create restaurant: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (r *restaurantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Restaurant, error) {
	query := `
		SELECT id, name, code, region_id,
		       COALESCE(address, '')      AS address,
		       COALESCE(city, '')         AS city,
		       COALESCE(postal_code, '')  AS postal_code,
		       COALESCE(phone, '')        AS phone,
		       is_active, created_at, updated_at, deleted_at
		FROM restaurants
		WHERE id = $1 AND deleted_at IS NULL
	`

	restaurant := &domain.Restaurant{}
	err := r.db.GetContext(ctx, restaurant, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("restaurant not found")
		}
		return nil, fmt.Errorf("get restaurant: %w", err)
	}

	return restaurant, nil
}

func (r *restaurantRepo) GetByIDWithRegion(ctx context.Context, id uuid.UUID) (*domain.Restaurant, error) {
	query := `
		SELECT ` + joinedRestaurantColumns + `
		FROM restaurants res
		JOIN regions reg ON reg.id = res.region_id
		WHERE res.id = $1 AND res.deleted_at IS NULL
	`

	var row restaurantRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("restaurant not found")
		}
		return nil, fmt.Errorf("get restaurant with region: %w", err)
	}

	return row.toDomain(), nil
}

func (r *restaurantRepo) GetByCode(ctx context.Context, code string) (*domain.Restaurant, error) {
	query := `
		SELECT id, name, code, region_id,
		       COALESCE(address, '')      AS address,
		       COALESCE(city, '')         AS city,
		       COALESCE(postal_code, '')  AS postal_code,
		       COALESCE(phone, '')        AS phone,
		       is_active, created_at, updated_at, deleted_at
		FROM restaurants
		WHERE code = $1 AND deleted_at IS NULL
	`

	restaurant := &domain.Restaurant{}
	err := r.db.GetContext(ctx, restaurant, query, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("restaurant not found")
		}
		return nil, fmt.Errorf("get restaurant by code: %w", err)
	}

	return restaurant, nil
}

func (r *restaurantRepo) List(ctx context.Context, filter domain.RestaurantFilter) ([]*domain.Restaurant, int, error) {
	query := `
		SELECT ` + joinedRestaurantColumns + `
		FROM restaurants res
		JOIN regions reg ON reg.id = res.region_id
		WHERE res.deleted_at IS NULL
	`

	countQuery := `SELECT COUNT(*) FROM restaurants res WHERE res.deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions,
			fmt.Sprintf("(res.name ILIKE $%d OR res.code ILIKE $%d OR res.city ILIKE $%d)",
				argCount, argCount, argCount))
		argCount++
	}

	if filter.RegionID != nil {
		args = append(args, *filter.RegionID)
		conditions = append(conditions, fmt.Sprintf("res.region_id = $%d", argCount))
		argCount++
	}

	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("res.is_active = $%d", argCount))
		argCount++
	}

	if len(conditions) > 0 {
		where := " AND " + strings.Join(conditions, " AND ")
		query += where
		countQuery += where
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count restaurants: %w", err)
	}

	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query += " ORDER BY res.name ASC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	rows := make([]restaurantRow, 0)
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list restaurants: %w", err)
	}

	restaurants := make([]*domain.Restaurant, 0, len(rows))
	for i := range rows {
		restaurants = append(restaurants, rows[i].toDomain())
	}

	return restaurants, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *restaurantRepo) Update(ctx context.Context, restaurant *domain.Restaurant) error {
	query := `
		UPDATE restaurants
		SET name = $2, code = $3, region_id = $4,
		    address = $5, city = $6, postal_code = $7, phone = $8,
		    is_active = $9, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		restaurant.ID,
		restaurant.Name,
		restaurant.Code,
		restaurant.RegionID,
		nullableString(restaurant.Address),
		nullableString(restaurant.City),
		nullableString(restaurant.PostalCode),
		nullableString(restaurant.Phone),
		restaurant.IsActive,
	).Scan(&restaurant.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("restaurant not found")
		}
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("restaurant with this code already exists")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("region does not exist")
		}
		return fmt.Errorf("update restaurant: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE (soft)
// ═══════════════════════════════════════════════════════════════════

func (r *restaurantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE restaurants
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete restaurant: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("restaurant not found")
	}

	return nil
}
