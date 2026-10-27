import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { DEBOUNCE_SEARCH_DELAY } from "@/types/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { type Issue, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";

const mocks = vi.hoisted(() => ({
  searchAgentProjects: vi.fn(),
  searchAgentDatabases: vi.fn(),
  searchAgentIssues: vi.fn(),
  onSelectProject: vi.fn(),
  onSelectDatabase: vi.fn(),
  onSelectIssue: vi.fn(),
  onOpenChange: vi.fn(),
  fetchProject: vi.fn(),
  loadProjectIamPolicy: vi.fn(),
  canListWorkspaceProjects: true,
  canListWorkspaceDatabases: true,
  canGetWorkspaceProjects: true,
  projectGetPermissions: {} as Record<string, boolean>,
  projectDatabaseListPermissions: {} as Record<string, boolean>,
  projectsByName: {} as Record<string, Project>,
  projectPoliciesByName: {} as Record<string, object>,
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
  searchAgentDatabases: mocks.searchAgentDatabases,
  searchAgentIssues: mocks.searchAgentIssues,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: {
    hasWorkspacePermission: (permission: string) => boolean;
    hasProjectPermission: (project: Project, permission: string) => boolean;
    projectsByName: Record<string, Project>;
    projectPoliciesByName: Record<string, object>;
    fetchProject: typeof mocks.fetchProject;
    loadProjectIamPolicy: typeof mocks.loadProjectIamPolicy;
    getUserByIdentifier: (identifier: string) => undefined;
  }) => unknown) =>
    selector({
      hasWorkspacePermission: (permission) =>
        (permission === "bb.projects.list" && mocks.canListWorkspaceProjects) ||
        (permission === "bb.databases.list" && mocks.canListWorkspaceDatabases) ||
        (permission === "bb.projects.get" && mocks.canGetWorkspaceProjects),
      hasProjectPermission: (project, permission) =>
        Boolean(mocks.projectPoliciesByName[project.name]) &&
        ((permission === "bb.databases.list" &&
          mocks.projectDatabaseListPermissions[project.name] === true) ||
          (permission === "bb.projects.get" &&
            mocks.projectGetPermissions[project.name] === true)),
      projectsByName: mocks.projectsByName,
      projectPoliciesByName: mocks.projectPoliciesByName,
      fetchProject: mocks.fetchProject,
      loadProjectIamPolicy: mocks.loadProjectIamPolicy,
      getUserByIdentifier: () => undefined,
    }),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "agent.resource-picker.add-context": "Add context",
        "agent.resource-picker.projects": "Projects",
        "agent.resource-picker.databases": "Databases",
        "agent.resource-picker.issues": "Issues",
        "agent.resource-picker.search-projects": "Search projects",
        "agent.resource-picker.search-databases": "Search databases",
        "agent.resource-picker.search-issues": "Search issues",
        "agent.resource-picker.no-projects": "No projects found",
        "agent.resource-picker.no-databases": "No databases found",
        "agent.resource-picker.no-issues": "No issues found",
        "agent.resource-picker.load-error": "Could not load projects",
        "agent.resource-picker.load-databases-error": "Could not load databases",
        "agent.resource-picker.load-issues-error": "Could not load issues",
        "agent.resource-picker.retry": "Retry",
        "agent.resource-picker.reference-limit":
          "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message.",
        "common.back": "Back",
        "common.close": "Close",
        "common.loading": "Loading",
        "common.load-more": "Load more",
      })[key] ?? key,
  }),
}));

import { AgentResourcePicker } from "./AgentResourcePicker";

const project = create(ProjectSchema, {
  name: "projects/orders",
  title: "Orders",
});
const secondProject = create(ProjectSchema, {
  name: "projects/payments",
  title: "Payments",
});
const database = create(DatabaseSchema$, {
  name: "instances/main/databases/orders",
  project: project.name,
});
const secondDatabase = create(DatabaseSchema$, {
  name: "instances/reporting/databases/orders",
  project: project.name,
});
const issue = create(IssueSchema, {
  name: "projects/orders/issues/123",
  title: "Add an orders index",
});
const secondIssue = create(IssueSchema, {
  name: "projects/orders/issues/124",
  title: "Review query access",
});
const otherProjectIssue = create(IssueSchema, {
  name: "projects/payments/issues/125",
  title: "Review payment changes",
});

function deferredPage() {
  let resolve!: (page: { items: Project[]; nextPageToken: string }) => void;
  const promise = new Promise<{ items: Project[]; nextPageToken: string }>(
    (complete) => { resolve = complete; }
  );
  return { promise, resolve };
}

