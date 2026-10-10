import type { ReactNode } from "react";
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types/iam";
import { ApprovalStatus, Engine, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { storageKeyProjectHomeShortcuts } from "@/utils/storage-keys";

const mocks = vi.hoisted(() => ({
  state: {
    projectsByName: {
      "projects/orders": {
        name: "projects/orders",
        title: "Order service",
      },
      "projects/second": {
        name: "projects/second",
        title: "Second project",
      },
    },
    serverInfo: {
      defaultProject: "projects/default",
      aiEnabled: true,
      workspace: "workspaces/demo",
    },
    currentUser: { name: "users/alice", email: "alice@example.com", workspace: "workspaces/demo" },
    fetchDatabases: vi.fn(),
    listIssues: vi.fn(),
    listPlans: vi.fn(),
    environmentList: [{ name: "environments/prod", title: "Production" }],
    loadEnvironmentList: vi.fn(),
    getEnvironmentByName: vi.fn((name: string) =>
      name === "environments/prod"
        ? { name, title: "Production" }
        : undefined
    ),
    hasInstanceFeature: vi.fn(() => true),
    hasFeature: vi.fn(() => true),
    loadSubscription: vi.fn(),
  },
  missedPermissions: [] as Permission[],
  open: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: typeof mocks.state) => unknown) => selector(mocks.state),
    { getState: () => mocks.state }
  ),
}));
vi.mock("@/components/ComponentPermissionGuard", () => ({
  useComponentPermissionState: () => ({
    missedPermissions: mocks.missedPermissions,
  }),
}));
vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));
vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    to,
    children,
    ...props
  }: {
    to: { name: string; params?: Record<string, string> };
    children: ReactNode;
  }) => (
    <a href={to.name} data-to={JSON.stringify(to)} {...props}>
      {children}
    </a>
  ),
}));
vi.mock("@/modules/agent/store/agent", () => ({
  useAgentStore: { getState: () => ({ open: mocks.open }) },
}));

import { ProjectHomePage } from "./ProjectHomePage";

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
  mocks.missedPermissions = [];
  mocks.state.serverInfo.aiEnabled = true;
  mocks.state.serverInfo.defaultProject = "projects/default";
  mocks.state.hasInstanceFeature.mockReturnValue(true);
  mocks.state.fetchDatabases.mockResolvedValue({
    databases: [
      create(DatabaseSchema$, {
        name: "instances/primary/databases/default",
        project: "projects/orders",
      }),
    ],
  });
  mocks.state.listIssues.mockResolvedValue({ issues: [] });
  mocks.state.listPlans.mockResolvedValue({ plans: [] });
});

