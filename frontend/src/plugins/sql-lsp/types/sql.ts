export const EngineTypesUsingSQL = [
  "MYSQL",
  "CLICKHOUSE",
  "POSTGRES",
  "SNOWFLAKE",
  "TIDB",
  "SPANNER",
] as const;

export type SQLDialect = typeof EngineTypesUsingSQL[number];
