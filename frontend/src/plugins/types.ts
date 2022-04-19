import { Store } from "vuex";
import {
  Database,
  Environment,
  Issue,
  IssueCreate,
  Policy,
  Principal,
} from "../types";

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

export const INPUT_CUSTOM_FIELD_ID_BEGIN = "100";
export const OUTPUT_CUSTOM_FIELD_ID_BEGIN = "200";

export type InputFieldType = "Boolean" | "String";

export type OutputFieldType = "String" | "Database";

export type IssueContext = {
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

export type FieldInfo = {
  name: string;
  type: InputFieldType | OutputFieldType;
};

export type InputField = {
  // Used as the key to store the data. This must NOT be changed after
  // in use, otherwise, it will cause data loss/corruption. Its design
  // is very similar to the message field id in Protocol Buffers.
  id: FieldId;
  // Used as the key to generate UI artifacts (e.g. query parameter).
  // Though changing it won't have catastrophic consequence like changing
  // id, we strongly recommend NOT to change it as well, otherwise, previous
  // generated artifacts based on this info such as URL would become invalid.
  slug: string;
  // The display name. OK to change.
  name: string;
  // Field type. This must NOT be changed after in use. Similar to id field.
  type: InputFieldType;
  // Whether to allow edit after creation.
  allowEditAfterCreation: boolean;
  // Whether the field is resolved.
  // One use case is together with "required" field to validate whether it's ready to submit the data.
  // For OUTPUT, one use case is to validate whether the field is filled properly according to the issue.
  resolved: (ctx: IssueContext) => boolean;
  // Placeholder displayed on UI.
  placeholder?: string;
};

export type OutputField = {
  // Same as InputField.id
  id: FieldId;
  // Same as InputField.name
  name: string;
  // Same as InputField.type
  type: OutputFieldType;
  // Whether the field is resolved.
  // One use case is to validate whether the field is filled properly according to the issue.
  resolved: (ctx: IssueContext) => boolean;
  // Same as InputField.placeholder
  placeholder?: string;
  // Link text
  actionText: string;
  // Link to the place fulfilling this field resource
  actionLink: (ctx: IssueContext) => string;
  // Link to view the field resource
  viewLink: (ctx: IssueContext) => string;
  // Corresponding text based on whether the field is resolved
  resolveStatusText: (resolved: boolean) => string;
};

// Template
export type TemplateContext = {
  databaseList: Database[];
  environmentList: Environment[];
  approvalPolicyList: Policy[];
  currentUser: Principal;
  statementList?: string[];
};

export type IssueTemplate = {
  type: string;
  buildIssue: (
    ctx: TemplateContext
  ) => Omit<IssueCreate, "projectId" | "creatorId">;
  inputFieldList: InputField[];
  outputFieldList: OutputField[];
};
