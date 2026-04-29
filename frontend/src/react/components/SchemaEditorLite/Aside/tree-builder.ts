import type {
  ColumnMetadata,
  Database,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseEngine,
  getInstanceResource,
  groupBy,
} from "@/utils";
import { keyForResource } from "../core/keyForResource";
import {
  engineSupportsEditFunctions,
  engineSupportsEditProcedures,
  engineSupportsEditViews,
} from "../core/spec";
import type { EditTarget } from "../types";

// Tree node types — framework-agnostic (no naive-ui TreeOption dependency).
// react-arborist uses `id` + `name` + `children` by default,
// but we keep `key` + `label` + `children` and map in the component.

export interface BaseTreeNode {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

export interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  instance: InstanceResource;
  children: TreeNodeForDatabase[];
}

export interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
  };
  children: TreeNodeForSchema[] | TreeNodeForGroup[];
}

export interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
  children: TreeNodeForGroup[];
}

export interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
  children: TreeNodeForColumn[] | TreeNodeForPlaceholder<"column">[];
}

export interface TreeNodeForColumn extends BaseTreeNode {
  type: "column";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
    column: ColumnMetadata;
  };
  children: undefined;
  isLeaf: true;
}

export interface TreeNodeForView extends BaseTreeNode {
  type: "view";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    view: ViewMetadata;
  };
  children: undefined;
  isLeaf: true;
}

export interface TreeNodeForProcedure extends BaseTreeNode {
  type: "procedure";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    procedure: ProcedureMetadata;
  };
  children: undefined;
  isLeaf: true;
}

export interface TreeNodeForFunction extends BaseTreeNode {
  type: "function";
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    function: FunctionMetadata;
  };
  children: undefined;
  isLeaf: true;
}

export type GroupNodeType = "table" | "view" | "procedure" | "function";

export interface TreeNodeForGroup<T extends GroupNodeType = GroupNodeType>
  extends BaseTreeNode {
  type: "group";
  group: T;
  db: Database;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
  isLeaf: false;
  children: T extends "table"
    ? TreeNodeForTable[] | TreeNodeForPlaceholder<"table">[]
    : T extends "view"
      ? TreeNodeForView[] | TreeNodeForPlaceholder<"view">[]
      : T extends "procedure"
        ? TreeNodeForProcedure[] | TreeNodeForPlaceholder<"procedure">[]
        : T extends "function"
          ? TreeNodeForFunction[] | TreeNodeForPlaceholder<"function">[]
          : never;
}

export type PlaceholderNodeType =
  | "table"
  | "column"
  | "view"
  | "procedure"
  | "function";

export interface TreeNodeForPlaceholder<
  T extends PlaceholderNodeType = PlaceholderNodeType,
> extends BaseTreeNode {
  type: "placeholder";
  placeholder: T;
  isLeaf: true;
  children: undefined;
}

export type TreeNode =
  | TreeNodeForInstance
  | TreeNodeForDatabase
  | TreeNodeForSchema
  | TreeNodeForTable
  | TreeNodeForColumn
  | TreeNodeForView
  | TreeNodeForProcedure
  | TreeNodeForFunction
  | TreeNodeForGroup
  | TreeNodeForPlaceholder;

export type BuildTreeOptions = {
  byInstance: boolean;
};

export type BuildTreeResult = {
  tree: TreeNode[];
  nodeMap: Map<string, TreeNode>;
};

export function buildTree(
  targets: EditTarget[],
  options: BuildTreeOptions = { byInstance: true }
): BuildTreeResult {
  const nodeMap = new Map<string, TreeNode>();

  let tree: TreeNode[];
  if (options.byInstance) {
    tree = buildInstanceNodeList(targets, nodeMap);
  } else {
    const target = targets[0];
    tree = buildSchemaNodeList(
      target.metadata.schemas,
      nodeMap,
      target.database,
      target.metadata
    );
  }

  return { tree, nodeMap };
}

