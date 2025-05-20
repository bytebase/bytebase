type IssueTypeDatabase =
  | "bb.issue.database.create"
  | "bb.issue.database.grant"
  | "bb.issue.database.schema.update"
  | "bb.issue.database.data.update"
  | "bb.issue.database.data.export";

type IssueTypeGrantRequest = "bb.issue.grant.request";

export type IssueType = IssueTypeDatabase | IssueTypeGrantRequest;
