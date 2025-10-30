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
        <h2 className="text-base font-semibold text-slate-800">Catalog</h2>
        <button
          type="button"
          onClick={handleCreate}
          className="rounded bg-blue-600 px-3 py-1 text-xs font-semibold text-white disabled:bg-blue-300"
          disabled={!trimmedOrgId || createMutation.isPending}
        >
          {createMutation.isPending ? 'Creating…' : 'New transformation'}
        </button>
      </div>

      {!canLoad ? (
        <p className="rounded border border-dashed border-slate-300 p-6 text-sm text-slate-500">
          Select an organization to load transformations.
        </p>
      ) : listQuery.isLoading ? (
        <p className="rounded border border-slate-200 p-6 text-sm text-slate-500">Loading…</p>
      ) : listQuery.error ? (
        <p className="rounded border border-rose-300 bg-rose-50 p-6 text-sm text-rose-700">
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
