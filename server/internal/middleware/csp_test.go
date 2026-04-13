package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentSecurityPolicy(t *testing.T) {
	// Reset global state for test isolation.
	InitCSP(nil)

	handler := ContentSecurityPolicy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}

	required := []string{
		"script-src 'self'",
		"connect-src 'self' ws: wss:",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	for _, directive := range required {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP missing directive %q; got: %s", directive, csp)
		}
	}
}

func TestBuildCSP_WithOrigins(t *testing.T) {
	csp := BuildCSP([]string{"https://app.example.com", "https://dev.example.com"})

	if !strings.Contains(csp, "connect-src 'self' ws: wss: https://app.example.com https://dev.example.com") {
		t.Errorf("CSP connect-src should include allowed origins; got: %s", csp)
	}
}

func TestBuildCSP_DeduplicatesOrigins(t *testing.T) {
	csp := BuildCSP([]string{"https://app.example.com", "https://app.example.com", ""})

	count := strings.Count(csp, "https://app.example.com")
	if count != 1 {
		t.Errorf("expected origin to appear once, appeared %d times; got: %s", count, csp)
	}
}
