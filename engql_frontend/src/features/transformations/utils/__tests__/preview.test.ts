import { describe, expect, it } from 'vitest'

import type { EntityTransformationRecordEdge } from '@/generated/graphql'

import { summarizeTransformationEdges } from '../../utils/preview'

function createEntity({
  id,
  entityType,
  properties,
}: {
  id: string
  entityType: string
  properties: Record<string, unknown>
}) {
  return {
    id,
    entityType,
    properties: JSON.stringify(properties),
    path: `/entities/${id}`,
    organizationId: 'org-1',
    schemaId: 'schema-1',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    version: 1,
    linkedEntities: [],
  }
}

describe('summarizeTransformationEdges', () => {
  it('groups entities by alias and extracts sample property values', () => {
    const edges: EntityTransformationRecordEdge[] = [
      {
        entities: [
          {
            alias: 'facility',
            entity: createEntity({
              id: 'f-1',
              entityType: 'Facility',
              properties: { name: 'North Plant', capacity: 12 },
            }),
          },
          {
            alias: 'equipment',
            entity: createEntity({
              id: 'e-1',
              entityType: 'Equipment',
              properties: { model: 'HX-10', settings: { pressure: 4.2 } },
            }),
          },
        ],
      },
      {
        entities: [
          {
            alias: 'facility',
            entity: createEntity({
              id: 'f-2',
              entityType: 'FacilityVersion',
              properties: { status: 'Active' },
            }),
          },
        ],
      },
    ]

    const summaries = summarizeTransformationEdges(edges)

    expect(summaries).toHaveLength(2)

    const facilitySummary = summaries.find((summary) => summary.alias === 'facility')
    expect(facilitySummary).toBeDefined()
    expect(facilitySummary?.entityTypes).toEqual(['Facility', 'FacilityVersion'])
    expect(facilitySummary?.sampleFields).toEqual([
      { key: 'name', value: 'North Plant' },
      { key: 'capacity', value: '12' },
      { key: 'status', value: 'Active' },
    ])

    const equipmentSummary = summaries.find((summary) => summary.alias === 'equipment')
    expect(equipmentSummary).toBeDefined()
    expect(equipmentSummary?.entityTypes).toEqual(['Equipment'])
    expect(equipmentSummary?.sampleFields).toEqual([
      { key: 'model', value: 'HX-10' },
      { key: 'settings', value: '{"pressure":4.2}' },
    ])
  })
})
