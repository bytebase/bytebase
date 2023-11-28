import { TreeOption } from "naive-ui";
import { RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import { useSQLEditorTreeStore } from "@/store";
import { Engine } from "./proto/v1/common";
import {
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "./proto/v1/database_service";
import { Environment } from "./proto/v1/environment_service";
import { ComposedDatabase, ComposedInstance, ComposedProject } from "./v1";

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
  | "schema"
  | "table"
  | "label"
  | "view"
  | "partition-table"
  | "expandable-text" // Text nodes to display "Tables / Views / Functions / Triggers" etc.
  | "dummy"; // Dummy nodes to display "<Empty>" etc.

export type RichSchemaMetadata = {
  database: ComposedDatabase;
  schema: SchemaMetadata;
};
export type RichTableMetadata = {
  database: ComposedDatabase;
  schema: SchemaMetadata;
  table: TableMetadata;
};
export type RichPartitionTableMetadata = {
  database: ComposedDatabase;
  schema: SchemaMetadata;
  table: TableMetadata;
  parentPartition?: TablePartitionMetadata;
  partition: TablePartitionMetadata;
};
export type RichViewMetadata = {
  database: ComposedDatabase;
  schema: SchemaMetadata;
  view: ViewMetadata;
};
export type TextTarget<E extends boolean> = {
  expandable: E;
  type: SQLEditorTreeNodeType;
  text: string | (() => string);
  render?: RenderFunction;
  searchable?: boolean;
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
    : T extends "schema"
    ? RichSchemaMetadata
    : T extends "table"
    ? RichTableMetadata
    : T extends "partition-table"
    ? RichPartitionTableMetadata
    : T extends "view"
    ? RichViewMetadata
    : T extends "label"
    ? { key: string; value: string }
    : T extends "expandable-text"
    ? TextTarget<true>
    : T extends "dummy"
    ? { type: SQLEditorTreeNodeType; error?: unknown }
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
  "table",
  "partition-table",
  "expandable-text",
  "label",
] as const;

export const ConnectableTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "instance",
  "database",
] as const;

export const LeafTreeNodeTypes: readonly SQLEditorTreeNodeType[] = [
  "view",
  "dummy",
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
  const treeStore = useSQLEditorTreeStore();
  if (factor === "project") {
    if (treeStore.projectMode) {
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
  const { type, target } = node.meta;
  if (type === "database") {
    return true;
  }
  if (type === "instance") {
    const instance = target as ComposedInstance;
    const { engine } = instance;
    return engine === Engine.MYSQL || engine === Engine.TIDB;
  }
  return false;
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
