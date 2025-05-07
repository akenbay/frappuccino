package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"frappuccino/internal/models"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order models.Order) (int, error)
	GetOrderByID(ctx context.Context, id int) (models.Order, error)
	UpdateOrder(ctx context.Context, id int, order models.Order) error
	DeleteOrder(ctx context.Context, id int) error
	CloseOrder(ctx context.Context, id int) error
}

type orderRepository struct {
	*Repository
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{NewRepository(db)}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO orders (customer_id, payment_method, status, total_price, special_instructions) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		order.CustomerID, order.PaymentMethod, order.Status, order.TotalPrice, order.SpecialInstructions,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create order: %w", err)
	}

	// Insert order items
	for _, item := range order.Items {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, quantity, price_at_order, customizations)
			VALUES ($1, $2, $3, $4, $5)`,
			id, item.MenuItemID, item.Quantity, item.PriceAtOrder, item.Customizations,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to add order item: %w", err)
		}
	}

	return id, nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id int) (models.Order, error) {
	// Initialize empty order
	var order models.Order

	// 1. Get basic order info
	err := r.db.QueryRowContext(ctx, `
        SELECT 
            id, 
            customer_id, 
            status, 
            payment_method,
            total_price, 
            special_instructions, 
            created_at, 
            updated_at
        FROM orders 
        WHERE id = $1`, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.Status,
		&order.PaymentMethod,
		&order.TotalPrice,
		&order.SpecialInstructions,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Order{}, fmt.Errorf("order not found: %w", err)
		}
		return models.Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	// 2. Get order items
	rows, err := r.db.QueryContext(ctx, `
        SELECT 
            id,
            menu_item_id,
            quantity,
            price_at_order,
            customizations
        FROM order_items
        WHERE order_id = $1`, id)
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.MenuItemID,
			&item.Quantity,
			&item.PriceAtOrder,
			&item.Customizations,
		); err != nil {
			return models.Order{}, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return models.Order{}, fmt.Errorf("error after scanning order items: %w", err)
	}

	order.Items = items
	return order, nil
}
