package dal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"frappuccino/internal/models"

	"github.com/lib/pq"
)

type ReportRepository interface {
	GetTotalSales(ctx context.Context, startDate, endDate string) (float64, error)
	GetPopularItems(ctx context.Context, limit int) ([]models.PopularItem, error)
	GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) (models.PeriodReportResponse, error)
	GetFullTextSearch(ctx context.Context, query string, filter string, minPrice, maxPrice float64) (models.SearchResult, error)
}

type reportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetTotalSales(ctx context.Context, startDate, endDate string) (float64, error) {
	query := `
        SELECT COALESCE(SUM(total_price), 0)
        FROM orders
    `

	var args []interface{}
	var whereClauses []string

	// Handle start date if provided
	if startDate != "" {
		whereClauses = append(whereClauses, "created_at >= $1")
		args = append(args, startDate)
	}

	// Handle end date if provided
	if endDate != "" {
		pos := len(args) + 1
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", pos))
		args = append(args, endDate)
	}

	// Add WHERE clause if we have any conditions
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var totalSales float64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&totalSales)
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

func (r *reportRepository) GetOrderedItemsByPeriod(ctx context.Context, period string, month time.Month, year int) (models.PeriodReportResponse, error) {
	var query string
	var args []interface{}
	response := models.PeriodReportResponse{
		PeriodType: period,
		Year:       year,
	}

	switch period {
	case "day":
		response.Month = month.String()
		query = `
            SELECT 
                EXTRACT(DAY FROM created_at)::int as day,
                COUNT(*) as order_count,
                COALESCE(SUM(total_price), 0) as total_sales
            FROM orders
            WHERE EXTRACT(MONTH FROM created_at) = $1
            AND EXTRACT(YEAR FROM created_at) = $2
            GROUP BY day
            ORDER BY day
        `
		args = []interface{}{month, year}
	case "month":
		query = `
            SELECT 
                TO_CHAR(created_at, 'Month') as month_name,
                COUNT(*) as order_count,
                COALESCE(SUM(total_price), 0) as total_sales
            FROM orders
            WHERE EXTRACT(YEAR FROM created_at) = $1
            GROUP BY month_name, EXTRACT(MONTH FROM created_at)
            ORDER BY EXTRACT(MONTH FROM created_at)
        `
		args = []interface{}{year}
	default:
		return models.PeriodReportResponse{}, fmt.Errorf("invalid period: %s", period)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return models.PeriodReportResponse{}, fmt.Errorf("failed to get ordered items by period: %w", err)
	}
	defer rows.Close()

	var reports []models.PeriodReport
	for rows.Next() {
		var report models.PeriodReport
		if period == "day" {
			var day int
			if err := rows.Scan(&day, &report.OrderCount, &report.TotalSales); err != nil {
				return models.PeriodReportResponse{}, fmt.Errorf("failed to scan day report: %w", err)
			}
			report.Period = day
		} else {
			var monthName string
			if err := rows.Scan(&monthName, &report.OrderCount, &report.TotalSales); err != nil {
				return models.PeriodReportResponse{}, fmt.Errorf("failed to scan month report: %w", err)
			}
			report.Period = monthName
		}
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return models.PeriodReportResponse{}, fmt.Errorf("rows error: %w", err)
	}

	response.Reports = reports
	return response, nil
}

func (r *reportRepository) GetFullTextSearch(ctx context.Context, query string, filter string, minPrice, maxPrice float64) (models.SearchResult, error) {
	result := models.SearchResult{}

	// Validate empty query
	if query == "" {
		return result, nil
	}

	// Set default filter if empty
	if filter == "" || filter == "orders,menu" || filter == "menu,orders" {
		filter = "all"
	}

	// Validate filter
	validFilters := map[string]bool{
		"all":    true,
		"menu":   true,
		"orders": true,
	}
	if !validFilters[filter] {
		return models.SearchResult{}, fmt.Errorf("invalid filter value: %s", filter)
	}

	// Search menu items if filter includes "menu" or "all"
	if filter == "all" || filter == "menu" {
		menuQuery := `
            SELECT id, name, description, price, 
                   ts_rank(search_vector, plainto_tsquery('english', $1)) as relevance
            FROM menu_items
            WHERE search_vector @@ plainto_tsquery('english', $1)
            AND ($2 = 0 OR price >= $2)
            AND ($3 = 0 OR price <= $3)
            ORDER BY relevance DESC
            LIMIT 10
        `

		rows, err := r.db.QueryContext(ctx, menuQuery, query, minPrice, maxPrice)
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
		if err = rows.Err(); err != nil {
			return models.SearchResult{}, fmt.Errorf("error after scanning menu items: %w", err)
		}
	}

	// Search orders if filter includes "orders" or "all"
	if filter == "all" || filter == "orders" {
		orderQuery := `
            SELECT 
                o.id, 
                COALESCE(c.first_name || ' ' || c.last_name, '') as customer_name,
                array_agg(mi.name) as items,
                o.total_price,
                o.status,
                ts_rank(
                    setweight(to_tsvector('english', COALESCE(c.first_name || ' ' || c.last_name, '')), 'A') ||
                    setweight(to_tsvector('english', COALESCE(o.special_instructions::text, '')), 'B'),
                    plainto_tsquery('english', $1)
                ) as relevance
            FROM orders o
            LEFT JOIN customers c ON o.customer_id = c.id
            JOIN order_items oi ON o.id = oi.order_id
            JOIN menu_items mi ON oi.menu_item_id = mi.id
            WHERE (
                to_tsvector('english', COALESCE(c.first_name || ' ' || c.last_name, '')) @@ plainto_tsquery('english', $1) OR
                to_tsvector('english', COALESCE(o.special_instructions::text, '')) @@ plainto_tsquery('english', $1)
            )
            AND ($2 = 0 OR o.total_price >= $2)
            AND ($3 = 0 OR o.total_price <= $3)
            GROUP BY o.id, c.first_name, c.last_name, o.total_price, o.status, o.special_instructions
            ORDER BY relevance DESC
            LIMIT 10
        `

		rows, err := r.db.QueryContext(ctx, orderQuery, query, minPrice, maxPrice)
		if err != nil {
			return models.SearchResult{}, fmt.Errorf("failed to search orders: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var order models.SearchOrder
			var items []string
			if err := rows.Scan(&order.ID, &order.CustomerName, pq.Array(&items), &order.Total, &order.Status, &order.Relevance); err != nil {
				return models.SearchResult{}, fmt.Errorf("failed to scan order: %w", err)
			}
			order.Items = items
			result.Orders = append(result.Orders, order)
		}
		if err = rows.Err(); err != nil {
			return models.SearchResult{}, fmt.Errorf("error after scanning orders: %w", err)
		}
	}

	result.Total = len(result.MenuItems) + len(result.Orders) + len(result.Customers)
	return result, nil
}
