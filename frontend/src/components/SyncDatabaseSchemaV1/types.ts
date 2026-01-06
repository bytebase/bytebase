import { Engine } from "@/types/proto-es/v1/common_pb";

export enum SourceSchemaType {
  SCHEMA_HISTORY_VERSION,
  RAW_SQL,
}

export interface ChangelogSourceSchema {
  environmentName?: string;
  databaseName?: string;
  changelogName?: string;
  targetChangelogName?: string; // For rollback: the previous changelog to compare against
}

export interface RawSQLState {
  engine: Engine;
  statement: string;
}

export const ALLOWED_ENGINES: Engine[] = [
  Engine.MYSQL,
  Engine.POSTGRES,
  Engine.TIDB,
  Engine.ORACLE,
  Engine.MSSQL,
  Engine.COCKROACHDB,
];
