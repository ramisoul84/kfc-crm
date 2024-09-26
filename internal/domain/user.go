package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents a staff member in the system.
type User struct {
	ID               uuid.UUID  `db:"id"                json:"id"`
	Email            string     `db:"email"             json:"email"`
	PasswordHash     string     `db:"password_hash"     json:"-"`
	FirstName        string     `db:"first_name"        json:"first_name,omitempty"`
	LastName         string     `db:"last_name"         json:"last_name,omitempty"`
	Phone            string     `db:"phone"             json:"phone,omitempty"`
	Role             Role       `db:"role"              json:"role"`
	RegionID         *uuid.UUID `db:"region_id"         json:"region_id,omitempty"`
	RestaurantID     *uuid.UUID `db:"restaurant_id"     json:"restaurant_id,omitempty"`
	IsActive         bool       `db:"is_active"         json:"is_active"`
	ProfileCompleted bool       `db:"profile_completed" json:"profile_completed"`
	LastLoginAt      *time.Time `db:"last_login_at"     json:"last_login_at,omitempty"`
	CreatedAt        time.Time  `db:"created_at"        json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"        json:"updated_at"`
	DeletedAt        *time.Time `db:"deleted_at"        json:"-"`
}

// CreateUserRequest is the payload for creating a user.
type CreateUserRequest struct {
	Email        string     `json:"email"         validate:"required,email,max=255"`
	Role         Role       `json:"role"          validate:"required"`
	RegionID     *uuid.UUID `json:"region_id,omitempty"`
	RestaurantID *uuid.UUID `json:"restaurant_id,omitempty"`
}

// FirstLoginSetupRequest is the payload for completing profile on first login.
type FirstLoginSetupRequest struct {
	FirstName   string `json:"first_name"   validate:"required,min=2,max=100"`
	LastName    string `json:"last_name"    validate:"required,min=2,max=100"`
	Phone       string `json:"phone"        validate:"omitempty,phone"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

// UpdateUserRequest is the payload for updating a user.
type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=2,max=100"`
	LastName  *string `json:"last_name,omitempty"  validate:"omitempty,min=2,max=100"`
	Phone     *string `json:"phone,omitempty"      validate:"omitempty,phone"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// UserFilter represents filtering and pagination options for listing users.
type UserFilter struct {
	Search       string
	Role         Role
	RegionID     *uuid.UUID
	RestaurantID *uuid.UUID
	IsActive     *bool
	Limit        int
	Offset       int
}

// UserResponse is the API response for a user.
type UserResponse struct {
	ID               uuid.UUID  `json:"id"`
	Email            string     `json:"email"`
	FirstName        string     `json:"first_name,omitempty"`
	LastName         string     `json:"last_name,omitempty"`
	Phone            string     `json:"phone,omitempty"`
	Role             Role       `json:"role"`
	RegionID         *uuid.UUID `json:"region_id,omitempty"`
	RestaurantID     *uuid.UUID `json:"restaurant_id,omitempty"`
	IsActive         bool       `json:"is_active"`
	ProfileCompleted bool       `json:"profile_completed"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ToResponse converts a User into a UserResponse.
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:               u.ID,
		Email:            u.Email,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Phone:            u.Phone,
		Role:             u.Role,
		RegionID:         u.RegionID,
		RestaurantID:     u.RestaurantID,
		IsActive:         u.IsActive,
		ProfileCompleted: u.ProfileCompleted,
		CreatedAt:        u.CreatedAt,
	}
}
