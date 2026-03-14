package models

import "time"

// User represents a registered Sanchay IMS account.
type User struct {
	ID        string    `json:"id"`
	LoginID   string    `json:"login_id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // bcrypt hash — never serialised in API responses
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginHistory records every login attempt (successful or failed).
type LoginHistory struct {
	ID            string
	UserID        *string // nil when the identifier didn't match any user
	IPAddress     string
	UserAgent     string
	Browser       string
	OS            string
	Success       bool
	FailureReason string
	CreatedAt     time.Time
}
