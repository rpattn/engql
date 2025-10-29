import { useEffect, useMemo, useRef } from 'react'

import { useExecuteEntityTransformationQuery } from '@/generated/graphql'

import {
  summarizeTransformationEdges,
  type TransformationAliasSummary,
} from '../utils/preview'

const DEFAULT_LIMIT = 5

type TransformationPreviewPanelProps = {
  transformationId: string
  isDirty: boolean
  highlightedAliases?: string[]
  refreshKey?: number
  onSchemaSummariesChange?: (summaries: TransformationAliasSummary[]) => void
}

export function TransformationPreviewPanel({
  transformationId,
  isDirty,
  highlightedAliases = [],
  refreshKey = 0,
  onSchemaSummariesChange,
}: TransformationPreviewPanelProps) {
  const variables = useMemo(
    () => ({
      input: {
        transformationId,
        pagination: {
          limit: DEFAULT_LIMIT,
          offset: 0,
        },
      },
    }),
    [transformationId],
  )

  const { data, error, isFetching, refetch } = useExecuteEntityTransformationQuery(variables, {
    enabled: false,
  })

  const previousIsDirty = useRef(isDirty)

  useEffect(() => {
    void refetch()
  }, [refetch, refreshKey, transformationId])

  useEffect(() => {
    if (previousIsDirty.current && !isDirty) {
      void refetch()
    }
    previousIsDirty.current = isDirty
  }, [isDirty, refetch])

  const edges = data?.executeEntityTransformation?.edges

  const summaries = useMemo(() => summarizeTransformationEdges(edges), [edges])

  const highlighted = useMemo(() => new Set(highlightedAliases.filter(Boolean)), [highlightedAliases])

  useEffect(() => {
    if (!onSchemaSummariesChange) {
      return
    }

    if (error) {
      onSchemaSummariesChange([])
      return
    }

    if (!data?.executeEntityTransformation) {
      return
    }

    onSchemaSummariesChange(summaries)
  }, [data?.executeEntityTransformation, error, summaries, onSchemaSummariesChange])

  return (
    <aside className="flex max-h-full flex-col rounded-md border border-slate-200 bg-white p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-700">Transformation preview</h3>
          <p className="text-xs text-slate-500">
            {isDirty
              ? 'Preview reflects the last saved version. Save changes to refresh automatically.'
              : 'Preview reflects the most recently saved version.'}
          </p>
        </div>
        <button
          type="button"
          onClick={() => void refetch()}
          disabled={isFetching}
          className="rounded border border-slate-200 px-2 py-1 text-[11px] font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-50"
        >
          {isFetching ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>

      {error && (
        <p className="mt-3 rounded border border-rose-300 bg-rose-50 px-3 py-2 text-xs text-rose-700">
          {(error as Error).message}
        </p>
      )}

      {!error && summaries.length === 0 && !isFetching && (
        <p className="mt-4 rounded border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-500">
          No sample records were returned for this transformation.
        </p>
      )}

      <div className="mt-3 space-y-2 overflow-y-auto">
        {summaries.map((summary) => {
          const isHighlighted = highlighted.has(summary.alias)

          return (
            <div
              key={summary.alias}
              className={`rounded border px-3 py-2 ${
                isHighlighted
                  ? 'border-blue-300 bg-blue-50'
                  : 'border-slate-200 bg-slate-50'
              }`}
            >
              <div className="flex items-baseline justify-between gap-2">
                <p className="text-xs font-semibold text-slate-700">{summary.alias}</p>
                {summary.entityTypes.length > 0 && (
                  <p className="text-[11px] text-slate-500">
                    {summary.entityTypes.join(', ')}
                  </p>
                )}
              </div>
              {summary.sampleFields.length > 0 ? (
                <ul className="mt-2 space-y-1 text-[11px] text-slate-600">
                  {summary.sampleFields.map((field) => (
                    <li key={field.key} className="flex items-start gap-2">
                      <span className="font-medium text-slate-700">{field.key}:</span>
                      <span className="break-all text-slate-600">{field.value}</span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-[11px] text-slate-500">No properties available.</p>
              )}
            </div>
          )
        })}
      </div>
    </aside>
  )
}
