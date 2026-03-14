package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/config"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/handlers"
	"madhavbhayani/SANCHAY-The-Inventory-OS-for-Modern-Businesses-Inventory-Management-System-IMS/internal/middleware"
)

func main() {
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	defer db.Close()

	auth := handlers.NewAuthHandler(db)

	// Go 1.22+ method-pattern routing: only POST requests match these paths.
	// net/http spawns a goroutine per request — all handlers run in parallel
	// automatically. The DB pool (25 max open conns) backs concurrent queries.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/signup", auth.Signup)
	mux.HandleFunc("POST /api/auth/login", auth.Login)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      middleware.CORS(mux), // CORS wraps the entire mux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start listening in its own goroutine.
	go func() {
		log.Println("[SERVER] Sanchay IMS backend  →  http://localhost:8080")
		log.Println("[SERVER] Endpoints:")
		log.Println("[SERVER]   POST /api/auth/signup")
		log.Println("[SERVER]   POST /api/auth/login")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] ListenAndServe: %v", err)
		}
	}()

	// Block until SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[SERVER] Shutting down gracefully…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[SERVER] Shutdown error: %v", err)
	}
	log.Println("[SERVER] Stopped cleanly")
}
