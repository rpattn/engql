package middleware

import (
	"context"
	"net/http"

	"github.com/rpattn/engql/internal/entityloader"
	"github.com/rpattn/engql/internal/repository"

	"github.com/graph-gophers/dataloader"
)

type ctxKey string

const entityLoaderKey ctxKey = "entityLoader"

// DataLoaderMiddleware attaches a dataloader to the request context
func DataLoaderMiddleware(repo repository.EntityRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create the entity loader
			loader := entityloader.NewEntityLoader(repo)

			// Store the underlying dataloader.Loader in context
			ctx := context.WithValue(r.Context(), entityLoaderKey, loader.Loader)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EntityLoaderFromContext retrieves the dataloader from context
func EntityLoaderFromContext(ctx context.Context) *dataloader.Loader {
	if l, ok := ctx.Value(entityLoaderKey).(*dataloader.Loader); ok {
		return l
	}
	return nil
}
