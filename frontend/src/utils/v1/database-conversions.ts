import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Database as OldDatabase } from "@/types/proto/v1/database_service";
import { Database as OldDatabaseProto } from "@/types/proto/v1/database_service";
import type { Database as NewDatabase } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSchema$ as NewDatabaseSchema } from "@/types/proto-es/v1/database_service_pb";

import type { DatabaseMetadata as OldDatabaseMetadata } from "@/types/proto/v1/database_service";
import { DatabaseMetadata as OldDatabaseMetadataProto } from "@/types/proto/v1/database_service";
import type { DatabaseMetadata as NewDatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseMetadataSchema as NewDatabaseMetadataSchema } from "@/types/proto-es/v1/database_service_pb";

import type { Changelog as OldChangelog } from "@/types/proto/v1/database_service";
import { Changelog as OldChangelogProto } from "@/types/proto/v1/database_service";
import type { Changelog as NewChangelog } from "@/types/proto-es/v1/database_service_pb";
import { ChangelogSchema as NewChangelogSchema } from "@/types/proto-es/v1/database_service_pb";

import type { Secret as OldSecret } from "@/types/proto/v1/database_service";
import { Secret as OldSecretProto } from "@/types/proto/v1/database_service";
import type { Secret as NewSecret } from "@/types/proto-es/v1/database_service_pb";
import { SecretSchema as NewSecretSchema } from "@/types/proto-es/v1/database_service_pb";

import type { DatabaseSchema as OldDatabaseSchema } from "@/types/proto/v1/database_service";
import { DatabaseSchema as OldDatabaseSchemaProto } from "@/types/proto/v1/database_service";
import type { DatabaseSchema as NewDatabaseSchemaType } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSchemaSchema as NewDatabaseSchemaSchema } from "@/types/proto-es/v1/database_service_pb";

import type { DiffSchemaResponse as OldDiffSchemaResponse } from "@/types/proto/v1/database_service";
import { DiffSchemaResponse as OldDiffSchemaResponseProto } from "@/types/proto/v1/database_service";
import type { DiffSchemaResponse as NewDiffSchemaResponse } from "@/types/proto-es/v1/database_service_pb";
import { DiffSchemaResponseSchema as NewDiffSchemaResponseSchema } from "@/types/proto-es/v1/database_service_pb";

import type { GetSchemaStringResponse as OldGetSchemaStringResponse } from "@/types/proto/v1/database_service";
import { GetSchemaStringResponse as OldGetSchemaStringResponseProto } from "@/types/proto/v1/database_service";
import type { GetSchemaStringResponse as NewGetSchemaStringResponse } from "@/types/proto-es/v1/database_service_pb";
import { GetSchemaStringResponseSchema as NewGetSchemaStringResponseSchema } from "@/types/proto-es/v1/database_service_pb";

import { ChangelogView as OldChangelogView } from "@/types/proto/v1/database_service";
import { ChangelogView as NewChangelogView } from "@/types/proto-es/v1/database_service_pb";

import { GetSchemaStringRequest_ObjectType as OldObjectType } from "@/types/proto/v1/database_service";
import { GetSchemaStringRequest_ObjectType as NewObjectType } from "@/types/proto-es/v1/database_service_pb";

import type { SchemaMetadata as OldSchemaMetadata } from "@/types/proto/v1/database_service";
import { SchemaMetadata as OldSchemaMetadataProto } from "@/types/proto/v1/database_service";
import type { SchemaMetadata as NewSchemaMetadata } from "@/types/proto-es/v1/database_service_pb";
import { SchemaMetadataSchema as NewSchemaMetadataSchema } from "@/types/proto-es/v1/database_service_pb";

import type { TableMetadata as OldTableMetadata } from "@/types/proto/v1/database_service";
import { TableMetadata as OldTableMetadataProto } from "@/types/proto/v1/database_service";
import type { TableMetadata as NewTableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { TableMetadataSchema as NewTableMetadataSchema } from "@/types/proto-es/v1/database_service_pb";

import type { ColumnMetadata as OldColumnMetadata } from "@/types/proto/v1/database_service";
import { ColumnMetadata as OldColumnMetadataProto } from "@/types/proto/v1/database_service";
import type { ColumnMetadata as NewColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { ColumnMetadataSchema as NewColumnMetadataSchema } from "@/types/proto-es/v1/database_service_pb";



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
  return fromJson(NewDatabaseSchema, json);
};

// Convert proto-es Database to old proto
export const convertNewDatabaseToOld = (newDatabase: NewDatabase): OldDatabase => {
  const json = toJson(NewDatabaseSchema, newDatabase);
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

