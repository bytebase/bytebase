import { DataSource } from ".";
import { RowStatus } from "./common";
import { Environment } from "./environment";
import { EnvironmentId, InstanceId, MigrationHistoryId } from "./id";
import { Principal } from "./principal";
import { VCSPushEvent } from "./vcs";

export type EngineType =
  | "CLICKHOUSE"
  | "MYSQL"
  | "POSTGRES"
  | "SNOWFLAKE"
  | "TIDB"
  | "MONGODB";

export function defaultCharset(type: EngineType): string {
  switch (type) {
    case "CLICKHOUSE":
    case "SNOWFLAKE":
      return "";
    case "MYSQL":
    case "TIDB":
      return "utf8mb4";
    case "POSTGRES":
      return "UTF8";
    case "MONGODB":
      return "";
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
  }
}

export function defaultCollation(type: EngineType): string {
  switch (type) {
    case "CLICKHOUSE":
    case "SNOWFLAKE":
      return "";
    case "MYSQL":
    case "TIDB":
      return "utf8mb4_general_ci";
    // For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
    // If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
    // install it.
    case "POSTGRES":
      return "";
    case "MONGODB":
      return "";
  }
}

export type Instance = {
  id: InstanceId;

  // Related fields
  environment: Environment;
  // An instance must have a admin data source, maybe a read-only data source.
  dataSourceList: DataSource[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  engine: EngineType;
  engineVersion: string;
  externalLink?: string;
  srv: boolean;
  authSource: string;
};

export type InstanceCreate = {
  // Related fields
  environmentId: EnvironmentId;

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
  authSource: string;
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
