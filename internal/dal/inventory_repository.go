package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"frappuccino/internal/models"
)

type InventoryRepository interface {
	CreateIngredient(ctx context.Context, ingredient models.Inventory) (int, error)
	GetAllIngredients(ctx context.Context, id int) error
	GetIngredientByID(ctx context.Context, id int) (models.Inventory, error)
	UpdateIngredient(ctx context.Context, id int, ingredient models.Inventory) error
	DeleteIngredient(ctx context.Context, id int) error
}

type inventoryRepository struct {
	*Repository
}

func NewInventoryRepository(db *sql.DB) *inventoryRepository {
	return &inventoryRepository{NewRepository(db)}
}

func (r *inventoryRepository) AddIngredient(ctx context.Context, ingredient models.Inventory) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO  (name, quantity, unit, cost_per_unit, reorder_level, supplier_info) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		ingredient.Name, ingredient.Quantity, ingredient.Unit, ingredient.CostPerUnit, ingredient.ReOrderLevel, ingredient.SupplierInfo,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create ingredient: %w", err)
	}

	return id, nil
}

func (r *inventoryRepository) GetAllIngredients(ctx context.Context) ([]models.Inventory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT 
			id,
            name,
            quantity,
            unit,
            reorder_level,
            supplier_info,
            created_at, 
            updated_at
		FROM inventory`)
	if err != nil {
		return nil, fmt.Errorf("failed to query inventory: %w", err)
	}
	defer rows.Close()

	var inventory []models.Inventory
	for rows.Next() {
		var ingredient models.Inventory
		err := rows.Scan(&ingredient.ID, &ingredient.Name, &ingredient.Quantity, &ingredient.Unit, &ingredient.ReOrderLevel, &ingredient.SupplierInfo, &ingredient.CreatedAt, &ingredient.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ingredient: %w", err)
		}
		inventory = append(inventory, ingredient)
	}
	return inventory, nil
}

func (r *inventoryRepository) GetIngredientByID(ctx context.Context, id int) (models.Inventory, error) {
	// Initialize empty ingredient
	var ingredient models.Inventory

	err := r.db.QueryRowContext(ctx, `
        SELECT 
            id,
            name,
            quantity,
            unit,
            reorder_level,
            supplier_info,
            created_at, 
            updated_at
        FROM inventory 
        WHERE id = $1`, id).Scan(
		&ingredient.ID,
		&ingredient.Name,
		&ingredient.Quantity,
		&ingredient.Unit,
		&ingredient.CostPerUnit,
		&ingredient.ReOrderLevel,
		&ingredient.SupplierInfo,
		&ingredient.CreatedAt,
		&ingredient.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Inventory{}, fmt.Errorf("ingredient not found: %w", err)
		}
		return models.Inventory{}, fmt.Errorf("failed to get ingredient: %w", err)
	}

	return ingredient, nil
}

func (r *orderRepository) UpdateIngredient(ctx context.Context, id int, ingredient models.Inventory) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update ingredient metadata
	result, err := tx.ExecContext(ctx, `
        UPDATE inventory 
        SET 
            name = $1,
            quantity = $2,
            unit = $3,
            cost_per_unit = $4,
            reorder_level = $5,
			supplier_info = $6,
            updated_at = NOW()
        WHERE id = $7`,
		ingredient.Name,
		ingredient.Quantity,
		ingredient.Unit,
		ingredient.CostPerUnit,
		ingredient.ReOrderLevel,
		ingredient.SupplierInfo,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update ingredient: %w", err)
	}

	// Verify exactly one row was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) DeleteIngredient(ctx context.Context, id int) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe rollback if error occurs

	// Delete the ingredient
	result, err := tx.ExecContext(ctx, `
        DELETE FROM inventory 
        WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete ingredient: %w", err)
	}

	// Verify exactly one row was deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Commit transaction if everything succeeded
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
