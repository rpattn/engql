package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunMigrations runs pending database migrations once using golang-migrate.
func RunMigrations(pool *pgxpool.Pool, migrationsPath string) error {
	// Convert pgxpool.Pool to *sql.DB (golang-migrate needs *sql.DB)
	db := stdlibOpen(pool)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migration driver error: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migration setup error: %w", err)
	}

	// Run all "up" migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("✅ Migrations applied (or no change).")
	return nil
}

// stdlibOpen converts pgxpool.Pool -> *sql.DB for golang-migrate
func stdlibOpen(pool *pgxpool.Pool) *sql.DB {
	conn := pool.Config().ConnConfig.ConnString()
	db, err := sql.Open("pgx", conn)
	if err != nil {
		panic(fmt.Sprintf("failed to create stdlib DB: %v", err))
	}
	return db
}
