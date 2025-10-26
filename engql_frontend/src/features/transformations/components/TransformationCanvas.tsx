import { useMemo } from 'react'
import ReactFlow, { Background, Controls } from 'reactflow'

import type {
  TransformationCanvasNode,
  TransformationGraphState,
} from '../types'
import type { TransformationGraphController } from '../hooks/useTransformationGraph'

import 'reactflow/dist/style.css'

export function TransformationCanvas({
  controller,
  onSelect,
  selectedNodeId,
}: {
  controller: TransformationGraphController
  onSelect: (node: TransformationCanvasNode | null) => void
  selectedNodeId: string | null
}) {
  const nodesWithSelection = useMemo(() => {
    return controller.graph.nodes.map((node) => ({
      ...node,
      selected: node.id === selectedNodeId,
    })) as TransformationGraphState['nodes']
  }, [controller.graph.nodes, selectedNodeId])

  return (
    <div className="flex h-full flex-col gap-2">
      {controller.error && (
        <div className="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          {controller.error}
          <button
            type="button"
            onClick={controller.clearError}
            className="ml-2 font-semibold underline"
          >
            Dismiss
          </button>
        </div>
      )}
      <div className="flex-1 overflow-hidden rounded border border-slate-200">
        <ReactFlow
          fitView
          nodes={nodesWithSelection}
          edges={controller.graph.edges}
          onNodesChange={controller.onNodesChange}
          onEdgesChange={controller.onEdgesChange}
          onConnect={controller.onConnect}
          onSelectionChange={(changes) => {
            const next = changes.nodes?.find((node) => node.selected) ?? null
            onSelect(next as TransformationCanvasNode | null)
          }}
          onNodeClick={(_, node) => onSelect(node as TransformationCanvasNode)}
          onPaneClick={() => onSelect(null)}
          minZoom={0.2}
          maxZoom={1.75}
        >
          <Background color="#d4d4d8" size={1.2} />
          <Controls showInteractive={false} />
        </ReactFlow>
      </div>
    </div>
  )
}
