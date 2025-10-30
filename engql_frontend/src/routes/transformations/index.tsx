import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'

import {
  useCreateEntityTransformationMutation,
  useDeleteEntityTransformationMutation,
  useEntityTransformationsQuery,
} from '@/generated/graphql'
import { useTransformationsContext } from '@/features/transformations/context'
import { TransformationListTable } from '@/features/transformations/components/TransformationListTable'

export const Route = createFileRoute('/transformations/')({
  component: TransformationsListRoute,
})

function TransformationsListRoute() {
  const { organizationId } = useTransformationsContext()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const trimmedOrgId = organizationId.trim()

  const listQuery = useEntityTransformationsQuery(
    { organizationId: trimmedOrgId },
    {
      enabled: Boolean(trimmedOrgId),
    },
  )

  const createMutation = useCreateEntityTransformationMutation({
    onSuccess: (data) => {
      const created = data.createEntityTransformation
      queryClient.invalidateQueries({
        queryKey: useEntityTransformationsQuery.getKey({ organizationId: trimmedOrgId }),
      })
      navigate({
        to: '/transformations/$transformationId',
        params: { transformationId: created.id },
      })
    },
  })

  const deleteMutation = useDeleteEntityTransformationMutation({
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: useEntityTransformationsQuery.getKey({ organizationId: trimmedOrgId }),
      })
    },
  })

  const canLoad = Boolean(trimmedOrgId)

  const rows = useMemo(
    () => listQuery.data?.entityTransformations ?? [],
    [listQuery.data?.entityTransformations],
  )

  const handleCreate = () => {
    if (!trimmedOrgId) {
      alert('Please select an organization before creating a transformation.')
      return
    }

    const name = prompt('Transformation name')?.trim()
    if (!name) {
      return
    }

    const description = prompt('Optional description')?.trim()

    createMutation.mutate({
      input: {
        organizationId: trimmedOrgId,
        name,
        description: description || undefined,
        nodes: [],
      },
    })
  }

  const handleDelete = (id: string) => {
    if (!confirm('Delete this transformation? This cannot be undone.')) {
      return
    }

    deleteMutation.mutate({ id })
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Catalog</h2>
        <button
          type="button"
          onClick={handleCreate}
          className="rounded-md bg-blue-600 px-4 py-2 text-xs font-semibold text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-400/60"
          disabled={!trimmedOrgId || createMutation.isPending}
        >
          {createMutation.isPending ? 'Creating…' : 'New transformation'}
        </button>
      </div>

      {!canLoad ? (
        <p className="rounded-lg border border-dashed border-subtle p-6 text-sm text-muted">
          Select an organization to load transformations.
        </p>
      ) : listQuery.isLoading ? (
        <p className="rounded-lg border border-subtle p-6 text-sm text-muted">Loading…</p>
      ) : listQuery.error ? (
        <p className="rounded-lg border border-red-500/40 bg-red-500/10 p-6 text-sm text-red-500">
          {(listQuery.error as Error).message}
        </p>
      ) : (
        <TransformationListTable
          transformations={rows}
          onDelete={handleDelete}
          onOpen={(id) =>
            navigate({
              to: '/transformations/$transformationId',
              params: { transformationId: id },
            })
          }
        />
      )}
    </div>
  )
}
