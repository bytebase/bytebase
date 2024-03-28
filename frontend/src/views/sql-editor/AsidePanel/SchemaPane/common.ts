import type { TreeOption } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import type { RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";

export type NodeType =
  | "database"
  | "schema"
  | "table"
  | "external-table"
  | "view"
  | "partition-table"
  | "column"
  | "expandable-text" // Text nodes to display "Tables / Views / Functions / Triggers" etc.
  | "error"; // Error nodes to display "<Empty>" or "Cannot fetch ..." etc.

export type RichDatabaseMetadata = {
  db: ComposedDatabase;
  database: DatabaseMetadata;
};
export type RichSchemaMetadata = RichDatabaseMetadata & {
  schema: SchemaMetadata;
};
export type RichTableMetadata = RichSchemaMetadata & {
  table: TableMetadata;
};
export type RichExternalTableMetadata = RichSchemaMetadata & {
  externalTable: ExternalTableMetadata;
};
export type RichColumnMetadata = (
  | RichSchemaMetadata
  | RichPartitionTableMetadata
) & {
  column: ColumnMetadata;
};
export type RichPartitionTableMetadata = RichTableMetadata & {
  parentPartition?: TablePartitionMetadata;
  partition: TablePartitionMetadata;
};
export type RichViewMetadata = RichSchemaMetadata & {
  view: ViewMetadata;
};
export type TextTarget<E extends boolean = any, S extends boolean = any> = {
  expandable: E;
  id: string;
  mockType?: NodeType; // mock as a node type if needed
  text: () => string;
  render?: RenderFunction;
  searchable?: S;
};
export type ErrorTarget = TextTarget<false, false> & {
  error?: unknown;
};

export type NodeTarget<T extends NodeType = NodeType> = T extends "database"
  ? RichDatabaseMetadata
  : T extends "schema"
    ? RichSchemaMetadata
    : T extends "table"
      ? RichTableMetadata
      : T extends "external-table"
        ? RichExternalTableMetadata
        : T extends "column"
          ? RichColumnMetadata
          : T extends "partition-table"
            ? RichPartitionTableMetadata
            : T extends "view"
              ? RichViewMetadata
              : T extends "expandable-text"
                ? TextTarget<true, any>
                : T extends "error"
                  ? ErrorTarget
                  : never;

export type TreeState = "UNSET" | "LOADING" | "READY";

export type NodeMeta<T extends NodeType = NodeType> = {
  type: T;
  target: NodeTarget<T>;
};

export type TreeNode<T extends NodeType = NodeType> = TreeOption & {
  key: string;
  meta: NodeMeta<T>;
  parent?: TreeNode;
  children?: TreeNode[];
};

export const ExpandableNodeTypes: readonly NodeType[] = [
  "database",
  "schema",
  "table",
  "external-table",
  "partition-table",
  "expandable-text",
] as const;
export const LeafNodeTypes: readonly NodeType[] = [
  "column",
  "view",
  "error",
] as const;

export const keyForNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  if (type === "database") {
    const { db } = target as NodeTarget<"database">;
    return db.name;
  }
  if (type === "schema") {
    const { db, schema } = target as NodeTarget<"schema">;
    return [db.name, `schemas/${schema.name}`].join("/");
  }
  if (type === "table") {
    const { db, schema, table } = target as NodeTarget<"table">;
    return [db.name, `schemas/${schema.name}`, `tables/${table.name}`].join(
      "/"
    );
  }
  if (type === "external-table") {
    const { db, schema, externalTable } =
      target as NodeTarget<"external-table">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `externalTables/${externalTable.externalServerName}/${externalTable.externalDatabaseName}/${externalTable.name}`,
    ].join("/");
  }
  if (type === "column") {
    const parentKey =
      "table" in target
        ? keyForNodeTarget("table", target as NodeTarget<"table">)
        : keyForNodeTarget(
            "external-table",
            target as NodeTarget<"external-table">
          );
    const { column } = target as NodeTarget<"column">;
    return [parentKey, `columns/${column.name}`].join("/");
  }
  if (type === "partition-table") {
    const { db, schema, parentPartition, partition } =
      target as NodeTarget<"partition-table">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `partitionTables/${parentPartition?.name ?? ""}/${partition.name}`,
    ].join("/");
  }
  if (type === "view") {
    const { db, schema, view } = target as NodeTarget<"view">;
    return [db.name, `schemas/${schema.name}`, `views/${view.name}`].join("/");
  }
  if (type === "expandable-text") {
    const { id } = target as NodeTarget<"expandable-text">;
    return `expandableTexts/${id}`;
  }
  if (type === "error") {
    const { id } = target as NodeTarget<"error">;
    return `errors/${id}`;
  }
  console.assert(false, "should never reach this line");
  return "";
};

const readableTextForNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  if (type === "database") {
    return (target as NodeTarget<"database">).db.databaseName;
  }
  if (type === "schema") {
    return (target as RichSchemaMetadata).schema.name;
  }
  if (type === "table") {
    return (target as RichTableMetadata).table.name;
  }
  if (type === "external-table") {
    return (target as RichExternalTableMetadata).externalTable.name;
  }
  if (type === "column") {
    return (target as RichColumnMetadata).column.name;
  }
  if (type === "partition-table") {
    return (target as RichPartitionTableMetadata).partition.name;
  }
  if (type === "view") {
    return (target as RichViewMetadata).view.name;
  }
  if (type === "expandable-text") {
    const { text, searchable } = target as TextTarget;
    if (!searchable) return "";
    return text();
  }
  if (type === "error") {
    // Use empty strings for error nodes to make them unsearchable
    return "";
  }
  console.assert(false, "should never reach this line");
  return "";
};

const isLeafNodeType = (type: NodeType) => {
  return LeafNodeTypes.includes(type);
};

export const mapTreeNodeByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  parent: TreeNode | undefined,
  overrides: Partial<TreeNode<T>> | undefined = undefined
): TreeNode<T> => {
  const key = keyForNodeTarget(type, target);
  const node: TreeNode<T> = {
    key,
    meta: { type, target },
    parent,
    label: readableTextForNodeTarget(type, target),
    isLeaf: isLeafNodeType(type),
    ...overrides,
  };

  return node;
};

