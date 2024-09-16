package domain

import (
	"time"

	"github.com/google/uuid"
)

// EffectiveMenu is the computed menu for a single restaurant.
// It is never stored — always computed from master + overrides + promotions.
type EffectiveMenu struct {
	RestaurantID uuid.UUID            `json:"restaurant_id"`
	Categories   []*EffectiveCategory `json:"categories"`
	GeneratedAt  time.Time            `json:"generated_at"`
}

// EffectiveCategory is a category with its effective items.
type EffectiveCategory struct {
	ID        uuid.UUID            `json:"id"`
	Name      string               `json:"name"`
	Code      string               `json:"code"`
	ImageURL  string               `json:"image_url,omitempty"`
	SortOrder int                  `json:"sort_order"`
	Items     []*EffectiveMenuItem `json:"items"`
}

// EffectiveMenuItem is a menu item with restaurant-specific price and
// availability plus any applied promotions.
type EffectiveMenuItem struct {
	ID          uuid.UUID `json:"id"`
	ProductCode string    `json:"product_code"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`

	BasePrice  float64 `json:"base_price"`  // master price
	FinalPrice float64 `json:"final_price"` // after override and promotion
	Currency   string  `json:"currency"`    // "RUB"

	IsAvailable bool `json:"is_available"`

	// Applied promotion, if any
	Promotion *EffectivePromotion `json:"promotion,omitempty"`

	// Variations for this item
	Variations []*EffectiveVariation `json:"variations,omitempty"`

	SortOrder int `json:"sort_order"`
}

// EffectiveVariation is a variation with the effective price.
type EffectiveVariation struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	PriceDelta float64   `json:"price_delta"`
	FinalPrice float64   `json:"final_price"` // FinalPrice + PriceDelta
	IsDefault  bool      `json:"is_default"`
	SortOrder  int       `json:"sort_order"`
}

// EffectivePromotion describes an applied promotion on an item.
type EffectivePromotion struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	DiscountType    DiscountType `json:"discount_type"`
	DiscountValue   float64      `json:"discount_value"`
	OriginalPrice   float64      `json:"original_price"`
	DiscountedPrice float64      `json:"discounted_price"`
	EndsAt          time.Time    `json:"ends_at"`
}
