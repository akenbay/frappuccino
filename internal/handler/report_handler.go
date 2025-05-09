package handler

import (
	"frappuccino/internal/models"
	"frappuccino/internal/service"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ReportHandler struct {
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// GetTotalSales handles GET /reports/total-sales
func (h *ReportHandler) GetTotalSales(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.reportService.GetTotalSales(r.Context(), startDate, endDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get total sales")
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

// GetPopularItems handles GET /reports/popular-items
func (h *ReportHandler) GetPopularItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			respondWithError(w, http.StatusBadRequest, "invalid limit parameter")
			return
		}
	}

	items, err := h.reportService.GetPopularItems(r.Context(), limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get popular items")
		return
	}

	respondWithJSON(w, http.StatusOK, items)
}

// GetOrderedItemsByPeriod handles GET /reports/period/{period}
func (h *ReportHandler) GetOrderedItemsByPeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		respondWithError(w, http.StatusBadRequest, "invalid path")
		return
	}
	period := pathParts[3]

	if period != "day" && period != "month" {
		respondWithError(w, http.StatusBadRequest, "invalid period type")
		return
	}

	year := time.Now().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "invalid year parameter")
			return
		}
	}

	var month time.Month = time.January
	if period == "day" {
		if monthStr := r.URL.Query().Get("month"); monthStr != "" {
			var err error
			month, err = parseMonth(monthStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, "invalid month parameter")
				return
			}
		} else {
			month = time.Now().Month()
		}
	}

	result, err := h.reportService.GetOrderedItemsByPeriod(r.Context(), period, month, year)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get period report")
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

// Search handles GET /reports/search
func (h *ReportHandler) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		respondWithError(w, http.StatusBadRequest, "query parameter is required")
		return
	}

	// Get and process filter parameter
	filterParam := r.URL.Query().Get("filter")
	if filterParam == "" {
		filterParam = "all"
	}

	// Split comma-separated filters and validate each
	filterParts := strings.Split(filterParam, ",")
	validFilters := make([]string, 0, len(filterParts))

	for _, part := range filterParts {
		trimmed := strings.TrimSpace(part)
		if isValidFilter(trimmed) {
			validFilters = append(validFilters, trimmed)
		}
	}

	// If no valid filters remain, use default "all"
	if len(validFilters) == 0 {
		validFilters = append(validFilters, "all")
	}

	// Join back to comma-separated string for service layer
	filter := strings.Join(validFilters, ",")

	result, err := h.reportService.Search(r.Context(), query, filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to perform search")
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func parseMonth(monthStr string) (time.Month, error) {
	month, err := strconv.Atoi(monthStr)
	if err == nil && month >= 1 && month <= 12 {
		return time.Month(month), nil
	}

	// Try parsing as month name
	for i := 1; i <= 12; i++ {
		if strings.EqualFold(time.Month(i).String(), monthStr) {
			return time.Month(i), nil
		}
	}

	return time.January, models.ErrInvalidMonth
}

func isValidFilter(filter string) bool {
	switch filter {
	case "all", "menu", "orders", "customers":
		return true
	default:
		return false
	}
}
