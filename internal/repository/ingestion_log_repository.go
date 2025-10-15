package repository

import (
	"context"
	"fmt"

	"github.com/rpattn/engql/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ingestionLogRepository struct {
	pool *pgxpool.Pool
}

// NewIngestionLogRepository wires a repository backed by pgxpool.
func NewIngestionLogRepository(pool *pgxpool.Pool) IngestionLogRepository {
	return &ingestionLogRepository{pool: pool}
}

func (r *ingestionLogRepository) Record(ctx context.Context, entry domain.IngestionLogEntry) error {
	if r.pool == nil {
		return fmt.Errorf("ingestion log repository not initialized")
	}

	var rowNumber any
	if entry.RowNumber != nil {
		rowNumber = *entry.RowNumber
	}

	_, err := r.pool.Exec(
		ctx,
		`INSERT INTO ingestion_logs (organization_id, schema_name, file_name, row_number, error_message)
		 VALUES ($1, $2, $3, $4, $5)`,
		entry.OrganizationID,
		entry.SchemaName,
		entry.FileName,
		rowNumber,
		entry.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to record ingestion log: %w", err)
	}

	return nil
}
