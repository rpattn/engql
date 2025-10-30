import { Link, createFileRoute } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'

import {
  SortDirection,
  useEntityTransformationQuery,
  useQueueTransformationExportMutation,
  useTransformationExecutionQuery,
} from '@/generated/graphql'
import TransformationResultsTable, {
  type TransformationResultsSortState,
} from '@/features/transformations/components/TransformationResultsTable'

const DEFAULT_PAGE_SIZE = 25

export const Route = createFileRoute('/transformations/$transformationId/results')({
  component: TransformationResultsRoute,
})

function TransformationResultsRoute() {
  const { transformationId } = Route.useParams()

  const detailQuery = useEntityTransformationQuery({ id: transformationId })

  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [sortState, setSortState] = useState<TransformationResultsSortState | null>(null)
  const [filters, setFilters] = useState<Record<string, string>>({})
  const [columnMetadata, setColumnMetadata] = useState<
    Record<string, { alias: string; field: string }>
  >({})
  const queueTransformationExportMutation = useQueueTransformationExportMutation()
  const [exportFeedback, setExportFeedback] = useState<
    { type: 'success' | 'error'; message: string } | null
  >(null)

  useEffect(() => {
    setPage(0)
    setFilters({})
    setSortState(null)
    setExportFeedback(null)
  }, [transformationId])

  const paginationInput = useMemo(
    () => ({ limit: pageSize, offset: page * pageSize }),
    [page, pageSize],
  )

  const filterInputs = useMemo(() => {
    return Object.entries(filters)
      .map(([columnKey, rawValue]) => {
        const column = columnMetadata[columnKey]
        if (!column) {
          return null
        }
        const trimmed = rawValue.trim()
        if (trimmed.length === 0) {
          return null
        }
        return {
          alias: column.alias,
          field: column.field,
          value: trimmed,
        }
      })
      .filter(
        (
          item,
        ): item is {
          alias: string
          field: string
          value: string
        } => Boolean(item),
      )
  }, [filters, columnMetadata])

  const sortInput = useMemo(() => {
    if (!sortState) {
      return undefined
    }
    const column = columnMetadata[sortState.columnKey]
    if (!column) {
      return undefined
    }
    return {
      alias: column.alias,
      field: column.field,
      direction: sortState.direction === 'asc' ? SortDirection.Asc : SortDirection.Desc,
    }
  }, [sortState, columnMetadata])

  const executionQuery = useTransformationExecutionQuery(
    {
      transformationId,
      pagination: paginationInput,
      filters: filterInputs.length > 0 ? filterInputs : undefined,
      sort: sortInput,
    },
    {
      enabled: Boolean(transformationId),
      keepPreviousData: true,
    },
  )

  const connection = executionQuery.data?.transformationExecution
  const columns = connection?.columns ?? []
  const rowsData = connection?.rows ?? []
  const pageInfo = connection?.pageInfo ?? null

  useEffect(() => {
    setColumnMetadata(() => {
      const next: Record<string, { alias: string; field: string }> = {}
      for (const column of columns) {
        next[column.key] = { alias: column.alias, field: column.field }
      }
      return next
    })
  }, [columns])

  const rows = useMemo(() => {
    return rowsData.map((row) => {
      const mapped: Record<string, string | null | undefined> = {}
      for (const value of row.values) {
        mapped[value.columnKey] = value.value
      }
      return mapped
    })
  }, [rowsData])

  const transformation = detailQuery.data?.entityTransformation

  const handleFilterChange = (columnKey: string, value: string | null) => {
    setPage(0)
    setFilters((current) => {
      const next = { ...current }
      const trimmed = value?.trim() ?? ''
      if (trimmed.length > 0) {
        next[columnKey] = trimmed
      } else {
        delete next[columnKey]
      }
      return next
    })
  }

  const handleSortChange = (next: TransformationResultsSortState | null) => {
    setPage(0)
    setSortState(next)
  }

  const handlePageChange = (nextPage: number) => {
    if (nextPage < 0) {
      return
    }
    setPage(nextPage)
  }

  const handlePageSizeChange = (nextSize: number) => {
    setPage(0)
    setPageSize(nextSize)
  }

  const handleQueueExport = async () => {
    if (!transformation?.organizationId) {
      setExportFeedback({
        type: 'error',
        message: 'Transformation organization is unavailable for export.',
      })
      return
    }

    const exportFilters = Object.entries(filters)
      .map(([columnKey, rawValue]) => {
        const column = columnMetadata[columnKey]
        if (!column) {
          return null
        }
        const trimmed = rawValue.trim()
        if (trimmed.length === 0) {
          return null
        }
        return { key: columnKey, value: trimmed }
      })
      .filter((item): item is { key: string; value: string } => Boolean(item))

    try {
      await queueTransformationExportMutation.mutateAsync({
        input: {
          organizationId: transformation.organizationId,
          transformationId,
          filters: exportFilters.length > 0 ? exportFilters : undefined,
        },
      })
      setExportFeedback({
        type: 'success',
        message: 'Export queued. Monitor progress on the Exports page.',
      })
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to queue export.'
      setExportFeedback({ type: 'error', message })
    }
  }

  const errorMessage = executionQuery.error
    ? (executionQuery.error as Error).message
    : null

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link
            to="/transformations/$transformationId"
            params={{ transformationId }}
            className="text-xs font-semibold text-blue-600 hover:underline"
          >
            ← Back to designer
          </Link>
          <h2 className="mt-2 text-lg font-semibold text-slate-900">
            {transformation?.name ?? 'Transformation results'}
          </h2>
          <p className="text-sm text-slate-500">
            View materialized rows for this transformation with filtering, sorting, and
            pagination.
          </p>
          {transformation?.organizationId ? (
            <p className="mt-1 text-xs text-slate-400">
              Organization: {transformation.organizationId}
            </p>
          ) : null}
        </div>
        <div className="flex items-center gap-2">
          <Link
            to="/transformations/$transformationId"
            params={{ transformationId }}
            className="rounded border border-slate-300 px-3 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100"
          >
            Edit transformation
          </Link>
          <button
            type="button"
            onClick={() => executionQuery.refetch()}
            disabled={executionQuery.isFetching}
            className="rounded border border-slate-300 px-3 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-40"
          >
            {executionQuery.isFetching ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>
      </div>

      {errorMessage ? (
        <div className="rounded border border-rose-300 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          {errorMessage}
        </div>
      ) : null}

      {exportFeedback ? (
        <div
          className={`rounded border px-4 py-3 text-sm ${
            exportFeedback.type === 'success'
              ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
              : 'border-red-200 bg-red-50 text-red-700'
          }`}
        >
          <p>{exportFeedback.message}</p>
          {exportFeedback.type === 'success' ? (
            <p className="mt-1 text-xs">
              Track progress on the{' '}
              <Link to="/exports" className="font-semibold text-blue-600 underline">
                Exports page
              </Link>
              .
            </p>
          ) : null}
        </div>
      ) : null}

      <TransformationResultsTable
        columns={columns}
        rows={rows}
        filters={filters}
        onFilterChange={handleFilterChange}
        sortState={sortState}
        onSortChange={handleSortChange}
        page={page}
        pageSize={pageSize}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
        pageInfo={pageInfo}
        isLoading={executionQuery.isLoading}
        isFetching={executionQuery.isFetching}
        onRefresh={() => executionQuery.refetch()}
        onExportResults={
          transformation?.organizationId ? handleQueueExport : undefined
        }
        isExporting={queueTransformationExportMutation.isPending}
      />
    </div>
  )
}
