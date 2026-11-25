import { afterEach, expect, test, vi } from "vitest";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import { useSQLEditorEditorStore } from "../store/editor";
import { useSQLEditorTabsStore } from "../store/tab";
import { captureSqlEditorContext } from "./sqlEditorAgentContext";

afterEach(() => {
  useSQLEditorEditorStore.setState(
    useSQLEditorEditorStore.getInitialState(),
    true
  );
  useSQLEditorTabsStore.setState(useSQLEditorTabsStore.getInitialState(), true);
  window.history.replaceState(null, "", "/");
  localStorage.clear();
});

test("does not source Project or database identity from the URL or a leftover tab", () => {
  window.history.replaceState(
    null,
    "",
    "/sql-editor/projects/alpha/instances/shared/databases/app"
  );
  useSQLEditorTabsStore.getState().addTab({
    connection: {
      instance: "instances/shared",
      database: "instances/shared/databases/app",
    },
  });
  useSQLEditorEditorStore.getState().setProject("");
  expect(captureSqlEditorContext()).toEqual({
    kind: "ready",
    snapshot: { version: 1 },
  });
});

test("does not reinterpret a malformed selected Project as Workspace help", () => {
  useSQLEditorTabsStore.setState({
    initProject: vi.fn().mockResolvedValue(undefined),
  });
  useSQLEditorEditorStore.getState().setProject("projects/alpha/databases/app");
  expect(captureSqlEditorContext()).toEqual({ kind: "pending" });
});

test("omits an invalid persisted row limit without dropping the tab or valid query options", () => {
  useSQLEditorTabsStore.setState({
    initProject: vi.fn().mockResolvedValue(undefined),
  });
  useSQLEditorEditorStore.setState({
    project: "projects/alpha",
    projectContextReady: true,
    resultRowsLimit: 8.5,
    redisCommandOption: QueryOption_RedisRunCommandsOn.ALL_NODES,
  });
  const tab = useSQLEditorTabsStore.getState().addTab();
  expect(captureSqlEditorContext()).toEqual({
    kind: "ready",
    snapshot: {
      version: 1,
      project: "projects/alpha",
      tab: { id: tab.id, mode: "SAVED_QUERY" },
      batchActive: false,
      queryOptions: {
        redisRunCommandsOn: QueryOption_RedisRunCommandsOn.ALL_NODES,
      },
    },
  });
});
