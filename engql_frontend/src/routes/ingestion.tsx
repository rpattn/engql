import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { graphqlRequest } from "../lib/graphql";

type EntitiesByTypeResponse = {
  entitiesByType: Array<{
    id: string;
    entityType: string;
    schemaId: string;
    version: number;
    properties: string;
  }>;
};

type EntitySchemaByNameResponse = {
  entitySchemaByName: {
    id: string;
    name: string;
    description?: string | null;
    version: string;
    status: string;
    previousVersionId?: string | null;
    fields: Array<{
      name: string;
      type: string;
      required: boolean;
      description?: string | null;
    }>;
  } | null;
};

type IngestionSummary = {
  totalRows: number;
  validRows: number;
  invalidRows: number;
  newFieldsDetected: string[];
  schemaChanges: Array<{
    field?: string;
    existingType?: string;
    detectedType?: string;
    message: string;
  }>;
  schemaCreated: boolean;
};

type ParsedEntityRow = {
  id: string;
  schemaId?: string;
  version?: number;
  [key: string]: unknown;
};

const ENTITIES_BY_TYPE_QUERY = `
  query EntitiesByType($organizationId: String!, $entityType: String!) {
    entitiesByType(organizationId: $organizationId, entityType: $entityType) {
      id
      entityType
      schemaId
      version
      properties
    }
  }
`;

const ENTITY_SCHEMA_QUERY = `
  query EntitySchemaByName($organizationId: String!, $name: String!) {
    entitySchemaByName(organizationId: $organizationId, name: $name) {
      id
      name
      description
      version
      status
      previousVersionId
      fields {
        name
        type
        required
        description
      }
    }
  }
`;

const API_BASE_URL =
  import.meta.env.VITE_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export const Route = createFileRoute("/ingestion")({
  component: IngestionPage,
});

