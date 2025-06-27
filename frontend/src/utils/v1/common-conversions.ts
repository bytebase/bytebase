/**
 * Common type conversions between old proto and proto-es for common types.
 * 
 * This file provides conversion functions for types in common.proto that are shared
 * across multiple services during the migration from old proto to proto-es.
 */

// Old proto imports
import { 
  Engine as OldEngine,
  State as OldState,
  ExportFormat as OldExportFormat,
  VCSType as OldVCSType,
  engineToJSON as oldEngineToJSON,
  engineFromJSON as oldEngineFromJSON,
} from "@/types/proto/v1/common";

import {
  Duration as OldDuration,
} from "@/types/proto/google/protobuf/duration"

import {
  DurationSchema as NewDurationSchema,
  type Duration as NewDuration,
} from "@bufbuild/protobuf/wkt"


// Proto-es imports
import { 
  Engine as NewEngine,
  State as NewState, 
  ExportFormat as NewExportFormat,
  VCSType as NewVCSType,
} from "@/types/proto-es/v1/common_pb";
import { create } from "@bufbuild/protobuf";

/**
 * Convert proto-es Engine to old proto Engine
 */
export const convertEngineToOld = (engine: NewEngine): OldEngine => {
  switch (engine) {
    case NewEngine.ENGINE_UNSPECIFIED:
      return OldEngine.ENGINE_UNSPECIFIED;
    case NewEngine.CLICKHOUSE:
      return OldEngine.CLICKHOUSE;
    case NewEngine.MYSQL:
      return OldEngine.MYSQL;
    case NewEngine.POSTGRES:
      return OldEngine.POSTGRES;
    case NewEngine.SNOWFLAKE:
      return OldEngine.SNOWFLAKE;
    case NewEngine.SQLITE:
      return OldEngine.SQLITE;
    case NewEngine.TIDB:
      return OldEngine.TIDB;
    case NewEngine.MONGODB:
      return OldEngine.MONGODB;
    case NewEngine.REDIS:
      return OldEngine.REDIS;
    case NewEngine.ORACLE:
      return OldEngine.ORACLE;
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
    case NewEngine.SPANNER:
      return OldEngine.SPANNER;
    case NewEngine.COCKROACHDB:
      return OldEngine.COCKROACHDB;
    case NewEngine.DATABRICKS:
      return OldEngine.DATABRICKS;
    case NewEngine.COSMOSDB:
      return OldEngine.COSMOSDB;
    case NewEngine.TRINO:
      return OldEngine.TRINO;
    case NewEngine.CASSANDRA:
      return OldEngine.CASSANDRA;
    default:
      return OldEngine.ENGINE_UNSPECIFIED;
  }
};

/**
 * Convert old proto Engine to proto-es Engine
 */
export const convertEngineToNew = (engine: OldEngine): NewEngine => {
  switch (engine) {
    case OldEngine.ENGINE_UNSPECIFIED:
      return NewEngine.ENGINE_UNSPECIFIED;
    case OldEngine.CLICKHOUSE:
      return NewEngine.CLICKHOUSE;
    case OldEngine.MYSQL:
      return NewEngine.MYSQL;
    case OldEngine.POSTGRES:
      return NewEngine.POSTGRES;
    case OldEngine.SNOWFLAKE:
      return NewEngine.SNOWFLAKE;
    case OldEngine.SQLITE:
      return NewEngine.SQLITE;
    case OldEngine.TIDB:
      return NewEngine.TIDB;
    case OldEngine.MONGODB:
      return NewEngine.MONGODB;
    case OldEngine.REDIS:
      return NewEngine.REDIS;
    case OldEngine.ORACLE:
      return NewEngine.ORACLE;
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
    case OldEngine.SPANNER:
      return NewEngine.SPANNER;
    case OldEngine.COCKROACHDB:
      return NewEngine.COCKROACHDB;
    case OldEngine.DATABRICKS:
      return NewEngine.DATABRICKS;
    case OldEngine.COSMOSDB:
      return NewEngine.COSMOSDB;
    case OldEngine.TRINO:
      return NewEngine.TRINO;
    case OldEngine.CASSANDRA:
      return NewEngine.CASSANDRA;
    default:
      return NewEngine.ENGINE_UNSPECIFIED;
  }
};

/**
 * Convert proto-es State to old proto State
 */
export const convertStateToOld = (state: NewState): OldState => {
  switch (state) {
    case NewState.STATE_UNSPECIFIED:
      return OldState.STATE_UNSPECIFIED;
    case NewState.ACTIVE:
      return OldState.ACTIVE;
    case NewState.DELETED:
      return OldState.DELETED;
    default:
      return OldState.STATE_UNSPECIFIED;
  }
};

