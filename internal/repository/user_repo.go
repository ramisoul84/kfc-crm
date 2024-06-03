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

// UserRepository defines data access for users.
type UserRepository interface {
	// Create
	Create(ctx context.Context, user *domain.User) error

	// Lookups
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// List
	List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int, error)

	// Updates
	Update(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error
	CompleteProfile(ctx context.Context, id uuid.UUID, firstName, lastName, phone, hashedPassword string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error

	// Delete (soft)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userRepo struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

// ═══════════════════════════════════════════════════════════════════
// BASE SELECT
// ═══════════════════════════════════════════════════════════════════

const userSelectColumns = `
	id, email, password_hash, role,
	COALESCE(first_name, '') AS first_name,
	COALESCE(last_name, '')  AS last_name,
	COALESCE(phone, '')      AS phone,
	region_id, restaurant_id,
	is_active, profile_completed,
	last_login_at, created_at, updated_at, deleted_at
`

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, role,
			region_id, restaurant_id
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.RegionID,
		user.RestaurantID,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if isDuplicateKeyError(err) {
			return domain.NewConflictError("user with this email already exists")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("referenced region or restaurant does not exist")
		}
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// LOOKUPS
// ═══════════════════════════════════════════════════════════════════

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + `
	          FROM users
	          WHERE id = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	if err := r.db.GetContext(ctx, user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userSelectColumns + `
	          FROM users
	          WHERE email = $1 AND deleted_at IS NULL`

	user := &domain.User{}
	if err := r.db.GetContext(ctx, user, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

func (r *userRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, email); err != nil {
		return false, fmt.Errorf("check user email existence: %w", err)
	}
	return exists, nil
}

// ═══════════════════════════════════════════════════════════════════
// LIST
// ═══════════════════════════════════════════════════════════════════

func (r *userRepo) List(ctx context.Context, filter domain.UserFilter) ([]*domain.User, int, error) {
	query := `SELECT ` + userSelectColumns + ` FROM users WHERE deleted_at IS NULL`
	countQuery := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`

	var conditions []string
	var args []interface{}
	argCount := 1

	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions,
			fmt.Sprintf("(email ILIKE $%d OR first_name ILIKE $%d OR last_name ILIKE $%d)",
				argCount, argCount, argCount))
		argCount++
	}

	if filter.Role != "" {
		args = append(args, filter.Role)
		conditions = append(conditions, fmt.Sprintf("role = $%d", argCount))
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
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Pagination
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

	users := make([]*domain.User, 0)
	if err := r.db.SelectContext(ctx, &users, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATES
// ═══════════════════════════════════════════════════════════════════

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET first_name = $2, last_name = $3, phone = $4,
		    role = $5, region_id = $6, restaurant_id = $7,
		    is_active = $8, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		user.ID,
		nullableString(user.FirstName),
		nullableString(user.LastName),
		nullableString(user.Phone),
		user.Role,
		user.RegionID,
		user.RestaurantID,
		user.IsActive,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.NewNotFoundError("user not found")
		}
		if isForeignKeyError(err) {
			return domain.NewValidationError("referenced region or restaurant does not exist")
		}
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *userRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hashedPassword string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, hashedPassword)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("user not found")
	}
	return nil
}

// CompleteProfile sets the profile fields and the new password in one update,
// and marks profile_completed = true.
func (r *userRepo) CompleteProfile(
	ctx context.Context,
	id uuid.UUID,
	firstName, lastName, phone, hashedPassword string,
) error {
	query := `
		UPDATE users
		SET first_name = $2, last_name = $3, phone = $4,
		    password_hash = $5,
		    profile_completed = true,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query,
		id,
		firstName,
		lastName,
		nullableString(phone),
		hashedPassword,
	)
	if err != nil {
		return fmt.Errorf("complete profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("user not found")
	}
	return nil
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE (soft)
// ═══════════════════════════════════════════════════════════════════

func (r *userRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = NOW(), is_active = false, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("user not found")
	}
	return nil
}
