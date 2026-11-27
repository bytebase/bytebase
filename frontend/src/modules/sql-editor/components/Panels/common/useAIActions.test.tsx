import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { MonacoModule } from "@/components/monaco/types";
import { useAIActions } from "./useAIActions";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  handoff: vi.fn(() => true),
  getOrFetchSettingByName: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("@/modules/sql-editor/model/agentHandoff", () => ({ handoffSqlEditorAction: mocks.handoff }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "plugin.ai.actions.explain-code": "Explain code",
        "plugin.ai.actions.find-problems": "Find problems",
        "plugin.ai.actions.new-chat-using-selection":
          "New chat using selection",
      })[key] ?? key,
  }),
}));

vi.mock("@/stores/app", () => {
  type AppStoreStub = {
    getOrFetchSettingByName: typeof mocks.getOrFetchSettingByName;
    getSettingByName: () => {
      value: { value: { case: string; value: { enabled: boolean } } };
    };
  };
  const state = {
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getSettingByName: () => ({
      value: { value: { case: "ai", value: { enabled: true } } },
    }),
  } satisfies AppStoreStub;
  return {
    useAppStore: (selector: (state: AppStoreStub) => unknown) => selector(state),
  };
});

const roots: ReturnType<typeof createRoot>[] = [];

afterEach(() => {
  for (const root of roots) {
    act(() => root.unmount());
  }
  roots.length = 0;
  vi.clearAllMocks();
});

describe("useAIActions", () => {
  test("identifies every editor context-menu action as AI", () => {
    const addAction = vi.fn((_action: { id: string; label: string; run: () => void }) => ({
      dispose: vi.fn(),
    }));
    const disposable = () => ({ dispose: vi.fn() });
    let selection = "SELECT 1";
    const editor = {
      addAction,
      createContextKey: vi.fn(() => ({ set: vi.fn() })),
      getModel: () => ({
        getLanguageId: () => "sql",
        getValue: () => "SELECT 1",
        getValueInRange: () => selection,
      }),
      getSelection: () => ({}),
      onDidChangeCursorSelection: disposable,
      onDidChangeModelContent: disposable,
    };
    const monaco = {} as MonacoModule;
    const activeStatement = () => "SELECT 2";
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    function Host() {
      useAIActions({
        monaco,
        editor: editor as never,
        actions: ["explain-code", "find-problems", "new-chat"],
        activeStatement,
      });
      return null;
    }

    act(() => root.render(<Host />));

    expect(addAction.mock.calls.map(([action]) => action)).toMatchObject([
      {
        id: "explain-code",
        label: "[AI] Explain code",
        contextMenuGroupId: "2_ai_assistant",
      },
      {
        id: "find-problems",
        label: "[AI] Find problems",
        contextMenuGroupId: "2_ai_assistant",
      },
      {
        id: "new-chat-using-selection",
        label: "[AI] New chat using selection",
        contextMenuGroupId: "2_ai_assistant",
      },
    ]);
    const [explain, find, draft] = addAction.mock.calls.map(([action]) => action);
    explain.run();
    draft.run();
    selection = "";
    find.run();
    draft.run();
    expect(mocks.handoff.mock.calls).toEqual([
      ["explain-code", "SELECT 1", "Explain code"],
      ["new-chat", "SELECT 1", "New chat using selection"],
      ["find-problems", "SELECT 2", "Find problems"],
    ]);
  });
});
