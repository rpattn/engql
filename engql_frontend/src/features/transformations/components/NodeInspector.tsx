import { useMemo } from 'react'

import { EntityTransformationNodeType } from '@/generated/graphql'

import type { TransformationCanvasNode, TransformationNodeData } from '../types'
import {
  generateUniqueAlias,
  isAliasDerivedFromEntityType,
  sanitizeAlias,
} from '../utils/alias'
import { formatNodeType } from '../utils/format'

type SchemaFieldOptions = Record<string, string[]>
type SchemaEntityTypeOptions = Record<string, string[]>

type NodeInspectorProps = {
  node: TransformationCanvasNode | null
  onUpdate: (
    nodeId: string,
    updater: (node: TransformationCanvasNode) => TransformationCanvasNode,
  ) => void
  onDelete: (nodeId: string) => void
  allNodes: TransformationCanvasNode[]
  schemaFieldOptions: SchemaFieldOptions
  schemaEntityTypeOptions: SchemaEntityTypeOptions
}

type FilterRow = {
  key: string
  value?: string | null
  exists?: boolean | null
  inArray?: string[] | null
}

export function NodeInspector({
  node,
  onUpdate,
  onDelete,
  allNodes,
  schemaFieldOptions,
  schemaEntityTypeOptions,
}: NodeInspectorProps) {
  if (!node) {
    return (
      <aside className="rounded-md border border-dashed border-slate-300 p-4 text-sm text-slate-500">
        Select a node on the canvas to configure it.
      </aside>
    )
  }

  const { data } = node

  const typeLabel = useMemo(() => formatNodeType(data.type), [data.type])

  const updateData = (updater: (data: TransformationNodeData) => TransformationNodeData) => {
    onUpdate(node.id, (current) => ({
      ...current,
      data: updater(current.data),
    }))
  }

  const updateConfig = (
    updater: (config: TransformationNodeData['config']) => TransformationNodeData['config'],
  ) => {
    updateData((current) => ({
      ...current,
      config: updater(current.config),
    }))
  }

  const getFieldOptions = (alias?: string | null) => {
    if (!alias) {
      return [] as string[]
    }

    const trimmed = alias.trim()
    if (!trimmed.length) {
      return [] as string[]
    }

    return (
      schemaFieldOptions[trimmed] ??
      schemaFieldOptions[sanitizeAlias(trimmed)] ??
      []
    )
  }

  const getEntityTypeOptions = (alias?: string | null) => {
    if (!alias) {
      return [] as string[]
    }

    const trimmed = alias.trim()
    if (!trimmed.length) {
      return [] as string[]
    }

    return (
      schemaEntityTypeOptions[trimmed] ??
      schemaEntityTypeOptions[sanitizeAlias(trimmed)] ??
      []
    )
  }

  const renderFilters = (
    filters: FilterRow[] | undefined,
    onChange: (rows: FilterRow[]) => void,
    alias?: string,
    contextKey = 'filters',
  ) => {
    const rows = filters ?? []

    const setRow = (index: number, row: FilterRow) => {
      const copy = [...rows]
      copy[index] = row
      onChange(copy)
    }

    const removeRow = (index: number) => {
      onChange(rows.filter((_, rowIndex) => rowIndex !== index))
    }

    const propertyOptions = getFieldOptions(alias)
    const datalistId =
      alias && propertyOptions.length
        ? `${contextKey}-properties-${node.id}-${sanitizeAlias(alias) || 'default'}`
        : undefined

    return (
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div key={`${row.key}-${index}`} className="rounded border border-slate-200 p-2">
            <label className="block text-xs font-medium text-slate-600">
              Property key
              <input
                value={row.key}
                onChange={(event) =>
                  setRow(index, {
                    ...row,
                    key: event.target.value,
                  })
                }
                list={datalistId}
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="mt-2 block text-xs font-medium text-slate-600">
              Value
              <input
                value={row.value ?? ''}
                onChange={(event) =>
                  setRow(index, {
                    ...row,
                    value: event.target.value,
                  })
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="mt-2 flex items-center gap-2 text-xs font-medium text-slate-600">
              <input
                type="checkbox"
                checked={Boolean(row.exists)}
                onChange={(event) =>
                  setRow(index, {
                    ...row,
                    exists: event.target.checked,
                  })
                }
              />
              Exists
            </label>
            <label className="mt-2 block text-xs font-medium text-slate-600">
              In array (comma separated)
              <input
                value={row.inArray?.join(', ') ?? ''}
                onChange={(event) =>
                  setRow(index, {
                    ...row,
                    inArray: event.target.value
                      .split(',')
                      .map((value) => value.trim())
                      .filter(Boolean),
                  })
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <button
              type="button"
              onClick={() => removeRow(index)}
              className="mt-2 text-xs font-medium text-rose-500"
            >
              Remove filter
            </button>
          </div>
        ))}
        {datalistId && (
          <datalist id={datalistId}>
            {propertyOptions.map((option) => (
              <option key={option} value={option} />
            ))}
          </datalist>
        )}
        <button
          type="button"
          onClick={() => onChange([...rows, { key: '' }])}
          className="text-xs font-medium text-blue-600"
        >
          Add filter condition
        </button>
      </div>
    )
  }

  return (
    <aside className="flex h-full flex-col rounded-md border border-slate-200 bg-white p-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-700">{typeLabel} node</h3>
          <p className="text-xs text-slate-500">Configure node metadata and behavior.</p>
        </div>
        <button
          type="button"
          onClick={() => onDelete(node.id)}
          className="rounded border border-rose-200 px-3 py-1 text-xs font-semibold text-rose-600 hover:bg-rose-50"
        >
          Delete
        </button>
      </div>

      <div className="mt-4 space-y-4 overflow-y-auto">
        <label className="block text-xs font-medium text-slate-600">
          Display name
          <input
            value={data.name}
            onChange={(event) =>
              updateData((current) => ({
                ...current,
                name: event.target.value,
              }))
            }
            className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
          />
        </label>

        {data.config.load && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Load configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Alias
              <input
                value={data.config.load.alias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    load: {
                      ...config.load!,
                      alias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Entity type
              {(() => {
                const entityTypeOptions = getEntityTypeOptions(
                  data.config.load.alias,
                )
                const entityTypeListId = entityTypeOptions.length
                  ? `load-entity-types-${node.id}`
                  : undefined

                return (
                  <>
                    <input
                      value={data.config.load.entityType}
                      onChange={(event) =>
                        updateConfig((config) => ({
                          ...config,
                          load: {
                            ...config.load!,
                            entityType: event.target.value,
                            alias:
                              isAliasDerivedFromEntityType(
                                config.load!.alias,
                                config.load!.entityType,
                              ) && event.target.value.trim()
                                ? generateUniqueAlias(
                                    event.target.value,
                                    allNodes,
                                    node.id,
                                  )
                                : config.load!.alias,
                          },
                        }))
                      }
                      list={entityTypeListId}
                      className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
                    />
                    {entityTypeListId && (
                      <datalist id={entityTypeListId}>
                        {entityTypeOptions.map((option) => (
                          <option key={option} value={option} />
                        ))}
                      </datalist>
                    )}
                  </>
                )
              })()}
            </label>
            {renderFilters(
              data.config.load.filters,
              (rows) =>
                updateConfig((config) => ({
                  ...config,
                  load: {
                    ...config.load!,
                    filters: rows,
                  },
                })),
              data.config.load.alias,
              'load',
            )}
          </div>
        )}

        {data.config.filter && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Filter configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Alias
              <input
                value={data.config.filter.alias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    filter: {
                      ...config.filter!,
                      alias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            {renderFilters(
              data.config.filter.filters,
              (rows) =>
                updateConfig((config) => ({
                  ...config,
                  filter: {
                    ...config.filter!,
                    filters: rows,
                  },
                })),
              data.config.filter.alias,
              'filter',
            )}
          </div>
        )}

        {data.config.project && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Project configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Alias
              <input
                value={data.config.project.alias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    project: {
                      ...config.project!,
                      alias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Fields (comma separated)
              <input
                value={data.config.project.fields.join(', ')}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    project: {
                      ...config.project!,
                      fields: event.target.value
                        .split(',')
                        .map((field) => field.trim())
                        .filter(Boolean),
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
          </div>
        )}

        {data.config.join && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Join configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Left alias
              <input
                value={data.config.join.leftAlias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    join: {
                      ...config.join!,
                      leftAlias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Right alias
              <input
                value={data.config.join.rightAlias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    join: {
                      ...config.join!,
                      rightAlias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Join field
              {(() => {
                const joinFieldOptions = Array.from(
                  new Set([
                    ...getFieldOptions(data.config.join.leftAlias),
                    ...getFieldOptions(data.config.join.rightAlias),
                  ]),
                )
                const joinFieldListId = joinFieldOptions.length
                  ? `join-field-${node.id}`
                  : undefined

                return (
                  <>
                    <input
                      value={data.config.join.onField}
                      onChange={(event) =>
                        updateConfig((config) => ({
                          ...config,
                          join: {
                            ...config.join!,
                            onField: event.target.value,
                          },
                        }))
                      }
                      list={joinFieldListId}
                      className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
                    />
                    {joinFieldListId && (
                      <datalist id={joinFieldListId}>
                        {joinFieldOptions.map((option) => (
                          <option key={option} value={option} />
                        ))}
                      </datalist>
                    )}
                  </>
                )
              })()}
            </label>
          </div>
        )}

        {data.config.sort && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Sort configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Alias
              <input
                value={data.config.sort.alias}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    sort: {
                      ...config.sort!,
                      alias: event.target.value,
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Field
              {(() => {
                const sortFieldOptions = getFieldOptions(data.config.sort.alias)
                const sortFieldListId = sortFieldOptions.length
                  ? `sort-field-${node.id}-${sanitizeAlias(data.config.sort.alias) || 'default'}`
                  : undefined

                return (
                  <>
                    <input
                      value={data.config.sort.field}
                      onChange={(event) =>
                        updateConfig((config) => ({
                          ...config,
                          sort: {
                            ...config.sort!,
                            field: event.target.value,
                          },
                        }))
                      }
                      list={sortFieldListId}
                      className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
                    />
                    {sortFieldListId && (
                      <datalist id={sortFieldListId}>
                        {sortFieldOptions.map((option) => (
                          <option key={option} value={option} />
                        ))}
                      </datalist>
                    )}
                  </>
                )
              })()}
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Direction
              <select
                value={data.config.sort.direction}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    sort: {
                      ...config.sort!,
                      direction: event.target.value as 'ASC' | 'DESC',
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              >
                <option value="ASC">Ascending</option>
                <option value="DESC">Descending</option>
              </select>
            </label>
          </div>
        )}

        {data.config.paginate && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Pagination configuration
            </h4>
            <label className="block text-xs font-medium text-slate-600">
              Limit
              <input
                type="number"
                value={data.config.paginate.limit ?? ''}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    paginate: {
                      ...config.paginate!,
                      limit: event.target.value === '' ? undefined : Number(event.target.value),
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Offset
              <input
                type="number"
                value={data.config.paginate.offset ?? ''}
                onChange={(event) =>
                  updateConfig((config) => ({
                    ...config,
                    paginate: {
                      ...config.paginate!,
                      offset: event.target.value === '' ? undefined : Number(event.target.value),
                    },
                  }))
                }
                className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm"
              />
            </label>
          </div>
        )}

        {![
          EntityTransformationNodeType.Load,
          EntityTransformationNodeType.Filter,
          EntityTransformationNodeType.Project,
          EntityTransformationNodeType.Join,
          EntityTransformationNodeType.LeftJoin,
          EntityTransformationNodeType.AntiJoin,
          EntityTransformationNodeType.Sort,
          EntityTransformationNodeType.Paginate,
        ].includes(data.type) && (
          <p className="text-xs text-slate-500">
            This node does not expose additional configuration.
          </p>
        )}

        {data.validationMessage && (
          <p className="rounded border border-amber-300 bg-amber-50 p-2 text-xs text-amber-800">
            {data.validationMessage}
          </p>
        )}
      </div>
    </aside>
  )
}
