import { useMemo, useCallback } from 'react'
import {
  Handle,
  Position,
  type NodeProps,
  type ReactFlowState,
  useStore,
} from 'reactflow'
import { shallow } from 'zustand/shallow'

import { EntityTransformationNodeType } from '@/generated/graphql'

import type { TransformationNodeData } from '../types'
import { formatNodeType } from '../utils/format'

const typeAccentColor: Record<EntityTransformationNodeType, string> = {
  [EntityTransformationNodeType.Load]: '#0ea5e9',
  [EntityTransformationNodeType.Filter]: '#22c55e',
  [EntityTransformationNodeType.Project]: '#8b5cf6',
  [EntityTransformationNodeType.Join]: '#f97316',
  [EntityTransformationNodeType.LeftJoin]: '#f97316',
  [EntityTransformationNodeType.AntiJoin]: '#f97316',
  [EntityTransformationNodeType.Sort]: '#eab308',
  [EntityTransformationNodeType.Paginate]: '#6366f1',
  [EntityTransformationNodeType.Union]: '#06b6d4',
}

const countsSelector = (
  nodeId: string,
): ((state: ReactFlowState) => { incoming: number; outgoing: number }) => {
  return (state) => {
    let incoming = 0
    let outgoing = 0

    for (const edge of state.edges) {
      if (edge.target === nodeId) {
        incoming += 1
      }
      if (edge.source === nodeId) {
        outgoing += 1
      }
    }

    return { incoming, outgoing }
  }
}

export function TransformationNode({ id, data, selected }: NodeProps<TransformationNodeData>) {
  const counts = useStore(useCallback(countsSelector(id), [id]), shallow)

  const accentColor = typeAccentColor[data.type] ?? '#334155'
  const typeLabel = useMemo(() => formatNodeType(data.type), [data.type])
  const configSummary = useMemo(() => summarizeConfiguration(data), [data])

  return (
    <div
      className={`group/node relative min-w-[220px] rounded-lg border bg-white shadow-sm transition-shadow ${
        selected ? 'border-blue-400 shadow-lg' : 'border-slate-200'
      }`}
    >
      <Handle
        id="input"
        type="target"
        position={Position.Left}
        className="!h-3 !w-3 !border-2 !border-white !bg-slate-400"
      />
      <Handle
        id="output"
        type="source"
        position={Position.Right}
        className="!h-3 !w-3 !border-2 !border-white !bg-slate-400"
      />
      <div
        className="rounded-t-lg px-3 py-2 text-xs font-semibold uppercase tracking-wide text-white"
        style={{ backgroundColor: accentColor }}
      >
        {typeLabel}
      </div>
      <div className="space-y-2 px-3 py-3">
        <div>
          <p className="text-sm font-semibold text-slate-800">{data.name}</p>
          {data.validationMessage && (
            <p className="mt-1 rounded border border-amber-300 bg-amber-50 px-2 py-1 text-[10px] font-medium text-amber-800">
              {data.validationMessage}
            </p>
          )}
        </div>

        {configSummary.length ? (
          <ul className="space-y-1 text-xs text-slate-600">
            {configSummary.map((item) => (
              <li key={item.label} className="flex items-start justify-between gap-2">
                <span className="font-medium text-slate-500">{item.label}</span>
                <span className="text-right text-slate-700">{item.value}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-xs text-slate-500">No additional configuration.</p>
        )}
      </div>
      <footer className="flex items-center justify-between border-t border-slate-200 px-3 py-2 text-[11px] font-medium text-slate-600">
        <span>Inputs: {counts.incoming}</span>
        <span>Outputs: {counts.outgoing}</span>
      </footer>
    </div>
  )
}

function summarizeConfiguration(data: TransformationNodeData) {
  const summary: { label: string; value: string }[] = []
  const { config } = data

  switch (data.type) {
    case EntityTransformationNodeType.Load: {
      const alias = config.load?.alias?.trim()
      const entity = config.load?.entityType?.trim()
      const filters = config.load?.filters?.length ?? 0
      if (alias) summary.push({ label: 'Alias', value: alias })
      if (entity) summary.push({ label: 'Entity', value: entity })
      if (filters) {
        summary.push({ label: 'Filters', value: `${filters}` })
      }
      break
    }
    case EntityTransformationNodeType.Filter: {
      const alias = config.filter?.alias?.trim()
      const filters = config.filter?.filters?.length ?? 0
      if (alias) summary.push({ label: 'Alias', value: alias })
      if (filters) summary.push({ label: 'Conditions', value: `${filters}` })
      break
    }
    case EntityTransformationNodeType.Project: {
      const alias = config.project?.alias?.trim()
      const fields = config.project?.fields?.length ?? 0
      if (alias) summary.push({ label: 'Alias', value: alias })
      if (fields) summary.push({ label: 'Fields', value: `${fields}` })
      break
    }
    case EntityTransformationNodeType.Join:
    case EntityTransformationNodeType.LeftJoin:
    case EntityTransformationNodeType.AntiJoin: {
      const leftAlias = config.join?.leftAlias?.trim()
      const rightAlias = config.join?.rightAlias?.trim()
      const onField = config.join?.onField?.trim()
      if (leftAlias) summary.push({ label: 'Left alias', value: leftAlias })
      if (rightAlias) summary.push({ label: 'Right alias', value: rightAlias })
      if (onField) summary.push({ label: 'On field', value: onField })
      break
    }
    case EntityTransformationNodeType.Sort: {
      const alias = config.sort?.alias?.trim()
      const field = config.sort?.field?.trim()
      const direction = config.sort?.direction
      if (alias) summary.push({ label: 'Alias', value: alias })
      if (field) summary.push({ label: 'Field', value: field })
      if (direction) summary.push({ label: 'Direction', value: direction })
      break
    }
    case EntityTransformationNodeType.Paginate: {
      const limit = config.paginate?.limit
      const offset = config.paginate?.offset
      if (typeof limit === 'number') summary.push({ label: 'Limit', value: `${limit}` })
      if (typeof offset === 'number') summary.push({ label: 'Offset', value: `${offset}` })
      break
    }
    case EntityTransformationNodeType.Union: {
      break
    }
    default:
      break
  }

  return summary
}
