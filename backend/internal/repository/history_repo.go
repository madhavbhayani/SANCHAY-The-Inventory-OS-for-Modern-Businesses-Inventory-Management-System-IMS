package repository

import (
	"database/sql"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

type HistoryRepo struct{ db *sql.DB }

func NewHistoryRepo(db *sql.DB) *HistoryRepo { return &HistoryRepo{db: db} }

// Record inserts a login attempt into login_history.
// Called in a goroutine (fire-and-forget) so errors are only logged, not returned.
func (r *HistoryRepo) Record(entry *models.LoginHistory) error {
	const q = `
		INSERT INTO login_history
			(user_id, ip_address, user_agent, browser, os, success, failure_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	// Convert pointer fields to sql-compatible nullable values.
	var uid interface{}
	if entry.UserID != nil && *entry.UserID != "" {
		uid = *entry.UserID
	}
	var failReason interface{}
	if entry.FailureReason != "" {
		failReason = entry.FailureReason
	}

	_, err := r.db.Exec(q,
		uid,
		entry.IPAddress,
		entry.UserAgent,
		entry.Browser,
		entry.OS,
		entry.Success,
		failReason,
	)
	return err
}
