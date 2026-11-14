import { act } from "react";
import { render, fireEvent, cleanup } from "@testing-library/react";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { Vendor } from "@/types/proto-es/v1/instance_service_pb";
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
  captureMetric: vi.fn(),
  pushNotification: vi.fn(),
  openAgent: vi.fn(),
  isSaaSMode: false,
  sampleAvailable: true,
  sampleInstances: [] as { instance: string }[],
  totalInstanceCount: 0,
  instanceCountLimit: 10,
  activatedInstanceCount: 0,
  providerProps: undefined as Record<string, unknown> | undefined,
  instanceFormContext: {
    basicInfo: { engine: 0, vendor: 0 },
    state: { isRequesting: false },
    setState: vi.fn(),
    valueChanged: false,
  },
}));

let CreateInstanceView: typeof import("./CreateInstanceView").CreateInstanceView;

vi.mock("./NeonImportForm", () => ({
  NeonStepIndicator: () => <div data-testid="neon-steps" />,
  neonDestinationLabel: (_t: unknown, parent?: string) =>
    parent ? `project ${parent}` : "workspace",
  NeonImportForm: ({
    active,
    onStepChange,
    onBusyChange,
    onDirtyChange,
  }: {
    active: boolean;
    onStepChange: (step: number) => void;
    onBusyChange: (busy: boolean) => void;
    onDirtyChange: (dirty: boolean) => void;
  }) => (
    <div hidden={!active}>
      <input
        aria-label="discovery draft"
        defaultValue=""
        onChange={() => onDirtyChange(true)}
      />
      <button type="button" onClick={() => onStepChange(1)}>
        mock discovered
      </button>
      <button type="button" onClick={() => onStepChange(0)}>
        mock back to connect
      </button>
      <button type="button" onClick={() => onBusyChange(true)}>
        mock busy
      </button>
    </div>
  ),
}));

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

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
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
        instanceCountLimit: () => mocks.instanceCountLimit,
        instanceLicenseCount: () => mocks.instanceCountLimit,
        activatedInstanceCount: () => mocks.activatedInstanceCount,
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

vi.mock("@/modules/agent/store/agent", () => ({
  useAgentStore: {
    getState: () => ({ open: mocks.openAgent }),
  },
}));

vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: vi.fn(),
}));

