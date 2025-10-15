# GraphQL Engineering Data Management API

A Go and PostgreSQL powered GraphQL API for engineering data management with dynamic schema capabilities, hierarchical data support, and JSONB-based flexible field storage.

## Features

- **Dynamic Schema Definition**: Users can define custom entity types and fields via GraphQL
- **Hierarchical Data**: Built-in support for tree structures using PostgreSQL ltree extension
- **JSONB Storage**: Flexible field storage using PostgreSQL JSONB for dynamic properties
- **Organization Scoping**: All data is scoped by organization ID (multi-tenant ready)
- **Type-Safe**: Generated GraphQL resolvers and SQL queries using gqlgen and sqlc
- **Functional Paradigm**: Immutable data structures and pure functions where possible
- **Extensible**: Designed for future features like file uploads, 3D data, and timeseries

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   GraphQL API   │────│   Repository     │────│   PostgreSQL    │
│   (gqlgen)      │    │   Layer          │    │   + ltree       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    ┌─────────┐            ┌──────────┐            ┌──────────┐
    │Resolvers│            │   SQLC   │            │JSONB +   │
    │         │            │ Generated│            │Hierarchy │
    └─────────┘            └──────────┘            └──────────┘
```

## Project Structure

```
github.com/rpattn/engql/
├── cmd/server/           # Application entry point
├── internal/
│   ├── domain/          # Core business entities
│   ├── repository/      # Data access layer interfaces & implementations
│   ├── graphql/         # GraphQL resolvers
│   └── db/              # Database connection & migrations
├── graph/               # gqlgen generated GraphQL code
├── migrations/          # SQL migration files
├── sql/                 # SQLC query definitions
├── gqlgen.yml           # GraphQL code generation config
├── sqlc.yaml           # SQL code generation config
└── README.md
```

## Database Schema

### Core Tables

- **organizations**: Organization registry
- **entity_schemas**: Schema definitions for entity types
- **entities**: Dynamic entity data with JSONB properties and ltree paths

### Key Features

- **ltree Extension**: Hierarchical path storage (e.g., "1.2.3" for nested entities)
- **JSONB Properties**: Flexible field storage for dynamic schemas
- **Automatic Validation**: Database-level validation of entity properties against schemas
- **Audit Trail**: Created/updated timestamps on all entities

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+ with ltree extension
- Make (optional)

### Setup

1. **Clone and install dependencies**:
   ```bash
   git clone <repository>
   cd github.com/rpattn/engql
   go mod tidy
   ```

2. **Setup PostgreSQL database**:
   ```bash
   # Create database
   createdb engineering_api
   
   # Enable ltree extension
   psql engineering_api -c "CREATE EXTENSION IF NOT EXISTS ltree;"
   ```

3. **Configure database connection**:
   Update the connection settings in `cmd/server/main.go` if needed:
   ```go
   config := db.Config{
       Host:     "localhost",
       Port:     5432,
       User:     "postgres",
       Password: "admin",
       DBName:   "engineering_api",
       SSLMode:  "disable",
   }
   ```

4. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

5. **Access GraphQL playground**:
   Open http://localhost:8080 in your browser

## Dev

To generate graphql stuff
```bash
go get github.com/99designs/gqlgen@v0.17.81
go run github.com/99designs/gqlgen generate
```

To generate sqlc
`sqlc generate`

To test
`go test ./...`

To run
`go run cmd/server/main.go`

## Example Usage

### 1. Create an Organization

```graphql
mutation {
  createOrganization(input: {
    name: "ACME Engineering"
    description: "Engineering team for ACME Corp"
  }) {
    id
    name
    description
    createdAt
  }
}
```

### 2. Define an Entity Schema

```graphql
mutation {
  createEntitySchema(input: {
    organizationId: "your-org-id"
    name: "Component"
    description: "Engineering components"
    fields: [
      {
        name: "name"
        type: STRING
        required: true
        description: "Component name"
      }
      {
        name: "material"
        type: STRING
        required: false
        description: "Component material"
      }
      {
        name: "weight"
        type: FLOAT
        required: false
        description: "Component weight in kg"
      }
    ]
  }) {
    id
    name
    fields {
      name
      type
      required
    }
  }
}
```

### 3. Create Entities

```graphql
mutation {
  createEntity(input: {
    organizationId: "your-org-id"
    entityType: "Component"
    path: "1"
    properties: "{\"name\": \"Steel Bracket\", \"material\": \"Steel\", \"weight\": 2.5}"
  }) {
    id
    entityType
    path
    properties
    createdAt
  }
}
```

### 4. Query Entities with Hierarchical Relationships

```graphql
query {
  entities(organizationId: "your-org-id") {
    entities {
      id
      entityType
      path
      properties
    }
    pageInfo {
      totalCount
      hasNextPage
    }
  }
}
```

### 5. Hierarchical Data Queries

```graphql
# Get all ancestors of an entity
query {
  getEntityAncestors(entityId: "entity-id") {
    id
    entityType
    path
    properties
  }
}

# Get all descendants of an entity
query {
  getEntityDescendants(entityId: "entity-id") {
    id
    entityType
    path
    properties
  }
}

# Get direct children of an entity
query {
  getEntityChildren(entityId: "entity-id") {
    id
    entityType
    path
    properties
  }
}

