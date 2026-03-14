package models

type DashboardOperationStats struct {
	CurrentOrders int `json:"current_orders"`
	Ready         int `json:"ready"`
	Done          int `json:"done"`
	Late          int `json:"late"`
}

type DashboardStockByLocation struct {
	LocationID        string `json:"location_id"`
	LocationName      string `json:"location_name"`
	LocationShortCode string `json:"location_short_code"`
	OnHandQuantity    int    `json:"on_hand_quantity"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
}

type DashboardCategoryFreeToUse struct {
	CategoryID        string `json:"category_id"`
	CategoryName      string `json:"category_name"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
}

type DashboardOverview struct {
	Receipts            DashboardOperationStats      `json:"receipts"`
	Delivery            DashboardOperationStats      `json:"delivery"`
	StockByLocation     []DashboardStockByLocation   `json:"stock_by_location"`
	FreeToUseByCategory []DashboardCategoryFreeToUse `json:"free_to_use_by_category"`
}
