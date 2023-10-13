import { TreeOption } from "naive-ui";
import { SchemaMetadata, TableMetadata } from "./proto/v1/database_service";
import { Environment } from "./proto/v1/environment_service";
import { ComposedDatabase, ComposedInstance, ComposedProject } from "./v1";

export type SQLEditorTreeFactor =
  | "project"
  | "instance"
  | "environment"
  | `label:${string}`; // "label:xxxxx" to group by certain label key

export type SQLEditorTreeNodeType =
  | "project"
  | "instance"
  | "environment"
  | "database"
  | "schema"
  | "table"
  | "label";

export type SQLEditorTreeNodeTarget<T extends SQLEditorTreeNodeType = any> =
  T extends "project"
    ? ComposedProject
    : T extends "instance"
    ? ComposedInstance
    : T extends "environment"
    ? Environment
    : T extends "database"
    ? ComposedDatabase
    : T extends "schema"
    ? SchemaMetadata
    : T extends "table"
    ? TableMetadata
    : T extends "label"
    ? { key: string; value: string }
    : never;

export type SQLEditorTreeState = "UNSET" | "LOADING" | "READY";

export type SQLEditorTreeNodeMeta<T extends SQLEditorTreeNodeType = any> = {
  type: T;
  target: SQLEditorTreeNodeTarget<T>;
};

export type SQLEditorTreeNode<T extends SQLEditorTreeNodeType = any> =
  TreeOption & {
    meta: SQLEditorTreeNodeMeta<T>;
    id: string;
    parent: string;
    key: string;
    children?: SQLEditorTreeNode[];
  };

export const isValidSQLEditorTreeFactor = (
  str: string
): str is SQLEditorTreeFactor => {
  if (str === "project") return true;
  if (str === "instance") return true;
  if (str === "environment") return true;
  if (str.match(/^label:.+$/)) return true;
  return false;
};

export const extractSQLEditorLabelFactor = (factor: string) => {
  const matches = factor.match(/^label:(.+)$/);
  if (!matches) return "";
  return matches[1] ?? "";
};
