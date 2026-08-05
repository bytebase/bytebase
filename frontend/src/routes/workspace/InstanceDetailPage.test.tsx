import type { ReactElement, ReactNode } from "react";
import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  location: {
    pathname: "/instances/prod",
    search: "?syncingInstance=prod",
    hash: "#databases",
  },
  navigate: vi.fn(),
  databases: [
    {
      name: "instances/prod/databases/app",
      project: "projects/default",
    },
  ] as Database[],
  projects: [{ name: "projects/default" }] as Project[],
  instance: {
    name: "instances/prod",
    title: "Prod",
    engine: 1,
    state: 1,
    environment: "environments/test",
    roles: [],
  } as unknown as Instance,
  fetchProjectList: vi.fn(async () => ({
    projects: [] as Project[],
    nextPageToken: "",
  })),
  getOrFetchInstanceByName: vi.fn(),
  removeDatabaseMetadataCache: vi.fn(),
  removeCacheByInstance: vi.fn(),
  syncInstance: vi.fn(),
  batchSyncDatabases: vi.fn(),
  batchUpdateDatabases: vi.fn(),
  pushNotification: vi.fn(),
  routerPush: vi.fn(),
}));

vi.mock("react-router", () => ({
  useLocation: () => mocks.location,
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/app/router", () => ({
  PROJECT_V1_ROUTE_DATABASES: "workspace.project.database",
  router: {
    beforeEach: () => () => {},
    push: mocks.routerPush,
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  Trans: ({
    i18nKey,
    components,
  }: {
    i18nKey: string;
    components?: { instance?: ReactNode };
  }) => (
    <>
      {i18nKey}
      {components?.instance}
    </>
  ),
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    databasesByName: {},
    projectsByName: {},
    getDatabaseByName: (name: string) =>
      mocks.databases.find((database) => database.name === name) ??
      mocks.databases[0],
    removeDatabaseMetadataCache: mocks.removeDatabaseMetadataCache,
    instancesByName: { "instances/prod": mocks.instance },
    environmentList: [],
    serverInfo: { defaultProject: "projects/default" },
  });
  const useAppStore = (selector?: (s: ReturnType<typeof getState>) => unknown) =>
    selector ? selector(getState()) : getState();
  useAppStore.getState = () => ({
    ...getState(),
    getOrFetchInstanceByName: mocks.getOrFetchInstanceByName,
    syncInstance: mocks.syncInstance,
    removeCacheByInstance: mocks.removeCacheByInstance,
    batchSyncDatabases: mocks.batchSyncDatabases,
    batchUpdateDatabases: mocks.batchUpdateDatabases,
    fetchProjectList: mocks.fetchProjectList,
  });
  return { useAppStore };
});

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const [, instanceName, , databaseName] = name.split("/");
    return { instanceName, databaseName };
  },
  extractInstanceResourceName: (name: string) => name.split("/").pop() ?? "",
  extractProjectResourceName: (name: string) => name.split("/").pop() ?? "",
  getDefaultPagination: () => 10,
  hasWorkspacePermissionV2: () => true,
  instanceV1Name: (instance: Instance) => instance.title,
  isNullOrUndefined: (value: unknown) => value === null || value === undefined,
  isValidDatabaseName: (name: string) =>
    /^instances\/[^/]+\/databases\/[^/]+$/.test(name),
  setDocumentTitle: vi.fn(),
}));

vi.mock("@/components/AdvancedSearch", () => ({
  AdvancedSearch: () => <div />,
  getValueFromScopes: () => undefined,
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: () => <div />,
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => <div />,
}));

vi.mock("@/components/EditEnvironmentSheet", () => ({
  EditEnvironmentSheet: () => null,
}));

vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsList: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsPanel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  TabsTrigger: ({ children }: { children: ReactNode }) => (
    <button type="button">{children}</button>
  ),
}));

vi.mock("@/components/instance", () => ({
  InstanceActionDropdown: () => null,
  InstanceFormBody: () => null,
  InstanceFormButtons: () => null,
  InstanceFormProvider: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  InstanceRoleTable: () => null,
  InstanceSyncButton: () => null,
  useInstanceFormContext: () => ({ isDirty: false }),
}));

