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
  isSaaSMode: true,
  isDev: false,
}));

vi.mock("@/utils/featureGates", () => ({
  isSaaSOrDev: (isSaaSMode: boolean) => isSaaSMode || mocks.isDev,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({ router: { push: mocks.push } }));

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
  useWorkspace: () => ({ name: "workspaces/acme", title: "Acme workspace" }),
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
  mocks.isSaaSMode = true;
  mocks.isDev = false;
  mocks.queuedUserMessage = null;
  mocks.lastVisitProjectPath = "";
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

  test("hides the Agent chatbox and skips starter discovery in self-hosted production", () => {
    mocks.isSaaSMode = false;

    render(<LandingPage />);

    expect(
      screen.queryByRole("textbox", { name: "landing.ai-placeholder" })
    ).not.toBeInTheDocument();
    expect(mocks.searchProjects).not.toHaveBeenCalled();
    expect(mocks.listInstances).not.toHaveBeenCalled();
    expect(mocks.listDatabases).not.toHaveBeenCalled();
  });

  test("shows the Agent chatbox in development", () => {
    mocks.isSaaSMode = false;
    mocks.isDev = true;

    render(<LandingPage />);

    expect(
      screen.getByRole("textbox", { name: "landing.ai-placeholder" })
    ).toBeInTheDocument();
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
    expect(screen.queryByTestId("landing-starter-primary")).not.toBeInTheDocument();
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
    mocks.isSaaSMode = false;
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
    mocks.isSaaSMode = true;
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
