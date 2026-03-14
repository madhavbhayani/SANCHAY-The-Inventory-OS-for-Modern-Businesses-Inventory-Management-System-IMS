package models

import "time"

// AdjustmentInventoryRow is one product stock row scoped to a location.
type AdjustmentInventoryRow struct {
	ProductID         string   `json:"product_id"`
	SKU               string   `json:"sku"`
	Name              string   `json:"name"`
	CategoryName      string   `json:"category_name"`
	LocationID        string   `json:"location_id"`
	LocationName      string   `json:"location_name"`
	LocationShortCode string   `json:"location_short_code"`
	WarehouseNames    []string `json:"warehouse_names"`
	OnHandQuantity    int      `json:"on_hand_quantity"`
	FreeToUseQuantity int      `json:"free_to_use_quantity"`
}

// AdjustmentHistoryEntry records a manual stock transfer or quantity correction.
type AdjustmentHistoryEntry struct {
	ID                        string    `json:"id"`
	ActionType                string    `json:"action_type"`
	ProductID                 string    `json:"product_id"`
	SKU                       string    `json:"sku"`
	Name                      string    `json:"name"`
	FromLocationID            string    `json:"from_location_id"`
	FromLocationName          string    `json:"from_location_name"`
	FromLocationShortCode     string    `json:"from_location_short_code"`
	ToLocationID              string    `json:"to_location_id"`
	ToLocationName            string    `json:"to_location_name"`
	ToLocationShortCode       string    `json:"to_location_short_code"`
	QuantityChanged           int       `json:"quantity_changed"`
	PreviousFreeToUseQuantity int       `json:"previous_free_to_use_quantity"`
	UpdatedFreeToUseQuantity  int       `json:"updated_free_to_use_quantity"`
	Reason                    string    `json:"reason"`
	CreatedAt                 time.Time `json:"created_at"`
}
