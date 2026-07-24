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
  const connectionListeners = new Set<() => void>();
  const model = {
    uri: {
      toString: () => "file:///test.sql",
    },
    getLanguageId: () => "sql",
    getValue: () => "select 1",
    getValueInRange: () => "",
    getAllDecorations: () => [],
  };
  const subscription = { dispose: vi.fn() };
  const handlers = {
    modelContent: undefined as (() => void) | undefined,
  };
  const editor = {
    addAction: vi.fn(() => subscription),
    createDecorationsCollection: vi.fn(() => ({ clear: vi.fn() })),
    deltaDecorations: vi.fn(() => []),
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
    onDidChangeModelContent: vi.fn((handler: () => void) => {
      handlers.modelContent = handler;
      return subscription;
    }),
    onDidContentSizeChange: vi.fn(() => subscription),
    setModel: vi.fn(),
    setValue: vi.fn(),
    updateOptions: vi.fn(),
  };
  return {
    connectionSnapshot,
    connectionListeners,
    currentWebSocket: undefined as
      | {
          addEventListener: ReturnType<typeof vi.fn>;
          removeEventListener: ReturnType<typeof vi.fn>;
        }
      | undefined,
    editor,
    handlers,
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

vi.mock("@/components/ui/tooltip", () => ({
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
  ensureLSPConnection: vi.fn(async () => ({})),
  executeCommand: vi.fn(async () => undefined),
  getConnectionStateSnapshot: vi.fn(() => mocks.connectionSnapshot),
  getConnectionWebSocket: vi.fn(() =>
    mocks.currentWebSocket
      ? Promise.resolve(mocks.currentWebSocket as unknown as WebSocket)
      : undefined
  ),
  initializeLSPClient: vi.fn(async () => ({})),
  subscribeConnectionState: vi.fn((listener: () => void) => {
    mocks.connectionListeners.add(listener);
    return () => {
      mocks.connectionListeners.delete(listener);
    };
  }),
}));

vi.mock("./suggest-icons", () => ({
  ensureSuggestOverrideStyle: vi.fn(() => ({ remove: vi.fn() })),
}));

vi.mock("./text-model", () => ({
  getUriByFilename: vi.fn(async (filename: string) => ({
    toString: () => `file:///${filename}`,
  })),
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
  mocks.connectionSnapshot = {
    heartbeat: {
      counter: 1,
      timer: undefined,
      timestamp: 1_714_000_000_000,
    },
    state: "ready",
  };
  mocks.connectionListeners.clear();
  mocks.currentWebSocket = undefined;
  mocks.handlers.modelContent = undefined;
});

const createWebSocket = () => ({
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
});

const setConnectionState = (state: typeof mocks.connectionSnapshot.state) => {
  mocks.connectionSnapshot = {
    ...mocks.connectionSnapshot,
    state,
  };
};

const notifyConnectionState = async () => {
  act(() => {
    mocks.connectionListeners.forEach((listener) => listener());
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));
};

describe("MonacoEditor", () => {
  test("keeps the legacy heartbeat interpolation text", () => {
    const locale = JSON.parse(
      readFileSync(resolve("src/locales/en-US.json"), "utf-8")
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

  test("rebinds statement-range listener when the LSP socket recovers", async () => {
    const firstWebSocket = createWebSocket();
    const recoveredWebSocket = createWebSocket();
    mocks.currentWebSocket = firstWebSocket;

    const { unmount } = await renderIntoContainer(
      createElement(MonacoEditor, {
        content: "select 1",
        enableDecorations: true,
      })
    );

    expect(firstWebSocket.addEventListener).toHaveBeenCalledWith(
      "message",
      expect.any(Function)
    );

    setConnectionState("closed");
    await notifyConnectionState();
    expect(firstWebSocket.removeEventListener).toHaveBeenCalledWith(
      "message",
      expect.any(Function)
    );

    mocks.currentWebSocket = recoveredWebSocket;
    setConnectionState("ready");
    await notifyConnectionState();

    expect(recoveredWebSocket.addEventListener).toHaveBeenCalledWith(
      "message",
      expect.any(Function)
    );

    unmount();
  });

  test("resends LSP metadata when the connection recovers", async () => {
    const { executeCommand } = await import("./lsp-client");
    setConnectionState("closed");

    const { unmount } = await renderIntoContainer(
      createElement(MonacoEditor, {
        autoCompleteContext: {
          database: "instances/prod/databases/db",
          instance: "instances/prod",
          scene: "query",
          schema: "public",
        },
        content: "select 1",
      })
    );

    await new Promise((resolve) => setTimeout(resolve, 600));
    expect(executeCommand).not.toHaveBeenCalled();

    setConnectionState("ready");
    await notifyConnectionState();
    await new Promise((resolve) => setTimeout(resolve, 600));

    expect(executeCommand).toHaveBeenCalledWith(
      expect.anything(),
      "setMetadata",
      [
        expect.objectContaining({
          databaseName: "db",
          documentUri: expect.stringContaining("file:///"),
          instanceId: "instances/prod",
          scene: "query",
          schema: "public",
        }),
      ]
    );

    unmount();
  });
});
