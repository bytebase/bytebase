import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const uiStore = { showConnectionPanel: false };
  return {
    uiStore,
    hasWorkspacePermissionV2: vi.fn(),
    routerPush: vi.fn(),
    useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  INSTANCE_ROUTE_DASHBOARD: "workspace.instance",
}));

vi.mock("@/store", () => ({
  useSQLEditorUIStore: () => mocks.uiStore,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({
    open,
    children,
    onOpenChange,
  }: {
    open: boolean;
    children: React.ReactNode;
    onOpenChange: (v: boolean) => void;
  }) =>
    open ? (
      <div data-testid="sheet" data-open="true">
        {children}
        <button
          type="button"
          data-testid="sheet-close"
          onClick={() => onOpenChange(false)}
        />
      </div>
    ) : null,
  SheetContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-header">{children}</div>
  ),
  SheetTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="sheet-title">{children}</h2>
  ),
  SheetBody: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    "aria-label"?: string;
  }) => (
    <button
      type="button"
      data-testid="settings-button"
      aria-label={ariaLabel}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("./ConnectionPane/ConnectionPane", () => ({
  ConnectionPane: ({ show }: { show: boolean }) => (
    <div data-testid="connection-pane" data-show={String(show)} />
  ),
}));

vi.mock("@/react/components/ui/feature-modal", () => ({
  FeatureModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="feature-modal" /> : null,
}));

let ConnectionPanel: typeof import("./ConnectionPanel").ConnectionPanel;

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
  mocks.uiStore.showConnectionPanel = false;
  mocks.useVueState.mockImplementation((getter) => getter());
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  ({ ConnectionPanel } = await import("./ConnectionPanel"));
});

describe("ConnectionPanel", () => {
  test("renders nothing when uiStore.showConnectionPanel is false", () => {
    mocks.uiStore.showConnectionPanel = false;
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPanel />
    );
    render();
    expect(container.querySelector("[data-testid='sheet']")).toBeNull();
    unmount();
  });

  test("renders header + ConnectionPane when uiStore.showConnectionPanel is true", () => {
    mocks.uiStore.showConnectionPanel = true;
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPanel />
    );
    render();
    expect(
      container.querySelector("[data-testid='sheet-title']")?.textContent
    ).toBe("database.select");
    expect(
      container
        .querySelector("[data-testid='connection-pane']")
        ?.getAttribute("data-show")
    ).toBe("true");
    unmount();
  });

  test("settings button navigates without closing the drawer (matches Vue)", () => {
    mocks.uiStore.showConnectionPanel = true;
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPanel />
    );
    render();
    const btn = container.querySelector(
      "[data-testid='settings-button']"
    ) as HTMLButtonElement;
    expect(btn).not.toBeNull();
    act(() => {
      btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    // Drawer state is left untouched — the route change unmounts the SQL
    // editor anyway, and pre-closing introduced an unnecessary transition.
    expect(mocks.uiStore.showConnectionPanel).toBe(true);
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
    });
    unmount();
  });

  test("hides settings button when user lacks bb.instances.list permission", () => {
    mocks.uiStore.showConnectionPanel = true;
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPanel />
    );
    render();
    expect(
      container.querySelector("[data-testid='settings-button']")
    ).toBeNull();
    unmount();
  });

  test("sheet-close writes uiStore.showConnectionPanel = false", () => {
    mocks.uiStore.showConnectionPanel = true;
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionPanel />
    );
    render();
    const closer = container.querySelector(
      "[data-testid='sheet-close']"
    ) as HTMLButtonElement;
    act(() => {
      closer.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.uiStore.showConnectionPanel).toBe(false);
    unmount();
  });
});
