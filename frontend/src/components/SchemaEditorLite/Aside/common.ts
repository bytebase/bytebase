import type { TreeOption } from "naive-ui";
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
import { keyForResource } from "../context/common";
import {
  engineSupportsEditFunctions,
  engineSupportsEditProcedures,
  engineSupportsEditViews,
} from "../spec";
import type { EditTarget } from "../types";

export interface BaseTreeNode extends TreeOption {
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
  parent?: TreeNodeForInstance;
  metadata: {
    database: DatabaseMetadata;
  };
  children:
    | TreeNodeForSchema[] // for multi-schema engines
    | TreeNodeForGroup[]; // for single-schema engines
}

export interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  db: Database;
  parent?: TreeNodeForDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
  children: TreeNodeForGroup[];
}

export interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  db: Database;
  parent: TreeNodeForGroup<"table">;
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
  parent: TreeNodeForTable;
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
  parent: TreeNodeForGroup<"view">;
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
  parent: TreeNodeForGroup<"procedure">;
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
  parent: TreeNodeForGroup<"function">;
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
  parent: TreeNodeForSchema | TreeNodeForDatabase | undefined;
  isLeaf: false;
  children: T extends "table"
    ? TreeNodeForTable[] | TreeNodeForPlaceholder<"table">[]
    : T extends "view"
      ? TreeNodeForView[] | TreeNodeForPlaceholder<"view">[]
      : T extends "procedure"
        ? TreeNodeForProcedure[] | TreeNodeForPlaceholder<"procedure">[]
        : T extends "function"
          ? TreeNodeForFunction[] | TreeNodeForPlaceholder<"function">[]
          : undefined;
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
  parent: TreeNode;
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

export const useBuildTree = () => {
  const buildTree = (
    targets: EditTarget[],
    map: Map<string, TreeNode>,
    options: BuildTreeOptions
  ) => {
    map.clear();
    if (options.byInstance) {
      return buildInstanceNodeList(targets, map);
    } else {
      const target = targets[0];
      return buildSchemaNodeList(
        target.metadata.schemas,
        map,
        target.database,
        target.metadata,
        undefined
      );
    }
  };

  const buildInstanceNodeList = (
    targets: EditTarget[],
    map: Map<string, TreeNode>
  ) => {
    const groupedByInstance = groupBy(
      targets,
      (target) => extractDatabaseResourceName(target.database.name).instance
    );
    return Array.from(groupedByInstance).map(([key, targets]) => {
      const instance = getInstanceResource(targets[0].database);
      const instanceNode: TreeNodeForInstance = {
        type: "instance",
        key: key,
        label: instance.title,
        isLeaf: false,
        instance,
        children: [],
      };
      map.set(instanceNode.key, instanceNode);
      instanceNode.children = buildDatabaseNodeList(targets, map, instanceNode);
      return instanceNode;
    });
  };

  const buildDatabaseNodeList = (
    targets: EditTarget[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForInstance | undefined
  ) => {
    return targets.map((target) => {
      const db = target.database;
      const databaseNode: TreeNodeForDatabase = {
        type: "database",
        key: db.name,
        parent,
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
        target.metadata,
        databaseNode
      );
      return databaseNode;
    });
  };
  const buildSchemaNodeList = (
    schemas: SchemaMetadata[],
    map: Map<string, TreeNode>,
    db: Database,
    database: DatabaseMetadata,
    parent: TreeNodeForDatabase | undefined
  ) => {
    const mapSchemaChildrenNodes = (
      schema: SchemaMetadata,
      parent: TreeNodeForSchema | TreeNodeForDatabase | undefined
    ) => {
      const groups: TreeNodeForGroup[] = [];
      const metadata = {
        database,
        schema,
      };
      // Tables
      const tableGroupNode: TreeNodeForGroup<"table"> = {
        type: "group",
        group: "table",
        key: keyForResource(db, metadata, "table-group"),
        parent,
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
          parent,
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
          parent,
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
          parent,
          db,
          metadata,
          label: "functions",
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
      return mapSchemaChildrenNodes(schemas[0], parent);
    }

    return schemas.map((schema) => {
      const metadata = {
        database,
        schema,
      };
      const schemaNode: TreeNodeForSchema = {
        type: "schema",
        parent: parent,
        key: keyForResource(db, metadata),
        label: schema.name,
        isLeaf: false,
        db,
        metadata,
        children: [],
      };
      schemaNode.children = mapSchemaChildrenNodes(schema, schemaNode);
      map.set(schemaNode.key, schemaNode);
      return schemaNode;
    });
  };

  const buildTableNodeList = (
    tables: TableMetadata[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForGroup<"table">
  ) => {
    return tables.map((table) => {
      const { db } = parent;
      const metadata = {
        ...parent.metadata,
        table,
      };
      const tableNode: TreeNodeForTable = {
        type: "table",
        parent,
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
  };

  const buildViewNodeList = (
    views: ViewMetadata[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForGroup<"view">
  ) => {
    return views.map((view) => {
      const { db } = parent;
      const metadata = {
        ...parent.metadata,
        view,
      };
      const viewNode: TreeNodeForView = {
        type: "view",
        parent,
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
  };

  const buildProcedureNodeList = (
    procedures: ProcedureMetadata[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForGroup<"procedure">
  ) => {
    return procedures.map((procedure) => {
      const { db } = parent;
      const metadata = {
        ...parent.metadata,
        procedure,
      };
      const procedureNode: TreeNodeForProcedure = {
        type: "procedure",
        parent,
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
  };

  const buildFunctionNodeList = (
    functions: FunctionMetadata[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForGroup<"function">
  ) => {
    return functions.map((func) => {
      const { db } = parent;
      const metadata = {
        ...parent.metadata,
        function: func,
      };
      const functionNode: TreeNodeForFunction = {
        type: "function",
        parent,
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
  };

  const buildColumnNodeList = (
    columns: ColumnMetadata[],
    map: Map<string, TreeNode>,
    parent: TreeNodeForTable
  ) => {
    return columns.map((column) => {
      const { db } = parent;
      const metadata = {
        ...parent.metadata,
        column,
      };
      const columnNode: TreeNodeForColumn = {
        type: "column",
        key: keyForResource(db, metadata),
        parent,
        label: column.name,
        db,
        metadata,
        isLeaf: true,
        children: undefined,
      };
      map.set(columnNode.key, columnNode);
      return columnNode;
    });
  };

  const buildPlaceholderNode = <T extends PlaceholderNodeType>(
    placeholder: T,
    parent: TreeNode,
    map: Map<string, TreeNode>
  ) => {
    const node: TreeNodeForPlaceholder<T> = {
      type: "placeholder",
      key: `${parent.key}/placeholder`,
      placeholder,
      parent,
      label: "",
      children: undefined,
      isLeaf: true,
    };
    map.set(node.key, node);
    return node;
  };

  return buildTree;
};
