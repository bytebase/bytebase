import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSourceSchema,
  DataSourceType,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { InstanceFormButtons } from "./InstanceFormButtons";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  routerCurrentQuery: {} as Record<string, unknown>,
  routerPush: vi.fn(),
  pushNotification: vi.fn(),
  createInstance: vi.fn(),
  fetchDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  context: undefined as Record<string, unknown> | undefined,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
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

vi.mock("@/react/stores/app", () => {
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

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  convertKVListToLabels: (list: { key: string; value: string }[]) =>
    Object.fromEntries(list.map(({ key, value }) => [key, value])),
  extractInstanceResourceName: (name: string) =>
    name.replace(/^instances\//, ""),
  isValidSpannerHost: (host: string) => host !== "",
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
      syncDatabases: [],
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
        project: "projects/demo",
      }
    );
    expect(mocks.fetchDatabases).not.toHaveBeenCalled();
    expect(mocks.batchUpdateDatabases).not.toHaveBeenCalled();
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "demo" },
      query: {
        syncingInstance: "prod",
        intro: "connect-database",
      },
    });
    expect(mocks.context?.onDismiss).not.toHaveBeenCalled();

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
