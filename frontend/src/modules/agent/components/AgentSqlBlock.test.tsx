import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { SHA256 } from "crypto-js";
import { afterEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { AgentSqlEditorSnapshot } from "../logic/sqlEditorContext";
import {
  type AgentSqlEditorHost,
  registerAgentSqlEditorHost,
} from "../logic/sqlEditorHost";
import { bindAgentSqlBlockActions } from "../logic/sqlResponse";
import { useAgentStore } from "../store/agent";
import { AgentSqlBlock } from "./AgentSqlBlock";

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);
vi.mock("crypto-js", async (importOriginal) => {
  const actual = await importOriginal<typeof import("crypto-js")>();
  return { ...actual, SHA256: vi.fn(actual.SHA256) };
});
const markdownParses = vi.hoisted(() => vi.fn());
vi.mock("unified", async (importOriginal) => {
  const actual = await importOriginal<typeof import("unified")>();
  return {
    ...actual,
    unified: () => {
      const processor = actual.unified();
      const parse = processor.parse.bind(processor);
      vi.spyOn(processor, "parse").mockImplementation((file) => {
        markdownParses(file);
        return parse(file);
      });
      return processor;
    },
  };
});

const initialState = useAgentStore.getState();
const initialUser = useAppStore.getState().currentUser;
afterEach(() => {
  cleanup();
  useAgentStore.setState(initialState, true);
  useAppStore.setState({ currentUser: initialUser });
  vi.unstubAllGlobals();
});

function prepareBlock(
  selection: { start: number; end: number; caret: number },
  connected = false
) {
  const workspaceName = "workspaces/sample";
  useAppStore.setState({ currentUser: create(UserSchema, { name: "users/member" }) });
  vi.stubGlobal("navigator", { ...navigator, locks: { request: vi.fn() } });
  useAgentStore.setState({ workspaceName, userEmail: "member@example.com" });
  const store = useAgentStore.getState();
  const connection = connected
    ? {
        project: "projects/sample",
        database: "instances/main/databases/sample",
        engine: Engine.POSTGRES,
      }
    : undefined;
  const chat = store.ensureCurrentChat(undefined, connection?.project, workspaceName);
  const snapshot: AgentSqlEditorSnapshot = {
    version: 1,
    ...(connection ? { project: connection.project, connection, rowLimit: 100 } : {}),
    tab: { id: "tab", mode: "SAVED_QUERY" },
    editorTarget: {
      actor: "users/member",
      workspace: workspaceName,
      hostId: "host",
      mountId: "mount",
      tabId: "tab",
      tabMode: "SAVED_QUERY",
      modelUri: "file:///query.sql",
      modelVersion: 3,
      selection,
      ...(connection ? { project: connection.project, connection } : {}),
    },
  };
  store.addMessage({
    chatId: chat.id,
    role: "user",
    content: "Write a query",
    metadata: { sqlEditor: snapshot },
  });
  const sql = "  SELECT\r\n    1;  ";
  store.addMessage({
    chatId: chat.id,
    role: "assistant",
    content: `~~~sql\n${sql}\n~~~`,
    metadata: {
      sqlEditor: snapshot,
      runId: "run",
      sqlActionProvenance: {
        version: 1, runId: "run", runMode: "default", diagnostic: false,
      },
      actions: [{ kind: "sql-editor-block", blockIndex: 0 }],
    },
  });
  const messages = store.getMessages(chat.id);
  const source = bindAgentSqlBlockActions(messages[1], messages)[0]!;
  expect(source).toBeDefined();
  const slot = document.createElement("div");
  document.body.appendChild(slot);
  const applySql = vi.fn<NonNullable<AgentSqlEditorHost["applySql"]>>(() => ({ status: "applied" }));
  const inspect = vi.fn<NonNullable<AgentSqlEditorHost["inspect"]>>(() => ({ status: "ready" }));
  let captured = snapshot;
  const host: AgentSqlEditorHost = {
    id: "host", slot, mainColumn: slot, resizer: slot, paneRatio: 0.3, inspect, applySql,
    captureContext: () => ({ kind: "ready", snapshot: captured }),
  };
  let unregister = registerAgentSqlEditorHost(host);
  const onRun = vi.fn(async () => true);
  const view = render(
    <AgentSqlBlock source={source} sql={sql} onRun={onRun}>
      <pre>Highlighted SQL</pre>
    </AgentSqlBlock>
  );
  return {
    ...view,
    snapshot,
    sql,
    applySql,
    inspect,
    onRun,
    publish(snapshot: AgentSqlEditorSnapshot) {
      captured = snapshot;
      unregister = registerAgentSqlEditorHost(host);
    },
    dispose: () => {
      view.unmount();
      unregister();
      slot.remove();
    },
  };
}

