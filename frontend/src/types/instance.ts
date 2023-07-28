import { DataSource } from ".";
import { RowStatus } from "./common";
import { Environment } from "./environment";
import { InstanceId, MigrationHistoryId, ResourceId } from "./id";
import { Engine } from "./proto/v1/common";
import { VCSPushEvent } from "./vcs";

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
] as const;

export type EngineType = typeof EngineTypeList[number];

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
  }
  return Engine.ENGINE_UNSPECIFIED;
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
  }
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
  }
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

export type InstanceCreate = {
  resourceId: ResourceId;

  // Related fields
  environmentId: number;

  // Domain specific fields
  name: string;
  engine: EngineType;
  externalLink?: string;
  host: string;
  port?: string;
  database?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  // DNS SRV record is only used for MongoDB.
  srv: boolean;
  // For MongoDB, the auth database is used to authenticate the user.
  authenticationDatabase: string;
  // sid and serviceName are used for Oracle database. Required one of them.
  sid: string;
  serviceName: string;
  // Connection over SSH.
  sshHost: string;
  sshPort: string;
  sshUser: string;
  sshPassword: string;
  sshPrivateKey: string;
};

export type InstancePatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  externalLink?: string;
};

export type MigrationSchemaStatus = "UNKNOWN" | "OK" | "NOT_EXIST";

export type InstanceMigration = {
  status: MigrationSchemaStatus;
  error: string;
};

export type MigrationSource = "UI" | "VCS" | "LIBRARY";

export type MigrationType = "BASELINE" | "MIGRATE" | "BRANCH" | "DATA";

export type MigrationStatus = "PENDING" | "DONE" | "FAILED";

export type MigrationHistoryPayload = {
  pushEvent?: VCSPushEvent;
};

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
  payload?: MigrationHistoryPayload;
};
