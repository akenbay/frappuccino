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
