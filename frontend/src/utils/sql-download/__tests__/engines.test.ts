import { describe, expect, it } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQL_ENGINE_QUOTES } from "../engines";

describe("SQL_ENGINE_QUOTES engine coverage", () => {
  // Mirrors backend SQLStatementPrefix in backend/component/export/sql.go:55-64.
  // serializeSQL throws UnsupportedFormat for any engine not in the map; this
  // truth table catches accidental engine removals.
  it.each([
    Engine.MYSQL,
    Engine.MARIADB,
    Engine.TIDB,
    Engine.OCEANBASE,
    Engine.SPANNER,
    Engine.CLICKHOUSE,
    Engine.MSSQL,
    Engine.ORACLE,
    Engine.POSTGRES,
    Engine.REDSHIFT,
    Engine.SQLITE,
    Engine.SNOWFLAKE,
  ])("supports engine %d", (engine) => {
    expect(SQL_ENGINE_QUOTES.has(engine)).toBe(true);
  });

  it.each([
    Engine.ENGINE_UNSPECIFIED,
    Engine.MONGODB,
    Engine.REDIS,
    Engine.ELASTICSEARCH,
    Engine.DYNAMODB,
    Engine.COSMOSDB,
    Engine.CASSANDRA,
    Engine.STARROCKS,
    Engine.DORIS,
    Engine.HIVE,
    Engine.BIGQUERY,
    Engine.COCKROACHDB,
    Engine.DATABRICKS,
    Engine.TRINO,
  ])("does not support engine %d", (engine) => {
    expect(SQL_ENGINE_QUOTES.has(engine)).toBe(false);
  });
});

describe("SQL_ENGINE_QUOTES", () => {
  it("uses backtick for MySQL family", () => {
    expect(SQL_ENGINE_QUOTES.get(Engine.MYSQL)).toBe("`");
    expect(SQL_ENGINE_QUOTES.get(Engine.MARIADB)).toBe("`");
    expect(SQL_ENGINE_QUOTES.get(Engine.TIDB)).toBe("`");
  });
  it("uses double-quote for Postgres family", () => {
    expect(SQL_ENGINE_QUOTES.get(Engine.POSTGRES)).toBe(`"`);
    expect(SQL_ENGINE_QUOTES.get(Engine.REDSHIFT)).toBe(`"`);
  });
  it("returns undefined for unsupported engine", () => {
    expect(SQL_ENGINE_QUOTES.get(Engine.MONGODB)).toBeUndefined();
  });
});
