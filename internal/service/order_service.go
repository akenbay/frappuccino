package service

import (
	"context"
	"frappuccino/internal/dal"
	"frappuccino/internal/models"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order models.Order) (int, error)
	GetOrder(ctx context.Context, id int) (models.Order, error)
	ListOrders(ctx context.Context, filters models.OrderFilters) ([]models.Order, error)
	UpdateOrder(ctx context.Context, id int, order models.Order) error
	DeleteOrder(ctx context.Context, id int) error
	CloseOrder(ctx context.Context, id int) error
	GetOrderedItemsReport(ctx context.Context, startDate, endDate string) (map[string]int, error)
	ProcessBatchOrders(ctx context.Context, orders []models.Order) (models.BatchOrderResponse, error)
}

type orderService struct {
	orderRepo dal.OrderRepository
}

func NewOrderService(orderRepo dal.OrderRepository) OrderService {
	return &orderService{orderRepo: orderRepo}
}

func (s *orderService) CreateOrder(ctx context.Context, order models.Order) (int, error) {
	// Validate order
	if len(order.Items) == 0 {
		return 0, models.ErrEmptyOrder
	}
	if order.TotalPrice <= 0 {
		return 0, models.ErrInvalidTotalPrice
	}

	// Set default status if not provided
	if order.Status == "" {
		order.Status = "pending"
	}

	return s.orderRepo.CreateOrder(ctx, order)
}

func (s *orderService) GetOrder(ctx context.Context, id int) (models.Order, error) {
	if id <= 0 {
		return models.Order{}, models.ErrInvalidOrderID
	}
	return s.orderRepo.GetOrderByID(ctx, id)
}

func (s *orderService) ListOrders(ctx context.Context, filters models.OrderFilters) ([]models.Order, error) {
	// Validate date range if both are provided
	if !filters.StartDate.IsZero() && !filters.EndDate.IsZero() && filters.StartDate.After(filters.EndDate) {
		return nil, models.ErrInvalidDateRange
	}

	return s.orderRepo.GetAllOrders(ctx, filters)
}

func (s *orderService) UpdateOrder(ctx context.Context, id int, order models.Order) error {
	if id <= 0 {
		return models.ErrInvalidOrderID
	}
	if len(order.Items) == 0 {
		return models.ErrEmptyOrder
	}
	if order.TotalPrice <= 0 {
		return models.ErrInvalidTotalPrice
	}

	return s.orderRepo.UpdateOrder(ctx, id, order)
}

func (s *orderService) DeleteOrder(ctx context.Context, id int) error {
	if id <= 0 {
		return models.ErrInvalidOrderID
	}
	return s.orderRepo.DeleteOrder(ctx, id)
}

func (s *orderService) CloseOrder(ctx context.Context, id int) error {
	if id <= 0 {
		return models.ErrInvalidOrderID
	}
	return s.orderRepo.CloseOrder(ctx, id)
}

func (s *orderService) GetOrderedItemsReport(ctx context.Context, startDate, endDate string) (map[string]int, error) {
	return s.orderRepo.GetNumberOfOrderedItems(ctx, startDate, endDate)
}

func (s *orderService) ProcessBatchOrders(ctx context.Context, orders []models.Order) (models.BatchOrderResponse, error) {
	if len(orders) == 0 {
		return models.BatchOrderResponse{}, models.ErrEmptyBatch
	}

	// Validate each order in the batch
	for _, order := range orders {
		if len(order.Items) == 0 {
			return models.BatchOrderResponse{}, models.ErrEmptyOrder
		}
		if order.TotalPrice <= 0 {
			return models.BatchOrderResponse{}, models.ErrInvalidTotalPrice
		}
	}

	return s.orderRepo.BatchProcessOrders(ctx, orders)
}
