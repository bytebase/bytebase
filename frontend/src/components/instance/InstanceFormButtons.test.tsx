import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSourceSchema,
  DataSourceType,
  InstanceSchema,
  SyncDatabasesSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { InstanceFormButtons } from "./InstanceFormButtons";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerCurrentName: "workspace.instance.create",
  routerCurrentQuery: {} as Record<string, unknown>,
  routerPush: vi.fn(),
  pushNotification: vi.fn(),
  createInstance: vi.fn(),
  fetchDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  captureMetric: vi.fn(),
  context: undefined as Record<string, unknown> | undefined,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
    currentRoute: {
      get value() {
        return {
          name: mocks.routerCurrentName,
          query: mocks.routerCurrentQuery,
        };
      },
    },
  },
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

vi.mock("@/stores/app", () => {
  const appState = {
    hasFeature: () => true,
    createInstance: mocks.createInstance,
    fetchDatabases: mocks.fetchDatabases,
    batchUpdateDatabases: mocks.batchUpdateDatabases,
  };
  const useAppStore = Object.assign(
    (selector: (state: typeof appState) => unknown) => selector(appState),
    { getState: () => appState }
  );
  return { useAppStore };
});

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  convertKVListToLabels: (list: { key: string; value: string }[]) =>
    Object.fromEntries(list.map(({ key, value }) => [key, value])),
  extractInstanceResourceName: (name: string) =>
    name.replace(/^instances\//, ""),
  isValidSpannerDataSource: (ds: { projectId: string; instanceId: string }) =>
    ds.projectId !== "" && ds.instanceId !== "",
  isValidBigQueryDataSource: (ds: { projectId: string }) =>
    ds.projectId !== "",
}));

vi.mock("../ui/alert-dialog", () => ({
  AlertDialog: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
  }) => (open ? <div>{children}</div> : null),
  AlertDialogContent: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => (
    <div className={className} data-testid="alert-dialog-content">
      {children}
    </div>
  ),
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("../ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
  }: {
    children: React.ReactNode;
    disabled?: boolean;
    onClick?: () => void;
  }) => (
    <button disabled={disabled} type="button" onClick={onClick}>
      {children}
    </button>
  ),
}));

vi.mock("../ui/sticky-action-footer", () => ({
  StickyActionFooter: ({
    left,
    right,
  }: {
    left?: React.ReactNode;
    right?: React.ReactNode;
  }) => (
    <div>
      {left}
      {right}
    </div>
  ),
}));

vi.mock("./InstanceFormContext", () => ({
  useInstanceFormContext: () => mocks.context,
}));

const flushPromises = async () => {
  await Promise.resolve();
  await Promise.resolve();
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.routerCurrentName = "workspace.instance.create";
  mocks.routerCurrentQuery = {};

  const adminDataSource = {
    ...create(DataSourceSchema, {
      id: "admin",
      type: DataSourceType.ADMIN,
      host: "127.0.0.1",
      port: "5432",
    }),
    pendingCreate: true,
    updatedPassword: "",
    updatedMasterPassword: "",
    updatedToken: "",
  };

  mocks.context = {
    state: { isRequesting: false, isTestingConnection: false },
    setState: vi.fn((updater) => {
      if (!mocks.context) return;
      const nextState =
        typeof updater === "function" ? updater(mocks.context.state) : updater;
      mocks.context = { ...mocks.context, state: nextState };
    }),
    instance: undefined,
    isCreating: true,
    allowEdit: true,
    allowCreate: true,
    basicInfo: create(InstanceSchema, {
      title: "Production",
      engine: Engine.POSTGRES,
      environment: "environments/prod",
    }),
    setBasicInfo: vi.fn(),
    labelKVList: [],
    adminDataSource,
    editingDataSource: adminDataSource,
    readonlyDataSourceList: [],
    setDataSourceEditState: vi.fn(),
    hasReadonlyReplicaFeature: true,
    setMissingFeature: vi.fn(),
    testConnection: vi.fn(async () => ({ success: true, message: "" })),
    checkDataSource: vi.fn(() => true),
    extractDataSourceFromEdit: vi.fn(() =>
      create(DataSourceSchema, {
        id: "admin",
        type: DataSourceType.ADMIN,
        host: "127.0.0.1",
        port: "5432",
      })
    ),
    valueChanged: true,
    onDismiss: vi.fn(),
    emitShowConnectionOptions: vi.fn(),
  };

  mocks.createInstance.mockResolvedValue(
    create(InstanceSchema, {
      name: "instances/prod",
      title: "Production",
      engine: Engine.POSTGRES,
    })
  );
  mocks.batchUpdateDatabases.mockResolvedValue([]);
});

