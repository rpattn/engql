import { useQuery, useMutation, UseQueryOptions, UseMutationOptions } from '@tanstack/react-query';
import { graphqlRequest } from '../lib/graphql';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
};

export type CreateEntityInput = {
  entityType: Scalars['String']['input'];
  linkedEntityId?: InputMaybe<Scalars['String']['input']>;
  linkedEntityIds?: InputMaybe<Array<Scalars['String']['input']>>;
  linkedEntityReference?: InputMaybe<Scalars['String']['input']>;
  linkedEntityReferences?: InputMaybe<Array<Scalars['String']['input']>>;
  organizationId: Scalars['String']['input'];
  path?: InputMaybe<Scalars['String']['input']>;
  properties: Scalars['String']['input'];
};

export type CreateEntityJoinDefinitionInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  joinField?: InputMaybe<Scalars['String']['input']>;
  joinType?: InputMaybe<JoinType>;
  leftEntityType: Scalars['String']['input'];
  leftFilters?: InputMaybe<Array<PropertyFilter>>;
  name: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
  rightEntityType: Scalars['String']['input'];
  rightFilters?: InputMaybe<Array<PropertyFilter>>;
  sortCriteria?: InputMaybe<Array<JoinSortInput>>;
};

export type CreateEntitySchemaInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  fields: Array<FieldDefinitionInput>;
  name: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
};

export type CreateEntityTransformationInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
  nodes: Array<EntityTransformationNodeInput>;
  organizationId: Scalars['String']['input'];
};

export type CreateOrganizationInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
};

export type Entity = {
  __typename?: 'Entity';
  createdAt: Scalars['String']['output'];
  entityType: Scalars['String']['output'];
  id: Scalars['String']['output'];
  linkedEntities: Array<Entity>;
  organizationId: Scalars['String']['output'];
  path: Scalars['String']['output'];
  properties: Scalars['String']['output'];
  referenceValue?: Maybe<Scalars['String']['output']>;
  schemaId: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
  version: Scalars['Int']['output'];
};

export type EntityConnection = {
  __typename?: 'EntityConnection';
  entities: Array<Entity>;
  pageInfo: PageInfo;
};

export type EntityDiffResult = {
  __typename?: 'EntityDiffResult';
  base?: Maybe<EntitySnapshotView>;
  target?: Maybe<EntitySnapshotView>;
  unifiedDiff?: Maybe<Scalars['String']['output']>;
};

export type EntityExportJob = {
  __typename?: 'EntityExportJob';
  bytesWritten: Scalars['Int']['output'];
  completedAt?: Maybe<Scalars['String']['output']>;
  downloadUrl?: Maybe<Scalars['String']['output']>;
  enqueuedAt: Scalars['String']['output'];
  entityType?: Maybe<Scalars['String']['output']>;
  errorMessage?: Maybe<Scalars['String']['output']>;
  fileByteSize?: Maybe<Scalars['Int']['output']>;
  fileMimeType?: Maybe<Scalars['String']['output']>;
  filters: Array<PropertyFilterConfig>;
  id: Scalars['String']['output'];
  jobType: EntityExportJobType;
  organizationId: Scalars['String']['output'];
  rowsExported: Scalars['Int']['output'];
  rowsRequested: Scalars['Int']['output'];
  startedAt?: Maybe<Scalars['String']['output']>;
  status: EntityExportJobStatus;
  transformationDefinition?: Maybe<EntityTransformation>;
  transformationId?: Maybe<Scalars['String']['output']>;
  updatedAt: Scalars['String']['output'];
};

export enum EntityExportJobStatus {
  Cancelled = 'CANCELLED',
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Pending = 'PENDING',
  Running = 'RUNNING'
}

export enum EntityExportJobType {
  EntityType = 'ENTITY_TYPE',
  Transformation = 'TRANSFORMATION'
}

export type EntityExportLog = {
  __typename?: 'EntityExportLog';
  createdAt: Scalars['String']['output'];
  errorMessage: Scalars['String']['output'];
  exportJobId: Scalars['String']['output'];
  id: Scalars['String']['output'];
  organizationId: Scalars['String']['output'];
  rowIdentifier?: Maybe<Scalars['String']['output']>;
};

export type EntityFilter = {
  entityType?: InputMaybe<Scalars['String']['input']>;
  pathFilter?: InputMaybe<PathFilter>;
  propertyFilters?: InputMaybe<Array<PropertyFilter>>;
  textSearch?: InputMaybe<Scalars['String']['input']>;
};

export type EntityHierarchy = {
  __typename?: 'EntityHierarchy';
  ancestors: Array<Entity>;
  children: Array<Entity>;
  current: Entity;
  siblings: Array<Entity>;
};

export type EntityJoinConnection = {
  __typename?: 'EntityJoinConnection';
  edges: Array<EntityJoinEdge>;
  pageInfo: PageInfo;
};

export type EntityJoinDefinition = {
  __typename?: 'EntityJoinDefinition';
  createdAt: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['String']['output'];
  joinField?: Maybe<Scalars['String']['output']>;
  joinFieldType?: Maybe<FieldType>;
  joinType: JoinType;
  leftEntityType: Scalars['String']['output'];
  leftFilters: Array<PropertyFilterConfig>;
  name: Scalars['String']['output'];
  organizationId: Scalars['String']['output'];
  rightEntityType: Scalars['String']['output'];
  rightFilters: Array<PropertyFilterConfig>;
  sortCriteria: Array<JoinSortCriterion>;
  updatedAt: Scalars['String']['output'];
};

export type EntityJoinEdge = {
  __typename?: 'EntityJoinEdge';
  left: Entity;
  right: Entity;
};

export type EntitySchema = {
  __typename?: 'EntitySchema';
  createdAt: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  fields: Array<FieldDefinition>;
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
  organizationId: Scalars['String']['output'];
  previousVersionId?: Maybe<Scalars['String']['output']>;
  status: SchemaStatus;
  updatedAt: Scalars['String']['output'];
  version: Scalars['String']['output'];
};

export type EntitySnapshotView = {
  __typename?: 'EntitySnapshotView';
  canonicalText: Array<Scalars['String']['output']>;
  entityType: Scalars['String']['output'];
  path: Scalars['String']['output'];
  schemaId: Scalars['String']['output'];
  version: Scalars['Int']['output'];
};

export enum EntitySortField {
  CreatedAt = 'CREATED_AT',
  EntityType = 'ENTITY_TYPE',
  Path = 'PATH',
  Property = 'PROPERTY',
  UpdatedAt = 'UPDATED_AT',
  Version = 'VERSION'
}

export type EntitySortInput = {
  direction?: InputMaybe<SortDirection>;
  field: EntitySortField;
  propertyKey?: InputMaybe<Scalars['String']['input']>;
};

export type EntityTransformation = {
  __typename?: 'EntityTransformation';
  createdAt: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
  nodes: Array<EntityTransformationNode>;
  organizationId: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type EntityTransformationConnection = {
  __typename?: 'EntityTransformationConnection';
  edges: Array<EntityTransformationRecordEdge>;
  pageInfo: PageInfo;
};

export type EntityTransformationFilterConfig = {
  __typename?: 'EntityTransformationFilterConfig';
  alias: Scalars['String']['output'];
  filters: Array<PropertyFilterConfig>;
};

export type EntityTransformationFilterConfigInput = {
  alias: Scalars['String']['input'];
  filters?: InputMaybe<Array<PropertyFilter>>;
};

export type EntityTransformationJoinConfig = {
  __typename?: 'EntityTransformationJoinConfig';
  leftAlias: Scalars['String']['output'];
  onField: Scalars['String']['output'];
  rightAlias: Scalars['String']['output'];
};

export type EntityTransformationJoinConfigInput = {
  leftAlias: Scalars['String']['input'];
  onField: Scalars['String']['input'];
  rightAlias: Scalars['String']['input'];
};

export type EntityTransformationLoadConfig = {
  __typename?: 'EntityTransformationLoadConfig';
  alias: Scalars['String']['output'];
  entityType: Scalars['String']['output'];
  filters: Array<PropertyFilterConfig>;
};

export type EntityTransformationLoadConfigInput = {
  alias: Scalars['String']['input'];
  entityType: Scalars['String']['input'];
  filters?: InputMaybe<Array<PropertyFilter>>;
};

export type EntityTransformationMaterializeConfig = {
  __typename?: 'EntityTransformationMaterializeConfig';
  outputs: Array<EntityTransformationMaterializeOutput>;
};

export type EntityTransformationMaterializeConfigInput = {
  outputs: Array<EntityTransformationMaterializeOutputInput>;
};

export type EntityTransformationMaterializeFieldMapping = {
  __typename?: 'EntityTransformationMaterializeFieldMapping';
  outputField: Scalars['String']['output'];
  sourceAlias: Scalars['String']['output'];
  sourceField: Scalars['String']['output'];
};

export type EntityTransformationMaterializeFieldMappingInput = {
  outputField: Scalars['String']['input'];
  sourceAlias: Scalars['String']['input'];
  sourceField: Scalars['String']['input'];
};

export type EntityTransformationMaterializeOutput = {
  __typename?: 'EntityTransformationMaterializeOutput';
  alias: Scalars['String']['output'];
  fields: Array<EntityTransformationMaterializeFieldMapping>;
};

export type EntityTransformationMaterializeOutputInput = {
  alias: Scalars['String']['input'];
  fields: Array<EntityTransformationMaterializeFieldMappingInput>;
};

export type EntityTransformationNode = {
  __typename?: 'EntityTransformationNode';
  filter?: Maybe<EntityTransformationFilterConfig>;
  id: Scalars['String']['output'];
  inputs: Array<Scalars['String']['output']>;
  join?: Maybe<EntityTransformationJoinConfig>;
  load?: Maybe<EntityTransformationLoadConfig>;
  materialize?: Maybe<EntityTransformationMaterializeConfig>;
  name: Scalars['String']['output'];
  paginate?: Maybe<EntityTransformationPaginateConfig>;
  project?: Maybe<EntityTransformationProjectConfig>;
  sort?: Maybe<EntityTransformationSortConfig>;
  type: EntityTransformationNodeType;
};

export type EntityTransformationNodeInput = {
  filter?: InputMaybe<EntityTransformationFilterConfigInput>;
  id?: InputMaybe<Scalars['String']['input']>;
  inputs?: InputMaybe<Array<Scalars['String']['input']>>;
  join?: InputMaybe<EntityTransformationJoinConfigInput>;
  load?: InputMaybe<EntityTransformationLoadConfigInput>;
  materialize?: InputMaybe<EntityTransformationMaterializeConfigInput>;
  name: Scalars['String']['input'];
  paginate?: InputMaybe<EntityTransformationPaginateConfigInput>;
  project?: InputMaybe<EntityTransformationProjectConfigInput>;
  sort?: InputMaybe<EntityTransformationSortConfigInput>;
  type: EntityTransformationNodeType;
};

export enum EntityTransformationNodeType {
  AntiJoin = 'ANTI_JOIN',
  Filter = 'FILTER',
  Join = 'JOIN',
  LeftJoin = 'LEFT_JOIN',
  Load = 'LOAD',
  Materialize = 'MATERIALIZE',
  Paginate = 'PAGINATE',
  Project = 'PROJECT',
  Sort = 'SORT',
  Union = 'UNION'
}

export type EntityTransformationPaginateConfig = {
  __typename?: 'EntityTransformationPaginateConfig';
  limit?: Maybe<Scalars['Int']['output']>;
  offset?: Maybe<Scalars['Int']['output']>;
};

export type EntityTransformationPaginateConfigInput = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};

