package mock

import (
	"net/http"
	"strings"
)

// MockCredentials maps credential → resolved role.
var MockCredentials = map[string]string{
	// API Keys (X-API-Key header)
	"finance-key-001": "finance_viewer",
	"finance-key-002": "finance_editor",
	"hr-key-001":      "hr_viewer",
	"hr-key-002":      "hr_manager",
	"inv-key-001":     "inv_viewer",
	"inv-key-002":     "inv_editor",
	"admin-key-001":   "admin",

	// Basic Auth tokens (pre-encoded "user:pass" in base64)
	"dXNlcjpwYXNz":      "finance_viewer", // user:pass
	"YWRtaW46YWRtaW4=":   "admin",          // admin:admin
}

// Auth wraps a handler with API Key + Basic Auth verification.
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, ok := resolveRole(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","code":401}`))
			return
		}
		// Pass the resolved role downstream — used by RBAC checks
		r.Header.Set("X-Resolved-Role", role)
		next(w, r)
	}
}

func resolveRole(r *http.Request) (string, bool) {
	// 1. API Key
	if key := r.Header.Get("X-API-Key"); key != "" {
		role, ok := MockCredentials[key]
		return role, ok
	}

	// 2. Basic Auth
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Basic ") {
		encoded := strings.TrimPrefix(auth, "Basic ")
		role, ok := MockCredentials[encoded]
		return role, ok
	}

	// 3. Bearer token (stub for Phase 3)
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		// Phase 3: validate JWT with Keycloak
		// For now, accept a known stub token
		if strings.TrimPrefix(auth, "Bearer ") == "dev-stub-token" {
			return "admin", true
		}
	}

	return "", false
}
