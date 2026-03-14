package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"
)

type MoveHistoryHandler struct {
	history *repository.MoveHistoryRepo
}

func NewMoveHistoryHandler(db *sql.DB) *MoveHistoryHandler {
	return &MoveHistoryHandler{history: repository.NewMoveHistoryRepo(db)}
}

type moveHistoryListResponse struct {
	Entries []models.StockLedgerEntry `json:"entries"`
}

// GET /api/move-history
func (h *MoveHistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 200
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	filter := repository.MoveHistoryListFilter{
		Limit:     limit,
		EventType: strings.TrimSpace(r.URL.Query().Get("event_type")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		Query:     strings.TrimSpace(r.URL.Query().Get("q")),
	}

	if fromRaw := strings.TrimSpace(r.URL.Query().Get("from_date")); fromRaw != "" {
		if parsed, err := time.Parse("2006-01-02", fromRaw); err == nil {
			filter.FromDate = &parsed
		}
	}

	if toRaw := strings.TrimSpace(r.URL.Query().Get("to_date")); toRaw != "" {
		if parsed, err := time.Parse("2006-01-02", toRaw); err == nil {
			filter.ToDate = &parsed
		}
	}

	entries, err := h.history.ListStockLedger(filter)
	if err != nil {
		log.Printf("[MoveHistory:List] query error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, moveHistoryListResponse{Entries: entries})
}
