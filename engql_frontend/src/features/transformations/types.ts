import type {
  EntityTransformationNodeInput,
  EntityTransformationNodeType,
  PropertyFilter,
  PropertyFilterConfig,
} from '@/generated/graphql'
import type { Edge, Node } from 'reactflow'

export type TransformationNodeConfig = {
  load?: EntityTransformationNodeInput['load']
  filter?: EntityTransformationNodeInput['filter']
  project?: EntityTransformationNodeInput['project']
  join?: EntityTransformationNodeInput['join']
  union?: EntityTransformationNodeInput['union']
  sort?: EntityTransformationNodeInput['sort']
  paginate?: EntityTransformationNodeInput['paginate']
}

export type TransformationNodeData = {
  name: string
  type: EntityTransformationNodeType
  config: TransformationNodeConfig
  validationMessage?: string
  persistedId?: string
}

export type TransformationCanvasNode = Node<TransformationNodeData>
export type TransformationCanvasEdge = Edge

export type TransformationGraphState = {
  nodes: TransformationCanvasNode[]
  edges: TransformationCanvasEdge[]
}

export type PropertyFilterValue = PropertyFilter | PropertyFilterConfig
