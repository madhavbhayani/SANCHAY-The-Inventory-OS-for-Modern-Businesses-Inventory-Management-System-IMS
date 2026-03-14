package repository

import (
	"database/sql"
	"errors"
	"sort"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

type stockLevelKey struct {
	ProductID  string
	LocationID string
}

type stockLevelDelta struct {
	OnHandDelta    int
	FreeToUseDelta int
}

type stockLevelStateSnapshot struct {
	PreviousOnHandQuantity    int
	CurrentOnHandQuantity     int
	PreviousFreeToUseQuantity int
	CurrentFreeToUseQuantity  int
}

func buildOrderStockEffect(operationType, status, locationID string, items map[string]int) map[stockLevelKey]stockLevelDelta {
	effect := make(map[stockLevelKey]stockLevelDelta)
	if len(items) == 0 {
		return effect
	}

	normalizedType := normalizeOperationType(operationType)
	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	normalizedLocationID := strings.TrimSpace(locationID)
	if normalizedType == "" || normalizedLocationID == "" {
		return effect
	}

	for productID, quantity := range items {
		if quantity <= 0 {
			continue
		}

		key := stockLevelKey{
			ProductID:  strings.TrimSpace(productID),
			LocationID: normalizedLocationID,
		}

		delta := stockLevelDelta{}
		switch normalizedType {
		case "IN":
			switch normalizedStatus {
			case "READY":
				delta.OnHandDelta = quantity
			case "DONE":
				delta.FreeToUseDelta = quantity
			}
		case "OUT":
			if normalizedStatus == "DONE" {
				delta.FreeToUseDelta = -quantity
			}
		}

		if delta.OnHandDelta == 0 && delta.FreeToUseDelta == 0 {
			continue
		}

		effect[key] = delta
	}

	return effect
}

func diffStockEffects(current, next map[stockLevelKey]stockLevelDelta) map[stockLevelKey]stockLevelDelta {
	diff := make(map[stockLevelKey]stockLevelDelta)

	for key, delta := range current {
		diff[key] = stockLevelDelta{
			OnHandDelta:    -delta.OnHandDelta,
			FreeToUseDelta: -delta.FreeToUseDelta,
		}
	}

	for key, delta := range next {
		existing := diff[key]
		existing.OnHandDelta += delta.OnHandDelta
		existing.FreeToUseDelta += delta.FreeToUseDelta
		if existing.OnHandDelta == 0 && existing.FreeToUseDelta == 0 {
			delete(diff, key)
			continue
		}
		diff[key] = existing
	}

	return diff
}

func applyStockEffectTx(tx *sql.Tx, effect map[stockLevelKey]stockLevelDelta) (map[stockLevelKey]stockLevelStateSnapshot, error) {
	stateByKey := make(map[stockLevelKey]stockLevelStateSnapshot)

	if len(effect) == 0 {
		return stateByKey, nil
	}

	keys := make([]stockLevelKey, 0, len(effect))
	for key, delta := range effect {
		if delta.OnHandDelta == 0 && delta.FreeToUseDelta == 0 {
			continue
		}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ProductID == keys[j].ProductID {
			return keys[i].LocationID < keys[j].LocationID
		}
		return keys[i].ProductID < keys[j].ProductID
	})

	for _, key := range keys {
		delta := effect[key]
		snapshot, err := applyProductStockLevelDeltaTx(
			tx,
			key.ProductID,
			key.LocationID,
			delta.OnHandDelta,
			delta.FreeToUseDelta,
		)
		if err != nil {
			return nil, err
		}

		stateByKey[key] = snapshot
	}

	return stateByKey, nil
}

func loadProductStockLevelsForUpdateTx(tx *sql.Tx, productID string) ([]models.ProductStockLevel, error) {
	const selectQuery = `
		SELECT COALESCE(stock_levels, '[]'::jsonb)
		FROM stocks.products
		WHERE id = $1
		FOR UPDATE`

	var rawStockLevels []byte
	if err := tx.QueryRow(selectQuery, productID).Scan(&rawStockLevels); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperationProductInvalid
		}
		return nil, err
	}

	return unmarshalProductStockLevels(rawStockLevels)
}

func saveProductStockLevelsTx(tx *sql.Tx, productID string, levels []models.ProductStockLevel) error {
	updatedInputs := make([]ProductStockLevelInput, 0, len(levels))
	for _, level := range levels {
		updatedInputs = append(updatedInputs, ProductStockLevelInput{
			LocationID:        strings.TrimSpace(level.LocationID),
			OnHandQuantity:    level.OnHandQuantity,
			FreeToUseQuantity: level.FreeToUseQuantity,
		})
	}

	serializedLevels, err := marshalStockLevels(normalizeStockLevels(updatedInputs))
	if err != nil {
		return err
	}

	const updateQuery = `
		UPDATE stocks.products
		SET stock_levels = $2::jsonb, updated_at = NOW()
		WHERE id = $1`

	if _, err := tx.Exec(updateQuery, productID, serializedLevels); err != nil {
		return err
	}

	return nil
}

func applyProductStockLevelDeltaTx(tx *sql.Tx, productID, locationID string, onHandDelta, freeToUseDelta int) (stockLevelStateSnapshot, error) {
	state := stockLevelStateSnapshot{}

	if onHandDelta == 0 && freeToUseDelta == 0 {
		return state, nil
	}

	levels, err := loadProductStockLevelsForUpdateTx(tx, productID)
	if err != nil {
		return state, err
	}

	normalizedLocationID := strings.TrimSpace(locationID)
	levelIndex := -1
	for index := range levels {
		if strings.TrimSpace(levels[index].LocationID) == normalizedLocationID {
			levelIndex = index
			break
		}
	}

	if levelIndex == -1 {
		if onHandDelta < 0 || freeToUseDelta < 0 {
			return state, ErrOperationStockInsufficient
		}
		state.PreviousOnHandQuantity = 0
		state.PreviousFreeToUseQuantity = 0
		state.CurrentOnHandQuantity = onHandDelta
		state.CurrentFreeToUseQuantity = freeToUseDelta
		levels = append(levels, models.ProductStockLevel{
			LocationID:        normalizedLocationID,
			OnHandQuantity:    onHandDelta,
			FreeToUseQuantity: freeToUseDelta,
		})
	} else {
		state.PreviousOnHandQuantity = levels[levelIndex].OnHandQuantity
		state.PreviousFreeToUseQuantity = levels[levelIndex].FreeToUseQuantity
		updatedOnHand := levels[levelIndex].OnHandQuantity + onHandDelta
		updatedFreeToUse := levels[levelIndex].FreeToUseQuantity + freeToUseDelta
		if updatedOnHand < 0 || updatedFreeToUse < 0 {
			return state, ErrOperationStockInsufficient
		}
		levels[levelIndex].OnHandQuantity = updatedOnHand
		levels[levelIndex].FreeToUseQuantity = updatedFreeToUse
		state.CurrentOnHandQuantity = updatedOnHand
		state.CurrentFreeToUseQuantity = updatedFreeToUse
	}

	if err := saveProductStockLevelsTx(tx, productID, levels); err != nil {
		return state, err
	}

	return state, nil
}
