package main

import (
	"net/http"

	"frappuccino/internal/handler"
	"frappuccino/internal/middleware"
)

func NewRouter(
	orderHandler *handler.OrderHandler,
	reportHandler *handler.ReportHandler,
) http.Handler {
	mux := http.NewServeMux()

	// Middleware chain
	handler := middleware.Logging(mux)
	handler = middleware.Recovery(handler)

	// Order routes
	mux.HandleFunc("POST /api/v1/orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /api/v1/orders/{id}", orderHandler.GetOrder)
	mux.HandleFunc("PUT /api/v1/orders/{id}", orderHandler.UpdateOrder)
	mux.HandleFunc("DELETE /api/v1/orders/{id}", orderHandler.DeleteOrder)
	mux.HandleFunc("POST /api/v1/orders/{id}/close", orderHandler.CloseOrder)
	mux.HandleFunc("GET /api/v1/orders", orderHandler.ListOrders)

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return handler
}
