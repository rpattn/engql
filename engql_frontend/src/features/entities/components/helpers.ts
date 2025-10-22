import type {
  Entity,
  EntitySchema,
  FieldDefinition,
} from '../../../generated/graphql'
import { FieldType } from '../../../generated/graphql'
import { FieldType } from '../../../generated/graphql'

export type FieldInputValue = string | boolean | string[]

export type EntityFormState = {
  entityType: string
  path: string
  fieldValues: Record<string, FieldInputValue>
  baseProperties: Record<string, unknown>
}

export type ParsedEntityRow = {
  entity: Entity
  props: Record<string, unknown>
  linkedById: Map<string, Entity>
}

export type ColumnFilterValue = string

export function safeParseProperties(json: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(json)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
    return {}
  } catch {
    return {}
  }
}

export function extractEntityDisplayNameFromProperties(
  properties: string,
  fallback: string,
): string {
  const parsed = safeParseProperties(properties)
  const candidate =
    parsed.name ??
    parsed.title ??
    parsed.label ??
    parsed.displayName ??
    null
  if (typeof candidate === 'string' && candidate.trim().length > 0) {
    return candidate.trim()
  }
  return fallback
}

export function createLinkedEntityMap(linkedEntities: Array<Entity>): Map<string, Entity> {
  const map = new Map<string, Entity>()
  for (const linked of linkedEntities) {
    map.set(linked.id, linked)
  }
  return map
}

export function buildInitialEntityFormState(
  schemaFields: FieldDefinition[],
  schema?: EntitySchema,
  entity?: Entity,
): EntityFormState {
  const parsedProps = entity ? safeParseProperties(entity.properties) : {}
  const baseProperties: Record<string, unknown> = {}

  if (entity) {
    for (const [key, value] of Object.entries(parsedProps)) {
      if (!schemaFields.some((field) => field.name === key)) {
        baseProperties[key] = value
      }
    }
  }

  const fieldValues: Record<string, FieldInputValue> = {}
  for (const field of schemaFields) {
    fieldValues[field.name] = formatFieldInputValue(field, parsedProps[field.name])
  }

  return {
    entityType: entity?.entityType ?? schema?.name ?? '',
    path: entity?.path ?? '',
    fieldValues,
    baseProperties,
  }
}

export function formatFieldInputValue(
  field: FieldDefinition,
  value: unknown,
): FieldInputValue {
  if (value === undefined || value === null) {
    return defaultFieldInputValue(field)
  }

  switch (field.type) {
    case FieldType.Boolean:
      return Boolean(value)
    case FieldType.Integer:
    case FieldType.Float:
      return String(value)
    case FieldType.Timestamp:
      if (typeof value === 'string') {
        return formatTimestampForInput(value)
      }
      return ''
    case FieldType.EntityReference:
      return typeof value === 'string' ? value : String(value ?? '')
    case FieldType.EntityReferenceArray:
      if (Array.isArray(value)) {
        return value.map((item) => String(item))
      }
      if (typeof value === 'string') {
        return value ? [value] : []
      }
      return []
    case FieldType.Json:
      try {
        return JSON.stringify(value, null, 2)
      } catch (error) {
        return typeof value === 'string' ? value : ''
      }
    default:
      return typeof value === 'string' ? value : String(value ?? '')
  }
}

export function defaultFieldInputValue(field: FieldDefinition): FieldInputValue {
  if (field.default && field.default.length > 0) {
    switch (field.type) {
      case FieldType.Boolean:
        return parseBooleanString(field.default)
      case FieldType.EntityReferenceArray: {
        try {
          const parsed = JSON.parse(field.default)
          if (Array.isArray(parsed)) {
            return parsed.map((item) => String(item))
          }
        } catch {
          // ignore
        }
        return field.default ? [field.default] : []
      }
      case FieldType.Json:
        try {
          return JSON.stringify(JSON.parse(field.default), null, 2)
        } catch (error) {
          return field.default
        }
      default:
        return field.default
    }
  }

  switch (field.type) {
    case FieldType.Boolean:
      return false
    case FieldType.EntityReferenceArray:
      return []
    case FieldType.Json:
      return ''
    default:
      return ''
  }
}

