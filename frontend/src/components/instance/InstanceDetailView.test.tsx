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
  actionProps: undefined as Record<string, unknown> | undefined,
  syncProps: undefined as Record<string, unknown> | undefined,
  providerProps: undefined as Record<string, unknown> | undefined,
  batchProps: undefined as Record<string, unknown> | undefined,
}));

vi.mock("react-router", () => ({
  useLocation: () => mocks.location,
  useNavigate: () => mocks.navigate,
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
    instancesByName: { [mocks.instance.name]: mocks.instance },
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
  extractInstanceResourceName: (name: string) => name.split("/").pop() ?? "",
  extractProjectResourceName: (name: string) => name.split("/").pop() ?? "",
  getDefaultPagination: () => 10,
  hasWorkspacePermissionV2: () => true,
  instanceV1Name: (instance: Instance) => instance.title,
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
  InstanceActionDropdown: (props: Record<string, unknown>) => {
    mocks.actionProps = props;
    return null;
  },
  InstanceFormBody: () => null,
  InstanceFormButtons: () => null,
  InstanceFormProvider: ({
    children,
    ...props
  }: {
    children: ReactNode;
  }) => {
    mocks.providerProps = props;
    return <div>{children}</div>;
  },
  InstanceRoleTable: () => null,
  InstanceSyncButton: (props: Record<string, unknown>) => {
    mocks.syncProps = props;
    return null;
  },
  useInstanceFormContext: () => ({ isDirty: false }),
}));

vi.mock("@/components/database", () => ({
  CreateDatabaseSheet: () => <div data-testid="create-database-sheet" />,
  DatabaseBatchOperationsBar: (props: Record<string, unknown>) => {
    mocks.batchProps = props;
    return null;
  },
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
  TransferProjectSheet: () => <div data-testid="transfer-project-sheet" />,
}));

vi.mock("@/lib/productIntro", () => ({
  PREPARE_DATABASE_PRODUCT_INTRO: "prepare-database",
  PREPARE_DATABASE_TRANSFER_TIP: "transfer-databases-to-project",
  PRODUCT_INTRO_TIP_QUERY_KEY: "tip",
  useProductIntro: vi.fn(),
}));

import { InstanceDetailView } from "./InstanceDetailView";

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
  mocks.instance = {
    name: "instances/prod",
    title: "Prod",
    engine: 1,
    state: 1,
    environment: "environments/test",
    roles: [],
  } as unknown as Instance;
  mocks.actionProps = undefined;
  mocks.syncProps = undefined;
  mocks.providerProps = undefined;
  mocks.batchProps = undefined;
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

describe("InstanceDetailView", () => {
  it("does not show the post-sync transfer action when there is no user project", async () => {
    await render(<InstanceDetailView instanceName="instances/prod" />);

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

    await render(<InstanceDetailView instanceName="instances/prod" />);

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

    await render(<InstanceDetailView instanceName="instances/prod" />);

    expect(
      container
        .querySelector("[data-selection-column-intro-target]")
        ?.getAttribute("data-selection-column-intro-target")
    ).toBe("prepare-database");
  });

  it("uses project ownership and hides workspace-only database actions", async () => {
    const project = { name: "projects/app", title: "App" } as Project;
    mocks.instance = {
      ...mocks.instance,
      name: "projects/app/instances/prod",
    } as Instance;
    mocks.location.pathname = "/projects/app/instances/prod";

    await render(
      <InstanceDetailView
        instanceName="projects/app/instances/prod"
        project={project}
      />
    );

    expect(mocks.getOrFetchInstanceByName).toHaveBeenCalledWith(
      "projects/app/instances/prod"
    );
    expect(mocks.actionProps).toMatchObject({ project });
    expect(mocks.syncProps).toMatchObject({ project });
    expect(mocks.providerProps).toMatchObject({ project });
    expect(mocks.batchProps?.onTransferProject).toBeUndefined();
    expect(container.querySelector("[data-testid='transfer-project-sheet']"))
      .toBeNull();
    expect(container.querySelector("[data-testid='create-database-sheet']"))
      .toBeNull();
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
  });
});
