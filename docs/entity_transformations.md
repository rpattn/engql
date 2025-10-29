# Entity Transformation Frontend Implementation Guide

This guide explains how to build the web experience for defining, managing, and executing entity transformation DAGs against the GraphQL API. It assumes the existing React + TanStack frontend located in `engql_frontend/` and the GraphQL schema generated in `graph/schema.graphqls`.

## Product Goals

1. **Designer** – allow analysts to visually compose transformation pipelines by chaining LOAD, FILTER, PROJECT, JOIN, SORT, PAGINATE, UNION, LEFT_JOIN, and ANTI_JOIN nodes into a DAG.
2. **Catalog** – provide list/detail views for transformation definitions scoped to an organization.
3. **Runner** – execute a saved transformation and inspect the resulting entity edges with pagination controls.

## API Contracts Recap

The GraphQL schema exposes the primitives required by the UI:

- `EntityTransformation` object
  - `id`, `organizationId`, `name`, `description`
  - `nodes: [EntityTransformationNode!]!` describing the DAG
- `EntityTransformationNode`
  - `id`, `type: EntityTransformationNodeType!`
  - Optional config objects (`load`, `filter`, `project`, `join`, `sort`, `paginate`)
  - `sources: [String!]!` linking to upstream node IDs
- `EntityTransformationConnection`
  - Returned by `executeEntityTransformation`; contains `edges` (each edge is an ordered list of entity snapshots representing the traversal path).
- Inputs
  - `CreateEntityTransformationInput`, `UpdateEntityTransformationInput`, `EntityTransformationNodeInput` mirror the model shape.
  - `ExecuteEntityTransformationInput` requires `id` plus optional runtime overrides.

Mutations & queries:

| Operation | Purpose |
|-----------|---------|
| `createEntityTransformation` | Persist a new definition. |
| `updateEntityTransformation` | Modify name/description/nodes. |
| `deleteEntityTransformation` | Remove a definition. |
| `entityTransformation` | Fetch one definition by ID. |
| `entityTransformations` | List definitions by `organizationId`. |
| `executeEntityTransformation` | Run a transformation and stream the result connection. |

> Run `npm run codegen` inside `engql_frontend/` after updating GraphQL operations to refresh types and React Query hooks.

## UI Architecture

### Route Structure

Create a dedicated route tree under `src/routes/transformations/` with three entry points:

1. `index.tsx` – organization-scoped grid/table listing transformations with create button.
2. `$transformationId.tsx` – detail view containing editor canvas and metadata form.
3. `$transformationId.execute.tsx` – execution console with form for runtime parameters and results table.

Use TanStack Router nested routes so the editor and runner share a common layout (breadcrumbs, org selector, secondary navigation tabs).

### State & Data Fetching

- **Queries**: Use generated hooks (e.g., `useEntityTransformationsQuery`, `useEntityTransformationQuery`) with TanStack Query to keep cache entries normalized by transformation ID.
- **Mutations**: Wrap `create`, `update`, `delete` in mutation hooks that invalidate relevant query keys.
- **Optimistic UI**: When reordering nodes or editing labels, update local component state immediately, and persist on save.
- **Form State**: Use React Hook Form or TanStack Form for validation and to mirror GraphQL input shapes.

### Graph Editor

