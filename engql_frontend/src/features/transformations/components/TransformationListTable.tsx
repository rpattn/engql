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
      <div className="rounded-lg border border-dashed border-subtle p-8 text-center text-sm text-muted">
        No transformations found for this organization yet. Create one to get started.
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border border-subtle">
      <table className="min-w-full divide-y divide-subtle text-left text-sm">
        <thead className="bg-subtle text-xs font-semibold uppercase tracking-wide text-muted">
          <tr>
            <th className="px-4 py-2">Name</th>
            <th className="px-4 py-2">Description</th>
            <th className="px-4 py-2">Nodes</th>
            <th className="px-4 py-2">Updated</th>
            <th className="px-4 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-subtle/60 bg-surface">
          {transformations.map((transformation) => (
            <tr key={transformation.id} className="transition hover:bg-subtle">
              <td className="px-4 py-3 font-medium">
                <Link
                  to="/transformations/$transformationId"
                  params={{ transformationId: transformation.id }}
                  className="text-blue-600 transition hover:text-blue-400 hover:underline"
                >
                  {transformation.name}
                </Link>
              </td>
              <td className="px-4 py-3 text-muted">
                {transformation.description ?? '—'}
              </td>
              <td className="px-4 py-3 text-muted">
                {transformation.nodes.length}
              </td>
              <td className="px-4 py-3 text-muted">
                {new Date(transformation.updatedAt).toLocaleString()}
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center justify-end gap-2 text-xs">
                  <Link
                    to="/transformations/$transformationId/execute"
                    params={{ transformationId: transformation.id }}
                    className="rounded border border-blue-500/40 px-2 py-1 text-blue-500 transition hover:border-blue-500 hover:bg-blue-500/10"
                  >
                    Execute
                  </Link>
                  <Link
                    to="/transformations/$transformationId/results"
                    params={{ transformationId: transformation.id }}
                    className="rounded border border-emerald-500/40 px-2 py-1 text-emerald-400 transition hover:border-emerald-500 hover:bg-emerald-500/10"
                  >
                    Results
                  </Link>
                  <button
                    type="button"
                    onClick={() => onOpen(transformation.id)}
                    className="rounded border border-subtle px-2 py-1 text-muted transition hover:border-blue-500/60 hover:text-blue-500"
                  >
                    Open
                  </button>
                  <button
                    type="button"
                    onClick={() => onDelete(transformation.id)}
                    className="rounded border border-red-500/40 px-2 py-1 text-red-500 transition hover:border-red-500/60 hover:bg-red-500/10"
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
