package models

import "errors"

var (
	ErrInvalidOrderID    = errors.New("invalid order ID")
	ErrEmptyOrder        = errors.New("order must contain at least one item")
	ErrInvalidTotalPrice = errors.New("total price must be positive")
	ErrInvalidDateRange  = errors.New("invalid date range")
	ErrEmptyBatch        = errors.New("batch must contain at least one order")
)
