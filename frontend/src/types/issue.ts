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
