import type { TreeOption } from "naive-ui";
import type { Database } from "../proto-es/v1/database_service_pb";
import type { InstanceResource } from "../proto-es/v1/instance_service_pb";
import type { Environment } from "../v1/environment";

export type SQLEditorTreeFactor =
  | "instance"
  | "environment"
  | `label:${string}`; // "label:xxxxx" to group by certain label key

export type StatefulSQLEditorTreeFactor<
  F extends SQLEditorTreeFactor = SQLEditorTreeFactor,
> = {
  factor: F;
  disabled: boolean;
};

export type SQLEditorTreeNodeType =
  | "instance"
  | "environment"
  | "database"
  | "label";

type LabelTarget = {
  key: string;
  value: string;
};

export type SQLEditorTreeNodeTarget<
  T extends SQLEditorTreeNodeType = SQLEditorTreeNodeType,
> = T extends "instance"
  ? Omit<InstanceResource, "$typeName">
  : T extends "environment"
    ? Environment
    : T extends "database"
      ? Database
      : T extends "label"
        ? LabelTarget
        : never;

export type SQLEditorTreeState = "UNSET" | "LOADING" | "READY";

type SQLEditorTreeNodeMeta<
  T extends SQLEditorTreeNodeType = SQLEditorTreeNodeType,
> = {
  type: T;
  target: SQLEditorTreeNodeTarget<T>;
};

export type SQLEditorTreeNode<
  T extends SQLEditorTreeNodeType = SQLEditorTreeNodeType,
> = TreeOption & {
  meta: SQLEditorTreeNodeMeta<T>;
  key: string;
  children?: SQLEditorTreeNode[];
};

export const LeafTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "database",
] as const;

export const extractSQLEditorLabelFactor = (factor: string) => {
  const matches = factor.match(/^label:(.+)$/);
  if (!matches) return "";
  return matches[1] ?? "";
};

export const isConnectableSQLEditorTreeNode = (
  node: SQLEditorTreeNode
): boolean => {
  if (node.disabled) {
    return false;
  }
  const { type } = node.meta;
  return type === "database";
};

export const instanceOfSQLEditorTreeNode = (node: SQLEditorTreeNode) => {
  const { type, target } = node.meta;
  if (type === "instance") {
    return target as InstanceResource;
  }
  if (type === "database") {
    return (target as Database).instanceResource;
  }
  return undefined;
};
