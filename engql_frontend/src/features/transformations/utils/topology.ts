import type {
  TransformationCanvasEdge,
  TransformationCanvasNode,
} from '../types'

export function topologicalSort(
  nodes: TransformationCanvasNode[],
  edges: TransformationCanvasEdge[],
) {
  const order = new Map(nodes.map((node, index) => [node.id, index]))
  const inDegree = new Map<string, number>()
  const adjacency = new Map<string, Set<string>>()

  for (const node of nodes) {
    inDegree.set(node.id, 0)
    adjacency.set(node.id, new Set())
  }

  for (const edge of edges) {
    if (!adjacency.has(edge.source) || !inDegree.has(edge.target)) {
      continue
    }

    adjacency.get(edge.source)!.add(edge.target)
    inDegree.set(edge.target, (inDegree.get(edge.target) ?? 0) + 1)
  }

  const queue = nodes
    .filter((node) => (inDegree.get(node.id) ?? 0) === 0)
    .sort((a, b) => (order.get(a.id)! - order.get(b.id)!))

  const sorted: TransformationCanvasNode[] = []

  while (queue.length > 0) {
    const node = queue.shift()!
    sorted.push(node)

    for (const neighbor of adjacency.get(node.id) ?? []) {
      const nextInDegree = (inDegree.get(neighbor) ?? 0) - 1
      inDegree.set(neighbor, nextInDegree)

      if (nextInDegree === 0) {
        const neighborNode = nodes.find((candidate) => candidate.id === neighbor)
        if (neighborNode) {
          queue.push(neighborNode)
          queue.sort((a, b) => (order.get(a.id)! - order.get(b.id)!))
        }
      }
    }
  }

  if (sorted.length !== nodes.length) {
    throw new Error('Cycle detected in transformation graph')
  }

  return sorted
}

export function wouldIntroduceCycle(
  nodes: TransformationCanvasNode[],
  edges: TransformationCanvasEdge[],
  sourceId: string,
  targetId: string,
) {
  if (sourceId === targetId) {
    return true
  }

  const adjacency = new Map<string, Set<string>>()
  for (const node of nodes) {
    adjacency.set(node.id, new Set())
  }

  for (const edge of edges) {
    if (!adjacency.has(edge.source)) {
      adjacency.set(edge.source, new Set())
    }
    adjacency.get(edge.source)!.add(edge.target)
  }

  // include the proposed edge
  if (!adjacency.has(sourceId)) {
    adjacency.set(sourceId, new Set())
  }
  adjacency.get(sourceId)!.add(targetId)

  const stack = [targetId]
  const visited = new Set<string>()

  while (stack.length) {
    const current = stack.pop()!
    if (current === sourceId) {
      return true
    }

    if (visited.has(current)) {
      continue
    }
    visited.add(current)

    for (const neighbor of adjacency.get(current) ?? []) {
      stack.push(neighbor)
    }
  }

  return false
}

export function incomingFor(
  edges: TransformationCanvasEdge[],
  nodeId: string,
) {
  return edges.filter((edge) => edge.target === nodeId).map((edge) => edge.source)
}