function IngestionPage() {
  const [organizationId, setOrganizationId] = useState("");
  const [schemaName, setSchemaName] = useState("");
  const [description, setDescription] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [summary, setSummary] = useState<IngestionSummary | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const entitiesQuery = useQuery({
    queryKey: ["entities-by-type", organizationId, schemaName],
    enabled: false,
    queryFn: async () => {
      return graphqlRequest<EntitiesByTypeResponse>(ENTITIES_BY_TYPE_QUERY, {
        organizationId: organizationId.trim(),
        entityType: schemaName.trim(),
      });
    },
  });

  const schemaQuery = useQuery({
    queryKey: ["schema-by-name", organizationId, schemaName],
    enabled: false,
    queryFn: async () => {
      return graphqlRequest<EntitySchemaByNameResponse>(ENTITY_SCHEMA_QUERY, {
        organizationId: organizationId.trim(),
        name: schemaName.trim(),
      });
    },
  });

  const ingestionMutation = useMutation({
    mutationFn: async () => {
      if (!file) {
        throw new Error("Please choose a CSV or XLSX file to upload.");
      }
      if (!organizationId.trim() || !schemaName.trim()) {
        throw new Error("Organization ID and schema name are required.");
      }

      const formData = new FormData();
      formData.append("file", file);
      formData.append("organizationId", organizationId.trim());
      formData.append("schemaName", schemaName.trim());
      if (description.trim()) {
        formData.append("description", description.trim());
      }

      const response = await fetch(`${API_BASE_URL}/ingestion`, {
        method: "POST",
        body: formData,
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Ingestion failed.");
      }

      const payload = (await response.json()) as IngestionSummary;
      return payload;
    },
    onSuccess: async (result) => {
      const normalized: IngestionSummary = {
        ...result,
        newFieldsDetected: result.newFieldsDetected ?? [],
        schemaChanges: result.schemaChanges ?? [],
      };
      setSummary(normalized);
      setErrorMessage(null);
      await entitiesQuery.refetch();
    },
    onError: (error) => {
      if (error instanceof Error) {
        setErrorMessage(error.message);
      } else {
        setErrorMessage("Unknown error occurred.");
      }
    },
  });

  const parsedEntities = useMemo<ParsedEntityRow[]>(() => {
    const entities = entitiesQuery.data?.entitiesByType ?? [];
    return entities.map((entity) => {
      const properties = safeParseProperties(entity.properties);
      return {
        id: entity.id,
        schemaId: entity.schemaId,
        version: entity.version,
        ...properties,
      };
    });
  }, [entitiesQuery.data]);

  const schemaFields =
    schemaQuery.data?.entitySchemaByName?.fields?.map((field) => ({
      name: field.name,
      type: field.type,
      required: field.required,
      description: field.description ?? "",
    })) ?? [];

  const activeSchema = schemaQuery.data?.entitySchemaByName;
  const schemaBadgeText = activeSchema
    ? `${activeSchema.name} - v${activeSchema.version} (${activeSchema.status})`
    : schemaName
      ? `${schemaName} (unloaded)`
      : "No schema loaded";
  const entityPreviewLabel = activeSchema
    ? `${activeSchema.name} - v${activeSchema.version}`
    : schemaName
      ? schemaName
      : "No schema selected";

  const propertyKeys = useMemo(() => {
    const keys = new Set<string>();
    parsedEntities.forEach((entity) => {
      Object.keys(entity).forEach((key) => {
        if (!["id", "schemaId", "version"].includes(key)) {
          keys.add(key);
        }
      });
    });
    return Array.from(keys);
  }, [parsedEntities]);

  const hasEntities = parsedEntities.length > 0;

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 text-slate-200">
      <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6 shadow-xl shadow-slate-950/40">
        <header className="mb-6">
          <h1 className="text-2xl font-semibold text-cyan-300">
            Data Ingestion
          </h1>
          <p className="mt-2 text-sm text-slate-400">
            Upload a CSV or Excel file to detect the schema, validate rows, and
            push entities into the metadata store.
          </p>
        </header>

        <div className="grid gap-4 md:grid-cols-2">
          <label className="flex flex-col text-sm">
            <span className="mb-1 text-slate-300">Organization ID</span>
            <input
              type="text"
              value={organizationId}
              onChange={(event) => setOrganizationId(event.target.value)}
              className="rounded-lg border border-slate-700 bg-slate-950/70 px-3 py-2 text-slate-100 outline-none focus:border-cyan-400"
              placeholder="UUID of the organization"
            />
          </label>

          <label className="flex flex-col text-sm">
            <span className="mb-1 text-slate-300">Schema Name</span>
            <input
              type="text"
              value={schemaName}
              onChange={(event) => setSchemaName(event.target.value)}
              className="rounded-lg border border-slate-700 bg-slate-950/70 px-3 py-2 text-slate-100 outline-none focus:border-cyan-400"
              placeholder="Entity type name"
            />
          </label>

          <label className="md:col-span-2 flex flex-col text-sm">
            <span className="mb-1 text-slate-300">Description (optional)</span>
            <input
              type="text"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="rounded-lg border border-slate-700 bg-slate-950/70 px-3 py-2 text-slate-100 outline-none focus:border-cyan-400"
              placeholder="Describe this schema"
            />
          </label>

          <label className="md:col-span-2 flex flex-col text-sm">
            <span className="mb-1 text-slate-300">Upload CSV/XLSX</span>
            <input
              type="file"
              accept=".csv, .xlsx"
              onChange={(event) => {
                const selected = event.target.files?.[0];
                setFile(selected ?? null);
              }}
              className="rounded-lg border border-dashed border-slate-600 bg-slate-950/70 px-3 py-2 text-slate-100 outline-none focus:border-cyan-400"
            />
          </label>
        </div>

        <div className="mt-6 flex flex-wrap items-center gap-3">
          <button
            onClick={() => ingestionMutation.mutate()}
            disabled={ingestionMutation.isPending}
            className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {ingestionMutation.isPending ? "Uploading..." : "Upload & Ingest"}
          </button>

          <button
            onClick={() => entitiesQuery.refetch()}
            disabled={
              !organizationId.trim() ||
              !schemaName.trim() ||
              entitiesQuery.isFetching
            }
            className="inline-flex items-center justify-center rounded-lg border border-cyan-500 px-4 py-2 text-sm font-medium text-cyan-300 hover:bg-cyan-500/10 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {entitiesQuery.isFetching ? "Loading..." : "Refresh Entities"}
          </button>

          <button
            onClick={() => schemaQuery.refetch()}
            disabled={
              !organizationId.trim() ||
              !schemaName.trim() ||
              schemaQuery.isFetching
            }
            className="inline-flex items-center justify-center rounded-lg border border-slate-600 px-4 py-2 text-sm font-medium text-slate-200 hover:bg-slate-700/40 focus:outline-none focus:ring-2 focus:ring-slate-400 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {schemaQuery.isFetching ? "Fetching Schema..." : "Fetch Schema"}
          </button>
        </div>

        {errorMessage && (
          <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/20 px-3 py-2 text-sm text-red-100">
            {errorMessage}
          </div>
        )}

        {summary && (
          <div className="mt-6 grid gap-4 rounded-xl border border-slate-800 bg-slate-950/60 p-4 text-sm">
            <h2 className="text-lg font-semibold text-slate-200">Summary</h2>
            <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
              <SummaryStat label="Total Rows" value={summary.totalRows} />
              <SummaryStat label="Valid Rows" value={summary.validRows} />
              <SummaryStat label="Invalid Rows" value={summary.invalidRows} />
              <SummaryStat
                label="Schema Created"
                value={summary.schemaCreated ? "Yes" : "No"}
              />
              {activeSchema && (
                <SummaryStat
                  label="Active Schema Version"
                  value={`v${activeSchema.version} (${activeSchema.status})`}
                />
              )}
              <SummaryStat
                label="New Fields"
                value={
                  summary.newFieldsDetected.length
                    ? summary.newFieldsDetected.join(", ")
                    : "None"
                }
              />
            </div>

            {!!summary.schemaChanges.length && (
              <div>
                <h3 className="text-sm font-semibold text-cyan-300">
                  Schema Changes
                </h3>
                <ul className="mt-2 space-y-2 text-xs text-slate-300">
                  {summary.schemaChanges.map((change, index) => (
                    <li key={`${change.message}-${index}`}>{change.message}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </section>

      <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6 shadow-xl shadow-slate-950/40">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-cyan-300">
              Schema Definition
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              View fields detected or previously defined for this schema.
            </p>
            {activeSchema && (
              <p className="mt-2 text-xs text-slate-500">
                Current version: v{activeSchema.version} - {activeSchema.status}
                {activeSchema.previousVersionId && (
                  <>
                    {" - prev "}
                    {activeSchema.previousVersionId.slice(0, 8)}...
                  </>
                )}
              </p>
            )}
          </div>
          <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
            {schemaBadgeText}
          </span>
        </header>

        {schemaQuery.isError && (
          <div className="mb-4 rounded-md border border-red-500/60 bg-red-500/20 px-3 py-2 text-sm text-red-100">
            {schemaQuery.error instanceof Error
              ? schemaQuery.error.message
              : "Failed to load schema."}
          </div>
        )}

        {!organizationId.trim() || !schemaName.trim() ? (
          <p className="text-sm text-slate-400">
            Enter an organization ID and schema name, then fetch the schema.
          </p>
        ) : schemaQuery.isFetching ? (
          <p className="text-sm text-slate-400">Loading schema...</p>
        ) : schemaQuery.data?.entitySchemaByName == null ? (
          <p className="text-sm text-slate-400">
            No schema found for these parameters. Upload a file or double-check
            the inputs.
          </p>
        ) : schemaFields.length === 0 ? (
          <p className="text-sm text-slate-400">
            Schema has no fields defined yet.
          </p>
        ) : (
          <div className="overflow-auto rounded-xl border border-slate-800">
            <table className="min-w-full divide-y divide-slate-800 text-sm">
              <thead className="bg-slate-900/80 text-xs uppercase text-slate-300">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">Field</th>
                  <th className="px-4 py-3 text-left font-semibold">Type</th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Required
                  </th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Description
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {schemaFields.map((field) => (
                  <tr key={field.name} className="hover:bg-slate-900/60">
                    <td className="px-4 py-3 text-slate-200">{field.name}</td>
                    <td className="px-4 py-3 text-slate-200">{field.type}</td>
                    <td className="px-4 py-3 text-slate-200">
                      {field.required ? "Yes" : "No"}
                    </td>
                    <td className="px-4 py-3 text-slate-300">
                      {field.description || "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6 shadow-xl shadow-slate-950/40">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-cyan-300">
              Entities Preview
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              Inspect the rows stored under the selected schema.
            </p>
          </div>
          <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
            {entityPreviewLabel}
          </span>
        </header>

        {!organizationId.trim() || !schemaName.trim() ? (
          <p className="text-sm text-slate-400">
            Provide an organization ID and schema name to load entities.
          </p>
        ) : hasEntities ? (
          <div className="overflow-auto rounded-xl border border-slate-800">
            <table className="min-w-full divide-y divide-slate-800 text-sm">
              <thead className="bg-slate-900/80 text-xs uppercase text-slate-300">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">ID</th>
                  <th className="px-4 py-3 text-left font-semibold">Version</th>
                  <th className="px-4 py-3 text-left font-semibold">Schema</th>
                  {propertyKeys.map((key) => (
                    <th key={key} className="px-4 py-3 text-left font-semibold">
                      {key}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {parsedEntities.map((entity) => (
                  <tr key={entity.id} className="hover:bg-slate-900/60">
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">
                      {entity.id}
                    </td>
                    <td className="px-4 py-3 text-xs text-slate-300">
                      {entity.version ?? "-"}
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">
                      {typeof entity.schemaId === "string"
                        ? `${entity.schemaId.slice(0, 8)}...`
                        : "-"}
                    </td>
                    {propertyKeys.map((key) => (
                      <td key={key} className="px-4 py-3 text-slate-200">
                        {formatCellValue(entity[key as keyof typeof entity])}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-sm text-slate-400">
            No entities found yet. Upload a file or refresh after ingestion.
          </p>
        )}
      </section>
    </div>
  );
}

function SummaryStat({ label, value }: { label: string; value: unknown }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/70 px-3 py-2 text-slate-200">
      <div className="text-xs uppercase tracking-wide text-slate-500">
        {label}
      </div>
      <div className="mt-1 text-sm font-semibold text-slate-100">
        {renderSummaryValue(value)}
      </div>
    </div>
  );
}

function renderSummaryValue(value: unknown) {
  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return String(value);
  }
  return JSON.stringify(value);
}

function safeParseProperties(raw: string) {
  try {
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === "object") {
      return parsed as Record<string, unknown>;
    }
  } catch (error) {
    console.warn("Failed to parse entity properties", error);
  }
  return {};
}

function formatCellValue(value: unknown) {
  if (value == null) {
    return "";
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}
