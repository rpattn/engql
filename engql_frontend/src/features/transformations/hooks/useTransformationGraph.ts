import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type {
  Connection,
  EdgeChange,
  NodeChange,
} from 'reactflow'
import {
  applyEdgeChanges,
  applyNodeChanges,
} from 'reactflow'
import useUndo from 'use-undo'

import type {
  TransformationCanvasEdge,
  TransformationCanvasNode,
  TransformationGraphState,
  TransformationNodeData,
} from '../types'
import { EntityTransformationNodeType } from '@/generated/graphql'
import { formatNodeType } from '../utils/format'
import {
  diffNodeAliases,
  getNodePrimaryAlias,
  replaceAliasInNode,
  type AliasChange,
} from '../utils/alias'
import { buildEdgeId, createNewNode, serializeGraph } from '../utils/nodes'
import { wouldIntroduceCycle } from '../utils/topology'

const MAX_INPUTS_BY_TYPE: Partial<Record<EntityTransformationNodeType, number>> = {
  [EntityTransformationNodeType.Load]: 0,
  [EntityTransformationNodeType.Filter]: 1,
  [EntityTransformationNodeType.Project]: 1,
  [EntityTransformationNodeType.Sort]: 1,
  [EntityTransformationNodeType.Paginate]: 1,
  [EntityTransformationNodeType.Join]: 2,
  [EntityTransformationNodeType.LeftJoin]: 2,
  [EntityTransformationNodeType.AntiJoin]: 2,
}

function getMaxInputsForType(type: EntityTransformationNodeType) {
  return MAX_INPUTS_BY_TYPE[type] ?? null
}

function createEdgeId(
  sourceId: string,
  targetId: string,
  edges: TransformationCanvasEdge[],
) {
  const existingForTarget = edges.filter((edge) => edge.target === targetId)
  const usedIds = new Set(existingForTarget.map((edge) => edge.id))

  let index = existingForTarget.length
  let candidate = buildEdgeId(sourceId, targetId, index)

  while (usedIds.has(candidate)) {
    index += 1
    candidate = buildEdgeId(sourceId, targetId, index)
  }

  return candidate
}

function applyAliasFromConnection(
  node: TransformationCanvasNode,
  alias: string,
  existingIncoming: number,
): TransformationCanvasNode {
  const { data } = node
  const { config } = data

  const updateNode = (updatedConfig: TransformationNodeData['config']) => ({
    ...node,
    data: {
      ...data,
      config: updatedConfig,
    },
  })

  switch (data.type) {
    case EntityTransformationNodeType.Filter: {
      if (!config.filter) {
        return node
      }
      if (config.filter.alias === alias) {
        return node
      }
      return updateNode({
        ...config,
        filter: { ...config.filter, alias },
      })
    }
    case EntityTransformationNodeType.Project: {
      if (!config.project) {
        return node
      }
      if (config.project.alias === alias) {
        return node
      }
      return updateNode({
        ...config,
        project: { ...config.project, alias },
      })
    }
    case EntityTransformationNodeType.Sort: {
      if (!config.sort) {
        return node
      }
      if (config.sort.alias === alias) {
        return node
      }
      return updateNode({
        ...config,
        sort: { ...config.sort, alias },
      })
    }
    case EntityTransformationNodeType.Join:
    case EntityTransformationNodeType.LeftJoin:
    case EntityTransformationNodeType.AntiJoin: {
      if (!config.join) {
        return node
      }

      const nextJoin = { ...config.join }

      if (existingIncoming === 0) {
        if (nextJoin.leftAlias === alias) {
          return node
        }
        nextJoin.leftAlias = alias
      } else {
        if (nextJoin.rightAlias === alias) {
          return node
        }
        nextJoin.rightAlias = alias
      }

      return updateNode({
        ...config,
        join: nextJoin,
      })
    }
    default:
      return node
  }
}

export type TransformationGraphController = {
  graph: TransformationGraphState
  setGraph: (updater: (graph: TransformationGraphState) => TransformationGraphState) => void
  addNode: (type: TransformationCanvasNode['data']['type']) => TransformationCanvasNode
  removeNode: (nodeId: string) => void
  updateNode: (
    nodeId: string,
    updater: (node: TransformationCanvasNode) => TransformationCanvasNode,
  ) => void
  onNodesChange: (changes: NodeChange<TransformationNodeData>[]) => void
  onEdgesChange: (changes: EdgeChange<TransformationNodeData>[]) => void
  onConnect: (connection: Connection) => void
  undo: () => void
  redo: () => void
  canUndo: boolean
  canRedo: boolean
  error: string | null
  clearError: () => void
  serialize: () => ReturnType<typeof serializeGraph>
}

