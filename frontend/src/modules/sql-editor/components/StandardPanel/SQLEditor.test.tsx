import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { useEffect, useLayoutEffect, useRef } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { extractSavedQueryConnection } from "@/lib/sqlEditorConnection";
import {
  lazyExtractDomRefSuggestions,
  lazyPrepareDomAction,
} from "@/modules/agent/dom";
import { getElementByRef } from "@/modules/agent/dom/snapshotIndex";
import {
  type AgentSqlEditResult,
  getRegisteredAgentSqlEditorHost,
} from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";
import { useSQLEditorEditorStore } from "@/modules/sql-editor/store/editor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabsStore,
} from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { SavedQuerySchema } from "@/types/proto-es/v1/saved_query_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { SQLEditorAgentHost } from "../SQLEditorAgentHost";
import { useSQLEditorAutoSave } from "../useSQLEditorAutoSave";
import { SQLEditor } from "./SQLEditor";

const mocks = vi.hoisted(() => ({
  createSavedQuery: vi.fn(),
  maybeUpdateSavedQuery: vi.fn(),
  editor: undefined as
    | {
        getDomNode(): Element | null;
        getValue(): string;
        setValue(value: string): void;
      }
    | undefined,
}));
vi.mock("@/components/monaco/MonacoEditor", () => {
  function MonacoEditor({
    content,
    onChange,
    onReady,
  }: {
    content: string;
    onChange: (value: string) => void;
    onReady: (monaco: unknown, editor: unknown) => void;
  }) {
    const node = useRef<HTMLDivElement>(null);
    const value = useRef(content);
    const ready = useRef(onReady);
    const syncModel = useRef<(value: string) => void>(undefined);
    ready.current = onReady;
    useLayoutEffect(() => {
      let version = 1;
      const noSubscription = () => ({ dispose() {} });
      const onModelChange = new Set<() => void>();
      const publishModelChange = () => {
        for (const listener of onModelChange) listener();
      };
      syncModel.current = (next) => {
        if (next === value.current) return;
        value.current = next;
        version++;
        publishModelChange();
      };
      const initialPosition = { lineNumber: 1, column: value.current.length + 1 };
      const model = {
        uri: { toString: () => "inmemory://sql/test" },
        getVersionId: () => version,
        isDisposed: () => false,
        getOffsetAt: ({ column }: { column: number }) => column - 1,
        getPositionAt: (offset: number) => ({ lineNumber: 1, column: offset + 1 }),
        getValue: () => value.current,
        getEOL: () => "\n",
      };
      const editor = {
        getDomNode: () => node.current,
        getValue: model.getValue,
        getModel: () => model,
        getOption: () => false,
        getSelections: () => [{
          getPosition: () => initialPosition,
          getStartPosition: () => initialPosition,
          getEndPosition: () => initialPosition,
        }],
        onDidChangeCursorSelection: noSubscription,
        onDidChangeModelContent: (listener: () => void) => {
          onModelChange.add(listener);
          return { dispose: () => onModelChange.delete(listener) };
        },
        onDidChangeConfiguration: noSubscription,
        onDidChangeModel: noSubscription,
        onDidDispose: noSubscription,
        addAction: noSubscription,
        addCommand: noSubscription,
        pushUndoStop: () => true,
        setPosition() {},
        setValue: onChange,
        executeEdits: (_source: string, edits: { range: { startColumn: number; endColumn: number }; text: string }[]) => {
          const [{ range, text }] = edits;
          version++;
          value.current = value.current.slice(0, range.startColumn - 1) + text + value.current.slice(range.endColumn - 1);
          onChange(value.current);
          publishModelChange();
          return true;
        },
      };
      mocks.editor = editor;
      ready.current({ KeyMod: { CtrlCmd: 1, Shift: 2 }, KeyCode: { Enter: 4, KeyS: 8 }, editor: { EditorOption: { readOnly: 99 } } }, editor);
      return () => {
        mocks.editor = undefined;
        syncModel.current = undefined;
      };
    }, [onChange]);
    useEffect(() => syncModel.current?.(content), [content]);
    return (
      <div className="monaco-editor" ref={node}>
        <textarea
          aria-label="SQL"
          value={content}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      </div>
    );
  }
  return { MonacoEditor };
});
vi.mock("@/components/monaco/core", () => ({
  isMonacoLoaded: () => true,
  getMonacoEditor: async () => ({
    editor: { getEditors: () => [mocks.editor] },
  }),
}));
vi.mock("monaco-editor", () => ({}));
vi.mock("@/components/monaco/utils", () => ({
  extensionNameOfLanguage: () => "sql",
}));
vi.mock("@/hooks/useSavedQueryAndTab", () => ({
  useSavedQueryAndTab: () => ({ isReadOnly: false }),
}));
vi.mock("../theme/useActiveSQLEditorTheme", () => ({
  useActiveSQLEditorTheme: () => "light",
}));
vi.mock("../Panels/common/useAIActions", () => ({ useAIActions: () => {} }));
vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useConnectionOfCurrentSQLEditorTab: () => ({
    instance: { name: "instances/i" },
    database: { name: "instances/i/databases/db" },
  }),
}));
vi.mock("@/modules/sql-editor/store", () => {
  const state = {
    ...mocks,
    abortAutoSave: vi.fn(),
    setAutoSaveController: vi.fn(),
  };
  return {
    useSQLEditorStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});
vi.mock("@/utils", async (original) => ({
  ...(await original<typeof import("@/utils")>()),
  canCreateSavedQueryInProject: () => true,
  isSavedQueryWritableV1: () => true,
}));

function Editor() {
  useSQLEditorAutoSave();
  return (
    <>
      <input aria-label="Focused field" />
      <SQLEditor onExecute={vi.fn()} />
    </>
  );
}
afterEach(() => {
  cleanup();
  useAppStore.setState(useAppStore.getInitialState(), true);
  useAgentStore.getState().setWorkspaceName(undefined, "");
  useSQLEditorEditorStore.setState(useSQLEditorEditorStore.getInitialState(), true);
  useSQLEditorTabsStore.setState(useSQLEditorTabsStore.getInitialState());
  vi.useRealTimers();
  vi.clearAllMocks();
});

const workspace = "workspaces/demo";
const user = create(UserSchema, {
  name: "users/member@example.com",
  email: "member@example.com",
  workspace,
});

function bindOwner() {
  useAppStore.setState({
    currentUser: user,
    currentUserName: user.name,
    workspace: create(WorkspaceSchema, { name: workspace }),
  });
  useAgentStore.getState().setWorkspaceName(workspace, user.email);
}

function getNativeTarget() {
  const context = getRegisteredAgentSqlEditorHost()?.captureContext();
  return context?.kind === "ready" ? context.snapshot.editorTarget : undefined;
}

test("a mounted Saved Query native insertion dirties the exact tab and runs its usual autosave without moving focus", async () => {
  vi.useFakeTimers();
  bindOwner();
  useSQLEditorEditorStore.getState().setProject("");
  const tab = getSQLEditorTabsState().addTab({
    statement: "SELECT 1;",
    status: "CLEAN",
  });
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>
      <Editor />
    </SQLEditorAgentHost>
  );
  const buttonField = view.getByRole("textbox", { name: "Focused field" });
  buttonField.focus();
  const host = getRegisteredAgentSqlEditorHost();
  const target = getNativeTarget();
  expect(target).toMatchObject({
    actor: user.name,
    workspace,
    hostId: host?.id,
    tabId: tab.id,
    tabMode: "SAVED_QUERY",
    modelUri: "inmemory://sql/test",
    selection: { start: 9, end: 9, caret: 9 },
  });
  expect(target?.connection).toBeUndefined();
  expect(target?.project).toBeUndefined();
  expect(buttonField).toHaveFocus();

  let result: AgentSqlEditResult | undefined;
  act(() => {
    result = host?.applySql?.({ target: target!, sql: "SELECT 2;", mode: "insert" });
  });
  expect(result).toEqual({ status: "applied" });
  expect(buttonField).toHaveFocus();
  expect(getSQLEditorTabsState().tabsById.get(tab.id)).toMatchObject({
    statement: "SELECT 1;\nSELECT 2;\n",
    status: "DIRTY",
  });
  await act(() => vi.advanceTimersByTimeAsync(2000));
  expect(mocks.createSavedQuery).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({ tabId: tab.id, statement: "SELECT 1;\nSELECT 2;\n" })
  );
});