vi.mock("@/components/instance", () => ({
  InfoPanel: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  InfoPanelContent: () => <div />,
  InstanceFormBody: ({
    hideEngine,
    onAskAgent,
  }: {
    hideEngine?: boolean;
    onAskAgent?: () => void;
  }) => (
    <div data-testid="instance-form-body">
      {hideEngine && <input aria-label="manual draft" defaultValue="" />}
      {onAskAgent && (
        <button type="button" onClick={() => onAskAgent()}>
          instance.ask-agent
        </button>
      )}
    </div>
  ),
  InstanceFormButtons: ({ className }: { className?: string }) => (
    <div>
      <button
        type="button"
        data-testid="instance-form-buttons"
        className={className}
        disabled={mocks.instanceFormContext.state.isRequesting}
      >
        Create
      </button>
    </div>
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
  mocks.prepareSampleProjectInstance.mockResolvedValue({
    name: "projects/demo/instances/sample",
  });
  mocks.providerProps = undefined;
  mocks.instanceFormContext.state.isRequesting = false;
  mocks.instanceFormContext.basicInfo = { engine: 0, vendor: 0 };
  mocks.instanceFormContext.setState.mockImplementation(
    (
      updater: (state: { isRequesting: boolean }) => {
        isRequesting: boolean;
      }
    ) => {
      mocks.instanceFormContext.state = updater(
        mocks.instanceFormContext.state
      );
    }
  );
  mocks.instanceFormContext.valueChanged = false;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as typeof ResizeObserver;
  ({ CreateInstanceView } = await import("./CreateInstanceView"));
});

describe("CreateInstanceView", () => {
  test("opens the page-detected Agent landing from the create-instance form", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        <CreateInstanceView
          onDismiss={mocks.onDismiss}
          onCreated={mocks.onCreated}
        />
      );
    });

    const askAgentButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("instance.ask-agent"));
    expect(askAgentButton).toBeDefined();

    await act(async () => {
      askAgentButton?.click();
    });

    expect(mocks.openAgent).toHaveBeenCalledOnce();

    await act(async () => {
      root.unmount();
    });
  });

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

    const scroller = container
      .querySelector("[data-testid='instance-form-body']")
      ?.closest(".overflow-auto");
    expect(scroller?.firstElementChild).toHaveClass("px-4");
    expect(scroller?.firstElementChild).toHaveClass("sm:px-6");

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
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "sample instance requested",
      properties: { source: "instance_creation" },
    });
    expect(mocks.captureMetric.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.prepareSampleProjectInstance.mock.invocationCallOrder[0]
    );
    expect(mocks.onCreated).toHaveBeenCalledWith(instance);

    act(() => {
      root.unmount();
    });
  });

  test("hides sample instance creation when the target is unavailable", () => {
    mocks.isSaaSMode = true;
    mocks.sampleAvailable = false;
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

    act(() => {
      root.unmount();
    });
  });

  test("hides sample instance creation when a sample was already provisioned", () => {
    mocks.isSaaSMode = true;
    mocks.sampleInstances = [{ instance: "instances/sample" }];
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

    act(() => {
      root.unmount();
    });
  });

  test("hides sample instance creation when the workspace has an instance", () => {
    mocks.isSaaSMode = true;
    mocks.totalInstanceCount = 1;
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
    expect(
      container.querySelector("[data-testid='instance-form-buttons']")
    ).toBeDisabled();

    await act(async () => {
      resolvePreparation!({ name: "projects/demo/instances/sample" });
    });

    act(() => {
      root.unmount();
    });
  });

  test("disables the sample action while regular creation is running", () => {
    mocks.isSaaSMode = true;
    mocks.instanceFormContext.state.isRequesting = true;
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
    expect(button).toBeDisabled();

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

    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to prepare Sample Project Instance.",
      description:
        "[internal] The sample instance cannot be provisioned. Try again later.",
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "sample instance requested",
      properties: { source: "instance_creation" },
    });

    act(() => {
      root.unmount();
    });
  });

  test("shows permanent sample creation for a self-hosted project", () => {
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

    expect(container.textContent).toContain("Use sample instance");
    expect(container.textContent).toContain("ready-to-use database.");
    expect(container.textContent).not.toContain("7 days");

    act(() => {
      root.unmount();
    });
  });

  test("keeps seven-day sample copy for SaaS", () => {
    mocks.isSaaSMode = true;
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

    expect(container.textContent).toContain(
      "Use sample instance (Available for 7 days)"
    );
    expect(container.textContent).toContain("ready-to-use database for 7 days.");

    act(() => {
      root.unmount();
    });
  });

  test("keeps sample creation out of workspace-owned instance creation", () => {
    mocks.isSaaSMode = true;
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

    expect(container.textContent).not.toContain("Use sample instance");

    act(() => {
      root.unmount();
    });
  });
});

const neon = () => {
  mocks.instanceFormContext.basicInfo = {
    engine: Engine.POSTGRES,
    vendor: Vendor.NEON,
  };
};
const props = { onDismiss: mocks.onDismiss, onCreated: mocks.onCreated };

test("Neon reveals both setup methods as peers and keeps a separate draft for each", () => {
  neon();
  const view = render(<CreateInstanceView {...props} />);
  const discovery = view.getByRole("radio", {
    name: /instance.neon.method.discover/,
  });
  expect(discovery).toHaveAttribute("aria-checked", "true");
  expect(view.getByTestId("neon-steps")).toBeInTheDocument();
  fireEvent.change(view.getByLabelText("discovery draft"), {
    target: { value: "retained key" },
  });
  fireEvent.click(
    view.getByRole("radio", { name: /instance.neon.method.manual/ })
  );
  expect(view.getByLabelText("manual draft")).toBeVisible();
  expect(view.getByLabelText("discovery draft")).not.toBeVisible();
  // Manual entry is the existing form, with no step indicator.
  expect(view.queryByTestId("neon-steps")).toBeNull();
  fireEvent.change(view.getByLabelText("manual draft"), {
    target: { value: "retained host" },
  });
  fireEvent.click(discovery);
  expect(view.getByLabelText("discovery draft")).toHaveValue("retained key");
  fireEvent.click(
    view.getByRole("radio", { name: /instance.neon.method.manual/ })
  );
  expect(view.getByLabelText("manual draft")).toHaveValue("retained host");
  cleanup();
});

