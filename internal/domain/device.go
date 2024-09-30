package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeviceType represents the type of device.
type DeviceType string

const (
	DeviceTypePOS          DeviceType = "pos"           // Point of Sale
	DeviceTypeKiosk        DeviceType = "kiosk"         // Self-service terminal
	DeviceTypeKDS          DeviceType = "kds"           // Kitchen Display System
	DeviceTypeOrderDisplay DeviceType = "order_display" // Customer-facing order screen
)

// IsValid reports whether the device type is known.
func (d DeviceType) IsValid() bool {
	switch d {
	case DeviceTypePOS, DeviceTypeKiosk, DeviceTypeKDS, DeviceTypeOrderDisplay:
		return true
	}
	return false
}

// Device represents a physical device installed at a restaurant.
type Device struct {
	ID           uuid.UUID  `db:"id"            json:"id"`
	RestaurantID uuid.UUID  `db:"restaurant_id" json:"restaurant_id"`
	Type         DeviceType `db:"type"          json:"type"`
	SerialNumber string     `db:"serial_number" json:"serial_number"`
	IsActive     bool       `db:"is_active"     json:"is_active"`
	CreatedAt    time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"    json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"    json:"-"`
}

// CreateDeviceRequest is the payload for creating a device.
type CreateDeviceRequest struct {
	RestaurantID uuid.UUID  `json:"restaurant_id" validate:"required"`
	Type         DeviceType `json:"type"          validate:"required"`
	SerialNumber string     `json:"serial_number" validate:"required,min=4,max=100"`
}

// UpdateDeviceRequest is the payload for updating a device.
type UpdateDeviceRequest struct {
	IsActive *bool `json:"is_active,omitempty"`
}

// DeviceFilter represents filtering and pagination options.
type DeviceFilter struct {
	RestaurantID *uuid.UUID
	Type         DeviceType
	IsActive     *bool
	Limit        int
	Offset       int
}

// DeviceResponse is the API response for a device.
type DeviceResponse struct {
	ID           uuid.UUID  `json:"id"`
	RestaurantID uuid.UUID  `json:"restaurant_id"`
	Type         DeviceType `json:"type"`
	SerialNumber string     `json:"serial_number"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ToResponse converts a Device into a DeviceResponse.
func (d *Device) ToResponse() *DeviceResponse {
	return &DeviceResponse{
		ID:           d.ID,
		RestaurantID: d.RestaurantID,
		Type:         d.Type,
		SerialNumber: d.SerialNumber,
		IsActive:     d.IsActive,
		CreatedAt:    d.CreatedAt,
	}
}
