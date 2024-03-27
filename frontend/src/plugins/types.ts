// Issue
// It has to be string type because the id for stage field contain multiple parts.
export type FieldId = string;

export enum IssueBuiltinFieldId {
  NAME = "1",
  STATUS = "2",
  ASSIGNEE = "3",
  DESCRIPTION = "4",
  PROJECT = "5",
  SUBSCRIBER_LIST = "6",
  SQL = "7",
}
