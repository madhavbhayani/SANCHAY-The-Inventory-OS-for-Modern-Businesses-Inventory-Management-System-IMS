package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"

	"github.com/lib/pq"
)

var (
	ErrOperationTypeInvalid       = errors.New("operation type is invalid")
	ErrOperationStatusInvalid     = errors.New("operation status is invalid")
	ErrOperationItemsRequired     = errors.New("at least one product line item is required")
	ErrOperationLocationInvalid   = errors.New("invalid location reference")
	ErrOperationProductInvalid    = errors.New("invalid product reference")
	ErrOperationNotFound          = errors.New("operation order not found")
	ErrOperationStockInsufficient = errors.New("insufficient stock for delivery")
	ErrOperationFinalized         = errors.New("finalized order cannot be modified")
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

type OperationUpdateInput struct {
	OperationType   string
	ReferenceNumber string
	FromParty       string
	ToParty         string
	LocationID      string
	ContactNumber   string
	ScheduledDate   time.Time
	Status          string
	Items           []OperationCreateItemInput
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

	locationsByID, err := r.loadLocationsByID()
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT
			p.id,
			p.sku,
			p.name,
			c.name AS category_name,
			COALESCE(p.stock_levels, '[]'::jsonb)
		FROM stocks.products p
		JOIN stocks.categories c ON c.id = p.category_id
		ORDER BY p.updated_at DESC
		LIMIT $1`

	rows, err := r.db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.OperationProductOption, 0)
	for rows.Next() {
		var (
			product        models.OperationProductOption
			rawStockLevels []byte
		)
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.CategoryName,
			&rawStockLevels,
		); err != nil {
			return nil, err
		}
		if err := populateOperationProductStockFields(&product, rawStockLevels, locationsByID); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

func (r *OperationsRepo) ListOrders(operationType string, limit int) ([]models.OperationOrder, error) {
	opType := normalizeOperationType(operationType)
	if opType == "" {
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

	itemsByOrderID, err := r.listItemsByOrderIDs(orderIDs)
	if err != nil {
		return nil, err
	}

	for orderID, items := range itemsByOrderID {
		if index, ok := indexByID[orderID]; ok {
			orders[index].Items = append(orders[index].Items, items...)
		}
	}

	return orders, nil
}

func (r *OperationsRepo) CreateOrder(input OperationCreateInput) (*models.OperationOrder, error) {
	opType := normalizeOperationType(input.OperationType)
	if opType == "" {
		return nil, ErrOperationTypeInvalid
	}

	status := strings.ToUpper(strings.TrimSpace(input.Status))
	if status == "" {
		status = "DRAFT"
	}
	if status == "CANCELLED" || !isStatusAllowedForOperation(opType, status) {
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
		RETURNING id`

	var orderID int64
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
	).Scan(&orderID); err != nil {
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

	if err := applyStockEffectTx(tx, buildOrderStockEffect(opType, status, locationID, aggregatedItems)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetOrderByReference(opType, referenceNumber)
}

func (r *OperationsRepo) GetOrderByReference(operationType, referenceNumber string) (*models.OperationOrder, error) {
	opType := normalizeOperationType(operationType)
	if opType == "" {
		return nil, ErrOperationTypeInvalid
	}
	ref := strings.TrimSpace(referenceNumber)
	if ref == "" {
		return nil, ErrOperationNotFound
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
		  AND o.reference_number = $2
		GROUP BY
			o.id, o.reference_sequence, o.reference_number, o.operation_type,
			o.from_party, o.to_party, o.location_id, l.name, l.short_code,
			o.warehouse_short_code, o.contact_number, o.scheduled_date, o.status,
			o.created_at, o.updated_at`

	var (
		order      models.OperationOrder
		warehouses pq.StringArray
	)

	err := r.db.QueryRow(q, opType, ref).Scan(
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
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationNotFound
		}
		return nil, err
	}
	order.WarehouseNames = append([]string(nil), warehouses...)
	order.Items = make([]models.OperationOrderItem, 0)

	itemsByOrderID, err := r.listItemsByOrderIDs([]int64{order.ID})
	if err != nil {
		return nil, err
	}
	order.Items = append(order.Items, itemsByOrderID[order.ID]...)

	return &order, nil
}

func (r *OperationsRepo) UpdateOrderByReference(input OperationUpdateInput) (*models.OperationOrder, error) {
	opType := normalizeOperationType(input.OperationType)
	if opType == "" {
		return nil, ErrOperationTypeInvalid
	}
	ref := strings.TrimSpace(input.ReferenceNumber)
	if ref == "" {
		return nil, ErrOperationNotFound
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

	orderRecord, err := getOrderForUpdateTx(tx, opType, ref)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationNotFound
		}
		return nil, err
	}

	currentItems, err := listOrderItemQuantitiesTx(tx, orderRecord.ID)
	if err != nil {
		return nil, err
	}

	orderID := orderRecord.ID
	currentStatus := orderRecord.Status

	if isTerminalOperationStatus(currentStatus) {
		return nil, ErrOperationFinalized
	}

	if status == "CANCELLED" {
		if err := ensureOrdersStatusConstraintTx(tx); err != nil {
			return nil, err
		}
	}

	warehouseShortCode, err := resolveWarehouseShortCodeTx(tx, locationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationLocationInvalid
		}
		return nil, err
	}

	const updateOrder = `
		UPDATE operations.orders
		SET
			from_party = $2,
			to_party = $3,
			location_id = $4,
			warehouse_short_code = $5,
			contact_number = $6,
			scheduled_date = $7,
			status = $8,
			updated_at = NOW()
		WHERE id = $1`

	if _, err := tx.Exec(
		updateOrder,
		orderID,
		nullableText(strings.TrimSpace(input.FromParty)),
		nullableText(strings.TrimSpace(input.ToParty)),
		locationID,
		warehouseShortCode,
		nullableText(strings.TrimSpace(input.ContactNumber)),
		input.ScheduledDate,
		status,
	); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return nil, ErrOperationLocationInvalid
		}
		return nil, err
	}

	if _, err := tx.Exec(`DELETE FROM operations.order_items WHERE order_id = $1`, orderID); err != nil {
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

	currentEffect := buildOrderStockEffect(opType, currentStatus, orderRecord.LocationID, currentItems)
	newEffect := buildOrderStockEffect(opType, status, locationID, aggregatedItems)
	if err := applyStockEffectTx(tx, diffStockEffects(currentEffect, newEffect)); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetOrderByReference(opType, ref)
}

func (r *OperationsRepo) UpdateOrderStatusByReference(operationType, referenceNumber, status string) (*models.OperationOrder, error) {
	opType := normalizeOperationType(operationType)
	if opType == "" {
		return nil, ErrOperationTypeInvalid
	}
	ref := strings.TrimSpace(referenceNumber)
	if ref == "" {
		return nil, ErrOperationNotFound
	}

	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if !isStatusAllowedForOperation(opType, normalizedStatus) {
		return nil, ErrOperationStatusInvalid
	}

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	orderRecord, err := getOrderForUpdateTx(tx, opType, ref)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationNotFound
		}
		return nil, err
	}

	currentStatus := strings.ToUpper(strings.TrimSpace(orderRecord.Status))
	if currentStatus == normalizedStatus {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return r.GetOrderByReference(opType, ref)
	}

	if isTerminalOperationStatus(currentStatus) {
		return nil, ErrOperationFinalized
	}

	if normalizedStatus == "CANCELLED" {
		if err := ensureOrdersStatusConstraintTx(tx); err != nil {
			return nil, err
		}
	}

	items, err := listOrderItemQuantitiesTx(tx, orderRecord.ID)
	if err != nil {
		return nil, err
	}
	currentEffect := buildOrderStockEffect(opType, currentStatus, orderRecord.LocationID, items)
	newEffect := buildOrderStockEffect(opType, normalizedStatus, orderRecord.LocationID, items)
	if err := applyStockEffectTx(tx, diffStockEffects(currentEffect, newEffect)); err != nil {
		return nil, err
	}

	const q = `
		UPDATE operations.orders
		SET status = $3, updated_at = NOW()
		WHERE operation_type = $1 AND reference_number = $2`

	result, err := tx.Exec(q, opType, ref, normalizedStatus)
	if err != nil {
		return nil, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, ErrOperationNotFound
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetOrderByReference(opType, ref)
}

func (r *OperationsRepo) ValidateOrderByReference(operationType, referenceNumber string) (*models.OperationOrder, bool, error) {
	order, err := r.GetOrderByReference(operationType, referenceNumber)
	if err != nil {
		return nil, false, err
	}

	if order.OperationType == "IN" {
		if order.Status != "READY" && order.Status != "DONE" && order.Status != "CANCELLED" {
			updated, err := r.UpdateOrderStatusByReference(order.OperationType, order.ReferenceNumber, "READY")
			if err != nil {
				return nil, false, err
			}
			return updated, true, nil
		}

		return order, true, nil
	}

	allItemsInStock := true
	for _, item := range order.Items {
		if item.AvailableQuantity < item.Quantity {
			allItemsInStock = false
			break
		}
	}

	if allItemsInStock && order.Status != "READY" && order.Status != "DONE" && order.Status != "CANCELLED" {
		updated, err := r.UpdateOrderStatusByReference(order.OperationType, order.ReferenceNumber, "READY")
		if err != nil {
			return nil, false, err
		}
		return updated, true, nil
	}

	return order, allItemsInStock, nil
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

func (r *OperationsRepo) listItemsByOrderIDs(orderIDs []int64) (map[int64][]models.OperationOrderItem, error) {
	itemsByOrderID := make(map[int64][]models.OperationOrderItem)
	if len(orderIDs) == 0 {
		return itemsByOrderID, nil
	}

	const q = `
		SELECT
			oi.id,
			oi.order_id,
			oi.product_id,
			p.sku,
			p.name,
			oi.quantity,
			COALESCE(p.stock_levels, '[]'::jsonb),
			o.location_id
		FROM operations.order_items oi
		JOIN stocks.products p ON p.id = oi.product_id
		JOIN operations.orders o ON o.id = oi.order_id
		WHERE oi.order_id = ANY($1)
		ORDER BY oi.order_id DESC, oi.created_at ASC, oi.id ASC`

	rows, err := r.db.Query(q, pq.Int64Array(orderIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			item           models.OperationOrderItem
			orderID        int64
			rawStockLevels []byte
			locationID     string
		)
		if err := rows.Scan(
			&item.ID,
			&orderID,
			&item.ProductID,
			&item.ProductSKU,
			&item.ProductName,
			&item.Quantity,
			&rawStockLevels,
			&locationID,
		); err != nil {
			return nil, err
		}

		availableQuantity, err := freeToUseQuantityForLocation(rawStockLevels, locationID)
		if err != nil {
			return nil, err
		}
		item.AvailableQuantity = availableQuantity

		itemsByOrderID[orderID] = append(itemsByOrderID[orderID], item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return itemsByOrderID, nil
}

func (r *OperationsRepo) loadLocationsByID() (map[string]models.OperationLocationOption, error) {
	locations, err := r.ListLocations()
	if err != nil {
		return nil, err
	}

	locationsByID := make(map[string]models.OperationLocationOption, len(locations))
	for _, location := range locations {
		locationsByID[location.ID] = location
	}

	return locationsByID, nil
}

func populateOperationProductStockFields(product *models.OperationProductOption, rawStockLevels []byte, locationsByID map[string]models.OperationLocationOption) error {
	levels, err := unmarshalProductStockLevels(rawStockLevels)
	if err != nil {
		return err
	}

	sort.SliceStable(levels, func(i, j int) bool {
		leftName := locationNameForOperationLevel(levels[i].LocationID, locationsByID)
		rightName := locationNameForOperationLevel(levels[j].LocationID, locationsByID)
		if leftName == rightName {
			return levels[i].LocationID < levels[j].LocationID
		}
		if leftName == "" {
			return false
		}
		if rightName == "" {
			return true
		}
		return leftName < rightName
	})

	product.StockLevels = make([]models.OperationProductStockLevel, 0, len(levels))
	product.OnHandQuantity = 0
	product.FreeToUseQuantity = 0
	product.LocationID = ""
	product.LocationName = ""
	product.LocationShortCode = ""

	for index, level := range levels {
		operationLevel := models.OperationProductStockLevel{
			LocationID:        level.LocationID,
			OnHandQuantity:    level.OnHandQuantity,
			FreeToUseQuantity: level.FreeToUseQuantity,
		}

		if metadata, ok := locationsByID[level.LocationID]; ok {
			operationLevel.LocationName = metadata.Name
			operationLevel.LocationShortCode = metadata.ShortCode
		}

		product.StockLevels = append(product.StockLevels, operationLevel)
		product.OnHandQuantity += operationLevel.OnHandQuantity
		product.FreeToUseQuantity += operationLevel.FreeToUseQuantity

		if index == 0 {
			product.LocationID = operationLevel.LocationID
			product.LocationName = operationLevel.LocationName
			product.LocationShortCode = operationLevel.LocationShortCode
		}
	}

	return nil
}

func locationNameForOperationLevel(locationID string, locationsByID map[string]models.OperationLocationOption) string {
	if metadata, ok := locationsByID[locationID]; ok {
		return metadata.Name
	}
	return ""
}

func freeToUseQuantityForLocation(rawStockLevels []byte, locationID string) (int, error) {
	levels, err := unmarshalProductStockLevels(rawStockLevels)
	if err != nil {
		return 0, err
	}

	normalizedLocationID := strings.TrimSpace(locationID)
	for _, level := range levels {
		if strings.TrimSpace(level.LocationID) == normalizedLocationID {
			return level.FreeToUseQuantity, nil
		}
	}

	return 0, nil
}

type operationOrderRecord struct {
	ID         int64
	Status     string
	LocationID string
}

func getOrderForUpdateTx(tx *sql.Tx, operationType, referenceNumber string) (*operationOrderRecord, error) {
	const q = `
		SELECT id, status, location_id
		FROM operations.orders
		WHERE operation_type = $1 AND reference_number = $2
		FOR UPDATE`

	var order operationOrderRecord
	if err := tx.QueryRow(q, operationType, referenceNumber).Scan(&order.ID, &order.Status, &order.LocationID); err != nil {
		return nil, err
	}

	return &order, nil
}

func listOrderItemQuantitiesTx(tx *sql.Tx, orderID int64) (map[string]int, error) {
	const q = `
		SELECT product_id, quantity
		FROM operations.order_items
		WHERE order_id = $1`

	rows, err := tx.Query(q, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(map[string]int)
	for rows.Next() {
		var productID string
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			return nil, err
		}
		items[productID] += quantity
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func ensureOrdersStatusConstraintTx(tx *sql.Tx) error {
	const q = `
		DO $$
		DECLARE
			status_constraint RECORD;
		BEGIN
			FOR status_constraint IN
				SELECT c.conname
				FROM pg_constraint c
				JOIN pg_class t ON t.oid = c.conrelid
				JOIN pg_namespace n ON n.oid = t.relnamespace
				WHERE n.nspname = 'operations'
				  AND t.relname = 'orders'
				  AND c.contype = 'c'
				  AND pg_get_constraintdef(c.oid) ILIKE '%status%'
			LOOP
				EXECUTE format('ALTER TABLE "operations".orders DROP CONSTRAINT %I', status_constraint.conname);
			END LOOP;

			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint c
				JOIN pg_class t ON t.oid = c.conrelid
				JOIN pg_namespace n ON n.oid = t.relnamespace
				WHERE n.nspname = 'operations'
				  AND t.relname = 'orders'
				  AND c.conname = 'operations_orders_status_check'
			) THEN
				EXECUTE '
					ALTER TABLE "operations".orders
					ADD CONSTRAINT operations_orders_status_check
					CHECK (status IN (''DRAFT'', ''WAITING'', ''READY'', ''DONE'', ''CANCELLED''))
				';
			END IF;
		END $$;`

	_, err := tx.Exec(q)
	return err
}

func isTerminalOperationStatus(status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	return normalized == "DONE" || normalized == "CANCELLED"
}

func resolveWarehouseShortCodeTx(tx *sql.Tx, locationID string) (string, error) {
	const q = `
		SELECT COALESCE(w.short_code, l.short_code)
		FROM locations.locations l
		LEFT JOIN locations.location_warehouses lw ON lw.location_id = l.id
		LEFT JOIN locations.warehouses w ON w.id = lw.warehouse_id
		WHERE l.id = $1
		ORDER BY (w.short_code IS NULL), w.short_code ASC
		LIMIT 1`

	var shortCode string
	err := tx.QueryRow(q, locationID).Scan(&shortCode)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(shortCode), nil
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

func normalizeOperationType(operationType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(operationType))
	if normalized == "IN" || normalized == "OUT" {
		return normalized
	}
	return ""
}

func isStatusAllowedForOperation(operationType, status string) bool {
	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if normalizedStatus == "" {
		return false
	}

	if operationType == "IN" {
		switch normalizedStatus {
		case "DRAFT", "READY", "DONE", "CANCELLED":
			return true
		default:
			return false
		}
	}

	if operationType == "OUT" {
		switch normalizedStatus {
		case "DRAFT", "WAITING", "READY", "DONE", "CANCELLED":
			return true
		default:
			return false
		}
	}

	return false
}
