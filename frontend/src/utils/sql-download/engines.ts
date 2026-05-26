import { Engine } from "@/types/proto-es/v1/common_pb";

/**
 * Mirrors backend SQLStatementPrefix in backend/component/export/sql.go:55-64.
 * Adding an engine here without adding it on the backend will break goldens.
 */
export const SQL_ENGINE_QUOTES: ReadonlyMap<Engine, "`" | '"'> = new Map([
  [Engine.MYSQL, "`"],
  [Engine.MARIADB, "`"],
  [Engine.TIDB, "`"],
  [Engine.OCEANBASE, "`"],
  [Engine.SPANNER, "`"],
  [Engine.CLICKHOUSE, '"'],
  [Engine.MSSQL, '"'],
  [Engine.ORACLE, '"'],
  [Engine.POSTGRES, '"'],
  [Engine.REDSHIFT, '"'],
  [Engine.SQLITE, '"'],
  [Engine.SNOWFLAKE, '"'],
]);