describe("InstanceFormButtons", () => {
  test("uses project-aware create action text when creating from a project", async () => {
    mocks.routerCurrentQuery = { project: "demo" };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    expect(container.textContent).toContain(
      "instance.connect-database-to-project"
    );
    expect(container.textContent).not.toContain("common.create");

    await act(async () => {
      root.unmount();
    });
  });

  test("passes project context to instance creation without client-side database transfer", async () => {
    mocks.routerCurrentQuery = { project: "demo" };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) =>
        button.textContent?.includes("instance.connect-database-to-project")
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
      await flushPromises();
    });

    expect(mocks.createInstance).toHaveBeenCalledWith(
      expect.anything(),
      false,
      {
        initialDatabaseProject: "projects/demo",
      }
    );
    expect(mocks.fetchDatabases).not.toHaveBeenCalled();
    expect(mocks.batchUpdateDatabases).not.toHaveBeenCalled();
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "demo" },
      query: {
        intro: "project-instance-synced",
        syncingInstance: "prod",
      },
    });
    expect(mocks.context?.onDismiss).not.toHaveBeenCalled();
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "instance create clicked",
      properties: {
        route_id: "workspace.instance.create",
        resource: "projects/demo",
      },
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("captures standalone connection test clicks on instance edit", async () => {
    mocks.routerCurrentName = "workspace.instance.detail";
    const adminDataSource = create(DataSourceSchema, {
      id: "admin",
      type: DataSourceType.ADMIN,
      host: "127.0.0.1",
      port: "5432",
    });
    mocks.context = {
      ...mocks.context,
      instance: create(InstanceSchema, {
        name: "instances/prod",
        title: "Production",
        engine: Engine.POSTGRES,
        dataSources: [adminDataSource],
      }),
      isCreating: false,
      adminDataSource: {
        ...adminDataSource,
        pendingCreate: false,
        updatedPassword: "",
        updatedMasterPassword: "",
        updatedToken: "",
      },
      editingDataSource: {
        ...adminDataSource,
        pendingCreate: false,
        updatedPassword: "",
        updatedMasterPassword: "",
        updatedToken: "",
      },
      valueChanged: true,
    };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const testConnectionButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) =>
      button.textContent?.includes("instance.test-connection")
    ) as HTMLButtonElement;
    await act(async () => {
      testConnectionButton.click();
      await flushPromises();
    });

    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "instance connection test clicked",
      properties: {
        route_id: "workspace.instance.detail",
      },
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("sets empty sync databases when sync-all is unchecked and no databases are selected", async () => {
    mocks.context = {
      ...mocks.context,
      basicInfo: create(InstanceSchema, {
        title: "Production",
        engine: Engine.POSTGRES,
        environment: "environments/prod",
        syncDatabases: create(SyncDatabasesSchema, { databases: [] }),
      }),
    };
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.create")
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
      await flushPromises();
    });

    expect(mocks.createInstance).toHaveBeenCalledWith(
      expect.objectContaining({
        syncDatabases: create(SyncDatabasesSchema, { databases: [] }),
      }),
      false,
      {
        initialDatabaseProject: undefined,
      }
    );

    await act(async () => {
      root.unmount();
    });
  });

  test("redirects no-project instance creation to the instance databases tab", async () => {
    mocks.routerCurrentQuery = {};
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.create")
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
      await flushPromises();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance.detail",
      params: { instanceId: "prod" },
      query: { syncingInstance: "prod" },
      hash: "databases",
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("uses structured recovery category from connection test failures", async () => {
    const context = mocks.context as {
      emitShowConnectionOptions: ReturnType<typeof vi.fn>;
      testConnection: ReturnType<typeof vi.fn>;
    };
    context.testConnection.mockResolvedValue({
      success: false,
      message: "dial tcp 10.0.0.5:5432: i/o timeout",
      failureCategory: "ssl_tls_failed",
    });

    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.create")
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
      await flushPromises();
    });

    expect(container.textContent).toContain(
      "instance.connection-recovery.tls.title"
    );
    expect(container.textContent).toContain(
      "instance.connection-recovery.tls.description"
    );
    expect(context.emitShowConnectionOptions).not.toHaveBeenCalled();
    expect(mocks.createInstance).not.toHaveBeenCalled();

    await act(async () => {
      root.unmount();
    });
  });

  test("renders a wider connection failure dialog", async () => {
    const context = mocks.context as {
      testConnection: ReturnType<typeof vi.fn>;
    };
    context.testConnection.mockResolvedValue({
      success: false,
      message: "permission denied",
      failureCategory: "permission_denied",
    });

    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent?.includes("common.create")
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
      await flushPromises();
    });

    expect(
      container.querySelector('[data-testid="alert-dialog-content"]')
    ).toHaveClass("max-w-2xl");

    await act(async () => {
      root.unmount();
    });
  });
});
