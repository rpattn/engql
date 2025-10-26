import { useEffect, useMemo, useState } from 'react'

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

  const run = () => {
    query.refetch()
  }

  const connection: ExecuteEntityTransformationQuery['executeEntityTransformation'] | undefined =
    query.data?.executeEntityTransformation

  return (
    <div className="space-y-4">
      <form
        className="flex flex-wrap items-end gap-3 rounded border border-slate-200 bg-white px-4 py-3"
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
              className="mt-1 w-28 rounded border border-slate-200 px-2 py-1 text-sm"
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
              className="mt-1 w-28 rounded border border-slate-200 px-2 py-1 text-sm"
            />
          </label>
        </div>
        <button
          type="submit"
          className="rounded bg-blue-600 px-3 py-1 text-xs font-semibold text-white disabled:bg-blue-300"
          disabled={query.isFetching}
        >
          {query.isFetching ? 'Executing…' : 'Run transformation'}
        </button>
      </form>

      {query.error && (
        <p className="rounded border border-rose-300 bg-rose-50 px-3 py-2 text-xs text-rose-700">
          {(query.error as Error).message}
        </p>
      )}

      {connection && (
        <div className="space-y-3">
          <div className="flex items-center justify-between text-xs text-slate-600">
            <span>
              {connection.pageInfo.totalCount} edges • Offset {offset}
            </span>
            <span>
              {connection.pageInfo.hasPreviousPage ? 'Has previous page' : 'No previous page'} ·{' '}
              {connection.pageInfo.hasNextPage ? 'Has next page' : 'No next page'}
            </span>
          </div>
          <div className="grid gap-3">
            {connection.edges.map((edge, index) => (
              <ResultEdgeCard key={`${version}-${index}`} edge={edge} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
