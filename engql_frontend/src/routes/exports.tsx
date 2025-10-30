import { createFileRoute, Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'

import {
  useCancelEntityExportJobMutation,
  EntityExportJobStatus,
  type EntityExportJobsQuery,
  useEntityExportJobsQuery,
} from '@/generated/graphql'

const ACTIVE_POLL_INTERVAL = 5_000
const IDLE_POLL_INTERVAL = 15_000

const API_BASE_URL =
  import.meta.env.VITE_API_URL?.replace(/\/$/, '') ?? 'http://localhost:8080'

type EntityExportJob = EntityExportJobsQuery['entityExportJobs'][number]

export const Route = createFileRoute('/exports')({
  component: ExportsPage,
})

function ExportsPage() {
  const [organizationId, setOrganizationId] = useState('')
  const [limit, setLimit] = useState(50)
  const [offset, setOffset] = useState(0)
  const [cancelError, setCancelError] = useState<string | null>(null)

  const trimmedOrgId = organizationId.trim()
  const enabled = trimmedOrgId.length > 0

  const exportsQuery = useEntityExportJobsQuery(
    {
      organizationId: trimmedOrgId,
      limit,
      offset,
    },
    {
      enabled,
      refetchInterval: (query) => {
        const data = query.state.data
        if (!data) {
          return false
        }
        const hasActiveJob = data.entityExportJobs.some((job) =>
          job.status === EntityExportJobStatus.Pending ||
          job.status === EntityExportJobStatus.Running,
        )
        if (!hasActiveJob) {
          return IDLE_POLL_INTERVAL
        }
        return ACTIVE_POLL_INTERVAL
      },
      refetchIntervalInBackground: true,
    },
  )

  const cancelExportJobMutation = useCancelEntityExportJobMutation({
    onSuccess: () => {
      setCancelError(null)
      void exportsQuery.refetch()
    },
    onError: (error) => {
      if (error instanceof Error) {
        setCancelError(error.message)
      } else {
        setCancelError('Unable to cancel export job')
      }
    },
  })

  const jobs = exportsQuery.data?.entityExportJobs ?? []

  const sortedJobs = useMemo(() => {
    return [...jobs].sort((a, b) => {
      const left = new Date(a.enqueuedAt).getTime()
      const right = new Date(b.enqueuedAt).getTime()
      return right - left
    })
  }, [jobs])

  const inProgressJobs = useMemo(
    () =>
      sortedJobs.filter((job) =>
        job.status === EntityExportJobStatus.Pending ||
        job.status === EntityExportJobStatus.Running,
      ),
    [sortedJobs],
  )

  const historicalJobs = useMemo(
    () =>
      sortedJobs.filter((job) =>
        job.status === EntityExportJobStatus.Completed ||
        job.status === EntityExportJobStatus.Failed ||
        job.status === EntityExportJobStatus.Cancelled,
      ),
    [sortedJobs],
  )

  const stats = useMemo(() => {
    let rowsExported = 0
    let rowsRequested = 0
    let bytesWritten = 0
    let pending = 0
    let running = 0
    let completed = 0
    let cancelled = 0
    let failed = 0

    for (const job of jobs) {
      rowsExported += job.rowsExported ?? 0
      rowsRequested += job.rowsRequested ?? 0
      bytesWritten += job.bytesWritten ?? 0
      switch (job.status) {
        case EntityExportJobStatus.Pending: {
          pending += 1
          break
        }
        case EntityExportJobStatus.Running: {
          running += 1
          break
        }
        case EntityExportJobStatus.Completed: {
          completed += 1
          break
        }
        case EntityExportJobStatus.Cancelled: {
          cancelled += 1
          break
        }
        case EntityExportJobStatus.Failed: {
          failed += 1
          break
        }
      }
    }

    return {
      total: jobs.length,
      pending,
      running,
      completed,
      cancelled,
      failed,
      rowsExported,
      rowsRequested,
      bytesWritten,
    }
  }, [jobs])

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100">
      <div className="mx-auto max-w-7xl px-6 py-8">
        <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold text-cyan-300">Exports</h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-400">
              Monitor export jobs, download completed files, and review failures. Jobs in the
              pending or running state refresh automatically.
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Link
              to="/ingestion/batches"
              className="rounded-lg border border-slate-700 px-4 py-2 text-sm font-medium text-slate-200 transition hover:bg-slate-800"
            >
              View ingestion batches
            </Link>
            <button
              type="button"
              onClick={() => exportsQuery.refetch()}
              className="rounded-lg border border-cyan-500 px-4 py-2 text-sm font-medium text-cyan-300 transition hover:bg-cyan-500/10 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={exportsQuery.isFetching || !enabled}
            >
              {exportsQuery.isFetching ? 'Refreshing…' : 'Refresh jobs'}
            </button>
          </div>
        </header>

        <section className="mb-8 rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40">
          <h2 className="text-lg font-semibold text-slate-200">Filter exports</h2>
          <div className="mt-4 grid gap-4 md:grid-cols-4">
            <label className="flex flex-col text-sm text-slate-300">
              <span className="mb-1 font-medium">Organization ID</span>
              <input
                value={organizationId}
                onChange={(event) => {
                  setOrganizationId(event.target.value)
                  setOffset(0)
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
                onChange={(event) => {
                  const parsed = Number(event.target.value)
                  if (!Number.isNaN(parsed) && parsed > 0) {
                    setLimit(parsed)
                  }
                }}
                className="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white outline-none focus:border-cyan-400"
              />
            </label>
            <label className="flex flex-col text-sm text-slate-300">
              <span className="mb-1 font-medium">Offset</span>
              <input
                type="number"
                min={0}
                value={offset}
                onChange={(event) => {
                  const parsed = Number(event.target.value)
                  if (!Number.isNaN(parsed) && parsed >= 0) {
                    setOffset(parsed)
                  }
                }}
                className="rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-white outline-none focus:border-cyan-400"
              />
            </label>
            <div className="flex flex-col justify-end text-xs text-slate-400">
              <p>Exports are scoped to a single organization.</p>
              <p className="mt-1">
                Need different data? Queue exports from the{' '}
                <Link
                  to="/entities"
                  className="font-semibold text-cyan-300 hover:underline"
                >
                  Entities
                </Link>{' '}
                or{' '}
                <Link
                  to="/transformations"
                  className="font-semibold text-cyan-300 hover:underline"
                >
                  Transformations
                </Link>{' '}
                screens.
              </p>
            </div>
          </div>
        </section>

        {exportsQuery.isError ? (
          <p className="mb-6 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {(exportsQuery.error as Error).message}
          </p>
        ) : null}

        {cancelError ? (
          <p className="mb-6 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {cancelError}
          </p>
        ) : null}

        {enabled ? (
          <section className="mb-8 grid gap-4 md:grid-cols-3 lg:grid-cols-6">
            <StatCard label="Total jobs" value={stats.total} />
            <StatCard label="Pending" value={stats.pending} />
            <StatCard label="Running" value={stats.running} />
            <StatCard label="Completed" value={stats.completed} />
            <StatCard label="Cancelled" value={stats.cancelled} />
            <StatCard label="Failed" value={stats.failed} />
            <StatCard label="Rows exported" value={stats.rowsExported} />
          </section>
        ) : (
          <p className="mb-8 text-sm text-slate-500">
            Provide an organization ID to load export activity.
          </p>
        )}

        <section className="mb-10 rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40 text-slate-100">
          <header className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-slate-200">In-progress jobs</h2>
            <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
              {inProgressJobs.length} active
            </span>
          </header>

          {!enabled ? (
            <p className="text-sm text-slate-500">Enter an organization to view jobs.</p>
          ) : exportsQuery.isLoading ? (
            <p className="text-sm text-slate-500">Loading export jobs…</p>
          ) : inProgressJobs.length === 0 ? (
            <p className="text-sm text-slate-500">No pending or running exports.</p>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {inProgressJobs.map((job) => {
                const isCancelling =
                  cancelExportJobMutation.isPending &&
                  cancelExportJobMutation.variables?.id === job.id
                return (
                  <article
                    key={job.id}
                    className="rounded-lg border border-slate-800 bg-slate-900/70 p-4 shadow-md shadow-slate-950/40"
                  >
                    <header className="flex items-center justify-between text-sm text-slate-400">
                      <span className="font-semibold text-cyan-300">{jobDisplayName(job)}</span>
                      <StatusBadge status={job.status} />
                    </header>
                    <dl className="mt-3 space-y-2 text-xs text-slate-400">
                      <div className="flex justify-between">
                        <dt>Job ID</dt>
                        <dd className="font-mono text-slate-300">{job.id.slice(0, 8)}…</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt>Requested rows</dt>
                        <dd>{job.rowsRequested.toLocaleString()}</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt>Rows exported</dt>
                        <dd>{job.rowsExported.toLocaleString()}</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt>Started</dt>
                        <dd>{formatTimestamp(job.startedAt ?? job.enqueuedAt)}</dd>
                      </div>
                      <div className="flex justify-between">
                        <dt>Last update</dt>
                        <dd>{formatTimestamp(job.updatedAt)}</dd>
                      </div>
                    </dl>
                    <div className="mt-4 flex justify-end">
                      <button
                        type="button"
                        onClick={() => {
                          setCancelError(null)
                          cancelExportJobMutation.mutate({ id: job.id })
                        }}
                        className="inline-flex items-center rounded-md border border-red-400 px-3 py-1 text-xs font-medium text-red-200 transition hover:bg-red-500/10 disabled:cursor-not-allowed disabled:opacity-60"
                        disabled={cancelExportJobMutation.isPending}
                      >
                        {isCancelling ? 'Cancelling…' : 'Cancel job'}
                      </button>
                    </div>
                  </article>
                )
              })}
            </div>
          )}
        </section>

        <section className="rounded-xl border border-slate-800 bg-slate-950/60 p-5 shadow-lg shadow-slate-950/40">
          <header className="mb-4 flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-slate-200">Completed, failed, & cancelled jobs</h2>
              <p className="mt-1 text-xs text-slate-500">
                Download completed files or review errors and cancellation reasons.
              </p>
            </div>
            <span className="rounded-full border border-slate-700 bg-slate-900 px-3 py-1 text-xs text-slate-400">
              {historicalJobs.length} jobs
            </span>
          </header>

          {!enabled ? (
            <p className="text-sm text-slate-500">Enter an organization to view job history.</p>
          ) : exportsQuery.isLoading ? (
            <p className="text-sm text-slate-500">Loading job history…</p>
          ) : historicalJobs.length === 0 ? (
            <p className="text-sm text-slate-500">No completed or failed exports yet.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-800 text-sm text-slate-100">
                <thead className="bg-slate-900/80 text-xs uppercase text-slate-400">
                  <tr>
                    <th className="px-4 py-3 text-left font-semibold">Type</th>
                    <th className="px-4 py-3 text-left font-semibold">Target</th>
                    <th className="px-4 py-3 text-left font-semibold">Status</th>
                    <th className="px-4 py-3 text-left font-semibold">Rows</th>
                    <th className="px-4 py-3 text-left font-semibold">File size</th>
                    <th className="px-4 py-3 text-left font-semibold">Enqueued</th>
                    <th className="px-4 py-3 text-left font-semibold">Completed</th>
                    <th className="px-4 py-3 text-left font-semibold">Actions</th>
                    <th className="px-4 py-3 text-left font-semibold">Error</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800">
                  {historicalJobs.map((job) => {
                    const downloadHref = buildDownloadHref(job.downloadUrl)
                    return (
                      <tr key={job.id} className="hover:bg-slate-900/60">
                        <td className="px-4 py-3 capitalize text-slate-200">{job.jobType.toLowerCase()}</td>
                        <td className="px-4 py-3 text-slate-300">{jobDisplayName(job)}</td>
                        <td className="px-4 py-3 text-slate-200">
                          <StatusBadge status={job.status} />
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          {job.rowsExported.toLocaleString()} / {job.rowsRequested.toLocaleString()}
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          {job.fileByteSize != null ? formatBytes(job.fileByteSize) : '—'}
                        </td>
                        <td className="px-4 py-3 text-slate-400">{formatTimestamp(job.enqueuedAt)}</td>
                        <td className="px-4 py-3 text-slate-400">{job.completedAt ? formatTimestamp(job.completedAt) : '—'}</td>
                        <td className="px-4 py-3 text-slate-200">
                          {job.status === EntityExportJobStatus.Completed ? (
                            downloadHref ? (
                              <a
                                href={downloadHref}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="inline-flex items-center rounded-md border border-cyan-500 px-3 py-1 text-xs font-medium text-cyan-200 transition hover:bg-cyan-500/10"
                              >
                                Download
                              </a>
                            ) : (
                              <span className="text-xs text-slate-500">Invalid download</span>
                            )
                          ) : (
                            <span className="text-xs text-slate-500">—</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-slate-400">{job.errorMessage ?? '—'}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </main>
  )
}

function jobDisplayName(job: EntityExportJob) {
  if (job.jobType === 'ENTITY_TYPE') {
    return job.entityType ?? 'Entity export'
  }
  if (job.transformationDefinition?.name) {
    return job.transformationDefinition.name
  }
  if (job.transformationId) {
    return `Transformation ${job.transformationId.slice(0, 8)}…`
  }
  return 'Transformation export'
}

function buildDownloadHref(downloadUrl?: string | null) {
  if (!downloadUrl) {
    return null
  }
  try {
    if (/^https?:\/\//i.test(downloadUrl)) {
      return downloadUrl
    }
    const url = new URL(downloadUrl, `${API_BASE_URL}`)
    return url.toString()
  } catch {
    return null
  }
}

function formatTimestamp(input?: string | null) {
  if (!input) {
    return '—'
  }
  const date = new Date(input)
  if (Number.isNaN(date.getTime())) {
    return input
  }
  return date.toLocaleString()
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const exponent = Math.min(units.length - 1, Math.floor(Math.log(size) / Math.log(1024)))
  const value = size / 1024 ** exponent
  return `${value.toFixed(value >= 10 || exponent === 0 ? 0 : 1)} ${units[exponent]}`
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/70 px-4 py-3 text-slate-100 shadow shadow-slate-950/40">
      <div className="text-xs uppercase tracking-wide text-slate-500">{label}</div>
      <div className="mt-2 text-lg font-semibold text-cyan-300">{value.toLocaleString()}</div>
    </div>
  )
}

function StatusBadge({ status }: { status: EntityExportJobStatus | string }) {
  const normalized = status.toString().toUpperCase()
  let colorClass = 'text-slate-300 border-slate-600 bg-slate-900/60'
  if (normalized === 'COMPLETED') {
    colorClass = 'text-emerald-200 border-emerald-500/60 bg-emerald-500/10'
  } else if (normalized === 'FAILED') {
    colorClass = 'text-red-200 border-red-500/60 bg-red-500/10'
  } else if (normalized === 'CANCELLED') {
    colorClass = 'text-purple-200 border-purple-500/60 bg-purple-500/10'
  } else if (normalized === 'PENDING' || normalized === 'RUNNING') {
    colorClass = 'text-amber-200 border-amber-500/60 bg-amber-500/10'
  }

  return (
    <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${colorClass}`}>
      {normalized}
    </span>
  )
}
