import type { ReactNode } from "react";
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types/iam";
import { ApprovalStatus, Engine, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { IssueSchema, Issue_Type } from "@/types/proto-es/v1/issue_service_pb";

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
    fetchInstanceList: vi.fn(),
    listIssues: vi.fn(),
    hasWorkspacePermission: vi.fn((_permission: Permission) => true),
    workspaceResourceName: vi.fn(() => "workspaces/demo"),
    projectPoliciesByName: {
      "projects/orders": { bindings: [{ role: "roles/projectOwner", members: ["user:alice@example.com"] }] },
      "projects/second": { bindings: [{ role: "roles/projectOwner", members: ["user:alice@example.com"] }] },
    },
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
  probedPermissions: [] as Permission[],
  open: vi.fn(),
  createChat: vi.fn(() => ({ id: "new-chat" })),
  queueUserMessage: vi.fn(),
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
  useComponentPermissionState: ({ permissions }: { permissions: Permission[] }) => {
    mocks.probedPermissions = permissions;
    return { missedPermissions: mocks.missedPermissions };
  },
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
  useAgentStore: Object.assign(
    (selector: (state: { chats: []; queuedUserMessage: null }) => unknown) =>
      selector({ chats: [], queuedUserMessage: null }),
    {
      getState: () => ({
        open: mocks.open,
        createChat: mocks.createChat,
        queueUserMessage: mocks.queueUserMessage,
        chats: [],
        queuedUserMessage: null,
      }),
    }
  ),
  selectHasRunningChat: () => false,
}));

import { ProjectHomePage } from "./ProjectHomePage";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.state.fetchDatabases.mockReset();
  localStorage.clear();
  mocks.missedPermissions = [];
  mocks.probedPermissions = [];
  mocks.state.serverInfo.aiEnabled = true;
  mocks.state.serverInfo.defaultProject = "projects/default";
  mocks.state.hasInstanceFeature.mockReturnValue(true);
  mocks.state.hasWorkspacePermission.mockReturnValue(true);
  mocks.state.fetchDatabases.mockResolvedValue({
    databases: [
      create(DatabaseSchema$, {
        name: "instances/primary/databases/default",
        project: "projects/orders",
      }),
    ],
  });
  mocks.state.fetchInstanceList.mockResolvedValue({ instances: [] });
  mocks.state.listIssues.mockResolvedValue({ issues: [] });
});

