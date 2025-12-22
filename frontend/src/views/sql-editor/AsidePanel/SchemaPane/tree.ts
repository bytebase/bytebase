import type { TreeOption } from "naive-ui";
import { type RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import type { ComposedDatabase } from "@/types";
import type {
  CheckConstraintMetadata,
  ColumnMetadata,
  DatabaseMetadata,
  DependencyColumn,
  ForeignKeyMetadata,
  IndexMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyForDependencyColumn } from "@/utils";
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
  | "check"
  | "dependency-column"
  | "expandable-text" // Text nodes to display "Tables / Views / Functions / Triggers" etc.
  | "error"; // Error nodes to display "<Empty>" or "Cannot fetch ..." etc.

export type RichDatabaseMetadata = {
  database: string;
};
export type RichSchemaMetadata = RichDatabaseMetadata & {
  schema: string;
};
export type RichTableMetadata = RichSchemaMetadata & {
  table: string;
};
export type RichExternalTableMetadata = RichSchemaMetadata & {
  externalTable: string;
};
export type RichColumnMetadata = (
  | RichTableMetadata
  | RichExternalTableMetadata
  | RichViewMetadata
) & {
  column: string;
};
export type RichDependencyColumnMetadata = RichViewMetadata & {
  dependencyColumn: DependencyColumn;
};
export type RichIndexMetadata = RichTableMetadata & {
  index: string;
};
export type RichForeignKeyMetadata = RichTableMetadata & {
  foreignKey: string;
};
export type RichCheckMetadata = RichTableMetadata & {
  check: string;
};
export type RichPartitionTableMetadata = RichTableMetadata & {
  partition: string;
};
export type RichTriggerMetadata = RichTableMetadata & {
  trigger: string;
  position: number;
};
export type RichViewMetadata = RichSchemaMetadata & {
  view: string;
};
export type RichSequenceMetadata = RichSchemaMetadata & {
  sequence: string;
  position: number;
};
export type RichProcedureMetadata = RichSchemaMetadata & {
  procedure: string;
  position: number;
};
export type RichPackageMetadata = RichSchemaMetadata & {
  package: string;
  position: number;
};
export type RichFunctionMetadata = RichSchemaMetadata & {
  function: string;
  position: number;
};
export type TextTarget<
  E extends boolean = boolean,
  S extends boolean = boolean,
> = {
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
          : T extends "dependency-column"
            ? RichDependencyColumnMetadata
            : T extends "index"
              ? RichIndexMetadata
              : T extends "foreign-key"
                ? RichForeignKeyMetadata
                : T extends "check"
                  ? RichCheckMetadata
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
                                  ? TextTarget<true>
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
  "check",
  "procedure",
  "package",
  "function",
  "sequence",
  "trigger",
  "dependency-column",
  "error",
] as const;

export const keyForNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  const { database } = target as NodeTarget<"database">;

  switch (type) {
    case "database":
      return database;
    case "schema":
      const { schema } = target as NodeTarget<"schema">;
      return [database, `schemas/${schema}`].join("/");
    case "table":
      const { table } = target as NodeTarget<"table">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `tables/${table}`,
      ].join("/");
    case "external-table":
      const { externalTable } = target as NodeTarget<"external-table">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `externalTables/${externalTable}`,
      ].join("/");
    case "view":
      const { view } = target as NodeTarget<"view">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `views/${view}`,
      ].join("/");
    case "procedure":
      const { procedure, position } = target as NodeTarget<"procedure">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `procedures/${keyWithPosition(procedure, position)}`,
      ].join("/");
    case "package":
      const { package: pack, position: pos } = target as NodeTarget<"package">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `packages/${keyWithPosition(pack, pos)}`,
      ].join("/");
    case "function":
      const { function: func, position: fp } = target as NodeTarget<"function">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `functions/${keyWithPosition(func, fp)}`,
      ].join("/");
    case "sequence":
      const { sequence, position: sp } = target as NodeTarget<"sequence">;
      return [
        keyForNodeTarget("schema", target as NodeTarget<"schema">),
        `sequences/${keyWithPosition(sequence, sp)}`,
      ].join("/");
    case "trigger":
      const { trigger, position: tp } = target as NodeTarget<"trigger">;
      return [
        keyForNodeTarget("table", target as NodeTarget<"table">),
        `triggers/${keyWithPosition(trigger, tp)}`,
      ].join("/");
    case "index":
      const { index } = target as NodeTarget<"index">;
      return [
        keyForNodeTarget("table", target as NodeTarget<"table">),
        `indexes/${index}`,
      ].join("/");
    case "foreign-key":
      const { foreignKey } = target as NodeTarget<"foreign-key">;
      return [
        keyForNodeTarget("table", target as NodeTarget<"table">),
        `foreignKeys/${foreignKey}`,
      ].join("/");
    case "check":
      const { check } = target as NodeTarget<"check">;
      return [
        keyForNodeTarget("table", target as NodeTarget<"table">),
        `checks/${check}`,
      ].join("/");
    case "partition-table":
      const { partition } = target as NodeTarget<"partition-table">;
      return [
        keyForNodeTarget("table", target as NodeTarget<"table">),
        `partitionTables/${partition}`,
      ].join("/");
    case "dependency-column":
      const { dependencyColumn } = target as NodeTarget<"dependency-column">;
      return [
        keyForNodeTarget("view", target as NodeTarget<"view">),
        `dependencyColumns/${keyForDependencyColumn(dependencyColumn)}`,
      ].join("/");
    case "column":
      const { column } = target as NodeTarget<"column">;
      const key = `columns/${column}`;
      if ("table" in target) {
        return [
          keyForNodeTarget("table", target as NodeTarget<"table">),
          key,
        ].join("/");
      } else if ("external-table" in target) {
        return [
          keyForNodeTarget(
            "external-table",
            target as NodeTarget<"external-table">
          ),
          key,
        ].join("/");
      } else if ("view" in target) {
        return [
          keyForNodeTarget("view", target as NodeTarget<"view">),
          key,
        ].join("/");
      }
      return "";
    default:
      const { id } = target as NodeTarget<"expandable-text" | "error">;
      return `${type}/${id}`;
  }
};

