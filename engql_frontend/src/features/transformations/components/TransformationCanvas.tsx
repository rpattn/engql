import { MutableRefObject, useMemo } from 'react'

import ReactFlow, { Background, Controls } from 'reactflow'

import type { TransformationCanvasNode } from '../types'
import type { TransformationGraphController } from '../hooks/useTransformationGraph'
import { TransformationNode } from './TransformationNode'

import 'reactflow/dist/style.css'

export function TransformationCanvas({
  controller,
  onSelect,
  onDeselect,
  selectedNodeId,
  preserveSelectionRef,
  onBackgroundPointerDown,
}: {
  controller: TransformationGraphController
  onSelect: (node: TransformationCanvasNode | null) => void
  onDeselect: () => void
  selectedNodeId: string | null
  preserveSelectionRef: MutableRefObject<boolean>
  onBackgroundPointerDown: () => void
}) {
  const nodeTypes = useMemo(
    () => ({
      transformation: TransformationNode,
    }),
    [],
  )

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
          nodes={controller.graph.nodes}
          edges={controller.graph.edges}
          nodeTypes={nodeTypes}
          onNodesChange={controller.onNodesChange}
          onEdgesChange={controller.onEdgesChange}
          onConnect={controller.onConnect}
          onPaneMouseDown={onBackgroundPointerDown}
          onPaneClick={onDeselect}
          onSelectionChange={(changes) => {
            const shouldPreserveSelection = preserveSelectionRef.current
            const candidates = changes.nodes ?? []
            let next: typeof candidates[number] | null = null

            for (let index = candidates.length - 1; index >= 0; index -= 1) {
              const candidate = candidates[index]
              if (candidate.selected) {
                next = candidate
                break
              }
            }

            const nextId = next?.id ?? null

            if (nextId) {
              if (nextId === selectedNodeId) {
                return
              }

              onSelect(next as TransformationCanvasNode | null)
              return
            }

            if (shouldPreserveSelection) {
              // React Flow emitted a selection reset for the existing node (for example
              // after the graph re-renders). Keep our explicit selection state so the
              // inspector stays open.
              return
            }

            onSelect(null)
          }}
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
