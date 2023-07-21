import { Schema, Table } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { Ref } from "vue";

export enum SchemaDesignerTabType {
  TabForTable = "table",
}

// Tab context for editing table.
export interface TableTabContext {
  id: string;
  type: SchemaDesignerTabType.TabForTable;
  schemaId: string;
  tableId: string;
  selectedSubtab?: string;
}

export type TabContext = TableTabContext;

export interface SchemaDesignerTabState {
  tabMap: Map<string, TabContext>;
  currentTabId?: string;
}

export interface SchemaDesignerContext {
  readonly: Ref<boolean>;
  baselineMetadata: Ref<DatabaseMetadata>;
  engine: Ref<Engine>;
  metadata: Ref<DatabaseMetadata>;
  tabState: Ref<SchemaDesignerTabState>;

  originalSchemas: Schema[];
  editableSchemas: Ref<Schema[]>;

  // Tab related functions.
  getCurrentTab: () => TabContext | undefined;
  addTab: (tab: TabContext, setAsCurrentTab?: boolean) => void;

  // Table related functions.
  getTable: (schemaId: string, tableId: string) => Table;
  dropTable: (schemaId: string, tableId: string) => void;
}