export const readableTextForNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  switch (type) {
    case "error":
      // Use empty strings for error nodes to make them unsearchable
      return "";
    case "expandable-text":
      const { text, searchable } = target as TextTarget;
      if (!searchable) return "";
      return text();
    case "dependency-column":
      const dep = (target as NodeTarget<"dependency-column">).dependencyColumn;
      const parts = [dep.table, dep.column];
      if (dep.schema) parts.unshift(dep.schema);
      return parts.join(".");
    default:
      const key = keyForNodeTarget(type, target);
      return key.split("/").slice(-1)[0];
  }
};

const isLeafNodeType = (type: NodeType) => {
  return LeafNodeTypes.includes(type);
};

export const mapTreeNodeByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  overrides: Partial<TreeNode<T>> | undefined = undefined
): TreeNode<T> => {
  const key = keyForNodeTarget(type, target);
  const node: TreeNode<T> = {
    key,
    meta: { type, target },
    label: readableTextForNodeTarget(type, target),
    isLeaf: isLeafNodeType(type),
    ...overrides,
  };

  return node;
};

const createDummyNode = <T extends NodeType>(
  type: T,
  parentKey: string,
  key: string | number = 0
) => {
  return mapTreeNodeByType(
    "error",
    {
      id: `${parentKey}/dummy-${type}-${key}`,
      expandable: false,
      mockType: type,
      text: () => "",
    },
    {
      disabled: true,
    }
  );
};

