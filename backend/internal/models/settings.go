package models

import "time"

// Warehouse is a storage entity where stock physically exists.
type Warehouse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ShortCode   string    `json:"short_code"`
	Address     string    `json:"address"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Location maps logical or physical sub-areas and can belong to multiple warehouses.
type Location struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ShortCode      string    `json:"short_code"`
	WarehouseIDs   []string  `json:"warehouse_ids"`
	WarehouseNames []string  `json:"warehouse_names"`
	CreatedAt      time.Time `json:"created_at"`
}

// LoginHistoryItem is a UI-friendly representation of a login attempt.
type LoginHistoryItem struct {
	ID            string    `json:"id"`
	IPAddress     string    `json:"ip_address"`
	Browser       string    `json:"browser"`
	OS            string    `json:"os"`
	Success       bool      `json:"success"`
	FailureReason string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
