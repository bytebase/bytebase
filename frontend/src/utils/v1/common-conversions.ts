/**
 * Common type conversions between old proto and proto-es for common types.
 *
 * This file provides conversion functions for types in common.proto that are shared
 * across multiple services during the migration from old proto to proto-es.
 */
import {
  Engine as NewEngine,
  State as NewState,
  ExportFormat as NewExportFormat,
  VCSType as NewVCSType,
} from "@/types/proto-es/v1/common_pb";

/**
 * Convert proto-es Engine to string for display/logging
 */
export const engineToString = (engine: NewEngine): string => {
  return NewEngine[engine] || "ENGINE_UNSPECIFIED";
};

/**
 * Convert proto-es State to string for display/logging
 */
export const stateToString = (state: NewState): string => {
  return NewState[state] || "STATE_UNSPECIFIED";
};

/**
 * Convert proto-es ExportFormat to string for display/logging
 */
export const exportFormatToString = (format: NewExportFormat): string => {
  return NewExportFormat[format] || "FORMAT_UNSPECIFIED";
};

/**
 * Convert proto-es VCSType to string for display/logging
 */
export const vcsTypeToString = (vcsType: NewVCSType): string => {
  return NewVCSType[vcsType] || "VCS_TYPE_UNSPECIFIED";
};

/**
 * Convert scope value (string or number) to proto-es Engine
 * Handles both string names and numeric enum values from scope.value
 */
export const convertScopeValueToEngine = (
  value: string | number
): NewEngine => {
  switch (value) {
    case 0:
    case "ENGINE_UNSPECIFIED":
      return NewEngine.ENGINE_UNSPECIFIED;
    case 1:
    case "CLICKHOUSE":
      return NewEngine.CLICKHOUSE;
    case 2:
    case "MYSQL":
      return NewEngine.MYSQL;
    case 3:
    case "POSTGRES":
      return NewEngine.POSTGRES;
    case 4:
    case "SNOWFLAKE":
      return NewEngine.SNOWFLAKE;
    case 5:
    case "SQLITE":
      return NewEngine.SQLITE;
    case 6:
    case "TIDB":
      return NewEngine.TIDB;
    case 7:
    case "MONGODB":
      return NewEngine.MONGODB;
    case 8:
    case "REDIS":
      return NewEngine.REDIS;
    case 9:
    case "ORACLE":
      return NewEngine.ORACLE;
    case 10:
    case "SPANNER":
      return NewEngine.SPANNER;
    case 11:
    case "MSSQL":
      return NewEngine.MSSQL;
    case 12:
    case "REDSHIFT":
      return NewEngine.REDSHIFT;
    case 13:
    case "MARIADB":
      return NewEngine.MARIADB;
    case 14:
    case "OCEANBASE":
      return NewEngine.OCEANBASE;
    case 18:
    case "STARROCKS":
      return NewEngine.STARROCKS;
    case 19:
    case "DORIS":
      return NewEngine.DORIS;
    case 20:
    case "HIVE":
      return NewEngine.HIVE;
    case 21:
    case "ELASTICSEARCH":
      return NewEngine.ELASTICSEARCH;
    case 22:
    case "BIGQUERY":
      return NewEngine.BIGQUERY;
    case 23:
    case "DYNAMODB":
      return NewEngine.DYNAMODB;
    case 24:
    case "DATABRICKS":
      return NewEngine.DATABRICKS;
    case 25:
    case "COCKROACHDB":
      return NewEngine.COCKROACHDB;
    case 26:
    case "COSMOSDB":
      return NewEngine.COSMOSDB;
    case 27:
    case "TRINO":
      return NewEngine.TRINO;
    case 28:
    case "CASSANDRA":
      return NewEngine.CASSANDRA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return NewEngine.ENGINE_UNSPECIFIED;
  }
};
