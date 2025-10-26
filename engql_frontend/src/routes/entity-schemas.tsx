import { useCallback, useEffect, useMemo, useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import type { EntitySchema } from '../generated/graphql'
import {
  FieldType,
  useCreateSchemaMutation,
  useDeleteSchemaMutation,
  useEntitySchemasQuery,
  useGetOrganizationsQuery,
  useUpdateSchemaMutation,
} from '../generated/graphql'
import { loadLastOrganizationId, persistLastOrganizationId } from '../lib/browserStorage'

type ModalMode = 'create' | 'edit'

type FieldFormValue = {
  clientId: string
  name: string
  type: FieldType
  required: boolean
  description: string
  defaultValue: string
  validation: string
  referenceEntityType: string
}

type SchemaFormState = {
  name: string
  description: string
  fields: FieldFormValue[]
}

type ModalState =
  | {
      mode: 'create'
    }
  | {
      mode: 'edit'
      schema: EntitySchema
    }

type FeedbackState =
  | {
      type: 'success' | 'error'
      message: string
    }
  | null

export const Route = createFileRoute('/entity-schemas')({
  component: EntitySchemasPage,
})

function EntitySchemasPage() {
  const queryClient = useQueryClient()
  const [selectedOrgId, setSelectedOrgId] = useState<string | null>(
    () => loadLastOrganizationId(),
  )
  const [modalState, setModalState] = useState<ModalState | null>(null)
  const [modalError, setModalError] = useState<string | null>(null)
  const [feedback, setFeedback] = useState<FeedbackState>(null)

  const organizationsQuery = useGetOrganizationsQuery()
  const organizations = organizationsQuery.data?.organizations ?? []

  useEffect(() => {
    if (organizations.length === 0) {
      setSelectedOrgId(null)
      return
    }

    setSelectedOrgId((current) => {
      const activeId =
        current && organizations.some((org) => org.id === current)
          ? current
          : loadLastOrganizationId()

      if (activeId && organizations.some((org) => org.id === activeId)) {
        return activeId
      }

      return organizations[0]?.id ?? null
    })
  }, [organizations])

  useEffect(() => {
    persistLastOrganizationId(selectedOrgId)
  }, [selectedOrgId])

  const entitySchemasQueryKey = useMemo(
    () =>
      selectedOrgId
        ? useEntitySchemasQuery.getKey({ organizationId: selectedOrgId })
        : undefined,
    [selectedOrgId],
  )

  const entitySchemasQuery = useEntitySchemasQuery(
    {
      organizationId: selectedOrgId ?? '',
    },
    {
      enabled: Boolean(selectedOrgId),
    },
  )

  const entitySchemasErrorMessage =
    entitySchemasQuery.error instanceof Error
      ? entitySchemasQuery.error.message
      : entitySchemasQuery.error
        ? String(entitySchemasQuery.error)
        : null

  const createSchemaMutation = useCreateSchemaMutation()
  const updateSchemaMutation = useUpdateSchemaMutation()
  const deleteSchemaMutation = useDeleteSchemaMutation()

  const activeOrganization = useMemo(
    () =>
      selectedOrgId
        ? organizations.find((org) => org.id === selectedOrgId) ?? null
        : null,
    [organizations, selectedOrgId],
  )

  const invalidateSchemas = useCallback(async () => {
    if (!entitySchemasQueryKey) {
      return
    }
    await queryClient.invalidateQueries({ queryKey: entitySchemasQueryKey })
  }, [entitySchemasQueryKey, queryClient])

  const handleCreateClick = () => {
    setModalError(null)
    setModalState({ mode: 'create' })
  }

  const handleEditClick = (schema: EntitySchema) => {
    setModalError(null)
    setModalState({ mode: 'edit', schema })
  }

  const handleModalClose = () => {
    setModalState(null)
    setModalError(null)
  }

  const handleModalSubmit = async (formState: SchemaFormState) => {
    if (!selectedOrgId || !modalState) {
      return
    }

    const normalizedFields = formState.fields.map((field) => ({
      name: field.name.trim(),
      type: normalizeFieldType(field.type),
      required: field.required,
      description: field.description.trim() ? field.description.trim() : undefined,
      default: field.defaultValue.trim() ? field.defaultValue.trim() : undefined,
      validation: field.validation.trim() ? field.validation.trim() : undefined,
      referenceEntityType: field.referenceEntityType.trim()
        ? field.referenceEntityType.trim()
        : undefined,
    }))

    try {
      setModalError(null)

      if (modalState.mode === 'create') {
        await createSchemaMutation.mutateAsync({
          input: {
            organizationId: selectedOrgId,
            name: formState.name.trim(),
            description: formState.description.trim()
              ? formState.description.trim()
              : undefined,
            fields: normalizedFields,
          },
        })
        setFeedback({
          type: 'success',
          message: 'Entity schema created successfully.',
        })
      } else {
        await updateSchemaMutation.mutateAsync({
          input: {
            id: modalState.schema.id,
            name: formState.name.trim(),
            description: formState.description.trim()
              ? formState.description.trim()
              : undefined,
            fields: normalizedFields,
          },
        })
        setFeedback({
          type: 'success',
          message: 'Entity schema updated successfully.',
        })
      }

      await invalidateSchemas()
      setModalState(null)
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Something went wrong while saving the entity schema.'
      setModalError(message)
    }
  }

  const handleDelete = async (schema: EntitySchema) => {
    if (!selectedOrgId) {
      return
    }

    const confirmed = window.confirm(
      `Delete schema "${schema.name}"? This cannot be undone.`,
    )

    if (!confirmed) {
      return
    }

    try {
      await deleteSchemaMutation.mutateAsync({
        id: schema.id,
      })
      setFeedback({
        type: 'success',
        message: `Deleted schema "${schema.name}".`,
      })
      await invalidateSchemas()
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to delete the entity schema.'
      setFeedback({ type: 'error', message })
    }
  }

  const schemas = entitySchemasQuery.data?.entitySchemas ?? []
  const isLoading =
    organizationsQuery.isLoading || entitySchemasQuery.isLoading
  const isFetching = entitySchemasQuery.isFetching

  const isSubmitting =
    createSchemaMutation.isPending || updateSchemaMutation.isPending

  const lastUpdated = entitySchemasQuery.dataUpdatedAt
    ? new Date(entitySchemasQuery.dataUpdatedAt)
    : null

  return (
    <div className="mx-auto max-w-6xl px-6 py-8 text-gray-900">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold">Entity Schemas</h1>
          <p className="mt-1 text-sm text-gray-600">
            Review, create, and maintain entity schema definitions for your
            organizations.
          </p>
        </div>
        <button
          type="button"
          onClick={handleCreateClick}
          disabled={!selectedOrgId}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:bg-blue-300"
        >
          Create Schema
        </button>
      </div>

      <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-end">
        <label className="flex flex-col text-sm font-medium text-gray-700">
          Organization
          <select
            value={selectedOrgId ?? ''}
            onChange={(event) => {
              setSelectedOrgId(event.target.value || null)
              setFeedback(null)
            }}
            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
          >
            <option value="" disabled>
              Select an organization
            </option>
            {organizations.map((organization) => (
              <option key={organization.id} value={organization.id}>
                {organization.name}
              </option>
            ))}
          </select>
        </label>
        <div className="text-xs text-gray-500">
          {activeOrganization ? (
            <p>
              Working in <span className="font-semibold">{activeOrganization.name}</span>
            </p>
          ) : (
            <p>Select an organization to manage its schemas.</p>
          )}
        </div>
      </div>

      {feedback && (
        <div
          className={`mt-4 rounded-md border px-4 py-2 text-sm ${
            feedback.type === 'success'
              ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
              : 'border-red-200 bg-red-50 text-red-700'
          }`}
        >
          {feedback.message}
        </div>
      )}

      {entitySchemasErrorMessage && (
        <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
          {entitySchemasErrorMessage}
        </div>
      )}

      <div className="mt-8 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
        <div className="flex items-center justify-between border-b border-gray-200 px-4 py-3 text-sm text-gray-600">
          <div>
            {isLoading
              ? 'Loading schemas…'
              : `${schemas.length} schema${schemas.length === 1 ? '' : 's'}${
                  isFetching ? ' (refreshing…)' : ''
                }`}
          </div>
          {activeOrganization && (
            <div className="text-xs text-gray-500">
              Last updated:{' '}
              {lastUpdated ? formatTimestamp(lastUpdated) : 'pending…'}
            </div>
          )}
        </div>

        {selectedOrgId ? (
          schemas.length > 0 ? (
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600">
                    Schema
                  </th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600">
                    Description
                  </th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600">
                    Status
                  </th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600">
                    Version
                  </th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600">
                    Fields
                  </th>
                  <th className="px-4 py-3 text-right font-semibold text-gray-600">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {schemas.map((schema) => (
                  <tr key={schema.id} className="bg-white align-top">
                    <td className="px-4 py-4">
                      <div className="font-medium text-gray-900">
                        {schema.name}
                      </div>
                      <div className="text-xs text-gray-500">
                        ID: {schema.id}
                      </div>
                      <div className="mt-1 text-xs text-gray-500">
                        Updated {formatRelative(schema.updatedAt)}
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      {schema.description ? (
                        <p className="text-sm text-gray-700">
                          {schema.description}
                        </p>
                      ) : (
                        <span className="text-xs uppercase tracking-wide text-gray-400">
                          No description
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-1 text-xs font-semibold uppercase tracking-wide text-gray-700">
                        {schema.status}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="text-sm font-medium text-gray-800">
                        {schema.version}
                      </div>
                      {schema.previousVersionId && (
                        <div className="text-xs text-gray-500">
                          Prev: {schema.previousVersionId}
                        </div>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      {schema.fields.length > 0 ? (
                        <div className="flex flex-wrap gap-2">
                          {schema.fields.map((field) => (
                            <span
                              key={`${schema.id}-${field.name}-${field.type}`}
                              className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-1 text-[11px] font-medium uppercase tracking-wide text-blue-700"
                            >
                              {field.name}
                              <span className="rounded bg-blue-100 px-1 py-0.5 text-[10px] text-blue-600">
                                {field.type}
                              </span>
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-xs uppercase tracking-wide text-gray-400">
                          No fields configured
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => handleEditClick(schema)}
                          className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition hover:border-gray-400 hover:text-gray-900"
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(schema)}
                          className="rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 transition hover:border-red-300 hover:bg-red-50"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="px-6 py-10 text-center text-sm text-gray-600">
              {isLoading
                ? 'Loading entity schemas…'
                : 'No entity schemas found for this organization yet.'}
            </div>
          )
        ) : (
          <div className="px-6 py-10 text-center text-sm text-gray-600">
            Select an organization to view its entity schemas.
          </div>
        )}
      </div>

      {modalState && (
        <SchemaModal
          isOpen={Boolean(modalState)}
          mode={modalState.mode}
          schema={modalState.mode === 'edit' ? modalState.schema : undefined}
          organizationName={activeOrganization?.name}
          onClose={handleModalClose}
          onSubmit={handleModalSubmit}
          isSubmitting={isSubmitting}
          errorMessage={modalError}
        />
      )}
    </div>
  )
}

type SchemaModalProps = {
  isOpen: boolean
  mode: ModalMode
  schema?: EntitySchema
  organizationName?: string | null
  onClose: () => void
  onSubmit: (state: SchemaFormState) => Promise<void>
  isSubmitting: boolean
  errorMessage?: string | null
}

function SchemaModal({
  isOpen,
  mode,
  schema,
  organizationName,
  onClose,
  onSubmit,
  isSubmitting,
  errorMessage,
}: SchemaModalProps) {
  const [formState, setFormState] = useState<SchemaFormState>(() =>
    schema ? buildFormStateFromSchema(schema) : createEmptyFormState(),
  )

  useEffect(() => {
    if (isOpen) {
      setFormState(schema ? buildFormStateFromSchema(schema) : createEmptyFormState())
    }
  }, [isOpen, schema])

  const fieldTypeOptions = useMemo(
    () =>
      (Object.values(FieldType) as FieldType[]).filter(
        (option) => option !== ('ENTITY_ID' as unknown as FieldType),
      ),
    [],
  )

  const referenceFieldOrder = useMemo(() => {
    const indices: number[] = []
    formState.fields.forEach((field, index) => {
      if (field.type === FieldType.Reference) {
        indices.push(index)
      }
    })
    return indices
  }, [formState.fields])

  const canonicalReferenceIndex = referenceFieldOrder[0] ?? -1

  const handleFieldChange = <Key extends keyof FieldFormValue>(
    id: string,
    key: Key,
    value: FieldFormValue[Key],
  ) => {
    setFormState((current) => ({
      ...current,
      fields: current.fields.map((field) =>
        field.clientId === id ? { ...field, [key]: value } : field,
      ),
    }))
  }

  const handleAddField = () => {
    setFormState((current) => ({
      ...current,
      fields: [...current.fields, createEmptyField()],
    }))
  }

  const handleRemoveField = (id: string) => {
    setFormState((current) => ({
      ...current,
      fields:
        current.fields.length > 1
          ? current.fields.filter((field) => field.clientId !== id)
          : current.fields,
    }))
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    await onSubmit(formState)
  }

  const allFieldsValid =
    formState.fields.length > 0 &&
    formState.fields.every((field) => field.name.trim() && field.type)

  const canSubmit =
    formState.name.trim().length > 0 && allFieldsValid && !isSubmitting

  if (!isOpen) {
    return null
  }

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
              {mode === 'create' ? 'Create Entity Schema' : 'Edit Entity Schema'}
            </h2>
            {organizationName && (
              <p className="mt-1 text-xs text-gray-500">
                Organization: {organizationName}
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
        <form onSubmit={handleSubmit} className="px-6 py-6">
          <div className="space-y-4">
            <label className="block text-sm font-medium text-gray-700">
              Schema name
              <input
                type="text"
                required
                value={formState.name}
                onChange={(event) =>
                  setFormState((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            </label>
            <label className="block text-sm font-medium text-gray-700">
              Description
              <textarea
                rows={3}
                value={formState.description}
                onChange={(event) =>
                  setFormState((current) => ({
                    ...current,
                    description: event.target.value,
                  }))
                }
                placeholder="Optional short description for this schema"
                className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
              />
            </label>
            <div>
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-gray-800">Fields</h3>
                <button
                  type="button"
                  onClick={handleAddField}
                  className="text-xs font-medium text-blue-600 hover:text-blue-500"
                >
                  + Add field
                </button>
              </div>
              <p className="mt-1 text-xs text-gray-500">
                Define the shape of entities stored under this schema. At least
                one field is required.
              </p>
              <div className="mt-3 space-y-3">
                {formState.fields.map((field, index) => {
                  const showReferenceEntityType =
                    field.type === FieldType.EntityReference ||
                    field.type === FieldType.EntityReferenceArray ||
                    field.type === FieldType.Reference
                  const isReferenceField = field.type === FieldType.Reference
                  const isCanonicalReferenceField =
                    isReferenceField && index === canonicalReferenceIndex
                  return (
                    <div
                      key={field.clientId}
                      className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-4"
                    >
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="text-sm font-medium text-gray-700">
                          Field {index + 1}
                        </div>
                        <button
                          type="button"
                          onClick={() => handleRemoveField(field.clientId)}
                          disabled={formState.fields.length === 1}
                          className="text-xs font-semibold text-red-500 hover:text-red-600 disabled:cursor-not-allowed disabled:text-gray-400"
                        >
                          Remove
                        </button>
                      </div>
                      <div className="mt-3 grid gap-3 md:grid-cols-2">
                        <label className="flex flex-col text-xs font-medium text-gray-600">
                          Name
                          <input
                            type="text"
                            required
                            value={field.name}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'name',
                                event.target.value,
                              )
                            }
                            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          />
                        </label>
                        <label className="flex flex-col text-xs font-medium text-gray-600">
                          Type
                          <select
                            value={field.type}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'type',
                                event.target.value as FieldType,
                              )
                            }
                            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          >
                            {fieldTypeOptions.map((option) => (
                              <option key={option} value={option}>
                                {option}
                              </option>
                            ))}
                          </select>
                        </label>
                      </div>
                      <div className="mt-3 grid gap-3 md:grid-cols-2">
                        <label className="flex items-center gap-2 text-xs font-medium text-gray-600">
                          <input
                            type="checkbox"
                            checked={field.required}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'required',
                                event.target.checked,
                              )
                            }
                            className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          Required field
                        </label>
                        <label className="flex flex-col text-xs font-medium text-gray-600">
                          Default value
                          <input
                            type="text"
                            value={field.defaultValue}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'defaultValue',
                                event.target.value,
                              )
                            }
                            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          />
                        </label>
                      </div>
                      <div className="mt-3 grid gap-3 md:grid-cols-2">
                        <label className="flex flex-col text-xs font-medium text-gray-600">
                          Description
                          <input
                            type="text"
                            value={field.description}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'description',
                                event.target.value,
                              )
                            }
                            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          />
                        </label>
                        <label className="flex flex-col text-xs font-medium text-gray-600">
                          Validation rules
                          <input
                            type="text"
                            value={field.validation}
                            onChange={(event) =>
                              handleFieldChange(
                                field.clientId,
                                'validation',
                                event.target.value,
                              )
                            }
                            placeholder="Optional JSON or rule definition"
                            className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                          />
                        </label>
                        {isReferenceField && (
                          <p className="md:col-span-2 mt-1 text-xs text-blue-600">
                            {isCanonicalReferenceField
                              ? 'This REFERENCE field acts as the canonical reference value for uniqueness and linking.'
                              : 'Additional REFERENCE fields stay searchable but do not replace the canonical reference value.'}
                          </p>
                        )}
                        {showReferenceEntityType && (
                          <label className="md:col-span-2 flex flex-col text-xs font-medium text-gray-600">
                            Reference entity type
                            <input
                              type="text"
                              value={field.referenceEntityType}
                              onChange={(event) =>
                                handleFieldChange(
                                  field.clientId,
                                  'referenceEntityType',
                                  event.target.value,
                                )
                              }
                              placeholder="Entity type this field references"
                              className="mt-1 rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-200"
                            />
                          </label>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>

          {errorMessage && (
            <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-600">
              {errorMessage}
            </div>
          )}

          <div className="mt-6 flex justify-end gap-3">
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
                ? 'Saving…'
                : mode === 'create'
                  ? 'Create schema'
                  : 'Save changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function normalizeFieldType(type: string | null | undefined): FieldType {
  if (!type) {
    return FieldType.String
  }

  if ((Object.values(FieldType) as FieldType[]).includes(type as FieldType)) {
    return type as FieldType
  }

  const matchingKey = (Object.keys(FieldType) as Array<keyof typeof FieldType>).find(
    (key) => key.toLowerCase() === type.toLowerCase(),
  )

  if (matchingKey) {
    return FieldType[matchingKey]
  }

  return FieldType.String
}

function createEmptyField(): FieldFormValue {
  return {
    clientId: generateFieldClientId(),
    name: '',
    type: FieldType.String,
    required: false,
    description: '',
    defaultValue: '',
    validation: '',
    referenceEntityType: '',
  }
}

function createEmptyFormState(): SchemaFormState {
  return {
    name: '',
    description: '',
    fields: [createEmptyField()],
  }
}

function buildFormStateFromSchema(schema: EntitySchema): SchemaFormState {
  if (!schema.fields || schema.fields.length === 0) {
    return {
      name: schema.name ?? '',
      description: schema.description ?? '',
      fields: [createEmptyField()],
    }
  }

  return {
    name: schema.name ?? '',
    description: schema.description ?? '',
    fields: schema.fields.map((field, index) => ({
      clientId: `${schema.id}-${field.name}-${index}`,
      name: field.name ?? '',
      type: normalizeFieldType(field.type),
      required: field.required ?? false,
      description: field.description ?? '',
      defaultValue: field.default ?? '',
      validation: field.validation ?? '',
      referenceEntityType: field.referenceEntityType ?? '',
    })),
  }
}

function generateFieldClientId(): string {
  return `field-${Math.random().toString(36).slice(2, 10)}-${Date.now()}`
}

function formatRelative(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) {
    return 'unknown'
  }

  const now = Date.now()
  const diff = now - date.getTime()

  const minute = 60_000
  const hour = 60 * minute
  const day = 24 * hour

  if (diff < minute) {
    return 'just now'
  }
  if (diff < hour) {
    const minutes = Math.round(diff / minute)
    return `${minutes} minute${minutes === 1 ? '' : 's'} ago`
  }
  if (diff < day) {
    const hours = Math.round(diff / hour)
    return `${hours} hour${hours === 1 ? '' : 's'} ago`
  }

  return date.toLocaleString()
}

function formatTimestamp(date: Date): string {
  return date.toLocaleString()
}
