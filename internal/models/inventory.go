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

// InventoryItem represents a simplified view of inventory for reporting purposes
type InventoryItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	CostPerUnit float64 `json:"cost_per_unit,omitempty"`
}

// PaginatedInventoryResponse contains the paginated results and metadata
type PaginatedInventoryResponse struct {
	Items       []InventoryItem `json:"items"`
	TotalCount  int             `json:"total_count"`
	CurrentPage int             `json:"current_page"`
	PageSize    int             `json:"page_size"`
	TotalPages  int             `json:"total_pages"`
	HasNext     bool            `json:"has_next"`
}

// InventoryAlert represents items that are below reorder level
type InventoryAlert struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	CurrentStock  float64 `json:"current_stock"`
	ReorderLevel  float64 `json:"reorder_level"`
	DaysRemaining float64 `json:"days_remaining,omitempty"`
}

// InventoryUpdateRequest represents payload for updating inventory
type InventoryUpdateRequest struct {
	Name         *string          `json:"name,omitempty"`
	Quantity     *float64         `json:"quantity,omitempty"`
	Unit         *string          `json:"unit,omitempty"`
	CostPerUnit  *float64         `json:"cost_per_unit,omitempty"`
	ReOrderLevel *float64         `json:"reorder_level,omitempty"`
	SupplierInfo *json.RawMessage `json:"supplier_info,omitempty"`
}
