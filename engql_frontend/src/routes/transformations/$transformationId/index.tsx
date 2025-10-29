import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import {
  useDeleteEntityTransformationMutation,
  useEntitySchemasQuery,
  useEntityTransformationQuery,
  useEntityTransformationsQuery,
  useUpdateEntityTransformationMutation,
} from '@/generated/graphql'
import { NodeInspector } from '@/features/transformations/components/NodeInspector'
import { NodePalette } from '@/features/transformations/components/NodePalette'
import { TransformationCanvas } from '@/features/transformations/components/TransformationCanvas'
import { TransformationPreviewPanel } from '@/features/transformations/components/TransformationPreviewPanel'
import { TransformationToolbar } from '@/features/transformations/components/TransformationToolbar'
import { useTransformationGraph } from '@/features/transformations/hooks/useTransformationGraph'
import {
  createGraphStateFromDefinition,
  serializeGraph,
} from '@/features/transformations/utils/nodes'
import type { TransformationAliasSummary } from '@/features/transformations/utils/preview'
import { sanitizeAlias } from '@/features/transformations/utils/alias'
import type { TransformationCanvasNode } from '@/features/transformations/types'

const AUTO_SAVE_DEBOUNCE_MS = 800

export const Route = createFileRoute('/transformations/$transformationId/')({
  component: TransformationDetailRoute,
})

