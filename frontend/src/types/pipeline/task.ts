import { ErrorCode, MigrationHistoryId, TaskCheckRunId } from "..";
import { Database } from "../database";
import {
  BackupId,
  DatabaseId,
  InstanceId,
  ProjectId,
  TaskId,
  TaskRunId,
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
  | "bb.task.database.data.update"
  | "bb.task.database.restore"
  | "bb.task.database.schema.update.ghost"
  | "bb.task.database.schema.update.ghost.cutover"
  | "bb.task.database.schema.update.ghost.drop-original-table";

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

export type TaskEarliestAllowedTimePayload = {
  earliestAllowedTs: number;
};

export type TaskDatabaseCreatePayload = {
  projectId: ProjectId;
  statement: string;
  databaseName: string;
  characterSet: string;
  collation: string;
};

export type TaskDatabaseSchemaUpdatePayload = {
  migrationType: MigrationType;
  statement: string;
  pushEvent?: VCSPushEvent;
};

export type TaskDatabaseSchemaUpdateGhostPayload =
  TaskDatabaseSchemaUpdatePayload & {
    // empty by now
    // more input and output parameters in the future
  };

export type TaskDatabaseSchemaUpdateGhostCutoverPayload = {
  parentTaskId: TaskId;
};

export type TaskDatabaseSchemaUpdateGhostDropOriginalTablePayload = {
  databaseName: string;
  tableName: string;
};

export type TaskDatabaseDataUpdatePayload = {
  statement: string;
  pushEvent?: VCSPushEvent;
};

export type TaskDatabaseRestorePayload = {
  databaseName: string;
  backupId: BackupId;
};

export type TaskPayload =
  | TaskGeneralPayload
  | TaskDatabaseCreatePayload
  | TaskDatabaseSchemaUpdatePayload
  | TaskDatabaseSchemaUpdateGhostPayload
  | TaskDatabaseSchemaUpdateGhostCutoverPayload
  | TaskDatabaseDataUpdatePayload
  | TaskDatabaseRestorePayload
  | TaskEarliestAllowedTimePayload;

export type Task = {
  id: TaskId;

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
  earliestAllowedTs: number;
  // Tasks like creating database may not have database.
  database?: Database;
  payload?: TaskPayload;
};

export type TaskCreate = {
  // Domain specific fields
  name: string;
  status: TaskStatus;
  type: TaskType;
  instanceId: InstanceId;
  databaseId?: DatabaseId;
  statement: string;
  databaseName?: string;
  characterSet?: string;
  collation?: string;
  backupId?: BackupId;
  migrationType?: MigrationType;
  earliestAllowedTs: number;
};

export type TaskPatch = {
  statement?: string;
  earliestAllowedTs?: number;
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
  migrationId?: MigrationHistoryId;
  version?: string;
};

export type TaskRun = {
  id: TaskRunId;

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
  | "bb.task-check.instance.migration-schema"
  | "bb.task-check.general.earliest-allowed-time"
  | "bb.task-check.database.schema-update.ghost";

export type TaskCheckDatabaseStatementAdvisePayload = {
  statement: string;
};

export type TaskCheckDatabaseSchemaUpdateGhostPayload = {
  statement: string;
  instanceId: InstanceId;
  databaseName: string;
  tableName: string;
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
  id: TaskCheckRunId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  taskId: TaskId;
  status: TaskCheckRunStatus;
  type: TaskCheckType;
  code: ErrorCode;
  comment: string;
  result: TaskCheckRunResultPayload;
  payload?: TaskPayload;
};
