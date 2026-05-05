package mock

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Invoice struct {
	ID         string  `json:"id"`
	Number     string  `json:"number"`
	VendorID   string  `json:"vendor_id"`
	VendorName string  `json:"vendor_name"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Status     string  `json:"status"`   // draft | pending | paid | overdue
	DueDate    string  `json:"due_date"` // RFC3339
	CreatedAt  string  `json:"created_at"`
}

var invoices = []Invoice{
	{
		ID: "inv-001", Number: "INV-2026-001",
		VendorID: "v-001", VendorName: "Acme Supplies Ltd",
		Amount: 125000.00, Currency: "LKR",
		Status: "pending", DueDate: "2026-06-01T00:00:00Z",
		CreatedAt: "2026-05-01T09:00:00Z",
	},
	{
		ID: "inv-002", Number: "INV-2026-002",
		VendorID: "v-002", VendorName: "TechParts PVT",
		Amount: 48750.50, Currency: "LKR",
		Status: "paid", DueDate: "2026-05-15T00:00:00Z",
		CreatedAt: "2026-04-20T14:30:00Z",
	},
}

func InvoicesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]any{
			"data":  invoices,
			"total": len(invoices),
			"page":  1,
		})
	case http.MethodPost:
		var inv Invoice
		if err := json.NewDecoder(r.Body).Decode(&inv); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
			return
		}
		// Role check — only finance_editor can create
		role := r.Header.Get("X-Resolved-Role")
		if role != "finance_editor" && role != "admin" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
			return
		}
		inv.ID = "inv-new"
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(inv)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func InvoiceByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/finance/invoices/")
	for _, inv := range invoices {
		if inv.ID == id {
			json.NewEncoder(w).Encode(inv)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "not found", "id": id})
}

func PaymentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[], "total":0}`))
}

func LedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[], "total":0}`))
}