function TransformationDetailRoute() {
  const { transformationId } = Route.useParams()
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const detailQuery = useEntityTransformationQuery({ id: transformationId })

  const updateMutation = useUpdateEntityTransformationMutation({
    onSuccess: (data) => {
      const updated = data.updateEntityTransformation
      queryClient.setQueryData(
        useEntityTransformationQuery.getKey({ id: transformationId }),
        { entityTransformation: updated },
      )
      queryClient.invalidateQueries({
        queryKey: useEntityTransformationsQuery.getKey({
          organizationId: updated.organizationId,
        }),
      })
    },
  })

  const deleteMutation = useDeleteEntityTransformationMutation({
    onSuccess: () => {
      const organizationId = detailQuery.data?.entityTransformation?.organizationId
      if (organizationId) {
        queryClient.invalidateQueries({
          queryKey: useEntityTransformationsQuery.getKey({ organizationId }),
        })
      }
      navigate({ to: '/transformations' })
    },
  })

  const transformation = detailQuery.data?.entityTransformation
  const organizationId = (transformation?.organizationId ?? '').trim()

  const entitySchemasQuery = useEntitySchemasQuery(
    { organizationId },
    { enabled: Boolean(organizationId) },
  )

  const initialGraph = useMemo(() => {
    if (!transformation) {
      return { nodes: [], edges: [] }
    }
    return createGraphStateFromDefinition(transformation.nodes)
  }, [transformation])

  const graphController = useTransformationGraph(initialGraph)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [previewRefreshKey, setPreviewRefreshKey] = useState(0)
  const [schemaSummaries, setSchemaSummaries] = useState<TransformationAliasSummary[]>([])
  const [isAutoSaveEnabled, setIsAutoSaveEnabled] = useState(false)
  const [baseline, setBaseline] = useState({
    name: '',
    description: '',
    graphSignature: '',
  })
  const pendingBaselineRef = useRef<typeof baseline | null>(null)
  const autoSaveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const canvasContainerRef = useRef<HTMLDivElement | null>(null)
  const inspectorRef = useRef<HTMLDivElement | null>(null)
  const preserveSelectionRef = useRef(false)

  const disableSelectionPreservation = useCallback(() => {
    preserveSelectionRef.current = false
  }, [])

  const clearSelection = useCallback(() => {
    disableSelectionPreservation()
    setSelectedNodeId(null)
  }, [disableSelectionPreservation])

  const selectNodeById = useCallback((nodeId: string) => {
    preserveSelectionRef.current = true
    setSelectedNodeId(nodeId)
  }, [])

  const handleCanvasSelect = useCallback(
    (node: TransformationCanvasNode | null) => {
      if (node) {
        selectNodeById(node.id)
        return
      }

      clearSelection()
    },
    [clearSelection, selectNodeById],
  )

  const { onNodesChange } = graphController

  const handleCanvasBackgroundPointerDown = useCallback(() => {
    disableSelectionPreservation()
  }, [disableSelectionPreservation])

  const handleCanvasDeselect = useCallback(() => {
    const previousSelection = selectedNodeId

    clearSelection()

    if (previousSelection) {
      onNodesChange([
        { id: previousSelection, type: 'select', selected: false },
      ])
    }
  }, [clearSelection, onNodesChange, selectedNodeId])

  useEffect(() => {
    if (transformation) {
      setName(transformation.name)
      setDescription(transformation.description ?? '')
      clearSelection()
    }
  }, [clearSelection, transformation?.id])

  const initialGraphSignature = useMemo(() => {
    return JSON.stringify(serializeGraph(initialGraph))
  }, [initialGraph])

  const currentGraphSignature = useMemo(() => {
    return JSON.stringify(graphController.serialize())
  }, [graphController.graph])

  const trimmedName = name.trim()
  const trimmedDescription = description.trim()

  const isDirty = useMemo(() => {
    return (
      trimmedName !== baseline.name ||
      trimmedDescription !== baseline.description ||
      currentGraphSignature !== baseline.graphSignature
    )
  }, [
    trimmedName,
    baseline.name,
    trimmedDescription,
    baseline.description,
    currentGraphSignature,
    baseline.graphSignature,
  ])

  const selectedNode = useMemo(
    () =>
      graphController.graph.nodes.find((node) => node.id === selectedNodeId) ??
      null,
    [graphController.graph.nodes, selectedNodeId],
  )

  useEffect(() => {
    if (!selectedNodeId) {
      return
    }

    const nodes = graphController.graph.nodes
    const existing = nodes.find((node) => node.id === selectedNodeId)

    if (!existing) {
      clearSelection()
      return
    }

    const shouldPreserveSelection = preserveSelectionRef.current
    const otherSelected = nodes.filter(
      (node) => node.selected && node.id !== selectedNodeId,
    )

    if (otherSelected.length === 0) {
      if (!existing.selected && shouldPreserveSelection) {
        graphController.onNodesChange([
          { id: selectedNodeId, type: 'select', selected: true },
        ])
      }
      return
    }

    graphController.onNodesChange(
      otherSelected.map((node) => ({
        id: node.id,
        type: 'select',
        selected: false,
      })),
    )

    if (!existing.selected && shouldPreserveSelection) {
      graphController.onNodesChange([
        { id: selectedNodeId, type: 'select', selected: true },
      ])
    }
  }, [
    clearSelection,
    graphController.graph.nodes,
    graphController.onNodesChange,
    selectedNodeId,
  ])

  useEffect(() => {
    if (!selectedNodeId) {
      return
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (!(event.target instanceof Node)) {
        return
      }

      const canvasElement = canvasContainerRef.current
      const inspectorElement = inspectorRef.current

      if (
        canvasElement?.contains(event.target) ||
        inspectorElement?.contains(event.target)
      ) {
        return
      }

      disableSelectionPreservation()
      clearSelection()
      graphController.onNodesChange([
        { id: selectedNodeId, type: 'select', selected: false },
      ])
    }

    window.addEventListener('pointerdown', handlePointerDown)

    return () => {
      window.removeEventListener('pointerdown', handlePointerDown)
    }
  }, [
    clearSelection,
    disableSelectionPreservation,
    graphController,
    selectedNodeId,
  ])

  const selectedAliases = useMemo(() => {
    if (!selectedNode) {
      return [] as string[]
    }

    const aliases = new Set<string>()
    const { config } = selectedNode.data

    if (config.load?.alias) {
      aliases.add(config.load.alias)
    }

    if (config.filter?.alias) {
      aliases.add(config.filter.alias)
    }

    if (config.project?.alias) {
      aliases.add(config.project.alias)
    }

    if (config.join?.leftAlias) {
      aliases.add(config.join.leftAlias)
    }

    if (config.join?.rightAlias) {
      aliases.add(config.join.rightAlias)
    }

    if (config.sort?.alias) {
      aliases.add(config.sort.alias)
    }

    return Array.from(aliases)
  }, [selectedNode])

  const schemaFieldOptions = useMemo(() => {
    const map: Record<string, string[]> = {}

    for (const summary of schemaSummaries) {
      const keys = summary.sampleFields
        .map((field) => field.key.trim())
        .filter(Boolean)
      if (!keys.length) {
        continue
      }

      const sortedUnique = Array.from(new Set(keys)).sort((a, b) =>
        a.localeCompare(b),
      )

      const aliasKeys = new Set<string>()
      if (summary.alias.trim()) {
        aliasKeys.add(summary.alias.trim())
      }
      const sanitized = sanitizeAlias(summary.alias)
      if (sanitized) {
        aliasKeys.add(sanitized)
      }

      for (const key of aliasKeys) {
        const existing = map[key] ?? []
        const combined = new Set([...existing, ...sortedUnique])
        map[key] = Array.from(combined).sort((a, b) => a.localeCompare(b))
      }
    }

    return map
  }, [schemaSummaries])

  const entityTypeOptions = useMemo(() => {
    const schemas = entitySchemasQuery.data?.entitySchemas ?? []
    const names = schemas
      .map((schema) => schema.name.trim())
      .filter(Boolean)

    return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b))
  }, [entitySchemasQuery.data?.entitySchemas])

  const handleSave = useCallback(() => {
    if (!isDirty || updateMutation.isPending) {
      return
    }

    if (!trimmedName) {
      alert('Name is required')
      return
    }

    pendingBaselineRef.current = {
      name: trimmedName,
      description: trimmedDescription,
      graphSignature: currentGraphSignature,
    }

    updateMutation.mutate({
      input: {
        id: transformationId,
        name: trimmedName,
        description: trimmedDescription.length ? trimmedDescription : null,
        nodes: graphController.serialize(),
      },
    })
  }, [
    updateMutation,
    trimmedName,
    trimmedDescription,
    graphController,
    transformationId,
    isDirty,
    currentGraphSignature,
  ])

  useEffect(() => {
    if (updateMutation.isSuccess && pendingBaselineRef.current) {
      setBaseline(pendingBaselineRef.current)
      pendingBaselineRef.current = null
    }
  }, [updateMutation.isSuccess])

  useEffect(() => {
    if (updateMutation.isError) {
      pendingBaselineRef.current = null
    }
  }, [updateMutation.isError])

  useEffect(() => {
    if (autoSaveTimeoutRef.current) {
      clearTimeout(autoSaveTimeoutRef.current)
      autoSaveTimeoutRef.current = null
    }

    if (!isAutoSaveEnabled || !trimmedName || !isDirty || updateMutation.isPending) {
      return
    }

    autoSaveTimeoutRef.current = window.setTimeout(() => {
      autoSaveTimeoutRef.current = null
      handleSave()
    }, AUTO_SAVE_DEBOUNCE_MS)

    return () => {
      if (autoSaveTimeoutRef.current) {
        clearTimeout(autoSaveTimeoutRef.current)
        autoSaveTimeoutRef.current = null
      }
    }
  }, [
    handleSave,
    isAutoSaveEnabled,
    isDirty,
    trimmedName,
    updateMutation.isPending,
  ])

  useEffect(() => {
    if (!transformation) {
      setBaseline({ name: '', description: '', graphSignature: '' })
      return
    }

    setBaseline({
      name: transformation.name.trim(),
      description: (transformation.description ?? '').trim(),
      graphSignature: initialGraphSignature,
    })
  }, [transformation, initialGraphSignature])

  useEffect(() => {
    const handleKeydown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        event.preventDefault()
        handleSave()
      }
    }

    window.addEventListener('keydown', handleKeydown)
    return () => window.removeEventListener('keydown', handleKeydown)
  }, [handleSave])

  if (detailQuery.isLoading) {
    return (
      <p className="rounded border border-slate-200 p-6 text-sm text-slate-500">Loading…</p>
    )
  }

  if (detailQuery.error) {
    return (
      <p className="rounded border border-rose-300 bg-rose-50 p-6 text-sm text-rose-700">
        {(detailQuery.error as Error).message}
      </p>
    )
  }

  if (!transformation) {
    return (
      <p className="rounded border border-slate-200 p-6 text-sm text-slate-500">
        Transformation not found.
      </p>
    )
  }

  const handleExecute = () => {
    navigate({
      to: '/transformations/$transformationId/execute',
      params: { transformationId },
    })
  }

  const handleDelete = () => {
    if (!confirm('Delete this transformation?')) {
      return
    }

    deleteMutation.mutate({ id: transformationId })
  }

  return (
    <div className="flex flex-col gap-4">
      <TransformationToolbar
        onSave={handleSave}
        onExecute={handleExecute}
        onUndo={graphController.undo}
        onRedo={graphController.redo}
        onDelete={handleDelete}
        canUndo={graphController.canUndo}
        canRedo={graphController.canRedo}
        isSaving={updateMutation.isPending}
        isExecuting={false}
        isDirty={isDirty}
        extra={
          <div className="flex items-center gap-3">
            <label className="flex items-center gap-2 text-xs font-medium text-slate-600">
              <input
                type="checkbox"
                className="h-3 w-3 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                checked={isAutoSaveEnabled}
                onChange={(event) => setIsAutoSaveEnabled(event.target.checked)}
              />
              Auto-save
            </label>
            <button
              type="button"
              onClick={() => setPreviewRefreshKey((key) => key + 1)}
              className="rounded border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100"
            >
              Refresh preview
            </button>
          </div>
        }
      />

      <section className="grid gap-4 md:grid-cols-2">
        <div className="space-y-3 rounded border border-slate-200 bg-white p-4">
          <label className="block text-xs font-semibold text-slate-600">
            Name
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              className="mt-1 w-full rounded border border-slate-300 px-3 py-1 text-sm"
            />
          </label>
          <label className="block text-xs font-semibold text-slate-600">
            Description
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="mt-1 w-full rounded border border-slate-300 px-3 py-1 text-sm"
              rows={3}
            />
          </label>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-500">
            <div>
              <dt className="font-semibold text-slate-600">Transformation ID</dt>
              <dd className="truncate text-slate-500">{transformation.id}</dd>
            </div>
            <div>
              <dt className="font-semibold text-slate-600">Organization</dt>
              <dd className="text-slate-500">{transformation.organizationId}</dd>
            </div>
            <div>
              <dt className="font-semibold text-slate-600">Created</dt>
              <dd>{new Date(transformation.createdAt).toLocaleString()}</dd>
            </div>
            <div>
              <dt className="font-semibold text-slate-600">Updated</dt>
              <dd>{new Date(transformation.updatedAt).toLocaleString()}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section className="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)_260px]">
        <NodePalette
          onAdd={(type) => {
            const node = graphController.addNode(type)
            selectNodeById(node.id)
          }}
        />
        <div
          ref={canvasContainerRef}
          className="min-h-[520px] rounded border border-slate-200 bg-white p-2"
        >
          <TransformationCanvas
            controller={graphController}
            selectedNodeId={selectedNodeId}
            onSelect={handleCanvasSelect}
            onDeselect={handleCanvasDeselect}
            onBackgroundPointerDown={handleCanvasBackgroundPointerDown}
            preserveSelectionRef={preserveSelectionRef}
          />
        </div>
        <div className="flex flex-col gap-3">
          <div className="flex-1" ref={inspectorRef}>
            <NodeInspector
              node={selectedNode}
              onUpdate={graphController.updateNode}
              onDelete={(nodeId) => {
                graphController.removeNode(nodeId)
                clearSelection()
              }}
              allNodes={graphController.graph.nodes}
              schemaFieldOptions={schemaFieldOptions}
              entityTypeOptions={entityTypeOptions}
            />
          </div>
          <TransformationPreviewPanel
            transformationId={transformationId}
            isDirty={isDirty}
            highlightedAliases={selectedAliases}
            refreshKey={previewRefreshKey}
            onSchemaSummariesChange={setSchemaSummaries}
          />
        </div>
      </section>
    </div>
  )
}
