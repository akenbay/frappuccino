package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"frappuccino/internal/models"

	"github.com/lib/pq"
)

type MenuRepository interface {
	CreateMenuItem(ctx context.Context, menuitem models.MenuItems) (int, error)
	GetAllMenu(ctx context.Context) ([]models.MenuItems, error)
	GetMenuItemByID(ctx context.Context, id int) (models.MenuItems, error)
	UpdateMenuItem(ctx context.Context, id int, menuitem models.MenuItems) error
	DeleteMenuItem(ctx context.Context, id int) error
}

type menuRepository struct {
	*Repository
}

func NewMenuRepository(db *sql.DB) MenuRepository {
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

func (r *menuRepository) GetAllMenu(ctx context.Context) ([]models.MenuItems, error) {
	// Execute query
	rows, err := r.db.QueryContext(ctx, `
        SELECT id, name, description, price, category, is_active, created_at, updated_at
        FROM menu_items`)
	if err != nil {
		return nil, fmt.Errorf("failed to query menu items: %w", err)
	}
	defer rows.Close()

	var menuItems []models.MenuItems
	for rows.Next() {
		var item models.MenuItems
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			pq.Array(&item.Category),
			&item.IsActive,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan menu item: %w", err)
		}

		addrows, err := r.db.QueryContext(ctx, `
        	SELECT 
            	ingredient_id,
            	quantity
        	FROM menu_item_ingredients
        	WHERE menu_item_id = $1`, item.ID)
		if err != nil {
			return []models.MenuItems{}, fmt.Errorf("failed to get ingredients: %w", err)
		}
		defer addrows.Close()

		var ingredients []models.MenuItemIngredients
		for addrows.Next() {
			var ingredient models.MenuItemIngredients
			if err := addrows.Scan(
				&ingredient.IngredientID,
				&ingredient.Quantity,
			); err != nil {
				return []models.MenuItems{}, fmt.Errorf("failed to scan ingredient: %w", err)
			}
			ingredients = append(ingredients, ingredient)
		}

		if err = addrows.Err(); err != nil {
			return []models.MenuItems{}, fmt.Errorf("error after scanning order items: %w", err)
		}

		item.Ingredients = ingredients

		menuItems = append(menuItems, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning menu items: %w", err)
	}

	return menuItems, nil
}

func (r *menuRepository) GetMenuItemByID(ctx context.Context, id int) (models.MenuItems, error) {
	// Initialize empty order
	var menuitem models.MenuItems

	// 1. Get basic order info
	err := r.db.QueryRowContext(ctx, `
        SELECT 
            id, 
            name, 
            description, 
            price,
            category, 
            is_active, 
            created_at, 
            updated_at
        FROM menu_items 
        WHERE id = $1`, id).Scan(
		&menuitem.ID,
		&menuitem.Name,
		&menuitem.Description,
		&menuitem.Price,
		pq.Array(&menuitem.Category),
		&menuitem.IsActive,
		&menuitem.CreatedAt,
		&menuitem.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.MenuItems{}, fmt.Errorf("menu item not found: %w", err)
		}
		return models.MenuItems{}, fmt.Errorf("failed to get menu item: %w", err)
	}

	// 2. Get order items
	rows, err := r.db.QueryContext(ctx, `
        SELECT 
            ingredient_id,
            quantity
        FROM menu_item_ingredients
        WHERE menu_item_id = $1`, id)
	if err != nil {
		return models.MenuItems{}, fmt.Errorf("failed to get ingredients: %w", err)
	}
	defer rows.Close()

	var ingredients []models.MenuItemIngredients
	for rows.Next() {
		var ingredient models.MenuItemIngredients
		if err := rows.Scan(
			&ingredient.IngredientID,
			&ingredient.Quantity,
		); err != nil {
			return models.MenuItems{}, fmt.Errorf("failed to scan ingredient: %w", err)
		}
		ingredients = append(ingredients, ingredient)
	}

	if err = rows.Err(); err != nil {
		return models.MenuItems{}, fmt.Errorf("error after scanning order items: %w", err)
	}

	menuitem.Ingredients = ingredients
	return menuitem, nil
}

// UpdateMenuItem updates a menu item
func (r *menuRepository) UpdateMenuItem(ctx context.Context, id int, item models.MenuItems) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Record price change history if the price has changed
	var oldPrice float64
	err = r.db.QueryRowContext(ctx, `SELECT price FROM menu_items WHERE id = $1`, id).Scan(&oldPrice)
	if err != nil {
		return fmt.Errorf("failed to get old price: %v", err)
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE menu_items SET name = $1, description = $2, price = $3, category = $4, is_active = $5
		WHERE id = $6`,
		item.Name, item.Description, item.Price, pq.Array(item.Category), item.IsActive, id)
	if err != nil {
		return fmt.Errorf("failed update menu item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows")
	}
	if affected == 0 {
		return fmt.Errorf("menu item not found")
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM menu_item_ingredients WHERE menu_item_id = $1`, id)
	if err != nil {
		return fmt.Errorf("clear ingredients: %w", err)
	}

	for _, ing := range item.Ingredients {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO menu_item_ingredients (menu_item_id, ingredient_id, quantity)
			VALUES ($1, $2, $3)`,
			id, ing.IngredientID, ing.Quantity)
		if err != nil {
			return fmt.Errorf("insert new ingredients: %w", err)
		}
	}

	if oldPrice != item.Price {
		_, err = r.db.Exec(`
            INSERT INTO price_history (menu_item_id, old_price, new_price, changed_at)
            VALUES ($1, $2, $3, NOW())`,
			id, oldPrice, item.Price)
		if err != nil {
			return fmt.Errorf("failed to log price history: %v", err)
		}
	}

	return tx.Commit()
}

// DeleteMenuItem deletes a menu item if it’s not used in any orders
func (r *menuRepository) DeleteMenuItem(ctx context.Context, id int) error {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM order_items WHERE menu_item_id = $1`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("check order usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot delete menu item in use")
	}

	_, err = r.db.ExecContext(ctx, `DELETE FROM menu_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete menu item: %w", err)
	}
	return nil
}
