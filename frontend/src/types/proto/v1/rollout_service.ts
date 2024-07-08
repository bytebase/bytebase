/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";
import { ExportFormat, exportFormatFromJSON, exportFormatToJSON, exportFormatToNumber } from "./common";
import { Plan } from "./plan_service";

export const protobufPackage = "bytebase.v1";

export interface BatchRunTasksRequest {
  /**
   * The name of the parent of the tasks.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}
   */
  parent: string;
  /**
   * The tasks to run.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  tasks: string[];
  reason: string;
}

export interface BatchRunTasksResponse {
}

export interface BatchSkipTasksRequest {
  /**
   * The name of the parent of the tasks.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}
   */
  parent: string;
  /**
   * The tasks to skip.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  tasks: string[];
  reason: string;
}

export interface BatchSkipTasksResponse {
}

export interface BatchCancelTaskRunsRequest {
  /**
   * The name of the parent of the taskRuns.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   * Use `projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/-` to cancel task runs under the same stage.
   */
  parent: string;
  /**
   * The taskRuns to cancel.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
   */
  taskRuns: string[];
  reason: string;
}

export interface BatchCancelTaskRunsResponse {
}

export interface GetRolloutRequest {
  /**
   * The name of the rollout to retrieve.
   * Format: projects/{project}/rollouts/{rollout}
   */
  name: string;
}

export interface CreateRolloutRequest {
  /**
   * The parent project where this rollout will be created.
   * Format: projects/{project}
   */
  parent: string;
  /** The rollout to create. */
  rollout: Rollout | undefined;
}

export interface PreviewRolloutRequest {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  project: string;
  /** The plan used to preview rollout. */
  plan: Plan | undefined;
}

export interface ListTaskRunsRequest {
  /**
   * The parent, which owns this collection of plans.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   * Use "projects/{project}/rollouts/{rollout}/stages/-/tasks/-" to list all taskRuns from a rollout.
   */
  parent: string;
  /**
   * The maximum number of taskRuns to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 taskRuns will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListRolloutTaskRuns` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListRolloutTaskRuns` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListTaskRunsResponse {
  /** The taskRuns from the specified request. */
  taskRuns: TaskRun[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface GetTaskRunLogRequest {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} */
  parent: string;
}

export interface Rollout {
  /**
   * The resource name of the rollout.
   * Format: projects/{project}/rollouts/{rollout}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  /**
   * The plan that this rollout is based on.
   * Format: projects/{project}/plans/{plan}
   */
  plan: string;
  title: string;
  /** stages and thus tasks of the rollout. */
  stages: Stage[];
}

export interface Stage {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage} */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  tasks: Task[];
}

export interface Task {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  /**
   * A UUID4 string that uniquely identifies the Spec.
   * Could be empty if the rollout of the task does not have an associating plan.
   */
  specId: string;
  /** Status is the status of the task. */
  status: Task_Status;
  skippedReason: string;
  type: Task_Type;
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} */
  dependsOnTasks: string[];
  /**
   * Format: instances/{instance} if the task is DatabaseCreate.
   * Format: instances/{instance}/databases/{database}
   */
  target: string;
  databaseCreate?: Task_DatabaseCreate | undefined;
  databaseSchemaBaseline?: Task_DatabaseSchemaBaseline | undefined;
  databaseSchemaUpdate?: Task_DatabaseSchemaUpdate | undefined;
  databaseDataUpdate?: Task_DatabaseDataUpdate | undefined;
  databaseDataExport?: Task_DatabaseDataExport | undefined;
}

export enum Task_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  NOT_STARTED = "NOT_STARTED",
  PENDING = "PENDING",
  RUNNING = "RUNNING",
  DONE = "DONE",
  FAILED = "FAILED",
  CANCELED = "CANCELED",
  SKIPPED = "SKIPPED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function task_StatusFromJSON(object: any): Task_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return Task_Status.STATUS_UNSPECIFIED;
    case 1:
    case "NOT_STARTED":
      return Task_Status.NOT_STARTED;
    case 2:
    case "PENDING":
      return Task_Status.PENDING;
    case 3:
    case "RUNNING":
      return Task_Status.RUNNING;
    case 4:
    case "DONE":
      return Task_Status.DONE;
    case 5:
    case "FAILED":
      return Task_Status.FAILED;
    case 6:
    case "CANCELED":
      return Task_Status.CANCELED;
    case 7:
    case "SKIPPED":
      return Task_Status.SKIPPED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Task_Status.UNRECOGNIZED;
  }
}

