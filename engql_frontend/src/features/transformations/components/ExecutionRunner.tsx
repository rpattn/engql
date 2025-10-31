import { useCallback, useEffect, useMemo, useState } from 'react'

import {
  ExecuteEntityTransformationQuery,
  useExecuteEntityTransformationQuery,
} from '@/generated/graphql'

import { ResultEdgeCard } from './ResultEdgeCard'

export function ExecutionRunner({ transformationId }: { transformationId: string }) {
  const [limit, setLimit] = useState<number>(25)
  const [offset, setOffset] = useState<number>(0)
  const [version, setVersion] = useState(0)

  const variables = useMemo(
    () => ({
      input: {
        transformationId,
        pagination: {
          limit,
          offset,
        },
      },
    }),
    [transformationId, limit, offset],
  )

  const query = useExecuteEntityTransformationQuery(variables, {
    enabled: false,
  })

  useEffect(() => {
    // refresh when transformation id changes
    setVersion((prev) => prev + 1)
  }, [transformationId])

  const run = useCallback(() => {
    void query.refetch()
  }, [query.refetch])

  useEffect(() => {
    run()
  }, [run, transformationId])

  const connection: ExecuteEntityTransformationQuery['executeEntityTransformation'] | undefined =
    query.data?.executeEntityTransformation

  return (
    <div className="space-y-4">
      <form
        className="flex flex-wrap items-end gap-3 rounded-xl border border-subtle bg-surface px-4 py-3 shadow-sm"
        onSubmit={(event) => {
          event.preventDefault()
          run()
        }}
      >
        <div>
          <label className="block text-xs font-medium text-slate-600">
            Limit
            <input
              type="number"
              min={1}
              value={limit}
              onChange={(event) => setLimit(Number(event.target.value))}
              className="mt-1 w-28 rounded-md border border-subtle bg-surface px-2 py-1 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </label>
        </div>
        <div>
          <label className="block text-xs font-medium text-slate-600">
            Offset
            <input
              type="number"
              min={0}
              value={offset}
              onChange={(event) => setOffset(Number(event.target.value))}
              className="mt-1 w-28 rounded-md border border-subtle bg-surface px-2 py-1 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </label>
        </div>
        <button
          type="submit"
          className="rounded-md bg-blue-600 px-3 py-1 text-xs font-semibold text-white shadow-sm transition hover:bg-blue-500 disabled:bg-blue-300"
          disabled={query.isFetching}
        >
          {query.isFetching ? 'Executing…' : 'Run transformation'}
        </button>
      </form>

      {query.error && (
        <p className="rounded-lg border border-rose-500/40 bg-rose-500/10 px-3 py-2 text-xs text-rose-400">
          {(query.error as Error).message}
        </p>
      )}

      {connection && (
        <div className="space-y-3">
          <div className="flex items-center justify-between text-xs text-muted">
            <span>
              {connection.pageInfo.totalCount} edges • Offset {offset}
            </span>
            <span>
              {connection.pageInfo.hasPreviousPage ? 'Has previous page' : 'No previous page'} ·{' '}
              {connection.pageInfo.hasNextPage ? 'Has next page' : 'No next page'}
            </span>
          </div>
          {connection.edges.length > 0 ? (
            <div className="grid gap-3">
              {connection.edges.map((edge, index) => (
                <ResultEdgeCard key={`${version}-${index}`} edge={edge} />
              ))}
            </div>
          ) : (
            !query.isFetching && (
              <p className="rounded-lg border border-subtle bg-surface px-3 py-2 text-xs text-muted">
                No records were returned for this transformation.
              </p>
            )
          )}
        </div>
      )}
    </div>
  )
}
