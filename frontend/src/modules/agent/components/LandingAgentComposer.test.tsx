import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  canListWorkspaceProjects: true,
  canListWorkspaceDatabases: true,
  canListProjectDatabases: true,
  projectsByName: {} as Record<string, { name: string; title: string }>,
  projectPoliciesByName: {} as Record<string, object>,
  fetchProject: vi.fn(),
  loadProjectIamPolicy: vi.fn(),
  searchAgentProjects: vi.fn(),
  searchAgentDatabases: vi.fn(),
  searchAgentIssues: vi.fn(),
  selectOrCreateChatForContext: vi.fn(() => ({ id: "chat-1" })),
  queueUserMessage: vi.fn(),
  openAgent: vi.fn(),
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
        "common.back": "Back",
        "common.loading": "Loading",
      })[key] ?? key,
  }),
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
  searchAgentDatabases: mocks.searchAgentDatabases,
  searchAgentIssues: mocks.searchAgentIssues,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: {
    hasWorkspacePermission: (permission: string) => boolean;
    hasProjectPermission: (project: { name: string }, permission: string) => boolean;
    projectsByName: Record<string, { name: string; title: string }>;
    projectPoliciesByName: Record<string, object>;
    fetchProject: typeof mocks.fetchProject;
    loadProjectIamPolicy: typeof mocks.loadProjectIamPolicy;
    getUserByIdentifier: (identifier: string) => undefined;
  }) => unknown) =>
    selector({
      hasWorkspacePermission: (permission) => {
        if (permission === "bb.projects.list") return mocks.canListWorkspaceProjects;
        if (permission === "bb.databases.list") return mocks.canListWorkspaceDatabases;
        if (permission === "bb.projects.get") return true;
        return false;
      },
      hasProjectPermission: (project, permission) =>
        permission === "bb.databases.list" &&
        mocks.canListProjectDatabases &&
        Boolean(project?.name),
      projectsByName: mocks.projectsByName,
      projectPoliciesByName: mocks.projectPoliciesByName,
      fetchProject: mocks.fetchProject,
      loadProjectIamPolicy: mocks.loadProjectIamPolicy,
      getUserByIdentifier: () => undefined,
    }),
}));

vi.mock("../store/agent", () => ({
  selectHasRunningChat: () => false,
  useAgentStore: Object.assign(
    (selector: (state: object) => unknown) =>
      selector({ queuedUserMessage: null }),
    {
      getState: () => ({
        selectOrCreateChatForContext: mocks.selectOrCreateChatForContext,
        queueUserMessage: mocks.queueUserMessage,
        open: mocks.openAgent,
        queuedUserMessage: null,
      }),
    }
  ),
}));

vi.mock("./LandingStarterSelector", () => ({
  LandingStarterSelector: () => null,
}));

import { LandingAgentComposer } from "./LandingAgentComposer";

const workspaceName = "workspaces/default";
const project = create(ProjectSchema, {
  name: "projects/orders",
  title: "Orders",
});
const database = create(DatabaseSchema$, {
  name: "instances/main/databases/orders",
  project: project.name,
  instanceResource: create(InstanceResourceSchema, {
    title: "Main production",
  }),
});
const issue = create(IssueSchema, {
  name: "projects/orders/issues/42",
  title: "Update orders schema",
});

function renderComposer(contextName: string) {
  return render(
    <LandingAgentComposer
      contextName={contextName}
      contextTitle="Current context"
      workspaceName={workspaceName}
      placeholder="Ask a question"
      starterIds={[]}
    />
  );
}

function triggerMention(value: string) {
  const prompt = screen.getByRole("textbox", {
    name: "Ask a question",
  });
  fireEvent.change(prompt, {
    target: { value, selectionStart: value.length },
  });
  return prompt;
}

