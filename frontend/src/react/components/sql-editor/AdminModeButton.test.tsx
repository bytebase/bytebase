import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorVueState: vi.fn(),
  useSQLEditorTabStore: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
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
  // Default: allow admin, WORKSHEET mode, connected
  mocks.useSQLEditorVueState.mockReturnValue({ allowAdmin: true });
  mocks.useSQLEditorTabStore.mockReturnValue({
    updateCurrentTab: vi.fn(),
  });
  mocks.useVueState.mockImplementation((getter) => getter());
  ({ AdminModeButton } = await import("./AdminModeButton"));
});

describe("AdminModeButton", () => {
  test("renders nothing when allowAdmin is false", () => {
    mocks.useSQLEditorVueState.mockReturnValue({ allowAdmin: false });
    mocks.useVueState.mockImplementation((getter) => getter());
    // Need to also mock the tab state reads. Configure useVueState to return
    // appropriate values per call: allowAdmin=false, mode="WORKSHEET", isDisconnected=false
    let callIdx = 0;
    mocks.useVueState.mockImplementation(() => {
      const values = [false, "WORKSHEET", false];
      return values[callIdx++];
    });
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    expect(container.querySelector("button")).toBeNull();
    unmount();
  });

  test("renders nothing when current tab mode is not WORKSHEET", () => {
    let callIdx = 0;
    mocks.useVueState.mockImplementation(() => {
      const values = [true, "ADMIN", false];
      return values[callIdx++];
    });
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    expect(container.querySelector("button")).toBeNull();
    unmount();
  });

  test("renders disabled button when isDisconnected is true", () => {
    let callIdx = 0;
    mocks.useVueState.mockImplementation(() => {
      const values = [true, "WORKSHEET", true];
      return values[callIdx++];
    });
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
    const updateCurrentTab = vi.fn();
    mocks.useSQLEditorTabStore.mockReturnValue({ updateCurrentTab });
    let callIdx = 0;
    mocks.useVueState.mockImplementation(() => {
      const values = [true, "WORKSHEET", false];
      return values[callIdx++];
    });
    const { container, render, unmount } = renderIntoContainer(
      <AdminModeButton />
    );
    render();
    act(() => {
      container.querySelector("button")?.click();
    });
    expect(updateCurrentTab).toHaveBeenCalledWith({ mode: "ADMIN" });
    unmount();
  });
});