vi.mock("@/components/database", () => ({
  CreateDatabaseSheet: () => null,
  DatabaseBatchOperationsBar: () => null,
  DatabaseTable: ({
    onDatabasesChange,
    selectionColumnIntroTarget,
  }: {
    onDatabasesChange?: (databases: Database[]) => void;
    selectionColumnIntroTarget?: string;
  }) => {
    useEffect(() => {
      onDatabasesChange?.(mocks.databases);
    }, [onDatabasesChange]);
    return (
      <div
        data-selection-column-intro-target={selectionColumnIntroTarget ?? ""}
      />
    );
  },
  LabelEditorSheet: () => null,
  TransferProjectSheet: ({
    open,
    onTransfer,
  }: {
    open: boolean;
    onTransfer: (projectName: string) => Promise<void>;
  }) =>
    open ? (
      <button
        type="button"
        data-testid="transfer-project-sheet"
        onClick={() => {
          void onTransfer("projects/app");
        }}
      >
        transfer
      </button>
    ) : null,
}));

vi.mock("@/lib/productIntro", () => ({
  PREPARE_DATABASE_PRODUCT_INTRO: "prepare-database",
  PREPARE_DATABASE_TRANSFER_TIP: "transfer-databases-to-project",
  PRODUCT_INTRO_TIP_QUERY_KEY: "tip",
  useProductIntro: vi.fn(),
}));

import { InstanceDetailPage } from "./InstanceDetailPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

const render = async (element: ReactElement) => {
  await act(async () => {
    root.render(element);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.location = {
    pathname: "/instances/prod",
    search: "?syncingInstance=prod",
    hash: "#databases",
  };
  mocks.databases = [
    {
      name: "instances/prod/databases/app",
      project: "projects/default",
    },
  ] as Database[];
  mocks.projects = [{ name: "projects/default" }] as Project[];
  mocks.fetchProjectList.mockImplementation(async () => ({
    projects: mocks.projects,
    nextPageToken: "",
  }));
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

describe("InstanceDetailPage", () => {
  it("does not show the post-sync transfer action when there is no user project", async () => {
    await render(<InstanceDetailPage instanceId="prod" />);

    expect(container.textContent).not.toContain(
      "db.instance-databases-synced-title"
    );
    expect(mocks.fetchProjectList).toHaveBeenCalled();
  });

  it("shows the post-sync transfer action when there is a user project", async () => {
    mocks.projects = [
      { name: "projects/default" },
      { name: "projects/app" },
    ] as Project[];

    await render(<InstanceDetailPage instanceId="prod" />);

    expect(container.textContent).toContain(
      "db.instance-databases-synced-title"
    );
  });

  it("highlights the database selection column for the post-sync transfer guide", async () => {
    mocks.location.search =
      "?syncingInstance=prod&intro=prepare-database&tip=transfer-databases-to-project";
    mocks.projects = [
      { name: "projects/default" },
      { name: "projects/app" },
    ] as Project[];

    await render(<InstanceDetailPage instanceId="prod" />);

    expect(
      container
        .querySelector("[data-selection-column-intro-target]")
        ?.getAttribute("data-selection-column-intro-target")
    ).toBe("prepare-database");
  });

  it("redirects to the target project databases page after transferring from the instance database list", async () => {
    mocks.projects = [
      { name: "projects/default" },
      { name: "projects/app" },
    ] as Project[];

    await render(<InstanceDetailPage instanceId="prod" />);

    const transferSyncedButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) =>
      button.textContent?.includes("db.instance-databases-synced-action")
    ) as HTMLButtonElement;
    await act(async () => {
      transferSyncedButton.click();
      await Promise.resolve();
    });

    const transferSheet = container.querySelector(
      '[data-testid="transfer-project-sheet"]'
    ) as HTMLButtonElement;
    await act(async () => {
      transferSheet.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: {
        projectId: "app",
      },
    });
  });
});
