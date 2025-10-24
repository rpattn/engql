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

export type ExecuteEntityJoinInput = {
  joinId: Scalars['String']['input'];
  leftFilters?: InputMaybe<Array<PropertyFilter>>;
  pagination?: InputMaybe<PaginationInput>;
  rightFilters?: InputMaybe<Array<PropertyFilter>>;
  sortCriteria?: InputMaybe<Array<JoinSortInput>>;
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
  EntityId = 'ENTITY_ID',
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
  createEntity: Entity;
  createEntityJoinDefinition: EntityJoinDefinition;
  createEntitySchema: EntitySchema;
  createOrganization: Organization;
  deleteEntity: Scalars['Boolean']['output'];
  deleteEntityJoinDefinition: Scalars['Boolean']['output'];
  deleteEntitySchema: Scalars['Boolean']['output'];
  deleteOrganization: Scalars['Boolean']['output'];
  removeFieldFromSchema: EntitySchema;
  rollbackEntity: Entity;
  updateEntity: Entity;
  updateEntityJoinDefinition: EntityJoinDefinition;
  updateEntitySchema: EntitySchema;
  updateOrganization: Organization;
};


export type MutationAddFieldToSchemaArgs = {
  field: FieldDefinitionInput;
  schemaId: Scalars['String']['input'];
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


export type MutationDeleteOrganizationArgs = {
  id: Scalars['String']['input'];
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
  entityJoinDefinition?: Maybe<EntityJoinDefinition>;
  entityJoinDefinitions: Array<EntityJoinDefinition>;
  entitySchema?: Maybe<EntitySchema>;
  entitySchemaByName?: Maybe<EntitySchema>;
  entitySchemaVersions: Array<EntitySchema>;
  entitySchemas: Array<EntitySchema>;
  executeEntityJoin: EntityJoinConnection;
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
  validateEntityAgainstSchema: ValidationResult;
};


export type QueryEntitiesArgs = {
  filter?: InputMaybe<EntityFilter>;
  organizationId: Scalars['String']['input'];
  pagination?: InputMaybe<PaginationInput>;
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


export type QueryExecuteEntityJoinArgs = {
  input: ExecuteEntityJoinInput;
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


export type QueryValidateEntityAgainstSchemaArgs = {
  entityId: Scalars['String']['input'];
};

export enum SchemaStatus {
  Active = 'ACTIVE',
  Archived = 'ARCHIVED',
  Deprecated = 'DEPRECATED',
  Draft = 'DRAFT'
}

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
}>;


export type EntitiesManagementQuery = { __typename?: 'Query', entities: { __typename?: 'EntityConnection', entities: Array<{ __typename?: 'Entity', id: string, organizationId: string, schemaId: string, entityType: string, path: string, properties: string, referenceValue?: string | null, version: number, createdAt: string, updatedAt: string, linkedEntities: Array<{ __typename?: 'Entity', id: string, entityType: string, properties: string, referenceValue?: string | null }> }>, pageInfo: { __typename?: 'PageInfo', totalCount: number, hasNextPage: boolean, hasPreviousPage: boolean } } };

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
    query EntitiesManagement($organizationId: String!, $pagination: PaginationInput, $filter: EntityFilter) {
  entities(
    organizationId: $organizationId
    pagination: $pagination
    filter: $filter
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
      linkedEntities {
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