const createExpandableTextNode = (
  type: NodeType,
  parentKey: string,
  text: () => string,
  render?: RenderFunction
) => {
  return mapTreeNodeByType("expandable-text", {
    id: `${parentKey}/${type}`,
    mockType: type,
    expandable: true,
    text,
    render,
  });
};

const mapColumnNodes = (
  target:
    | NodeTarget<"table">
    | NodeTarget<"external-table">
    | NodeTarget<"view">,
  columns: ColumnMetadata[],
  parentKey: string
) => {
  if (columns.length === 0) {
    // Create a "<Empty>" columns node placeholder
    return [createDummyNode("column", parentKey)];
  }

  const children = columns.map((column) => {
    const node = mapTreeNodeByType("column", {
      ...target,
      column: column.name,
    });
    return node;
  });
  return children;
};

const mapIndexNodes = (
  target: NodeTarget<"table">,
  indexes: IndexMetadata[],
  parentKey: string
) => {
  if (indexes.length === 0) {
    // Create a "<Empty>" index node placeholder
    return [createDummyNode("index", parentKey)];
  }

  const children = indexes.map((index) => {
    const node = mapTreeNodeByType("index", { ...target, index: index.name });
    return node;
  });
  return children;
};

const mapForeignKeyNodes = (
  target: NodeTarget<"table">,
  foreignKeys: ForeignKeyMetadata[],
  parentKey: string
) => {
  if (foreignKeys.length === 0) {
    // Create a "<Empty>" foreignKey node placeholder
    return [createDummyNode("foreign-key", parentKey)];
  }

  const children = foreignKeys.map((foreignKey) => {
    const node = mapTreeNodeByType("foreign-key", {
      ...target,
      foreignKey: foreignKey.name,
    });
    return node;
  });
  return children;
};

const mapCheckNodes = (
  target: NodeTarget<"table">,
  checks: CheckConstraintMetadata[],
  parentKey: string
) => {
  if (checks.length === 0) {
    // Create a "<Empty>" check node placeholder
    return [createDummyNode("check", parentKey)];
  }

  const children = checks.map((check) => {
    const node = mapTreeNodeByType("check", {
      ...target,
      check: check.name,
    });
    return node;
  });
  return children;
};

const mapTableNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.tables.map((table) => {
    const node = mapTreeNodeByType("table", {
      ...target,
      table: table.name,
    });
    const columnsFolderNode = createExpandableTextNode("column", node.key, () =>
      t("database.columns")
    );
    node.children = [columnsFolderNode];
    // Map column columns
    columnsFolderNode.children = mapColumnNodes(
      node.meta.target,
      table.columns,
      columnsFolderNode.key
    );

    // Map indexes
    if (table.indexes.length > 0) {
      const indexesFolderNode = createExpandableTextNode(
        "index",
        node.key,
        () => t("database.indexes")
      );
      indexesFolderNode.children = mapIndexNodes(
        node.meta.target,
        table.indexes,
        indexesFolderNode.key
      );
      node.children.push(indexesFolderNode);
    }

    // Map foreign keys
    if (table.foreignKeys.length > 0) {
      const foreignKeysFolderNode = createExpandableTextNode(
        "foreign-key",
        node.key,
        () => t("database.foreign-keys")
      );
      foreignKeysFolderNode.children = mapForeignKeyNodes(
        node.meta.target,
        table.foreignKeys,
        foreignKeysFolderNode.key
      );
      node.children.push(foreignKeysFolderNode);
    }

    // Show "Triggers" if there's at least 1 function
    if (table.triggers.length > 0) {
      const triggerNode = createExpandableTextNode("trigger", node.key, () =>
        t("db.triggers")
      );
      triggerNode.children = mapTriggerNodes(
        table,
        node.meta.target,
        triggerNode.key
      );
      node.children.push(triggerNode);
    }

    // Map checks
    if (table.checkConstraints.length > 0) {
      const checksFolderNode = createExpandableTextNode("check", node.key, () =>
        t("database.checks")
      );
      checksFolderNode.children = mapCheckNodes(
        node.meta.target,
        table.checkConstraints,
        checksFolderNode.key
      );
      node.children.push(checksFolderNode);
    }

    // Map table-level partitions.
    if (table.partitions.length > 0) {
      const partitionsFolderNode = createExpandableTextNode(
        "partition-table",
        node.key,
        () => t("db.partitions")
      );
      partitionsFolderNode.children = [];
      for (const partition of table.partitions) {
        const subnode = mapTreeNodeByType("partition-table", {
          ...node.meta.target,
          partition: partition.name,
        });
        if (partition.subpartitions.length > 0) {
          subnode.isLeaf = false;
          subnode.children = mapPartitionTableNodes(
            partition,
            subnode.meta.target
          );
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
    return [createDummyNode("table", parentKey)];
  }
  return children;
};

const mapExternalTableNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">
) => {
  const externalTableNodes = schema.externalTables.map((externalTable) => {
    const node = mapTreeNodeByType("external-table", {
      ...target,
      externalTable: externalTable.name,
    });
    const folderNode = createExpandableTextNode("column", node.key, () =>
      t("database.columns")
    );
    node.children = [folderNode];

    // columns
    folderNode.children = mapColumnNodes(
      node.meta.target,
      externalTable.columns,
      folderNode.key
    );

    return node;
  });
  return externalTableNodes;
};

// Map partition-table-level partitions.
const mapPartitionTableNodes = (
  parentPartition: TablePartitionMetadata,
  target: RichPartitionTableMetadata
) => {
  const children = parentPartition.subpartitions.map((partition) => {
    const node = mapTreeNodeByType("partition-table", {
      ...target,
      partition: partition.name,
    });
    if (partition.subpartitions.length > 0) {
      node.isLeaf = false;
      node.children = mapPartitionTableNodes(partition, node.meta.target);
    } else {
      node.isLeaf = true;
    }
    return node;
  });
  return children;
};

const mapViewNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.views.map((view) => {
    const viewNode = mapTreeNodeByType("view", {
      ...target,
      view: view.name,
    });
    const columnsFolderNode = createExpandableTextNode(
      "column",
      viewNode.key,
      () => t("database.columns")
    );
    viewNode.children = [columnsFolderNode];
    // Map column columns
    columnsFolderNode.children = mapColumnNodes(
      viewNode.meta.target,
      view.columns,
      columnsFolderNode.key
    );
    return viewNode;
  });
  if (children.length === 0) {
    return [createDummyNode("view", parentKey)];
  }
  return children;
};

const mapProcedureNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.procedures.map((procedure, position) =>
    mapTreeNodeByType("procedure", {
      ...target,
      procedure: procedure.name,
      position,
    })
  );
  if (children.length === 0) {
    return [createDummyNode("procedure", parentKey)];
  }
  return children;
};

const mapPackageNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.packages.map((pack, position) =>
    mapTreeNodeByType("package", {
      ...target,
      package: pack.name,
      position,
    })
  );
  if (children.length === 0) {
    return [createDummyNode("package", parentKey)];
  }
  return children;
};

const mapFunctionNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.functions.map((func, position) =>
    mapTreeNodeByType("function", {
      ...target,
      function: func.name,
      position,
    })
  );
  if (children.length === 0) {
    return [createDummyNode("function", parentKey)];
  }
  return children;
};

const mapSequenceNodes = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  const children = schema.sequences.map((sequence, position) =>
    mapTreeNodeByType("sequence", {
      ...target,
      sequence: sequence.name,
      position,
    })
  );
  if (children.length === 0) {
    return [createDummyNode("function", parentKey)];
  }
  return children;
};

const mapTriggerNodes = (
  table: TableMetadata,
  target: NodeTarget<"table">,
  parentKey: string
) => {
  const children = table.triggers.map((trigger, position) =>
    mapTreeNodeByType("trigger", { ...target, trigger: trigger.name, position })
  );
  if (children.length === 0) {
    return [createDummyNode("function", parentKey)];
  }
  return children;
};

