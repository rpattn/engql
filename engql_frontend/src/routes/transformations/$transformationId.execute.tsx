import { createFileRoute, useNavigate } from '@tanstack/react-router'

import { useEntityTransformationQuery } from '@/generated/graphql'
import { ExecutionRunner } from '@/features/transformations/components/ExecutionRunner'

export const Route = createFileRoute('/transformations/$transformationId.execute')({
  component: TransformationExecuteRoute,
})

function TransformationExecuteRoute() {
  const { transformationId } = Route.useParams()
  const navigate = useNavigate()

  const detailQuery = useEntityTransformationQuery({ id: transformationId })

  if (detailQuery.isLoading) {
    return (
      <p className="rounded border border-slate-200 p-6 text-sm text-slate-500">Loading…</p>
    )
  }

  if (detailQuery.error) {
    return (
      <p className="rounded border border-rose-300 bg-rose-50 p-6 text-sm text-rose-700">
        {(detailQuery.error as Error).message}
      </p>
    )
  }

  const transformation = detailQuery.data?.entityTransformation

  if (!transformation) {
    return (
      <p className="rounded border border-slate-200 p-6 text-sm text-slate-500">
        Transformation not found.
      </p>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-base font-semibold text-slate-800">
            Execute “{transformation.name}”
          </h2>
          <p className="text-xs text-slate-500">
            Updated {new Date(transformation.updatedAt).toLocaleString()}
          </p>
        </div>
        <button
          type="button"
          onClick={() =>
            navigate({
              to: '/transformations/$transformationId',
              params: { transformationId },
            })
          }
          className="rounded border border-slate-200 px-3 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100"
        >
          Back to designer
        </button>
      </div>

      <ExecutionRunner transformationId={transformationId} />
    </div>
  )
}
