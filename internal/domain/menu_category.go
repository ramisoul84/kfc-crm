package domain

import (
	"time"

	"github.com/google/uuid"
)

// MenuCategory represents a menu category (Burgers, Drinks, Desserts).
type MenuCategory struct {
	ID          uuid.UUID  `db:"id"          json:"id"`
	Name        string     `db:"name"        json:"name"`
	Code        string     `db:"code"        json:"code"`
	Description string     `db:"description" json:"description,omitempty"`
	ImageURL    string     `db:"image_url"   json:"image_url,omitempty"`
	SortOrder   int        `db:"sort_order"  json:"sort_order"`
	IsActive    bool       `db:"is_active"   json:"is_active"`
	CreatedAt   time.Time  `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"  json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"  json:"-"`
}

// CreateMenuCategoryRequest is the payload for creating a category.
type CreateMenuCategoryRequest struct {
	Name        string `json:"name"        validate:"required,min=2,max=100"`
	Code        string `json:"code"        validate:"required,min=2,max=50,uppercase"`
	Description string `json:"description" validate:"omitempty,max=500"`
	ImageURL    string `json:"image_url"   validate:"omitempty,url"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateMenuCategoryRequest is the payload for updating a category.
type UpdateMenuCategoryRequest struct {
	Name        string `json:"name"        validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	ImageURL    string `json:"image_url"   validate:"omitempty,url"`
	SortOrder   int    `json:"sort_order"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// MenuCategoryFilter represents filtering and pagination options.
type MenuCategoryFilter struct {
	Search   string
	IsActive *bool
	Limit    int
	Offset   int
}

// MenuCategoryResponse is the API response for a category.
type MenuCategoryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ToResponse converts a MenuCategory into a MenuCategoryResponse.
func (c *MenuCategory) ToResponse() *MenuCategoryResponse {
	return &MenuCategoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Code:        c.Code,
		Description: c.Description,
		ImageURL:    c.ImageURL,
		SortOrder:   c.SortOrder,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
	}
}
