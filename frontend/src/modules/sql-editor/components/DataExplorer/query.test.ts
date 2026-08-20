import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  type DataExplorerQueryTarget,
  getDataExplorerQueryPrefix,
  getDataExplorerStatement,
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
    expect(getDataExplorerStatement(cosmos, "", 500)).toBe("SELECT * FROM c");
    expect(getDataExplorerStatement(cosmos, "  WHERE c.id = '1';  ", 500)).toBe(
      "SELECT * FROM c WHERE c.id = '1'"
    );
  });

  test("quotes relational targets and leaves row limiting to the request", () => {
    const postgres = target(Engine.POSTGRES);
    expect(getDataExplorerQueryPrefix(postgres)).toBe(
      'SELECT * FROM "public"."users"'
    );
    expect(getDataExplorerStatement(postgres, "WHERE active = true", 500)).toBe(
      'SELECT * FROM "public"."users" WHERE active = true;'
    );

    const mssql = target(Engine.MSSQL, "dbo");
    expect(getDataExplorerStatement(mssql, "WHERE active = 1", 500)).toBe(
      "SELECT * FROM [dbo].[users] WHERE active = 1;"
    );

    const oracle = target(Engine.ORACLE, "APP");
    expect(getDataExplorerStatement(oracle, "WHERE active = 1", 500)).toBe(
      'SELECT * FROM "APP"."users" WHERE active = 1;'
    );
  });

  test("builds MongoDB and Elasticsearch queries from their filter syntax", () => {
    const mongodb = target(Engine.MONGODB, "", 'user"events');
    expect(getDataExplorerQueryPrefix(mongodb)).toBe(
      'db["user\\"events"].find('
    );
    expect(getDataExplorerStatement(mongodb, '{ "active": true }', 500)).toBe(
      'db["user\\"events"].find({ "active": true });'
    );

    const elasticsearch = target(Engine.ELASTICSEARCH, "", "events");
    expect(getDataExplorerStatement(elasticsearch, "", 500)).toBe(
      "GET events/_search?q=*&size=500"
    );
    expect(getDataExplorerStatement(elasticsearch, "status:active", 500)).toBe(
      "GET events/_search?q=status:active&size=500"
    );
  });

  test("rejects engines without table previews", () => {
    expect(getDataExplorerQueryPrefix(target(Engine.REDIS))).toBeUndefined();
    expect(
      getDataExplorerStatement(target(Engine.ENGINE_UNSPECIFIED), "", 500)
    ).toBeUndefined();
  });
});
