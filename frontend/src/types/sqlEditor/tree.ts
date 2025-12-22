import type { TreeOption } from "naive-ui";
import { t } from "@/plugins/i18n";
import type { InstanceResource } from "../proto-es/v1/instance_service_pb";
import type { ComposedDatabase } from "../v1";
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

export type LabelTarget = {
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
      ? ComposedDatabase
      : T extends "label"
        ? LabelTarget
        : never;

export type SQLEditorTreeState = "UNSET" | "LOADING" | "READY";

export type SQLEditorTreeNodeMeta<
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

export const isValidSQLEditorTreeFactor = (
  str: string
): str is SQLEditorTreeFactor => {
  if (str === "instance") return true;
  if (str === "environment") return true;
  if (str.match(/^label:.+$/)) return true;
  return false;
};

export const ConnectableTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "database",
] as const;

export const LeafTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "database",
] as const;

export const extractSQLEditorLabelFactor = (factor: string) => {
  const matches = factor.match(/^label:(.+)$/);
  if (!matches) return "";
  return matches[1] ?? "";
};

export const readableSQLEditorTreeFactor = (
  factor: SQLEditorTreeFactor,
  labelPrefix: string | undefined = ""
) => {
  if (factor === "environment") {
    return t("common.environment");
  }
  if (factor === "instance") {
    return t("common.instance");
  }
  const label = extractSQLEditorLabelFactor(factor);

  return `${labelPrefix}${label}`;
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
    return (target as ComposedDatabase).instanceResource;
  }
  return undefined;
};
