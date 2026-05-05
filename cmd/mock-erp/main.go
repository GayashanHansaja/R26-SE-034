package main

import (
	"log"
	"net/http"

	"github.com/nimendra/ERPBridge/internal/mock"
)

func main() {
	mux := http.NewServeMux()

	// Finance
	mux.HandleFunc("/api/v1/finance/invoices",    mock.Auth(mock.InvoicesHandler))
	mux.HandleFunc("/api/v1/finance/invoices/",   mock.Auth(mock.InvoiceByIDHandler))
	mux.HandleFunc("/api/v1/finance/payments",    mock.Auth(mock.PaymentsHandler))
	mux.HandleFunc("/api/v1/finance/ledger",      mock.Auth(mock.LedgerHandler))

	// HR
	mux.HandleFunc("/api/v1/hr/employees",        mock.Auth(mock.EmployeesHandler))
	mux.HandleFunc("/api/v1/hr/employees/",       mock.Auth(mock.EmployeeByIDHandler))
	mux.HandleFunc("/api/v1/hr/leave-requests",   mock.Auth(mock.LeaveRequestsHandler))
	mux.HandleFunc("/api/v1/hr/departments",      mock.Auth(mock.DepartmentsHandler))

	// Inventory
	mux.HandleFunc("/api/v1/inventory/items",          mock.Auth(mock.ItemsHandler))
	mux.HandleFunc("/api/v1/inventory/items/",         mock.Auth(mock.ItemByIDHandler))
	mux.HandleFunc("/api/v1/inventory/purchase-orders",mock.Auth(mock.PurchaseOrdersHandler))
	mux.HandleFunc("/api/v1/inventory/stock-levels",   mock.Auth(mock.StockLevelsHandler))

	log.Println("Mock ERP listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
}