export function parseBooleanString(value: string): boolean {
  return value.toLowerCase() === 'true'
}

export function formatTimestampForInput(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ''
  }
  const tzOffset = date.getTimezoneOffset() * 60_000
  const localISO = new Date(date.getTime() - tzOffset).toISOString()
  return localISO.slice(0, 16)
}

export type FieldValueResult =
  | { ok: true; value?: unknown }
  | { ok: false; message: string }

export function prepareFieldValueForSubmit(
  field: FieldDefinition,
  rawValue: FieldInputValue | undefined,
): FieldValueResult {
  const isRequired = field.required ?? false

  if (field.type === FieldType.Boolean) {
    return { ok: true, value: Boolean(rawValue) }
  }

  if (field.type === FieldType.EntityReferenceArray) {
    const arrayValue = Array.isArray(rawValue)
      ? rawValue.filter((item): item is string => typeof item === 'string' && item.trim().length > 0)
      : []

    if (arrayValue.length === 0) {
      if (isRequired) {
        return {
          ok: false,
          message: `Field "${field.name}" requires at least one selection.`,
        }
      }
      return { ok: true, value: undefined }
    }

    return { ok: true, value: arrayValue }
  }

  const stringValue = typeof rawValue === 'string' ? rawValue.trim() : ''

  if (!stringValue) {
    if (isRequired) {
      return {
        ok: false,
        message: `Field "${field.name}" is required.`,
      }
    }
    return { ok: true, value: undefined }
  }

  switch (field.type) {
    case FieldType.Integer: {
      const parsed = Number.parseInt(stringValue, 10)
      if (Number.isNaN(parsed)) {
        return {
          ok: false,
          message: `Field "${field.name}" must be an integer.`,
        }
      }
      return { ok: true, value: parsed }
    }
    case FieldType.Float: {
      const parsed = Number.parseFloat(stringValue)
      if (Number.isNaN(parsed)) {
        return {
          ok: false,
          message: `Field "${field.name}" must be a number.`,
        }
      }
      return { ok: true, value: parsed }
    }
    case FieldType.Timestamp: {
      const iso = convertLocalInputToISO(stringValue)
      if (!iso) {
        return {
          ok: false,
          message: `Field "${field.name}" must be a valid date & time.`,
        }
      }
      return { ok: true, value: iso }
    }
    case FieldType.Json: {
      try {
        const parsed = JSON.parse(stringValue)
        return { ok: true, value: parsed }
      } catch (error) {
        return {
          ok: false,
          message: `Field "${field.name}" must contain valid JSON.`,
        }
      }
    }
    default:
      return { ok: true, value: stringValue }
  }
}

export function convertLocalInputToISO(value: string): string | null {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return null
  }
  return new Date(date.getTime()).toISOString()
}

export function formatJsonPreview(value: unknown): string {
  try {
    if (typeof value === 'string') {
      const trimmed = value.trim()
      if (!trimmed) {
        return ''
      }
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    }
    return JSON.stringify(value, null, 2)
  } catch (error) {
    return typeof value === 'string' ? value : ''
  }
}

export function formatJsonPreviewLimited(value: unknown, maxLines = 4): string {
  const formatted = formatJsonPreview(value)
  if (!formatted) {
    return ''
  }
  const lines = formatted.split('\n')
  if (lines.length > maxLines) {
    return [...lines.slice(0, maxLines), ' ...'].join('\n')
  }
  return formatted
}

export function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) {
    return 'unknown'
  }
  return date.toLocaleString()
}

export function formatRelative(timestamp: string): string {
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) {
    return 'unknown'
  }

  const diff = Date.now() - date.getTime()
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
  const days = Math.round(diff / day)
  if (days < 7) {
    return `${days} day${days === 1 ? '' : 's'} ago`
  }

  return date.toLocaleDateString()
}
