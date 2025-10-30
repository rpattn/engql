import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { EntityTransformationNodeType } from '@/generated/graphql'

import type {
  TransformationCanvasEdge,
  TransformationCanvasNode,
  TransformationNodeData,
} from '../types'
import {
  generateUniqueAlias,
  isAliasDerivedFromEntityType,
  sanitizeAlias,
  getNodeAliases,
  ANY_SOURCE_ALIAS,
} from '../utils/alias'
import { formatNodeType } from '../utils/format'

type SchemaFieldOptions = Record<string, string[]>

type NodeInspectorProps = {
  node: TransformationCanvasNode | null
  onUpdate: (
    nodeId: string,
    updater: (node: TransformationCanvasNode) => TransformationCanvasNode,
  ) => void
  onDelete: (nodeId: string) => void
  allNodes: TransformationCanvasNode[]
  edges: TransformationCanvasEdge[]
  schemaFieldOptions: SchemaFieldOptions
  entityTypeOptions: string[]
}

type FilterRow = {
  key: string
  value?: string | null
  exists?: boolean | null
  inArray?: string[] | null
}

type ProjectFieldsInputProps = {
  nodeId: string
  draftValue: string
  onDraftChange: (value: string) => void
  onCommit: (value: string) => void
  onRemoveLast: () => void
  options: string[]
  alias?: string | null
}

