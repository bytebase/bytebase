export const EngineTypesUsingSQL = [
  "MYSQL",
  "CLICKHOUSE",
  "POSTGRES",
  "SNOWFLAKE",
  "TIDB",
  "SPANNER",
  "OCEANBASE",
] as const;

export type SQLDialect = typeof EngineTypesUsingSQL[number];
