import { createFileRoute } from '@tanstack/react-router'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { graphqlRequest } from '../lib/graphql'

type FieldDefinitionInput = {
  name: string
  type: string
  required?: boolean
  description?: string
  default?: string
  validation?: string
  referenceEntityType?: string
}

type CreateSchemaResponse = {
  createEntitySchema: {
    id: string
    name: string
    description?: string | null
  }
}

type CreateEntityResponse = {
  createEntity: {
    id: string
    entityType: string
    properties: string
  }
}

type EntitiesByTypeResponse = {
  entitiesByType: Array<{
    id: string
    entityType: string
    properties: string
    linkedEntities: Array<{
      id: string
      entityType: string
      properties: string
    }>
  }>
}

const CREATE_SCHEMA_MUTATION = `
  mutation CreateSchema($input: CreateEntitySchemaInput!) {
    createEntitySchema(input: $input) {
      id
      name
      description
    }
  }
`

const CREATE_ENTITY_MUTATION = `
  mutation CreateEntity($input: CreateEntityInput!) {
    createEntity(input: $input) {
      id
      entityType
      properties
    }
  }
`

const ENTITIES_BY_TYPE_QUERY = `
  query EntitiesByType($organizationId: String!, $entityType: String!) {
    entitiesByType(organizationId: $organizationId, entityType: $entityType) {
      id
      entityType
      properties
      linkedEntities {
        id
        entityType
        properties
      }
    }
  }
`

export const Route = createFileRoute('/')({
  component: App,
})

