package service

import (
	"context"

	"frappuccino/internal/dal"
	"frappuccino/internal/models"
)

type MenuService interface {
	GetAllMenu(ctx context.Context) ([]models.MenuItems, error)
	GetMenuItemByID(ctx context.Context, id int) (models.MenuItems, error)
	CreateMenuItem(ctx context.Context, item models.MenuItems) (int, error)
	UpdateMenuItem(ctx context.Context, id int, item models.MenuItems) error
	DeleteMenuItem(ctx context.Context, id int) error
}

type menuService struct {
	menuRepo dal.MenuRepository
}

func NewMenuService(menuRepo dal.MenuRepository) MenuService {
	return &menuService{menuRepo: menuRepo}
}

func (s *menuService) GetAllMenu(ctx context.Context) ([]models.MenuItems, error) {
	return s.menuRepo.GetAllMenu(ctx)
}

func (s *menuService) GetMenuItemByID(ctx context.Context, id int) (models.MenuItems, error) {
	if id <= 0 {
		return models.MenuItems{}, models.ErrInvalidMenuItemID
	}
	return s.menuRepo.GetMenuItemByID(ctx, id)
}

func (s *menuService) CreateMenuItem(ctx context.Context, item models.MenuItems) (int, error) {
	if item.Name == "" {
		return 0, models.ErrInvalidMenuItemName
	}
	if item.Price <= 0 {
		return 0, models.ErrInvalidMenuItemPrice
	}
	return s.menuRepo.CreateMenuItem(ctx, item)
}

func (s *menuService) UpdateMenuItem(ctx context.Context, id int, item models.MenuItems) error {
	if id <= 0 {
		return models.ErrInvalidMenuItemID
	}
	if item.Name == "" {
		return models.ErrInvalidMenuItemName
	}
	if item.Price <= 0 {
		return models.ErrInvalidMenuItemPrice
	}
	return s.menuRepo.UpdateMenuItem(ctx, id, item)
}

func (s *menuService) DeleteMenuItem(ctx context.Context, id int) error {
	if id <= 0 {
		return models.ErrInvalidMenuItemID
	}
	return s.menuRepo.DeleteMenuItem(ctx, id)
}
