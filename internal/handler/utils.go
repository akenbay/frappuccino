package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"frappuccino/internal/models"
)

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	if startDateStr == "" || endDateStr == "" {
		return time.Time{}, time.Time{}, models.ErrInvalidDateRange
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, models.ErrInvalidDateRange
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, models.ErrInvalidDateRange
	}

	if startDate.After(endDate) {
		return time.Time{}, time.Time{}, models.ErrInvalidDateRange
	}

	// Adjust endDate to end of day
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return startDate, endDate, nil
}
