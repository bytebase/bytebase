import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Per-test controllable tab-derived state.
  tabState: {
    isDisconnected: false,
    currentMode: "WORKSHEET" as string | undefined,
  },
  // New zustand state mirror.
  state: { showAIPanel: false },
  setShowAIPanel: vi.fn((v: boolean) => {
    mocks.state.showAIPanel = v;
  }),
  useSettingV1Store: vi.fn(),
  // App-store AI setting + fetch — the component reads
  // `useAppStore((s) => s.getSettingByName(AI))` for the enabled state and
  // `useAppStore((s) => s.getOrFetchSettingByName)` in an effect.
  aiSetting: undefined as unknown,
  getOrFetchSettingByName: vi.fn().mockResolvedValue(undefined),
  useConnectionOfCurrentSQLEditorTab: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
  nextAnimationFrame: vi.fn(() => Promise.resolve()),
  emit: vi.fn(),
  explainCode: vi.fn((s: string) => `EXPLAIN:${s}`),
  findProblems: vi.fn((s: string) => `FIND:${s}`),
  routerPush: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/store", () => ({
  useSettingV1Store: mocks.useSettingV1Store,
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getSettingByName: () => mocks.aiSetting,
  };
  return {
    useAppStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

// `useConnectionOfCurrentSQLEditorTab` now lives on the Pinia bridge hook.
vi.mock("@/react/hooks/useSQLEditorBridge", () => ({
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
}));

// Zustand tab store — derived hook + selector hook for connection/mode.
vi.mock("@/react/stores/sqlEditor/tab", () => ({
  useIsDisconnected: () => mocks.tabState.isDisconnected,
  useSQLEditorTabState: (
    selector: (s: {
      currentTabId: string;
      tabsById: Map<string, { mode: string | undefined }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: "tab1",
      tabsById: new Map([["tab1", { mode: mocks.tabState.currentMode }]]),
    }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: {
      showAIPanel: boolean;
      setShowAIPanel: (v: boolean) => void;
    }) => unknown
  ) =>
    selector({
      showAIPanel: mocks.state.showAIPanel,
      setShowAIPanel: mocks.setShowAIPanel,
    }),
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  nextAnimationFrame: mocks.nextAnimationFrame,
}));

vi.mock("@/plugins/ai/logic", () => ({
  aiContextEvents: { emit: mocks.emit },
}));

vi.mock("@/plugins/ai/logic/prompt", () => ({
  explainCode: mocks.explainCode,
  findProblems: mocks.findProblems,
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: String(to), fullPath: String(to) }),
  },
}));

// Minimal primitive stubs.
vi.mock("@/react/components/ui/button", () => ({
  buttonVariants: ({ className }: { className?: string } = {}) => className,
  Button: ({
    children,
    onClick,
    disabled,
    "aria-disabled": ariaDisabled,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    "aria-disabled"?: boolean;
    "aria-label"?: string;
  }) => (
    <button
      data-testid="button"
      aria-label={ariaLabel}
      disabled={disabled}
      aria-disabled={ariaDisabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
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

vi.mock("@/react/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
  }) => (
    <div data-testid="dropdown-menu" data-open={String(open ?? false)}>
      {children}
    </div>
  ),
  DropdownMenuTrigger: ({ render }: { render?: React.ReactElement }) => (
    <div data-testid="dropdown-menu-trigger">{render}</div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-menu-content">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button
      data-testid="dropdown-menu-item"
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

let OpenAIButton: typeof import("./OpenAIButton").OpenAIButton;

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

// Default state: connected, worksheet mode, AI enabled, showAIPanel=false
type VueStateValues = {
  isDisconnected: boolean;
  currentMode: string | undefined;
  showAIPanel: boolean;
  instance: { engine: Engine };
  openAIEnabled: boolean;
};

const setupDefaultMocks = (overrides: Partial<VueStateValues> = {}) => {
  const values: VueStateValues = {
    isDisconnected: false,
    currentMode: "WORKSHEET",
    showAIPanel: false,
    instance: { engine: Engine.POSTGRES },
    openAIEnabled: true,
    ...overrides,
  };

  mocks.state.showAIPanel = values.showAIPanel;
  mocks.tabState.isDisconnected = values.isDisconnected;
  mocks.tabState.currentMode = values.currentMode;

  // `openAIEnabled` is derived from the app-store AI setting in the
  // component (`s.getSettingByName(AI)` → `.value.value.case === "ai"
  // ? .enabled : false`). Build a setting whose shape matches the
  // configured value.
  mocks.aiSetting = values.openAIEnabled
    ? { value: { value: { case: "ai", value: { enabled: true } } } }
    : undefined;

  // Migrated hook returns PLAIN values — no Vue `.value` wrapper.
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    instance: values.instance,
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  ({ OpenAIButton } = await import("./OpenAIButton"));
});

describe("OpenAIButton", () => {
  test("renders nothing when disconnected", () => {
    setupDefaultMocks({ isDisconnected: true });
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton />
    );
    render();

    expect(container.querySelector("[data-testid='button']")).toBeNull();

    unmount();
  });

  test("renders nothing when not in WORKSHEET mode", () => {
    setupDefaultMocks({ currentMode: "ADMIN" });
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton />
    );
    render();

    expect(container.querySelector("[data-testid='button']")).toBeNull();

    unmount();
  });

  test("shows disabled button + configure popover when AI not configured", () => {
    setupDefaultMocks({ openAIEnabled: false });
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton />
    );
    render();

    const button = container.querySelector(
      "[data-testid='button']"
    ) as HTMLButtonElement | null;
    expect(button).not.toBeNull();
    expect(button?.getAttribute("aria-disabled")).toBeTruthy();

    // Popover body includes the not-configured key
    expect(container.textContent).toContain("plugin.ai.not-configured.self");

    unmount();
  });

  test("click toggles showAIPanel when enabled", () => {
    setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton />
    );
    render();

    const button = container.querySelector(
      "[data-testid='button']"
    ) as HTMLButtonElement | null;
    expect(button).not.toBeNull();

    act(() => {
      button?.click();
    });

    expect(mocks.setShowAIPanel).toHaveBeenCalledWith(true);

    unmount();
  });

  test("selecting explain-code action emits send-chat with prompt", async () => {
    setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton statement="SELECT 1" />
    );
    render();

    const items = container.querySelectorAll(
      "[data-testid='dropdown-menu-item']"
    );
    // First item = explain-code
    const explainItem = items[0] as HTMLButtonElement | undefined;
    expect(explainItem?.textContent).toBe("plugin.ai.actions.explain-code");

    await act(async () => {
      explainItem?.click();
    });

    expect(mocks.setShowAIPanel).toHaveBeenCalledWith(true);
    expect(mocks.explainCode).toHaveBeenCalledWith("SELECT 1", Engine.POSTGRES);
    expect(mocks.emit).toHaveBeenCalledWith("send-chat", {
      content: "EXPLAIN:SELECT 1",
      newChat: true,
    });

    unmount();
  });
});
