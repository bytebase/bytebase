import type { DatabaseResource } from "@/types";
import type {
  DatabaseId,
  InstanceId,
  IssueId,
  PrincipalId,
  ProjectId,
  SheetId,
  TaskId,
} from "./id";
import type { MigrationType } from "./instance";
import type { Pipeline, PipelineCreate } from "./pipeline";
import type { Principal } from "./principal";
import type { Project } from "./project";
import type { Expr } from "./proto/google/type/expr";
import type { IssuePayload as IssueProtoPayload } from "./proto/store/issue";

type IssueTypeGeneral = "bb.issue.general";

type IssueTypeDataSource = "bb.issue.data-source.request";

type IssueTypeDatabase =
  | "bb.issue.database.general" // For V1 API compatibility
  | "bb.issue.database.create"
  | "bb.issue.database.grant"
  | "bb.issue.database.schema.update"
  | "bb.issue.database.data.update"
  | "bb.issue.database.rollback"
  | "bb.issue.database.schema.update.ghost";

type IssueTypeGrantRequest = "bb.issue.grant.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDataSource
  | IssueTypeDatabase
  | IssueTypeGrantRequest;

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
  labels?: string; // JSON encoded
};

export type MigrationDetail = {
  migrationType: MigrationType;
  statement: string;
  sheetId: SheetId;
  earliestAllowedTs: number;
  databaseId?: DatabaseId;
  databaseGroupName?: string;
  schemaGroupName?: string;
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

// eslint-disable-next-line @typescript-eslint/ban-types
export type EmptyContext = {};

export interface GrantRequestContext {
  role: "EXPORTER" | "QUERIER";
  // Conditions in CEL expression.
  databaseResources: DatabaseResource[];
  expireDays: number;
  maxRowCount: number;
  statement: string;
  exportFormat: "CSV" | "JSON";
}

export type IssueCreateContext =
  | CreateDatabaseContext
  | MigrationContext
  | GrantRequestContext
  | EmptyContext;

export interface GrantRequestPayload {
  // The requested role, e.g. PresetRoleType.PROJECT_EXPORTER
  role: string;
  // The requested user, e.g. users/hello@bytebase.com
  user: string;
  // IAM binding condition in expr.
  condition: Expr;
}

export type IssuePayload = IssueProtoPayload | { [key: string]: any };

export type Issue = {
  id: IssueId;

  // Related fields
  project: Project;
  pipeline?: Pipeline;

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
  projectId: number;
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
export const APPLICABLE_ISSUE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["RESOLVE", "CANCEL"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);
