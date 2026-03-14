package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	users    *repository.UserRepo
	history  *repository.HistoryRepo
	settings *repository.SettingsRepo
}

func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{
		users:    repository.NewUserRepo(db),
		history:  repository.NewHistoryRepo(db),
		settings: repository.NewSettingsRepo(db),
	}
}

type settingsOverviewResponse struct {
	User struct {
		ID      string `json:"id"`
		LoginID string `json:"login_id"`
		Email   string `json:"email"`
	} `json:"user"`
	Warehouses   []models.Warehouse        `json:"warehouses"`
	Locations    []models.Location         `json:"locations"`
	LoginHistory []models.LoginHistoryItem `json:"login_history"`
}

type createWarehouseRequest struct {
	Name        string `json:"name"`
	ShortCode   string `json:"short_code"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

type createLocationRequest struct {
	Name         string   `json:"name"`
	ShortCode    string   `json:"short_code"`
	WarehouseIDs []string `json:"warehouse_ids"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// GET /api/settings
func (h *SettingsHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	userID, err := authUserIDFromRequest(r)
	if err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := h.users.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("[Settings:GetOverview] user lookup error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	warehouses, err := h.settings.ListWarehouses()
	if err != nil {
		log.Printf("[Settings:GetOverview] warehouses error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	locations, err := h.settings.ListLocations()
	if err != nil {
		log.Printf("[Settings:GetOverview] locations error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	history, err := h.history.ListByUserID(userID, 25)
	if err != nil {
		log.Printf("[Settings:GetOverview] history error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var response settingsOverviewResponse
	response.User.ID = user.ID
	response.User.LoginID = user.LoginID
	response.User.Email = user.Email
	response.Warehouses = warehouses
	response.Locations = locations
	response.LoginHistory = history

	writeJSON(w, http.StatusOK, response)
}

// POST /api/settings/warehouses
func (h *SettingsHandler) CreateWarehouse(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createWarehouseRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ShortCode = strings.ToUpper(strings.TrimSpace(req.ShortCode))
	req.Address = strings.TrimSpace(req.Address)
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" || req.ShortCode == "" {
		writeError(w, "warehouse name and short code are required", http.StatusBadRequest)
		return
	}

	warehouse, err := h.settings.CreateWarehouse(req.Name, req.ShortCode, req.Address, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrWarehouseShortCodeTaken) {
			writeError(w, "warehouse short code already exists", http.StatusConflict)
			return
		}
		log.Printf("[Settings:CreateWarehouse] create error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"warehouse": warehouse})
}

// POST /api/settings/locations
func (h *SettingsHandler) CreateLocation(w http.ResponseWriter, r *http.Request) {
	if _, err := authUserIDFromRequest(r); err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createLocationRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.ShortCode = strings.ToUpper(strings.TrimSpace(req.ShortCode))
	req.WarehouseIDs = uniqueTrimmed(req.WarehouseIDs)

	if req.Name == "" || req.ShortCode == "" {
		writeError(w, "location name and short code are required", http.StatusBadRequest)
		return
	}
	if len(req.WarehouseIDs) == 0 {
		writeError(w, "select at least one warehouse", http.StatusBadRequest)
		return
	}

	location, err := h.settings.CreateLocation(req.Name, req.ShortCode, req.WarehouseIDs)
	if err != nil {
		if errors.Is(err, repository.ErrLocationShortCodeTaken) {
			writeError(w, "location short code already exists", http.StatusConflict)
			return
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			writeError(w, "one or more selected warehouses are invalid", http.StatusBadRequest)
			return
		}
		log.Printf("[Settings:CreateLocation] create error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"location": location})
}

// POST /api/settings/change-password
func (h *SettingsHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, err := authUserIDFromRequest(r)
	if err != nil {
		writeError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req changePasswordRequest
	if err := jsonDecode(r, &req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, "current and new password are required", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, "new password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	user, err := h.users.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("[Settings:ChangePassword] user lookup error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		writeError(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		log.Printf("[Settings:ChangePassword] bcrypt error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.users.UpdatePassword(userID, string(hash)); err != nil {
		log.Printf("[Settings:ChangePassword] update error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

func authUserIDFromRequest(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("missing bearer token")
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return "", errors.New("missing bearer token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	subject, _ := claims["sub"].(string)
	if strings.TrimSpace(subject) == "" {
		return "", errors.New("missing user id in token")
	}

	return subject, nil
}

func uniqueTrimmed(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func jsonDecode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}
