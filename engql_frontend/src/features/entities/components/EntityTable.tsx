import { useMemo, useState } from 'react'
import { Filter, Pencil, Trash2 } from 'lucide-react'
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

type EntityTableProps = {
  rows: ParsedEntityRow[]
  schemaFields: FieldDefinition[]
  columnFilters: Record<string, ColumnFilterValue>
  onColumnFilterChange: (fieldName: string, value: string | null) => void
  onEdit: (entity: Entity) => void
  onDelete: (entity: Entity) => void
  summaryLabel: string
  isLoading: boolean
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
}: EntityTableProps) {
  const [activeFilterField, setActiveFilterField] = useState<string | null>(null)

  const isEmpty = rows.length === 0

  const sortedFields = useMemo(
    () => schemaFields.slice().sort((a, b) => a.name.localeCompare(b.name)),
    [schemaFields],
  )

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
      <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 text-sm text-gray-600">
        <div>{isLoading ? 'Loading entities…' : summaryLabel}</div>
      </div>

      {isEmpty ? (
        <div className="px-6 py-10 text-center text-sm text-gray-600">
          {isLoading ? 'Loading entities…' : 'No entities found for this schema and filters.'}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Entity
                </th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Path
                </th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Version
                </th>
                {sortedFields.map((field) => {
                  const isActive = Boolean(columnFilters[field.name])
                  const isOpen = activeFilterField === field.name
                  return (
                    <th key={field.name} className="relative px-4 py-3 text-left font-semibold text-gray-600">
                      <div className="group flex items-center gap-2">
                        <div>
                          <div>{field.name}</div>
                          <div className="text-[10px] uppercase tracking-wide text-gray-400">
                            {field.type}
                          </div>
                        </div>
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
                      {isOpen && (
                        <ColumnFilterPopover
                          field={field}
                          initialValue={columnFilters[field.name]}
                          onApply={(next) => onColumnFilterChange(field.name, next || null)}
                          onClear={() => onColumnFilterChange(field.name, null)}
                          onClose={() => setActiveFilterField(null)}
                        />
                      )}
                    </th>
                  )
                })}
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Updated
                </th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Created
                </th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">
                  Properties
                </th>
                <th className="px-4 py-3 text-right font-semibold text-gray-600">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {rows.map(({ entity, props, linkedById }) => (
                <tr key={entity.id} className="bg-white align-top">
                  <td className="px-4 py-4">
                    <div className="font-medium text-gray-900">
                      {entity.entityType}
                    </div>
                    <div className="mt-1 text-xs text-gray-500">ID: {entity.id}</div>
                    <div className="mt-1 text-xs text-gray-500">
                      Schema ID: {entity.schemaId}
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    {entity.path ? (
                      <span className="text-sm text-gray-700">{entity.path}</span>
                    ) : (
                      <span className="text-xs uppercase tracking-wide text-gray-400">
                        —
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-4">
                    <span className="rounded bg-gray-100 px-2 py-1 text-xs font-semibold text-gray-700">
                      v{entity.version}
                    </span>
                  </td>
                  {sortedFields.map((field) => (
                    <td key={`${entity.id}-${field.name}`} className="px-4 py-4">
                      {renderFieldValue(field, props[field.name], linkedById)}
                    </td>
                  ))}
                  <td className="px-4 py-4">
                    <div className="text-sm text-gray-800">
                      {formatTimestamp(entity.updatedAt)}
                    </div>
                    <div className="text-xs text-gray-500">
                      {formatRelative(entity.updatedAt)}
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <div className="text-sm text-gray-800">
                      {formatTimestamp(entity.createdAt)}
                    </div>
                    <div className="text-xs text-gray-500">
                      {formatRelative(entity.createdAt)}
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <pre className="max-h-32 overflow-auto rounded bg-gray-50 px-3 py-2 text-xs text-gray-700">
                      {formatJsonPreviewLimited(entity.properties)}
                    </pre>
                  </td>
                  <td className="px-4 py-4">
                    <div className="flex justify-end gap-2">
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
              ))}
            </tbody>
          </table>
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
  return extractEntityDisplayNameFromProperties(entity.properties, entity.id)
}