const createDummyNode = (
  type: "column" | "table" | "view",
  parent: TreeNode,
  error: unknown | undefined = undefined
) => {
  return mapTreeNodeByType(
    "error",
    {
      id: `${parent.key}/${uuidv1()}`,
      expandable: false,
      mockType: type,
      error,
      text: () => "",
    },
    parent,
    {
      disabled: true,
    }
  );
};
const createExpandableTextNode = (
  type: NodeType,
  parent: TreeNode,
  text: () => string,
  render?: RenderFunction
) => {
  return mapTreeNodeByType(
    "expandable-text",
    {
      id: `${parent.key}/${uuidv1()}`,
      mockType: type,
      expandable: true,
      text,
      render,
    },
    parent
  );
};
const mapColumnNodes = (
  target: NodeTarget<"table"> | NodeTarget<"external-table">,
  columns: ColumnMetadata[],
  parent: TreeNode
) => {
  if (columns.length === 0) {
    // Create a "<Empty>" columns node placeholder
    return [createDummyNode("column", parent)];
  }

  const children = columns.map((column) => {
    const node = mapTreeNodeByType("column", { ...target, column }, parent);
    return node;
  });
  return children;
};
const mapTableNodes = (target: NodeTarget<"schema">, parent: TreeNode) => {
  const { schema } = target;
  const children = schema.tables.map((table) => {
    const node = mapTreeNodeByType("table", { ...target, table }, parent);
    // Map table columns
    node.children = mapColumnNodes(node.meta.target, table.columns, node);

    // Map table-level partitions.
    if (table.partitions.length > 0) {
      for (const partition of table.partitions) {
        const subnode = mapTreeNodeByType(
          "partition-table",
          { ...node.meta.target, partition },
          node
        );
        if (partition.subpartitions.length > 0) {
          subnode.isLeaf = false;
          subnode.children = mapPartitionTableNodes(partition, subnode);
        } else {
          subnode.isLeaf = true;
        }
        node.children.push(subnode);
      }
    }
    return node;
  });
  if (children.length === 0) {
    return [createDummyNode("table", parent)];
  }
  return children;
};
const mapExternalTableNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode
) => {
  const { schema } = target;
  const children = schema.externalTables.map((externalTable) => {
    const node = mapTreeNodeByType(
      "external-table",
      { ...target, externalTable },
      parent
    );

    // columns
    node.children = mapColumnNodes(
      node.meta.target,
      externalTable.columns,
      node
    );

    return node;
  });
  return children;
};
// Map partition-table-level partitions.
const mapPartitionTableNodes = (
  parentPartition: TablePartitionMetadata,
  parent: TreeNode<"partition-table">
) => {
  const children = parentPartition.subpartitions.map((partition) => {
    const node = mapTreeNodeByType(
      "partition-table",
      {
        ...parent.meta.target,
        parentPartition,
        partition,
      },
      parent
    );
    if (partition.subpartitions.length > 0) {
      node.isLeaf = false;
      node.children = mapPartitionTableNodes(partition, node);
    } else {
      node.isLeaf = true;
    }
    return node;
  });
  return children;
};
const mapViewNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.views.map((view) =>
    mapTreeNodeByType("view", { ...target, view }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("view", parent)];
  }
  return children;
};
const buildSchemaNodeChildren = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"schema"> | TreeNode<"database">
) => {
  const { schema } = target;
  if (
    schema.tables.length === 0 &&
    schema.externalTables.length === 0 &&
    schema.views.length === 0
  ) {
    return [createDummyNode("table", parent)];
  }

  const children: TreeNode[] = [];

  // Always show "Tables" node
  // If no tables, show "<Empty>"
  const tablesNode = createExpandableTextNode("table", parent, () =>
    t("db.tables")
  );
  tablesNode.children = mapTableNodes(target, tablesNode);
  children.push(tablesNode);

  // Only show "External Tables" node if the schema do have external tables.
  if (schema.externalTables.length > 0) {
    const externalTablesNode = createExpandableTextNode(
      "external-table",
      parent,
      () => t("db.external-tables")
    );
    externalTablesNode.children = mapExternalTableNodes(
      target,
      externalTablesNode
    );
    children.push(externalTablesNode);
  }

  // Only show "Views" node if the schema do have views.
  if (schema.views.length > 0) {
    const viewsNode = createExpandableTextNode("view", parent, () =>
      t("db.views")
    );
    viewsNode.children = mapViewNodes(target, viewsNode);
    children.push(viewsNode);
  }
  return children;
};
export const buildDatabaseSchemaTree = (
  database: ComposedDatabase,
  metadata: DatabaseMetadata
) => {
  const root = mapTreeNodeByType(
    "database",
    {
      db: database,
      database: metadata,
    },
    /* parent */ undefined
  );
  const { schemas } = metadata;
  if (schemas.length === 0) {
    // Empty database, show "<Empty>"
    root.children = [createDummyNode("table", root)];
    return root;
  }

  if (schemas.length === 1 && schemas[0].name === "") {
    const schema = schemas[0];
    // A single schema database, should render tables as views directly as a database
    // node's children
    root.children = buildSchemaNodeChildren(
      { ...root.meta.target, schema },
      root
    );
  } else {
    // Multiple schema database
    root.children = schemas.map((schema) => {
      const schemaNode = mapTreeNodeByType(
        "schema",
        { ...root.meta.target, schema },
        root
      );

      schemaNode.children = buildSchemaNodeChildren(
        schemaNode.meta.target,
        schemaNode
      );
      return schemaNode;
    });
  }
  return root;
};
