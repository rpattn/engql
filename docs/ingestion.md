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


## Performance roadmap (batching):

# 🧭 Part 1: Batch Inserts with PostgreSQL + `sqlc`

> Goal: Replace per-row inserts with batched inserts (`CreateBatch`) to ingest 100 000 + rows in seconds instead of tens of seconds.

---

## ⚙️ 1. Background

Your ingestion service currently calls:

```go
s.entityRepo.Create(ctx, entity)
```

per row → 100 000 SQL round-trips.

Even if each insert takes only 0.2 ms, 100 000 inserts = 20 seconds.

We’ll implement:

```go
s.entityRepo.CreateBatch(ctx, []domain.Entity)
```

which inserts **many rows in one statement** using **PostgreSQL’s multi-value `INSERT`** or **`pgx.CopyFrom`** (if you want near–`COPY` speed).

This guide uses `sqlc` with standard `INSERT ... VALUES` batching — simple, safe, and already supported by your stack.

---

## 🧩 2. SQL changes

In your SQLC queries file (e.g., `repository/sql/entity.sql`):

```sql
-- name: CreateEntity :one
INSERT INTO entities (
    id,
    organization_id,
    schema_id,
    schema_name,
    path,
    properties
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: CreateEntitiesBatch :copyfrom
COPY entities (id, organization_id, schema_id, schema_name, path, properties)
FROM STDIN BINARY;
```

✅ Notes:
- If you prefer simpler (non–COPY) batching:
  ```sql
  -- name: CreateEntitiesBatch :many
  INSERT INTO entities (
      id, organization_id, schema_id, schema_name, path, properties
  )
  VALUES %s
  RETURNING id;
  ```
  But `sqlc` doesn’t natively support `%s` expansion; you’d generate this dynamically in Go.

- The **COPY** approach (with pgx’s CopyFrom) is the fastest and sqlc-compatible using `:copyfrom`.

---

## 🧱 3. Repository interface

In your `repository/entity_repository.go`:

```go
type EntityRepository interface {
    Create(ctx context.Context, e domain.Entity) (domain.Entity, error)
    CreateBatch(ctx context.Context, entities []domain.Entity) (int, error)
}
```

---

## 🧰 4. Repository implementation (using pgx and sqlc)

If you use sqlc’s generated queries, your repo might already look like:

```go
type entityRepo struct {
    db *pgxpool.Pool
    q  *Queries
}

func NewEntityRepository(db *pgxpool.Pool) repository.EntityRepository {
    return &entityRepo{db: db, q: New(db)}
}
```

Now add the batch insert method:

```go
func (r *entityRepo) CreateBatch(ctx context.Context, entities []domain.Entity) (int, error) {
    if len(entities) == 0 {
        return 0, nil
    }

    rows := make([][]any, 0, len(entities))
    for _, e := range entities {
        rows = append(rows, []any{
            e.ID,
            e.OrganizationID,
            e.SchemaID,
            e.SchemaName,
            e.Path,
            e.Properties,
        })
    }

    copyCount, err := r.db.CopyFrom(
        ctx,
        pgx.Identifier{"entities"},
        []string{"id", "organization_id", "schema_id", "schema_name", "path", "properties"},
        pgx.CopyFromRows(rows),
    )
    return int(copyCount), err
}
```

✅ Notes:
- This uses `pgx.CopyFrom`, which streams data into PostgreSQL’s internal COPY command — **very fast**.
- No need for explicit transactions unless you want rollback control.

---

## 🧠 5. Change ingestion loop to use batching

In your `Service.Ingest` method:

**Before:**
```go
if _, err := s.entityRepo.Create(ctx, entity); err != nil {
    s.summaryRowError(ctx, req, rowNumber, fmt.Errorf("failed to insert entity: %w", err))
    summary.InvalidRows++
    continue
}
summary.ValidRows++
```

**After:**
```go
const batchSize = 2000
batch := make([]domain.Entity, 0, batchSize)

for rowIdx, row := range table.rows {
    ...
    entity := domain.NewEntity(req.OrganizationID, workingSchema.ID, workingSchema.Name, path, properties)
    batch = append(batch, entity)

    if len(batch) >= batchSize {
        n, err := s.entityRepo.CreateBatch(ctx, batch)
        if err != nil {
            s.summaryRowError(ctx, req, rowIdx, fmt.Errorf("batch insert failed: %w", err))
        } else {
            summary.ValidRows += n
        }
        batch = batch[:0]
    }
}

// flush remaining
if len(batch) > 0 {
    n, err := s.entityRepo.CreateBatch(ctx, batch)
    if err == nil {
        summary.ValidRows += n
    }
}
```

---

## 📈 6. Optional optimizations

| Optimization | Description | Benefit |
|---------------|--------------|----------|
| Wrap batch inserts in a transaction | Use `tx, _ := r.db.Begin(ctx)` → `tx.CopyFrom(...)` → `tx.Commit()` | Atomic commit of batch |
| Tune `batchSize` | 1 000–5 000 is usually ideal | Avoid memory blowup or network stalls |
| Compress JSONB | Use `pgtype.JSONB` for `properties` | Slightly faster COPY parsing |
| Add `ON CONFLICT DO NOTHING` if duplicates are common | Prevents failure of whole batch | More resilient |

---

## ✅ 7. Benchmark checklist

After implementing:

- [ ] Measure ingestion before (e.g., 30 s for 100k rows)
- [ ] Measure after: it should drop to **< 5 s**
- [ ] Check DB CPU utilization (should rise → faster pipeline)
- [ ] Validate row counts match expectations

---

## 🧩 Example performance test snippet

```go
start := time.Now()
summary, err := svc.Ingest(ctx, req)
fmt.Printf("Ingested %d valid / %d invalid in %v\n",
    summary.ValidRows, summary.InvalidRows, time.Since(start))
```

---

## 💡 Expected gains

| Change | Typical Time (100k rows) |
|---------|--------------------------|
| Current (`Create` per row) | ~30 s |
| With `CopyFrom` batch inserts | **2–4 s** |
| With streaming CSV (Part 2) + batching | **< 2 s** |
