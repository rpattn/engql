-- Drop triggers
DROP TRIGGER IF EXISTS validate_entity_properties_trigger ON entities;
DROP TRIGGER IF EXISTS update_entities_updated_at ON entities;
DROP TRIGGER IF EXISTS update_entity_schemas_updated_at ON entity_schemas;
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop functions
DROP FUNCTION IF EXISTS validate_entity_properties();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables
DROP TABLE IF EXISTS entities;
DROP TABLE IF EXISTS entity_schemas;
DROP TABLE IF EXISTS organizations;

-- Drop extensions (be careful - only if not used elsewhere)
-- DROP EXTENSION IF EXISTS "ltree";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
