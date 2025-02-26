import { Engine } from "@/types/proto/v1/common";

export enum SourceSchemaType {
  SCHEMA_HISTORY_VERSION,
  RAW_SQL,
}

export interface ChangelogSourceSchema {
  environmentName?: string;
  databaseName?: string;
  changelogName?: string;
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
