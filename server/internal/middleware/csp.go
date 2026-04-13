package middleware

import (
	"net/http"
	"strings"
	"sync"
)

var (
	cspOnce   sync.Once
	cspHeader string
)

// BuildCSP constructs the Content-Security-Policy header value.
// allowedOrigins are included in connect-src so that cross-origin API calls
// and WebSocket connections are permitted by the policy.
func BuildCSP(allowedOrigins []string) string {
	// Deduplicate and collect origins for connect-src.
	seen := make(map[string]bool)
	var extra []string
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		extra = append(extra, origin)
	}

	connectSrc := "'self' ws: wss:"
	if len(extra) > 0 {
		connectSrc += " " + strings.Join(extra, " ")
	}

	return "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' https: data:; " +
		"connect-src " + connectSrc + "; " +
		"frame-ancestors 'none'; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"
}

// ContentSecurityPolicy returns middleware that sets the CSP header.
// Call InitCSP before mounting this middleware to include allowed origins.
func ContentSecurityPolicy(next http.Handler) http.Handler {
	// Ensure a default header exists even if InitCSP was never called.
	cspOnce.Do(func() {
		if cspHeader == "" {
			cspHeader = BuildCSP(nil)
		}
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", cspHeader)
		next.ServeHTTP(w, r)
	})
}

// InitCSP pre-computes the CSP header with the given allowed origins.
// Must be called before the first request is served.
func InitCSP(allowedOrigins []string) {
	cspHeader = BuildCSP(allowedOrigins)
}
