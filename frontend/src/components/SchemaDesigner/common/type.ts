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
  schema: string;
  table: string;
  selectedSubtab?: string;
}

export type TabContext = TableTabContext;

export interface SchemaDesignerTabState {
  tabMap: Map<string, TabContext>;
  currentTabId?: string;
}

export interface SchemaDesignerContext {
  baselineMetadata: DatabaseMetadata;
  engine: Engine;

  metadata: Ref<DatabaseMetadata>;
  tabState: Ref<SchemaDesignerTabState>;

  // Tab related functions.
  getCurrentTab: () => TabContext | undefined;
  addTab: (tab: TabContext, setAsCurrentTab?: boolean) => void;

  // Schema related functions.
  dropSchema: (schema: string) => void;

  // Table related functions.
  dropTable: (schema: string, table: string) => void;
}
