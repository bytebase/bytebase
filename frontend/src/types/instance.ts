import type { DataSource } from ".";
import type { RowStatus } from "./common";
import type { Environment } from "./environment";
import type { InstanceId, MigrationHistoryId } from "./id";
import { Engine } from "./proto/v1/common";

export const EngineTypeList = [
  "CLICKHOUSE",
  "MYSQL",
  "POSTGRES",
  "SNOWFLAKE",
  "TIDB",
  "MONGODB",
  "SPANNER",
  "REDIS",
  "ORACLE",
  "MSSQL",
  "REDSHIFT",
  "MARIADB",
  "OCEANBASE",
  "DM",
  "RISINGWAVE",
  "BIGQUERY",
  "DYNAMODB",
  "DATABRICKS",
] as const;

export type EngineType = (typeof EngineTypeList)[number];

export function convertEngineType(type: EngineType): Engine {
  switch (type) {
    case "CLICKHOUSE":
      return Engine.CLICKHOUSE;
    case "MYSQL":
      return Engine.MYSQL;
    case "POSTGRES":
      return Engine.POSTGRES;
    case "SNOWFLAKE":
      return Engine.SNOWFLAKE;
    case "TIDB":
      return Engine.TIDB;
    case "MONGODB":
      return Engine.MONGODB;
    case "SPANNER":
      return Engine.SPANNER;
    case "REDIS":
      return Engine.REDIS;
    case "ORACLE":
      return Engine.ORACLE;
    case "MSSQL":
      return Engine.MSSQL;
    case "REDSHIFT":
      return Engine.REDSHIFT;
    case "MARIADB":
      return Engine.MARIADB;
    case "OCEANBASE":
      return Engine.OCEANBASE;
    case "DM":
      return Engine.DM;
    case "RISINGWAVE":
      return Engine.RISINGWAVE;
    case "BIGQUERY":
      return Engine.BIGQUERY;
    case "DYNAMODB":
      return Engine.DYNAMODB;
    case "DATABRICKS":
      return Engine.DATABRICKS;
  }
  return Engine.ENGINE_UNSPECIFIED;
}

export function isPostgresFamily(type: Engine): boolean {
  return (
    type == Engine.POSTGRES ||
    type == Engine.REDSHIFT ||
    type == Engine.RISINGWAVE
  );
}

export function defaultCharset(type: EngineType): string {
  switch (type) {
    case "CLICKHOUSE":
    case "SNOWFLAKE":
      return "";
    case "MYSQL":
    case "TIDB":
    case "MARIADB":
    case "OCEANBASE":
      return "utf8mb4";
    case "POSTGRES":
      return "UTF8";
    case "MONGODB":
      return "";
    case "SPANNER":
      return "";
    case "REDIS":
      return "";
    case "ORACLE":
      return "UTF8";
    case "DM":
      return "UTF8";
    case "MSSQL":
      return "";
    case "REDSHIFT":
      return "UNICODE";
    case "RISINGWAVE":
      return "UTF8";
    case "BIGQUERY":
      return "";
    case "DYNAMODB":
      return "";
    case "DATABRICKS":
      return "";
  }
  return "";
}

export function engineName(type: EngineType): string {
  switch (type) {
    case "CLICKHOUSE":
      return "ClickHouse";
    case "MYSQL":
      return "MySQL";
    case "POSTGRES":
      return "PostgreSQL";
    case "SNOWFLAKE":
      return "Snowflake";
    case "TIDB":
      return "TiDB";
    case "MONGODB":
      return "MongoDB";
    case "SPANNER":
      return "Spanner";
    case "REDIS":
      return "Redis";
    case "ORACLE":
      return "Oracle";
    case "MSSQL":
      return "MSSQL";
    case "REDSHIFT":
      return "Redshift";
    case "MARIADB":
      return "MariaDB";
    case "OCEANBASE":
      return "OceanBase";
    case "DM":
      return "DM";
    case "RISINGWAVE":
      return "RisingWave";
    case "BIGQUERY":
      return "BigQuery";
    case "DYNAMODB":
      return "DynamoDB";
    case "DATABRICKS":
      return "Databricks";
  }
  return "";
}

export function defaultCollation(type: EngineType): string {
  switch (type) {
    case "CLICKHOUSE":
    case "SNOWFLAKE":
      return "";
    case "MYSQL":
    case "TIDB":
    case "MARIADB":
    case "OCEANBASE":
      return "utf8mb4_general_ci";
    // For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
    // If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
    // install it.
    case "POSTGRES":
    case "RISINGWAVE":
      return "";
    case "MONGODB":
      return "";
    case "SPANNER":
      return "";
    case "REDIS":
      return "";
    case "ORACLE":
      return "BINARY_CI";
    case "MSSQL":
      return "";
    case "REDSHIFT":
      return "";
    case "DM":
      return "BINARY_CI";
    case "BIGQUERY":
      return "";
    case "DYNAMODB":
      return "";
    case "DATABRICKS":
      return "";
  }
}

export type Instance = {
  id: InstanceId;
  resourceId: string;
  rowStatus: RowStatus;

  // Related fields
  environment: Environment;
  // An instance must have a admin data source, maybe a read-only data source.
  dataSourceList: DataSource[];

  // Domain specific fields
  name: string;
  engine: EngineType;
  engineVersion: string;
  externalLink?: string;
  srv: boolean;
  authenticationDatabase: string;
};

export type MigrationSchemaStatus = "UNKNOWN" | "OK" | "NOT_EXIST";

export type InstanceMigration = {
  status: MigrationSchemaStatus;
  error: string;
};

export type MigrationSource = "UI" | "VCS" | "LIBRARY";

export type MigrationType = "BASELINE" | "MIGRATE" | "BRANCH" | "DATA";

export type MigrationStatus = "PENDING" | "DONE" | "FAILED";

export type MigrationHistory = {
  id: MigrationHistoryId;
  creator: string;
  createdTs: number;
  updater: string;
  updatedTs: number;
  releaseVersion: string;
  database: string;
  source: MigrationSource;
  type: MigrationType;
  status: MigrationStatus;
  version: string;
  description: string;
  statement: string;
  schema: string;
  schemaPrev: string;
  executionDurationNs: number;
  issueId: number;
};