function buildInstanceNodeList(
  targets: EditTarget[],
  map: Map<string, TreeNode>
): TreeNodeForInstance[] {
  const groupedByInstance = groupBy(
    targets,
    (target) => extractDatabaseResourceName(target.database.name).instance
  );
  return Array.from(groupedByInstance).map(([key, targets]) => {
    const instance = getInstanceResource(targets[0].database);
    const instanceNode: TreeNodeForInstance = {
      type: "instance",
      key,
      label: instance.title,
      isLeaf: false,
      instance,
      children: [],
    };
    map.set(instanceNode.key, instanceNode);
    instanceNode.children = buildDatabaseNodeList(targets, map);
    return instanceNode;
  });
}

function buildDatabaseNodeList(
  targets: EditTarget[],
  map: Map<string, TreeNode>
): TreeNodeForDatabase[] {
  return targets.map((target) => {
    const db = target.database;
    const databaseNode: TreeNodeForDatabase = {
      type: "database",
      key: db.name,
      label: extractDatabaseResourceName(db.name).databaseName,
      isLeaf: false,
      db,
      metadata: {
        database: target.metadata,
      },
      children: [],
    };
    map.set(databaseNode.key, databaseNode);
    databaseNode.children = buildSchemaNodeList(
      target.metadata.schemas,
      map,
      db,
      target.metadata
    ) as TreeNodeForSchema[] | TreeNodeForGroup[];
    return databaseNode;
  });
}

function buildSchemaNodeList(
  schemas: SchemaMetadata[],
  map: Map<string, TreeNode>,
  db: Database,
  database: DatabaseMetadata
): (TreeNodeForSchema | TreeNodeForGroup)[] {
  const mapSchemaChildrenNodes = (
    schema: SchemaMetadata
  ): TreeNodeForGroup[] => {
    const groups: TreeNodeForGroup[] = [];
    const metadata = { database, schema };

    // Tables
    const tableGroupNode: TreeNodeForGroup<"table"> = {
      type: "group",
      group: "table",
      key: keyForResource(db, metadata, "table-group"),
      db,
      metadata,
      label: "Tables",
      children: [],
      isLeaf: false,
    };
    tableGroupNode.children = buildTableNodeList(
      schema.tables,
      map,
      tableGroupNode
    );
    if (tableGroupNode.children.length === 0) {
      tableGroupNode.children = [
        buildPlaceholderNode("table", tableGroupNode, map),
      ];
    }
    groups.push(tableGroupNode);
    map.set(tableGroupNode.key, tableGroupNode);

    // Views
    if (engineSupportsEditViews(getDatabaseEngine(db))) {
      const viewGroupNode: TreeNodeForGroup<"view"> = {
        type: "group",
        group: "view",
        key: keyForResource(db, metadata, "view-group"),
        db,
        metadata,
        label: "Views",
        children: [],
        isLeaf: false,
      };
      viewGroupNode.children = buildViewNodeList(
        schema.views,
        map,
        viewGroupNode
      );
      if (viewGroupNode.children.length === 0) {
        viewGroupNode.children = [
          buildPlaceholderNode("view", viewGroupNode, map),
        ];
      }
      groups.push(viewGroupNode);
      map.set(viewGroupNode.key, viewGroupNode);
    }

    // Procedures
    if (engineSupportsEditProcedures(getDatabaseEngine(db))) {
      const procedureGroupNode: TreeNodeForGroup<"procedure"> = {
        type: "group",
        group: "procedure",
        key: keyForResource(db, metadata, "procedure-group"),
        db,
        metadata,
        label: "Procedures",
        children: [],
        isLeaf: false,
      };
      procedureGroupNode.children = buildProcedureNodeList(
        schema.procedures,
        map,
        procedureGroupNode
      );
      if (procedureGroupNode.children.length === 0) {
        procedureGroupNode.children = [
          buildPlaceholderNode("procedure", procedureGroupNode, map),
        ];
      }
      groups.push(procedureGroupNode);
      map.set(procedureGroupNode.key, procedureGroupNode);
    }

    // Functions
    if (engineSupportsEditFunctions(getDatabaseEngine(db))) {
      const functionGroupNode: TreeNodeForGroup<"function"> = {
        type: "group",
        group: "function",
        key: keyForResource(db, metadata, "function-group"),
        db,
        metadata,
        label: "Functions",
        children: [],
        isLeaf: false,
      };
      functionGroupNode.children = buildFunctionNodeList(
        schema.functions,
        map,
        functionGroupNode
      );
      if (functionGroupNode.children.length === 0) {
        functionGroupNode.children = [
          buildPlaceholderNode("function", functionGroupNode, map),
        ];
      }
      groups.push(functionGroupNode);
      map.set(functionGroupNode.key, functionGroupNode);
    }

    return groups;
  };

  // MySQL, TiDB has only one "schema" with empty name
  if (schemas.length === 1 && schemas[0].name === "") {
    return mapSchemaChildrenNodes(schemas[0]);
  }

  return schemas.map((schema) => {
    const metadata = { database, schema };
    const schemaNode: TreeNodeForSchema = {
      type: "schema",
      key: keyForResource(db, metadata),
      label: schema.name,
      isLeaf: false,
      db,
      metadata,
      children: [],
    };
    schemaNode.children = mapSchemaChildrenNodes(schema);
    map.set(schemaNode.key, schemaNode);
    return schemaNode;
  });
}

