package dal

import (
	"context"
	"database/sql"

	"frappuccino/internal/models"
)

type MenuRepository interface {
	CreateMenuItem(ctx context.Context, menuitem models.MenuItems) (int, error)
	GetAllMenu(ctx context.Context, id int) error
	GetMenuItemByID(ctx context.Context, id int) (models.MenuItems, error)
	UpdateMenuItem(ctx context.Context, id int, menuitem models.MenuItems) error
	DeleteMenuItem(ctx context.Context, id int) error
}

type menuRepository struct {
	*Repository
}

func NewMenuRepository(db *sql.DB) *menuRepository {
	return &menuRepository{NewRepository(db)}
}
