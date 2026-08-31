import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Per-test controllable tab-derived state.
  tabState: {
    isDisconnected: false,
    currentMode: "SAVED_QUERY" as string | undefined,
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
  hasWorkspacePermissionV2: vi.fn(() => true),
  routerPush: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/stores", () => ({
  useSettingV1Store: mocks.useSettingV1Store,
}));

vi.mock("@/stores/app", () => {
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

// Zustand tab store — derived hook + selector hook for connection/mode.
vi.mock("@/modules/sql-editor/store/tab", () => ({
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

vi.mock("@/modules/sql-editor/store", () => ({
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
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: String(to), fullPath: String(to) }),
  },
}));

// Minimal primitive stubs.
vi.mock("@/components/ui/button", () => ({
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

vi.mock("@/components/ui/popover", () => ({
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

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactNode;
    content: React.ReactNode;
  }) => (
    <div data-testid="tooltip" data-content={String(content)}>
      {children}
    </div>
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

// Default state: connected, saved query mode, AI enabled, showAIPanel=false
type VueStateValues = {
  isDisconnected: boolean;
  currentMode: string | undefined;
  showAIPanel: boolean;
  openAIEnabled: boolean;
};

const setupDefaultMocks = (overrides: Partial<VueStateValues> = {}) => {
  const values: VueStateValues = {
    isDisconnected: false,
    currentMode: "SAVED_QUERY",
    showAIPanel: false,
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

  test("renders nothing when not in SAVED_QUERY mode", () => {
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

  test("routes AI configure link to general settings with intro", () => {
    setupDefaultMocks({ openAIEnabled: false });
    const { container, render, unmount } = renderIntoContainer(
      <OpenAIButton />
    );
    render();

    const configureLink = Array.from(container.querySelectorAll("a")).find(
      (link) =>
        link.textContent === "plugin.ai.not-configured.go-to-configure"
    ) as HTMLAnchorElement;

    expect(configureLink).toBeTruthy();

    act(() => {
      configureLink.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "setting.workspace.general",
      hash: "#ai-assistant",
      query: { intro: "ai-assistant" },
    });

    unmount();
  });

  test("uses the icon only to toggle the panel without an action menu", () => {
    setupDefaultMocks();
    const LegacyOpenAIButton = OpenAIButton as React.ComponentType<{
      statement?: string;
    }>;
    const { container, render, unmount } = renderIntoContainer(
      <LegacyOpenAIButton statement="SELECT 1" />
    );
    render();

    expect(container.querySelector("[data-testid='dropdown-menu']")).toBeNull();
    expect(
      container
        .querySelector("[data-testid='tooltip']")
        ?.getAttribute("data-content")
    ).toBe("plugin.ai.ai-assistant");

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
});
