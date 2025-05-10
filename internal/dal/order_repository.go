package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"frappuccino/internal/models"
	"strings"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order models.Order) (int, error)
	GetOrderByID(ctx context.Context, id int) (models.Order, error)
	GetAllOrders(ctx context.Context, filters models.OrderFilters) ([]models.Order, error)
	UpdateOrder(ctx context.Context, id int, order models.Order) error
	DeleteOrder(ctx context.Context, id int) error
	CloseOrder(ctx context.Context, id int) error
	GetNumberOfOrderedItems(ctx context.Context, startDate, endDate string) (map[string]int, error)
	BatchProcessOrders(ctx context.Context, orders []models.Order) (models.BatchOrderResponse, error)
}

type orderRepository struct {
	*Repository
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{NewRepository(db)}
}

func (r *orderRepository) CreateOrder(ctx context.Context, order models.Order) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Check inventory availability first
	for _, item := range order.Items {
		var sufficient bool
		err := tx.QueryRowContext(ctx, `
            SELECT (i.quantity >= (mi.quantity * $1)) 
            FROM menu_item_ingredients mi
            JOIN inventory i ON mi.ingredient_id = i.id
            WHERE mi.menu_item_id = $2`,
			item.Quantity, item.MenuItemID,
		).Scan(&sufficient)

		if err != nil || !sufficient {
			return 0, fmt.Errorf("insufficient inventory for menu item %d: %w",
				item.MenuItemID, err)
		}
	}

	// Calculate total price based on items
	totalPrice, err := r.calculateOrderTotal(ctx, tx, order.Items)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate order total: %w", err)
	}
	order.TotalPrice = totalPrice

	// 2. Insert order

	var id int
	var special_instructions interface{} = nil
	if len(order.SpecialInstructions) > 0 {
		special_instructions = order.SpecialInstructions
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO orders (customer_id, payment_method, total_price, special_instructions) 
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		order.CustomerID, order.PaymentMethod, order.TotalPrice, special_instructions,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create order: %w", err)
	}

	// 3. Insert order items
	for _, item := range order.Items {
		var customizations interface{} = nil
		if len(item.Customizations) > 0 {
			customizations = item.Customizations
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, quantity, price_at_order, customizations)
			VALUES ($1, $2, $3, $4, $5)`,
			id, item.MenuItemID, item.Quantity, item.PriceAtOrder, customizations,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to add order item: %w", err)
		}
	}

	// 4. Deduct inventory
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
            WITH ingredients AS (
                SELECT ingredient_id, quantity 
                FROM menu_item_ingredients 
                WHERE menu_item_id = $1
            )
            UPDATE inventory i
            SET quantity = i.quantity - (ing.quantity * $2)
            FROM ingredients ing
            WHERE i.id = ing.ingredient_id`,
			item.MenuItemID, item.Quantity,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to deduct ingredient from inventory: %w", err)
		}
	}

	// 5. Record inventory transactions
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
            WITH ingredients AS (
                SELECT ingredient_id, quantity 
                FROM menu_item_ingredients 
                WHERE menu_item_id = $1
            )
            INSERT INTO inventory_transactions
                (ingredient_id, delta, transaction_type, reference_id)
            SELECT 
                ingredient_id, 
                -(quantity * $2), 
                'order_usage', 
                $3
            FROM ingredients`,
			item.MenuItemID, item.Quantity, id,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to record inventory transaction: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id int) (models.Order, error) {
	// Initialize empty order
	var order models.Order

	// 1. Get basic order info
	var specialInstructions sql.NullString
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
		&specialInstructions,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Order{}, fmt.Errorf("order not found: %w", err)
		}
		return models.Order{}, fmt.Errorf("failed to get order: %w", err)
	}

	if specialInstructions.Valid {
		order.SpecialInstructions = json.RawMessage(specialInstructions.String)
	} else {
		order.SpecialInstructions = nil
	}

	// 2. Get order items
	rows, err := r.db.QueryContext(ctx, `
        SELECT 
            id,
            menu_item_id,
            quantity,
            price_at_order,
            customizations,
			order_id
        FROM order_items
        WHERE order_id = $1`, id)
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to get order items: %w", err)
	}
	defer rows.Close()

	var customizations sql.NullString
	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.MenuItemID,
			&item.Quantity,
			&item.PriceAtOrder,
			&customizations,
			&item.OrderID,
		); err != nil {
			return models.Order{}, fmt.Errorf("failed to scan order item: %w", err)
		}

		if customizations.Valid {
			item.Customizations = json.RawMessage(customizations.String)
		} else {
			item.Customizations = nil
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return models.Order{}, fmt.Errorf("error after scanning order items: %w", err)
	}

	order.Items = items
	return order, nil
}

func (r *orderRepository) UpdateOrder(ctx context.Context, id int, updatedOrder models.Order) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Calculate new total price
	totalPrice, err := r.calculateOrderTotal(ctx, tx, updatedOrder.Items)
	if err != nil {
		return fmt.Errorf("failed to calculate order total: %w", err)
	}
	updatedOrder.TotalPrice = totalPrice

	// 1. Get current order items (to calculate inventory delta)
	var currentItems []struct {
		MenuItemID int
		Quantity   int
	}
	rows, err := tx.QueryContext(ctx, `
        SELECT menu_item_id, quantity 
        FROM order_items 
        WHERE order_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to get current items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item struct{ MenuItemID, Quantity int }
		if err := rows.Scan(&item.MenuItemID, &item.Quantity); err != nil {
			return fmt.Errorf("failed to scan current item: %w", err)
		}
		currentItems = append(currentItems, item)
	}

	// 2. Calculate net inventory changes
	inventoryDeltas := make(map[int]int) // ingredient_id → delta
	for _, currItem := range currentItems {
		// Subtract old quantities
		ingredientRows, err := tx.QueryContext(ctx, `
            SELECT ingredient_id, quantity 
            FROM menu_item_ingredients 
            WHERE menu_item_id = $1`, currItem.MenuItemID)
		if err != nil {
			return fmt.Errorf("failed to get ingredients for menu item %d: %w", currItem.MenuItemID, err)
		}

		for ingredientRows.Next() {
			var ingredientID int
			var quantityPerUnit float64
			if err := ingredientRows.Scan(&ingredientID, &quantityPerUnit); err != nil {
				return fmt.Errorf("failed to scan ingredient: %w", err)
			}
			inventoryDeltas[ingredientID] -= int(quantityPerUnit * float64(currItem.Quantity))
		}
		ingredientRows.Close()
	}

	for _, newItem := range updatedOrder.Items {
		// Add new quantities
		ingredientRows, err := tx.QueryContext(ctx, `
            SELECT ingredient_id, quantity 
            FROM menu_item_ingredients 
            WHERE menu_item_id = $1`, newItem.MenuItemID)
		if err != nil {
			return fmt.Errorf("failed to get ingredients for menu item %d: %w", newItem.MenuItemID, err)
		}

		for ingredientRows.Next() {
			var ingredientID int
			var quantityPerUnit float64
			if err := ingredientRows.Scan(&ingredientID, &quantityPerUnit); err != nil {
				return fmt.Errorf("failed to scan ingredient: %w", err)
			}
			inventoryDeltas[ingredientID] += int(quantityPerUnit * float64(newItem.Quantity))
		}
		ingredientRows.Close()
	}

	// 3. Verify inventory availability (for positive deltas)
	for ingredientID, delta := range inventoryDeltas {
		if delta > 0 { // Only check for new usage (not restocks)
			var currentStock int
			err := tx.QueryRowContext(ctx, `
                SELECT quantity FROM inventory 
                WHERE id = $1 FOR UPDATE`, ingredientID).Scan(&currentStock)
			if err != nil {
				return fmt.Errorf("failed to check inventory for ingredient %d: %w", ingredientID, err)
			}

			if currentStock < delta {
				return fmt.Errorf("insufficient stock for ingredient %d (need %d, have %d)",
					ingredientID, delta, currentStock)
			}
		}
	}

	// 4. Update inventory
	for ingredientID, delta := range inventoryDeltas {
		if delta != 0 { // Skip if no net change
			_, err := tx.ExecContext(ctx, `
                UPDATE inventory 
                SET quantity = quantity + $1 
                WHERE id = $2`, -delta, ingredientID)
			if err != nil {
				return fmt.Errorf("failed to update inventory for ingredient %d: %w", ingredientID, err)
			}

			// Record transaction
			_, err = tx.ExecContext(ctx, `
                INSERT INTO inventory_transactions (
                    ingredient_id, delta, transaction_type, reference_id
                ) VALUES (
                    $1, $2, 'order_update', $3
                )`, ingredientID, -delta, id)
			if err != nil {
				return fmt.Errorf("failed to record inventory transaction: %w", err)
			}
		}
	}

	// 5. Update order metadata
	var special_instructions interface{} = nil
	if len(updatedOrder.SpecialInstructions) > 0 {
		special_instructions = updatedOrder.SpecialInstructions
	}
	result, err := tx.ExecContext(ctx, `
        UPDATE orders 
        SET 
            customer_id = $1,
            status = $2,
            payment_method = $3,
            total_price = $4,
            special_instructions = $5,
            updated_at = NOW()
        WHERE id = $6`,
		updatedOrder.CustomerID,
		updatedOrder.Status,
		updatedOrder.PaymentMethod,
		updatedOrder.TotalPrice,
		special_instructions,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	// Verify exactly one row was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// 6. Delete existing order items
	_, err = tx.ExecContext(ctx, `
        DELETE FROM order_items 
        WHERE order_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to clear order items: %w", err)
	}

	// 7. Insert new order items
	for _, item := range updatedOrder.Items {
		var customizations interface{} = nil
		if len(item.Customizations) > 0 {
			customizations = item.Customizations
		}
		_, err = tx.ExecContext(ctx, `
            INSERT INTO order_items (
                order_id, 
                menu_item_id, 
                quantity, 
                price_at_order, 
                customizations
            ) VALUES ($1, $2, $3, $4, $5)`,
			id,
			item.MenuItemID,
			item.Quantity,
			item.PriceAtOrder,
			customizations,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) DeleteOrder(ctx context.Context, id int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Get all items first to restore inventory
	var items []struct {
		MenuItemID int
		Quantity   int
	}
	rows, err := tx.QueryContext(ctx, `
        SELECT menu_item_id, quantity 
        FROM order_items 
        WHERE order_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to get all items from deleting order: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item struct{ MenuItemID, Quantity int }
		if err := rows.Scan(&item.MenuItemID, &item.Quantity); err != nil {
			return fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	// 2. Restore inventory
	for _, item := range items {
		_, err = tx.ExecContext(ctx, `
            WITH ingredients AS (
                SELECT ingredient_id, quantity 
                FROM menu_item_ingredients 
                WHERE menu_item_id = $1
            )
            UPDATE inventory i
            SET quantity = i.quantity + (ing.quantity * $2)
            FROM ingredients ing
            WHERE i.id = ing.ingredient_id`,
			item.MenuItemID, item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to restore inventory: %w", err)
		}
	}

	// 3. Record inventory transactions (for restoring stock)
	for _, item := range items {
		_, err = tx.ExecContext(ctx, `
            WITH ingredients AS (
                SELECT 
                    ingredient_id, 
                    quantity AS required_quantity
                FROM menu_item_ingredients 
                WHERE menu_item_id = $1
            )
            INSERT INTO inventory_transactions (
                ingredient_id, 
                delta, 
                transaction_type, 
                reference_id,
                notes
            )
            SELECT 
                ingredient_id,
                (required_quantity * $2::numeric),  -- Explicit cast
                'order_deletion',
                $3::integer,                        -- Explicit cast
                CONCAT('Restored from cancelled order #', $3::integer, ' for menu item #', $1::integer)
            FROM ingredients`,
			item.MenuItemID,
			item.Quantity,
			id,
		)
		if err != nil {
			return fmt.Errorf("failed to record inventory restoration for menu item %d: %w",
				item.MenuItemID, err)
		}
	}

	// 4. Delete order items
	if _, err = tx.ExecContext(ctx, `DELETE FROM order_items WHERE order_id = $1`, id); err != nil {
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	// 5. Delete the order
	result, err := tx.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (r *orderRepository) CloseOrder(ctx context.Context, id int) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify order exists and is in a closable state
	var currentStatus string
	err = tx.QueryRowContext(ctx, `
        SELECT status FROM orders 
        WHERE id = $1 FOR UPDATE`, id).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("order not found: %w", err)
		}
		return fmt.Errorf("failed to check order status: %w", err)
	}

	// Validate order can be closed
	if currentStatus == "cancelled" {
		return fmt.Errorf("cannot close already cancelled order")
	}
	if currentStatus == "delivered" {
		return fmt.Errorf("order already closed")
	}

	// 2. Update order status to "delivered"
	result, err := tx.ExecContext(ctx, `
        UPDATE orders 
        SET status = 'delivered', 
            updated_at = NOW() 
        WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Verify exactly one row was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// 3. Record status change in history
	_, err = tx.ExecContext(ctx, `
        INSERT INTO order_status_history (order_id, status) 
        VALUES ($1, 'delivered')`, id)
	if err != nil {
		return fmt.Errorf("failed to record status change: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) GetAllOrders(ctx context.Context, filters models.OrderFilters) ([]models.Order, error) {
	// Build base query
	query := `
        SELECT 
            o.id,
            o.customer_id,
            o.status,
            o.payment_method,
            o.total_price,
            o.special_instructions,
            o.created_at,
            o.updated_at,
            COALESCE(
                json_agg(
                    json_build_object(
                        'id', oi.id,
                        'menu_item_id', oi.menu_item_id,
                        'quantity', oi.quantity,
                        'price_at_order', oi.price_at_order,
                        'customizations', oi.customizations
                    )
                ) FILTER (WHERE oi.id IS NOT NULL),
                '[]'
            ) AS items
        FROM orders o
        LEFT JOIN order_items oi ON o.id = oi.order_id
    `

	// Add filters (status, date range, etc.)
	var args []interface{}
	whereClauses := []string{}

	if filters.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("o.status = $%d", len(args)+1))
		args = append(args, filters.Status)
	}

	if !filters.StartDate.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("o.created_at >= $%d", len(args)+1))
		args = append(args, filters.StartDate)
	}

	if !filters.EndDate.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("o.created_at <= $%d", len(args)+1))
		args = append(args, filters.EndDate)
	}

	// Combine WHERE clauses
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Group and order
	query += `
        GROUP BY o.id
        ORDER BY o.created_at DESC
    `

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	var specialInstructions sql.NullString
	for rows.Next() {
		var order models.Order
		var itemsJSON []byte

		err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&order.Status,
			&order.PaymentMethod,
			&order.TotalPrice,
			&specialInstructions,
			&order.CreatedAt,
			&order.UpdatedAt,
			&itemsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if specialInstructions.Valid {
			order.SpecialInstructions = json.RawMessage(specialInstructions.String)
		} else {
			order.SpecialInstructions = nil
		}

		// Unmarshal JSON items
		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal order items: %w", err)
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after scanning orders: %w", err)
	}

	return orders, nil
}

func (r *orderRepository) GetNumberOfOrderedItems(ctx context.Context, startDate, endDate string) (map[string]int, error) {
	query := `
		SELECT mi.name, SUM(oi.quantity) as total_quantity
		FROM order_items oi
		JOIN menu_items mi ON oi.menu_item_id = mi.id
		JOIN orders o ON oi.order_id = o.id
	`

	if startDate != "" && endDate != "" {
		query += `
		WHERE o.created_at BETWEEN $1 AND $2
	`
	} else if startDate == "" && endDate != "" {
	}

	query += `
		GROUP BY mi.name
		ORDER BY total_quantity DESC
	`

	rows, err := r.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query ordered items: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var name string
		var quantity int
		if err := rows.Scan(&name, &quantity); err != nil {
			return nil, fmt.Errorf("failed to scan ordered item: %w", err)
		}
		result[name] = quantity
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

func (r *orderRepository) BatchProcessOrders(ctx context.Context, orders []models.Order) (models.BatchOrderResponse, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.BatchOrderResponse{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	response := models.BatchOrderResponse{
		ProcessedOrders: make([]models.ProcessedOrder, 0, len(orders)),
		Summary: models.BatchSummary{
			InventoryUsed: make([]models.InventoryUsage, 0),
		},
	}

	inventoryUpdates := make(map[int]float64) // ingredientID -> quantity used

	for _, order := range orders {
		processed := models.ProcessedOrder{
			CustomerName: fmt.Sprintf("Customer %d", order.CustomerID),
			Total:        order.TotalPrice,
		}

		// Check inventory availability first
		canFulfill, rejectReason := r.checkInventory(ctx, tx, order.Items, inventoryUpdates)
		if !canFulfill {
			processed.Status = "rejected"
			processed.Rejected = true
			processed.RejectReason = rejectReason
			response.Summary.Rejected++
			response.ProcessedOrders = append(response.ProcessedOrders, processed)
			continue
		}

		// Process the order
		orderID, err := r.CreateOrder(ctx, order)
		if err != nil {
			return models.BatchOrderResponse{}, fmt.Errorf("failed to create order: %w", err)
		}

		processed.OrderID = orderID
		processed.Status = "accepted"
		response.Summary.Accepted++
		response.ProcessedOrders = append(response.ProcessedOrders, processed)
		response.Summary.TotalRevenue += order.TotalPrice
	}

	// Prepare inventory usage report
	for id, qty := range inventoryUpdates {
		var name string
		var remaining float64
		err := tx.QueryRowContext(ctx, `
			SELECT name, quantity FROM inventory WHERE id = $1`, id).Scan(&name, &remaining)
		if err != nil {
			return models.BatchOrderResponse{}, fmt.Errorf("failed to get inventory info: %w", err)
		}

		response.Summary.InventoryUsed = append(response.Summary.InventoryUsed, models.InventoryUsage{
			IngredientID:   id,
			Name:           name,
			QuantityUsed:   qty,
			RemainingStock: remaining,
		})
	}

	response.Summary.TotalOrders = len(orders)

	if err := tx.Commit(); err != nil {
		return models.BatchOrderResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return response, nil
}

// Helper functions
func (r *orderRepository) checkInventory(ctx context.Context, tx *sql.Tx, items []models.OrderItem, inventoryUpdates map[int]float64) (bool, string) {
	for _, item := range items {
		// Get required ingredients for this menu item
		rows, err := tx.QueryContext(ctx, `
			SELECT ingredient_id, quantity 
			FROM menu_item_ingredients 
			WHERE menu_item_id = $1`, item.MenuItemID)
		if err != nil {
			return false, "inventory check failed"
		}

		for rows.Next() {
			var ingredientID int
			var requiredQty float64
			if err := rows.Scan(&ingredientID, &requiredQty); err != nil {
				return false, "inventory check failed"
			}

			totalNeeded := requiredQty * float64(item.Quantity)
			var currentStock float64
			err := tx.QueryRowContext(ctx, `
				SELECT quantity FROM inventory WHERE id = $1`, ingredientID).Scan(&currentStock)
			if err != nil {
				return false, "ingredient not found"
			}

			// Account for previously used inventory in this batch
			alreadyUsed := inventoryUpdates[ingredientID]
			if currentStock-alreadyUsed < totalNeeded {
				return false, fmt.Sprintf("insufficient stock for ingredient %d", ingredientID)
			}

			inventoryUpdates[ingredientID] = alreadyUsed + totalNeeded
		}
	}
	return true, ""
}

func (r *orderRepository) calculateOrderTotal(ctx context.Context, tx *sql.Tx, items []models.OrderItem) (float64, error) {
	var total float64

	for _, item := range items {
		// Get current price of the menu item
		var price float64
		err := tx.QueryRowContext(ctx, `
            SELECT price FROM menu_items 
            WHERE id = $1`, item.MenuItemID).Scan(&price)
		if err != nil {
			return 0, fmt.Errorf("failed to get price for menu item %d: %w", item.MenuItemID, err)
		}

		// Add to total
		total += price * float64(item.Quantity)
	}

	return total, nil
}