describe("AgentSqlBlock", () => {
  test.each([
    { rowLimit: 25 },
    { queryOptions: { explainFormat: 0 } },
    { batchActive: true },
  ])("disables only Run when live query settings change: %j", (changed) => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 }, true);
    try {
      const run = view.getByRole("button", { name: "agent.sql-block.run-query" });
      expect(run).toBeEnabled();
      act(() => view.publish({ ...view.snapshot, ...changed }));
      expect(run).toBeDisabled();
      expect(view.getByRole("button", { name: "agent.sql-block.insert" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.replace-selection" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.copy" })).toBeEnabled();
      fireEvent.click(run);
      expect(view.onRun).not.toHaveBeenCalled();
      act(() => view.publish(view.snapshot));
      expect(run).toBeEnabled();
    } finally {
      view.dispose();
    }
  });

  test("host changes do not rebind an unchanged answer, but an edit click does", () => {
    const view = prepareBlock({ start: 2, end: 2, caret: 2 });
    try {
      vi.mocked(SHA256).mockClear();
      markdownParses.mockClear();
      const model = view.snapshot.editorTarget!;
      const insert = view.getByRole("button", { name: "agent.sql-block.insert" });
      act(() => view.publish(view.snapshot));
      act(() => view.publish({
        ...view.snapshot,
        editorTarget: { ...model, selection: { start: 3, end: 3, caret: 3 } },
      }));
      view.inspect.mockReturnValue({ status: "stale" });
      act(() => view.publish({
        ...view.snapshot,
        editorTarget: { ...model, modelVersion: 4 },
      }));
      expect(insert).toBeDisabled();
      expect(markdownParses).not.toHaveBeenCalled();
      expect(SHA256).not.toHaveBeenCalled();
      view.inspect.mockReturnValue({ status: "ready" });
      act(() => view.publish(view.snapshot));
      expect(insert).toBeEnabled();
      fireEvent.click(insert);
      expect(markdownParses).toHaveBeenCalledTimes(1);
      expect(SHA256).toHaveBeenCalledTimes(1);
      expect(view.applySql).toHaveBeenCalledTimes(1);
    } finally {
      view.dispose();
    }
  });

  test("hands a connected Run click the exact source without editing the model", async () => {
    const view = prepareBlock({ start: 2, end: 2, caret: 2 }, true);
    try {
      await act(async () => {
        fireEvent.click(view.getByRole("button", { name: "agent.sql-block.run-query" }));
      });
      expect(view.onRun).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({
        sql: "  SELECT\r\n    1;  ",
        blockIndex: 0,
        snapshot: view.snapshot,
      }));
      expect(view.applySql).not.toHaveBeenCalled();
    } finally {
      view.dispose();
    }
  });

  test("passes exact block SQL to a no-database editor and does not offer Run", () => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 });
    try {
      fireEvent.click(view.getByRole("button", { name: "agent.sql-block.insert" }));
      fireEvent.click(view.getByRole("button", { name: "agent.sql-block.replace-selection" }));
      expect(view.applySql.mock.calls).toEqual([
        [{ target: view.snapshot.editorTarget, sql: view.sql, mode: "insert" }],
        [{ target: view.snapshot.editorTarget, sql: view.sql, mode: "replace-selection" }],
      ]);
      expect(view.queryByRole("button", { name: "agent.sql-block.run-query" })).toBeNull();
      expect(view.onRun).not.toHaveBeenCalled();
    } finally {
      view.dispose();
    }
  });

  test("rechecks the target at click and never reinterprets a caret as a selection", () => {
    const view = prepareBlock({ start: 2, end: 2, caret: 2 });
    try {
      expect(view.queryByRole("button", { name: "agent.sql-block.replace-selection" })).toBeNull();
      view.inspect.mockReturnValue({ status: "stale" });
      act(() => fireEvent.click(view.getByRole("button", { name: "agent.sql-block.insert" })));
      expect(view.applySql).not.toHaveBeenCalled();
      expect(view.getByRole("button", { name: "agent.sql-block.copy" })).toBeEnabled();
    } finally {
      view.dispose();
    }
  });
});
