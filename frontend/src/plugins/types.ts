import { Store } from "vuex";
import {
  Environment,
  Principal,
  IssueNew,
  Issue,
  DatabaseId,
  UNKNOWN_ID,
  Database,
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
  ENVIRONMENT = "6",
  INSTANCE = "7",
  DATABASE = "8",
  DATA_SOURCE = "9",
  STAGE = "10", // The full id is concatenated with the actual stage id e.g. "8".<<stage id>>
  SQL = "11",
  ROLLBACK_SQL = "12",
  SUBSCRIBER_LIST = "13",
}

export const INPUT_CUSTOM_FIELD_ID_BEGIN = "100";
export const OUTPUT_CUSTOM_FIELD_ID_BEGIN = "200";

export type InputFieldType =
  | "Boolean"
  | "String"
  | "Environment"
  | "Project"
  | "Database"
  | "NewDatabase";

export type OutputFieldType =
  | "Boolean"
  | "String"
  | "Environment"
  | "Project"
  | "Database"
  | "NewDatabase";

export type IssueContext = {
  store: Store<any>;
  currentUser: Principal;
  new: boolean;
  issue: Issue | IssueNew;
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
};

// Field payload for "Database" and "NewDatabase" field
export type DatabaseFieldPayload = {
  isNew: boolean;
  // If isNew is true, name stores the new database name, otherwise, is null.
  name?: string;
  // If isNew is false, id stores the database id, otherwise, is null.
  id?: DatabaseId;
  readOnly: boolean;
};

// Template
export type TemplateContext = {
  databaseList: Database[];
  environmentList: Environment[];
  currentUser: Principal;
};

export type IssueTemplate = {
  type: string;
  buildIssue: (
    ctx: TemplateContext
  ) => Omit<IssueNew, "projectId" | "creatorId">;
  inputFieldList: InputField[];
  outputFieldList: OutputField[];
};
