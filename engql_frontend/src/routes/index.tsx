import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { graphqlRequest } from "../lib/graphql";

type FieldDefinitionInput = {
  name: string;
  type: string;
  required?: boolean;
  description?: string;
  default?: string;
  validation?: string;
  referenceEntityType?: string;
};

type CreateSchemaResponse = {
  createEntitySchema: {
    id: string;
    name: string;
    description?: string | null;
    version: string;
    status: string;
    previousVersionId?: string | null;
  };
};

type CreateEntityResponse = {
  createEntity: {
    id: string;
    entityType: string;
    schemaId: string;
    version: number;
    properties: string;
  };
};

type EntitiesByTypeResponse = {
  entitiesByType: Array<{
    id: string;
    entityType: string;
    schemaId: string;
    version: number;
    properties: string;
    linkedEntities: Array<{
      id: string;
      entityType: string;
      properties: string;
    }>;
  }>;
};

type SchemaVersionInfo = {
  id: string;
  version: string;
  status: string;
  createdAt: string;
  previousVersionId?: string | null;
};

type SchemaVersionsResponse = {
  entitySchemaVersions: SchemaVersionInfo[];
};

type RollbackEntityResponse = {
  rollbackEntity: {
    id: string;
    version: number;
    properties: string;
  };
};

type SchemaMeta = {
  id: string;
  name: string;
  version: string;
  status: string;
  previousVersionId?: string | null;
};

type EntityRow = {
  id: string;
  entityType: string;
  schemaId: string;
  version: number;
  name: string | null;
  linkedCount: number;
  linkedSummary: string;
  propertiesJSON: string;
};

const entityColumnHelper = createColumnHelper<EntityRow>();

const CREATE_SCHEMA_MUTATION = `
  mutation CreateSchema($input: CreateEntitySchemaInput!) {
    createEntitySchema(input: $input) {
      id
      name
      description
      version
      status
      previousVersionId
    }
  }
`;

const CREATE_ENTITY_MUTATION = `
  mutation CreateEntity($input: CreateEntityInput!) {
    createEntity(input: $input) {
      id
      entityType
      schemaId
      version
      properties
    }
  }
`;

const ENTITIES_BY_TYPE_QUERY = `
  query EntitiesByType($organizationId: String!, $entityType: String!) {
    entitiesByType(organizationId: $organizationId, entityType: $entityType) {
      id
      entityType
      schemaId
      version
      properties
      linkedEntities {
        id
        entityType
        properties
      }
    }
  }
`;

const SCHEMA_VERSIONS_QUERY = `
  query SchemaVersions($organizationId: String!, $name: String!) {
    entitySchemaVersions(organizationId: $organizationId, name: $name) {
      id
      version
      status
      previousVersionId
      createdAt
    }
  }
`;

const ROLLBACK_ENTITY_MUTATION = `
  mutation RollbackEntity($id: String!, $toVersion: Int!, $reason: String) {
    rollbackEntity(id: $id, toVersion: $toVersion, reason: $reason) {
      id
      version
      properties
    }
  }
`;

export const Route = createFileRoute("/")({
  component: App,
});

