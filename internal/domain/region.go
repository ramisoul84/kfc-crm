package domain

import (
	"time"

	"github.com/google/uuid"
)

// Region represents a geographic region (city or region) in the KFC network.
type Region struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	Name      string     `db:"name"       json:"name"`
	Code      string     `db:"code"       json:"code"`
	Timezone  string     `db:"timezone"   json:"timezone"`
	IsActive  bool       `db:"is_active"  json:"is_active"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

// CreateRegionRequest is the payload for creating a region.
type CreateRegionRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Code     string `json:"code"     validate:"required,min=2,max=10,uppercase"`
	Timezone string `json:"timezone" validate:"required"`
}

// UpdateRegionRequest is the payload for updating a region.
type UpdateRegionRequest struct {
	Name     string `json:"name"     validate:"required,min=2,max=100"`
	Code     string `json:"code"     validate:"required,min=2,max=10,uppercase"`
	Timezone string `json:"timezone" validate:"required"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// RegionFilter represents filtering and pagination options for listing regions.
type RegionFilter struct {
	Search   string
	IsActive *bool
	Limit    int
	Offset   int
}

// RegionResponse is the API response for a region.
type RegionResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Timezone  string    `json:"timezone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a Region into a RegionResponse.
func (r *Region) ToResponse() *RegionResponse {
	return &RegionResponse{
		ID:        r.ID,
		Name:      r.Name,
		Code:      r.Code,
		Timezone:  r.Timezone,
		IsActive:  r.IsActive,
		CreatedAt: r.CreatedAt,
	}
}
