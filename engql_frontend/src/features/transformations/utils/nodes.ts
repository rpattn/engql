import type {
  EntityTransformationNode,
  EntityTransformationNodeInput,
  EntityTransformationNodeType,
  PropertyFilter,
  PropertyFilterConfig,
} from '@/generated/graphql'
import { EntityTransformationNodeType as NodeType } from '@/generated/graphql'
import type {
  TransformationCanvasNode,
  TransformationGraphState,
  TransformationNodeConfig,
  TransformationNodeData,
  PropertyFilterValue,
} from '../types'
import { topologicalSort } from './topology'

export function buildEdgeId(sourceId: string, targetId: string, index: number) {
  return `edge-${sourceId}-${targetId}-${index}`
}

let counter = 0

export function createNodeId(prefix = 'node') {
  counter += 1
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  return `${prefix}-${Date.now()}-${counter}`
}

export function createNewNode(
  type: EntityTransformationNodeType,
  positionIndex: number,
): TransformationCanvasNode {
  const id = createNodeId('node')
  const baseName = type.replace('_', ' ').toLowerCase()
  const data: TransformationNodeData = {
    name: `${capitalize(baseName)} ${counter}`,
    type,
    config: buildDefaultConfig(type),
  }

  return {
    id,
    type: 'transformation',
    data,
    position: derivePosition(positionIndex),
  }
}

export function buildDefaultConfig(
  type: EntityTransformationNodeType,
): TransformationNodeConfig {
  switch (type) {
    case NodeType.Load:
      return {
        load: { alias: 'source', entityType: '', filters: [] },
      }
    case NodeType.Filter:
      return {
        filter: { alias: 'filtered', filters: [] },
      }
    case NodeType.Project:
      return {
        project: { alias: 'projection', fields: [] },
      }
    case NodeType.Join:
    case NodeType.LeftJoin:
    case NodeType.AntiJoin:
      return {
        join: { leftAlias: 'left', rightAlias: 'right', onField: '' },
      }
    case NodeType.Sort:
      return {
        sort: { alias: 'sorted', field: '', direction: 'ASC' },
      }
    case NodeType.Paginate:
      return {
        paginate: { limit: 25, offset: 0 },
      }
    case NodeType.Union:
    default:
      return {}
  }
}

export function createGraphStateFromDefinition(
  definitionNodes: EntityTransformationNode[],
): TransformationGraphState {
  const nodes = definitionNodes.map((node, index) => {
    const config: TransformationNodeConfig = {
      load: node.load ? { ...node.load, filters: node.load.filters.map(cloneFilter) } : undefined,
      filter: node.filter
        ? { ...node.filter, filters: node.filter.filters.map(cloneFilter) }
        : undefined,
      project: node.project ? { ...node.project } : undefined,
      join: node.join ? { ...node.join } : undefined,
      sort: node.sort ? { ...node.sort } : undefined,
      paginate: node.paginate ? { ...node.paginate } : undefined,
    }

    const data: TransformationNodeData = {
      name: node.name,
      type: node.type,
      config,
      persistedId: node.id,
    }

    return {
      id: node.id,
      type: 'transformation',
      position: derivePosition(index),
      data,
    }
  })

  const edges = definitionNodes.flatMap((node) =>
    node.inputs.map((sourceId, index) => ({
      id: buildEdgeId(sourceId, node.id, index),
      source: sourceId,
      target: node.id,
      animated: false,
      type: 'default' as const,
    })),
  )

  return { nodes, edges }
}

export function serializeGraph(
  graph: TransformationGraphState,
): EntityTransformationNodeInput[] {
  const orderedNodes = topologicalSort(graph.nodes, graph.edges)

  return orderedNodes.map((node) => {
    const incoming = graph.edges
      .filter((edge) => edge.target === node.id)
      .map((edge) => edge.source)

    const payload: EntityTransformationNodeInput = {
      id: node.data.persistedId ?? undefined,
      name: node.data.name,
      type: node.data.type,
      inputs: incoming,
      load: node.data.config.load
        ? {
            ...node.data.config.load,
            filters: (node.data.config.load.filters ?? []).map(coerceFilterInput),
          }
        : undefined,
      filter: node.data.config.filter
        ? {
            ...node.data.config.filter,
            filters: (node.data.config.filter.filters ?? []).map(coerceFilterInput),
          }
        : undefined,
      project: node.data.config.project
        ? { ...node.data.config.project }
        : undefined,
      join: node.data.config.join ? { ...node.data.config.join } : undefined,
      sort: node.data.config.sort ? { ...node.data.config.sort } : undefined,
      paginate: node.data.config.paginate ? { ...node.data.config.paginate } : undefined,
    }

    return payload
  })
}

function derivePosition(index: number) {
  const column = index % 3
  const row = Math.floor(index / 3)
  return { x: column * 280, y: row * 180 }
}

function cloneFilter(filter: PropertyFilterConfig): PropertyFilterConfig {
  return {
    key: filter.key,
    value: filter.value ?? undefined,
    exists: filter.exists ?? undefined,
    inArray: filter.inArray ? [...filter.inArray] : undefined,
  }
}

function coerceFilterInput(filter: PropertyFilterValue): PropertyFilter {
  return {
    key: filter.key,
    value: filter.value ?? undefined,
    exists: filter.exists ?? undefined,
    inArray: filter.inArray ? [...filter.inArray] : undefined,
  }
}

function capitalize(text: string) {
  if (!text.length) return text
  return text.charAt(0).toUpperCase() + text.slice(1)
}
