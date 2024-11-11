package domain

import (
	"time"

	"github.com/google/uuid"
)

// Orders live in kfc-order. These are CRM's read models for them, with
// money converted from kopecks to rubles and restaurant names attached.

// OrderFilter is the query for GET /orders.
type OrderFilter struct {
	RestaurantID *uuid.UUID
	Status       string
	Channel      string
	Number       int
	MemberID     *uuid.UUID
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

// OrderStatsFilter is the query for GET /orders/stats.
type OrderStatsFilter struct {
	RestaurantID *uuid.UUID
	From         time.Time
	To           time.Time
}

// CancelOrderRequest is the body of POST /orders/:id/cancel.
type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"required,min=3,max=500"`
}

type OrderLineResponse struct {
	ItemID        string  `json:"item_id"`
	ProductCode   string  `json:"product_code"`
	Name          string  `json:"name"`
	VariationName string  `json:"variation_name,omitempty"`
	Quantity      int     `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	LineTotal     float64 `json:"line_total"`
}

type OrderResponse struct {
	ID             string              `json:"id"`
	RestaurantID   string              `json:"restaurant_id"`
	RestaurantName string              `json:"restaurant_name,omitempty"`
	RestaurantCode string              `json:"restaurant_code,omitempty"`
	Number         int                 `json:"number"`
	BusinessDate   string              `json:"business_date"`
	Channel        string              `json:"channel"`
	OrderType      string              `json:"order_type"`
	Status         string              `json:"status"`
	PaymentMethod  string              `json:"payment_method"`
	PaymentStatus  string              `json:"payment_status"`
	CustomerName   string              `json:"customer_name,omitempty"`
	CustomerPhone  string              `json:"customer_phone,omitempty"`
	Note           string              `json:"note,omitempty"`
	DeviceID       string              `json:"device_id,omitempty"`
	Subtotal       float64             `json:"subtotal"`
	Discount       float64             `json:"discount"`
	PointsRedeemed int                 `json:"points_redeemed,omitempty"`
	MemberID       string              `json:"member_id,omitempty"`
	Total          float64             `json:"total"`
	Currency       string              `json:"currency"`
	CancelReason   string              `json:"cancel_reason,omitempty"`
	Lines          []OrderLineResponse `json:"lines"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	PreparingAt    *time.Time          `json:"preparing_at,omitempty"`
	ReadyAt        *time.Time          `json:"ready_at,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	CancelledAt    *time.Time          `json:"cancelled_at,omitempty"`
}

type OrderStatsResponse struct {
	From            time.Time         `json:"from"`
	To              time.Time         `json:"to"`
	TotalOrders     int               `json:"total_orders"`
	Revenue         float64           `json:"revenue"`
	AverageTicket   float64           `json:"average_ticket"`
	AvgPrepSeconds  int               `json:"avg_prep_seconds"`
	CancelledOrders int               `json:"cancelled_orders"`
	ActiveOrders    int               `json:"active_orders"`
	ByChannel       []ChannelStatResp `json:"by_channel"`
	ByHour          []HourStatResp    `json:"by_hour"`
	TopItems        []ItemStatResp    `json:"top_items"`
	ByStatus        []StatusStatResp  `json:"by_status"`
}

type ChannelStatResp struct {
	Channel string  `json:"channel"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type HourStatResp struct {
	Hour    int     `json:"hour"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type ItemStatResp struct {
	ItemID   string  `json:"item_id"`
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Revenue  float64 `json:"revenue"`
}

type StatusStatResp struct {
	Status string `json:"status"`
	Orders int    `json:"orders"`
}