describe("ProjectHomePage", () => {
  test("shows reachable work and opens the current Agent window", async () => {
    render(<ProjectHomePage projectId="orders" />);
    expect(screen.getByText("Order service")).toBeTruthy();
    await waitFor(() => {
      expect(screen.getByText("project.home.prepare")).toBeTruthy();
    });
    expect(screen.getByText("project.home.query")).toBeTruthy();
    expect(screen.getByText("project.home.access")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "project.home.ask-ai" }));
    await waitFor(() => expect(mocks.open).toHaveBeenCalledOnce());
  });

  test("opens instance creation from both empty-project entry points", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getAllByText("project.home.connect").length).toBeGreaterThan(0);
    });
    const emptyAction = screen.getByRole("link", { name: "project.home.connect" });
    const shortcut = within(
      screen.getByRole("region", { name: "project.home.shortcuts" })
    ).getByRole("link", { name: /project.home.connect/ });
    expect(emptyAction.getAttribute("href")).toBe(
      "workspace.project.instance.create"
    );
    expect(shortcut.getAttribute("href")).toBe(
      "workspace.project.instance.create"
    );
    expect(emptyAction.classList.contains("py-1")).toBe(true);
    expect(emptyAction.classList.contains("outline-item")).toBe(false);
    expect(emptyAction.classList.contains("px-2")).toBe(false);
    expect(emptyAction.classList.contains("hover:underline")).toBe(true);
    expect(screen.queryByText("project.home.query")).toBeNull();
  });

  test("keeps browsing Instances on the list when the member cannot connect", async () => {
    mocks.missedPermissions = ["bb.instances.create"];
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    const shortcut = await screen.findByRole("link", {
      name: /project.home.browse-instances/,
    });
    expect(shortcut.getAttribute("href")).toBe("workspace.project.instance");
    expect(screen.queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("does not infer emptiness from an error or offer project instances for the default project", async () => {
    mocks.state.fetchDatabases.mockRejectedValueOnce(new Error("unavailable"));
    const view = render(<ProjectHomePage projectId="orders" />);
    await waitFor(() =>
      expect(screen.getByText("project.home.databases-load-error")).toBeTruthy()
    );
    expect(screen.queryByText("project.home.connect")).toBeNull();
    view.unmount();

    mocks.state.serverInfo.defaultProject = "projects/orders";
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(screen.queryByText("project.home.connect")).toBeNull();
  });

  test("hides the Agent region when AI is not enabled", async () => {
    mocks.state.serverInfo.aiEnabled = false;
    render(<ProjectHomePage projectId="orders" />);
    expect(screen.queryByRole("button", { name: "project.home.ask-ai" })).toBeNull();
    await waitFor(() => expect(screen.getByText("default")).toBeTruthy());
  });

  test("uses the project page top inset and available width without Agent", async () => {
    mocks.state.serverInfo.aiEnabled = false;
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getByText("project.home.query")).toBeTruthy();
    });

    const title = screen.getByRole("heading", {
      level: 1,
      name: "Order service",
    });
    expect(title.closest("header")?.classList.contains("py-2")).toBe(false);

    const work = screen.getByRole("region", { name: "project.home.shortcuts" });
    expect(work.parentElement?.classList.contains("lg:max-w-5xl")).toBe(false);
  });

  test("places the primary Ask AI button beside the title and the resource below", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getByText("project.home.query")).toBeTruthy();
    });

    const title = screen.getByRole("heading", {
      level: 1,
      name: "Order service",
    });
    const header = title.closest("header");
    const resourceName = screen.getByText("projects/orders");
    const askAI = screen.getByRole("button", { name: "project.home.ask-ai" });
    expect(title.parentElement).toBe(askAI.parentElement);
    expect(title.parentElement?.classList.contains("flex-wrap")).toBe(true);
    expect(title.parentElement?.parentElement).toBe(header);
    expect(resourceName.parentElement).toBe(header);
    expect(header?.classList.contains("min-w-0")).toBe(true);
    expect(askAI.classList.contains("bg-accent")).toBe(true);
    expect(askAI.classList.contains("h-7")).toBe(true);
    expect(resourceName.classList.contains("break-all")).toBe(true);
    expect(header?.querySelector("svg")).toBeNull();
  });

  test("lets a member customize and reset project work without changing another project", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.query")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "project.home.customize" }));
    expect(screen.getAllByText("project.home.query-description")).toHaveLength(2);
    expect(screen.getByText("project.home.instances-description")).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: /project.home.query/ }));
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    expect(JSON.parse(localStorage.getItem(key)!)).not.toContain("query");
    fireEvent.click(screen.getByRole("button", { name: "common.done" }));
    expect(screen.queryByRole("link", { name: /project.home.query/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "project.home.customize" }));
    fireEvent.click(screen.getByRole("button", { name: "project.home.reset" }));
    expect(localStorage.getItem(key)).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "common.done" }));
    expect(screen.getByRole("link", { name: /project.home.query/ })).toBeTruthy();
  });

  test("offers GitOps as an optional shortcut with its description", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.prepare")).toBeTruthy());
    expect(screen.queryByRole("link", { name: /gitops.self/ })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "project.home.customize" }));
    expect(screen.getByText("project.home.gitops-description")).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: /gitops.self/ }));
    fireEvent.click(screen.getByRole("button", { name: "common.done" }));
    expect(screen.getByRole("link", { name: /gitops.self/ }).getAttribute("href")).toBe(
      "workspace.project.gitops"
    );
  });

  test("preserves an explicit empty selection instead of regenerating suggestions", async () => {
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    localStorage.setItem(key, "[]");
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(screen.getByText("project.home.no-shortcuts")).toBeTruthy();
    expect(screen.queryByRole("link", { name: /project.home.query/ })).toBeNull();
  });

  test("temporarily filters saved choices without erasing them", async () => {
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    localStorage.setItem(key, '["query","issues"]');
    mocks.missedPermissions = ["bb.projects.getIamPolicy"];
    const view = render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.issues")).toBeTruthy());
    expect(screen.queryByText("project.home.query")).toBeNull();
    expect(localStorage.getItem(key)).toBe('["query","issues"]');

    mocks.missedPermissions = [];
    view.rerender(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.query")).toBeTruthy());
    expect(localStorage.getItem(key)).toBe('["query","issues"]');
  });

  test("keeps a saved Instances destination after a database is available", async () => {
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    localStorage.setItem(key, '["instances"]');
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getByRole("link", { name: /project.home.instances/ })).toBeTruthy();
    });
    expect(screen.queryByText("project.home.connect")).toBeNull();
    expect(localStorage.getItem(key)).toBe('["instances"]');
  });

  test("sends every available shortcut to its project-scoped destination", async () => {
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    localStorage.setItem(
      key,
      '["plans","query","access","issues","instances","databases","members","gitops"]'
    );
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.prepare")).toBeTruthy());
    const shortcuts = within(
      screen.getByRole("region", { name: "project.home.shortcuts" })
    );
    for (const [label, route, params] of [
      ["project.home.prepare", "workspace.project.plan", { projectId: "orders" }],
      ["project.home.query", "sql-editor.project", { project: "orders" }],
      ["project.home.access", "workspace.project.access-grants", { projectId: "orders" }],
      ["project.home.issues", "workspace.project.issue", { projectId: "orders" }],
      ["project.home.instances", "workspace.project.instance", { projectId: "orders" }],
      ["project.home.databases", "workspace.project.database", { projectId: "orders" }],
      ["project.home.members", "workspace.project.members", { projectId: "orders" }],
      ["gitops.self", "workspace.project.gitops", { projectId: "orders" }],
    ] as const) {
      const link = shortcuts.getByRole("link", { name: new RegExp(label) });
      expect(JSON.parse(link.getAttribute("data-to")!)).toEqual({
        name: route,
        params,
      });
    }
  });

  test("reorders selected shortcuts by dragging rows with workspace-style icons", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.query")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: "project.home.customize" }));
    const first = screen.getByRole("checkbox", { name: /project.home.prepare/ })
      .closest('[draggable="true"]');
    const second = screen.getByRole("checkbox", { name: /project.home.query/ })
      .closest('[draggable="true"]');
    expect(first).toBeTruthy();
    expect(second).toBeTruthy();
    expect(first?.querySelector("svg.lucide-workflow")).toBeTruthy();
    expect(first?.querySelector("svg.lucide-grip-vertical")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /project.home.move-(up|down)/ })).toBeNull();
    fireEvent.dragStart(first!);
    fireEvent.dragEnter(second!);
    fireEvent.dragEnd(first!);
    const key = storageKeyProjectHomeShortcuts(
      "workspaces/demo",
      "alice@example.com",
      "projects/orders"
    );
    expect(JSON.parse(localStorage.getItem(key)!)).toEqual([
      "query",
      "plans",
      "access",
    ]);
  });

  test("keeps an edit visible and reports when browser storage cannot save it", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.query")).toBeTruthy());
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("storage unavailable");
      });
    try {
      fireEvent.click(screen.getByRole("button", { name: "project.home.customize" }));
      fireEvent.click(screen.getByRole("checkbox", { name: /project.home.query/ }));
      expect(screen.getByRole("alert").textContent).toBe(
        "project.home.save-error"
      );
      fireEvent.click(screen.getByRole("button", { name: "common.done" }));
      expect(screen.queryByRole("link", { name: /project.home.query/ })).toBeNull();
    } finally {
      setItem.mockRestore();
    }
  });

  test("does not offer access grants without the JIT feature", async () => {
    mocks.state.hasInstanceFeature.mockReturnValue(false);
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(screen.queryByText("project.home.access")).toBeNull();
  });

  test("labels plans as review-only without create permission", async () => {
    mocks.missedPermissions = ["bb.plans.create"];
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getByText("project.home.review-plans")).toBeTruthy();
    });
    expect(screen.queryByText("project.home.prepare")).toBeNull();
  });

  test("does not reuse the previous project's database result during navigation", async () => {
    const view = render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.query")).toBeTruthy());

    mocks.state.fetchDatabases.mockImplementation(() => new Promise(() => {}));
    view.rerender(<ProjectHomePage projectId="second" />);
    expect(screen.getByText("Second project")).toBeTruthy();
    expect(screen.queryByText("project.home.query")).toBeNull();
    expect(screen.queryByText("project.home.connect")).toBeNull();
    await waitFor(() =>
      expect(screen.getByText("project.home.no-active-issues")).toBeTruthy()
    );
  });

  test("shows member-related issues and accessible databases above vertical shortcuts", async () => {
    const review = create(IssueSchema, {
      name: "projects/orders/issues/review",
      title: "Review migration",
      status: IssueStatus.OPEN,
      updateTime: create(TimestampSchema, { seconds: 1760000000n }),
    });
    mocks.state.listIssues.mockImplementation(({ find }) =>
      Promise.resolve({ issues: find.currentApprover ? [review] : [] })
    );
    mocks.state.fetchDatabases.mockResolvedValue({
      databases: [
        create(DatabaseSchema$, {
          name: "instances/primary/databases/orders",
          project: "projects/orders",
          effectiveEnvironment: "environments/prod",
          instanceResource: create(InstanceResourceSchema, { engine: Engine.POSTGRES, title: "Primary instance" }),
        }),
      ],
    });

    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("Review migration")).toBeTruthy());
    expect(screen.getByRole("heading", { name: "project.home.active-issues" })).toBeTruthy();
    const activeIssues = screen.getByRole("region", {
      name: "project.home.active-issues",
    });
    expect(activeIssues.querySelector("[data-slot='issue-status-icon']")).toBeTruthy();
    expect(within(activeIssues).queryByText("common.issue")).toBeNull();
    expect(screen.getByText("orders")).toBeTruthy();
    expect(screen.getByText("Production")).toBeTruthy();
    expect(screen.getByText("Primary instance")).toBeTruthy();
    expect(screen.getByRole("region", { name: "project.home.active-issues" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "project.home.databases" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "project.home.shortcuts" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "project.home.ask-ai" }).closest("header")).toBeTruthy();
    expect(mocks.state.listIssues).toHaveBeenCalledWith(
      expect.objectContaining({
        find: expect.objectContaining({
          currentApprover: "users/alice",
          approvalStatus: ApprovalStatus.PENDING,
        }),
      })
    );
    expect(mocks.state.listIssues).toHaveBeenCalledWith(
      expect.objectContaining({
        find: expect.objectContaining({
          creator: "users/alice",
          statusList: [IssueStatus.OPEN],
        }),
      })
    );
    expect(mocks.state.listPlans).not.toHaveBeenCalled();
  });

  test("shows distinct borderless destination links beside both preview headings", async () => {
    const issue = create(IssueSchema, {
      name: "projects/orders/issues/one",
      title: "Recent change",
      creator: "users/alice",
      status: IssueStatus.OPEN,
    });
    mocks.state.listIssues.mockImplementation(({ find }) =>
      Promise.resolve({ issues: find.creator ? [issue] : [] })
    );
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("Recent change")).toBeTruthy());
    const recent = screen.getByRole("region", { name: "project.home.active-issues" });
    const databases = screen.getByRole("region", { name: "project.home.databases" });
    const issuesLink = within(recent).getByRole("link", { name: "project.home.view-issues" });
    const databasesLink = within(databases).getByRole("link", { name: "project.home.view-all" });
    expect(issuesLink.getAttribute("href")).toBe("workspace.project.issue");
    expect(databasesLink.getAttribute("href")).toBe("workspace.project.database");
    expect(recent.firstElementChild?.contains(issuesLink)).toBe(true);
    expect(databases.firstElementChild?.contains(databasesLink)).toBe(true);
    expect(issuesLink.classList.contains("border")).toBe(false);
    expect(databasesLink.classList.contains("border")).toBe(false);
    expect(issuesLink.classList.contains("outline-item")).toBe(false);
    expect(databasesLink.classList.contains("outline-item")).toBe(false);
  });

  test("keeps completed issues and the empty Issues link off Home", async () => {
    const done = create(IssueSchema, {
      name: "projects/orders/issues/done",
      title: "Completed request",
      creator: "users/alice",
      status: IssueStatus.DONE,
    });
    mocks.state.listIssues.mockImplementation(({ find }) =>
      Promise.resolve({ issues: find.creator ? [done] : [] })
    );

    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.no-active-issues")).toBeTruthy());
    expect(screen.queryByText("Completed request")).toBeNull();
    expect(
      within(screen.getByRole("region", { name: "project.home.active-issues" }))
        .queryByRole("link", { name: "project.home.view-issues" })
    ).toBeNull();
  });

  test("hides the project Issues link without list permission", async () => {
    mocks.missedPermissions = ["bb.issues.list"];
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(
      within(screen.getByRole("region", { name: "project.home.active-issues" }))
        .queryByRole("link", { name: "project.home.view-issues" })
    ).toBeNull();
  });

  test("shows an honest database empty state and connects only when eligible", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("project.home.no-databases")).toBeTruthy());
    expect(screen.getByRole("link", { name: "project.home.connect" }).getAttribute("href")).toContain("workspace.project.instance");
    expect(
      within(screen.getByRole("region", { name: "project.home.databases" }))
        .queryByRole("link", { name: "project.home.view-all" })
    ).toBeNull();
  });

  test("explains the database prerequisite without a read-only connect action", async () => {
    mocks.missedPermissions = ["bb.instances.create"];
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() =>
      expect(screen.getByText("project.home.no-databases-prerequisite")).toBeTruthy()
    );
    expect(screen.queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("does not query protected previews without their permissions", async () => {
    mocks.missedPermissions = ["bb.databases.get", "bb.issues.get"];
    render(<ProjectHomePage projectId="orders" />);
    expect(mocks.state.fetchDatabases).not.toHaveBeenCalled();
    expect(mocks.state.listIssues).not.toHaveBeenCalled();
    expect(mocks.state.listPlans).not.toHaveBeenCalled();
    expect(screen.getByText("project.home.databases-unavailable")).toBeTruthy();
  });

  test("previews readable databases even when the full list route is unavailable", async () => {
    mocks.missedPermissions = ["bb.databases.list"];
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(screen.getByText("default")).toBeTruthy();
    expect(
      within(screen.getByRole("region", { name: "project.home.databases" })).queryByRole("link", { name: "project.home.view-all" })
    ).toBeNull();
  });

  test("does not show another project's recent work during navigation", async () => {
    const issue = create(IssueSchema, {
      name: "projects/orders/issues/42",
      title: "Orders change",
      creator: "users/alice",
    });
    mocks.state.listIssues.mockImplementation(({ find }) =>
      Promise.resolve({ issues: find.creator ? [issue] : [] })
    );
    const view = render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByText("Orders change")).toBeTruthy());

    mocks.state.listIssues.mockImplementation(() => new Promise(() => {}));
    view.rerender(<ProjectHomePage projectId="second" />);
    expect(screen.queryByText("Orders change")).toBeNull();
    await waitFor(() => expect(screen.getByText("default")).toBeTruthy());
  });
});
