import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  DatabaseMetadata,
  FunctionMetadata,
  ProcedureMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";

export type EditTarget = {
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  baselineMetadata: DatabaseMetadata;
};

export type TabType = "database" | "table" | "view" | "procedure" | "function";

export type CommonTabContext = {
  id: string;
  type: TabType;
};

// Tab context for editing database.
export type DatabaseTabContext = CommonTabContext & {
  type: "database";
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
  };
  selectedSchema?: string;
};

// Tab context for editing table.
export type TableTabContext = CommonTabContext & {
  type: "table";
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table: TableMetadata;
  };
};

export type ViewTabContext = CommonTabContext & {
  type: "view";
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    view: ViewMetadata;
  };
};

export type ProcedureTabContext = CommonTabContext & {
  type: "procedure";
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    procedure: ProcedureMetadata;
  };
};

export type FunctionTabContext = CommonTabContext & {
  type: "function";
  database: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    function: FunctionMetadata;
  };
};

export type TabContext =
  | DatabaseTabContext
  | TableTabContext
  | ViewTabContext
  | ProcedureTabContext
  | FunctionTabContext;

export type CoreTabContext =
  | Omit<DatabaseTabContext, "id">
  | Omit<TableTabContext, "id">
  | Omit<ViewTabContext, "id">
  | Omit<ProcedureTabContext, "id">
  | Omit<FunctionTabContext, "id">;

export type EditStatus = "normal" | "created" | "dropped" | "updated";

/**
 * Only tables are selectable rollout objects by now
 */
export type RolloutObject = {
  db: ComposedDatabase;
  metadata: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    table?: TableMetadata;
    column?: ColumnMetadata;
    view?: ViewMetadata;
    procedure?: ProcedureMetadata;
    function?: FunctionMetadata;
  };
};