export function task_StatusToJSON(object: Task_Status): string {
  switch (object) {
    case Task_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case Task_Status.NOT_STARTED:
      return "NOT_STARTED";
    case Task_Status.PENDING:
      return "PENDING";
    case Task_Status.RUNNING:
      return "RUNNING";
    case Task_Status.DONE:
      return "DONE";
    case Task_Status.FAILED:
      return "FAILED";
    case Task_Status.CANCELED:
      return "CANCELED";
    case Task_Status.SKIPPED:
      return "SKIPPED";
    case Task_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function task_StatusToNumber(object: Task_Status): number {
  switch (object) {
    case Task_Status.STATUS_UNSPECIFIED:
      return 0;
    case Task_Status.NOT_STARTED:
      return 1;
    case Task_Status.PENDING:
      return 2;
    case Task_Status.RUNNING:
      return 3;
    case Task_Status.DONE:
      return 4;
    case Task_Status.FAILED:
      return 5;
    case Task_Status.CANCELED:
      return 6;
    case Task_Status.SKIPPED:
      return 7;
    case Task_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum Task_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  GENERAL = "GENERAL",
  /** DATABASE_CREATE - use payload DatabaseCreate */
  DATABASE_CREATE = "DATABASE_CREATE",
  /** DATABASE_SCHEMA_BASELINE - use payload DatabaseSchemaBaseline */
  DATABASE_SCHEMA_BASELINE = "DATABASE_SCHEMA_BASELINE",
  /** DATABASE_SCHEMA_UPDATE - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE = "DATABASE_SCHEMA_UPDATE",
  /** DATABASE_SCHEMA_UPDATE_SDL - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE_SDL = "DATABASE_SCHEMA_UPDATE_SDL",
  /** DATABASE_SCHEMA_UPDATE_GHOST_SYNC - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE_GHOST_SYNC = "DATABASE_SCHEMA_UPDATE_GHOST_SYNC",
  /** DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER - use payload nil */
  DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER = "DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER",
  /** DATABASE_DATA_UPDATE - use payload DatabaseDataUpdate */
  DATABASE_DATA_UPDATE = "DATABASE_DATA_UPDATE",
  /** DATABASE_DATA_EXPORT - use payload DatabaseDataExport */
  DATABASE_DATA_EXPORT = "DATABASE_DATA_EXPORT",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function task_TypeFromJSON(object: any): Task_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Task_Type.TYPE_UNSPECIFIED;
    case 1:
    case "GENERAL":
      return Task_Type.GENERAL;
    case 2:
    case "DATABASE_CREATE":
      return Task_Type.DATABASE_CREATE;
    case 3:
    case "DATABASE_SCHEMA_BASELINE":
      return Task_Type.DATABASE_SCHEMA_BASELINE;
    case 4:
    case "DATABASE_SCHEMA_UPDATE":
      return Task_Type.DATABASE_SCHEMA_UPDATE;
    case 5:
    case "DATABASE_SCHEMA_UPDATE_SDL":
      return Task_Type.DATABASE_SCHEMA_UPDATE_SDL;
    case 6:
    case "DATABASE_SCHEMA_UPDATE_GHOST_SYNC":
      return Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC;
    case 7:
    case "DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER":
      return Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER;
    case 8:
    case "DATABASE_DATA_UPDATE":
      return Task_Type.DATABASE_DATA_UPDATE;
    case 12:
    case "DATABASE_DATA_EXPORT":
      return Task_Type.DATABASE_DATA_EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Task_Type.UNRECOGNIZED;
  }
}

export function task_TypeToJSON(object: Task_Type): string {
  switch (object) {
    case Task_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Task_Type.GENERAL:
      return "GENERAL";
    case Task_Type.DATABASE_CREATE:
      return "DATABASE_CREATE";
    case Task_Type.DATABASE_SCHEMA_BASELINE:
      return "DATABASE_SCHEMA_BASELINE";
    case Task_Type.DATABASE_SCHEMA_UPDATE:
      return "DATABASE_SCHEMA_UPDATE";
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      return "DATABASE_SCHEMA_UPDATE_SDL";
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC:
      return "DATABASE_SCHEMA_UPDATE_GHOST_SYNC";
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
      return "DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER";
    case Task_Type.DATABASE_DATA_UPDATE:
      return "DATABASE_DATA_UPDATE";
    case Task_Type.DATABASE_DATA_EXPORT:
      return "DATABASE_DATA_EXPORT";
    case Task_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function task_TypeToNumber(object: Task_Type): number {
  switch (object) {
    case Task_Type.TYPE_UNSPECIFIED:
      return 0;
    case Task_Type.GENERAL:
      return 1;
    case Task_Type.DATABASE_CREATE:
      return 2;
    case Task_Type.DATABASE_SCHEMA_BASELINE:
      return 3;
    case Task_Type.DATABASE_SCHEMA_UPDATE:
      return 4;
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      return 5;
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC:
      return 6;
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
      return 7;
    case Task_Type.DATABASE_DATA_UPDATE:
      return 8;
    case Task_Type.DATABASE_DATA_EXPORT:
      return 12;
    case Task_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface Task_DatabaseCreate {
  /**
   * The project owning the database.
   * Format: projects/{project}
   */
  project: string;
  /** database name */
  database: string;
  /** table name */
  table: string;
  /** Format: projects/{project}/sheets/{sheet} */
  sheet: string;
  characterSet: string;
  collation: string;
  environment: string;
  labels: { [key: string]: string };
}

export interface Task_DatabaseCreate_LabelsEntry {
  key: string;
  value: string;
}

export interface Task_DatabaseSchemaBaseline {
  schemaVersion: string;
}

export interface Task_DatabaseSchemaUpdate {
  /** Format: projects/{project}/sheets/{sheet} */
  sheet: string;
  schemaVersion: string;
}

export interface Task_DatabaseDataUpdate {
  /** Format: projects/{project}/sheets/{sheet} */
  sheet: string;
  schemaVersion: string;
}

export interface Task_DatabaseDataExport {
  /**
   * The resource name of the target.
   * Format: instances/{instance-id}/databases/{database-name}
   */
  target: string;
  /**
   * The resource name of the sheet.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  /** The format of the exported file. */
  format: ExportFormat;
  /**
   * The zip password provide by users.
   * Leave it empty if no needs to encrypt the zip file.
   */
  password?: string | undefined;
}

export interface TaskRun {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  /** Format: user/hello@world.com */
  creator: string;
  /** Format: user/hello@world.com */
  updater: string;
  createTime: Date | undefined;
  updateTime: Date | undefined;
  title: string;
  status: TaskRun_Status;
  /** Below are the results of a task run. */
  detail: string;
  /**
   * The resource name of the change history
   * Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory}
   */
  changeHistory: string;
  schemaVersion: string;
  executionStatus: TaskRun_ExecutionStatus;
  executionDetail: TaskRun_ExecutionDetail | undefined;
  startTime: Date | undefined;
  exportArchiveStatus: TaskRun_ExportArchiveStatus;
}

export enum TaskRun_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  PENDING = "PENDING",
  RUNNING = "RUNNING",
  DONE = "DONE",
  FAILED = "FAILED",
  CANCELED = "CANCELED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRun_StatusFromJSON(object: any): TaskRun_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return TaskRun_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return TaskRun_Status.PENDING;
    case 2:
    case "RUNNING":
      return TaskRun_Status.RUNNING;
    case 3:
    case "DONE":
      return TaskRun_Status.DONE;
    case 4:
    case "FAILED":
      return TaskRun_Status.FAILED;
    case 5:
    case "CANCELED":
      return TaskRun_Status.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRun_Status.UNRECOGNIZED;
  }
}

export function taskRun_StatusToJSON(object: TaskRun_Status): string {
  switch (object) {
    case TaskRun_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case TaskRun_Status.PENDING:
      return "PENDING";
    case TaskRun_Status.RUNNING:
      return "RUNNING";
    case TaskRun_Status.DONE:
      return "DONE";
    case TaskRun_Status.FAILED:
      return "FAILED";
    case TaskRun_Status.CANCELED:
      return "CANCELED";
    case TaskRun_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRun_StatusToNumber(object: TaskRun_Status): number {
  switch (object) {
    case TaskRun_Status.STATUS_UNSPECIFIED:
      return 0;
    case TaskRun_Status.PENDING:
      return 1;
    case TaskRun_Status.RUNNING:
      return 2;
    case TaskRun_Status.DONE:
      return 3;
    case TaskRun_Status.FAILED:
      return 4;
    case TaskRun_Status.CANCELED:
      return 5;
    case TaskRun_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum TaskRun_ExecutionStatus {
  EXECUTION_STATUS_UNSPECIFIED = "EXECUTION_STATUS_UNSPECIFIED",
  PRE_EXECUTING = "PRE_EXECUTING",
  EXECUTING = "EXECUTING",
  POST_EXECUTING = "POST_EXECUTING",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRun_ExecutionStatusFromJSON(object: any): TaskRun_ExecutionStatus {
  switch (object) {
    case 0:
    case "EXECUTION_STATUS_UNSPECIFIED":
      return TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED;
    case 1:
    case "PRE_EXECUTING":
      return TaskRun_ExecutionStatus.PRE_EXECUTING;
    case 2:
    case "EXECUTING":
      return TaskRun_ExecutionStatus.EXECUTING;
    case 3:
    case "POST_EXECUTING":
      return TaskRun_ExecutionStatus.POST_EXECUTING;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRun_ExecutionStatus.UNRECOGNIZED;
  }
}

export function taskRun_ExecutionStatusToJSON(object: TaskRun_ExecutionStatus): string {
  switch (object) {
    case TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED:
      return "EXECUTION_STATUS_UNSPECIFIED";
    case TaskRun_ExecutionStatus.PRE_EXECUTING:
      return "PRE_EXECUTING";
    case TaskRun_ExecutionStatus.EXECUTING:
      return "EXECUTING";
    case TaskRun_ExecutionStatus.POST_EXECUTING:
      return "POST_EXECUTING";
    case TaskRun_ExecutionStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRun_ExecutionStatusToNumber(object: TaskRun_ExecutionStatus): number {
  switch (object) {
    case TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED:
      return 0;
    case TaskRun_ExecutionStatus.PRE_EXECUTING:
      return 1;
    case TaskRun_ExecutionStatus.EXECUTING:
      return 2;
    case TaskRun_ExecutionStatus.POST_EXECUTING:
      return 3;
    case TaskRun_ExecutionStatus.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum TaskRun_ExportArchiveStatus {
  EXPORT_ARCHIVE_STATUS_UNSPECIFIED = "EXPORT_ARCHIVE_STATUS_UNSPECIFIED",
  READY = "READY",
  EXPORTED = "EXPORTED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRun_ExportArchiveStatusFromJSON(object: any): TaskRun_ExportArchiveStatus {
  switch (object) {
    case 0:
    case "EXPORT_ARCHIVE_STATUS_UNSPECIFIED":
      return TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED;
    case 1:
    case "READY":
      return TaskRun_ExportArchiveStatus.READY;
    case 2:
    case "EXPORTED":
      return TaskRun_ExportArchiveStatus.EXPORTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRun_ExportArchiveStatus.UNRECOGNIZED;
  }
}

export function taskRun_ExportArchiveStatusToJSON(object: TaskRun_ExportArchiveStatus): string {
  switch (object) {
    case TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED:
      return "EXPORT_ARCHIVE_STATUS_UNSPECIFIED";
    case TaskRun_ExportArchiveStatus.READY:
      return "READY";
    case TaskRun_ExportArchiveStatus.EXPORTED:
      return "EXPORTED";
    case TaskRun_ExportArchiveStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRun_ExportArchiveStatusToNumber(object: TaskRun_ExportArchiveStatus): number {
  switch (object) {
    case TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED:
      return 0;
    case TaskRun_ExportArchiveStatus.READY:
      return 1;
    case TaskRun_ExportArchiveStatus.EXPORTED:
      return 2;
    case TaskRun_ExportArchiveStatus.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface TaskRun_ExecutionDetail {
  /** Currently, the following fields are only used for EXECUTING status. */
  commandsTotal: number;
  commandsCompleted: number;
  commandStartPosition: TaskRun_ExecutionDetail_Position | undefined;
  commandEndPosition: TaskRun_ExecutionDetail_Position | undefined;
}

export interface TaskRun_ExecutionDetail_Position {
  /** The line number, starting from 0. */
  line: number;
  /** The column number, starting from 0. */
  column: number;
}

export interface TaskRunLog {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/log */
  name: string;
  entries: TaskRunLogEntry[];
}

export interface TaskRunLogEntry {
  type: TaskRunLogEntry_Type;
  logTime: Date | undefined;
  schemaDump: TaskRunLogEntry_SchemaDump | undefined;
  commandExecute: TaskRunLogEntry_CommandExecute | undefined;
  databaseSync: TaskRunLogEntry_DatabaseSync | undefined;
  taskRunStatusUpdate: TaskRunLogEntry_TaskRunStatusUpdate | undefined;
  transactionControl: TaskRunLogEntry_TransactionControl | undefined;
}

export enum TaskRunLogEntry_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  SCHEMA_DUMP = "SCHEMA_DUMP",
  COMMAND_EXECUTE = "COMMAND_EXECUTE",
  DATABASE_SYNC = "DATABASE_SYNC",
  TASK_RUN_STATUS_UPDATE = "TASK_RUN_STATUS_UPDATE",
  TRANSACTION_CONTROL = "TRANSACTION_CONTROL",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRunLogEntry_TypeFromJSON(object: any): TaskRunLogEntry_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return TaskRunLogEntry_Type.TYPE_UNSPECIFIED;
    case 1:
    case "SCHEMA_DUMP":
      return TaskRunLogEntry_Type.SCHEMA_DUMP;
    case 2:
    case "COMMAND_EXECUTE":
      return TaskRunLogEntry_Type.COMMAND_EXECUTE;
    case 3:
    case "DATABASE_SYNC":
      return TaskRunLogEntry_Type.DATABASE_SYNC;
    case 4:
    case "TASK_RUN_STATUS_UPDATE":
      return TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE;
    case 5:
    case "TRANSACTION_CONTROL":
      return TaskRunLogEntry_Type.TRANSACTION_CONTROL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRunLogEntry_Type.UNRECOGNIZED;
  }
}

export function taskRunLogEntry_TypeToJSON(object: TaskRunLogEntry_Type): string {
  switch (object) {
    case TaskRunLogEntry_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return "SCHEMA_DUMP";
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return "COMMAND_EXECUTE";
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return "DATABASE_SYNC";
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
      return "TASK_RUN_STATUS_UPDATE";
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return "TRANSACTION_CONTROL";
    case TaskRunLogEntry_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRunLogEntry_TypeToNumber(object: TaskRunLogEntry_Type): number {
  switch (object) {
    case TaskRunLogEntry_Type.TYPE_UNSPECIFIED:
      return 0;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return 1;
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return 2;
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return 3;
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
      return 4;
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return 5;
    case TaskRunLogEntry_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface TaskRunLogEntry_SchemaDump {
  startTime: Date | undefined;
  endTime: Date | undefined;
  error: string;
}

export interface TaskRunLogEntry_CommandExecute {
  logTime:
    | Date
    | undefined;
  /** The indexes of the executed commands. */
  commandIndexes: number[];
  response: TaskRunLogEntry_CommandExecute_CommandResponse | undefined;
}

export interface TaskRunLogEntry_CommandExecute_CommandResponse {
  logTime: Date | undefined;
  error: string;
  affectedRows: number;
  /**
   * `all_affected_rows` is the affected rows of each command.
   * `all_affected_rows` may be unavailable if the database driver doesn't support it. Caller should fallback to `affected_rows` in that case.
   */
  allAffectedRows: number[];
}

export interface TaskRunLogEntry_DatabaseSync {
  startTime: Date | undefined;
  endTime: Date | undefined;
  error: string;
}

export interface TaskRunLogEntry_TaskRunStatusUpdate {
  status: TaskRunLogEntry_TaskRunStatusUpdate_Status;
}

export enum TaskRunLogEntry_TaskRunStatusUpdate_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  /** RUNNING_WAITING - the task run is ready to be executed by the scheduler */
  RUNNING_WAITING = "RUNNING_WAITING",
  /** RUNNING_RUNNING - the task run is being executed by the scheduler */
  RUNNING_RUNNING = "RUNNING_RUNNING",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRunLogEntry_TaskRunStatusUpdate_StatusFromJSON(
  object: any,
): TaskRunLogEntry_TaskRunStatusUpdate_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED;
    case 1:
    case "RUNNING_WAITING":
      return TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING;
    case 2:
    case "RUNNING_RUNNING":
      return TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRunLogEntry_TaskRunStatusUpdate_Status.UNRECOGNIZED;
  }
}

export function taskRunLogEntry_TaskRunStatusUpdate_StatusToJSON(
  object: TaskRunLogEntry_TaskRunStatusUpdate_Status,
): string {
  switch (object) {
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING:
      return "RUNNING_WAITING";
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING:
      return "RUNNING_RUNNING";
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRunLogEntry_TaskRunStatusUpdate_StatusToNumber(
  object: TaskRunLogEntry_TaskRunStatusUpdate_Status,
): number {
  switch (object) {
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED:
      return 0;
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_WAITING:
      return 1;
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.RUNNING_RUNNING:
      return 2;
    case TaskRunLogEntry_TaskRunStatusUpdate_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface TaskRunLogEntry_TransactionControl {
  type: TaskRunLogEntry_TransactionControl_Type;
  error: string;
}

export enum TaskRunLogEntry_TransactionControl_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  BEGIN = "BEGIN",
  COMMIT = "COMMIT",
  ROLLBACK = "ROLLBACK",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRunLogEntry_TransactionControl_TypeFromJSON(object: any): TaskRunLogEntry_TransactionControl_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED;
    case 1:
    case "BEGIN":
      return TaskRunLogEntry_TransactionControl_Type.BEGIN;
    case 2:
    case "COMMIT":
      return TaskRunLogEntry_TransactionControl_Type.COMMIT;
    case 3:
    case "ROLLBACK":
      return TaskRunLogEntry_TransactionControl_Type.ROLLBACK;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRunLogEntry_TransactionControl_Type.UNRECOGNIZED;
  }
}

export function taskRunLogEntry_TransactionControl_TypeToJSON(object: TaskRunLogEntry_TransactionControl_Type): string {
  switch (object) {
    case TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case TaskRunLogEntry_TransactionControl_Type.BEGIN:
      return "BEGIN";
    case TaskRunLogEntry_TransactionControl_Type.COMMIT:
      return "COMMIT";
    case TaskRunLogEntry_TransactionControl_Type.ROLLBACK:
      return "ROLLBACK";
    case TaskRunLogEntry_TransactionControl_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRunLogEntry_TransactionControl_TypeToNumber(
  object: TaskRunLogEntry_TransactionControl_Type,
): number {
  switch (object) {
    case TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED:
      return 0;
    case TaskRunLogEntry_TransactionControl_Type.BEGIN:
      return 1;
    case TaskRunLogEntry_TransactionControl_Type.COMMIT:
      return 2;
    case TaskRunLogEntry_TransactionControl_Type.ROLLBACK:
      return 3;
    case TaskRunLogEntry_TransactionControl_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

function createBaseBatchRunTasksRequest(): BatchRunTasksRequest {
  return { parent: "", tasks: [], reason: "" };
}

export const BatchRunTasksRequest = {
  encode(message: BatchRunTasksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.tasks) {
      writer.uint32(18).string(v!);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchRunTasksRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchRunTasksRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tasks.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.reason = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchRunTasksRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      tasks: globalThis.Array.isArray(object?.tasks) ? object.tasks.map((e: any) => globalThis.String(e)) : [],
      reason: isSet(object.reason) ? globalThis.String(object.reason) : "",
    };
  },

  toJSON(message: BatchRunTasksRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.tasks?.length) {
      obj.tasks = message.tasks;
    }
    if (message.reason !== "") {
      obj.reason = message.reason;
    }
    return obj;
  },

  create(base?: DeepPartial<BatchRunTasksRequest>): BatchRunTasksRequest {
    return BatchRunTasksRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BatchRunTasksRequest>): BatchRunTasksRequest {
    const message = createBaseBatchRunTasksRequest();
    message.parent = object.parent ?? "";
    message.tasks = object.tasks?.map((e) => e) || [];
    message.reason = object.reason ?? "";
    return message;
  },
};

function createBaseBatchRunTasksResponse(): BatchRunTasksResponse {
  return {};
}

export const BatchRunTasksResponse = {
  encode(_: BatchRunTasksResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchRunTasksResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchRunTasksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): BatchRunTasksResponse {
    return {};
  },

  toJSON(_: BatchRunTasksResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<BatchRunTasksResponse>): BatchRunTasksResponse {
    return BatchRunTasksResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<BatchRunTasksResponse>): BatchRunTasksResponse {
    const message = createBaseBatchRunTasksResponse();
    return message;
  },
};

function createBaseBatchSkipTasksRequest(): BatchSkipTasksRequest {
  return { parent: "", tasks: [], reason: "" };
}

export const BatchSkipTasksRequest = {
  encode(message: BatchSkipTasksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.tasks) {
      writer.uint32(18).string(v!);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchSkipTasksRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchSkipTasksRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tasks.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.reason = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchSkipTasksRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      tasks: globalThis.Array.isArray(object?.tasks) ? object.tasks.map((e: any) => globalThis.String(e)) : [],
      reason: isSet(object.reason) ? globalThis.String(object.reason) : "",
    };
  },

  toJSON(message: BatchSkipTasksRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.tasks?.length) {
      obj.tasks = message.tasks;
    }
    if (message.reason !== "") {
      obj.reason = message.reason;
    }
    return obj;
  },

  create(base?: DeepPartial<BatchSkipTasksRequest>): BatchSkipTasksRequest {
    return BatchSkipTasksRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BatchSkipTasksRequest>): BatchSkipTasksRequest {
    const message = createBaseBatchSkipTasksRequest();
    message.parent = object.parent ?? "";
    message.tasks = object.tasks?.map((e) => e) || [];
    message.reason = object.reason ?? "";
    return message;
  },
};

function createBaseBatchSkipTasksResponse(): BatchSkipTasksResponse {
  return {};
}

export const BatchSkipTasksResponse = {
  encode(_: BatchSkipTasksResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchSkipTasksResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchSkipTasksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): BatchSkipTasksResponse {
    return {};
  },

  toJSON(_: BatchSkipTasksResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<BatchSkipTasksResponse>): BatchSkipTasksResponse {
    return BatchSkipTasksResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<BatchSkipTasksResponse>): BatchSkipTasksResponse {
    const message = createBaseBatchSkipTasksResponse();
    return message;
  },
};

function createBaseBatchCancelTaskRunsRequest(): BatchCancelTaskRunsRequest {
  return { parent: "", taskRuns: [], reason: "" };
}

export const BatchCancelTaskRunsRequest = {
  encode(message: BatchCancelTaskRunsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.taskRuns) {
      writer.uint32(18).string(v!);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchCancelTaskRunsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchCancelTaskRunsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.taskRuns.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.reason = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchCancelTaskRunsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      taskRuns: globalThis.Array.isArray(object?.taskRuns) ? object.taskRuns.map((e: any) => globalThis.String(e)) : [],
      reason: isSet(object.reason) ? globalThis.String(object.reason) : "",
    };
  },

  toJSON(message: BatchCancelTaskRunsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.taskRuns?.length) {
      obj.taskRuns = message.taskRuns;
    }
    if (message.reason !== "") {
      obj.reason = message.reason;
    }
    return obj;
  },

  create(base?: DeepPartial<BatchCancelTaskRunsRequest>): BatchCancelTaskRunsRequest {
    return BatchCancelTaskRunsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BatchCancelTaskRunsRequest>): BatchCancelTaskRunsRequest {
    const message = createBaseBatchCancelTaskRunsRequest();
    message.parent = object.parent ?? "";
    message.taskRuns = object.taskRuns?.map((e) => e) || [];
    message.reason = object.reason ?? "";
    return message;
  },
};

function createBaseBatchCancelTaskRunsResponse(): BatchCancelTaskRunsResponse {
  return {};
}

export const BatchCancelTaskRunsResponse = {
  encode(_: BatchCancelTaskRunsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchCancelTaskRunsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchCancelTaskRunsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): BatchCancelTaskRunsResponse {
    return {};
  },

  toJSON(_: BatchCancelTaskRunsResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<BatchCancelTaskRunsResponse>): BatchCancelTaskRunsResponse {
    return BatchCancelTaskRunsResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<BatchCancelTaskRunsResponse>): BatchCancelTaskRunsResponse {
    const message = createBaseBatchCancelTaskRunsResponse();
    return message;
  },
};

function createBaseGetRolloutRequest(): GetRolloutRequest {
  return { name: "" };
}

export const GetRolloutRequest = {
  encode(message: GetRolloutRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetRolloutRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetRolloutRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetRolloutRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetRolloutRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetRolloutRequest>): GetRolloutRequest {
    return GetRolloutRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetRolloutRequest>): GetRolloutRequest {
    const message = createBaseGetRolloutRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseCreateRolloutRequest(): CreateRolloutRequest {
  return { parent: "", rollout: undefined };
}

export const CreateRolloutRequest = {
  encode(message: CreateRolloutRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.rollout !== undefined) {
      Rollout.encode(message.rollout, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateRolloutRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateRolloutRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.rollout = Rollout.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateRolloutRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      rollout: isSet(object.rollout) ? Rollout.fromJSON(object.rollout) : undefined,
    };
  },

  toJSON(message: CreateRolloutRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.rollout !== undefined) {
      obj.rollout = Rollout.toJSON(message.rollout);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateRolloutRequest>): CreateRolloutRequest {
    return CreateRolloutRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateRolloutRequest>): CreateRolloutRequest {
    const message = createBaseCreateRolloutRequest();
    message.parent = object.parent ?? "";
    message.rollout = (object.rollout !== undefined && object.rollout !== null)
      ? Rollout.fromPartial(object.rollout)
      : undefined;
    return message;
  },
};

function createBasePreviewRolloutRequest(): PreviewRolloutRequest {
  return { project: "", plan: undefined };
}

export const PreviewRolloutRequest = {
  encode(message: PreviewRolloutRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.plan !== undefined) {
      Plan.encode(message.plan, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PreviewRolloutRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePreviewRolloutRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.plan = Plan.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PreviewRolloutRequest {
    return {
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      plan: isSet(object.plan) ? Plan.fromJSON(object.plan) : undefined,
    };
  },

  toJSON(message: PreviewRolloutRequest): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.plan !== undefined) {
      obj.plan = Plan.toJSON(message.plan);
    }
    return obj;
  },

  create(base?: DeepPartial<PreviewRolloutRequest>): PreviewRolloutRequest {
    return PreviewRolloutRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PreviewRolloutRequest>): PreviewRolloutRequest {
    const message = createBasePreviewRolloutRequest();
    message.project = object.project ?? "";
    message.plan = (object.plan !== undefined && object.plan !== null) ? Plan.fromPartial(object.plan) : undefined;
    return message;
  },
};

function createBaseListTaskRunsRequest(): ListTaskRunsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListTaskRunsRequest = {
  encode(message: ListTaskRunsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTaskRunsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTaskRunsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTaskRunsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListTaskRunsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListTaskRunsRequest>): ListTaskRunsRequest {
    return ListTaskRunsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListTaskRunsRequest>): ListTaskRunsRequest {
    const message = createBaseListTaskRunsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListTaskRunsResponse(): ListTaskRunsResponse {
  return { taskRuns: [], nextPageToken: "" };
}

export const ListTaskRunsResponse = {
  encode(message: ListTaskRunsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.taskRuns) {
      TaskRun.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListTaskRunsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListTaskRunsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.taskRuns.push(TaskRun.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListTaskRunsResponse {
    return {
      taskRuns: globalThis.Array.isArray(object?.taskRuns) ? object.taskRuns.map((e: any) => TaskRun.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListTaskRunsResponse): unknown {
    const obj: any = {};
    if (message.taskRuns?.length) {
      obj.taskRuns = message.taskRuns.map((e) => TaskRun.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListTaskRunsResponse>): ListTaskRunsResponse {
    return ListTaskRunsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListTaskRunsResponse>): ListTaskRunsResponse {
    const message = createBaseListTaskRunsResponse();
    message.taskRuns = object.taskRuns?.map((e) => TaskRun.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetTaskRunLogRequest(): GetTaskRunLogRequest {
  return { parent: "" };
}

export const GetTaskRunLogRequest = {
  encode(message: GetTaskRunLogRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetTaskRunLogRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetTaskRunLogRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetTaskRunLogRequest {
    return { parent: isSet(object.parent) ? globalThis.String(object.parent) : "" };
  },

  toJSON(message: GetTaskRunLogRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    return obj;
  },

  create(base?: DeepPartial<GetTaskRunLogRequest>): GetTaskRunLogRequest {
    return GetTaskRunLogRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetTaskRunLogRequest>): GetTaskRunLogRequest {
    const message = createBaseGetTaskRunLogRequest();
    message.parent = object.parent ?? "";
    return message;
  },
};

function createBaseRollout(): Rollout {
  return { name: "", uid: "", plan: "", title: "", stages: [] };
}

export const Rollout = {
  encode(message: Rollout, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.plan !== "") {
      writer.uint32(26).string(message.plan);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    for (const v of message.stages) {
      Stage.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Rollout {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRollout();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.plan = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.stages.push(Stage.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Rollout {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      plan: isSet(object.plan) ? globalThis.String(object.plan) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      stages: globalThis.Array.isArray(object?.stages) ? object.stages.map((e: any) => Stage.fromJSON(e)) : [],
    };
  },

  toJSON(message: Rollout): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.plan !== "") {
      obj.plan = message.plan;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.stages?.length) {
      obj.stages = message.stages.map((e) => Stage.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<Rollout>): Rollout {
    return Rollout.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Rollout>): Rollout {
    const message = createBaseRollout();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.plan = object.plan ?? "";
    message.title = object.title ?? "";
    message.stages = object.stages?.map((e) => Stage.fromPartial(e)) || [];
    return message;
  },
};

function createBaseStage(): Stage {
  return { name: "", uid: "", title: "", tasks: [] };
}

export const Stage = {
  encode(message: Stage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    for (const v of message.tasks) {
      Task.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Stage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.tasks.push(Task.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Stage {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      tasks: globalThis.Array.isArray(object?.tasks) ? object.tasks.map((e: any) => Task.fromJSON(e)) : [],
    };
  },

  toJSON(message: Stage): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.tasks?.length) {
      obj.tasks = message.tasks.map((e) => Task.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<Stage>): Stage {
    return Stage.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Stage>): Stage {
    const message = createBaseStage();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.tasks = object.tasks?.map((e) => Task.fromPartial(e)) || [];
    return message;
  },
};

function createBaseTask(): Task {
  return {
    name: "",
    uid: "",
    title: "",
    specId: "",
    status: Task_Status.STATUS_UNSPECIFIED,
    skippedReason: "",
    type: Task_Type.TYPE_UNSPECIFIED,
    dependsOnTasks: [],
    target: "",
    databaseCreate: undefined,
    databaseSchemaBaseline: undefined,
    databaseSchemaUpdate: undefined,
    databaseDataUpdate: undefined,
    databaseDataExport: undefined,
  };
}

export const Task = {
  encode(message: Task, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.specId !== "") {
      writer.uint32(34).string(message.specId);
    }
    if (message.status !== Task_Status.STATUS_UNSPECIFIED) {
      writer.uint32(40).int32(task_StatusToNumber(message.status));
    }
    if (message.skippedReason !== "") {
      writer.uint32(122).string(message.skippedReason);
    }
    if (message.type !== Task_Type.TYPE_UNSPECIFIED) {
      writer.uint32(48).int32(task_TypeToNumber(message.type));
    }
    for (const v of message.dependsOnTasks) {
      writer.uint32(58).string(v!);
    }
    if (message.target !== "") {
      writer.uint32(66).string(message.target);
    }
    if (message.databaseCreate !== undefined) {
      Task_DatabaseCreate.encode(message.databaseCreate, writer.uint32(74).fork()).ldelim();
    }
    if (message.databaseSchemaBaseline !== undefined) {
      Task_DatabaseSchemaBaseline.encode(message.databaseSchemaBaseline, writer.uint32(82).fork()).ldelim();
    }
    if (message.databaseSchemaUpdate !== undefined) {
      Task_DatabaseSchemaUpdate.encode(message.databaseSchemaUpdate, writer.uint32(90).fork()).ldelim();
    }
    if (message.databaseDataUpdate !== undefined) {
      Task_DatabaseDataUpdate.encode(message.databaseDataUpdate, writer.uint32(98).fork()).ldelim();
    }
    if (message.databaseDataExport !== undefined) {
      Task_DatabaseDataExport.encode(message.databaseDataExport, writer.uint32(130).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.specId = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.status = task_StatusFromJSON(reader.int32());
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.skippedReason = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.type = task_TypeFromJSON(reader.int32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.dependsOnTasks.push(reader.string());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.target = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.databaseCreate = Task_DatabaseCreate.decode(reader, reader.uint32());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.databaseSchemaBaseline = Task_DatabaseSchemaBaseline.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.databaseSchemaUpdate = Task_DatabaseSchemaUpdate.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.databaseDataUpdate = Task_DatabaseDataUpdate.decode(reader, reader.uint32());
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.databaseDataExport = Task_DatabaseDataExport.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      specId: isSet(object.specId) ? globalThis.String(object.specId) : "",
      status: isSet(object.status) ? task_StatusFromJSON(object.status) : Task_Status.STATUS_UNSPECIFIED,
      skippedReason: isSet(object.skippedReason) ? globalThis.String(object.skippedReason) : "",
      type: isSet(object.type) ? task_TypeFromJSON(object.type) : Task_Type.TYPE_UNSPECIFIED,
      dependsOnTasks: globalThis.Array.isArray(object?.dependsOnTasks)
        ? object.dependsOnTasks.map((e: any) => globalThis.String(e))
        : [],
      target: isSet(object.target) ? globalThis.String(object.target) : "",
      databaseCreate: isSet(object.databaseCreate) ? Task_DatabaseCreate.fromJSON(object.databaseCreate) : undefined,
      databaseSchemaBaseline: isSet(object.databaseSchemaBaseline)
        ? Task_DatabaseSchemaBaseline.fromJSON(object.databaseSchemaBaseline)
        : undefined,
      databaseSchemaUpdate: isSet(object.databaseSchemaUpdate)
        ? Task_DatabaseSchemaUpdate.fromJSON(object.databaseSchemaUpdate)
        : undefined,
      databaseDataUpdate: isSet(object.databaseDataUpdate)
        ? Task_DatabaseDataUpdate.fromJSON(object.databaseDataUpdate)
        : undefined,
      databaseDataExport: isSet(object.databaseDataExport)
        ? Task_DatabaseDataExport.fromJSON(object.databaseDataExport)
        : undefined,
    };
  },

  toJSON(message: Task): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.specId !== "") {
      obj.specId = message.specId;
    }
    if (message.status !== Task_Status.STATUS_UNSPECIFIED) {
      obj.status = task_StatusToJSON(message.status);
    }
    if (message.skippedReason !== "") {
      obj.skippedReason = message.skippedReason;
    }
    if (message.type !== Task_Type.TYPE_UNSPECIFIED) {
      obj.type = task_TypeToJSON(message.type);
    }
    if (message.dependsOnTasks?.length) {
      obj.dependsOnTasks = message.dependsOnTasks;
    }
    if (message.target !== "") {
      obj.target = message.target;
    }
    if (message.databaseCreate !== undefined) {
      obj.databaseCreate = Task_DatabaseCreate.toJSON(message.databaseCreate);
    }
    if (message.databaseSchemaBaseline !== undefined) {
      obj.databaseSchemaBaseline = Task_DatabaseSchemaBaseline.toJSON(message.databaseSchemaBaseline);
    }
    if (message.databaseSchemaUpdate !== undefined) {
      obj.databaseSchemaUpdate = Task_DatabaseSchemaUpdate.toJSON(message.databaseSchemaUpdate);
    }
    if (message.databaseDataUpdate !== undefined) {
      obj.databaseDataUpdate = Task_DatabaseDataUpdate.toJSON(message.databaseDataUpdate);
    }
    if (message.databaseDataExport !== undefined) {
      obj.databaseDataExport = Task_DatabaseDataExport.toJSON(message.databaseDataExport);
    }
    return obj;
  },

  create(base?: DeepPartial<Task>): Task {
    return Task.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task>): Task {
    const message = createBaseTask();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.specId = object.specId ?? "";
    message.status = object.status ?? Task_Status.STATUS_UNSPECIFIED;
    message.skippedReason = object.skippedReason ?? "";
    message.type = object.type ?? Task_Type.TYPE_UNSPECIFIED;
    message.dependsOnTasks = object.dependsOnTasks?.map((e) => e) || [];
    message.target = object.target ?? "";
    message.databaseCreate = (object.databaseCreate !== undefined && object.databaseCreate !== null)
      ? Task_DatabaseCreate.fromPartial(object.databaseCreate)
      : undefined;
    message.databaseSchemaBaseline =
      (object.databaseSchemaBaseline !== undefined && object.databaseSchemaBaseline !== null)
        ? Task_DatabaseSchemaBaseline.fromPartial(object.databaseSchemaBaseline)
        : undefined;
    message.databaseSchemaUpdate = (object.databaseSchemaUpdate !== undefined && object.databaseSchemaUpdate !== null)
      ? Task_DatabaseSchemaUpdate.fromPartial(object.databaseSchemaUpdate)
      : undefined;
    message.databaseDataUpdate = (object.databaseDataUpdate !== undefined && object.databaseDataUpdate !== null)
      ? Task_DatabaseDataUpdate.fromPartial(object.databaseDataUpdate)
      : undefined;
    message.databaseDataExport = (object.databaseDataExport !== undefined && object.databaseDataExport !== null)
      ? Task_DatabaseDataExport.fromPartial(object.databaseDataExport)
      : undefined;
    return message;
  },
};

function createBaseTask_DatabaseCreate(): Task_DatabaseCreate {
  return {
    project: "",
    database: "",
    table: "",
    sheet: "",
    characterSet: "",
    collation: "",
    environment: "",
    labels: {},
  };
}

export const Task_DatabaseCreate = {
  encode(message: Task_DatabaseCreate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.database !== "") {
      writer.uint32(18).string(message.database);
    }
    if (message.table !== "") {
      writer.uint32(26).string(message.table);
    }
    if (message.sheet !== "") {
      writer.uint32(34).string(message.sheet);
    }
    if (message.characterSet !== "") {
      writer.uint32(42).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(50).string(message.collation);
    }
    if (message.environment !== "") {
      writer.uint32(58).string(message.environment);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      Task_DatabaseCreate_LabelsEntry.encode({ key: key as any, value }, writer.uint32(66).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseCreate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseCreate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.database = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.table = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.environment = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          const entry8 = Task_DatabaseCreate_LabelsEntry.decode(reader, reader.uint32());
          if (entry8.value !== undefined) {
            message.labels[entry8.key] = entry8.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseCreate {
    return {
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      characterSet: isSet(object.characterSet) ? globalThis.String(object.characterSet) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      environment: isSet(object.environment) ? globalThis.String(object.environment) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Task_DatabaseCreate): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.environment !== "") {
      obj.environment = message.environment;
    }
    if (message.labels) {
      const entries = Object.entries(message.labels);
      if (entries.length > 0) {
        obj.labels = {};
        entries.forEach(([k, v]) => {
          obj.labels[k] = v;
        });
      }
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseCreate>): Task_DatabaseCreate {
    return Task_DatabaseCreate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseCreate>): Task_DatabaseCreate {
    const message = createBaseTask_DatabaseCreate();
    message.project = object.project ?? "";
    message.database = object.database ?? "";
    message.table = object.table ?? "";
    message.sheet = object.sheet ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.environment = object.environment ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseTask_DatabaseCreate_LabelsEntry(): Task_DatabaseCreate_LabelsEntry {
  return { key: "", value: "" };
}

export const Task_DatabaseCreate_LabelsEntry = {
  encode(message: Task_DatabaseCreate_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseCreate_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseCreate_LabelsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseCreate_LabelsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: Task_DatabaseCreate_LabelsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseCreate_LabelsEntry>): Task_DatabaseCreate_LabelsEntry {
    return Task_DatabaseCreate_LabelsEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseCreate_LabelsEntry>): Task_DatabaseCreate_LabelsEntry {
    const message = createBaseTask_DatabaseCreate_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseTask_DatabaseSchemaBaseline(): Task_DatabaseSchemaBaseline {
  return { schemaVersion: "" };
}

export const Task_DatabaseSchemaBaseline = {
  encode(message: Task_DatabaseSchemaBaseline, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaVersion !== "") {
      writer.uint32(10).string(message.schemaVersion);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseSchemaBaseline {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseSchemaBaseline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseSchemaBaseline {
    return { schemaVersion: isSet(object.schemaVersion) ? globalThis.String(object.schemaVersion) : "" };
  },

  toJSON(message: Task_DatabaseSchemaBaseline): unknown {
    const obj: any = {};
    if (message.schemaVersion !== "") {
      obj.schemaVersion = message.schemaVersion;
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseSchemaBaseline>): Task_DatabaseSchemaBaseline {
    return Task_DatabaseSchemaBaseline.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseSchemaBaseline>): Task_DatabaseSchemaBaseline {
    const message = createBaseTask_DatabaseSchemaBaseline();
    message.schemaVersion = object.schemaVersion ?? "";
    return message;
  },
};

function createBaseTask_DatabaseSchemaUpdate(): Task_DatabaseSchemaUpdate {
  return { sheet: "", schemaVersion: "" };
}

export const Task_DatabaseSchemaUpdate = {
  encode(message: Task_DatabaseSchemaUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheet !== "") {
      writer.uint32(10).string(message.sheet);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(18).string(message.schemaVersion);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseSchemaUpdate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseSchemaUpdate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseSchemaUpdate {
    return {
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      schemaVersion: isSet(object.schemaVersion) ? globalThis.String(object.schemaVersion) : "",
    };
  },

  toJSON(message: Task_DatabaseSchemaUpdate): unknown {
    const obj: any = {};
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.schemaVersion !== "") {
      obj.schemaVersion = message.schemaVersion;
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseSchemaUpdate>): Task_DatabaseSchemaUpdate {
    return Task_DatabaseSchemaUpdate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseSchemaUpdate>): Task_DatabaseSchemaUpdate {
    const message = createBaseTask_DatabaseSchemaUpdate();
    message.sheet = object.sheet ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    return message;
  },
};

function createBaseTask_DatabaseDataUpdate(): Task_DatabaseDataUpdate {
  return { sheet: "", schemaVersion: "" };
}

export const Task_DatabaseDataUpdate = {
  encode(message: Task_DatabaseDataUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheet !== "") {
      writer.uint32(10).string(message.sheet);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(18).string(message.schemaVersion);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseDataUpdate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseDataUpdate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseDataUpdate {
    return {
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      schemaVersion: isSet(object.schemaVersion) ? globalThis.String(object.schemaVersion) : "",
    };
  },

  toJSON(message: Task_DatabaseDataUpdate): unknown {
    const obj: any = {};
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.schemaVersion !== "") {
      obj.schemaVersion = message.schemaVersion;
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseDataUpdate>): Task_DatabaseDataUpdate {
    return Task_DatabaseDataUpdate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseDataUpdate>): Task_DatabaseDataUpdate {
    const message = createBaseTask_DatabaseDataUpdate();
    message.sheet = object.sheet ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    return message;
  },
};

function createBaseTask_DatabaseDataExport(): Task_DatabaseDataExport {
  return { target: "", sheet: "", format: ExportFormat.FORMAT_UNSPECIFIED, password: undefined };
}

export const Task_DatabaseDataExport = {
  encode(message: Task_DatabaseDataExport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.sheet !== "") {
      writer.uint32(18).string(message.sheet);
    }
    if (message.format !== ExportFormat.FORMAT_UNSPECIFIED) {
      writer.uint32(24).int32(exportFormatToNumber(message.format));
    }
    if (message.password !== undefined) {
      writer.uint32(34).string(message.password);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseDataExport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseDataExport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.target = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.format = exportFormatFromJSON(reader.int32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.password = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseDataExport {
    return {
      target: isSet(object.target) ? globalThis.String(object.target) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      format: isSet(object.format) ? exportFormatFromJSON(object.format) : ExportFormat.FORMAT_UNSPECIFIED,
      password: isSet(object.password) ? globalThis.String(object.password) : undefined,
    };
  },

  toJSON(message: Task_DatabaseDataExport): unknown {
    const obj: any = {};
    if (message.target !== "") {
      obj.target = message.target;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.format !== ExportFormat.FORMAT_UNSPECIFIED) {
      obj.format = exportFormatToJSON(message.format);
    }
    if (message.password !== undefined) {
      obj.password = message.password;
    }
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseDataExport>): Task_DatabaseDataExport {
    return Task_DatabaseDataExport.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Task_DatabaseDataExport>): Task_DatabaseDataExport {
    const message = createBaseTask_DatabaseDataExport();
    message.target = object.target ?? "";
    message.sheet = object.sheet ?? "";
    message.format = object.format ?? ExportFormat.FORMAT_UNSPECIFIED;
    message.password = object.password ?? undefined;
    return message;
  },
};

function createBaseTaskRun(): TaskRun {
  return {
    name: "",
    uid: "",
    creator: "",
    updater: "",
    createTime: undefined,
    updateTime: undefined,
    title: "",
    status: TaskRun_Status.STATUS_UNSPECIFIED,
    detail: "",
    changeHistory: "",
    schemaVersion: "",
    executionStatus: TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED,
    executionDetail: undefined,
    startTime: undefined,
    exportArchiveStatus: TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED,
  };
}

export const TaskRun = {
  encode(message: TaskRun, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(34).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(50).fork()).ldelim();
    }
    if (message.title !== "") {
      writer.uint32(58).string(message.title);
    }
    if (message.status !== TaskRun_Status.STATUS_UNSPECIFIED) {
      writer.uint32(64).int32(taskRun_StatusToNumber(message.status));
    }
    if (message.detail !== "") {
      writer.uint32(74).string(message.detail);
    }
    if (message.changeHistory !== "") {
      writer.uint32(82).string(message.changeHistory);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(90).string(message.schemaVersion);
    }
    if (message.executionStatus !== TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED) {
      writer.uint32(96).int32(taskRun_ExecutionStatusToNumber(message.executionStatus));
    }
    if (message.executionDetail !== undefined) {
      TaskRun_ExecutionDetail.encode(message.executionDetail, writer.uint32(122).fork()).ldelim();
    }
    if (message.startTime !== undefined) {
      Timestamp.encode(toTimestamp(message.startTime), writer.uint32(114).fork()).ldelim();
    }
    if (message.exportArchiveStatus !== TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED) {
      writer.uint32(128).int32(taskRun_ExportArchiveStatusToNumber(message.exportArchiveStatus));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRun {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRun();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.title = reader.string();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.status = taskRun_StatusFromJSON(reader.int32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.detail = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.changeHistory = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
        case 12:
          if (tag !== 96) {
            break;
          }

          message.executionStatus = taskRun_ExecutionStatusFromJSON(reader.int32());
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.executionDetail = TaskRun_ExecutionDetail.decode(reader, reader.uint32());
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.startTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.exportArchiveStatus = taskRun_ExportArchiveStatusFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRun {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      status: isSet(object.status) ? taskRun_StatusFromJSON(object.status) : TaskRun_Status.STATUS_UNSPECIFIED,
      detail: isSet(object.detail) ? globalThis.String(object.detail) : "",
      changeHistory: isSet(object.changeHistory) ? globalThis.String(object.changeHistory) : "",
      schemaVersion: isSet(object.schemaVersion) ? globalThis.String(object.schemaVersion) : "",
      executionStatus: isSet(object.executionStatus)
        ? taskRun_ExecutionStatusFromJSON(object.executionStatus)
        : TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED,
      executionDetail: isSet(object.executionDetail)
        ? TaskRun_ExecutionDetail.fromJSON(object.executionDetail)
        : undefined,
      startTime: isSet(object.startTime) ? fromJsonTimestamp(object.startTime) : undefined,
      exportArchiveStatus: isSet(object.exportArchiveStatus)
        ? taskRun_ExportArchiveStatusFromJSON(object.exportArchiveStatus)
        : TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED,
    };
  },

  toJSON(message: TaskRun): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.status !== TaskRun_Status.STATUS_UNSPECIFIED) {
      obj.status = taskRun_StatusToJSON(message.status);
    }
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    if (message.changeHistory !== "") {
      obj.changeHistory = message.changeHistory;
    }
    if (message.schemaVersion !== "") {
      obj.schemaVersion = message.schemaVersion;
    }
    if (message.executionStatus !== TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED) {
      obj.executionStatus = taskRun_ExecutionStatusToJSON(message.executionStatus);
    }
    if (message.executionDetail !== undefined) {
      obj.executionDetail = TaskRun_ExecutionDetail.toJSON(message.executionDetail);
    }
    if (message.startTime !== undefined) {
      obj.startTime = message.startTime.toISOString();
    }
    if (message.exportArchiveStatus !== TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED) {
      obj.exportArchiveStatus = taskRun_ExportArchiveStatusToJSON(message.exportArchiveStatus);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRun>): TaskRun {
    return TaskRun.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRun>): TaskRun {
    const message = createBaseTaskRun();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.title = object.title ?? "";
    message.status = object.status ?? TaskRun_Status.STATUS_UNSPECIFIED;
    message.detail = object.detail ?? "";
    message.changeHistory = object.changeHistory ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    message.executionStatus = object.executionStatus ?? TaskRun_ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED;
    message.executionDetail = (object.executionDetail !== undefined && object.executionDetail !== null)
      ? TaskRun_ExecutionDetail.fromPartial(object.executionDetail)
      : undefined;
    message.startTime = object.startTime ?? undefined;
    message.exportArchiveStatus = object.exportArchiveStatus ??
      TaskRun_ExportArchiveStatus.EXPORT_ARCHIVE_STATUS_UNSPECIFIED;
    return message;
  },
};

function createBaseTaskRun_ExecutionDetail(): TaskRun_ExecutionDetail {
  return { commandsTotal: 0, commandsCompleted: 0, commandStartPosition: undefined, commandEndPosition: undefined };
}

export const TaskRun_ExecutionDetail = {
  encode(message: TaskRun_ExecutionDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.commandsTotal !== 0) {
      writer.uint32(8).int32(message.commandsTotal);
    }
    if (message.commandsCompleted !== 0) {
      writer.uint32(16).int32(message.commandsCompleted);
    }
    if (message.commandStartPosition !== undefined) {
      TaskRun_ExecutionDetail_Position.encode(message.commandStartPosition, writer.uint32(26).fork()).ldelim();
    }
    if (message.commandEndPosition !== undefined) {
      TaskRun_ExecutionDetail_Position.encode(message.commandEndPosition, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRun_ExecutionDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRun_ExecutionDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.commandsTotal = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.commandsCompleted = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.commandStartPosition = TaskRun_ExecutionDetail_Position.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.commandEndPosition = TaskRun_ExecutionDetail_Position.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRun_ExecutionDetail {
    return {
      commandsTotal: isSet(object.commandsTotal) ? globalThis.Number(object.commandsTotal) : 0,
      commandsCompleted: isSet(object.commandsCompleted) ? globalThis.Number(object.commandsCompleted) : 0,
      commandStartPosition: isSet(object.commandStartPosition)
        ? TaskRun_ExecutionDetail_Position.fromJSON(object.commandStartPosition)
        : undefined,
      commandEndPosition: isSet(object.commandEndPosition)
        ? TaskRun_ExecutionDetail_Position.fromJSON(object.commandEndPosition)
        : undefined,
    };
  },

  toJSON(message: TaskRun_ExecutionDetail): unknown {
    const obj: any = {};
    if (message.commandsTotal !== 0) {
      obj.commandsTotal = Math.round(message.commandsTotal);
    }
    if (message.commandsCompleted !== 0) {
      obj.commandsCompleted = Math.round(message.commandsCompleted);
    }
    if (message.commandStartPosition !== undefined) {
      obj.commandStartPosition = TaskRun_ExecutionDetail_Position.toJSON(message.commandStartPosition);
    }
    if (message.commandEndPosition !== undefined) {
      obj.commandEndPosition = TaskRun_ExecutionDetail_Position.toJSON(message.commandEndPosition);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRun_ExecutionDetail>): TaskRun_ExecutionDetail {
    return TaskRun_ExecutionDetail.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRun_ExecutionDetail>): TaskRun_ExecutionDetail {
    const message = createBaseTaskRun_ExecutionDetail();
    message.commandsTotal = object.commandsTotal ?? 0;
    message.commandsCompleted = object.commandsCompleted ?? 0;
    message.commandStartPosition = (object.commandStartPosition !== undefined && object.commandStartPosition !== null)
      ? TaskRun_ExecutionDetail_Position.fromPartial(object.commandStartPosition)
      : undefined;
    message.commandEndPosition = (object.commandEndPosition !== undefined && object.commandEndPosition !== null)
      ? TaskRun_ExecutionDetail_Position.fromPartial(object.commandEndPosition)
      : undefined;
    return message;
  },
};

function createBaseTaskRun_ExecutionDetail_Position(): TaskRun_ExecutionDetail_Position {
  return { line: 0, column: 0 };
}

export const TaskRun_ExecutionDetail_Position = {
  encode(message: TaskRun_ExecutionDetail_Position, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int32(message.line);
    }
    if (message.column !== 0) {
      writer.uint32(16).int32(message.column);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRun_ExecutionDetail_Position {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRun_ExecutionDetail_Position();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.line = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.column = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRun_ExecutionDetail_Position {
    return {
      line: isSet(object.line) ? globalThis.Number(object.line) : 0,
      column: isSet(object.column) ? globalThis.Number(object.column) : 0,
    };
  },

  toJSON(message: TaskRun_ExecutionDetail_Position): unknown {
    const obj: any = {};
    if (message.line !== 0) {
      obj.line = Math.round(message.line);
    }
    if (message.column !== 0) {
      obj.column = Math.round(message.column);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRun_ExecutionDetail_Position>): TaskRun_ExecutionDetail_Position {
    return TaskRun_ExecutionDetail_Position.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRun_ExecutionDetail_Position>): TaskRun_ExecutionDetail_Position {
    const message = createBaseTaskRun_ExecutionDetail_Position();
    message.line = object.line ?? 0;
    message.column = object.column ?? 0;
    return message;
  },
};

function createBaseTaskRunLog(): TaskRunLog {
  return { name: "", entries: [] };
}

export const TaskRunLog = {
  encode(message: TaskRunLog, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.entries) {
      TaskRunLogEntry.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.entries.push(TaskRunLogEntry.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLog {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      entries: globalThis.Array.isArray(object?.entries)
        ? object.entries.map((e: any) => TaskRunLogEntry.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TaskRunLog): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.entries?.length) {
      obj.entries = message.entries.map((e) => TaskRunLogEntry.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog>): TaskRunLog {
    return TaskRunLog.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLog>): TaskRunLog {
    const message = createBaseTaskRunLog();
    message.name = object.name ?? "";
    message.entries = object.entries?.map((e) => TaskRunLogEntry.fromPartial(e)) || [];
    return message;
  },
};

function createBaseTaskRunLogEntry(): TaskRunLogEntry {
  return {
    type: TaskRunLogEntry_Type.TYPE_UNSPECIFIED,
    logTime: undefined,
    schemaDump: undefined,
    commandExecute: undefined,
    databaseSync: undefined,
    taskRunStatusUpdate: undefined,
    transactionControl: undefined,
  };
}

export const TaskRunLogEntry = {
  encode(message: TaskRunLogEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== TaskRunLogEntry_Type.TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(taskRunLogEntry_TypeToNumber(message.type));
    }
    if (message.logTime !== undefined) {
      Timestamp.encode(toTimestamp(message.logTime), writer.uint32(50).fork()).ldelim();
    }
    if (message.schemaDump !== undefined) {
      TaskRunLogEntry_SchemaDump.encode(message.schemaDump, writer.uint32(18).fork()).ldelim();
    }
    if (message.commandExecute !== undefined) {
      TaskRunLogEntry_CommandExecute.encode(message.commandExecute, writer.uint32(26).fork()).ldelim();
    }
    if (message.databaseSync !== undefined) {
      TaskRunLogEntry_DatabaseSync.encode(message.databaseSync, writer.uint32(34).fork()).ldelim();
    }
    if (message.taskRunStatusUpdate !== undefined) {
      TaskRunLogEntry_TaskRunStatusUpdate.encode(message.taskRunStatusUpdate, writer.uint32(42).fork()).ldelim();
    }
    if (message.transactionControl !== undefined) {
      TaskRunLogEntry_TransactionControl.encode(message.transactionControl, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = taskRunLogEntry_TypeFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.logTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schemaDump = TaskRunLogEntry_SchemaDump.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.commandExecute = TaskRunLogEntry_CommandExecute.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.databaseSync = TaskRunLogEntry_DatabaseSync.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.taskRunStatusUpdate = TaskRunLogEntry_TaskRunStatusUpdate.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.transactionControl = TaskRunLogEntry_TransactionControl.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry {
    return {
      type: isSet(object.type) ? taskRunLogEntry_TypeFromJSON(object.type) : TaskRunLogEntry_Type.TYPE_UNSPECIFIED,
      logTime: isSet(object.logTime) ? fromJsonTimestamp(object.logTime) : undefined,
      schemaDump: isSet(object.schemaDump) ? TaskRunLogEntry_SchemaDump.fromJSON(object.schemaDump) : undefined,
      commandExecute: isSet(object.commandExecute)
        ? TaskRunLogEntry_CommandExecute.fromJSON(object.commandExecute)
        : undefined,
      databaseSync: isSet(object.databaseSync) ? TaskRunLogEntry_DatabaseSync.fromJSON(object.databaseSync) : undefined,
      taskRunStatusUpdate: isSet(object.taskRunStatusUpdate)
        ? TaskRunLogEntry_TaskRunStatusUpdate.fromJSON(object.taskRunStatusUpdate)
        : undefined,
      transactionControl: isSet(object.transactionControl)
        ? TaskRunLogEntry_TransactionControl.fromJSON(object.transactionControl)
        : undefined,
    };
  },

  toJSON(message: TaskRunLogEntry): unknown {
    const obj: any = {};
    if (message.type !== TaskRunLogEntry_Type.TYPE_UNSPECIFIED) {
      obj.type = taskRunLogEntry_TypeToJSON(message.type);
    }
    if (message.logTime !== undefined) {
      obj.logTime = message.logTime.toISOString();
    }
    if (message.schemaDump !== undefined) {
      obj.schemaDump = TaskRunLogEntry_SchemaDump.toJSON(message.schemaDump);
    }
    if (message.commandExecute !== undefined) {
      obj.commandExecute = TaskRunLogEntry_CommandExecute.toJSON(message.commandExecute);
    }
    if (message.databaseSync !== undefined) {
      obj.databaseSync = TaskRunLogEntry_DatabaseSync.toJSON(message.databaseSync);
    }
    if (message.taskRunStatusUpdate !== undefined) {
      obj.taskRunStatusUpdate = TaskRunLogEntry_TaskRunStatusUpdate.toJSON(message.taskRunStatusUpdate);
    }
    if (message.transactionControl !== undefined) {
      obj.transactionControl = TaskRunLogEntry_TransactionControl.toJSON(message.transactionControl);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry>): TaskRunLogEntry {
    return TaskRunLogEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry>): TaskRunLogEntry {
    const message = createBaseTaskRunLogEntry();
    message.type = object.type ?? TaskRunLogEntry_Type.TYPE_UNSPECIFIED;
    message.logTime = object.logTime ?? undefined;
    message.schemaDump = (object.schemaDump !== undefined && object.schemaDump !== null)
      ? TaskRunLogEntry_SchemaDump.fromPartial(object.schemaDump)
      : undefined;
    message.commandExecute = (object.commandExecute !== undefined && object.commandExecute !== null)
      ? TaskRunLogEntry_CommandExecute.fromPartial(object.commandExecute)
      : undefined;
    message.databaseSync = (object.databaseSync !== undefined && object.databaseSync !== null)
      ? TaskRunLogEntry_DatabaseSync.fromPartial(object.databaseSync)
      : undefined;
    message.taskRunStatusUpdate = (object.taskRunStatusUpdate !== undefined && object.taskRunStatusUpdate !== null)
      ? TaskRunLogEntry_TaskRunStatusUpdate.fromPartial(object.taskRunStatusUpdate)
      : undefined;
    message.transactionControl = (object.transactionControl !== undefined && object.transactionControl !== null)
      ? TaskRunLogEntry_TransactionControl.fromPartial(object.transactionControl)
      : undefined;
    return message;
  },
};

function createBaseTaskRunLogEntry_SchemaDump(): TaskRunLogEntry_SchemaDump {
  return { startTime: undefined, endTime: undefined, error: "" };
}

export const TaskRunLogEntry_SchemaDump = {
  encode(message: TaskRunLogEntry_SchemaDump, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.startTime !== undefined) {
      Timestamp.encode(toTimestamp(message.startTime), writer.uint32(10).fork()).ldelim();
    }
    if (message.endTime !== undefined) {
      Timestamp.encode(toTimestamp(message.endTime), writer.uint32(18).fork()).ldelim();
    }
    if (message.error !== "") {
      writer.uint32(26).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_SchemaDump {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_SchemaDump();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.startTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.endTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_SchemaDump {
    return {
      startTime: isSet(object.startTime) ? fromJsonTimestamp(object.startTime) : undefined,
      endTime: isSet(object.endTime) ? fromJsonTimestamp(object.endTime) : undefined,
      error: isSet(object.error) ? globalThis.String(object.error) : "",
    };
  },

  toJSON(message: TaskRunLogEntry_SchemaDump): unknown {
    const obj: any = {};
    if (message.startTime !== undefined) {
      obj.startTime = message.startTime.toISOString();
    }
    if (message.endTime !== undefined) {
      obj.endTime = message.endTime.toISOString();
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry_SchemaDump>): TaskRunLogEntry_SchemaDump {
    return TaskRunLogEntry_SchemaDump.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry_SchemaDump>): TaskRunLogEntry_SchemaDump {
    const message = createBaseTaskRunLogEntry_SchemaDump();
    message.startTime = object.startTime ?? undefined;
    message.endTime = object.endTime ?? undefined;
    message.error = object.error ?? "";
    return message;
  },
};

function createBaseTaskRunLogEntry_CommandExecute(): TaskRunLogEntry_CommandExecute {
  return { logTime: undefined, commandIndexes: [], response: undefined };
}

export const TaskRunLogEntry_CommandExecute = {
  encode(message: TaskRunLogEntry_CommandExecute, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.logTime !== undefined) {
      Timestamp.encode(toTimestamp(message.logTime), writer.uint32(10).fork()).ldelim();
    }
    writer.uint32(18).fork();
    for (const v of message.commandIndexes) {
      writer.int32(v);
    }
    writer.ldelim();
    if (message.response !== undefined) {
      TaskRunLogEntry_CommandExecute_CommandResponse.encode(message.response, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_CommandExecute {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_CommandExecute();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.logTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag === 16) {
            message.commandIndexes.push(reader.int32());

            continue;
          }

          if (tag === 18) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.commandIndexes.push(reader.int32());
            }

            continue;
          }

          break;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.response = TaskRunLogEntry_CommandExecute_CommandResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_CommandExecute {
    return {
      logTime: isSet(object.logTime) ? fromJsonTimestamp(object.logTime) : undefined,
      commandIndexes: globalThis.Array.isArray(object?.commandIndexes)
        ? object.commandIndexes.map((e: any) => globalThis.Number(e))
        : [],
      response: isSet(object.response)
        ? TaskRunLogEntry_CommandExecute_CommandResponse.fromJSON(object.response)
        : undefined,
    };
  },

  toJSON(message: TaskRunLogEntry_CommandExecute): unknown {
    const obj: any = {};
    if (message.logTime !== undefined) {
      obj.logTime = message.logTime.toISOString();
    }
    if (message.commandIndexes?.length) {
      obj.commandIndexes = message.commandIndexes.map((e) => Math.round(e));
    }
    if (message.response !== undefined) {
      obj.response = TaskRunLogEntry_CommandExecute_CommandResponse.toJSON(message.response);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry_CommandExecute>): TaskRunLogEntry_CommandExecute {
    return TaskRunLogEntry_CommandExecute.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry_CommandExecute>): TaskRunLogEntry_CommandExecute {
    const message = createBaseTaskRunLogEntry_CommandExecute();
    message.logTime = object.logTime ?? undefined;
    message.commandIndexes = object.commandIndexes?.map((e) => e) || [];
    message.response = (object.response !== undefined && object.response !== null)
      ? TaskRunLogEntry_CommandExecute_CommandResponse.fromPartial(object.response)
      : undefined;
    return message;
  },
};

function createBaseTaskRunLogEntry_CommandExecute_CommandResponse(): TaskRunLogEntry_CommandExecute_CommandResponse {
  return { logTime: undefined, error: "", affectedRows: 0, allAffectedRows: [] };
}

export const TaskRunLogEntry_CommandExecute_CommandResponse = {
  encode(
    message: TaskRunLogEntry_CommandExecute_CommandResponse,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.logTime !== undefined) {
      Timestamp.encode(toTimestamp(message.logTime), writer.uint32(10).fork()).ldelim();
    }
    if (message.error !== "") {
      writer.uint32(18).string(message.error);
    }
    if (message.affectedRows !== 0) {
      writer.uint32(24).int32(message.affectedRows);
    }
    writer.uint32(34).fork();
    for (const v of message.allAffectedRows) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_CommandExecute_CommandResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_CommandExecute_CommandResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.logTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.error = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.affectedRows = reader.int32();
          continue;
        case 4:
          if (tag === 32) {
            message.allAffectedRows.push(reader.int32());

            continue;
          }

          if (tag === 34) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.allAffectedRows.push(reader.int32());
            }

            continue;
          }

          break;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_CommandExecute_CommandResponse {
    return {
      logTime: isSet(object.logTime) ? fromJsonTimestamp(object.logTime) : undefined,
      error: isSet(object.error) ? globalThis.String(object.error) : "",
      affectedRows: isSet(object.affectedRows) ? globalThis.Number(object.affectedRows) : 0,
      allAffectedRows: globalThis.Array.isArray(object?.allAffectedRows)
        ? object.allAffectedRows.map((e: any) => globalThis.Number(e))
        : [],
    };
  },

  toJSON(message: TaskRunLogEntry_CommandExecute_CommandResponse): unknown {
    const obj: any = {};
    if (message.logTime !== undefined) {
      obj.logTime = message.logTime.toISOString();
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    if (message.affectedRows !== 0) {
      obj.affectedRows = Math.round(message.affectedRows);
    }
    if (message.allAffectedRows?.length) {
      obj.allAffectedRows = message.allAffectedRows.map((e) => Math.round(e));
    }
    return obj;
  },

  create(
    base?: DeepPartial<TaskRunLogEntry_CommandExecute_CommandResponse>,
  ): TaskRunLogEntry_CommandExecute_CommandResponse {
    return TaskRunLogEntry_CommandExecute_CommandResponse.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<TaskRunLogEntry_CommandExecute_CommandResponse>,
  ): TaskRunLogEntry_CommandExecute_CommandResponse {
    const message = createBaseTaskRunLogEntry_CommandExecute_CommandResponse();
    message.logTime = object.logTime ?? undefined;
    message.error = object.error ?? "";
    message.affectedRows = object.affectedRows ?? 0;
    message.allAffectedRows = object.allAffectedRows?.map((e) => e) || [];
    return message;
  },
};

function createBaseTaskRunLogEntry_DatabaseSync(): TaskRunLogEntry_DatabaseSync {
  return { startTime: undefined, endTime: undefined, error: "" };
}

export const TaskRunLogEntry_DatabaseSync = {
  encode(message: TaskRunLogEntry_DatabaseSync, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.startTime !== undefined) {
      Timestamp.encode(toTimestamp(message.startTime), writer.uint32(10).fork()).ldelim();
    }
    if (message.endTime !== undefined) {
      Timestamp.encode(toTimestamp(message.endTime), writer.uint32(18).fork()).ldelim();
    }
    if (message.error !== "") {
      writer.uint32(26).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_DatabaseSync {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_DatabaseSync();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.startTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.endTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_DatabaseSync {
    return {
      startTime: isSet(object.startTime) ? fromJsonTimestamp(object.startTime) : undefined,
      endTime: isSet(object.endTime) ? fromJsonTimestamp(object.endTime) : undefined,
      error: isSet(object.error) ? globalThis.String(object.error) : "",
    };
  },

  toJSON(message: TaskRunLogEntry_DatabaseSync): unknown {
    const obj: any = {};
    if (message.startTime !== undefined) {
      obj.startTime = message.startTime.toISOString();
    }
    if (message.endTime !== undefined) {
      obj.endTime = message.endTime.toISOString();
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry_DatabaseSync>): TaskRunLogEntry_DatabaseSync {
    return TaskRunLogEntry_DatabaseSync.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry_DatabaseSync>): TaskRunLogEntry_DatabaseSync {
    const message = createBaseTaskRunLogEntry_DatabaseSync();
    message.startTime = object.startTime ?? undefined;
    message.endTime = object.endTime ?? undefined;
    message.error = object.error ?? "";
    return message;
  },
};

function createBaseTaskRunLogEntry_TaskRunStatusUpdate(): TaskRunLogEntry_TaskRunStatusUpdate {
  return { status: TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED };
}

export const TaskRunLogEntry_TaskRunStatusUpdate = {
  encode(message: TaskRunLogEntry_TaskRunStatusUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED) {
      writer.uint32(8).int32(taskRunLogEntry_TaskRunStatusUpdate_StatusToNumber(message.status));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_TaskRunStatusUpdate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_TaskRunStatusUpdate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = taskRunLogEntry_TaskRunStatusUpdate_StatusFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_TaskRunStatusUpdate {
    return {
      status: isSet(object.status)
        ? taskRunLogEntry_TaskRunStatusUpdate_StatusFromJSON(object.status)
        : TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED,
    };
  },

  toJSON(message: TaskRunLogEntry_TaskRunStatusUpdate): unknown {
    const obj: any = {};
    if (message.status !== TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED) {
      obj.status = taskRunLogEntry_TaskRunStatusUpdate_StatusToJSON(message.status);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry_TaskRunStatusUpdate>): TaskRunLogEntry_TaskRunStatusUpdate {
    return TaskRunLogEntry_TaskRunStatusUpdate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry_TaskRunStatusUpdate>): TaskRunLogEntry_TaskRunStatusUpdate {
    const message = createBaseTaskRunLogEntry_TaskRunStatusUpdate();
    message.status = object.status ?? TaskRunLogEntry_TaskRunStatusUpdate_Status.STATUS_UNSPECIFIED;
    return message;
  },
};

function createBaseTaskRunLogEntry_TransactionControl(): TaskRunLogEntry_TransactionControl {
  return { type: TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED, error: "" };
}

export const TaskRunLogEntry_TransactionControl = {
  encode(message: TaskRunLogEntry_TransactionControl, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(taskRunLogEntry_TransactionControl_TypeToNumber(message.type));
    }
    if (message.error !== "") {
      writer.uint32(18).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLogEntry_TransactionControl {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLogEntry_TransactionControl();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = taskRunLogEntry_TransactionControl_TypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLogEntry_TransactionControl {
    return {
      type: isSet(object.type)
        ? taskRunLogEntry_TransactionControl_TypeFromJSON(object.type)
        : TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED,
      error: isSet(object.error) ? globalThis.String(object.error) : "",
    };
  },

  toJSON(message: TaskRunLogEntry_TransactionControl): unknown {
    const obj: any = {};
    if (message.type !== TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED) {
      obj.type = taskRunLogEntry_TransactionControl_TypeToJSON(message.type);
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLogEntry_TransactionControl>): TaskRunLogEntry_TransactionControl {
    return TaskRunLogEntry_TransactionControl.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLogEntry_TransactionControl>): TaskRunLogEntry_TransactionControl {
    const message = createBaseTaskRunLogEntry_TransactionControl();
    message.type = object.type ?? TaskRunLogEntry_TransactionControl_Type.TYPE_UNSPECIFIED;
    message.error = object.error ?? "";
    return message;
  },
};

export type RolloutServiceDefinition = typeof RolloutServiceDefinition;
export const RolloutServiceDefinition = {
  name: "RolloutService",
  fullName: "bytebase.v1.RolloutService",
  methods: {
    getRollout: {
      name: "GetRollout",
      requestType: GetRolloutRequest,
      requestStream: false,
      responseType: Rollout,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    createRollout: {
      name: "CreateRollout",
      requestType: CreateRolloutRequest,
      requestStream: false,
      responseType: Rollout,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([14, 112, 97, 114, 101, 110, 116, 44, 114, 111, 108, 108, 111, 117, 116])],
          578365826: [
            new Uint8Array([
              43,
              58,
              7,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              34,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
            ]),
          ],
        },
      },
    },
    previewRollout: {
      name: "PreviewRollout",
      requestType: PreviewRolloutRequest,
      requestStream: false,
      responseType: Rollout,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              44,
              58,
              1,
              42,
              34,
              39,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              58,
              112,
              114,
              101,
              118,
              105,
              101,
              119,
              82,
              111,
              108,
              108,
              111,
              117,
              116,
            ]),
          ],
        },
      },
    },
    listTaskRuns: {
      name: "ListTaskRuns",
      requestType: ListTaskRunsRequest,
      requestStream: false,
      responseType: ListTaskRunsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              62,
              18,
              60,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              47,
              115,
              116,
              97,
              103,
              101,
              115,
              47,
              42,
              47,
              116,
              97,
              115,
              107,
              115,
              47,
              42,
              125,
              47,
              116,
              97,
              115,
              107,
              82,
              117,
              110,
              115,
            ]),
          ],
        },
      },
    },
    getTaskRunLog: {
      name: "GetTaskRunLog",
      requestType: GetTaskRunLogRequest,
      requestStream: false,
      responseType: TaskRunLog,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              68,
              18,
              66,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              47,
              115,
              116,
              97,
              103,
              101,
              115,
              47,
              42,
              47,
              116,
              97,
              115,
              107,
              115,
              47,
              42,
              47,
              116,
              97,
              115,
              107,
              82,
              117,
              110,
              115,
              47,
              42,
              125,
              47,
              108,
              111,
              103,
            ]),
          ],
        },
      },
    },
    batchRunTasks: {
      name: "BatchRunTasks",
      requestType: BatchRunTasksRequest,
      requestStream: false,
      responseType: BatchRunTasksResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              63,
              58,
              1,
              42,
              34,
              58,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              47,
              115,
              116,
              97,
              103,
              101,
              115,
              47,
              42,
              125,
              47,
              116,
              97,
              115,
              107,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              82,
              117,
              110,
            ]),
          ],
        },
      },
    },
    batchSkipTasks: {
      name: "BatchSkipTasks",
      requestType: BatchSkipTasksRequest,
      requestStream: false,
      responseType: BatchSkipTasksResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              64,
              58,
              1,
              42,
              34,
              59,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              47,
              115,
              116,
              97,
              103,
              101,
              115,
              47,
              42,
              125,
              47,
              116,
              97,
              115,
              107,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              83,
              107,
              105,
              112,
            ]),
          ],
        },
      },
    },
    batchCancelTaskRuns: {
      name: "BatchCancelTaskRuns",
      requestType: BatchCancelTaskRunsRequest,
      requestStream: false,
      responseType: BatchCancelTaskRunsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              77,
              58,
              1,
              42,
              34,
              72,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              111,
              108,
              108,
              111,
              117,
              116,
              115,
              47,
              42,
              47,
              115,
              116,
              97,
              103,
              101,
              115,
              47,
              42,
              47,
              116,
              97,
              115,
              107,
              115,
              47,
              42,
              125,
              47,
              116,
              97,
              115,
              107,
              82,
              117,
              110,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              67,
              97,
              110,
              99,
              101,
              108,
            ]),
          ],
        },
      },
    },
  },
} as const;

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
