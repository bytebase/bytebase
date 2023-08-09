import { Engine } from "@/types/proto/v1/common";
import { ChangeHistory } from "@/types/proto/v1/database_service";

export type SourceSchemaType =
  | "SCHEMA_HISTORY_VERSION"
  | "SCHEMA_DESIGN"
  | "RAW_SQL";

export interface ChangeHistorySourceSchema {
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

export interface RawSQLState {
  projectId?: string;
  engine: Engine;
  statement: string;
  sheetId?: number;
}