# Get siblings of an entity
query {
  getEntitySiblings(entityId: "entity-id") {
    id
    entityType
    path
    properties
  }
}

# Get complete hierarchy tree
query {
  getEntityHierarchy(entityId: "entity-id") {
    current {
      id
      entityType
      path
      properties
    }
    ancestors {
      id
      path
    }
    children {
      id
      path
    }
    siblings {
      id
      path
    }
  }
}
```

### 6. JSONB Property Queries

```graphql
# Search entities by specific property value
query {
  searchEntitiesByProperty(
    organizationId: "your-org-id"
    propertyKey: "material"
    propertyValue: "Steel"
  ) {
    id
    entityType
    properties
  }
}

# Search entities by multiple properties
query {
  searchEntitiesByMultipleProperties(
    organizationId: "your-org-id"
    filters: "{\"material\": \"Steel\", \"weight\": 2.5}"
  ) {
    id
    entityType
    properties
  }
}

# Search entities by property range
query {
  searchEntitiesByPropertyRange(
    organizationId: "your-org-id"
    propertyKey: "weight"
    minValue: 1.0
    maxValue: 5.0
  ) {
    id
    entityType
    properties
  }
}

# Search entities that have a specific property
query {
  searchEntitiesByPropertyExists(
    organizationId: "your-org-id"
    propertyKey: "material"
  ) {
    id
    entityType
    properties
  }
}

# Search entities by property contains
query {
  searchEntitiesByPropertyContains(
    organizationId: "your-org-id"
    propertyKey: "name"
    searchTerm: "bracket"
  ) {
    id
    entityType
    properties
  }
}

# Validate entity against its schema
query {
  validateEntityAgainstSchema(entityId: "entity-id") {
    isValid
    errors
    warnings
  }
}
```

### 7. Query Entity Schemas

```graphql
query {
  entitySchemas(organizationId: "your-org-id") {
    id
    name
    description
    fields {
      name
      type
      required
      description
    }
  }
}
```

## Advanced Features

### Hierarchical Queries

The system supports comprehensive hierarchical queries using PostgreSQL ltree:

- **Ancestors**: Get all parent entities up the tree
- **Descendants**: Get all child entities down the tree
- **Children**: Get direct child entities only
- **Siblings**: Get entities at the same hierarchical level
- **Complete Hierarchy**: Get the full tree structure (current + ancestors + children + siblings)
- **Path Management**: Automatic path generation and validation
- **Path Operations**: Parent/child relationships, depth calculation, path comparison

### JSONB Property Filtering & Validation

Comprehensive dynamic field operations using PostgreSQL JSONB:

**Property Queries:**
```sql
-- Filter by property existence
SELECT * FROM entities WHERE properties ? 'material';

-- Filter by property value
SELECT * FROM entities WHERE properties ->> 'material' = 'Steel';

-- Filter by property in array
SELECT * FROM entities WHERE properties ->> 'category' = ANY(ARRAY['Mechanical', 'Electrical']);
```

**Advanced JSONB Operations:**
- **Property Search**: Find entities by specific property values
- **Multi-Property Filters**: Query by multiple property combinations
- **Range Queries**: Search numeric properties within min/max ranges
- **Existence Checks**: Find entities that have specific properties
- **Substring Search**: Case-insensitive text search within properties
- **Schema Validation**: Validate entity properties against defined schemas
- **Type Validation**: Ensure properties match expected field types
- **Custom Validation Rules**: Min/max values, length constraints, pattern matching

### Field Type Support

Current field types:
- `STRING`: Text values
- `INTEGER`: Whole numbers
- `FLOAT`: Decimal numbers
- `BOOLEAN`: True/false values
- `TIMESTAMP`: Date/time values
- `JSON`: Complex nested objects

Future field types (extensible):
- `FILE_REFERENCE`: File uploads and references
- `GEOMETRY`: 3D/2D geometric data
- `TIMESERIES`: Time-series data points

## Development

### Code Generation

Generate GraphQL resolvers:
```bash
gqlgen generate
```

Generate SQL queries:
```bash
sqlc generate
```

### Database Migrations

Create a new migration:
```bash
# Create migration files
touch migrations/002_add_indexes.up.sql
touch migrations/002_add_indexes.down.sql
```

Run migrations:
```bash
# Migrations run automatically on server start
go run cmd/server/main.go
```

### Testing

Run tests:
```bash
go test ./...
```

## API Reference

### Queries

- `organizations`: List all organizations
- `organization(id)`: Get organization by ID
- `entitySchemas(organizationId)`: List entity schemas
- `entities(organizationId, filter, pagination)`: List entities with filtering
- `entity(id)`: Get entity by ID

### Mutations

- `createOrganization(input)`: Create new organization
- `createEntitySchema(input)`: Define new entity schema
- `addFieldToSchema(schemaId, field)`: Add field to existing schema
- `createEntity(input)`: Create new entity instance
- `updateEntity(input)`: Update existing entity

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Roadmap

- [ ] Authentication and authorization
- [ ] File upload support
- [ ] 3D geometry data types
- [ ] Time-series data support
- [ ] GraphQL subscriptions
- [ ] API rate limiting
- [ ] Multi-language support
- [ ] Advanced query optimization
