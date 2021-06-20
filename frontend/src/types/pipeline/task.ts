import { Database } from "../database";
import { DatabaseId, InstanceId, PrincipalId, TaskId, TaskRunId } from "../id";
import { Instance } from "../instance";
import { Principal } from "../principal";
import { Pipeline } from "./pipeline";
import { Stage } from "./stage";

export type TaskType = "bb.task.general" | "bb.task.database.schema.update";

export type TaskStatus =
  | "PENDING"
  | "PENDING_APPROVAL"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED";

export type DatabaseSchemaUpdateTaskPayload = {
  statement: string;
  rollbackStatement: string;
};

export type TaskPayload = DatabaseSchemaUpdateTaskPayload;

export type Task = {
  id: TaskId;

  // Related fields
  taskRunList: TaskRun[];
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
  // We may get an empty database for tasks like creating database.
  database: Database;
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
  rollbackStatement: string;
};

export type TaskStatusPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: TaskStatus;
  comment?: string;
};

// TaskRun is one run of a particular task
export type TaskRunStatus = "RUNNING" | "DONE" | "FAILED" | "CANCELED";

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
  error: string;
  payload?: TaskPayload;
};
