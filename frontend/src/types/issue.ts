import {
  BackupId,
  DatabaseId,
  InstanceId,
  IssueId,
  PrincipalId,
  ProjectId,
  SheetId,
  TaskId,
} from "./id";
import { Pipeline, PipelineCreate } from "./pipeline";
import { Principal } from "./principal";
import { Project } from "./project";
import { MigrationType } from "./instance";

type IssueTypeGeneral = "bb.issue.general";

type IssueTypeDatabase =
  | "bb.issue.database.create"
  | "bb.issue.database.grant"
  | "bb.issue.database.schema.update"
  | "bb.issue.database.data.update"
  | "bb.issue.database.rollback"
  | "bb.issue.database.schema.update.ghost"
  | "bb.issue.database.restore.pitr";

type IssueTypeDataSource = "bb.issue.data-source.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDatabase
  | IssueTypeDataSource;

export type IssueStatus = "OPEN" | "DONE" | "CANCELED";

export type CreateDatabaseContext = {
  instanceId: InstanceId;
  databaseName: string;
  tableName: string;
  // Only applicable to PostgreSQL for "WITH OWNER <<owner>>"
  owner: string;
  characterSet: string;
  collation: string;
  cluster: string;
  backupId?: BackupId;
  backupName?: string;
  labels?: string; // JSON encoded
};

export type MigrationDetail = {
  migrationType: MigrationType;
  statement: string;
  sheetId?: SheetId;
  earliestAllowedTs: number;
  databaseId?: DatabaseId;
  rollbackEnabled?: boolean;
  rollbackDetail?: RollbackDetail;
};

export type UpdateSchemaGhostDetail = MigrationDetail & {
  // empty by now
  // more input parameters in the future
};

// RollbackDetail is the detail for rolling back a task.
export type RollbackDetail = {
  // IssueID is the id of the issue to rollback.
  issueId: IssueId;
  // TaskID is the task id to rollback.
  taskId: TaskId;
};

export type MigrationContext = {
  detailList: MigrationDetail[];
};

export type PITRContext = {
  databaseId: DatabaseId;
  pointInTimeTs?: number; // UNIX timestamp
  backupId?: BackupId;
  createDatabaseContext?: CreateDatabaseContext;
};

// eslint-disable-next-line @typescript-eslint/ban-types
export type EmptyContext = {};

export type IssueCreateContext =
  | CreateDatabaseContext
  | MigrationContext
  | PITRContext
  | EmptyContext;

export type IssuePayload = { [key: string]: any };

export type Issue = {
  id: IssueId;

  // Related fields
  project: Project;
  pipeline: Pipeline;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: IssueStatus;
  type: IssueType;
  description: string;
  assignee: Principal;
  assigneeNeedAttention: boolean;
  subscriberList: Principal[];
  payload: IssuePayload;
};

export type IssueCreate = {
  // Related fields
  projectId: ProjectId;
  pipeline?: PipelineCreate;

  // Domain specific fields
  name: string;
  type: IssueType;
  description: string;
  assigneeId: PrincipalId;
  createContext: IssueCreateContext;
  payload: IssuePayload;
};

export type IssueFind = {
  projectId?: ProjectId;
  principalId?: PrincipalId;
  creatorId?: PrincipalId;
  assigneeId?: PrincipalId;
  subscriberId?: PrincipalId;
  statusList?: IssueStatus[];
  limit?: number;

  // defined in Go but not used yet
  // id?: IssueId;
  // pipelineId?: PipelineId;
  // maxId?: IssueId;
};

export type IssuePatch = {
  // Domain specific fields
  name?: string;
  description?: string;
  assigneeId?: PrincipalId;
  assigneeNeedAttention?: boolean;
  payload?: IssuePayload;
};

export type IssueStatusPatch = {
  // Domain specific fields
  status: IssueStatus;
  comment?: string;
};

export type IssueStatusTransitionType = "RESOLVE" | "CANCEL" | "REOPEN";

export interface IssueStatusTransition {
  type: IssueStatusTransitionType;
  to: IssueStatus;
  buttonName: string;
  buttonClass: string;
}

export const ISSUE_STATUS_TRANSITION_LIST: Map<
  IssueStatusTransitionType,
  IssueStatusTransition
> = new Map([
  [
    "RESOLVE",
    {
      type: "RESOLVE",
      to: "DONE",
      buttonName: "issue.status-transition.dropdown.resolve",
      buttonClass: "btn-success",
    },
  ],
  [
    "CANCEL",
    {
      type: "CANCEL",
      to: "CANCELED",
      buttonName: "issue.status-transition.dropdown.cancel",
      buttonClass: "btn-normal",
    },
  ],
  [
    "REOPEN",
    {
      type: "REOPEN",
      to: "OPEN",
      buttonName: "issue.status-transition.dropdown.reopen",
      buttonClass: "btn-normal",
    },
  ],
]);

// The first transition in the list is the primary action and the rests are
// the normal action. For now there are at most 1 primary 1 normal action.
export const CREATOR_APPLICABLE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["CANCEL"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

export const ASSIGNEE_APPLICABLE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["RESOLVE", "CANCEL"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);
