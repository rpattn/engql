package db

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations runs SQL migrations from the migrations directory
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsPath string) error {
	// Read all .up.sql files in the migrations directory
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name (assuming they start with numbers like 001_, 002_, etc.)
	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	// Execute each migration
	for _, fileName := range migrationFiles {
		filePath := filepath.Join(migrationsPath, fileName)
		sql, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		// Execute the migration
		_, err = pool.Exec(ctx, string(sql))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		fmt.Printf("Successfully executed migration: %s\n", fileName)
	}

	return nil
}
