package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// responseWriter captures HTTP status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// ResolverLoggerExtension logs resolver execution times
type ResolverLoggerExtension struct{}

// ExtensionName implements graphql.HandlerExtension
func (r *ResolverLoggerExtension) ExtensionName() string {
	return "ResolverLogger"
}

// Validate implements graphql.HandlerExtension
func (r *ResolverLoggerExtension) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

// InterceptField logs each resolver duration and errors
func (r *ResolverLoggerExtension) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	start := time.Now()
	res, err = next(ctx)
	duration := time.Since(start).Seconds() * 1000 //convert to ms
	fc := graphql.GetFieldContext(ctx)
	log.Printf("[GRAPHQL] %s.%s took %.3fms, error: %v", fc.Object, fc.Field.Name, duration, err)
	return res, err
}

// CombinedLoggingMiddleware logs HTTP request + resolver info
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process HTTP request
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s %d %s from %s", r.Method, r.URL.Path, rw.statusCode, duration, r.RemoteAddr)
	})
}