function App() {
  const [organizationId, setOrganizationId] = useState("");

  const [schemaName, setSchemaName] = useState("");
  const [schemaDescription, setSchemaDescription] = useState("");
  const [schemaFieldsInput, setSchemaFieldsInput] = useState(
    JSON.stringify(
      [
        {
          name: "name",
          type: "STRING",
          required: true,
        },
      ],
      null,
      2,
    ),
  );
  const [schemaFormError, setSchemaFormError] = useState<string | null>(null);
  const [lastSchemaMeta, setLastSchemaMeta] = useState<SchemaMeta | null>(null);

  const [entityType, setEntityType] = useState("");
  const [entityPath, setEntityPath] = useState("");
  const [entityPropertiesInput, setEntityPropertiesInput] = useState(
    JSON.stringify(
      {
        name: "Example Entity",
        linked_ids: [],
      },
      null,
      2,
    ),
  );
  const [primaryLinkedId, setPrimaryLinkedId] = useState("");
  const [additionalLinkedIds, setAdditionalLinkedIds] = useState("");
  const [entityFormError, setEntityFormError] = useState<string | null>(null);

  const [queryEntityType, setQueryEntityType] = useState("");
  const [queryError, setQueryError] = useState<string | null>(null);
  const [resultView, setResultView] = useState<"cards" | "grid">("cards");
  const [rollbackEntityId, setRollbackEntityId] = useState("");
  const [rollbackTargetVersion, setRollbackTargetVersion] = useState("");
  const [rollbackReason, setRollbackReason] = useState("");
  const [rollbackMessage, setRollbackMessage] = useState<string | null>(null);

  const createSchemaMutation = useMutation({
    mutationFn: (variables: {
      organizationId: string;
      name: string;
      description?: string;
      fields: FieldDefinitionInput[];
    }) =>
      graphqlRequest<CreateSchemaResponse>(CREATE_SCHEMA_MUTATION, {
        input: {
          organizationId: variables.organizationId,
          name: variables.name,
          description: variables.description,
          fields: variables.fields,
        },
      }),
  });

  const createEntityMutation = useMutation({
    mutationFn: (variables: {
      organizationId: string;
      entityType: string;
      path?: string;
      properties: string;
      linkedEntityId?: string;
      linkedEntityIds?: string[];
    }) =>
      graphqlRequest<CreateEntityResponse>(CREATE_ENTITY_MUTATION, {
        input: {
          organizationId: variables.organizationId,
          entityType: variables.entityType,
          path: variables.path,
          properties: variables.properties,
          linkedEntityId: variables.linkedEntityId,
          linkedEntityIds: variables.linkedEntityIds,
        },
      }),
  });

  const rollbackEntityMutation = useMutation({
    mutationFn: (variables: {
      id: string;
      toVersion: number;
      reason?: string;
    }) =>
      graphqlRequest<RollbackEntityResponse>(ROLLBACK_ENTITY_MUTATION, {
        id: variables.id,
        toVersion: variables.toVersion,
        reason: variables.reason ?? null,
      }),
  });

  const {
    data: entitiesData,
    isFetching: isFetchingEntities,
    refetch: refetchEntities,
  } = useQuery({
    queryKey: ["entitiesByType", organizationId, queryEntityType],
    queryFn: () =>
      graphqlRequest<EntitiesByTypeResponse>(ENTITIES_BY_TYPE_QUERY, {
        organizationId,
        entityType: queryEntityType,
      }),
    enabled: false,
  });

  const {
    data: schemaVersionsData,
    isFetching: isFetchingSchemaVersions,
    refetch: refetchSchemaVersions,
    error: schemaVersionsError,
  } = useQuery({
    queryKey: ["schemaVersions", organizationId, schemaName],
    queryFn: () =>
      graphqlRequest<SchemaVersionsResponse>(SCHEMA_VERSIONS_QUERY, {
        organizationId: organizationId.trim(),
        name: schemaName.trim(),
      }),
    enabled: false,
  });

  const schemaVersions = schemaVersionsData?.entitySchemaVersions ?? [];
  const schemaVersionsErrorMessage =
    schemaVersionsError instanceof Error ? schemaVersionsError.message : null;

  const createdEntityInfo = createEntityMutation.data?.createEntity;

  const safeParseProperties = (value: string) => {
    try {
      return JSON.parse(value);
    } catch {
      return value;
    }
  };

  const extractName = (value: unknown): string | undefined => {
    if (
      value &&
      typeof value === "object" &&
      value !== null &&
      "name" in value &&
      typeof (value as Record<string, unknown>)["name"] === "string"
    ) {
      return (value as Record<string, unknown>)["name"] as string;
    }
    return undefined;
  };

  const entityRows = useMemo<EntityRow[]>(() => {
    if (!entitiesData?.entitiesByType) {
      return [];
    }

    return entitiesData.entitiesByType.map((entity) => {
      const parsedProps = safeParseProperties(entity.properties);
      const name = extractName(parsedProps) ?? null;

      const propertiesJSON =
        typeof parsedProps === "string"
          ? parsedProps
          : JSON.stringify(parsedProps, null, 2);

      const linkedSummary = entity.linkedEntities
        .map((link) => {
          const linkedProps = safeParseProperties(link.properties);
          const linkedName = extractName(linkedProps);
          return linkedName
            ? `${link.entityType}: ${linkedName}`
            : `${link.entityType}: ${link.id}`;
        })
        .join(", ");

      return {
        id: entity.id,
        entityType: entity.entityType,
        schemaId: entity.schemaId,
        version: entity.version,
        name,
        linkedCount: entity.linkedEntities.length,
        linkedSummary: linkedSummary || "-",
        propertiesJSON,
      };
    });
  }, [entitiesData]);

  const columns = useMemo(
    () => [
      entityColumnHelper.accessor("entityType", {
        header: "Type",
        cell: (info) => info.getValue(),
      }),
      entityColumnHelper.accessor("id", {
        header: "ID",
        cell: (info) => (
          <code className="text-xs text-slate-300">{info.getValue()}</code>
        ),
      }),
      entityColumnHelper.accessor("schemaId", {
        header: "Schema",
        cell: (info) => {
          const schemaId = info.getValue();
          const label =
            schemaId.length > 12 ? `${schemaId.slice(0, 8)}...` : schemaId;
          return <code className="text-xs text-slate-400">{label}</code>;
        },
      }),
      entityColumnHelper.accessor("version", {
        header: "Version",
        cell: (info) => info.getValue(),
      }),
      entityColumnHelper.accessor("name", {
        header: "Name",
        cell: (info) => info.getValue() ?? "—",
      }),
      entityColumnHelper.accessor("linkedCount", {
        header: "# Linked",
        cell: (info) => info.getValue(),
      }),
      entityColumnHelper.accessor("linkedSummary", {
        header: "Linked Entities",
        cell: (info) => (
          <span className="whitespace-pre-wrap text-slate-300">
            {info.getValue()}
          </span>
        ),
      }),
      entityColumnHelper.accessor("propertiesJSON", {
        header: "Properties",
        cell: (info) => (
          <pre className="max-h-48 overflow-auto rounded-md bg-slate-950/70 p-2 text-[11px] leading-snug text-slate-200">
            {info.getValue()}
          </pre>
        ),
      }),
    ],
    [],
  );

  const table = useReactTable({
    data: entityRows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  const hasResults = entityRows.length > 0;

  const handleCreateSchema = async (
    event: React.FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();
    if (!organizationId.trim()) {
      setSchemaFormError("Organization ID is required.");
      return;
    }
    if (!schemaName.trim()) {
      setSchemaFormError("Schema name is required.");
      return;
    }

    let fields: FieldDefinitionInput[];
    try {
      fields = JSON.parse(schemaFieldsInput);
    } catch (error) {
      setSchemaFormError("Fields must be valid JSON.");
      return;
    }

    if (!Array.isArray(fields)) {
      setSchemaFormError("Fields JSON must describe an array.");
      return;
    }

    setSchemaFormError(null);

    try {
      const result = await createSchemaMutation.mutateAsync({
        organizationId: organizationId.trim(),
        name: schemaName.trim(),
        description: schemaDescription.trim() || undefined,
        fields,
      });
      const created = result.createEntitySchema;
      setLastSchemaMeta({
        id: created.id,
        name: schemaName.trim(),
        version: created.version,
        status: created.status,
        previousVersionId: created.previousVersionId ?? null,
      });
      if (organizationId.trim() && schemaName.trim()) {
        await refetchSchemaVersions();
      }
    } catch (error) {
      if (error instanceof Error) {
        setSchemaFormError(error.message);
      } else {
        setSchemaFormError("Failed to create schema.");
      }
    }
  };

  const handleFetchSchemaVersions = async () => {
    if (!organizationId.trim() || !schemaName.trim()) {
      setSchemaFormError(
        "Organization ID and schema name are required to load versions.",
      );
      return;
    }
    await refetchSchemaVersions();
  };

  const handleRollbackEntity = async (
    event: React.FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();
    if (!rollbackEntityId.trim()) {
      setRollbackMessage("Entity ID is required to rollback.");
      return;
    }
    const target = Number(rollbackTargetVersion);
    if (!Number.isInteger(target) || target < 1) {
      setRollbackMessage("Target version must be a positive integer.");
      return;
    }

    setRollbackMessage(null);

    try {
      const response = await rollbackEntityMutation.mutateAsync({
        id: rollbackEntityId.trim(),
        toVersion: target,
        reason: rollbackReason.trim() || undefined,
      });
      const rolled = response.rollbackEntity;
      setRollbackMessage(
        `Entity ${rolled.id} restored to version ${rolled.version}.`,
      );
      setRollbackEntityId("");
      setRollbackTargetVersion("");
      setRollbackReason("");

      if (organizationId.trim() && queryEntityType.trim()) {
        await refetchEntities();
      }
      if (organizationId.trim() && schemaName.trim()) {
        await refetchSchemaVersions();
      }
    } catch (error) {
      if (error instanceof Error) {
        setRollbackMessage(error.message);
      } else {
        setRollbackMessage("Rollback failed.");
      }
    }
  };

  const handleCreateEntity = async (
    event: React.FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();
    if (!organizationId.trim()) {
      setEntityFormError("Organization ID is required.");
      return;
    }
    if (!entityType.trim()) {
      setEntityFormError("Entity type is required.");
      return;
    }

    let propertiesObj: unknown;
    try {
      propertiesObj = JSON.parse(entityPropertiesInput);
    } catch (error) {
      setEntityFormError("Properties must be valid JSON.");
      return;
    }

    if (typeof propertiesObj !== "object" || propertiesObj === null) {
      setEntityFormError("Properties JSON must describe an object.");
      return;
    }

    const linkedIdsFromInput = additionalLinkedIds
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);

    const primaryLinked = primaryLinkedId.trim();
    const uniqueLinkedIds = Array.from(
      new Set(
        [
          primaryLinked.length ? primaryLinked : null,
          ...linkedIdsFromInput,
        ].filter(Boolean) as string[],
      ),
    );

    setEntityFormError(null);

    try {
      await createEntityMutation.mutateAsync({
        organizationId: organizationId.trim(),
        entityType: entityType.trim(),
        path: entityPath.trim() || undefined,
        properties: JSON.stringify(propertiesObj),
        linkedEntityId: primaryLinked.length > 0 ? primaryLinked : undefined,
        linkedEntityIds:
          uniqueLinkedIds.length > (primaryLinked.length ? 1 : 0)
            ? uniqueLinkedIds
            : undefined,
      });
    } catch (error) {
      if (error instanceof Error) {
        setEntityFormError(error.message);
      } else {
        setEntityFormError("Failed to create entity.");
      }
    }
  };

  const handleFetchEntities = async () => {
    if (!organizationId.trim() || !queryEntityType.trim()) {
      setQueryError("Organization ID and entity type are required.");
      return;
    }
    setQueryError(null);
    await refetchEntities();
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900">
      <main className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-10 text-slate-100">
        <section className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
          <h2 className="text-lg font-semibold text-white">
            Organization Context
          </h2>
          <p className="mt-1 text-sm text-slate-300">
            Provide the organization ID to scope schema and entity operations.
          </p>
          <div className="mt-4">
            <label className="block text-sm font-medium text-slate-200">
              Organization ID
            </label>
            <input
              value={organizationId}
              onChange={(event) => setOrganizationId(event.target.value)}
              placeholder="e.g. 4dc7d89e-..."
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
            />
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <div className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-lg font-semibold text-white">
                  Create Entity Schema
                </h2>
                {createSchemaMutation.isSuccess && lastSchemaMeta && (
                  <div className="mt-1 text-sm text-emerald-400">
                    Saved as v{lastSchemaMeta.version} · {lastSchemaMeta.status}
                  </div>
                )}
                {!createSchemaMutation.isSuccess && lastSchemaMeta && (
                  <div className="mt-1 text-xs text-slate-400">
                    Latest version: v{lastSchemaMeta.version} ·{" "}
                    {lastSchemaMeta.status}
                  </div>
                )}
              </div>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleFetchSchemaVersions}
                  disabled={
                    isFetchingSchemaVersions ||
                    !organizationId.trim() ||
                    !schemaName.trim()
                  }
                  className="inline-flex items-center justify-center rounded-lg border border-cyan-500/60 bg-transparent px-3 py-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/10 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {isFetchingSchemaVersions ? "Loading..." : "Load Versions"}
                </button>
              </div>
            </div>
            {schemaVersionsErrorMessage && (
              <div className="mt-3 rounded-md border border-red-500/60 bg-red-500/10 px-3 py-2 text-xs text-red-200">
                {schemaVersionsErrorMessage}
              </div>
            )}
            <form
              className="mt-4 flex flex-col gap-4"
              onSubmit={handleCreateSchema}
            >
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Schema Name
                </label>
                <input
                  value={schemaName}
                  onChange={(event) => setSchemaName(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Description (optional)
                </label>
                <input
                  value={schemaDescription}
                  onChange={(event) => setSchemaDescription(event.target.value)}
                  placeholder="Short description"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Fields JSON
                </label>
                <textarea
                  value={schemaFieldsInput}
                  onChange={(event) => setSchemaFieldsInput(event.target.value)}
                  rows={8}
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Provide an array of FieldDefinitionInput objects. Supported
                  types include STRING, ENTITY_REFERENCE, and
                  ENTITY_REFERENCE_ARRAY. ENTITY_REFERENCE fields link by
                  entity ID and auto-resolve the referenced entity when
                  querying.
                </p>
              </div>

              {schemaFormError && (
                <div className="rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                  {schemaFormError}
                </div>
              )}

              <button
                type="submit"
                disabled={createSchemaMutation.isPending}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {createSchemaMutation.isPending
                  ? "Creating..."
                  : "Create Schema"}
              </button>
            </form>
            {schemaVersions.length > 0 && (
              <div className="mt-6">
                <h3 className="text-sm font-semibold text-slate-200">
                  Version History
                </h3>
                <ul className="mt-3 space-y-2 text-xs">
                  {schemaVersions.map((versionInfo) => (
                    <li
                      key={versionInfo.id}
                      className="rounded-lg border border-slate-800 bg-slate-950/40 px-3 py-2 text-slate-300"
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-cyan-200">
                          v{versionInfo.version}
                        </span>
                        <span className="text-[11px] text-slate-500">
                          {new Date(versionInfo.createdAt).toLocaleString()}
                        </span>
                      </div>
                      <div className="mt-1 text-[11px] text-slate-400">
                        Status: {versionInfo.status}
                        {versionInfo.previousVersionId && (
                          <>
                            {" "}
                            - prev {versionInfo.previousVersionId.slice(0, 8)}
                            ...
                          </>
                        )}
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          <div className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">
                Create Entity
              </h2>
              {createEntityMutation.isSuccess && createdEntityInfo && (
                <span className="text-sm text-emerald-400">
                  Entity saved v{createdEntityInfo.version} - schema{" "}
                  {createdEntityInfo.schemaId.slice(0, 8)}...
                </span>
              )}
            </div>

            <form
              className="mt-4 flex flex-col gap-4"
              onSubmit={handleCreateEntity}
            >
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Entity Type
                </label>
                <input
                  value={entityType}
                  onChange={(event) => setEntityType(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Path (optional)
                </label>
                <input
                  value={entityPath}
                  onChange={(event) => setEntityPath(event.target.value)}
                  placeholder="root.component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Properties JSON
                </label>
                <textarea
                  value={entityPropertiesInput}
                  onChange={(event) =>
                    setEntityPropertiesInput(event.target.value)
                  }
                  rows={8}
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Provide an object. The UI will automatically maintain the
                  special linked_ids array.
                </p>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-slate-200">
                    Primary Linked Entity ID
                  </label>
                  <input
                    value={primaryLinkedId}
                    onChange={(event) => setPrimaryLinkedId(event.target.value)}
                    placeholder="Primary linked entity (optional)"
                    className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-200">
                    Additional Linked IDs
                  </label>
                  <input
                    value={additionalLinkedIds}
                    onChange={(event) =>
                      setAdditionalLinkedIds(event.target.value)
                    }
                    placeholder="Comma separated list"
                    className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                  />
                </div>
              </div>

              {entityFormError && (
                <div className="rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                  {entityFormError}
                </div>
              )}

              <button
                type="submit"
                disabled={createEntityMutation.isPending}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {createEntityMutation.isPending
                  ? "Creating..."
                  : "Create Entity"}
              </button>
            </form>
          </div>

          <div className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">
                Rollback Entity Version
              </h2>
              {rollbackMessage && (
                <span
                  className={`text-sm ${rollbackMessage.startsWith("Entity") ? "text-emerald-400" : "text-amber-300"}`}
                >
                  {rollbackMessage}
                </span>
              )}
            </div>
            <form
              className="mt-4 grid gap-4 md:grid-cols-2"
              onSubmit={handleRollbackEntity}
            >
              <div className="md:col-span-1">
                <label className="block text-sm font-medium text-slate-200">
                  Entity ID
                </label>
                <input
                  value={rollbackEntityId}
                  onChange={(event) => setRollbackEntityId(event.target.value)}
                  placeholder="UUID of the entity"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>
              <div className="md:col-span-1">
                <label className="block text-sm font-medium text-slate-200">
                  Target Version
                </label>
                <input
                  value={rollbackTargetVersion}
                  onChange={(event) =>
                    setRollbackTargetVersion(event.target.value)
                  }
                  placeholder="e.g. 1"
                  inputMode="numeric"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-slate-200">
                  Reason (optional)
                </label>
                <input
                  value={rollbackReason}
                  onChange={(event) => setRollbackReason(event.target.value)}
                  placeholder="Captured in audit trail"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>
              <div className="md:col-span-2 flex items-center justify-end">
                <button
                  type="submit"
                  disabled={rollbackEntityMutation.isPending}
                  className="inline-flex items-center justify-center rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-amber-500 focus:outline-none focus:ring-2 focus:ring-amber-300 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  {rollbackEntityMutation.isPending
                    ? "Rolling back..."
                    : "Rollback Entity"}
                </button>
              </div>
            </form>
          </div>
        </section>

        <section className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-white">
                Query Entities by Type
              </h2>
              <p className="mt-1 text-sm text-slate-300">
                Fetch entities for the selected organization and type. Linked
                entities resolve automatically.
              </p>
            </div>
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Entity Type
                </label>
                <input
                  value={queryEntityType}
                  onChange={(event) => setQueryEntityType(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40 md:w-56"
                />
              </div>
              <div className="inline-flex items-center rounded-lg bg-slate-900/80 p-1 ring-1 ring-slate-700/60">
                <button
                  type="button"
                  onClick={() => setResultView("cards")}
                  className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    resultView === "cards"
                      ? "bg-cyan-600 text-white shadow"
                      : "text-slate-300 hover:bg-slate-800/60"
                  }`}
                  aria-pressed={resultView === "cards"}
                >
                  Cards
                </button>
                <button
                  type="button"
                  onClick={() => setResultView("grid")}
                  className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    resultView === "grid"
                      ? "bg-cyan-600 text-white shadow"
                      : "text-slate-300 hover:bg-slate-800/60"
                  }`}
                  aria-pressed={resultView === "grid"}
                >
                  Grid
                </button>
              </div>
              <button
                onClick={handleFetchEntities}
                disabled={isFetchingEntities}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isFetchingEntities ? "Fetching..." : "Fetch Entities"}
              </button>
            </div>
          </div>

          {queryError && (
            <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
              {queryError}
            </div>
          )}

          {entitiesData?.entitiesByType && (
            <>
              <div className="mt-6 text-sm text-slate-300">
                {hasResults
                  ? `${entityRows.length} entities fetched`
                  : "No entities found for that type."}
              </div>

              {hasResults ? (
                resultView === "grid" ? (
                  <div className="mt-6 overflow-auto rounded-xl border border-slate-700/60">
                    <table className="min-w-full divide-y divide-slate-700">
                      <thead className="bg-slate-900/80">
                        {table.getHeaderGroups().map((headerGroup) => (
                          <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => (
                              <th
                                key={header.id}
                                className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-300"
                              >
                                {header.isPlaceholder
                                  ? null
                                  : flexRender(
                                      header.column.columnDef.header,
                                      header.getContext(),
                                    )}
                              </th>
                            ))}
                          </tr>
                        ))}
                      </thead>
                      <tbody className="divide-y divide-slate-800">
                        {table.getRowModel().rows.map((row) => (
                          <tr key={row.id} className="hover:bg-slate-900/60">
                            {row.getVisibleCells().map((cell) => (
                              <td
                                key={cell.id}
                                className="whitespace-pre-wrap px-4 py-3 text-sm text-slate-200"
                              >
                                {flexRender(
                                  cell.column.columnDef.cell,
                                  cell.getContext(),
                                )}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <div className="mt-6 grid gap-4 md:grid-cols-2">
                    {entitiesData.entitiesByType.map((entity) => {
                      const parsedProps = safeParseProperties(
                        entity.properties,
                      );
                      return (
                        <div
                          key={entity.id}
                          className="rounded-xl border border-slate-700/60 bg-slate-900/70 p-4"
                        >
                          <div className="text-sm font-medium text-cyan-300">
                            {entity.entityType}
                          </div>
                          <div className="mt-1 text-xs text-slate-400">
                            ID: {entity.id}
                          </div>
                          <pre className="mt-3 max-h-40 overflow-auto rounded-lg bg-slate-950/60 p-3 text-xs text-slate-200">
                            {JSON.stringify(parsedProps, null, 2)}
                          </pre>
                          <div className="mt-3 text-sm text-slate-300">
                            Linked Entities:
                          </div>
                          {entity.linkedEntities.length ? (
                            <ul className="mt-1 space-y-1 text-xs text-slate-400">
                              {entity.linkedEntities.map((link) => {
                                const linkedProps = safeParseProperties(
                                  link.properties,
                                );
                                const linkedName = extractName(linkedProps);

                                return (
                                  <li key={link.id}>
                                    <div>
                                      <span className="text-slate-200">
                                        {link.entityType}
                                      </span>{" "}
                                      —{" "}
                                      <span className="text-slate-300">
                                        {linkedName ?? link.id}
                                      </span>
                                    </div>
                                  </li>
                                );
                              })}
                            </ul>
                          ) : (
                            <p className="mt-1 text-xs text-slate-500">
                              No linked entities.
                            </p>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )
              ) : null}
            </>
          )}
        </section>
      </main>
    </div>
  );
}