export type EntityTransformationProjectConfig = {
  __typename?: 'EntityTransformationProjectConfig';
  alias: Scalars['String']['output'];
  fields: Array<Scalars['String']['output']>;
};

export type EntityTransformationProjectConfigInput = {
  alias: Scalars['String']['input'];
  fields: Array<Scalars['String']['input']>;
};

export type EntityTransformationRecordEdge = {
  __typename?: 'EntityTransformationRecordEdge';
  entities: Array<EntityTransformationRecordEntity>;
};

export type EntityTransformationRecordEntity = {
  __typename?: 'EntityTransformationRecordEntity';
  alias: Scalars['String']['output'];
  entity?: Maybe<Entity>;
};

export type EntityTransformationSortConfig = {
  __typename?: 'EntityTransformationSortConfig';
  alias: Scalars['String']['output'];
  direction: JoinSortDirection;
  field: Scalars['String']['output'];
};

export type EntityTransformationSortConfigInput = {
  alias: Scalars['String']['input'];
  direction: JoinSortDirection;
  field: Scalars['String']['input'];
};

export type ExecuteEntityJoinInput = {
  joinId: Scalars['String']['input'];
  leftFilters?: InputMaybe<Array<PropertyFilter>>;
  pagination?: InputMaybe<PaginationInput>;
  rightFilters?: InputMaybe<Array<PropertyFilter>>;
  sortCriteria?: InputMaybe<Array<JoinSortInput>>;
};

export type ExecuteEntityTransformationInput = {
  pagination?: InputMaybe<PaginationInput>;
  transformationId: Scalars['String']['input'];
};

export type FieldDefinition = {
  __typename?: 'FieldDefinition';
  default?: Maybe<Scalars['String']['output']>;
  description?: Maybe<Scalars['String']['output']>;
  name: Scalars['String']['output'];
  referenceEntityType?: Maybe<Scalars['String']['output']>;
  required: Scalars['Boolean']['output'];
  type: FieldType;
  validation?: Maybe<Scalars['String']['output']>;
};

export type FieldDefinitionInput = {
  default?: InputMaybe<Scalars['String']['input']>;
  description?: InputMaybe<Scalars['String']['input']>;
  name: Scalars['String']['input'];
  referenceEntityType?: InputMaybe<Scalars['String']['input']>;
  required?: InputMaybe<Scalars['Boolean']['input']>;
  type: FieldType;
  validation?: InputMaybe<Scalars['String']['input']>;
};

export enum FieldType {
  Boolean = 'BOOLEAN',
  EntityReference = 'ENTITY_REFERENCE',
  EntityReferenceArray = 'ENTITY_REFERENCE_ARRAY',
  FileReference = 'FILE_REFERENCE',
  Float = 'FLOAT',
  Geometry = 'GEOMETRY',
  Integer = 'INTEGER',
  Json = 'JSON',
  Reference = 'REFERENCE',
  String = 'STRING',
  Timeseries = 'TIMESERIES',
  Timestamp = 'TIMESTAMP'
}

export enum JoinSide {
  Left = 'LEFT',
  Right = 'RIGHT'
}

export type JoinSortCriterion = {
  __typename?: 'JoinSortCriterion';
  direction: JoinSortDirection;
  field: Scalars['String']['output'];
  side: JoinSide;
};

export enum JoinSortDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type JoinSortInput = {
  direction?: InputMaybe<JoinSortDirection>;
  field: Scalars['String']['input'];
  side: JoinSide;
};

export enum JoinType {
  Cross = 'CROSS',
  Reference = 'REFERENCE'
}

export type Mutation = {
  __typename?: 'Mutation';
  addFieldToSchema: EntitySchema;
  cancelEntityExportJob: EntityExportJob;
  createEntity: Entity;
  createEntityJoinDefinition: EntityJoinDefinition;
  createEntitySchema: EntitySchema;
  createEntityTransformation: EntityTransformation;
  createOrganization: Organization;
  deleteEntity: Scalars['Boolean']['output'];
  deleteEntityJoinDefinition: Scalars['Boolean']['output'];
  deleteEntitySchema: Scalars['Boolean']['output'];
  deleteEntityTransformation: Scalars['Boolean']['output'];
  deleteOrganization: Scalars['Boolean']['output'];
  queueEntityTypeExport: EntityExportJob;
  queueTransformationExport: EntityExportJob;
  removeFieldFromSchema: EntitySchema;
  rollbackEntity: Entity;
  updateEntity: Entity;
  updateEntityJoinDefinition: EntityJoinDefinition;
  updateEntitySchema: EntitySchema;
  updateEntityTransformation: EntityTransformation;
  updateOrganization: Organization;
};


export type MutationAddFieldToSchemaArgs = {
  field: FieldDefinitionInput;
  schemaId: Scalars['String']['input'];
};


export type MutationCancelEntityExportJobArgs = {
  id: Scalars['String']['input'];
};


export type MutationCreateEntityArgs = {
  input: CreateEntityInput;
};


export type MutationCreateEntityJoinDefinitionArgs = {
  input: CreateEntityJoinDefinitionInput;
};


export type MutationCreateEntitySchemaArgs = {
  input: CreateEntitySchemaInput;
};


export type MutationCreateEntityTransformationArgs = {
  input: CreateEntityTransformationInput;
};


export type MutationCreateOrganizationArgs = {
  input: CreateOrganizationInput;
};


export type MutationDeleteEntityArgs = {
  id: Scalars['String']['input'];
};


export type MutationDeleteEntityJoinDefinitionArgs = {
  id: Scalars['String']['input'];
};


export type MutationDeleteEntitySchemaArgs = {
  id: Scalars['String']['input'];
};


export type MutationDeleteEntityTransformationArgs = {
  id: Scalars['String']['input'];
};


export type MutationDeleteOrganizationArgs = {
  id: Scalars['String']['input'];
};


export type MutationQueueEntityTypeExportArgs = {
  input: QueueEntityTypeExportInput;
};


export type MutationQueueTransformationExportArgs = {
  input: QueueTransformationExportInput;
};


export type MutationRemoveFieldFromSchemaArgs = {
  fieldName: Scalars['String']['input'];
  schemaId: Scalars['String']['input'];
};


export type MutationRollbackEntityArgs = {
  id: Scalars['String']['input'];
  reason?: InputMaybe<Scalars['String']['input']>;
  toVersion: Scalars['Int']['input'];
};


export type MutationUpdateEntityArgs = {
  input: UpdateEntityInput;
};


export type MutationUpdateEntityJoinDefinitionArgs = {
  input: UpdateEntityJoinDefinitionInput;
};


export type MutationUpdateEntitySchemaArgs = {
  input: UpdateEntitySchemaInput;
};


export type MutationUpdateEntityTransformationArgs = {
  input: UpdateEntityTransformationInput;
};


export type MutationUpdateOrganizationArgs = {
  input: UpdateOrganizationInput;
};

export type Organization = {
  __typename?: 'Organization';
  createdAt: Scalars['String']['output'];
  description?: Maybe<Scalars['String']['output']>;
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
  updatedAt: Scalars['String']['output'];
};

export type PageInfo = {
  __typename?: 'PageInfo';
  hasNextPage: Scalars['Boolean']['output'];
  hasPreviousPage: Scalars['Boolean']['output'];
  totalCount: Scalars['Int']['output'];
};

export type PaginationInput = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};

export type PathFilter = {
  ancestorsOf?: InputMaybe<Scalars['String']['input']>;
  childrenOf?: InputMaybe<Scalars['String']['input']>;
  descendantsOf?: InputMaybe<Scalars['String']['input']>;
  siblingsOf?: InputMaybe<Scalars['String']['input']>;
};

export type PropertyFilter = {
  exists?: InputMaybe<Scalars['Boolean']['input']>;
  inArray?: InputMaybe<Array<Scalars['String']['input']>>;
  key: Scalars['String']['input'];
  value?: InputMaybe<Scalars['String']['input']>;
};

export type PropertyFilterConfig = {
  __typename?: 'PropertyFilterConfig';
  exists?: Maybe<Scalars['Boolean']['output']>;
  inArray?: Maybe<Array<Scalars['String']['output']>>;
  key: Scalars['String']['output'];
  value?: Maybe<Scalars['String']['output']>;
};

export type Query = {
  __typename?: 'Query';
  entities: EntityConnection;
  entitiesByIDs: Array<Entity>;
  entitiesByType: Array<Entity>;
  entity?: Maybe<Entity>;
  entityDiff?: Maybe<EntityDiffResult>;
  entityExportJob?: Maybe<EntityExportJob>;
  entityExportJobs: Array<EntityExportJob>;
  entityHistory: Array<EntitySnapshotView>;
  entityJoinDefinition?: Maybe<EntityJoinDefinition>;
  entityJoinDefinitions: Array<EntityJoinDefinition>;
  entitySchema?: Maybe<EntitySchema>;
  entitySchemaByName?: Maybe<EntitySchema>;
  entitySchemaVersions: Array<EntitySchema>;
  entitySchemas: Array<EntitySchema>;
  entityTransformation?: Maybe<EntityTransformation>;
  entityTransformations: Array<EntityTransformation>;
  executeEntityJoin: EntityJoinConnection;
  executeEntityTransformation: EntityTransformationConnection;
  getEntityAncestors: Array<Entity>;
  getEntityChildren: Array<Entity>;
  getEntityDescendants: Array<Entity>;
  getEntityHierarchy: EntityHierarchy;
  getEntitySiblings: Array<Entity>;
  organization?: Maybe<Organization>;
  organizationByName?: Maybe<Organization>;
  organizations: Array<Organization>;
  searchEntitiesByMultipleProperties: Array<Entity>;
  searchEntitiesByProperty: Array<Entity>;
  searchEntitiesByPropertyContains: Array<Entity>;
  searchEntitiesByPropertyExists: Array<Entity>;
  searchEntitiesByPropertyRange: Array<Entity>;
  transformationExecution: TransformationExecutionConnection;
  validateEntityAgainstSchema: ValidationResult;
};


export type QueryEntitiesArgs = {
  filter?: InputMaybe<EntityFilter>;
  organizationId: Scalars['String']['input'];
  pagination?: InputMaybe<PaginationInput>;
  sort?: InputMaybe<EntitySortInput>;
};


export type QueryEntitiesByIDsArgs = {
  ids: Array<Scalars['String']['input']>;
};


export type QueryEntitiesByTypeArgs = {
  entityType: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
};


export type QueryEntityArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntityDiffArgs = {
  baseVersion: Scalars['Int']['input'];
  id: Scalars['String']['input'];
  targetVersion: Scalars['Int']['input'];
};


export type QueryEntityExportJobArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntityExportJobsArgs = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
  organizationId: Scalars['String']['input'];
  statuses?: InputMaybe<Array<EntityExportJobStatus>>;
};


export type QueryEntityHistoryArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntityJoinDefinitionArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntityJoinDefinitionsArgs = {
  organizationId: Scalars['String']['input'];
};