function App() {
  const [organizationId, setOrganizationId] = useState('')

  const [schemaName, setSchemaName] = useState('')
  const [schemaDescription, setSchemaDescription] = useState('')
  const [schemaFieldsInput, setSchemaFieldsInput] = useState(
    JSON.stringify(
      [
        {
          name: 'name',
          type: 'STRING',
          required: true,
        },
      ],
      null,
      2,
    ),
  )
  const [schemaFormError, setSchemaFormError] = useState<string | null>(null)

  const [entityType, setEntityType] = useState('')
  const [entityPath, setEntityPath] = useState('')
  const [entityPropertiesInput, setEntityPropertiesInput] = useState(
    JSON.stringify(
      {
        name: 'Example Entity',
        linked_ids: [],
      },
      null,
      2,
    ),
  )
  const [primaryLinkedId, setPrimaryLinkedId] = useState('')
  const [additionalLinkedIds, setAdditionalLinkedIds] = useState('')
  const [entityFormError, setEntityFormError] = useState<string | null>(null)

  const [queryEntityType, setQueryEntityType] = useState('')
  const [queryError, setQueryError] = useState<string | null>(null)

  const createSchemaMutation = useMutation({
    mutationFn: (variables: {
      organizationId: string
      name: string
      description?: string
      fields: FieldDefinitionInput[]
    }) =>
      graphqlRequest<CreateSchemaResponse>(CREATE_SCHEMA_MUTATION, {
        input: {
          organizationId: variables.organizationId,
          name: variables.name,
          description: variables.description,
          fields: variables.fields,
        },
      }),
  })

  const createEntityMutation = useMutation({
    mutationFn: (variables: {
      organizationId: string
      entityType: string
      path?: string
      properties: string
      linkedEntityId?: string
      linkedEntityIds?: string[]
    }) =>
      graphqlRequest<CreateEntityResponse>(CREATE_ENTITY_MUTATION, {
        input: {
          organizationId: variables.organizationId,
          entityType: variables.entityType,
          path: variables.path,
          properties: variables.properties,
          linkedEntityId: variables.linkedEntityId,
          linkedEntityIds: variables.linkedEntityIds,
        },
      }),
  })

  const {
    data: entitiesData,
    isFetching: isFetchingEntities,
    refetch: refetchEntities,
  } = useQuery({
    queryKey: ['entitiesByType', organizationId, queryEntityType],
    queryFn: () =>
      graphqlRequest<EntitiesByTypeResponse>(ENTITIES_BY_TYPE_QUERY, {
        organizationId,
        entityType: queryEntityType,
      }),
    enabled: false,
  })

  const handleCreateSchema = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!organizationId.trim()) {
      setSchemaFormError('Organization ID is required.')
      return
    }
    if (!schemaName.trim()) {
      setSchemaFormError('Schema name is required.')
      return
    }

    let fields: FieldDefinitionInput[]
    try {
      fields = JSON.parse(schemaFieldsInput)
    } catch (error) {
      setSchemaFormError('Fields must be valid JSON.')
      return
    }

    if (!Array.isArray(fields)) {
      setSchemaFormError('Fields JSON must describe an array.')
      return
    }

    setSchemaFormError(null)

    try {
      await createSchemaMutation.mutateAsync({
        organizationId: organizationId.trim(),
        name: schemaName.trim(),
        description: schemaDescription.trim() || undefined,
        fields,
      })
    } catch (error) {
      if (error instanceof Error) {
        setSchemaFormError(error.message)
      } else {
        setSchemaFormError('Failed to create schema.')
      }
    }
  }

  const handleCreateEntity = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!organizationId.trim()) {
      setEntityFormError('Organization ID is required.')
      return
    }
    if (!entityType.trim()) {
      setEntityFormError('Entity type is required.')
      return
    }

    let propertiesObj: unknown
    try {
      propertiesObj = JSON.parse(entityPropertiesInput)
    } catch (error) {
      setEntityFormError('Properties must be valid JSON.')
      return
    }

    if (typeof propertiesObj !== 'object' || propertiesObj === null) {
      setEntityFormError('Properties JSON must describe an object.')
      return
    }

    const linkedIdsFromInput = additionalLinkedIds
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)

    const primaryLinked = primaryLinkedId.trim()
    const uniqueLinkedIds = Array.from(
      new Set(
        [
          primaryLinked.length ? primaryLinked : null,
          ...linkedIdsFromInput,
        ].filter(Boolean) as string[],
      ),
    )

    setEntityFormError(null)

    try {
      await createEntityMutation.mutateAsync({
        organizationId: organizationId.trim(),
        entityType: entityType.trim(),
        path: entityPath.trim() || undefined,
        properties: JSON.stringify(propertiesObj),
        linkedEntityId:
          primaryLinked.length > 0 ? primaryLinked : undefined,
        linkedEntityIds:
          uniqueLinkedIds.length > (primaryLinked.length ? 1 : 0)
            ? uniqueLinkedIds
            : undefined,
      })
    } catch (error) {
      if (error instanceof Error) {
        setEntityFormError(error.message)
      } else {
        setEntityFormError('Failed to create entity.')
      }
    }
  }

  const handleFetchEntities = async () => {
    if (!organizationId.trim() || !queryEntityType.trim()) {
      setQueryError('Organization ID and entity type are required.')
      return
    }
    setQueryError(null)
    await refetchEntities()
  }

  const safeParseProperties = (value: string) => {
    try {
      return JSON.parse(value)
    } catch {
      return value
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-900 via-slate-800 to-slate-900">
      <main className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-10 text-slate-100">
        <section className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
          <h2 className="text-lg font-semibold text-white">
            Organization Context
          </h2>
          <p className="mt-1 text-sm text-slate-300">
            Provide the organization ID to scope schema and entity operations.
          </p>
          <div className="mt-4">
            <label className="block text-sm font-medium text-slate-200">
              Organization ID
            </label>
            <input
              value={organizationId}
              onChange={(event) => setOrganizationId(event.target.value)}
              placeholder="e.g. 4dc7d89e-..."
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
            />
          </div>
        </section>

        <section className="grid gap-6 lg:grid-cols-2">
          <div className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">
                Create Entity Schema
              </h2>
              {createSchemaMutation.isSuccess && (
                <span className="text-sm text-emerald-400">
                  Schema saved!
                </span>
              )}
            </div>
            <form
              className="mt-4 flex flex-col gap-4"
              onSubmit={handleCreateSchema}
            >
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Schema Name
                </label>
                <input
                  value={schemaName}
                  onChange={(event) => setSchemaName(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Description (optional)
                </label>
                <input
                  value={schemaDescription}
                  onChange={(event) =>
                    setSchemaDescription(event.target.value)
                  }
                  placeholder="Short description"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Fields JSON
                </label>
                <textarea
                  value={schemaFieldsInput}
                  onChange={(event) => setSchemaFieldsInput(event.target.value)}
                  rows={8}
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Provide an array of FieldDefinitionInput objects. Supported
                  types include STRING, ENTITY_ID, ENTITY_REFERENCE, and
                  ENTITY_REFERENCE_ARRAY. ENTITY_ID fields auto-resolve to the
                  referenced entity when querying.
                </p>
              </div>

              {schemaFormError && (
                <div className="rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                  {schemaFormError}
                </div>
              )}

              <button
                type="submit"
                disabled={createSchemaMutation.isPending}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {createSchemaMutation.isPending
                  ? 'Creating...'
                  : 'Create Schema'}
              </button>
            </form>
          </div>

          <div className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">
                Create Entity
              </h2>
              {createEntityMutation.isSuccess && (
                <span className="text-sm text-emerald-400">
                  Entity created!
                </span>
              )}
            </div>

            <form
              className="mt-4 flex flex-col gap-4"
              onSubmit={handleCreateEntity}
            >
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Entity Type
                </label>
                <input
                  value={entityType}
                  onChange={(event) => setEntityType(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Path (optional)
                </label>
                <input
                  value={entityPath}
                  onChange={(event) => setEntityPath(event.target.value)}
                  placeholder="root.component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Properties JSON
                </label>
                <textarea
                  value={entityPropertiesInput}
                  onChange={(event) =>
                    setEntityPropertiesInput(event.target.value)
                  }
                  rows={8}
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                />
                <p className="mt-1 text-xs text-slate-400">
                  Provide an object. The UI will automatically maintain the
                  special linked_ids array.
                </p>
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-slate-200">
                    Primary Linked Entity ID
                  </label>
                  <input
                    value={primaryLinkedId}
                    onChange={(event) => setPrimaryLinkedId(event.target.value)}
                    placeholder="Primary linked entity (optional)"
                    className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-200">
                    Additional Linked IDs
                  </label>
                  <input
                    value={additionalLinkedIds}
                    onChange={(event) =>
                      setAdditionalLinkedIds(event.target.value)
                    }
                    placeholder="Comma separated list"
                    className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
                  />
                </div>
              </div>

              {entityFormError && (
                <div className="rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                  {entityFormError}
                </div>
              )}

              <button
                type="submit"
                disabled={createEntityMutation.isPending}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {createEntityMutation.isPending
                  ? 'Creating...'
                  : 'Create Entity'}
              </button>
            </form>
          </div>
        </section>

        <section className="rounded-2xl bg-slate-900/60 p-6 shadow-xl ring-1 ring-white/10 backdrop-blur">
          <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-white">
                Query Entities by Type
              </h2>
              <p className="mt-1 text-sm text-slate-300">
                Fetch entities for the selected organization and type. Linked
                entities resolve automatically.
              </p>
            </div>
            <div className="flex flex-col gap-2 md:flex-row md:items-center">
              <div>
                <label className="block text-sm font-medium text-slate-200">
                  Entity Type
                </label>
                <input
                  value={queryEntityType}
                  onChange={(event) => setQueryEntityType(event.target.value)}
                  placeholder="Component"
                  className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40 md:w-56"
                />
              </div>
              <button
                onClick={handleFetchEntities}
                disabled={isFetchingEntities}
                className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isFetchingEntities ? 'Fetching...' : 'Fetch Entities'}
              </button>
            </div>
          </div>

          {queryError && (
            <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
              {queryError}
            </div>
          )}

          {entitiesData?.entitiesByType && (
            <div className="mt-6 grid gap-4 md:grid-cols-2">
              {entitiesData.entitiesByType.map((entity) => {
                const parsedProps = safeParseProperties(entity.properties)
                return (
                  <div
                    key={entity.id}
                    className="rounded-xl border border-slate-700/60 bg-slate-900/70 p-4"
                  >
                    <div className="text-sm font-medium text-cyan-300">
                      {entity.entityType}
                    </div>
                    <div className="mt-1 text-xs text-slate-400">
                      ID: {entity.id}
                    </div>
                    <pre className="mt-3 max-h-40 overflow-auto rounded-lg bg-slate-950/60 p-3 text-xs text-slate-200">
                      {JSON.stringify(parsedProps, null, 2)}
                    </pre>
                    <div className="mt-3 text-sm text-slate-300">
                      Linked Entities:
                    </div>
                    {entity.linkedEntities.length ? (
                      <ul className="mt-1 space-y-1 text-xs text-slate-400">
                        {entity.linkedEntities.map((link) => {
                          const linkedProps = safeParseProperties(link.properties)
                          let name: string | undefined
                          if (
                            linkedProps &&
                            typeof linkedProps === 'object' &&
                            'name' in linkedProps
                          ) {
                            const maybeString = linkedProps['name']
                            if (typeof maybeString === 'string') {
                              name = maybeString
                            }
                          }

                          return (
                            <li key={link.id} className="space-y-1">
                              <div>
                                <span className="text-slate-200">
                                  {link.entityType}
                                </span>{' '}
                                – {link.id}
                                {name ? (
                                  <span className="text-slate-400">
                                    {' '}
                                    ({name})
                                  </span>
                                ) : null}
                              </div>
                              <pre className="max-h-32 overflow-auto rounded-md bg-slate-950/70 p-2 text-[11px] text-slate-200">
                                {JSON.stringify(linkedProps, null, 2)}
                              </pre>
                            </li>
                          )
                        })}
                      </ul>
                    ) : (
                      <p className="mt-1 text-xs text-slate-500">
                        No linked entities.
                      </p>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </main>
    </div>
  )
}
