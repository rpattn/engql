package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"graphql-engineering-api/graph"
	"graphql-engineering-api/internal/db"
	"graphql-engineering-api/internal/graphql"
	"graphql-engineering-api/internal/middleware"
	"graphql-engineering-api/internal/repository"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
)

func main() {
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup database connection
	config := db.DefaultConfig()
	conn, err := db.NewConnection(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Run migrations
	if err := db.RunMigrations(ctx, conn.Pool, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create sqlc queries instance
	queries := db.New(conn.Pool)

	// Create repositories
	orgRepo := repository.NewOrganizationRepository(queries)
	entitySchemaRepo := repository.NewEntitySchemaRepository(queries)
	entityRepo := repository.NewEntityRepository(queries)
	entityJoinRepo := repository.NewEntityJoinRepository(queries, conn.Pool)

	// Create GraphQL resolver
	resolver := graphql.NewResolver(orgRepo, entitySchemaRepo, entityRepo, entityJoinRepo)

	// Create GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Add the resolver logging extension
	srv.Use(&middleware.ResolverLoggerExtension{})

	// Setup CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
	})

	graphqlHandler := middleware.LoggingMiddleware(
		middleware.DataLoaderMiddleware(entityRepo)(srv),
	)

	http.Handle("/query", corsHandler.Handler(graphqlHandler))
	http.Handle("/", corsHandler.Handler(middleware.LoggingMiddleware(playground.Handler("GraphQL playground", "/query"))))

	// Create HTTP server
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Starting GraphQL server on :8080")
		log.Println("GraphQL playground available at http://localhost:8080")
		log.Println("GraphQL endpoint available at http://localhost:8080/query")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
