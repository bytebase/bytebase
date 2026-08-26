import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import { sqlEditorEvents } from "@/modules/sql-editor/model/events";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  isSaaSMode: vi.fn(() => true),
  projectsByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
  databasesByName: {} as Record<string, unknown>,
  fetchProjectList: vi.fn(async (_?: { pageToken?: string }) => ({
    projects: [] as unknown[],
    nextPageToken: "",
  })),
  fetchInstanceList: vi.fn(
    async (_?: { parent?: string; pageToken?: string }) => ({
      instances: [] as unknown[],
      nextPageToken: "",
    })
  ),
  fetchDatabases: vi.fn(
    async (_?: { parent?: string; pageToken?: string }) => ({
      databases: [] as unknown[],
      nextPageToken: "",
    })
  ),
  introState: {} as Record<string, boolean>,
  getIntroStateByKey: vi.fn((_key: string) => false),
  saveIntroStateByKey: vi.fn(
    (_: { key: string; newState: boolean }) => undefined
  ),
  introStateVersion: 0,
  searchQueryHistories: vi.fn(async (_?: { pageToken?: string }) => ({
    queryHistories: [] as unknown[],
    nextPageToken: "",
  })),
  workspacePolicy: {
    bindings: [{ role: "roles/workspaceAdmin", members: ["user@example.com"] }],
  },
  hasWorkspacePermissionV2: vi.fn(() => true),
  preCreateIssue: vi.fn(),
  currentRoute: { name: "workspace.home" } as {
    name?: string;
    params?: Record<string, string | undefined>;
  },
  routerPush: vi.fn(),
  captureMetric: vi.fn(),
  defaultProjectName: "projects/default",
  sampleInstances: [] as { instance: string }[],
  userCountInIam: 1,
  hideQuickStart: false,
  workspaceName: "workspaces/default",
}));

const dismissedIntroStateKey = "workspace-setup-guide.dismissed";
const databaseExploredIntroStateKey =
  "workspace-setup-guide.database-explored";
const queryExecutedIntroStateKey = "workspace-setup-guide.query-executed";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    to,
    ...props
  }: {
    children: ReactNode;
    to: {
      name?: string;
      params?: Record<string, string>;
      query?: Record<string, string>;
      hash?: string;
    };
  }) => (
    <a
      data-route-name={to.name}
      data-route-params={JSON.stringify(to.params ?? {})}
      data-query={to.query?.intro}
      data-query-project={to.query?.project}
      data-hash={to.hash}
      {...props}
    >
      {children}
    </a>
  ),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: mocks.routerPush,
  },
  useCurrentRoute: () => mocks.currentRoute,
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
  SQL_EDITOR_HOME_MODULE: "sql-editor.home",
  SQL_EDITOR_PROJECT_MODULE: "sql-editor.project",
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    isSaaSMode: mocks.isSaaSMode,
    projectsByName: mocks.projectsByName,
    instancesByName: mocks.instancesByName,
    databasesByName: mocks.databasesByName,
    currentUserName: "users/user@example.com",
    currentUser: {
      name: "users/user@example.com",
      email: "user@example.com",
      workspace: "workspaces/default",
    },
    workspace: { name: "workspaces/default" },
    workspaceResourceName: () => mocks.workspaceName,
    workspacePolicy: mocks.workspacePolicy,
    hasWorkspacePermission: mocks.hasWorkspacePermissionV2,
    serverInfo: {
      defaultProject: mocks.defaultProjectName,
      sample: { instances: mocks.sampleInstances },
    },
    introStateVersion: mocks.introStateVersion,
    getIntroStateByKey: mocks.getIntroStateByKey,
    workspaceSetupGuideEnabled: () =>
      !mocks.hideQuickStart &&
      mocks.userCountInIam === 1 &&
      ["bb.projects.list", "bb.instances.list", "bb.databases.list"].every(
        mocks.hasWorkspacePermissionV2
      ),
  });
  const useAppStore = (selector?: (s: ReturnType<typeof getState>) => unknown) =>
    selector ? selector(getState()) : getState();
  useAppStore.getState = () => ({
    ...getState(),
    fetchProjectList: mocks.fetchProjectList,
    fetchInstanceList: mocks.fetchInstanceList,
    fetchDatabases: mocks.fetchDatabases,
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  });
  return { useAppStore };
});