export type QueryEntitySchemaArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntitySchemaByNameArgs = {
  name: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
};


export type QueryEntitySchemaVersionsArgs = {
  name: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
};


export type QueryEntitySchemasArgs = {
  organizationId: Scalars['String']['input'];
};


export type QueryEntityTransformationArgs = {
  id: Scalars['String']['input'];
};


export type QueryEntityTransformationsArgs = {
  organizationId: Scalars['String']['input'];
};


export type QueryExecuteEntityJoinArgs = {
  input: ExecuteEntityJoinInput;
};


export type QueryExecuteEntityTransformationArgs = {
  input: ExecuteEntityTransformationInput;
};


export type QueryGetEntityAncestorsArgs = {
  entityId: Scalars['String']['input'];
};


export type QueryGetEntityChildrenArgs = {
  entityId: Scalars['String']['input'];
};


export type QueryGetEntityDescendantsArgs = {
  entityId: Scalars['String']['input'];
};


export type QueryGetEntityHierarchyArgs = {
  entityId: Scalars['String']['input'];
};


export type QueryGetEntitySiblingsArgs = {
  entityId: Scalars['String']['input'];
};


export type QueryOrganizationArgs = {
  id: Scalars['String']['input'];
};


export type QueryOrganizationByNameArgs = {
  name: Scalars['String']['input'];
};


export type QuerySearchEntitiesByMultiplePropertiesArgs = {
  filters: Scalars['String']['input'];
  organizationId: Scalars['String']['input'];
};


export type QuerySearchEntitiesByPropertyArgs = {
  organizationId: Scalars['String']['input'];
  propertyKey: Scalars['String']['input'];
  propertyValue: Scalars['String']['input'];
};


export type QuerySearchEntitiesByPropertyContainsArgs = {
  organizationId: Scalars['String']['input'];
  propertyKey: Scalars['String']['input'];
  searchTerm: Scalars['String']['input'];
};


export type QuerySearchEntitiesByPropertyExistsArgs = {
  organizationId: Scalars['String']['input'];
  propertyKey: Scalars['String']['input'];
};


export type QuerySearchEntitiesByPropertyRangeArgs = {
  maxValue?: InputMaybe<Scalars['Float']['input']>;
  minValue?: InputMaybe<Scalars['Float']['input']>;
  organizationId: Scalars['String']['input'];
  propertyKey: Scalars['String']['input'];
};


export type QueryTransformationExecutionArgs = {
  filters?: InputMaybe<Array<TransformationExecutionFilterInput>>;
  pagination?: InputMaybe<PaginationInput>;
  sort?: InputMaybe<TransformationExecutionSortInput>;
  transformationId: Scalars['String']['input'];
};


export type QueryValidateEntityAgainstSchemaArgs = {
  entityId: Scalars['String']['input'];
};

export type QueueEntityTypeExportInput = {
  entityType: Scalars['String']['input'];
  filters?: InputMaybe<Array<PropertyFilter>>;
  organizationId: Scalars['String']['input'];
};

export type QueueTransformationExportInput = {
  filters?: InputMaybe<Array<PropertyFilter>>;
  options?: InputMaybe<TransformationExecutionOptionsInput>;
  organizationId: Scalars['String']['input'];
  transformationId: Scalars['String']['input'];
};

export enum SchemaStatus {
  Active = 'ACTIVE',
  Archived = 'ARCHIVED',
  Deprecated = 'DEPRECATED',
  Draft = 'DRAFT'
}

export enum SortDirection {
  Asc = 'ASC',
  Desc = 'DESC'
}

export type TransformationExecutionColumn = {
  __typename?: 'TransformationExecutionColumn';
  alias: Scalars['String']['output'];
  field: Scalars['String']['output'];
  key: Scalars['String']['output'];
  label: Scalars['String']['output'];
  sourceAlias: Scalars['String']['output'];
  sourceField: Scalars['String']['output'];
};

export type TransformationExecutionConnection = {
  __typename?: 'TransformationExecutionConnection';
  columns: Array<TransformationExecutionColumn>;
  pageInfo: PageInfo;
  rows: Array<TransformationExecutionRow>;
};

export type TransformationExecutionFilterInput = {
  alias: Scalars['String']['input'];
  exists?: InputMaybe<Scalars['Boolean']['input']>;
  field: Scalars['String']['input'];
  inArray?: InputMaybe<Array<Scalars['String']['input']>>;
  value?: InputMaybe<Scalars['String']['input']>;
};

export type TransformationExecutionOptionsInput = {
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
};

export type TransformationExecutionRow = {
  __typename?: 'TransformationExecutionRow';
  values: Array<TransformationExecutionValue>;
};

export type TransformationExecutionSortInput = {
  alias: Scalars['String']['input'];
  direction?: InputMaybe<SortDirection>;
  field: Scalars['String']['input'];
};

export type TransformationExecutionValue = {
  __typename?: 'TransformationExecutionValue';
  columnKey: Scalars['String']['output'];
  value?: Maybe<Scalars['String']['output']>;
};

export type UpdateEntityInput = {
  entityType?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['String']['input'];
  path?: InputMaybe<Scalars['String']['input']>;
  properties?: InputMaybe<Scalars['String']['input']>;
};

export type UpdateEntityJoinDefinitionInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['String']['input'];
  joinField?: InputMaybe<Scalars['String']['input']>;
  joinType?: InputMaybe<JoinType>;
  leftEntityType?: InputMaybe<Scalars['String']['input']>;
  leftFilters?: InputMaybe<Array<PropertyFilter>>;
  name?: InputMaybe<Scalars['String']['input']>;
  rightEntityType?: InputMaybe<Scalars['String']['input']>;
  rightFilters?: InputMaybe<Array<PropertyFilter>>;
  sortCriteria?: InputMaybe<Array<JoinSortInput>>;
};

export type UpdateEntitySchemaInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  fields?: InputMaybe<Array<FieldDefinitionInput>>;
  id: Scalars['String']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
};

export type UpdateEntityTransformationInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['String']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
  nodes?: InputMaybe<Array<EntityTransformationNodeInput>>;
};

export type UpdateOrganizationInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['String']['input'];
  name?: InputMaybe<Scalars['String']['input']>;
};

export type ValidationResult = {
  __typename?: 'ValidationResult';
  errors: Array<Scalars['String']['output']>;
  isValid: Scalars['Boolean']['output'];
  warnings: Array<Scalars['String']['output']>;
};

export type GetOrganizationsQueryVariables = Exact<{ [key: string]: never; }>;


export type GetOrganizationsQuery = { __typename?: 'Query', organizations: Array<{ __typename?: 'Organization', id: string, name: string, description?: string | null }> };

export type GetEntitiesByOrgQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
}>;


export type GetEntitiesByOrgQuery = { __typename?: 'Query', entities: { __typename?: 'EntityConnection', entities: Array<{ __typename?: 'Entity', id: string, entityType: string, version: number, path: string, properties: string, referenceValue?: string | null, createdAt: string, updatedAt: string }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

export type CreateSchemaMutationVariables = Exact<{
  input: CreateEntitySchemaInput;
}>;


export type CreateSchemaMutation = { __typename?: 'Mutation', createEntitySchema: { __typename?: 'EntitySchema', id: string, name: string, description?: string | null, version: string, status: SchemaStatus, previousVersionId?: string | null } };

export type CreateEntityMutationVariables = Exact<{
  input: CreateEntityInput;
}>;


export type CreateEntityMutation = { __typename?: 'Mutation', createEntity: { __typename?: 'Entity', id: string, entityType: string, schemaId: string, version: number, properties: string } };

export type UpdateEntityMutationVariables = Exact<{
  input: UpdateEntityInput;
}>;


export type UpdateEntityMutation = { __typename?: 'Mutation', updateEntity: { __typename?: 'Entity', id: string, organizationId: string, schemaId: string, entityType: string, path: string, properties: string, version: number, updatedAt: string } };

export type QueueEntityTypeExportMutationVariables = Exact<{
  input: QueueEntityTypeExportInput;
}>;


export type QueueEntityTypeExportMutation = { __typename?: 'Mutation', queueEntityTypeExport: { __typename?: 'EntityExportJob', id: string, organizationId: string, jobType: EntityExportJobType, entityType?: string | null, transformationId?: string | null, status: EntityExportJobStatus, rowsRequested: number, rowsExported: number, bytesWritten: number, fileMimeType?: string | null, fileByteSize?: number | null, errorMessage?: string | null, enqueuedAt: string, startedAt?: string | null, completedAt?: string | null, updatedAt: string, downloadUrl?: string | null, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, transformationDefinition?: { __typename?: 'EntityTransformation', id: string, name: string } | null } };

export type QueueTransformationExportMutationVariables = Exact<{
  input: QueueTransformationExportInput;
}>;


export type QueueTransformationExportMutation = { __typename?: 'Mutation', queueTransformationExport: { __typename?: 'EntityExportJob', id: string, organizationId: string, jobType: EntityExportJobType, entityType?: string | null, transformationId?: string | null, status: EntityExportJobStatus, rowsRequested: number, rowsExported: number, bytesWritten: number, fileMimeType?: string | null, fileByteSize?: number | null, errorMessage?: string | null, enqueuedAt: string, startedAt?: string | null, completedAt?: string | null, updatedAt: string, downloadUrl?: string | null, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, transformationDefinition?: { __typename?: 'EntityTransformation', id: string, name: string } | null } };

export type CancelEntityExportJobMutationVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type CancelEntityExportJobMutation = { __typename?: 'Mutation', cancelEntityExportJob: { __typename?: 'EntityExportJob', id: string, organizationId: string, jobType: EntityExportJobType, entityType?: string | null, transformationId?: string | null, status: EntityExportJobStatus, rowsRequested: number, rowsExported: number, bytesWritten: number, fileMimeType?: string | null, fileByteSize?: number | null, errorMessage?: string | null, enqueuedAt: string, startedAt?: string | null, completedAt?: string | null, updatedAt: string, downloadUrl?: string | null, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, transformationDefinition?: { __typename?: 'EntityTransformation', id: string, name: string } | null } };

export type DeleteEntityMutationVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type DeleteEntityMutation = { __typename?: 'Mutation', deleteEntity: boolean };

export type EntitiesByTypeFullQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  entityType: Scalars['String']['input'];
}>;


export type EntitiesByTypeFullQuery = { __typename?: 'Query', entitiesByType: Array<{ __typename?: 'Entity', id: string, entityType: string, schemaId: string, version: number, properties: string, referenceValue?: string | null, linkedEntities: Array<{ __typename?: 'Entity', id: string, entityType: string, properties: string, referenceValue?: string | null }> }> };

export type SchemaVersionsQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  name: Scalars['String']['input'];
}>;


export type SchemaVersionsQuery = { __typename?: 'Query', entitySchemaVersions: Array<{ __typename?: 'EntitySchema', id: string, version: string, status: SchemaStatus, previousVersionId?: string | null, createdAt: string }> };

export type EntitySchemasQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
}>;


export type EntitySchemasQuery = { __typename?: 'Query', entitySchemas: Array<{ __typename?: 'EntitySchema', id: string, organizationId: string, name: string, description?: string | null, status: SchemaStatus, version: string, createdAt: string, updatedAt: string, previousVersionId?: string | null, fields: Array<{ __typename?: 'FieldDefinition', name: string, type: FieldType, required: boolean, description?: string | null, default?: string | null, validation?: string | null, referenceEntityType?: string | null }> }> };

