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
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  useSQLEditorUIStore: vi.fn(),
  useSettingV1Store: vi.fn(),
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

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
  useSQLEditorUIStore: mocks.useSQLEditorUIStore,
  useSettingV1Store: mocks.useSettingV1Store,
  useConnectionOfCurrentSQLEditorTab: mocks.useConnectionOfCurrentSQLEditorTab,
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

vi.mock("@/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/router/dashboard/workspaceSetting", () => ({
  SETTING_ROUTE_WORKSPACE_GENERAL: "settings.workspace.general",
}));

// Minimal primitive stubs.
vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    "aria-label"?: string;
  }) => (
    <button
      data-testid="button"
      aria-label={ariaLabel}
      disabled={disabled}
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

  const uiStore = { showAIPanel: values.showAIPanel };
  const tabStore = {
    isDisconnected: values.isDisconnected,
    currentTab: { mode: values.currentMode },
  };
  const settingStore = {
    getOrFetchSettingByName: vi.fn().mockResolvedValue(undefined),
    getSettingByName: vi.fn(),
  };

  mocks.useSQLEditorTabStore.mockReturnValue(tabStore);
  mocks.useSQLEditorUIStore.mockReturnValue(uiStore);
  mocks.useSettingV1Store.mockReturnValue(settingStore);
  mocks.useConnectionOfCurrentSQLEditorTab.mockReturnValue({
    instance: { value: values.instance },
  });

  // Order of useVueState calls: isDisconnected, currentMode, showAIPanel, instance, openAIEnabled
  const ordered = [
    values.isDisconnected,
    values.currentMode,
    values.showAIPanel,
    values.instance,
    values.openAIEnabled,
  ];
  let idx = 0;
  mocks.useVueState.mockImplementation(() => ordered[idx++]);

  return { uiStore, tabStore, settingStore };
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
    expect(button?.disabled).toBe(true);

    // Popover body includes the not-configured key
    expect(container.textContent).toContain("plugin.ai.not-configured.self");

    unmount();
  });

  test("click toggles showAIPanel when enabled", () => {
    const { uiStore } = setupDefaultMocks();
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

    expect(uiStore.showAIPanel).toBe(true);

    unmount();
  });

  test("selecting explain-code action emits send-chat with prompt", async () => {
    const { uiStore } = setupDefaultMocks();
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

    expect(uiStore.showAIPanel).toBe(true);
    expect(mocks.explainCode).toHaveBeenCalledWith("SELECT 1", Engine.POSTGRES);
    expect(mocks.emit).toHaveBeenCalledWith("send-chat", {
      content: "EXPLAIN:SELECT 1",
      newChat: true,
    });

    unmount();
  });
});
