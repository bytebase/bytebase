import { RowStatus } from "./common";
import { Environment } from "./environment";
import { EnvironmentId, InstanceId, MigrationHistoryId } from "./id";
import { Principal } from "./principal";
import { VCSPushEvent } from "./vcs";

export type EngineType = "MYSQL" | "POSTGRES" | "TIDB";

export type Instance = {
  id: InstanceId;

  // Related fields
  environment: Environment;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  engine: EngineType;
  externalLink?: string;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
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
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type InstancePatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  externalLink?: string;
  host?: string;
  port?: string;
  username?: string;
  password?: string;
};

export type MigrationSchemaStatus = "UNKNOWN" | "OK" | "NOT_EXIST";

export type InstanceMigration = {
  status: MigrationSchemaStatus;
  error: string;
};

export type MigrationEngine = "UI" | "VCS";

export type MigrationType = "BASELINE" | "MIGRATE" | "BRANCH";

export type MigrationStatus = "PENDING" | "DONE";

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
  engine: MigrationEngine;
  type: MigrationType;
  status: MigrationStatus;
  version: string;
  description: string;
  statement: string;
  schema: string;
  schemaPrev: string;
  executionDuration: number;
  issueId: number;
  payload?: MigrationHistoryPayload;
};
