package models

import "time"

// OperationLocationOption is a location selector option shown in operations forms.
type OperationLocationOption struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ShortCode      string   `json:"short_code"`
	WarehouseNames []string `json:"warehouse_names"`
}

// OperationProductOption is a product selector option shown in operations forms.
type OperationProductOption struct {
	ID                string `json:"id"`
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	CategoryName      string `json:"category_name"`
	LocationID        string `json:"location_id"`
	LocationName      string `json:"location_name"`
	LocationShortCode string `json:"location_short_code"`
	OnHandQuantity    int    `json:"on_hand_quantity"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
}

// OperationOrderItem is one product line attached to an operation order.
type OperationOrderItem struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductSKU  string `json:"product_sku"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

// OperationOrder is either a receipt (IN) or delivery (OUT) order.
type OperationOrder struct {
	ID                 int64                `json:"id"`
	ReferenceSequence  int64                `json:"reference_sequence"`
	ReferenceNumber    string               `json:"reference_number"`
	OperationType      string               `json:"operation_type"`
	FromParty          string               `json:"from_party"`
	ToParty            string               `json:"to_party"`
	LocationID         string               `json:"location_id"`
	LocationName       string               `json:"location_name"`
	LocationShortCode  string               `json:"location_short_code"`
	WarehouseShortCode string               `json:"warehouse_short_code"`
	WarehouseNames     []string             `json:"warehouse_names"`
	ContactNumber      string               `json:"contact_number"`
	ScheduledDate      time.Time            `json:"scheduled_date"`
	Status             string               `json:"status"`
	Items              []OperationOrderItem `json:"items"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}
