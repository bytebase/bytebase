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
  chatRunning: false,
  queuedUserMessage: null as { chatId: string } | null,
  createChat: vi.fn(() => ({ id: "workspace-chat" })),
  lastVisitProjectPath: "",
  openAgent: vi.fn(),
  push: vi.fn(),
  queueUserMessage: vi.fn(),
  isSaaSMode: false,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({ router: { push: mocks.push } }));

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
  useOptionalCurrentUser: () => ({ email: "creator@example.com" }),
  useServerState: () => ({ isSaaSMode: mocks.isSaaSMode, version: "", changelogURL: "" }),
  useWorkspacePermission: () => mocks.canConfigureAI,
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
        createChat: mocks.createChat,
        open: mocks.openAgent,
        queueUserMessage: mocks.queueUserMessage,
        queuedUserMessage: mocks.queuedUserMessage,
      }),
    }
  ),
}));

vi.mock("./WorkspaceLandingGettingStarted", () => ({
  WorkspaceLandingGettingStarted: () => null,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: object) => unknown) =>
      selector({ serverInfo: { aiEnabled: mocks.aiEnabled } }),
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
  mocks.chatRunning = false;
  mocks.isSaaSMode = false;
  mocks.queuedUserMessage = null;
  mocks.lastVisitProjectPath = "";
  mocks.createChat.mockClear();
  mocks.openAgent.mockClear();
  mocks.queueUserMessage.mockClear();
  localStorage.clear();
});

afterEach(cleanup);

describe("LandingPage", () => {
  test("sends a prompt in a new workspace-bound Agent chat", async () => {
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
    fireEvent.click(screen.getByRole("button", { name: "agent.send" }));

    expect(mocks.createChat).toHaveBeenCalledWith({
      context: "workspaces/acme",
      page: expect.objectContaining({ path: expect.any(String) }),
    });
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
    expect(mocks.createChat).not.toHaveBeenCalled();
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
    expect(mocks.createChat).not.toHaveBeenCalled();
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
