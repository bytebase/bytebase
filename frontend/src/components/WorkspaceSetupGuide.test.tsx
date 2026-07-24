import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  isSaaSMode: vi.fn(() => true),
  projectsByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
  databasesByName: {} as Record<string, unknown>,
  fetchProjectList: vi.fn(async () => ({
    projects: [] as unknown[],
    nextPageToken: "",
  })),
  fetchInstanceList: vi.fn(async () => ({
    instances: [] as unknown[],
    nextPageToken: "",
  })),
  fetchDatabases: vi.fn(async (_?: { parent?: string }) => ({
    databases: [] as unknown[],
    nextPageToken: "",
  })),
  introState: {} as Record<string, boolean>,
  getIntroStateByKey: vi.fn((_key: string) => false),
  saveIntroStateByKey: vi.fn(
    (_: { key: string; newState: boolean }) => undefined
  ),
  introStateVersion: 0,
  searchQueryHistories: vi.fn(async () => ({
    queryHistories: [] as unknown[],
    nextPageToken: "",
  })),
  workspacePolicy: {
    bindings: [{ role: "roles/workspaceAdmin", members: ["user@example.com"] }],
  },
  hasWorkspacePermissionV2: vi.fn(() => true),
  preCreateIssue: vi.fn(),
  currentRoute: { name: "workspace.home" } as { name?: string },
  routerPush: vi.fn(),
  captureMetric: vi.fn(),
  defaultProjectName: "projects/default",
}));

const dismissedIntroStateKey = "workspace-setup-guide.dismissed";

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
    workspacePolicy: mocks.workspacePolicy,
    serverInfo: { defaultProject: mocks.defaultProjectName },
    introStateVersion: mocks.introStateVersion,
    getIntroStateByKey: mocks.getIntroStateByKey,
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
  sqlServiceClientConnect: {
    searchQueryHistories: mocks.searchQueryHistories,
  },
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const match = name.match(/^instances\/([^/]+)\/databases\/([^/]+)$/);
    return {
      instanceName: match?.[1] ?? "",
      databaseName: match?.[2] ?? "",
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
      container.querySelector("[data-testid='setup-step-hasProjectDatabase']")
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

  it("hides when the workspace has more than one member", async () => {
    mocks.workspacePolicy = {
      bindings: [
        {
          role: "roles/workspaceAdmin",
          members: ["user@example.com", "teammate@example.com"],
        },
      ],
    };

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toBe("");
    expect(mocks.fetchProjectList).not.toHaveBeenCalled();
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
      "[data-testid='setup-step-hasProjectDatabase']"
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

  it("advances to the project database connection step after a project exists", async () => {
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
      name: "workspace.project.database",
      params: { projectId: "project-a" },
      query: { intro: "connect-database" },
    });
    expect(
      container.querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.tagName
    ).toBe("BUTTON");
  });

  it("disables the prepare database step until a project and instance exist", async () => {
    await render(<WorkspaceSetupGuide />);

    expect(
      container.querySelector(
        "[data-testid='setup-step-hasProjectDatabase']"
      )
    ).toBeDisabled();

    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container.querySelector(
        "[data-testid='setup-step-hasProjectDatabase']"
      )
    ).toBeDisabled();
  });

  it("routes prepare database with transfer context for the database page to resolve after loading", async () => {
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
      "[data-testid='setup-step-hasProjectDatabase']"
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

  it("routes prepare database with a transfer tip when workspace databases exist outside the project", async () => {
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
        parent === "-"
          ? [{ name: "instances/instance-a/databases/db-a" }]
          : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    const prepareStep = container.querySelector(
      "[data-testid='setup-step-hasProjectDatabase']"
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
      databases: [{ name: "instances/instance-a/databases/db-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.self"
    );

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("does not use the default project as the setup project", async () => {
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
    mocks.fetchDatabases.mockImplementation(async ({ parent } = {}) => ({
      databases:
        parent === "projects/project-a"
          ? [{ name: "instances/instance-a/databases/db-a" }]
          : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    expect(mocks.fetchDatabases).toHaveBeenCalledWith({
      parent: "projects/project-a",
      pageSize: 1,
      silent: true,
    });

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

  it("highlights the setup step matching the current route", async () => {
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
    ).toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasProjectDatabase']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
  });

  it("does not highlight the create project step on project database pages", async () => {
    mocks.currentRoute = { name: "workspace.project.database" };
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
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
        .querySelector("[data-testid='setup-step-hasInstance']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");

    expect(container.querySelector("[data-testid='active-action']")).toBeNull();
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
      "hasProjectDatabase",
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
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [{ name: "instances/instance-a/databases/db-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.steps.query"
    );
    expect(mocks.searchQueryHistories).toHaveBeenCalledWith(
      expect.objectContaining({
        pageSize: 1,
        filter: 'type == "QUERY"',
      })
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

    const secondaryAction = container.querySelector(
      "[data-testid='secondary-action']"
    );
    expect(secondaryAction).not.toBeNull();
    expect(
      secondaryAction!.compareDocumentPosition(actionLink as Node) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it("activates the query step when users click it after visiting another step", async () => {
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
      databases: [{ name: "instances/instance-a/databases/db-a" }],
      nextPageToken: "",
    });

    await render(<WorkspaceSetupGuide />);

    expect(
      container
        .querySelector("[data-testid='setup-step-hasProjectDatabase']")
        ?.getAttribute("class")
    ).toContain("bg-accent/10");
    expect(
      container
        .querySelector("[data-testid='setup-step-hasFirstQuery']")
        ?.getAttribute("class")
    ).not.toContain("bg-accent/10");

    const queryStep = container.querySelector(
      "[data-testid='setup-step-hasFirstQuery']"
    );

    expect(queryStep?.tagName).toBe("BUTTON");
    expect(queryStep?.getAttribute("class")).not.toContain("bg-accent/10");

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
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [{ name: "instances/instance-a/databases/db-a" }],
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

    await act(async () => {
      changeButton?.click();
      await Promise.resolve();
    });

    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/project-a", [
      "instances/instance-a/databases/db-a",
    ]);
  });

  it("stays visible after the first query exists", async () => {
    mocks.fetchProjectList.mockResolvedValue({
      projects: [{ name: "projects/project-a" }],
      nextPageToken: "",
    });
    mocks.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/instance-a" }],
      nextPageToken: "",
    });
    mocks.fetchDatabases.mockResolvedValue({
      databases: [{ name: "instances/instance-a/databases/db-a" }],
      nextPageToken: "",
    });
    mocks.searchQueryHistories.mockResolvedValue({
      queryHistories: [{ name: "projects/project-a/queryHistories/1" }],
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

  it("checks first query before a project database exists", async () => {
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

    expect(mocks.searchQueryHistories).toHaveBeenCalledWith(
      expect.objectContaining({
        pageSize: 1,
        filter: 'type == "QUERY"',
      })
    );
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
    mocks.fetchDatabases.mockImplementation(async ({ parent } = {}) => ({
      databases:
        parent === "projects/project-a"
          ? [{ name: "instances/instance-a/databases/db-a" }]
          : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.actions.change"
    );
  });

  it("refreshes when the route changes after setup progress changes elsewhere", async () => {
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
    mocks.fetchDatabases.mockImplementation(async ({ parent } = {}) => ({
      databases:
        parent === "projects/project-a"
          ? [{ name: "instances/instance-a/databases/db-a" }]
          : [],
      nextPageToken: "",
    }));

    await render(<WorkspaceSetupGuide />);

    expect(container.textContent).toContain(
      "workspace-setup-guide.actions.change"
    );
  });
});
