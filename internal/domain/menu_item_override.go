package domain

import (
	"time"

	"github.com/google/uuid"
)

// RestaurantMenuItemOverride represents a per-restaurant modification
// of a global menu item.
//
// A row exists ONLY when a restaurant differs from the master menu.
// If no row exists for a (restaurant_id, menu_item_id) pair, the
// master values apply.
//
// Nil fields mean "use master value".
type RestaurantMenuItemOverride struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	RestaurantID uuid.UUID `db:"restaurant_id" json:"restaurant_id"`
	MenuItemID   uuid.UUID `db:"menu_item_id"  json:"menu_item_id"`

	PriceOverride       *float64 `db:"price_override"        json:"price_override,omitempty"`
	IsAvailableOverride *bool    `db:"is_available_override" json:"is_available_override,omitempty"`

	Reason    string    `db:"reason"     json:"reason,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// CreateMenuItemOverrideRequest is the payload for setting an override.
type CreateMenuItemOverrideRequest struct {
	RestaurantID        uuid.UUID `json:"restaurant_id"         validate:"required"`
	MenuItemID          uuid.UUID `json:"menu_item_id"          validate:"required"`
	PriceOverride       *float64  `json:"price_override,omitempty"        validate:"omitempty,gt=0"`
	IsAvailableOverride *bool     `json:"is_available_override,omitempty"`
	Reason              string    `json:"reason"                validate:"omitempty,max=500"`
}

// UpdateMenuItemOverrideRequest is the payload for updating an override.
type UpdateMenuItemOverrideRequest struct {
	PriceOverride       *float64 `json:"price_override,omitempty"        validate:"omitempty,gt=0"`
	IsAvailableOverride *bool    `json:"is_available_override,omitempty"`
	Reason              string   `json:"reason"                validate:"omitempty,max=500"`
}

// MenuItemOverrideFilter represents filtering and pagination options.
type MenuItemOverrideFilter struct {
	RestaurantID *uuid.UUID
	MenuItemID   *uuid.UUID
	Limit        int
	Offset       int
}

// MenuItemOverrideResponse is the API response for an override.
type MenuItemOverrideResponse struct {
	ID                  uuid.UUID `json:"id"`
	RestaurantID        uuid.UUID `json:"restaurant_id"`
	MenuItemID          uuid.UUID `json:"menu_item_id"`
	PriceOverride       *float64  `json:"price_override,omitempty"`
	IsAvailableOverride *bool     `json:"is_available_override,omitempty"`
	Reason              string    `json:"reason,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ToResponse converts a RestaurantMenuItemOverride into a response.
func (o *RestaurantMenuItemOverride) ToResponse() *MenuItemOverrideResponse {
	return &MenuItemOverrideResponse{
		ID:                  o.ID,
		RestaurantID:        o.RestaurantID,
		MenuItemID:          o.MenuItemID,
		PriceOverride:       o.PriceOverride,
		IsAvailableOverride: o.IsAvailableOverride,
		Reason:              o.Reason,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
	}
}