const buildSchemaNodeChildren = (
  schema: SchemaMetadata,
  target: NodeTarget<"schema">,
  parentKey: string
) => {
  if (
    schema.tables.length === 0 &&
    schema.externalTables.length === 0 &&
    schema.views.length === 0 &&
    schema.procedures.length === 0 &&
    schema.functions.length === 0
  ) {
    return [createDummyNode("table", parentKey)];
  }

  const children: TreeNode[] = [];

  // Always show "Tables" node
  // If no tables, show "<Empty>"
  const tablesNode = createExpandableTextNode("table", parentKey, () =>
    t("db.tables")
  );
  tablesNode.children = mapTableNodes(schema, target, tablesNode.key);
  children.push(tablesNode);

  // Only show "External Tables" node if the schema do have external tables.
  if (schema.externalTables.length > 0) {
    const externalTablesNode = createExpandableTextNode(
      "external-table",
      parentKey,
      () => t("db.external-tables")
    );
    externalTablesNode.children = mapExternalTableNodes(schema, target);
    children.push(externalTablesNode);
  }

  // Only show "Views" node if the schema do have views.
  if (schema.views.length > 0) {
    const viewsNode = createExpandableTextNode("view", parentKey, () =>
      t("db.views")
    );
    viewsNode.children = mapViewNodes(schema, target, viewsNode.key);
    children.push(viewsNode);
  }

  // Show "Procedures" if there's at least 1 procedure
  if (schema.procedures.length > 0) {
    const procedureNode = createExpandableTextNode("procedure", parentKey, () =>
      t("db.procedures")
    );
    procedureNode.children = mapProcedureNodes(
      schema,
      target,
      procedureNode.key
    );
    children.push(procedureNode);
  }

  // Show "Packages" if there's at least 1 package
  if (schema.packages.length > 0) {
    const packageNode = createExpandableTextNode("package", parentKey, () =>
      t("db.packages")
    );
    packageNode.children = mapPackageNodes(schema, target, packageNode.key);
    children.push(packageNode);
  }

  // Show "Functions" if there's at least 1 function
  if (schema.functions.length > 0) {
    const functionNode = createExpandableTextNode("function", parentKey, () =>
      t("db.functions")
    );
    functionNode.children = mapFunctionNodes(schema, target, functionNode.key);
    children.push(functionNode);
  }

  // Show "Sequences" if there's at least 1 function
  if (schema.sequences.length > 0) {
    const sequenceNode = createExpandableTextNode("sequence", parentKey, () =>
      t("db.sequences")
    );
    sequenceNode.children = mapSequenceNodes(schema, target, sequenceNode.key);
    children.push(sequenceNode);
  }

  return children;
};

export const buildDatabaseSchemaTree = (
  database: ComposedDatabase,
  metadata: DatabaseMetadata
) => {
  const dummyRoot = mapTreeNodeByType("database", {
    database: database.name,
  });
  const { schemas } = metadata;
  if (schemas.length === 0) {
    // Empty database, show "<Empty>"
    return [createDummyNode("table", dummyRoot.key)];
  }

  if (schemas.length === 1 && schemas[0].name === "") {
    const schema = schemas[0];
    // A single schema database, should render tables as views directly as a database
    // node's children
    return buildSchemaNodeChildren(
      schema,
      { ...dummyRoot.meta.target, schema: schema.name },
      dummyRoot.key
    );
  } else {
    // Multiple schema database
    return schemas.map((schema) => {
      const schemaNode = mapTreeNodeByType("schema", {
        ...dummyRoot.meta.target,
        schema: schema.name,
      });

      schemaNode.children = buildSchemaNodeChildren(
        schema,
        schemaNode.meta.target,
        schemaNode.key
      );
      return schemaNode;
    });
  }
};
