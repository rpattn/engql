import { EntityTransformationNodeType } from '@/generated/graphql'

export function formatNodeType(type: EntityTransformationNodeType) {
  return type
    .toLowerCase()
    .split('_')
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
}
