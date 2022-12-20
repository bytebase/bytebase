import { Database, DatabaseId } from "..";
import { Table } from "./atomType";

export enum SchemaEditorTabType {
  TabForDatabase = "database",
  TabForTable = "table",
}

// Tab context for editing database.
export interface DatabaseTabContext {
  id: string;
  type: SchemaEditorTabType.TabForDatabase;
  databaseId: DatabaseId;
}

// Tab context for editing table.
export interface TableTabContext {
  id: string;
  type: SchemaEditorTabType.TabForTable;
  databaseId: DatabaseId;
  tableName: string;
}

export type TabContext = DatabaseTabContext | TableTabContext;

type TabId = string;

interface DatabaseState {
  database: Database;
  originTableList: Table[];
  tableList: Table[];
}

export interface SchemaEditorState {
  tabState: {
    tabMap: Map<TabId, TabContext>;
    currentTabId?: TabId;
  };
  databaseStateById: Map<DatabaseId, DatabaseState>;
}

/**
 * Type definition for API message.
 */
export interface DatabaseEdit {
  databaseId: DatabaseId;

  createTableList: CreateTableContext[];
  alterTableList: AlterTableContext[];
  renameTableList: RenameTableContext[];
  dropTableList: DropTableContext[];
}

export interface CreateTableContext {
  name: string;
  engine: string;
  characterSet: string;
  collation: string;
  comment: string;

  addColumnList: AddColumnContext[];
}

export interface AlterTableContext {
  name: string;

  addColumnList: AddColumnContext[];
  changeColumnList: ChangeColumnContext[];
  dropColumnList: DropColumnContext[];
}

export interface RenameTableContext {
  oldName: string;
  newName: string;
}

export interface DropTableContext {
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
