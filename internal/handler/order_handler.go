package handler

import (
	"encoding/json"
	"frappuccino/internal/models"
	"frappuccino/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/orders")
	path = strings.TrimSuffix(path, "/")

	switch {
	case r.Method == http.MethodPost && path == "":
		h.createOrder(w, r)
	case r.Method == http.MethodGet && path == "":
		h.listOrders(w, r)
	case r.Method == http.MethodGet && isSingleOrderPath(path):
		h.getOrder(w, r)
	case r.Method == http.MethodPut && isSingleOrderPath(path):
		h.updateOrder(w, r)
	case r.Method == http.MethodDelete && isSingleOrderPath(path):
		h.deleteOrder(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/close"):
		h.closeOrder(w, r)
	case r.Method == http.MethodGet && path == "/numberOfOrderedItems":
		h.getOrderedItemsCount(w, r)
	case r.Method == http.MethodPost && path == "/batch-process":
		h.processBatchOrders(w, r)
	default:
		http.NotFound(w, r)
	}
}

func isSingleOrderPath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return len(parts) == 1 && parts[0] != ""
}

func extractOrderID(path string) (int, error) {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 {
		return 0, models.ErrInvalidOrderID
	}
	return strconv.Atoi(parts[0])
}

func (h *OrderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	id, err := h.orderService.CreateOrder(r.Context(), order)
	if err != nil {
		switch err {
		case models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]int{"id": id})
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := extractOrderID(strings.TrimPrefix(r.URL.Path, "/orders"))
	if err != nil || id <= 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	order, err := h.orderService.GetOrder(r.Context(), id)
	if err != nil {
		if err == models.ErrInvalidOrderID {
			respondWithError(w, http.StatusNotFound, "Order not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	filters := models.OrderFilters{
		Status: r.URL.Query().Get("status"),
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
		if err == models.ErrInvalidDateRange {
			respondWithError(w, http.StatusBadRequest, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) updateOrder(w http.ResponseWriter, r *http.Request) {
	id, err := extractOrderID(strings.TrimPrefix(r.URL.Path, "/orders"))
	if err != nil || id <= 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.orderService.UpdateOrder(r.Context(), id, order); err != nil {
		switch err {
		case models.ErrInvalidOrderID:
			respondWithError(w, http.StatusNotFound, "Order not found")
		case models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Order updated successfully"})
}

func (h *OrderHandler) deleteOrder(w http.ResponseWriter, r *http.Request) {
	id, err := extractOrderID(strings.TrimPrefix(r.URL.Path, "/orders"))
	if err != nil || id <= 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	if err := h.orderService.DeleteOrder(r.Context(), id); err != nil {
		if err == models.ErrInvalidOrderID {
			respondWithError(w, http.StatusNotFound, "Order not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Order deleted successfully"})
}

func (h *OrderHandler) closeOrder(w http.ResponseWriter, r *http.Request) {
	id, err := extractOrderID(strings.TrimPrefix(r.URL.Path, "/orders"))
	if err != nil || id <= 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	if err := h.orderService.CloseOrder(r.Context(), id); err != nil {
		switch err {
		case models.ErrInvalidOrderID:
			respondWithError(w, http.StatusNotFound, "Order not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Order closed successfully"})
}

func (h *OrderHandler) getOrderedItemsCount(w http.ResponseWriter, r *http.Request) {
	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	counts, err := h.orderService.GetOrderedItemsReport(r.Context(), startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, counts)
}

func (h *OrderHandler) processBatchOrders(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Orders []models.Order `json:"orders"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(request.Orders) == 0 {
		respondWithError(w, http.StatusBadRequest, "Empty batch")
		return
	}

	response, err := h.orderService.ProcessBatchOrders(r.Context(), request.Orders)
	if err != nil {
		switch err {
		case models.ErrEmptyBatch, models.ErrEmptyOrder, models.ErrInvalidTotalPrice:
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}
