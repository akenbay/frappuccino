package dal

import (
	"context"
	"database/sql"
	"fmt"
	"frappuccino/internal/models"
	"time"
)

type ReportRepository interface {
	GetTotalSales(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetPopularItems(ctx context.Context, limit int) ([]models.PopularItem, error)
	GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) ([]models.PeriodReport, error)
	GetFullTextSearch(ctx context.Context, query string, filter string) (models.SearchResult, error)
}

type reportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetTotalSales(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(total_price), 0)
		FROM orders
		WHERE created_at BETWEEN $1 AND $2
	`

	var totalSales float64
	err := r.db.QueryRowContext(ctx, query, startDate, endDate).Scan(&totalSales)
	if err != nil {
		return 0, fmt.Errorf("failed to get total sales: %w", err)
	}

	return totalSales, nil
}

func (r *reportRepository) GetPopularItems(ctx context.Context, limit int) ([]models.PopularItem, error) {
	query := `
		SELECT 
			mi.id,
			mi.name,
			COUNT(DISTINCT oi.order_id) as order_count,
			SUM(oi.quantity) as total_quantity
		FROM order_items oi
		JOIN menu_items mi ON oi.menu_item_id = mi.id
		GROUP BY mi.id, mi.name
		ORDER BY total_quantity DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular items: %w", err)
	}
	defer rows.Close()

	var popularItems []models.PopularItem
	for rows.Next() {
		var item models.PopularItem
		if err := rows.Scan(&item.MenuItemID, &item.Name, &item.OrderCount, &item.TotalQuantity); err != nil {
			return nil, fmt.Errorf("failed to scan popular item: %w", err)
		}
		popularItems = append(popularItems, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return popularItems, nil
}

func (r *reportRepository) GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) ([]models.PeriodReport, error) {
	var query string
	var args []interface{}

	switch period {
	case "day":
		query = `
			SELECT 
				EXTRACT(DAY FROM created_at)::int as period,
				COUNT(*) as order_count,
				COALESCE(SUM(total_price), 0) as total_sales
			FROM orders
			WHERE EXTRACT(MONTH FROM created_at) = $1
			AND EXTRACT(YEAR FROM created_at) = $2
			GROUP BY period
			ORDER BY period
		`
		args = []interface{}{month, year}
	case "month":
		query = `
			SELECT 
				EXTRACT(MONTH FROM created_at)::int as period,
				COUNT(*) as order_count,
				COALESCE(SUM(total_price), 0) as total_sales
			FROM orders
			WHERE EXTRACT(YEAR FROM created_at) = $1
			GROUP BY period
			ORDER BY period
		`
		args = []interface{}{year}
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get ordered items by period: %w", err)
	}
	defer rows.Close()

	var reports []models.PeriodReport
	for rows.Next() {
		var report models.PeriodReport
		if err := rows.Scan(&report.Period, &report.OrderCount, &report.TotalSales); err != nil {
			return nil, fmt.Errorf("failed to scan period report: %w", err)
		}
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return reports, nil
}

func (r *reportRepository) GetFullTextSearch(ctx context.Context, query string, filter string) (models.SearchResult, error) {
	result := models.SearchResult{}

	// Search menu items if filter includes "menu" or "all"
	if filter == "all" || filter == "menu" {
		menuQuery := `
			SELECT id, name, description, price, 
				   ts_rank(search_vector, plainto_tsquery('english', $1)) as relevance
			FROM menu_items
			WHERE search_vector @@ plainto_tsquery('english', $1)
			ORDER BY relevance DESC
			LIMIT 10
		`

		rows, err := r.db.QueryContext(ctx, menuQuery, query)
		if err != nil {
			return models.SearchResult{}, fmt.Errorf("failed to search menu items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item models.SearchMenuItem
			if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Relevance); err != nil {
				return models.SearchResult{}, fmt.Errorf("failed to scan menu item: %w", err)
			}
			result.MenuItems = append(result.MenuItems, item)
		}
	}

	// Search orders if filter includes "orders" or "all"
	if filter == "all" || filter == "orders" {
		orderQuery := `
			SELECT 
				o.id, 
				c.first_name || ' ' || c.last_name as customer_name,
				array_agg(mi.name) as items,
				o.total_price,
				o.status,
				ts_rank(
					setweight(to_tsvector('english', c.first_name || ' ' || c.last_name), 'A') ||
					setweight(to_tsvector('english', o.special_instructions::text), 'B'),
					plainto_tsquery('english', $1)
				) as relevance
			FROM orders o
			JOIN customers c ON o.customer_id = c.id
			JOIN order_items oi ON o.id = oi.order_id
			JOIN menu_items mi ON oi.menu_item_id = mi.id
			WHERE (
				to_tsvector('english', c.first_name || ' ' || c.last_name) @@ plainto_tsquery('english', $1) OR
				to_tsvector('english', o.special_instructions::text) @@ plainto_tsquery('english', $1)
			)
			GROUP BY o.id, c.first_name, c.last_name, o.total_price, o.status, o.special_instructions
			ORDER BY relevance DESC
			LIMIT 10
		`

		rows, err := r.db.QueryContext(ctx, orderQuery, query)
		if err != nil {
			return models.SearchResult{}, fmt.Errorf("failed to search orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var order models.SearchOrder
			var items []string
			if err := rows.Scan(&order.ID, &order.CustomerName, &items, &order.Total, &order.Status, &order.Relevance); err != nil {
				return models.SearchResult{}, fmt.Errorf("failed to scan order: %w", err)
			}
			order.Items = items
			result.Orders = append(result.Orders, order)
		}
	}

	result.Total = len(result.MenuItems) + len(result.Orders)
	return result, nil
}
