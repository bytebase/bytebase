import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/app/router/handles";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { LandingPage } from "./LandingPage";

const mocks = vi.hoisted(() => ({
  aiEnabled: true as boolean | undefined,
  canConfigureAI: true,
  canSearchAuditLogs: true,
  hasAuditLogFeature: true,
  chatRunning: false,
  queuedUserMessage: null as { chatId: string } | null,
  selectOrCreateChatForContext: vi.fn(() => ({ id: "workspace-chat" })),
  lastVisitProjectPath: "",
  openAgent: vi.fn(),
  push: vi.fn(),
  queueUserMessage: vi.fn(),
  searchProjects: vi.fn(),
  listInstances: vi.fn(),
  listDatabases: vi.fn(),
  workspace: { name: "workspaces/acme", title: "Acme workspace" },
  dev: true,
  isSaaSMode: false,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({ router: { push: mocks.push } }));

vi.mock("@/utils/featureGates", () => ({
  isSaaSOrDev: (isSaaSMode: boolean) => isSaaSMode || mocks.dev,
}));

vi.mock("@/api", () => ({
  projectServiceClientConnect: { searchProjects: mocks.searchProjects },
  instanceServiceClientConnect: { listInstances: mocks.listInstances },
  databaseServiceClientConnect: { listDatabases: mocks.listDatabases },
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children, to }: { children: ReactNode; to: unknown }) => (
    <a href="#" data-route={JSON.stringify(to)}>{children}</a>
  ),
}));

vi.mock("@/components/ui/quick-link", () => ({
  QuickLinkButton: ({ children }: { children: ReactNode }) => (
    <button type="button">{children}</button>
  ),
  QuickLinkLink: ({ children }: { children: ReactNode }) => (
    <a>{children}</a>
  ),
}));

vi.mock("@/components/ui/sheet", () => ({
  Sheet: () => null,
  SheetBody: () => null,
  SheetContent: () => null,
  SheetHeader: () => null,
  SheetTitle: () => null,
}));

vi.mock("@/hooks/useAppState", () => ({
  useOptionalCurrentUser: () => ({
    name: "users/creator",
    email: "creator@example.com",
  }),
  useServerState: () => ({ isSaaSMode: mocks.isSaaSMode, version: "", changelogURL: "" }),
  usePlanFeature: () => mocks.hasAuditLogFeature,
  useWorkspacePermission: (permission: string) =>
    permission === "bb.auditLogs.search"
      ? mocks.canSearchAuditLogs
      : mocks.canConfigureAI,
  useWorkspace: () => mocks.workspace,
}));

vi.mock("@/hooks/useRecentVisit", () => ({
  useRecentVisit: () => ({ lastVisitProjectPath: mocks.lastVisitProjectPath }),
}));

vi.mock("@/modules/agent/store/agent", () => ({
  selectHasRunningChat: () => mocks.chatRunning,
  useAgentStore: Object.assign(
    (selector: (state: object) => unknown) =>
      selector({ queuedUserMessage: mocks.queuedUserMessage }),
    {
      getState: () => ({
        selectOrCreateChatForContext: mocks.selectOrCreateChatForContext,
        open: mocks.openAgent,
        queueUserMessage: mocks.queueUserMessage,
        queuedUserMessage: mocks.queuedUserMessage,
      }),
    }
  ),
}));

vi.mock("@/modules/agent/components/LandingStarterSelector", () => ({
  LandingStarterSelector: ({
    starterIds,
    disabled,
    onSelect,
  }: {
    starterIds: string[];
    disabled: boolean;
    onSelect: (starterId: string) => void;
  }) => (
    <button
      type="button"
      disabled={disabled}
      data-testid="landing-starter-primary"
      data-starter-ids={starterIds.join(",")}
      onClick={() => onSelect(starterIds[0])}
    >
      {starterIds[0]}
    </button>
  ),
}));

