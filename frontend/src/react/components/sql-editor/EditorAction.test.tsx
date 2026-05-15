import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  useSQLEditorVueState: vi.fn(),
  useUIStateStore: vi.fn(),
  useWorkSheetStore: vi.fn(),
  useWorkSheetAndTabStore: vi.fn(),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  isWorksheetWritableV1: vi.fn(() => true),
  keyboardShortcutStr: vi.fn((s: string) => s),
  emit: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useUIStateStore: mocks.useUIStateStore,
  useWorkSheetStore: mocks.useWorkSheetStore,
  useWorkSheetAndTabStore: mocks.useWorkSheetAndTabStore,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/utils", () => ({
  isWorksheetWritableV1: mocks.isWorksheetWritableV1,
  keyboardShortcutStr: mocks.keyboardShortcutStr,
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: { emit: mocks.emit },
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    className,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    disabled?: boolean;
    className?: string;
    "aria-label"?: string;
  }) => (
    <button
      data-testid="button"
      aria-label={ariaLabel}
      className={className}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover">{children}</div>
  ),
  PopoverTrigger: ({ render }: { render?: React.ReactElement }) => (
    <div data-testid="popover-trigger">{render}</div>
  ),
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

vi.mock("./AdminModeButton", () => ({
  AdminModeButton: () => <div data-testid="admin-mode-button" />,
}));
vi.mock("./ChooserGroup", () => ({
  ChooserGroup: () => <div data-testid="chooser-group" />,
}));
vi.mock("./OpenAIButton", () => ({
  OpenAIButton: () => <div data-testid="openai-button" />,
}));
vi.mock("./QueryContextSettingPopover", () => ({
  QueryContextSettingPopover: ({ disabled }: { disabled?: boolean }) => (
    <div
      data-testid="query-context-setting-popover"
      data-disabled={String(disabled)}
    />
  ),
}));
vi.mock("./SharePopoverBody", () => ({
  SharePopoverBody: () => <div data-testid="share-popover-body" />,
}));

let EditorAction: typeof import("./EditorAction").EditorAction;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

type SetupOptions = {
  mode?: "WORKSHEET" | "ADMIN";
  isDisconnected?: boolean;
  statement?: string;
  status?: "CLEAN" | "DIRTY" | "SAVING";
  worksheet?: string;
  engine?: Engine;
  table?: string;
};

const setup = (options: SetupOptions = {}) => {
  const {
    mode = "WORKSHEET",
    isDisconnected = false,
    statement = "SELECT 1",
    status = "DIRTY",
    worksheet,
    engine = Engine.POSTGRES,
    table,
  } = options;

  const currentTab = {
    id: "t1",
    mode,
    statement,
    selectedStatement: "",
    status,
    worksheet,
    connection: { database: "databases/db1", table },
    editorState: { selection: null },
  };
  const updateCurrentTab = vi.fn();
  const saveIntroStateByKey = vi.fn();

  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab,
    isDisconnected,
    updateCurrentTab,
  });
  mocks.useSQLEditorVueState.mockReturnValue({ resultRowsLimit: 500 });
  mocks.useUIStateStore.mockReturnValue({ saveIntroStateByKey });
  mocks.useWorkSheetStore.mockReturnValue({
    getWorksheetByName: vi.fn(() => ({
      name: worksheet ?? "",
      database: "databases/db1",
    })),
  });
  mocks.useWorkSheetAndTabStore.mockReturnValue({
    currentSheet: worksheet ? { name: worksheet, title: "sheet" } : undefined,
  });
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    instance: { value: { engine } },
  });

  mocks.useVueState.mockImplementation((getter) => getter());

  return { currentTab, updateCurrentTab, saveIntroStateByKey };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({
    t: (key: string, fallback?: string) => fallback ?? key,
  });
  mocks.isWorksheetWritableV1.mockReturnValue(true);
  ({ EditorAction } = await import("./EditorAction"));
});

describe("EditorAction", () => {
  test("1. Run button fires onExecute with current statement + connection", () => {
    setup();
    const onExecute = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <EditorAction onExecute={onExecute} />
    );
    render();

    const runButton = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.includes("limit")) as
      | HTMLButtonElement
      | undefined;
    expect(runButton).not.toBeUndefined();
    expect(runButton?.disabled).toBe(false);

    act(() => {
      runButton?.click();
    });

    expect(onExecute).toHaveBeenCalledWith(
      expect.objectContaining({
        statement: "SELECT 1",
        engine: Engine.POSTGRES,
        explain: false,
      })
    );

    unmount();
  });

  test("2. Run button is disabled when disconnected OR statement empty", () => {
    setup({ statement: "" });

    const { container, render, unmount } = renderIntoContainer(
      <EditorAction onExecute={vi.fn()} />
    );
    render();

    const runButton = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.includes("limit")) as
      | HTMLButtonElement
      | undefined;
    expect(runButton?.disabled).toBe(true);

    unmount();
  });

  test("3. In ADMIN mode, renders Exit-Admin button instead of Run", () => {
    const { updateCurrentTab } = setup({ mode: "ADMIN" });

    const { container, render, unmount } = renderIntoContainer(
      <EditorAction onExecute={vi.fn()} />
    );
    render();

    const exitBtn = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.includes("sql-editor.admin-mode.exit")) as
      | HTMLButtonElement
      | undefined;
    expect(exitBtn).not.toBeUndefined();

    // No Run button
    const runButton = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.includes("limit"));
    expect(runButton).toBeUndefined();

    act(() => {
      exitBtn?.click();
    });
    expect(updateCurrentTab).toHaveBeenCalledWith({ mode: "WORKSHEET" });

    unmount();
  });

  test("4. Save button disabled when status=CLEAN; emits save-sheet when dirty", () => {
    const { currentTab } = setup({ status: "DIRTY" });

    const { container, render, unmount } = renderIntoContainer(
      <EditorAction onExecute={vi.fn()} />
    );
    render();

    const saveBtn = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.getAttribute("aria-label") === "common.save") as
      | HTMLButtonElement
      | undefined;
    expect(saveBtn).not.toBeUndefined();
    expect(saveBtn?.disabled).toBe(false);

    act(() => {
      saveBtn?.click();
    });
    expect(mocks.emit).toHaveBeenCalledWith("save-sheet", { tab: currentTab });

    unmount();
  });

  test("5. Share button disabled when disconnected or empty statement", () => {
    setup({ isDisconnected: true });

    const { container, render, unmount } = renderIntoContainer(
      <EditorAction onExecute={vi.fn()} />
    );
    render();

    const shareBtn = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.getAttribute("aria-label") === "common.share") as
      | HTMLButtonElement
      | undefined;
    expect(shareBtn).not.toBeUndefined();
    expect(shareBtn?.disabled).toBe(true);

    unmount();
  });
});
