import { useDeferredValue, useEffect, useMemo, useState } from 'react'
import type {
  EntitySchema,
  FieldDefinition,
} from '../../../generated/graphql'
import { useEntitiesByTypeFullQuery } from '../../../generated/graphql'
import {
  extractEntityDisplayNameFromProperties,
  FieldInputValue,
} from './helpers'

type ReferenceFieldInputProps = {
  organizationId: string
  schema?: EntitySchema
  field: FieldDefinition
  value: FieldInputValue | undefined
  onChange: (value: FieldInputValue) => void
}

type ReferenceOption = {
  id: string
  label: string
}

export default function ReferenceFieldInput({
  organizationId,
  schema,
  field,
  value,
  onChange,
}: ReferenceFieldInputProps) {
  const isArrayField = field.type === 'ENTITY_REFERENCE_ARRAY'
  const [searchTerm, setSearchTerm] = useState('')
  const deferredSearchTerm = useDeferredValue(searchTerm)
  const trimmedSearch = deferredSearchTerm.trim().toLowerCase()
  const targetEntityType = field.referenceEntityType ?? schema?.name ?? ''

  const shouldFetch = Boolean(organizationId) && Boolean(targetEntityType)
  const entitiesQuery = useEntitiesByTypeFullQuery(
    {
      organizationId,
      entityType: targetEntityType,
    },
    {
      enabled: shouldFetch,
      staleTime: 30_000,
    },
  )

  const allEntities = entitiesQuery.data?.entitiesByType ?? []

  const suggestions = useMemo<ReferenceOption[]>(() => {
    return allEntities.map((entity) => ({
      id: entity.id,
      label: extractEntityDisplayNameFromProperties(entity.properties, entity.id),
    }))
  }, [allEntities])

  const [labels, setLabels] = useState<Record<string, string>>({})

  useEffect(() => {
    if (suggestions.length === 0) {
      return
    }
    setLabels((current) => {
      let changed = false
      const next = { ...current }
      for (const suggestion of suggestions) {
        if (next[suggestion.id] !== suggestion.label) {
          next[suggestion.id] = suggestion.label
          changed = true
        }
      }
      return changed ? next : current
    })
  }, [suggestions])

  useEffect(() => {
    const ids = isArrayField
      ? Array.isArray(value)
        ? value
        : typeof value === 'string' && value.length > 0
          ? [value]
          : []
      : typeof value === 'string' && value.length > 0
        ? [value]
        : []

    if (ids.length === 0) {
      return
    }

    setLabels((current) => {
      let changed = false
      const next = { ...current }
      for (const id of ids) {
        if (!next[id]) {
          next[id] = id
          changed = true
        }
      }
      return changed ? next : current
    })
  }, [value, isArrayField])

  const selectedIds = isArrayField
    ? Array.isArray(value)
      ? value
      : typeof value === 'string' && value.length > 0
        ? [value]
        : []
    : typeof value === 'string' && value.length > 0
      ? [value]
      : []

  const filteredOptions = useMemo(() => {
    if (trimmedSearch.length < 2) {
      return []
    }
    const searchLower = trimmedSearch.toLowerCase()
    return suggestions
      .filter((option) => option.label.toLowerCase().includes(searchLower))
      .slice(0, 20)
  }, [suggestions, trimmedSearch])

  const handleSelect = (option: ReferenceOption) => {
    setLabels((current) => ({
      ...current,
      [option.id]: option.label,
    }))

    if (isArrayField) {
      const currentIds = Array.isArray(value)
        ? value
        : typeof value === 'string' && value.length > 0
          ? [value]
          : []
      if (currentIds.includes(option.id)) {
        setSearchTerm('')
        return
      }
      const next = [...currentIds, option.id]
      onChange(next)
      setSearchTerm('')
    } else {
      onChange(option.id)
      setSearchTerm('')
    }
  }

  const handleRemove = (id: string) => {
    if (isArrayField) {
      const currentIds = Array.isArray(value)
        ? value
        : typeof value === 'string' && value.length > 0
          ? [value]
          : []
      const next = currentIds.filter((item) => item !== id)
      onChange(next)
    } else {
      onChange('')
      setSearchTerm('')
    }
  }

  const shouldShowDropdown =
    trimmedSearch.length >= 2 &&
    shouldFetch &&
    (entitiesQuery.isFetching || filteredOptions.length > 0 || entitiesQuery.isError)

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        {isArrayField ? (
          <div className="flex flex-wrap gap-2">
            {selectedIds.length === 0 ? (
              <span className="text-xs text-gray-500">No entities selected.</span>
            ) : (
              selectedIds.map((id) => (
                <span
                  key={id}
                  className="inline-flex items-center gap-2 rounded-full bg-blue-100 px-3 py-1 text-xs font-semibold text-blue-700"
                >
                  {labels[id] ?? id}
                  <button
                    type="button"
                    onClick={() => handleRemove(id)}
                    className="text-blue-700 transition hover:text-blue-900"
                    aria-label={`Remove ${labels[id] ?? id}`}
                  >
                    ×
                  </button>
                </span>
              ))
            )}
          </div>
        ) : (
          <div className="flex items-center gap-3">
            {selectedIds.length > 0 ? (
              <span className="inline-flex items-center rounded-full bg-blue-100 px-3 py-1 text-xs font-semibold text-blue-700">
                {labels[selectedIds[0]] ?? selectedIds[0]}
              </span>
            ) : (
              <span className="text-xs text-gray-500">No entity selected.</span>
            )}
            {selectedIds.length > 0 && (
              <button
                type="button"
                onClick={() => handleRemove(selectedIds[0]!)}
                className="text-xs font-medium text-red-500 hover:text-red-600"
              >
                Clear
              </button>
            )}
          </div>
        )}

        {targetEntityType && (
          <p className="text-xs text-gray-500">
            Searching within entity type{' '}
            <span className="font-semibold">{targetEntityType}</span>.
          </p>
        )}
      </div>

      <div className="relative">
        <input
          type="text"
          value={searchTerm}
          onChange={(event) => setSearchTerm(event.target.value)}
          placeholder="Search by name (min. 2 characters)"
          disabled={!organizationId || !shouldFetch}
          className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200 disabled:cursor-not-allowed disabled:bg-gray-100"
        />
        {!organizationId && (
          <p className="mt-1 text-xs text-gray-500">
            Select an organization to enable search.
          </p>
        )}
        {organizationId && shouldFetch && trimmedSearch.length < 2 && (
          <p className="mt-1 text-xs text-gray-500">
            Enter at least two characters to search by name.
          </p>
        )}
        {!targetEntityType && (
          <p className="mt-1 text-xs text-gray-500">
            Reference entity type not configured for this field.
          </p>
        )}

        {shouldShowDropdown && (
          <div className="absolute z-10 mt-2 max-h-60 w-full overflow-y-auto rounded-md border border-gray-200 bg-white shadow-lg">
            {entitiesQuery.isFetching ? (
              <div className="px-4 py-3 text-sm text-gray-500">Searching...</div>
            ) : entitiesQuery.isError ? (
              <div className="px-4 py-3 text-sm text-red-600">
                {(entitiesQuery.error as Error).message}
              </div>
            ) : filteredOptions.length > 0 ? (
              <ul>
                {filteredOptions.map((option) => {
                  const isSelected = selectedIds.includes(option.id)
                  return (
                    <li key={option.id}>
                      <button
                        type="button"
                        onClick={() => handleSelect(option)}
                        className={`flex w-full items-center justify-between px-4 py-2 text-left text-sm transition ${
                          isSelected ? 'bg-blue-50 text-blue-700' : 'hover:bg-gray-100'
                        }`}
                      >
                        <span>{option.label}</span>
                        {isSelected && (
                          <span className="text-xs font-semibold uppercase text-blue-600">
                            Selected
                          </span>
                        )}
                      </button>
                    </li>
                  )
                })}
              </ul>
            ) : (
              <div className="px-4 py-3 text-sm text-gray-500">
                No matching entities found.
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

