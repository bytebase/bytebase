/* eslint-disable */
import * as Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface GetPlanRequest {
  /**
   * The name of the plan to retrieve.
   * Format: projects/{project}/plans/{plan}
   */
  name: string;
}

export interface ListPlansRequest {
  /**
   * The parent, which owns this collection of plans.
   * Format: projects/{project}
   * Use "projects/-" to list all plans from all projects.
   */
  parent: string;
  /**
   * The maximum number of plans to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 plans will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListPlans` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListPlans` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListPlansResponse {
  /** The plans from the specified request. */
  plans: Plan[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreatePlanRequest {
  /**
   * The parent project where this plan will be created.
   * Format: projects/{project}
   */
  parent: string;
  /** The plan to create. */
  plan?: Plan | undefined;
}

export interface UpdatePlanRequest {
  /**
   * The plan to update.
   *
   * The plan's `name` field is used to identify the plan to update.
   * Format: projects/{project}/plans/{plan}
   */
  plan?:
    | Plan
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface Plan {
  /**
   * The name of the plan.
   * `plan` is a system generated ID.
   * Format: projects/{project}/plans/{plan}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  /**
   * The resource name of the issue associated with this plan.
   * Format: projects/{project}/issues/{issue}
   */
  issue: string;
  title: string;
  description: string;
  steps: Plan_Step[];
}

export interface Plan_Step {
  specs: Plan_Spec[];
}

export interface Plan_Spec {
  /** earliest_allowed_time the earliest execution time of the change. */
  earliestAllowedTime?:
    | Date
    | undefined;
  /** A UUID4 string that uniquely identifies the Spec. */
  id: string;
  createDatabaseConfig?: Plan_CreateDatabaseConfig | undefined;
  changeDatabaseConfig?: Plan_ChangeDatabaseConfig | undefined;
  restoreDatabaseConfig?: Plan_RestoreDatabaseConfig | undefined;
}

export interface Plan_CreateDatabaseConfig {
  /**
   * The resource name of the instance on which the database is created.
   * Format: instances/{instance}
   */
  target: string;
  /** The name of the database to create. */
  database: string;
  /**
   * table is the name of the table, if it is not empty, Bytebase should create a table after creating the database.
   * For example, in MongoDB, it only creates the database when we first store data in that database.
   */
  table: string;
  /** character_set is the character set of the database. */
  characterSet: string;
  /** collation is the collation of the database. */
  collation: string;
  /** cluster is the cluster of the database. This is only applicable to ClickHouse for "ON CLUSTER <<cluster>>". */
  cluster: string;
  /** owner is the owner of the database. This is only applicable to Postgres for "WITH OWNER <<owner>>". */
  owner: string;
  /**
   * backup is the resource name of the backup.
   * Format: instances/{instance}/databases/{database}/backups/{backup-name}
   */
  backup: string;
  /** labels of the database. */
  labels: { [key: string]: string };
}

export interface Plan_CreateDatabaseConfig_LabelsEntry {
  key: string;
  value: string;
}

export interface Plan_ChangeDatabaseConfig {
  /**
   * The resource name of the target.
   * Format: instances/{instance-id}/databases/{database-name}.
   */
  target: string;
  /**
   * The resource name of the sheet.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  type: Plan_ChangeDatabaseConfig_Type;
  /**
   * schema_version is parsed from VCS file name.
   * It is automatically generated in the UI workflow.
   */
  schemaVersion: string;
  /** If RollbackEnabled, build the RollbackSheetID of the task. */
  rollbackEnabled: boolean;
  rollbackDetail?: Plan_ChangeDatabaseConfig_RollbackDetail | undefined;
}

/** Type is the database change type. */
export enum Plan_ChangeDatabaseConfig_Type {
  TYPE_UNSPECIFIED = 0,
  /**
   * BASELINE - Used for establishing schema baseline, this is used when
   * 1. Onboard the database into Bytebase since Bytebase needs to know the current database schema.
   * 2. Had schema drift and need to re-establish the baseline.
   */
  BASELINE = 1,
  /** MIGRATE - Used for DDL changes including CREATE DATABASE. */
  MIGRATE = 2,
  /** MIGRATE_SDL - Used for schema changes via state-based schema migration including CREATE DATABASE. */
  MIGRATE_SDL = 3,
  /** MIGRATE_GHOST - Used for DDL changes using gh-ost. */
  MIGRATE_GHOST = 4,
  /** BRANCH - Used when restoring from a backup (the restored database branched from the original backup). */
  BRANCH = 5,
  /** DATA - Used for DML change. */
  DATA = 6,
  UNRECOGNIZED = -1,
}

export function plan_ChangeDatabaseConfig_TypeFromJSON(object: any): Plan_ChangeDatabaseConfig_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Plan_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED;
    case 1:
    case "BASELINE":
      return Plan_ChangeDatabaseConfig_Type.BASELINE;
    case 2:
    case "MIGRATE":
      return Plan_ChangeDatabaseConfig_Type.MIGRATE;
    case 3:
    case "MIGRATE_SDL":
      return Plan_ChangeDatabaseConfig_Type.MIGRATE_SDL;
    case 4:
    case "MIGRATE_GHOST":
      return Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST;
    case 5:
    case "BRANCH":
      return Plan_ChangeDatabaseConfig_Type.BRANCH;
    case 6:
    case "DATA":
      return Plan_ChangeDatabaseConfig_Type.DATA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Plan_ChangeDatabaseConfig_Type.UNRECOGNIZED;
  }
}

export function plan_ChangeDatabaseConfig_TypeToJSON(object: Plan_ChangeDatabaseConfig_Type): string {
  switch (object) {
    case Plan_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Plan_ChangeDatabaseConfig_Type.BASELINE:
      return "BASELINE";
    case Plan_ChangeDatabaseConfig_Type.MIGRATE:
      return "MIGRATE";
    case Plan_ChangeDatabaseConfig_Type.MIGRATE_SDL:
      return "MIGRATE_SDL";
    case Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
      return "MIGRATE_GHOST";
    case Plan_ChangeDatabaseConfig_Type.BRANCH:
      return "BRANCH";
    case Plan_ChangeDatabaseConfig_Type.DATA:
      return "DATA";
    case Plan_ChangeDatabaseConfig_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Plan_ChangeDatabaseConfig_RollbackDetail {
  /**
   * rollback_from_task is the task from which the rollback SQL statement is generated for this task.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  rollbackFromTask: string;
  /**
   * rollback_from_issue is the issue containing the original task from which the rollback SQL statement is generated for this task.
   * Format: projects/{project}/issues/{issue}
   */
  rollbackFromIssue: string;
}

export interface Plan_RestoreDatabaseConfig {
  /**
   * The resource name of the target to restore.
   * Format: instances/{instance}/databases/{database}
   */
  target: string;
  /** create_database_config is present if the user wants to restore to a new database. */
  createDatabaseConfig?: Plan_CreateDatabaseConfig | undefined;
  backup?:
    | string
    | undefined;
  /** After the PITR operations, the database will be recovered to the state at this time. */
  pointInTime?: Date | undefined;
}

export interface ListPlanCheckRunsRequest {
  /**
   * The parent, which owns this collection of plan check runs.
   * Format: projects/{project}/plans/{plan}
   */
  parent: string;
  /**
   * The maximum number of plan check runs to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 plans will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListPlanCheckRuns` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListPlanCheckRuns` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListPlanCheckRunsResponse {
  /** The plan check runs from the specified request. */
  planCheckRuns: PlanCheckRun[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

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
}

export interface BatchSkipTasksResponse {
}

export interface BatchCancelTaskRunsRequest {
  /**
   * The name of the parent of the taskRuns.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  parent: string;
  /**
   * The taskRuns to cancel.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
   */
  taskRuns: string[];
}

export interface BatchCancelTaskRunsResponse {
}

export interface PlanCheckRun {
  /** Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  type: PlanCheckRun_Type;
  status: PlanCheckRun_Status;
  /** Format: instances/{instance}/databases/{database} */
  target: string;
  /** Format: projects/{project}/sheets/{sheet} */
  sheet: string;
  results: PlanCheckRun_Result[];
  /** error is set if the Status is FAILED. */
  error: string;
}

export enum PlanCheckRun_Type {
  TYPE_UNSPECIFIED = 0,
  DATABASE_STATEMENT_FAKE_ADVISE = 1,
  DATABASE_STATEMENT_SYNTAX = 2,
  DATABASE_STATEMENT_COMPATIBILITY = 3,
  DATABASE_STATEMENT_ADVISE = 4,
  DATABASE_STATEMENT_TYPE = 5,
  DATABASE_STATEMENT_TYPE_REPORT = 6,
  DATABASE_STATEMENT_AFFECTED_ROWS_REPORT = 7,
  DATABASE_CONNECT = 8,
  DATABASE_GHOST_SYNC = 9,
  DATABASE_PITR_MYSQL = 10,
  UNRECOGNIZED = -1,
}

export function planCheckRun_TypeFromJSON(object: any): PlanCheckRun_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return PlanCheckRun_Type.TYPE_UNSPECIFIED;
    case 1:
    case "DATABASE_STATEMENT_FAKE_ADVISE":
      return PlanCheckRun_Type.DATABASE_STATEMENT_FAKE_ADVISE;
    case 2:
    case "DATABASE_STATEMENT_SYNTAX":
      return PlanCheckRun_Type.DATABASE_STATEMENT_SYNTAX;
    case 3:
    case "DATABASE_STATEMENT_COMPATIBILITY":
      return PlanCheckRun_Type.DATABASE_STATEMENT_COMPATIBILITY;
    case 4:
    case "DATABASE_STATEMENT_ADVISE":
      return PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE;
    case 5:
    case "DATABASE_STATEMENT_TYPE":
      return PlanCheckRun_Type.DATABASE_STATEMENT_TYPE;
    case 6:
    case "DATABASE_STATEMENT_TYPE_REPORT":
      return PlanCheckRun_Type.DATABASE_STATEMENT_TYPE_REPORT;
    case 7:
    case "DATABASE_STATEMENT_AFFECTED_ROWS_REPORT":
      return PlanCheckRun_Type.DATABASE_STATEMENT_AFFECTED_ROWS_REPORT;
    case 8:
    case "DATABASE_CONNECT":
      return PlanCheckRun_Type.DATABASE_CONNECT;
    case 9:
    case "DATABASE_GHOST_SYNC":
      return PlanCheckRun_Type.DATABASE_GHOST_SYNC;
    case 10:
    case "DATABASE_PITR_MYSQL":
      return PlanCheckRun_Type.DATABASE_PITR_MYSQL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRun_Type.UNRECOGNIZED;
  }
}

export function planCheckRun_TypeToJSON(object: PlanCheckRun_Type): string {
  switch (object) {
    case PlanCheckRun_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case PlanCheckRun_Type.DATABASE_STATEMENT_FAKE_ADVISE:
      return "DATABASE_STATEMENT_FAKE_ADVISE";
    case PlanCheckRun_Type.DATABASE_STATEMENT_SYNTAX:
      return "DATABASE_STATEMENT_SYNTAX";
    case PlanCheckRun_Type.DATABASE_STATEMENT_COMPATIBILITY:
      return "DATABASE_STATEMENT_COMPATIBILITY";
    case PlanCheckRun_Type.DATABASE_STATEMENT_ADVISE:
      return "DATABASE_STATEMENT_ADVISE";
    case PlanCheckRun_Type.DATABASE_STATEMENT_TYPE:
      return "DATABASE_STATEMENT_TYPE";
    case PlanCheckRun_Type.DATABASE_STATEMENT_TYPE_REPORT:
      return "DATABASE_STATEMENT_TYPE_REPORT";
    case PlanCheckRun_Type.DATABASE_STATEMENT_AFFECTED_ROWS_REPORT:
      return "DATABASE_STATEMENT_AFFECTED_ROWS_REPORT";
    case PlanCheckRun_Type.DATABASE_CONNECT:
      return "DATABASE_CONNECT";
    case PlanCheckRun_Type.DATABASE_GHOST_SYNC:
      return "DATABASE_GHOST_SYNC";
    case PlanCheckRun_Type.DATABASE_PITR_MYSQL:
      return "DATABASE_PITR_MYSQL";
    case PlanCheckRun_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum PlanCheckRun_Status {
  STATUS_UNSPECIFIED = 0,
  RUNNING = 1,
  DONE = 2,
  FAILED = 3,
  CANCELED = 4,
  UNRECOGNIZED = -1,
}

export function planCheckRun_StatusFromJSON(object: any): PlanCheckRun_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return PlanCheckRun_Status.STATUS_UNSPECIFIED;
    case 1:
    case "RUNNING":
      return PlanCheckRun_Status.RUNNING;
    case 2:
    case "DONE":
      return PlanCheckRun_Status.DONE;
    case 3:
    case "FAILED":
      return PlanCheckRun_Status.FAILED;
    case 4:
    case "CANCELED":
      return PlanCheckRun_Status.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRun_Status.UNRECOGNIZED;
  }
}

export function planCheckRun_StatusToJSON(object: PlanCheckRun_Status): string {
  switch (object) {
    case PlanCheckRun_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case PlanCheckRun_Status.RUNNING:
      return "RUNNING";
    case PlanCheckRun_Status.DONE:
      return "DONE";
    case PlanCheckRun_Status.FAILED:
      return "FAILED";
    case PlanCheckRun_Status.CANCELED:
      return "CANCELED";
    case PlanCheckRun_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanCheckRun_Result {
  status: PlanCheckRun_Result_Status;
  title: string;
  content: string;
  code: number;
  statementTypeReport?: PlanCheckRun_Result_StatementTypeReport | undefined;
  affectedRowsReport?: PlanCheckRun_Result_AffectedRowsReport | undefined;
  sqlReviewReport?: PlanCheckRun_Result_SqlReviewReport | undefined;
}

export enum PlanCheckRun_Result_Status {
  STATUS_UNSPECIFIED = 0,
  ERROR = 1,
  WARNING = 2,
  SUCCESS = 3,
  UNRECOGNIZED = -1,
}

export function planCheckRun_Result_StatusFromJSON(object: any): PlanCheckRun_Result_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;
    case 1:
    case "ERROR":
      return PlanCheckRun_Result_Status.ERROR;
    case 2:
    case "WARNING":
      return PlanCheckRun_Result_Status.WARNING;
    case 3:
    case "SUCCESS":
      return PlanCheckRun_Result_Status.SUCCESS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRun_Result_Status.UNRECOGNIZED;
  }
}

export function planCheckRun_Result_StatusToJSON(object: PlanCheckRun_Result_Status): string {
  switch (object) {
    case PlanCheckRun_Result_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case PlanCheckRun_Result_Status.ERROR:
      return "ERROR";
    case PlanCheckRun_Result_Status.WARNING:
      return "WARNING";
    case PlanCheckRun_Result_Status.SUCCESS:
      return "SUCCESS";
    case PlanCheckRun_Result_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanCheckRun_Result_StatementTypeReport {
  statementType: string;
}

export interface PlanCheckRun_Result_AffectedRowsReport {
  affectedRows: number;
}

export interface PlanCheckRun_Result_SqlReviewReport {
  line: number;
  detail: string;
  code: number;
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
  /** The plan used to create rollout. */
  plan: string;
}

export interface PreviewRolloutRequest {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  project: string;
  /** The plan used to preview rollout. */
  plan?: Plan | undefined;
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
  /** Format: environments/{environment} */
  environment: string;
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
  /**
   * Status is the status of the task.
   * TODO(p0ny): migrate old task status and use this field as a summary of the task runs.
   */
  status: Task_Status;
  type: Task_Type;
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} */
  blockedByTasks: string[];
  /**
   * Format: instances/{instance} if the task is DatabaseCreate.
   * Format: instances/{instance}/databases/{database}
   */
  target: string;
  databaseCreate?: Task_DatabaseCreate | undefined;
  databaseSchemaBaseline?: Task_DatabaseSchemaBaseline | undefined;
  databaseSchemaUpdate?: Task_DatabaseSchemaUpdate | undefined;
  databaseDataUpdate?: Task_DatabaseDataUpdate | undefined;
  databaseBackup?: Task_DatabaseBackup | undefined;
  databaseRestoreRestore?: Task_DatabaseRestoreRestore | undefined;
}

export enum Task_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING_APPROVAL = 1,
  PENDING = 2,
  RUNNING = 3,
  DONE = 4,
  FAILED = 5,
  CANCELED = 6,
  SKIPPED = 7,
  UNRECOGNIZED = -1,
}

export function task_StatusFromJSON(object: any): Task_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return Task_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING_APPROVAL":
      return Task_Status.PENDING_APPROVAL;
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
    case Task_Status.PENDING_APPROVAL:
      return "PENDING_APPROVAL";
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

export enum Task_Type {
  TYPE_UNSPECIFIED = 0,
  GENERAL = 1,
  /** DATABASE_CREATE - use payload DatabaseCreate */
  DATABASE_CREATE = 2,
  /** DATABASE_SCHEMA_BASELINE - use payload DatabaseSchemaBaseline */
  DATABASE_SCHEMA_BASELINE = 3,
  /** DATABASE_SCHEMA_UPDATE - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE = 4,
  /** DATABASE_SCHEMA_UPDATE_SDL - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE_SDL = 5,
  /** DATABASE_SCHEMA_UPDATE_GHOST_SYNC - use payload DatabaseSchemaUpdate */
  DATABASE_SCHEMA_UPDATE_GHOST_SYNC = 6,
  /** DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER - use payload nil */
  DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER = 7,
  /** DATABASE_DATA_UPDATE - use payload DatabaseDataUpdate */
  DATABASE_DATA_UPDATE = 8,
  /** DATABASE_BACKUP - use payload DatabaseBackup */
  DATABASE_BACKUP = 9,
  /** DATABASE_RESTORE_RESTORE - use payload DatabaseRestoreRestore */
  DATABASE_RESTORE_RESTORE = 10,
  /** DATABASE_RESTORE_CUTOVER - use payload nil */
  DATABASE_RESTORE_CUTOVER = 11,
  UNRECOGNIZED = -1,
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
    case 9:
    case "DATABASE_BACKUP":
      return Task_Type.DATABASE_BACKUP;
    case 10:
    case "DATABASE_RESTORE_RESTORE":
      return Task_Type.DATABASE_RESTORE_RESTORE;
    case 11:
    case "DATABASE_RESTORE_CUTOVER":
      return Task_Type.DATABASE_RESTORE_CUTOVER;
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
    case Task_Type.DATABASE_BACKUP:
      return "DATABASE_BACKUP";
    case Task_Type.DATABASE_RESTORE_RESTORE:
      return "DATABASE_RESTORE_RESTORE";
    case Task_Type.DATABASE_RESTORE_CUTOVER:
      return "DATABASE_RESTORE_CUTOVER";
    case Task_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
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
  /** Build the rollback SQL if rollback_enabled. */
  rollbackEnabled: boolean;
  /** The status of the rollback SQL generation. */
  rollbackSqlStatus: Task_DatabaseDataUpdate_RollbackSqlStatus;
  rollbackError: string;
  /**
   * rollback_sheet is the resource name of
   * the sheet that stores the generated rollback SQL statement.
   * Format: projects/{project}/sheets/{sheet}
   */
  rollbackSheet: string;
  /**
   * rollback_from_issue is the resource name of the issue that
   * the rollback SQL statement is generated from.
   * Format: projects/{project}/issues/{issue}
   */
  rollbackFromIssue: string;
  /**
   * rollback_from_task is the resource name of the task that
   * the rollback SQL statement is generated from.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  rollbackFromTask: string;
}

export enum Task_DatabaseDataUpdate_RollbackSqlStatus {
  ROLLBACK_SQL_STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  DONE = 2,
  FAILED = 3,
  UNRECOGNIZED = -1,
}

export function task_DatabaseDataUpdate_RollbackSqlStatusFromJSON(
  object: any,
): Task_DatabaseDataUpdate_RollbackSqlStatus {
  switch (object) {
    case 0:
    case "ROLLBACK_SQL_STATUS_UNSPECIFIED":
      return Task_DatabaseDataUpdate_RollbackSqlStatus.ROLLBACK_SQL_STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return Task_DatabaseDataUpdate_RollbackSqlStatus.PENDING;
    case 2:
    case "DONE":
      return Task_DatabaseDataUpdate_RollbackSqlStatus.DONE;
    case 3:
    case "FAILED":
      return Task_DatabaseDataUpdate_RollbackSqlStatus.FAILED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Task_DatabaseDataUpdate_RollbackSqlStatus.UNRECOGNIZED;
  }
}

export function task_DatabaseDataUpdate_RollbackSqlStatusToJSON(
  object: Task_DatabaseDataUpdate_RollbackSqlStatus,
): string {
  switch (object) {
    case Task_DatabaseDataUpdate_RollbackSqlStatus.ROLLBACK_SQL_STATUS_UNSPECIFIED:
      return "ROLLBACK_SQL_STATUS_UNSPECIFIED";
    case Task_DatabaseDataUpdate_RollbackSqlStatus.PENDING:
      return "PENDING";
    case Task_DatabaseDataUpdate_RollbackSqlStatus.DONE:
      return "DONE";
    case Task_DatabaseDataUpdate_RollbackSqlStatus.FAILED:
      return "FAILED";
    case Task_DatabaseDataUpdate_RollbackSqlStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Task_DatabaseBackup {
  /**
   * The resource name of the backup.
   * Format: instances/{instance}/databases/{database}/backups/{backup-name}
   */
  backup: string;
}

export interface Task_DatabaseRestoreRestore {
  /**
   * Target is only used when doing restore to a new database now.
   * It is empty for the case of in-place restore.
   * Target {instance} must be within the same environment as the instance of the original database.
   * {database} is the target database name.
   * Format: instances/{instance}/databases/database
   */
  target: string;
  /**
   * Only used when doing restore full backup only.
   * Format: instances/{instance}/databases/{database}/backups/{backup-name}
   */
  backup?:
    | string
    | undefined;
  /** After the PITR operations, the database will be recovered to the state at this time. */
  pointInTime?: Date | undefined;
}

export interface TaskRun {
  /** Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskrun} */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  /** Format: user:hello@world.com */
  creator: string;
  /** Format: user:hello@world.com */
  updater: string;
  createTime?: Date | undefined;
  updateTime?: Date | undefined;
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
}

export enum TaskRun_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  RUNNING = 2,
  DONE = 3,
  FAILED = 4,
  CANCELED = 5,
  SKIPPED = 6,
  UNRECOGNIZED = -1,
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
    case 6:
    case "SKIPPED":
      return TaskRun_Status.SKIPPED;
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
    case TaskRun_Status.SKIPPED:
      return "SKIPPED";
    case TaskRun_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseGetPlanRequest(): GetPlanRequest {
  return { name: "" };
}

export const GetPlanRequest = {
  encode(message: GetPlanRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetPlanRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetPlanRequest();
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

  fromJSON(object: any): GetPlanRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetPlanRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetPlanRequest>): GetPlanRequest {
    return GetPlanRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetPlanRequest>): GetPlanRequest {
    const message = createBaseGetPlanRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPlansRequest(): ListPlansRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListPlansRequest = {
  encode(message: ListPlansRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlansRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlansRequest();
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

  fromJSON(object: any): ListPlansRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListPlansRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlansRequest>): ListPlansRequest {
    return ListPlansRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlansRequest>): ListPlansRequest {
    const message = createBaseListPlansRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListPlansResponse(): ListPlansResponse {
  return { plans: [], nextPageToken: "" };
}

export const ListPlansResponse = {
  encode(message: ListPlansResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.plans) {
      Plan.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlansResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlansResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.plans.push(Plan.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListPlansResponse {
    return {
      plans: Array.isArray(object?.plans) ? object.plans.map((e: any) => Plan.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPlansResponse): unknown {
    const obj: any = {};
    if (message.plans) {
      obj.plans = message.plans.map((e) => e ? Plan.toJSON(e) : undefined);
    } else {
      obj.plans = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlansResponse>): ListPlansResponse {
    return ListPlansResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlansResponse>): ListPlansResponse {
    const message = createBaseListPlansResponse();
    message.plans = object.plans?.map((e) => Plan.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreatePlanRequest(): CreatePlanRequest {
  return { parent: "", plan: undefined };
}

export const CreatePlanRequest = {
  encode(message: CreatePlanRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.plan !== undefined) {
      Plan.encode(message.plan, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreatePlanRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreatePlanRequest();
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

  fromJSON(object: any): CreatePlanRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      plan: isSet(object.plan) ? Plan.fromJSON(object.plan) : undefined,
    };
  },

  toJSON(message: CreatePlanRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.plan !== undefined && (obj.plan = message.plan ? Plan.toJSON(message.plan) : undefined);
    return obj;
  },

  create(base?: DeepPartial<CreatePlanRequest>): CreatePlanRequest {
    return CreatePlanRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreatePlanRequest>): CreatePlanRequest {
    const message = createBaseCreatePlanRequest();
    message.parent = object.parent ?? "";
    message.plan = (object.plan !== undefined && object.plan !== null) ? Plan.fromPartial(object.plan) : undefined;
    return message;
  },
};

function createBaseUpdatePlanRequest(): UpdatePlanRequest {
  return { plan: undefined, updateMask: undefined };
}

export const UpdatePlanRequest = {
  encode(message: UpdatePlanRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.plan !== undefined) {
      Plan.encode(message.plan, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdatePlanRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdatePlanRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.plan = Plan.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdatePlanRequest {
    return {
      plan: isSet(object.plan) ? Plan.fromJSON(object.plan) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdatePlanRequest): unknown {
    const obj: any = {};
    message.plan !== undefined && (obj.plan = message.plan ? Plan.toJSON(message.plan) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdatePlanRequest>): UpdatePlanRequest {
    return UpdatePlanRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdatePlanRequest>): UpdatePlanRequest {
    const message = createBaseUpdatePlanRequest();
    message.plan = (object.plan !== undefined && object.plan !== null) ? Plan.fromPartial(object.plan) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBasePlan(): Plan {
  return { name: "", uid: "", issue: "", title: "", description: "", steps: [] };
}

export const Plan = {
  encode(message: Plan, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.issue !== "") {
      writer.uint32(26).string(message.issue);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    for (const v of message.steps) {
      Plan_Step.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan();
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

          message.issue = reader.string();
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

          message.description = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.steps.push(Plan_Step.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      issue: isSet(object.issue) ? String(object.issue) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => Plan_Step.fromJSON(e)) : [],
    };
  },

  toJSON(message: Plan): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.issue !== undefined && (obj.issue = message.issue);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    if (message.steps) {
      obj.steps = message.steps.map((e) => e ? Plan_Step.toJSON(e) : undefined);
    } else {
      obj.steps = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Plan>): Plan {
    return Plan.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan>): Plan {
    const message = createBasePlan();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.issue = object.issue ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.steps = object.steps?.map((e) => Plan_Step.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlan_Step(): Plan_Step {
  return { specs: [] };
}

export const Plan_Step = {
  encode(message: Plan_Step, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.specs) {
      Plan_Spec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_Step {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_Step();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.specs.push(Plan_Spec.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan_Step {
    return { specs: Array.isArray(object?.specs) ? object.specs.map((e: any) => Plan_Spec.fromJSON(e)) : [] };
  },

  toJSON(message: Plan_Step): unknown {
    const obj: any = {};
    if (message.specs) {
      obj.specs = message.specs.map((e) => e ? Plan_Spec.toJSON(e) : undefined);
    } else {
      obj.specs = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Plan_Step>): Plan_Step {
    return Plan_Step.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_Step>): Plan_Step {
    const message = createBasePlan_Step();
    message.specs = object.specs?.map((e) => Plan_Spec.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlan_Spec(): Plan_Spec {
  return {
    earliestAllowedTime: undefined,
    id: "",
    createDatabaseConfig: undefined,
    changeDatabaseConfig: undefined,
    restoreDatabaseConfig: undefined,
  };
}

export const Plan_Spec = {
  encode(message: Plan_Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.earliestAllowedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.earliestAllowedTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.id !== "") {
      writer.uint32(42).string(message.id);
    }
    if (message.createDatabaseConfig !== undefined) {
      Plan_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(10).fork()).ldelim();
    }
    if (message.changeDatabaseConfig !== undefined) {
      Plan_ChangeDatabaseConfig.encode(message.changeDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.restoreDatabaseConfig !== undefined) {
      Plan_RestoreDatabaseConfig.encode(message.restoreDatabaseConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_Spec {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_Spec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 4:
          if (tag !== 34) {
            break;
          }

          message.earliestAllowedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.id = reader.string();
          continue;
        case 1:
          if (tag !== 10) {
            break;
          }

          message.createDatabaseConfig = Plan_CreateDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changeDatabaseConfig = Plan_ChangeDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.restoreDatabaseConfig = Plan_RestoreDatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan_Spec {
    return {
      earliestAllowedTime: isSet(object.earliestAllowedTime)
        ? fromJsonTimestamp(object.earliestAllowedTime)
        : undefined,
      id: isSet(object.id) ? String(object.id) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? Plan_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      changeDatabaseConfig: isSet(object.changeDatabaseConfig)
        ? Plan_ChangeDatabaseConfig.fromJSON(object.changeDatabaseConfig)
        : undefined,
      restoreDatabaseConfig: isSet(object.restoreDatabaseConfig)
        ? Plan_RestoreDatabaseConfig.fromJSON(object.restoreDatabaseConfig)
        : undefined,
    };
  },

  toJSON(message: Plan_Spec): unknown {
    const obj: any = {};
    message.earliestAllowedTime !== undefined && (obj.earliestAllowedTime = message.earliestAllowedTime.toISOString());
    message.id !== undefined && (obj.id = message.id);
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? Plan_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.changeDatabaseConfig !== undefined && (obj.changeDatabaseConfig = message.changeDatabaseConfig
      ? Plan_ChangeDatabaseConfig.toJSON(message.changeDatabaseConfig)
      : undefined);
    message.restoreDatabaseConfig !== undefined && (obj.restoreDatabaseConfig = message.restoreDatabaseConfig
      ? Plan_RestoreDatabaseConfig.toJSON(message.restoreDatabaseConfig)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<Plan_Spec>): Plan_Spec {
    return Plan_Spec.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_Spec>): Plan_Spec {
    const message = createBasePlan_Spec();
    message.earliestAllowedTime = object.earliestAllowedTime ?? undefined;
    message.id = object.id ?? "";
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? Plan_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
      : undefined;
    message.changeDatabaseConfig = (object.changeDatabaseConfig !== undefined && object.changeDatabaseConfig !== null)
      ? Plan_ChangeDatabaseConfig.fromPartial(object.changeDatabaseConfig)
      : undefined;
    message.restoreDatabaseConfig =
      (object.restoreDatabaseConfig !== undefined && object.restoreDatabaseConfig !== null)
        ? Plan_RestoreDatabaseConfig.fromPartial(object.restoreDatabaseConfig)
        : undefined;
    return message;
  },
};

function createBasePlan_CreateDatabaseConfig(): Plan_CreateDatabaseConfig {
  return {
    target: "",
    database: "",
    table: "",
    characterSet: "",
    collation: "",
    cluster: "",
    owner: "",
    backup: "",
    labels: {},
  };
}

export const Plan_CreateDatabaseConfig = {
  encode(message: Plan_CreateDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.database !== "") {
      writer.uint32(18).string(message.database);
    }
    if (message.table !== "") {
      writer.uint32(26).string(message.table);
    }
    if (message.characterSet !== "") {
      writer.uint32(34).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(42).string(message.collation);
    }
    if (message.cluster !== "") {
      writer.uint32(50).string(message.cluster);
    }
    if (message.owner !== "") {
      writer.uint32(58).string(message.owner);
    }
    if (message.backup !== "") {
      writer.uint32(66).string(message.backup);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      Plan_CreateDatabaseConfig_LabelsEntry.encode({ key: key as any, value }, writer.uint32(74).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_CreateDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_CreateDatabaseConfig();
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

          message.characterSet = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.cluster = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.owner = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.backup = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          const entry9 = Plan_CreateDatabaseConfig_LabelsEntry.decode(reader, reader.uint32());
          if (entry9.value !== undefined) {
            message.labels[entry9.key] = entry9.value;
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

  fromJSON(object: any): Plan_CreateDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      database: isSet(object.database) ? String(object.database) : "",
      table: isSet(object.table) ? String(object.table) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      cluster: isSet(object.cluster) ? String(object.cluster) : "",
      owner: isSet(object.owner) ? String(object.owner) : "",
      backup: isSet(object.backup) ? String(object.backup) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Plan_CreateDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.database !== undefined && (obj.database = message.database);
    message.table !== undefined && (obj.table = message.table);
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    message.cluster !== undefined && (obj.cluster = message.cluster);
    message.owner !== undefined && (obj.owner = message.owner);
    message.backup !== undefined && (obj.backup = message.backup);
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
    }
    return obj;
  },

  create(base?: DeepPartial<Plan_CreateDatabaseConfig>): Plan_CreateDatabaseConfig {
    return Plan_CreateDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_CreateDatabaseConfig>): Plan_CreateDatabaseConfig {
    const message = createBasePlan_CreateDatabaseConfig();
    message.target = object.target ?? "";
    message.database = object.database ?? "";
    message.table = object.table ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.cluster = object.cluster ?? "";
    message.owner = object.owner ?? "";
    message.backup = object.backup ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBasePlan_CreateDatabaseConfig_LabelsEntry(): Plan_CreateDatabaseConfig_LabelsEntry {
  return { key: "", value: "" };
}

export const Plan_CreateDatabaseConfig_LabelsEntry = {
  encode(message: Plan_CreateDatabaseConfig_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_CreateDatabaseConfig_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_CreateDatabaseConfig_LabelsEntry();
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

  fromJSON(object: any): Plan_CreateDatabaseConfig_LabelsEntry {
    return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
  },

  toJSON(message: Plan_CreateDatabaseConfig_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  create(base?: DeepPartial<Plan_CreateDatabaseConfig_LabelsEntry>): Plan_CreateDatabaseConfig_LabelsEntry {
    return Plan_CreateDatabaseConfig_LabelsEntry.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_CreateDatabaseConfig_LabelsEntry>): Plan_CreateDatabaseConfig_LabelsEntry {
    const message = createBasePlan_CreateDatabaseConfig_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBasePlan_ChangeDatabaseConfig(): Plan_ChangeDatabaseConfig {
  return { target: "", sheet: "", type: 0, schemaVersion: "", rollbackEnabled: false, rollbackDetail: undefined };
}

export const Plan_ChangeDatabaseConfig = {
  encode(message: Plan_ChangeDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.sheet !== "") {
      writer.uint32(18).string(message.sheet);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(34).string(message.schemaVersion);
    }
    if (message.rollbackEnabled === true) {
      writer.uint32(40).bool(message.rollbackEnabled);
    }
    if (message.rollbackDetail !== undefined) {
      Plan_ChangeDatabaseConfig_RollbackDetail.encode(message.rollbackDetail, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_ChangeDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_ChangeDatabaseConfig();
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

          message.type = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.rollbackEnabled = reader.bool();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.rollbackDetail = Plan_ChangeDatabaseConfig_RollbackDetail.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan_ChangeDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      type: isSet(object.type) ? plan_ChangeDatabaseConfig_TypeFromJSON(object.type) : 0,
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      rollbackEnabled: isSet(object.rollbackEnabled) ? Boolean(object.rollbackEnabled) : false,
      rollbackDetail: isSet(object.rollbackDetail)
        ? Plan_ChangeDatabaseConfig_RollbackDetail.fromJSON(object.rollbackDetail)
        : undefined,
    };
  },

  toJSON(message: Plan_ChangeDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.type !== undefined && (obj.type = plan_ChangeDatabaseConfig_TypeToJSON(message.type));
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    message.rollbackEnabled !== undefined && (obj.rollbackEnabled = message.rollbackEnabled);
    message.rollbackDetail !== undefined && (obj.rollbackDetail = message.rollbackDetail
      ? Plan_ChangeDatabaseConfig_RollbackDetail.toJSON(message.rollbackDetail)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<Plan_ChangeDatabaseConfig>): Plan_ChangeDatabaseConfig {
    return Plan_ChangeDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_ChangeDatabaseConfig>): Plan_ChangeDatabaseConfig {
    const message = createBasePlan_ChangeDatabaseConfig();
    message.target = object.target ?? "";
    message.sheet = object.sheet ?? "";
    message.type = object.type ?? 0;
    message.schemaVersion = object.schemaVersion ?? "";
    message.rollbackEnabled = object.rollbackEnabled ?? false;
    message.rollbackDetail = (object.rollbackDetail !== undefined && object.rollbackDetail !== null)
      ? Plan_ChangeDatabaseConfig_RollbackDetail.fromPartial(object.rollbackDetail)
      : undefined;
    return message;
  },
};

function createBasePlan_ChangeDatabaseConfig_RollbackDetail(): Plan_ChangeDatabaseConfig_RollbackDetail {
  return { rollbackFromTask: "", rollbackFromIssue: "" };
}

export const Plan_ChangeDatabaseConfig_RollbackDetail = {
  encode(message: Plan_ChangeDatabaseConfig_RollbackDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.rollbackFromTask !== "") {
      writer.uint32(10).string(message.rollbackFromTask);
    }
    if (message.rollbackFromIssue !== "") {
      writer.uint32(18).string(message.rollbackFromIssue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_ChangeDatabaseConfig_RollbackDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_ChangeDatabaseConfig_RollbackDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.rollbackFromTask = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.rollbackFromIssue = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan_ChangeDatabaseConfig_RollbackDetail {
    return {
      rollbackFromTask: isSet(object.rollbackFromTask) ? String(object.rollbackFromTask) : "",
      rollbackFromIssue: isSet(object.rollbackFromIssue) ? String(object.rollbackFromIssue) : "",
    };
  },

  toJSON(message: Plan_ChangeDatabaseConfig_RollbackDetail): unknown {
    const obj: any = {};
    message.rollbackFromTask !== undefined && (obj.rollbackFromTask = message.rollbackFromTask);
    message.rollbackFromIssue !== undefined && (obj.rollbackFromIssue = message.rollbackFromIssue);
    return obj;
  },

  create(base?: DeepPartial<Plan_ChangeDatabaseConfig_RollbackDetail>): Plan_ChangeDatabaseConfig_RollbackDetail {
    return Plan_ChangeDatabaseConfig_RollbackDetail.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_ChangeDatabaseConfig_RollbackDetail>): Plan_ChangeDatabaseConfig_RollbackDetail {
    const message = createBasePlan_ChangeDatabaseConfig_RollbackDetail();
    message.rollbackFromTask = object.rollbackFromTask ?? "";
    message.rollbackFromIssue = object.rollbackFromIssue ?? "";
    return message;
  },
};

function createBasePlan_RestoreDatabaseConfig(): Plan_RestoreDatabaseConfig {
  return { target: "", createDatabaseConfig: undefined, backup: undefined, pointInTime: undefined };
}

export const Plan_RestoreDatabaseConfig = {
  encode(message: Plan_RestoreDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.createDatabaseConfig !== undefined) {
      Plan_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.backup !== undefined) {
      writer.uint32(26).string(message.backup);
    }
    if (message.pointInTime !== undefined) {
      Timestamp.encode(toTimestamp(message.pointInTime), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan_RestoreDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan_RestoreDatabaseConfig();
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

          message.createDatabaseConfig = Plan_CreateDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.backup = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pointInTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan_RestoreDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? Plan_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      backup: isSet(object.backup) ? String(object.backup) : undefined,
      pointInTime: isSet(object.pointInTime) ? fromJsonTimestamp(object.pointInTime) : undefined,
    };
  },

  toJSON(message: Plan_RestoreDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? Plan_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.backup !== undefined && (obj.backup = message.backup);
    message.pointInTime !== undefined && (obj.pointInTime = message.pointInTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<Plan_RestoreDatabaseConfig>): Plan_RestoreDatabaseConfig {
    return Plan_RestoreDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan_RestoreDatabaseConfig>): Plan_RestoreDatabaseConfig {
    const message = createBasePlan_RestoreDatabaseConfig();
    message.target = object.target ?? "";
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? Plan_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
      : undefined;
    message.backup = object.backup ?? undefined;
    message.pointInTime = object.pointInTime ?? undefined;
    return message;
  },
};

function createBaseListPlanCheckRunsRequest(): ListPlanCheckRunsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListPlanCheckRunsRequest = {
  encode(message: ListPlanCheckRunsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlanCheckRunsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlanCheckRunsRequest();
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

  fromJSON(object: any): ListPlanCheckRunsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListPlanCheckRunsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlanCheckRunsRequest>): ListPlanCheckRunsRequest {
    return ListPlanCheckRunsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlanCheckRunsRequest>): ListPlanCheckRunsRequest {
    const message = createBaseListPlanCheckRunsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListPlanCheckRunsResponse(): ListPlanCheckRunsResponse {
  return { planCheckRuns: [], nextPageToken: "" };
}

export const ListPlanCheckRunsResponse = {
  encode(message: ListPlanCheckRunsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.planCheckRuns) {
      PlanCheckRun.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlanCheckRunsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlanCheckRunsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.planCheckRuns.push(PlanCheckRun.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListPlanCheckRunsResponse {
    return {
      planCheckRuns: Array.isArray(object?.planCheckRuns)
        ? object.planCheckRuns.map((e: any) => PlanCheckRun.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPlanCheckRunsResponse): unknown {
    const obj: any = {};
    if (message.planCheckRuns) {
      obj.planCheckRuns = message.planCheckRuns.map((e) => e ? PlanCheckRun.toJSON(e) : undefined);
    } else {
      obj.planCheckRuns = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlanCheckRunsResponse>): ListPlanCheckRunsResponse {
    return ListPlanCheckRunsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlanCheckRunsResponse>): ListPlanCheckRunsResponse {
    const message = createBaseListPlanCheckRunsResponse();
    message.planCheckRuns = object.planCheckRuns?.map((e) => PlanCheckRun.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseBatchRunTasksRequest(): BatchRunTasksRequest {
  return { parent: "", tasks: [] };
}

export const BatchRunTasksRequest = {
  encode(message: BatchRunTasksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.tasks) {
      writer.uint32(18).string(v!);
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      tasks: Array.isArray(object?.tasks) ? object.tasks.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: BatchRunTasksRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.tasks) {
      obj.tasks = message.tasks.map((e) => e);
    } else {
      obj.tasks = [];
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
  return { parent: "", tasks: [] };
}

export const BatchSkipTasksRequest = {
  encode(message: BatchSkipTasksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.tasks) {
      writer.uint32(18).string(v!);
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      tasks: Array.isArray(object?.tasks) ? object.tasks.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: BatchSkipTasksRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.tasks) {
      obj.tasks = message.tasks.map((e) => e);
    } else {
      obj.tasks = [];
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
  return { parent: "", taskRuns: [] };
}

export const BatchCancelTaskRunsRequest = {
  encode(message: BatchCancelTaskRunsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.taskRuns) {
      writer.uint32(18).string(v!);
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      taskRuns: Array.isArray(object?.taskRuns) ? object.taskRuns.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: BatchCancelTaskRunsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.taskRuns) {
      obj.taskRuns = message.taskRuns.map((e) => e);
    } else {
      obj.taskRuns = [];
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

function createBasePlanCheckRun(): PlanCheckRun {
  return { name: "", uid: "", type: 0, status: 0, target: "", sheet: "", results: [], error: "" };
}

export const PlanCheckRun = {
  encode(message: PlanCheckRun, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.status !== 0) {
      writer.uint32(32).int32(message.status);
    }
    if (message.target !== "") {
      writer.uint32(42).string(message.target);
    }
    if (message.sheet !== "") {
      writer.uint32(50).string(message.sheet);
    }
    for (const v of message.results) {
      PlanCheckRun_Result.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.error !== "") {
      writer.uint32(66).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRun {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRun();
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
          if (tag !== 24) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.target = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.results.push(PlanCheckRun_Result.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag !== 66) {
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

  fromJSON(object: any): PlanCheckRun {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      type: isSet(object.type) ? planCheckRun_TypeFromJSON(object.type) : 0,
      status: isSet(object.status) ? planCheckRun_StatusFromJSON(object.status) : 0,
      target: isSet(object.target) ? String(object.target) : "",
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      results: Array.isArray(object?.results) ? object.results.map((e: any) => PlanCheckRun_Result.fromJSON(e)) : [],
      error: isSet(object.error) ? String(object.error) : "",
    };
  },

  toJSON(message: PlanCheckRun): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.type !== undefined && (obj.type = planCheckRun_TypeToJSON(message.type));
    message.status !== undefined && (obj.status = planCheckRun_StatusToJSON(message.status));
    message.target !== undefined && (obj.target = message.target);
    message.sheet !== undefined && (obj.sheet = message.sheet);
    if (message.results) {
      obj.results = message.results.map((e) => e ? PlanCheckRun_Result.toJSON(e) : undefined);
    } else {
      obj.results = [];
    }
    message.error !== undefined && (obj.error = message.error);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRun>): PlanCheckRun {
    return PlanCheckRun.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRun>): PlanCheckRun {
    const message = createBasePlanCheckRun();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.type = object.type ?? 0;
    message.status = object.status ?? 0;
    message.target = object.target ?? "";
    message.sheet = object.sheet ?? "";
    message.results = object.results?.map((e) => PlanCheckRun_Result.fromPartial(e)) || [];
    message.error = object.error ?? "";
    return message;
  },
};

function createBasePlanCheckRun_Result(): PlanCheckRun_Result {
  return {
    status: 0,
    title: "",
    content: "",
    code: 0,
    statementTypeReport: undefined,
    affectedRowsReport: undefined,
    sqlReviewReport: undefined,
  };
}

export const PlanCheckRun_Result = {
  encode(message: PlanCheckRun_Result, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(26).string(message.content);
    }
    if (message.code !== 0) {
      writer.uint32(32).int64(message.code);
    }
    if (message.statementTypeReport !== undefined) {
      PlanCheckRun_Result_StatementTypeReport.encode(message.statementTypeReport, writer.uint32(42).fork()).ldelim();
    }
    if (message.affectedRowsReport !== undefined) {
      PlanCheckRun_Result_AffectedRowsReport.encode(message.affectedRowsReport, writer.uint32(50).fork()).ldelim();
    }
    if (message.sqlReviewReport !== undefined) {
      PlanCheckRun_Result_SqlReviewReport.encode(message.sqlReviewReport, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRun_Result {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRun_Result();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.content = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.code = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.statementTypeReport = PlanCheckRun_Result_StatementTypeReport.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.affectedRowsReport = PlanCheckRun_Result_AffectedRowsReport.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.sqlReviewReport = PlanCheckRun_Result_SqlReviewReport.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRun_Result {
    return {
      status: isSet(object.status) ? planCheckRun_Result_StatusFromJSON(object.status) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
      statementTypeReport: isSet(object.statementTypeReport)
        ? PlanCheckRun_Result_StatementTypeReport.fromJSON(object.statementTypeReport)
        : undefined,
      affectedRowsReport: isSet(object.affectedRowsReport)
        ? PlanCheckRun_Result_AffectedRowsReport.fromJSON(object.affectedRowsReport)
        : undefined,
      sqlReviewReport: isSet(object.sqlReviewReport)
        ? PlanCheckRun_Result_SqlReviewReport.fromJSON(object.sqlReviewReport)
        : undefined,
    };
  },

  toJSON(message: PlanCheckRun_Result): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = planCheckRun_Result_StatusToJSON(message.status));
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.statementTypeReport !== undefined && (obj.statementTypeReport = message.statementTypeReport
      ? PlanCheckRun_Result_StatementTypeReport.toJSON(message.statementTypeReport)
      : undefined);
    message.affectedRowsReport !== undefined && (obj.affectedRowsReport = message.affectedRowsReport
      ? PlanCheckRun_Result_AffectedRowsReport.toJSON(message.affectedRowsReport)
      : undefined);
    message.sqlReviewReport !== undefined && (obj.sqlReviewReport = message.sqlReviewReport
      ? PlanCheckRun_Result_SqlReviewReport.toJSON(message.sqlReviewReport)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRun_Result>): PlanCheckRun_Result {
    return PlanCheckRun_Result.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRun_Result>): PlanCheckRun_Result {
    const message = createBasePlanCheckRun_Result();
    message.status = object.status ?? 0;
    message.title = object.title ?? "";
    message.content = object.content ?? "";
    message.code = object.code ?? 0;
    message.statementTypeReport = (object.statementTypeReport !== undefined && object.statementTypeReport !== null)
      ? PlanCheckRun_Result_StatementTypeReport.fromPartial(object.statementTypeReport)
      : undefined;
    message.affectedRowsReport = (object.affectedRowsReport !== undefined && object.affectedRowsReport !== null)
      ? PlanCheckRun_Result_AffectedRowsReport.fromPartial(object.affectedRowsReport)
      : undefined;
    message.sqlReviewReport = (object.sqlReviewReport !== undefined && object.sqlReviewReport !== null)
      ? PlanCheckRun_Result_SqlReviewReport.fromPartial(object.sqlReviewReport)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRun_Result_StatementTypeReport(): PlanCheckRun_Result_StatementTypeReport {
  return { statementType: "" };
}

export const PlanCheckRun_Result_StatementTypeReport = {
  encode(message: PlanCheckRun_Result_StatementTypeReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.statementType !== "") {
      writer.uint32(10).string(message.statementType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRun_Result_StatementTypeReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRun_Result_StatementTypeReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.statementType = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRun_Result_StatementTypeReport {
    return { statementType: isSet(object.statementType) ? String(object.statementType) : "" };
  },

  toJSON(message: PlanCheckRun_Result_StatementTypeReport): unknown {
    const obj: any = {};
    message.statementType !== undefined && (obj.statementType = message.statementType);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRun_Result_StatementTypeReport>): PlanCheckRun_Result_StatementTypeReport {
    return PlanCheckRun_Result_StatementTypeReport.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRun_Result_StatementTypeReport>): PlanCheckRun_Result_StatementTypeReport {
    const message = createBasePlanCheckRun_Result_StatementTypeReport();
    message.statementType = object.statementType ?? "";
    return message;
  },
};

function createBasePlanCheckRun_Result_AffectedRowsReport(): PlanCheckRun_Result_AffectedRowsReport {
  return { affectedRows: 0 };
}

export const PlanCheckRun_Result_AffectedRowsReport = {
  encode(message: PlanCheckRun_Result_AffectedRowsReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.affectedRows !== 0) {
      writer.uint32(8).int64(message.affectedRows);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRun_Result_AffectedRowsReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRun_Result_AffectedRowsReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.affectedRows = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRun_Result_AffectedRowsReport {
    return { affectedRows: isSet(object.affectedRows) ? Number(object.affectedRows) : 0 };
  },

  toJSON(message: PlanCheckRun_Result_AffectedRowsReport): unknown {
    const obj: any = {};
    message.affectedRows !== undefined && (obj.affectedRows = Math.round(message.affectedRows));
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRun_Result_AffectedRowsReport>): PlanCheckRun_Result_AffectedRowsReport {
    return PlanCheckRun_Result_AffectedRowsReport.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRun_Result_AffectedRowsReport>): PlanCheckRun_Result_AffectedRowsReport {
    const message = createBasePlanCheckRun_Result_AffectedRowsReport();
    message.affectedRows = object.affectedRows ?? 0;
    return message;
  },
};

function createBasePlanCheckRun_Result_SqlReviewReport(): PlanCheckRun_Result_SqlReviewReport {
  return { line: 0, detail: "", code: 0 };
}

export const PlanCheckRun_Result_SqlReviewReport = {
  encode(message: PlanCheckRun_Result_SqlReviewReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int64(message.line);
    }
    if (message.detail !== "") {
      writer.uint32(18).string(message.detail);
    }
    if (message.code !== 0) {
      writer.uint32(24).int64(message.code);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRun_Result_SqlReviewReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRun_Result_SqlReviewReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.line = longToNumber(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.detail = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.code = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRun_Result_SqlReviewReport {
    return {
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
    };
  },

  toJSON(message: PlanCheckRun_Result_SqlReviewReport): unknown {
    const obj: any = {};
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.detail !== undefined && (obj.detail = message.detail);
    message.code !== undefined && (obj.code = Math.round(message.code));
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRun_Result_SqlReviewReport>): PlanCheckRun_Result_SqlReviewReport {
    return PlanCheckRun_Result_SqlReviewReport.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRun_Result_SqlReviewReport>): PlanCheckRun_Result_SqlReviewReport {
    const message = createBasePlanCheckRun_Result_SqlReviewReport();
    message.line = object.line ?? 0;
    message.detail = object.detail ?? "";
    message.code = object.code ?? 0;
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
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetRolloutRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
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
  return { parent: "", plan: "" };
}

export const CreateRolloutRequest = {
  encode(message: CreateRolloutRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.plan !== "") {
      writer.uint32(18).string(message.plan);
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

          message.plan = reader.string();
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      plan: isSet(object.plan) ? String(object.plan) : "",
    };
  },

  toJSON(message: CreateRolloutRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.plan !== undefined && (obj.plan = message.plan);
    return obj;
  },

  create(base?: DeepPartial<CreateRolloutRequest>): CreateRolloutRequest {
    return CreateRolloutRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateRolloutRequest>): CreateRolloutRequest {
    const message = createBaseCreateRolloutRequest();
    message.parent = object.parent ?? "";
    message.plan = object.plan ?? "";
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
      project: isSet(object.project) ? String(object.project) : "",
      plan: isSet(object.plan) ? Plan.fromJSON(object.plan) : undefined,
    };
  },

  toJSON(message: PreviewRolloutRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.plan !== undefined && (obj.plan = message.plan ? Plan.toJSON(message.plan) : undefined);
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListTaskRunsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
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
      taskRuns: Array.isArray(object?.taskRuns) ? object.taskRuns.map((e: any) => TaskRun.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListTaskRunsResponse): unknown {
    const obj: any = {};
    if (message.taskRuns) {
      obj.taskRuns = message.taskRuns.map((e) => e ? TaskRun.toJSON(e) : undefined);
    } else {
      obj.taskRuns = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
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
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      plan: isSet(object.plan) ? String(object.plan) : "",
      title: isSet(object.title) ? String(object.title) : "",
      stages: Array.isArray(object?.stages) ? object.stages.map((e: any) => Stage.fromJSON(e)) : [],
    };
  },

  toJSON(message: Rollout): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.plan !== undefined && (obj.plan = message.plan);
    message.title !== undefined && (obj.title = message.title);
    if (message.stages) {
      obj.stages = message.stages.map((e) => e ? Stage.toJSON(e) : undefined);
    } else {
      obj.stages = [];
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
  return { name: "", uid: "", environment: "", title: "", tasks: [] };
}

export const Stage = {
  encode(message: Stage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.environment !== "") {
      writer.uint32(26).string(message.environment);
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
        case 3:
          if (tag !== 26) {
            break;
          }

          message.environment = reader.string();
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
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      environment: isSet(object.environment) ? String(object.environment) : "",
      title: isSet(object.title) ? String(object.title) : "",
      tasks: Array.isArray(object?.tasks) ? object.tasks.map((e: any) => Task.fromJSON(e)) : [],
    };
  },

  toJSON(message: Stage): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.environment !== undefined && (obj.environment = message.environment);
    message.title !== undefined && (obj.title = message.title);
    if (message.tasks) {
      obj.tasks = message.tasks.map((e) => e ? Task.toJSON(e) : undefined);
    } else {
      obj.tasks = [];
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
    message.environment = object.environment ?? "";
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
    status: 0,
    type: 0,
    blockedByTasks: [],
    target: "",
    databaseCreate: undefined,
    databaseSchemaBaseline: undefined,
    databaseSchemaUpdate: undefined,
    databaseDataUpdate: undefined,
    databaseBackup: undefined,
    databaseRestoreRestore: undefined,
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
      writer.uint32(98).string(message.specId);
    }
    if (message.status !== 0) {
      writer.uint32(104).int32(message.status);
    }
    if (message.type !== 0) {
      writer.uint32(32).int32(message.type);
    }
    for (const v of message.blockedByTasks) {
      writer.uint32(42).string(v!);
    }
    if (message.target !== "") {
      writer.uint32(50).string(message.target);
    }
    if (message.databaseCreate !== undefined) {
      Task_DatabaseCreate.encode(message.databaseCreate, writer.uint32(58).fork()).ldelim();
    }
    if (message.databaseSchemaBaseline !== undefined) {
      Task_DatabaseSchemaBaseline.encode(message.databaseSchemaBaseline, writer.uint32(66).fork()).ldelim();
    }
    if (message.databaseSchemaUpdate !== undefined) {
      Task_DatabaseSchemaUpdate.encode(message.databaseSchemaUpdate, writer.uint32(74).fork()).ldelim();
    }
    if (message.databaseDataUpdate !== undefined) {
      Task_DatabaseDataUpdate.encode(message.databaseDataUpdate, writer.uint32(82).fork()).ldelim();
    }
    if (message.databaseBackup !== undefined) {
      Task_DatabaseBackup.encode(message.databaseBackup, writer.uint32(114).fork()).ldelim();
    }
    if (message.databaseRestoreRestore !== undefined) {
      Task_DatabaseRestoreRestore.encode(message.databaseRestoreRestore, writer.uint32(90).fork()).ldelim();
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
        case 12:
          if (tag !== 98) {
            break;
          }

          message.specId = reader.string();
          continue;
        case 13:
          if (tag !== 104) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.blockedByTasks.push(reader.string());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.target = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.databaseCreate = Task_DatabaseCreate.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.databaseSchemaBaseline = Task_DatabaseSchemaBaseline.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.databaseSchemaUpdate = Task_DatabaseSchemaUpdate.decode(reader, reader.uint32());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.databaseDataUpdate = Task_DatabaseDataUpdate.decode(reader, reader.uint32());
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.databaseBackup = Task_DatabaseBackup.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.databaseRestoreRestore = Task_DatabaseRestoreRestore.decode(reader, reader.uint32());
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
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      title: isSet(object.title) ? String(object.title) : "",
      specId: isSet(object.specId) ? String(object.specId) : "",
      status: isSet(object.status) ? task_StatusFromJSON(object.status) : 0,
      type: isSet(object.type) ? task_TypeFromJSON(object.type) : 0,
      blockedByTasks: Array.isArray(object?.blockedByTasks) ? object.blockedByTasks.map((e: any) => String(e)) : [],
      target: isSet(object.target) ? String(object.target) : "",
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
      databaseBackup: isSet(object.databaseBackup) ? Task_DatabaseBackup.fromJSON(object.databaseBackup) : undefined,
      databaseRestoreRestore: isSet(object.databaseRestoreRestore)
        ? Task_DatabaseRestoreRestore.fromJSON(object.databaseRestoreRestore)
        : undefined,
    };
  },

  toJSON(message: Task): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.specId !== undefined && (obj.specId = message.specId);
    message.status !== undefined && (obj.status = task_StatusToJSON(message.status));
    message.type !== undefined && (obj.type = task_TypeToJSON(message.type));
    if (message.blockedByTasks) {
      obj.blockedByTasks = message.blockedByTasks.map((e) => e);
    } else {
      obj.blockedByTasks = [];
    }
    message.target !== undefined && (obj.target = message.target);
    message.databaseCreate !== undefined &&
      (obj.databaseCreate = message.databaseCreate ? Task_DatabaseCreate.toJSON(message.databaseCreate) : undefined);
    message.databaseSchemaBaseline !== undefined && (obj.databaseSchemaBaseline = message.databaseSchemaBaseline
      ? Task_DatabaseSchemaBaseline.toJSON(message.databaseSchemaBaseline)
      : undefined);
    message.databaseSchemaUpdate !== undefined && (obj.databaseSchemaUpdate = message.databaseSchemaUpdate
      ? Task_DatabaseSchemaUpdate.toJSON(message.databaseSchemaUpdate)
      : undefined);
    message.databaseDataUpdate !== undefined && (obj.databaseDataUpdate = message.databaseDataUpdate
      ? Task_DatabaseDataUpdate.toJSON(message.databaseDataUpdate)
      : undefined);
    message.databaseBackup !== undefined &&
      (obj.databaseBackup = message.databaseBackup ? Task_DatabaseBackup.toJSON(message.databaseBackup) : undefined);
    message.databaseRestoreRestore !== undefined && (obj.databaseRestoreRestore = message.databaseRestoreRestore
      ? Task_DatabaseRestoreRestore.toJSON(message.databaseRestoreRestore)
      : undefined);
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
    message.status = object.status ?? 0;
    message.type = object.type ?? 0;
    message.blockedByTasks = object.blockedByTasks?.map((e) => e) || [];
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
    message.databaseBackup = (object.databaseBackup !== undefined && object.databaseBackup !== null)
      ? Task_DatabaseBackup.fromPartial(object.databaseBackup)
      : undefined;
    message.databaseRestoreRestore =
      (object.databaseRestoreRestore !== undefined && object.databaseRestoreRestore !== null)
        ? Task_DatabaseRestoreRestore.fromPartial(object.databaseRestoreRestore)
        : undefined;
    return message;
  },
};

function createBaseTask_DatabaseCreate(): Task_DatabaseCreate {
  return { project: "", database: "", table: "", sheet: "", characterSet: "", collation: "", labels: {} };
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
    Object.entries(message.labels).forEach(([key, value]) => {
      Task_DatabaseCreate_LabelsEntry.encode({ key: key as any, value }, writer.uint32(58).fork()).ldelim();
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

          const entry7 = Task_DatabaseCreate_LabelsEntry.decode(reader, reader.uint32());
          if (entry7.value !== undefined) {
            message.labels[entry7.key] = entry7.value;
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
      project: isSet(object.project) ? String(object.project) : "",
      database: isSet(object.database) ? String(object.database) : "",
      table: isSet(object.table) ? String(object.table) : "",
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
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
    message.project !== undefined && (obj.project = message.project);
    message.database !== undefined && (obj.database = message.database);
    message.table !== undefined && (obj.table = message.table);
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
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
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = String(value);
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
    return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
  },

  toJSON(message: Task_DatabaseCreate_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
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
    return { schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "" };
  },

  toJSON(message: Task_DatabaseSchemaBaseline): unknown {
    const obj: any = {};
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
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
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
    };
  },

  toJSON(message: Task_DatabaseSchemaUpdate): unknown {
    const obj: any = {};
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
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
  return {
    sheet: "",
    schemaVersion: "",
    rollbackEnabled: false,
    rollbackSqlStatus: 0,
    rollbackError: "",
    rollbackSheet: "",
    rollbackFromIssue: "",
    rollbackFromTask: "",
  };
}

export const Task_DatabaseDataUpdate = {
  encode(message: Task_DatabaseDataUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheet !== "") {
      writer.uint32(10).string(message.sheet);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(18).string(message.schemaVersion);
    }
    if (message.rollbackEnabled === true) {
      writer.uint32(24).bool(message.rollbackEnabled);
    }
    if (message.rollbackSqlStatus !== 0) {
      writer.uint32(32).int32(message.rollbackSqlStatus);
    }
    if (message.rollbackError !== "") {
      writer.uint32(42).string(message.rollbackError);
    }
    if (message.rollbackSheet !== "") {
      writer.uint32(50).string(message.rollbackSheet);
    }
    if (message.rollbackFromIssue !== "") {
      writer.uint32(58).string(message.rollbackFromIssue);
    }
    if (message.rollbackFromTask !== "") {
      writer.uint32(66).string(message.rollbackFromTask);
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
        case 3:
          if (tag !== 24) {
            break;
          }

          message.rollbackEnabled = reader.bool();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.rollbackSqlStatus = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.rollbackError = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.rollbackSheet = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.rollbackFromIssue = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.rollbackFromTask = reader.string();
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
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      rollbackEnabled: isSet(object.rollbackEnabled) ? Boolean(object.rollbackEnabled) : false,
      rollbackSqlStatus: isSet(object.rollbackSqlStatus)
        ? task_DatabaseDataUpdate_RollbackSqlStatusFromJSON(object.rollbackSqlStatus)
        : 0,
      rollbackError: isSet(object.rollbackError) ? String(object.rollbackError) : "",
      rollbackSheet: isSet(object.rollbackSheet) ? String(object.rollbackSheet) : "",
      rollbackFromIssue: isSet(object.rollbackFromIssue) ? String(object.rollbackFromIssue) : "",
      rollbackFromTask: isSet(object.rollbackFromTask) ? String(object.rollbackFromTask) : "",
    };
  },

  toJSON(message: Task_DatabaseDataUpdate): unknown {
    const obj: any = {};
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    message.rollbackEnabled !== undefined && (obj.rollbackEnabled = message.rollbackEnabled);
    message.rollbackSqlStatus !== undefined &&
      (obj.rollbackSqlStatus = task_DatabaseDataUpdate_RollbackSqlStatusToJSON(message.rollbackSqlStatus));
    message.rollbackError !== undefined && (obj.rollbackError = message.rollbackError);
    message.rollbackSheet !== undefined && (obj.rollbackSheet = message.rollbackSheet);
    message.rollbackFromIssue !== undefined && (obj.rollbackFromIssue = message.rollbackFromIssue);
    message.rollbackFromTask !== undefined && (obj.rollbackFromTask = message.rollbackFromTask);
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseDataUpdate>): Task_DatabaseDataUpdate {
    return Task_DatabaseDataUpdate.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Task_DatabaseDataUpdate>): Task_DatabaseDataUpdate {
    const message = createBaseTask_DatabaseDataUpdate();
    message.sheet = object.sheet ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    message.rollbackEnabled = object.rollbackEnabled ?? false;
    message.rollbackSqlStatus = object.rollbackSqlStatus ?? 0;
    message.rollbackError = object.rollbackError ?? "";
    message.rollbackSheet = object.rollbackSheet ?? "";
    message.rollbackFromIssue = object.rollbackFromIssue ?? "";
    message.rollbackFromTask = object.rollbackFromTask ?? "";
    return message;
  },
};

function createBaseTask_DatabaseBackup(): Task_DatabaseBackup {
  return { backup: "" };
}

export const Task_DatabaseBackup = {
  encode(message: Task_DatabaseBackup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.backup !== "") {
      writer.uint32(10).string(message.backup);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseBackup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseBackup();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.backup = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseBackup {
    return { backup: isSet(object.backup) ? String(object.backup) : "" };
  },

  toJSON(message: Task_DatabaseBackup): unknown {
    const obj: any = {};
    message.backup !== undefined && (obj.backup = message.backup);
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseBackup>): Task_DatabaseBackup {
    return Task_DatabaseBackup.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Task_DatabaseBackup>): Task_DatabaseBackup {
    const message = createBaseTask_DatabaseBackup();
    message.backup = object.backup ?? "";
    return message;
  },
};

function createBaseTask_DatabaseRestoreRestore(): Task_DatabaseRestoreRestore {
  return { target: "", backup: undefined, pointInTime: undefined };
}

export const Task_DatabaseRestoreRestore = {
  encode(message: Task_DatabaseRestoreRestore, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.backup !== undefined) {
      writer.uint32(18).string(message.backup);
    }
    if (message.pointInTime !== undefined) {
      Timestamp.encode(toTimestamp(message.pointInTime), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Task_DatabaseRestoreRestore {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTask_DatabaseRestoreRestore();
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

          message.backup = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pointInTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Task_DatabaseRestoreRestore {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      backup: isSet(object.backup) ? String(object.backup) : undefined,
      pointInTime: isSet(object.pointInTime) ? fromJsonTimestamp(object.pointInTime) : undefined,
    };
  },

  toJSON(message: Task_DatabaseRestoreRestore): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.backup !== undefined && (obj.backup = message.backup);
    message.pointInTime !== undefined && (obj.pointInTime = message.pointInTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<Task_DatabaseRestoreRestore>): Task_DatabaseRestoreRestore {
    return Task_DatabaseRestoreRestore.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Task_DatabaseRestoreRestore>): Task_DatabaseRestoreRestore {
    const message = createBaseTask_DatabaseRestoreRestore();
    message.target = object.target ?? "";
    message.backup = object.backup ?? undefined;
    message.pointInTime = object.pointInTime ?? undefined;
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
    status: 0,
    detail: "",
    changeHistory: "",
    schemaVersion: "",
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
    if (message.status !== 0) {
      writer.uint32(64).int32(message.status);
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

          message.status = reader.int32() as any;
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
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      creator: isSet(object.creator) ? String(object.creator) : "",
      updater: isSet(object.updater) ? String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      title: isSet(object.title) ? String(object.title) : "",
      status: isSet(object.status) ? taskRun_StatusFromJSON(object.status) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
      changeHistory: isSet(object.changeHistory) ? String(object.changeHistory) : "",
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
    };
  },

  toJSON(message: TaskRun): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.creator !== undefined && (obj.creator = message.creator);
    message.updater !== undefined && (obj.updater = message.updater);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
    message.title !== undefined && (obj.title = message.title);
    message.status !== undefined && (obj.status = taskRun_StatusToJSON(message.status));
    message.detail !== undefined && (obj.detail = message.detail);
    message.changeHistory !== undefined && (obj.changeHistory = message.changeHistory);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
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
    message.status = object.status ?? 0;
    message.detail = object.detail ?? "";
    message.changeHistory = object.changeHistory ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    return message;
  },
};

export type RolloutServiceDefinition = typeof RolloutServiceDefinition;
export const RolloutServiceDefinition = {
  name: "RolloutService",
  fullName: "bytebase.v1.RolloutService",
  methods: {
    getPlan: {
      name: "GetPlan",
      requestType: GetPlanRequest,
      requestStream: false,
      responseType: Plan,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              31,
              18,
              29,
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
              112,
              108,
              97,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listPlans: {
      name: "ListPlans",
      requestType: ListPlansRequest,
      requestStream: false,
      responseType: ListPlansResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              31,
              18,
              29,
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
              112,
              108,
              97,
              110,
              115,
            ]),
          ],
        },
      },
    },
    createPlan: {
      name: "CreatePlan",
      requestType: CreatePlanRequest,
      requestStream: false,
      responseType: Plan,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              37,
              58,
              4,
              112,
              108,
              97,
              110,
              34,
              29,
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
              112,
              108,
              97,
              110,
              115,
            ]),
          ],
        },
      },
    },
    updatePlan: {
      name: "UpdatePlan",
      requestType: UpdatePlanRequest,
      requestStream: false,
      responseType: Plan,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 112, 108, 97, 110, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          578365826: [
            new Uint8Array([
              42,
              58,
              4,
              112,
              108,
              97,
              110,
              50,
              34,
              47,
              118,
              49,
              47,
              123,
              112,
              108,
              97,
              110,
              46,
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
              112,
              108,
              97,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
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
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              40,
              58,
              4,
              112,
              108,
              97,
              110,
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
    listPlanCheckRuns: {
      name: "ListPlanCheckRuns",
      requestType: ListPlanCheckRunsRequest,
      requestStream: false,
      responseType: ListPlanCheckRunsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              47,
              18,
              45,
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
              112,
              108,
              97,
              110,
              115,
              47,
              42,
              125,
              47,
              112,
              108,
              97,
              110,
              67,
              104,
              101,
              99,
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

export interface RolloutServiceImplementation<CallContextExt = {}> {
  getPlan(request: GetPlanRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Plan>>;
  listPlans(request: ListPlansRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListPlansResponse>>;
  createPlan(request: CreatePlanRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Plan>>;
  updatePlan(request: UpdatePlanRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Plan>>;
  getRollout(request: GetRolloutRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Rollout>>;
  createRollout(request: CreateRolloutRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Rollout>>;
  previewRollout(request: PreviewRolloutRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Rollout>>;
  listTaskRuns(
    request: ListTaskRunsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListTaskRunsResponse>>;
  listPlanCheckRuns(
    request: ListPlanCheckRunsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListPlanCheckRunsResponse>>;
  batchRunTasks(
    request: BatchRunTasksRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchRunTasksResponse>>;
  batchSkipTasks(
    request: BatchSkipTasksRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchSkipTasksResponse>>;
  batchCancelTaskRuns(
    request: BatchCancelTaskRunsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchCancelTaskRunsResponse>>;
}

export interface RolloutServiceClient<CallOptionsExt = {}> {
  getPlan(request: DeepPartial<GetPlanRequest>, options?: CallOptions & CallOptionsExt): Promise<Plan>;
  listPlans(request: DeepPartial<ListPlansRequest>, options?: CallOptions & CallOptionsExt): Promise<ListPlansResponse>;
  createPlan(request: DeepPartial<CreatePlanRequest>, options?: CallOptions & CallOptionsExt): Promise<Plan>;
  updatePlan(request: DeepPartial<UpdatePlanRequest>, options?: CallOptions & CallOptionsExt): Promise<Plan>;
  getRollout(request: DeepPartial<GetRolloutRequest>, options?: CallOptions & CallOptionsExt): Promise<Rollout>;
  createRollout(request: DeepPartial<CreateRolloutRequest>, options?: CallOptions & CallOptionsExt): Promise<Rollout>;
  previewRollout(request: DeepPartial<PreviewRolloutRequest>, options?: CallOptions & CallOptionsExt): Promise<Rollout>;
  listTaskRuns(
    request: DeepPartial<ListTaskRunsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListTaskRunsResponse>;
  listPlanCheckRuns(
    request: DeepPartial<ListPlanCheckRunsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListPlanCheckRunsResponse>;
  batchRunTasks(
    request: DeepPartial<BatchRunTasksRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchRunTasksResponse>;
  batchSkipTasks(
    request: DeepPartial<BatchSkipTasksRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchSkipTasksResponse>;
  batchCancelTaskRuns(
    request: DeepPartial<BatchCancelTaskRunsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchCancelTaskRunsResponse>;
}

declare const self: any | undefined;
declare const window: any | undefined;
declare const global: any | undefined;
const tsProtoGlobalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new tsProtoGlobalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

// If you get a compile-error about 'Constructor<Long> and ... have no overlap',
// add '--ts_proto_opt=esModuleInterop=true' as a flag when calling 'protoc'.
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
