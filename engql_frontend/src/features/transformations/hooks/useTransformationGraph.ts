import { useCallback, useEffect, useMemo, useState } from 'react'
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
import { createNewNode, serializeGraph } from '../utils/nodes'
import { wouldIntroduceCycle } from '../utils/topology'

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

  useEffect(() => {
    reset(initialGraph)
    setError(null)
  }, [initialGraph, reset])

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
      if (!connection.source || !connection.target) {
        return
      }

      setGraph((current) => {
        if (
          wouldIntroduceCycle(
            current.nodes,
            current.edges,
            connection.source!,
            connection.target!,
          )
        ) {
          setError('This connection would create a cycle. Try a different target.')
          return current
        }

        const edgeId = `${connection.source}-${connection.target}`
        if (current.edges.some((edge) => edge.id === edgeId)) {
          return current
        }

        const nextEdge: TransformationCanvasEdge = {
          id: edgeId,
          source: connection.source!,
          target: connection.target!,
          animated: false,
          type: connection.type ?? 'default',
        }

        return {
          nodes: current.nodes,
          edges: [...current.edges, nextEdge],
        }
      })
      setError(null)
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
