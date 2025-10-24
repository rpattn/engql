import { useEffect, useMemo, useState } from 'react'
import type {
  Entity,
  EntitySchema,
} from '../../../generated/graphql'
import { FieldType } from '../../../generated/graphql'
import ReferenceFieldInput from './ReferenceFieldInput'
import {
  buildInitialEntityFormState,
  EntityFormState,
  FieldInputValue,
  formatRelative,
  formatTimestamp,
} from './helpers'

type EntityEditorModalProps = {
  isOpen: boolean
  mode: 'create' | 'edit'
  schema?: EntitySchema
  organizationId: string
  entity?: Entity
  isSubmitting: boolean
  errorMessage?: string | null
  onClose: () => void
  onSubmit: (state: EntityFormState) => Promise<void>
}

export default function EntityEditorModal({
  isOpen,
  mode,
  schema,
  organizationId,
  entity,
  isSubmitting,
  errorMessage,
  onClose,
  onSubmit,
}: EntityEditorModalProps) {
  const schemaFields = schema?.fields ?? []

  const [formState, setFormState] = useState<EntityFormState>(() =>
    buildInitialEntityFormState(schemaFields, schema, entity),
  )

  const schemaFieldsKey = useMemo(
    () => schemaFields.map((field) => `${field.name}:${field.type}`).join('|'),
    [schemaFields],
  )

  useEffect(() => {
    if (isOpen) {
      setFormState(buildInitialEntityFormState(schemaFields, schema, entity))
    }
  }, [isOpen, entity?.id, schema?.id, schemaFieldsKey])

  if (!isOpen) {
    return null
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSubmit(formState)
  }

  const updateFieldValue = (fieldName: string, nextValue: FieldInputValue) => {
    setFormState((current) => ({
      ...current,
      fieldValues: {
        ...current.fieldValues,
        [fieldName]: nextValue,
      },
    }))
  }

  const canSubmit = formState.entityType.trim().length > 0 && !isSubmitting

  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4"
    >
      <div className="max-h-[92vh] w-full max-w-3xl overflow-y-auto rounded-lg bg-white shadow-xl">
        <div className="flex items-start justify-between border-b border-gray-200 px-6 py-4">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">
              {mode === 'create' ? 'Add Entity' : 'Edit Entity'}
            </h2>
            {schema && (
              <p className="mt-1 text-xs text-gray-500">
                Schema: {schema.name}
              </p>
            )}
            {entity && (
              <p className="mt-1 text-xs text-gray-500">
                Last updated {formatRelative(entity.updatedAt)} ({formatTimestamp(entity.updatedAt)})
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-gray-200 px-2 py-1 text-xs font-medium text-gray-600 transition hover:border-gray-300 hover:bg-gray-50"
          >
            Close
          </button>
        </div>
        <form onSubmit={handleSubmit} className="px-6 py-6 space-y-4">
          <label className="block text-sm font-medium text-gray-700">
            Entity type
            <input
              type="text"
              required
              value={formState.entityType}
              onChange={(event) =>
                setFormState((current) => ({
                  ...current,
                  entityType: event.target.value,
                }))
              }
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </label>

          <label className="block text-sm font-medium text-gray-700">
            Path
            <input
              type="text"
              value={formState.path}
              onChange={(event) =>
                setFormState((current) => ({
                  ...current,
                  path: event.target.value,
                }))
              }
              placeholder="Optional resource path"
              className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
            />
          </label>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-gray-800">Schema fields</h3>
              <span className="text-xs text-gray-500">
                {schemaFields.length} field{schemaFields.length === 1 ? '' : 's'}
              </span>
            </div>

            {schemaFields.length === 0 ? (
              <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">
                No fields defined for this schema.
              </div>
            ) : (
              <div className="grid gap-3">
                {schemaFields.map((field) => {
                  const rawValue = formState.fieldValues[field.name]
                  const isBooleanField = field.type === FieldType.Boolean
                  const isJsonField = field.type === FieldType.Json
                  const isNumericField =
                    field.type === FieldType.Integer || field.type === FieldType.Float
                  const isTimestampField = field.type === FieldType.Timestamp
                  const isEntityLinkField =
                    field.type === FieldType.EntityReference ||
                    field.type === FieldType.EntityReferenceArray
                  const isCanonicalReferenceField = field.type === FieldType.Reference

                  return (
                    <div
                      key={field.name}
                      className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-4"
                    >
                      <div className="flex items-start justify-between">
                        <div>
                          <div className="text-sm font-medium text-gray-800">
                            {field.name}
                            {field.required ? (
                              <span className="ml-1 text-red-500">*</span>
                            ) : null}
                          </div>
                          <div className="text-[11px] uppercase tracking-wide text-gray-500">
                            {field.type}
                          </div>
                        </div>
                      </div>
                      {field.description && (
                        <p className="mt-2 text-xs text-gray-600">{field.description}</p>
                      )}
                      <div className="mt-3">
                        {isEntityLinkField ? (
                          <ReferenceFieldInput
                            organizationId={organizationId}
                            schema={schema}
                            field={field}
                            value={rawValue}
                            onChange={(value) => updateFieldValue(field.name, value)}
                          />
                        ) : isBooleanField ? (
                          <label className="inline-flex items-center gap-2 text-sm font-medium text-gray-700">
                            <input
                              type="checkbox"
                              checked={Boolean(rawValue)}
                              onChange={(event) =>
                                updateFieldValue(field.name, event.target.checked)
                              }
                              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                            />
                            Enabled
                          </label>
                        ) : isJsonField ? (
                          <textarea
                            rows={4}
                            value={typeof rawValue === 'string' ? rawValue : ''}
                            onChange={(event) =>
                              updateFieldValue(field.name, event.target.value)
                            }
                            placeholder="Enter JSON"
                            className="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          />
                        ) : (
                          <>
                            <input
                              type={
                                isNumericField
                                  ? 'number'
                                  : isTimestampField
                                    ? 'datetime-local'
                                    : 'text'
                              }
                              value={
                                typeof rawValue === 'string'
                                  ? rawValue
                                  : isTimestampField
                                    ? ''
                                    : String(rawValue ?? '')
                              }
                              onChange={(event) =>
                                updateFieldValue(field.name, event.target.value)
                              }
                              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                            />
                            {isCanonicalReferenceField && (
                              <p className="mt-2 text-xs text-gray-500">
                                Reference values are trimmed and must remain unique for this
                                entity type. Linked fields resolve against the normalized
                                value.
                              </p>
                            )}
                          </>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          {errorMessage && (
            <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">
              {errorMessage}
            </div>
          )}

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!canSubmit}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-300"
            >
              {isSubmitting
                ? 'Saving...'
                : mode === 'create'
                  ? 'Create entity'
                  : 'Save changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
