package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"

	"github.com/lib/pq"
)

var (
	ErrOperationTypeInvalid     = errors.New("operation type is invalid")
	ErrOperationStatusInvalid   = errors.New("operation status is invalid")
	ErrOperationItemsRequired   = errors.New("at least one product line item is required")
	ErrOperationLocationInvalid = errors.New("invalid location reference")
	ErrOperationProductInvalid  = errors.New("invalid product reference")
	ErrOperationNotFound        = errors.New("operation order not found")
)

type OperationsRepo struct{ db *sql.DB }

func NewOperationsRepo(db *sql.DB) *OperationsRepo { return &OperationsRepo{db: db} }

type OperationCreateItemInput struct {
	ProductID string
	Quantity  int
}

type OperationCreateInput struct {
	OperationType string
	FromParty     string
	ToParty       string
	LocationID    string
	ContactNumber string
	ScheduledDate time.Time
	Status        string
	Items         []OperationCreateItemInput
}

func (r *OperationsRepo) ListLocations() ([]models.OperationLocationOption, error) {
	const q = `
		SELECT
			l.id,
			l.name,
			l.short_code,
			COALESCE(array_agg(w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names
		FROM locations.locations l
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		GROUP BY l.id, l.name, l.short_code
		ORDER BY l.name ASC`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	locations := make([]models.OperationLocationOption, 0)
	for rows.Next() {
		var location models.OperationLocationOption
		var warehouses pq.StringArray
		if err := rows.Scan(&location.ID, &location.Name, &location.ShortCode, &warehouses); err != nil {
			return nil, err
		}
		location.WarehouseNames = append([]string(nil), warehouses...)
		locations = append(locations, location)
	}

	return locations, rows.Err()
}

