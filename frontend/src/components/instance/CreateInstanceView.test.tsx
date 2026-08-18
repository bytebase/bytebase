import { act } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  onDismiss: vi.fn(),
  onCreated: vi.fn(),
  prepareSampleProjectInstance: vi.fn(),
  pushNotification: vi.fn(),
  isSaaSMode: false,
  instanceCountLimit: 10,
  activatedInstanceCount: 0,
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
  useTranslation: () => ({
    t: (key: string, values?: { total?: number }) => {
      const translation =
        ({
        "instance.use-sample-instance":
          "Use sample instance (Available for 7 days)",
        "instance.preparing-sample-instance":
          "Preparing Sample Project Instance…",
        "instance.prepare-sample-instance-failed":
          "Failed to prepare Sample Project Instance.",
        "instance.sample-project-instance-description":
          "Use a Sample Project Instance to explore Bytebase with a ready-to-use database for 7 days.",
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

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    <T,>(selector: (state: Record<string, unknown>) => T) =>
      selector({
        isSaaSMode: () => mocks.isSaaSMode,
        prepareSampleProjectInstance: mocks.prepareSampleProjectInstance,
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
  mocks.isSaaSMode = false;
  mocks.instanceCountLimit = 10;
  mocks.activatedInstanceCount = 0;
  mocks.prepareSampleProjectInstance.mockResolvedValue({
    name: "projects/demo/instances/sample",
  });
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

  test("prepares the sample instance and follows the existing created flow", async () => {
    mocks.isSaaSMode = true;
    const instance = { name: "projects/demo/instances/sample" };
    mocks.prepareSampleProjectInstance.mockResolvedValue(instance);
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

    const button = [...container.querySelectorAll("button")].find((element) =>
      element.textContent?.includes("Use sample instance")
    ) as HTMLButtonElement;
    await act(async () => {
      button.click();
    });

    expect(mocks.prepareSampleProjectInstance).toHaveBeenCalledWith(
      "projects/demo"
    );
    expect(mocks.onCreated).toHaveBeenCalledWith(instance);

    act(() => {
      root.unmount();
    });
  });

  test("disables the sample action while preparation is running", async () => {
    mocks.isSaaSMode = true;
    let resolvePreparation: (instance: { name: string }) => void;
    mocks.prepareSampleProjectInstance.mockReturnValue(
      new Promise((resolve) => {
        resolvePreparation = resolve;
      })
    );
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

    const button = [...container.querySelectorAll("button")].find((element) =>
      element.textContent?.includes("Use sample instance")
    ) as HTMLButtonElement;
    act(() => {
      button.click();
    });

    expect(button).toBeDisabled();
    expect(button).toHaveTextContent("Preparing Sample Project Instance…");

    await act(async () => {
      resolvePreparation!({ name: "projects/demo/instances/sample" });
    });

    act(() => {
      root.unmount();
    });
  });

  test("shows localized feedback when sample preparation fails", async () => {
    mocks.isSaaSMode = true;
    mocks.prepareSampleProjectInstance.mockRejectedValue(
      new ConnectError(
        "The sample instance cannot be provisioned. Try again later.",
        Code.Internal
      )
    );
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

    const button = [...container.querySelectorAll("button")].find((element) =>
      element.textContent?.includes("Use sample instance")
    ) as HTMLButtonElement;
    await act(async () => {
      button.click();
    });

    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to prepare Sample Project Instance.",
      description:
        "[internal] The sample instance cannot be provisioned. Try again later.",
    });

    act(() => {
      root.unmount();
    });
  });

  test("keeps sample instance creation out of self-hosted project and SaaS workspace creation", () => {
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

    expect(container.textContent).not.toContain("Use sample instance");

    mocks.isSaaSMode = true;
    act(() => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    expect(container.textContent).not.toContain("Use sample instance");

    act(() => {
      root.unmount();
    });
  });
});
