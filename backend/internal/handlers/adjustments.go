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

type AdjustmentsHandler struct {
	adjustments *repository.AdjustmentsRepo
	stocks      *repository.StockRepo
}

func NewAdjustmentsHandler(db *sql.DB) *AdjustmentsHandler {
	return &AdjustmentsHandler{
		adjustments: repository.NewAdjustmentsRepo(db),
		stocks:      repository.NewStockRepo(db),
	}
}

type adjustmentsOverviewResponse struct {
	Locations []models.ProductLocation        `json:"locations"`
	Rows      []models.AdjustmentInventoryRow `json:"rows"`
	History   []models.AdjustmentHistoryEntry `json:"history"`
}

type adjustmentTransferRequest struct {
	ProductID      string `json:"product_id"`
	FromLocationID string `json:"from_location_id"`
	ToLocationID   string `json:"to_location_id"`
	Quantity       int    `json:"quantity"`
	Reason         string `json:"reason"`
}

type adjustmentQuantityRequest struct {
	ProductID         string `json:"product_id"`
	LocationID        string `json:"location_id"`
	FreeToUseQuantity int    `json:"free_to_use_quantity"`
	Reason            string `json:"reason"`
}

// GET /api/operations/adjustments
func (h *AdjustmentsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 320
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	locations, err := h.stocks.ListLocations()
	if err != nil {
		log.Printf("[Adjustments:GetOverview] locations error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	rows, err := h.adjustments.ListInventoryRows(limit)
	if err != nil {
		log.Printf("[Adjustments:GetOverview] rows error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	history, err := h.adjustments.ListHistory(120)
	if err != nil {
		log.Printf("[Adjustments:GetOverview] history error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, adjustmentsOverviewResponse{
		Locations: locations,
		Rows:      rows,
		History:   history,
	})
}

// POST /api/operations/adjustments/transfer
func (h *AdjustmentsHandler) TransferStock(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req adjustmentTransferRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.adjustments.TransferStock(repository.AdjustmentTransferInput{
		ProductID:      strings.TrimSpace(req.ProductID),
		FromLocationID: strings.TrimSpace(req.FromLocationID),
		ToLocationID:   strings.TrimSpace(req.ToLocationID),
		Quantity:       req.Quantity,
		Reason:         strings.TrimSpace(req.Reason),
	})
	if err != nil {
		status, message := mapAdjustmentsError(err)
		if status >= 500 {
			log.Printf("[Adjustments:TransferStock] transfer error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "stock transferred successfully"})
}

// POST /api/operations/adjustments/quantity
func (h *AdjustmentsHandler) AdjustQuantity(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req adjustmentQuantityRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.adjustments.AdjustFreeToUseQuantity(repository.AdjustmentQuantityInput{
		ProductID:         strings.TrimSpace(req.ProductID),
		LocationID:        strings.TrimSpace(req.LocationID),
		FreeToUseQuantity: req.FreeToUseQuantity,
		Reason:            strings.TrimSpace(req.Reason),
	})
	if err != nil {
		status, message := mapAdjustmentsError(err)
		if status >= 500 {
			log.Printf("[Adjustments:AdjustQuantity] adjust error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "free-to-use quantity updated successfully"})
}

func mapAdjustmentsError(err error) (int, string) {
	switch {
	case errors.Is(err, repository.ErrAdjustmentProductInvalid):
		return http.StatusBadRequest, "invalid product reference"
	case errors.Is(err, repository.ErrAdjustmentLocationInvalid):
		return http.StatusBadRequest, "invalid source location"
	case errors.Is(err, repository.ErrAdjustmentDestinationRequired):
		return http.StatusBadRequest, "destination location is required"
	case errors.Is(err, repository.ErrAdjustmentDestinationInvalid):
		return http.StatusBadRequest, "invalid destination location"
	case errors.Is(err, repository.ErrAdjustmentQuantityInvalid):
		return http.StatusBadRequest, "quantity must be a non-negative whole number"
	case errors.Is(err, repository.ErrAdjustmentReasonRequired):
		return http.StatusBadRequest, "reason is required"
	case errors.Is(err, repository.ErrAdjustmentNoChange):
		return http.StatusBadRequest, "no stock change requested"
	case errors.Is(err, repository.ErrAdjustmentStockInsufficient):
		return http.StatusBadRequest, "insufficient free-to-use stock at the source location"
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23503":
			return http.StatusBadRequest, "invalid product or location reference"
		case "23514":
			return http.StatusBadRequest, "invalid adjustment values"
		}
	}

	return http.StatusInternalServerError, "internal server error"
}
