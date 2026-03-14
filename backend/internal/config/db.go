package config

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// ConnectDB opens a connection pool to the Sanchay IMS PostgreSQL database.
// The pool is tuned for parallel request handling.
func ConnectDB() (*sql.DB, error) {
	// search_path=users ensures every unqualified table reference resolves to
	// the "users" schema rather than the default public schema.
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s search_path=%s",
		"localhost", 5432, "sanchay-ims", "postgres", "admin", "disable", "users",
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Connection pool — each goroutine/request gets its own connection.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	log.Println("[DB] Connected to PostgreSQL → sanchay-ims")
	return db, nil
}
