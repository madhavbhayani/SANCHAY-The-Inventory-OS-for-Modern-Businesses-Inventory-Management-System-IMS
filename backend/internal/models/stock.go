package models

import "time"

// ProductCategory is a category option for inventory items.
type ProductCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ProductLocation is a location option with its mapped warehouses.
type ProductLocation struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ShortCode      string   `json:"short_code"`
	WarehouseNames []string `json:"warehouse_names"`
}

// ProductStockLevel is the stock quantity for one product at one location.
type ProductStockLevel struct {
	LocationID        string   `json:"location_id"`
	LocationName      string   `json:"location_name"`
	LocationShortCode string   `json:"location_short_code"`
	WarehouseNames    []string `json:"warehouse_names"`
	OnHandQuantity    int      `json:"on_hand_quantity"`
	FreeToUseQuantity int      `json:"free_to_use_quantity"`
}

// Product is the inventory entity stored in stocks.products.
type Product struct {
	ID                string              `json:"id"`
	SKU               string              `json:"sku"`
	Name              string              `json:"name"`
	Cost              float64             `json:"cost"`
	OnHandQuantity    int                 `json:"on_hand_quantity"`
	FreeToUseQuantity int                 `json:"free_to_use_quantity"`
	CategoryID        string              `json:"category_id"`
	CategoryName      string              `json:"category_name"`
	LocationID        string              `json:"location_id"`
	LocationName      string              `json:"location_name"`
	LocationShortCode string              `json:"location_short_code"`
	WarehouseNames    []string            `json:"warehouse_names"`
	StockLevels       []ProductStockLevel `json:"stock_levels"`
	Description       string              `json:"description"`
	UpdatedAt         time.Time           `json:"updated_at"`
}
