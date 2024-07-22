type IssueTypeGeneral = "bb.issue.general";

type IssueTypeDataSource = "bb.issue.data-source.request";

type IssueTypeDatabase =
  | "bb.issue.database.general" // For V1 API compatibility
  | "bb.issue.database.create"
  | "bb.issue.database.grant"
  | "bb.issue.database.schema.update"
  | "bb.issue.database.data.update"
  | "bb.issue.database.rollback"
  | "bb.issue.database.schema.update.ghost"
  | "bb.issue.database.data.export";

type IssueTypeGrantRequest = "bb.issue.grant.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDataSource
  | IssueTypeDatabase
  | IssueTypeGrantRequest;

export type IssueStatus = "OPEN" | "DONE" | "CANCELED";

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
