import { TreeOption } from "naive-ui";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

export interface BaseTreeNode extends TreeOption {
  key: string;
  label: string;
  isLeaf: boolean;
  children?: TreeNode[];
}

export interface TreeNodeForDatabase extends BaseTreeNode {
  type: "database";
  db: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
  };
}

export interface TreeNodeForSchema extends BaseTreeNode {
  type: "schema";
  db: ComposedDatabase;
  parent?: TreeNodeForDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
  };
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
}

export type TreeNode =
  | TreeNodeForDatabase
  | TreeNodeForSchema
  | TreeNodeForTable
  | TreeNodeForColumn;
