package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

type MoveHistoryRepo struct{ db *sql.DB }

func NewMoveHistoryRepo(db *sql.DB) *MoveHistoryRepo { return &MoveHistoryRepo{db: db} }

type MoveHistoryListFilter struct {
	Limit     int
	EventType string
	Status    string
	Query     string
	FromDate  *time.Time
	ToDate    *time.Time
}

type stockLedgerInsertInput struct {
	EventType                 string
	OperationType             string
	OrderID                   int64
	ReferenceNumber           string
	ProductID                 string
	LocationID                string
	FromLocationID            string
	ToLocationID              string
	OnHandDelta               int
	FreeToUseDelta            int
	PreviousOnHandQuantity    int
	CurrentOnHandQuantity     int
	PreviousFreeToUseQuantity int
	CurrentFreeToUseQuantity  int
	PreviousStatus            string
	CurrentStatus             string
	Reason                    string
}

func (r *MoveHistoryRepo) ListStockLedger(filter MoveHistoryListFilter) ([]models.StockLedgerEntry, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 220
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT
			h.id,
			h.event_type,
			COALESCE(h.operation_type, ''),
			COALESCE(h.order_id, 0),
			COALESCE(h.reference_number, ''),
			h.product_id,
			p.sku,
			p.name,
			COALESCE(c.name, ''),
			COALESCE(h.location_id::text, ''),
			COALESCE(l.name, ''),
			COALESCE(l.short_code, ''),
			COALESCE(h.from_location_id::text, ''),
			COALESCE(fl.name, ''),
			COALESCE(fl.short_code, ''),
			COALESCE(h.to_location_id::text, ''),
			COALESCE(tl.name, ''),
			COALESCE(tl.short_code, ''),
			h.quantity_on_hand_delta,
			h.quantity_free_to_use_delta,
			h.previous_on_hand_quantity,
			h.current_on_hand_quantity,
			h.previous_free_to_use_quantity,
			h.current_free_to_use_quantity,
			COALESCE(h.previous_status, ''),
			COALESCE(h.current_status, ''),
			COALESCE(h.reason, ''),
			h.created_at
		FROM movehistory.stock_ledger h
		JOIN stocks.products p ON p.id = h.product_id
		LEFT JOIN stocks.categories c ON c.id = p.category_id
		LEFT JOIN locations.locations l ON l.id = h.location_id
		LEFT JOIN locations.locations fl ON fl.id = h.from_location_id
		LEFT JOIN locations.locations tl ON tl.id = h.to_location_id
	`)

	conditions := make([]string, 0)
	args := make([]any, 0)
	argIndex := 1

	if eventType := strings.ToUpper(strings.TrimSpace(filter.EventType)); eventType != "" {
		conditions = append(conditions, fmt.Sprintf("h.event_type = $%d", argIndex))
		args = append(args, eventType)
		argIndex++
	}

	if status := strings.ToUpper(strings.TrimSpace(filter.Status)); status != "" {
		conditions = append(conditions, fmt.Sprintf("UPPER(COALESCE(h.current_status, '')) = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if searchText := strings.TrimSpace(filter.Query); searchText != "" {
		conditions = append(conditions, fmt.Sprintf(`(
			COALESCE(h.reference_number, '') ILIKE $%d
			OR p.sku ILIKE $%d
			OR p.name ILIKE $%d
			OR COALESCE(h.reason, '') ILIKE $%d
		)`, argIndex, argIndex, argIndex, argIndex))
		args = append(args, "%"+searchText+"%")
		argIndex++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("h.created_at >= $%d", argIndex))
		args = append(args, *filter.FromDate)
		argIndex++
	}

	if filter.ToDate != nil {
		upperBound := filter.ToDate.Add(24 * time.Hour)
		conditions = append(conditions, fmt.Sprintf("h.created_at < $%d", argIndex))
		args = append(args, upperBound)
		argIndex++
	}

	if len(conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(conditions, " AND "))
	}

	query.WriteString(fmt.Sprintf(" ORDER BY h.created_at DESC LIMIT $%d", argIndex))
	args = append(args, limit)

	rows, err := r.db.Query(query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]models.StockLedgerEntry, 0)
	for rows.Next() {
		var entry models.StockLedgerEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.EventType,
			&entry.OperationType,
			&entry.OrderID,
			&entry.ReferenceNumber,
			&entry.ProductID,
			&entry.SKU,
			&entry.ProductName,
			&entry.CategoryName,
			&entry.LocationID,
			&entry.LocationName,
			&entry.LocationShortCode,
			&entry.FromLocationID,
			&entry.FromLocationName,
			&entry.FromLocationShortCode,
			&entry.ToLocationID,
			&entry.ToLocationName,
			&entry.ToLocationShortCode,
			&entry.OnHandDelta,
			&entry.FreeToUseDelta,
			&entry.PreviousOnHandQuantity,
			&entry.CurrentOnHandQuantity,
			&entry.PreviousFreeToUseQuantity,
			&entry.CurrentFreeToUseQuantity,
			&entry.PreviousStatus,
			&entry.CurrentStatus,
			&entry.Reason,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func insertStockLedgerTx(tx *sql.Tx, input stockLedgerInsertInput) error {
	const q = `
		INSERT INTO movehistory.stock_ledger (
			event_type,
			operation_type,
			order_id,
			reference_number,
			product_id,
			location_id,
			from_location_id,
			to_location_id,
			quantity_on_hand_delta,
			quantity_free_to_use_delta,
			previous_on_hand_quantity,
			current_on_hand_quantity,
			previous_free_to_use_quantity,
			current_free_to_use_quantity,
			previous_status,
			current_status,
			reason
		)
		VALUES (
			$1,
			NULLIF($2, ''),
			NULLIF($3, 0),
			NULLIF($4, ''),
			$5,
			NULLIF($6, '')::uuid,
			NULLIF($7, '')::uuid,
			NULLIF($8, '')::uuid,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			NULLIF($15, ''),
			NULLIF($16, ''),
			NULLIF($17, '')
		)`

	_, err := tx.Exec(
		q,
		strings.ToUpper(strings.TrimSpace(input.EventType)),
		strings.ToUpper(strings.TrimSpace(input.OperationType)),
		input.OrderID,
		strings.TrimSpace(input.ReferenceNumber),
		strings.TrimSpace(input.ProductID),
		strings.TrimSpace(input.LocationID),
		strings.TrimSpace(input.FromLocationID),
		strings.TrimSpace(input.ToLocationID),
		input.OnHandDelta,
		input.FreeToUseDelta,
		input.PreviousOnHandQuantity,
		input.CurrentOnHandQuantity,
		input.PreviousFreeToUseQuantity,
		input.CurrentFreeToUseQuantity,
		strings.ToUpper(strings.TrimSpace(input.PreviousStatus)),
		strings.ToUpper(strings.TrimSpace(input.CurrentStatus)),
		strings.TrimSpace(input.Reason),
	)

	return err
}
