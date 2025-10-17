--
-- PostgreSQL database dump
--

\restrict 06OgbcTkdfM3zUE2mhNPLfwlfltTN1Oz2ZaxD4kKjzEk4pSHzau4hafCBvPez2W

-- Dumped from database version 16.10 (Ubuntu 16.10-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.10 (Ubuntu 16.10-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: ltree; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA public;


--
-- Name: EXTENSION ltree; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION ltree IS 'data type for hierarchical tree-like structures';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: archive_entity_change(); Type: FUNCTION; Schema: public; Owner: rpatt
--

CREATE FUNCTION public.archive_entity_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    audit_reason TEXT;
    change_kind TEXT := TG_OP;
BEGIN
    audit_reason := current_setting('app.reason', true);
    IF audit_reason IS NULL OR audit_reason = '' THEN
        audit_reason := TG_OP;
    ELSIF audit_reason ~* '^ROLLBACK' THEN
        change_kind := 'ROLLBACK';
    END IF;

    INSERT INTO entities_history (
        entity_id,
        organization_id,
        schema_id,
        entity_type,
        path,
        properties,
        created_at,
        updated_at,
        version,
        change_type,
        changed_at,
        reason
    )
    VALUES (
        OLD.id,
        OLD.organization_id,
        OLD.schema_id,
        OLD.entity_type,
        OLD.path,
        OLD.properties,
        OLD.created_at,
        OLD.updated_at,
        OLD.version,
        change_kind,
        NOW(),
        audit_reason
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.archive_entity_change() OWNER TO rpatt;

--
-- Name: bump_entity_version(); Type: FUNCTION; Schema: public; Owner: rpatt
--

CREATE FUNCTION public.bump_entity_version() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.version := COALESCE(OLD.version, 0) + 1;
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.bump_entity_version() OWNER TO rpatt;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: rpatt
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO rpatt;

--
-- Name: validate_entity_properties(); Type: FUNCTION; Schema: public; Owner: rpatt
--

CREATE FUNCTION public.validate_entity_properties() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    schema_fields JSONB;
    field_def JSONB;
    field_value JSONB;
    i INT;
    field_name TEXT;
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

    -- Loop over array elements
    FOR i IN 0 .. jsonb_array_length(schema_fields) - 1
    LOOP
        field_def := schema_fields->i;
        field_name := field_def->>'name';
        field_value := NEW.properties->field_name;

        -- Check required fields
        IF (field_def->>'required')::boolean AND (field_value IS NULL OR field_value = 'null') THEN
            RAISE EXCEPTION 'Required field % is missing or null', field_name;
        END IF;

        -- TODO: Add type validation here (string, integer, float, boolean, etc.)
    END LOOP;

    RETURN NEW;
END;
$$;


ALTER FUNCTION public.validate_entity_properties() OWNER TO rpatt;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: entities; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.entities (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    entity_type character varying(255) NOT NULL,
    path public.ltree,
    properties jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    schema_id uuid NOT NULL,
    version bigint DEFAULT 1 NOT NULL
);


ALTER TABLE public.entities OWNER TO rpatt;

--
-- Name: entities_history; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.entities_history (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    entity_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    schema_id uuid NOT NULL,
    entity_type character varying(255) NOT NULL,
    path public.ltree,
    properties jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    version bigint NOT NULL,
    change_type text NOT NULL,
    changed_at timestamp with time zone DEFAULT now() NOT NULL,
    reason text,
    CONSTRAINT entities_history_change_type_check CHECK ((change_type = ANY (ARRAY['UPDATE'::text, 'DELETE'::text, 'ROLLBACK'::text])))
);


ALTER TABLE public.entities_history OWNER TO rpatt;

--
-- Name: entity_joins; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.entity_joins (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    left_entity_type character varying(255) NOT NULL,
    right_entity_type character varying(255) NOT NULL,
    join_field character varying(255),
    join_field_type character varying(64),
    left_filters jsonb DEFAULT '[]'::jsonb NOT NULL,
    right_filters jsonb DEFAULT '[]'::jsonb NOT NULL,
    sort_criteria jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    join_type character varying(32) DEFAULT 'REFERENCE'::character varying NOT NULL
);


ALTER TABLE public.entity_joins OWNER TO rpatt;

--
-- Name: entity_schemas; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.entity_schemas (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    fields jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    version text DEFAULT '1.0.0'::text NOT NULL,
    previous_version_id uuid,
    status text DEFAULT 'ACTIVE'::text NOT NULL,
    CONSTRAINT entity_schemas_status_check CHECK ((status = ANY (ARRAY['ACTIVE'::text, 'DEPRECATED'::text, 'ARCHIVED'::text, 'DRAFT'::text]))),
    CONSTRAINT entity_schemas_version_format_check CHECK ((version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'::text))
);


ALTER TABLE public.entity_schemas OWNER TO rpatt;

--
-- Name: ingestion_logs; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.ingestion_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    organization_id uuid NOT NULL,
    schema_name character varying(255) NOT NULL,
    file_name text NOT NULL,
    row_number integer,
    error_message text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.ingestion_logs OWNER TO rpatt;

--
-- Name: organizations; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.organizations OWNER TO rpatt;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: rpatt
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO rpatt;

--
-- Name: entities_history entities_history_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entities_history
    ADD CONSTRAINT entities_history_pkey PRIMARY KEY (id);


--
-- Name: entities entities_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_pkey PRIMARY KEY (id);


--
-- Name: entity_joins entity_joins_organization_id_name_key; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_joins
    ADD CONSTRAINT entity_joins_organization_id_name_key UNIQUE (organization_id, name);


--
-- Name: entity_joins entity_joins_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_joins
    ADD CONSTRAINT entity_joins_pkey PRIMARY KEY (id);


--
-- Name: entity_schemas entity_schemas_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_schemas
    ADD CONSTRAINT entity_schemas_pkey PRIMARY KEY (id);


--
-- Name: ingestion_logs ingestion_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.ingestion_logs
    ADD CONSTRAINT ingestion_logs_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: entities_history_entity_version_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE UNIQUE INDEX entities_history_entity_version_idx ON public.entities_history USING btree (entity_id, version);


--
-- Name: entities_id_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX entities_id_idx ON public.entities USING btree (id);


--
-- Name: entity_joins_left_type_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX entity_joins_left_type_idx ON public.entity_joins USING btree (left_entity_type);


--
-- Name: entity_joins_org_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX entity_joins_org_idx ON public.entity_joins USING btree (organization_id);


--
-- Name: entity_joins_right_type_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX entity_joins_right_type_idx ON public.entity_joins USING btree (right_entity_type);


--
-- Name: entity_schemas_id_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX entity_schemas_id_idx ON public.entity_schemas USING btree (id);


--
-- Name: entity_schemas_org_name_version_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE UNIQUE INDEX entity_schemas_org_name_version_idx ON public.entity_schemas USING btree (organization_id, name, version);


--
-- Name: idx_entities_created_at; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_entities_created_at ON public.entities USING btree (created_at);


--
-- Name: idx_entities_org_type; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_entities_org_type ON public.entities USING btree (organization_id, entity_type);


--
-- Name: idx_entities_path; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_entities_path ON public.entities USING gist (path);


--
-- Name: idx_entities_properties; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_entities_properties ON public.entities USING gin (properties);


--
-- Name: idx_entity_schemas_org_name; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_entity_schemas_org_name ON public.entity_schemas USING btree (organization_id, name);


--
-- Name: idx_ingestion_logs_org; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_ingestion_logs_org ON public.ingestion_logs USING btree (organization_id);


--
-- Name: idx_ingestion_logs_org_schema; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_ingestion_logs_org_schema ON public.ingestion_logs USING btree (organization_id, schema_name);


--
-- Name: idx_organizations_name; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX idx_organizations_name ON public.organizations USING btree (name);


--
-- Name: organizations_id_idx; Type: INDEX; Schema: public; Owner: rpatt
--

CREATE INDEX organizations_id_idx ON public.organizations USING btree (id);


--
-- Name: entities entities_archive_change; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER entities_archive_change BEFORE DELETE OR UPDATE ON public.entities FOR EACH ROW EXECUTE FUNCTION public.archive_entity_change();


--
-- Name: entities entities_version_bump; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER entities_version_bump BEFORE UPDATE ON public.entities FOR EACH ROW EXECUTE FUNCTION public.bump_entity_version();


--
-- Name: entities update_entities_updated_at; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER update_entities_updated_at BEFORE UPDATE ON public.entities FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: entity_joins update_entity_joins_updated_at; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER update_entity_joins_updated_at BEFORE UPDATE ON public.entity_joins FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: entity_schemas update_entity_schemas_updated_at; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER update_entity_schemas_updated_at BEFORE UPDATE ON public.entity_schemas FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: organizations update_organizations_updated_at; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON public.organizations FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


--
-- Name: entities validate_entity_properties_trigger; Type: TRIGGER; Schema: public; Owner: rpatt
--

CREATE TRIGGER validate_entity_properties_trigger BEFORE INSERT OR UPDATE ON public.entities FOR EACH ROW EXECUTE FUNCTION public.validate_entity_properties();


--
-- Name: entities_history entities_history_schema_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entities_history
    ADD CONSTRAINT entities_history_schema_id_fkey FOREIGN KEY (schema_id) REFERENCES public.entity_schemas(id) ON DELETE CASCADE;


--
-- Name: entities entities_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: entities entities_schema_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_schema_id_fkey FOREIGN KEY (schema_id) REFERENCES public.entity_schemas(id);


--
-- Name: entity_joins entity_joins_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_joins
    ADD CONSTRAINT entity_joins_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: entity_schemas entity_schemas_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_schemas
    ADD CONSTRAINT entity_schemas_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: entity_schemas entity_schemas_previous_version_fk; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.entity_schemas
    ADD CONSTRAINT entity_schemas_previous_version_fk FOREIGN KEY (previous_version_id) REFERENCES public.entity_schemas(id);


--
-- Name: ingestion_logs ingestion_logs_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: rpatt
--

ALTER TABLE ONLY public.ingestion_logs
    ADD CONSTRAINT ingestion_logs_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict 06OgbcTkdfM3zUE2mhNPLfwlfltTN1Oz2ZaxD4kKjzEk4pSHzau4hafCBvPez2W

