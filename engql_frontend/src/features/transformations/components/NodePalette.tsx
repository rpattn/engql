import { EntityTransformationNodeType } from '@/generated/graphql'

const paletteItems: Array<{
  type: EntityTransformationNodeType
  title: string
  description: string
}> = [
  {
    type: EntityTransformationNodeType.Load,
    title: 'Load',
    description: 'Bootstrap the DAG with an entity type and optional filters.',
  },
  {
    type: EntityTransformationNodeType.Filter,
    title: 'Filter',
    description: 'Narrow the stream using property filters.',
  },
  {
    type: EntityTransformationNodeType.Project,
    title: 'Project',
    description: 'Select a subset of fields to expose downstream.',
  },
  {
    type: EntityTransformationNodeType.Join,
    title: 'Join',
    description: 'Join two inputs on a shared field.',
  },
  {
    type: EntityTransformationNodeType.LeftJoin,
    title: 'Left Join',
    description: 'Left join two inputs, preserving the left side.',
  },
  {
    type: EntityTransformationNodeType.AntiJoin,
    title: 'Anti Join',
    description: 'Emit left records that are missing matches on the right.',
  },
  {
    type: EntityTransformationNodeType.Union,
    title: 'Union',
    description: 'Merge multiple compatible inputs into a single stream.',
  },
  {
    type: EntityTransformationNodeType.Sort,
    title: 'Sort',
    description: 'Order entities using alias, field, and direction.',
  },
  {
    type: EntityTransformationNodeType.Materialize,
    title: 'Materialize',
    description: 'Flatten entities into a table for downstream consumers.',
  },
  {
    type: EntityTransformationNodeType.Paginate,
    title: 'Paginate',
    description: 'Control page size and offset.',
  },
]

export function NodePalette({
  onAdd,
}: {
  onAdd: (type: EntityTransformationNodeType) => void
}) {
  return (
    <aside className="space-y-4">
      <h3 className="text-sm font-semibold text-slate-600">Node palette</h3>
      <div className="space-y-3">
        {paletteItems.map((item) => (
          <button
            key={item.type}
            type="button"
            onClick={() => onAdd(item.type)}
            className="w-full rounded-md border border-slate-200 bg-white p-3 text-left shadow-sm transition hover:border-blue-500 hover:shadow"
          >
            <div className="font-medium text-slate-900">{item.title}</div>
            <p className="mt-1 text-xs text-slate-500">{item.description}</p>
          </button>
        ))}
      </div>
    </aside>
  )
}
