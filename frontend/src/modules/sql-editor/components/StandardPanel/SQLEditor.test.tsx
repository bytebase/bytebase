import { act, cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  lazyExtractDomRefSuggestions,
  lazyPrepareDomAction,
} from "@/modules/agent/dom";
import { getElementByRef } from "@/modules/agent/dom/snapshotIndex";
import {
  getSQLEditorTabsState,
  useSQLEditorTabsStore,
} from "@/modules/sql-editor/store/tab";
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
vi.mock("@/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({
    content,
    onChange,
  }: {
    content: string;
    onChange: (value: string) => void;
  }) => (
    <div
      className="monaco-editor"
      ref={(node) => {
        mocks.editor = node
          ? {
              getDomNode: () => node,
              getValue: () => content,
              setValue: onChange,
            }
          : undefined;
      }}
    >
      <textarea
        aria-label="SQL"
        value={content}
        onChange={(e) => onChange(e.currentTarget.value)}
      />
    </div>
  ),
}));
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
  useSQLEditorTabsStore.setState(useSQLEditorTabsStore.getInitialState());
  vi.useRealTimers();
  vi.clearAllMocks();
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