function PickerHarness({
  workspaceName = "workspaces/default",
  projectName,
  selectedProjectName,
  hasSelectedReference = false,
}: {
  workspaceName?: string;
  projectName?: string;
  selectedProjectName?: string;
  hasSelectedReference?: boolean;
} = {}) {
  const anchorRef = useRef<HTMLTextAreaElement>(null);
  const [open, setOpen] = useState(true);
  return (
    <>
      <textarea ref={anchorRef} aria-label="Question" />
      <AgentResourcePicker
        open={open}
        onOpenChange={(nextOpen) => {
          mocks.onOpenChange(nextOpen);
          setOpen(nextOpen);
        }}
        anchorRef={anchorRef}
        workspaceName={workspaceName}
        projectName={projectName}
        selectedProjectName={selectedProjectName}
        hasSelectedReference={hasSelectedReference}
        onSelectProject={mocks.onSelectProject}
        onSelectDatabase={mocks.onSelectDatabase}
        onSelectIssue={mocks.onSelectIssue}
      />
    </>
  );
}

beforeEach(() => {
  mocks.searchAgentProjects.mockReset();
  mocks.searchAgentDatabases.mockReset();
  mocks.searchAgentIssues.mockReset();
  mocks.onSelectProject.mockReset();
  mocks.onSelectDatabase.mockReset();
  mocks.onSelectIssue.mockReset();
  mocks.onOpenChange.mockReset();
  mocks.fetchProject.mockReset();
  mocks.loadProjectIamPolicy.mockReset();
  mocks.canListWorkspaceProjects = true;
  mocks.canListWorkspaceDatabases = true;
  mocks.canGetWorkspaceProjects = true;
  mocks.projectGetPermissions = {
    [project.name]: true,
    [secondProject.name]: true,
  };
  mocks.projectDatabaseListPermissions = {
    [project.name]: true,
    [secondProject.name]: true,
  };
  mocks.projectsByName = {
    [project.name]: project,
    [secondProject.name]: secondProject,
  };
  mocks.projectPoliciesByName = {
    [project.name]: {},
    [secondProject.name]: {},
  };
  mocks.fetchProject.mockImplementation(async (name: string) =>
    mocks.projectsByName[name]
  );
  mocks.loadProjectIamPolicy.mockImplementation(async (name: string) =>
    mocks.projectPoliciesByName[name]
  );
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AgentResourcePicker", () => {
  test("moves the active category to the hovered option", async () => {
    render(<PickerHarness />);

    const projects = screen.getByRole("button", { name: "Projects" });
    const databases = screen.getByRole("button", { name: "Databases" });
    const issues = screen.getByRole("button", { name: "Issues" });
    await waitFor(() => expect(projects).toHaveFocus());

    fireEvent.mouseEnter(issues);
    expect(issues).toHaveFocus();
    expect(issues).toHaveClass("focus:ring-2");
    fireEvent.keyDown(issues, { key: "ArrowUp" });
    expect(databases).toHaveFocus();

    fireEvent.mouseEnter(projects);
    expect(projects).toHaveFocus();
  });

  test("moves the active result to the hovered option", async () => {
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project, secondProject],
      nextPageToken: "",
    });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    const first = await screen.findByRole("button", {
      name: "Orders projects/orders",
    });
    const second = screen.getByRole("button", {
      name: "Payments projects/payments",
    });
    fireEvent.keyDown(screen.getByRole("textbox", { name: "Search projects" }), {
      key: "ArrowDown",
    });
    expect(first).toHaveFocus();

    fireEvent.mouseEnter(second);
    expect(second).toHaveFocus();
    expect(second).toHaveClass("focus:ring-2");
    expect(second).toHaveClass("focus:ring-offset-2");
    fireEvent.keyDown(second, { key: "Enter" });
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(secondProject);
  });

  test("hides Projects without workspace list permission", async () => {
    mocks.canListWorkspaceProjects = false;
    render(<PickerHarness />);

    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    const databaseCategory = screen.getByRole("button", { name: "Databases" });
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    await waitFor(() => expect(databaseCategory).toHaveFocus());
    fireEvent.keyDown(databaseCategory, { key: "ArrowDown" });
    expect(screen.getByRole("button", { name: "Issues" })).toHaveFocus();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
  });

  test("leaves project search when list permission is revoked", async () => {
    const oldPage = deferredPage();
    mocks.searchAgentProjects.mockReturnValueOnce(oldPage.promise);
    const { rerender } = render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() => expect(mocks.searchAgentProjects).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentProjects.mock.calls[0][2] as AbortSignal;

    mocks.canListWorkspaceProjects = false;
    rerender(<PickerHarness />);
    expect(oldSignal.aborted).toBe(true);
    expect(screen.queryByRole("textbox", { name: "Search projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();

    await act(async () => {
      oldPage.resolve({ items: [project], nextPageToken: "" });
      await oldPage.promise;
    });
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
  });

  test("hides workspace Databases without list permission instead of offering a project chooser", async () => {
    mocks.canListWorkspaceDatabases = false;
    render(<PickerHarness />);

    const projectCategory = screen.getByRole("button", { name: "Projects" });
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    fireEvent.keyDown(projectCategory, { key: "ArrowDown" });
    expect(screen.getByRole("button", { name: "Issues" })).toHaveFocus();
    expect(screen.queryByText("Choose a project to browse databases")).toBeNull();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
  });

  test("keeps Issues available when project and database listing are denied", async () => {
    mocks.canListWorkspaceProjects = false;
    mocks.canListWorkspaceDatabases = false;
    mocks.searchAgentIssues.mockResolvedValueOnce({
      items: [issue],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    const issueCategory = screen.getByRole("button", { name: "Issues" });
    await waitFor(() => expect(issueCategory).toHaveFocus());
    fireEvent.click(issueCategory);
    expect(await screen.findByRole("button", {
      name: /Add an orders index.*#123/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
      "projects/-", "", "", expect.any(AbortSignal)
    );
  });

  test("uses project-scoped database list permission when workspace listing is denied", async () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.canGetWorkspaceProjects = false;
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(await screen.findByRole("button", { name: /main.*orders/ })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name, "", "", expect.any(AbortSignal)
    );
  });

  test("waits for the current project before offering its databases", async () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.projectsByName = {};
    mocks.projectPoliciesByName = {};
    const { rerender } = render(<PickerHarness projectName={project.name} />);

    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    await waitFor(() => expect(mocks.fetchProject).toHaveBeenCalledWith(project.name, true));
    expect(mocks.loadProjectIamPolicy).not.toHaveBeenCalled();

    mocks.projectsByName = { [project.name]: project };
    rerender(<PickerHarness projectName={project.name} />);
    await waitFor(() =>
      expect(mocks.loadProjectIamPolicy).toHaveBeenCalledWith(project.name, true)
    );
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();

    mocks.projectPoliciesByName = { [project.name]: {} };
    rerender(<PickerHarness projectName={project.name} />);
    expect(screen.getByRole("button", { name: "Databases" })).toBeInTheDocument();
  });

  test("silently checks an inline project's policy before offering its databases", async () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.projectPoliciesByName = {};
    const { rerender } = render(
      <PickerHarness selectedProjectName={secondProject.name} />
    );

    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    await waitFor(() =>
      expect(mocks.loadProjectIamPolicy).toHaveBeenCalledWith(
        secondProject.name,
        true
      )
    );
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();

    mocks.projectPoliciesByName = { [secondProject.name]: {} };
    rerender(<PickerHarness selectedProjectName={secondProject.name} />);
    expect(screen.getByRole("button", { name: "Databases" })).toBeInTheDocument();
  });

  test("does not load project policy when workspace database listing is granted", () => {
    mocks.projectPoliciesByName = {};
    render(<PickerHarness projectName={project.name} />);

    expect(screen.getByRole("button", { name: "Databases" })).toBeInTheDocument();
    expect(mocks.loadProjectIamPolicy).not.toHaveBeenCalled();
  });

  test("hides project Databases when project list permission is denied", () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.projectDatabaseListPermissions[project.name] = false;
    render(<PickerHarness projectName={project.name} />);

    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(screen.getByRole("button", { name: "Projects" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
  });

  test("hides project Databases when its list API cannot read the project", () => {
    mocks.canGetWorkspaceProjects = false;
    mocks.projectGetPermissions[project.name] = false;
    render(<PickerHarness projectName={project.name} />);

    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
  });

  test("uses an inline project's database permission over the chat project", () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.projectDatabaseListPermissions[project.name] = false;
    render(
      <PickerHarness
        projectName={secondProject.name}
        selectedProjectName={project.name}
      />
    );

    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
  });

  test("gives issue results more room without widening other picker views", async () => {
    mocks.searchAgentIssues.mockResolvedValueOnce({
      items: [issue],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    const popup = screen.getByRole("dialog", { name: "Add context" });
    expect(popup).toHaveClass("w-80");
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    await screen.findByRole("button", { name: /Add an orders index.*#123/ });
    expect(popup).toHaveClass("w-112");
    expect(popup).toHaveClass("max-w-[calc(100vw-2rem)]");

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(popup).toHaveClass("w-80");
  });

  test("offers Issues in workspace and project contexts", async () => {
    const { rerender } = render(<PickerHarness />);
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    expect(mocks.searchAgentIssues).not.toHaveBeenCalled();

    rerender(<PickerHarness projectName="projects/-" />);
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();

    rerender(<PickerHarness projectName={project.name} />);
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();

    rerender(<PickerHarness selectedProjectName={secondProject.name} />);
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
  });

  test("searches accessible issues across projects from workspace context", async () => {
    mocks.searchAgentIssues.mockResolvedValueOnce({
      items: [issue, otherProjectIssue],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    const row = await screen.findByRole("button", {
      name: /Review payment changes.*#125/,
    });
    expect(row).toHaveClass("focus:ring-offset-2");
    expect(mocks.searchAgentIssues).toHaveBeenCalledExactlyOnceWith(
      "projects/-", "", "", expect.any(AbortSignal)
    );
    fireEvent.click(row);
    expect(mocks.onSelectIssue).toHaveBeenCalledExactlyOnceWith(otherProjectIssue);
  });

  test("does not offer malformed issue names from a workspace search", async () => {
    mocks.searchAgentIssues.mockResolvedValueOnce({
      items: [create(IssueSchema, { name: "projects/-/issues/1", title: "Invalid issue" })],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    expect(await screen.findByText("No issues found")).toBeInTheDocument();
    expect(mocks.onSelectIssue).not.toHaveBeenCalled();
  });

  test("searches issues in the inline project instead of chat context", async () => {
    mocks.searchAgentIssues.mockResolvedValueOnce({ items: [issue], nextPageToken: "" });
    render(<PickerHarness projectName={secondProject.name} selectedProjectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    const row = await screen.findByRole("button", {
      name: /Add an orders index.*#123/,
    });
    expect(mocks.searchAgentIssues).toHaveBeenCalledExactlyOnceWith(
      project.name, "", "", expect.any(AbortSignal)
    );
    expect(row).toHaveTextContent(issue.title);
    expect(row).toHaveTextContent("#123");
    expect(row).not.toHaveTextContent(issue.name);
    fireEvent.click(row);
    expect(mocks.onSelectIssue).toHaveBeenCalledExactlyOnceWith(issue);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("filters out issues outside the selected project", async () => {
    mocks.searchAgentIssues.mockResolvedValueOnce({ items: [issue], nextPageToken: "" });
    render(<PickerHarness selectedProjectName={secondProject.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    expect(await screen.findByText("No issues found")).toBeInTheDocument();
    expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
      secondProject.name, "", "", expect.any(AbortSignal)
    );
    expect(mocks.onSelectIssue).not.toHaveBeenCalled();
  });

  test("searches and paginates issues with shared loading and keyboard UX", async () => {
    mocks.searchAgentIssues
      .mockResolvedValueOnce({ items: [issue], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [issue], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondIssue], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    await screen.findByRole("button", { name: /Add an orders index.*#123/ });
    const search = screen.getByRole("textbox", { name: "Search issues" });
    fireEvent.change(search, { target: { value: "review" } });
    expect(screen.getByRole("status")).toHaveTextContent("Loading");
    expect(screen.queryByRole("button", { name: /Add an orders index.*#123/ })).toBeNull();
    expect(await screen.findByRole("button", { name: "Load more" })).toBeInTheDocument();
    expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
      project.name, "review", "", expect.any(AbortSignal)
    );
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    const secondRow = await screen.findByRole("button", {
      name: /Review query access.*#124/,
    });
    expect(mocks.searchAgentIssues).toHaveBeenLastCalledWith(
      project.name, "review", "next", expect.any(AbortSignal)
    );
    fireEvent.keyDown(search, { key: "ArrowDown" });
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowDown" });
    expect(secondRow).toHaveFocus();
    fireEvent.keyDown(secondRow, { key: "Enter" });
    expect(mocks.onSelectIssue).toHaveBeenCalledExactlyOnceWith(secondIssue);
  });

  test("retries an issue search failure", async () => {
    mocks.searchAgentIssues
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [issue], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load issues");
    expect(screen.queryByText("No issues found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("button", {
      name: /Add an orders index.*#123/,
    })).toBeInTheDocument();
  });

  test("discards an old issue search when the selected project changes", async () => {
    let resolveOldPage!: (page: { items: Issue[]; nextPageToken: string }) => void;
    const oldPage = new Promise<{ items: Issue[]; nextPageToken: string }>(
      (resolve) => { resolveOldPage = resolve; }
    );
    mocks.searchAgentIssues
      .mockReturnValueOnce(oldPage)
      .mockResolvedValueOnce({ items: [otherProjectIssue], nextPageToken: "" });
    const { rerender } = render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    await waitFor(() => expect(mocks.searchAgentIssues).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentIssues.mock.calls[0][3] as AbortSignal;

    rerender(<PickerHarness projectName={project.name} selectedProjectName={secondProject.name} />);
    expect(oldSignal.aborted).toBe(true);
    expect(await screen.findByRole("button", {
      name: /Review payment changes.*#125/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentIssues).toHaveBeenLastCalledWith(
      secondProject.name, "", "", expect.any(AbortSignal)
    );
    await act(async () => {
      resolveOldPage({ items: [issue], nextPageToken: "" });
      await oldPage;
    });
    expect(screen.queryByRole("button", {
      name: /Add an orders index.*#123/,
    })).toBeNull();
  });

  test("explains the project and database limit when another reference is blocked", async () => {
    render(<PickerHarness hasSelectedReference />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message."
    );
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Close" })).toHaveFocus()
    );
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("blocks a third reference after a project and database pair", async () => {
    render(<PickerHarness selectedProjectName={project.name} hasSelectedReference />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message."
    );
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
  });

  test("browses projects without a request until the category is chosen", async () => {
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Projects" })).toHaveFocus()
    );
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(
      getLayerRoot("agent").querySelector("[data-agent-resource-picker]")
    ).not.toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    expect(mocks.searchAgentProjects).toHaveBeenCalledWith(
      "",
      "",
      expect.any(AbortSignal),
      false
    );
    await waitFor(() =>
      expect(screen.getByRole("textbox", { name: "Search projects" })).toHaveFocus()
    );

    fireEvent.click(screen.getByRole("button", { name: "Orders projects/orders" }));
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("browses the current project's databases without choosing a project again", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    const row = await screen.findByRole("button", {
      name: /main.*orders/,
    });
    expect(row).toHaveClass("focus:ring-offset-2");
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    expect(row).toHaveTextContent("main / orders");
    expect(row).not.toHaveTextContent(database.name);
    fireEvent.click(row);
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(database);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("uses an inline project instead of chat context for database search", async () => {
    mocks.canListWorkspaceDatabases = false;
    mocks.projectDatabaseListPermissions[secondProject.name] = false;
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(
      <PickerHarness
        projectName={secondProject.name}
        selectedProjectName={project.name}
      />
    );

    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("button", { name: /main.*orders/ })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
  });

  test("rejects a database returned outside the selected inline project", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness selectedProjectName={secondProject.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByText("No databases found")).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      secondProject.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    expect(mocks.onSelectDatabase).not.toHaveBeenCalled();
  });

  test("shows instance and database names instead of resource paths", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database, secondDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("button", { name: /main.*orders/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reporting.*orders/ })).toBeInTheDocument();
    expect(screen.queryByText(database.name)).toBeNull();
    expect(screen.queryByText(secondDatabase.name)).toBeNull();
  });

  test("uses the unified database display with the instance title", async () => {
    const titledDatabase = create(DatabaseSchema$, {
      name: database.name,
      project: project.name,
      instanceResource: create(InstanceResourceSchema, {
        title: "Main production",
      }),
    });
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [titledDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(
      await screen.findByRole("button", { name: /Main production.*orders/ })
    ).toBeInTheDocument();
  });

  test("browses and selects workspace databases directly when permitted", async () => {
    mocks.canListWorkspaceDatabases = true;
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(screen.queryByText("Choose a project to browse databases")).toBeNull();
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      "workspaces/default",
      "",
      "",
      expect.any(AbortSignal)
    );

    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    fireEvent.click(await screen.findByRole("button", {
      name: /reporting.*orders/,
    }));
    expect(mocks.searchAgentDatabases).toHaveBeenLastCalledWith(
      "workspaces/default",
      "",
      "next",
      expect.any(AbortSignal)
    );
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(secondDatabase);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Add context" })).toBeNull()
    );
  });

  test("does not select a workspace database with a noncanonical owner", async () => {
    mocks.canListWorkspaceDatabases = true;
    const invalidDatabase = create(DatabaseSchema$, {
      name: "instances/main/databases/invalid-owner",
      project: "orders",
    });
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [invalidDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(await screen.findByText("No databases found")).toBeInTheDocument();
    expect(screen.queryByRole("button", {
      name: /main.*invalid-owner/,
    })).toBeNull();
    expect(mocks.onSelectDatabase).not.toHaveBeenCalled();
    expect(mocks.onOpenChange).not.toHaveBeenCalled();
  });

  test("does not call a filtered first page empty when later databases remain", async () => {
    mocks.canListWorkspaceDatabases = true;
    const invalidDatabase = create(DatabaseSchema$, {
      name: "instances/main/databases/invalid-owner",
      project: "orders",
    });
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({
        items: [invalidDatabase],
        nextPageToken: "next",
      })
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(await screen.findByRole("button", { name: "Load more" })).toBeInTheDocument();
    expect(screen.queryByText("No databases found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
  });

  test("leaves workspace database search if list permission is revoked", async () => {
    let resolveOldPage!: (page: {
      items: Database[];
      nextPageToken: string;
    }) => void;
    const oldPage = new Promise<{ items: Database[]; nextPageToken: string }>(
      (resolve) => { resolveOldPage = resolve; }
    );
    mocks.searchAgentDatabases.mockReturnValueOnce(oldPage);
    const { rerender } = render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    await waitFor(() => expect(mocks.searchAgentDatabases).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentDatabases.mock.calls[0][3] as AbortSignal;

    mocks.canListWorkspaceDatabases = false;
    rerender(<PickerHarness />);
    expect(oldSignal.aborted).toBe(true);
    expect(screen.queryByRole("textbox", { name: "Search databases" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(screen.getByRole("button", { name: "Projects" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    await act(async () => {
      resolveOldPage({ items: [database], nextPageToken: "" });
      await oldPage;
    });
    expect(screen.queryByRole("button", {
      name: /main.*orders/,
    })).toBeNull();
  });

  test("searches and paginates databases remotely without dropping earlier results", async () => {
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [database], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    const search = await screen.findByRole("textbox", {
      name: "Search databases",
    });
    await screen.findByRole("button", {
      name: /main.*orders/,
    });
    fireEvent.change(search, { target: { value: "orders" } });
    await waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        project.name,
        "orders",
        "",
        expect.any(AbortSignal)
      )
    );
    fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
    expect(await screen.findByRole("button", {
      name: /reporting.*orders/,
    })).toBeInTheDocument();
    expect(screen.getByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenLastCalledWith(
      project.name,
      "orders",
      "next",
      expect.any(AbortSignal)
    );
  });

  test("shows a loading indicator immediately while database search is debounced", async () => {
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    await screen.findByRole("button", { name: /main.*orders/ });

    fireEvent.change(screen.getByRole("textbox", { name: "Search databases" }), {
      target: { value: "reporting" },
    });
    const loading = screen.getByRole("status");
    expect(loading).toHaveTextContent("Loading");
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.queryByRole("button", { name: /main.*orders/ })).toBeNull();

    await screen.findByRole("button", { name: /reporting.*orders/ });
    expect(screen.queryByRole("status")).toBeNull();
  });

  test("retries a failed database search without calling it empty", async () => {
    mocks.searchAgentDatabases
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load databases");
    expect(screen.queryByText("No databases found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
  });

  test("selects a database with keyboard navigation", async () => {
    mocks.searchAgentDatabases.mockResolvedValue({
      items: [database, secondDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);
    const category = screen.getByRole("button", { name: "Projects" });
    await waitFor(() => expect(category).toHaveFocus());
    fireEvent.keyDown(category, { key: "ArrowDown" });
    const databaseCategory = screen.getByRole("button", { name: "Databases" });
    expect(databaseCategory).toHaveFocus();
    fireEvent.keyDown(databaseCategory, { key: "Enter" });
    const search = await screen.findByRole("textbox", { name: "Search databases" });
    await waitFor(() => expect(search).toHaveFocus());
    await screen.findByRole("button", {
      name: /reporting.*orders/,
    });
    fireEvent.keyDown(search, { key: "ArrowDown" });
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowDown" });
    const second = screen.getByRole("button", {
      name: /reporting.*orders/,
    });
    expect(second).toHaveFocus();
    fireEvent.keyDown(second, { key: "Enter" });
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(secondDatabase);
  });

  test("switches database search to the new workspace when context changes", async () => {
    let resolveOldPage!: (page: {
      items: Database[];
      nextPageToken: string;
    }) => void;
    const oldPage = new Promise<{ items: Database[]; nextPageToken: string }>(
      (resolve) => { resolveOldPage = resolve; }
    );
    mocks.searchAgentDatabases
      .mockReturnValueOnce(oldPage)
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    const { rerender } = render(
      <PickerHarness workspaceName="workspaces/first" />
    );
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    await waitFor(() => expect(mocks.searchAgentDatabases).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentDatabases.mock.calls[0][3] as AbortSignal;

    rerender(<PickerHarness workspaceName="workspaces/second" />);
    expect(oldSignal.aborted).toBe(true);
    expect(await screen.findByRole("button", { name: /reporting.*orders/ })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenLastCalledWith(
      "workspaces/second", "", "", expect.any(AbortSignal)
    );
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    await act(async () => {
      resolveOldPage({ items: [database], nextPageToken: "" });
      await oldPage;
    });
    expect(screen.queryByRole("button", {
      name: /main.*orders/,
    })).toBeNull();
  });

  test("loads another page without dropping the first page", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument();
    expect(mocks.searchAgentProjects).toHaveBeenNthCalledWith(
      2, "", "next", expect.any(AbortSignal), false
    );
  });

  test("shows a loading indicator while fetching the next page", async () => {
    const nextPage = deferredPage();
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockReturnValueOnce(nextPage.promise);
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const loadMore = await screen.findByRole("button", { name: "Load more" });

    fireEvent.click(loadMore);
    const loading = screen.getByRole("button", { name: "Loading" });
    expect(loading).toBeDisabled();
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.getByRole("button", {
      name: "Orders projects/orders",
    })).toBeInTheDocument();

    await act(async () => {
      nextPage.resolve({ items: [secondProject], nextPageToken: "" });
      await nextPage.promise;
    });
    expect(screen.queryByRole("button", { name: "Loading" })).toBeNull();
    expect(screen.getByRole("button", {
      name: "Payments projects/payments",
    })).toBeInTheDocument();
  });

  test("shows each project's canonical resource ID beneath its title", async () => {
    const idLikeTitle = create(ProjectSchema, {
      name: "projects/actual-id",
      title: "projects/looks-like-an-id",
    });
    const matchingTitle = create(ProjectSchema, {
      name: "projects/same",
      title: "projects/same",
    });
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project, idLikeTitle, matchingTitle],
      nextPageToken: "",
    });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    for (const item of [project, idLikeTitle, matchingTitle]) {
      const row = await screen.findByRole("button", {
        name: `${item.title} ${item.name}`,
      });
      const titleAndId = row.querySelector("span.flex-col");
      expect(titleAndId).toHaveClass("flex-col");
      expect(titleAndId?.children).toHaveLength(2);
      expect(titleAndId?.children[0]).toHaveTextContent(item.title);
      expect(titleAndId?.children[1]).toHaveTextContent(item.name);
    }
  });

  test("disambiguates duplicate project titles after loading another page", async () => {
    const first = create(ProjectSchema, {
      name: "projects/new-one",
      title: "New Project",
    });
    const second = create(ProjectSchema, {
      name: "projects/new-two",
      title: "New Project",
    });
    mocks.searchAgentProjects
      .mockResolvedValueOnce({
        items: [first, secondProject],
        nextPageToken: "next",
      })
      .mockResolvedValueOnce({ items: [second], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    const firstRow = await screen.findByRole("button", { name: "New Project projects/new-one" });
    expect(firstRow).toHaveTextContent(first.name);
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() =>
      expect(
        document.querySelector('[data-agent-resource-project="projects/new-two"]')
      ).not.toBeNull()
    );
    expect(
      document.querySelector('[data-agent-resource-project="projects/new-one"]')
    ).toHaveTextContent(first.name);
    expect(
      document.querySelector('[data-agent-resource-project="projects/new-two"]')
    ).toHaveTextContent(second.name);
    expect(screen.getByRole("button", { name: "Payments projects/payments" })).toHaveTextContent(
      secondProject.name
    );
    fireEvent.click(
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-project="projects/new-two"]'
      )!
    );
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(second);
  });

  test("shows a retryable error instead of an empty result", async () => {
    mocks.searchAgentProjects
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("Could not load projects")
    );
    expect(screen.queryByText("No projects found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
  });

  test("shows a loading indicator immediately while project search is debounced", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await screen.findByRole("button", { name: "Orders projects/orders" });

    fireEvent.change(screen.getByRole("textbox", { name: "Search projects" }), {
      target: { value: "payments" },
    });
    const loading = screen.getByRole("status");
    expect(loading).toHaveTextContent("Loading");
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();

    await screen.findByRole("button", { name: "Payments projects/payments" });
    expect(screen.queryByRole("status")).toBeNull();
  });

  test("reloads results when the pending search returns to the current query", async () => {
    mocks.searchAgentProjects.mockResolvedValue({
      items: [project],
      nextPageToken: "",
    });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await screen.findByRole("button", { name: "Orders projects/orders" });

    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    fireEvent.change(search, { target: { value: "" } });
    expect(screen.getByRole("status")).toHaveTextContent("Loading");
    await waitFor(() => expect(mocks.searchAgentProjects).toHaveBeenCalledTimes(2), {
      timeout: DEBOUNCE_SEARCH_DELAY + 1000,
    });
    expect(await screen.findByRole("button", {
      name: "Orders projects/orders",
    })).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
  });

  test("distinguishes an empty project search from a failed search", async () => {
    mocks.searchAgentProjects.mockResolvedValueOnce({ items: [], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByText("No projects found")).toBeInTheDocument()
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });

  test("retries a failed continuation without replacing earlier results", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Load more" })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(mocks.searchAgentProjects).toHaveBeenNthCalledWith(
      3, "", "next", expect.any(AbortSignal), false
    );
  });

  test("keeps the newest query result when an older request finishes late", async () => {
    const oldPage = deferredPage();
    const newPage = deferredPage();
    mocks.searchAgentProjects
      .mockReturnValueOnce(oldPage.promise)
      .mockReturnValueOnce(newPage.promise);
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    await waitFor(() =>
      expect(mocks.searchAgentProjects).toHaveBeenCalledWith(
        "pay", "", expect.any(AbortSignal), false
      ),
      { timeout: DEBOUNCE_SEARCH_DELAY + 1000 }
    );

    await act(async () => {
      newPage.resolve({ items: [secondProject], nextPageToken: "" });
      await newPage.promise;
    });
    expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument();

    await act(async () => {
      oldPage.resolve({ items: [project], nextPageToken: "old-next" });
      await oldPage.promise;
    });
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull();
  });

  test("does not show results from a previous workspace", async () => {
    const oldWorkspacePage = deferredPage();
    mocks.searchAgentProjects
      .mockReturnValueOnce(oldWorkspacePage.promise)
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    const { rerender } = render(<PickerHarness workspaceName="workspaces/first" />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const oldSignal = mocks.searchAgentProjects.mock.calls[0][2] as AbortSignal;

    rerender(<PickerHarness workspaceName="workspaces/second" />);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(oldSignal.aborted).toBe(true);
    await act(async () => {
      oldWorkspacePage.resolve({ items: [project], nextPageToken: "" });
      await oldWorkspacePage.promise;
    });
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
  });

  test("supports keyboard selection, back navigation, and Escape", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project, secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    const category = screen.getByRole("button", { name: "Projects" });
    await waitFor(() => expect(category).toHaveFocus());
    fireEvent.keyDown(category, { key: "Enter" });
    const search = await screen.findByRole("textbox", { name: "Search projects" });
    await waitFor(() => expect(search).toHaveFocus());

    fireEvent.keyDown(search, { key: "ArrowDown" });
    const firstProject = screen.getByRole("button", { name: "Orders projects/orders" });
    expect(firstProject).toHaveFocus();
    expect(firstProject).toHaveClass("focus:bg-control-bg");
    expect(firstProject).toHaveClass("focus:ring-offset-2");
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowDown" });
    const secondProjectButton = screen.getByRole("button", { name: "Payments projects/payments" });
    expect(secondProjectButton).toHaveFocus();
    expect(secondProjectButton).toHaveClass("focus:bg-control-bg");
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowUp" });
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toHaveFocus();

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("button", { name: "Projects" })).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("button", { name: "Projects" }), { key: "Escape" });
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
  });

  test("selects the first search result with Enter", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    fireEvent.keyDown(screen.getByRole("textbox", { name: "Search projects" }), { key: "Enter" });
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
  });

  test("does not select stale results before a new query is sent", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );

    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    fireEvent.keyDown(search, { key: "Enter" });
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
  });

  test("selects a focused result with Enter but not during IME composition", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const result = await screen.findByRole("button", { name: "Orders projects/orders" });
    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.keyDown(search, { key: "Enter", isComposing: true });
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    result.focus();
    fireEvent.keyDown(result, { key: "Enter" });
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
  });
});
