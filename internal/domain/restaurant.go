package domain

import (
	"time"

	"github.com/google/uuid"
)

// Restaurant represents a KFC restaurant location.
type Restaurant struct {
	ID         uuid.UUID  `db:"id"            json:"id"`
	Name       string     `db:"name"          json:"name"`
	Code       string     `db:"code"          json:"code"`
	RegionID   uuid.UUID  `db:"region_id"     json:"region_id"`
	Address    string     `db:"address"       json:"address,omitempty"`
	City       string     `db:"city"          json:"city,omitempty"`
	PostalCode string     `db:"postal_code"   json:"postal_code,omitempty"`
	Phone      string     `db:"phone"         json:"phone,omitempty"`
	IsActive   bool       `db:"is_active"     json:"is_active"`
	CreatedAt  time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"    json:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"    json:"-"`

	// Relations
	Region *Region `db:"-" json:"region,omitempty"`
}

// CreateRestaurantRequest is the payload for creating a restaurant.
type CreateRestaurantRequest struct {
	Name       string    `json:"name"        validate:"required,min=2,max=200"`
	Code       string    `json:"code"        validate:"required,min=2,max=20,uppercase"`
	RegionID   uuid.UUID `json:"region_id"   validate:"required"`
	Address    string    `json:"address"     validate:"omitempty,max=500"`
	City       string    `json:"city"        validate:"omitempty,max=100"`
	PostalCode string    `json:"postal_code" validate:"omitempty,max=10"`
	Phone      string    `json:"phone"       validate:"omitempty,phone"`
}

// UpdateRestaurantRequest is the payload for updating a restaurant.
type UpdateRestaurantRequest struct {
	Name       string `json:"name"        validate:"required,min=2,max=200"`
	Code       string `json:"code"        validate:"required,min=2,max=20,uppercase"`
	Address    string `json:"address"     validate:"omitempty,max=500"`
	City       string `json:"city"        validate:"omitempty,max=100"`
	PostalCode string `json:"postal_code" validate:"omitempty,max=10"`
	Phone      string `json:"phone"       validate:"omitempty,phone"`
	IsActive   *bool  `json:"is_active,omitempty"`
}

// RestaurantFilter represents filtering and pagination options.
type RestaurantFilter struct {
	Search   string
	RegionID *uuid.UUID
	IsActive *bool
	Limit    int
	Offset   int
}

// RestaurantResponse is the API response for a restaurant.
type RestaurantResponse struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Code       string          `json:"code"`
	RegionID   uuid.UUID       `json:"region_id"`
	Region     *RegionResponse `json:"region,omitempty"`
	Address    string          `json:"address,omitempty"`
	City       string          `json:"city,omitempty"`
	PostalCode string          `json:"postal_code,omitempty"`
	Phone      string          `json:"phone,omitempty"`
	IsActive   bool            `json:"is_active"`
	CreatedAt  time.Time       `json:"created_at"`
}

// ToResponse converts a Restaurant into a RestaurantResponse.
func (r *Restaurant) ToResponse() *RestaurantResponse {
	resp := &RestaurantResponse{
		ID:         r.ID,
		Name:       r.Name,
		Code:       r.Code,
		RegionID:   r.RegionID,
		Address:    r.Address,
		City:       r.City,
		PostalCode: r.PostalCode,
		Phone:      r.Phone,
		IsActive:   r.IsActive,
		CreatedAt:  r.CreatedAt,
	}
	if r.Region != nil {
		resp.Region = r.Region.ToResponse()
	}
	return resp
}
