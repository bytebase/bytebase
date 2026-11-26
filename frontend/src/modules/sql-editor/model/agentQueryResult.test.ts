import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { QueryResponseSchema } from "@/types/proto-es/v1/sql_service_pb";
import { defaultSQLEditorTab } from "@/utils";
import { useSQLEditorStore } from "../store";
import { getSQLEditorTabsState, useSQLEditorTabsStore } from "../store/tab";
import { captureAgentQueryResultTarget } from "./agentQueryResult";
import { sqlEditorEvents } from "./events";

const database = "instances/demo/databases/example";
const project = "projects/example";
const params: SQLEditorQueryParams = {
  connection: { instance: "instances/demo", database },
  statement: " SELECT 1 ",
  engine: Engine.POSTGRES,
  explain: false,
  selection: null,
  limit: 500,
};
function setup() {
  const tab = { ...defaultSQLEditorTab(), connection: params.connection };
  useSQLEditorTabsStore.setState({
    currentTabId: tab.id,
    tabsById: new Map([[tab.id, tab]]),
  });
  const publish = captureAgentQueryResultTarget(tab.id, database, project)!;
  const merge = vi
    .spyOn(useSQLEditorStore.getState(), "mergeLatest")
    .mockResolvedValue(undefined as never);
  const event = vi.spyOn(sqlEditorEvents, "emit");
  return { tab, publish, merge, event };
}
afterEach(() => vi.restoreAllMocks());
test("publishes the full typed result and masking and access-grant metadata once in its original tab", async () => {
  const { tab, publish, merge, event } = setup();
  const response = create(QueryResponseSchema, {
    appliedAccessGrant: "accessGrants/example",
    results: [
      {
        columnNames: ["v"],
        masked: [{}],
        rowsCount: 120n,
        rows: Array.from({ length: 120 }, () => ({ values: [] })),
      },
    ],
  });
  expect(publish(params, { kind: "response", response })).toBe(true);
  expect(publish(params, { kind: "response", response })).toBe(false);
  const results = getSQLEditorTabsState()
    .tabsById.get(tab.id)
    ?.databaseQueryContexts?.get(database);
  expect(results).toHaveLength(1);
  expect(results![0]).toMatchObject({
    status: "DONE",
    params,
    resultSet: {
      error: "",
      appliedAccessGrant: response.appliedAccessGrant,
      results: response.results,
    },
  });
  expect(results![0].resultSet!.results[0].rows).toHaveLength(120);
  expect(merge).toHaveBeenCalledExactlyOnceWith({ project, database });
  await vi.waitFor(() =>
    expect(event).toHaveBeenCalledExactlyOnceWith("query-executed", {
      project,
      database,
    })
  );
});
test("a known RPC error uses native status; uncertainty, newer results and another tab never overwrite it", () => {
  const { tab, publish, merge } = setup();
  expect(publish(params, { kind: "uncertain" })).toBe(false);
  const old = captureAgentQueryResultTarget(tab.id, database, project)!;
  const error = {
    kind: "error" as const,
    error: new ConnectError("Denied", Code.PermissionDenied),
  };
  expect(publish(params, error)).toBe(true);
  expect(old(params, error)).toBe(false);
  expect(
    getSQLEditorTabsState()
      .tabsById.get(tab.id)
      ?.databaseQueryContexts?.get(database)?.[0].resultSet?.status
  ).toBe(Code.PermissionDenied);
  const newer = captureAgentQueryResultTarget(tab.id, database, project)!;
  useSQLEditorTabsStore.setState({ currentTabId: "another-tab" });
  expect(newer(params, error)).toBe(false);
  expect(merge).toHaveBeenCalledTimes(1);
});
