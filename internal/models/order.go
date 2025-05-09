package models

import (
	"encoding/json"
	"time"
)

type Order struct {
	ID                  int             `json:"id"`
	CustomerID          int             `json:"customer_id"`
	Status              string          `json:"status"`
	PaymentMethod       string          `json:"payment_method,omitempty"`
	TotalPrice          float64         `json:"total_price"`
	SpecialInstructions json.RawMessage `json:"special_instructions,omitempty"`
	Items               []OrderItem     `json:"items"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type OrderItem struct {
	ID             int             `json:"id"`
	OrderID        int             `json:"order_id"`
	MenuItemID     int             `json:"menu_item_id"`
	Quantity       int             `json:"quantity"`
	Customizations json.RawMessage `json:"customizations,omitempty"`
	PriceAtOrder   float64         `json:"price_at_order"`
}

type OrderFilters struct {
	Status     string    `json:"status"`      // e.g., "pending", "completed"
	StartDate  time.Time `json:"start_date"`  // Filter orders after this date
	EndDate    time.Time `json:"end_date"`    // Filter orders before this date
	CustomerID int       `json:"customer_id"` // Optional: filter by customer
}

type BatchOrderRequest struct {
	Orders []Order `json:"orders"`
}

// BatchOrderResponse represents the result of batch processing
type BatchOrderResponse struct {
	ProcessedOrders []ProcessedOrder `json:"processed_orders"`
	Summary         BatchSummary     `json:"summary"`
}

type ProcessedOrder struct {
	OrderID      int     `json:"order_id"`
	CustomerName string  `json:"customer_name"`
	Status       string  `json:"status"`
	Total        float64 `json:"total"`
	Rejected     bool    `json:"rejected,omitempty"`
	RejectReason string  `json:"reject_reason,omitempty"`
}

type BatchSummary struct {
	TotalOrders   int              `json:"total_orders"`
	Accepted      int              `json:"accepted"`
	Rejected      int              `json:"rejected"`
	TotalRevenue  float64          `json:"total_revenue"`
	InventoryUsed []InventoryUsage `json:"inventory_used"`
}
