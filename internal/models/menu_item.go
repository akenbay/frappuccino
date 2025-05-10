package models

import (
	"time"
)

type MenuItems struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Price       float64               `json:"price"`
	Category    []string              `json:"category,omitempty"`
	IsActive    bool                  `json:"is_active"`
	Ingredients []MenuItemIngredients `json:"ingredients"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type PriceHistory struct {
	ID         int       `json:"id"`
	MenuItemID int       `json:"menu_item_id"`
	OldPrice   float64   `json:"old_price"`
	NewPrice   float64   `json:"new_price"`
	ChangedAt  time.Time `json:"updated_at"`
}

type MenuItemIngredients struct {
	IngredientID int     `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
}
