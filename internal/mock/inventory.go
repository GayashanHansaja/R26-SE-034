package mock

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Price       float64 `json:"price"`
	StockLevel  int     `json:"stock_level"`
	ReorderPoint int     `json:"reorder_point"`
}

var items = []Item{
	{ID: "item-001", Name: "Laptop", SKU: "LAP-001", Price: 1500.00, StockLevel: 50, ReorderPoint: 10},
	{ID: "item-002", Name: "Mouse", SKU: "MOU-001", Price: 25.00, StockLevel: 200, ReorderPoint: 50},
}

func ItemsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"data":  items,
		"total": len(items),
	})
}

func ItemByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/items/")
	for _, item := range items {
		if item.ID == id {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "not found", "id": id})
}

func PurchaseOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[], "total":0}`))
}

func StockLevelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[], "total":0}`))
}
