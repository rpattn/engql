import type { ReactNode } from 'react'

export function TransformationToolbar({
  onSave,
  onExecute,
  onUndo,
  onRedo,
  onDelete,
  canUndo,
  canRedo,
  isSaving,
  isExecuting,
  isDirty,
  extra,
}: {
  onSave: () => void
  onExecute: () => void
  onUndo: () => void
  onRedo: () => void
  onDelete?: () => void
  canUndo: boolean
  canRedo: boolean
  isSaving: boolean
  isExecuting: boolean
  isDirty: boolean
  extra?: ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-slate-200 bg-white px-4 py-2">
      <div className="flex items-center gap-2 text-xs text-slate-500">
        <span className="h-2 w-2 rounded-full" style={{ backgroundColor: isDirty ? '#2563eb' : '#9ca3af' }} />
        {isDirty ? 'Unsaved changes' : 'All changes saved'}
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onUndo}
          disabled={!canUndo}
          className="rounded border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 disabled:opacity-50"
        >
          Undo
        </button>
        <button
          type="button"
          onClick={onRedo}
          disabled={!canRedo}
          className="rounded border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 disabled:opacity-50"
        >
          Redo
        </button>
        {extra}
        {onDelete && (
          <button
            type="button"
            onClick={onDelete}
            className="rounded border border-rose-300 px-3 py-1 text-xs font-semibold text-rose-600 hover:bg-rose-50"
          >
            Delete
          </button>
        )}
        <button
          type="button"
          onClick={onExecute}
          disabled={isExecuting}
          className="rounded border border-blue-200 px-3 py-1 text-xs font-semibold text-blue-600 hover:bg-blue-50 disabled:opacity-50"
        >
          {isExecuting ? 'Running…' : 'Execute'}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={isSaving || !isDirty}
          className="rounded bg-blue-600 px-3 py-1 text-xs font-semibold text-white disabled:bg-blue-300"
        >
          {isSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  )
}
