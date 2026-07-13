import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  routerCurrentQuery: {} as Record<string, unknown>,
  routerPush: vi.fn(),
  instanceFormContext: {
    basicInfo: { engine: 0 },
    state: { isRequesting: false },
    valueChanged: false,
  },
}));

let CreateInstancePage: typeof import("./CreateInstancePage").CreateInstancePage;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/react/router", () => ({
  router: {
    push: mocks.routerPush,
    currentRoute: {
      get value() {
        return { query: mocks.routerCurrentQuery };
      },
    },
  },
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      instanceCountLimit: () => 10,
      activatedInstanceCount: () => 0,
    }),
  },
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/react/components/instance", () => ({
  InfoPanel: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  InfoPanelContent: () => <div />,
  InstanceFormBody: () => <div data-testid="instance-form-body" />,
  InstanceFormButtons: ({ className }: { className?: string }) => (
    <div data-testid="instance-form-buttons" className={className} />
  ),
  InstanceFormProvider: ({
    children,
    onDismiss,
  }: {
    children: React.ReactNode;
    onDismiss?: () => void;
  }) => (
    <div>
      <button type="button" data-testid="dismiss" onClick={onDismiss}>
        dismiss
      </button>
      {children}
    </div>
  ),
  useInstanceFormContext: () => mocks.instanceFormContext,
}));

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.routerCurrentQuery = {};
  mocks.instanceFormContext.state.isRequesting = false;
  mocks.instanceFormContext.valueChanged = false;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as typeof ResizeObserver;
  ({ CreateInstancePage } = await import("./CreateInstancePage"));
});

describe("CreateInstancePage", () => {
  test("keeps the scroll container flush with the page edge", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    const page = container.firstElementChild;
    expect(page).not.toHaveClass("px-4");
    expect(page).not.toHaveClass("sm:px-6");

    const bodyPadding = container.querySelector(
      "[data-testid='instance-form-body']"
    )?.parentElement;
    expect(bodyPadding).toHaveClass("px-4");
    expect(bodyPadding).toHaveClass("sm:px-6");

    const buttons = container.querySelector(
      "[data-testid='instance-form-buttons']"
    );
    expect(buttons).not.toHaveClass("px-4");
    expect(buttons).not.toHaveClass("sm:px-6");

    act(() => {
      root.unmount();
    });
  });

  test("guards navigation when the create form has unsaved changes", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    mocks.instanceFormContext.valueChanged = true;
    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    mocks.instanceFormContext.state.isRequesting = true;
    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    act(() => {
      root.unmount();
    });
  });

  test("dismisses to the workspace instance dashboard by default", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    const dismiss = container.querySelector(
      "[data-testid='dismiss']"
    ) as HTMLButtonElement;
    act(() => {
      dismiss.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
    });

    act(() => {
      root.unmount();
    });
  });

  test("dismisses to the project database page when project context is present", () => {
    mocks.routerCurrentQuery = { project: "demo" };
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    const dismiss = container.querySelector(
      "[data-testid='dismiss']"
    ) as HTMLButtonElement;
    act(() => {
      dismiss.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "demo" },
    });

    act(() => {
      root.unmount();
    });
  });

  test("keeps syncing instance and intro context when dismissing to the project database page", () => {
    mocks.routerCurrentQuery = {
      project: "demo",
      syncInstance: "prod",
      intro: "connect-database",
    };
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    const dismiss = container.querySelector(
      "[data-testid='dismiss']"
    ) as HTMLButtonElement;
    act(() => {
      dismiss.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "demo" },
      query: {
        syncingInstance: "prod",
        intro: "connect-database",
      },
    });

    act(() => {
      root.unmount();
    });
  });

  test("shows the instance form directly for project-context setup", () => {
    mocks.routerCurrentQuery = { project: "demo" };
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(container.textContent).not.toContain(
      "instance.create-readiness.title"
    );
    expect(
      container.querySelector("[data-testid='instance-form-body']")
    ).not.toBe(null);

    act(() => {
      root.unmount();
    });
  });
});
