package domain

import (
	"time"

	"github.com/google/uuid"
)

// MenuItem represents a product in the master menu.
type MenuItem struct {
	ID          uuid.UUID  `db:"id"           json:"id"`
	CategoryID  uuid.UUID  `db:"category_id"  json:"category_id"`
	ProductCode string     `db:"product_code" json:"product_code"`
	Name        string     `db:"name"         json:"name"`
	Description string     `db:"description"  json:"description,omitempty"`
	ImageURL    string     `db:"image_url"    json:"image_url,omitempty"`
	BasePrice   float64    `db:"base_price"   json:"base_price"`
	SortOrder   int        `db:"sort_order"   json:"sort_order"`
	IsActive    bool       `db:"is_active"    json:"is_active"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"   json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"   json:"-"`

	// Relations
	Category   *MenuCategory        `db:"-" json:"category,omitempty"`
	Variations []*MenuItemVariation `db:"-" json:"variations,omitempty"`
}

// CreateMenuItemRequest is the payload for creating a menu item.
type CreateMenuItemRequest struct {
	CategoryID  uuid.UUID `json:"category_id"  validate:"required"`
	ProductCode string    `json:"product_code" validate:"required,min=2,max=50"`
	Name        string    `json:"name"         validate:"required,min=2,max=200"`
	Description string    `json:"description"  validate:"omitempty,max=1000"`
	ImageURL    string    `json:"image_url"    validate:"omitempty,url"`
	BasePrice   float64   `json:"base_price"   validate:"required,gt=0"`
	SortOrder   int       `json:"sort_order"`
}

// UpdateMenuItemRequest is the payload for updating a menu item.
type UpdateMenuItemRequest struct {
	CategoryID  uuid.UUID `json:"category_id"  validate:"required"`
	Name        string    `json:"name"         validate:"required,min=2,max=200"`
	Description string    `json:"description"  validate:"omitempty,max=1000"`
	ImageURL    string    `json:"image_url"    validate:"omitempty,url"`
	BasePrice   float64   `json:"base_price"   validate:"required,gt=0"`
	SortOrder   int       `json:"sort_order"`
	IsActive    *bool     `json:"is_active,omitempty"`
}

// MenuItemFilter represents filtering and pagination options.
type MenuItemFilter struct {
	Search     string
	CategoryID *uuid.UUID
	IsActive   *bool
	Limit      int
	Offset     int
}

// MenuItemResponse is the API response for a menu item.
type MenuItemResponse struct {
	ID          uuid.UUID             `json:"id"`
	CategoryID  uuid.UUID             `json:"category_id"`
	Category    *MenuCategoryResponse `json:"category,omitempty"`
	ProductCode string                `json:"product_code"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	ImageURL    string                `json:"image_url,omitempty"`
	BasePrice   float64               `json:"base_price"`
	SortOrder   int                   `json:"sort_order"`
	IsActive    bool                  `json:"is_active"`
	CreatedAt   time.Time             `json:"created_at"`
}

// ToResponse converts a MenuItem into a MenuItemResponse.
func (m *MenuItem) ToResponse() *MenuItemResponse {
	resp := &MenuItemResponse{
		ID:          m.ID,
		CategoryID:  m.CategoryID,
		ProductCode: m.ProductCode,
		Name:        m.Name,
		Description: m.Description,
		ImageURL:    m.ImageURL,
		BasePrice:   m.BasePrice,
		SortOrder:   m.SortOrder,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
	}
	if m.Category != nil {
		resp.Category = m.Category.ToResponse()
	}
	return resp
}
