package models

import (
	"encoding/json"
	"time"
)

type Inventory struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Quantity     float64         `json:"quantity"`
	Unit         string          `json:"unit"`
	CostPerUnit  float64         `json:"cost_per_unit,omitempty"`
	ReOrderLevel float64         `json:"reorder_level,omitempty"`
	SupplierInfo json.RawMessage `json:"supplier_info,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type InventoryTransactions struct {
	ID              int       `json:"id"`
	IngredientID    int       `json:"ingredient_id"`
	Delta           float64   `json:"delta"`
	TransactionType string    `json:"transaction_type"`
	ReferenceID     int       `json:"reference_id,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}
