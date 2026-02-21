package middleware

import (
	"net/http"
)

// InternalOnly checks for a secret header before allowing the request through
func InternalOnly(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip check for health endpoint if needed, or keep it for total privacy
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Header.Get("X-Internal-Secret") != secret {
				http.Error(w, "Unauthorized: Internal access only", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}