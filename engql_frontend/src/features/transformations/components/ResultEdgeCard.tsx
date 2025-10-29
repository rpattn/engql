import type { EntityTransformationRecordEdge } from '@/generated/graphql'

function formatProperties(raw: string) {
  try {
    const parsed = JSON.parse(raw)
    return typeof parsed === 'string' ? parsed : JSON.stringify(parsed, null, 2)
  } catch {
    return raw
  }
}

export function ResultEdgeCard({ edge }: { edge: EntityTransformationRecordEdge }) {
  return (
    <div className="rounded border border-slate-200 bg-white p-3">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        Edge
      </h4>
      <div className="mt-2 grid gap-3 md:grid-cols-2">
        {edge.entities.map((node) => (
          <div key={`${node.alias}-${node.entity?.id ?? 'missing'}`} className="rounded border border-slate-100 p-2">
            <p className="text-xs font-semibold text-slate-600">{node.alias}</p>
            {node.entity ? (
              <ul className="mt-1 space-y-1 text-xs text-slate-600">
                <li>
                  <span className="font-medium">ID:</span> {node.entity.id}
                </li>
                <li>
                  <span className="font-medium">Type:</span> {node.entity.entityType}
                </li>
                <li>
                  <span className="font-medium">Path:</span> {node.entity.path}
                </li>
                {node.entity.referenceValue && (
                  <li>
                    <span className="font-medium">Reference:</span> {node.entity.referenceValue}
                  </li>
                )}
                <li>
                  <span className="font-medium">Properties:</span>
                  <pre className="mt-1 max-h-40 overflow-auto rounded border border-slate-200 bg-slate-50 p-2 text-[11px] leading-snug text-slate-600">
                    {formatProperties(node.entity.properties)}
                  </pre>
                </li>
              </ul>
            ) : (
              <p className="mt-1 text-xs text-slate-400">No entity returned.</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
