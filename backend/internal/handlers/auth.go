package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// jwtSecret must be overridden via an environment variable in production.
const jwtSecret = "sanchay-ims-dev-secret-change-in-production"
const bcryptCost = 12

// AuthHandler holds repositories for auth operations.
type AuthHandler struct {
	users   *repository.UserRepo
	history *repository.HistoryRepo
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{
		users:   repository.NewUserRepo(db),
		history: repository.NewHistoryRepo(db),
	}
}

// ── Request / Response types ──────────────────────────────────────────────────

type signupRequest struct {
	LoginID  string `json:"login_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Identifier string `json:"identifier"` // login_id or email
	Password   string `json:"password"`
}

type authResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /api/auth/signup
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.LoginID = strings.TrimSpace(req.LoginID)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.LoginID == "" || req.Email == "" || req.Password == "" {
		writeError(w, "all fields are required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		log.Printf("[Signup] bcrypt error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.users.Create(req.LoginID, req.Email, string(hash))
	if err != nil {
		switch err {
		case repository.ErrLoginIDTaken:
			writeError(w, "Login ID is already taken", http.StatusConflict)
		case repository.ErrEmailTaken:
			writeError(w, "Email is already registered", http.StatusConflict)
		default:
			log.Printf("[Signup] DB error: %v", err)
			writeError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	token, err := generateJWT(user.ID, user.LoginID)
	if err != nil {
		log.Printf("[Signup] JWT error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

// POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Identifier = strings.TrimSpace(req.Identifier)

	// Collect client metadata for login_history (captured before any DB work).
	ip := extractIP(r)
	ua := r.Header.Get("User-Agent")
	browser, osName := parseUserAgent(ua)

	user, err := h.users.FindByLoginIDOrEmail(req.Identifier)
	if err != nil {
		// Record attempt without user_id — runs in its own goroutine (parallel).
		go func() {
			if err := h.history.Record(&models.LoginHistory{
				IPAddress:     ip,
				UserAgent:     ua,
				Browser:       browser,
				OS:            osName,
				Success:       false,
				FailureReason: "user not found",
			}); err != nil {
				log.Printf("[History] record error: %v", err)
			}
		}()
		writeError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		go func() {
			if err := h.history.Record(&models.LoginHistory{
				UserID:        &user.ID,
				IPAddress:     ip,
				UserAgent:     ua,
				Browser:       browser,
				OS:            osName,
				Success:       false,
				FailureReason: "incorrect password",
			}); err != nil {
				log.Printf("[History] record error: %v", err)
			}
		}()
		writeError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Record success in parallel — does not block the JWT/response path.
	go func() {
		if err := h.history.Record(&models.LoginHistory{
			UserID:    &user.ID,
			IPAddress: ip,
			UserAgent: ua,
			Browser:   browser,
			OS:        osName,
			Success:   true,
		}); err != nil {
			log.Printf("[History] record error: %v", err)
		}
	}()

	token, err := generateJWT(user.ID, user.LoginID)
	if err != nil {
		log.Printf("[Login] JWT error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateJWT(userID, loginID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userID,
		"login_id": loginID,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
}

// extractIP resolves the real client IP, honouring common proxy headers.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// r.RemoteAddr is "IP:port" — strip the port.
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}

// parseUserAgent extracts browser and OS strings from the User-Agent header.
func parseUserAgent(ua string) (browser, osName string) {
	lower := strings.ToLower(ua)

	switch {
	case strings.Contains(lower, "edg/") || strings.Contains(lower, "edge/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome") && !strings.Contains(lower, "chromium"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox"):
		browser = "Firefox"
	case strings.Contains(lower, "safari") && !strings.Contains(lower, "chrome"):
		browser = "Safari"
	case strings.Contains(lower, "opr") || strings.Contains(lower, "opera"):
		browser = "Opera"
	default:
		browser = "Unknown"
	}

	switch {
	case strings.Contains(lower, "windows"):
		osName = "Windows"
	case strings.Contains(lower, "android"):
		osName = "Android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		osName = "iOS"
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os x"):
		osName = "macOS"
	case strings.Contains(lower, "linux"):
		osName = "Linux"
	default:
		osName = "Unknown"
	}
	return
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[writeJSON] encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, status, map[string]string{"error": msg})
}
