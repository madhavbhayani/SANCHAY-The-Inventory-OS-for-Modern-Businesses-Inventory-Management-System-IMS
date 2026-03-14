package repository

import (
	"database/sql"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

type DashboardRepo struct{ db *sql.DB }

func NewDashboardRepo(db *sql.DB) *DashboardRepo { return &DashboardRepo{db: db} }

func (r *DashboardRepo) GetOverview() (*models.DashboardOverview, error) {
	overview := &models.DashboardOverview{
		StockByLocation:     make([]models.DashboardStockByLocation, 0),
		FreeToUseByCategory: make([]models.DashboardCategoryFreeToUse, 0),
	}

	if err := r.db.QueryRow(`
		SELECT
			COUNT(*) FILTER (
				WHERE operation_type = 'IN'
				  AND status NOT IN ('DONE', 'CANCELLED')
			) AS current_orders,
			COUNT(*) FILTER (WHERE operation_type = 'IN' AND status = 'READY') AS ready,
			COUNT(*) FILTER (WHERE operation_type = 'IN' AND status = 'DONE') AS done,
			COUNT(*) FILTER (
				WHERE operation_type = 'IN'
				  AND status NOT IN ('DONE', 'CANCELLED')
				  AND scheduled_date < CURRENT_DATE
			) AS late
		FROM operations.orders`).Scan(
		&overview.Receipts.CurrentOrders,
		&overview.Receipts.Ready,
		&overview.Receipts.Done,
		&overview.Receipts.Late,
	); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(`
		SELECT
			COUNT(*) FILTER (
				WHERE operation_type = 'OUT'
				  AND status NOT IN ('DONE', 'CANCELLED')
			) AS current_orders,
			COUNT(*) FILTER (WHERE operation_type = 'OUT' AND status = 'READY') AS ready,
			COUNT(*) FILTER (WHERE operation_type = 'OUT' AND status = 'DONE') AS done,
			COUNT(*) FILTER (
				WHERE operation_type = 'OUT'
				  AND status NOT IN ('DONE', 'CANCELLED')
				  AND scheduled_date < CURRENT_DATE
			) AS late
		FROM operations.orders`).Scan(
		&overview.Delivery.CurrentOrders,
		&overview.Delivery.Ready,
		&overview.Delivery.Done,
		&overview.Delivery.Late,
	); err != nil {
		return nil, err
	}

	locationRows, err := r.db.Query(`
		WITH stock_by_location AS (
			SELECT
				entry.value->>'location_id' AS location_id,
				SUM(COALESCE((entry.value->>'on_hand_quantity')::int, 0)) AS on_hand_quantity,
				SUM(COALESCE((entry.value->>'free_to_use_quantity')::int, 0)) AS free_to_use_quantity
			FROM stocks.products p
			CROSS JOIN LATERAL jsonb_array_elements(COALESCE(p.stock_levels, '[]'::jsonb)) AS entry(value)
			GROUP BY entry.value->>'location_id'
		)
		SELECT
			l.id,
			l.name,
			l.short_code,
			COALESCE(s.on_hand_quantity, 0),
			COALESCE(s.free_to_use_quantity, 0)
		FROM locations.locations l
		LEFT JOIN stock_by_location s ON s.location_id = l.id::text
		ORDER BY l.name ASC`)
	if err != nil {
		return nil, err
	}
	defer locationRows.Close()

	for locationRows.Next() {
		var row models.DashboardStockByLocation
		if err := locationRows.Scan(
			&row.LocationID,
			&row.LocationName,
			&row.LocationShortCode,
			&row.OnHandQuantity,
			&row.FreeToUseQuantity,
		); err != nil {
			return nil, err
		}
		overview.StockByLocation = append(overview.StockByLocation, row)
	}
	if err := locationRows.Err(); err != nil {
		return nil, err
	}

	categoryRows, err := r.db.Query(`
		SELECT
			c.id,
			c.name,
			COALESCE(SUM(COALESCE((entry.value->>'free_to_use_quantity')::int, 0)), 0) AS free_to_use_quantity
		FROM stocks.categories c
		LEFT JOIN stocks.products p ON p.category_id = c.id
		LEFT JOIN LATERAL jsonb_array_elements(COALESCE(p.stock_levels, '[]'::jsonb)) AS entry(value) ON true
		GROUP BY c.id, c.name
		ORDER BY c.name ASC`)
	if err != nil {
		return nil, err
	}
	defer categoryRows.Close()

	for categoryRows.Next() {
		var row models.DashboardCategoryFreeToUse
		if err := categoryRows.Scan(&row.CategoryID, &row.CategoryName, &row.FreeToUseQuantity); err != nil {
			return nil, err
		}
		overview.FreeToUseByCategory = append(overview.FreeToUseByCategory, row)
	}

	if err := categoryRows.Err(); err != nil {
		return nil, err
	}

	return overview, nil
}
