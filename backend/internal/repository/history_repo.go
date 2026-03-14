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

// ListByUserID returns recent login attempts for a specific user.
func (r *HistoryRepo) ListByUserID(userID string, limit int) ([]models.LoginHistoryItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	const q = `
		SELECT id, COALESCE(ip_address, ''), COALESCE(browser, 'Unknown'), COALESCE(os, 'Unknown'), success,
			COALESCE(failure_reason, ''), created_at
		FROM login_history
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.Query(q, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]models.LoginHistoryItem, 0)
	for rows.Next() {
		var item models.LoginHistoryItem
		if err := rows.Scan(
			&item.ID,
			&item.IPAddress,
			&item.Browser,
			&item.OS,
			&item.Success,
			&item.FailureReason,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}