Adopt a node-based canvas library (e.g., [React Flow](https://reactflow.dev/)) to render DAGs. Map GraphQL nodes to canvas elements:

- Node `data` contains the config payload and metadata (name, entity type, filters, etc.).
- Edges are drawn from `sources` -> node ID.
- Enforce acyclicity by blocking connections that would introduce a cycle (React Flow has helper utilities; fall back to custom DFS guard).

**Node Palette**

Provide a side panel listing supported node types with drag-and-drop onto the canvas. Templates:

- **LOAD** – requires `entityType` and optional `filters`.
- **FILTER** – accepts `PropertyFilterInput` array (reuse existing filter builders).
- **PROJECT** – multi-select of properties; maintain an ordered list.
- **JOIN** – configure `leftSource`, `rightSource`, `type`, `onField`, aliasing.
- **LEFT_JOIN / ANTI_JOIN** – same UI as JOIN but locked to the respective type.
- **UNION** – allow selecting N upstream sources; UI should ensure identical schemas.
- **SORT** – reorderable list of `{ side, field, direction }`.
- **PAGINATE** – numeric inputs for `limit`, `offset`.

**Inspector Panel**

When a node is selected, show form controls for its configuration. Persist edits back into local DAG state. Provide validation hints (e.g., required fields, incompatible types).

**Persistence Flow**

1. Maintain DAG state as `{ nodes: Record<string, CanvasNode>, edges: Edge[] }`.
2. On save, serialize into `EntityTransformationNodeInput[]`:
   - Order nodes topologically (React Flow `getNodes` + custom sort) to match backend expectations.
   - Include `sources` array for each node based on incoming edges.
   - Strip presentation-only fields (positions, UI labels).
3. Submit via `updateEntityTransformation` mutation.

**Undo/Redo**

Leverage a stack-based history to allow designers to revert mistakes quickly. Libraries like `use-undo` integrate well with React.

### List & Detail Views

- **List**: Use an existing data grid component (the join pages use TanStack Table) to show `name`, `nodeCount`, `lastUpdated`, `tags`. Provide inline actions for edit, execute, duplicate, delete.
- **Detail**: Split layout into left canvas (graph) and right inspector. Metadata header includes name, description, org, createdBy, and CTA buttons (Save, Execute, Delete).
- **Activity Log**: Optionally embed a timeline of updates (if API available) to help track changes.

### Execution Console

The runner view should:

1. Fetch the transformation definition.
2. Render optional runtime override controls (e.g., filter overrides, pagination adjustments).
3. Execute via `executeEntityTransformation` mutation or query.
4. Display results in a tabular view where each edge renders as a row, with nested entity cards showing the path.
5. Support pagination controls using `limit`/`offset` on the execution input.
6. Provide error/latency indicators and download to CSV/JSON.

### Validation Strategy

- **Graph Integrity**: Every node except LOAD must have at least one source. JOIN/UNION nodes require two or more sources. Detect cycles before save.
- **Config Completeness**: Validate type-specific fields (e.g., JOIN requires `onField` unless `type == CROSS`).
- **Backend Errors**: Surface GraphQL errors inline, highlighting offending nodes when possible.

## Component Breakdown

| Component | Responsibility |
|-----------|----------------|
| `TransformationListTable` | Displays all transformations for the selected organization. |
| `TransformationCanvas` | Wraps React Flow canvas, handles drag-and-drop and edge creation. |
| `NodePalette` | Lists available node templates. |
| `NodeInspector` | Renders form for editing selected node config. |
| `TransformationToolbar` | Global actions (save, undo, redo, execute). |
| `ExecutionRunner` | Form + results grid for execution view. |
| `ResultEdgeCard` | Renders a single `EntityTransformationRecordEdge`. |

## Styling & UX Notes

- Keep visual language consistent with existing entity joins tooling (use Tailwind primitives and design tokens already in `src/styles`).
- Show status badges on nodes (e.g., warning if config invalid, success when last run succeeded).
- Provide keyboard shortcuts for zoom/reset, copy/paste nodes, and toggling layout.
- Make the canvas responsive; persist viewport state per user via local storage.

## Developer Workflow

1. **Scaffold Routes & Components**
   - Create route files, wrap them with Suspense + TanStack Query boundaries.
2. **Define GraphQL Operations**
   - Add `.graphql` documents alongside components (e.g., `TransformationsList.graphql` for list query).
   - Run `npm run codegen` to generate hooks.
3. **Implement Canvas State**
   - Use `useState`/`useReducer` to maintain DAG, or integrate React Flow's state management.
   - Derive `EntityTransformationNodeInput[]` via selector functions.
4. **Hook Up Mutations**
   - On save, call mutation and sync returned definition into cache.
5. **Testing**
   - Write Vitest component tests for serialization helpers.
   - Add Cypress (or Playwright) e2e scripts to validate basic workflows (optional but recommended).

## Sample Serialization Helper

```ts
import { EntityTransformationNodeInput } from "@/generated/graphql";
import { TopologicalSorter } from "./topological";

export function serializeCanvas(nodes, edges): EntityTransformationNodeInput[] {
  const sorter = new TopologicalSorter(nodes, edges);
  return sorter.order().map((node) => ({
    id: node.id,
    type: node.data.type,
    sources: edges
      .filter((edge) => edge.target === node.id)
      .map((edge) => edge.source),
    load: node.data.load ?? null,
    filter: node.data.filter ?? null,
    project: node.data.project ?? null,
    join: node.data.join ?? null,
    sort: node.data.sort ?? null,
    paginate: node.data.paginate ?? null,
  }));
}
```

## Manual QA Checklist

- [ ] Creating a transformation persists nodes with correct topological order.
- [ ] Updating node configuration reflects immediately in canvas and saved payload.
- [ ] Attempting to create a cycle is prevented with a tooltip error.
- [ ] Executing a transformation shows results and respects pagination inputs.
- [ ] Deleting a transformation removes it from the list and navigates away.
- [ ] GraphQL errors produce inline toasts and keep unsaved state intact.

Following this guide ensures the frontend delivers a polished, analyst-friendly interface for building and running entity transformation DAGs while staying aligned with the backend schema and tooling.
