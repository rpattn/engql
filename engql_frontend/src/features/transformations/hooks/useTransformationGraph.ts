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

  useEffect(() => {
    resetRef.current = reset
  }, [reset])

  useEffect(() => {
    const nextSignature = JSON.stringify(initialGraph)

    if (initialGraphSignatureRef.current === nextSignature) {
      return
    }

    initialGraphSignatureRef.current = nextSignature

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
      setGraph((current) => ({
        nodes: current.nodes.map((node) => (node.id === nodeId ? updater(node) : node)),
        edges: current.edges,
      }))
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

        return {
          nodes: current.nodes,
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
