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
    <div className="flex items-center justify-between gap-3 rounded-xl border border-subtle bg-surface px-4 py-2 shadow-sm">
      <div className="flex items-center gap-2 text-xs text-muted">
        <span
          className={`h-2 w-2 rounded-full ${isDirty ? 'bg-blue-500' : 'bg-slate-400'}`}
          aria-hidden
        />
        {isDirty ? 'Unsaved changes' : 'All changes saved'}
      </div>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onUndo}
          disabled={!canUndo}
          className="rounded-md border border-subtle px-3 py-1 text-xs font-medium text-muted transition hover:bg-subtle disabled:opacity-50"
        >
          Undo
        </button>
        <button
          type="button"
          onClick={onRedo}
          disabled={!canRedo}
          className="rounded-md border border-subtle px-3 py-1 text-xs font-medium text-muted transition hover:bg-subtle disabled:opacity-50"
        >
          Redo
        </button>
        {extra}
        {onDelete && (
          <button
            type="button"
            onClick={onDelete}
            className="rounded-md border border-rose-500/40 px-3 py-1 text-xs font-semibold text-rose-500 transition hover:bg-rose-500/10"
          >
            Delete
          </button>
        )}
        <button
          type="button"
          onClick={onExecute}
          disabled={isExecuting}
          className="rounded-md border border-blue-500/40 px-3 py-1 text-xs font-semibold text-blue-500 transition hover:bg-blue-500/10 disabled:opacity-50"
        >
          {isExecuting ? 'Running…' : 'Execute'}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={isSaving || !isDirty}
          className="rounded-md bg-blue-600 px-3 py-1 text-xs font-semibold text-white shadow-sm transition hover:bg-blue-500 disabled:bg-blue-300"
        >
          {isSaving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </div>
  )
}
