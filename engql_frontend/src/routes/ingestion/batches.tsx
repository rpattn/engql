import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

const API_BASE_URL =
  import.meta.env.VITE_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

type BatchRecord = {
  id: string;
  organizationId: string;
  schemaId: string;
  entityType: string;
  fileName?: string | null;
  rowsStaged: number;
  rowsFlushed: number;
  skipValidation: boolean;
  status: string;
  errorMessage?: string | null;
  enqueuedAt: string;
  startedAt?: string | null;
  completedAt?: string | null;
  updatedAt: string;
};

type BatchStats = {
  totalBatches: number;
  inProgressBatches: number;
  completedBatches: number;
  failedBatches: number;
  totalRowsStaged: number;
  totalRowsFlushed: number;
};

type BatchOverview = {
  current: BatchRecord[];
  completed: BatchRecord[];
  failed: BatchRecord[];
  stats: BatchStats;
};

type BatchLogEntry = {
  id: string;
  rowNumber?: number | null;
  errorMessage: string;
  createdAt: string;
};

export const Route = createFileRoute("/ingestion/batches")({
  component: IngestionBatchesPage,
});

function IngestionBatchesPage() {
  const [organizationId, setOrganizationId] = useState("");
  const [limit, setLimit] = useState(25);
  const [offset, setOffset] = useState(0);
  const [selectedBatch, setSelectedBatch] = useState<BatchRecord | null>(null);

  useEffect(() => {
    setSelectedBatch(null);
  }, [organizationId]);

  const overviewQuery = useQuery({
    queryKey: ["ingestion-batch-overview", organizationId.trim(), limit, offset],
    enabled: organizationId.trim().length > 0,
    queryFn: async () => {
      const params = new URLSearchParams();
      params.set("organizationId", organizationId.trim());
      params.set("limit", String(limit));
      params.set("offset", String(offset));

      const response = await fetch(
        `${API_BASE_URL}/ingestion/batches?${params.toString()}`,
      );

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to load batch overview");
      }

      const payload = (await response.json()) as BatchOverview;
      return payload;
    },
  });

  const batchLogsQuery = useQuery({
    queryKey: [
      "ingestion-batch-logs",
      selectedBatch?.id,
      organizationId.trim(),
    ],
    enabled:
      Boolean(selectedBatch) &&
      organizationId.trim().length > 0 &&
      Boolean(selectedBatch?.fileName),
    queryFn: async () => {
      if (!selectedBatch) {
        return [] as BatchLogEntry[];
      }
      const params = new URLSearchParams();
      params.set("organizationId", organizationId.trim());
      params.set("schemaName", selectedBatch.entityType);
      if (selectedBatch.fileName) {
        params.set("fileName", selectedBatch.fileName);
      }

      const response = await fetch(
        `${API_BASE_URL}/ingestion/logs?${params.toString()}`,
      );

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || "Failed to load ingestion logs");
      }

      const payload = (await response.json()) as BatchLogEntry[];
      return payload;
    },
  });

  const allHistoricalBatches = useMemo(() => {
    const completed = overviewQuery.data?.completed ?? [];
    const failed = overviewQuery.data?.failed ?? [];
    return [...completed, ...failed].sort((a, b) => {
      return new Date(b.enqueuedAt).getTime() - new Date(a.enqueuedAt).getTime();
    });
  }, [overviewQuery.data]);

  useEffect(() => {
    if (!selectedBatch) {
      return;
    }
    const stillExists = allHistoricalBatches.some(
      (batch) => batch.id === selectedBatch.id,
    );
    if (!stillExists) {
      setSelectedBatch(null);
    }
  }, [allHistoricalBatches, selectedBatch]);

  const currentBatches = overviewQuery.data?.current ?? [];
  const stats = overviewQuery.data?.stats;

  return (
    <main className="bg-app">
      <div className="mx-auto w-full max-w-6xl px-6 py-8">
      <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold text-cyan-300">
            Ingestion Batches
          </h1>
          <p className="mt-2 max-w-2xl text-sm text-slate-400">
            Monitor staged ingestion jobs, track their progress, and inspect
            rows that failed validation.
          </p>
        </div>
        <button
          type="button"
          onClick={() => overviewQuery.refetch()}
          className="rounded-lg border border-cyan-500 px-4 py-2 text-sm font-medium text-cyan-300 transition hover:bg-cyan-500/10 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={overviewQuery.isFetching || organizationId.trim().length === 0}
        >
          {overviewQuery.isFetching ? "Refreshing..." : "Refresh Overview"}
        </button>
      </header>

      <section className="mb-8 rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40">
        <h2 className="text-lg font-semibold text-slate-200">
          Filter Batches
        </h2>
        <div className="mt-4 grid gap-4 md:grid-cols-4">
          <label className="flex flex-col text-sm text-slate-300">
            <span className="mb-1 font-medium">Organization ID</span>
            <input
              value={organizationId}
              onChange={(event) => {
                setOrganizationId(event.target.value);
              }}
              placeholder="UUID"
              className="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white outline-none focus:border-cyan-400"
            />
          </label>
          <label className="flex flex-col text-sm text-slate-300">
            <span className="mb-1 font-medium">Limit</span>
            <input
              type="number"
              min={1}
              value={limit}
              onChange={(event) => setLimit(Number(event.target.value) || 1)}
              className="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white outline-none focus:border-cyan-400"
            />
          </label>
          <label className="flex flex-col text-sm text-slate-300">
            <span className="mb-1 font-medium">Offset</span>
            <input
              type="number"
              min={0}
              value={offset}
              onChange={(event) => setOffset(Math.max(0, Number(event.target.value) || 0))}
              className="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white outline-none focus:border-cyan-400"
            />
          </label>
        </div>
        <p className="mt-3 text-xs text-slate-500">
          Enter an organization ID to load batch data. Use limit and offset to
          page through results.
        </p>
      </section>

      {overviewQuery.isError ? (
        <p className="mb-4 text-sm text-red-400">
          {(overviewQuery.error as Error).message}
        </p>
      ) : null}

      {stats ? (
        <section className="mb-8 grid gap-4 md:grid-cols-3 lg:grid-cols-6">
          <StatCard label="Total Batches" value={stats.totalBatches} />
          <StatCard label="Current" value={stats.inProgressBatches} />
          <StatCard label="Completed" value={stats.completedBatches} />
          <StatCard label="Failed" value={stats.failedBatches} />
          <StatCard label="Rows Staged" value={stats.totalRowsStaged} />
          <StatCard label="Rows Flushed" value={stats.totalRowsFlushed} />
        </section>
      ) : (
        <p className="mb-8 text-sm text-slate-500">
          Stats will appear once an organization is selected.
        </p>
      )}

      <section className="mb-10 rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40 text-slate-100">
        <header className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-200">
            Current Jobs
          </h2>
          <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
            {currentBatches.length} active
          </span>
        </header>

        {organizationId.trim().length === 0 ? (
          <p className="text-sm text-slate-500">
            Provide an organization ID to load current jobs.
          </p>
        ) : overviewQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading current batches…</p>
        ) : currentBatches.length === 0 ? (
          <p className="text-sm text-slate-500">
            No active ingestion batches detected.
          </p>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {currentBatches.map((batch) => (
              <article
                key={batch.id}
                className="rounded-lg border border-slate-800 bg-slate-900/70 p-4 shadow-md shadow-slate-950/40"
              >
                <header className="flex items-center justify-between text-sm text-slate-400">
                  <span className="font-semibold text-cyan-300">
                    {batch.entityType}
                  </span>
                  <StatusBadge status={batch.status} />
                </header>
                <dl className="mt-3 space-y-2 text-xs text-slate-400">
                  <div className="flex justify-between">
                    <dt>Batch ID</dt>
                    <dd className="font-mono text-slate-300">
                      {batch.id.slice(0, 8)}…
                    </dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>Rows staged</dt>
                    <dd>{batch.rowsStaged.toLocaleString()}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>Rows flushed</dt>
                    <dd>{batch.rowsFlushed.toLocaleString()}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>File</dt>
                    <dd>{batch.fileName ?? "—"}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt>Started</dt>
                    <dd>{formatTimestamp(batch.startedAt ?? batch.enqueuedAt)}</dd>
                  </div>
                </dl>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="mb-10 rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40">
        <header className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-200">
              Historical Batches
            </h2>
            <p className="mt-1 text-xs text-slate-500">
              Click a batch to inspect row-level validation errors.
            </p>
          </div>
          <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
            {allHistoricalBatches.length} batches
          </span>
        </header>

        {organizationId.trim().length === 0 ? (
          <p className="text-sm text-slate-500">
            Provide an organization ID to load batches.
          </p>
        ) : overviewQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading batches…</p>
        ) : allHistoricalBatches.length === 0 ? (
          <p className="text-sm text-slate-500">
            No completed or failed batches found.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-800 text-sm text-slate-100">
              <thead className="bg-slate-900/80 text-xs uppercase text-slate-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">Schema</th>
                  <th className="px-4 py-3 text-left font-semibold">File</th>
                  <th className="px-4 py-3 text-left font-semibold">Rows</th>
                  <th className="px-4 py-3 text-left font-semibold">Status</th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Enqueued
                  </th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Completed
                  </th>
                  <th className="px-4 py-3 text-left font-semibold">Error</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {allHistoricalBatches.map((batch) => {
                  const isSelected = selectedBatch?.id === batch.id;
                  return (
                    <tr
                      key={batch.id}
                      onClick={() => setSelectedBatch(batch)}
                      className={`cursor-pointer transition hover:bg-slate-900/60 ${
                        isSelected ? "bg-slate-900/70" : ""
                      }`}
                    >
                      <td className="px-4 py-3 text-slate-200">
                        {batch.entityType}
                      </td>
                      <td className="px-4 py-3 text-slate-400">
                        {batch.fileName ?? "—"}
                      </td>
                      <td className="px-4 py-3 text-slate-200">
                        {batch.rowsFlushed.toLocaleString()} /{" "}
                        {batch.rowsStaged.toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-slate-200">
                        <StatusBadge status={batch.status} />
                      </td>
                      <td className="px-4 py-3 text-slate-400">
                        {formatTimestamp(batch.enqueuedAt)}
                      </td>
                      <td className="px-4 py-3 text-slate-400">
                        {batch.completedAt
                          ? formatTimestamp(batch.completedAt)
                          : "—"}
                      </td>
                      <td className="px-4 py-3 text-slate-400">
                        {batch.errorMessage ?? "—"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40 text-slate-100">
        <header className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-200">
            Failed Rows
          </h2>
          {selectedBatch ? (
            <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
              Batch {selectedBatch.id.slice(0, 8)} •{" "}
              {selectedBatch.fileName ?? "unknown file"}
            </span>
          ) : null}
        </header>

        {!selectedBatch ? (
          <p className="text-sm text-slate-500">
            Select a batch from the table above to view its failed rows.
          </p>
        ) : batchLogsQuery.isLoading ? (
          <p className="text-sm text-slate-500">Loading failed rows…</p>
        ) : batchLogsQuery.isError ? (
          <p className="text-sm text-red-400">
            {(batchLogsQuery.error as Error).message}
          </p>
        ) : batchLogsQuery.data.length === 0 ? (
          <p className="text-sm text-slate-500">
            No failed rows recorded for this batch.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-slate-800 text-sm text-slate-100">
              <thead className="bg-slate-900/80 text-xs uppercase text-slate-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">
                    Row Number
                  </th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Error Message
                  </th>
                  <th className="px-4 py-3 text-left font-semibold">
                    Logged At
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {batchLogsQuery.data.map((entry) => (
                  <tr key={entry.id} className="hover:bg-slate-900/60">
                    <td className="px-4 py-3 text-slate-200">
                      {entry.rowNumber ?? "—"}
                    </td>
                    <td className="px-4 py-3 text-slate-300">
                      {entry.errorMessage}
                    </td>
                    <td className="px-4 py-3 text-slate-400">
                      {formatTimestamp(entry.createdAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      </div>
    </main>
  );
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/70 px-4 py-3 text-slate-100 shadow shadow-slate-950/40">
      <div className="text-xs uppercase tracking-wide text-slate-500">
        {label}
      </div>
      <div className="mt-2 text-lg font-semibold text-cyan-300">
        {value.toLocaleString()}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const normalized = status.toUpperCase();
  let colorClass = "text-slate-300 border-slate-600 bg-slate-900/60";
  if (normalized === "COMPLETED") {
    colorClass = "text-emerald-200 border-emerald-500/60 bg-emerald-500/10";
  } else if (normalized === "FAILED") {
    colorClass = "text-red-200 border-red-500/60 bg-red-500/10";
  } else if (normalized === "FLUSHING" || normalized === "PENDING") {
    colorClass = "text-amber-200 border-amber-500/60 bg-amber-500/10";
  }

  return (
    <span
      className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${colorClass}`}
    >
      {normalized}
    </span>
  );
}

function formatTimestamp(input: string) {
  if (!input) {
    return "—";
  }
  const date = new Date(input);
  if (Number.isNaN(date.getTime())) {
    return input;
  }
  return date.toLocaleString();
}
