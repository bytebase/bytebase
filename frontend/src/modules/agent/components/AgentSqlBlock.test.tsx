import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { SHA256 } from "crypto-js";
import { isEqual } from "lodash-es";
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
  connected = false,
  executed = false
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
  let captured = snapshot;
  const inspect = vi.fn<NonNullable<AgentSqlEditorHost["inspect"]>>((target) => ({ status: isEqual(target, captured.editorTarget) ? "ready" : "stale" }));
  const host: AgentSqlEditorHost = {
    id: "host", slot, mainColumn: slot, resizer: slot, paneRatio: 0.3, inspect, applySql,
    captureContext: () => ({ kind: "ready", snapshot: captured }),
  };
  let unregister = registerAgentSqlEditorHost(host);
  const onRun = vi.fn(async () => true);
  const view = render(
    <AgentSqlBlock source={source} sql={sql} onRun={onRun} executed={executed}>
      <pre>Highlighted SQL</pre>
    </AgentSqlBlock>
  );
  return {
    ...view,
    snapshot,
    source,
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
  test("collapses executed SQL while keeping explicit Run again available", async () => {
    const view = prepareBlock({ start: 2, end: 2, caret: 2 }, true, true);
    try {
      const details = view.getByRole("button", { name: "agent.sql-block.executed-sql" });
      expect(details).toHaveAttribute("aria-expanded", "false");
      expect(view.queryByText("Highlighted SQL")).not.toBeInTheDocument();
      expect(view.queryByRole("button", { name: "agent.sql-block.run-query" })).not.toBeInTheDocument();
      const run = view.getByRole("button", { name: "agent.sql-block.run-again" });
      expect(run).toBeEnabled();
      await act(async () => fireEvent.click(run));
      expect(view.onRun).toHaveBeenCalledExactlyOnceWith(view.source);
      fireEvent.click(details);
      expect(details).toHaveAttribute("aria-expanded", "true");
      expect(view.getByText("Highlighted SQL")).toBeVisible();
    } finally { view.dispose(); }
  });

  test("keeps compact, labeled icon actions in the SQL block header", () => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 }, true);
    try {
      const block = view.container.querySelector("[data-agent-sql-block]")!;
      const toolbar = block.querySelector("[data-agent-sql-toolbar]");
      expect(toolbar).not.toBeNull();
      const actions = view.getByRole("group", { name: "agent.sql-block.actions" });
      expect(toolbar).toContainElement(actions);
      expect(toolbar!.compareDocumentPosition(block.querySelector("pre")!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
      for (const label of ["copy", "insert", "replace-selection", "run-query"]) {
        const button = view.getByRole("button", { name: `agent.sql-block.${label}` });
        expect(actions).toContainElement(button);
        expect(button.textContent).toBe("");
        expect(button).toHaveClass("h-6");
        expect(button.querySelector("svg")).toHaveClass("size-3.5");
      }
    } finally { view.dispose(); }
  });

  test.each([
    { changed: { rowLimit: 25 }, runEnabled: true },
    { changed: { queryOptions: { explainFormat: 0 } }, runEnabled: true },
    { changed: { batchActive: true }, runEnabled: false },
  ])("checks current Run settings independently of editor actions: %j", async ({ changed, runEnabled }) => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 }, true);
    try {
      const run = view.getByRole("button", { name: "agent.sql-block.run-query" });
      expect(run).toBeEnabled();
      act(() => view.publish({ ...view.snapshot, ...changed }));
      expect(run).toHaveProperty("disabled", !runEnabled);
      expect(view.getByRole("button", { name: "agent.sql-block.insert" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.replace-selection" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.copy" })).toBeEnabled();
      await act(async () => fireEvent.click(run));
      expect(view.onRun).toHaveBeenCalledTimes(runEnabled ? 1 : 0);
      act(() => view.publish(view.snapshot));
      expect(run).toBeEnabled();
    } finally {
      view.dispose();
    }
  });

  test.each(["caret", "edit", "remount", "new-tab"])("Insert and Run stay usable after %s while Replace remains strict", (change) => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 }, true);
    try {
      const original = structuredClone(view.source);
      const tab = change === "new-tab" ? { id: "new-tab", mode: "SAVED_QUERY" as const } : view.snapshot.tab;
      const target = {
        ...view.snapshot.editorTarget!,
        ...(change === "remount" ? { mountId: "remounted", modelUri: "inmemory://remounted" } : {}),
        ...(change === "edit" ? { modelVersion: 8 } : {}),
        ...(change === "new-tab" ? { tabId: "new-tab", mountId: "another-mount", modelUri: "inmemory://new-tab" } : {}),
        selection: { start: 10, end: 10, caret: 10 },
      };
      act(() => view.publish({ ...view.snapshot, tab, editorTarget: target }));
      expect(view.getByRole("button", { name: "agent.sql-block.insert" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.run-query" })).toBeEnabled();
      expect(view.getByRole("button", { name: "agent.sql-block.replace-selection" })).toBeDisabled();
      fireEvent.click(view.getByRole("button", { name: "agent.sql-block.insert" }));
      expect(view.applySql).toHaveBeenCalledExactlyOnceWith({ target, sql: view.sql, mode: "insert" });
      expect(view.source).toEqual(original);
    } finally { view.dispose(); }
  });

  test.each(["project", "database", "schema", "container", "data-source", "actor", "workspace"])("never retargets an answer across %s", change => {
    const view = prepareBlock({ start: 2, end: 5, caret: 5 }, true);
    try {
      const project = change === "project" ? "projects/other" : view.snapshot.project;
      const connection = { ...view.snapshot.connection!,
        ...(change === "project" ? { project: project! } : {}),
        ...(change === "database" ? { database: "instances/main/databases/other" } : {}),
        ...(change === "schema" ? { querySchema: "other" } : {}),
        ...(change === "container" ? { container: "other" } : {}),
        ...(change === "data-source" ? { dataSourceId: "other" } : {}),
      };
      const target = { ...view.snapshot.editorTarget!, project, connection,
        ...(change === "actor" ? { actor: "users/other" } : {}),
        ...(change === "workspace" ? { workspace: "workspaces/other" } : {}),
      };
      act(() => view.publish({ ...view.snapshot, project, connection, editorTarget: target }));
      expect(view.getByRole("button", { name: "agent.sql-block.insert" })).toBeDisabled();
      expect(view.getByRole("button", { name: "agent.sql-block.run-query" })).toBeDisabled();
      expect(view.getByRole("button", { name: "agent.sql-block.copy" })).toBeEnabled();
      fireEvent.click(view.getByRole("button", { name: "agent.sql-block.insert" }));
      expect(view.applySql).not.toHaveBeenCalled();
    } finally { view.dispose(); }
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