vi.mock("@/modules/agent/components/AgentResourcePicker", () => ({
  AgentResourcePicker: ({
    open,
    hasSelectedReference,
    selectedProjectName,
    onSelectProject,
    onSelectDatabase,
  }: {
    open: boolean;
    hasSelectedReference: boolean;
    selectedProjectName?: string;
    onSelectProject: (project: { name: string; title: string }) => void;
    onSelectDatabase: (database: { name: string; project: string }) => void;
  }) =>
    open ? (
      hasSelectedReference ? (
        <p role="status">agent.resource-picker.one-reference-per-message</p>
      ) : (
        <>
          {!selectedProjectName && (
            <>
              <button
                type="button"
                onClick={() =>
                  onSelectProject({ name: "projects/orders", title: "Orders" })
                }
              >
                Select Orders project
              </button>
              <button
                type="button"
                onClick={() =>
                  onSelectProject({ name: "projects/billing", title: "Billing" })
                }
              >
                Select Billing project
              </button>
            </>
          )}
          <button
            type="button"
            data-selected-project-name={selectedProjectName}
            onClick={() =>
              onSelectDatabase({
                name: "instances/primary/databases/orders",
                project: selectedProjectName ?? "projects/orders",
              })
            }
          >
            Select Orders database
          </button>
        </>
      )
    ) : null,
}));

vi.mock("./WorkspaceLandingGettingStarted", () => ({
  WorkspaceLandingGettingStarted: () => null,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: object) => unknown) =>
      selector({
        serverInfo: { aiEnabled: mocks.aiEnabled },
      }),
    {
      getState: () => ({
        getOrFetchProjectByName: async () => ({
          name: "projects/project-sample",
          title: "Sample project",
        }),
      }),
    }
  ),
}));

beforeEach(() => {
  mocks.aiEnabled = true;
  mocks.canConfigureAI = true;
  mocks.canSearchAuditLogs = true;
  mocks.hasAuditLogFeature = true;
  mocks.chatRunning = false;
  mocks.dev = true;
  mocks.isSaaSMode = false;
  mocks.queuedUserMessage = null;
  mocks.lastVisitProjectPath = "";
  mocks.workspace = { name: "workspaces/acme", title: "Acme workspace" };
  mocks.selectOrCreateChatForContext.mockClear();
  mocks.openAgent.mockClear();
  mocks.queueUserMessage.mockClear();
  mocks.searchProjects.mockReset();
  mocks.listInstances.mockReset();
  mocks.listDatabases.mockReset();
  mocks.searchProjects.mockResolvedValue({ projects: [] });
  mocks.listInstances.mockResolvedValue({ instances: [] });
  mocks.listDatabases.mockResolvedValue({ databases: [] });
  localStorage.clear();
});

afterEach(cleanup);

