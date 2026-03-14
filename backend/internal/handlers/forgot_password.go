package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

const (
	forgotPasswordOTPExpiry      = 10 * time.Minute
	forgotPasswordVerifiedExpiry = 15 * time.Minute
	forgotPasswordMaxAttempts    = 6
	defaultSMTPHost              = "smtp.gmail.com"
	defaultSMTPPort              = "587"
	defaultSMTPUsername          = "madhavbhayani21@gmail.com"
)

type forgotPasswordRequestOTPBody struct {
	Email string `json:"email"`
}

type forgotPasswordVerifyOTPBody struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type forgotPasswordResetBody struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}

type forgotPasswordOTPRecord struct {
	UserID        string
	OTPHash       string
	ExpiresAt     time.Time
	VerifiedUntil time.Time
	Attempts      int
}

var forgotPasswordState = struct {
	mu      sync.Mutex
	records map[string]forgotPasswordOTPRecord
}{
	records: map[string]forgotPasswordOTPRecord{},
}

// POST /api/auth/forgot-password/request
func (h *AuthHandler) ForgotPasswordRequestOTP(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequestOTPBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		writeError(w, "email is required", http.StatusBadRequest)
		return
	}
	if !isValidEmailFormat(email) {
		writeError(w, "invalid email format", http.StatusBadRequest)
		return
	}

	user, err := h.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, "email is not registered", http.StatusNotFound)
			return
		}
		log.Printf("[ForgotPassword:RequestOTP] find user error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	otp, err := generateSixDigitOTP()
	if err != nil {
		log.Printf("[ForgotPassword:RequestOTP] otp generation error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcryptCost)
	if err != nil {
		log.Printf("[ForgotPassword:RequestOTP] otp hash error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	forgotPasswordState.mu.Lock()
	forgotPasswordState.records[email] = forgotPasswordOTPRecord{
		UserID:        user.ID,
		OTPHash:       string(hash),
		ExpiresAt:     time.Now().Add(forgotPasswordOTPExpiry),
		VerifiedUntil: time.Time{},
		Attempts:      0,
	}
	forgotPasswordState.mu.Unlock()

	if err := sendForgotPasswordOTPEmail(email, otp); err != nil {
		log.Printf("[ForgotPassword:RequestOTP] send email error: %v", err)
		writeError(w, "failed to send otp email", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "otp sent to your email",
	})
}

// POST /api/auth/forgot-password/verify
func (h *AuthHandler) ForgotPasswordVerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordVerifyOTPBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(req.Email)
	otp := strings.TrimSpace(req.OTP)
	if email == "" || otp == "" {
		writeError(w, "email and otp are required", http.StatusBadRequest)
		return
	}
	if !isSixDigitOTP(otp) {
		writeError(w, "otp must be exactly 6 digits", http.StatusBadRequest)
		return
	}

	forgotPasswordState.mu.Lock()
	record, ok := forgotPasswordState.records[email]
	if !ok {
		forgotPasswordState.mu.Unlock()
		writeError(w, "request otp first", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if now.After(record.ExpiresAt) {
		delete(forgotPasswordState.records, email)
		forgotPasswordState.mu.Unlock()
		writeError(w, "otp has expired", http.StatusBadRequest)
		return
	}

	if record.Attempts >= forgotPasswordMaxAttempts {
		delete(forgotPasswordState.records, email)
		forgotPasswordState.mu.Unlock()
		writeError(w, "too many invalid otp attempts", http.StatusTooManyRequests)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.OTPHash), []byte(otp)); err != nil {
		record.Attempts++
		forgotPasswordState.records[email] = record
		forgotPasswordState.mu.Unlock()
		writeError(w, "invalid otp", http.StatusUnauthorized)
		return
	}

	record.VerifiedUntil = now.Add(forgotPasswordVerifiedExpiry)
	record.Attempts = 0
	forgotPasswordState.records[email] = record
	forgotPasswordState.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "otp verified successfully",
	})
}

// POST /api/auth/forgot-password/reset
func (h *AuthHandler) ForgotPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordResetBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(req.Email)
	if email == "" || req.NewPassword == "" {
		writeError(w, "email and new password are required", http.StatusBadRequest)
		return
	}
	if err := validateResetPasswordStrength(req.NewPassword); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	forgotPasswordState.mu.Lock()
	record, ok := forgotPasswordState.records[email]
	if !ok {
		forgotPasswordState.mu.Unlock()
		writeError(w, "request and verify otp first", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if now.After(record.ExpiresAt) {
		delete(forgotPasswordState.records, email)
		forgotPasswordState.mu.Unlock()
		writeError(w, "otp has expired", http.StatusBadRequest)
		return
	}
	if record.VerifiedUntil.IsZero() || now.After(record.VerifiedUntil) {
		forgotPasswordState.mu.Unlock()
		writeError(w, "otp verification is required before reset", http.StatusUnauthorized)
		return
	}
	forgotPasswordState.mu.Unlock()

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcryptCost)
	if err != nil {
		log.Printf("[ForgotPassword:Reset] bcrypt error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, "email is not registered", http.StatusNotFound)
			return
		}
		log.Printf("[ForgotPassword:Reset] find user error: %v", err)
		writeError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.users.UpdatePassword(user.ID, string(hash)); err != nil {
		log.Printf("[ForgotPassword:Reset] update password error: %v", err)
		writeError(w, "failed to update password", http.StatusInternalServerError)
		return
	}

	forgotPasswordState.mu.Lock()
	delete(forgotPasswordState.records, email)
	forgotPasswordState.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "password reset successful",
	})
}

func generateSixDigitOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidEmailFormat(email string) bool {
	return emailRegex.MatchString(email)
}

func isSixDigitOTP(otp string) bool {
	return otpRegex.MatchString(otp)
}

func validateResetPasswordStrength(password string) error {
	trimmed := strings.TrimSpace(password)
	if len(trimmed) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !lowerRegex.MatchString(trimmed) {
		return errors.New("password must include at least one lowercase letter")
	}
	if !upperRegex.MatchString(trimmed) {
		return errors.New("password must include at least one uppercase letter")
	}
	if !specialRegex.MatchString(trimmed) {
		return errors.New("password must include at least one special character")
	}
	return nil
}

func sendForgotPasswordOTPEmail(email, otp string) error {
	host := readEnvOrDefault("SMTP_HOST", defaultSMTPHost)
	port := readEnvOrDefault("SMTP_PORT", defaultSMTPPort)
	username := readEnvOrDefault("SMTP_USERNAME", defaultSMTPUsername)
	password := normalizeSMTPPassword(os.Getenv("SMTP_PASSWORD"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = username
	}

	if password == "" {
		log.Printf("[ForgotPassword] SMTP_PASSWORD missing. OTP for %s is %s", email, otp)
		return errors.New("smtp password is missing")
	}

	message := strings.Join([]string{
		"Subject: Sanchay IMS Password Reset OTP",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("Your Sanchay IMS OTP is %s.", otp),
		"This OTP is valid for 10 minutes.",
		"If you did not request this, you can safely ignore this email.",
	}, "\r\n")

	auth := smtp.PlainAuth("", username, password, host)
	return smtp.SendMail(host+":"+port, auth, from, []string{email}, []byte(message))
}

func readEnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func normalizeSMTPPassword(raw string) string {
	// Gmail app passwords are often copied in groups like "xxxx xxxx xxxx xxxx".
	trimmed := strings.TrimSpace(raw)
	return strings.ReplaceAll(trimmed, " ", "")
}

var (
	emailRegex   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	otpRegex     = regexp.MustCompile(`^[0-9]{6}$`)
	lowerRegex   = regexp.MustCompile(`[a-z]`)
	upperRegex   = regexp.MustCompile(`[A-Z]`)
	specialRegex = regexp.MustCompile(`[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?` + "`" + `~]`)
)
