# Entity Join Support

This project now supports defining, persisting, and executing reusable join definitions between entity types in the GraphQL API. Join definitions capture how two entity types relate through a reference field, along with optional filter and sorting rules, so complex lookups can be rebuilt consistently without duplicating business logic.

## Data Model

- **`entity_joins` table** stores each definition, including the organization scope, related entity types, the reference field, JSONB filter stacks for each side, and structured sort criteria.
- SQLC-generated accessors expose CRUD operations for these definitions, and the repository layer provides a high-level `ExecuteJoin` helper that materializes the paired entities with optional overrides for filters, sorts, and pagination.

## GraphQL Surface

The schema introduces several new types (`EntityJoinDefinition`, `EntityJoinConnection`, `JoinSortCriterion`, and `PropertyFilterConfig`) plus the following operations:

- `createEntityJoinDefinition`, `updateEntityJoinDefinition`, and `deleteEntityJoinDefinition` mutations for lifecycle management.
- `entityJoinDefinition` and `entityJoinDefinitions` queries to inspect definitions.
- `executeEntityJoin` query to stream paired entity results (`left` and `right`) based on the stored definition, with optional runtime filters and pagination.

All operations accept/return strongly typed filter and sort structures, mirroring the backend repository contract.

## Manual Testing

1. **Apply migrations & run services**
   - Run the Go API so the new `entity_joins` table is migrated: `go run ./cmd/server` (from the project root).
   - Launch the React frontend: `npm install && npm run dev` inside `engql_frontend`.

2. **Navigate to the Join Testing page**
   - Visit `http://localhost:3000/join-testing`.
   - Use the *Create Join Definition* form to supply organization + schema details. Filters and sort criteria accept JSON arrays (helpers and defaults are pre-filled).
   - Use *List Join Definitions* to fetch definitions for an organization. The grid view reuses the entity table styles for easy scanning.
   - Execute a join via *Run Join* to preview paired entities. Results stream into the shared grid viewer with parsed property summaries.
   - Update or delete a definition from the action buttons embedded in the table to validate round-trips quickly.

3. **Backend validation (optional)**
   - The integration test `tests/entity_joins_test.go` exercises the full lifecycle (create/list/execute/update/delete) against the GraphQL endpoint for automated coverage.

With the join definitions persisted, downstream systems can query `executeEntityJoin` at runtime instead of manually stitching JSONB fields, ensuring a single source of truth for relationship logic.