export type EntitiesManagementQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  pagination?: InputMaybe<PaginationInput>;
  filter?: InputMaybe<EntityFilter>;
  includeLinkedEntities?: Scalars['Boolean']['input'];
  sort?: InputMaybe<EntitySortInput>;
}>;


export type EntitiesManagementQuery = { __typename?: 'Query', entities: { __typename?: 'EntityConnection', entities: Array<{ __typename?: 'Entity', id: string, organizationId: string, schemaId: string, entityType: string, path: string, properties: string, referenceValue?: string | null, version: number, createdAt: string, updatedAt: string, linkedEntities?: Array<{ __typename?: 'Entity', id: string, entityType: string, properties: string, referenceValue?: string | null }> }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

export type EntityDetailQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type EntityDetailQuery = { __typename?: 'Query', entity?: { __typename?: 'Entity', id: string, organizationId: string, schemaId: string, entityType: string, path: string, properties: string, referenceValue?: string | null, version: number, createdAt: string, updatedAt: string, linkedEntities: Array<{ __typename?: 'Entity', id: string, entityType: string, properties: string, referenceValue?: string | null }> } | null };

export type EntityHistoryQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type EntityHistoryQuery = { __typename?: 'Query', entityHistory: Array<{ __typename?: 'EntitySnapshotView', version: number, path: string, schemaId: string, entityType: string, canonicalText: Array<string> }> };

export type EntityDiffQueryVariables = Exact<{
  id: Scalars['String']['input'];
  baseVersion: Scalars['Int']['input'];
  targetVersion: Scalars['Int']['input'];
}>;


export type EntityDiffQuery = { __typename?: 'Query', entityDiff?: { __typename?: 'EntityDiffResult', unifiedDiff?: string | null, base?: { __typename?: 'EntitySnapshotView', version: number, path: string, schemaId: string, entityType: string, canonicalText: Array<string> } | null, target?: { __typename?: 'EntitySnapshotView', version: number, path: string, schemaId: string, entityType: string, canonicalText: Array<string> } | null } | null };

export type RollbackEntityMutationVariables = Exact<{
  id: Scalars['String']['input'];
  toVersion: Scalars['Int']['input'];
  reason?: InputMaybe<Scalars['String']['input']>;
}>;


export type RollbackEntityMutation = { __typename?: 'Mutation', rollbackEntity: { __typename?: 'Entity', id: string, version: number, properties: string } };

export type EntitiesByTypeQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  entityType: Scalars['String']['input'];
}>;


export type EntitiesByTypeQuery = { __typename?: 'Query', entitiesByType: Array<{ __typename?: 'Entity', id: string, entityType: string, schemaId: string, version: number, properties: string, referenceValue?: string | null }> };

export type EntitySchemaByNameQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  name: Scalars['String']['input'];
}>;


export type EntitySchemaByNameQuery = { __typename?: 'Query', entitySchemaByName?: { __typename?: 'EntitySchema', id: string, name: string, description?: string | null, version: string, status: SchemaStatus, previousVersionId?: string | null, fields: Array<{ __typename?: 'FieldDefinition', name: string, type: FieldType, required: boolean, description?: string | null }> } | null };

export type CreateJoinMutationVariables = Exact<{
  input: CreateEntityJoinDefinitionInput;
}>;


export type CreateJoinMutation = { __typename?: 'Mutation', createEntityJoinDefinition: { __typename?: 'EntityJoinDefinition', id: string, name: string, description?: string | null, joinType: JoinType, leftEntityType: string, rightEntityType: string, joinField?: string | null, joinFieldType?: FieldType | null, createdAt: string, updatedAt: string, leftFilters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, rightFilters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, sortCriteria: Array<{ __typename?: 'JoinSortCriterion', side: JoinSide, field: string, direction: JoinSortDirection }> } };

export type UpdateSchemaMutationVariables = Exact<{
  input: UpdateEntitySchemaInput;
}>;


export type UpdateSchemaMutation = { __typename?: 'Mutation', updateEntitySchema: { __typename?: 'EntitySchema', id: string, organizationId: string, name: string, description?: string | null, status: SchemaStatus, version: string, createdAt: string, updatedAt: string, previousVersionId?: string | null, fields: Array<{ __typename?: 'FieldDefinition', name: string, type: FieldType, required: boolean, description?: string | null, default?: string | null, validation?: string | null, referenceEntityType?: string | null }> } };

export type EntityJoinDefinitionsQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
}>;


export type EntityJoinDefinitionsQuery = { __typename?: 'Query', entityJoinDefinitions: Array<{ __typename?: 'EntityJoinDefinition', id: string, name: string, description?: string | null, joinType: JoinType, leftEntityType: string, rightEntityType: string, joinField?: string | null, joinFieldType?: FieldType | null, createdAt: string, updatedAt: string, leftFilters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, rightFilters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, sortCriteria: Array<{ __typename?: 'JoinSortCriterion', side: JoinSide, field: string, direction: JoinSortDirection }> }> };

export type DeleteSchemaMutationVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type DeleteSchemaMutation = { __typename?: 'Mutation', deleteEntitySchema: boolean };

export type DeleteJoinMutationVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type DeleteJoinMutation = { __typename?: 'Mutation', deleteEntityJoinDefinition: boolean };

export type ExecuteJoinQueryVariables = Exact<{
  input: ExecuteEntityJoinInput;
}>;


