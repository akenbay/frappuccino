package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"frappuccino/internal/models"
	"frappuccino/internal/service"
)

type InventoryHandler struct {
	inventoryService service.InventoryService
}

func NewInventoryHandler(service service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventoryService: service}
}

func (h *InventoryHandler) CreateIngredient(w http.ResponseWriter, r *http.Request) {
	var ingredient models.Inventory
	if err := json.NewDecoder(r.Body).Decode(&ingredient); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	id, err := h.inventoryService.CreateIngredient(r.Context(), ingredient)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create ingredient: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"message": "Ingredient created successfully",
	})
}

func (h *InventoryHandler) GetIngredient(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ingredient ID", http.StatusBadRequest)
		return
	}

	ingredient, err := h.inventoryService.GetIngredient(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get ingredient: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ingredient)
}

func (h *InventoryHandler) ListIngredients(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.inventoryService.ListIngredients(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list ingredients: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ingredients)
}

func (h *InventoryHandler) UpdateIngredient(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ingredient ID", http.StatusBadRequest)
		return
	}

	var ingredient models.Inventory
	if err := json.NewDecoder(r.Body).Decode(&ingredient); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	err = h.inventoryService.UpdateIngredient(r.Context(), id, ingredient)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update ingredient: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Ingredient updated successfully",
	})
}

func (h *InventoryHandler) DeleteIngredient(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid ingredient ID", http.StatusBadRequest)
		return
	}

	err = h.inventoryService.DeleteIngredient(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete ingredient: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Ingredient deleted successfully",
	})
}
