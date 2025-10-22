import { useCallback, useEffect, useMemo, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import type { Entity, EntityFilter } from '../generated/graphql'
import {
  useCreateEntityMutation,
  useDeleteEntityMutation,
  useEntitiesManagementQuery,
  useEntitySchemasQuery,
  useGetOrganizationsQuery,
  useUpdateEntityMutation,
} from '../generated/graphql'
import EntityTable from '../features/entities/components/EntityTable'
import EntityEditorModal from '../features/entities/components/EntityEditorModal'
import {
  ColumnFilterValue,
  EntityFormState,
  ParsedEntityRow,
  createLinkedEntityMap,
  prepareFieldValueForSubmit,
  safeParseProperties,
} from '../features/entities/components/helpers'

type ModalState =
  | { mode: 'create' }
  | { mode: 'edit'; entity: Entity }

export const Route = createFileRoute('/entities')({
  component: EntitiesPage,
})

function EntitiesPage() {
  const queryClient = useQueryClient()
  const organizationsQuery = useGetOrganizationsQuery()
  const organizations = organizationsQuery.data?.organizations ?? []

  const [selectedOrgId, setSelectedOrgId] = useState<string | null>(null)
  const [selectedSchemaId, setSelectedSchemaId] = useState<string | null>(null)
  const [columnFilters, setColumnFilters] = useState<Record<string, ColumnFilterValue>>({})
  const [modalState, setModalState] = useState<ModalState | null>(null)
  const [modalError, setModalError] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<string | null>(null)
  const [page, setPage] = useState(0)

  useEffect(() => {
    if (!selectedOrgId && organizations.length > 0) {
      setSelectedOrgId(organizations[0].id)
    }
  }, [organizations, selectedOrgId])

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
  }, [selectedSchemaId, selectedOrgId])

  const selectedSchema = useMemo(
    () => schemas.find((schema) => schema.id === selectedSchemaId) ?? null,
    [schemas, selectedSchemaId],
  )

  const propertyFilters = useMemo(() => {
    return Object.entries(columnFilters)
      .map(([key, raw]) => [key, raw.trim()] as const)
      .filter(([, value]) => value.length > 0)
      .map(([key, value]) => ({
        key,
        value,
      }))
  }, [columnFilters])

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

  const queryVariables = useMemo(() => {
    if (!selectedOrgId) {
      return null
    }
    return {
      organizationId: selectedOrgId,
      pagination: {
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      },
      filter: entityFilter,
    }
  }, [entityFilter, page, selectedOrgId])

  const entitiesQuery = useEntitiesManagementQuery(
    queryVariables ?? {
      organizationId: '',
      pagination: { limit: PAGE_SIZE, offset: 0 },
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

  const totalCount = entitiesQuery.data?.entities.pageInfo.totalCount ?? 0
  const totalPages = totalCount > 0 ? Math.ceil(totalCount / PAGE_SIZE) : 0

  useEffect(() => {
    if (page > 0 && totalPages > 0 && page >= totalPages) {
      setPage(Math.max(totalPages - 1, 0))
    }
  }, [page, totalPages])

  const createEntityMutation = useCreateEntityMutation()
  const updateEntityMutation = useUpdateEntityMutation()
  const deleteEntityMutation = useDeleteEntityMutation()

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
        setFeedback('Entity created successfully.')
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
        setFeedback('Entity updated successfully.')
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
      setFeedback('Entity deleted successfully.')
      await refetchEntities()
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to delete entity.'
      setFeedback(message)
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
    <div className="mx-auto max-w-6xl px-6 py-8 text-gray-900">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold">Entities</h1>
          <p className="mt-1 text-sm text-gray-600">
            Browse, create, and maintain entities for a chosen schema.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setModalState({ mode: 'create' })}
          disabled={!selectedOrgId || !selectedSchema}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-300"
        >
          Add Entity
        </button>
      </div>

      <div className="mt-6 grid gap-4 sm:grid-cols-[2fr,2fr,1fr] sm:items-end">
        <label className="flex flex-col text-sm font-medium text-gray-700">
          Organization
          <select
            value={selectedOrgId ?? ''}
            onChange={(event) => {
              setSelectedOrgId(event.target.value || null)
              setFeedback(null)
            }}
            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
          >
            <option value="" disabled>
              Select an organization
            </option>
            {organizations.map((organization) => (
              <option key={organization.id} value={organization.id}>
                {organization.name}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col text-sm font-medium text-gray-700">
          Entity schema
          <select
            value={selectedSchemaId ?? ''}
            onChange={(event) => {
              setSelectedSchemaId(event.target.value || null)
              setFeedback(null)
            }}
            disabled={schemas.length === 0}
            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200 disabled:cursor-not-allowed disabled:bg-gray-100"
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

        <div className="text-xs text-gray-500">
          {activeOrganization ? (
            <p>
              Working in <span className="font-semibold">{activeOrganization.name}</span>
            </p>
          ) : organizationsQuery.isLoading ? (
            <p>Loading organizations…</p>
          ) : (
            <p>Select an organization.</p>
          )}
        </div>
      </div>

      {feedback && (
        <div className="mt-4 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-2 text-sm text-emerald-700">
          {feedback}
        </div>
      )}

      {entitiesQuery.error && (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
          {(entitiesQuery.error as Error).message}
        </div>
      )}

      <div className="mt-6 space-y-4">
        <EntityTable
          rows={tableRows}
          schemaFields={selectedSchema?.fields ?? []}
          columnFilters={columnFilters}
          onColumnFilterChange={handleColumnFilterChange}
          onEdit={(entity) => setModalState({ mode: 'edit', entity })}
          onDelete={handleDeleteEntity}
          summaryLabel={summaryLabel}
          isLoading={entitiesQuery.isLoading}
        />

        {totalPages > 1 && (
          <div className="flex items-center justify-between">
            <button
              type="button"
              onClick={() => setPage((value) => Math.max(value - 1, 0))}
              disabled={page === 0}
              className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900 disabled:cursor-not-allowed disabled:text-gray-400"
            >
              Previous
            </button>
            <div className="text-sm text-gray-600">
              Page {page + 1} of {totalPages}
            </div>
            <button
              type="button"
              onClick={() => {
                if (page + 1 < totalPages) {
                  setPage((value) => value + 1)
                }
              }}
              disabled={page + 1 >= totalPages}
              className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900 disabled:cursor-not-allowed disabled:text-gray-400"
            >
              Next
            </button>
          </div>
        )}
      </div>

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

const PAGE_SIZE = 10