describe("ProjectHomePage", () => {
  test("starts a project-bound Agent chat from the composer", async () => {
    render(<ProjectHomePage projectId="orders" />);
    expect(screen.getByText("Order service")).toBeTruthy();
    await waitFor(() => {
      expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy();
    });
    const prompt = screen.getByRole("textbox", { name: "project.home.ai-placeholder" });
    expect(screen.getByRole("button", { name: "agent.send" }).hasAttribute("disabled")).toBe(true);
    fireEvent.change(prompt, { target: { value: "List databases" } });
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.createChat).toHaveBeenCalledWith(
      expect.objectContaining({ context: "projects/orders" })
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({ chatId: "new-chat", text: "List databases" })
    );
    expect(mocks.open).toHaveBeenCalledOnce();
    expect((prompt as HTMLTextAreaElement).value).toBe("");
  });

  test("keeps Enter for a new line and submits a multiline prompt with Send", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy());
    const prompt = screen.getByRole("textbox", { name: "project.home.ai-placeholder" });
    fireEvent.change(prompt, { target: { value: "Explain this project" } });

    const enterWasNotCanceled = fireEvent.keyDown(prompt, { key: "Enter" });
    expect(enterWasNotCanceled).toBe(true);
    expect(mocks.createChat).not.toHaveBeenCalled();

    fireEvent.change(prompt, { target: { value: "Explain this\nproject" } });
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.createChat).toHaveBeenCalledOnce();
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({ text: "Explain this\nproject" })
    );
  });

  test("does not probe SQL select permission for the Agent composer", async () => {
    mocks.missedPermissions = ["bb.sql.select"];
    render(<ProjectHomePage projectId="orders" />);

    expect(mocks.probedPermissions).not.toContain("bb.sql.select");
    expect(screen.getByRole("textbox", { name: "project.home.ai-placeholder" })).toBeTruthy();
    await waitFor(() => expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy());
  });

  test("opens instance creation from the setup step without duplicating the empty state action", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => {
      expect(screen.getAllByText("project.home.connect").length).toBeGreaterThan(0);
    });
    const action = within(screen.getByRole("region", { name: "project.home.next-step" }))
      .getByRole("link", { name: "project.home.connect" });
    expect(action.getAttribute("href")).toBe(
      "workspace.project.instance.create"
    );
    expect(within(screen.getByRole("region", { name: "project.home.databases" }))
      .queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("suggests managing members for a new project only with write permission", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(within(next).getByRole("link", { name: "project.home.members" })).toBeTruthy();
    expect(within(next).getByRole("link", { name: "project.home.members" }).getAttribute("href"))
      .toBe("workspace.project.members");

    mocks.missedPermissions = ["bb.projects.setIamPolicy"];
    render(<ProjectHomePage projectId="second" />);
    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(screen.getAllByRole("link", { name: "project.home.members" })).toHaveLength(1);
  });

  test("suggests creating a database when an instance exists but no database does", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "projects/orders/instances/primary" }],
    });
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    const createLink = within(next).getByRole("link", {
      name: "project.home.create-database",
    });
    expect(JSON.parse(createLink.getAttribute("data-to")!)).toEqual({
      name: "workspace.project.database",
      params: { projectId: "orders" },
      query: { createDatabase: "1" },
    });
    expect(within(screen.getByRole("region", { name: "project.home.databases" }))
      .queryByRole("link", { name: "project.home.create-database" })).toBeNull();
    expect(within(next).queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("suggests creating when a workspace instance exists but has no database to transfer", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockImplementation(async ({ parent }) => ({
      instances: parent ? [] : [{ name: "instances/shared" }],
    }));
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(await within(next).findByRole("link", { name: "project.home.create-database" })).toBeTruthy();
    expect(within(next).queryByRole("link", { name: "project.home.connect" })).toBeNull();
    expect(mocks.state.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ parent: "projects/orders" })
    );
    expect(mocks.state.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ parent: undefined })
    );
  });

  test("offers transfer first and creation second when unassigned databases exist", async () => {
    mocks.state.fetchDatabases.mockImplementation(async ({ parent }) => ({
      databases: parent === "workspaces/demo"
        ? [create(DatabaseSchema$, {
            name: "instances/shared/databases/existing",
            project: "projects/default",
          })]
        : [],
    }));
    mocks.state.fetchInstanceList.mockImplementation(async ({ parent }) => ({
      instances: parent ? [] : [{ name: "instances/shared" }],
    }));
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    const transfer = await within(next).findByRole("link", {
      name: "project.home.transfer-databases",
    });
    const createLink = within(next).getByRole("link", {
      name: "project.home.create-database",
    });
    expect(JSON.parse(transfer.getAttribute("data-to")!)).toEqual({
      name: "workspace.database",
      query: {
        intro: "prepare-database",
        tip: "transfer-databases-to-project",
        targetProject: "projects/orders",
      },
    });
    expect(transfer.classList.contains("bg-accent")).toBe(true);
    expect(createLink.classList.contains("bg-accent")).toBe(false);
    expect(mocks.state.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "workspaces/demo",
        filter: { project: "projects/default" },
      })
    );
  });

  test("offers transfer without permission to list instances", async () => {
    mocks.missedPermissions = ["bb.instances.list"];
    mocks.state.hasWorkspacePermission.mockImplementation(
      (permission) => permission !== "bb.instances.list"
    );
    mocks.state.fetchDatabases.mockImplementation(async ({ parent }) => ({
      databases: parent === "workspaces/demo"
        ? [create(DatabaseSchema$, {
            name: "instances/shared/databases/existing",
            project: "projects/default",
          })]
        : [],
    }));

    render(<ProjectHomePage projectId="orders" />);

    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(within(next).getByRole("link", {
      name: "project.home.transfer-databases",
    })).toBeTruthy();
    expect(mocks.state.fetchInstanceList).not.toHaveBeenCalled();
    expect(mocks.state.fetchDatabases).toHaveBeenCalledWith(
      expect.objectContaining({ parent: "workspaces/demo" })
    );
  });

  test("offers database creation for an instance before its first sync without polling", async () => {
    const setTimeoutSpy = vi.spyOn(window, "setTimeout");
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/primary" }],
    });
    const view = render(<ProjectHomePage projectId="orders" />);
    try {
      const next = await screen.findByRole("region", { name: "project.home.next-step" });
      const createLink = await within(next).findByRole("link", { name: "project.home.create-database" });
      expect(JSON.parse(createLink.getAttribute("data-to")!)).toEqual({
        name: "workspace.project.database",
        params: { projectId: "orders" },
        query: { createDatabase: "1" },
      });
      expect(screen.getByText("project.home.no-databases-create-description")).toBeTruthy();
      expect(setTimeoutSpy.mock.calls.some(([, delay]) => delay === 5000)).toBe(false);
      expect(mocks.state.fetchDatabases.mock.calls.filter(
        ([request]) => request.parent === "projects/orders"
      )).toHaveLength(1);
    } finally {
      view.unmount();
      setTimeoutSpy.mockRestore();
    }
  });

  test("retries a failed database preview without leaving Project Home", async () => {
    mocks.state.fetchDatabases.mockRejectedValueOnce(new Error("unavailable"));
    mocks.state.fetchDatabases.mockResolvedValueOnce({
      databases: [create(DatabaseSchema$, {
        name: "instances/primary/databases/orders",
        project: "projects/orders",
      })],
    });
    render(<ProjectHomePage projectId="orders" />);

    const databases = screen.getByRole("region", { name: "project.home.databases" });
    await within(databases).findByText("project.home.databases-load-error");
    fireEvent.click(within(databases).getByRole("button", { name: "common.refresh" }));

    await within(databases).findByText("orders");
    expect(mocks.state.fetchDatabases).toHaveBeenCalledTimes(2);
  });

  test("does not suggest transfer without workspace update permission", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockResolvedValue({
      instances: [{ name: "instances/shared" }],
    });
    mocks.state.hasWorkspacePermission.mockImplementation(
      (permission) => permission !== "bb.databases.update"
    );
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(await within(next).findByRole("link", { name: "project.home.create-database" })).toBeTruthy();
    expect(within(next).queryByRole("link", { name: "project.home.transfer-databases" })).toBeNull();
    expect(mocks.state.fetchDatabases).not.toHaveBeenCalledWith(
      expect.objectContaining({ parent: "workspaces/demo" })
    );
  });

  test("finds a synced instance on a later page", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockImplementation(async ({ parent, pageToken }) => {
      if (!parent) return { instances: [], nextPageToken: "" };
      if (!pageToken) {
        return {
          instances: [{ name: "instances/first" }],
          nextPageToken: "next",
        };
      }
      return {
        instances: [{ name: "instances/second" }],
        nextPageToken: "",
      };
    });
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(within(next).getByRole("link", { name: "project.home.create-database" })).toBeTruthy();
    expect(mocks.state.fetchInstanceList).toHaveBeenCalledWith(
      expect.objectContaining({ parent: "projects/orders", pageToken: "next" })
    );
  });

  test("does not claim a connection is needed when instance discovery fails", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    mocks.state.fetchInstanceList.mockRejectedValue(new Error("unavailable"));
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchInstanceList).toHaveBeenCalled());
    const next = screen.getByRole("region", { name: "project.home.next-step" });
    expect(within(next).getByRole("link", { name: "project.home.members" })).toBeTruthy();
    expect(screen.queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("does not suggest connecting when the member cannot create instances", async () => {
    mocks.missedPermissions = ["bb.instances.create"];
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(mocks.state.fetchInstanceList).toHaveBeenCalled());
    expect(screen.queryByRole("link", { name: "project.home.connect" })).toBeNull();
  });

  test("does not infer project instance state without project list permission", async () => {
    mocks.missedPermissions = ["bb.instances.list"];
    mocks.state.hasWorkspacePermission.mockReturnValue(false);
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);

    await waitFor(() => expect(mocks.state.fetchDatabases).toHaveBeenCalled());
    expect(mocks.state.fetchInstanceList).not.toHaveBeenCalled();
    expect(screen.queryByRole("link", { name: "project.home.connect" })).toBeNull();
    expect(screen.queryByRole("link", { name: "project.home.create-database" })).toBeNull();
    expect(screen.getByRole("link", { name: "project.home.members" })).toBeTruthy();
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
    expect(screen.queryByRole("textbox", { name: "project.home.ai-placeholder" })).toBeNull();
    await waitFor(() => expect(screen.getByText("default")).toBeTruthy());
  });

  test("uses the project page top inset and available width without Agent", async () => {
    mocks.state.serverInfo.aiEnabled = false;
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy());

    const title = screen.getByRole("heading", {
      level: 1,
      name: "Order service",
    });
    expect(title.closest("header")?.classList.contains("py-2")).toBe(false);

    const work = screen.getByRole("region", { name: "project.home.next-step" });
    expect(work.parentElement?.classList.contains("lg:max-w-5xl")).toBe(false);
  });

  test("places the Agent composer below the project title and resource", async () => {
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy());

    const title = screen.getByRole("heading", {
      level: 1,
      name: "Order service",
    });
    const header = title.closest("header");
    const resourceName = screen.getByText("projects/orders");
    const composer = screen.getByRole("textbox", { name: "project.home.ai-placeholder" });
    expect(title.parentElement).toBe(header);
    expect(resourceName.parentElement).toBe(header);
    expect(composer.closest("form")?.previousElementSibling).toBe(header);
    expect(header?.classList.contains("min-w-0")).toBe(true);
    expect(resourceName.classList.contains("break-all")).toBe(true);
    expect(screen.queryByRole("button", { name: "project.home.ask-ai" })).toBeNull();
  });

  test("does not reuse the previous project's database result during navigation", async () => {
    const view = render(<ProjectHomePage projectId="orders" />);
    await waitFor(() => expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy());

    mocks.state.fetchDatabases.mockImplementation(() => new Promise(() => {}));
    view.rerender(<ProjectHomePage projectId="second" />);
    expect(screen.getByText("Second project")).toBeTruthy();
    expect(screen.queryByRole("region", { name: "project.home.next-step" })).toBeNull();
    expect(screen.queryByText("project.home.connect")).toBeNull();
    await waitFor(() =>
      expect(screen.getByText("project.home.no-active-issues")).toBeTruthy()
    );
  });

  test("shows member-related issues and accessible databases below setup", async () => {
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
    expect(await screen.findByRole("region", { name: "project.home.next-step" })).toBeTruthy();
    expect(screen.getByRole("textbox", { name: "project.home.ai-placeholder" })).toBeTruthy();
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
    expect(mocks.state.listIssues).toHaveBeenCalledWith(
      expect.objectContaining({
        find: expect.objectContaining({ typeList: [Issue_Type.DATABASE_CHANGE] }),
        pageSize: 1,
      })
    );
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
    expect((await screen.findByRole("link", { name: "project.home.connect" })).getAttribute("href")).toContain("workspace.project.instance");
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

  test("replaces shortcuts with project setup steps", async () => {
    mocks.state.fetchDatabases.mockResolvedValue({ databases: [] });
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(within(next).getByRole("link", { name: "project.home.connect" }).getAttribute("href")).toBe(
      "workspace.project.instance.create"
    );
    expect(within(next).getByRole("link", { name: "project.home.members" })).toBeTruthy();
    expect(screen.queryByRole("region", { name: "project.home.shortcuts" })).toBeNull();
    expect(screen.queryByRole("button", { name: "project.home.customize" })).toBeNull();
    expect(
      within(screen.getByRole("region", { name: "project.home.databases" })).queryByRole("link", {
        name: "project.home.connect",
      })
    ).toBeNull();
  });

  test("shows database change after database creation, but hides setup after a submitted change", async () => {
    const accessIssue = create(IssueSchema, {
      name: "projects/orders/issues/access",
      title: "Access request",
      status: IssueStatus.OPEN,
    });
    mocks.state.listIssues.mockImplementation(async ({ find }) => ({
      issues: find.typeList ? [] : [accessIssue],
    }));
    const view = render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect((await within(next).findByRole("link", { name: "project.home.change-database" })).getAttribute("href"))
      .toBe("workspace.project.plan.detail");
    expect(mocks.state.listIssues).toHaveBeenCalledWith(
      expect.objectContaining({
        find: expect.objectContaining({ typeList: [Issue_Type.DATABASE_CHANGE] }),
        pageSize: 1,
      })
    );
    expect(screen.getByRole("region", { name: "project.home.next-step" })).toBeTruthy();
    view.unmount();

    mocks.state.listIssues.mockImplementation(async ({ find }) => ({
      issues: find.typeList
        ? [create(IssueSchema, { type: Issue_Type.DATABASE_CHANGE })]
        : [],
    }));
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() =>
      expect(mocks.state.listIssues).toHaveBeenCalledWith(
        expect.objectContaining({
          find: expect.objectContaining({ typeList: [Issue_Type.DATABASE_CHANGE] }),
          pageSize: 1,
        })
      )
    );
    await waitFor(() => expect(screen.queryByRole("region", { name: "project.home.next-step" })).toBeNull());
  });

  test("checks submitted database changes with one issue-list request", async () => {
    mocks.state.listIssues.mockImplementation(async ({ find }) => ({
      issues: find.typeList
        ? [create(IssueSchema, { type: Issue_Type.DATABASE_CHANGE })]
        : [],
    }));
    render(<ProjectHomePage projectId="orders" />);
    await waitFor(() =>
      expect(mocks.state.listIssues).toHaveBeenCalledWith(
        expect.objectContaining({
          find: expect.objectContaining({ typeList: [Issue_Type.DATABASE_CHANGE] }),
          pageSize: 1,
        })
      )
    );
    expect(
      mocks.state.listIssues.mock.calls.filter(([request]) => request.find.typeList)
    ).toHaveLength(1);
    expect(screen.queryByRole("region", { name: "project.home.next-step" })).toBeNull();
  });

  test("keeps setup visible when no submitted database-change issue exists", async () => {
    render(<ProjectHomePage projectId="orders" />);
    const next = await screen.findByRole("region", { name: "project.home.next-step" });
    expect(await within(next).findByRole("link", { name: "project.home.change-database" })).toBeTruthy();
    expect(mocks.state.listIssues).toHaveBeenCalledWith(
      expect.objectContaining({
        find: expect.objectContaining({ typeList: [Issue_Type.DATABASE_CHANGE] }),
        pageSize: 1,
      })
    );
  });
});
