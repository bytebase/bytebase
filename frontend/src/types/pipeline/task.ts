import { ErrorCode, MigrationHistoryID, TaskCheckRunID } from "..";
import { Database } from "../database";
import {
  BackupID,
  DatabaseID,
  InstanceID,
  ProjectID,
  TaskID,
  TaskRunID,
} from "../id";
import { Instance, MigrationType } from "../instance";
import { Principal } from "../principal";
import { VCSPushEvent } from "../vcs";
import { Pipeline } from "./pipeline";
import { Stage } from "./stage";

export type TaskType =
  | "bb.task.general"
  | "bb.task.database.create"
  | "bb.task.database.schema.update"
  | "bb.task.database.restore";

export type TaskStatus =
  | "PENDING"
  | "PENDING_APPROVAL"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED";

export type TaskGeneralPayload = {
  statement: string;
};

export type TaskDatabaseCreatePayload = {
  projectID: ProjectID;
  statement: string;
  databaseName: string;
  characterSet: string;
  collation: string;
};

export type TaskDatabaseSchemaUpdatePayload = {
  migrationType: MigrationType;
  statement: string;
  rollbackStatement: string;
  pushEvent?: VCSPushEvent;
};

export type TaskDatabaseRestorePayload = {
  databaseName: string;
  backupID: BackupID;
};

export type TaskPayload =
  | TaskGeneralPayload
  | TaskDatabaseCreatePayload
  | TaskDatabaseSchemaUpdatePayload
  | TaskDatabaseRestorePayload;

export type Task = {
  id: TaskID;

  // Related fields
  taskRunList: TaskRun[];
  taskCheckRunList: TaskCheckRun[];
  pipeline: Pipeline;
  stage: Stage;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: TaskStatus;
  type: TaskType;
  instance: Instance;
  // Tasks like creating database may not have database.
  database?: Database;
  payload?: TaskPayload;
};

export type TaskCreate = {
  // Domain specific fields
  name: string;
  status: TaskStatus;
  type: TaskType;
  instanceID: InstanceID;
  databaseID?: DatabaseID;
  statement: string;
  rollbackStatement: string;
  databaseName?: string;
  characterSet?: string;
  collation?: string;
  backupID?: BackupID;
  migrationType?: MigrationType;
};

export type TaskPatch = {
  statement?: string;
};

export type TaskStatusPatch = {
  // Domain specific fields
  status: TaskStatus;
  comment?: string;
};

// TaskRun is one run of a particular task
export type TaskRunStatus = "RUNNING" | "DONE" | "FAILED" | "CANCELED";

export type TaskRunResultPayload = {
  detail: string;
  migrationID?: MigrationHistoryID;
  version?: string;
};

export type TaskRun = {
  id: TaskRunID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: TaskRunStatus;
  type: TaskType;
  code: ErrorCode;
  comment: string;
  result: TaskRunResultPayload;
  payload?: TaskPayload;
};

export type TaskCheckRunStatus = "RUNNING" | "DONE" | "FAILED" | "CANCELED";

export type TaskCheckType =
  | "bb.task-check.database.statement.fake-advise"
  | "bb.task-check.database.statement.syntax"
  | "bb.task-check.database.statement.compatibility"
  | "bb.task-check.database.connect"
  | "bb.task-check.instance.migration-schema";

export type TaskCheckDatabaseStatementAdvisePayload = {
  statement: string;
};

export type TaskCheckStatus = "SUCCESS" | "WARN" | "ERROR";

export type TaskCheckResult = {
  status: TaskCheckStatus;
  code: ErrorCode;
  title: string;
  content: string;
};

export type TaskCheckRunResultPayload = {
  detail: string;
  resultList: TaskCheckResult[];
};

export type TaskCheckRun = {
  id: TaskCheckRunID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  taskID: TaskID;
  status: TaskCheckRunStatus;
  type: TaskCheckType;
  code: ErrorCode;
  comment: string;
  result: TaskCheckRunResultPayload;
  payload?: TaskPayload;
};
