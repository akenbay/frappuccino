package dal

import (
	"context"
	"database/sql"
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
		INSERT INTO orders (customer_id, status, total_price, special_instructions) 
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		order.CustomerID, order.Status, order.TotalPrice, order.SpecialInstructions,
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

// Implement other methods (GetOrderByID, UpdateOrder, etc.) similarly...
