import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  onDismiss: vi.fn(),
  onCreated: vi.fn(),
  providerProps: undefined as Record<string, unknown> | undefined,
  instanceFormContext: {
    basicInfo: { engine: 0 },
    state: { isRequesting: false },
    valueChanged: false,
  },
}));

let CreateInstanceView: typeof import("./CreateInstanceView").CreateInstanceView;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      instanceCountLimit: () => 10,
      activatedInstanceCount: () => 0,
    }),
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/components/instance", () => ({
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
    ...props
  }: {
    children: React.ReactNode;
    onDismiss?: () => void;
    parent?: string;
    project?: unknown;
  }) => (
    <div
      ref={() => {
        mocks.providerProps = props;
      }}
    >
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
  mocks.providerProps = undefined;
  mocks.instanceFormContext.state.isRequesting = false;
  mocks.instanceFormContext.valueChanged = false;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as typeof ResizeObserver;
  ({ CreateInstanceView } = await import("./CreateInstanceView"));
});

describe("CreateInstanceView", () => {
  test("keeps the scroll container flush with the page edge", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
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
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    mocks.instanceFormContext.valueChanged = true;
    act(() => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    mocks.instanceFormContext.state.isRequesting = true;
    act(() => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    act(() => {
      root.unmount();
    });
  });

  test("uses the provided dismissal callback", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    const dismiss = container.querySelector(
      "[data-testid='dismiss']"
    ) as HTMLButtonElement;
    act(() => {
      dismiss.click();
    });

    expect(mocks.onDismiss).toHaveBeenCalledOnce();

    act(() => {
      root.unmount();
    });
  });

  test("forwards project ownership to the form provider", () => {
    const container = document.createElement("div");
    const root = createRoot(container);
    const project = { name: "projects/demo" };

    act(() => {
      root.render(
        <CreateInstanceView
          parent="projects/demo"
          project={project as never}
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(mocks.providerProps).toEqual({
      parent: "projects/demo",
      project,
    });

    act(() => {
      root.unmount();
    });
  });
});
