package repository

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrLoginIDTaken = errors.New("login ID already taken")
	ErrEmailTaken   = errors.New("email already registered")
)

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

// Create inserts a new user and returns it. Returns ErrLoginIDTaken or
// ErrEmailTaken if a unique constraint is violated.
func (r *UserRepo) Create(loginID, email, hashedPassword string) (*models.User, error) {
	const q = `
		INSERT INTO users (login_id, email, password)
		VALUES ($1, $2, $3)
		RETURNING id, login_id, email, created_at, updated_at`

	var u models.User
	err := r.db.QueryRow(q, loginID, email, hashedPassword).Scan(
		&u.ID, &u.LoginID, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Constraint, "login_id") {
				return nil, ErrLoginIDTaken
			}
			if strings.Contains(pqErr.Constraint, "email") {
				return nil, ErrEmailTaken
			}
		}
		return nil, err
	}
	return &u, nil
}

// FindByLoginIDOrEmail looks up a user by either field (case-insensitive for email).
func (r *UserRepo) FindByLoginIDOrEmail(identifier string) (*models.User, error) {
	const q = `
		SELECT id, login_id, email, password, created_at, updated_at
		FROM users
		WHERE login_id = $1 OR lower(email) = lower($1)
		LIMIT 1`

	var u models.User
	err := r.db.QueryRow(q, identifier).Scan(
		&u.ID, &u.LoginID, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
