import { expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { AgentSqlEditorSnapshot } from "./sqlEditorContext";
import { buildAgentSqlRequest } from "./sqlExecution";

const snapshot: AgentSqlEditorSnapshot = {
  version: 1,
  project: "projects/p",
  tab: { id: "t", mode: "SAVED_QUERY" },
  connection: {
    project: "projects/p",
    database: "instances/a/databases/d",
    engine: Engine.POSTGRES,
    querySchema: "public",
    container: "part",
    dataSourceId: "reader",
  },
  rowLimit: 504,
  queryOptions: {},
  batchActive: false,
};
test("Query uses only the captured single connection, options and exact SQL bytes", () => {
  const sql = " SELECT 1;\r\nSELECT 2; ";
  expect(buildAgentSqlRequest(snapshot, sql)).toEqual({
    operationId: "bytebase.v1.SQLService.Query",
    body: {
      name: "instances/a/databases/d",
      statement: sql,
      schema: "public",
      container: "part",
      dataSourceId: "reader",
      queryOption: {},
      limit: 504,
    },
  });
  expect(
    buildAgentSqlRequest({ ...snapshot, batchActive: true }, sql)
  ).toBeUndefined();
  expect(
    buildAgentSqlRequest({ ...snapshot, tab: { id: "t", mode: "ADMIN" } }, sql)
  ).toBeUndefined();
  expect(
    buildAgentSqlRequest({ ...snapshot, project: "projects/foreign" }, sql)
  ).toBeUndefined();
});
