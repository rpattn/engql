import { createFileRoute } from '@tanstack/react-router'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import type { FilterFn } from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { graphqlRequest } from '../lib/graphql'

type PropertyFilter = {
  key: string
  value?: string | null
  exists?: boolean | null
  inArray?: string[]
}

type JoinType = 'REFERENCE' | 'CROSS'

type JoinSortInput = {
  side: 'LEFT' | 'RIGHT'
  field: string
  direction?: 'ASC' | 'DESC'
}

type JoinDefinition = {
  id: string
  name: string
  description?: string | null
  leftEntityType: string
  rightEntityType: string
  joinType: JoinType
  joinField?: string | null
  joinFieldType?: string | null
  createdAt: string
  updatedAt: string
  leftFilters?: PropertyFilter[]
  rightFilters?: PropertyFilter[]
  sortCriteria?: JoinSortInput[]
}

type JoinDefinitionRow = JoinDefinition & {
  descriptionLabel: string
  leftFiltersText: string
  rightFiltersText: string
  sortCriteriaText: string
}

type JoinResultRow = {
  key: string
  leftId: string
  leftType: string
  leftSummary: string
  rightId: string
  rightType: string
  rightSummary: string
}

type ListJoinDefinitionsResponse = {
  entityJoinDefinitions: JoinDefinition[]
}

type CreateJoinResponse = {
  createEntityJoinDefinition: JoinDefinition
}

type DeleteJoinResponse = {
  deleteEntityJoinDefinition: boolean
}

type ExecuteJoinResponse = {
  executeEntityJoin: {
    edges: Array<{
      left: {
        id: string
        entityType: string
        properties: string
      }
      right: {
        id: string
        entityType: string
        properties: string
      }
    }>
    pageInfo: {
      totalCount: number
      hasNextPage: boolean
      hasPreviousPage: boolean
    }
  }
}

const CREATE_JOIN_MUTATION = `
  mutation CreateJoin($input: CreateEntityJoinDefinitionInput!) {
    createEntityJoinDefinition(input: $input) {
      id
      name
      description
      joinType
      leftEntityType
      rightEntityType
      joinField
      joinFieldType
      createdAt
      updatedAt
      leftFilters { key value exists inArray }
      rightFilters { key value exists inArray }
      sortCriteria { side field direction }
    }
  }
`

const LIST_JOIN_DEFINITIONS_QUERY = `
  query EntityJoinDefinitions($organizationId: String!) {
    entityJoinDefinitions(organizationId: $organizationId) {
      id
      name
      description
      joinType
      leftEntityType
      rightEntityType
      joinField
      joinFieldType
      createdAt
      updatedAt
      leftFilters { key value exists inArray }
      rightFilters { key value exists inArray }
      sortCriteria { side field direction }
    }
  }
`

const DELETE_JOIN_MUTATION = `
  mutation DeleteJoin($id: String!) {
    deleteEntityJoinDefinition(id: $id)
  }
`

const EXECUTE_JOIN_QUERY = `
  query ExecuteJoin($input: ExecuteEntityJoinInput!) {
    executeEntityJoin(input: $input) {
      edges {
        left { id entityType properties }
        right { id entityType properties }
      }
      pageInfo { totalCount hasNextPage hasPreviousPage }
    }
  }
`

const joinColumnHelper = createColumnHelper<JoinDefinitionRow>()
const resultColumnHelper = createColumnHelper<JoinResultRow>()

export const Route = createFileRoute('/join-testing')({
  component: JoinTestingPage,
})

const noopFuzzyFilter: FilterFn<unknown> = () => true

function parseJSONInput<T>(raw: string, fallback: T): T | Error {
  const trimmed = raw.trim()
  if (!trimmed) return fallback
  try {
    return JSON.parse(trimmed) as T
  } catch (error) {
    return error as Error
  }
}

