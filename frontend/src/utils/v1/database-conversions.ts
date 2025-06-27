import { fromJson, toJson } from "@bufbuild/protobuf";
import type { 
  Database as OldDatabase,
  GetDatabaseRequest as OldGetDatabaseRequest,
  BatchGetDatabasesRequest as OldBatchGetDatabasesRequest,
  BatchGetDatabasesResponse as OldBatchGetDatabasesResponse,
  ListDatabasesRequest as OldListDatabasesRequest,
  ListDatabasesResponse as OldListDatabasesResponse,
  UpdateDatabaseRequest as OldUpdateDatabaseRequest,
  BatchUpdateDatabasesRequest as OldBatchUpdateDatabasesRequest,
  BatchUpdateDatabasesResponse as OldBatchUpdateDatabasesResponse,
  BatchSyncDatabasesRequest as OldBatchSyncDatabasesRequest,
  BatchSyncDatabasesResponse as OldBatchSyncDatabasesResponse,
  SyncDatabaseRequest as OldSyncDatabaseRequest,
  SyncDatabaseResponse as OldSyncDatabaseResponse,
  GetDatabaseMetadataRequest as OldGetDatabaseMetadataRequest,
  GetDatabaseSchemaRequest as OldGetDatabaseSchemaRequest,
  DiffSchemaRequest as OldDiffSchemaRequest,
  ListSecretsRequest as OldListSecretsRequest,
  ListSecretsResponse as OldListSecretsResponse,
  UpdateSecretRequest as OldUpdateSecretRequest,
  DeleteSecretRequest as OldDeleteSecretRequest,
  ListChangelogsRequest as OldListChangelogsRequest,
  ListChangelogsResponse as OldListChangelogsResponse,
  GetChangelogRequest as OldGetChangelogRequest,
  GetSchemaStringRequest as OldGetSchemaStringRequest,
  DatabaseMetadata as OldDatabaseMetadata,
  Changelog as OldChangelog,
  Secret as OldSecret,
  DatabaseSchema as OldDatabaseSchema,
  DiffSchemaResponse as OldDiffSchemaResponse,
  GetSchemaStringResponse as OldGetSchemaStringResponse,
  SchemaMetadata as OldSchemaMetadata,
  TableMetadata as OldTableMetadata,
  ColumnMetadata as OldColumnMetadata,
} from "@/types/proto/v1/database_service";
import {
  Database as OldDatabaseProto,
  GetDatabaseRequest as OldGetDatabaseRequestProto,
  BatchGetDatabasesRequest as OldBatchGetDatabasesRequestProto,
  BatchGetDatabasesResponse as OldBatchGetDatabasesResponseProto,
  ListDatabasesRequest as OldListDatabasesRequestProto,
  ListDatabasesResponse as OldListDatabasesResponseProto,
  UpdateDatabaseRequest as OldUpdateDatabaseRequestProto,
  BatchUpdateDatabasesRequest as OldBatchUpdateDatabasesRequestProto,
  BatchUpdateDatabasesResponse as OldBatchUpdateDatabasesResponseProto,
  BatchSyncDatabasesRequest as OldBatchSyncDatabasesRequestProto,
  BatchSyncDatabasesResponse as OldBatchSyncDatabasesResponseProto,
  SyncDatabaseRequest as OldSyncDatabaseRequestProto,
  SyncDatabaseResponse as OldSyncDatabaseResponseProto,
  GetDatabaseMetadataRequest as OldGetDatabaseMetadataRequestProto,
  GetDatabaseSchemaRequest as OldGetDatabaseSchemaRequestProto,
  DiffSchemaRequest as OldDiffSchemaRequestProto,
  ListSecretsRequest as OldListSecretsRequestProto,
  ListSecretsResponse as OldListSecretsResponseProto,
  UpdateSecretRequest as OldUpdateSecretRequestProto,
  DeleteSecretRequest as OldDeleteSecretRequestProto,
  ListChangelogsRequest as OldListChangelogsRequestProto,
  ListChangelogsResponse as OldListChangelogsResponseProto,
  GetChangelogRequest as OldGetChangelogRequestProto,
  GetSchemaStringRequest as OldGetSchemaStringRequestProto,
  DatabaseMetadata as OldDatabaseMetadataProto,
  Changelog as OldChangelogProto,
  Secret as OldSecretProto,
  DatabaseSchema as OldDatabaseSchemaProto,
  DiffSchemaResponse as OldDiffSchemaResponseProto,
  GetSchemaStringResponse as OldGetSchemaStringResponseProto,
  SchemaMetadata as OldSchemaMetadataProto,
  TableMetadata as OldTableMetadataProto,
  ColumnMetadata as OldColumnMetadataProto,
  ChangelogView as OldChangelogView,
  GetSchemaStringRequest_ObjectType as OldObjectType
} from "@/types/proto/v1/database_service";
import type { 
  Database as NewDatabase,
  GetDatabaseRequest as NewGetDatabaseRequest,
  BatchGetDatabasesRequest as NewBatchGetDatabasesRequest,
  BatchGetDatabasesResponse as NewBatchGetDatabasesResponse,
  ListDatabasesRequest as NewListDatabasesRequest,
  ListDatabasesResponse as NewListDatabasesResponse,
  UpdateDatabaseRequest as NewUpdateDatabaseRequest,
  BatchUpdateDatabasesRequest as NewBatchUpdateDatabasesRequest,
  BatchUpdateDatabasesResponse as NewBatchUpdateDatabasesResponse,
  BatchSyncDatabasesRequest as NewBatchSyncDatabasesRequest,
  BatchSyncDatabasesResponse as NewBatchSyncDatabasesResponse,
  SyncDatabaseRequest as NewSyncDatabaseRequest,
  SyncDatabaseResponse as NewSyncDatabaseResponse,
  GetDatabaseMetadataRequest as NewGetDatabaseMetadataRequest,
  GetDatabaseSchemaRequest as NewGetDatabaseSchemaRequest,
  DiffSchemaRequest as NewDiffSchemaRequest,
  ListSecretsRequest as NewListSecretsRequest,
  ListSecretsResponse as NewListSecretsResponse,
  UpdateSecretRequest as NewUpdateSecretRequest,
  DeleteSecretRequest as NewDeleteSecretRequest,
  ListChangelogsRequest as NewListChangelogsRequest,
  ListChangelogsResponse as NewListChangelogsResponse,
  GetChangelogRequest as NewGetChangelogRequest,
  GetSchemaStringRequest as NewGetSchemaStringRequest,
  DatabaseMetadata as NewDatabaseMetadata,
  Changelog as NewChangelog,
  Secret as NewSecret,
  DatabaseSchema as NewDatabaseSchemaType,
  DiffSchemaResponse as NewDiffSchemaResponse,
  GetSchemaStringResponse as NewGetSchemaStringResponse,
  SchemaMetadata as NewSchemaMetadata,
  TableMetadata as NewTableMetadata,
  ColumnMetadata as NewColumnMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  GetDatabaseRequestSchema,
  BatchGetDatabasesRequestSchema,
  BatchGetDatabasesResponseSchema,
  ListDatabasesRequestSchema,
  ListDatabasesResponseSchema,
  UpdateDatabaseRequestSchema,
  BatchUpdateDatabasesRequestSchema,
  BatchUpdateDatabasesResponseSchema,
  BatchSyncDatabasesRequestSchema,
  BatchSyncDatabasesResponseSchema,
  SyncDatabaseRequestSchema,
  SyncDatabaseResponseSchema,
  GetDatabaseMetadataRequestSchema,
  GetDatabaseSchemaRequestSchema,
  DiffSchemaRequestSchema,
  ListSecretsRequestSchema,
  ListSecretsResponseSchema,
  UpdateSecretRequestSchema,
  DeleteSecretRequestSchema,
  ListChangelogsRequestSchema,
  ListChangelogsResponseSchema,
  GetChangelogRequestSchema,
  GetSchemaStringRequestSchema,
  DatabaseMetadataSchema as NewDatabaseMetadataSchema,
  ChangelogSchema as NewChangelogSchema,
  SecretSchema as NewSecretSchema,
  DatabaseSchemaSchema as NewDatabaseSchemaSchema,
  DiffSchemaResponseSchema as NewDiffSchemaResponseSchema,
  GetSchemaStringResponseSchema as NewGetSchemaStringResponseSchema,
  SchemaMetadataSchema as NewSchemaMetadataSchema,
  TableMetadataSchema as NewTableMetadataSchema,
  ColumnMetadataSchema as NewColumnMetadataSchema,
  ChangelogView as NewChangelogView,
  GetSchemaStringRequest_ObjectType as NewObjectType,
  DatabaseSchema$
} from "@/types/proto-es/v1/database_service_pb";



