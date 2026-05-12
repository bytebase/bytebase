import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { MonacoEditor } from "./MonacoEditor";

const mocks = vi.hoisted(() => {
  const connectionSnapshot = {
    state: "ready",
    heartbeat: {
      counter: 1,
      timer: undefined,
      timestamp: 1_714_000_000_000,
    },
  };
  const model = {
    uri: {
      toString: () => "file:///test.sql",
    },
    getLanguageId: () => "sql",
    getValue: () => "select 1",
    getValueInRange: () => "",
  };
  const subscription = { dispose: vi.fn() };
  const editor = {
    addAction: vi.fn(() => subscription),
    createDecorationsCollection: vi.fn(() => ({ clear: vi.fn() })),
    dispose: vi.fn(),
    focus: vi.fn(),
    getContentHeight: vi.fn(() => 120),
    getModel: vi.fn(() => model),
    getPosition: vi.fn(() => null),
    getSelection: vi.fn(() => null),
    getValue: vi.fn(() => "select 1"),
    onDidChangeCursorPosition: vi.fn(() => subscription),
    onDidChangeCursorSelection: vi.fn(() => subscription),
    onDidChangeModel: vi.fn(() => subscription),
    onDidChangeModelContent: vi.fn(() => subscription),
    onDidContentSizeChange: vi.fn(() => subscription),
    setModel: vi.fn(),
    setValue: vi.fn(),
    updateOptions: vi.fn(),
  };
  return {
    connectionSnapshot,
    editor,
    model,
    subscription,
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const translations: Record<string, string> = {
        "sql-editor.web-socket.connection-status.title": "Language server",
        "sql-editor.web-socket.connection-status.connected": "connected",
        "sql-editor.web-socket.connection-status.last-heartbeat":
          "Last heartbeat at {{time}}",
      };
      return (translations[key] ?? key).replace("{{time}}", params?.time ?? "");
    },
  }),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactNode;
    content: React.ReactNode;
  }) => (
    <div data-testid="tooltip">
      {children}
      <div data-testid="tooltip-content">{content}</div>
    </div>
  ),
}));

vi.mock("@/types", () => ({
  UNKNOWN_ID: "UNKNOWN_ID",
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: () => ({ databaseName: "db" }),
  extractInstanceResourceName: () => "instance",
  formatAbsoluteDateTime: () => "2024-04-25 12:26:40",
  isDev: () => true,
}));

vi.mock("@/utils/v1/position", () => ({
  batchConvertPositionToMonacoPosition: vi.fn(() => []),
}));

vi.mock("./core", () => ({
  createMonacoEditor: vi.fn(async () => mocks.editor),
  getResolvedTheme: vi.fn(() => "vs"),
  loadMonacoEditor: vi.fn(async () => ({
    editor: {
      EditorOption: {
        readOnly: 1,
      },
      setTheme: vi.fn(),
      TrackedRangeStickiness: {
        AlwaysGrowsWhenTypingAtEdges: 1,
      },
    },
    KeyCode: {
      KeyF: 1,
    },
    KeyMod: {
      Alt: 1,
      Shift: 2,
    },
    Range: vi.fn(),
  })),
  setMonacoModelLanguage: vi.fn(async () => undefined),
}));

vi.mock("./lsp-client", () => ({
  executeCommand: vi.fn(async () => undefined),
  getConnectionStateSnapshot: vi.fn(() => mocks.connectionSnapshot),
  getConnectionWebSocket: vi.fn(() => undefined),
  initializeLSPClient: vi.fn(async () => ({})),
  subscribeConnectionState: vi.fn(() => () => undefined),
}));

vi.mock("./suggest-icons", () => ({
  ensureSuggestOverrideStyle: vi.fn(() => ({ remove: vi.fn() })),
}));

vi.mock("./text-model", () => ({
  getOrCreateTextModel: vi.fn(async () => mocks.model),
  restoreViewState: vi.fn(),
  storeViewState: vi.fn(),
}));

vi.mock("./utils", () => ({
  buildAdviceHoverMessage: vi.fn(() => ""),
  configureMonacoMessages: vi.fn(),
  extensionNameOfLanguage: vi.fn(() => "sql"),
  formatEditorContent: vi.fn(async () => undefined),
  trySetContentWithUndo: vi.fn(),
}));

const renderIntoContainer = async (
  element: ReturnType<typeof createElement>
) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  act(() => {
    root.render(element);
  });
  await new Promise((resolve) => setTimeout(resolve, 0));

  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("MonacoEditor", () => {
  test("keeps the legacy heartbeat interpolation text", () => {
    const locale = JSON.parse(
      readFileSync(resolve("src/react/locales/en-US.json"), "utf-8")
    ) as {
      "sql-editor": {
        "web-socket": {
          "connection-status": {
            "last-heartbeat": string;
          };
        };
      };
    };

    expect(
      locale["sql-editor"]["web-socket"]["connection-status"]["last-heartbeat"]
    ).toBe("Last heartbeat at {{time}}");
  });

  test("shows the LSP connection status tooltip content", async () => {
    const { container, unmount } = await renderIntoContainer(
      createElement(MonacoEditor, {
        content: "select 1",
        enableDecorations: true,
      })
    );

    expect(
      container.querySelector("[data-testid='tooltip-content']")
    ).toHaveTextContent("Language server");
    expect(
      container.querySelector("[data-testid='tooltip-content']")
    ).toHaveTextContent("connected");
    expect(
      container.querySelector("[data-testid='tooltip-content']")
    ).toHaveTextContent("Last heartbeat at 2024-04-25 12:26:40");

    unmount();
  });

  test("hides the LSP connection status tooltip when LSP is disabled", async () => {
    const { container, unmount } = await renderIntoContainer(
      createElement(MonacoEditor, {
        content: "select 1",
      })
    );

    expect(container.querySelector("[data-testid='tooltip']")).toBeNull();

    unmount();
  });
});
