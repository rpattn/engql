import { useEffect, useMemo, useRef, useState } from 'react'
import type { Dispatch, MouseEvent as ReactMouseEvent, SetStateAction } from 'react'
import { Link } from '@tanstack/react-router'
import {
  ArrowDown,
  ArrowUp,
  ArrowUpDown,
  Columns,
  Filter,
  History,
  Loader2,
  Pencil,
  Trash2,
} from 'lucide-react'
import type { Entity, FieldDefinition } from '../../../generated/graphql'
import { FieldType } from '../../../generated/graphql'
import ColumnFilterPopover from './ColumnFilterPopover'
import {
  ColumnFilterValue,
  ParsedEntityRow,
  extractEntityDisplayNameFromProperties,
  formatJsonPreviewLimited,
  formatRelative,
  formatTimestamp,
} from './helpers'

const FIELD_COLUMN_PREFIX = 'field:' as const
const MIN_COLUMN_WIDTH = 140
const DEFAULT_FIELD_WIDTH = 220
const BASE_COLUMN_IDS = {
  entity: 'base:entity',
  path: 'base:path',
  version: 'base:version',
  updatedAt: 'base:updatedAt',
  createdAt: 'base:createdAt',
  actions: 'base:actions',
} as const

type BaseColumnId = (typeof BASE_COLUMN_IDS)[keyof typeof BASE_COLUMN_IDS]

const BASE_COLUMN_COUNT = Object.keys(BASE_COLUMN_IDS).length

const BASE_COLUMN_WIDTHS: Record<BaseColumnId, number> = {
  [BASE_COLUMN_IDS.entity]: 180,
  [BASE_COLUMN_IDS.path]: 220,
  [BASE_COLUMN_IDS.version]: 140,
  [BASE_COLUMN_IDS.updatedAt]: 210,
  [BASE_COLUMN_IDS.createdAt]: 210,
  [BASE_COLUMN_IDS.actions]: 220,
}

type SortState = {
  columnId: string
  direction: 'asc' | 'desc'
}

type EntityTableProps = {
  rows: ParsedEntityRow[]
  schemaFields: FieldDefinition[]
  columnFilters: Record<string, ColumnFilterValue>
  onColumnFilterChange: (fieldName: string, value: string | null) => void
  onEdit: (entity: Entity) => void
  onDelete: (entity: Entity) => void
  summaryLabel: string
  isLoading: boolean
  isFetching: boolean
  hiddenFieldNames: string[]
  onHiddenFieldNamesChange: Dispatch<SetStateAction<string[]>>
}