func (r *OperationsRepo) ListProducts(limit int) ([]models.OperationProductOption, error) {
	if limit <= 0 || limit > 400 {
		limit = 220
	}

	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			c.name AS category_name,
			p.location_id,
			l.name AS location_name,
			l.short_code AS location_short_code,
			p.on_hand_quantity,
			p.free_to_use_quantity
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		JOIN locations.locations l ON l.id = p.location_id
		ORDER BY p.updated_at DESC
		LIMIT $1`

	rows, err := r.db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.OperationProductOption, 0)
	for rows.Next() {
		var product models.OperationProductOption
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.CategoryName,
			&product.LocationID,
			&product.LocationName,
			&product.LocationShortCode,
			&product.OnHandQuantity,
			&product.FreeToUseQuantity,
		); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

func (r *OperationsRepo) ListOrders(operationType string, limit int) ([]models.OperationOrder, error) {
	opType := strings.ToUpper(strings.TrimSpace(operationType))
	if opType != "IN" && opType != "OUT" {
		return nil, ErrOperationTypeInvalid
	}
	if limit <= 0 || limit > 300 {
		limit = 120
	}

	const q = `
		SELECT
			o.id,
			o.reference_sequence,
			o.reference_number,
			o.operation_type,
			COALESCE(o.from_party, ''),
			COALESCE(o.to_party, ''),
			o.location_id,
			l.name,
			l.short_code,
			o.warehouse_short_code,
			COALESCE(array_agg(DISTINCT w.name ORDER BY w.name)
				FILTER (WHERE w.name IS NOT NULL), '{}') AS warehouse_names,
			COALESCE(o.contact_number, ''),
			o.scheduled_date,
			o.status,
			o.created_at,
			o.updated_at
		FROM operations.orders o
		JOIN locations.locations l ON l.id = o.location_id
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE o.operation_type = $1
		GROUP BY
			o.id, o.reference_sequence, o.reference_number, o.operation_type,
			o.from_party, o.to_party, o.location_id, l.name, l.short_code,
			o.warehouse_short_code, o.contact_number, o.scheduled_date, o.status,
			o.created_at, o.updated_at
		ORDER BY o.created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(q, opType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]models.OperationOrder, 0)
	orderIDs := make([]int64, 0)
	indexByID := make(map[int64]int)

	for rows.Next() {
		var order models.OperationOrder
		var warehouses pq.StringArray
		if err := rows.Scan(
			&order.ID,
			&order.ReferenceSequence,
			&order.ReferenceNumber,
			&order.OperationType,
			&order.FromParty,
			&order.ToParty,
			&order.LocationID,
			&order.LocationName,
			&order.LocationShortCode,
			&order.WarehouseShortCode,
			&warehouses,
			&order.ContactNumber,
			&order.ScheduledDate,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		order.WarehouseNames = append([]string(nil), warehouses...)
		order.Items = make([]models.OperationOrderItem, 0)

		indexByID[order.ID] = len(orders)
		orderIDs = append(orderIDs, order.ID)
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(orderIDs) == 0 {
		return orders, nil
	}

	const itemsQuery = `
		SELECT
			oi.id,
			oi.order_id,
			oi.product_id,
			p.sku,
			p.name,
			oi.quantity
		FROM operations.order_items oi
		JOIN stocks.products p ON p.id = oi.product_id
		WHERE oi.order_id = ANY($1)
		ORDER BY oi.order_id DESC, oi.created_at ASC, oi.id ASC`

	itemRows, err := r.db.Query(itemsQuery, pq.Int64Array(orderIDs))
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var (
			item    models.OperationOrderItem
			orderID int64
		)
		if err := itemRows.Scan(
			&item.ID,
			&orderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
		); err != nil {
			return nil, err
		}

		if idx, ok := indexByID[orderID]; ok {
			orders[idx].Items = append(orders[idx].Items, item)
		}
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OperationsRepo) CreateOrder(input OperationCreateInput) (*models.OperationOrder, error) {
	opType := strings.ToUpper(strings.TrimSpace(input.OperationType))
	if opType != "IN" && opType != "OUT" {
		return nil, ErrOperationTypeInvalid
	}

	status := strings.ToUpper(strings.TrimSpace(input.Status))
	if status == "" {
		status = "DRAFT"
	}

	if !isStatusAllowedForOperation(opType, status) {
		return nil, ErrOperationStatusInvalid
	}

	locationID := strings.TrimSpace(input.LocationID)
	if locationID == "" {
		return nil, ErrOperationLocationInvalid
	}

	aggregatedItems := aggregateItems(input.Items)
	if len(aggregatedItems) == 0 {
		return nil, ErrOperationItemsRequired
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	warehouseShortCode, err := resolveWarehouseShortCodeTx(tx, locationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationLocationInvalid
		}
		return nil, err
	}

	var sequence int64
	if err := tx.QueryRow(`SELECT nextval('operations.reference_seq')`).Scan(&sequence); err != nil {
		return nil, err
	}

	referenceNumber := fmt.Sprintf("%s/%s/%d", warehouseShortCode, opType, sequence)

	fromParty := strings.TrimSpace(input.FromParty)
	toParty := strings.TrimSpace(input.ToParty)
	contactNumber := strings.TrimSpace(input.ContactNumber)

	const insertOrder = `
		INSERT INTO operations.orders (
			reference_sequence,
			reference_number,
			operation_type,
			from_party,
			to_party,
			location_id,
			warehouse_short_code,
			contact_number,
			scheduled_date,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	var (
		orderID   int64
		createdAt time.Time
		updatedAt time.Time
	)
	if err := tx.QueryRow(
		insertOrder,
		sequence,
		referenceNumber,
		opType,
		nullableText(fromParty),
		nullableText(toParty),
		locationID,
		warehouseShortCode,
		nullableText(contactNumber),
		input.ScheduledDate,
		status,
	).Scan(&orderID, &createdAt, &updatedAt); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrOperationLocationInvalid
		}
		return nil, err
	}

	const insertItem = `
		INSERT INTO operations.order_items (order_id, product_id, quantity)
		VALUES ($1, $2, $3)`

	for productID, qty := range aggregatedItems {
		if _, err := tx.Exec(insertItem, orderID, productID, qty); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23503" {
				return nil, ErrOperationProductInvalid
			}
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	orders, err := r.ListOrders(opType, 500)
	if err != nil {
		return nil, err
	}

	for index := range orders {
		if orders[index].ID == orderID {
			return &orders[index], nil
		}
	}

	return nil, errors.New("order created but could not be loaded")
}

func resolveWarehouseShortCodeTx(tx *sql.Tx, locationID string) (string, error) {
	const q = `
		SELECT w.short_code
		FROM locations.location_warehouses lw
		JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE lw.location_id = $1
		ORDER BY w.short_code ASC
		LIMIT 1`

	var shortCode string
	err := tx.QueryRow(q, locationID).Scan(&shortCode)
	return shortCode, err
}

func aggregateItems(items []OperationCreateItemInput) map[string]int {
	aggregated := make(map[string]int)
	for _, item := range items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			continue
		}
		if item.Quantity <= 0 {
			continue
		}
		aggregated[productID] += item.Quantity
	}
	return aggregated
}

func isStatusAllowedForOperation(operationType, status string) bool {
	if operationType == "IN" {
		switch status {
		case "DRAFT", "READY", "DONE":
			return true
		default:
			return false
		}
	}

	if operationType == "OUT" {
		switch status {
		case "DRAFT", "WAITING", "READY", "DONE":
			return true
		default:
			return false
		}
	}

	return false
}

func (r *OperationsRepo) DeleteOrder(orderID int64) error {
	const q = `DELETE FROM operations.orders WHERE id = $1`

	result, err := r.db.Exec(q, orderID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrOperationNotFound
	}

	return nil
}