/**
 * Convert old proto State to proto-es State
 */
export const convertStateToNew = (state: OldState): NewState => {
  switch (state) {
    case OldState.STATE_UNSPECIFIED:
      return NewState.STATE_UNSPECIFIED;
    case OldState.ACTIVE:
      return NewState.ACTIVE;
    case OldState.DELETED:
      return NewState.DELETED;
    default:
      return NewState.STATE_UNSPECIFIED;
  }
};

/**
 * Convert proto-es ExportFormat to old proto ExportFormat
 */
export const convertExportFormatToOld = (format: NewExportFormat): OldExportFormat => {
  switch (format) {
    case NewExportFormat.FORMAT_UNSPECIFIED:
      return OldExportFormat.FORMAT_UNSPECIFIED;
    case NewExportFormat.CSV:
      return OldExportFormat.CSV;
    case NewExportFormat.JSON:
      return OldExportFormat.JSON;
    case NewExportFormat.SQL:
      return OldExportFormat.SQL;
    case NewExportFormat.XLSX:
      return OldExportFormat.XLSX;
    default:
      return OldExportFormat.FORMAT_UNSPECIFIED;
  }
};

/**
 * Convert old proto ExportFormat to proto-es ExportFormat
 */
export const convertExportFormatToNew = (format: OldExportFormat): NewExportFormat => {
  switch (format) {
    case OldExportFormat.FORMAT_UNSPECIFIED:
      return NewExportFormat.FORMAT_UNSPECIFIED;
    case OldExportFormat.CSV:
      return NewExportFormat.CSV;
    case OldExportFormat.JSON:
      return NewExportFormat.JSON;
    case OldExportFormat.SQL:
      return NewExportFormat.SQL;
    case OldExportFormat.XLSX:
      return NewExportFormat.XLSX;
    default:
      return NewExportFormat.FORMAT_UNSPECIFIED;
  }
};

/**
 * Convert proto-es VCSType to old proto VCSType
 */
export const convertVCSTypeToOld = (vcsType: NewVCSType): OldVCSType => {
  switch (vcsType) {
    case NewVCSType.VCS_TYPE_UNSPECIFIED:
      return OldVCSType.VCS_TYPE_UNSPECIFIED;
    case NewVCSType.GITHUB:
      return OldVCSType.GITHUB;
    case NewVCSType.GITLAB:
      return OldVCSType.GITLAB;
    case NewVCSType.BITBUCKET:
      return OldVCSType.BITBUCKET;
    case NewVCSType.AZURE_DEVOPS:
      return OldVCSType.AZURE_DEVOPS;
    default:
      return OldVCSType.VCS_TYPE_UNSPECIFIED;
  }
};

/**
 * Convert old proto VCSType to proto-es VCSType
 */
export const convertVCSTypeToNew = (vcsType: OldVCSType): NewVCSType => {
  switch (vcsType) {
    case OldVCSType.VCS_TYPE_UNSPECIFIED:
      return NewVCSType.VCS_TYPE_UNSPECIFIED;
    case OldVCSType.GITHUB:
      return NewVCSType.GITHUB;
    case OldVCSType.GITLAB:
      return NewVCSType.GITLAB;
    case OldVCSType.BITBUCKET:
      return NewVCSType.BITBUCKET;
    case OldVCSType.AZURE_DEVOPS:
      return NewVCSType.AZURE_DEVOPS;
    default:
      return NewVCSType.VCS_TYPE_UNSPECIFIED;
  }
};

/**
 * Convert old proto Engine to JSON string (backwards compatibility)
 */
export const engineToJSON = (engine: OldEngine): string => {
  return oldEngineToJSON(engine);
};

/**
 * Convert JSON string to old proto Engine (backwards compatibility)
 */
export const engineFromJSON = (json: string): OldEngine => {
  return oldEngineFromJSON(json);
};

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
}

/**
 * Convert scope value (string or number) to proto-es Engine
 * Handles both string names and numeric enum values from scope.value
 */
export const convertScopeValueToEngine = (value: string | number): NewEngine => {
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
    case 15:
    case "DM":
      return NewEngine.DM;
    case 16:
    case "RISINGWAVE":
      return NewEngine.RISINGWAVE;
    case 17:
    case "OCEANBASE_ORACLE":
      return NewEngine.OCEANBASE_ORACLE;
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

export const convertDurationToOld = (value: NewDuration): OldDuration => {
  return OldDuration.fromPartial({
    seconds: Number(value.seconds),
    nanos: value.nanos,
  })
}

export const convertDurationToNew = (value: OldDuration): NewDuration => {
  return create(NewDurationSchema, {
    seconds: value.seconds.toBigInt(),
    nanos: value.nanos,
  })
}