test("publishes connected native proof only when the Project and connection are ready and keeps prior mounts stale", () => {
  bindOwner();
  const project = create(ProjectSchema, { name: "projects/demo", state: State.ACTIVE });
  const instance = create(InstanceResourceSchema, {
    name: "instances/i",
    engine: Engine.POSTGRES,
    activation: true,
    dataSources: [{ id: "replica" }],
  });
  const database = create(DatabaseSchema$, {
    name: "instances/i/databases/db",
    project: project.name,
    state: State.ACTIVE,
    instanceResource: instance,
  });
  useAppStore.setState({
    projectsByName: { [project.name]: project },
    databasesByName: { [database.name]: database },
    hasProjectPermission: () => true,
  });
  useSQLEditorEditorStore.setState({ project: project.name, projectContextReady: false });
  const connection = { database: database.name, instance: instance.name, schema: "public", table: "items", dataSourceId: "replica" };
  const first = getSQLEditorTabsState().addTab({ statement: "SELECT 1;", connection });
  const second = getSQLEditorTabsState().addTab({ statement: "SELECT 2;", connection });
  getSQLEditorTabsState().setCurrentTabId(first.id);
  render(
    <SQLEditorAgentHost paneRatio={0.35}>
      <SQLEditor onExecute={vi.fn()} />
    </SQLEditorAgentHost>
  );
  const host = getRegisteredAgentSqlEditorHost();
  expect(host?.captureContext()).toEqual({ kind: "pending" });
  act(() => useSQLEditorEditorStore.getState().setProjectContextReady(true));
  const target = getNativeTarget()!;
  expect(target).toMatchObject({
    project: project.name,
    tabId: first.id,
    connection: {
      database: database.name,
      project: project.name,
      engine: Engine.POSTGRES,
      querySchema: "public",
      container: "items",
      dataSourceId: "replica",
    },
  });
  expect(host?.inspect?.(target)).toEqual({ status: "ready" });
  act(() => getSQLEditorTabsState().setCurrentTabId(second.id));
  const next = getNativeTarget()!;
  expect(next.tabId).toBe(second.id);
  expect(next.hostId).toBe(target.hostId);
  expect(next.mountId).not.toBe(target.mountId);
  expect(host?.inspect?.(target)).toEqual({ status: "stale" });
  expect(host?.inspect?.(next)).toEqual({ status: "ready" });

  act(() => useAppStore.setState({ databasesByName: {} }));
  expect(getNativeTarget()).toBeUndefined();
  expect(host?.inspect?.(next)).toEqual({ status: "unavailable" });
  expect(host?.applySql?.({ target: next, sql: "SELECT 3;", mode: "insert" })).toEqual({ status: "unavailable" });
  expect(getSQLEditorTabsState().tabsById.get(second.id)?.statement).toBe("SELECT 2;");
});