function buildTableNodeList(
  tables: TableMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForGroup<"table">
): TreeNodeForTable[] {
  return tables.map((table) => {
    const { db } = parent;
    const metadata = { ...parent.metadata, table };
    const tableNode: TreeNodeForTable = {
      type: "table",
      key: keyForResource(db, metadata),
      label: table.name,
      isLeaf: false,
      db,
      metadata,
      children: [],
    };
    map.set(tableNode.key, tableNode);
    tableNode.children = buildColumnNodeList(table.columns, map, tableNode);
    if (tableNode.children.length === 0) {
      tableNode.children = [buildPlaceholderNode("column", tableNode, map)];
    }
    return tableNode;
  });
}

function buildViewNodeList(
  views: ViewMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForGroup<"view">
): TreeNodeForView[] {
  return views.map((view) => {
    const { db } = parent;
    const metadata = { ...parent.metadata, view };
    const viewNode: TreeNodeForView = {
      type: "view",
      key: keyForResource(db, metadata),
      label: view.name,
      isLeaf: true,
      db,
      metadata,
      children: undefined,
    };
    map.set(viewNode.key, viewNode);
    return viewNode;
  });
}

function buildProcedureNodeList(
  procedures: ProcedureMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForGroup<"procedure">
): TreeNodeForProcedure[] {
  return procedures.map((procedure) => {
    const { db } = parent;
    const metadata = { ...parent.metadata, procedure };
    const procedureNode: TreeNodeForProcedure = {
      type: "procedure",
      key: keyForResource(db, metadata),
      label: procedure.signature || procedure.name,
      isLeaf: true,
      db,
      metadata,
      children: undefined,
    };
    map.set(procedureNode.key, procedureNode);
    return procedureNode;
  });
}

function buildFunctionNodeList(
  functions: FunctionMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForGroup<"function">
): TreeNodeForFunction[] {
  return functions.map((func) => {
    const { db } = parent;
    const metadata = { ...parent.metadata, function: func };
    const functionNode: TreeNodeForFunction = {
      type: "function",
      key: keyForResource(db, metadata),
      label: func.signature || func.name,
      isLeaf: true,
      db,
      metadata,
      children: undefined,
    };
    map.set(functionNode.key, functionNode);
    return functionNode;
  });
}

function buildColumnNodeList(
  columns: ColumnMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForTable
): TreeNodeForColumn[] {
  return columns.map((column) => {
    const { db } = parent;
    const metadata = { ...parent.metadata, column };
    const columnNode: TreeNodeForColumn = {
      type: "column",
      key: keyForResource(db, metadata),
      label: column.name,
      db,
      metadata,
      isLeaf: true,
      children: undefined,
    };
    map.set(columnNode.key, columnNode);
    return columnNode;
  });
}

function buildPlaceholderNode<T extends PlaceholderNodeType>(
  placeholder: T,
  parent: TreeNode,
  map: Map<string, TreeNode>
): TreeNodeForPlaceholder<T> {
  const node: TreeNodeForPlaceholder<T> = {
    type: "placeholder",
    key: `${parent.key}/placeholder`,
    placeholder,
    label: "",
    children: undefined,
    isLeaf: true,
  };
  map.set(node.key, node);
  return node;
}
