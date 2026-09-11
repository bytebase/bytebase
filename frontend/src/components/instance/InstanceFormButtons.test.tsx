import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSourceExternalSecret_SecretType,
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
  updateInstance: vi.fn(),
  getInstanceByName: vi.fn(),
  fetchDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  captureMetric: vi.fn(),
  hasFeature: vi.fn(() => true),
  onCreated: vi.fn(),
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
    hasFeature: mocks.hasFeature,
    instanceLicenseCount: () => 100,
    activatedInstanceCount: () => 1,
    isSaaSMode: () => false,
    createInstance: mocks.createInstance,
    updateInstance: mocks.updateInstance,
    getInstanceByName: mocks.getInstanceByName,
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
  calcUpdateMask: () => [],
  convertKVListToLabels: (list: { key: string; value: string }[]) =>
    Object.fromEntries(list.map(({ key, value }) => [key, value])),
  extractInstanceResourceName: (name: string) => name.split("/").at(-1) ?? "",
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
  mocks.hasFeature.mockReturnValue(true);
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
    labelErrors: [],
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
    emitDataSourceReset: vi.fn(),
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
  test("does not require the external-secret feature for an inactive IAM draft", async () => {
    mocks.hasFeature.mockReturnValue(false);
    const dataSource = create(DataSourceSchema, {
      id: "admin", type: DataSourceType.ADMIN,
      authenticationType: DataSource_AuthenticationType.AZURE_IAM,
      host: "db.example.com", password: "{{inactive-password}}",
      externalSecret: { secretType: DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT },
    });
    const saved = create(InstanceSchema, {
      name: "instances/prod", title: "Before", engine: Engine.POSTGRES,
      dataSources: [dataSource],
    });
    mocks.getInstanceByName.mockReturnValue(saved);
    mocks.updateInstance.mockResolvedValue(saved);
    mocks.context = {
      ...mocks.context, instance: saved, isCreating: false,
      basicInfo: { ...saved, title: "After" },
      adminDataSource: dataSource, editingDataSource: dataSource,
    };
    const container = document.createElement("div");
    const root = createRoot(container);
    try {
      await act(async () => { root.render(<InstanceFormButtons />); });
      const update = Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "common.update")!;
      await act(async () => { update.click(); });
      expect(mocks.context.setMissingFeature).not.toHaveBeenCalled();
      expect(mocks.updateInstance).toHaveBeenCalledOnce();
    } finally {
      await act(async () => { root.unmount(); });
    }
  });

  test.each([
    { title: "Production", labelErrors: ["invalid label"] },
    { title: "   ", labelErrors: [] },
  ])("blocks Update but allows connection testing for invalid metadata: %j", async ({ title, labelErrors }) => {
    mocks.context = {
      ...mocks.context,
      isCreating: false,
      instance: create(InstanceSchema, { name: "instances/prod" }),
      basicInfo: create(InstanceSchema, { title, engine: Engine.POSTGRES }),
      labelErrors,
    };
    const container = document.createElement("div");
    const root = createRoot(container);
    try {
      await act(async () => { root.render(<InstanceFormButtons />); });
      const buttons = Array.from(container.querySelectorAll("button"));
      const update = buttons.find((button) => button.textContent === "common.update")!;
      const testConnection = buttons.find((button) => button.textContent === "instance.test-connection")!;
      expect(update.disabled).toBe(true);
      expect(testConnection.disabled).toBe(false);
      await act(async () => {
        update.click();
        testConnection.click();
      });
      expect(mocks.updateInstance).not.toHaveBeenCalled();
      expect(mocks.context.testConnection).toHaveBeenCalledExactlyOnceWith(
        mocks.context.editingDataSource, false
      );
    } finally {
      await act(async () => { root.unmount(); });
    }
  });

  test.each(["isRequesting", "isTestingConnection"])(
    "blocks connection testing during %s on edit",
    async (pendingState) => {
      mocks.context = {
        ...mocks.context,
        isCreating: false,
        instance: create(InstanceSchema, { name: "instances/prod" }),
        state: { isRequesting: false, isTestingConnection: false, [pendingState]: true },
      };
      const container = document.createElement("div");
      const root = createRoot(container);
      try {
        await act(async () => { root.render(<InstanceFormButtons />); });
        const testConnection = Array.from(container.querySelectorAll("button")).find(
          (button) => ["instance.test-connection", "instance.testing-connection"].includes(button.textContent ?? "")
        )!;
        expect(testConnection.disabled).toBe(true);
        await act(async () => { testConnection.click(); });
        expect(mocks.context.testConnection).not.toHaveBeenCalled();
      } finally {
        await act(async () => { root.unmount(); });
      }
    }
  );

  test("invalidates provider drafts only after a successful server-backed save", async () => {
    const saved = create(InstanceSchema, {
      name: "instances/prod",
      title: "Updated production",
      engine: Engine.POSTGRES,
      dataSources: [
        create(DataSourceSchema, {
          id: "admin",
          type: DataSourceType.ADMIN,
          host: "127.0.0.1",
          port: "5432",
        }),
      ],
    });
    let finishSave!: () => void;
    mocks.updateInstance.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishSave = resolve;
        })
    );
    mocks.getInstanceByName.mockReturnValue(saved);
    const emitDataSourceReset = vi.fn();
    mocks.context = {
      ...mocks.context,
      instance: create(InstanceSchema, { ...saved, title: "Old production" }),
      basicInfo: saved,
      isCreating: false,
      emitDataSourceReset,
    };
    const container = document.createElement("div");
    const root = createRoot(container);
    await act(async () => {
      root.render(<InstanceFormButtons />);
    });
    const update = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.update"
    )!;
    expect(update.disabled).toBe(false);
    await act(async () => {
      update.click();
    });
    expect(mocks.updateInstance).toHaveBeenCalledOnce();
    expect(emitDataSourceReset).not.toHaveBeenCalled();
    await act(async () => {
      finishSave();
    });
    expect(mocks.getInstanceByName).toHaveBeenCalledWith(saved.name);
    expect(emitDataSourceReset).toHaveBeenCalledOnce();
    expect(mocks.context?.setDataSourceEditState).toHaveBeenCalledOnce();
    await act(async () => {
      root.unmount();
    });
  });

  test("uses project-aware create action text when creating from a project", async () => {
    mocks.context = { ...mocks.context, parent: "projects/demo" };
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
    mocks.routerCurrentName = "workspace.project.instance.create";
    mocks.context = { ...mocks.context, parent: "projects/demo" };
    mocks.createInstance.mockResolvedValue(
      create(InstanceSchema, {
        name: "projects/demo/instances/prod",
        title: "Production",
        engine: Engine.POSTGRES,
      })
    );
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons onCreated={mocks.onCreated} />);
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
        parent: "projects/demo",
      }
    );
    expect(mocks.fetchDatabases).not.toHaveBeenCalled();
    expect(mocks.batchUpdateDatabases).not.toHaveBeenCalled();
    expect(mocks.onCreated).toHaveBeenCalledWith(
      expect.objectContaining({ name: "projects/demo/instances/prod" })
    );
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.context?.onDismiss).not.toHaveBeenCalled();
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "instance create clicked",
      properties: {
        route_id: "workspace.project.instance.create",
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
      undefined
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
      query: {
        syncingInstance: "prod",
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
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
    expect(container.textContent).toContain(
      "instance.failed-to-connect-instance"
    );
    expect(container.textContent).not.toContain("common.warning");
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

  // Pins the contract every field-level validation in this form rests on:
  // failing checkDataSource is what keeps a data source away from the
  // connection test, so the operator reads the form's own message instead of a
  // connection failure. The keytab resupply gate is one caller of it; this
  // test covers the wiring, not that gate — InstanceFormContext.test.tsx
  // covers which data sources checkDataSource fails.
  test("a failing data source check keeps both save actions off the connection test", async () => {
    const adminDataSource = create(DataSourceSchema, {
      id: "admin",
      type: DataSourceType.ADMIN,
      host: "prod.example.com",
      port: "5432",
    });
    const editState = {
      ...adminDataSource,
      pendingCreate: false,
      updatedPassword: "",
      updatedMasterPassword: "",
      updatedToken: "",
    };
    mocks.context = {
      ...mocks.context,
      instance: create(InstanceSchema, {
        name: "instances/prod",
        title: "Production",
        engine: Engine.POSTGRES,
        dataSources: [adminDataSource],
      }),
      isCreating: false,
      adminDataSource: editState,
      editingDataSource: editState,
      checkDataSource: vi.fn(() => false),
      valueChanged: true,
    };
    const context = mocks.context as {
      testConnection: ReturnType<typeof vi.fn>;
    };

    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(<InstanceFormButtons />);
    });

    const buttons = Array.from(container.querySelectorAll("button"));
    const updateButton = buttons.find((button) =>
      button.textContent?.includes("common.update")
    ) as HTMLButtonElement;
    const testConnectionButton = buttons.find((button) =>
      button.textContent?.includes("instance.test-connection")
    ) as HTMLButtonElement;

    expect(updateButton.disabled).toBe(true);
    expect(testConnectionButton.disabled).toBe(true);

    await act(async () => {
      updateButton.click();
      testConnectionButton.click();
      await flushPromises();
    });

    expect(context.testConnection).not.toHaveBeenCalled();

    await act(async () => {
      root.unmount();
    });
  });
});
