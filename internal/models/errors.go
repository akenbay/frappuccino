package models

import "errors"

var (
	ErrInvalidOrderID       = errors.New("invalid order ID")
	ErrEmptyOrder           = errors.New("order must contain at least one item")
	ErrInvalidTotalPrice    = errors.New("total price must be positive")
	ErrInvalidDateRange     = errors.New("invalid date range")
	ErrEmptyBatch           = errors.New("batch must contain at least one order")
	ErrInvalidMonth         = errors.New("invalid month")
	ErrInvalidYear          = errors.New("invalid year")
	ErrEmptySearchQuery     = errors.New("search query cannot be empty")
	ErrInvalidPriceRange    = errors.New("invalid price range")
	ErrInvalidNumberRange   = errors.New("invalid number range")
	ErrInvalidPeriod        = errors.New("invalid period, must be 'day' or 'month'")
	ErrInvalidPage          = errors.New("invalid page")
	ErrInvalidPageSize      = errors.New("invalid page size")
	ErrInvalidMenuItemID    = errors.New("invalid menu item id")
	ErrInvalidMenuItemName  = errors.New("invalid menu item name")
	ErrInvalidMenuItemPrice = errors.New("invalid menu item price")
)
