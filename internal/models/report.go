package models

import (
	"time"
)

// TotalSalesResponse - For GET /reports/total-sales
type TotalSalesResponse struct {
	TotalSales float64 `json:"total_sales"`
	StartDate  string  `json:"start_date,omitempty"`
	EndDate    string  `json:"end_date,omitempty"`
}

// PopularItem - For GET /reports/popular-items
type PopularItem struct {
	MenuItemID    int     `json:"menu_item_id"`
	Name          string  `json:"name"`
	OrderCount    int     `json:"order_count"`
	TotalQuantity int     `json:"total_quantity"`
	Percentage    float64 `json:"percentage,omitempty"` // Can be calculated client-side
}

// PeriodReport - For GET /reports/ordered-items-by-period
type PeriodReport struct {
	Period     interface{} `json:"period"` // Can be int (day/month) or string (month name)
	OrderCount int         `json:"order_count"`
	TotalSales float64     `json:"total_sales"`
}

// SearchResult - For GET /reports/search
type SearchResult struct {
	MenuItems []SearchMenuItem `json:"menu_items"`
	Orders    []SearchOrder    `json:"orders,omitempty"`
	Customers []SearchCustomer `json:"customers,omitempty"`
	Total     int              `json:"total_matches"`
}

type SearchMenuItem struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Relevance   float64 `json:"relevance,omitempty"`
}

type SearchOrder struct {
	ID           int      `json:"id"`
	CustomerName string   `json:"customer_name"`
	Items        []string `json:"items"`
	Total        float64  `json:"total"`
	Status       string   `json:"status"`
	Relevance    float64  `json:"relevance,omitempty"`
}

type SearchCustomer struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type PaginatedInventory struct {
	Items       []InventoryItem `json:"data"`
	TotalItems  int             `json:"total_items"`
	TotalPages  int             `json:"total_pages"`
	CurrentPage int             `json:"current_page"`
	PageSize    int             `json:"page_size"`
	HasNextPage bool            `json:"has_next_page"`
}

// ReportFilters - Common filters for reports
type ReportFilters struct {
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`
	Status    string    `json:"status,omitempty"`
	Category  string    `json:"category,omitempty"`
	SortBy    string    `json:"sort_by,omitempty"`
	Page      int       `json:"page,omitempty"`
	PageSize  int       `json:"page_size,omitempty"`
}

// SalesTrend - For future sales analytics
type SalesTrend struct {
	Date       time.Time `json:"date"`
	TotalSales float64   `json:"total_sales"`
	OrderCount int       `json:"order_count"`
	AvgOrder   float64   `json:"average_order_value"`
}
