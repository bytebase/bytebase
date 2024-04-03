import type { TreeOption } from "naive-ui";
import { t } from "@/plugins/i18n";
import { useSQLEditorStore } from "@/store";
import type { Environment } from "../proto/v1/environment_service";
import type {
  ComposedDatabase,
  ComposedInstance,
  ComposedProject,
} from "../v1";

export type SQLEditorTreeFactor =
  | "project"
  | "instance"
  | "environment"
  | `label:${string}`; // "label:xxxxx" to group by certain label key

export type StatefulSQLEditorTreeFactor<F extends SQLEditorTreeFactor = any> = {
  factor: F;
  disabled: boolean;
};

export type SQLEditorTreeNodeType =
  | "project"
  | "instance"
  | "environment"
  | "database"
  | "label";

export type LabelTarget = {
  key: string;
  value: string;
};

export type SQLEditorTreeNodeTarget<T extends SQLEditorTreeNodeType = any> =
  T extends "project"
    ? ComposedProject
    : T extends "instance"
      ? ComposedInstance
      : T extends "environment"
        ? Environment
        : T extends "database"
          ? ComposedDatabase
          : T extends "label"
            ? LabelTarget
            : never;

export type SQLEditorTreeState = "UNSET" | "LOADING" | "READY";

export type SQLEditorTreeNodeMeta<T extends SQLEditorTreeNodeType = any> = {
  type: T;
  target: SQLEditorTreeNodeTarget<T>;
};

export type SQLEditorTreeNode<T extends SQLEditorTreeNodeType = any> =
  TreeOption & {
    meta: SQLEditorTreeNodeMeta<T>;
    key: string;
    parent?: SQLEditorTreeNode;
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

export const ExpandableTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "instance",
  "environment",
  "project",
  "label",
] as const;

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
  const editorStore = useSQLEditorStore();
  if (factor === "project") {
    if (editorStore.currentProject) {
      return t("common.database");
    } else {
      return t("common.project");
    }
  }
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
    return target as ComposedInstance;
  }
  if (type === "database") {
    return (target as ComposedDatabase).instanceEntity;
  }
  return undefined;
};
