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

export type TabContext = DatabaseTabContext | TableTabContext;

export type CoreTabContext =
  | Omit<DatabaseTabContext, "id">
  | Omit<TableTabContext, "id">;

export type EditStatus = "normal" | "created" | "dropped" | "updated";
