import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Per-test controllable state read by the migrated Zustand/bridge hooks.
  state: {
    project: "projects/test" as string,
    allowAdmin: true,
    currentTabMode: "WORKSHEET" as string | undefined,
    isDisconnected: false,
  },
  updateCurrentTab: vi.fn(),
  useSQLEditorAllowAdmin: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/store", () => ({}));

// Zustand editor store — selector hook returns the active project.
vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.state.project }),
}));

// Zustand tab store — selector hook, derived hooks, and imperative getter.
vi.mock("@/react/stores/sqlEditor/tab", () => ({
  useSQLEditorTabState: (
    selector: (s: {
      currentTabId: string;
      tabsById: Map<string, { mode: string | undefined }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: "tab1",
      tabsById: new Map([["tab1", { mode: mocks.state.currentTabMode }]]),
    }),
  useIsDisconnected: () => mocks.state.isDisconnected,
  getSQLEditorTabsState: () => ({
    updateCurrentTab: mocks.updateCurrentTab,
  }),
}));

// Pinia bridge hook that resolves the admin permission for the project.
vi.mock("@/react/hooks/useSQLEditorBridge", () => ({
  useSQLEditorAllowAdmin: mocks.useSQLEditorAllowAdmin,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let AdminModeButton: typeof import("./AdminModeButton").AdminModeButton;

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

beforeEach(async () => {
  vi.clearAllMocks();
  // Default: allow admin, WORKSHEET mode, connected.
  mocks.state.project = "projects/test";
  mocks.state.allowAdmin = true;
  mocks.state.currentTabMode = "WORKSHEET";
  mocks.state.isDisconnected = false;
  mocks.useSQLEditorAllowAdmin.mockImplementation(() => mocks.state.allowAdmin);
  ({ AdminModeButton } = await import("./AdminModeButton"));
});

describe("AdminModeButton", () => {
  test("renders nothing when allowAdmin is false", () => {
    mocks.state.allowAdmin = false;
    mocks.state.currentTabMode = "WORKSHEET";
    mocks.state.isDisconnected = false;
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    expect(container.querySelector("button")).toBeNull();
    unmount();
  });

  test("renders nothing when current tab mode is not WORKSHEET", () => {
    mocks.state.allowAdmin = true;
    mocks.state.currentTabMode = "ADMIN";
    mocks.state.isDisconnected = false;
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    expect(container.querySelector("button")).toBeNull();
    unmount();
  });

  test("renders disabled button when isDisconnected is true", () => {
    mocks.state.allowAdmin = true;
    mocks.state.currentTabMode = "WORKSHEET";
    mocks.state.isDisconnected = true;
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    const button = container.querySelector("button");
    expect(button).not.toBeNull();
    expect(button?.hasAttribute("disabled")).toBe(true);
    unmount();
  });

  test("click sets currentTab.mode to ADMIN", () => {
    mocks.state.allowAdmin = true;
    mocks.state.currentTabMode = "WORKSHEET";
    mocks.state.isDisconnected = false;
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(mocks.updateCurrentTab).toHaveBeenCalledWith({ mode: "ADMIN" });
    unmount();
  });
});
