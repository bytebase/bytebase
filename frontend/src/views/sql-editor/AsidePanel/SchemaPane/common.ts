import Emittery from "emittery";
import type { TreeOption } from "naive-ui";
import { ref, type RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  DependentColumn,
  ExternalTableMetadata,
  ForeignKeyMetadata,
  FunctionMetadata,
  IndexMetadata,
  PackageMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  SequenceMetadata,
  TableMetadata,
  TablePartitionMetadata,
  TriggerMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";
import { keyForDependentColumn } from "@/utils";
import { keyWithPosition } from "../../EditorCommon";

export type NodeType =
  | "database"
  | "schema"
  | "table"
  | "external-table"
  | "view"
  | "procedure"
  | "function"
  | "sequence"
  | "trigger"
  | "package"
  | "partition-table"
  | "column"
  | "index"
  | "foreign-key"
  | "dependent-column"
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
  | RichTableMetadata
  | RichExternalTableMetadata
  | RichViewMetadata
) & {
  column: ColumnMetadata;
};
export type RichDependentColumnMetadata = RichViewMetadata & {
  dependentColumn: DependentColumn;
};
export type RichIndexMetadata = RichTableMetadata & {
  index: IndexMetadata;
};
export type RichForeignKeyMetadata = RichTableMetadata & {
  foreignKey: ForeignKeyMetadata;
};
export type RichPartitionTableMetadata = RichTableMetadata & {
  parentPartition?: TablePartitionMetadata;
  partition: TablePartitionMetadata;
};
export type RichViewMetadata = RichSchemaMetadata & {
  view: ViewMetadata;
};
export type RichSequenceMetadata = RichSchemaMetadata & {
  sequence: SequenceMetadata;
  position: number;
};
export type RichTriggerMetadata = RichSchemaMetadata & {
  trigger: TriggerMetadata;
  position: number;
};
export type RichProcedureMetadata = RichSchemaMetadata & {
  procedure: ProcedureMetadata;
  position: number;
};
export type RichPackageMetadata = RichSchemaMetadata & {
  package: PackageMetadata;
  position: number;
};
export type RichFunctionMetadata = RichSchemaMetadata & {
  function: FunctionMetadata;
  position: number;
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
          : T extends "dependent-column"
            ? RichDependentColumnMetadata
            : T extends "index"
              ? RichIndexMetadata
              : T extends "foreign-key"
                ? RichForeignKeyMetadata
                : T extends "partition-table"
                  ? RichPartitionTableMetadata
                  : T extends "view"
                    ? RichViewMetadata
                    : T extends "procedure"
                      ? RichProcedureMetadata
                      : T extends "package"
                        ? RichPackageMetadata
                        : T extends "function"
                          ? RichFunctionMetadata
                          : T extends "sequence"
                            ? RichSequenceMetadata
                            : T extends "trigger"
                              ? RichTriggerMetadata
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
  "schema",
  "table",
  "view",
  "external-table",
  "partition-table",
  "expandable-text",
] as const;
export const LeafNodeTypes: readonly NodeType[] = [
  "column",
  "index",
  "foreign-key",
  "procedure",
  "package",
  "function",
  "sequence",
  "trigger",
  "dependent-column",
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
      `externalTables/${externalTable.name}`,
    ].join("/");
  }
  if (type === "column") {
    const parentKey =
      "table" in target
        ? keyForNodeTarget("table", target as NodeTarget<"table">)
        : "external-table" in target
          ? keyForNodeTarget(
              "external-table",
              target as NodeTarget<"external-table">
            )
          : "view" in target
            ? keyForNodeTarget("view", target as NodeTarget<"view">)
            : ""; // Fall back to empty string.
    const { column } = target as NodeTarget<"column">;
    return [parentKey, `columns/${column.name}`].join("/");
  }
  if (type === "dependent-column") {
    const { db, schema, view, dependentColumn } =
      target as NodeTarget<"dependent-column">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `views/${view.name}`,
      `dependentColumns/${keyForDependentColumn(dependentColumn)}`,
    ].join("/");
  }
  if (type === "index") {
    const { db, schema, table, index } = target as NodeTarget<"index">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `tables/${table.name}`,
      `indexes/${index.name}`,
    ].join("/");
  }
  if (type === "foreign-key") {
    const { db, schema, table, foreignKey } =
      target as NodeTarget<"foreign-key">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `tables/${table}`,
      `foreignKeys/${foreignKey.name}`,
    ].join("/");
  }
  if (type === "partition-table") {
    const { db, schema, table, partition } =
      target as NodeTarget<"partition-table">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `tables/${table.name}`,
      `partitionTables/${partition.name}`,
    ].join("/");
  }
  if (type === "view") {
    const { db, schema, view } = target as NodeTarget<"view">;
    return [db.name, `schemas/${schema.name}`, `views/${view.name}`].join("/");
  }
  if (type === "procedure") {
    const { db, schema, procedure, position } =
      target as NodeTarget<"procedure">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `procedures/${keyWithPosition(procedure.name, position)}`,
    ].join("/");
  }
  if (type === "package") {
    const {
      db,
      schema,
      package: pack,
      position,
    } = target as NodeTarget<"package">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `packages/${keyWithPosition(pack.name, position)}`,
    ].join("/");
  }
  if (type === "function") {
    const {
      db,
      schema,
      function: func,
      position,
    } = target as NodeTarget<"function">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `functions/${keyWithPosition(func.name, position)}`,
    ].join("/");
  }
  if (type === "sequence") {
    const { db, schema, sequence, position } = target as NodeTarget<"sequence">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `sequences/${keyWithPosition(sequence.name, position)}`,
    ].join("/");
  }
  if (type === "trigger") {
    const { db, schema, trigger, position } = target as NodeTarget<"trigger">;
    return [
      db.name,
      `schemas/${schema.name}`,
      `triggers/${keyWithPosition(trigger.name, position)}`,
    ].join("/");
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
  if (type === "index") {
    return (target as RichIndexMetadata).index.name;
  }
  if (type === "foreign-key") {
    return (target as RichForeignKeyMetadata).foreignKey.name;
  }
  if (type === "partition-table") {
    return (target as RichPartitionTableMetadata).partition.name;
  }
  if (type === "view") {
    return (target as RichViewMetadata).view.name;
  }
  if (type === "dependent-column") {
    const dep = (target as RichDependentColumnMetadata).dependentColumn;
    const parts = [dep.table, dep.column];
    if (dep.schema) parts.unshift(dep.schema);
    return parts.join(".");
  }
  if (type === "procedure") {
    return (target as RichProcedureMetadata).procedure.name;
  }
  if (type === "package") {
    return (target as RichPackageMetadata).package.name;
  }
  if (type === "function") {
    return (target as RichFunctionMetadata).function.name;
  }
  if (type === "sequence") {
    return (target as RichSequenceMetadata).sequence.name;
  }
  if (type === "trigger") {
    return (target as RichTriggerMetadata).trigger.name;
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
  type:
    | "column"
    | "dependent-column"
    | "index"
    | "foreign-key"
    | "table"
    | "view"
    | "procedure"
    | "package"
    | "function",
  parent: TreeNode,
  key: string | number = 0,
  error: unknown | undefined = undefined
) => {
  return mapTreeNodeByType(
    "error",
    {
      id: `${parent.key}/dummy-${type}-${key}`,
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
      id: `${parent.key}/${type}`,
      mockType: type,
      expandable: true,
      text,
      render,
    },
    parent
  );
};
const mapColumnNodes = (
  target:
    | NodeTarget<"table">
    | NodeTarget<"external-table">
    | NodeTarget<"view">,
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
const mapDependentColumnNodes = (
  target: NodeTarget<"view">,
  dependentColumns: DependentColumn[],
  parent: TreeNode
) => {
  if (dependentColumns.length === 0) {
    // Create a "<Empty>" node placeholder
    return [createDummyNode("column", parent)];
  }

  const children = dependentColumns.map((dependentColumn) => {
    const node = mapTreeNodeByType(
      "dependent-column",
      { ...target, dependentColumn },
      parent
    );
    return node;
  });
  return children;
};
const mapIndexNodes = (
  target: NodeTarget<"table">,
  indexes: IndexMetadata[],
  parent: TreeNode
) => {
  if (indexes.length === 0) {
    // Create a "<Empty>" index node placeholder
    return [createDummyNode("index", parent)];
  }

  const children = indexes.map((index) => {
    const node = mapTreeNodeByType("index", { ...target, index }, parent);
    return node;
  });
  return children;
};
const mapForeignKeyNodes = (
  target: NodeTarget<"table">,
  foreignKeys: ForeignKeyMetadata[],
  parent: TreeNode
) => {
  if (foreignKeys.length === 0) {
    // Create a "<Empty>" foreignKey node placeholder
    return [createDummyNode("foreign-key", parent)];
  }

  const children = foreignKeys.map((foreignKey) => {
    const node = mapTreeNodeByType(
      "foreign-key",
      { ...target, foreignKey },
      parent
    );
    return node;
  });
  return children;
};
const mapTableNodes = (target: NodeTarget<"schema">, parent: TreeNode) => {
  const { schema } = target;
  const children = schema.tables.map((table) => {
    const node = mapTreeNodeByType("table", { ...target, table }, parent);
    const columnsFolderNode = createExpandableTextNode("column", node, () =>
      t("database.columns")
    );
    node.children = [columnsFolderNode];
    // Map column columns
    columnsFolderNode.children = mapColumnNodes(
      node.meta.target,
      table.columns,
      columnsFolderNode
    );

    // Map indexes
    if (table.indexes.length > 0) {
      const indexesFolderNode = createExpandableTextNode("index", node, () =>
        t("database.indexes")
      );
      indexesFolderNode.children = mapIndexNodes(
        node.meta.target,
        table.indexes,
        indexesFolderNode
      );
      node.children.push(indexesFolderNode);
    }

    // Map foreign keys
    if (table.foreignKeys.length > 0) {
      const foreignKeysFolderNode = createExpandableTextNode(
        "foreign-key",
        node,
        () => t("database.foreign-keys")
      );
      foreignKeysFolderNode.children = mapForeignKeyNodes(
        node.meta.target,
        table.foreignKeys,
        foreignKeysFolderNode
      );
      node.children.push(foreignKeysFolderNode);
    }

    // Map table-level partitions.
    if (table.partitions.length > 0) {
      const partitionsFolderNode = createExpandableTextNode(
        "partition-table",
        node,
        () => t("db.partitions")
      );
      partitionsFolderNode.children = [];
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
        partitionsFolderNode.children?.push(subnode);
      }
      node.children?.push(partitionsFolderNode);
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
  const externalTableNodes = schema.externalTables.map((externalTable) => {
    const node = mapTreeNodeByType(
      "external-table",
      { ...target, externalTable },
      parent
    );
    const folderNode = createExpandableTextNode("column", node, () =>
      t("database.columns")
    );
    node.children = [folderNode];

    // columns
    folderNode.children = mapColumnNodes(
      node.meta.target,
      externalTable.columns,
      folderNode
    );

    return node;
  });
  return externalTableNodes;
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
  const children = schema.views.map((view) => {
    const viewNode = mapTreeNodeByType("view", { ...target, view }, parent);
    const columnsFolderNode = createExpandableTextNode("column", viewNode, () =>
      t("database.columns")
    );
    viewNode.children = [columnsFolderNode];
    // Map column columns
    columnsFolderNode.children = mapColumnNodes(
      viewNode.meta.target,
      view.columns,
      columnsFolderNode
    );
    if (view.dependentColumns.length > 0) {
      const dependentColumnsFolderNode = createExpandableTextNode(
        "dependent-column",
        viewNode,
        () => t("schema-editor.index.dependent-columns")
      );
      dependentColumnsFolderNode.children = mapDependentColumnNodes(
        viewNode.meta.target,
        view.dependentColumns,
        dependentColumnsFolderNode
      );
      viewNode.children!.push(dependentColumnsFolderNode);
    }
    return viewNode;
  });
  if (children.length === 0) {
    return [createDummyNode("view", parent)];
  }
  return children;
};
const mapProcedureNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.procedures.map((procedure, position) =>
    mapTreeNodeByType("procedure", { ...target, procedure, position }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("procedure", parent)];
  }
  return children;
};
const mapPackageNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.packages.map((pack, position) =>
    mapTreeNodeByType("package", { ...target, package: pack, position }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("package", parent)];
  }
  return children;
};
const mapFunctionNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.functions.map((func, position) =>
    mapTreeNodeByType(
      "function",
      { ...target, function: func, position },
      parent
    )
  );
  if (children.length === 0) {
    return [createDummyNode("function", parent)];
  }
  return children;
};
const mapSequenceNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.sequences.map((sequence, position) =>
    mapTreeNodeByType("sequence", { ...target, sequence, position }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("function", parent)];
  }
  return children;
};
const mapTriggerNodes = (
  target: NodeTarget<"schema">,
  parent: TreeNode<"expandable-text">
) => {
  const { schema } = target;
  const children = schema.triggers.map((trigger, position) =>
    mapTreeNodeByType("trigger", { ...target, trigger, position }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("function", parent)];
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
    schema.views.length === 0 &&
    schema.procedures.length === 0 &&
    schema.functions.length === 0
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

  // Show "Procedures" if there's at least 1 procedure
  if (schema.procedures.length > 0) {
    const procedureNode = createExpandableTextNode("procedure", parent, () =>
      t("db.procedures")
    );
    procedureNode.children = mapProcedureNodes(target, procedureNode);
    children.push(procedureNode);
  }

  // Show "Packages" if there's at least 1 package
  if (schema.packages.length > 0) {
    const packageNode = createExpandableTextNode("package", parent, () =>
      t("db.packages")
    );
    packageNode.children = mapPackageNodes(target, packageNode);
    children.push(packageNode);
  }

  // Show "Functions" if there's at least 1 function
  if (schema.functions.length > 0) {
    const functionNode = createExpandableTextNode("function", parent, () =>
      t("db.functions")
    );
    functionNode.children = mapFunctionNodes(target, functionNode);
    children.push(functionNode);
  }

  // Show "Sequences" if there's at least 1 function
  if (schema.sequences.length > 0) {
    const sequenceNode = createExpandableTextNode("sequence", parent, () =>
      t("db.sequences")
    );
    sequenceNode.children = mapSequenceNodes(target, sequenceNode);
    children.push(sequenceNode);
  }

  // Show "Triggers" if there's at least 1 function
  if (schema.triggers.length > 0) {
    const triggerNode = createExpandableTextNode("trigger", parent, () =>
      t("db.triggers")
    );
    triggerNode.children = mapTriggerNodes(target, triggerNode);
    children.push(triggerNode);
  }

  return children;
};
export const buildDatabaseSchemaTree = (
  database: ComposedDatabase,
  metadata: DatabaseMetadata
) => {
  const dummyRoot = mapTreeNodeByType(
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
    return [createDummyNode("table", dummyRoot)];
  }

  if (schemas.length === 1 && schemas[0].name === "") {
    const schema = schemas[0];
    // A single schema database, should render tables as views directly as a database
    // node's children
    return buildSchemaNodeChildren(
      { ...dummyRoot.meta.target, schema },
      dummyRoot
    );
  } else {
    // Multiple schema database
    return schemas.map((schema) => {
      const schemaNode = mapTreeNodeByType(
        "schema",
        { ...dummyRoot.meta.target, schema },
        dummyRoot
      );

      schemaNode.children = buildSchemaNodeChildren(
        schemaNode.meta.target,
        schemaNode
      );
      return schemaNode;
    });
  }

  console.error("should never reach this line");
  return [];
};

export const useClickEvents = () => {
  const DELAY = 250;
  const state = ref<{
    timeout: ReturnType<typeof setTimeout>;
    node: TreeNode;
  }>();
  const events = new Emittery<{
    "single-click": { node: TreeNode };
    "double-click": { node: TreeNode };
  }>();

  const clear = () => {
    if (!state.value) return;
    clearTimeout(state.value.timeout);
    state.value = undefined;
  };
  const queue = (node: TreeNode) => {
    state.value = {
      timeout: setTimeout(() => {
        events.emit("single-click", { node });
        clear();
      }, DELAY),
      node,
    };
  };

  const handleClick = (node: TreeNode) => {
    if (state.value && state.value.node.key === node.key) {
      events.emit("double-click", { node });
      clear();
      return;
    }
    clear();
    queue(node);
  };

  return { events, handleClick };
};