beforeEach(() => {
  mocks.canListWorkspaceProjects = true;
  mocks.canListWorkspaceDatabases = true;
  mocks.canListProjectDatabases = true;
  mocks.projectsByName = { [project.name]: project };
  mocks.fetchProject.mockReset();
  mocks.fetchProject.mockImplementation(async (name: string) => mocks.projectsByName[name]);
  mocks.loadProjectIamPolicy.mockReset();
  mocks.loadProjectIamPolicy.mockResolvedValue(undefined);
  mocks.searchAgentProjects.mockReset();
  mocks.searchAgentDatabases.mockReset();
  mocks.searchAgentIssues.mockReset();
  mocks.selectOrCreateChatForContext.mockClear();
  mocks.queueUserMessage.mockClear();
  mocks.openAgent.mockClear();
  mocks.searchAgentProjects.mockResolvedValue({
    items: [project],
    nextPageToken: "",
  });
  mocks.searchAgentDatabases.mockResolvedValue({
    items: [database],
    nextPageToken: "",
  });
  mocks.searchAgentIssues.mockResolvedValue({
    items: [issue],
    nextPageToken: "",
  });
});

afterEach(cleanup);

describe("LandingAgentComposer", () => {
  test("removes an unfinished @ trigger when the picker is canceled", () => {
    renderComposer(workspaceName);
    const prompt = triggerMention("Check @");

    fireEvent.keyDown(screen.getByRole("button", { name: "Projects" }), {
      key: "Escape",
    });

    expect(prompt).toHaveValue("Check ");
    expect(screen.queryByRole("dialog", { name: "Add context" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({ text: "Check" })
    );
  });

  test("searches workspace databases from workspace landing", async () => {
    renderComposer(workspaceName);
    triggerMention("@");
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    await waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        workspaceName,
        "",
        "",
        expect.any(AbortSignal)
      )
    );
  });

  test("searches project databases from a project landing page", async () => {
    renderComposer(project.name);
    triggerMention("@");
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    await waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        project.name,
        "",
        "",
        expect.any(AbortSignal)
      )
    );
  });

  test("uses an inline project for its database and queues both references", async () => {
    renderComposer("projects/other");
    triggerMention("@");
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    fireEvent.click(
      await screen.findByRole("button", { name: "Orders projects/orders" })
    );

    expect(triggerMention("@Orders @")).toHaveValue("@Orders @");
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    const row = await screen.findByRole("button", {
      name: /Main production.*orders/,
    });
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    fireEvent.click(row);
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "projects/other",
      expect.objectContaining({ path: expect.any(String) }),
      workspaceName
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "chat-1",
      text: "@Orders @orders (Main production)",
      page: expect.objectContaining({ path: expect.any(String) }),
      resourceReference: {
        kind: "project",
        name: project.name,
        project: project.name,
        label: project.title,
      },
      resourceReferences: [
        {
          kind: "project",
          name: project.name,
          project: project.name,
          label: project.title,
        },
        {
          kind: "database",
          name: database.name,
          project: project.name,
          label: "orders (Main production)",
        },
      ],
    });
  });

  test("offers issues within the project context and queues a selected issue", async () => {
    renderComposer(project.name);
    triggerMention("@");
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    await waitFor(() =>
      expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
        project.name,
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    fireEvent.click(
      await screen.findByRole("button", {
        name: /Update orders schema.*#42/,
      })
    );
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        text: `@${issue.title}`,
        resourceReference: {
          kind: "issue",
          name: issue.name,
          project: project.name,
          label: issue.title,
        },
      })
    );
  });

  test("offers issues across accessible projects in a workspace landing composer", async () => {
    renderComposer(workspaceName);
    triggerMention("@");
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    await waitFor(() =>
      expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
        "projects/-",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    fireEvent.click(
      await screen.findByRole("button", { name: /Update orders schema.*#42/ })
    );
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      workspaceName,
      expect.objectContaining({ path: expect.any(String) }),
      workspaceName
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        resourceReference: {
          kind: "issue",
          name: issue.name,
          project: project.name,
          label: issue.title,
        },
      })
    );
  });

  test("offers issues from an inline project in a workspace landing composer", async () => {
    renderComposer(workspaceName);
    triggerMention("@");
    expect(screen.getByRole("button", { name: "Issues" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    fireEvent.click(
      await screen.findByRole("button", { name: "Orders projects/orders" })
    );

    triggerMention("@Orders @");
    fireEvent.click(screen.getByRole("button", { name: "Issues" }));
    fireEvent.click(
      await screen.findByRole("button", {
        name: /Update orders schema.*#42/,
      })
    );
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        resourceReferences: [
          expect.objectContaining({ kind: "project", name: project.name }),
          expect.objectContaining({ kind: "issue", name: issue.name }),
        ],
      })
    );
  });
});
