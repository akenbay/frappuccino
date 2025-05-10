package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	period := strings.ToLower(r.URL.Query().Get("period"))
	monthStr := strings.ToLower(r.URL.Query().Get("month"))
	yearStr := r.URL.Query().Get("year")

	// Validate period
	validPeriods := map[string]bool{"day": true, "month": true}
	if !validPeriods[period] {
		http.Error(w, "period must be one of: day, month", http.StatusBadRequest)
		return
	}

	// Parse month
	var month time.Month
	if monthStr != "" {
		// Try to parse as number first
		if monthInt, err := strconv.Atoi(monthStr); err == nil {
			if monthInt < 1 || monthInt > 12 {
				http.Error(w, "month must be between 1 and 12", http.StatusBadRequest)
				return
			}
			month = time.Month(monthInt)
		} else {
			// Parse as month name
			parsedMonth, err := parseMonthName(monthStr)
			if err != nil {
				http.Error(w, "month must be a valid month name or number (1-12)", http.StatusBadRequest)
				return
			}
			month = parsedMonth
		}
	} else {
		if period == "daily" || period == "weekly" {
			month = time.Now().Month()
		}
	}

	// Parse year
	var year int
	var err error
	if yearStr != "" {
		year, err = strconv.Atoi(yearStr)
		if err != nil || year < 2000 || year > time.Now().Year() {
			http.Error(w, "year must be between 2000 and current year", http.StatusBadRequest)
			return
		}
	} else {
		year = time.Now().Year()
	}

	// Additional validation for monthly reports
	if period == "monthly" && monthStr != "" {
		http.Error(w, "month parameter should not be provided for monthly period reports", http.StatusBadRequest)
		return
	}

	response, err := h.reportService.GetOrderedItemsByPeriod(r.Context(), period, month, year)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get period report: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to parse month names
func parseMonthName(monthStr string) (time.Month, error) {
	// Map of common month name formats
	monthAbbreviations := map[string]time.Month{
		"jan": time.January, "january": time.January,
		"feb": time.February, "february": time.February,
		"mar": time.March, "march": time.March,
		"apr": time.April, "april": time.April,
		"may": time.May,
		"jun": time.June, "june": time.June,
		"jul": time.July, "july": time.July,
		"aug": time.August, "august": time.August,
		"sep": time.September, "september": time.September,
		"oct": time.October, "october": time.October,
		"nov": time.November, "november": time.November,
		"dec": time.December, "december": time.December,
	}

	if month, ok := monthAbbreviations[monthStr]; ok {
		return month, nil
	}
	return 0, fmt.Errorf("invalid month name")
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
