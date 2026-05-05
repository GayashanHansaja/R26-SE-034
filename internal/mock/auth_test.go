package mock

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_Middleware(t *testing.T) {
	handler := Auth(func(w http.ResponseWriter, r *http.Request) {
		role := r.Header.Get("X-Resolved-Role")
		w.Write([]byte(role))
	})

	t.Run("Valid API Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", "finance-key-001")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "finance_viewer" {
			t.Errorf("expected role 'finance_viewer', got '%s'", rec.Body.String())
		}
	})

	t.Run("Invalid API Key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", "invalid")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("Valid Basic Auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "finance_viewer" {
			t.Errorf("expected role 'finance_viewer', got '%s'", rec.Body.String())
		}
	})
}
