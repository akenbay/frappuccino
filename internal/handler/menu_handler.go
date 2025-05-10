package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"frappuccino/internal/models"
	"frappuccino/internal/service"
)

type MenuHandler struct {
	menuService service.MenuService
}

func NewMenuHandler(menuService service.MenuService) *MenuHandler {
	return &MenuHandler{menuService: menuService}
}

func (h *MenuHandler) ListMenuItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.menuService.GetAllMenu(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get menu items: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *MenuHandler) GetMenuItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidMenuItemID.Error(), http.StatusBadRequest)
		return
	}

	item, err := h.menuService.GetMenuItemByID(r.Context(), id)
	if err != nil {
		if err == models.ErrInvalidMenuItemID {
			http.Error(w, "Menu item not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to get menu item: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *MenuHandler) CreateMenuItem(w http.ResponseWriter, r *http.Request) {
	var item models.MenuItems
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	id, err := h.menuService.CreateMenuItem(r.Context(), item)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add menu item: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"message": "Menu item added successfully",
	})
}

func (h *MenuHandler) UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidMenuItemID.Error(), http.StatusBadRequest)
		return
	}

	var item models.MenuItems
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := h.menuService.UpdateMenuItem(r.Context(), id, item); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update menu item: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Menu item updated successfully",
	})
}

func (h *MenuHandler) DeleteMenuItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, models.ErrInvalidMenuItemID.Error(), http.StatusBadRequest)
		return
	}

	if err := h.menuService.DeleteMenuItem(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete menu item: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Menu item deleted successfully",
	})
}
