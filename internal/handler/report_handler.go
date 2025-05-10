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

type ReportHandler struct {
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) GetTotalSales(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	response, err := h.reportService.GetTotalSales(r.Context(), startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get total sales: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ReportHandler) GetPopularItems(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default value
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
			return
		}
	}

	items, err := h.reportService.GetPopularItems(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get popular items: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *ReportHandler) GetOrderedItemsByPeriod(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	period := r.URL.Query().Get("period")
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	// Validate period
	if period != "daily" && period != "weekly" && period != "monthly" {
		http.Error(w, "period must be one of: daily, weekly, monthly", http.StatusBadRequest)
		return
	}

	// Parse month and year
	var month time.Month
	var year int
	var err error

	if monthStr != "" {
		monthInt, err := strconv.Atoi(monthStr)
		if err != nil || monthInt < 1 || monthInt > 12 {
			http.Error(w, "month must be between 1 and 12", http.StatusBadRequest)
			return
		}
		month = time.Month(monthInt)
	} else {
		month = time.Now().Month()
	}

	if yearStr != "" {
		year, err = strconv.Atoi(yearStr)
		if err != nil || year < 2000 || year > time.Now().Year() {
			http.Error(w, "year must be between 2000 and current year", http.StatusBadRequest)
			return
		}
	} else {
		year = time.Now().Year()
	}

	response, err := h.reportService.GetOrderedItemsByPeriod(r.Context(), period, month, year)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get period report: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ReportHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	filter := r.URL.Query().Get("filter")

	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.SearchResult{})
		return
	}

	result, err := h.reportService.Search(r.Context(), query, filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
