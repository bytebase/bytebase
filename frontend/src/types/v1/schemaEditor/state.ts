import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Schema } from "./atomType";

export enum SchemaEditorTabType {
  TabForDatabase = "database",
  TabForTable = "table",
}

// Tab context for editing database.
export interface DatabaseTabContext {
  id: string;
  type: SchemaEditorTabType.TabForDatabase;
  // Parent could be a database.
  parentName: string;
  selectedSchemaId?: string;
}

// Tab context for editing table.
export interface TableTabContext {
  id: string;
  type: SchemaEditorTabType.TabForTable;
  // Parent could be a database.
  parentName: string;
  schemaId: string;
  tableId: string;
}

export type TabContext = {
  name?: string;
} & (DatabaseTabContext | TableTabContext);

export interface DatabaseSchema {
  database: Database;
  schemaList: Schema[];
  originSchemaList: Schema[];
}
