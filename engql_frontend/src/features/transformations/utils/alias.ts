import { EntityTransformationNodeType } from '@/generated/graphql'

import type { TransformationCanvasNode } from '../types'

type AliasChange = { oldAlias: string; newAlias: string }

export function sanitizeAlias(input: string | undefined | null): string {
  if (!input) {
    return ''
  }

  const normalized = input
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')

  return normalized
}

export function getNodeAliases(node: TransformationCanvasNode): string[] {
  const aliases: string[] = []
  const { config } = node.data

  if (config.load?.alias) {
    aliases.push(config.load.alias)
  }

  if (config.filter?.alias) {
    aliases.push(config.filter.alias)
  }

  if (config.project?.alias) {
    aliases.push(config.project.alias)
  }

  if (config.join?.leftAlias) {
    aliases.push(config.join.leftAlias)
  }

  if (config.join?.rightAlias) {
    aliases.push(config.join.rightAlias)
  }

  if (config.sort?.alias) {
    aliases.push(config.sort.alias)
  }

  return aliases
}

export function getNodePrimaryAlias(node: TransformationCanvasNode): string | null {
  switch (node.data.type) {
    case EntityTransformationNodeType.Load:
      return node.data.config.load?.alias ?? null
    case EntityTransformationNodeType.Filter:
      return node.data.config.filter?.alias ?? null
    case EntityTransformationNodeType.Project:
      return node.data.config.project?.alias ?? null
    case EntityTransformationNodeType.Sort:
      return node.data.config.sort?.alias ?? null
    default:
      return null
  }
}

export function generateUniqueAlias(
  base: string,
  nodes: TransformationCanvasNode[],
  excludeNodeId?: string,
): string {
  let candidate = sanitizeAlias(base)
  if (!candidate) {
    candidate = 'alias'
  }

  const used = new Set<string>()
  for (const node of nodes) {
    if (node.id === excludeNodeId) {
      continue
    }

    for (const alias of getNodeAliases(node)) {
      const sanitized = sanitizeAlias(alias)
      if (sanitized) {
        used.add(sanitized)
      }
    }
  }

  if (!used.has(candidate)) {
    return candidate
  }

  let index = 1
  let nextCandidate = `${candidate}_${index}`
  while (used.has(nextCandidate)) {
    index += 1
    nextCandidate = `${candidate}_${index}`
  }

  return nextCandidate
}

export function isAliasDerivedFromEntityType(
  alias: string | undefined,
  entityType: string | undefined,
): boolean {
  if (!alias) {
    return true
  }

  const normalizedAlias = sanitizeAlias(alias)
  if (!normalizedAlias || normalizedAlias === 'source') {
    return true
  }

  const base = sanitizeAlias(entityType)
  if (!base) {
    return normalizedAlias === 'source'
  }

  if (normalizedAlias === base) {
    return true
  }

  const suffixPattern = new RegExp(`^${base}_(\\d+)$`)
  return suffixPattern.test(normalizedAlias)
}

export function diffNodeAliases(
  before: TransformationCanvasNode,
  after: TransformationCanvasNode,
): AliasChange[] {
  const beforeMap = collectAliasMap(before)
  const afterMap = collectAliasMap(after)

  const keys = new Set([...Object.keys(beforeMap), ...Object.keys(afterMap)])
  const changes: AliasChange[] = []

  for (const key of keys) {
    const oldAlias = beforeMap[key]
    const newAlias = afterMap[key]

    if (!oldAlias || !newAlias) {
      continue
    }

    if (oldAlias === newAlias) {
      continue
    }

    changes.push({ oldAlias, newAlias })
  }

  return changes
}

export function replaceAliasInNode(
  node: TransformationCanvasNode,
  oldAlias: string,
  newAlias: string,
): TransformationCanvasNode {
  let changed = false
  const { config } = node.data

  const nextConfig = { ...config }

  if (config.filter?.alias === oldAlias) {
    nextConfig.filter = { ...config.filter, alias: newAlias }
    changed = true
  }

  if (config.project?.alias === oldAlias) {
    nextConfig.project = { ...config.project, alias: newAlias }
    changed = true
  }

  if (config.join?.leftAlias === oldAlias) {
    nextConfig.join = { ...config.join, leftAlias: newAlias }
    changed = true
  } else if (config.join?.rightAlias === oldAlias) {
    nextConfig.join = { ...config.join, rightAlias: newAlias }
    changed = true
  }

  if (config.sort?.alias === oldAlias) {
    nextConfig.sort = { ...config.sort, alias: newAlias }
    changed = true
  }

  if (!changed) {
    return node
  }

  return {
    ...node,
    data: {
      ...node.data,
      config: nextConfig,
    },
  }
}

function collectAliasMap(node: TransformationCanvasNode): Record<string, string | undefined> {
  const { config } = node.data
  return {
    load: config.load?.alias,
    filter: config.filter?.alias,
    project: config.project?.alias,
    joinLeft: config.join?.leftAlias,
    joinRight: config.join?.rightAlias,
    sort: config.sort?.alias,
  }
}

export type { AliasChange }
