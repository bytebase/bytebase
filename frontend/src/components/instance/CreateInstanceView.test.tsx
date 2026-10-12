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
  routerPush: vi.fn(),
  runSampleInstance: vi.fn(),
  pushNotification: vi.fn(),
  isSaaSMode: false,
  sampleAvailable: true,
  sampleInstances: [] as { instance: string }[],
  totalInstanceCount: 0,
  instanceCountLimit: 10,
  activatedInstanceCount: 0,
  providerProps: undefined as Record<string, unknown> | undefined,
  instanceFormContext: {
    basicInfo: { engine: 0 },
    state: { isRequesting: false },
    valueChanged: false,
    sampleInstance: undefined as
      | { kind: "prepare" | "open"; pending: boolean; run: () => void }
      | undefined,
  },
}));

let CreateInstanceView: typeof import("./CreateInstanceView").CreateInstanceView;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, values?: { total?: number }) => {
      const translation =
        ({
        "instance.use-sample-instance":
          "Use sample instance (Available for 7 days)",
        "instance.use-sample-instance-self-host": "Use sample instance",
        "instance.preparing-sample-instance":
          "Preparing Sample Project Instance…",
        "instance.prepare-sample-instance-failed":
          "Failed to prepare Sample Project Instance.",
        "instance.sample-project-instance-description":
          "Use a Sample Project Instance to explore Bytebase with a ready-to-use database for 7 days.",
        "instance.sample-project-instance-description-self-host":
          "Use a Sample Project Instance to explore Bytebase with a ready-to-use database.",
        "instance.sample-project-instance-title":
          "Try a Sample Project Instance",
        "subscription.usage.instance-count.title": "Instance quota reached",
        "subscription.usage.instance-count.runoutof":
          "You have reached the limit of {{total}} instances.",
        })[key] ?? key;
      return translation.replace("{{total}}", String(values?.total ?? ""));
    },
  }),
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/app/router", () => ({
  router: { push: mocks.routerPush },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    <T,>(selector: (state: Record<string, unknown>) => T) =>
      selector({
        isSaaSMode: () => mocks.isSaaSMode,
        serverInfo: {
          sample: {
            available: mocks.sampleAvailable,
            instances: mocks.sampleInstances,
          },
        },
        totalInstanceCount: () => mocks.totalInstanceCount,
      }),
    {
      getState: () => ({
        instanceCountLimit: () => mocks.instanceCountLimit,
        activatedInstanceCount: () => mocks.activatedInstanceCount,
      }),
    }
  ),
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/components/instance", () => ({
  InfoPanel: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  InfoPanelContent: () => <div />,
  InstanceFormBody: () => <div data-testid="instance-form-body" />,
  InstanceFormButtons: ({ className }: { className?: string }) => (
    <button
      type="button"
      data-testid="instance-form-buttons"
      className={className}
      disabled={mocks.instanceFormContext.state.isRequesting}
    >
      Create
    </button>
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
  mocks.isSaaSMode = false;
  mocks.sampleAvailable = true;
  mocks.sampleInstances = [];
  mocks.totalInstanceCount = 0;
  mocks.instanceCountLimit = 10;
  mocks.activatedInstanceCount = 0;
  mocks.providerProps = undefined;
  mocks.instanceFormContext.state.isRequesting = false;
  mocks.instanceFormContext.valueChanged = false;
  mocks.instanceFormContext.sampleInstance = undefined;
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
      onSampleInstancePrepared: mocks.onCreated,
      onOpenSampleInstance: undefined,
    });

    act(() => {
      root.unmount();
    });
  });

  test("dismisses SaaS sample creation when the instance quota is reached", () => {
    mocks.isSaaSMode = true;
    mocks.instanceCountLimit = 1;
    mocks.activatedInstanceCount = 1;
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(
        <CreateInstanceView
          parent="projects/demo"
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "Instance quota reached",
      description: "You have reached the limit of 1 instances.",
    });
    expect(mocks.onDismiss).toHaveBeenCalledOnce();

    act(() => {
      root.unmount();
    });
  });

  const renderView = (parent?: string) => {
    const container = document.createElement("div");
    const root = createRoot(container);
    act(() => {
      root.render(
        <CreateInstanceView
          parent={parent}
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });
    return { container, unmount: () => act(() => root.unmount()) };
  };

  const sampleButton = (container: HTMLElement) =>
    [...container.querySelectorAll("button")].find((element) =>
      /sample/i.test(element.textContent ?? "")
    ) as HTMLButtonElement | undefined;

  describe("which sample option the form provider is given", () => {
    test("a project with no instance may prepare its sample", () => {
      mocks.isSaaSMode = true;
      const view = renderView("projects/demo");
      expect(mocks.providerProps?.onSampleInstancePrepared).toBe(
        mocks.onCreated
      );
      expect(mocks.providerProps?.onOpenSampleInstance).toBeUndefined();
      view.unmount();
    });

    test("none when the sample target is unavailable", () => {
      mocks.sampleAvailable = false;
      const view = renderView("projects/demo");
      expect(mocks.providerProps?.onSampleInstancePrepared).toBeUndefined();
      view.unmount();
    });

    test("none when the workspace already has an instance", () => {
      mocks.totalInstanceCount = 1;
      const view = renderView("projects/demo");
      expect(mocks.providerProps?.onSampleInstancePrepared).toBeUndefined();
      view.unmount();
    });

    test("none for workspace-owned instance creation", () => {
      const view = renderView();
      expect(mocks.providerProps?.onSampleInstancePrepared).toBeUndefined();
      view.unmount();
    });

    test("a project-owned sample that exists is opened, not prepared again", () => {
      mocks.sampleInstances = [{ instance: "projects/demo/instances/sample" }];
      const view = renderView("projects/other");
      expect(mocks.providerProps?.onSampleInstancePrepared).toBeUndefined();

      (mocks.providerProps?.onOpenSampleInstance as () => void)();
      expect(mocks.routerPush).toHaveBeenCalledWith({
        name: "workspace.project.database",
        params: { projectId: "demo" },
      });
      view.unmount();
    });

    test("a workspace-owned sample has no project to open", () => {
      mocks.sampleInstances = [{ instance: "instances/sample" }];
      const view = renderView("projects/demo");
      expect(mocks.providerProps?.onSampleInstancePrepared).toBeUndefined();
      expect(mocks.providerProps?.onOpenSampleInstance).toBeUndefined();
      view.unmount();
    });
  });

  describe("the sample offer above the form", () => {
    const prepare = (pending = false) => ({
      kind: "prepare" as const,
      pending,
      run: mocks.runSampleInstance,
    });

    test("is absent when the provider has no sample to prepare", () => {
      const view = renderView("projects/demo");
      expect(sampleButton(view.container)).toBeUndefined();
      view.unmount();
    });

    test("is absent when the sample only needs opening", () => {
      mocks.instanceFormContext.sampleInstance = {
        ...prepare(),
        kind: "open",
      };
      const view = renderView("projects/demo");
      expect(sampleButton(view.container)).toBeUndefined();
      view.unmount();
    });

    test("prepares the sample from the instance creation page", () => {
      mocks.instanceFormContext.sampleInstance = prepare();
      const view = renderView("projects/demo");
      act(() => sampleButton(view.container)?.click());
      expect(mocks.runSampleInstance).toHaveBeenCalledWith("instance_creation");
      view.unmount();
    });

    test("is disabled while preparation is running", () => {
      mocks.instanceFormContext.sampleInstance = prepare(true);
      const view = renderView("projects/demo");
      expect(sampleButton(view.container)).toBeDisabled();
      expect(sampleButton(view.container)).toHaveTextContent(
        "Preparing Sample Project Instance…"
      );
      view.unmount();
    });

    test("is disabled while regular creation is running", () => {
      mocks.instanceFormContext.sampleInstance = prepare();
      mocks.instanceFormContext.state.isRequesting = true;
      const view = renderView("projects/demo");
      expect(sampleButton(view.container)).toBeDisabled();
      view.unmount();
    });

    test("is permanent for a self-hosted project", () => {
      mocks.instanceFormContext.sampleInstance = prepare();
      const view = renderView("projects/demo");
      expect(view.container.textContent).toContain("Use sample instance");
      expect(view.container.textContent).toContain("ready-to-use database.");
      expect(view.container.textContent).not.toContain("7 days");
      view.unmount();
    });

    test("keeps seven-day copy for SaaS", () => {
      mocks.isSaaSMode = true;
      mocks.instanceFormContext.sampleInstance = prepare();
      const view = renderView("projects/demo");
      expect(view.container.textContent).toContain(
        "Use sample instance (Available for 7 days)"
      );
      expect(view.container.textContent).toContain(
        "ready-to-use database for 7 days."
      );
      view.unmount();
    });
  });
});
