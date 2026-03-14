package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"

	"github.com/lib/pq"
)

type StockHandler struct {
	stocks *repository.StockRepo
}

func NewStockHandler(db *sql.DB) *StockHandler {
	return &StockHandler{stocks: repository.NewStockRepo(db)}
}

type stockMetadataResponse struct {
	Categories []models.ProductCategory `json:"categories"`
	Locations  []models.ProductLocation `json:"locations"`
}

type stockProductsResponse struct {
	Products []models.Product `json:"products"`
}

type stockLevelUpsertRequest struct {
	LocationID        string `json:"location_id"`
	OnHandQuantity    int    `json:"on_hand_quantity"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
}

type productUpsertRequest struct {
	Name              string                    `json:"name"`
	Cost              float64                   `json:"cost"`
	CategoryID        string                    `json:"category_id"`
	Description       string                    `json:"description"`
	StockLevels       []stockLevelUpsertRequest `json:"stock_levels"`
	OnHandQuantity    int                       `json:"on_hand_quantity"`
	FreeToUseQuantity int                       `json:"free_to_use_quantity"`
	LocationID        string                    `json:"location_id"`
}

type createCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GET /api/stocks/meta
func (h *StockHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	categories, err := h.stocks.ListCategories()
	if err != nil {
		log.Printf("[Stock:GetMeta] categories error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	locations, err := h.stocks.ListLocations()
	if err != nil {
		log.Printf("[Stock:GetMeta] locations error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stockMetadataResponse{
		Categories: categories,
		Locations:  locations,
	})
}

// POST /api/stocks/categories
func (h *StockHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createCategoryRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)
	if name == "" {
		writeError(w, "category name is required", http.StatusBadRequest)
		return
	}

	category, err := h.stocks.CreateCategory(name, description)
	if err != nil {
		status, message := mapStockError(err)
		if status >= 500 {
			log.Printf("[Stock:CreateCategory] create error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"category": category})
}

// GET /api/stocks/products?q=<search>&limit=<n>
func (h *StockHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 120
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	products, err := h.stocks.ListProducts(search, limit)
	if err != nil {
		log.Printf("[Stock:ListProducts] list error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, stockProductsResponse{Products: products})
}

// POST /api/stocks/products
func (h *StockHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req productUpsertRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload, err := validateProductPayload(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	product, err := h.stocks.CreateProduct(payload)
	if err != nil {
		status, message := mapStockError(err)
		if status >= 500 {
			log.Printf("[Stock:CreateProduct] create error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"product": product})
}

// PUT /api/stocks/products/{id}
func (h *StockHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	productID := strings.TrimSpace(r.PathValue("id"))
	if productID == "" {
		writeError(w, "product id is required", http.StatusBadRequest)
		return
	}

	var req productUpsertRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payload, err := validateProductPayload(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	product, err := h.stocks.UpdateProduct(productID, payload)
	if err != nil {
		status, message := mapStockError(err)
		if status >= 500 {
			log.Printf("[Stock:UpdateProduct] update error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"product": product})
}

// DELETE /api/stocks/products/{id}
func (h *StockHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	productID := strings.TrimSpace(r.PathValue("id"))
	if productID == "" {
		writeError(w, "product id is required", http.StatusBadRequest)
		return
	}

	if err := h.stocks.DeleteProduct(productID); err != nil {
		status, message := mapStockError(err)
		if status >= 500 {
			log.Printf("[Stock:DeleteProduct] delete error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "product deleted"})
}

func validateProductPayload(req productUpsertRequest) (repository.ProductUpsertInput, error) {
	payload := repository.ProductUpsertInput{
		Name:        strings.TrimSpace(req.Name),
		Cost:        req.Cost,
		CategoryID:  strings.TrimSpace(req.CategoryID),
		Description: strings.TrimSpace(req.Description),
		StockLevels: make([]repository.ProductStockLevelInput, 0),
	}

	if payload.Name == "" {
		return payload, errors.New("product name is required")
	}
	if payload.Cost < 0 {
		return payload, errors.New("cost cannot be negative")
	}
	if payload.CategoryID == "" {
		return payload, errors.New("product category is required")
	}

	if len(req.StockLevels) > 0 {
		for _, level := range req.StockLevels {
			payload.StockLevels = append(payload.StockLevels, repository.ProductStockLevelInput{
				LocationID:        strings.TrimSpace(level.LocationID),
				OnHandQuantity:    level.OnHandQuantity,
				FreeToUseQuantity: level.FreeToUseQuantity,
			})
		}
	} else {
		legacyLocationID := strings.TrimSpace(req.LocationID)
		if legacyLocationID != "" {
			payload.StockLevels = append(payload.StockLevels, repository.ProductStockLevelInput{
				LocationID:        legacyLocationID,
				OnHandQuantity:    req.OnHandQuantity,
				FreeToUseQuantity: req.FreeToUseQuantity,
			})
		}
	}

	if len(payload.StockLevels) == 0 {
		return payload, errors.New("at least one stock location is required")
	}

	for _, level := range payload.StockLevels {
		if level.LocationID == "" {
			return payload, errors.New("location is required for each stock row")
		}
		if level.OnHandQuantity < 0 {
			return payload, errors.New("on hand quantity cannot be negative")
		}
		if level.FreeToUseQuantity < 0 {
			return payload, errors.New("free to use quantity cannot be negative")
		}
	}

	return payload, nil
}

func mapStockError(err error) (int, string) {
	if errors.Is(err, repository.ErrProductNotFound) {
		return http.StatusNotFound, "product not found"
	}
	if errors.Is(err, repository.ErrCategoryNameTaken) {
		return http.StatusConflict, "category name already exists"
	}
	if errors.Is(err, repository.ErrProductStockLevelsRequired) {
		return http.StatusBadRequest, "at least one stock location is required"
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23503":
			return http.StatusBadRequest, "invalid category or location reference"
		case "23514":
			return http.StatusBadRequest, "invalid quantity or cost values"
		case "23505":
			return http.StatusConflict, "duplicate unique value in request"
		}
	}

	return http.StatusInternalServerError, "internal server error"
}
