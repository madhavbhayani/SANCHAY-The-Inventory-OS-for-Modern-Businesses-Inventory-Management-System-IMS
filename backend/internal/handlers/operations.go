package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"
)

type OperationsHandler struct {
	operations *repository.OperationsRepo
}

func NewOperationsHandler(db *sql.DB) *OperationsHandler {
	return &OperationsHandler{operations: repository.NewOperationsRepo(db)}
}

type operationsMetaResponse struct {
	Locations []models.OperationLocationOption `json:"locations"`
	Products  []models.OperationProductOption  `json:"products"`
}

type operationsListResponse struct {
	Orders []models.OperationOrder `json:"orders"`
}

type createOperationItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type createOperationRequest struct {
	From          string                       `json:"from"`
	To            string                       `json:"to"`
	LocationID    string                       `json:"location_id"`
	ContactNumber string                       `json:"contact_number"`
	ScheduleDate  string                       `json:"schedule_date"`
	Status        string                       `json:"status"`
	Items         []createOperationItemRequest `json:"items"`
}

// GET /api/operations/meta
func (h *OperationsHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	locations, err := h.operations.ListLocations()
	if err != nil {
		log.Printf("[Operations:GetMeta] locations error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	products, err := h.operations.ListProducts(260)
	if err != nil {
		log.Printf("[Operations:GetMeta] products error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, operationsMetaResponse{
		Locations: locations,
		Products:  products,
	})
}

// GET /api/operations/receipts
func (h *OperationsHandler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	h.listOrdersByType(w, r, "IN")
}

// GET /api/operations/delivery
func (h *OperationsHandler) ListDelivery(w http.ResponseWriter, r *http.Request) {
	h.listOrdersByType(w, r, "OUT")
}

// POST /api/operations/receipts
func (h *OperationsHandler) CreateReceipt(w http.ResponseWriter, r *http.Request) {
	h.createOrderByType(w, r, "IN")
}

// POST /api/operations/delivery
func (h *OperationsHandler) CreateDelivery(w http.ResponseWriter, r *http.Request) {
	h.createOrderByType(w, r, "OUT")
}

// DELETE /api/operations/orders/{id}
func (h *OperationsHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rawID := strings.TrimSpace(r.PathValue("id"))
	if rawID == "" {
		writeError(w, "order id is required", http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || orderID <= 0 {
		writeError(w, "invalid order id", http.StatusBadRequest)
		return
	}

	if err := h.operations.DeleteOrder(orderID); err != nil {
		status, message := mapOperationsError(err)
		if status >= 500 {
			log.Printf("[Operations:Delete] delete error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "operation order deleted"})
}

func (h *OperationsHandler) listOrdersByType(w http.ResponseWriter, r *http.Request, operationType string) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 120
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}

	orders, err := h.operations.ListOrders(operationType, limit)
	if err != nil {
		status, message := mapOperationsError(err)
		if status >= 500 {
			log.Printf("[Operations:List:%s] list error: %v", operationType, err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, operationsListResponse{Orders: orders})
}

func (h *OperationsHandler) createOrderByType(w http.ResponseWriter, r *http.Request, operationType string) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createOperationRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	scheduledDate, err := parseScheduleDate(req.ScheduleDate)
	if err != nil {
		writeError(w, "schedule date must be valid (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = "DRAFT"
	}

	input := repository.OperationCreateInput{
		OperationType: operationType,
		FromParty:     strings.TrimSpace(req.From),
		ToParty:       strings.TrimSpace(req.To),
		LocationID:    strings.TrimSpace(req.LocationID),
		ContactNumber: strings.TrimSpace(req.ContactNumber),
		ScheduledDate: scheduledDate,
		Status:        status,
		Items:         make([]repository.OperationCreateItemInput, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		input.Items = append(input.Items, repository.OperationCreateItemInput{
			ProductID: strings.TrimSpace(item.ProductID),
			Quantity:  item.Quantity,
		})
	}

	if err := validateCreateOperationRequest(operationType, input); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.operations.CreateOrder(input)
	if err != nil {
		statusCode, message := mapOperationsError(err)
		if statusCode >= 500 {
			log.Printf("[Operations:Create:%s] create error: %v", operationType, err)
		}
		writeError(w, message, statusCode)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"order": order})
}

func validateCreateOperationRequest(operationType string, input repository.OperationCreateInput) error {
	if input.LocationID == "" {
		return errors.New("location is required")
	}

	if len(input.Items) == 0 {
		return errors.New("at least one product item is required")
	}

	for _, item := range input.Items {
		if item.ProductID == "" {
			return errors.New("each product row must have a product")
		}
		if item.Quantity <= 0 {
			return errors.New("item quantity must be greater than zero")
		}
	}

	if operationType == "IN" {
		if input.FromParty == "" {
			return errors.New("vendor name is required in receipt")
		}
		if input.ContactNumber == "" {
			return errors.New("contact number is required in receipt")
		}
	} else {
		if input.ToParty == "" {
			return errors.New("vendor name is required in delivery")
		}
	}

	return nil
}

func parseScheduleDate(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, errors.New("empty schedule date")
	}

	if dateValue, err := time.Parse("2006-01-02", trimmed); err == nil {
		return dateValue, nil
	}

	if dateValue, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return dateValue, nil
	}

	return time.Time{}, errors.New("invalid schedule date")
}

func mapOperationsError(err error) (int, string) {
	switch {
	case errors.Is(err, repository.ErrOperationTypeInvalid):
		return http.StatusBadRequest, "operation type is invalid"
	case errors.Is(err, repository.ErrOperationStatusInvalid):
		return http.StatusBadRequest, "status is invalid for this operation"
	case errors.Is(err, repository.ErrOperationItemsRequired):
		return http.StatusBadRequest, "at least one valid product line is required"
	case errors.Is(err, repository.ErrOperationLocationInvalid):
		return http.StatusBadRequest, "invalid location reference"
	case errors.Is(err, repository.ErrOperationProductInvalid):
		return http.StatusBadRequest, "invalid product reference"
	case errors.Is(err, repository.ErrOperationNotFound):
		return http.StatusNotFound, "operation order not found"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
