import type { IssueId, TaskId } from "./id";
import type { Pipeline } from "./pipeline";
import type { Principal } from "./principal";
import type { Project } from "./project";
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

// RollbackDetail is the detail for rolling back a task.
export type RollbackDetail = {
  // IssueID is the id of the issue to rollback.
  issueId: IssueId;
  // TaskID is the task id to rollback.
  taskId: TaskId;
};

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
