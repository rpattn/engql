import { Link } from '@tanstack/react-router'

import type { EntityTransformationsQuery } from '@/generated/graphql'

export function TransformationListTable({
  transformations,
  onDelete,
  onOpen,
}: {
  transformations: EntityTransformationsQuery['entityTransformations']
  onDelete: (id: string) => void
  onOpen: (id: string) => void
}) {
  if (!transformations.length) {
    return (
      <div className="rounded border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500">
        No transformations found for this organization yet. Create one to get started.
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded border border-slate-200">
      <table className="min-w-full divide-y divide-slate-200 text-left text-sm">
        <thead className="bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">Description</th>
            <th className="px-4 py-2">Nodes</th>
            <th className="px-4 py-2">Updated</th>
            <th className="px-4 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white">
          {transformations.map((transformation) => (
            <tr key={transformation.id} className="hover:bg-slate-50">
              <td className="px-4 py-3 font-medium text-slate-900">
                <Link
                  to="/transformations/$transformationId"
                  params={{ transformationId: transformation.id }}
                  className="text-blue-600 hover:underline"
                >
                  {transformation.name}
                </Link>
              </td>
              <td className="px-4 py-3 text-slate-600">
                {transformation.description ?? '—'}
              </td>
              <td className="px-4 py-3 text-slate-600">
                {transformation.nodes.length}
              </td>
              <td className="px-4 py-3 text-slate-600">
                {new Date(transformation.updatedAt).toLocaleString()}
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center justify-end gap-2 text-xs">
                  <Link
                    to="/transformations/$transformationId/execute"
                    params={{ transformationId: transformation.id }}
                    className="rounded border border-blue-200 px-2 py-1 text-blue-600 hover:bg-blue-50"
                  >
                    Execute
                  </Link>
                  <button
                    type="button"
                    onClick={() => onOpen(transformation.id)}
                    className="rounded border border-slate-200 px-2 py-1 text-slate-600 hover:bg-slate-100"
                  >
                    Open
                  </button>
                  <button
                    type="button"
                    onClick={() => onDelete(transformation.id)}
                    className="rounded border border-rose-200 px-2 py-1 text-rose-600 hover:bg-rose-50"
                  >
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