// Convert old ChangelogView enum to new
export const convertOldChangelogViewToNew = (oldView: OldChangelogView): NewChangelogView => {
  const mapping: Record<OldChangelogView, NewChangelogView> = {
    [OldChangelogView.CHANGELOG_VIEW_UNSPECIFIED]: NewChangelogView.UNSPECIFIED,
    [OldChangelogView.CHANGELOG_VIEW_BASIC]: NewChangelogView.BASIC,
    [OldChangelogView.CHANGELOG_VIEW_FULL]: NewChangelogView.FULL,
    [OldChangelogView.UNRECOGNIZED]: NewChangelogView.UNSPECIFIED,
  };
  return mapping[oldView] ?? NewChangelogView.UNSPECIFIED;
};

// Convert new ChangelogView enum to old
export const convertNewChangelogViewToOld = (newView: NewChangelogView): OldChangelogView => {
  const mapping: Record<NewChangelogView, OldChangelogView> = {
    [NewChangelogView.UNSPECIFIED]: OldChangelogView.CHANGELOG_VIEW_UNSPECIFIED,
    [NewChangelogView.BASIC]: OldChangelogView.CHANGELOG_VIEW_BASIC,
    [NewChangelogView.FULL]: OldChangelogView.CHANGELOG_VIEW_FULL,
  };
  return mapping[newView] ?? OldChangelogView.UNRECOGNIZED;
};

