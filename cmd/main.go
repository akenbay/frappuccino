package main

import (
	"context"
	"database/sql"
	"fmt"
	"frappuccino/internal/dal"
	"frappuccino/internal/handler"
	"frappuccino/internal/middleware"
	"frappuccino/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Initialize database connection
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	orderRepo := dal.NewOrderRepository(db)
	reportRepo := dal.NewReportRepository(db)
	inventoryRepo := dal.NewInventoryRepository(db)

	// Initialize services
	orderService := service.NewOrderService(orderRepo)
	reportService := service.NewReportService(reportRepo)
	inventoryService := service.NewInventoryService(inventoryRepo)

	// Initialize handlers
	orderHandler := handler.NewOrderHandler(orderService)
	reportHandler := handler.NewReportHandler(reportService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)

	// Create router
	router := NewRouter(orderHandler, reportHandler, inventoryHandler)

	// Configure server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	port = "9090"

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func initDB() (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")

	// Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Verify connection
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to database")
	return db, nil
}

func NewRouter(
	orderHandler *handler.OrderHandler,
	reportHandler *handler.ReportHandler,
	inventoryHanlder *handler.InventoryHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Middleware chain
	handler := middleware.Logging(mux)
	handler = middleware.Recovery(handler)

	// Order routes
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", orderHandler.GetOrder)
	mux.HandleFunc("PUT /orders/{id}", orderHandler.UpdateOrder)
	mux.HandleFunc("DELETE /orders/{id}", orderHandler.DeleteOrder)
	mux.HandleFunc("POST /orders/{id}/close", orderHandler.CloseOrder)
	mux.HandleFunc("GET /orders", orderHandler.ListOrders)
	mux.HandleFunc("POST /orders/batch-process", orderHandler.ProcessBatchOrders)
	mux.HandleFunc("GET /orders/numberOfOrderedItems", orderHandler.GetOrderedItemsReport)

	// Inventory routes
	mux.HandleFunc("POST /inventory", inventoryHanlder.CreateIngredient)
	mux.HandleFunc("GET /inventory/{id}", inventoryHanlder.GetIngredient)
	mux.HandleFunc("PUT /inventory/{id}", inventoryHanlder.UpdateIngredient)
	mux.HandleFunc("DELETE /inventory/{id}", inventoryHanlder.DeleteIngredient)
	mux.HandleFunc("GET /inventory", inventoryHanlder.ListIngredients)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return handler
}
