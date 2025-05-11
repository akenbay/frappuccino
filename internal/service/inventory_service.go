package service

import (
	"context"
	"frappuccino/internal/dal"
	"frappuccino/internal/models"
)

type InventoryService interface {
	CreateIngredient(ctx context.Context, ingredient models.Inventory) (int, error)
	GetIngredient(ctx context.Context, id int) (models.Inventory, error)
	ListIngredients(ctx context.Context) ([]models.Inventory, error)
	UpdateIngredient(ctx context.Context, id int, ingredient models.Inventory) error
	DeleteIngredient(ctx context.Context, id int) error
	GetLeftOversWithPagination(ctx context.Context, sortBy string, page int, pageSize int) (models.PaginatedInventoryResponse, error)
}

type inventoryService struct {
	inventoryRepo dal.InventoryRepository
}

func NewInventoryService(inventoryRepo dal.InventoryRepository) InventoryService {
	return &inventoryService{inventoryRepo: inventoryRepo}
}

func (s *inventoryService) CreateIngredient(ctx context.Context, ingredient models.Inventory) (int, error) {
	if ingredient.Quantity < 0 {
		return 0, models.ErrInvalidQuantity
	}
	if ingredient.CostPerUnit < 0 {
		return 0, models.ErrInvalidCostPerUnit
	}
	if ingredient.ReOrderLevel < 0 {
		return 0, models.ErrInvalidReOrderLevel
	}
	return s.inventoryRepo.CreateIngredient(ctx, ingredient)
}

func (s *inventoryService) GetIngredient(ctx context.Context, id int) (models.Inventory, error) {
	if id <= 0 {
		return models.Inventory{}, models.ErrInvalidOrderID
	}
	return s.inventoryRepo.GetIngredientByID(ctx, id)
}

func (s *inventoryService) ListIngredients(ctx context.Context) ([]models.Inventory, error) {
	return s.inventoryRepo.GetAllIngredients(ctx)
}

func (s *inventoryService) UpdateIngredient(ctx context.Context, id int, ingredient models.Inventory) error {
	if id <= 0 {
		return models.ErrInvalidOrderID
	}
	if ingredient.Quantity < 0 {
		return models.ErrInvalidQuantity
	}
	if ingredient.CostPerUnit < 0 {
		return models.ErrInvalidCostPerUnit
	}
	if ingredient.ReOrderLevel < 0 {
		return models.ErrInvalidReOrderLevel
	}
	return s.inventoryRepo.UpdateIngredient(ctx, id, ingredient)
}

func (s *inventoryService) DeleteIngredient(ctx context.Context, id int) error {
	if id <= 0 {
		return models.ErrInvalidOrderID
	}
	return s.inventoryRepo.DeleteIngredient(ctx, id)
}

func (s *inventoryService) GetLeftOversWithPagination(ctx context.Context, sortBy string, page int, pageSize int) (models.PaginatedInventoryResponse, error) {
	if !(sortBy == "price" || sortBy == "quantity") {
		return models.PaginatedInventoryResponse{}, models.ErrInvalidSortByValue
	}
	if pageSize <= 0 {
		return models.PaginatedInventoryResponse{}, models.ErrInvalidPageSize
	}
	if page <= 0 {
		return models.PaginatedInventoryResponse{}, models.ErrInvalidPage
	}
	return s.inventoryRepo.GetLeftOversWithPagination(ctx, sortBy, page, pageSize)
}