function formatPropertiesSummary(raw: string) {
  if (!raw) return '-'
  try {
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed === 'object') {
      if (
        'name' in parsed &&
        typeof (parsed as Record<string, unknown>).name === 'string'
      ) {
        return (parsed as Record<string, unknown>).name as string
      }
      const serialised = JSON.stringify(parsed)
      return serialised.length > 140
        ? `${serialised.slice(0, 140)}...`
        : serialised
    }
  } catch {
    return raw
  }
  return raw
}

function stringifyForTextarea(value: unknown, fallback = '[]') {
  try {
    const json = JSON.stringify(value ?? [], null, 2)
    return json && json.length > 0 ? json : fallback
  } catch {
    return fallback
  }
}

function JoinTestingPage() {
  const [createOrgId, setCreateOrgId] = useState('')
  const [createName, setCreateName] = useState('')
  const [createDescription, setCreateDescription] = useState('')
  const [createLeftType, setCreateLeftType] = useState('')
  const [createRightType, setCreateRightType] = useState('')
  const [createJoinField, setCreateJoinField] = useState('')
  const [createJoinType, setCreateJoinType] = useState<JoinType>('REFERENCE')
  const [createLeftFilters, setCreateLeftFilters] = useState('[]')
  const [createRightFilters, setCreateRightFilters] = useState('[]')
  const [createSortCriteria, setCreateSortCriteria] = useState('[]')
  const [createError, setCreateError] = useState<string | null>(null)

  const [listOrgId, setListOrgId] = useState('')
  const [listError, setListError] = useState<string | null>(null)

  const [executeJoinId, setExecuteJoinId] = useState('')
  const [executeLeftFilters, setExecuteLeftFilters] = useState('[]')
  const [executeRightFilters, setExecuteRightFilters] = useState('[]')
  const [executeSortCriteria, setExecuteSortCriteria] = useState('[]')
  const [executeLimit, setExecuteLimit] = useState('25')
  const [executeOffset, setExecuteOffset] = useState('0')
  const [executeError, setExecuteError] = useState<string | null>(null)

  const [joinResults, setJoinResults] = useState<JoinResultRow[]>([])
  const [resultPageInfo, setResultPageInfo] = useState<{
    totalCount: number
    hasNextPage: boolean
    hasPreviousPage: boolean
  } | null>(null)

  const listQuery = useQuery({
    queryKey: ['entityJoinDefinitions', listOrgId],
    enabled: false,
    queryFn: () =>
      graphqlRequest<ListJoinDefinitionsResponse>(
        LIST_JOIN_DEFINITIONS_QUERY,
        { organizationId: listOrgId },
      ),
  })

  const createJoinMutation = useMutation({
    mutationFn: (variables: {
      organizationId: string
      name: string
      description?: string
      joinType: JoinType
      leftEntityType: string
      rightEntityType: string
      joinField?: string
      leftFilters: PropertyFilter[]
      rightFilters: PropertyFilter[]
      sortCriteria: JoinSortInput[]
    }) =>
      graphqlRequest<CreateJoinResponse>(CREATE_JOIN_MUTATION, {
        input: variables,
      }),
    onSuccess: () => {
      setCreateError(null)
      if (listOrgId) {
        void listQuery.refetch()
      }
    },
  })

  const deleteJoinMutation = useMutation({
    mutationFn: (id: string) =>
      graphqlRequest<DeleteJoinResponse>(DELETE_JOIN_MUTATION, { id }),
    onSuccess: (_data, deletedId) => {
      if (listOrgId) {
        void listQuery.refetch()
      }
      if (deletedId && executeJoinId === deletedId) {
        setExecuteJoinId('')
      }
    },
  })

  const executeJoinMutation = useMutation({
    mutationFn: (variables: {
      joinId: string
      leftFilters: PropertyFilter[]
      rightFilters: PropertyFilter[]
      sortCriteria: JoinSortInput[]
      pagination: { limit: number; offset: number }
    }) =>
      graphqlRequest<ExecuteJoinResponse>(EXECUTE_JOIN_QUERY, {
        input: variables,
      }),
    onSuccess: (data) => {
      const edges = data.executeEntityJoin.edges ?? []
      const rows: JoinResultRow[] = edges.map((edge, index) => ({
        key: `${edge.left.id}-${edge.right.id}-${index}`,
        leftId: edge.left.id,
        leftType: edge.left.entityType,
        leftSummary: formatPropertiesSummary(edge.left.properties),
        rightId: edge.right.id,
        rightType: edge.right.entityType,
        rightSummary: formatPropertiesSummary(edge.right.properties),
      }))
      setJoinResults(rows)
      setResultPageInfo(data.executeEntityJoin.pageInfo)
      setExecuteError(null)
    },
  })

  const joinDefinitions =
    useMemo(
      () =>
        listQuery.data?.entityJoinDefinitions?.map((definition) => ({
          ...definition,
          descriptionLabel: definition.description ?? '',
          leftFiltersText: stringifyForTextarea(definition.leftFilters),
          rightFiltersText: stringifyForTextarea(definition.rightFilters),
          sortCriteriaText: stringifyForTextarea(definition.sortCriteria),
        })) ?? [],
      [listQuery.data],
    )

  const joinColumns = useMemo(
    () => [
      joinColumnHelper.accessor('name', {
        header: 'Name',
        cell: (info) => (
          <div className="font-medium text-slate-100">{info.getValue()}</div>
        ),
      }),
      joinColumnHelper.accessor('joinType', {
        header: 'Join Type',
        cell: (info) => info.getValue(),
      }),
      joinColumnHelper.accessor('leftEntityType', {
        header: 'Left Type',
        cell: (info) => info.getValue(),
      }),
      joinColumnHelper.accessor('rightEntityType', {
        header: 'Right Type',
        cell: (info) => info.getValue(),
      }),
      joinColumnHelper.accessor('joinField', {
        header: 'Join Field',
        cell: (info) => {
          const value = info.getValue()
          if (typeof value === 'string' && value.trim().length > 0) {
            return value
          }
          return '—'
        },
      }),
      joinColumnHelper.display({
        id: 'actions',
        header: 'Actions',
        cell: (info) => {
          const row = info.row.original
          return (
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => prefillRunForm(row)}
                className="rounded-md bg-cyan-600 px-3 py-1 text-xs font-medium text-white hover:bg-cyan-500"
              >
                Use
              </button>
              <button
                type="button"
                onClick={() => handleDeleteJoin(row.id)}
                className="rounded-md bg-red-600 px-3 py-1 text-xs font-medium text-white hover:bg-red-500 disabled:opacity-60"
                disabled={deleteJoinMutation.isPending}
              >
                Delete
              </button>
            </div>
          )
        },
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [deleteJoinMutation.isPending],
  )

  const joinTable = useReactTable({
    data: joinDefinitions,
    columns: joinColumns,
    filterFns: { fuzzy: noopFuzzyFilter },
    getCoreRowModel: getCoreRowModel(),
  })

  const resultColumns = useMemo(
    () => [
      resultColumnHelper.accessor('leftType', {
        header: 'Left Type',
        cell: (info) => info.getValue(),
      }),
      resultColumnHelper.accessor('leftSummary', {
        header: 'Left Summary',
        cell: (info) => (
          <div className="whitespace-pre-wrap text-slate-100">
            {info.getValue()}
          </div>
        ),
      }),
      resultColumnHelper.accessor('rightType', {
        header: 'Right Type',
        cell: (info) => info.getValue(),
      }),
      resultColumnHelper.accessor('rightSummary', {
        header: 'Right Summary',
        cell: (info) => (
          <div className="whitespace-pre-wrap text-slate-100">
            {info.getValue()}
          </div>
        ),
      }),
    ],
    [],
  )

  const resultTable = useReactTable({
    data: joinResults,
    columns: resultColumns,
    filterFns: { fuzzy: noopFuzzyFilter },
    getCoreRowModel: getCoreRowModel(),
  })

  function prefillRunForm(definition: JoinDefinitionRow) {
    setExecuteJoinId(definition.id)
    setExecuteLeftFilters(definition.leftFiltersText)
    setExecuteRightFilters(definition.rightFiltersText)
    setExecuteSortCriteria(definition.sortCriteriaText)
    setJoinResults([])
    setExecuteError(null)
  }

  function handleDeleteJoin(id: string) {
    if (!window.confirm('Delete this join definition?')) {
      return
    }
    deleteJoinMutation.mutate(id)
  }

  function handleCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setCreateError(null)

    if (!createOrgId || !createName || !createLeftType || !createRightType) {
      setCreateError('Organization, name, and entity types are required.')
      return
    }

    const trimmedJoinField = createJoinField.trim()
    if (createJoinType === 'REFERENCE' && !trimmedJoinField) {
      setCreateError('Join field is required for reference joins.')
      return
    }
    if (createJoinType === 'CROSS' && trimmedJoinField) {
      setCreateError('Remove the join field when creating a cross join.')
      return
    }

    const parsedLeftFilters = parseJSONInput<PropertyFilter[]>(
      createLeftFilters,
      [],
    )
    if (parsedLeftFilters instanceof Error) {
      setCreateError('Invalid JSON for left filters.')
      return
    }

    const parsedRightFilters = parseJSONInput<PropertyFilter[]>(
      createRightFilters,
      [],
    )
    if (parsedRightFilters instanceof Error) {
      setCreateError('Invalid JSON for right filters.')
      return
    }

    const parsedSortCriteria = parseJSONInput<JoinSortInput[]>(
      createSortCriteria,
      [],
    )
    if (parsedSortCriteria instanceof Error) {
      setCreateError('Invalid JSON for sort criteria.')
      return
    }

    createJoinMutation.mutate({
      organizationId: createOrgId,
      name: createName,
      description: createDescription || undefined,
      joinType: createJoinType,
      leftEntityType: createLeftType,
      rightEntityType: createRightType,
      joinField: createJoinType === 'REFERENCE' ? trimmedJoinField : undefined,
      leftFilters: parsedLeftFilters,
      rightFilters: parsedRightFilters,
      sortCriteria: parsedSortCriteria,
    })
  }

  async function handleFetchDefinitions() {
    setListError(null)
    if (!listOrgId) {
      setListError('Organization ID is required to list definitions.')
      return
    }
    await listQuery.refetch()
  }

  function handleExecute(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setExecuteError(null)

    if (!executeJoinId) {
      setExecuteError('Select or enter a join definition ID.')
      return
    }

    const parsedLeftFilters = parseJSONInput<PropertyFilter[]>(
      executeLeftFilters,
      [],
    )
    if (parsedLeftFilters instanceof Error) {
      setExecuteError('Invalid JSON for left filters.')
      return
    }

    const parsedRightFilters = parseJSONInput<PropertyFilter[]>(
      executeRightFilters,
      [],
    )
    if (parsedRightFilters instanceof Error) {
      setExecuteError('Invalid JSON for right filters.')
      return
    }

    const parsedSortCriteria = parseJSONInput<JoinSortInput[]>(
      executeSortCriteria,
      [],
    )
    if (parsedSortCriteria instanceof Error) {
      setExecuteError('Invalid JSON for sort criteria.')
      return
    }

    const limit = Number.parseInt(executeLimit, 10)
    const offset = Number.parseInt(executeOffset, 10)
    if (Number.isNaN(limit) || Number.isNaN(offset)) {
      setExecuteError('Limit and offset must be valid numbers.')
      return
    }

    executeJoinMutation.mutate({
      joinId: executeJoinId,
      leftFilters: parsedLeftFilters,
      rightFilters: parsedRightFilters,
      sortCriteria: parsedSortCriteria,
      pagination: { limit, offset: Math.max(offset, 0) },
    })
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-10">
      <header>
        <h1 className="text-3xl font-semibold text-white">Entity Join Testing</h1>
        <p className="mt-2 text-slate-300">
          Define joins, inspect saved definitions, and execute them with optional
          runtime filters.
        </p>
      </header>

      <section className="rounded-2xl bg-slate-900/70 p-6 shadow-xl ring-1 ring-white/10">
        <h2 className="text-xl font-semibold text-white">Create Join Definition</h2>
        <p className="mt-1 text-sm text-slate-300">
          Supply the linking details and optional filter or sort rules.
        </p>

        {createError && (
          <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
            {createError}
          </div>
        )}

        <form className="mt-4 grid gap-4 md:grid-cols-2" onSubmit={handleCreate}>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Organization ID
            </label>
            <input
              value={createOrgId}
              onChange={(event) => setCreateOrgId(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="UUID"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">Name</label>
            <input
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="ComponentsToTeams"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Left Entity Type
            </label>
            <input
              value={createLeftType}
              onChange={(event) => setCreateLeftType(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="Component"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Right Entity Type
            </label>
            <input
              value={createRightType}
              onChange={(event) => setCreateRightType(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="Team"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Join Type
            </label>
            <select
              value={createJoinType}
              onChange={(event) => {
                const nextType = event.target.value as JoinType
                setCreateJoinType(nextType)
                if (nextType === 'CROSS') {
                  setCreateJoinField('')
                }
              }}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
            >
              <option value="REFERENCE">Reference (field match)</option>
              <option value="CROSS">Cross (all combinations)</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Join Field
            </label>
            <input
              value={createJoinField}
              onChange={(event) => setCreateJoinField(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40 disabled:cursor-not-allowed disabled:opacity-60"
              placeholder={
                createJoinType === 'REFERENCE'
                  ? 'owner'
                  : 'Not required for cross joins'
              }
              disabled={createJoinType === 'CROSS'}
            />
            <p className="mt-1 text-xs text-slate-400">
              {createJoinType === 'REFERENCE'
                ? 'Name of the left-side field that references the right entity.'
                : 'Cross joins ignore join fields and pair every left entity with every right entity.'}
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Description (optional)
            </label>
            <input
              value={createDescription}
              onChange={(event) => setCreateDescription(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="Join component owners to teams"
            />
          </div>
          <div className="md:col-span-2 grid gap-4 md:grid-cols-3">
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Left Filters JSON
              </label>
              <textarea
                value={createLeftFilters}
                onChange={(event) => setCreateLeftFilters(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Right Filters JSON
              </label>
              <textarea
                value={createRightFilters}
                onChange={(event) => setCreateRightFilters(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Sort Criteria JSON
              </label>
              <textarea
                value={createSortCriteria}
                onChange={(event) => setCreateSortCriteria(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
          </div>
          <div className="md:col-span-2">
            <button
              type="submit"
              disabled={createJoinMutation.isPending}
              className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {createJoinMutation.isPending ? 'Creating...' : 'Create Join'}
            </button>
          </div>
        </form>
      </section>

      <section className="rounded-2xl bg-slate-900/70 p-6 shadow-xl ring-1 ring-white/10">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white">Join Definitions</h2>
            <p className="mt-1 text-sm text-slate-300">
              Fetch stored definitions for an organization and pick one to execute.
            </p>
          </div>
          <div className="flex flex-col gap-2 md:flex-row md:items-center md:gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Organization ID
              </label>
              <input
                value={listOrgId}
                onChange={(event) => setListOrgId(event.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40 md:w-56"
                placeholder="UUID"
              />
            </div>
            <button
              type="button"
              onClick={() => void handleFetchDefinitions()}
              disabled={listQuery.isFetching}
              className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {listQuery.isFetching ? 'Loading...' : 'Fetch Definitions'}
            </button>
          </div>
        </div>

        {listError && (
          <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
            {listError}
          </div>
        )}

        {listQuery.error instanceof Error && (
          <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
            {listQuery.error.message}
          </div>
        )}

        {joinDefinitions.length > 0 ? (
          <div className="mt-6 overflow-auto rounded-xl border border-slate-700/60">
            <table className="min-w-full divide-y divide-slate-700">
              <thead className="bg-slate-900/80">
                {joinTable.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <th
                        key={header.id}
                        className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-300"
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody className="divide-y divide-slate-800">
                {joinTable.getRowModel().rows.map((row) => (
                  <tr key={row.id} className="hover:bg-slate-900/60">
                    {row.getVisibleCells().map((cell) => (
                      <td
                        key={cell.id}
                        className="whitespace-pre-wrap px-4 py-3 text-sm text-slate-200"
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          listQuery.isFetched && (
            <div className="mt-6 rounded-md border border-slate-700/60 bg-slate-900/70 px-4 py-3 text-sm text-slate-200">
              No join definitions found.
            </div>
          )
        )}
      </section>

      <section className="rounded-2xl bg-slate-900/70 p-6 shadow-xl ring-1 ring-white/10">
        <h2 className="text-xl font-semibold text-white">Execute Join</h2>
        <p className="mt-1 text-sm text-slate-300">
          Use a saved definition to return paired entities. Filters are optional overrides.
        </p>

        {executeError && (
          <div className="mt-4 rounded-md border border-red-500/70 bg-red-500/10 px-3 py-2 text-sm text-red-200">
            {executeError}
          </div>
        )}

        <form className="mt-4 grid gap-4 md:grid-cols-2" onSubmit={handleExecute}>
          <div>
            <label className="block text-sm font-medium text-slate-200">
              Join Definition ID
            </label>
            <input
              value={executeJoinId}
              onChange={(event) => setExecuteJoinId(event.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              placeholder="Select from the table or paste an ID"
            />
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-200">
                Limit
              </label>
              <input
                value={executeLimit}
                onChange={(event) => setExecuteLimit(event.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-200">
                Offset
              </label>
              <input
                value={executeOffset}
                onChange={(event) => setExecuteOffset(event.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
          </div>
          <div className="md:col-span-2 grid gap-4 md:grid-cols-3">
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Left Filters JSON
              </label>
              <textarea
                value={executeLeftFilters}
                onChange={(event) => setExecuteLeftFilters(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Right Filters JSON
              </label>
              <textarea
                value={executeRightFilters}
                onChange={(event) => setExecuteRightFilters(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-200">
                Sort Criteria JSON
              </label>
              <textarea
                value={executeSortCriteria}
                onChange={(event) => setExecuteSortCriteria(event.target.value)}
                className="mt-1 h-28 w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-500/40"
              />
            </div>
          </div>
          <div className="md:col-span-2">
            <button
              type="submit"
              disabled={executeJoinMutation.isPending}
              className="inline-flex items-center justify-center rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white shadow hover:bg-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-300 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {executeJoinMutation.isPending ? 'Executing...' : 'Run Join'}
            </button>
          </div>
        </form>

        {resultPageInfo && (
          <div className="mt-4 text-sm text-slate-300">
            Total matches: {resultPageInfo.totalCount} · Next page:{' '}
            {resultPageInfo.hasNextPage ? 'yes' : 'no'} · Previous page:{' '}
            {resultPageInfo.hasPreviousPage ? 'yes' : 'no'}
          </div>
        )}

        {joinResults.length > 0 && (
          <div className="mt-6 overflow-auto rounded-xl border border-slate-700/60">
            <table className="min-w-full divide-y divide-slate-700">
              <thead className="bg-slate-900/80">
                {resultTable.getHeaderGroups().map((headerGroup) => (
                  <tr key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
                      <th
                        key={header.id}
                        className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-300"
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(
                              header.column.columnDef.header,
                              header.getContext(),
                            )}
                      </th>
                    ))}
                  </tr>
                ))}
              </thead>
              <tbody className="divide-y divide-slate-800">
                {resultTable.getRowModel().rows.map((row) => (
                  <tr key={row.id} className="hover:bg-slate-900/60">
                    {row.getVisibleCells().map((cell) => (
                      <td
                        key={cell.id}
                        className="whitespace-pre-wrap px-4 py-3 text-sm text-slate-200"
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
