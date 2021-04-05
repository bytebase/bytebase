import { Store } from "vuex";
import {
  Environment,
  Principal,
  TaskNew,
  Task,
  DatabaseId,
  UNKNOWN_ID,
} from "../types";

// Task
// It has to be string type because the id for stage field contain multiple parts.
export type TaskFieldId = string;

export enum TaskBuiltinFieldId {
  NAME = "1",
  STATUS = "2",
  ASSIGNEE = "3",
  DESCRIPTION = "4",
  ENVIRONMENT = "5",
  INSTANCE = "6",
  DATABASE = "7",
  DATA_SOURCE = "8",
  STAGE = "9", // The full id is concatenated with the actual stage id e.g. "8".<<stage id>>
  SQL = "10",
  ROLLBACK_SQL = "11",
}

export const INPUT_CUSTOM_FIELD_ID_BEGIN = "100";
export const OUTPUT_CUSTOM_FIELD_ID_BEGIN = "200";

export type TaskFieldType =
  | "Boolean"
  | "String"
  | "Environment"
  | "Database"
  | "NewDatabase";

export type TaskFieldReferenceProvider = {
  title: string;
  link: string;
};

export type TaskFieldReferenceProviderContext = {
  task: Task;
  field: TaskField;
};

export type TaskContext = {
  store: Store<any>;
  currentUser: Principal;
  new: boolean;
  task: Task | TaskNew;
};

export type TaskField = {
  category: "INPUT" | "OUTPUT";
  // Used as the key to store the data. This must NOT be changed after
  // in use, otherwise, it will cause data loss/corruption. Its design
  // is very similar to the message field id in Protocol Buffers.
  id: TaskFieldId;
  // Used as the key to generate UI artifacts (e.g. query parameter).
  // Though changing it won't have catastrophic consequence like changing
  // id, we strongly recommend NOT to change it as well, otherwise, previous
  // generated artifacts based on this info such as URL would become invalid.
  // slug will be formmatted to lowercase and replace any space with "-",
  // e.g. "foo Bar" => "foo-bar"
  slug: string;
  // The display name. OK to change.
  name: string;
  // Field type. This must NOT be changed after in use. Similar to id field.
  type: TaskFieldType;
  // Whether the field is required.
  required: boolean;
  // Whether the field is resolved.
  // For INPUT, one use case is together with "required" field to validate whether it's ready to submit the data.
  // For OUTPUT, one use case is to validate whether the field is filled properly according to the task.
  resolved: (ctx: TaskContext) => boolean;
  // Placeholder displayed on UI.
  placeholder?: string;
};

export const UNKNOWN_FIELD: TaskField = {
  category: "INPUT",
  id: UNKNOWN_ID,
  slug: "",
  name: "<<Unknown field>>",
  type: "String",
  required: false,
  resolved: () => false,
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
  environmentList: Environment[];
  currentUser: Principal;
};

export type TaskTemplate = {
  type: string;
  buildTask: (ctx: TemplateContext) => TaskNew;
  fieldList: TaskField[];
};
