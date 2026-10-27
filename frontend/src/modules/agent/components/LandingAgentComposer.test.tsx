import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  canListWorkspaceDatabases: true,
  searchAgentProjects: vi.fn(),
  searchAgentDatabases: vi.fn(),
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
        "agent.resource-picker.search-projects": "Search projects",
        "agent.resource-picker.search-databases": "Search databases",
        "agent.resource-picker.no-projects": "No projects found",
        "agent.resource-picker.no-databases": "No databases found",
        "common.back": "Back",
        "common.loading": "Loading",
      })[key] ?? key,
  }),
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
  searchAgentDatabases: mocks.searchAgentDatabases,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: {
    hasWorkspacePermission: (permission: string) => boolean;
  }) => unknown) =>
    selector({
      hasWorkspacePermission: (permission) =>
        permission === "bb.databases.list" && mocks.canListWorkspaceDatabases,
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
  mocks.canListWorkspaceDatabases = true;
  mocks.searchAgentProjects.mockReset();
  mocks.searchAgentDatabases.mockReset();
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
});

afterEach(cleanup);

describe("LandingAgentComposer", () => {
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
    const row = await screen.findByRole("button", { name: /main.*orders/ });
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
      text: "@Orders @orders",
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
          label: "orders",
        },
      ],
    });
  });
});
