package service

import (
	"context"
	"frappuccino/internal/dal"
	"frappuccino/internal/models"
	"time"
)

type ReportService interface {
	GetTotalSales(ctx context.Context, startDate, endDate string) (*models.TotalSalesResponse, error)
	GetPopularItems(ctx context.Context, limit int) ([]models.PopularItem, error)
	GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) (*models.PeriodReportResponse, error)
	Search(ctx context.Context, query string, filter string) (*models.SearchResult, error)
}

type reportService struct {
	repo dal.ReportRepository
}

func NewReportService(repo dal.ReportRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) GetTotalSales(ctx context.Context, startDate, endDate string) (*models.TotalSalesResponse, error) {
	total, err := s.repo.GetTotalSales(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &models.TotalSalesResponse{
		TotalSales: total,
		StartDate:  startDate,
		EndDate:    endDate,
	}, nil
}

func (s *reportService) GetPopularItems(ctx context.Context, limit int) ([]models.PopularItem, error) {
	items, err := s.repo.GetPopularItems(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Calculate total quantity for percentage calculation
	totalQuantity := 0
	for _, item := range items {
		totalQuantity += item.TotalQuantity
	}

	// Add percentage to each item if there are items
	if totalQuantity > 0 {
		for i := range items {
			items[i].Percentage = float64(items[i].TotalQuantity) / float64(totalQuantity) * 100
		}
	}

	return items, nil
}

func (s *reportService) GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) (*models.PeriodReportResponse, error) {
	response, err := s.repo.GetOrderedItemsByPeriod(ctx, period, month, year)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *reportService) Search(ctx context.Context, query string, filter string) (*models.SearchResult, error) {
	if query == "" {
		return &models.SearchResult{}, nil
	}

	result, err := s.repo.GetFullTextSearch(ctx, query, filter)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