export default function EntityTable({
  rows,
  schemaFields,
  columnFilters,
  onColumnFilterChange,
  onEdit,
  onDelete,
  summaryLabel,
  isLoading,
  isFetching,
  hiddenFieldNames,
  onHiddenFieldNamesChange,
}: EntityTableProps) {
  const [activeFilterField, setActiveFilterField] = useState<string | null>(null)
  const [columnsMenuOpen, setColumnsMenuOpen] = useState(false)
  const [sortState, setSortState] = useState<SortState | null>(null)
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>({})
  const [resizingState, setResizingState] = useState<{
    columnId: string
    startX: number
    startWidth: number
  } | null>(null)
  const columnsMenuRef = useRef<HTMLDivElement | null>(null)

  const isBusy = isLoading || isFetching
  const isEmpty = rows.length === 0
  const showEmptyState = !isBusy && isEmpty
  const totalVisibleColumns = BASE_COLUMN_COUNT + visibleFields.length

  const sortedFields = useMemo(
    () => schemaFields.slice().sort((a, b) => a.name.localeCompare(b.name)),
    [schemaFields],
  )

  const visibleFields = useMemo(
    () => sortedFields.filter((field) => !hiddenFieldNames.includes(field.name)),
    [hiddenFieldNames, sortedFields],
  )

  const displayedRows = useMemo(() => {
    if (!sortState) {
      return rows
    }

    const { columnId, direction } = sortState
    const multiplier = direction === 'asc' ? 1 : -1

    const normalizeValue = (value: unknown): string | number => {
      if (value === undefined || value === null) {
        return ''
      }
      if (typeof value === 'number') {
        return value
      }
      if (typeof value === 'boolean') {
        return value ? 1 : 0
      }
      if (Array.isArray(value)) {
        return value.map((item) => String(item)).join(', ').toLowerCase()
      }
      if (typeof value === 'object') {
        try {
          return JSON.stringify(value).toLowerCase()
        } catch {
          return ''
        }
      }
      return String(value).toLowerCase()
    }

    const getSortValue = (row: ParsedEntityRow): unknown => {
      switch (columnId) {
        case BASE_COLUMN_IDS.entity:
          return row.entity.entityType
        case BASE_COLUMN_IDS.path:
          return row.entity.path
        case BASE_COLUMN_IDS.version:
          return row.entity.version
        case BASE_COLUMN_IDS.updatedAt:
          return row.entity.updatedAt
        case BASE_COLUMN_IDS.createdAt:
          return row.entity.createdAt
        default: {
          if (columnId.startsWith(FIELD_COLUMN_PREFIX)) {
            const fieldName = columnId.slice(FIELD_COLUMN_PREFIX.length)
            return row.props[fieldName]
          }
          return null
        }
      }
    }

    const parseTemporalValue = (value: unknown): number | null => {
      if (typeof value === 'string') {
        const parsed = Date.parse(value)
        return Number.isNaN(parsed) ? null : parsed
      }
      return null
    }

    return rows.slice().sort((a, b) => {
      const aValue = getSortValue(a)
      const bValue = getSortValue(b)

      if ((aValue === undefined || aValue === null) && (bValue === undefined || bValue === null)) {
        return 0
      }
      if (aValue === undefined || aValue === null) {
        return 1 * multiplier
      }
      if (bValue === undefined || bValue === null) {
        return -1 * multiplier
      }

      if (columnId === BASE_COLUMN_IDS.version) {
        const aNumber = typeof aValue === 'number' ? aValue : Number(aValue)
        const bNumber = typeof bValue === 'number' ? bValue : Number(bValue)
        if (!Number.isNaN(aNumber) && !Number.isNaN(bNumber)) {
          return (aNumber - bNumber) * multiplier
        }
      }

      if (columnId === BASE_COLUMN_IDS.updatedAt || columnId === BASE_COLUMN_IDS.createdAt) {
        const aTime = parseTemporalValue(aValue)
        const bTime = parseTemporalValue(bValue)
        if (aTime !== null && bTime !== null) {
          return (aTime - bTime) * multiplier
        }
      }

      const normalizedA = normalizeValue(aValue)
      const normalizedB = normalizeValue(bValue)

      if (typeof normalizedA === 'number' && typeof normalizedB === 'number') {
        return (normalizedA - normalizedB) * multiplier
      }

      return normalizedA.toString().localeCompare(normalizedB.toString()) * multiplier
    })
  }, [rows, sortState])

  useEffect(() => {
    setActiveFilterField((current) =>
      current && hiddenFieldNames.includes(current) ? null : current,
    )
  }, [hiddenFieldNames])

  useEffect(() => {
    if (!sortState) {
      return
    }
    if (sortState.columnId.startsWith(FIELD_COLUMN_PREFIX)) {
      const fieldName = sortState.columnId.slice(FIELD_COLUMN_PREFIX.length)
      if (hiddenFieldNames.includes(fieldName)) {
        setSortState(null)
      }
    }
  }, [hiddenFieldNames, sortState])

  useEffect(() => {
    setColumnWidths((current) => {
      const next: Record<string, number> = { ...current }
      for (const key of Object.values(BASE_COLUMN_IDS) as BaseColumnId[]) {
        if (!next[key]) {
          next[key] = BASE_COLUMN_WIDTHS[key]
        }
      }
      for (const field of schemaFields) {
        const columnId = `${FIELD_COLUMN_PREFIX}${field.name}`
        if (!next[columnId]) {
          next[columnId] = DEFAULT_FIELD_WIDTH
        }
      }
      const validKeys = new Set<string>([
        ...Object.values(BASE_COLUMN_IDS),
        ...schemaFields.map((field) => `${FIELD_COLUMN_PREFIX}${field.name}`),
      ])
      for (const key of Object.keys(next)) {
        if (!validKeys.has(key)) {
          delete next[key]
        }
      }
      return next
    })
  }, [schemaFields])

  useEffect(() => {
    if (!resizingState) {
      return
    }

    const handleMouseMove = (event: MouseEvent) => {
      event.preventDefault()
      const delta = event.clientX - resizingState.startX
      const nextWidth = Math.max(MIN_COLUMN_WIDTH, resizingState.startWidth + delta)
      setColumnWidths((current) => ({
        ...current,
        [resizingState.columnId]: nextWidth,
      }))
    }

    const handleMouseUp = () => {
      setResizingState(null)
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)

    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [resizingState])

  useEffect(() => {
    if (!columnsMenuOpen) {
      return
    }

    const handleClick = (event: MouseEvent) => {
      if (columnsMenuRef.current && !columnsMenuRef.current.contains(event.target as Node)) {
        setColumnsMenuOpen(false)
      }
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setColumnsMenuOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [columnsMenuOpen])

  const toggleSort = (columnId: string) => {
    setSortState((current) => {
      if (!current || current.columnId !== columnId) {
        return { columnId, direction: 'asc' }
      }
      if (current.direction === 'asc') {
        return { columnId, direction: 'desc' }
      }
      return null
    })
  }

  const getSortDirection = (columnId: string): SortState['direction'] | null => {
    if (sortState?.columnId === columnId) {
      return sortState.direction
    }
    return null
  }

  const getColumnStyle = (columnId: string) => {
    const width = columnWidths[columnId]
    if (!width) {
      return { minWidth: `${MIN_COLUMN_WIDTH}px` }
    }
    const clampedWidth = Math.max(width, MIN_COLUMN_WIDTH)
    return {
      width: `${clampedWidth}px`,
      minWidth: `${MIN_COLUMN_WIDTH}px`,
      maxWidth: `${clampedWidth}px`,
    }
  }

  const startResize = (columnId: string, event: ReactMouseEvent) => {
    event.preventDefault()
    event.stopPropagation()
    const currentWidth = columnWidths[columnId] ?? MIN_COLUMN_WIDTH
    setResizingState({
      columnId,
      startX: event.clientX,
      startWidth: currentWidth,
    })
  }

  const handleFieldVisibilityChange = (fieldName: string, visible: boolean) => {
    if (visible) {
      onHiddenFieldNamesChange((current) => current.filter((name) => name !== fieldName))
    } else {
      onHiddenFieldNamesChange((current) =>
        current.includes(fieldName) ? current : [...current, fieldName],
      )
    }
  }

  const renderResizeHandle = (columnId: string) => (
    <span
      aria-hidden="true"
      onMouseDown={(event) => startResize(columnId, event)}
      className="absolute right-0 top-0 h-full w-2 cursor-col-resize select-none"
    />
  )

  const renderSortButton = (columnId: string, label: string) => {
    const direction = getSortDirection(columnId)
    const isActive = Boolean(direction)
    return (
      <button
        type="button"
        aria-label={`Sort by ${label}`}
        onClick={() => toggleSort(columnId)}
        className={`rounded p-1 transition ${
          isActive ? 'text-blue-600' : 'text-gray-400 opacity-0 group-hover:opacity-100'
        } hover:text-blue-600`}
      >
        {direction === 'asc' ? (
          <ArrowUp className="h-4 w-4" />
        ) : direction === 'desc' ? (
          <ArrowDown className="h-4 w-4" />
        ) : (
          <ArrowUpDown className="h-4 w-4" />
        )}
      </button>
    )
  }

  const renderColumnsMenu = () => (
    <div ref={columnsMenuRef} className="relative">
      <button
        type="button"
        onClick={() => setColumnsMenuOpen((open) => !open)}
        className="flex items-center gap-2 rounded-md border border-gray-300 px-2 py-1.5 text-xs font-medium text-gray-600 transition hover:border-gray-400 hover:text-gray-900"
      >
        <Columns className="h-3.5 w-3.5" />
        Columns
      </button>
      {columnsMenuOpen && (
        <div className="absolute right-0 z-20 mt-2 w-72 rounded-md border border-gray-200 bg-white p-3 text-sm shadow-lg">
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Schema fields
          </p>
          {sortedFields.length === 0 ? (
            <p className="text-xs text-gray-500">No schema fields available.</p>
          ) : (
            <div className="max-h-64 space-y-2 overflow-auto pr-1">
              {sortedFields.map((field) => {
                const visible = !hiddenFieldNames.includes(field.name)
                return (
                  <label
                    key={field.name}
                    className="flex items-center justify-between gap-3 rounded px-1 py-1 hover:bg-gray-50"
                  >
                    <span>
                      <span className="block text-sm font-medium text-gray-800">{field.name}</span>
                      <span className="block text-[10px] uppercase tracking-wide text-gray-400">
                        {field.type}
                      </span>
                    </span>
                    <input
                      type="checkbox"
                      className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      checked={visible}
                      onChange={(event) =>
                        handleFieldVisibilityChange(field.name, event.target.checked)
                      }
                    />
                  </label>
                )
              })}
            </div>
          )}
          <button
            type="button"
            className="mt-3 w-full rounded-md border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-600 transition hover:border-gray-300 hover:text-gray-900 disabled:cursor-not-allowed disabled:text-gray-400"
            onClick={() => onHiddenFieldNamesChange([])}
            disabled={hiddenFieldNames.length === 0}
          >
            Show all columns
          </button>
        </div>
      )}
    </div>
  )

  const headerLabel = isBusy && isEmpty ? 'Loading entities…' : summaryLabel

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 text-sm text-gray-600">
        <div>{headerLabel}</div>
        {renderColumnsMenu()}
      </div>

      {showEmptyState ? (
        <div className="px-6 py-10 text-center text-sm text-gray-600">
          No entities found for this schema and filters.
        </div>
      ) : (
        <div className="relative">
          <div className="max-h-[480px] overflow-auto">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50">
              <tr>
                <th
                  className="relative px-4 py-3 text-left font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.entity)}
                >
                  <div className="group flex items-center gap-2">
                    <span>Entity</span>
                    {renderSortButton(BASE_COLUMN_IDS.entity, 'Entity')}
                  </div>
                  {renderResizeHandle(BASE_COLUMN_IDS.entity)}
                </th>
                <th
                  className="relative px-4 py-3 text-left font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.path)}
                >
                  <div className="group flex items-center gap-2">
                    <span>Path</span>
                    {renderSortButton(BASE_COLUMN_IDS.path, 'Path')}
                  </div>
                  {renderResizeHandle(BASE_COLUMN_IDS.path)}
                </th>
                <th
                  className="relative px-4 py-3 text-left font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.version)}
                >
                  <div className="group flex items-center gap-2">
                    <span>Version</span>
                    {renderSortButton(BASE_COLUMN_IDS.version, 'Version')}
                  </div>
                  {renderResizeHandle(BASE_COLUMN_IDS.version)}
                </th>
                {visibleFields.map((field) => {
                  const isActive = Boolean(columnFilters[field.name])
                  const isOpen = activeFilterField === field.name
                  const columnId = `${FIELD_COLUMN_PREFIX}${field.name}`
                  return (
                    <th
                      key={field.name}
                      className="relative px-4 py-3 text-left font-semibold text-gray-600"
                      style={getColumnStyle(columnId)}
                    >
                      <div className="group flex items-center justify-between gap-2">
                        <div>
                          <div>{field.name}</div>
                          <div className="text-[10px] uppercase tracking-wide text-gray-400">
                            {field.type}
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          {renderSortButton(columnId, field.name)}
                          <button
                            type="button"
                            aria-label={`Filter ${field.name}`}
                            onClick={() =>
                              setActiveFilterField((current) =>
                                current === field.name ? null : field.name,
                              )
                            }
                            className={`rounded p-1 transition ${
                              isActive
                                ? 'text-blue-600'
                                : 'text-gray-400 opacity-0 group-hover:opacity-100'
                            } hover:text-blue-600`}
                          >
                            <Filter className="h-4 w-4" />
                          </button>
                        </div>
                      </div>
                      {isOpen && (
                        <ColumnFilterPopover
                          field={field}
                          initialValue={columnFilters[field.name]}
                          onApply={(next) => onColumnFilterChange(field.name, next || null)}
                          onClear={() => onColumnFilterChange(field.name, null)}
                          onClose={() => setActiveFilterField(null)}
                        />
                      )}
                      {renderResizeHandle(columnId)}
                    </th>
                  )
                })}
                <th
                  className="relative px-4 py-3 text-left font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.updatedAt)}
                >
                  <div className="group flex items-center gap-2">
                    <span>Updated</span>
                    {renderSortButton(BASE_COLUMN_IDS.updatedAt, 'Updated')}
                  </div>
                  {renderResizeHandle(BASE_COLUMN_IDS.updatedAt)}
                </th>
                <th
                  className="relative px-4 py-3 text-left font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.createdAt)}
                >
                  <div className="group flex items-center gap-2">
                    <span>Created</span>
                    {renderSortButton(BASE_COLUMN_IDS.createdAt, 'Created')}
                  </div>
                  {renderResizeHandle(BASE_COLUMN_IDS.createdAt)}
                </th>
                <th
                  className="relative px-4 py-3 text-right font-semibold text-gray-600"
                  style={getColumnStyle(BASE_COLUMN_IDS.actions)}
                >
                  Actions
                  {renderResizeHandle(BASE_COLUMN_IDS.actions)}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {displayedRows.length === 0 ? (
                <tr>
                  <td
                    colSpan={totalVisibleColumns}
                    className="px-4 py-10 text-center text-sm text-gray-600"
                  >
                    Loading entities…
                  </td>
                </tr>
              ) : (
                displayedRows.map(({ entity, props, linkedById }) => (
                  <tr key={entity.id} className="bg-white align-top">
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.entity)}>
                      <div className="font-medium text-gray-900">
                        <Link
                          to="/entity/$entityId"
                          params={{ entityId: entity.id }}
                          className="text-blue-600 transition hover:text-blue-700 hover:underline"
                        >
                          {entity.entityType}
                        </Link>
                      </div>
                    </td>
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.path)}>
                      {entity.path ? (
                        <span className="text-sm text-gray-700">{entity.path}</span>
                      ) : (
                        <span className="text-xs uppercase tracking-wide text-gray-400">
                          —
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.version)}>
                      <span className="rounded bg-gray-100 px-2 py-1 text-xs font-semibold text-gray-700">
                        v{entity.version}
                      </span>
                    </td>
                    {visibleFields.map((field) => {
                      const columnId = `${FIELD_COLUMN_PREFIX}${field.name}`
                      return (
                        <td
                          key={`${entity.id}-${field.name}`}
                          className="px-4 py-4 align-top"
                          style={getColumnStyle(columnId)}
                        >
                          {renderFieldValue(field, props[field.name], linkedById)}
                        </td>
                      )
                    })}
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.updatedAt)}>
                      <div className="text-sm text-gray-800">
                        {formatTimestamp(entity.updatedAt)}
                      </div>
                      <div className="text-xs text-gray-500">
                        {formatRelative(entity.updatedAt)}
                      </div>
                    </td>
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.createdAt)}>
                      <div className="text-sm text-gray-800">
                        {formatTimestamp(entity.createdAt)}
                      </div>
                      <div className="text-xs text-gray-500">
                        {formatRelative(entity.createdAt)}
                      </div>
                    </td>
                    <td className="px-4 py-4" style={getColumnStyle(BASE_COLUMN_IDS.actions)}>
                      <div className="flex justify-end gap-2">
                        <Link
                          to="/entities/$entityId/versions"
                          params={{ entityId: entity.id }}
                          className="flex items-center gap-1 rounded-md border border-indigo-200 px-3 py-1.5 text-xs font-medium text-indigo-600 transition hover:border-indigo-300 hover:bg-indigo-50"
                        >
                          <History className="h-3.5 w-3.5" />
                          Versions
                        </Link>
                        <button
                          type="button"
                          onClick={() => onEdit(entity)}
                          className="flex items-center gap-1 rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900"
                        >
                          <Pencil className="h-3.5 w-3.5" />
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => onDelete(entity)}
                          className="flex items-center gap-1 rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 transition hover:border-red-300 hover:bg-red-50"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
          </div>
          {isBusy && (
            <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-white/70">
              <Loader2 className="h-6 w-6 animate-spin text-blue-600" />
              <span className="sr-only">Loading entities…</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function renderFieldValue(
  field: FieldDefinition,
  value: unknown,
  linkedById: Map<string, Entity>,
) {
  if (
    value === undefined ||
    value === null ||
    (typeof value === 'string' && value.trim().length === 0)
  ) {
    return (
      <span className="text-xs uppercase tracking-wide text-gray-400">
        —
      </span>
    )
  }

  switch (field.type) {
    case FieldType.Boolean:
      return (
        <span
          className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold ${
            value ? 'bg-emerald-100 text-emerald-700' : 'bg-gray-200 text-gray-600'
          }`}
        >
          {value ? 'True' : 'False'}
        </span>
      )
    case FieldType.Integer:
    case FieldType.Float:
      return <span className="text-sm text-gray-800">{String(value)}</span>
    case FieldType.EntityReference: {
      const id = typeof value === 'string' ? value : ''
      if (!id) {
        return (
          <span className="text-xs uppercase tracking-wide text-gray-400">
            —
          </span>
        )
      }
      const linked = linkedById.get(id)
      const label = linked
        ? extractEntityLabel(linked)
        : id
      return (
        <span className="inline-flex items-center rounded-full bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-700">
          {label}
        </span>
      )
    }
    case FieldType.EntityReferenceArray: {
      const ids = Array.isArray(value)
        ? value.filter((item): item is string => typeof item === 'string' && item.length > 0)
        : typeof value === 'string'
          ? value
            ? [value]
            : []
          : []

      if (ids.length === 0) {
        return (
          <span className="text-xs uppercase tracking-wide text-gray-400">
            —
          </span>
        )
      }

      return (
        <div className="flex flex-wrap gap-1">
          {ids.map((id) => {
            const linked = linkedById.get(id)
            const label = linked ? extractEntityLabel(linked) : id
            return (
              <span
                key={id}
                className="inline-flex items-center rounded-full bg-blue-50 px-2 py-0.5 text-[11px] font-semibold uppercase text-blue-700"
              >
                {label}
              </span>
            )
          })}
        </div>
      )
    }
    case FieldType.Timestamp:
      return (
        <div className="text-xs text-gray-700">
          <div>{formatTimestamp(String(value))}</div>
          <div className="text-[10px] text-gray-500">{formatRelative(String(value))}</div>
        </div>
      )
    case FieldType.Json: {
      const preview = formatJsonPreviewLimited(value)
      return (
        <pre className="max-h-32 overflow-auto rounded bg-gray-50 px-3 py-2 text-xs text-gray-700">
          {preview}
        </pre>
      )
    }
    default: {
      if (Array.isArray(value) || typeof value === 'object') {
        const preview = formatJsonPreviewLimited(value)
        return (
          <pre className="max-h-32 overflow-auto rounded bg-gray-50 px-3 py-2 text-xs text-gray-700">
            {preview}
          </pre>
        )
      }
      return <span className="text-sm text-gray-800">{String(value)}</span>
    }
  }
}

function extractEntityLabel(entity: Entity): string {
  const reference = entity.referenceValue?.trim()
  const displayName = extractEntityDisplayNameFromProperties(
    entity.properties,
    entity.id,
  )

  if (reference) {
    if (displayName && displayName !== reference) {
      return `${reference} • ${displayName}`
    }
    return reference
  }

  return displayName
}
