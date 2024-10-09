package events

import "time"

// Topic names.
const (
	TopicMenuUpdated = "crm.menu.updated"
)

// MenuUpdatedEvent is published whenever a menu entity changes:
// categories, items, variations, overrides, or promotions.
//
// userapi consumes this and deletes the affected restaurant's menu
// cache so the next request refetches the fresh effective menu.
type MenuUpdatedEvent struct {
	// What changed and where.
	RestaurantID string `json:"restaurant_id"`

	// RegionID is included so region-scoped promotions can invalidate
	// all restaurants in the region if needed later. Optional.
	RegionID string `json:"region_id,omitempty"`

	// Reason is a short string like "item.updated" or "promotion.created".
	// Purely informational for logs and debugging.
	Reason string `json:"reason"`

	// OccurredAt is when the event was emitted.
	OccurredAt time.Time `json:"occurred_at"`
}
