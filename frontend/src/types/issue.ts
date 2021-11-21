import { IssueID, PrincipalID, ProjectID } from "./id";
import { Pipeline, PipelineCreate } from "./pipeline";
import { Principal } from "./principal";
import { Project } from "./project";

type IssueTypeGeneral = "bb.issue.general";

type IssueTypeDatabase =
  | "bb.issue.database.create"
  | "bb.issue.database.grant"
  | "bb.issue.database.schema.update";

type IssueTypeDataSource = "bb.issue.data-source.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDatabase
  | IssueTypeDataSource;

export type IssueStatus = "OPEN" | "DONE" | "CANCELED";

export type IssuePayload = { [key: string]: any };

export type Issue = {
  id: IssueID;

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
  subscriberIDList: PrincipalID[];
  payload: IssuePayload;
};

export type IssueCreate = {
  // Related fields
  projectID: ProjectID;
  pipeline: PipelineCreate;

  // Domain specific fields
  name: string;
  type: IssueType;
  description: string;
  assigneeID: PrincipalID;
  rollbackIssueID?: IssueID;
  payload: IssuePayload;
};

export type IssuePatch = {
  // Domain specific fields
  name?: string;
  description?: string;
  assigneeID?: PrincipalID;
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
      buttonName: "Resolve",
      buttonClass: "btn-success",
    },
  ],
  [
    "CANCEL",
    {
      type: "CANCEL",
      to: "CANCELED",
      buttonName: "Cancel issue",
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