test("refuses native editing when a hydrated Saved Query's empty connection masks a lost database association", async () => {
  vi.useFakeTimers();
  bindOwner();
  const project = create(ProjectSchema, {
    name: "projects/demo",
    state: State.ACTIVE,
  });
  const statement = "SELECT 1;";
  const savedQuery = create(SavedQuerySchema, {
    name: "projects/demo/savedQueries/example",
    creator: user.name,
    project: project.name,
    content: new TextEncoder().encode(statement),
    contentSize: 9n,
  });
  const unresolvable = {
    ...savedQuery,
    database: "instances/i/databases/unavailable",
  };
  useAppStore.setState({
    projectsByName: { [project.name]: project },
    savedQueriesByKey: { "example:FULL": savedQuery },
    getOrFetchDatabaseByName: vi.fn().mockRejectedValue(new Error("unavailable")),
  });
  const connection = await extractSavedQueryConnection(unresolvable);
  expect(connection).toEqual({ database: "", instance: "" });
  useSQLEditorEditorStore.setState({
    project: project.name,
    projectContextReady: true,
  });
  const tab = getSQLEditorTabsState().addTab({
    savedQuery: savedQuery.name,
    connection,
    statement,
    status: "CLEAN",
  });
  render(
    <SQLEditorAgentHost paneRatio={0.35}>
      <Editor />
    </SQLEditorAgentHost>
  );
  const host = getRegisteredAgentSqlEditorHost();
  const target = getNativeTarget()!;
  expect(target.savedQuery).toBe(savedQuery.name);
  expect(host?.inspect?.(target)).toEqual({ status: "ready" });

  act(() => {
    useAppStore.setState({ savedQueriesByKey: { "example:FULL": unresolvable } });
  });
  expect(host?.inspect?.(target)).toEqual({ status: "unavailable" });
  expect(getNativeTarget()).toBeUndefined();
  let result: AgentSqlEditResult | undefined;
  act(() => {
    result = host?.applySql?.({ target, sql: "SELECT 2;", mode: "insert" });
  });
  expect(result).toEqual({ status: "unavailable" });
  await act(() => vi.advanceTimersByTimeAsync(2200));
  expect(getSQLEditorTabsState().tabsById.get(tab.id)).toMatchObject({
    statement,
    status: "CLEAN",
  });
  expect(mocks.maybeUpdateSavedQuery).not.toHaveBeenCalled();
});

