package dal

import (
	"context"
	"database/sql"
	"fmt"

	"frappuccino/internal/models"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
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

func (r *menuRepository) CreateMenuItem(ctx context.Context, menuitem models.MenuItems) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert menuitem
	var id int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO menu_items (name, description, price, category) 
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		menuitem.Name, menuitem.Description, menuitem.Price, pq.Array(menuitem.Category),
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create menu item: %w", err)
	}

	// Insert menuitem ingredients
	for _, ingredient := range menuitem.Ingredients {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO menu_item_ingredients (menu_item_id, ingredient_id, quantity)
			VALUES ($1, $2, $3)`,
			id, ingredient.IngredientID, ingredient.Quantity,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to add order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}
