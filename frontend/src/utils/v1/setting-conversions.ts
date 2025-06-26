import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Setting as OldSetting } from "@/types/proto/v1/setting_service";
import { Setting as OldSettingProto, Setting_SettingName as OldSettingName } from "@/types/proto/v1/setting_service";
import type { Setting as NewSetting } from "@/types/proto-es/v1/setting_service_pb";
import { SettingSchema, Setting_SettingName as NewSettingName } from "@/types/proto-es/v1/setting_service_pb";
import { Engine as NewEngine } from "@/types/proto-es/v1/common_pb";
import { Engine as OldEngine } from "@/types/proto/v1/common";

// Convert old proto to proto-es
export const convertOldSettingToNew = (oldSetting: OldSetting): NewSetting => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldSettingProto.toJSON(oldSetting) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(SettingSchema, json);
};

// Convert proto-es to old proto
export const convertNewSettingToOld = (newSetting: NewSetting): OldSetting => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(SettingSchema, newSetting);
  return OldSettingProto.fromJSON(json);
};

// Convert old enum to new string format
export const convertOldSettingNameToNew = (oldName: OldSettingName): string => {
  // Map string enum to numeric enum
  const mapping: Record<OldSettingName, NewSettingName> = {
    [OldSettingName.SETTING_NAME_UNSPECIFIED]: NewSettingName.SETTING_NAME_UNSPECIFIED,
    [OldSettingName.AUTH_SECRET]: NewSettingName.AUTH_SECRET,
    [OldSettingName.BRANDING_LOGO]: NewSettingName.BRANDING_LOGO,
    [OldSettingName.WORKSPACE_ID]: NewSettingName.WORKSPACE_ID,
    [OldSettingName.WORKSPACE_PROFILE]: NewSettingName.WORKSPACE_PROFILE,
    [OldSettingName.WORKSPACE_APPROVAL]: NewSettingName.WORKSPACE_APPROVAL,
    [OldSettingName.WORKSPACE_EXTERNAL_APPROVAL]: NewSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [OldSettingName.ENTERPRISE_LICENSE]: NewSettingName.ENTERPRISE_LICENSE,
    [OldSettingName.APP_IM]: NewSettingName.APP_IM,
    [OldSettingName.WATERMARK]: NewSettingName.WATERMARK,
    [OldSettingName.AI]: NewSettingName.AI,
    [OldSettingName.SCHEMA_TEMPLATE]: NewSettingName.SCHEMA_TEMPLATE,
    [OldSettingName.DATA_CLASSIFICATION]: NewSettingName.DATA_CLASSIFICATION,
    [OldSettingName.SEMANTIC_TYPES]: NewSettingName.SEMANTIC_TYPES,
    [OldSettingName.SQL_RESULT_SIZE_LIMIT]: NewSettingName.SQL_RESULT_SIZE_LIMIT,
    [OldSettingName.SCIM]: NewSettingName.SCIM,
    [OldSettingName.PASSWORD_RESTRICTION]: NewSettingName.PASSWORD_RESTRICTION,
    [OldSettingName.ENVIRONMENT]: NewSettingName.ENVIRONMENT,
    [OldSettingName.UNRECOGNIZED]: NewSettingName.SETTING_NAME_UNSPECIFIED,
  };
  const newEnumValue = mapping[oldName] ?? NewSettingName.SETTING_NAME_UNSPECIFIED;
  return NewSettingName[newEnumValue];
};

// Convert new string format to old enum
export const convertNewSettingNameToOld = (newNameString: string): OldSettingName => {
  // Find the numeric enum value from the string
  const newEnumValue = Object.entries(NewSettingName).find(([key]) => key === newNameString)?.[1] as NewSettingName | undefined;
  if (newEnumValue === undefined) {
    return OldSettingName.UNRECOGNIZED;
  }
  return convertNewSettingNameEnumToOld(newEnumValue);
};

// Convert new enum to old enum (internal helper)
const convertNewSettingNameEnumToOld = (newName: NewSettingName): OldSettingName => {
  // Map numeric enum to string enum
  const mapping: Record<NewSettingName, OldSettingName> = {
    [NewSettingName.SETTING_NAME_UNSPECIFIED]: OldSettingName.SETTING_NAME_UNSPECIFIED,
    [NewSettingName.AUTH_SECRET]: OldSettingName.AUTH_SECRET,
    [NewSettingName.BRANDING_LOGO]: OldSettingName.BRANDING_LOGO,
    [NewSettingName.WORKSPACE_ID]: OldSettingName.WORKSPACE_ID,
    [NewSettingName.WORKSPACE_PROFILE]: OldSettingName.WORKSPACE_PROFILE,
    [NewSettingName.WORKSPACE_APPROVAL]: OldSettingName.WORKSPACE_APPROVAL,
    [NewSettingName.WORKSPACE_EXTERNAL_APPROVAL]: OldSettingName.WORKSPACE_EXTERNAL_APPROVAL,
    [NewSettingName.ENTERPRISE_LICENSE]: OldSettingName.ENTERPRISE_LICENSE,
    [NewSettingName.APP_IM]: OldSettingName.APP_IM,
    [NewSettingName.WATERMARK]: OldSettingName.WATERMARK,
    [NewSettingName.AI]: OldSettingName.AI,
    [NewSettingName.SCHEMA_TEMPLATE]: OldSettingName.SCHEMA_TEMPLATE,
    [NewSettingName.DATA_CLASSIFICATION]: OldSettingName.DATA_CLASSIFICATION,
    [NewSettingName.SEMANTIC_TYPES]: OldSettingName.SEMANTIC_TYPES,
    [NewSettingName.SQL_RESULT_SIZE_LIMIT]: OldSettingName.SQL_RESULT_SIZE_LIMIT,
    [NewSettingName.SCIM]: OldSettingName.SCIM,
    [NewSettingName.PASSWORD_RESTRICTION]: OldSettingName.PASSWORD_RESTRICTION,
    [NewSettingName.ENVIRONMENT]: OldSettingName.ENVIRONMENT,
  };
  return mapping[newName] ?? OldSettingName.UNRECOGNIZED;
};