test("publishes loss of proof until a store-only statement change reaches Monaco", () => {
  bindOwner();
  useSQLEditorEditorStore.getState().setProject("");
  const tab = getSQLEditorTabsState().addTab({ statement: "SELECT 1;" });
  render(
    <SQLEditorAgentHost paneRatio={0.35}>
      <SQLEditor onExecute={vi.fn()} />
    </SQLEditorAgentHost>
  );
  const host = getRegisteredAgentSqlEditorHost();
  const target = getNativeTarget()!;
  expect(host?.inspect?.(target)).toEqual({ status: "ready" });
  act(() => {
    getSQLEditorTabsState().updateTab(tab.id, { statement: "SELECT 3;" });
    expect(mocks.editor?.getValue()).toBe("SELECT 1;");
    expect(host?.inspect?.(target)).toEqual({ status: "stale" });
    expect(getNativeTarget()).toBeUndefined();
    expect(getRegisteredAgentSqlEditorHost()).not.toBe(host);
  });
  expect(mocks.editor?.getValue()).toBe("SELECT 3;");
  const next = getNativeTarget()!;
  expect(next.mountId).toBe(target.mountId);
  expect(next.modelVersion).toBeGreaterThan(target.modelVersion);
  expect(host?.inspect?.(next)).toEqual({ status: "ready" });
});

test("unregistered SQL Editor input cannot dirty the tab or start Saved Query autosave; member typing still can", async () => {
  vi.useFakeTimers();
  const tab = getSQLEditorTabsState().addTab({
    statement: "SELECT 1;",
    status: "CLEAN",
  });
  const view = render(<Editor />);
  const active = view.getByRole("textbox", { name: "Focused field" });
  active.focus();
  const ctx = {
    actor: "users/u",
    workspace: "workspaces/demo",
    chatId: "chat",
    runId: "run",
    runMode: "default" as const,
    pageKey: "/sql-editor",
    pageGeneration: 1,
    isCurrent: () => true,
  };
  const refs = await lazyExtractDomRefSuggestions(ctx);
  const monaco = refs.find(
    (r) => getElementByRef(r.ref)?.element === mocks.editor?.getDomNode()
  );
  expect(monaco?.actions).toBe("manual-required");
  const result = await act(() =>
    lazyPrepareDomAction(
      { ref: monaco!.ref, type: "input", value: "SELECT 2;" },
      ctx
    )
  );
  expect(result).toMatchObject({ result: { success: false } });
  await act(() => vi.advanceTimersByTimeAsync(2200));
  expect(getSQLEditorTabsState().tabsById.get(tab.id)).toMatchObject({
    statement: "SELECT 1;",
    status: "CLEAN",
  });
  expect(active).toHaveFocus();
  expect(mocks.createSavedQuery).not.toHaveBeenCalled();
  expect(mocks.maybeUpdateSavedQuery).not.toHaveBeenCalled();

  fireEvent.change(view.getByRole("textbox", { name: "SQL" }), {
    target: { value: "SELECT 3;" },
  });
  expect(getSQLEditorTabsState().tabsById.get(tab.id)).toMatchObject({
    statement: "SELECT 3;",
    status: "DIRTY",
  });
  await act(() => vi.advanceTimersByTimeAsync(2000));
  expect(mocks.createSavedQuery).toHaveBeenCalledExactlyOnceWith(
    expect.objectContaining({ tabId: tab.id, statement: "SELECT 3;" })
  );
});
