package repository

import (
	"context"
	"fmt"

	"github.com/rpattn/engql/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *ingestionLogRepository) List(ctx context.Context, organizationID uuid.UUID, schemaName string, fileName string, limit int, offset int) ([]domain.IngestionLogEntry, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("ingestion log repository not initialized")
	}

	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(
		ctx,
		`SELECT id, organization_id, schema_name, file_name, row_number, error_message, created_at
		 FROM ingestion_logs
		 WHERE organization_id = $1
		   AND schema_name = $2
		   AND file_name = $3
		 ORDER BY created_at DESC
		 LIMIT $4 OFFSET $5`,
		organizationID,
		schemaName,
		fileName,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list ingestion logs: %w", err)
	}
	defer rows.Close()

	logs := []domain.IngestionLogEntry{}
	for rows.Next() {
		var (
			entry     domain.IngestionLogEntry
			rowNumber pgtype.Int4
			createdAt pgtype.Timestamptz
		)
		if scanErr := rows.Scan(
			&entry.ID,
			&entry.OrganizationID,
			&entry.SchemaName,
			&entry.FileName,
			&rowNumber,
			&entry.ErrorMessage,
			&createdAt,
		); scanErr != nil {
			return nil, fmt.Errorf("failed to scan ingestion log: %w", scanErr)
		}

		if rowNumber.Valid {
			value := int(rowNumber.Int32)
			entry.RowNumber = &value
		}
		if createdAt.Valid {
			entry.CreatedAt = createdAt.Time
		}

		logs = append(logs, entry)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("failed to iterate ingestion logs: %w", rowsErr)
	}

	return logs, nil
}
