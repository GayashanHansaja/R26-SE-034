package mock

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvoicesHandler_GET(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/finance/invoices", nil)
	req.Header.Set("X-API-Key", "finance-key-001")
	rec := httptest.NewRecorder()

	Auth(InvoicesHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !testing.Short() {
		// More detailed checks if needed
	}
}

func TestInvoicesHandler_POST_Forbidden(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/finance/invoices", strings.NewReader(`{"vendor_id":"v-001"}`))
	req.Header.Set("X-API-Key", "finance-key-001") // viewer only
	rec := httptest.NewRecorder()

	Auth(InvoicesHandler)(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}
}
