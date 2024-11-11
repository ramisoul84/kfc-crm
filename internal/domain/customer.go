package domain

import "time"

// Customers are loyalty members, owned by kfc-loyalty. These are CRM's
// read models for them.

type CustomerResponse struct {
	ID              string    `json:"id"`
	Phone           string    `json:"phone"`
	FirstName       string    `json:"first_name,omitempty"`
	Email           string    `json:"email,omitempty"`
	PointsBalance   int       `json:"points_balance"`
	PointsAvailable int       `json:"points_available"`
	LifetimePoints  int       `json:"lifetime_points"`
	CreatedAt       time.Time `json:"created_at"`
}

type PointsTransactionResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Points       int       `json:"points"`
	BalanceAfter int       `json:"balance_after"`
	OrderID      string    `json:"order_id,omitempty"`
	OrderNumber  int       `json:"order_number,omitempty"`
	RestaurantID string    `json:"restaurant_id,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type LoyaltyProgramResponse struct {
	EarnPercent      int     `json:"earn_percent"`
	MaxRedeemPercent int     `json:"max_redeem_percent"`
	WelcomeBonus     int     `json:"welcome_bonus"`
	PointValue       float64 `json:"point_value"`
}

// AdjustPointsRequest is the body of POST /customers/:id/points.
type AdjustPointsRequest struct {
	Points int    `json:"points"  validate:"required,min=-100000,max=100000"`
	Reason string `json:"reason"  validate:"required,min=3,max=200"`
}
