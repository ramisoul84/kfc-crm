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

// RegionRepository defines data access for regions.
type RegionRepository interface {
	Create(ctx context.Context, region *domain.Region) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Region, error)
	GetByCode(ctx context.Context, code string) (*domain.Region, error)
	List(ctx context.Context, filter domain.RegionFilter) ([]*domain.Region, int, error)
	Update(ctx context.Context, region *domain.Region) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type regionRepo struct {
	db *sqlx.DB
}

// NewRegionRepository creates a new RegionRepository.
func NewRegionRepository(db *sqlx.DB) RegionRepository {
	return &regionRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *regionRepo) Create(ctx context.Context, region *domain.Region) error {
	query := `
		INSERT INTO regions (id, name, code, timezone, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		region.ID,
		region.Name,
		region.Code,
		region.Timezone,
		region.IsActive,
	).Scan(&region.CreatedAt, &region.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("region with this code already exists")
		}
		return fmt.Errorf("create region: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (r *regionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Region, error) {
	query := `
		SELECT id, name, code, timezone, is_active, created_at, updated_at, deleted_at
		FROM regions
		WHERE id = $1 AND deleted_at IS NULL
	`

	region := &domain.Region{}
	err := r.db.GetContext(ctx, region, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("region not found")
		}
		return nil, fmt.Errorf("get region: %w", err)
	}

	return region, nil
}

func (r *regionRepo) GetByCode(ctx context.Context, code string) (*domain.Region, error) {
	query := `
		SELECT id, name, code, timezone, is_active, created_at, updated_at, deleted_at
		FROM regions
		WHERE code = $1 AND deleted_at IS NULL
	`

	region := &domain.Region{}
	err := r.db.GetContext(ctx, region, query, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("region not found")
		}
		return nil, fmt.Errorf("get region by code: %w", err)
	}

	return region, nil
}

func (r *regionRepo) List(ctx context.Context, filter domain.RegionFilter) ([]*domain.Region, int, error) {
	// Base query
	query := `SELECT id, name, code, timezone, is_active, created_at, updated_at
	          FROM regions
	          WHERE deleted_at IS NULL`

	countQuery := `SELECT COUNT(*) FROM regions WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	// Search by name or code
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions,
			fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", argCount, argCount))
		argCount++
	}

	// Filter by active flag
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

	// Total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count regions: %w", err)
	}

	// Pagination defaults
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	// Add pagination
	query += " ORDER BY name ASC"
	args = append(args, filter.Limit)
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	argCount++
	args = append(args, filter.Offset)
	query += fmt.Sprintf(" OFFSET $%d", argCount)

	// Execute
	regions := make([]*domain.Region, 0)
	if err := r.db.SelectContext(ctx, &regions, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list regions: %w", err)
	}

	return regions, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *regionRepo) Update(ctx context.Context, region *domain.Region) error {
	query := `
		UPDATE regions
		SET name = $2, code = $3, timezone = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		region.ID,
		region.Name,
		region.Code,
		region.Timezone,
		region.IsActive,
	).Scan(&region.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("region not found")
		}
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("region with this code already exists")
		}
		return fmt.Errorf("update region: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE (soft)
// ═══════════════════════════════════════════════════════════════════

func (r *regionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE regions
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete region: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("region not found")
	}

	return nil
}
