# Ingestion Pipeline Overview

This document captures the current state of the metadata ingestion flow that was introduced for importing CSV and Excel files into entity schemas.

## What Was Shipped

- **Service layer (`internal/ingestion/service.go`)**
  - Detects schema from CSV/XLSX uploads (header sanitisation, type inference, required column detection).
  - Reconciles with existing entity schemas (append new fields, warn on incompatible types).
  - Validates and coerces row values before handing off to the entity repository.
  - Records row-level issues in the `ingestion_logs` table for later review.

- **REST endpoint (`/ingestion`)**
  - Exposed via `internal/ingestion/http_handler.go` and wired to the server in `cmd/server/main.go`.
  - Accepts multipart form data: `file`, `organizationId`, `schemaName`, and optional `description`.
  - Responds with a JSON summary (`totalRows`, `validRows`, `invalidRows`, `newFieldsDetected`, `schemaChanges`, `schemaCreated`).

- **Persistence**
  - New `ingestion_logs` table (migration `004_ingestion_logs`) stores `(organization_id, schema_name, file_name, row_number, error_message)`.
  - Repository wrapper (`internal/repository/ingestion_log_repository.go`) handles inserts.

- **Frontend tooling**
  - Dedicated ingestion page (`engql_frontend/src/routes/ingestion.tsx`) with:
    - Upload form targeting the REST endpoint.
    - Summary display of ingestion results.
    - Entity preview grid powered by the `entitiesByType` GraphQL query.
    - “Fetch Schema” button that loads field definitions via `entitySchemaByName`.

- **Tests**
  - Unit tests in `internal/ingestion/service_test.go` covering schema creation, schema extension, and type-conflict handling.

## Operational Checklist

1. Run database migration `migrations/004_ingestion_logs.up.sql`.
2. Deploy the updated Go service and ensure `/ingestion` is reachable.
3. Rebuild and ship the frontend bundle (ingestion page depends on the new endpoint).

## Follow-Up Ideas

1. **Surface ingestion logs** – add an API/graph view to browse `ingestion_logs` entries per upload.
2. **S3 or object storage integration** – support large files by streaming directly from storage rather than reading into memory.
3. **Scheduled re-ingestion** – allow reprocessing of staged files with updated schema rules.
4. **Enhanced validation** – plug in organisation-specific rules or custom validators for certain field types.
5. **Observability** – emit ingestion metrics (row throughput, failure counts) to tracing/metrics backends.
