package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"
)

type DashboardHandler struct {
	dashboard *repository.DashboardRepo
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{dashboard: repository.NewDashboardRepo(db)}
}

// GET /api/dashboard/overview
func (h *DashboardHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	overview, err := h.dashboard.GetOverview()
	if err != nil {
		log.Printf("[Dashboard:GetOverview] query error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, overview)
}
