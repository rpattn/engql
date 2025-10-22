import { useEffect, useMemo, useRef, useState } from 'react'
import type { FieldDefinition } from '../../../generated/graphql'
import { FieldType } from '../../../generated/graphql'

type ColumnFilterPopoverProps = {
  field: FieldDefinition
  initialValue?: string
  onApply: (value: string) => void
  onClear: () => void
  onClose: () => void
}

export default function ColumnFilterPopover({
  field,
  initialValue = '',
  onApply,
  onClear,
  onClose,
}: ColumnFilterPopoverProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [value, setValue] = useState(initialValue)

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!containerRef.current) {
        return
      }
      if (!containerRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [onClose])

  useEffect(() => {
    setValue(initialValue)
  }, [initialValue])

  const isBooleanField = field.type === FieldType.Boolean

  const booleanOptions = useMemo(
    () => [
      { label: 'Any', value: '' },
      { label: 'True', value: 'true' },
      { label: 'False', value: 'false' },
    ],
    [],
  )

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onApply(value.trim())
    onClose()
  }

  const handleClear = () => {
    setValue('')
    onClear()
    onClose()
  }

  return (
    <div
      ref={containerRef}
      className="absolute right-0 top-7 z-20 w-64 rounded-lg border border-gray-200 bg-white p-4 text-sm shadow-lg"
    >
      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wide text-gray-500">
            Filter {field.name}
          </div>
          <div className="mt-2">
            {isBooleanField ? (
              <select
                value={value}
                onChange={(event) => setValue(event.target.value)}
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              >
                {booleanOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            ) : (
              <input
                type="text"
                value={value}
                onChange={(event) => setValue(event.target.value)}
                placeholder="Equal to..."
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            )}
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <button
            type="button"
            onClick={handleClear}
            className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900"
          >
            Clear
          </button>
          <button
            type="submit"
            className="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-blue-500"
          >
            Apply
          </button>
        </div>
      </form>
    </div>
  )
}

