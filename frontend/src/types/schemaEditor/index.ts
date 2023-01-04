import { Database, DatabaseId } from "..";
import { Schema } from "./atomType";

export enum SchemaEditorTabType {
  TabForDatabase = "database",
  TabForTable = "table",
}

// Tab context for editing database.
export interface DatabaseTabContext {
  id: string;
  type: SchemaEditorTabType.TabForDatabase;
  databaseId: DatabaseId;
  selectedSchemaName?: string;
}

// Tab context for editing table.
export interface TableTabContext {
  id: string;
  type: SchemaEditorTabType.TabForTable;
  databaseId: DatabaseId;
  schemaName: string;
  tableId: string;
}

export type TabContext = DatabaseTabContext | TableTabContext;

type TabId = string;

export interface DatabaseSchema {
  database: Database;
  schemaList: Schema[];
  originSchemaList: Schema[];
}

export interface SchemaEditorState {
  tabState: {
    tabMap: Map<TabId, TabContext>;
    currentTabId?: TabId;
  };
  databaseSchemaById: Map<DatabaseId, DatabaseSchema>;
}

/**
 * Type definition for API message.
 */
export interface DatabaseEdit {
  databaseId: DatabaseId;

  createSchemaList: CreateSchemaContext[];
  createTableList: CreateTableContext[];
  alterTableList: AlterTableContext[];
  renameTableList: RenameTableContext[];
  dropTableList: DropTableContext[];
}

export interface CreateSchemaContext {
  schema: string;
}

export interface CreateTableContext {
  schema: string;
  name: string;
  engine: string;
  characterSet: string;
  collation: string;
  comment: string;

  addColumnList: AddColumnContext[];
  primaryKeyList: string[];
  addForeignKeyList: AddForeignKeyContext[];
}

export interface AlterTableContext {
  schema: string;
  name: string;

  addColumnList: AddColumnContext[];
  alterColumnList: AlterColumnContext[];
  changeColumnList: ChangeColumnContext[];
  dropColumnList: DropColumnContext[];
  dropPrimaryKey: boolean;
  dropPrimaryKeyList: string[];
  primaryKeyList?: string[];
  dropForeignKeyList: string[];
  addForeignKeyList: AddForeignKeyContext[];
}

export interface RenameTableContext {
  schema: string;
  oldName: string;
  newName: string;
}

export interface DropTableContext {
  schema: string;
  name: string;
}

export interface AddColumnContext {
  name: string;
  type: string;
  characterSet: string;
  collation: string;
  comment: string;
  nullable: boolean;
  default?: string;
}

export interface AlterColumnContext {
  oldName: string;
  newName: string;
  type?: string;
  comment?: string;
  nullable?: boolean;
  defaultChanged: boolean;
  default?: string;
}

export interface ChangeColumnContext {
  oldName: string;
  newName: string;
  type: string;
  characterSet: string;
  collation: string;
  comment: string;
  nullable: boolean;
  default?: string;
}

export interface DropColumnContext {
  name: string;
}

export interface AddForeignKeyContext {
  columnList: string[];
  referencedSchema: string;
  referencedTable: string;
  referencedColumnList: string[];
}

/**
 * Type definition for DatabaseEdit validation API message.
 */
export interface ValidateResult {
  type: string;
  message: string;
}

export interface DatabaseEditResult {
  statement: string;
  validateResultList: ValidateResult[];
}
