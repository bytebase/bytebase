/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum State {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  ACTIVE = "ACTIVE",
  DELETED = "DELETED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function stateFromJSON(object: any): State {
  switch (object) {
    case 0:
    case "STATE_UNSPECIFIED":
      return State.STATE_UNSPECIFIED;
    case 1:
    case "ACTIVE":
      return State.ACTIVE;
    case 2:
    case "DELETED":
      return State.DELETED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return State.UNRECOGNIZED;
  }
}

export function stateToJSON(object: State): string {
  switch (object) {
    case State.STATE_UNSPECIFIED:
      return "STATE_UNSPECIFIED";
    case State.ACTIVE:
      return "ACTIVE";
    case State.DELETED:
      return "DELETED";
    case State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function stateToNumber(object: State): number {
  switch (object) {
    case State.STATE_UNSPECIFIED:
      return 0;
    case State.ACTIVE:
      return 1;
    case State.DELETED:
      return 2;
    case State.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum Engine {
  ENGINE_UNSPECIFIED = "ENGINE_UNSPECIFIED",
  CLICKHOUSE = "CLICKHOUSE",
  MYSQL = "MYSQL",
  POSTGRES = "POSTGRES",
  SNOWFLAKE = "SNOWFLAKE",
  SQLITE = "SQLITE",
  TIDB = "TIDB",
  MONGODB = "MONGODB",
  REDIS = "REDIS",
  ORACLE = "ORACLE",
  SPANNER = "SPANNER",
  MSSQL = "MSSQL",
  REDSHIFT = "REDSHIFT",
  MARIADB = "MARIADB",
  OCEANBASE = "OCEANBASE",
  DM = "DM",
  RISINGWAVE = "RISINGWAVE",
  OCEANBASE_ORACLE = "OCEANBASE_ORACLE",
  STARROCKS = "STARROCKS",
  DORIS = "DORIS",
  HIVE = "HIVE",
  ELASTICSEARCH = "ELASTICSEARCH",
  BIGQUERY = "BIGQUERY",
  DYNAMODB = "DYNAMODB",
  DATABRICKS = "DATABRICKS",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function engineFromJSON(object: any): Engine {
  switch (object) {
    case 0:
    case "ENGINE_UNSPECIFIED":
      return Engine.ENGINE_UNSPECIFIED;
    case 1:
    case "CLICKHOUSE":
      return Engine.CLICKHOUSE;
    case 2:
    case "MYSQL":
      return Engine.MYSQL;
    case 3:
    case "POSTGRES":
      return Engine.POSTGRES;
    case 4:
    case "SNOWFLAKE":
      return Engine.SNOWFLAKE;
    case 5:
    case "SQLITE":
      return Engine.SQLITE;
    case 6:
    case "TIDB":
      return Engine.TIDB;
    case 7:
    case "MONGODB":
      return Engine.MONGODB;
    case 8:
    case "REDIS":
      return Engine.REDIS;
    case 9:
    case "ORACLE":
      return Engine.ORACLE;
    case 10:
    case "SPANNER":
      return Engine.SPANNER;
    case 11:
    case "MSSQL":
      return Engine.MSSQL;
    case 12:
    case "REDSHIFT":
      return Engine.REDSHIFT;
    case 13:
    case "MARIADB":
      return Engine.MARIADB;
    case 14:
    case "OCEANBASE":
      return Engine.OCEANBASE;
    case 15:
    case "DM":
      return Engine.DM;
    case 16:
    case "RISINGWAVE":
      return Engine.RISINGWAVE;
    case 17:
    case "OCEANBASE_ORACLE":
      return Engine.OCEANBASE_ORACLE;
    case 18:
    case "STARROCKS":
      return Engine.STARROCKS;
    case 19:
    case "DORIS":
      return Engine.DORIS;
    case 20:
    case "HIVE":
      return Engine.HIVE;
    case 21:
    case "ELASTICSEARCH":
      return Engine.ELASTICSEARCH;
    case 22:
    case "BIGQUERY":
      return Engine.BIGQUERY;
    case 23:
    case "DYNAMODB":
      return Engine.DYNAMODB;
    case 24:
    case "DATABRICKS":
      return Engine.DATABRICKS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Engine.UNRECOGNIZED;
  }
}

export function engineToJSON(object: Engine): string {
  switch (object) {
    case Engine.ENGINE_UNSPECIFIED:
      return "ENGINE_UNSPECIFIED";
    case Engine.CLICKHOUSE:
      return "CLICKHOUSE";
    case Engine.MYSQL:
      return "MYSQL";
    case Engine.POSTGRES:
      return "POSTGRES";
    case Engine.SNOWFLAKE:
      return "SNOWFLAKE";
    case Engine.SQLITE:
      return "SQLITE";
    case Engine.TIDB:
      return "TIDB";
    case Engine.MONGODB:
      return "MONGODB";
    case Engine.REDIS:
      return "REDIS";
    case Engine.ORACLE:
      return "ORACLE";
    case Engine.SPANNER:
      return "SPANNER";
    case Engine.MSSQL:
      return "MSSQL";
    case Engine.REDSHIFT:
      return "REDSHIFT";
    case Engine.MARIADB:
      return "MARIADB";
    case Engine.OCEANBASE:
      return "OCEANBASE";
    case Engine.DM:
      return "DM";
    case Engine.RISINGWAVE:
      return "RISINGWAVE";
    case Engine.OCEANBASE_ORACLE:
      return "OCEANBASE_ORACLE";
    case Engine.STARROCKS:
      return "STARROCKS";
    case Engine.DORIS:
      return "DORIS";
    case Engine.HIVE:
      return "HIVE";
    case Engine.ELASTICSEARCH:
      return "ELASTICSEARCH";
    case Engine.BIGQUERY:
      return "BIGQUERY";
    case Engine.DYNAMODB:
      return "DYNAMODB";
    case Engine.DATABRICKS:
      return "DATABRICKS";
    case Engine.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function engineToNumber(object: Engine): number {
  switch (object) {
    case Engine.ENGINE_UNSPECIFIED:
      return 0;
    case Engine.CLICKHOUSE:
      return 1;
    case Engine.MYSQL:
      return 2;
    case Engine.POSTGRES:
      return 3;
    case Engine.SNOWFLAKE:
      return 4;
    case Engine.SQLITE:
      return 5;
    case Engine.TIDB:
      return 6;
    case Engine.MONGODB:
      return 7;
    case Engine.REDIS:
      return 8;
    case Engine.ORACLE:
      return 9;
    case Engine.SPANNER:
      return 10;
    case Engine.MSSQL:
      return 11;
    case Engine.REDSHIFT:
      return 12;
    case Engine.MARIADB:
      return 13;
    case Engine.OCEANBASE:
      return 14;
    case Engine.DM:
      return 15;
    case Engine.RISINGWAVE:
      return 16;
    case Engine.OCEANBASE_ORACLE:
      return 17;
    case Engine.STARROCKS:
      return 18;
    case Engine.DORIS:
      return 19;
    case Engine.HIVE:
      return 20;
    case Engine.ELASTICSEARCH:
      return 21;
    case Engine.BIGQUERY:
      return 22;
    case Engine.DYNAMODB:
      return 23;
    case Engine.DATABRICKS:
      return 24;
    case Engine.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum VCSType {
  VCS_TYPE_UNSPECIFIED = "VCS_TYPE_UNSPECIFIED",
  /** GITHUB - GitHub type. Using for GitHub community edition(ce). */
  GITHUB = "GITHUB",
  /** GITLAB - GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). */
  GITLAB = "GITLAB",
  /** BITBUCKET - BitBucket type. Using for BitBucket cloud or BitBucket server. */
  BITBUCKET = "BITBUCKET",
  /** AZURE_DEVOPS - Azure DevOps. Using for Azure DevOps GitOps workflow. */
  AZURE_DEVOPS = "AZURE_DEVOPS",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function vCSTypeFromJSON(object: any): VCSType {
  switch (object) {
    case 0:
    case "VCS_TYPE_UNSPECIFIED":
      return VCSType.VCS_TYPE_UNSPECIFIED;
    case 1:
    case "GITHUB":
      return VCSType.GITHUB;
    case 2:
    case "GITLAB":
      return VCSType.GITLAB;
    case 3:
    case "BITBUCKET":
      return VCSType.BITBUCKET;
    case 4:
    case "AZURE_DEVOPS":
      return VCSType.AZURE_DEVOPS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VCSType.UNRECOGNIZED;
  }
}

export function vCSTypeToJSON(object: VCSType): string {
  switch (object) {
    case VCSType.VCS_TYPE_UNSPECIFIED:
      return "VCS_TYPE_UNSPECIFIED";
    case VCSType.GITHUB:
      return "GITHUB";
    case VCSType.GITLAB:
      return "GITLAB";
    case VCSType.BITBUCKET:
      return "BITBUCKET";
    case VCSType.AZURE_DEVOPS:
      return "AZURE_DEVOPS";
    case VCSType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function vCSTypeToNumber(object: VCSType): number {
  switch (object) {
    case VCSType.VCS_TYPE_UNSPECIFIED:
      return 0;
    case VCSType.GITHUB:
      return 1;
    case VCSType.GITLAB:
      return 2;
    case VCSType.BITBUCKET:
      return 3;
    case VCSType.AZURE_DEVOPS:
      return 4;
    case VCSType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum MaskingLevel {
  MASKING_LEVEL_UNSPECIFIED = "MASKING_LEVEL_UNSPECIFIED",
  NONE = "NONE",
  PARTIAL = "PARTIAL",
  FULL = "FULL",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function maskingLevelFromJSON(object: any): MaskingLevel {
  switch (object) {
    case 0:
    case "MASKING_LEVEL_UNSPECIFIED":
      return MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
    case 1:
    case "NONE":
      return MaskingLevel.NONE;
    case 2:
    case "PARTIAL":
      return MaskingLevel.PARTIAL;
    case 3:
    case "FULL":
      return MaskingLevel.FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return MaskingLevel.UNRECOGNIZED;
  }
}

export function maskingLevelToJSON(object: MaskingLevel): string {
  switch (object) {
    case MaskingLevel.MASKING_LEVEL_UNSPECIFIED:
      return "MASKING_LEVEL_UNSPECIFIED";
    case MaskingLevel.NONE:
      return "NONE";
    case MaskingLevel.PARTIAL:
      return "PARTIAL";
    case MaskingLevel.FULL:
      return "FULL";
    case MaskingLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function maskingLevelToNumber(object: MaskingLevel): number {
  switch (object) {
    case MaskingLevel.MASKING_LEVEL_UNSPECIFIED:
      return 0;
    case MaskingLevel.NONE:
      return 1;
    case MaskingLevel.PARTIAL:
      return 2;
    case MaskingLevel.FULL:
      return 3;
    case MaskingLevel.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum ExportFormat {
  FORMAT_UNSPECIFIED = "FORMAT_UNSPECIFIED",
  CSV = "CSV",
  JSON = "JSON",
  SQL = "SQL",
  XLSX = "XLSX",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function exportFormatFromJSON(object: any): ExportFormat {
  switch (object) {
    case 0:
    case "FORMAT_UNSPECIFIED":
      return ExportFormat.FORMAT_UNSPECIFIED;
    case 1:
    case "CSV":
      return ExportFormat.CSV;
    case 2:
    case "JSON":
      return ExportFormat.JSON;
    case 3:
    case "SQL":
      return ExportFormat.SQL;
    case 4:
    case "XLSX":
      return ExportFormat.XLSX;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ExportFormat.UNRECOGNIZED;
  }
}

export function exportFormatToJSON(object: ExportFormat): string {
  switch (object) {
    case ExportFormat.FORMAT_UNSPECIFIED:
      return "FORMAT_UNSPECIFIED";
    case ExportFormat.CSV:
      return "CSV";
    case ExportFormat.JSON:
      return "JSON";
    case ExportFormat.SQL:
      return "SQL";
    case ExportFormat.XLSX:
      return "XLSX";
    case ExportFormat.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function exportFormatToNumber(object: ExportFormat): number {
  switch (object) {
    case ExportFormat.FORMAT_UNSPECIFIED:
      return 0;
    case ExportFormat.CSV:
      return 1;
    case ExportFormat.JSON:
      return 2;
    case ExportFormat.SQL:
      return 3;
    case ExportFormat.XLSX:
      return 4;
    case ExportFormat.UNRECOGNIZED:
    default:
      return -1;
  }
}
