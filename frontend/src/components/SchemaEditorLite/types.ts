import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

export type EditTarget = {
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  baselineMetadata: DatabaseMetadata;
};

export type ResourceType = "branch" | "database";

export type TabType = "database" | "table";

// Tab context for editing database.
export type DatabaseTabContext = {
  id: string;
  type: "database";
  database: ComposedDatabase;
};

// export interface DatabaseTabContext {
//   id: string;
//   type: SchemaEditorTabType.TabForDatabase;
//   // Parent could be a database.
//   parentName: string;
//   selectedSchemaId?: string;
// }

// Tab context for editing table.
export type TableTabContext = {
  id: string;
  type: "table";
  schema: SchemaMetadata;
  table: TableMetadata;
};

// export interface TableTabContext {
//   id: string;
//   type: SchemaEditorTabType.TabForTable;
//   // Parent could be a database or a branch.
//   parentName: string;
//   schemaId: string;
//   tableId: string;
// }

// export type TabContext = {
//   name?: string;
// } & (DatabaseTabContext | TableTabContext);

// export interface DatabaseSchema {
//   database: ComposedDatabase;
//   schemaList: Schema[];
//   originSchemaList: Schema[];
// }

// export interface BranchSchema {
//   branch: Branch;
//   schemaList: Schema[];
//   originSchemaList: Schema[];
// }

// export interface SchemaEditorV1State {
//   project: ComposedProject;
//   readonly: boolean;
//   resourceType: "database" | "branch";
//   resourceMap: {
//     database: Map<string, DatabaseSchema>;
//     branch: Map<string, BranchSchema>;
//   };
//   tabState: {
//     tabMap: Map<string, TabContext>;
//     currentTabId?: string;
//   };
// }
