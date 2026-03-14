package handlers

import (
	"database/sql"
	"errors"
	"fmt"
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

type operationDetailResponse struct {
	Order *models.OperationOrder `json:"order"`
}

type operationValidateResponse struct {
	Order           *models.OperationOrder `json:"order"`
	AllItemsInStock bool                   `json:"all_items_in_stock"`
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

type updateOperationRequest struct {
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

// GET /api/operations/orders/{operationType}/{referenceNumber}
func (h *OperationsHandler) GetOrderDetail(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	opType := normalizeOperationTypePathValue(r.PathValue("operationType"))
	if opType == "" {
		writeError(w, "invalid operation type", http.StatusBadRequest)
		return
	}

	referenceNumber := strings.TrimSpace(r.PathValue("referenceNumber"))
	if referenceNumber == "" {
		writeError(w, "reference number is required", http.StatusBadRequest)
		return
	}

	order, err := h.operations.GetOrderByReference(opType, referenceNumber)
	if err != nil {
		status, message := mapOperationsError(err)
		if status >= 500 {
			log.Printf("[Operations:GetOrderDetail] get error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, operationDetailResponse{Order: order})
}

// PUT /api/operations/orders/{operationType}/{referenceNumber}
func (h *OperationsHandler) UpdateOrderDetail(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	opType := normalizeOperationTypePathValue(r.PathValue("operationType"))
	if opType == "" {
		writeError(w, "invalid operation type", http.StatusBadRequest)
		return
	}

	referenceNumber := strings.TrimSpace(r.PathValue("referenceNumber"))
	if referenceNumber == "" {
		writeError(w, "reference number is required", http.StatusBadRequest)
		return
	}

	var req updateOperationRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	scheduledDate, err := parseScheduleDate(req.ScheduleDate)
	if err != nil {
		writeError(w, "schedule date must be valid (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if err := validateScheduleDateNotPast(scheduledDate); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status == "" {
		status = "DRAFT"
	}

	input := repository.OperationUpdateInput{
		OperationType:   opType,
		ReferenceNumber: referenceNumber,
		FromParty:       strings.TrimSpace(req.From),
		ToParty:         strings.TrimSpace(req.To),
		LocationID:      strings.TrimSpace(req.LocationID),
		ContactNumber:   strings.TrimSpace(req.ContactNumber),
		ScheduledDate:   scheduledDate,
		Status:          status,
		Items:           make([]repository.OperationCreateItemInput, 0, len(req.Items)),
	}

	for _, item := range req.Items {
		input.Items = append(input.Items, repository.OperationCreateItemInput{
			ProductID: strings.TrimSpace(item.ProductID),
			Quantity:  item.Quantity,
		})
	}

	if err := h.normalizeOrderPartiesAndContact(&input); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateUpdateOperationRequest(input); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.operations.UpdateOrderByReference(input)
	if err != nil {
		statusCode, message := mapOperationsError(err)
		if statusCode >= 500 {
			log.Printf("[Operations:UpdateOrderDetail] update error: %v", err)
		}
		writeError(w, message, statusCode)
		return
	}

	writeJSON(w, http.StatusOK, operationDetailResponse{Order: order})
}

// POST /api/operations/orders/{operationType}/{referenceNumber}/validate
func (h *OperationsHandler) ValidateOrder(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	opType := normalizeOperationTypePathValue(r.PathValue("operationType"))
	if opType == "" {
		writeError(w, "invalid operation type", http.StatusBadRequest)
		return
	}

	referenceNumber := strings.TrimSpace(r.PathValue("referenceNumber"))
	if referenceNumber == "" {
		writeError(w, "reference number is required", http.StatusBadRequest)
		return
	}

	order, allItemsInStock, err := h.operations.ValidateOrderByReference(opType, referenceNumber)
	if err != nil {
		status, message := mapOperationsError(err)
		if status >= 500 {
			log.Printf("[Operations:ValidateOrder] validate error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, operationValidateResponse{
		Order:           order,
		AllItemsInStock: allItemsInStock,
	})
}

// POST /api/operations/orders/{operationType}/{referenceNumber}/cancel
func (h *OperationsHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	opType := normalizeOperationTypePathValue(r.PathValue("operationType"))
	if opType == "" {
		writeError(w, "invalid operation type", http.StatusBadRequest)
		return
	}

	referenceNumber := strings.TrimSpace(r.PathValue("referenceNumber"))
	if referenceNumber == "" {
		writeError(w, "reference number is required", http.StatusBadRequest)
		return
	}

	order, err := h.operations.UpdateOrderStatusByReference(opType, referenceNumber, "CANCELLED")
	if err != nil {
		status, message := mapOperationsError(err)
		if status >= 500 {
			log.Printf("[Operations:CancelOrder] cancel error: %v", err)
		}
		writeError(w, message, status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "operation order cancelled",
		"order":   order,
	})
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
	if err := validateScheduleDateNotPast(scheduledDate); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
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

	if err := h.normalizeOrderPartiesAndContactForCreate(&input); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
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

func (h *OperationsHandler) normalizeOrderPartiesAndContactForCreate(input *repository.OperationCreateInput) error {
	if input == nil {
		return errors.New("invalid order input")
	}

	contactNumber, err := normalizeIndianPhone(input.ContactNumber)
	if err != nil {
		return err
	}
	input.ContactNumber = contactNumber

	if input.OperationType == "IN" {
		if strings.TrimSpace(input.ToParty) == "" {
			locationLabel, err := h.resolveLocationLabel(input.LocationID)
			if err != nil {
				return err
			}
			input.ToParty = locationLabel
		}
	}

	return nil
}

func (h *OperationsHandler) normalizeOrderPartiesAndContact(input *repository.OperationUpdateInput) error {
	if input == nil {
		return errors.New("invalid order input")
	}

	contactNumber, err := normalizeIndianPhone(input.ContactNumber)
	if err != nil {
		return err
	}
	input.ContactNumber = contactNumber

	if input.OperationType == "IN" {
		if strings.TrimSpace(input.ToParty) == "" {
			locationLabel, err := h.resolveLocationLabel(input.LocationID)
			if err != nil {
				return err
			}
			input.ToParty = locationLabel
		}
	}

	return nil
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

	if strings.ToUpper(strings.TrimSpace(input.Status)) == "CANCELLED" {
		return errors.New("new order cannot start with cancelled status")
	}

	if operationType == "IN" {
		if strings.TrimSpace(input.FromParty) == "" {
			return errors.New("vendor name is required in receipt")
		}
		if strings.TrimSpace(input.ToParty) == "" {
			return errors.New("destination location is required in receipt")
		}
	} else {
		if strings.TrimSpace(input.ToParty) == "" {
			return errors.New("vendor name is required in delivery")
		}
	}

	if strings.TrimSpace(input.ContactNumber) == "" {
		return errors.New("contact number is required")
	}

	return nil
}

func validateUpdateOperationRequest(input repository.OperationUpdateInput) error {
	if strings.TrimSpace(input.LocationID) == "" {
		return errors.New("location is required")
	}

	if len(input.Items) == 0 {
		return errors.New("at least one product item is required")
	}

	for _, item := range input.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return errors.New("each product row must have a product")
		}
		if item.Quantity <= 0 {
			return errors.New("item quantity must be greater than zero")
		}
	}

	if input.OperationType == "IN" {
		if strings.TrimSpace(input.FromParty) == "" {
			return errors.New("vendor name is required in receipt")
		}
		if strings.TrimSpace(input.ToParty) == "" {
			return errors.New("destination location is required in receipt")
		}
	} else if strings.TrimSpace(input.ToParty) == "" {
		return errors.New("vendor name is required in delivery")
	}

	if strings.TrimSpace(input.ContactNumber) == "" {
		return errors.New("contact number is required")
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

func validateScheduleDateNotPast(dateValue time.Time) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	scheduleDate := time.Date(dateValue.Year(), dateValue.Month(), dateValue.Day(), 0, 0, 0, 0, now.Location())
	if scheduleDate.Before(today) {
		return errors.New("schedule date must be today or a future date")
	}
	return nil
}

func normalizeIndianPhone(raw string) (string, error) {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return "", errors.New("contact number is required")
	}

	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	cleaned = replacer.Replace(cleaned)

	if strings.HasPrefix(cleaned, "+91") {
		cleaned = cleaned[3:]
	} else if strings.HasPrefix(cleaned, "91") && len(cleaned) > 10 {
		cleaned = cleaned[2:]
	}

	for _, ch := range cleaned {
		if ch < '0' || ch > '9' {
			return "", errors.New("contact number must contain only digits")
		}
	}

	if len(cleaned) != 10 {
		return "", errors.New("contact number must be 10 digits")
	}

	return fmt.Sprintf("+91%s", cleaned), nil
}

func normalizeOperationTypePathValue(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "IN" || normalized == "OUT" {
		return normalized
	}
	return ""
}

func (h *OperationsHandler) resolveLocationLabel(locationID string) (string, error) {
	locationID = strings.TrimSpace(locationID)
	if locationID == "" {
		return "", errors.New("location is required")
	}

	locations, err := h.operations.ListLocations()
	if err != nil {
		return "", err
	}

	for _, location := range locations {
		if location.ID == locationID {
			return formatOperationLocationLabel(location), nil
		}
	}

	return "", errors.New("invalid location reference")
}

func formatOperationLocationLabel(location models.OperationLocationOption) string {
	warehouses := strings.TrimSpace(strings.Join(location.WarehouseNames, ", "))
	if warehouses == "" {
		return fmt.Sprintf("%s (%s)", location.Name, location.ShortCode)
	}
	return fmt.Sprintf("%s (%s) - %s", location.Name, location.ShortCode, warehouses)
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
	case errors.Is(err, repository.ErrOperationStockInsufficient):
		return http.StatusBadRequest, "insufficient stock for this delivery"
	case errors.Is(err, repository.ErrOperationFinalized):
		return http.StatusBadRequest, "done or cancelled orders cannot be changed"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}
