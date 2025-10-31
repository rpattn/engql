import { useEffect, useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, ArrowUpDown, Filter, Loader2 } from 'lucide-react'

import ColumnFilterPopover from '@/features/entities/components/ColumnFilterPopover'
import { FieldType, type TransformationExecutionQuery } from '@/generated/graphql'

export type TransformationResultsSortState = {
  columnKey: string
  direction: 'asc' | 'desc'
}

type TransformationResultsTableProps = {
  columns: TransformationExecutionQuery['transformationExecution']['columns']
  rows: Array<Record<string, string | null | undefined>>
  filters: Record<string, string>
  onFilterChange: (columnKey: string, value: string | null) => void
  sortState: TransformationResultsSortState | null
  onSortChange: (next: TransformationResultsSortState | null) => void
  page: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  pageInfo: TransformationExecutionQuery['transformationExecution']['pageInfo'] | null
  isLoading: boolean
  isFetching: boolean
  onRefresh?: () => void
  onExportResults?: () => void
  isExporting?: boolean
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]

export function TransformationResultsTable({
  columns,
  rows,
  filters,
  onFilterChange,
  sortState,
  onSortChange,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageInfo,
  isLoading,
  isFetching,
  onRefresh,
  onExportResults,
  isExporting,
}: TransformationResultsTableProps) {
  const [activeFilterKey, setActiveFilterKey] = useState<string | null>(null)

  useEffect(() => {
    if (!activeFilterKey) {
      return
    }
    if (!columns.some((column) => column.key === activeFilterKey)) {
      setActiveFilterKey(null)
    }
  }, [activeFilterKey, columns])

  const totalCount = pageInfo?.totalCount ?? 0
  const showingStart = totalCount === 0 ? 0 : page * pageSize + 1
  const showingEnd = totalCount === 0 ? 0 : Math.min(totalCount, page * pageSize + rows.length)

  const summary = useMemo(() => {
    if (isLoading) {
      return 'Loading results…'
    }
    if (totalCount === 0) {
      return 'No rows found'
    }
    return `Showing ${showingStart}–${showingEnd} of ${totalCount}`
  }, [isLoading, totalCount, showingStart, showingEnd])

  const handleSortToggle = (columnKey: string) => {
    if (!sortState || sortState.columnKey !== columnKey) {
      onSortChange({ columnKey, direction: 'asc' })
      return
    }
    if (sortState.direction === 'asc') {
      onSortChange({ columnKey, direction: 'desc' })
      return
    }
    onSortChange(null)
  }

  const handleFilterButtonClick = (columnKey: string) => {
    setActiveFilterKey((current) => (current === columnKey ? null : columnKey))
  }

  const handlePageSizeChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const nextSize = Number.parseInt(event.target.value, 10)
    if (!Number.isNaN(nextSize)) {
      onPageSizeChange(nextSize)
    }
  }

  const renderSortIcon = (columnKey: string) => {
    if (sortState?.columnKey !== columnKey) {
      return <ArrowUpDown className="h-3 w-3 text-slate-400" />
    }
    if (sortState.direction === 'asc') {
      return <ArrowUp className="h-3 w-3 text-blue-500" />
    }
    return <ArrowDown className="h-3 w-3 text-blue-500" />
  }

  return (
    <div className="overflow-hidden rounded-xl border border-subtle bg-surface shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-subtle px-4 py-2 text-xs text-muted">
        <div className="flex items-center gap-2">
          {summary}
          {isFetching && !isLoading ? <Loader2 className="h-3 w-3 animate-spin text-blue-500" /> : null}
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2">
            <span>Page size</span>
            <select
              value={pageSize}
              onChange={handlePageSizeChange}
              className="rounded-md border border-subtle bg-surface px-2 py-1 text-xs shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              {PAGE_SIZE_OPTIONS.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => onPageChange(Math.max(0, page - 1))}
              disabled={page === 0 || !pageInfo?.hasPreviousPage}
              className="rounded-md border border-subtle px-2 py-1 text-xs font-medium text-muted transition hover:bg-subtle disabled:opacity-40"
            >
              Previous
            </button>
            <button
              type="button"
              onClick={() => onPageChange(page + 1)}
              disabled={!pageInfo?.hasNextPage}
              className="rounded-md border border-subtle px-2 py-1 text-xs font-medium text-muted transition hover:bg-subtle disabled:opacity-40"
            >
              Next
            </button>
          </div>
          <div className="flex items-center gap-2">
            {onExportResults ? (
              <button
                type="button"
                onClick={onExportResults}
                disabled={isExporting}
                className="rounded-md border border-blue-500/50 px-2 py-1 text-xs font-medium text-blue-500 transition hover:bg-blue-500/10 disabled:cursor-not-allowed disabled:opacity-50"
              >
                {isExporting ? 'Exporting…' : 'Export results'}
              </button>
            ) : null}
            {onRefresh ? (
              <button
                type="button"
                onClick={onRefresh}
                className="rounded-md border border-subtle px-2 py-1 text-xs font-medium text-muted transition hover:bg-subtle"
              >
                Refresh
              </button>
            ) : null}
          </div>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-subtle">
            <tr>
              {columns.map((column) => {
                const hasFilter = Boolean(filters[column.key]?.length)
                const filterField = {
                  __typename: 'FieldDefinition' as const,
                  name: column.label,
                  type: FieldType.String,
                  required: false,
                  description: null,
                  default: null,
                  validation: null,
                  referenceEntityType: null,
                }

                return (
                  <th
                    key={column.key}
                    className="relative px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600"
                  >
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => handleSortToggle(column.key)}
                        className="flex items-center gap-1 text-xs font-semibold text-slate-600 hover:text-blue-600"
                      >
                        <span>{column.label}</span>
                        {renderSortIcon(column.key)}
                      </button>
                      <button
                        type="button"
                        onClick={() => handleFilterButtonClick(column.key)}
                        className={`rounded border px-1.5 py-1 ${
                          hasFilter
                            ? 'border-blue-300 bg-blue-50 text-blue-600'
                            : 'border-slate-300 text-slate-500 hover:bg-subtle'
                        }`}
                        aria-label={`Filter ${column.label}`}
                      >
                        <Filter className="h-3 w-3" />
                      </button>
                      {activeFilterKey === column.key ? (
                        <ColumnFilterPopover
                          field={filterField}
                          initialValue={filters[column.key] ?? ''}
                          onApply={(value) => {
                            onFilterChange(column.key, value)
                          }}
                          onClear={() => {
                            onFilterChange(column.key, null)
                          }}
                          onClose={() => setActiveFilterKey(null)}
                        />
                      ) : null}
                    </div>
                    <div className="mt-1 text-[10px] font-medium uppercase tracking-wide text-slate-400">
                      {column.alias}.{column.field}
                    </div>
                  </th>
                )
              })}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200">
            {isLoading ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-6 text-center text-sm text-slate-500">
                  Loading results…
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-6 text-center text-sm text-slate-500">
                  No rows match the current criteria.
                </td>
              </tr>
            ) : (
              rows.map((row, index) => (
                <tr key={index} className="bg-surface hover:bg-subtle">
                  {columns.map((column) => (
                    <td key={column.key} className="whitespace-pre-wrap px-4 py-2 text-xs text-slate-700">
                      {row[column.key] ?? ''}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default TransformationResultsTable
