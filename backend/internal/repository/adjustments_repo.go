package repository

import (
	"database/sql"
	"errors"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

var (
	ErrAdjustmentProductInvalid      = errors.New("invalid adjustment product")
	ErrAdjustmentLocationInvalid     = errors.New("invalid adjustment location")
	ErrAdjustmentQuantityInvalid     = errors.New("invalid adjustment quantity")
	ErrAdjustmentReasonRequired      = errors.New("adjustment reason is required")
	ErrAdjustmentDestinationRequired = errors.New("adjustment destination location is required")
	ErrAdjustmentNoChange            = errors.New("no adjustment change requested")
	ErrAdjustmentStockInsufficient   = errors.New("insufficient free-to-use stock")
)

type AdjustmentsRepo struct{ db *sql.DB }

func NewAdjustmentsRepo(db *sql.DB) *AdjustmentsRepo { return &AdjustmentsRepo{db: db} }

type AdjustmentTransferInput struct {
	ProductID      string
	FromLocationID string
	ToLocationID   string
	Quantity       int
	Reason         string
}

type AdjustmentQuantityInput struct {
	ProductID         string
	LocationID        string
	FreeToUseQuantity int
	Reason            string
}

func (r *AdjustmentsRepo) ListInventoryRows(limit int) ([]models.AdjustmentInventoryRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 320
	}

	stockRepo := NewStockRepo(r.db)
	products, err := stockRepo.ListProducts("", limit)
	if err != nil {
		return nil, err
	}

	rows := make([]models.AdjustmentInventoryRow, 0)
	for _, product := range products {
		for _, level := range product.StockLevels {
			rows = append(rows, models.AdjustmentInventoryRow{
				ProductID:         product.ID,
				SKU:               product.SKU,
				Name:              product.Name,
				CategoryName:      product.CategoryName,
				LocationID:        level.LocationID,
				LocationName:      level.LocationName,
				LocationShortCode: level.LocationShortCode,
				WarehouseNames:    append([]string(nil), level.WarehouseNames...),
				OnHandQuantity:    level.OnHandQuantity,
				FreeToUseQuantity: level.FreeToUseQuantity,
			})
		}
	}

	return rows, nil
}

func (r *AdjustmentsRepo) ListHistory(limit int) ([]models.AdjustmentHistoryEntry, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}

	const q = `
		SELECT
			h.id,
			h.action_type,
			h.product_id,
			p.sku,
			p.name,
			COALESCE(h.from_location_id::text, ''),
			COALESCE(fl.name, ''),
			COALESCE(fl.short_code, ''),
			COALESCE(h.to_location_id::text, ''),
			COALESCE(tl.name, ''),
			COALESCE(tl.short_code, ''),
			h.quantity_changed,
			h.previous_free_to_use_quantity,
			h.updated_free_to_use_quantity,
			h.reason,
			h.created_at
		FROM adjustments.internal_transfer_history h
		JOIN stocks.products p ON p.id = h.product_id
		LEFT JOIN locations.locations fl ON fl.id = h.from_location_id
		LEFT JOIN locations.locations tl ON tl.id = h.to_location_id
		ORDER BY h.created_at DESC
		LIMIT $1`

	rows, err := r.db.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]models.AdjustmentHistoryEntry, 0)
	for rows.Next() {
		var entry models.AdjustmentHistoryEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.ActionType,
			&entry.ProductID,
			&entry.SKU,
			&entry.Name,
			&entry.FromLocationID,
			&entry.FromLocationName,
			&entry.FromLocationShortCode,
			&entry.ToLocationID,
			&entry.ToLocationName,
			&entry.ToLocationShortCode,
			&entry.QuantityChanged,
			&entry.PreviousFreeToUseQuantity,
			&entry.UpdatedFreeToUseQuantity,
			&entry.Reason,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	return history, rows.Err()
}