export function useTransformationGraph(
  initialGraph: TransformationGraphState,
): TransformationGraphController {
  const [history, { set: setHistory, reset, undo, redo, canUndo, canRedo }] =
    useUndo(initialGraph)
  const { present } = history
  const [error, setError] = useState<string | null>(null)

  const resetRef = useRef(reset)
  const initialGraphSignatureRef = useRef<string | null>(null)
  const latestGraphSignatureRef = useRef(JSON.stringify(initialGraph))

  useEffect(() => {
    resetRef.current = reset
  }, [reset])

  useEffect(() => {
    latestGraphSignatureRef.current = JSON.stringify(present)
  }, [present])

  useEffect(() => {
    const nextSignature = JSON.stringify(initialGraph)

    if (initialGraphSignatureRef.current === nextSignature) {
      return
    }

    initialGraphSignatureRef.current = nextSignature

    if (latestGraphSignatureRef.current === nextSignature) {
      setError(null)
      return
    }

    resetRef.current(initialGraph)
    setError(null)
  }, [initialGraph])

  const setGraph = useCallback(
    (updater: (graph: TransformationGraphState) => TransformationGraphState) => {
      setHistory(updater(present))
    },
    [present, setHistory],
  )

  const addNode = useCallback(
    (type: TransformationCanvasNode['data']['type']) => {
      let created: TransformationCanvasNode | null = null

      setGraph((current) => {
        const node = createNewNode(type, current.nodes.length)
        created = node
        return {
          nodes: [...current.nodes, node],
          edges: current.edges,
        }
      })

      setError(null)

      if (!created) {
        throw new Error('Failed to create node')
      }

      return created
    },
    [setGraph],
  )

  const removeNode = useCallback(
    (nodeId: string) => {
      setGraph((current) => ({
        nodes: current.nodes.filter((node) => node.id !== nodeId),
        edges: current.edges.filter(
          (edge) => edge.source !== nodeId && edge.target !== nodeId,
        ),
      }))
    },
    [setGraph],
  )

  const updateNode = useCallback(
    (
      nodeId: string,
      updater: (node: TransformationCanvasNode) => TransformationCanvasNode,
    ) => {
      setGraph((current) => {
        const adjacency = new Map<string, string[]>()
        for (const edge of current.edges) {
          if (!adjacency.has(edge.source)) {
            adjacency.set(edge.source, [])
          }
          adjacency.get(edge.source)!.push(edge.target)
        }

        const nextNodes = [...current.nodes]
        let aliasChanges: AliasChange[] = []

        for (let index = 0; index < nextNodes.length; index += 1) {
          const node = nextNodes[index]
          if (node.id !== nodeId) {
            continue
          }

          const updated = updater(node)
          aliasChanges = diffNodeAliases(node, updated)
          nextNodes[index] = updated
          break
        }

        if (aliasChanges.length > 0) {
          const queue = aliasChanges.map((change) => ({
            sourceId: nodeId,
            oldAlias: change.oldAlias,
            newAlias: change.newAlias,
          }))
          const visited = new Set<string>()

          while (queue.length > 0) {
            const currentItem = queue.shift()!
            const key = `${currentItem.sourceId}:${currentItem.oldAlias}->${currentItem.newAlias}`
            if (visited.has(key)) {
              continue
            }
            visited.add(key)

            const targets = adjacency.get(currentItem.sourceId) ?? []
            for (const targetId of targets) {
              const targetIndex = nextNodes.findIndex((node) => node.id === targetId)
              if (targetIndex === -1) {
                continue
              }

              const targetNode = nextNodes[targetIndex]
              const updatedTarget = replaceAliasInNode(
                targetNode,
                currentItem.oldAlias,
                currentItem.newAlias,
              )

              if (updatedTarget !== targetNode) {
                nextNodes[targetIndex] = updatedTarget
                queue.push({
                  sourceId: targetId,
                  oldAlias: currentItem.oldAlias,
                  newAlias: currentItem.newAlias,
                })
              }
            }
          }
        }

        return {
          nodes: nextNodes,
          edges: current.edges,
        }
      })
    },
    [setGraph],
  )

  const onNodesChange = useCallback(
    (changes: NodeChange<TransformationNodeData>[]) => {
      if (!changes.length) return
      setGraph((current) => ({
        nodes: applyNodeChanges(changes, current.nodes),
        edges: current.edges,
      }))
    },
    [setGraph],
  )

  const onEdgesChange = useCallback(
    (changes: EdgeChange<TransformationNodeData>[]) => {
      if (!changes.length) return
      setGraph((current) => ({
        nodes: current.nodes,
        edges: applyEdgeChanges(changes, current.edges),
      }))
    },
    [setGraph],
  )

  const onConnect = useCallback(
    (connection: Connection) => {
      const sourceId = connection.source
      const targetId = connection.target

      if (!sourceId || !targetId) {
        return
      }

      setGraph((current) => {
        const targetNode = current.nodes.find((node) => node.id === targetId)
        if (!targetNode) {
          return current
        }

        const sourceNode = current.nodes.find((node) => node.id === sourceId)
        const sourceAlias = sourceNode ? getNodePrimaryAlias(sourceNode) : null

        const incoming = current.edges.filter((edge) => edge.target === targetId)

        if (incoming.some((edge) => edge.source === sourceId)) {
          setError('These nodes are already connected.')
          return current
        }

        const maxInputs = getMaxInputsForType(targetNode.data.type)
        if (maxInputs !== null && incoming.length >= maxInputs) {
          const label = formatNodeType(targetNode.data.type)
          if (maxInputs === 0) {
            setError(`${label} nodes cannot receive incoming connections.`)
          } else {
            setError(`${label} nodes can only accept ${maxInputs} input${maxInputs === 1 ? '' : 's'}.`)
          }
          return current
        }

        if (wouldIntroduceCycle(current.nodes, current.edges, sourceId, targetId)) {
          setError('This connection would create a cycle. Try a different target.')
          return current
        }

        const edgeId = createEdgeId(sourceId, targetId, current.edges)

        const nextEdge: TransformationCanvasEdge = {
          id: edgeId,
          source: sourceId,
          target: targetId,
          animated: false,
          type: connection.type ?? 'default',
        }

        setError(null)

        let nextNodes = current.nodes
        let aliasChanges: AliasChange[] = []

        if (sourceAlias) {
          nextNodes = current.nodes.map((node) => {
            if (node.id !== targetId) {
              return node
            }

            const updated = applyAliasFromConnection(node, sourceAlias, incoming.length)
            aliasChanges = diffNodeAliases(node, updated)
            return updated
          })
        }

        if (aliasChanges.length) {
          const adjacency = new Map<string, string[]>()
          for (const edge of current.edges) {
            if (!adjacency.has(edge.source)) {
              adjacency.set(edge.source, [])
            }
            adjacency.get(edge.source)!.push(edge.target)
          }

          const queue = aliasChanges.map((change) => ({
            sourceId: targetId,
            oldAlias: change.oldAlias,
            newAlias: change.newAlias,
          }))
          const visited = new Set<string>()

          while (queue.length) {
            const currentItem = queue.shift()!
            const key = `${currentItem.sourceId}:${currentItem.oldAlias}->${currentItem.newAlias}`
            if (visited.has(key)) {
              continue
            }
            visited.add(key)

            const targets = adjacency.get(currentItem.sourceId) ?? []
            for (const downstreamId of targets) {
              const index = nextNodes.findIndex((node) => node.id === downstreamId)
              if (index === -1) {
                continue
              }

              const existingNode = nextNodes[index]
              const updatedNode = replaceAliasInNode(
                existingNode,
                currentItem.oldAlias,
                currentItem.newAlias,
              )

              if (updatedNode !== existingNode) {
                nextNodes[index] = updatedNode
                queue.push({
                  sourceId: downstreamId,
                  oldAlias: currentItem.oldAlias,
                  newAlias: currentItem.newAlias,
                })
              }
            }
          }
        }

        return {
          nodes: nextNodes,
          edges: [...current.edges, nextEdge],
        }
      })
    },
    [setGraph],
  )

  const serialize = useCallback(() => serializeGraph(present), [present])

  const controller: TransformationGraphController = useMemo(
    () => ({
      graph: present,
      setGraph,
      addNode,
      removeNode,
      updateNode,
      onNodesChange,
      onEdgesChange,
      onConnect,
      undo,
      redo,
      canUndo,
      canRedo,
      error,
      clearError: () => setError(null),
      serialize,
    }),
    [
      present,
      setGraph,
      addNode,
      removeNode,
      updateNode,
      onNodesChange,
      onEdgesChange,
      onConnect,
      undo,
      redo,
      canUndo,
      canRedo,
      error,
      serialize,
    ],
  )

  return controller
}
