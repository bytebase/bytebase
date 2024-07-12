import type { Engine } from "@/types/proto/v1/common";
import type { ChangeHistory } from "@/types/proto/v1/database_service";

export type SourceSchemaType = "SCHEMA_HISTORY_VERSION" | "RAW_SQL";

export interface ChangeHistorySourceSchema {
  projectName?: string;
  environmentName?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
  conciseHistory?: string;
  isFetching?: boolean;
}

export interface RawSQLState {
  projectName?: string;
  engine: Engine;
  statement: string;
  sheetId?: number;
}