func (r *AdjustmentsRepo) TransferStock(input AdjustmentTransferInput) error {
	productID := strings.TrimSpace(input.ProductID)
	fromLocationID := strings.TrimSpace(input.FromLocationID)
	toLocationID := strings.TrimSpace(input.ToLocationID)
	reason := strings.TrimSpace(input.Reason)

	if productID == "" {
		return ErrAdjustmentProductInvalid
	}
	if fromLocationID == "" {
		return ErrAdjustmentLocationInvalid
	}
	if toLocationID == "" {
		return ErrAdjustmentDestinationRequired
	}
	if fromLocationID == toLocationID {
		return ErrAdjustmentNoChange
	}
	if input.Quantity <= 0 {
		return ErrAdjustmentQuantityInvalid
	}
	if reason == "" {
		return ErrAdjustmentReasonRequired
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := ensureLocationExistsTx(tx, fromLocationID); err != nil {
		return err
	}
	if err := ensureLocationExistsTx(tx, toLocationID); err != nil {
		if errors.Is(err, ErrAdjustmentLocationInvalid) {
			return ErrAdjustmentDestinationRequired
		}
		return err
	}

	levels, err := loadProductStockLevelsForUpdateTx(tx, productID)
	if err != nil {
		if errors.Is(err, ErrOperationProductInvalid) {
			return ErrAdjustmentProductInvalid
		}
		return err
	}

	sourceIndex := -1
	sourceBefore := 0
	for index := range levels {
		if strings.TrimSpace(levels[index].LocationID) == fromLocationID {
			sourceIndex = index
			sourceBefore = levels[index].FreeToUseQuantity
			break
		}
	}
	if sourceIndex == -1 || sourceBefore < input.Quantity {
		return ErrAdjustmentStockInsufficient
	}

	levels[sourceIndex].FreeToUseQuantity -= input.Quantity

	destinationIndex := -1
	for index := range levels {
		if strings.TrimSpace(levels[index].LocationID) == toLocationID {
			destinationIndex = index
			break
		}
	}
	if destinationIndex == -1 {
		levels = append(levels, models.ProductStockLevel{
			LocationID:        toLocationID,
			OnHandQuantity:    0,
			FreeToUseQuantity: input.Quantity,
		})
	} else {
		levels[destinationIndex].FreeToUseQuantity += input.Quantity
	}

	if err := saveProductStockLevelsTx(tx, productID, levels); err != nil {
		return err
	}

	if err := insertAdjustmentHistoryTx(tx, adjustmentHistoryRecord{
		ActionType:                "TRANSFER",
		ProductID:                 productID,
		FromLocationID:            fromLocationID,
		ToLocationID:              toLocationID,
		QuantityChanged:           input.Quantity,
		PreviousFreeToUseQuantity: sourceBefore,
		UpdatedFreeToUseQuantity:  sourceBefore - input.Quantity,
		Reason:                    reason,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *AdjustmentsRepo) AdjustFreeToUseQuantity(input AdjustmentQuantityInput) error {
	productID := strings.TrimSpace(input.ProductID)
	locationID := strings.TrimSpace(input.LocationID)
	reason := strings.TrimSpace(input.Reason)

	if productID == "" {
		return ErrAdjustmentProductInvalid
	}
	if locationID == "" {
		return ErrAdjustmentLocationInvalid
	}
	if input.FreeToUseQuantity < 0 {
		return ErrAdjustmentQuantityInvalid
	}
	if reason == "" {
		return ErrAdjustmentReasonRequired
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := ensureLocationExistsTx(tx, locationID); err != nil {
		return err
	}

	levels, err := loadProductStockLevelsForUpdateTx(tx, productID)
	if err != nil {
		if errors.Is(err, ErrOperationProductInvalid) {
			return ErrAdjustmentProductInvalid
		}
		return err
	}

	levelIndex := -1
	currentFreeToUse := 0
	for index := range levels {
		if strings.TrimSpace(levels[index].LocationID) == locationID {
			levelIndex = index
			currentFreeToUse = levels[index].FreeToUseQuantity
			break
		}
	}
	if levelIndex == -1 {
		return ErrAdjustmentLocationInvalid
	}
	if currentFreeToUse == input.FreeToUseQuantity {
		return ErrAdjustmentNoChange
	}

	levels[levelIndex].FreeToUseQuantity = input.FreeToUseQuantity
	if err := saveProductStockLevelsTx(tx, productID, levels); err != nil {
		return err
	}

	if err := insertAdjustmentHistoryTx(tx, adjustmentHistoryRecord{
		ActionType:                "QUANTITY_ADJUSTMENT",
		ProductID:                 productID,
		FromLocationID:            locationID,
		ToLocationID:              locationID,
		QuantityChanged:           input.FreeToUseQuantity - currentFreeToUse,
		PreviousFreeToUseQuantity: currentFreeToUse,
		UpdatedFreeToUseQuantity:  input.FreeToUseQuantity,
		Reason:                    reason,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

type adjustmentHistoryRecord struct {
	ActionType                string
	ProductID                 string
	FromLocationID            string
	ToLocationID              string
	QuantityChanged           int
	PreviousFreeToUseQuantity int
	UpdatedFreeToUseQuantity  int
	Reason                    string
}

func insertAdjustmentHistoryTx(tx *sql.Tx, record adjustmentHistoryRecord) error {
	const q = `
		INSERT INTO adjustments.internal_transfer_history (
			action_type,
			product_id,
			from_location_id,
			to_location_id,
			quantity_changed,
			previous_free_to_use_quantity,
			updated_free_to_use_quantity,
			reason
		)
		VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, $6, $7, $8)`

	_, err := tx.Exec(
		q,
		record.ActionType,
		record.ProductID,
		record.FromLocationID,
		record.ToLocationID,
		record.QuantityChanged,
		record.PreviousFreeToUseQuantity,
		record.UpdatedFreeToUseQuantity,
		record.Reason,
	)
	return err
}

func ensureLocationExistsTx(tx *sql.Tx, locationID string) error {
	const q = `SELECT 1 FROM locations.locations WHERE id = $1`

	var exists int
	if err := tx.QueryRow(q, locationID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAdjustmentLocationInvalid
		}
		return err
	}

	return nil
}
