import { IssueId, PrincipalId, ProjectId } from "./id";
import { Pipeline, PipelineCreate } from "./pipeline";
import { Principal } from "./principal";
import { Project } from "./project";

type IssueTypeGeneral = "bb.general";

type IssueTypeDatabase =
  | "bb.database.create"
  | "bb.database.grant"
  | "bb.database.schema.update";

type IssueTypeDataSource = "bb.data-source.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDatabase
  | IssueTypeDataSource;

export type IssueStatus = "OPEN" | "DONE" | "CANCELED";

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
  assignee?: Principal;
  subscriberList: Principal[];
  sql?: string;
  rollbackSql?: string;
  payload: IssuePayload;
};

export type IssueCreate = {
  // Related fields
  projectId: ProjectId;
  pipeline: PipelineCreate;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  type: IssueType;
  description: string;
  assigneeId?: PrincipalId;
  sql?: string;
  rollbackSql?: string;
  payload: IssuePayload;
};

export type IssuePatch = {
  // Related fields
  projectId?: ProjectId;

  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  name?: string;
  description?: string;
  assigneeId?: PrincipalId;
  subscriberIdList?: PrincipalId[];
  sql?: string;
  rollbackSql?: string;
  payload?: IssuePayload;
};

export type IssueStatusPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: IssueStatus;
  comment?: string;
};

export type IssueStatusTransitionType = "RESOLVE" | "ABORT" | "REOPEN";

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
      buttonName: "Resolve",
      buttonClass: "btn-success",
    },
  ],
  [
    "ABORT",
    {
      type: "ABORT",
      to: "CANCELED",
      buttonName: "Abort",
      buttonClass: "btn-normal",
    },
  ],
  [
    "REOPEN",
    {
      type: "REOPEN",
      to: "OPEN",
      buttonName: "Reopen",
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
  ["OPEN", ["ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

export const ASSIGNEE_APPLICABLE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["RESOLVE", "ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);
