package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rpattn/engql/graph"
	"github.com/rpattn/engql/internal/config"
	"github.com/rpattn/engql/internal/db"
	"github.com/rpattn/engql/internal/export"
	"github.com/rpattn/engql/internal/graphql"
	"github.com/rpattn/engql/internal/ingestion"
	"github.com/rpattn/engql/internal/middleware"
	"github.com/rpattn/engql/internal/repository"
	"github.com/rpattn/engql/internal/transformations"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
)

func main() {
	// 1. CONTEXT & SIGNAL HANDLING
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. DATABASE SETUP
	dbCfg, err := config.LoadDBConfig(".")
	if err != nil {
		log.Fatalf("Failed to load DB config: %v", err)
	}

	conn, err := db.NewConnection(ctx, dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// 3. MIGRATIONS
	log.Println("🔄 Running database migrations...")
	if err := db.RunMigrations(conn.Pool, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations complete")

	queries := db.New(conn.Pool)

	orgRepo := repository.NewOrganizationRepository(queries)
	entitySchemaRepo := repository.NewEntitySchemaRepository(queries)
	entityRepo := repository.NewEntityRepository(queries, conn.Pool)
	entityJoinRepo := repository.NewEntityJoinRepository(queries, conn.Pool)
	entityTransformationRepo := repository.NewEntityTransformationRepository(queries, conn.Pool)
	exportRepo := repository.NewEntityExportRepository(queries)
	ingestionLogRepo := repository.NewIngestionLogRepository(conn.Pool)

	transformationExecutor := transformations.NewExecutor(entityRepo, entitySchemaRepo)
	ingestionService := ingestion.NewService(entitySchemaRepo, entityRepo, ingestionLogRepo)
	exportService := export.NewService(orgRepo, entitySchemaRepo, entityRepo, exportRepo, entityTransformationRepo)

	resolver := graphql.NewResolver(
		orgRepo,
		entitySchemaRepo,
		entityRepo,
		entityJoinRepo,
		entityTransformationRepo,
		transformationExecutor,
		exportService,
	)

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.Use(&middleware.ResolverLoggerExtension{})

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://engql.rpattn.co.uk",
			"http://localhost:3000",
			"http://localhost:3010",
			"http://web:3000", // Internal Docker network origin
		},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"*"},
		Debug:            false, 
	})

	graphqlHandler := middleware.LoggingMiddleware(
		middleware.DataLoaderMiddleware(entityRepo)(srv),
	)
	ingestionHandler := middleware.LoggingMiddleware(
		ingestion.NewHTTPHandler(ingestionService),
	)
	exportHandler := middleware.LoggingMiddleware(
		export.NewHTTPHandler(exportService),
	)

	// Health check (for Docker/Coolify health monitoring)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.Handle("/query", corsHandler.Handler(graphqlHandler))
	http.Handle("/ingestion", corsHandler.Handler(ingestionHandler))
	http.Handle("/ingestion/", corsHandler.Handler(ingestionHandler))
	http.Handle("/exports", corsHandler.Handler(exportHandler))
	http.Handle("/exports/", corsHandler.Handler(exportHandler))

	renderPlayground := dbCfg.EnablePlayground

	http.Handle("/", corsHandler.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !renderPlayground {
			http.NotFound(w, r)
			return
		}
		playground.Handler("GraphQL playground", "/query").ServeHTTP(w, r)
	})))

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("API Backend starting on :8080")
		log.Println("Health check: http://localhost:8080/health")
		log.Println("Playground:   http://localhost:8080/")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}