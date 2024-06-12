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

// DeviceRepository defines data access for devices.
type DeviceRepository interface {
	Create(ctx context.Context, device *domain.Device) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error)
	GetBySerialNumber(ctx context.Context, serial string) (*domain.Device, error)
	List(ctx context.Context, filter domain.DeviceFilter) ([]*domain.Device, int, error)
	Update(ctx context.Context, device *domain.Device) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type deviceRepo struct {
	db *sqlx.DB
}

// NewDeviceRepository creates a new DeviceRepository.
func NewDeviceRepository(db *sqlx.DB) DeviceRepository {
	return &deviceRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *deviceRepo) Create(ctx context.Context, device *domain.Device) error {
	query := `
		INSERT INTO devices (id, restaurant_id, type, serial_number, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		device.ID,
		device.RestaurantID,
		device.Type,
		device.SerialNumber,
		device.IsActive,
	).Scan(&device.CreatedAt, &device.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("device with this serial number already exists")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("restaurant does not exist")
		}
		return fmt.Errorf("create device: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// LOOKUPS
// ═══════════════════════════════════════════════════════════════════

func (r *deviceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	query := `
		SELECT id, restaurant_id, type, serial_number, is_active,
		       created_at, updated_at, deleted_at
		FROM devices
		WHERE id = $1 AND deleted_at IS NULL
	`

	device := &domain.Device{}
	if err := r.db.GetContext(ctx, device, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("device not found")
		}
		return nil, fmt.Errorf("get device: %w", err)
	}
	return device, nil
}

func (r *deviceRepo) GetBySerialNumber(ctx context.Context, serial string) (*domain.Device, error) {
	query := `
		SELECT id, restaurant_id, type, serial_number, is_active,
		       created_at, updated_at, deleted_at
		FROM devices
		WHERE serial_number = $1 AND deleted_at IS NULL
	`

	device := &domain.Device{}
	if err := r.db.GetContext(ctx, device, query, serial); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("device not found")
		}
		return nil, fmt.Errorf("get device by serial: %w", err)
	}
	return device, nil
}

// ═══════════════════════════════════════════════════════════════════
// LIST
// ═══════════════════════════════════════════════════════════════════

func (r *deviceRepo) List(ctx context.Context, filter domain.DeviceFilter) ([]*domain.Device, int, error) {
	query := `
		SELECT id, restaurant_id, type, serial_number, is_active,
		       created_at, updated_at
		FROM devices
		WHERE deleted_at IS NULL
	`
	countQuery := `SELECT COUNT(*) FROM devices WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.RestaurantID != nil {
		args = append(args, *filter.RestaurantID)
		conditions = append(conditions, fmt.Sprintf("restaurant_id = $%d", argCount))
		argCount++
	}

	if filter.Type != "" {
		args = append(args, filter.Type)
		conditions = append(conditions, fmt.Sprintf("type = $%d", argCount))
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
		return nil, 0, fmt.Errorf("count devices: %w", err)
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

	devices := make([]*domain.Device, 0)
	if err := r.db.SelectContext(ctx, &devices, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}
	return devices, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (r *deviceRepo) Update(ctx context.Context, device *domain.Device) error {
	query := `
		UPDATE devices
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		device.ID,
		device.IsActive,
	).Scan(&device.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("device not found")
		}
		return fmt.Errorf("update device: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE (soft)
// ═══════════════════════════════════════════════════════════════════

func (r *deviceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE devices
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("device not found")
	}
	return nil
}