test("collapses the engine, hosting, and method controls after Connect", () => {
  neon();
  const view = render(<CreateInstanceView {...props} />);
  expect(view.getByText("instance.neon.entry.summary")).toBeVisible();
  fireEvent.click(view.getByText("mock discovered"));
  expect(
    view.queryByRole("radiogroup", { name: "instance.neon.method.title" })
  ).toBeNull();
  expect(view.getByText("instance.neon.entry.summary")).not.toBeVisible();
  fireEvent.click(view.getByText("mock back to connect"));
  expect(
    view.getByRole("radiogroup", { name: "instance.neon.method.title" })
  ).toBeVisible();
  cleanup();
});

test("opens each wizard step at its top", () => {
  neon();
  const view = render(<CreateInstanceView {...props} />);
  const scroller = view.container.querySelector(".overflow-auto") as HTMLElement;
  scroller.scrollTop = 480;
  fireEvent.click(view.getByText("mock discovered"));
  expect(scroller.scrollTop).toBe(0);
  scroller.scrollTop = 480;
  fireEvent.click(view.getByText("mock back to connect"));
  expect(scroller.scrollTop).toBe(0);
  cleanup();
});

test("warns on the Neon entry screen, for either method, when no instance can be added", () => {
  neon();
  mocks.totalInstanceCount = 10;
  const view = render(<CreateInstanceView {...props} />);
  expect(view.getByText("instance.neon.limit-reached.title")).toBeVisible();
  fireEvent.click(
    view.getByRole("radio", { name: /instance.neon.method.manual/ })
  );
  expect(view.getByText("instance.neon.limit-reached.title")).toBeVisible();
  cleanup();
  mocks.totalInstanceCount = 9;
  const room = render(<CreateInstanceView {...props} />);
  expect(room.queryByText("instance.neon.limit-reached.title")).toBeNull();
  cleanup();
});

test("keeps the engine and hosting pickers behind Change once Neon is chosen", () => {
  neon();
  const view = render(<CreateInstanceView {...props} />);
  const engine = view.getAllByTestId("instance-form-body")[0];
  expect(engine).not.toBeVisible();
  fireEvent.click(
    view.getByRole("button", { name: "instance.neon.entry.change" })
  );
  expect(engine).toBeVisible();
  cleanup();
});

test("states the destination setup was opened from and offers no control to change it", () => {
  neon();
  const workspace = render(<CreateInstanceView {...props} />);
  expect(
    workspace.getByText("instance.neon.destination.adding")
  ).toBeVisible();
  expect(
    workspace.getByText(/instance.neon.destination.workspace-help/)
  ).toBeVisible();
  cleanup();
  const project = render(
    <CreateInstanceView {...props} parent="projects/checkout" />
  );
  expect(
    project.getByText(/instance.neon.destination.project-help/)
  ).toBeVisible();
  expect(project.queryAllByRole("combobox")).toHaveLength(0);
  cleanup();
});

test("disables switching methods while either one is submitting", () => {
  neon();
  const view = render(<CreateInstanceView {...props} />);
  fireEvent.click(view.getByText("mock busy"));
  for (const radio of view.getAllByRole("radio"))
    expect(radio).toHaveAttribute("aria-disabled", "true");
  cleanup();
  mocks.instanceFormContext.state.isRequesting = true;
  const manual = render(<CreateInstanceView {...props} />);
  for (const radio of manual.getAllByRole("radio"))
    expect(radio).toHaveAttribute("aria-disabled", "true");
  cleanup();
});
