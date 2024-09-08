package domain

import (
	"time"

	"github.com/google/uuid"
)

// PromotionScope defines where a promotion applies.
type PromotionScope string

const (
	PromotionScopeGlobal     PromotionScope = "global"
	PromotionScopeRegion     PromotionScope = "region"
	PromotionScopeRestaurant PromotionScope = "restaurant"
)

// IsValid reports whether the scope is known.
func (s PromotionScope) IsValid() bool {
	switch s {
	case PromotionScopeGlobal, PromotionScopeRegion, PromotionScopeRestaurant:
		return true
	}
	return false
}

// DiscountType defines how a discount is calculated.
type DiscountType string

const (
	DiscountTypePercentage DiscountType = "percentage"
	DiscountTypeFixed      DiscountType = "fixed"
)

// IsValid reports whether the discount type is known.
func (d DiscountType) IsValid() bool {
	switch d {
	case DiscountTypePercentage, DiscountTypeFixed:
		return true
	}
	return false
}

// Promotion represents a scheduled discount.
type Promotion struct {
	ID            uuid.UUID      `db:"id"             json:"id"`
	Name          string         `db:"name"           json:"name"`
	Description   string         `db:"description"    json:"description,omitempty"`
	Scope         PromotionScope `db:"scope"          json:"scope"`
	RegionID      *uuid.UUID     `db:"region_id"      json:"region_id,omitempty"`
	RestaurantID  *uuid.UUID     `db:"restaurant_id"  json:"restaurant_id,omitempty"`
	DiscountType  DiscountType   `db:"discount_type"  json:"discount_type"`
	DiscountValue float64        `db:"discount_value" json:"discount_value"`
	StartsAt      time.Time      `db:"starts_at"      json:"starts_at"`
	EndsAt        time.Time      `db:"ends_at"        json:"ends_at"`
	IsActive      bool           `db:"is_active"      json:"is_active"`
	Priority      int            `db:"priority"       json:"priority"`
	CreatedAt     time.Time      `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"     json:"updated_at"`
	DeletedAt     *time.Time     `db:"deleted_at"     json:"-"`

	// Menu items this promotion applies to. Empty = applies to all items.
	MenuItemIDs []uuid.UUID `db:"-" json:"menu_item_ids,omitempty"`
}

// CreatePromotionRequest is the payload for creating a promotion.
type CreatePromotionRequest struct {
	Name          string         `json:"name"           validate:"required,min=2,max=200"`
	Description   string         `json:"description"    validate:"omitempty,max=1000"`
	Scope         PromotionScope `json:"scope"          validate:"required,scope"`
	RegionID      *uuid.UUID     `json:"region_id,omitempty"`
	RestaurantID  *uuid.UUID     `json:"restaurant_id,omitempty"`
	DiscountType  DiscountType   `json:"discount_type"  validate:"required,discount_type"`
	DiscountValue float64        `json:"discount_value" validate:"required,gt=0"`
	StartsAt      time.Time      `json:"starts_at"      validate:"required"`
	EndsAt        time.Time      `json:"ends_at"        validate:"required,gtfield=StartsAt"`
	IsActive      *bool          `json:"is_active,omitempty"`
	Priority      int            `json:"priority"`
	MenuItemIDs   []uuid.UUID    `json:"menu_item_ids,omitempty"`
}

// UpdatePromotionRequest is the payload for updating a promotion.
type UpdatePromotionRequest struct {
	Name          string       `json:"name"           validate:"required,min=2,max=200"`
	Description   string       `json:"description"    validate:"omitempty,max=1000"`
	DiscountType  DiscountType `json:"discount_type"  validate:"required,discount_type"`
	DiscountValue float64      `json:"discount_value" validate:"required,gt=0"`
	StartsAt      time.Time    `json:"starts_at"      validate:"required"`
	EndsAt        time.Time    `json:"ends_at"        validate:"required,gtfield=StartsAt"`
	IsActive      *bool        `json:"is_active,omitempty"`
	Priority      *int         `json:"priority,omitempty"`
	MenuItemIDs   []uuid.UUID  `json:"menu_item_ids,omitempty"`
}

// PromotionFilter represents filtering and pagination options.
type PromotionFilter struct {
	Scope        PromotionScope
	RegionID     *uuid.UUID
	RestaurantID *uuid.UUID
	ActiveNow    bool
	Limit        int
	Offset       int
}

// PromotionResponse is the API response for a promotion.
type PromotionResponse struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Scope         PromotionScope `json:"scope"`
	RegionID      *uuid.UUID     `json:"region_id,omitempty"`
	RestaurantID  *uuid.UUID     `json:"restaurant_id,omitempty"`
	DiscountType  DiscountType   `json:"discount_type"`
	DiscountValue float64        `json:"discount_value"`
	StartsAt      time.Time      `json:"starts_at"`
	EndsAt        time.Time      `json:"ends_at"`
	IsActive      bool           `json:"is_active"`
	Priority      int            `json:"priority"`
	MenuItemIDs   []uuid.UUID    `json:"menu_item_ids,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// ToResponse converts a Promotion into a PromotionResponse.
func (p *Promotion) ToResponse() *PromotionResponse {
	return &PromotionResponse{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		Scope:         p.Scope,
		RegionID:      p.RegionID,
		RestaurantID:  p.RestaurantID,
		DiscountType:  p.DiscountType,
		DiscountValue: p.DiscountValue,
		StartsAt:      p.StartsAt,
		EndsAt:        p.EndsAt,
		IsActive:      p.IsActive,
		Priority:      p.Priority,
		MenuItemIDs:   p.MenuItemIDs,
		CreatedAt:     p.CreatedAt,
	}
}