export type ExecuteJoinQuery = { __typename?: 'Query', executeEntityJoin: { __typename?: 'EntityJoinConnection', edges: Array<{ __typename?: 'EntityJoinEdge', left: { __typename?: 'Entity', id: string, entityType: string, properties: string }, right: { __typename?: 'Entity', id: string, entityType: string, properties: string } }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

export type EntityTransformationNodeFieldsFragment = { __typename?: 'EntityTransformationNode', id: string, name: string, type: EntityTransformationNodeType, inputs: Array<string>, load?: { __typename?: 'EntityTransformationLoadConfig', alias: string, entityType: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, filter?: { __typename?: 'EntityTransformationFilterConfig', alias: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, project?: { __typename?: 'EntityTransformationProjectConfig', alias: string, fields: Array<string> } | null, join?: { __typename?: 'EntityTransformationJoinConfig', leftAlias: string, rightAlias: string, onField: string } | null, materialize?: { __typename?: 'EntityTransformationMaterializeConfig', outputs: Array<{ __typename?: 'EntityTransformationMaterializeOutput', alias: string, fields: Array<{ __typename?: 'EntityTransformationMaterializeFieldMapping', sourceAlias: string, sourceField: string, outputField: string }> }> } | null, sort?: { __typename?: 'EntityTransformationSortConfig', alias: string, field: string, direction: JoinSortDirection } | null, paginate?: { __typename?: 'EntityTransformationPaginateConfig', limit?: number | null, offset?: number | null } | null };

export type EntityTransformationsQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
}>;


export type EntityTransformationsQuery = { __typename?: 'Query', entityTransformations: Array<{ __typename?: 'EntityTransformation', id: string, organizationId: string, name: string, description?: string | null, createdAt: string, updatedAt: string, nodes: Array<{ __typename?: 'EntityTransformationNode', id: string }> }> };

export type EntityTransformationQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type EntityTransformationQuery = { __typename?: 'Query', entityTransformation?: { __typename?: 'EntityTransformation', id: string, organizationId: string, name: string, description?: string | null, createdAt: string, updatedAt: string, nodes: Array<{ __typename?: 'EntityTransformationNode', id: string, name: string, type: EntityTransformationNodeType, inputs: Array<string>, load?: { __typename?: 'EntityTransformationLoadConfig', alias: string, entityType: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, filter?: { __typename?: 'EntityTransformationFilterConfig', alias: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, project?: { __typename?: 'EntityTransformationProjectConfig', alias: string, fields: Array<string> } | null, join?: { __typename?: 'EntityTransformationJoinConfig', leftAlias: string, rightAlias: string, onField: string } | null, materialize?: { __typename?: 'EntityTransformationMaterializeConfig', outputs: Array<{ __typename?: 'EntityTransformationMaterializeOutput', alias: string, fields: Array<{ __typename?: 'EntityTransformationMaterializeFieldMapping', sourceAlias: string, sourceField: string, outputField: string }> }> } | null, sort?: { __typename?: 'EntityTransformationSortConfig', alias: string, field: string, direction: JoinSortDirection } | null, paginate?: { __typename?: 'EntityTransformationPaginateConfig', limit?: number | null, offset?: number | null } | null }> } | null };

export type CreateEntityTransformationMutationVariables = Exact<{
  input: CreateEntityTransformationInput;
}>;


export type CreateEntityTransformationMutation = { __typename?: 'Mutation', createEntityTransformation: { __typename?: 'EntityTransformation', id: string, organizationId: string, name: string, description?: string | null, createdAt: string, updatedAt: string, nodes: Array<{ __typename?: 'EntityTransformationNode', id: string, name: string, type: EntityTransformationNodeType, inputs: Array<string>, load?: { __typename?: 'EntityTransformationLoadConfig', alias: string, entityType: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, filter?: { __typename?: 'EntityTransformationFilterConfig', alias: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, project?: { __typename?: 'EntityTransformationProjectConfig', alias: string, fields: Array<string> } | null, join?: { __typename?: 'EntityTransformationJoinConfig', leftAlias: string, rightAlias: string, onField: string } | null, materialize?: { __typename?: 'EntityTransformationMaterializeConfig', outputs: Array<{ __typename?: 'EntityTransformationMaterializeOutput', alias: string, fields: Array<{ __typename?: 'EntityTransformationMaterializeFieldMapping', sourceAlias: string, sourceField: string, outputField: string }> }> } | null, sort?: { __typename?: 'EntityTransformationSortConfig', alias: string, field: string, direction: JoinSortDirection } | null, paginate?: { __typename?: 'EntityTransformationPaginateConfig', limit?: number | null, offset?: number | null } | null }> } };

export type UpdateEntityTransformationMutationVariables = Exact<{
  input: UpdateEntityTransformationInput;
}>;


export type UpdateEntityTransformationMutation = { __typename?: 'Mutation', updateEntityTransformation: { __typename?: 'EntityTransformation', id: string, organizationId: string, name: string, description?: string | null, createdAt: string, updatedAt: string, nodes: Array<{ __typename?: 'EntityTransformationNode', id: string, name: string, type: EntityTransformationNodeType, inputs: Array<string>, load?: { __typename?: 'EntityTransformationLoadConfig', alias: string, entityType: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, filter?: { __typename?: 'EntityTransformationFilterConfig', alias: string, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }> } | null, project?: { __typename?: 'EntityTransformationProjectConfig', alias: string, fields: Array<string> } | null, join?: { __typename?: 'EntityTransformationJoinConfig', leftAlias: string, rightAlias: string, onField: string } | null, materialize?: { __typename?: 'EntityTransformationMaterializeConfig', outputs: Array<{ __typename?: 'EntityTransformationMaterializeOutput', alias: string, fields: Array<{ __typename?: 'EntityTransformationMaterializeFieldMapping', sourceAlias: string, sourceField: string, outputField: string }> }> } | null, sort?: { __typename?: 'EntityTransformationSortConfig', alias: string, field: string, direction: JoinSortDirection } | null, paginate?: { __typename?: 'EntityTransformationPaginateConfig', limit?: number | null, offset?: number | null } | null }> } };

export type DeleteEntityTransformationMutationVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type DeleteEntityTransformationMutation = { __typename?: 'Mutation', deleteEntityTransformation: boolean };

export type ExecuteEntityTransformationQueryVariables = Exact<{
  input: ExecuteEntityTransformationInput;
}>;


export type ExecuteEntityTransformationQuery = { __typename?: 'Query', executeEntityTransformation: { __typename?: 'EntityTransformationConnection', edges: Array<{ __typename?: 'EntityTransformationRecordEdge', entities: Array<{ __typename?: 'EntityTransformationRecordEntity', alias: string, entity?: { __typename?: 'Entity', id: string, entityType: string, path: string, referenceValue?: string | null, properties: string } | null }> }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

export type TransformationExecutionQueryVariables = Exact<{
  transformationId: Scalars['String']['input'];
  filters?: InputMaybe<Array<TransformationExecutionFilterInput> | TransformationExecutionFilterInput>;
  sort?: InputMaybe<TransformationExecutionSortInput>;
  pagination?: InputMaybe<PaginationInput>;
}>;


export type TransformationExecutionQuery = { __typename?: 'Query', transformationExecution: { __typename?: 'TransformationExecutionConnection', columns: Array<{ __typename?: 'TransformationExecutionColumn', key: string, alias: string, field: string, label: string, sourceAlias: string, sourceField: string }>, rows: Array<{ __typename?: 'TransformationExecutionRow', values: Array<{ __typename?: 'TransformationExecutionValue', columnKey: string, value?: string | null }> }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

export type EntityExportJobsQueryVariables = Exact<{
  organizationId: Scalars['String']['input'];
  statuses?: InputMaybe<Array<EntityExportJobStatus> | EntityExportJobStatus>;
  limit?: InputMaybe<Scalars['Int']['input']>;
  offset?: InputMaybe<Scalars['Int']['input']>;
}>;


export type EntityExportJobsQuery = { __typename?: 'Query', entityExportJobs: Array<{ __typename?: 'EntityExportJob', id: string, organizationId: string, jobType: EntityExportJobType, entityType?: string | null, transformationId?: string | null, status: EntityExportJobStatus, rowsRequested: number, rowsExported: number, bytesWritten: number, fileMimeType?: string | null, fileByteSize?: number | null, errorMessage?: string | null, enqueuedAt: string, startedAt?: string | null, completedAt?: string | null, updatedAt: string, downloadUrl?: string | null, filters: Array<{ __typename?: 'PropertyFilterConfig', key: string, value?: string | null, exists?: boolean | null, inArray?: Array<string> | null }>, transformationDefinition?: { __typename?: 'EntityTransformation', id: string, name: string } | null }> };


export const EntityTransformationNodeFieldsFragmentDoc = `
    fragment EntityTransformationNodeFields on EntityTransformationNode {
  id
  name
  type
  inputs
  load {
    alias
    entityType
    filters {
      key
      value
      exists
      inArray
    }
  }
  filter {
    alias
    filters {
      key
      value
      exists
      inArray
    }
  }
  project {
    alias
    fields
  }
  join {
    leftAlias
    rightAlias
    onField
  }
  materialize {
    outputs {
      alias
      fields {
        sourceAlias
        sourceField
        outputField
      }
    }
  }
  sort {
    alias
    field
    direction
  }
  paginate {
    limit
    offset
  }
}
    `;
export const GetOrganizationsDocument = `
    query GetOrganizations {
  organizations {
    id
    name
    description
  }
}
    `;

export const useGetOrganizationsQuery = <
      TData = GetOrganizationsQuery,
      TError = unknown
    >(
      variables?: GetOrganizationsQueryVariables,
      options?: Omit<UseQueryOptions<GetOrganizationsQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<GetOrganizationsQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<GetOrganizationsQuery, TError, TData>(
      {
    queryKey: variables === undefined ? ['GetOrganizations'] : ['GetOrganizations', variables],
    queryFn: () => graphqlRequest<GetOrganizationsQuery, GetOrganizationsQueryVariables>(GetOrganizationsDocument, variables),
    ...options
  }
    )};

useGetOrganizationsQuery.getKey = (variables?: GetOrganizationsQueryVariables) => variables === undefined ? ['GetOrganizations'] : ['GetOrganizations', variables];


useGetOrganizationsQuery.fetcher = (variables?: GetOrganizationsQueryVariables, options?: RequestInit['headers']) => graphqlRequest<GetOrganizationsQuery, GetOrganizationsQueryVariables>(GetOrganizationsDocument, variables, options);

export const GetEntitiesByOrgDocument = `
    query GetEntitiesByOrg($organizationId: String!, $limit: Int = 10, $offset: Int = 0) {
  entities(
    organizationId: $organizationId
    pagination: {limit: $limit, offset: $offset}
  ) {
    entities {
      id
      entityType
      version
      path
      properties
      referenceValue
      createdAt
      updatedAt
    }
    pageInfo {
      totalCount
      hasNextPage
      hasPreviousPage
    }
  }
}
    `;

export const useGetEntitiesByOrgQuery = <
      TData = GetEntitiesByOrgQuery,
      TError = unknown
    >(
      variables: GetEntitiesByOrgQueryVariables,
      options?: Omit<UseQueryOptions<GetEntitiesByOrgQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<GetEntitiesByOrgQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<GetEntitiesByOrgQuery, TError, TData>(
      {
    queryKey: ['GetEntitiesByOrg', variables],
    queryFn: () => graphqlRequest<GetEntitiesByOrgQuery, GetEntitiesByOrgQueryVariables>(GetEntitiesByOrgDocument, variables),
    ...options
  }
    )};

useGetEntitiesByOrgQuery.getKey = (variables: GetEntitiesByOrgQueryVariables) => ['GetEntitiesByOrg', variables];


useGetEntitiesByOrgQuery.fetcher = (variables: GetEntitiesByOrgQueryVariables, options?: RequestInit['headers']) => graphqlRequest<GetEntitiesByOrgQuery, GetEntitiesByOrgQueryVariables>(GetEntitiesByOrgDocument, variables, options);

export const CreateSchemaDocument = `
    mutation CreateSchema($input: CreateEntitySchemaInput!) {
  createEntitySchema(input: $input) {
    id
    name
    description
    version
    status
    previousVersionId
  }
}
    `;

export const useCreateSchemaMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<CreateSchemaMutation, TError, CreateSchemaMutationVariables, TContext>) => {
    
    return useMutation<CreateSchemaMutation, TError, CreateSchemaMutationVariables, TContext>(
      {
    mutationKey: ['CreateSchema'],
    mutationFn: (variables?: CreateSchemaMutationVariables) => graphqlRequest<CreateSchemaMutation, CreateSchemaMutationVariables>(CreateSchemaDocument, variables),
    ...options
  }
    )};

useCreateSchemaMutation.getKey = () => ['CreateSchema'];


useCreateSchemaMutation.fetcher = (variables: CreateSchemaMutationVariables, options?: RequestInit['headers']) => graphqlRequest<CreateSchemaMutation, CreateSchemaMutationVariables>(CreateSchemaDocument, variables, options);

export const CreateEntityDocument = `
    mutation CreateEntity($input: CreateEntityInput!) {
  createEntity(input: $input) {
    id
    entityType
    schemaId
    version
    properties
  }
}
    `;

export const useCreateEntityMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<CreateEntityMutation, TError, CreateEntityMutationVariables, TContext>) => {
    
    return useMutation<CreateEntityMutation, TError, CreateEntityMutationVariables, TContext>(
      {
    mutationKey: ['CreateEntity'],
    mutationFn: (variables?: CreateEntityMutationVariables) => graphqlRequest<CreateEntityMutation, CreateEntityMutationVariables>(CreateEntityDocument, variables),
    ...options
  }
    )};

useCreateEntityMutation.getKey = () => ['CreateEntity'];


useCreateEntityMutation.fetcher = (variables: CreateEntityMutationVariables, options?: RequestInit['headers']) => graphqlRequest<CreateEntityMutation, CreateEntityMutationVariables>(CreateEntityDocument, variables, options);

export const UpdateEntityDocument = `
    mutation UpdateEntity($input: UpdateEntityInput!) {
  updateEntity(input: $input) {
    id
    organizationId
    schemaId
    entityType
    path
    properties
    version
    updatedAt
  }
}
    `;

export const useUpdateEntityMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<UpdateEntityMutation, TError, UpdateEntityMutationVariables, TContext>) => {
    
    return useMutation<UpdateEntityMutation, TError, UpdateEntityMutationVariables, TContext>(
      {
    mutationKey: ['UpdateEntity'],
    mutationFn: (variables?: UpdateEntityMutationVariables) => graphqlRequest<UpdateEntityMutation, UpdateEntityMutationVariables>(UpdateEntityDocument, variables),
    ...options
  }
    )};

useUpdateEntityMutation.getKey = () => ['UpdateEntity'];


useUpdateEntityMutation.fetcher = (variables: UpdateEntityMutationVariables, options?: RequestInit['headers']) => graphqlRequest<UpdateEntityMutation, UpdateEntityMutationVariables>(UpdateEntityDocument, variables, options);

export const QueueEntityTypeExportDocument = `
    mutation QueueEntityTypeExport($input: QueueEntityTypeExportInput!) {
  queueEntityTypeExport(input: $input) {
    id
    organizationId
    jobType
    entityType
    transformationId
    status
    rowsRequested
    rowsExported
    bytesWritten
    fileMimeType
    fileByteSize
    errorMessage
    filters {
      key
      value
      exists
      inArray
    }
    transformationDefinition {
      id
      name
    }
    enqueuedAt
    startedAt
    completedAt
    updatedAt
    downloadUrl
  }
}
    `;

export const useQueueEntityTypeExportMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<QueueEntityTypeExportMutation, TError, QueueEntityTypeExportMutationVariables, TContext>) => {
    
    return useMutation<QueueEntityTypeExportMutation, TError, QueueEntityTypeExportMutationVariables, TContext>(
      {
    mutationKey: ['QueueEntityTypeExport'],
    mutationFn: (variables?: QueueEntityTypeExportMutationVariables) => graphqlRequest<QueueEntityTypeExportMutation, QueueEntityTypeExportMutationVariables>(QueueEntityTypeExportDocument, variables),
    ...options
  }
    )};

useQueueEntityTypeExportMutation.getKey = () => ['QueueEntityTypeExport'];


useQueueEntityTypeExportMutation.fetcher = (variables: QueueEntityTypeExportMutationVariables, options?: RequestInit['headers']) => graphqlRequest<QueueEntityTypeExportMutation, QueueEntityTypeExportMutationVariables>(QueueEntityTypeExportDocument, variables, options);

export const QueueTransformationExportDocument = `
    mutation QueueTransformationExport($input: QueueTransformationExportInput!) {
  queueTransformationExport(input: $input) {
    id
    organizationId
    jobType
    entityType
    transformationId
    status
    rowsRequested
    rowsExported
    bytesWritten
    fileMimeType
    fileByteSize
    errorMessage
    filters {
      key
      value
      exists
      inArray
    }
    transformationDefinition {
      id
      name
    }
    enqueuedAt
    startedAt
    completedAt
    updatedAt
    downloadUrl
  }
}
    `;

export const useQueueTransformationExportMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<QueueTransformationExportMutation, TError, QueueTransformationExportMutationVariables, TContext>) => {
    
    return useMutation<QueueTransformationExportMutation, TError, QueueTransformationExportMutationVariables, TContext>(
      {
    mutationKey: ['QueueTransformationExport'],
    mutationFn: (variables?: QueueTransformationExportMutationVariables) => graphqlRequest<QueueTransformationExportMutation, QueueTransformationExportMutationVariables>(QueueTransformationExportDocument, variables),
    ...options
  }
    )};

useQueueTransformationExportMutation.getKey = () => ['QueueTransformationExport'];


useQueueTransformationExportMutation.fetcher = (variables: QueueTransformationExportMutationVariables, options?: RequestInit['headers']) => graphqlRequest<QueueTransformationExportMutation, QueueTransformationExportMutationVariables>(QueueTransformationExportDocument, variables, options);

export const CancelEntityExportJobDocument = `
    mutation CancelEntityExportJob($id: String!) {
  cancelEntityExportJob(id: $id) {
    id
    organizationId
    jobType
    entityType
    transformationId
    status
    rowsRequested
    rowsExported
    bytesWritten
    fileMimeType
    fileByteSize
    errorMessage
    filters {
      key
      value
      exists
      inArray
    }
    transformationDefinition {
      id
      name
    }
    enqueuedAt
    startedAt
    completedAt
    updatedAt
    downloadUrl
  }
}
    `;

export const useCancelEntityExportJobMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<CancelEntityExportJobMutation, TError, CancelEntityExportJobMutationVariables, TContext>) => {
    
    return useMutation<CancelEntityExportJobMutation, TError, CancelEntityExportJobMutationVariables, TContext>(
      {
    mutationKey: ['CancelEntityExportJob'],
    mutationFn: (variables?: CancelEntityExportJobMutationVariables) => graphqlRequest<CancelEntityExportJobMutation, CancelEntityExportJobMutationVariables>(CancelEntityExportJobDocument, variables),
    ...options
  }
    )};

useCancelEntityExportJobMutation.getKey = () => ['CancelEntityExportJob'];


useCancelEntityExportJobMutation.fetcher = (variables: CancelEntityExportJobMutationVariables, options?: RequestInit['headers']) => graphqlRequest<CancelEntityExportJobMutation, CancelEntityExportJobMutationVariables>(CancelEntityExportJobDocument, variables, options);

export const DeleteEntityDocument = `
    mutation DeleteEntity($id: String!) {
  deleteEntity(id: $id)
}
    `;

export const useDeleteEntityMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<DeleteEntityMutation, TError, DeleteEntityMutationVariables, TContext>) => {
    
    return useMutation<DeleteEntityMutation, TError, DeleteEntityMutationVariables, TContext>(
      {
    mutationKey: ['DeleteEntity'],
    mutationFn: (variables?: DeleteEntityMutationVariables) => graphqlRequest<DeleteEntityMutation, DeleteEntityMutationVariables>(DeleteEntityDocument, variables),
    ...options
  }
    )};

useDeleteEntityMutation.getKey = () => ['DeleteEntity'];


useDeleteEntityMutation.fetcher = (variables: DeleteEntityMutationVariables, options?: RequestInit['headers']) => graphqlRequest<DeleteEntityMutation, DeleteEntityMutationVariables>(DeleteEntityDocument, variables, options);

export const EntitiesByTypeFullDocument = `
    query EntitiesByTypeFull($organizationId: String!, $entityType: String!) {
  entitiesByType(organizationId: $organizationId, entityType: $entityType) {
    id
    entityType
    schemaId
    version
    properties
    referenceValue
    linkedEntities {
      id
      entityType
      properties
      referenceValue
    }
  }
}
    `;

export const useEntitiesByTypeFullQuery = <
      TData = EntitiesByTypeFullQuery,
      TError = unknown
    >(
      variables: EntitiesByTypeFullQueryVariables,
      options?: Omit<UseQueryOptions<EntitiesByTypeFullQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntitiesByTypeFullQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntitiesByTypeFullQuery, TError, TData>(
      {
    queryKey: ['EntitiesByTypeFull', variables],
    queryFn: () => graphqlRequest<EntitiesByTypeFullQuery, EntitiesByTypeFullQueryVariables>(EntitiesByTypeFullDocument, variables),
    ...options
  }
    )};

useEntitiesByTypeFullQuery.getKey = (variables: EntitiesByTypeFullQueryVariables) => ['EntitiesByTypeFull', variables];


useEntitiesByTypeFullQuery.fetcher = (variables: EntitiesByTypeFullQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntitiesByTypeFullQuery, EntitiesByTypeFullQueryVariables>(EntitiesByTypeFullDocument, variables, options);

export const SchemaVersionsDocument = `
    query SchemaVersions($organizationId: String!, $name: String!) {
  entitySchemaVersions(organizationId: $organizationId, name: $name) {
    id
    version
    status
    previousVersionId
    createdAt
  }
}
    `;

export const useSchemaVersionsQuery = <
      TData = SchemaVersionsQuery,
      TError = unknown
    >(
      variables: SchemaVersionsQueryVariables,
      options?: Omit<UseQueryOptions<SchemaVersionsQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<SchemaVersionsQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<SchemaVersionsQuery, TError, TData>(
      {
    queryKey: ['SchemaVersions', variables],
    queryFn: () => graphqlRequest<SchemaVersionsQuery, SchemaVersionsQueryVariables>(SchemaVersionsDocument, variables),
    ...options
  }
    )};

useSchemaVersionsQuery.getKey = (variables: SchemaVersionsQueryVariables) => ['SchemaVersions', variables];


useSchemaVersionsQuery.fetcher = (variables: SchemaVersionsQueryVariables, options?: RequestInit['headers']) => graphqlRequest<SchemaVersionsQuery, SchemaVersionsQueryVariables>(SchemaVersionsDocument, variables, options);

export const EntitySchemasDocument = `
    query EntitySchemas($organizationId: String!) {
  entitySchemas(organizationId: $organizationId) {
    id
    organizationId
    name
    description
    status
    version
    createdAt
    updatedAt
    previousVersionId
    fields {
      name
      type
      required
      description
      default
      validation
      referenceEntityType
    }
  }
}
    `;

export const useEntitySchemasQuery = <
      TData = EntitySchemasQuery,
      TError = unknown
    >(
      variables: EntitySchemasQueryVariables,
      options?: Omit<UseQueryOptions<EntitySchemasQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntitySchemasQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntitySchemasQuery, TError, TData>(
      {
    queryKey: ['EntitySchemas', variables],
    queryFn: () => graphqlRequest<EntitySchemasQuery, EntitySchemasQueryVariables>(EntitySchemasDocument, variables),
    ...options
  }
    )};

useEntitySchemasQuery.getKey = (variables: EntitySchemasQueryVariables) => ['EntitySchemas', variables];


useEntitySchemasQuery.fetcher = (variables: EntitySchemasQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntitySchemasQuery, EntitySchemasQueryVariables>(EntitySchemasDocument, variables, options);

export const EntitiesManagementDocument = `
    query EntitiesManagement($organizationId: String!, $pagination: PaginationInput, $filter: EntityFilter, $includeLinkedEntities: Boolean! = true, $sort: EntitySortInput) {
  entities(
    organizationId: $organizationId
    pagination: $pagination
    filter: $filter
    sort: $sort
  ) {
    entities {
      id
      organizationId
      schemaId
      entityType
      path
      properties
      referenceValue
      version
      createdAt
      updatedAt
      linkedEntities @include(if: $includeLinkedEntities) {
        id
        entityType
        properties
        referenceValue
      }
    }
    pageInfo {
      totalCount
      hasNextPage
      hasPreviousPage
    }
  }
}
    `;

export const useEntitiesManagementQuery = <
      TData = EntitiesManagementQuery,
      TError = unknown
    >(
      variables: EntitiesManagementQueryVariables,
      options?: Omit<UseQueryOptions<EntitiesManagementQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntitiesManagementQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntitiesManagementQuery, TError, TData>(
      {
    queryKey: ['EntitiesManagement', variables],
    queryFn: () => graphqlRequest<EntitiesManagementQuery, EntitiesManagementQueryVariables>(EntitiesManagementDocument, variables),
    ...options
  }
    )};

useEntitiesManagementQuery.getKey = (variables: EntitiesManagementQueryVariables) => ['EntitiesManagement', variables];


useEntitiesManagementQuery.fetcher = (variables: EntitiesManagementQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntitiesManagementQuery, EntitiesManagementQueryVariables>(EntitiesManagementDocument, variables, options);

export const EntityDetailDocument = `
    query EntityDetail($id: String!) {
  entity(id: $id) {
    id
    organizationId
    schemaId
    entityType
    path
    properties
    referenceValue
    version
    createdAt
    updatedAt
    linkedEntities {
      id
      entityType
      properties
      referenceValue
    }
  }
}
    `;

export const useEntityDetailQuery = <
      TData = EntityDetailQuery,
      TError = unknown
    >(
      variables: EntityDetailQueryVariables,
      options?: Omit<UseQueryOptions<EntityDetailQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityDetailQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityDetailQuery, TError, TData>(
      {
    queryKey: ['EntityDetail', variables],
    queryFn: () => graphqlRequest<EntityDetailQuery, EntityDetailQueryVariables>(EntityDetailDocument, variables),
    ...options
  }
    )};

useEntityDetailQuery.getKey = (variables: EntityDetailQueryVariables) => ['EntityDetail', variables];


useEntityDetailQuery.fetcher = (variables: EntityDetailQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityDetailQuery, EntityDetailQueryVariables>(EntityDetailDocument, variables, options);

export const EntityHistoryDocument = `
    query EntityHistory($id: String!) {
  entityHistory(id: $id) {
    version
    path
    schemaId
    entityType
    canonicalText
  }
}
    `;

export const useEntityHistoryQuery = <
      TData = EntityHistoryQuery,
      TError = unknown
    >(
      variables: EntityHistoryQueryVariables,
      options?: Omit<UseQueryOptions<EntityHistoryQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityHistoryQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityHistoryQuery, TError, TData>(
      {
    queryKey: ['EntityHistory', variables],
    queryFn: () => graphqlRequest<EntityHistoryQuery, EntityHistoryQueryVariables>(EntityHistoryDocument, variables),
    ...options
  }
    )};

useEntityHistoryQuery.getKey = (variables: EntityHistoryQueryVariables) => ['EntityHistory', variables];


useEntityHistoryQuery.fetcher = (variables: EntityHistoryQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityHistoryQuery, EntityHistoryQueryVariables>(EntityHistoryDocument, variables, options);

export const EntityDiffDocument = `
    query EntityDiff($id: String!, $baseVersion: Int!, $targetVersion: Int!) {
  entityDiff(id: $id, baseVersion: $baseVersion, targetVersion: $targetVersion) {
    base {
      version
      path
      schemaId
      entityType
      canonicalText
    }
    target {
      version
      path
      schemaId
      entityType
      canonicalText
    }
    unifiedDiff
  }
}
    `;

export const useEntityDiffQuery = <
      TData = EntityDiffQuery,
      TError = unknown
    >(
      variables: EntityDiffQueryVariables,
      options?: Omit<UseQueryOptions<EntityDiffQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityDiffQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityDiffQuery, TError, TData>(
      {
    queryKey: ['EntityDiff', variables],
    queryFn: () => graphqlRequest<EntityDiffQuery, EntityDiffQueryVariables>(EntityDiffDocument, variables),
    ...options
  }
    )};

useEntityDiffQuery.getKey = (variables: EntityDiffQueryVariables) => ['EntityDiff', variables];


useEntityDiffQuery.fetcher = (variables: EntityDiffQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityDiffQuery, EntityDiffQueryVariables>(EntityDiffDocument, variables, options);

export const RollbackEntityDocument = `
    mutation RollbackEntity($id: String!, $toVersion: Int!, $reason: String) {
  rollbackEntity(id: $id, toVersion: $toVersion, reason: $reason) {
    id
    version
    properties
  }
}
    `;

export const useRollbackEntityMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<RollbackEntityMutation, TError, RollbackEntityMutationVariables, TContext>) => {
    
    return useMutation<RollbackEntityMutation, TError, RollbackEntityMutationVariables, TContext>(
      {
    mutationKey: ['RollbackEntity'],
    mutationFn: (variables?: RollbackEntityMutationVariables) => graphqlRequest<RollbackEntityMutation, RollbackEntityMutationVariables>(RollbackEntityDocument, variables),
    ...options
  }
    )};

useRollbackEntityMutation.getKey = () => ['RollbackEntity'];


useRollbackEntityMutation.fetcher = (variables: RollbackEntityMutationVariables, options?: RequestInit['headers']) => graphqlRequest<RollbackEntityMutation, RollbackEntityMutationVariables>(RollbackEntityDocument, variables, options);

export const EntitiesByTypeDocument = `
    query EntitiesByType($organizationId: String!, $entityType: String!) {
  entitiesByType(organizationId: $organizationId, entityType: $entityType) {
    id
    entityType
    schemaId
    version
    properties
    referenceValue
  }
}
    `;

export const useEntitiesByTypeQuery = <
      TData = EntitiesByTypeQuery,
      TError = unknown
    >(
      variables: EntitiesByTypeQueryVariables,
      options?: Omit<UseQueryOptions<EntitiesByTypeQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntitiesByTypeQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntitiesByTypeQuery, TError, TData>(
      {
    queryKey: ['EntitiesByType', variables],
    queryFn: () => graphqlRequest<EntitiesByTypeQuery, EntitiesByTypeQueryVariables>(EntitiesByTypeDocument, variables),
    ...options
  }
    )};

useEntitiesByTypeQuery.getKey = (variables: EntitiesByTypeQueryVariables) => ['EntitiesByType', variables];


useEntitiesByTypeQuery.fetcher = (variables: EntitiesByTypeQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntitiesByTypeQuery, EntitiesByTypeQueryVariables>(EntitiesByTypeDocument, variables, options);

export const EntitySchemaByNameDocument = `
    query EntitySchemaByName($organizationId: String!, $name: String!) {
  entitySchemaByName(organizationId: $organizationId, name: $name) {
    id
    name
    description
    version
    status
    previousVersionId
    fields {
      name
      type
      required
      description
    }
  }
}
    `;

export const useEntitySchemaByNameQuery = <
      TData = EntitySchemaByNameQuery,
      TError = unknown
    >(
      variables: EntitySchemaByNameQueryVariables,
      options?: Omit<UseQueryOptions<EntitySchemaByNameQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntitySchemaByNameQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntitySchemaByNameQuery, TError, TData>(
      {
    queryKey: ['EntitySchemaByName', variables],
    queryFn: () => graphqlRequest<EntitySchemaByNameQuery, EntitySchemaByNameQueryVariables>(EntitySchemaByNameDocument, variables),
    ...options
  }
    )};

useEntitySchemaByNameQuery.getKey = (variables: EntitySchemaByNameQueryVariables) => ['EntitySchemaByName', variables];


useEntitySchemaByNameQuery.fetcher = (variables: EntitySchemaByNameQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntitySchemaByNameQuery, EntitySchemaByNameQueryVariables>(EntitySchemaByNameDocument, variables, options);

export const CreateJoinDocument = `
    mutation CreateJoin($input: CreateEntityJoinDefinitionInput!) {
  createEntityJoinDefinition(input: $input) {
    id
    name
    description
    joinType
    leftEntityType
    rightEntityType
    joinField
    joinFieldType
    createdAt
    updatedAt
    leftFilters {
      key
      value
      exists
      inArray
    }
    rightFilters {
      key
      value
      exists
      inArray
    }
    sortCriteria {
      side
      field
      direction
    }
  }
}
    `;

export const useCreateJoinMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<CreateJoinMutation, TError, CreateJoinMutationVariables, TContext>) => {
    
    return useMutation<CreateJoinMutation, TError, CreateJoinMutationVariables, TContext>(
      {
    mutationKey: ['CreateJoin'],
    mutationFn: (variables?: CreateJoinMutationVariables) => graphqlRequest<CreateJoinMutation, CreateJoinMutationVariables>(CreateJoinDocument, variables),
    ...options
  }
    )};

useCreateJoinMutation.getKey = () => ['CreateJoin'];


useCreateJoinMutation.fetcher = (variables: CreateJoinMutationVariables, options?: RequestInit['headers']) => graphqlRequest<CreateJoinMutation, CreateJoinMutationVariables>(CreateJoinDocument, variables, options);

export const UpdateSchemaDocument = `
    mutation UpdateSchema($input: UpdateEntitySchemaInput!) {
  updateEntitySchema(input: $input) {
    id
    organizationId
    name
    description
    status
    version
    createdAt
    updatedAt
    previousVersionId
    fields {
      name
      type
      required
      description
      default
      validation
      referenceEntityType
    }
  }
}
    `;

export const useUpdateSchemaMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<UpdateSchemaMutation, TError, UpdateSchemaMutationVariables, TContext>) => {
    
    return useMutation<UpdateSchemaMutation, TError, UpdateSchemaMutationVariables, TContext>(
      {
    mutationKey: ['UpdateSchema'],
    mutationFn: (variables?: UpdateSchemaMutationVariables) => graphqlRequest<UpdateSchemaMutation, UpdateSchemaMutationVariables>(UpdateSchemaDocument, variables),
    ...options
  }
    )};

useUpdateSchemaMutation.getKey = () => ['UpdateSchema'];


useUpdateSchemaMutation.fetcher = (variables: UpdateSchemaMutationVariables, options?: RequestInit['headers']) => graphqlRequest<UpdateSchemaMutation, UpdateSchemaMutationVariables>(UpdateSchemaDocument, variables, options);

export const EntityJoinDefinitionsDocument = `
    query EntityJoinDefinitions($organizationId: String!) {
  entityJoinDefinitions(organizationId: $organizationId) {
    id
    name
    description
    joinType
    leftEntityType
    rightEntityType
    joinField
    joinFieldType
    createdAt
    updatedAt
    leftFilters {
      key
      value
      exists
      inArray
    }
    rightFilters {
      key
      value
      exists
      inArray
    }
    sortCriteria {
      side
      field
      direction
    }
  }
}
    `;

export const useEntityJoinDefinitionsQuery = <
      TData = EntityJoinDefinitionsQuery,
      TError = unknown
    >(
      variables: EntityJoinDefinitionsQueryVariables,
      options?: Omit<UseQueryOptions<EntityJoinDefinitionsQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityJoinDefinitionsQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityJoinDefinitionsQuery, TError, TData>(
      {
    queryKey: ['EntityJoinDefinitions', variables],
    queryFn: () => graphqlRequest<EntityJoinDefinitionsQuery, EntityJoinDefinitionsQueryVariables>(EntityJoinDefinitionsDocument, variables),
    ...options
  }
    )};

useEntityJoinDefinitionsQuery.getKey = (variables: EntityJoinDefinitionsQueryVariables) => ['EntityJoinDefinitions', variables];


useEntityJoinDefinitionsQuery.fetcher = (variables: EntityJoinDefinitionsQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityJoinDefinitionsQuery, EntityJoinDefinitionsQueryVariables>(EntityJoinDefinitionsDocument, variables, options);

export const DeleteSchemaDocument = `
    mutation DeleteSchema($id: String!) {
  deleteEntitySchema(id: $id)
}
    `;

export const useDeleteSchemaMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<DeleteSchemaMutation, TError, DeleteSchemaMutationVariables, TContext>) => {
    
    return useMutation<DeleteSchemaMutation, TError, DeleteSchemaMutationVariables, TContext>(
      {
    mutationKey: ['DeleteSchema'],
    mutationFn: (variables?: DeleteSchemaMutationVariables) => graphqlRequest<DeleteSchemaMutation, DeleteSchemaMutationVariables>(DeleteSchemaDocument, variables),
    ...options
  }
    )};

useDeleteSchemaMutation.getKey = () => ['DeleteSchema'];


useDeleteSchemaMutation.fetcher = (variables: DeleteSchemaMutationVariables, options?: RequestInit['headers']) => graphqlRequest<DeleteSchemaMutation, DeleteSchemaMutationVariables>(DeleteSchemaDocument, variables, options);

export const DeleteJoinDocument = `
    mutation DeleteJoin($id: String!) {
  deleteEntityJoinDefinition(id: $id)
}
    `;

export const useDeleteJoinMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<DeleteJoinMutation, TError, DeleteJoinMutationVariables, TContext>) => {
    
    return useMutation<DeleteJoinMutation, TError, DeleteJoinMutationVariables, TContext>(
      {
    mutationKey: ['DeleteJoin'],
    mutationFn: (variables?: DeleteJoinMutationVariables) => graphqlRequest<DeleteJoinMutation, DeleteJoinMutationVariables>(DeleteJoinDocument, variables),
    ...options
  }
    )};

useDeleteJoinMutation.getKey = () => ['DeleteJoin'];


useDeleteJoinMutation.fetcher = (variables: DeleteJoinMutationVariables, options?: RequestInit['headers']) => graphqlRequest<DeleteJoinMutation, DeleteJoinMutationVariables>(DeleteJoinDocument, variables, options);

export const ExecuteJoinDocument = `
    query ExecuteJoin($input: ExecuteEntityJoinInput!) {
  executeEntityJoin(input: $input) {
    edges {
      left {
        id
        entityType
        properties
      }
      right {
        id
        entityType
        properties
      }
    }
    pageInfo {
      totalCount
      hasNextPage
      hasPreviousPage
    }
  }
}
    `;

export const useExecuteJoinQuery = <
      TData = ExecuteJoinQuery,
      TError = unknown
    >(
      variables: ExecuteJoinQueryVariables,
      options?: Omit<UseQueryOptions<ExecuteJoinQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<ExecuteJoinQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<ExecuteJoinQuery, TError, TData>(
      {
    queryKey: ['ExecuteJoin', variables],
    queryFn: () => graphqlRequest<ExecuteJoinQuery, ExecuteJoinQueryVariables>(ExecuteJoinDocument, variables),
    ...options
  }
    )};

useExecuteJoinQuery.getKey = (variables: ExecuteJoinQueryVariables) => ['ExecuteJoin', variables];


useExecuteJoinQuery.fetcher = (variables: ExecuteJoinQueryVariables, options?: RequestInit['headers']) => graphqlRequest<ExecuteJoinQuery, ExecuteJoinQueryVariables>(ExecuteJoinDocument, variables, options);

export const EntityTransformationsDocument = `
    query EntityTransformations($organizationId: String!) {
  entityTransformations(organizationId: $organizationId) {
    id
    organizationId
    name
    description
    createdAt
    updatedAt
    nodes {
      id
    }
  }
}
    `;

export const useEntityTransformationsQuery = <
      TData = EntityTransformationsQuery,
      TError = unknown
    >(
      variables: EntityTransformationsQueryVariables,
      options?: Omit<UseQueryOptions<EntityTransformationsQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityTransformationsQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityTransformationsQuery, TError, TData>(
      {
    queryKey: ['EntityTransformations', variables],
    queryFn: () => graphqlRequest<EntityTransformationsQuery, EntityTransformationsQueryVariables>(EntityTransformationsDocument, variables),
    ...options
  }
    )};

useEntityTransformationsQuery.getKey = (variables: EntityTransformationsQueryVariables) => ['EntityTransformations', variables];


useEntityTransformationsQuery.fetcher = (variables: EntityTransformationsQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityTransformationsQuery, EntityTransformationsQueryVariables>(EntityTransformationsDocument, variables, options);

export const EntityTransformationDocument = `
    query EntityTransformation($id: String!) {
  entityTransformation(id: $id) {
    id
    organizationId
    name
    description
    createdAt
    updatedAt
    nodes {
      ...EntityTransformationNodeFields
    }
  }
}
    ${EntityTransformationNodeFieldsFragmentDoc}`;

export const useEntityTransformationQuery = <
      TData = EntityTransformationQuery,
      TError = unknown
    >(
      variables: EntityTransformationQueryVariables,
      options?: Omit<UseQueryOptions<EntityTransformationQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityTransformationQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityTransformationQuery, TError, TData>(
      {
    queryKey: ['EntityTransformation', variables],
    queryFn: () => graphqlRequest<EntityTransformationQuery, EntityTransformationQueryVariables>(EntityTransformationDocument, variables),
    ...options
  }
    )};

useEntityTransformationQuery.getKey = (variables: EntityTransformationQueryVariables) => ['EntityTransformation', variables];


useEntityTransformationQuery.fetcher = (variables: EntityTransformationQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityTransformationQuery, EntityTransformationQueryVariables>(EntityTransformationDocument, variables, options);

export const CreateEntityTransformationDocument = `
    mutation CreateEntityTransformation($input: CreateEntityTransformationInput!) {
  createEntityTransformation(input: $input) {
    id
    organizationId
    name
    description
    createdAt
    updatedAt
    nodes {
      ...EntityTransformationNodeFields
    }
  }
}
    ${EntityTransformationNodeFieldsFragmentDoc}`;

export const useCreateEntityTransformationMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<CreateEntityTransformationMutation, TError, CreateEntityTransformationMutationVariables, TContext>) => {
    
    return useMutation<CreateEntityTransformationMutation, TError, CreateEntityTransformationMutationVariables, TContext>(
      {
    mutationKey: ['CreateEntityTransformation'],
    mutationFn: (variables?: CreateEntityTransformationMutationVariables) => graphqlRequest<CreateEntityTransformationMutation, CreateEntityTransformationMutationVariables>(CreateEntityTransformationDocument, variables),
    ...options
  }
    )};

useCreateEntityTransformationMutation.getKey = () => ['CreateEntityTransformation'];


useCreateEntityTransformationMutation.fetcher = (variables: CreateEntityTransformationMutationVariables, options?: RequestInit['headers']) => graphqlRequest<CreateEntityTransformationMutation, CreateEntityTransformationMutationVariables>(CreateEntityTransformationDocument, variables, options);

export const UpdateEntityTransformationDocument = `
    mutation UpdateEntityTransformation($input: UpdateEntityTransformationInput!) {
  updateEntityTransformation(input: $input) {
    id
    organizationId
    name
    description
    createdAt
    updatedAt
    nodes {
      ...EntityTransformationNodeFields
    }
  }
}
    ${EntityTransformationNodeFieldsFragmentDoc}`;

export const useUpdateEntityTransformationMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<UpdateEntityTransformationMutation, TError, UpdateEntityTransformationMutationVariables, TContext>) => {
    
    return useMutation<UpdateEntityTransformationMutation, TError, UpdateEntityTransformationMutationVariables, TContext>(
      {
    mutationKey: ['UpdateEntityTransformation'],
    mutationFn: (variables?: UpdateEntityTransformationMutationVariables) => graphqlRequest<UpdateEntityTransformationMutation, UpdateEntityTransformationMutationVariables>(UpdateEntityTransformationDocument, variables),
    ...options
  }
    )};

useUpdateEntityTransformationMutation.getKey = () => ['UpdateEntityTransformation'];


useUpdateEntityTransformationMutation.fetcher = (variables: UpdateEntityTransformationMutationVariables, options?: RequestInit['headers']) => graphqlRequest<UpdateEntityTransformationMutation, UpdateEntityTransformationMutationVariables>(UpdateEntityTransformationDocument, variables, options);

export const DeleteEntityTransformationDocument = `
    mutation DeleteEntityTransformation($id: String!) {
  deleteEntityTransformation(id: $id)
}
    `;

export const useDeleteEntityTransformationMutation = <
      TError = unknown,
      TContext = unknown
    >(options?: UseMutationOptions<DeleteEntityTransformationMutation, TError, DeleteEntityTransformationMutationVariables, TContext>) => {
    
    return useMutation<DeleteEntityTransformationMutation, TError, DeleteEntityTransformationMutationVariables, TContext>(
      {
    mutationKey: ['DeleteEntityTransformation'],
    mutationFn: (variables?: DeleteEntityTransformationMutationVariables) => graphqlRequest<DeleteEntityTransformationMutation, DeleteEntityTransformationMutationVariables>(DeleteEntityTransformationDocument, variables),
    ...options
  }
    )};

useDeleteEntityTransformationMutation.getKey = () => ['DeleteEntityTransformation'];


useDeleteEntityTransformationMutation.fetcher = (variables: DeleteEntityTransformationMutationVariables, options?: RequestInit['headers']) => graphqlRequest<DeleteEntityTransformationMutation, DeleteEntityTransformationMutationVariables>(DeleteEntityTransformationDocument, variables, options);

export const ExecuteEntityTransformationDocument = `
    query ExecuteEntityTransformation($input: ExecuteEntityTransformationInput!) {
  executeEntityTransformation(input: $input) {
    edges {
      entities {
        alias
        entity {
          id
          entityType
          path
          referenceValue
          properties
        }
      }
    }
    pageInfo {
      totalCount
      hasNextPage
      hasPreviousPage
    }
  }
}
    `;

export const useExecuteEntityTransformationQuery = <
      TData = ExecuteEntityTransformationQuery,
      TError = unknown
    >(
      variables: ExecuteEntityTransformationQueryVariables,
      options?: Omit<UseQueryOptions<ExecuteEntityTransformationQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<ExecuteEntityTransformationQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<ExecuteEntityTransformationQuery, TError, TData>(
      {
    queryKey: ['ExecuteEntityTransformation', variables],
    queryFn: () => graphqlRequest<ExecuteEntityTransformationQuery, ExecuteEntityTransformationQueryVariables>(ExecuteEntityTransformationDocument, variables),
    ...options
  }
    )};

useExecuteEntityTransformationQuery.getKey = (variables: ExecuteEntityTransformationQueryVariables) => ['ExecuteEntityTransformation', variables];


useExecuteEntityTransformationQuery.fetcher = (variables: ExecuteEntityTransformationQueryVariables, options?: RequestInit['headers']) => graphqlRequest<ExecuteEntityTransformationQuery, ExecuteEntityTransformationQueryVariables>(ExecuteEntityTransformationDocument, variables, options);

export const TransformationExecutionDocument = `
    query TransformationExecution($transformationId: String!, $filters: [TransformationExecutionFilterInput!], $sort: TransformationExecutionSortInput, $pagination: PaginationInput) {
  transformationExecution(
    transformationId: $transformationId
    filters: $filters
    sort: $sort
    pagination: $pagination
  ) {
    columns {
      key
      alias
      field
      label
      sourceAlias
      sourceField
    }
    rows {
      values {
        columnKey
        value
      }
    }
    pageInfo {
      totalCount
      hasNextPage
      hasPreviousPage
    }
  }
}
    `;

export const useTransformationExecutionQuery = <
      TData = TransformationExecutionQuery,
      TError = unknown
    >(
      variables: TransformationExecutionQueryVariables,
      options?: Omit<UseQueryOptions<TransformationExecutionQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<TransformationExecutionQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<TransformationExecutionQuery, TError, TData>(
      {
    queryKey: ['TransformationExecution', variables],
    queryFn: () => graphqlRequest<TransformationExecutionQuery, TransformationExecutionQueryVariables>(TransformationExecutionDocument, variables),
    ...options
  }
    )};

useTransformationExecutionQuery.getKey = (variables: TransformationExecutionQueryVariables) => ['TransformationExecution', variables];


useTransformationExecutionQuery.fetcher = (variables: TransformationExecutionQueryVariables, options?: RequestInit['headers']) => graphqlRequest<TransformationExecutionQuery, TransformationExecutionQueryVariables>(TransformationExecutionDocument, variables, options);

export const EntityExportJobsDocument = `
    query EntityExportJobs($organizationId: String!, $statuses: [EntityExportJobStatus!], $limit: Int, $offset: Int) {
  entityExportJobs(
    organizationId: $organizationId
    statuses: $statuses
    limit: $limit
    offset: $offset
  ) {
    id
    organizationId
    jobType
    entityType
    transformationId
    status
    rowsRequested
    rowsExported
    bytesWritten
    fileMimeType
    fileByteSize
    errorMessage
    filters {
      key
      value
      exists
      inArray
    }
    transformationDefinition {
      id
      name
    }
    enqueuedAt
    startedAt
    completedAt
    updatedAt
    downloadUrl
  }
}
    `;

export const useEntityExportJobsQuery = <
      TData = EntityExportJobsQuery,
      TError = unknown
    >(
      variables: EntityExportJobsQueryVariables,
      options?: Omit<UseQueryOptions<EntityExportJobsQuery, TError, TData>, 'queryKey'> & { queryKey?: UseQueryOptions<EntityExportJobsQuery, TError, TData>['queryKey'] }
    ) => {
    
    return useQuery<EntityExportJobsQuery, TError, TData>(
      {
    queryKey: ['EntityExportJobs', variables],
    queryFn: () => graphqlRequest<EntityExportJobsQuery, EntityExportJobsQueryVariables>(EntityExportJobsDocument, variables),
    ...options
  }
    )};

useEntityExportJobsQuery.getKey = (variables: EntityExportJobsQueryVariables) => ['EntityExportJobs', variables];


useEntityExportJobsQuery.fetcher = (variables: EntityExportJobsQueryVariables, options?: RequestInit['headers']) => graphqlRequest<EntityExportJobsQuery, EntityExportJobsQueryVariables>(EntityExportJobsDocument, variables, options);
