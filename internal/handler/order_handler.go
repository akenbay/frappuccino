package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"frappuccino/internal/models"
	"frappuccino/internal/service"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	orderID, err := h.orderService.CreateOrder(r.Context(), order)
	if err != nil {
		switch err {
		case models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      orderID,
		"message": "Order created successfully",
	})
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidOrderID.Error(), http.StatusBadRequest)
		return
	}

	order, err := h.orderService.GetOrder(r.Context(), id)
	if err != nil {
		if err == models.ErrInvalidOrderID {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get order: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	filters := models.OrderFilters{}

	// Parse query parameters
	if status := r.URL.Query().Get("status"); status != "" {
		filters.Status = status
	}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if parsed, err := time.Parse(time.RFC3339, startDate); err == nil {
			filters.StartDate = parsed
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if parsed, err := time.Parse(time.RFC3339, endDate); err == nil {
			filters.EndDate = parsed
		}
	}
	if customerID := r.URL.Query().Get("customer_id"); customerID != "" {
		if id, err := strconv.Atoi(customerID); err == nil {
			filters.CustomerID = id
		}
	}

	orders, err := h.orderService.ListOrders(r.Context(), filters)
	if err != nil {
		switch err {
		case models.ErrInvalidDateRange:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Failed to list orders: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidOrderID.Error(), http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err = h.orderService.UpdateOrder(r.Context(), id, order)
	if err != nil {
		switch err {
		case models.ErrInvalidOrderID:
			http.Error(w, "Order not found", http.StatusNotFound)
		case models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Failed to update order: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Order updated successfully",
	})
}

func (h *OrderHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidOrderID.Error(), http.StatusBadRequest)
		return
	}

	err = h.orderService.DeleteOrder(r.Context(), id)
	if err != nil {
		if err == models.ErrInvalidOrderID {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete order: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Order deleted successfully",
	})
}

func (h *OrderHandler) CloseOrder(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidOrderID.Error(), http.StatusBadRequest)
		return
	}

	err = h.orderService.CloseOrder(r.Context(), id)
	if err != nil {
		switch err {
		case models.ErrInvalidOrderID:
			http.Error(w, "Order not found", http.StatusNotFound)
		default:
			http.Error(w, fmt.Sprintf("Failed to close order: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Order closed successfully",
	})
}

func (h *OrderHandler) GetOrderedItemsReport(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	report, err := h.orderService.GetOrderedItemsReport(r.Context(), startDate, endDate)
	if err != nil {
		switch err {
		case models.ErrInvalidDateRange:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *OrderHandler) ProcessBatchOrders(w http.ResponseWriter, r *http.Request) {
	var batchRequest models.BatchOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&batchRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	response, err := h.orderService.ProcessBatchOrders(r.Context(), batchRequest.Orders)
	if err != nil {
		switch err {
		case models.ErrEmptyBatch, models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Failed to process batch orders: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
