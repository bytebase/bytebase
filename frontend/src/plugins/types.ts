import { Environment, Principal, TaskNew, Task, DatabaseId } from "../types";

// Task
export type TaskFieldId = string;

export enum TaskBuiltinFieldId {
  NAME = "1",
  STATUS = "2",
  ASSIGNEE = "3",
  DESCRIPTION = "4",
  ENVIRONMENT = "5",
  INSTANCE = "6",
  DATABASE = "7",
  STAGE = "8", // The full id is concatenated with the actual stage id e.g. "8".<<stage id>>
}

export const CUSTOM_FIELD_ID_BEGIN = 101;

export type TaskFieldType =
  | "String"
  | "Environment"
  | "Database"
  | "NewDatabase"
  | "Switch";

export type TaskFieldReferenceProvider = {
  title: string;
  link: string;
};

export type TaskFieldReferenceProviderContext = {
  task: Task;
  field: TaskField;
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
  // Whether the value is empty, one use case is together with "required" field to validate
  // whether it's ready to submit the data.
  isEmpty: (value: any) => boolean;
  // Placeholder displayed on UI.
  placeholder?: string;
  // Provides the reference to the place where the field originates. e.g. the request database
  // task requires to fill the data source field when resolving the task, while the data source value
  // can be found in the data source / database page. This provider can return the relevant link to
  // be displayed on the UI, which helps user navigate to the place to acquire the required info.
  provider?: (
    ctx: TaskFieldReferenceProviderContext
  ) => TaskFieldReferenceProvider;
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