describe("LandingPage", () => {
  test("hides the Agent chatbox in self-hosted release builds", () => {
    mocks.dev = false;
    render(<LandingPage />);

    expect(
      screen.queryByRole("textbox", { name: "landing.ai-placeholder" })
    ).not.toBeInTheDocument();
    expect(mocks.searchProjects).not.toHaveBeenCalled();
    expect(mocks.listInstances).not.toHaveBeenCalled();
    expect(mocks.listDatabases).not.toHaveBeenCalled();
  });

  test("offers a starter above the empty composer and sends it in a read-only chat", async () => {
    render(<LandingPage />);

    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    const starter = screen.getByTestId("landing-starter-primary");
    await waitFor(() =>
      expect(starter).toHaveTextContent("workspace-setup-first")
    );
    expect(
      starter.compareDocumentPosition(prompt) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();

    fireEvent.click(starter);

    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "workspaces/acme",
      expect.objectContaining({ path: expect.any(String) }),
      "workspaces/acme"
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "workspace-chat",
      text: "agent.landing-starters.workspace-setup-first",
      page: expect.objectContaining({ path: expect.any(String) }),
      runMode: "read-only",
      starterId: "workspace-setup-first",
    });
    expect(mocks.openAgent).toHaveBeenCalledOnce();
  });

  test("keeps starters available while composing without discarding the draft", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "My unfinished question" } });

    fireEvent.click(screen.getByTestId("landing-starter-primary"));

    expect(prompt).toHaveValue("My unfinished question");
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        text: expect.stringMatching(/^agent\.landing-starters\./),
        runMode: "read-only",
      })
    );
  });

  test("updates workspace starters when setup resources are found", async () => {
    mocks.searchProjects.mockResolvedValue({
      projects: [{ name: "projects/orders" }],
    });
    render(<LandingPage />);

    await waitFor(() =>
      expect(screen.getByTestId("landing-starter-primary")).toHaveAttribute(
        "data-starter-ids",
        "workspace-attention,workspace-recent"
      )
    );
  });

  test("uses minimal resource reads to discover workspace setup", async () => {
    render(<LandingPage />);

    await waitFor(() =>
      expect(mocks.searchProjects).toHaveBeenCalledWith(
        expect.objectContaining({
          filter: 'exclude_default == true && state == "ACTIVE"',
          pageSize: 1,
        })
      )
    );
    expect(mocks.listInstances).toHaveBeenCalledWith(
      expect.objectContaining({ pageSize: 1, showDeleted: false })
    );
    expect(mocks.listDatabases).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "workspaces/acme",
        pageSize: 1,
        showDeleted: false,
      })
    );
  });

  test("does not assume a workspace needs setup when resource discovery is incomplete", async () => {
    mocks.searchProjects.mockRejectedValue(new Error("permission denied"));
    render(<LandingPage />);

    await waitFor(() =>
      expect(screen.getByTestId("landing-starter-primary")).toHaveAttribute(
        "data-starter-ids",
        "workspace-attention,workspace-recent"
      )
    );
  });

  test("does not offer workspace recent activity without audit-log access", async () => {
    mocks.listInstances.mockResolvedValue({
      instances: [{ name: "instances/primary" }],
    });
    mocks.canSearchAuditLogs = false;
    render(<LandingPage />);

    await waitFor(() =>
      expect(screen.getByTestId("landing-starter-primary")).toHaveAttribute(
        "data-starter-ids",
        "workspace-attention"
      )
    );
  });

  test("sends a prompt in a reusable workspace-bound Agent chat", async () => {
    render(<LandingPage />);

    expect(
      screen.getByRole("heading", { name: "Acme workspace" })
    ).toBeVisible();
    expect(screen.getByText("landing.quick-link.yours")).toBeVisible();
    expect(screen.getByText("settings.sidebar.general")).toBeVisible();
    expect(screen.getByText("agent.context: Acme workspace")).toBeVisible();
    expect(
      screen.queryByRole("button", { name: "project.home.ask-ai" })
    ).not.toBeInTheDocument();

    fireEvent.change(screen.getByRole("textbox", { name: "landing.ai-placeholder" }), {
      target: { value: "  Show the projects in this workspace  " },
    });
    expect(screen.getByTestId("landing-starter-primary")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "workspaces/acme",
      expect.objectContaining({ path: expect.any(String) }),
      "workspaces/acme"
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "workspace-chat",
      text: "Show the projects in this workspace",
      page: expect.objectContaining({ path: expect.any(String) }),
    });
    expect(mocks.openAgent).toHaveBeenCalledOnce();
    expect(screen.getByRole("textbox", { name: "landing.ai-placeholder" })).toHaveValue("");
  });

  test("stages a Project from @ as a one-turn workspace-chat reference", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });

    fireEvent.change(prompt, {
      target: { value: "Check @", selectionStart: 7 },
    });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    expect(prompt).toHaveValue("Check @Orders ");
    expect(document.querySelector("[data-agent-inline-mention]")).toHaveTextContent(
      "@Orders"
    );
    expect(screen.getByText("agent.context: Acme workspace")).toBeVisible();
    expect(screen.queryByTestId("agent-project-reference")).toBeNull();
    expect(screen.getByTestId("landing-starter-primary")).toBeVisible();
    expect(mocks.queueUserMessage).not.toHaveBeenCalled();

    fireEvent.change(prompt, {
      target: { value: "Check @Orders database status" },
    });
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "workspaces/acme",
      expect.objectContaining({ path: expect.any(String) }),
      "workspaces/acme"
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "workspace-chat",
      text: "Check @Orders database status",
      page: expect.objectContaining({ path: expect.any(String) }),
      resourceReference: {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      },
    });
  });

  test("stages a Database from @ without changing workspace chat context", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });

    fireEvent.change(prompt, {
      target: { value: "Check @", selectionStart: 7 },
    });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders database" }));

    expect(prompt).toHaveValue("Check @orders ");
    expect(document.querySelector("[data-agent-inline-mention]")).toHaveTextContent(
      "@orders"
    );
    expect(screen.getByText("agent.context: Acme workspace")).toBeVisible();
    expect(mocks.queueUserMessage).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "workspaces/acme",
      expect.objectContaining({ path: expect.any(String) }),
      "workspaces/acme"
    );
    expect(mocks.queueUserMessage).toHaveBeenCalledWith({
      chatId: "workspace-chat",
      text: "Check @orders",
      page: expect.objectContaining({ path: expect.any(String) }),
      resourceReference: {
        kind: "database",
        name: "instances/primary/databases/orders",
        project: "projects/orders",
        label: "orders",
      },
    });
  });

  test("clears the Project reference when its inline mention is removed", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "@", selectionStart: 1 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    expect(prompt).toHaveValue("@Orders ");
    fireEvent.change(prompt, { target: { value: "Show project status" } });
    expect(document.querySelector("[data-agent-inline-mention]")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(screen.getByText("agent.context: Acme workspace")).toBeVisible();
    expect(screen.getByTestId("landing-starter-primary")).toBeVisible();
    expect(mocks.queueUserMessage).toHaveBeenLastCalledWith({
      chatId: "workspace-chat",
      text: "Show project status",
      page: expect.objectContaining({ path: expect.any(String) }),
    });
  });

  test("Backspace after a selected Project removes the whole mention and its separator", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    }) as HTMLTextAreaElement;
    fireEvent.change(prompt, { target: { value: "Check @", selectionStart: 7 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    expect(prompt).toHaveValue("Check @Orders ");
    prompt.setSelectionRange(prompt.value.length, prompt.value.length);
    fireEvent.keyDown(prompt, { key: "Backspace" });

    expect(prompt).toHaveValue("Check ");
    expect(prompt.selectionStart).toBe(6);
    expect(document.querySelector("[data-agent-inline-mention]")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.queueUserMessage).toHaveBeenLastCalledWith({
      chatId: "workspace-chat",
      text: "Check",
      page: expect.objectContaining({ path: expect.any(String) }),
    });
  });

  test.each([
    { name: "Option+Backspace", key: "Backspace", altKey: true },
    { name: "Command+Backspace", key: "Backspace", metaKey: true },
    { name: "Control+Delete", key: "Delete", ctrlKey: true },
  ])("$name inside a selected Project removes the whole mention", (shortcut) => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    }) as HTMLTextAreaElement;
    fireEvent.change(prompt, { target: { value: "Check @", selectionStart: 7 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    prompt.setSelectionRange(9, 9);
    fireEvent.keyDown(prompt, shortcut);

    expect(prompt).toHaveValue("Check ");
    expect(prompt.selectionStart).toBe(6);
    expect(document.querySelector("[data-agent-inline-mention]")).toBeNull();
  });

  test("clears the draft and Project reference when the workspace changes", async () => {
    const view = render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "Check @", selectionStart: 7 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));
    expect(prompt).toHaveValue("Check @Orders ");
    expect(document.querySelector("[data-agent-inline-mention]")).toHaveTextContent(
      "@Orders"
    );

    mocks.workspace = { name: "workspaces/other", title: "Other workspace" };
    view.rerender(<LandingPage />);

    const newPrompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    await waitFor(() => expect(newPrompt).toHaveValue(""));
    expect(document.querySelector("[data-agent-inline-mention]")).toBeNull();
    expect(screen.queryByRole("button", { name: "Select Orders project" })).toBeNull();
    expect(screen.getByRole("button", { name: "agent.send" })).toBeDisabled();
    expect(screen.getByText("agent.context: Other workspace")).toBeVisible();

    fireEvent.change(newPrompt, { target: { value: "Show workspace status" } });
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.selectOrCreateChatForContext).toHaveBeenCalledWith(
      "workspaces/other",
      expect.objectContaining({ path: expect.any(String) }),
      "workspaces/other"
    );
    expect(mocks.queueUserMessage).toHaveBeenLastCalledWith({
      chatId: "workspace-chat",
      text: "Show workspace status",
      page: expect.objectContaining({ path: expect.any(String) }),
    });
  });

  test("blocks a second Project reference until the first mention is removed", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "Check @", selectionStart: 7 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    const secondTrigger = "Check @Orders and @";
    fireEvent.change(prompt, {
      target: { value: secondTrigger, selectionStart: secondTrigger.length },
    });
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.queryByRole("button", { name: "Select Billing project" })).toBeNull();
    expect(screen.getByRole("button", { name: "Select Orders database" })).toHaveAttribute(
      "data-selected-project-name",
      "projects/orders"
    );
    expect(prompt).toHaveValue(secondTrigger);
    expect(document.querySelector("[data-agent-inline-mention]")).toHaveTextContent(
      "@Orders"
    );
    expect(screen.getByRole("button", { name: "agent.send" })).toBeDisabled();

    fireEvent.keyDown(prompt, { key: "Escape" });
    expect(screen.queryByRole("status")).toBeNull();
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.getByRole("button", { name: "agent.send" })).toBeDisabled();

    const replacementTrigger = "Check and @";
    fireEvent.change(prompt, {
      target: { value: replacementTrigger, selectionStart: replacementTrigger.length },
    });
    expect(screen.queryByText("agent.resource-picker.one-reference-per-message")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Select Billing project" }));
    expect(prompt).toHaveValue("Check and @Billing ");
    expect(screen.getByRole("button", { name: "agent.send" })).toBeEnabled();

    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.queueUserMessage).toHaveBeenLastCalledWith(
      expect.objectContaining({
        text: "Check and @Billing",
        resourceReference: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      })
    );
  });

  test("allows a database reference after a Project and sends both references", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "Check @", selectionStart: 7 } });
    fireEvent.click(screen.getByRole("button", { name: "Select Orders project" }));

    const databaseTrigger = "Check @Orders and @";
    fireEvent.change(prompt, {
      target: { value: databaseTrigger, selectionStart: databaseTrigger.length },
    });
    const database = screen.getByRole("button", { name: "Select Orders database" });
    expect(database).toHaveAttribute("data-selected-project-name", "projects/orders");
    fireEvent.click(database);

    expect(prompt).toHaveValue("Check @Orders and @orders ");
    expect(screen.getByRole("button", { name: "agent.send" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));
    expect(mocks.queueUserMessage).toHaveBeenLastCalledWith(
      expect.objectContaining({
        text: "Check @Orders and @orders",
        resourceReferences: [
          {
            kind: "project",
            name: "projects/orders",
            project: "projects/orders",
            label: "Orders",
          },
          {
            kind: "database",
            name: "instances/primary/databases/orders",
            project: "projects/orders",
            label: "orders",
          },
        ],
      })
    );
  });

  test("keeps an email address literal instead of opening the Project picker", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, {
      target: { value: "ed@example.com", selectionStart: 14 },
    });

    expect(prompt).toHaveValue("ed@example.com");
    expect(
      screen.queryByRole("button", { name: "Select Orders project" })
    ).not.toBeInTheDocument();
  });

  test("does not send an @ trigger while the Project picker is open", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", {
      name: "landing.ai-placeholder",
    });
    fireEvent.change(prompt, { target: { value: "@", selectionStart: 1 } });

    expect(
      screen.getByRole("button", { name: "Select Orders project" })
    ).toBeVisible();
    expect(fireEvent.keyDown(prompt, { key: "Enter", shiftKey: true })).toBe(true);
    expect(fireEvent.keyDown(prompt, { key: "Enter" })).toBe(false);
    expect(mocks.queueUserMessage).not.toHaveBeenCalled();
  });

  test("shows the keyboard hint and sends only on unmodified Enter", () => {
    render(<LandingPage />);
    const prompt = screen.getByRole("textbox", { name: "landing.ai-placeholder" });
    expect(prompt).toHaveAttribute(
      "placeholder",
      "landing.ai-placeholder · agent.input-keyboard-hint"
    );
    expect(screen.queryByText("agent.input-keyboard-hint")).not.toBeInTheDocument();
    expect(
      screen.getByText("agent.context: Acme workspace").parentElement?.parentElement
    ).toHaveClass("items-end");
    fireEvent.change(prompt, { target: { value: "Workspace\nprojects" } });

    expect(fireEvent.keyDown(prompt, { key: "Enter", shiftKey: true })).toBe(true);
    expect(mocks.selectOrCreateChatForContext).not.toHaveBeenCalled();
    expect(fireEvent.keyDown(prompt, { key: "Enter" })).toBe(false);
    expect(mocks.queueUserMessage).toHaveBeenCalledWith(
      expect.objectContaining({ text: "Workspace\nprojects" })
    );
  });

  test("guides an eligible workspace admin to AI configuration", () => {
    mocks.aiEnabled = false;
    const { rerender } = render(<LandingPage />);

    expect(
      screen.queryByRole("textbox", { name: "landing.ai-placeholder" })
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "agent.send" })).not.toBeInTheDocument();
    expect(screen.getByText("landing.ai-setup.description")).toBeVisible();
    const configure = screen.getByRole("link", { name: "landing.ai-setup.action" });
    expect(JSON.parse(configure.getAttribute("data-route")!)).toEqual({
      name: SETTING_ROUTE_WORKSPACE_GENERAL,
      hash: "#ai-assistant",
      query: { [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO },
    });

    mocks.aiEnabled = true;
    rerender(<LandingPage />);
    expect(screen.queryByRole("link", { name: "landing.ai-setup.action" })).not.toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "landing.ai-placeholder" })).toBeVisible();
  });

  test.each([
    { aiEnabled: false, canConfigureAI: false, isSaaSMode: false },
    { aiEnabled: false, canConfigureAI: true, isSaaSMode: true },
    { aiEnabled: undefined, canConfigureAI: true, isSaaSMode: false },
  ])("does not offer AI setup when configuration is unavailable: %j", ({ aiEnabled, canConfigureAI, isSaaSMode }) => {
    mocks.aiEnabled = aiEnabled;
    mocks.canConfigureAI = canConfigureAI;
    mocks.isSaaSMode = isSaaSMode;
    render(<LandingPage />);

    expect(screen.queryByRole("link", { name: "landing.ai-setup.action" })).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "landing.ai-placeholder" })).not.toBeInTheDocument();
  });

  test("does not send an empty prompt or queue behind another Agent run", () => {
    const { rerender } = render(<LandingPage />);
    const send = screen.getByRole("button", { name: "agent.send" });
    expect(send).toBeDisabled();

    fireEvent.change(screen.getByRole("textbox", { name: "landing.ai-placeholder" }), {
      target: { value: "What is configured?" },
    });
    expect(send).toBeEnabled();

    mocks.chatRunning = true;
    rerender(<LandingPage />);
    expect(send).toBeDisabled();
    fireEvent.submit(send.closest("form")!);
    expect(mocks.selectOrCreateChatForContext).not.toHaveBeenCalled();
  });

  test("keeps Last visit beside the shortcuts title", async () => {
    mocks.lastVisitProjectPath = "/projects/project-sample";
    render(<LandingPage />);

    const title = screen.getByRole("heading", {
      name: "landing.quick-link.yours",
    });
    await waitFor(() =>
      expect(
        within(title.parentElement!).getByRole("link", {
          name: "landing.last-visit: Sample project",
        })
      ).toBeVisible()
    );
  });
});