// Convert proto-es Engine to old Engine for utility functions
export const convertEngineToOld = (engine: NewEngine): OldEngine => {
  switch (engine) {
    case NewEngine.MYSQL:
      return OldEngine.MYSQL;
    case NewEngine.POSTGRES:
      return OldEngine.POSTGRES;
    case NewEngine.TIDB:
      return OldEngine.TIDB;
    case NewEngine.SNOWFLAKE:
      return OldEngine.SNOWFLAKE;
    case NewEngine.CLICKHOUSE:
      return OldEngine.CLICKHOUSE;
    case NewEngine.MONGODB:
      return OldEngine.MONGODB;
    case NewEngine.REDIS:
      return OldEngine.REDIS;
    case NewEngine.ORACLE:
      return OldEngine.ORACLE;
    case NewEngine.SPANNER:
      return OldEngine.SPANNER;
    case NewEngine.MSSQL:
      return OldEngine.MSSQL;
    case NewEngine.REDSHIFT:
      return OldEngine.REDSHIFT;
    case NewEngine.MARIADB:
      return OldEngine.MARIADB;
    case NewEngine.OCEANBASE:
      return OldEngine.OCEANBASE;
    case NewEngine.DM:
      return OldEngine.DM;
    case NewEngine.RISINGWAVE:
      return OldEngine.RISINGWAVE;
    case NewEngine.OCEANBASE_ORACLE:
      return OldEngine.OCEANBASE_ORACLE;
    case NewEngine.STARROCKS:
      return OldEngine.STARROCKS;
    case NewEngine.DORIS:
      return OldEngine.DORIS;
    case NewEngine.HIVE:
      return OldEngine.HIVE;
    case NewEngine.ELASTICSEARCH:
      return OldEngine.ELASTICSEARCH;
    case NewEngine.BIGQUERY:
      return OldEngine.BIGQUERY;
    case NewEngine.DYNAMODB:
      return OldEngine.DYNAMODB;
    case NewEngine.COSMOSDB:
      return OldEngine.COSMOSDB;
    case NewEngine.CASSANDRA:
      return OldEngine.CASSANDRA;
    case NewEngine.DATABRICKS:
      return OldEngine.DATABRICKS;
    case NewEngine.TRINO:
      return OldEngine.TRINO;
    default:
      return OldEngine.ENGINE_UNSPECIFIED;
  }
};

// Convert old Engine to proto-es Engine (for completeness)
export const convertEngineToNew = (engine: OldEngine): NewEngine => {
  switch (engine) {
    case OldEngine.MYSQL:
      return NewEngine.MYSQL;
    case OldEngine.POSTGRES:
      return NewEngine.POSTGRES;
    case OldEngine.TIDB:
      return NewEngine.TIDB;
    case OldEngine.SNOWFLAKE:
      return NewEngine.SNOWFLAKE;
    case OldEngine.CLICKHOUSE:
      return NewEngine.CLICKHOUSE;
    case OldEngine.MONGODB:
      return NewEngine.MONGODB;
    case OldEngine.REDIS:
      return NewEngine.REDIS;
    case OldEngine.ORACLE:
      return NewEngine.ORACLE;
    case OldEngine.SPANNER:
      return NewEngine.SPANNER;
    case OldEngine.MSSQL:
      return NewEngine.MSSQL;
    case OldEngine.REDSHIFT:
      return NewEngine.REDSHIFT;
    case OldEngine.MARIADB:
      return NewEngine.MARIADB;
    case OldEngine.OCEANBASE:
      return NewEngine.OCEANBASE;
    case OldEngine.DM:
      return NewEngine.DM;
    case OldEngine.RISINGWAVE:
      return NewEngine.RISINGWAVE;
    case OldEngine.OCEANBASE_ORACLE:
      return NewEngine.OCEANBASE_ORACLE;
    case OldEngine.STARROCKS:
      return NewEngine.STARROCKS;
    case OldEngine.DORIS:
      return NewEngine.DORIS;
    case OldEngine.HIVE:
      return NewEngine.HIVE;
    case OldEngine.ELASTICSEARCH:
      return NewEngine.ELASTICSEARCH;
    case OldEngine.BIGQUERY:
      return NewEngine.BIGQUERY;
    case OldEngine.DYNAMODB:
      return NewEngine.DYNAMODB;
    case OldEngine.COSMOSDB:
      return NewEngine.COSMOSDB;
    case OldEngine.CASSANDRA:
      return NewEngine.CASSANDRA;
    case OldEngine.DATABRICKS:
      return NewEngine.DATABRICKS;
    case OldEngine.TRINO:
      return NewEngine.TRINO;
    default:
      return NewEngine.ENGINE_UNSPECIFIED;
  }
};