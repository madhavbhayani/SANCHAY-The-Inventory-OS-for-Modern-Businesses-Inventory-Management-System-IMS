package models

import "time"

type StockLedgerEntry struct {
	ID                        string    `json:"id"`
	EventType                 string    `json:"event_type"`
	OperationType             string    `json:"operation_type"`
	OrderID                   int64     `json:"order_id"`
	ReferenceNumber           string    `json:"reference_number"`
	ProductID                 string    `json:"product_id"`
	SKU                       string    `json:"sku"`
	ProductName               string    `json:"product_name"`
	CategoryName              string    `json:"category_name"`
	LocationID                string    `json:"location_id"`
	LocationName              string    `json:"location_name"`
	LocationShortCode         string    `json:"location_short_code"`
	FromLocationID            string    `json:"from_location_id"`
	FromLocationName          string    `json:"from_location_name"`
	FromLocationShortCode     string    `json:"from_location_short_code"`
	ToLocationID              string    `json:"to_location_id"`
	ToLocationName            string    `json:"to_location_name"`
	ToLocationShortCode       string    `json:"to_location_short_code"`
	OnHandDelta               int       `json:"on_hand_delta"`
	FreeToUseDelta            int       `json:"free_to_use_delta"`
	PreviousOnHandQuantity    int       `json:"previous_on_hand_quantity"`
	CurrentOnHandQuantity     int       `json:"current_on_hand_quantity"`
	PreviousFreeToUseQuantity int       `json:"previous_free_to_use_quantity"`
	CurrentFreeToUseQuantity  int       `json:"current_free_to_use_quantity"`
	PreviousStatus            string    `json:"previous_status"`
	CurrentStatus             string    `json:"current_status"`
	Reason                    string    `json:"reason"`
	CreatedAt                 time.Time `json:"created_at"`
}
