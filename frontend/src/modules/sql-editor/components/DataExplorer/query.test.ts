import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  getDataExplorerFilterPlaceholderKey,
  getDataExplorerQueryPrefix,
  getDataExplorerStatement,
  type DataExplorerQueryTarget,
} from "./query";

const target = (
  engine: Engine,
  schema = "public",
  table = "users"
): DataExplorerQueryTarget => ({ engine, schema, table });

describe("data explorer query", () => {
  test("builds a CosmosDB query with an optional predicate", () => {
    const cosmos = target(Engine.COSMOSDB, "", "users");
    expect(getDataExplorerQueryPrefix(cosmos)).toBe("SELECT * FROM c");
    expect(getDataExplorerStatement(cosmos, "")).toBe("SELECT * FROM c");
    expect(getDataExplorerStatement(cosmos, "  WHERE c.id = '1';  ")).toBe(
      "SELECT * FROM c WHERE c.id = '1'"
    );
  });

  test("quotes relational targets and keeps preview queries bounded", () => {
    const postgres = target(Engine.POSTGRES);
    expect(getDataExplorerQueryPrefix(postgres)).toBe(
      'SELECT * FROM "public"."users"'
    );
    expect(getDataExplorerStatement(postgres, "WHERE active = true")).toBe(
      'SELECT * FROM "public"."users" WHERE active = true LIMIT 50;'
    );

    const mssql = target(Engine.MSSQL, "dbo");
    expect(getDataExplorerStatement(mssql, "WHERE active = 1")).toBe(
      "SELECT TOP 50 * FROM [dbo].[users] WHERE active = 1;"
    );

    const oracle = target(Engine.ORACLE, "APP");
    expect(getDataExplorerStatement(oracle, "WHERE active = 1")).toBe(
      'SELECT * FROM (SELECT * FROM "APP"."users" WHERE active = 1) WHERE ROWNUM <= 50;'
    );
  });

  test("builds MongoDB and Elasticsearch queries from their filter syntax", () => {
    const mongodb = target(Engine.MONGODB, "", 'user"events');
    expect(getDataExplorerQueryPrefix(mongodb)).toBe(
      'db["user\\"events"].find('
    );
    expect(
      getDataExplorerStatement(mongodb, '{ "active": true }')
    ).toBe('db["user\\"events"].find({ "active": true }).limit(50);');

    const elasticsearch = target(Engine.ELASTICSEARCH, "", "events");
    expect(getDataExplorerStatement(elasticsearch, "")).toBe(
      "GET events/_search?q=*&size=50"
    );
    expect(getDataExplorerStatement(elasticsearch, "status:active")).toBe(
      "GET events/_search?q=status:active&size=50"
    );
  });

  test("provides engine-specific filter guidance", () => {
    expect(getDataExplorerFilterPlaceholderKey(Engine.COSMOSDB)).toBe(
      "sql-editor.data-explorer-filter-placeholder"
    );
    expect(getDataExplorerFilterPlaceholderKey(Engine.MONGODB)).toBe(
      "sql-editor.data-explorer-mongodb-filter-placeholder"
    );
    expect(getDataExplorerFilterPlaceholderKey(Engine.ELASTICSEARCH)).toBe(
      "sql-editor.data-explorer-elasticsearch-filter-placeholder"
    );
    expect(getDataExplorerFilterPlaceholderKey(Engine.POSTGRES)).toBe(
      "sql-editor.data-explorer-sql-filter-placeholder"
    );
  });

  test("rejects engines without table previews", () => {
    expect(getDataExplorerQueryPrefix(target(Engine.REDIS))).toBeUndefined();
    expect(
      getDataExplorerStatement(target(Engine.ENGINE_UNSPECIFIED), "")
    ).toBeUndefined();
  });
});
