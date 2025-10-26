import { useDeferredValue, useEffect, useMemo, useState } from 'react'
import type {
  EntitySchema,
  FieldDefinition,
} from '../../../generated/graphql'
import {
  FieldType,
  useEntitiesByTypeFullQuery,
  useEntitySchemasQuery,
} from '../../../generated/graphql'
import {
  extractEntityDisplayNameFromProperties,
  FieldInputValue,
  safeParseProperties,
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
  value: string
  referenceValue?: string
  label: string
  primaryLabel: string
  displayName: string
  searchTokens: string[]
  referenceValues: string[]
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
  const entitySchemasQuery = useEntitySchemasQuery(
    { organizationId },
    { enabled: shouldFetch, staleTime: 60_000 },
  )
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
  const targetSchema = useMemo(() => {
    if (!targetEntityType) {
      return undefined
    }
    return entitySchemasQuery.data?.entitySchemas.find(
      (entry) => entry.name === targetEntityType,
    )
  }, [entitySchemasQuery.data?.entitySchemas, targetEntityType])

  const referenceFieldNames = useMemo(() => {
    if (!targetSchema) {
      return [] as string[]
    }
    return targetSchema.fields
      .filter((schemaField) => schemaField.type === FieldType.Reference)
      .map((schemaField) => schemaField.name.trim())
      .filter((name) => name.length > 0)
  }, [targetSchema])

  const suggestions = useMemo<ReferenceOption[]>(() => {
    return allEntities.map((entity) => {
      const reference = entity.referenceValue?.trim() ?? ''
      const parsedProps = safeParseProperties(entity.properties)
      const referenceCandidates = new Set<string>()
      const referenceValues: string[] = []
      if (reference) {
        referenceCandidates.add(reference)
        referenceValues.push(reference)
      }
      for (const name of referenceFieldNames) {
        const raw = parsedProps[name]
        if (typeof raw === 'string') {
          const trimmed = raw.trim()
          if (trimmed.length > 0) {
            if (!referenceCandidates.has(trimmed)) {
              referenceValues.push(trimmed)
            }
            referenceCandidates.add(trimmed)
          }
        }
      }
      const displayName = extractEntityDisplayNameFromProperties(
        entity.properties,
        entity.id,
      )
      const labelParts = new Set<string>()
      for (const candidate of referenceCandidates) {
        labelParts.add(candidate)
      }
      if (displayName) {
        labelParts.add(displayName)
      }
      labelParts.add(entity.id)

      const label = Array.from(labelParts).join(' • ')
      const referenceValue = reference || undefined
      const primaryLabel = referenceValue ?? displayName ?? entity.id

      const allReferenceTokens = Array.from(referenceCandidates).map((token) =>
        token.toLowerCase(),
      )
      const searchTokens = [
        ...allReferenceTokens,
        displayName?.toLowerCase(),
        entity.id.toLowerCase(),
        primaryLabel.toLowerCase(),
      ].filter((token): token is string => Boolean(token && token.length > 0))

      return {
        id: entity.id,
        value: entity.id,
        referenceValue,
        label,
        primaryLabel,
        displayName,
        searchTokens,
        referenceValues,
      }
    })
  }, [allEntities, referenceFieldNames])

  const [labels, setLabels] = useState<Record<string, string>>({})

  useEffect(() => {
    if (suggestions.length === 0) {
      return
    }

    if (isArrayField) {
      const currentIds = Array.isArray(value)
        ? value
        : typeof value === 'string' && value.length > 0
          ? [value]
          : []
      if (currentIds.length === 0) {
        return
      }

      const converted = currentIds.map((item) => {
        if (suggestions.some((option) => option.value === item)) {
          return item
        }
        const match = suggestions.find(
          (option) => option.referenceValue && option.referenceValue === item,
        )
        return match?.value ?? item
      })

      const normalized = Array.from(new Set(converted))
      const hasChanged =
        normalized.length !== currentIds.length ||
        normalized.some((item, index) => item !== currentIds[index])
      if (hasChanged) {
        onChange(normalized)
      }
      return
    }

    const currentId = typeof value === 'string' ? value : ''
    if (!currentId) {
      return
    }

    if (suggestions.some((option) => option.value === currentId)) {
      return
    }

    const match = suggestions.find(
      (option) => option.referenceValue && option.referenceValue === currentId,
    )
    if (match) {
      onChange(match.value)
    }
  }, [isArrayField, onChange, suggestions, value])

  useEffect(() => {
    if (suggestions.length === 0) {
      return
    }
    setLabels((current) => {
      let changed = false
      const next = { ...current }
      for (const suggestion of suggestions) {
        if (next[suggestion.value] !== suggestion.primaryLabel) {
          next[suggestion.value] = suggestion.primaryLabel
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

  const selectedValues = isArrayField
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
      .filter((option) => {
        if (option.label.toLowerCase().includes(searchLower)) {
          return true
        }
        return option.searchTokens.some((token) => token.includes(searchLower))
      })
      .slice(0, 20)
  }, [suggestions, trimmedSearch])

  const handleSelect = (option: ReferenceOption) => {
    setLabels((current) => ({
      ...current,
      [option.value]: option.primaryLabel,
    }))

    if (isArrayField) {
      const currentIds = Array.isArray(value)
        ? value
        : typeof value === 'string' && value.length > 0
          ? [value]
          : []
      if (currentIds.includes(option.value)) {
        setSearchTerm('')
        return
      }
      const next = [...currentIds, option.value]
      onChange(next)
      setSearchTerm('')
    } else {
      onChange(option.value)
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
            {selectedValues.length === 0 ? (
              <span className="text-xs text-gray-500">No entities selected.</span>
            ) : (
              selectedValues.map((id) => (
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
            {selectedValues.length > 0 ? (
              <span className="inline-flex items-center rounded-full bg-blue-100 px-3 py-1 text-xs font-semibold text-blue-700">
                {labels[selectedValues[0]] ?? selectedValues[0]}
              </span>
            ) : (
              <span className="text-xs text-gray-500">No entity selected.</span>
            )}
            {selectedValues.length > 0 && (
              <button
                type="button"
                onClick={() => handleRemove(selectedValues[0]!)}
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
          placeholder="Search by name or reference (min. 2 characters)"
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
            Enter at least two characters to search by name or reference value.
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
                  const isSelected = selectedValues.includes(option.value)
                  const matchingReferences =
                    trimmedSearch.length >= 2
                      ? option.referenceValues.filter((candidate) =>
                          candidate.toLowerCase().includes(trimmedSearch),
                        )
                      : []
                  const extraMatchingReferences = option.referenceValue
                    ? matchingReferences.filter(
                        (candidate) => candidate !== option.referenceValue,
                      )
                    : matchingReferences
                  return (
                    <li key={option.id}>
                      <button
                        type="button"
                        onClick={() => handleSelect(option)}
                        className={`flex w-full items-start justify-between gap-4 px-4 py-3 text-left text-sm transition ${
                          isSelected ? 'bg-blue-50 text-blue-700' : 'hover:bg-gray-100'
                        }`}
                        aria-label={`Select ${option.primaryLabel}`}
                      >
                        <div className="flex flex-col text-left">
                          <span className="text-sm font-medium text-gray-900">
                            {option.primaryLabel}
                          </span>
                          <div className="mt-1 space-y-0.5 text-xs text-gray-500">
                            {option.referenceValue && (
                              <div>Reference: {option.referenceValue}</div>
                            )}
                            <div>ID: {option.id}</div>
                            {extraMatchingReferences.length > 0 && (
                              <div>
                                Matching references:{' '}
                                {extraMatchingReferences.join(', ')}
                              </div>
                            )}
                            {option.displayName &&
                              option.displayName !== option.primaryLabel &&
                              option.displayName !== option.referenceValue && (
                                <div>Display name: {option.displayName}</div>
                              )}
                          </div>
                        </div>
                        {isSelected && (
                          <span className="mt-1 text-xs font-semibold uppercase text-blue-600">
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

