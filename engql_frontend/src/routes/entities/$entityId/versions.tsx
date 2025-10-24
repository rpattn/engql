import { useEffect, useMemo, useState } from 'react'
import { Link, createFileRoute } from '@tanstack/react-router'
import { ArrowLeft, GitCompare, Target } from 'lucide-react'
import type { EntityDiffQueryVariables } from '@/generated/graphql'
import {
  useEntityDiffQuery,
  useEntityHistoryQuery,
} from '@/generated/graphql'

export const Route = createFileRoute('/entities/$entityId/versions')({
  component: EntityVersionsPage,
})

function EntityVersionsPage() {
  const { entityId } = Route.useParams()
  const historyQuery = useEntityHistoryQuery({ id: entityId }, { staleTime: 30_000 })
  const snapshots = historyQuery.data?.entityHistory ?? []

  const [targetVersion, setTargetVersion] = useState<number | null>(null)
  const [baseVersion, setBaseVersion] = useState<number | null>(null)

  useEffect(() => {
    if (snapshots.length === 0) {
      setTargetVersion(null)
      setBaseVersion(null)
      return
    }

    setTargetVersion((current) => {
      if (current === null || !snapshots.some((snapshot) => snapshot.version === current)) {
        return snapshots[0].version
      }
      return current
    })

    setBaseVersion((current) => {
      if (
        current !== null &&
        current !== snapshots[0].version &&
        snapshots.some((snapshot) => snapshot.version === current)
      ) {
        return current
      }
      const fallback = snapshots.find((snapshot) => snapshot.version !== snapshots[0].version)
      return fallback ? fallback.version : null
    })
  }, [snapshots])

  const targetSnapshot = useMemo(
    () => snapshots.find((snapshot) => snapshot.version === targetVersion) ?? null,
    [snapshots, targetVersion],
  )

  const baseSnapshot = useMemo(
    () => snapshots.find((snapshot) => snapshot.version === baseVersion) ?? null,
    [snapshots, baseVersion],
  )

  const diffReady =
    baseVersion !== null && targetVersion !== null && baseVersion !== targetVersion

  const diffVariables: EntityDiffQueryVariables = diffReady
    ? { id: entityId, baseVersion: baseVersion!, targetVersion: targetVersion! }
    : { id: entityId, baseVersion: baseVersion ?? 0, targetVersion: targetVersion ?? 0 }

  const diffQuery = useEntityDiffQuery(diffVariables, {
    enabled: diffReady,
    staleTime: 30_000,
  })

  const diffResult = diffQuery.data?.entityDiff ?? null
  const diffText = diffResult?.unifiedDiff ?? null

  const selectedTarget = diffResult?.target ?? targetSnapshot
  const selectedBase = diffResult?.base ?? baseSnapshot

  const renderCanonical = (lines: string[] | null | undefined) => {
    if (!lines || lines.length === 0) {
      return <p className="text-sm text-gray-500">No canonical text available.</p>
    }
    return (
      <pre className="max-h-72 overflow-auto rounded-md bg-gray-900 px-4 py-3 text-xs text-gray-100">
        {lines.join('\n')}
      </pre>
    )
  }

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6 px-4 py-8">
      <Link
        to="/entities"
        className="flex w-fit items-center gap-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700 shadow-sm transition hover:border-gray-300 hover:bg-gray-50"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to entities
      </Link>

      <div className="space-y-2">
        <h1 className="text-2xl font-semibold text-gray-900">Entity versions</h1>
        <p className="text-sm text-gray-600">Inspect and compare saved snapshots for entity {entityId}.</p>
      </div>

      <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-sm font-medium text-gray-700">Select versions to compare</p>
            <p className="mt-1 text-xs text-gray-500">
              Choose a target version to inspect and an earlier version to compare against.
            </p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <label className="flex flex-col text-sm text-gray-700">
              Target version
              <select
                className="mt-1 w-48 rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={targetVersion ?? ''}
                onChange={(event) => setTargetVersion(Number(event.target.value))}
                disabled={snapshots.length === 0}
              >
                {snapshots.map((snapshot) => (
                  <option key={`target-${snapshot.version}`} value={snapshot.version}>
                    Version {snapshot.version}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col text-sm text-gray-700">
              Base version
              <select
                className="mt-1 w-48 rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-900 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                value={baseVersion ?? ''}
                onChange={(event) => {
                  const value = Number(event.target.value)
                  setBaseVersion(Number.isNaN(value) ? null : value)
                }}
                disabled={snapshots.length < 2}
              >
                {snapshots
                  .filter((snapshot) => snapshot.version !== targetVersion)
                  .map((snapshot) => (
                    <option key={`base-${snapshot.version}`} value={snapshot.version}>
                      Version {snapshot.version}
                    </option>
                  ))}
                {snapshots.length < 2 && <option value="">No history available</option>}
              </select>
            </label>
          </div>
        </div>
        {!diffReady && snapshots.length >= 2 && (
          <p className="mt-4 rounded-md bg-yellow-50 px-3 py-2 text-sm text-yellow-800">
            Select two different versions to generate a diff.
          </p>
        )}
        {snapshots.length < 2 && (
          <p className="mt-4 rounded-md bg-blue-50 px-3 py-2 text-sm text-blue-800">
            Only one version is available for this entity so a diff cannot be generated yet.
          </p>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <section className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex items-center gap-2">
            <Target className="h-4 w-4 text-blue-500" />
            <h2 className="text-lg font-semibold text-gray-900">Available versions</h2>
          </div>
          <div className="mt-4 space-y-4">
            {historyQuery.isLoading && <p className="text-sm text-gray-500">Loading versions…</p>}
            {historyQuery.isError && (
              <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
                Unable to load entity history. Please try again later.
              </p>
            )}
            {!historyQuery.isLoading && snapshots.length === 0 && !historyQuery.isError && (
              <p className="text-sm text-gray-500">No snapshots have been recorded for this entity yet.</p>
            )}
            {snapshots.map((snapshot) => {
              const isTarget = snapshot.version === targetVersion
              const isBase = snapshot.version === baseVersion
              return (
                <div
                  key={`snapshot-${snapshot.version}`}
                  className={`rounded-md border px-4 py-3 text-sm transition ${
                    isTarget
                      ? 'border-blue-400 bg-blue-50'
                      : isBase
                        ? 'border-amber-300 bg-amber-50'
                        : 'border-gray-200 bg-white'
                  }`}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <div className="font-semibold text-gray-900">Version {snapshot.version}</div>
                      <div className="text-xs text-gray-500">
                        {snapshot.entityType} · {snapshot.path || 'No path'}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        className={`rounded-md border px-2 py-1 text-xs font-medium transition ${
                          isTarget
                            ? 'border-blue-400 bg-blue-500 text-white'
                            : 'border-blue-200 text-blue-600 hover:border-blue-300 hover:bg-blue-50'
                        }`}
                        onClick={() => setTargetVersion(snapshot.version)}
                      >
                        Set target
                      </button>
                      <button
                        type="button"
                        className={`rounded-md border px-2 py-1 text-xs font-medium transition ${
                          isBase
                            ? 'border-amber-400 bg-amber-500 text-white'
                            : 'border-amber-200 text-amber-600 hover:border-amber-300 hover:bg-amber-50'
                        }`}
                        onClick={() => setBaseVersion(snapshot.version)}
                        disabled={snapshots.length < 2}
                      >
                        Set base
                      </button>
                    </div>
                  </div>
                  <details className="mt-3">
                    <summary className="cursor-pointer text-xs font-medium text-gray-600">
                      View canonical text
                    </summary>
                    <div className="mt-2">{renderCanonical(snapshot.canonicalText)}</div>
                  </details>
                </div>
              )
            })}
          </div>
        </section>

        <section className="lg:col-span-2 rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex items-center gap-2">
            <GitCompare className="h-4 w-4 text-purple-500" />
            <h2 className="text-lg font-semibold text-gray-900">Diff preview</h2>
          </div>
          <div className="mt-4 space-y-4">
            {diffQuery.isLoading && (
              <p className="text-sm text-gray-500">Generating diff…</p>
            )}
            {diffQuery.isError && (
              <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
                Unable to generate the diff. Please adjust the selected versions and try again.
              </p>
            )}
            {!diffReady && !diffQuery.isLoading && (
              <p className="text-sm text-gray-500">Select two different versions to see the unified diff.</p>
            )}
            {diffReady && !diffQuery.isLoading && !diffQuery.isError && diffText && (
              <pre className="max-h-80 overflow-auto rounded-md bg-gray-900 px-4 py-3 text-xs text-green-100">
                {diffText}
              </pre>
            )}
            {diffReady && !diffQuery.isLoading && !diffQuery.isError && !diffText && (
              <p className="rounded-md bg-blue-50 px-3 py-2 text-sm text-blue-800">
                No diff output was produced. The selected versions may be identical or missing.
              </p>
            )}
            <div className="grid gap-4 lg:grid-cols-2">
              <div className="rounded-md border border-gray-200 p-4">
                <h3 className="text-sm font-semibold text-gray-800">
                  Target version {selectedTarget ? selectedTarget.version : '—'}
                </h3>
                <div className="mt-2 space-y-1 text-xs text-gray-500">
                  <div>Schema ID: {selectedTarget?.schemaId ?? '—'}</div>
                  <div>Type: {selectedTarget?.entityType ?? '—'}</div>
                  <div>Path: {selectedTarget?.path || '—'}</div>
                </div>
                <div className="mt-3">{renderCanonical(selectedTarget?.canonicalText)}</div>
              </div>
              <div className="rounded-md border border-gray-200 p-4">
                <h3 className="text-sm font-semibold text-gray-800">
                  Base version {selectedBase ? selectedBase.version : '—'}
                </h3>
                <div className="mt-2 space-y-1 text-xs text-gray-500">
                  <div>Schema ID: {selectedBase?.schemaId ?? '—'}</div>
                  <div>Type: {selectedBase?.entityType ?? '—'}</div>
                  <div>Path: {selectedBase?.path || '—'}</div>
                </div>
                <div className="mt-3">{renderCanonical(selectedBase?.canonicalText)}</div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
