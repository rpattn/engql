import { useCallback, useEffect, useMemo, useState } from 'react'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import type { Entity, EntityFilter, PropertyFilter as PropertyFilterInput } from '../generated/graphql'
import { EntitySortField, FieldType, SortDirection } from '../generated/graphql'
import {
  useCreateEntityMutation,
  useDeleteEntityMutation,
  useEntitiesManagementQuery,
  useEntitySchemasQuery,
  useQueueEntityTypeExportMutation,
  useUpdateEntityMutation,
} from '../generated/graphql'
import EntityTable, {
  BASE_COLUMN_IDS,
  FIELD_COLUMN_PREFIX,
  type SortState,
} from '../features/entities/components/EntityTable'
import EntityEditorModal from '../features/entities/components/EntityEditorModal'
import {
  ColumnFilterValue,
  EntityFormState,
  ParsedEntityRow,
  createLinkedEntityMap,
  prepareFieldValueForSubmit,
  safeParseProperties,
} from '../features/entities/components/helpers'
import { OrganizationSelect, useOrganizations } from '@/features/organizations'

type FeedbackState = { type: 'success' | 'error'; message: string; context?: 'export' }

type ModalState =
  | { mode: 'create' }
  | { mode: 'edit'; entity: Entity }

export const Route = createFileRoute('/entities/')({
  component: EntitiesPage,
})

