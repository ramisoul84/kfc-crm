package domain

import (
	"time"

	"github.com/google/uuid"
)

// MenuItemVariation represents a variation of a menu item
// (e.g., Small/Medium/Large).
type MenuItemVariation struct {
	ID         uuid.UUID `db:"id"           json:"id"`
	MenuItemID uuid.UUID `db:"menu_item_id" json:"menu_item_id"`
	Name       string    `db:"name"         json:"name"`
	PriceDelta float64   `db:"price_delta"  json:"price_delta"`
	IsDefault  bool      `db:"is_default"   json:"is_default"`
	SortOrder  int       `db:"sort_order"   json:"sort_order"`
	IsActive   bool      `db:"is_active"    json:"is_active"`
	CreatedAt  time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"   json:"updated_at"`
}

// CreateMenuItemVariationRequest is the payload for creating a variation.
type CreateMenuItemVariationRequest struct {
	MenuItemID uuid.UUID `json:"menu_item_id" validate:"required"`
	Name       string    `json:"name"         validate:"required,min=1,max=100"`
	PriceDelta float64   `json:"price_delta"`
	IsDefault  bool      `json:"is_default"`
	SortOrder  int       `json:"sort_order"`
}

// UpdateMenuItemVariationRequest is the payload for updating a variation.
type UpdateMenuItemVariationRequest struct {
	Name       string  `json:"name"        validate:"required,min=1,max=100"`
	PriceDelta float64 `json:"price_delta"`
	IsDefault  bool    `json:"is_default"`
	SortOrder  int     `json:"sort_order"`
	IsActive   *bool   `json:"is_active,omitempty"`
}

// MenuItemVariationFilter represents filtering and pagination options.
type MenuItemVariationFilter struct {
	MenuItemID *uuid.UUID
	IsActive   *bool
	Limit      int
	Offset     int
}

// MenuItemVariationResponse is the API response for a variation.
type MenuItemVariationResponse struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menu_item_id"`
	Name       string    `json:"name"`
	PriceDelta float64   `json:"price_delta"`
	IsDefault  bool      `json:"is_default"`
	SortOrder  int       `json:"sort_order"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToResponse converts a MenuItemVariation into a MenuItemVariationResponse.
func (v *MenuItemVariation) ToResponse() *MenuItemVariationResponse {
	return &MenuItemVariationResponse{
		ID:         v.ID,
		MenuItemID: v.MenuItemID,
		Name:       v.Name,
		PriceDelta: v.PriceDelta,
		IsDefault:  v.IsDefault,
		SortOrder:  v.SortOrder,
		IsActive:   v.IsActive,
		CreatedAt:  v.CreatedAt,
	}
}