export function NodeInspector({
  node,
  onUpdate,
  onDelete,
  allNodes,
  edges,
  schemaFieldOptions,
  entityTypeOptions,
}: NodeInspectorProps) {
  const [projectFieldDraft, setProjectFieldDraft] = useState('')

  const autoConfiguredSignatureRef = useRef<string | null>(null)

  const nodeId = node?.id ?? null

  const typeLabel = useMemo(
    () => (node ? formatNodeType(node.data.type) : ''),
    [node?.data.type],
  )

  useEffect(() => {
    setProjectFieldDraft('')
  }, [node?.id])

  const updateData = useCallback(
    (updater: (data: TransformationNodeData) => TransformationNodeData) => {
      if (!node) {
        return
      }

      onUpdate(node.id, (current) => ({
        ...current,
        data: updater(current.data),
      }))
    },
    [node, onUpdate],
  )

  const updateConfig = useCallback(
    (
      updater: (
        config: TransformationNodeData['config'],
      ) => TransformationNodeData['config'],
    ) => {
      if (!node) {
        return
      }

      updateData((current) => ({
        ...current,
        config: updater(current.config),
      }))
    },
    [node, updateData],
  )

  const combinedFieldOptions = useMemo(() => {
    const map: Record<string, string[]> = {}

    const addFields = (alias: string | undefined | null, fields: string[]) => {
      const trimmed = alias?.trim()
      if (!trimmed || !fields.length) {
        return
      }

      const normalizedFields = Array.from(
        new Set(fields.map((field) => field.trim()).filter(Boolean)),
      )

      if (!normalizedFields.length) {
        return
      }

      const pushFields = (key: string) => {
        const existing = map[key] ?? []
        map[key] = Array.from(
          new Set([...existing, ...normalizedFields]),
        ).sort((a, b) => a.localeCompare(b))
      }

      pushFields(trimmed)

      const sanitized = sanitizeAlias(trimmed)
      if (sanitized && sanitized !== trimmed) {
        pushFields(sanitized)
      }
    }

    for (const [alias, fields] of Object.entries(schemaFieldOptions)) {
      addFields(alias, fields)
    }

    for (const existingNode of allNodes) {
      const { config } = existingNode.data

      if (config.project?.alias && config.project.fields?.length) {
        addFields(config.project.alias, config.project.fields)
      }

      if (config.load?.alias) {
        addFields(config.load.alias, ['id'])
      }
    }

    return map
  }, [allNodes, schemaFieldOptions])

  const allFieldOptions = useMemo(() => {
    const set = new Set<string>()
    for (const fields of Object.values(combinedFieldOptions)) {
      for (const field of fields) {
        set.add(field)
      }
    }
    return Array.from(set).sort((a, b) => a.localeCompare(b))
  }, [combinedFieldOptions])

  const getFieldOptions = useCallback(
    (alias?: string | null) => {
      if (!alias) {
        return [] as string[]
      }

      if (alias === ANY_SOURCE_ALIAS) {
        return allFieldOptions
      }

      const trimmed = alias.trim()
      if (!trimmed.length) {
        return [] as string[]
      }

      return (
        combinedFieldOptions[trimmed] ??
        combinedFieldOptions[sanitizeAlias(trimmed)] ??
        []
      )
    },
    [allFieldOptions, combinedFieldOptions],
  )

  const availableSourceAliases = useMemo(() => {
    const set = new Set<string>()

    for (const existingNode of allNodes) {
      for (const alias of getNodeAliases(existingNode)) {
        const trimmed = alias.trim()
        if (trimmed) {
          set.add(trimmed)
        }
      }
    }

    const sorted = Array.from(set).sort((a, b) => a.localeCompare(b))
    if (!sorted.length) {
      return sorted
    }

    return [ANY_SOURCE_ALIAS, ...sorted]
  }, [allNodes])

  const concreteSourceAliases = useMemo(
    () => availableSourceAliases.filter((alias) => alias !== ANY_SOURCE_ALIAS),
    [availableSourceAliases],
  )

  const upstreamAliases = useMemo(() => {
    if (!node) {
      return [] as string[]
    }

    const incoming = edges.filter((edge) => edge.target === node.id)
    const set = new Set<string>()

    for (const edge of incoming) {
      const source = allNodes.find((candidate) => candidate.id === edge.source)
      if (!source) {
        continue
      }

      for (const alias of getNodeAliases(source)) {
        const trimmed = alias.trim()
        if (trimmed) {
          set.add(trimmed)
        }
      }
    }

    return Array.from(set)
  }, [allNodes, edges, node])

  const inferFieldsForAlias = useCallback(
    (alias: string | undefined | null) => getFieldOptions(alias),
    [getFieldOptions],
  )

  useEffect(() => {
    if (!node) {
      autoConfiguredSignatureRef.current = null
      return
    }

    const materialize = node.data.config.materialize
    if (!materialize) {
      autoConfiguredSignatureRef.current = null
      return
    }

    const outputs = materialize.outputs ?? []
    const signature = JSON.stringify(outputs)

    if (autoConfiguredSignatureRef.current === signature) {
      return
    }

    const nextOutputs = outputs.map((output) => {
      if (output.fields?.length) {
        return output
      }

      const aliasCandidates: string[] = [
        ...(output.fields
          ?.map((field) => field.sourceAlias)
          .filter((value): value is string => Boolean(value && value.trim())) ?? []),
        ...upstreamAliases,
        ...concreteSourceAliases,
      ]

      let chosenAlias: string | null = null
      let inferredFields: string[] = []

      for (const candidate of aliasCandidates) {
        const trimmedCandidate = candidate.trim()
        const fields = inferFieldsForAlias(trimmedCandidate)
        if (fields.length) {
          chosenAlias = trimmedCandidate
          inferredFields = fields
          break
        }
      }

      if (!chosenAlias || !inferredFields.length) {
        return output
      }

        return {
          ...output,
          fields: inferredFields.map((field) => ({
            sourceAlias: chosenAlias,
            sourceField: field,
            outputField: field,
          })),
        }
    })

    const nextSignature = JSON.stringify(nextOutputs)
    autoConfiguredSignatureRef.current = nextSignature

    if (nextSignature === signature) {
      return
    }

    onUpdate(node.id, (current) => {
      const currentMaterialize = current.data.config.materialize
      if (!currentMaterialize) {
        return current
      }

      return {
        ...current,
        data: {
          ...current.data,
          config: {
            ...current.data.config,
            materialize: {
              ...currentMaterialize,
              outputs: nextOutputs,
            },
          },
        },
      }
    })
  }, [
    concreteSourceAliases,
    inferFieldsForAlias,
    node,
    onUpdate,
    upstreamAliases,
  ])

  const renderFilters = (
    filters: FilterRow[] | undefined,
    onChange: (rows: FilterRow[]) => void,
    alias?: string,
    contextKey = 'filters',
  ) => {
    const rows = filters ?? []
    const activeNodeId = nodeId ?? 'detached'

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
        ? `${contextKey}-properties-${activeNodeId}-${sanitizeAlias(alias) || 'default'}`
        : undefined

    return (
      <div className="space-y-2">
        {rows.map((row, index) => (
          <div
            key={`${contextKey}-${activeNodeId}-${index}`}
            className="rounded-md border border-subtle bg-surface p-3 shadow-sm"
          >
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            </label>
            <label className="mt-2 flex items-center gap-2 text-xs font-medium text-slate-600">
              <input
                type="checkbox"
                className="h-4 w-4 rounded border-subtle text-blue-500 focus:ring-blue-500"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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

  if (!node) {
    return (
      <aside className="rounded-xl border border-dashed border-subtle/70 bg-subtle p-4 text-sm text-muted">
        Select a node on the canvas to configure it.
      </aside>
    )
  }

  const { data } = node


  return (
    <aside className="flex h-full flex-col rounded-xl border border-subtle bg-surface p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-700">{typeLabel} node</h3>
          <p className="text-xs text-slate-500">Configure node metadata and behavior.</p>
        </div>
        <button
          type="button"
          onClick={() => onDelete(node.id)}
          className="rounded-md border border-rose-500/40 px-3 py-1 text-xs font-semibold text-rose-500 transition hover:bg-rose-500/10"
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
            className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            </label>
            <label className="block text-xs font-medium text-slate-600">
              Entity type
              {(() => {
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
                      className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            </label>
            <div className="text-xs font-medium text-slate-600">
              <span>Fields</span>
              <div className="mt-1 flex min-h-[2.5rem] flex-wrap gap-1 rounded-lg border border-subtle bg-subtle px-2 py-1.5">
                {data.config.project.fields.map((field, index) => (
                  <span
                    key={`${node.id}-project-field-${field}-${index}`}
                    className="flex items-center gap-1 rounded-full bg-surface px-2 py-0.5 text-[11px] font-medium text-slate-700 shadow-sm"
                  >
                    {field}
                    <button
                      type="button"
                      onClick={() =>
                        updateConfig((config) => {
                          if (!config.project) {
                            return config
                          }

                          return {
                            ...config,
                            project: {
                              ...config.project,
                              fields: config.project.fields.filter((_, fieldIndex) => fieldIndex !== index),
                            },
                          }
                        })
                      }
                      className="rounded-full bg-subtle px-1 text-[10px] font-semibold text-muted transition hover:bg-surface"
                      aria-label={`Remove ${field}`}
                    >
                      ×
                    </button>
                  </span>
                ))}
                <ProjectFieldsInput
                  key={node.id}
                  nodeId={node.id}
                  draftValue={projectFieldDraft}
                  onDraftChange={setProjectFieldDraft}
                  onCommit={(rawValue) => {
                    const commitValues = rawValue
                      .split(',')
                      .map((field) => field.trim())
                      .filter(Boolean)

                    if (!commitValues.length) {
                      setProjectFieldDraft('')
                      return
                    }

                    updateConfig((config) => {
                      if (!config.project) {
                        return config
                      }

                      const nextFields = commitValues.reduce<string[]>((accumulator, value) => {
                        if (!accumulator.includes(value)) {
                          accumulator.push(value)
                        }
                        return accumulator
                      }, [...config.project.fields])

                      if (nextFields.length === config.project.fields.length) {
                        return config
                      }

                      return {
                        ...config,
                        project: {
                          ...config.project,
                          fields: nextFields,
                        },
                      }
                    })

                    setProjectFieldDraft('')
                  }}
                  onRemoveLast={() => {
                    if (!data.config.project?.fields.length) {
                      return
                    }

                    updateConfig((config) => {
                      if (!config.project || !config.project.fields.length) {
                        return config
                      }

                      return {
                        ...config,
                        project: {
                          ...config.project,
                          fields: config.project.fields.slice(0, -1),
                        },
                      }
                    })
                  }}
                  options={getFieldOptions(data.config.project.alias).filter(
                    (option) => !data.config.project!.fields.includes(option),
                  )}
                  alias={data.config.project.alias}
                />
              </div>
            </div>
          </div>
        )}

        {data.config.materialize && (
          <div className="space-y-3">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Materialize configuration
            </h4>
            <p className="text-xs text-slate-500">
              Define output aliases and the source fields that should be flattened.
            </p>
            <div className="space-y-3">
              {(data.config.materialize.outputs ?? []).map((output, outputIndex) => {
                const fields = output.fields ?? []

                return (
                  <div
                    key={`${node.id}-materialize-output-${outputIndex}`}
                    className="rounded border border-subtle p-3"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <label className="flex-1 text-xs font-medium text-slate-600">
                        Output alias
                        <input
                          value={output.alias}
                          onChange={(event) =>
                            updateConfig((config) => {
                              if (!config.materialize) {
                                return config
                              }

                              const nextOutputs = [...(config.materialize.outputs ?? [])]
                              nextOutputs[outputIndex] = {
                                ...nextOutputs[outputIndex],
                                alias: event.target.value,
                              }

                              return {
                                ...config,
                                materialize: {
                                  ...config.materialize,
                                  outputs: nextOutputs,
                                },
                              }
                            })
                          }
                          className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                        />
                      </label>
                      {(data.config.materialize.outputs?.length ?? 0) > 1 && (
                        <button
                          type="button"
                          onClick={() =>
                            updateConfig((config) => {
                              if (!config.materialize) {
                                return config
                              }

                              return {
                                ...config,
                                materialize: {
                                  ...config.materialize,
                                  outputs: (config.materialize.outputs ?? []).filter(
                                    (_, index) => index !== outputIndex,
                                  ),
                                },
                              }
                            })
                          }
                          className="mt-5 text-xs font-semibold text-rose-500"
                        >
                          Remove
                        </button>
                      )}
                    </div>

                    <div className="mt-3 space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-medium text-slate-600">Field mappings</span>
                        <button
                          type="button"
                          onClick={() => {
                            const candidates: string[] = [
                              ...fields
                                .map((field) => field.sourceAlias)
                                .filter((value): value is string => Boolean(value && value.trim())),
                              ...upstreamAliases,
                              ...concreteSourceAliases,
                            ]

                            for (const candidate of candidates) {
                                const trimmedCandidate = candidate.trim()
                                const inferred = inferFieldsForAlias(trimmedCandidate)
                                if (!inferred.length) {
                                  continue
                                }

                                updateConfig((config) => {
                                if (!config.materialize) {
                                  return config
                                }

                                  const nextOutputs = [...(config.materialize.outputs ?? [])]
                                  nextOutputs[outputIndex] = {
                                    ...nextOutputs[outputIndex],
                                    fields: inferred.map((field) => ({
                                      sourceAlias: trimmedCandidate,
                                      sourceField: field,
                                      outputField: field,
                                    })),
                                  }

                                return {
                                  ...config,
                                  materialize: {
                                    ...config.materialize,
                                    outputs: nextOutputs,
                                  },
                                }
                              })
                              return
                            }
                          }}
                          className="text-xs font-semibold text-blue-600"
                        >
                          Auto-fill fields
                        </button>
                      </div>
                      {fields.length === 0 ? (
                        <p className="text-xs text-slate-500">
                          No fields selected. Use auto-fill or add mappings manually.
                        </p>
                      ) : (
                        <div className="space-y-2">
                          {fields.map((field, fieldIndex) => {
                            const sourceOptions = availableSourceAliases
                            const fieldOptions = getFieldOptions(field.sourceAlias)
                            const fieldListId = fieldOptions.length
                              ? `materialize-field-${node.id}-${outputIndex}-${fieldIndex}`
                              : undefined

                            return (
                              <div
                                key={`${node.id}-materialize-field-${outputIndex}-${fieldIndex}`}
                                className="rounded-md border border-subtle bg-surface p-3 shadow-sm"
                              >
                                <label className="block text-xs font-medium text-slate-600">
                                  Source alias
                                  <select
                                    value={field.sourceAlias ?? ''}
                                    onChange={(event) =>
                                      updateConfig((config) => {
                                        if (!config.materialize) {
                                          return config
                                        }

                                        const nextOutputs = [...(config.materialize.outputs ?? [])]
                                        const nextFields = [...(nextOutputs[outputIndex].fields ?? [])]
                                        nextFields[fieldIndex] = {
                                          ...nextFields[fieldIndex],
                                          sourceAlias: event.target.value,
                                        }
                                        nextOutputs[outputIndex] = {
                                          ...nextOutputs[outputIndex],
                                          fields: nextFields,
                                        }

                                        return {
                                          ...config,
                                          materialize: {
                                            ...config.materialize,
                                            outputs: nextOutputs,
                                          },
                                        }
                                      })
                                    }
                                    className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                                  >
                                    <option value="">Select an alias</option>
                                    {sourceOptions.map((option) => (
                                      <option key={option} value={option}>
                                        {option === ANY_SOURCE_ALIAS
                                          ? 'Any available alias'
                                          : option}
                                      </option>
                                    ))}
                                  </select>
                                </label>
                                <label className="mt-2 block text-xs font-medium text-slate-600">
                                  Source field
                                  <input
                                    value={field.sourceField}
                                    onChange={(event) =>
                                      updateConfig((config) => {
                                        if (!config.materialize) {
                                          return config
                                        }

                                        const nextOutputs = [...(config.materialize.outputs ?? [])]
                                        const nextFields = [...(nextOutputs[outputIndex].fields ?? [])]
                                        nextFields[fieldIndex] = {
                                          ...nextFields[fieldIndex],
                                          sourceField: event.target.value,
                                        }
                                        nextOutputs[outputIndex] = {
                                          ...nextOutputs[outputIndex],
                                          fields: nextFields,
                                        }

                                        return {
                                          ...config,
                                          materialize: {
                                            ...config.materialize,
                                            outputs: nextOutputs,
                                          },
                                        }
                                      })
                                    }
                                    list={fieldListId}
                                    className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                                  />
                                  {fieldListId && (
                                    <datalist id={fieldListId}>
                                      {fieldOptions.map((option) => (
                                        <option key={option} value={option} />
                                      ))}
                                    </datalist>
                                  )}
                                </label>
                                <label className="mt-2 block text-xs font-medium text-slate-600">
                                  Output field name
                                  <input
                                    value={field.outputField}
                                    onChange={(event) =>
                                      updateConfig((config) => {
                                        if (!config.materialize) {
                                          return config
                                        }

                                        const nextOutputs = [...(config.materialize.outputs ?? [])]
                                        const nextFields = [...(nextOutputs[outputIndex].fields ?? [])]
                                        nextFields[fieldIndex] = {
                                          ...nextFields[fieldIndex],
                                          outputField: event.target.value,
                                        }
                                        nextOutputs[outputIndex] = {
                                          ...nextOutputs[outputIndex],
                                          fields: nextFields,
                                        }

                                        return {
                                          ...config,
                                          materialize: {
                                            ...config.materialize,
                                            outputs: nextOutputs,
                                          },
                                        }
                                      })
                                    }
                                    className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                                  />
                                </label>
                                <button
                                  type="button"
                                  onClick={() =>
                                    updateConfig((config) => {
                                      if (!config.materialize) {
                                        return config
                                      }

                                      const nextOutputs = [...(config.materialize.outputs ?? [])]
                                      const nextFields = [...(nextOutputs[outputIndex].fields ?? [])]
                                      nextOutputs[outputIndex] = {
                                        ...nextOutputs[outputIndex],
                                        fields: nextFields.filter((_, index) => index !== fieldIndex),
                                      }

                                      return {
                                        ...config,
                                        materialize: {
                                          ...config.materialize,
                                          outputs: nextOutputs,
                                        },
                                      }
                                    })
                                  }
                                  className="mt-3 text-xs font-semibold text-rose-500"
                                >
                                  Remove field
                                </button>
                              </div>
                            )
                          })}
                        </div>
                      )}
                      <button
                        type="button"
                        onClick={() =>
                          updateConfig((config) => {
                            if (!config.materialize) {
                              return config
                            }

                            const nextOutputs = [...(config.materialize.outputs ?? [])]
                            const nextFields = [
                              ...(nextOutputs[outputIndex].fields ?? []),
                              {
                                sourceAlias: '',
                                sourceField: '',
                                outputField: '',
                              },
                            ]
                            nextOutputs[outputIndex] = {
                              ...nextOutputs[outputIndex],
                              fields: nextFields,
                            }

                            return {
                              ...config,
                              materialize: {
                                ...config.materialize,
                                outputs: nextOutputs,
                              },
                            }
                          })
                        }
                        className="text-xs font-semibold text-blue-600"
                      >
                        Add field mapping
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
            <button
              type="button"
              onClick={() =>
                updateConfig((config) => {
                  const existingOutputs = config.materialize?.outputs ?? []

                  return {
                    ...config,
                    materialize: {
                      ...(config.materialize ?? { outputs: [] }),
                      outputs: [
                        ...existingOutputs,
                        {
                          alias: `result_${existingOutputs.length + 1}`,
                          fields: [],
                        },
                      ],
                    },
                  }
                })
              }
              className="text-xs font-semibold text-blue-600"
            >
              Add output
            </button>
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                      className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                      className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
                className="mt-1 w-full rounded-md border border-subtle bg-surface px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
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
          EntityTransformationNodeType.Materialize,
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

function ProjectFieldsInput({
  nodeId,
  draftValue,
  onDraftChange,
  onCommit,
  onRemoveLast,
  options,
  alias,
}: ProjectFieldsInputProps) {
  const datalistId =
    options.length > 0
      ? `project-fields-${nodeId}-${sanitizeAlias(alias ?? '') || 'default'}`
      : undefined

  return (
    <>
      <input
        value={draftValue}
        onChange={(event) => {
          const { value } = event.target

          if (value.includes(',')) {
            const parts = value.split(',')
            const remainder = parts.pop() ?? ''
            const committed = parts
              .map((part) => part.trim())
              .filter(Boolean)

            if (committed.length) {
              onCommit(committed.join(','))
            }

            onDraftChange(remainder.trimStart())
            return
          }

          onDraftChange(value)
        }}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            if (!draftValue.trim()) {
              event.preventDefault()
              return
            }

            event.preventDefault()
            onCommit(draftValue)
            return
          }

          if (event.key === 'Tab') {
            if (draftValue.trim()) {
              onCommit(draftValue)
            }
            return
          }

          if (event.key === ',') {
            event.preventDefault()
            if (draftValue.trim()) {
              onCommit(draftValue)
            }
            return
          }

          if (event.key === 'Backspace' && !draftValue.length) {
            onRemoveLast()
          }
        }}
        onBlur={() => {
          if (draftValue.trim()) {
            onCommit(draftValue)
          } else {
            onDraftChange('')
          }
        }}
        list={datalistId}
        placeholder="Add a field"
        className="flex-1 min-w-[6rem] border-0 bg-transparent text-sm focus:outline-none focus:ring-0"
      />
      {datalistId && (
        <datalist id={datalistId}>
          {options.map((option) => (
            <option key={`${nodeId}-project-field-option-${option}`} value={option} />
          ))}
        </datalist>
      )}
    </>
  )
}
