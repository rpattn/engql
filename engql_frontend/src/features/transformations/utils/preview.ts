import type { EntityTransformationRecordEdge } from '@/generated/graphql'

export type TransformationAliasSummary = {
  alias: string
  entityTypes: string[]
  sampleFields: Array<{ key: string; value: string }>
}

function formatPropertyValue(value: unknown): string {
  if (typeof value === 'string') {
    return value
  }

  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function extractPropertySamples(raw: string | null | undefined): Record<string, string> {
  if (!raw) {
    return {}
  }

  try {
    const parsed = JSON.parse(raw)

    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return Object.entries(parsed).reduce<Record<string, string>>((acc, [key, value]) => {
        acc[key] = formatPropertyValue(value)
        return acc
      }, {})
    }

    return { value: formatPropertyValue(parsed) }
  } catch {
    return { value: raw }
  }
}

export function summarizeTransformationEdges(
  edges: EntityTransformationRecordEdge[] | undefined,
): TransformationAliasSummary[] {
  if (!edges?.length) {
    return []
  }

  const summaries = new Map<
    string,
    {
      entityTypes: Set<string>
      fields: Map<string, string>
    }
  >()

  for (const edge of edges) {
    for (const record of edge.entities) {
      const { alias, entity } = record

      if (!summaries.has(alias)) {
        summaries.set(alias, {
          entityTypes: new Set<string>(),
          fields: new Map<string, string>(),
        })
      }

      const summary = summaries.get(alias)!

      if (entity?.entityType) {
        summary.entityTypes.add(entity.entityType)
      }

      if (entity?.properties) {
        const samples = extractPropertySamples(entity.properties)
        for (const [key, value] of Object.entries(samples)) {
          if (!summary.fields.has(key)) {
            summary.fields.set(key, value)
          }
        }
      }
    }
  }

  return Array.from(summaries.entries())
    .map(([alias, { entityTypes, fields }]) => ({
      alias,
      entityTypes: Array.from(entityTypes).sort((a, b) => a.localeCompare(b)),
      sampleFields: Array.from(fields.entries()).map(([key, value]) => ({ key, value })),
    }))
    .sort((a, b) => a.alias.localeCompare(b.alias))
}