// Convert old GetSchemaStringRequest_ObjectType enum to new
export const convertOldObjectTypeToNew = (oldType?: OldObjectType): NewObjectType | undefined => {
  if (!oldType) return undefined;
  const mapping: Record<OldObjectType, NewObjectType> = {
    [OldObjectType.OBJECT_TYPE_UNSPECIFIED]: NewObjectType.OBJECT_TYPE_UNSPECIFIED,
    [OldObjectType.DATABASE]: NewObjectType.DATABASE,
    [OldObjectType.SCHEMA]: NewObjectType.SCHEMA,
    [OldObjectType.TABLE]: NewObjectType.TABLE,
    [OldObjectType.VIEW]: NewObjectType.VIEW,
    [OldObjectType.MATERIALIZED_VIEW]: NewObjectType.MATERIALIZED_VIEW,
    [OldObjectType.FUNCTION]: NewObjectType.FUNCTION,
    [OldObjectType.PROCEDURE]: NewObjectType.PROCEDURE,
    [OldObjectType.SEQUENCE]: NewObjectType.SEQUENCE,
    [OldObjectType.UNRECOGNIZED]: NewObjectType.OBJECT_TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewObjectType.OBJECT_TYPE_UNSPECIFIED;
};

// Convert old Database proto to proto-es
export const convertOldDatabaseToNew = (oldDatabase: OldDatabase): NewDatabase => {
  const json = OldDatabaseProto.toJSON(oldDatabase) as any;
  return fromJson(DatabaseSchema$, json);
};

// Convert proto-es Database to old proto
export const convertNewDatabaseToOld = (newDatabase: NewDatabase): OldDatabase => {
  const json = toJson(DatabaseSchema$, newDatabase);
  return OldDatabaseProto.fromJSON(json);
};

// Convert old DatabaseMetadata proto to proto-es
export const convertOldDatabaseMetadataToNew = (oldMetadata: OldDatabaseMetadata): NewDatabaseMetadata => {
  const json = OldDatabaseMetadataProto.toJSON(oldMetadata) as any;
  return fromJson(NewDatabaseMetadataSchema, json);
};

// Convert proto-es DatabaseMetadata to old proto
export const convertNewDatabaseMetadataToOld = (newMetadata: NewDatabaseMetadata): OldDatabaseMetadata => {
  const json = toJson(NewDatabaseMetadataSchema, newMetadata);
  return OldDatabaseMetadataProto.fromJSON(json);
};

// Convert old Changelog proto to proto-es
export const convertOldChangelogToNew = (oldChangelog: OldChangelog): NewChangelog => {
  const json = OldChangelogProto.toJSON(oldChangelog) as any;
  return fromJson(NewChangelogSchema, json);
};

// Convert proto-es Changelog to old proto
export const convertNewChangelogToOld = (newChangelog: NewChangelog): OldChangelog => {
  const json = toJson(NewChangelogSchema, newChangelog);
  return OldChangelogProto.fromJSON(json);
};

// Convert old Secret proto to proto-es
export const convertOldSecretToNew = (oldSecret: OldSecret): NewSecret => {
  const json = OldSecretProto.toJSON(oldSecret) as any;
  return fromJson(NewSecretSchema, json);
};

// Convert proto-es Secret to old proto
export const convertNewSecretToOld = (newSecret: NewSecret): OldSecret => {
  const json = toJson(NewSecretSchema, newSecret);
  return OldSecretProto.fromJSON(json);
};

// Convert old DatabaseSchema proto to proto-es
export const convertOldDatabaseSchemaToNew = (oldSchema: OldDatabaseSchema): NewDatabaseSchemaType => {
  const json = OldDatabaseSchemaProto.toJSON(oldSchema) as any;
  return fromJson(NewDatabaseSchemaSchema, json);
};

// Convert proto-es DatabaseSchema to old proto
export const convertNewDatabaseSchemaToOld = (newSchema: NewDatabaseSchemaType): OldDatabaseSchema => {
  const json = toJson(NewDatabaseSchemaSchema, newSchema);
  return OldDatabaseSchemaProto.fromJSON(json);
};

// Convert old DiffSchemaResponse proto to proto-es
export const convertOldDiffSchemaResponseToNew = (oldResponse: OldDiffSchemaResponse): NewDiffSchemaResponse => {
  const json = OldDiffSchemaResponseProto.toJSON(oldResponse) as any;
  return fromJson(NewDiffSchemaResponseSchema, json);
};

// Convert proto-es DiffSchemaResponse to old proto
export const convertNewDiffSchemaResponseToOld = (newResponse: NewDiffSchemaResponse): OldDiffSchemaResponse => {
  const json = toJson(NewDiffSchemaResponseSchema, newResponse);
  return OldDiffSchemaResponseProto.fromJSON(json);
};

// Convert old GetSchemaStringResponse proto to proto-es
export const convertOldGetSchemaStringResponseToNew = (oldResponse: OldGetSchemaStringResponse): NewGetSchemaStringResponse => {
  const json = OldGetSchemaStringResponseProto.toJSON(oldResponse) as any;
  return fromJson(NewGetSchemaStringResponseSchema, json);
};

// Convert proto-es GetSchemaStringResponse to old proto
export const convertNewGetSchemaStringResponseToOld = (newResponse: NewGetSchemaStringResponse): OldGetSchemaStringResponse => {
  const json = toJson(NewGetSchemaStringResponseSchema, newResponse);
  return OldGetSchemaStringResponseProto.fromJSON(json);
};

// Convert old SchemaMetadata proto to proto-es
export const convertOldSchemaMetadataToNew = (oldMetadata: OldSchemaMetadata): NewSchemaMetadata => {
  const json = OldSchemaMetadataProto.toJSON(oldMetadata) as any;
  return fromJson(NewSchemaMetadataSchema, json);
};

// Convert proto-es SchemaMetadata to old proto
export const convertNewSchemaMetadataToOld = (newMetadata: NewSchemaMetadata): OldSchemaMetadata => {
  const json = toJson(NewSchemaMetadataSchema, newMetadata);
  return OldSchemaMetadataProto.fromJSON(json);
};

// Convert old TableMetadata proto to proto-es
export const convertOldTableMetadataToNew = (oldMetadata: OldTableMetadata): NewTableMetadata => {
  const json = OldTableMetadataProto.toJSON(oldMetadata) as any;
  return fromJson(NewTableMetadataSchema, json);
};

// Convert proto-es TableMetadata to old proto
export const convertNewTableMetadataToOld = (newMetadata: NewTableMetadata): OldTableMetadata => {
  const json = toJson(NewTableMetadataSchema, newMetadata);
  return OldTableMetadataProto.fromJSON(json);
};

// Convert old ColumnMetadata proto to proto-es
export const convertOldColumnMetadataToNew = (oldMetadata: OldColumnMetadata): NewColumnMetadata => {
  const json = OldColumnMetadataProto.toJSON(oldMetadata) as any;
  return fromJson(NewColumnMetadataSchema, json);
};

// Convert proto-es ColumnMetadata to old proto
export const convertNewColumnMetadataToOld = (newMetadata: NewColumnMetadata): OldColumnMetadata => {
  const json = toJson(NewColumnMetadataSchema, newMetadata);
  return OldColumnMetadataProto.fromJSON(json);
};

// ========== REQUEST/RESPONSE CONVERSIONS ==========

// Convert old GetDatabaseRequest proto to proto-es
export const convertOldGetDatabaseRequestToNew = (oldRequest: OldGetDatabaseRequest): NewGetDatabaseRequest => {
  const json = OldGetDatabaseRequestProto.toJSON(oldRequest) as any;
  return fromJson(GetDatabaseRequestSchema, json);
};

// Convert proto-es GetDatabaseRequest to old proto
export const convertNewGetDatabaseRequestToOld = (newRequest: NewGetDatabaseRequest): OldGetDatabaseRequest => {
  const json = toJson(GetDatabaseRequestSchema, newRequest);
  return OldGetDatabaseRequestProto.fromJSON(json);
};

// Convert old BatchGetDatabasesRequest proto to proto-es
export const convertOldBatchGetDatabasesRequestToNew = (oldRequest: OldBatchGetDatabasesRequest): NewBatchGetDatabasesRequest => {
  const json = OldBatchGetDatabasesRequestProto.toJSON(oldRequest) as any;
  return fromJson(BatchGetDatabasesRequestSchema, json);
};

// Convert proto-es BatchGetDatabasesRequest to old proto
export const convertNewBatchGetDatabasesRequestToOld = (newRequest: NewBatchGetDatabasesRequest): OldBatchGetDatabasesRequest => {
  const json = toJson(BatchGetDatabasesRequestSchema, newRequest);
  return OldBatchGetDatabasesRequestProto.fromJSON(json);
};

// Convert old BatchGetDatabasesResponse proto to proto-es
export const convertOldBatchGetDatabasesResponseToNew = (oldResponse: OldBatchGetDatabasesResponse): NewBatchGetDatabasesResponse => {
  const json = OldBatchGetDatabasesResponseProto.toJSON(oldResponse) as any;
  return fromJson(BatchGetDatabasesResponseSchema, json);
};

// Convert proto-es BatchGetDatabasesResponse to old proto
export const convertNewBatchGetDatabasesResponseToOld = (newResponse: NewBatchGetDatabasesResponse): OldBatchGetDatabasesResponse => {
  const json = toJson(BatchGetDatabasesResponseSchema, newResponse);
  return OldBatchGetDatabasesResponseProto.fromJSON(json);
};

// Convert old ListDatabasesRequest proto to proto-es
export const convertOldListDatabasesRequestToNew = (oldRequest: OldListDatabasesRequest): NewListDatabasesRequest => {
  const json = OldListDatabasesRequestProto.toJSON(oldRequest) as any;
  return fromJson(ListDatabasesRequestSchema, json);
};

// Convert proto-es ListDatabasesRequest to old proto
export const convertNewListDatabasesRequestToOld = (newRequest: NewListDatabasesRequest): OldListDatabasesRequest => {
  const json = toJson(ListDatabasesRequestSchema, newRequest);
  return OldListDatabasesRequestProto.fromJSON(json);
};

// Convert old ListDatabasesResponse proto to proto-es
export const convertOldListDatabasesResponseToNew = (oldResponse: OldListDatabasesResponse): NewListDatabasesResponse => {
  const json = OldListDatabasesResponseProto.toJSON(oldResponse) as any;
  return fromJson(ListDatabasesResponseSchema, json);
};

// Convert proto-es ListDatabasesResponse to old proto
export const convertNewListDatabasesResponseToOld = (newResponse: NewListDatabasesResponse): OldListDatabasesResponse => {
  const json = toJson(ListDatabasesResponseSchema, newResponse);
  return OldListDatabasesResponseProto.fromJSON(json);
};

// Convert old UpdateDatabaseRequest proto to proto-es
export const convertOldUpdateDatabaseRequestToNew = (oldRequest: OldUpdateDatabaseRequest): NewUpdateDatabaseRequest => {
  const json = OldUpdateDatabaseRequestProto.toJSON(oldRequest) as any;
  return fromJson(UpdateDatabaseRequestSchema, json);
};

// Convert proto-es UpdateDatabaseRequest to old proto
export const convertNewUpdateDatabaseRequestToOld = (newRequest: NewUpdateDatabaseRequest): OldUpdateDatabaseRequest => {
  const json = toJson(UpdateDatabaseRequestSchema, newRequest);
  return OldUpdateDatabaseRequestProto.fromJSON(json);
};

// Convert old BatchUpdateDatabasesRequest proto to proto-es
export const convertOldBatchUpdateDatabasesRequestToNew = (oldRequest: OldBatchUpdateDatabasesRequest): NewBatchUpdateDatabasesRequest => {
  const json = OldBatchUpdateDatabasesRequestProto.toJSON(oldRequest) as any;
  return fromJson(BatchUpdateDatabasesRequestSchema, json);
};

// Convert proto-es BatchUpdateDatabasesRequest to old proto
export const convertNewBatchUpdateDatabasesRequestToOld = (newRequest: NewBatchUpdateDatabasesRequest): OldBatchUpdateDatabasesRequest => {
  const json = toJson(BatchUpdateDatabasesRequestSchema, newRequest);
  return OldBatchUpdateDatabasesRequestProto.fromJSON(json);
};

// Convert old BatchUpdateDatabasesResponse proto to proto-es
export const convertOldBatchUpdateDatabasesResponseToNew = (oldResponse: OldBatchUpdateDatabasesResponse): NewBatchUpdateDatabasesResponse => {
  const json = OldBatchUpdateDatabasesResponseProto.toJSON(oldResponse) as any;
  return fromJson(BatchUpdateDatabasesResponseSchema, json);
};

// Convert proto-es BatchUpdateDatabasesResponse to old proto
export const convertNewBatchUpdateDatabasesResponseToOld = (newResponse: NewBatchUpdateDatabasesResponse): OldBatchUpdateDatabasesResponse => {
  const json = toJson(BatchUpdateDatabasesResponseSchema, newResponse);
  return OldBatchUpdateDatabasesResponseProto.fromJSON(json);
};

// Convert old BatchSyncDatabasesRequest proto to proto-es
export const convertOldBatchSyncDatabasesRequestToNew = (oldRequest: OldBatchSyncDatabasesRequest): NewBatchSyncDatabasesRequest => {
  const json = OldBatchSyncDatabasesRequestProto.toJSON(oldRequest) as any;
  return fromJson(BatchSyncDatabasesRequestSchema, json);
};

// Convert proto-es BatchSyncDatabasesRequest to old proto
export const convertNewBatchSyncDatabasesRequestToOld = (newRequest: NewBatchSyncDatabasesRequest): OldBatchSyncDatabasesRequest => {
  const json = toJson(BatchSyncDatabasesRequestSchema, newRequest);
  return OldBatchSyncDatabasesRequestProto.fromJSON(json);
};

// Convert old BatchSyncDatabasesResponse proto to proto-es
export const convertOldBatchSyncDatabasesResponseToNew = (oldResponse: OldBatchSyncDatabasesResponse): NewBatchSyncDatabasesResponse => {
  const json = OldBatchSyncDatabasesResponseProto.toJSON(oldResponse) as any;
  return fromJson(BatchSyncDatabasesResponseSchema, json);
};

// Convert proto-es BatchSyncDatabasesResponse to old proto
export const convertNewBatchSyncDatabasesResponseToOld = (newResponse: NewBatchSyncDatabasesResponse): OldBatchSyncDatabasesResponse => {
  const json = toJson(BatchSyncDatabasesResponseSchema, newResponse);
  return OldBatchSyncDatabasesResponseProto.fromJSON(json);
};

// Convert old SyncDatabaseRequest proto to proto-es
export const convertOldSyncDatabaseRequestToNew = (oldRequest: OldSyncDatabaseRequest): NewSyncDatabaseRequest => {
  const json = OldSyncDatabaseRequestProto.toJSON(oldRequest) as any;
  return fromJson(SyncDatabaseRequestSchema, json);
};

// Convert proto-es SyncDatabaseRequest to old proto
export const convertNewSyncDatabaseRequestToOld = (newRequest: NewSyncDatabaseRequest): OldSyncDatabaseRequest => {
  const json = toJson(SyncDatabaseRequestSchema, newRequest);
  return OldSyncDatabaseRequestProto.fromJSON(json);
};

// Convert old SyncDatabaseResponse proto to proto-es
export const convertOldSyncDatabaseResponseToNew = (oldResponse: OldSyncDatabaseResponse): NewSyncDatabaseResponse => {
  const json = OldSyncDatabaseResponseProto.toJSON(oldResponse) as any;
  return fromJson(SyncDatabaseResponseSchema, json);
};

// Convert proto-es SyncDatabaseResponse to old proto
export const convertNewSyncDatabaseResponseToOld = (newResponse: NewSyncDatabaseResponse): OldSyncDatabaseResponse => {
  const json = toJson(SyncDatabaseResponseSchema, newResponse);
  return OldSyncDatabaseResponseProto.fromJSON(json);
};

// Convert old GetDatabaseMetadataRequest proto to proto-es
export const convertOldGetDatabaseMetadataRequestToNew = (oldRequest: OldGetDatabaseMetadataRequest): NewGetDatabaseMetadataRequest => {
  const json = OldGetDatabaseMetadataRequestProto.toJSON(oldRequest) as any;
  return fromJson(GetDatabaseMetadataRequestSchema, json);
};

// Convert proto-es GetDatabaseMetadataRequest to old proto
export const convertNewGetDatabaseMetadataRequestToOld = (newRequest: NewGetDatabaseMetadataRequest): OldGetDatabaseMetadataRequest => {
  const json = toJson(GetDatabaseMetadataRequestSchema, newRequest);
  return OldGetDatabaseMetadataRequestProto.fromJSON(json);
};

// Convert old GetDatabaseSchemaRequest proto to proto-es
export const convertOldGetDatabaseSchemaRequestToNew = (oldRequest: OldGetDatabaseSchemaRequest): NewGetDatabaseSchemaRequest => {
  const json = OldGetDatabaseSchemaRequestProto.toJSON(oldRequest) as any;
  return fromJson(GetDatabaseSchemaRequestSchema, json);
};

// Convert proto-es GetDatabaseSchemaRequest to old proto
export const convertNewGetDatabaseSchemaRequestToOld = (newRequest: NewGetDatabaseSchemaRequest): OldGetDatabaseSchemaRequest => {
  const json = toJson(GetDatabaseSchemaRequestSchema, newRequest);
  return OldGetDatabaseSchemaRequestProto.fromJSON(json);
};

// Convert old DiffSchemaRequest proto to proto-es
export const convertOldDiffSchemaRequestToNew = (oldRequest: OldDiffSchemaRequest): NewDiffSchemaRequest => {
  const json = OldDiffSchemaRequestProto.toJSON(oldRequest) as any;
  return fromJson(DiffSchemaRequestSchema, json);
};

// Convert proto-es DiffSchemaRequest to old proto
export const convertNewDiffSchemaRequestToOld = (newRequest: NewDiffSchemaRequest): OldDiffSchemaRequest => {
  const json = toJson(DiffSchemaRequestSchema, newRequest);
  return OldDiffSchemaRequestProto.fromJSON(json);
};

// Convert old ListSecretsRequest proto to proto-es
export const convertOldListSecretsRequestToNew = (oldRequest: OldListSecretsRequest): NewListSecretsRequest => {
  const json = OldListSecretsRequestProto.toJSON(oldRequest) as any;
  return fromJson(ListSecretsRequestSchema, json);
};

// Convert proto-es ListSecretsRequest to old proto
export const convertNewListSecretsRequestToOld = (newRequest: NewListSecretsRequest): OldListSecretsRequest => {
  const json = toJson(ListSecretsRequestSchema, newRequest);
  return OldListSecretsRequestProto.fromJSON(json);
};

// Convert old ListSecretsResponse proto to proto-es
export const convertOldListSecretsResponseToNew = (oldResponse: OldListSecretsResponse): NewListSecretsResponse => {
  const json = OldListSecretsResponseProto.toJSON(oldResponse) as any;
  return fromJson(ListSecretsResponseSchema, json);
};

// Convert proto-es ListSecretsResponse to old proto
export const convertNewListSecretsResponseToOld = (newResponse: NewListSecretsResponse): OldListSecretsResponse => {
  const json = toJson(ListSecretsResponseSchema, newResponse);
  return OldListSecretsResponseProto.fromJSON(json);
};

// Convert old UpdateSecretRequest proto to proto-es
export const convertOldUpdateSecretRequestToNew = (oldRequest: OldUpdateSecretRequest): NewUpdateSecretRequest => {
  const json = OldUpdateSecretRequestProto.toJSON(oldRequest) as any;
  return fromJson(UpdateSecretRequestSchema, json);
};

// Convert proto-es UpdateSecretRequest to old proto
export const convertNewUpdateSecretRequestToOld = (newRequest: NewUpdateSecretRequest): OldUpdateSecretRequest => {
  const json = toJson(UpdateSecretRequestSchema, newRequest);
  return OldUpdateSecretRequestProto.fromJSON(json);
};

// Convert old DeleteSecretRequest proto to proto-es
export const convertOldDeleteSecretRequestToNew = (oldRequest: OldDeleteSecretRequest): NewDeleteSecretRequest => {
  const json = OldDeleteSecretRequestProto.toJSON(oldRequest) as any;
  return fromJson(DeleteSecretRequestSchema, json);
};

// Convert proto-es DeleteSecretRequest to old proto
export const convertNewDeleteSecretRequestToOld = (newRequest: NewDeleteSecretRequest): OldDeleteSecretRequest => {
  const json = toJson(DeleteSecretRequestSchema, newRequest);
  return OldDeleteSecretRequestProto.fromJSON(json);
};

// Convert old ListChangelogsRequest proto to proto-es
export const convertOldListChangelogsRequestToNew = (oldRequest: OldListChangelogsRequest): NewListChangelogsRequest => {
  const json = OldListChangelogsRequestProto.toJSON(oldRequest) as any;
  return fromJson(ListChangelogsRequestSchema, json);
};

// Convert proto-es ListChangelogsRequest to old proto
export const convertNewListChangelogsRequestToOld = (newRequest: NewListChangelogsRequest): OldListChangelogsRequest => {
  const json = toJson(ListChangelogsRequestSchema, newRequest);
  return OldListChangelogsRequestProto.fromJSON(json);
};

// Convert old ListChangelogsResponse proto to proto-es
export const convertOldListChangelogsResponseToNew = (oldResponse: OldListChangelogsResponse): NewListChangelogsResponse => {
  const json = OldListChangelogsResponseProto.toJSON(oldResponse) as any;
  return fromJson(ListChangelogsResponseSchema, json);
};

// Convert proto-es ListChangelogsResponse to old proto
export const convertNewListChangelogsResponseToOld = (newResponse: NewListChangelogsResponse): OldListChangelogsResponse => {
  const json = toJson(ListChangelogsResponseSchema, newResponse);
  return OldListChangelogsResponseProto.fromJSON(json);
};

// Convert old GetChangelogRequest proto to proto-es
export const convertOldGetChangelogRequestToNew = (oldRequest: OldGetChangelogRequest): NewGetChangelogRequest => {
  const json = OldGetChangelogRequestProto.toJSON(oldRequest) as any;
  return fromJson(GetChangelogRequestSchema, json);
};

// Convert proto-es GetChangelogRequest to old proto
export const convertNewGetChangelogRequestToOld = (newRequest: NewGetChangelogRequest): OldGetChangelogRequest => {
  const json = toJson(GetChangelogRequestSchema, newRequest);
  return OldGetChangelogRequestProto.fromJSON(json);
};

// Convert old GetSchemaStringRequest proto to proto-es
export const convertOldGetSchemaStringRequestToNew = (oldRequest: OldGetSchemaStringRequest): NewGetSchemaStringRequest => {
  const json = OldGetSchemaStringRequestProto.toJSON(oldRequest) as any;
  return fromJson(GetSchemaStringRequestSchema, json);
};

// Convert proto-es GetSchemaStringRequest to old proto
export const convertNewGetSchemaStringRequestToOld = (newRequest: NewGetSchemaStringRequest): OldGetSchemaStringRequest => {
  const json = toJson(GetSchemaStringRequestSchema, newRequest);
  return OldGetSchemaStringRequestProto.fromJSON(json);
};

