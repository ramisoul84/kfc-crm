package domain

import (
	"time"

	"github.com/google/uuid"
)

// MenuItemVariation represents a variation of a menu item
// (e.g., Small/Medium/Large).
type MenuItemVariation struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	MenuItemID uuid.UUID `db:"menu_item_id" json:"menu_item_id"`
	Name       string    `db:"name"        json:"name"`
	PriceDelta float64   `db:"price_delta" json:"price_delta"`
	IsDefault  bool      `db:"is_default"  json:"is_default"`
	SortOrder  int       `db:"sort_order"  json:"sort_order"`
	IsActive   bool      `db:"is_active"   json:"is_active"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}
