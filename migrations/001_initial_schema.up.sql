-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "ltree";

-- Organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Entity schemas table - stores field definitions for entity types
CREATE TABLE entity_schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    fields JSONB NOT NULL DEFAULT '{}', -- Schema definition: {"field_name": {"type": "string", "required": true, ...}}
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(organization_id, name)
);

-- Entities table - stores all dynamic data with JSONB properties
CREATE TABLE entities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type VARCHAR(255) NOT NULL, -- References entity_schemas.name
    path ltree, -- Hierarchical path using ltree
    properties JSONB NOT NULL DEFAULT '{}', -- Dynamic field data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (organization_id, entity_type) REFERENCES entity_schemas(organization_id, name)
);

-- Indexes for performance
CREATE INDEX idx_organizations_name ON organizations(name);
CREATE INDEX idx_entity_schemas_org_name ON entity_schemas(organization_id, name);
CREATE INDEX idx_entities_org_type ON entities(organization_id, entity_type);
CREATE INDEX idx_entities_path ON entities USING GIST(path);
CREATE INDEX idx_entities_properties ON entities USING GIN(properties);
CREATE INDEX idx_entities_created_at ON entities(created_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entity_schemas_updated_at BEFORE UPDATE ON entity_schemas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entities_updated_at BEFORE UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to validate entity properties against schema
CREATE OR REPLACE FUNCTION validate_entity_properties()
RETURNS TRIGGER AS $$
DECLARE
    schema_fields JSONB;
    field_name TEXT;
    field_def JSONB;
    field_value JSONB;
BEGIN
    -- Get the schema for this entity type
    SELECT fields INTO schema_fields 
    FROM entity_schemas 
    WHERE organization_id = NEW.organization_id 
    AND name = NEW.entity_type;
    
    -- If no schema found, allow empty properties
    IF schema_fields IS NULL THEN
        RETURN NEW;
    END IF;
    
    -- Validate each field in the schema
    FOR field_name IN SELECT jsonb_object_keys(schema_fields)
    LOOP
        field_def := schema_fields->field_name;
        field_value := NEW.properties->field_name;
        
        -- Check required fields
        IF (field_def->>'required')::boolean AND (field_value IS NULL OR field_value = 'null') THEN
            RAISE EXCEPTION 'Required field % is missing or null', field_name;
        END IF;
        
        -- TODO: Add type validation here (string, integer, float, boolean, etc.)
    END LOOP;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to validate entity properties
CREATE TRIGGER validate_entity_properties_trigger 
    BEFORE INSERT OR UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION validate_entity_properties();
