import { TreeOption } from "naive-ui";
import { ComposedDatabase, ComposedInstance } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { groupBy } from "@/utils";
import { keyForResource } from "../context/common";
import { EditTarget } from "../types";

export interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

export interface TreeNodeForInstance extends BaseTreeNode {
  type: "instance";
  instance: ComposedInstance;
  children: TreeNodeForDatabase[];
}

export interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  db: ComposedDatabase;
  parent?: TreeNodeForInstance;
  metadata: {
    database: DatabaseMetadata;
  };
  children: TreeNodeForSchema[];
}

export interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  db: ComposedDatabase;
  parent?: TreeNodeForDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
  children: TreeNodeForTable[];
}

export interface TreeNodeForTable extends BaseTreeNode {
  type: "table";
  db: ComposedDatabase;
  parent: TreeNodeForSchema;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
  children: TreeNodeForColumn[];
}

export interface TreeNodeForColumn extends BaseTreeNode {
  type: "column";
  db: ComposedDatabase;
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

export type TreeNode =
  | TreeNodeForInstance
  | TreeNodeForDatabase
  | TreeNodeForSchema
  | TreeNodeForTable
  | TreeNodeForColumn;

export type BuildTreeOptions = {
  byInstance: boolean;
};
export const buildTree = (
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
    (target) => target.database.instance
  );
  return Array.from(groupedByInstance).map(([_, targets]) => {
    const instance = targets[0].database.instanceEntity;
    const instanceNode: TreeNodeForInstance = {
      type: "instance",
      key: instance.name,
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
      label: db.databaseName,
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
  db: ComposedDatabase,
  database: DatabaseMetadata,
  parent: TreeNodeForDatabase | undefined
) => {
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
    map.set(schemaNode.key, schemaNode);
    schemaNode.children = buildTableNodeList(schema.tables, map, schemaNode);
    return schemaNode;
  });
};

const buildTableNodeList = (
  tables: TableMetadata[],
  map: Map<string, TreeNode>,
  parent: TreeNodeForSchema
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
    return tableNode;
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