vi.mock("@/api", () => ({
  queryHistoryServiceClientConnect: {
    searchQueryHistories: mocks.searchQueryHistories,
  },
}));

vi.mock("@/utils", () => ({
  autoSQLEditorDatabaseRoute: (database: {
    name: string;
    project: string;
  }) => {
    const match = database.name.match(
      /^(?:projects\/[^/]+\/)?instances\/(?<instance>[^/]+)\/databases\/(?<database>[^/]+)$/
    );
    return {
      name: "sql-editor.database",
      params: {
        project: database.project.replace(/^projects\//, ""),
        instance: match?.groups?.instance ?? "",
        database: match?.groups?.database ?? "",
      },
    };
  },
  extractDatabaseResourceName: (name: string) => {
    const match = name.match(
      /^(?<instance>(?:projects\/[^/]+\/)?instances\/(?<instanceName>[^/]+))\/databases\/(?<databaseName>[^/]+)$/
    );
    return {
      instance: match?.groups?.instance ?? "",
      instanceName: match?.groups?.instanceName ?? "",
      databaseName: match?.groups?.databaseName ?? "",
    };
  },
  extractInstanceResourceName: (name: string) =>
    name.replace(/^instances\//, ""),
  extractProjectResourceName: (name: string) =>
    name.replace(/^projects\//, ""),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/lib/plan/issue", () => ({
  preCreateIssue: mocks.preCreateIssue,
}));

import { WorkspaceSetupGuide } from "./WorkspaceSetupGuide";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.introState = {};
  mocks.introStateVersion = 0;
  mocks.getIntroStateByKey.mockImplementation(
    (key: string) => mocks.introState[key] ?? false
  );
  mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
    mocks.introState[key] = newState;
    mocks.introStateVersion += 1;
  });
  mocks.isSaaSMode.mockReturnValue(true);
  mocks.projectsByName = {};
  mocks.instancesByName = {};
  mocks.databasesByName = {};
  mocks.fetchProjectList.mockResolvedValue({
    projects: [],
    nextPageToken: "",
  });
  mocks.fetchInstanceList.mockResolvedValue({
    instances: [],
    nextPageToken: "",
  });
  mocks.fetchDatabases.mockResolvedValue({
    databases: [],
    nextPageToken: "",
  });
  mocks.searchQueryHistories.mockResolvedValue({
    queryHistories: [],
    nextPageToken: "",
  });
  mocks.workspacePolicy = {
    bindings: [{ role: "roles/workspaceAdmin", members: ["user@example.com"] }],
  };
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.currentRoute = { name: "workspace.home" };
  mocks.defaultProjectName = "projects/default";
  mocks.sampleInstances = [];
  mocks.userCountInIam = 1;
  mocks.hideQuickStart = false;
  mocks.workspaceName = "workspaces/default";
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  vi.useRealTimers();
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

const render = async (element: ReactElement) => {
  await act(async () => {
    root.render(element);
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
};

describe("WorkspaceSetupGuide", () => {
  it("renders outside SaaS mode", async () => {
    mocks.isSaaSMode.mockReturnValue(false);

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );
    expect(mocks.fetchProjectList).toHaveBeenCalled();
  });

  it("shows the first incomplete setup action for SaaS workspaces", async () => {
    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.project"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.instance"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.database"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.query"
    );
    expect(
      container.querySelector("[data-testid='setup-step-hasInstance']")?.tagName
    ).toBe("BUTTON");
    expect(
      container.querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.tagName
    ).toBe("BUTTON");
    expect(
      container.querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.tagName
    ).toBe("BUTTON");
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
    expect(
      container.textContent?.match(/workspace-setup-guide\.steps\.query/g)
        ?.length
    ).toBe(1);
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.project›workspace-setup-guide.steps.instance"
    );
    expect(container.textContent).not.toContain(
      "workspace-setup-guide.steps.project/workspace-setup-guide.steps.instance"
    );

    await act(async () => {
      (
        container.querySelector(
          "[data-testid='setup-step-hasInstance']"
        ) as HTMLButtonElement | null
      )?.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
      query: { intro: "create-instance" },
    });
  });

  it("uses caller-owned responsive button sizing", async () => {
    await render(<WorkspaceSetupGuide />);

    expect(container.firstElementChild?.getAttribute("class")).toContain(
      "py-2"
    );
    expect(container.firstElementChild?.getAttribute("class")).toContain(
      "2xl:py-4"
    );
    const projectStep = container.querySelector(
      "[data-testid='setup-step-hasProject']"
    );
    expect(projectStep?.getAttribute("class")).toContain("text-sm");
    expect(projectStep?.getAttribute("class")).toContain("2xl:text-base");
    expect(projectStep?.getAttribute("class")).toContain("py-1");
    expect(projectStep?.getAttribute("class")).toContain("2xl:py-2");
    expect(
      container
        .querySelector("[data-testid='dismiss-guide']")
        ?.getAttribute("class")
    ).toContain("h-7");
    expect(
      container
        .querySelector("[data-testid='dismiss-guide']")
        ?.getAttribute("class")
    ).toContain("2xl:h-9");
  });

  it("hides when the workspace has more than one member", async () => {
    mocks.userCountInIam = 2;

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
  });

  it.each([0, 2])("hides when IAM user count is %s", async (count) => {
    mocks.userCountInIam = count;

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
  });

  it("hides when the quick-start feature is disabled", async () => {
    mocks.hideQuickStart = true;

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
  });

  it("hides when the user cannot list instances", async () => {
    mocks.hasWorkspacePermissionV2.mockReturnValue(false);

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
    expect(mocks.fetchInstanceList).not.toHaveBeenCalled();
    expect(mocks.fetchDatabases).not.toHaveBeenCalled();
  });

  it("shows setup step descriptions in tooltips", async () => {
    vi.useFakeTimers();
    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).not.toContain(
      "workspace-setup-guide.descriptions.project"
    );

    const projectStep = container.querySelector(
      "[data-testid='setup-step-hasProject']"
    );

    await act(async () => {
      projectStep?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
      await Promise.resolve();
    });

    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("workspace-setup-guide.descriptions.project");
    expect(container.textContent).not.toContain(
      "workspace-setup-guide.descriptions.project"
    );
  });

  it("shows previous-step guidance in tooltips for disabled setup steps", async () => {
    vi.useFakeTimers();
    await render(<WorkspaceSetupGuide />);

    const prepareDatabaseStep = container.querySelector(
      "[data-testid='setup-step-hasExploredDatabase']"
    ) as HTMLButtonElement | null;

    expect(prepareDatabaseStep).toBeDisabled();

    await act(async () => {
      prepareDatabaseStep?.parentElement?.dispatchEvent(
        new FocusEvent("focusin", { bubbles: true })
      );
      vi.advanceTimersByTime(100);
      await Promise.resolve();
    });

    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("workspace-setup-guide.previous-step-required");
  });

  it("can be dismissed for the current workspace and user", async () => {
    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );

    const dismissButton = container.querySelector(
      "[data-testid='dismiss-guide']"
    ) as HTMLButtonElement | null;

    await act(async () => {
      dismissButton?.click();
      await Promise.resolve();
    });

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: dismissedIntroStateKey,
      newState: true,
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide dismissed",
      properties: {
        step: "hasProject",
      },
    });
    expect(mocks.introState[dismissedIntroStateKey]).toBe(true);

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
  });

  it("captures the selected setup guide action", async () => {
    await render(<WorkspaceSetupGuide />);

    const instanceStep = container.querySelector(
      "[data-testid='setup-step-hasInstance']"
    ) as HTMLButtonElement | null;

    await act(async () => {
      instanceStep?.click();
      await Promise.resolve();
    });

    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide action clicked",
      properties: {
        step: "hasInstance",
      },
    });
  });

  it("stays hidden after it is dismissed", async () => {
    mocks.introState[dismissedIntroStateKey] = true;

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
  });

  it("connects an instance from the selected project's instance page", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
    const instanceStep = container.querySelector(
      "[data-testid='setup-step-hasInstance']"
    );
    expect(instanceStep?.tagName).toBe("BUTTON");
    await act(async () => {
      (instanceStep as HTMLButtonElement | null)?.click();
      await Promise.resolve();
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.instance",
      params: { projectId: "project-a" },
      query: { intro: "create-instance" },
    });
    expect(
      container.querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.tagName
    ).toBe("BUTTON");
  });

  it("keeps the setup project when the first database belongs elsewhere", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/default",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    await act(async () => {
      (
        container.querySelector(
          "[data-testid='setup-step-hasInstance']"
        ) as HTMLButtonElement | null
      )?.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.instance",
      params: { projectId: "project-a" },
      query: { intro: "create-instance" },
    });
  });

  it("disables the prepare database step until a project and instance exist", async () => {
    await render(<WorkspaceSetupGuide />);

    expect(
      container.querySelector(
        "[data-testid='setup-step-hasExploredDatabase']"
      )
    ).toBeDisabled();

    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container.querySelector(
        "[data-testid='setup-step-hasExploredDatabase']"
      )
    ).toBeDisabled();
  });

  it("opens workspace databases with transfer guidance when the project has no databases", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    const prepareStep = container.querySelector(
      "[data-testid='setup-step-hasExploredDatabase']"
    ) as HTMLButtonElement | null;

    expect(prepareStep).not.toBeDisabled();

    await act(async () => {
      prepareStep?.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.database",
      query: {
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
      },
    });
  });

  it("opens project databases with query and change guidance when the project has a database", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    const prepareStep = container.querySelector(
      "[data-testid='setup-step-hasExploredDatabase']"
    ) as HTMLButtonElement | null;

    expect(prepareStep).not.toBeDisabled();

    await act(async () => {
      prepareStep?.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.project.database",
      params: { projectId: "project-a" },
      query: {
        intro: "project-instance-synced",
      },
    });
  });

  it("does not replay create guidance from completed steps", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    await act(async () => {
      (
        container.querySelector(
          "[data-testid='setup-step-hasProject']"
        ) as HTMLButtonElement | null
      )?.click();
      await Promise.resolve();
    });

    expect(mocks.routerPush).not.toHaveBeenCalled();
  });

  it("highlights the setup step matching the current route", async () => {
    mocks.currentRoute = { name: "workspace.instance" };

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasInstance']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("stays visible and asks for a project when an instance exists first", async () => {
    mocks.currentRoute = { name: "workspace.instance.detail" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/default" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/default",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("does not use the default project as the setup project", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [
        { name: "projects/default" },
        { name: "projects/project-a" },
      ],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(mocks.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "workspaces/default",
        pageSize: 1,
        filter: { project: "projects/project-a" },
      })
    );

    const actionButton = container.querySelector(
      "[data-testid='secondary-action']"
    ) as HTMLButtonElement | null;

    await act(async () => {
      actionButton?.click();
      await Promise.resolve();
    });

    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/project-a", [
      "instances/instance-a/databases/db-a",
    ]);
  });

  it("counts self-host sample resources as project and instance readiness", async () => {
    mocks.sampleInstances = [
      { instance: "instances/sample-one" },
      { instance: "instances/sample-two" },
    ];
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-sample" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [
        { name: "instances/sample-one" },
        { name: "instances/sample-two" },
      ],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample-one/databases/employee",
          project: "projects/project-sample",
        },
        {
          name: "instances/sample-two/databases/employee",
          project: "projects/project-sample",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    for (const key of ["hasProject", "hasInstance"]) {
      expect(
        container
          .querySelector(`[data-testid='setup-step-${key}']`)
          ?.querySelector(".lucide-circle-check-big")
      ).not.toBeNull();
    }
    for (const key of ["hasExploredDatabase", "hasFirstQuery"]) {
      expect(
        container
          .querySelector(`[data-testid='setup-step-${key}']`)
          ?.querySelector(".lucide-circle-check-big")
      ).toBeNull();
    }
  });

  it("counts SaaS sample resources as project and instance readiness", async () => {
    mocks.sampleInstances = [
      { instance: "projects/app/instances/saas-sample" },
    ];
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "projects/app/instances/saas-sample" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "projects/app/instances/saas-sample/databases/employee",
          project: "projects/app",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    for (const key of ["hasProject", "hasInstance"]) {
      expect(
        container
          .querySelector(`[data-testid='setup-step-${key}']`)
          ?.querySelector(".lucide-circle-check-big")
      ).not.toBeNull();
    }
    for (const key of ["hasExploredDatabase", "hasFirstQuery"]) {
      expect(
        container
          .querySelector(`[data-testid='setup-step-${key}']`)
          ?.querySelector(".lucide-circle-check-big")
      ).toBeNull();
    }
  });

  it("uses only the first resource page", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.sampleInstances = [{ instance: "instances/sample" }];
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-sample" }],
      nextPageToken: "project-page-2",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample" }],
      nextPageToken: "instance-page-2",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample/databases/employee",
          project: "projects/project-sample",
        },
      ],
      nextPageToken: "database-page-2",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.querySelector(".lucide-circle-check-big")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='active-action']")
    ).toHaveAttribute(
      "data-route-params",
      JSON.stringify({
        project: "project-sample",
        instance: "sample",
        database: "employee",
      })
    );
    expect(mocks.fetchProjectList).toHaveBeenCalledTimes(1);
    expect(mocks.fetchInstanceList).toHaveBeenCalledTimes(1);
    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(1);
    expect(mocks.fetchProjectList).toHaveBeenCalledWith(
      expect.objectContaining({ pageSize: 1 })
    );
    expect(mocks.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ pageSize: 1 })
    );
    expect(mocks.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "workspaces/default",
        pageSize: 1,
        filter: { project: "projects/project-sample" },
      })
    );
  });

  it.each([
    "fetchProjectList",
    "fetchInstanceList",
    "fetchDatabases",
  ] as const)("keeps the guide available when %s fails", async (key) => {
    if (key === "fetchDatabases") {
      mocks.fetchProjectList.mockResolvedValue({
        projects: [{ name: "projects/project-a" }],
        nextPageToken: "",
      });
    }
    mocks[key].mockRejectedValueOnce(new Error("permission denied"));

    await render(<WorkspaceSetupGuide />);

    expect(mocks[key]).toHaveBeenCalled();
    expect(container.textContent).toContain("workspace-setup-guide.self");
    expect(
      container.querySelector("[data-testid='setup-step-hasProject']")
    ).not.toBeNull();
  });

  it.each([
    {
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "project-sample",
        instanceId: "sample-one",
        databaseName: "employee",
      },
    },
    {
      name: INSTANCE_ROUTE_DATABASE_DETAIL,
      params: {
        instanceId: "sample-one",
        databaseName: "employee",
      },
    },
    {
      name: SQL_EDITOR_DATABASE_MODULE,
      params: {
        project: "project-sample",
        instance: "sample-one",
        database: "employee",
      },
    },
  ])("persists database exploration on $name", async (route) => {
    mocks.currentRoute = route;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-sample" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/sample-one" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/sample-one/databases/employee",
          project: "projects/project-sample",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: databaseExploredIntroStateKey,
      newState: true,
    });
  });

  it("does not count the workspace database list as exploration", async () => {
    mocks.currentRoute = { name: DATABASE_ROUTE_DASHBOARD };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: databaseExploredIntroStateKey,
      newState: true,
    });
    expect(
      container
        .querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.querySelector(".lucide-circle-check-big")
    ).toBeNull();
  });

  it("keeps database exploration complete after it is persisted", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.querySelector(".lucide-circle-check-big")
    ).not.toBeNull();
  });

  it("does not show the active guide action when already on its route", async () => {
    mocks.currentRoute = { name: "workspace.instance.create" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.instance"
    );
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
    expect(container.querySelector("[data-testid='dismiss-guide']")).not.toBeNull();
  });

  it("highlights the next incomplete step when the current route step is done", async () => {
    mocks.currentRoute = { name: "workspace.project" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasProject']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("highlights the active query step on project database pages", async () => {
    mocks.currentRoute = {
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "project-a",
        instanceId: "instance-a",
        databaseName: "db-a",
      },
    };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "projects/project-a/instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(mocks.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ pageSize: 1 })
    );
    expect(
      container
        .querySelector("[data-testid='setup-step-hasProject']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasInstance']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");
    expect(
      container.querySelector("[data-testid='active-action']")
    ).not.toBeNull();
  });

  it("does not highlight the next setup step on unrelated pages", async () => {
    mocks.currentRoute = { name: "workspace.home" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    for (const key of [
      "hasProject",
      "hasInstance",
      "hasExploredDatabase",
      "hasFirstQuery",
    ]) {
      expect(
        container
          .querySelector(`[data-testid='setup-step-${key}']`)
          ?.getAttribute("class")
      ).not.toContain("bg-accent/10");
    }

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("asks users to create a database when the connected instance has no databases", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockImplementation(async ({ parent } = {}) => ({
      databases: parent === "projects/project-a" ? [] : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.database"
    );
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("asks users to transfer a database when the connected instance has no project database", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockImplementation(async ({ parent } = {}) => ({
      databases:
        parent === "instances/instance-a"
          ? [{ name: "instances/instance-a/databases/db-a" }]
          : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.database"
    );
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("opens SQL Editor as the primary action after a database is connected", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.query"
    );
    const actionLink = container.querySelector(
      "[data-testid='active-action']"
    );
    expect(actionLink?.getAttribute("data-route-name")).toBe(
      "sql-editor.database"
    );
    expect(actionLink?.getAttribute("data-route-params")).toBe(
      JSON.stringify({
        project: "project-a",
        instance: "instance-a",
        database: "db-a",
      })
    );
    expect(actionLink?.textContent).toContain(
      "workspace-setup-guide.actions.query"
    );
    expect(actionLink?.querySelector(".lucide-square-terminal")).not.toBeNull();
    expect(actionLink).toHaveAttribute("target", "_blank");
    expect(actionLink).toHaveAttribute("rel", "noopener noreferrer");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.querySelector(".lucide-circle-check-big")
    ).toBeNull();

    act(() => {
      (actionLink as HTMLAnchorElement | null)?.click();
    });
    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: queryExecutedIntroStateKey,
      newState: true,
    });

    const secondaryAction = container.querySelector(
      "[data-testid='secondary-action']"
    );
    expect(secondaryAction).not.toBeNull();
    expect(secondaryAction?.getAttribute("class")).toContain(
      "2xl:inline-flex"
    );
    expect(
      secondaryAction!.compareDocumentPosition(actionLink as Node) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);

    act(() => {
      (secondaryAction as HTMLButtonElement).click();
    });
    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/project-a", [
      "instances/instance-a/databases/db-a",
    ]);
  });

  it("completes Query data only after SQL execution finishes", async () => {
    mocks.sampleInstances = [
      { instance: "projects/app/instances/saas-sample" },
    ];
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    const queryStep = container.querySelector(
      "[data-testid='setup-step-hasFirstQuery']"
    );
    expect(queryStep?.querySelector(".lucide-circle-check-big")).toBeNull();

    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "",
        project: "projects/project-a",
      });
    });

    expect(queryStep?.querySelector(".lucide-circle-check-big")).toBeNull();
    expect(mocks.saveIntroStateByKey).not.toHaveBeenCalledWith({
      key: queryExecutedIntroStateKey,
      newState: true,
    });

    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "projects/app/instances/saas-sample/databases/employee",
        project: "projects/project-sample",
      });
    });

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: databaseExploredIntroStateKey,
      newState: true,
    });
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: queryExecutedIntroStateKey,
      newState: true,
    });
    expect(
      queryStep?.querySelector(".lucide-circle-check-big")
    ).not.toBeNull();
  });

  it("keeps query progress and its event target while resource scans finish", async () => {
    type ProjectResponse = {
      projects: { name: string }[];
      nextPageToken: string;
    };
    let resolveFirstScan: ((value: ProjectResponse) => void) | undefined;
    let resolveSecondScan: ((value: ProjectResponse) => void) | undefined;
    mocks.fetchProjectList
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFirstScan = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveSecondScan = resolve;
          })
      );
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/discovered-instance" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/discovered-instance/databases/discovered-db",
          project: "projects/discovered-project",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);
    expect(mocks.fetchProjectList).toHaveBeenCalledTimes(1);

    await act(async () => {
      await sqlEditorEvents.emit("query-executed", {
        database: "instances/event-instance/databases/event-db",
        project: "projects/event-project",
      });
    });
    expect(mocks.fetchProjectList).toHaveBeenCalledTimes(2);

    const assertInteractionState = () => {
      for (const key of ["hasExploredDatabase", "hasFirstQuery"]) {
        expect(
          container
            .querySelector(`[data-testid='setup-step-${key}']`)
            ?.querySelector(".lucide-circle-check-big")
        ).not.toBeNull();
      }
      expect(
        container.querySelector("[data-testid='active-action']")
      ).toHaveAttribute(
        "data-route-params",
        JSON.stringify({
          project: "event-project",
          instance: "event-instance",
          database: "event-db",
        })
      );
    };

    await act(async () => {
      resolveFirstScan?.({
        projects: [{ name: "projects/discovered-project" }],
        nextPageToken: "",
      });
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
    assertInteractionState();

    await act(async () => {
      resolveSecondScan?.({
        projects: [{ name: "projects/discovered-project" }],
        nextPageToken: "",
      });
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
    assertInteractionState();
  });

  it("does not reconstruct query completion from query history", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/prod/databases/main",
          project: "projects/app",
        },
      ],
      nextPageToken: "",
    });
    mocks.searchQueryHistories.mockResolvedValue({
      queryHistories: [
        { database: "instances/prod/databases/main" },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.querySelector(".lucide-circle-check-big")
    ).toBeNull();
  });

  it("does not search query history for guide progress", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/prod/databases/main",
          project: "projects/app",
        },
      ],
      nextPageToken: "",
    });
    await render(<WorkspaceSetupGuide />);

    expect(mocks.searchQueryHistories).not.toHaveBeenCalled();
  });

  it("activates the query step when users click it after visiting another step", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.currentRoute = { name: "workspace.database" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasExploredDatabase']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");

    const queryStep = container.querySelector(
      "[data-testid='setup-step-hasFirstQuery']"
    );

    expect(queryStep?.tagName).toBe("BUTTON");
    expect(queryStep?.getAttribute("class")).toContain("bg-accent/10");

    await act(async () => {
      (queryStep as HTMLButtonElement | null)?.click();
      await Promise.resolve();
    });

    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");
    expect(
      container.querySelector("[data-testid='active-action']")?.textContent
    ).toContain("workspace-setup-guide.actions.query");
  });

  it("opens the first database change flow as a secondary action", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    const changeButton = container.querySelector(
      "[data-testid='secondary-action']"
    ) as HTMLButtonElement | null;
    expect(changeButton?.tagName).toBe("BUTTON");
    expect(changeButton?.textContent).toContain(
      "workspace-setup-guide.actions.change"
    );
    const secondaryAction = container.querySelector(
      "[data-testid='secondary-action']"
    );
    expect(secondaryAction?.getAttribute("class")).toContain("h-9");
    for (const testId of ["active-action", "dismiss-guide"]) {
      const action = container.querySelector(`[data-testid='${testId}']`);
      expect(action?.getAttribute("class")).toContain("h-7");
      expect(action?.getAttribute("class")).toContain("2xl:h-9");
    }

    await act(async () => {
      changeButton?.click();
      await Promise.resolve();
    });

    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/project-a", [
      "instances/instance-a/databases/db-a",
    ]);
  });

  it("stays visible after the first query exists", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.introState[queryExecutedIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });
    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.query"
    );
    expect(
      container.querySelector("[data-testid='active-action']")?.textContent
    ).toContain("workspace-setup-guide.actions.query");
  });

  it("skips first query check before a project database exists", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });
    await render(<WorkspaceSetupGuide />);

    expect(mocks.searchQueryHistories).not.toHaveBeenCalled();
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("keeps the query step disabled before a project database exists", async () => {
    mocks.currentRoute = { name: "workspace.database" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    const queryStep = container.querySelector(
      "[data-testid='setup-step-hasFirstQuery']"
    );

    expect(queryStep?.tagName).toBe("BUTTON");
    expect(queryStep).toBeDisabled();
    expect(queryStep?.getAttribute("class")).not.toContain("bg-accent/10");

    await act(async () => {
      (queryStep as HTMLButtonElement | null)?.click();
      await Promise.resolve();
    });

    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");
    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("refreshes when a new instance is added to the app store", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.instance"
    );

    mocks.instancesByName = { "instances/instance-a": {} };
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.database"
    );
  });

  it("keeps the guide visible while progress is refreshing", async () => {
    let resolveProjectList:
      | ((value: { projects: unknown[]; nextPageToken: string }) => void)
      | undefined;

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );

    mocks.currentRoute = { name: "workspace.instance.detail" };
    mocks.instancesByName = { "instances/instance-a": {} };
    mocks.fetchProjectList.mockReturnValue(
      new Promise((resolve) => {
        resolveProjectList = resolve;
      })
    );

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );
    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.project"
    );

    await act(async () => {
      resolveProjectList?.({ projects: [], nextPageToken: "" });
      await Promise.resolve();
    });
  });

  it("refreshes when a database is added to the app store", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();

    mocks.databasesByName = {
      "instances/instance-a/databases/db-a": {},
    };
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.actions.change"
    );
  });

  it("refreshes when the route changes after setup progress changes elsewhere", async () => {
    mocks.introState[databaseExploredIntroStateKey] = true;
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();

    mocks.currentRoute = { name: "workspace.member" };
    mocks.fetchDatabases.mockResolvedValue({
      databases: [
        {
          name: "instances/instance-a/databases/db-a",
          project: "projects/project-a",
        },
      ],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.actions.change"
    );
  });
});
