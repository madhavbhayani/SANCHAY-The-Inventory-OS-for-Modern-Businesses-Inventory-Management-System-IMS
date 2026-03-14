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
	settings := handlers.NewSettingsHandler(db)
	stocks := handlers.NewStockHandler(db)
	operations := handlers.NewOperationsHandler(db)

	// Go 1.22+ method-pattern routing: method + path pairs are matched exactly.
	// net/http spawns a goroutine per request — all handlers run in parallel
	// automatically. The DB pool (25 max open conns) backs concurrent queries.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/signup", auth.Signup)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.HandleFunc("GET /api/settings", settings.GetOverview)
	mux.HandleFunc("POST /api/settings/warehouses", settings.CreateWarehouse)
	mux.HandleFunc("POST /api/settings/locations", settings.CreateLocation)
	mux.HandleFunc("POST /api/settings/change-password", settings.ChangePassword)
	mux.HandleFunc("GET /api/stocks/meta", stocks.GetMeta)
	mux.HandleFunc("POST /api/stocks/categories", stocks.CreateCategory)
	mux.HandleFunc("GET /api/stocks/products", stocks.ListProducts)
	mux.HandleFunc("POST /api/stocks/products", stocks.CreateProduct)
	mux.HandleFunc("PUT /api/stocks/products/{id}", stocks.UpdateProduct)
	mux.HandleFunc("DELETE /api/stocks/products/{id}", stocks.DeleteProduct)
	mux.HandleFunc("GET /api/operations/meta", operations.GetMeta)
	mux.HandleFunc("GET /api/operations/receipts", operations.ListReceipts)
	mux.HandleFunc("POST /api/operations/receipts", operations.CreateReceipt)
	mux.HandleFunc("GET /api/operations/delivery", operations.ListDelivery)
	mux.HandleFunc("POST /api/operations/delivery", operations.CreateDelivery)
	mux.HandleFunc("DELETE /api/operations/orders/{id}", operations.DeleteOrder)

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
		log.Println("[SERVER]   GET  /api/settings")
		log.Println("[SERVER]   POST /api/settings/warehouses")
		log.Println("[SERVER]   POST /api/settings/locations")
		log.Println("[SERVER]   POST /api/settings/change-password")
		log.Println("[SERVER]   GET  /api/stocks/meta")
		log.Println("[SERVER]   POST /api/stocks/categories")
		log.Println("[SERVER]   GET  /api/stocks/products")
		log.Println("[SERVER]   POST /api/stocks/products")
		log.Println("[SERVER]   PUT  /api/stocks/products/{id}")
		log.Println("[SERVER]   DELETE /api/stocks/products/{id}")
		log.Println("[SERVER]   GET  /api/operations/meta")
		log.Println("[SERVER]   GET  /api/operations/receipts")
		log.Println("[SERVER]   POST /api/operations/receipts")
		log.Println("[SERVER]   GET  /api/operations/delivery")
		log.Println("[SERVER]   POST /api/operations/delivery")
		log.Println("[SERVER]   DELETE /api/operations/orders/{id}")
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