function EntitiesPage() {
  const queryClient = useQueryClient()
  const {
    organizations,
    selectedOrganizationId: selectedOrgId,
    setSelectedOrganizationId: setSelectedOrgId,
    isLoading: organizationsLoading,
  } = useOrganizations()
  const [selectedSchemaId, setSelectedSchemaId] = useState<string | null>(null)
  const [columnFilters, setColumnFilters] = useState<Record<string, ColumnFilterValue>>({})
  const [hiddenFieldNames, setHiddenFieldNames] = useState<string[]>([])
  const [modalState, setModalState] = useState<ModalState | null>(null)
  const [modalError, setModalError] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<FeedbackState | null>(null)
  const [page, setPage] = useState(0)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [sortState, setSortState] = useState<SortState | null>(null)

  const entitySchemasQuery = useEntitySchemasQuery(
    { organizationId: selectedOrgId ?? '' },
    { enabled: Boolean(selectedOrgId) },
  )
  const schemas = entitySchemasQuery.data?.entitySchemas ?? []

  useEffect(() => {
    if (!selectedSchemaId && schemas.length > 0) {
      setSelectedSchemaId(schemas[0].id)
    }
    if (selectedSchemaId && schemas.every((schema) => schema.id !== selectedSchemaId)) {
      setSelectedSchemaId(schemas[0]?.id ?? null)
    }
  }, [schemas, selectedSchemaId])

  useEffect(() => {
    setColumnFilters({})
    setPage(0)
    setHiddenFieldNames([])
  }, [selectedSchemaId, selectedOrgId])

  const selectedSchema = useMemo(
    () => schemas.find((schema) => schema.id === selectedSchemaId) ?? null,
    [schemas, selectedSchemaId],
  )

  useEffect(() => {
    if (!selectedSchema) {
      setHiddenFieldNames([])
      return
    }
    setHiddenFieldNames((current) => {
      const validNames = new Set((selectedSchema.fields ?? []).map((field) => field.name))
      return current.filter((name) => validNames.has(name))
    })
  }, [selectedSchema])

  const visibleSchemaFields = useMemo(() => {
    if (!selectedSchema?.fields) {
      return []
    }
    return selectedSchema.fields.filter((field) => !hiddenFieldNames.includes(field.name))
  }, [hiddenFieldNames, selectedSchema])

  const includeLinkedEntities = useMemo(() => {
    return visibleSchemaFields.some((field) =>
      field.type === FieldType.EntityReference || field.type === FieldType.EntityReferenceArray,
    )
  }, [visibleSchemaFields])

  const propertyFilters = useMemo<PropertyFilterInput[]>(() => {
    if (!selectedSchema?.fields) {
      return []
    }

    return Object.entries(columnFilters)
      .map(([key, raw]) => {
        const trimmed = raw.trim()
        if (trimmed.length === 0) {
          return null
        }

        const schemaField = selectedSchema.fields?.find((field) => field.name === key)
        if (!schemaField) {
          return null
        }

        let normalizedValue = trimmed

        if (schemaField.type === FieldType.Boolean) {
          const normalized = trimmed.toLowerCase()
          if (normalized !== 'true' && normalized !== 'false') {
            return null
          }
          normalizedValue = normalized
        }

        return {
          key,
          value: normalizedValue,
        }
      })
      .filter((filter): filter is PropertyFilterInput => Boolean(filter))
  }, [columnFilters, selectedSchema])

  const entityFilter = useMemo(() => {
    const filter: EntityFilter = {}
    if (selectedSchema?.name) {
      filter.entityType = selectedSchema.name
    }
    if (propertyFilters.length > 0) {
      filter.propertyFilters = propertyFilters
    }
    return Object.keys(filter).length > 0 ? filter : undefined
  }, [selectedSchema, propertyFilters])

  const sortInput = useMemo(() => {
    if (!sortState) {
      return undefined
    }

    const direction =
      sortState.direction === 'asc' ? SortDirection.Asc : SortDirection.Desc

    switch (sortState.columnId) {
      case BASE_COLUMN_IDS.createdAt:
        return { field: EntitySortField.CreatedAt, direction }
      case BASE_COLUMN_IDS.updatedAt:
        return { field: EntitySortField.UpdatedAt, direction }
      case BASE_COLUMN_IDS.entity:
        return { field: EntitySortField.EntityType, direction }
      case BASE_COLUMN_IDS.path:
        return { field: EntitySortField.Path, direction }
      case BASE_COLUMN_IDS.version:
        return { field: EntitySortField.Version, direction }
      default: {
        if (sortState.columnId.startsWith(FIELD_COLUMN_PREFIX)) {
          const propertyKey = sortState.columnId.slice(FIELD_COLUMN_PREFIX.length)
          if (propertyKey) {
            return {
              field: EntitySortField.Property,
              direction,
              propertyKey,
            }
          }
        }
        return undefined
      }
    }
  }, [sortState])

  const queryVariables = useMemo(() => {
    if (!selectedOrgId) {
      return null
    }
    return {
      organizationId: selectedOrgId,
      pagination: {
        limit: pageSize,
        offset: page * pageSize,
      },
      filter: entityFilter,
      includeLinkedEntities,
      sort: sortInput,
    }
  }, [entityFilter, includeLinkedEntities, page, pageSize, selectedOrgId, sortInput])

  const entitiesQuery = useEntitiesManagementQuery(
    queryVariables ?? {
      organizationId: '',
      pagination: { limit: pageSize, offset: 0 },
      sort: sortInput,
    },
    {
      enabled: Boolean(queryVariables),
      keepPreviousData: true,
    },
  )

  const entities = entitiesQuery.data?.entities.entities ?? []

  const tableRows = useMemo<ParsedEntityRow[]>(() => {
    return entities.map((entity) => ({
      entity,
      props: safeParseProperties(entity.properties),
      linkedById: createLinkedEntityMap(entity.linkedEntities ?? []),
    }))
  }, [entities])

  const handleSortChange = useCallback((next: SortState | null) => {
    setSortState(next)
    setPage(0)
  }, [])

  const totalCount = entitiesQuery.data?.entities.pageInfo.totalCount ?? 0
  const totalPages = totalCount > 0 ? Math.ceil(totalCount / pageSize) : 0

  useEffect(() => {
    if (page > 0 && totalPages > 0 && page >= totalPages) {
      setPage(Math.max(totalPages - 1, 0))
    }
  }, [page, totalPages])

  const createEntityMutation = useCreateEntityMutation()
  const updateEntityMutation = useUpdateEntityMutation()
  const deleteEntityMutation = useDeleteEntityMutation()
  const queueEntityTypeExportMutation = useQueueEntityTypeExportMutation()

  const entitiesQueryKey = useMemo(
    () => (queryVariables ? useEntitiesManagementQuery.getKey(queryVariables) : undefined),
    [queryVariables],
  )

  const refetchEntities = useCallback(async () => {
    if (entitiesQueryKey) {
      await queryClient.invalidateQueries({ queryKey: entitiesQueryKey })
    }
  }, [entitiesQueryKey, queryClient])

  const handleModalSubmit = async (formState: EntityFormState) => {
    if (!selectedOrgId || !modalState || !selectedSchema) {
      return
    }

    const schemaFields = selectedSchema.fields ?? []
    const nextProperties: Record<string, unknown> = {
      ...formState.baseProperties,
    }

    for (const field of schemaFields) {
      const rawValue = formState.fieldValues[field.name]
      const result = prepareFieldValueForSubmit(field, rawValue)
      if (!result.ok) {
        setModalError(result.message)
        return
      }
      if (result.value === undefined) {
        delete nextProperties[field.name]
      } else {
        nextProperties[field.name] = result.value
      }
    }

    const normalizedProperties = JSON.stringify(nextProperties)

    setModalError(null)

    try {
      if (modalState.mode === 'create') {
        await createEntityMutation.mutateAsync({
          input: {
            organizationId: selectedOrgId,
            entityType: formState.entityType.trim(),
            path: formState.path.trim() || undefined,
            properties: normalizedProperties,
          },
        })
        setFeedback({ type: 'success', message: 'Entity created successfully.' })
        setPage(0)
      } else {
        await updateEntityMutation.mutateAsync({
          input: {
            id: modalState.entity.id,
            entityType: formState.entityType.trim() || undefined,
            path: formState.path.trim() || undefined,
            properties: normalizedProperties,
          },
        })
        setFeedback({ type: 'success', message: 'Entity updated successfully.' })
      }

      await refetchEntities()
      setModalState(null)
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to save entity changes.'
      setModalError(message)
    }
  }

  const handleDeleteEntity = async (entity: Entity) => {
    const confirmed = window.confirm(
      `Delete entity "${entity.id}" (${entity.entityType})? This cannot be undone.`,
    )
    if (!confirmed) {
      return
    }

    try {
      await deleteEntityMutation.mutateAsync({ id: entity.id })
      setFeedback({ type: 'success', message: 'Entity deleted successfully.' })
      await refetchEntities()
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to delete entity.'
      setFeedback({ type: 'error', message })
    }
  }

  const handleQueueEntityExport = async () => {
    if (!selectedOrgId || !selectedSchema?.name) {
      return
    }

    const filters =
      propertyFilters.length > 0
        ? propertyFilters.map((filter) => ({
            key: filter.key,
            value: filter.value,
            exists: filter.exists,
            inArray: filter.inArray,
          }))
        : undefined

    try {
      await queueEntityTypeExportMutation.mutateAsync({
        input: {
          organizationId: selectedOrgId,
          entityType: selectedSchema.name,
          filters,
        },
      })
      setFeedback({
        type: 'success',
        message: `Export queued for ${selectedSchema.name}.`,
        context: 'export',
      })
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to queue export.'
      setFeedback({ type: 'error', message })
    }
  }

  const handleColumnFilterChange = (fieldName: string, value: string | null) => {
    setColumnFilters((current) => {
      const next = { ...current }
      if (value && value.trim().length > 0) {
        next[fieldName] = value.trim()
      } else {
        delete next[fieldName]
      }
      return next
    })
    setPage(0)
  }

  const summaryLabel = useMemo(() => {
    if (!queryVariables) {
      return 'Select an organization and schema to view entities.'
    }
    const shown = tableRows.length
    const refreshing = entitiesQuery.isFetching && !entitiesQuery.isLoading
    return `${totalCount} total entit${totalCount === 1 ? 'y' : 'ies'} (${shown} shown)${
      refreshing ? ' (refreshing...)' : ''
    }`
  }, [entitiesQuery.isFetching, entitiesQuery.isLoading, queryVariables, tableRows.length, totalCount])

  const activeOrganization =
    selectedOrgId && organizations.find((org) => org.id === selectedOrgId)

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-8">
      <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold">Entities</h1>
            <p className="mt-1 text-sm text-muted">
              Browse, create, and maintain entities for a chosen schema.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setModalState({ mode: 'create' })}
              disabled={!selectedOrgId || !selectedSchema}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-400/60"
            >
              Add Entity
            </button>
            <button
              type="button"
              onClick={handleQueueEntityExport}
              disabled={
                !selectedOrgId ||
                !selectedSchema ||
                queueEntityTypeExportMutation.isPending
              }
              className="rounded-md border border-blue-500/70 px-4 py-2 text-sm font-medium text-blue-600 transition hover:bg-blue-500/10 disabled:cursor-not-allowed disabled:border-blue-500/40 disabled:text-blue-500/60"
            >
              {queueEntityTypeExportMutation.isPending
                ? 'Queuing export…'
                : 'Export all rows'}
            </button>
          </div>
        </div>

        <div className="mt-6 grid gap-4 sm:grid-cols-[2fr,2fr,1fr] sm:items-end">
          <label className="flex flex-col text-sm">
            <span className="text-muted">Organization</span>
            <OrganizationSelect
              value={selectedOrgId}
              onChange={(value) => {
                setSelectedOrgId(value)
                setFeedback(null)
              }}
              className="mt-1"
            />
          </label>

          <label className="flex flex-col text-sm">
            <span className="text-muted">Entity schema</span>
            <select
              value={selectedSchemaId ?? ''}
              onChange={(event) => {
                setSelectedSchemaId(event.target.value || null)
                setFeedback(null)
              }}
              disabled={schemas.length === 0}
              className="mt-1 rounded-md border border-subtle bg-subtle px-3 py-2 text-sm transition focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200 disabled:cursor-not-allowed disabled:bg-subtle/70"
            >
              {schemas.length === 0 && (
                <option value="" disabled>
                  No schemas available
                </option>
              )}
              {schemas.map((schema) => (
                <option key={schema.id} value={schema.id}>
                  {schema.name}
                </option>
              ))}
            </select>
          </label>

          <div className="text-xs text-muted">
            {activeOrganization ? (
              <p>
                Working in{' '}
                <span className="font-semibold text-blue-500">
                  {activeOrganization.name}
                </span>
              </p>
            ) : organizationsLoading ? (
              <p>Loading organizations…</p>
            ) : (
              <p>Select an organization.</p>
            )}
          </div>
        </div>

        {feedback && (
          <div
            className={`mt-4 rounded-md border px-4 py-2 text-sm ${
              feedback.type === 'success'
                ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-500'
                : 'border-red-500/40 bg-red-500/10 text-red-500'
            }`}
          >
            <p>{feedback.message}</p>
            {feedback.context === 'export' && feedback.type === 'success' ? (
              <p className="mt-1 text-xs">
                Track progress on the{' '}
                <Link to="/exports" className="font-semibold text-blue-500 underline">
                  Exports page
                </Link>
                .
              </p>
            ) : null}
          </div>
        )}

        {entitiesQuery.error && (
          <div className="mt-4 rounded-md border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-500">
            {(entitiesQuery.error as Error).message}
          </div>
        )}
      </section>

      <section className="rounded-2xl border border-subtle bg-surface p-6 shadow-sm">
      <EntityTable
        rows={tableRows}
        schemaFields={selectedSchema?.fields ?? []}
        columnFilters={columnFilters}
        onColumnFilterChange={handleColumnFilterChange}
          onEdit={(entity) => setModalState({ mode: 'edit', entity })}
          onDelete={handleDeleteEntity}
          summaryLabel={summaryLabel}
          isLoading={entitiesQuery.isLoading}
          isFetching={entitiesQuery.isFetching}
          hiddenFieldNames={hiddenFieldNames}
          onHiddenFieldNamesChange={setHiddenFieldNames}
          sortState={sortState}
          onSortChange={handleSortChange}
        />

        <div className="mt-6 flex flex-wrap items-center justify-between gap-4">
          <label className="flex items-center gap-2 text-sm text-muted">
            <span>Rows per page:</span>
            <select
              value={pageSize}
              onChange={(event) => {
                const nextSize = Number(event.target.value)
                if (!Number.isNaN(nextSize)) {
                  setPageSize(nextSize)
                  setPage(0)
                }
              }}
              className="rounded-md border border-subtle bg-subtle px-2 py-1 text-sm transition focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            >
              {PAGE_SIZE_OPTIONS.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>

          {totalPages > 0 && (
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setPage((value) => Math.max(value - 1, 0))}
                disabled={page === 0}
                className="rounded-md border border-subtle px-4 py-2 text-sm font-medium text-muted transition hover:border-blue-500/60 hover:text-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                Previous
              </button>
              <div className="text-sm text-muted">
                Page {Math.min(page + 1, totalPages)} of {totalPages}
              </div>
              <button
                type="button"
                onClick={() => {
                  if (page + 1 < totalPages) {
                    setPage((value) => value + 1)
                  }
                }}
                disabled={page + 1 >= totalPages}
                className="rounded-md border border-subtle px-4 py-2 text-sm font-medium text-muted transition hover:border-blue-500/60 hover:text-blue-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                Next
              </button>
            </div>
          )}
        </div>
      </section>

      {modalState && selectedSchema && selectedOrgId && (
        <EntityEditorModal
          isOpen={Boolean(modalState)}
          mode={modalState.mode}
          schema={selectedSchema ?? undefined}
          organizationId={selectedOrgId}
          entity={modalState.mode === 'edit' ? modalState.entity : undefined}
          onClose={() => {
            setModalState(null)
            setModalError(null)
          }}
          onSubmit={handleModalSubmit}
          isSubmitting={createEntityMutation.isPending || updateEntityMutation.isPending}
          errorMessage={modalError}
        />
      )}
    </div>
  )
}

const DEFAULT_PAGE_SIZE = 10
const PAGE_SIZE_OPTIONS = [10, 25, 50, 100]
