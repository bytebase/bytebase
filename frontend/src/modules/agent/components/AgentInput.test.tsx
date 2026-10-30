import type { ReactElement } from "react";
import { act, createRef } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { AIChatResponse_StopReason } from "@/types/proto-es/v1/ai_service_pb";
import type { DomRefSuggestion } from "../dom";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  aiChat: vi.fn(),
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  })),
  routerPush: vi.fn(),
  lazyExtractDomRefSuggestions: vi.fn<() => Promise<DomRefSuggestion[]>>(
    async () => []
  ),
  runAgentLoop: vi.fn(),
  isAgentAIConfigurationError: vi.fn(() => false),
  buildOutboundHistory: vi.fn(() => []),
  buildSystemPrompt: vi.fn(() => "system"),
  executeTool: vi.fn(),
  createToolExecutor: vi.fn(() => mocks.executeTool),
  getToolDefinitions: vi.fn(() => []),
  getPendingApiCallPreview: vi.fn(),
  resolvePendingApiCall: vi.fn(),
  resolveDeniedQueryAccessDraft: vi.fn(),
  getRecentActivity: vi.fn(),
  getWorkspaceAttention: vi.fn(),
  getProjectAttention: vi.fn(),
  extractWorkspaceOverviewEvidence: vi.fn((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  }),
  extractIssueOverviewEvidence: vi.fn((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  }),
  extractPlanOverviewEvidence: vi.fn((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  }),
  extractDatabaseOverviewEvidence: vi.fn((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  }),
  getOrFetchSettingByName: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
  searchAgentProjects: vi.fn(),
  searchAgentDatabases: vi.fn(),
  searchAgentIssues: vi.fn(),
  searchAgentPlans: vi.fn(),
  canListWorkspaceProjects: true,
  canListWorkspaceDatabases: true,
  canListProjectDatabases: true,
  fetchProject: vi.fn(),
  loadProjectIamPolicy: vi.fn(),
  projectsByName: {} as Record<string, { name: string; title: string }>,
  projectPoliciesByName: {} as Record<string, object>,
  availableProjects: [] as { name: string; title: string }[],
  workspace: { name: "workspaces/demo", title: "Demo workspace" },
  currentContext: "projects/orders" as string | undefined,
  currentPath: "/demo",
}));

vi.mock("@/api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api")>()),
  aiServiceClientConnect: { chat: mocks.aiChat },
}));

let AgentInput: typeof import("./AgentInput").AgentInput;

function createMockStorage(): Storage {
  let store: Record<string, string> = {};
  return {
    get length() {
      return Object.keys(store).length;
    },
    key(index: number) {
      return Object.keys(store)[index] ?? null;
    },
    getItem(key: string) {
      return store[key] ?? null;
    },
    setItem(key: string, value: string) {
      store[key] = String(value);
    },
    removeItem(key: string) {
      delete store[key];
    },
    clear() {
      store = {};
    },
  };
}

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    currentRoute: {
      value: {
        get fullPath() {
          return mocks.currentPath;
        },
      },
    },
    push: mocks.routerPush,
  },
}));

vi.mock("@/stores/app", () => {
  const state = {
    currentUser: { name: "users/me@example.com" },
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    getSettingByName: () => undefined,
    hasWorkspacePermission: (permission: string) => {
      if (permission === "bb.projects.list") return mocks.canListWorkspaceProjects;
      if (permission === "bb.databases.list") return mocks.canListWorkspaceDatabases;
      if (permission === "bb.projects.get") return true;
      return false;
    },
    hasProjectPermission: (project: { name: string }, permission: string) =>
      permission === "bb.databases.list" &&
      mocks.canListProjectDatabases &&
      Boolean(project?.name),
    fetchProject: mocks.fetchProject,
    loadProjectIamPolicy: mocks.loadProjectIamPolicy,
    getUserByIdentifier: () => undefined,
    get projectsByName() {
      return mocks.projectsByName;
    },
    get projectPoliciesByName() {
      return mocks.projectPoliciesByName;
    },
    get workspace() {
      return mocks.workspace;
    },
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("@/hooks/useAppState", () => ({
  useProjectList: () => ({
    projects: mocks.availableProjects,
    isLoading: false,
    isFetchingMore: false,
    hasMore: false,
    loadMore: vi.fn(),
  }),
}));

vi.mock("../dom", () => ({
  lazyExtractDomRefSuggestions: mocks.lazyExtractDomRefSuggestions,
}));

vi.mock("../logic/agentLoop", () => ({
  runAgentLoop: mocks.runAgentLoop,
}));

vi.mock("../logic/aiConfiguration", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../logic/aiConfiguration")>()),
  isAgentAIConfigurationError: mocks.isAgentAIConfigurationError,
}));

vi.mock("../logic/outboundHistory", () => ({
  buildOutboundHistory: mocks.buildOutboundHistory,
}));

vi.mock("../logic/prompt", () => ({
  buildSystemPrompt: mocks.buildSystemPrompt,
}));

vi.mock("../logic/chatContext", () => ({
  getCurrentChatContext: () => mocks.currentContext,
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
  searchAgentDatabases: mocks.searchAgentDatabases,
  searchAgentIssues: mocks.searchAgentIssues,
  searchAgentPlans: mocks.searchAgentPlans,
}));

vi.mock("../logic/tools", () => ({
  createToolExecutor: mocks.createToolExecutor,
  getToolDefinitions: mocks.getToolDefinitions,
}));

vi.mock("../logic/tools/recentActivity", () => ({
  getRecentActivity: mocks.getRecentActivity,
}));

vi.mock("../logic/tools/workspaceAttention", () => ({
  createWorkspaceAttentionTool: vi.fn(() => mocks.getWorkspaceAttention),
}));

vi.mock("../logic/tools/projectOpenIssues", () => ({
  createProjectOpenIssuesTool: vi.fn(() => mocks.getProjectAttention),
}));

vi.mock("../logic/tools/workspaceOverview", () => ({
  extractWorkspaceOverviewEvidence: mocks.extractWorkspaceOverviewEvidence,
}));

vi.mock("../logic/tools/issueOverview", () => ({
  extractIssueOverviewEvidence: mocks.extractIssueOverviewEvidence,
}));

vi.mock("../logic/tools/planOverview", () => ({
  extractPlanOverviewEvidence: mocks.extractPlanOverviewEvidence,
}));

vi.mock("../logic/tools/databaseOverview", () => ({
  extractDatabaseOverviewEvidence: mocks.extractDatabaseOverviewEvidence,
}));

vi.mock("../logic/tools/pendingApiCall", () => ({
  getPendingApiCallPreview: mocks.getPendingApiCallPreview,
  resolvePendingApiCall: mocks.resolvePendingApiCall,
}));

vi.mock("../logic/deniedQueryAccess", () => ({
  resolveDeniedQueryAccessDraft: mocks.resolveDeniedQueryAccessDraft,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const setTextareaValue = (textarea: HTMLTextAreaElement, value: string) => {
  const descriptor = Object.getOwnPropertyDescriptor(
    HTMLTextAreaElement.prototype,
    "value"
  );
  descriptor?.set?.call(textarea, value);
};

const getQueuedUserMessage = (chatId: string) =>
  useAgentStore.getState().queuedUserMessagesByChatId[chatId];

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
  const initialState = createAgentStore().getState();
  useAgentStore.setState({
    workspaceName: "workspaces/demo",
    userEmail: "agent@example.com",
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    draftsByChatId: initialState.draftsByChatId,
    queuedUserMessagesByChatId: {},
    queuedUserMessageInFlightByChatId: {},
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
  });

  mocks.useTranslation.mockReset();
  mocks.aiChat.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  });
  mocks.routerPush.mockReset();
  mocks.lazyExtractDomRefSuggestions.mockReset();
  mocks.lazyExtractDomRefSuggestions.mockResolvedValue([]);
  mocks.runAgentLoop.mockReset();
  mocks.isAgentAIConfigurationError.mockReset();
  mocks.isAgentAIConfigurationError.mockReturnValue(false);
  mocks.buildOutboundHistory.mockReset();
  mocks.buildOutboundHistory.mockReturnValue([]);
  mocks.buildSystemPrompt.mockReset();
  mocks.buildSystemPrompt.mockReturnValue("system");
  mocks.createToolExecutor.mockReset();
  mocks.executeTool.mockReset();
  mocks.resolveDeniedQueryAccessDraft.mockReset();
  mocks.executeTool.mockResolvedValue({
    kind: "tool_result",
    result: JSON.stringify({
      project: "projects/billing",
      personalIssues: { status: "available", partial: false, issues: [] },
      databases: { status: "available", partial: false, databases: [] },
      openDatabaseChanges: { status: "available", partial: false, issues: [] },
      evidence: { project: "projects/billing", references: [] },
    }),
  });
  mocks.createToolExecutor.mockReturnValue(mocks.executeTool);
  mocks.getToolDefinitions.mockReset();
  mocks.getToolDefinitions.mockReturnValue([]);
  mocks.getPendingApiCallPreview.mockReset();
  mocks.resolvePendingApiCall.mockReset();
  mocks.resolvePendingApiCall.mockResolvedValue('{"status":200,"response":{}}');
  mocks.getRecentActivity.mockReset();
  mocks.getWorkspaceAttention.mockReset();
  mocks.getProjectAttention.mockReset();
  mocks.extractIssueOverviewEvidence.mockReset();
  mocks.extractIssueOverviewEvidence.mockImplementation((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  });
  mocks.extractPlanOverviewEvidence.mockReset();
  mocks.extractPlanOverviewEvidence.mockImplementation((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  });
  mocks.extractDatabaseOverviewEvidence.mockReset();
  mocks.extractDatabaseOverviewEvidence.mockImplementation((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  });
  mocks.getOrFetchSettingByName.mockReset();
  mocks.getOrFetchSettingByName.mockResolvedValue(undefined);
  mocks.getOrFetchProjectByName.mockReset();
  mocks.getOrFetchProjectByName.mockResolvedValue(undefined);
  mocks.searchAgentProjects.mockReset();
  mocks.searchAgentProjects.mockResolvedValue({
    items: [
      { name: "projects/orders", title: "Order service" },
      { name: "projects/billing", title: "Billing" },
    ],
    nextPageToken: "",
  });
  mocks.searchAgentDatabases.mockReset();
  mocks.canListWorkspaceProjects = true;
  mocks.canListWorkspaceDatabases = true;
  mocks.canListProjectDatabases = true;
  mocks.fetchProject.mockReset();
  mocks.fetchProject.mockImplementation(async (name: string) => mocks.projectsByName[name]);
  mocks.loadProjectIamPolicy.mockReset();
  mocks.loadProjectIamPolicy.mockResolvedValue(undefined);
  mocks.searchAgentDatabases.mockResolvedValue({
    items: [
      {
        name: "instances/prod/databases/orders",
        project: "projects/orders",
      },
    ],
    nextPageToken: "",
  });
  mocks.searchAgentIssues.mockReset();
  mocks.searchAgentIssues.mockResolvedValue({
    items: [
      {
        name: "projects/orders/issues/42",
        title: "Update orders schema",
      },
    ],
    nextPageToken: "",
  });
  mocks.searchAgentPlans.mockReset();
  mocks.searchAgentPlans.mockResolvedValue({
    items: [
      {
        name: "projects/orders/plans/7",
        title: "Change orders schema",
        state: 1,
      },
    ],
    nextPageToken: "",
  });
  mocks.projectsByName = {
    "projects/orders": { name: "projects/orders", title: "Order service" },
    "projects/billing": { name: "projects/billing", title: "Billing" },
  };
  mocks.availableProjects = [
    { name: "projects/orders", title: "Order service" },
    { name: "projects/billing", title: "Billing" },
  ];
  mocks.workspace = { name: "workspaces/demo", title: "Demo workspace" };
  mocks.currentContext = "projects/orders";
  mocks.currentPath = "/demo";

  ({ AgentInput } = await import("./AgentInput"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("AgentInput", () => {
  test("focuses the end of a prefilled composer draft", () => {
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
    });
    const focusActionRef = createRef<((atEnd?: boolean) => void) | null>();
    const view = renderIntoContainer(
      <AgentInput focusActionRef={focusActionRef} />
    );
    view.render();

    act(() => {
      useAgentStore.getState().setDraft(chat.id, {
        text: "@app (Primary) ",
      });
    });
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    act(() => {
      textarea.setSelectionRange(0, 0);
      focusActionRef.current?.(true);
    });

    expect(textarea.value).toBe("@app (Primary) ");
    expect(document.activeElement).toBe(textarea);
    expect(textarea.selectionStart).toBe(textarea.value.length);
    expect(textarea.selectionEnd).toBe(textarea.value.length);
    act(() => {
      textarea.setSelectionRange(3, 3);
      focusActionRef.current?.();
    });
    expect(textarea.selectionStart).toBe(3);
    expect(useAgentStore.getState().getMessages(chat.id)).toEqual([]);
    view.unmount();
  });

  test("uses a growing textarea without a row cap", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = container.querySelector(
      'textarea[aria-label="agent.input-placeholder"]'
    );
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);
    expect(textarea?.className).toContain("min-h-9");
    expect(textarea?.className).not.toContain("max-h-");

    unmount();
  });

  test("shows the send and newline keyboard hint only in the empty input placeholder", () => {
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.querySelector("[data-agent-input-placeholder]")?.textContent).toContain(
      "agent.input-keyboard-hint"
    );
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    act(() => {
      setTextareaValue(textarea, "List databases");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(view.container.querySelector("[data-agent-input-placeholder]")).toBeNull();
    expect(view.container.textContent).not.toContain("agent.input-keyboard-hint");
    view.unmount();
  });

  test("sends a queued Project Home message in its bound chat once", async () => {
    mocks.runAgentLoop.mockResolvedValue({
      kind: "completed",
      totalTokensUsed: 0,
    });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "List billing databases",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(useAgentStore.getState().getMessages(chat.id)[0]).toEqual(
      expect.objectContaining({ role: "user", content: "List billing databases" })
    );
    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "projects/billing",
        runMode: "read-only",
      })
    );
    expect(mocks.getToolDefinitions).toHaveBeenCalledWith({
      runMode: "read-only",
    });
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/billing",
        runMode: "read-only",
      })
    );
    expect(getQueuedUserMessage(chat.id)).toBeUndefined();
    view.unmount();
  });

  test("starts queued turns in separate chats while one is in the background", async () => {
    const resolveRuns: Array<
      (result: { kind: "completed"; totalTokensUsed: number }) => void
    > = [];
    mocks.runAgentLoop.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRuns.push(resolve);
        })
    );
    const store = useAgentStore.getState();
    const page = { path: "/projects/billing", title: "Billing" };
    const first = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: first.id, text: "First question", page });
    const second = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: second.id, text: "Second question", page });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });

    expect(store.getChat(first.id)?.status).toBe("running");
    expect(store.getChat(second.id)?.status).toBe("running");
    expect(store.getMessages(first.id).map((message) => message.content)).toEqual([
      "First question",
    ]);
    expect(store.getMessages(second.id).map((message) => message.content)).toEqual([
      "Second question",
    ]);
    const firstExecutorOptions = (
      mocks.createToolExecutor.mock.calls as unknown as Array<
        [unknown, { chatId: string; canUseCurrentPage: () => boolean }]
      >
    )[0][1];
    expect(firstExecutorOptions.chatId).toBe(first.id);
    expect(firstExecutorOptions.canUseCurrentPage()).toBe(false);
    act(() => store.setCurrentChat(first.id));
    expect(firstExecutorOptions.canUseCurrentPage()).toBe(true);

    await act(async () => {
      resolveRuns.forEach((resolve) =>
        resolve({ kind: "completed", totalTokensUsed: 0 })
      );
      await vi.waitFor(() => {
        expect(store.getChat(first.id)?.status).toBe("idle");
        expect(store.getChat(second.id)?.status).toBe("idle");
      });
    });
    view.unmount();
  });

  test("sends a queued Ask next question with both bound resources", async () => {
    mocks.runAgentLoop.mockResolvedValue({
      kind: "completed",
      totalTokensUsed: 0,
    });
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        database: {
          status: "available",
          name: "instances/prod/databases/orders",
          state: "ACTIVE",
        },
      }),
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    const references = [
      {
        kind: "project" as const,
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      },
      {
        kind: "database" as const,
        name: "instances/prod/databases/orders",
        project: "projects/orders",
        label: "orders (prod)",
      },
    ];
    store.queueUserMessage({
      chatId: chat.id,
      text: "@Orders @orders (prod) What should I review next?",
      page,
      runMode: "default",
      resourceReferences: references,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0]).toMatchObject({
      role: "user",
      content: "@Orders @orders (prod) What should I review next?",
      metadata: { resourceReferences: references },
    });
    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({ runMode: "read-only" })
    );
    expect(getQueuedUserMessage(chat.id)).toBeUndefined();
    view.unmount();
  });

  test("runs the workspace recent starter with one direct audit read", async () => {
    let resolveRecentActivity!: (value: string) => void;
    mocks.getRecentActivity.mockReturnValue(
      new Promise<string>((resolve) => {
        resolveRecentActivity = resolve;
      })
    );
    const page = { path: "/landing", title: "Workspace" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What changed recently in this workspace?",
      page,
      runMode: "read-only",
      starterId: "workspace-recent",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(mocks.getRecentActivity).toHaveBeenCalledWith("projects/-")
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getMessages(chat.id)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: "agent.recent-activity.checking",
          metadata: expect.objectContaining({ pending: true }),
        }),
      ])
    );

    await act(async () => {
      resolveRecentActivity(
        JSON.stringify({
          events: [
            {
              actor: "users/alice@example.com",
              action: "approved an Issue",
              resource: "projects/orders/issues/42",
              occurredAt: "2026-10-20T08:00:00.000Z",
            },
          ],
          partial: true,
        })
      );
    });
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .some((message) => message.metadata?.pending)
        ).toBe(false)
      );
    });

    expect(useAgentStore.getState().getMessages(chat.id)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: expect.stringContaining("agent.recent-activity.event"),
        }),
      ])
    );
    view.unmount();
  });

  test("runs the project recent starter with one direct project audit read", async () => {
    mocks.getRecentActivity.mockResolvedValue(
      JSON.stringify({ events: [], partial: false })
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What changed recently in this project?",
      page,
      runMode: "read-only",
      starterId: "project-open-recent",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(mocks.getRecentActivity).toHaveBeenCalledWith("projects/orders")
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("runs a project starter through the generic read-only Agent path", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "tool-project-overview",
              name: "get_project_overview",
              arguments: "{}",
            },
          ],
        });
        callbacks.onToolResult(
          "tool-project-overview",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "database",
                  name: "projects/orders/instances/prod/databases/orders",
                },
              ],
            },
          })
        );
        callbacks.onText("The project is ready for a governed change.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What can I do next?",
      page,
      runMode: "read-only",
      starterId: "project-next",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "projects/orders",
        runMode: "read-only",
      })
    );
    expect(mocks.getToolDefinitions).toHaveBeenCalledWith({
      runMode: "read-only",
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find(
          (message) =>
            message.content === "The project is ready for a governed change."
        )?.metadata?.evidence
    ).toEqual({
      project: "projects/orders",
      references: [
        {
          kind: "database",
          name: "projects/orders/instances/prod/databases/orders",
        },
      ],
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .some((message) => message.metadata?.pending)
    ).toBe(false);
    view.unmount();
  });

  test("attaches workspace overview evidence to the final answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "tool-workspace-overview",
              name: "get_workspace_overview",
              arguments: "{}",
            },
          ],
        });
        callbacks.onToolResult(
          "tool-workspace-overview",
          JSON.stringify({
            evidence: {
              workspace: "workspaces/default",
              references: [
                {
                  kind: "issue",
                  name: "projects/orders/issues/42",
                  title: "Add customer table",
                },
              ],
            },
          })
        );
        callbacks.onText("One issue needs your review.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/landing", title: "Workspace" };
    const chat = useAgentStore.getState().createChat({
      context: "workspaces/default",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "What can I do next?",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "One issue needs your review.")
        ?.metadata?.evidence
    ).toEqual({
      workspace: "workspaces/default",
      references: [
        {
          kind: "issue",
          name: "projects/orders/issues/42",
          title: "Add customer table",
        },
      ],
    });
    view.unmount();
  });

  test("attaches only an observed handoff to a final answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [{ id: "overview-1", name: "get_project_overview", arguments: "{}" }],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            personalIssues: {
              status: "available",
              issues: [{ name: "projects/orders/issues/42", relation: "review" }],
            },
          })
        );
        callbacks.onText("Review this issue.", {
          kind: "review-project-issues",
          project: "projects/orders",
        });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({ chatId: chat.id, text: "What should I review?", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(useAgentStore.getState().getMessages(chat.id).find((message) => message.content === "Review this issue.")?.metadata?.handoff).toEqual({
      kind: "review-project-issues",
      project: "projects/orders",
    });
    view.unmount();
  });

  test("generates three contextual questions for an accepted action without Continue", async () => {
    mocks.aiChat.mockResolvedValue({
      content: JSON.stringify([
        "What information do I need to connect a new instance?",
        "Which databases can I transfer?",
        "How do I verify the transfer?",
      ]),
      toolCalls: [],
      stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
      usage: { totalTokens: 7 },
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [{ id: "overview-1", name: "get_project_overview", arguments: "{}" }],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            databases: { status: "available", partial: false, databases: [] },
            setup: {
              isDefault: false,
              instance: "present",
              transferableDatabase: "present",
              canConnectInstance: true,
              canTransferDatabases: true,
              canCreateDatabase: true,
            },
          })
        );
        callbacks.onText("Transfer an existing database.", {
          kind: "transfer-project-databases",
          project: "projects/orders",
        });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "How do I connect a database to this project?",
      page,
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .some((message) => message.content === "Transfer an existing database.")
        ).toBe(true)
      );
    });
    const answer = useAgentStore
      .getState()
      .getMessages(chat.id)
      .find((message) => message.content === "Transfer an existing database.");
    expect(answer?.metadata?.handoff).toEqual({
      kind: "transfer-project-databases",
      project: "projects/orders",
    });
    expect(answer?.metadata?.actions).toEqual([
      { kind: "continue", prompt: "What information do I need to connect a new instance?" },
      { kind: "continue", prompt: "Which databases can I transfer?" },
      { kind: "continue", prompt: "How do I verify the transfer?" },
    ]);
    expect(mocks.aiChat).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(7);
    view.unmount();
  });

  test("generates follow-up questions for a successful answer without a primary action", async () => {
    mocks.aiChat.mockResolvedValue({
      content: JSON.stringify([
        "Which change should I review first?",
        "How can I check the rollout status?",
      ]),
      toolCalls: [],
      stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
      usage: { totalTokens: 5 },
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("Two changes need your review.");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({ chatId: chat.id, text: "What needs my attention?", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore.getState().getMessages(chat.id).find((message) =>
            message.content === "Two changes need your review."
          )?.metadata?.actions
        ).toHaveLength(2)
      );
    });
    expect(mocks.aiChat).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(5);
    view.unmount();
  });

  test("keeps a completed answer when optional follow-up generation is canceled", async () => {
    let resolveFollowUp!: (value: unknown) => void;
    mocks.aiChat.mockImplementation(
      () => new Promise((resolve) => (resolveFollowUp = resolve))
    );
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            personalIssues: {
              status: "available",
              issues: [
                { name: "projects/orders/issues/42", relation: "review" },
              ],
            },
          })
        );
        callbacks.onText("Review this issue.", {
          kind: "review-project-issues",
          project: "projects/orders",
        });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore
      .getState()
      .createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "What should I review?",
      page,
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.aiChat).toHaveBeenCalledOnce());
    });

    const completedAnswer = useAgentStore
      .getState()
      .getMessages(chat.id)
      .find((message) => message.content === "Review this issue.");
    expect(completedAnswer?.metadata?.handoff).toEqual({
      kind: "review-project-issues",
      project: "projects/orders",
    });
    expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("idle");

    act(() => useAgentStore.getState().cancel(chat.id, "user"));
    await act(async () => {
      resolveFollowUp({
        content: '["What should I check on the issue?"]',
        toolCalls: [],
        stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
      });
    });

    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.id === completedAnswer?.id)?.metadata?.actions
    ).toBeUndefined();
    expect(useAgentStore.getState().getChat(chat.id)?.interrupted).toBe(false);
    view.unmount();
  });

  test("counts a late suggestion call without attaching it to an older turn", async () => {
    let resolveFollowUp!: (value: unknown) => void;
    mocks.aiChat.mockImplementation(
      () => new Promise((resolve) => (resolveFollowUp = resolve))
    );
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            personalIssues: {
              status: "available",
              issues: [
                { name: "projects/orders/issues/42", relation: "review" },
              ],
            },
          })
        );
        callbacks.onText("Review this issue.", {
          kind: "review-project-issues",
          project: "projects/orders",
        });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      })
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("Second answer.");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore
      .getState()
      .createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "What should I review?",
      page,
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.aiChat).toHaveBeenCalledOnce());
    });

    act(() => {
      useAgentStore.getState().queueUserMessage({
        chatId: chat.id,
        text: "What changed?",
        page,
      });
    });
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    await act(async () => {
      resolveFollowUp({
        content: '["What else should I review?"]',
        toolCalls: [],
        stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
        usage: { totalTokens: 7 },
      });
    });

    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(7);
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Review this issue.")?.metadata
        ?.actions
    ).toBeUndefined();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .some((message) => message.content === "Second answer.")
    ).toBe(true);
    view.unmount();
  });

  test("tops up an existing question when Open matches current-turn evidence", async () => {
    mocks.aiChat.mockResolvedValue({
      content: JSON.stringify([
        "What should I configure first?",
        "How can I test the connection?",
        "How do I assign a database afterward?",
      ]),
      toolCalls: [],
      stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
    });
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({ project: "projects/orders" }),
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            databases: { status: "available", partial: false, databases: [] },
            setup: {
              isDefault: false,
              instance: "empty",
              transferableDatabase: "empty",
              canConnectInstance: true,
              canTransferDatabases: true,
              canCreateDatabase: true,
            },
          })
        );
        callbacks.onText("Set up an instance.");
        return {
          kind: "completed",
          text: "Set up an instance.",
          success: true,
          actions: [
            {
              kind: "open",
              target: {
                kind: "connect-project-instance",
                project: "projects/orders",
              },
            },
            { kind: "continue", prompt: "What should I configure first?" },
          ],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@Orders what should I set up next?",
      page,
      resourceReference: {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Set up an instance.")?.metadata
        ?.actions
    ).toEqual([
      {
        kind: "open",
        target: {
          kind: "connect-project-instance",
          project: "projects/orders",
        },
      },
      { kind: "continue", prompt: "What should I configure first?" },
      { kind: "continue", prompt: "How can I test the connection?" },
      { kind: "continue", prompt: "How do I assign a database afterward?" },
    ]);
    expect(mocks.aiChat).toHaveBeenCalledOnce();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Set up an instance.")?.metadata
        ?.resourceReferences
    ).toEqual([
      {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      },
    ]);
    view.unmount();
  });

  test("attaches a Plan draft for verified DDL even when the Agent omits its action", async () => {
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const answer = "Proposed change:\n\n```sql\nCREATE TABLE audit_log (id INT PRIMARY KEY);\n```";
    mocks.executeTool.mockImplementation(async (name) => ({
      kind: "tool_result",
      result: JSON.stringify(
        name === "call_api"
          ? { status: 200, response: { schema: "CREATE TABLE orders (id INT);" } }
          : {
              project,
              database: { status: "available", name: database, state: "ACTIVE" },
              evidence: { project, references: [{ kind: "database", name: database }] },
            }
      ),
    }));
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, executor, callbacks) => {
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/GetDatabaseSchema",
          body: { name: `${database}/schema` },
        },
        "schema-1"
      );
      callbacks.onText(answer);
      return {
        kind: "completed",
        text: answer,
        success: true,
        actions: [],
        totalTokensUsed: 0,
      };
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@orders Create an audit table",
      page,
      resourceReference: {
        kind: "database",
        name: database,
        project,
        label: "orders",
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_database_overview",
      { databaseName: database },
      expect.any(String)
    );
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "call_api",
      {
        operationId: "DatabaseService/GetDatabaseSchema",
        body: { name: `${database}/schema` },
      },
      "schema-1"
    );
    await act(async () => {
      await vi.waitFor(() => {
        expect(
          useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)?.metadata?.actions
        ).toEqual([{ kind: "draft-plan", project, database }]);
      });
    });
    view.unmount();
  });

  test("keeps the Plan draft action after a schema read and clarification reply", async () => {
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const answer = "Proposed change:\n\n```sql\nCREATE TABLE audit_log (id INT PRIMARY KEY);\n```";
    mocks.executeTool.mockImplementation(async (name) => ({
      kind: "tool_result",
      result: JSON.stringify(
        name === "call_api"
          ? { status: 200, response: { schema: "CREATE TABLE orders (id INT);" } }
          : {
              project,
              database: { status: "available", name: database, state: "ACTIVE" },
            }
      ),
    }));
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, executor, callbacks) => {
        const args = {
          operationId: "DatabaseService/GetDatabaseSchema",
          body: { name: `${database}/schema` },
        };
        callbacks.onAssistantMessage({
          role: "assistant",
          toolCalls: [
            { id: "schema-1", name: "call_api", arguments: JSON.stringify(args) },
          ],
        });
        const result = await executor("call_api", args, "schema-1");
        callbacks.onToolResult("schema-1", result.result);
        return {
          kind: "awaiting_user",
          ask: { toolCallId: "ask-1", prompt: "What should the table be named?", kind: "input" },
          totalTokensUsed: 0,
        };
      })
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText(answer);
        return {
          kind: "completed",
          text: answer,
          success: true,
          actions: [{ kind: "draft-plan", project, database }],
          totalTokensUsed: 0,
        };
      });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@orders Create an audit table",
      page,
      resourceReference: { kind: "database", name: database, project, label: "orders" },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("awaiting_user")
      );
    });
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "audit_log");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(
      useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)
        ?.metadata?.actions
    ).toEqual([{ kind: "draft-plan", project, database }]);
    view.unmount();
  });

  test("does not restore Plan draft evidence after an intervening write", async () => {
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const answer = "```sql\nALTER TABLE orders ADD COLUMN note TEXT;\n```";
    mocks.executeTool.mockImplementation(async (name) => ({
      kind: "tool_result",
      result: JSON.stringify(
        name === "call_api"
          ? { status: 200, response: {} }
          : {
              project,
              database: { status: "available", name: database, state: "ACTIVE" },
            }
      ),
    }));
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, executor, callbacks) => {
        for (const [id, args] of [
          ["schema-1", {
            operationId: "DatabaseService/GetDatabaseSchema",
            body: { name: `${database}/schema` },
          }],
          ["write-1", { operationId: "PlanService/CreatePlan", body: {} }],
        ] as const) {
          callbacks.onAssistantMessage({
            role: "assistant",
            toolCalls: [{ id, name: "call_api", arguments: JSON.stringify(args) }],
          });
          const result = await executor("call_api", args, id);
          callbacks.onToolResult(id, result.result);
        }
        return {
          kind: "awaiting_user",
          ask: { toolCallId: "ask-1", prompt: "Continue?", kind: "input" },
          totalTokensUsed: 0,
        };
      })
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText(answer);
        return {
          kind: "completed",
          text: answer,
          success: true,
          actions: [{ kind: "draft-plan", project, database }],
          totalTokensUsed: 0,
        };
      });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@orders Add a note column",
      page,
      resourceReference: { kind: "database", name: database, project, label: "orders" },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("awaiting_user")
      );
    });
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Yes");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(
      useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)
        ?.metadata?.actions
    ).toBeUndefined();
    view.unmount();
  });

  test("does not offer a Plan draft when schema retrieval fails", async () => {
    mocks.aiChat.mockResolvedValue({
      content: "[]",
      toolCalls: [],
      stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
    });
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const answer = "```sql\nALTER TABLE orders ADD COLUMN note TEXT;\n```";
    mocks.executeTool.mockImplementation(async (name) => ({
      kind: "tool_result",
      result: JSON.stringify(
        name === "call_api"
          ? { status: 403, error: "permission denied" }
          : {
              project,
              database: { status: "available", name: database, state: "ACTIVE" },
            }
      ),
    }));
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, executor, callbacks) => {
      await executor(
        "call_api",
        {
          operationId: "DatabaseService/GetDatabaseSchema",
          body: { name: `${database}/schema` },
        },
        "schema-1"
      );
      callbacks.onText(answer);
      return {
        kind: "completed",
        text: answer,
        success: true,
        actions: [{ kind: "draft-plan", project, database }],
        totalTokensUsed: 0,
      };
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@orders Add note",
      page,
      resourceReference: { kind: "database", name: database, project, label: "orders" },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)?.metadata?.actions
    ).toBeUndefined();
    expect(mocks.aiChat).toHaveBeenCalledOnce();
    view.unmount();
  });

  test("drops the Plan draft action after a write invalidates database evidence", async () => {
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const answer = "```sql\nALTER TABLE orders ADD COLUMN audit_id INT;\n```";
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({
        project,
        database: { status: "available", name: database, state: "ACTIVE" },
      }),
    });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, executor, callbacks) => {
      await executor("call_api", { operationId: "PlanService/CreatePlan" }, "write-1");
      callbacks.onText(answer);
      return {
        kind: "completed",
        text: answer,
        success: true,
        actions: [{ kind: "draft-plan", project, database }],
        totalTokensUsed: 0,
      };
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@orders Add audit_id",
      page,
      resourceReference: {
        kind: "database",
        name: database,
        project,
        label: "orders",
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => {
        expect(
          useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)
        ).toBeDefined();
      });
    });
    expect(
      useAgentStore.getState().getMessages(chat.id).find((message) => message.content === answer)?.metadata?.actions
    ).toBeUndefined();
    view.unmount();
  });

  test("does not attach response actions to a failed answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("The request failed.");
        return {
          kind: "completed",
          text: "The request failed.",
          success: false,
          actions: [
            { kind: "continue", prompt: "Try the request again?" },
          ],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Make a change",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The request failed.")?.metadata
        ?.actions
    ).toBeUndefined();
    view.unmount();
  });

  test("offers an Access Grant draft only after one actual denied Query", async () => {
    const args = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/prod/databases/app",
        statement: "SELECT * FROM orders",
      },
    };
    const result = JSON.stringify({
      status: 200,
      response: {
        results: [
          {
            error: "Permission denied",
            permissionDenied: {
              requiredPermissions: ["bb.sql.select"],
              resources: ["instances/prod/databases/app"],
            },
          },
        ],
      },
    });
    const draft = {
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
    };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result });
    mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, executeTool, callbacks) => {
        callbacks.onAssistantMessage({
          role: "assistant",
          toolCalls: [
            {
              id: "query-1",
              name: "call_api",
              arguments: JSON.stringify(args),
            },
          ],
        });
        const execution = await executeTool("call_api", args, "query-1");
        if (execution.kind === "tool_result") {
          callbacks.onToolResult("query-1", execution.result);
        }
        callbacks.onText("The API denied the query.");
        return {
          kind: "completed",
          success: false,
          actions: [],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Query the orders table",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .find((message) => message.content === "The API denied the query.")
        ).toBeDefined()
      );
    });
    expect(mocks.resolveDeniedQueryAccessDraft).toHaveBeenCalledWith(
      args,
      result
    );
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The API denied the query.")
        ?.metadata?.accessRequestDraft
    ).toEqual(draft);
    view.unmount();
  });

  test("offers an Access Grant draft when an approved Query is denied", async () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const args = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/prod/databases/app",
        statement: "SELECT * FROM orders",
      },
    };
    const result = JSON.stringify({
      status: 200,
      response: {
        results: [
          {
            error: "Permission denied",
            permissionDenied: {
              requiredPermissions: ["bb.sql.select"],
              resources: [
                "instances/prod/databases/app/schemas/public/tables/orders",
              ],
            },
          },
        ],
      },
    });
    const draft = {
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
    };
    store.addMessage({ chatId, role: "user", content: "Query orders" });
    store.addMessage({
      chatId,
      role: "assistant",
      toolCalls: [
        { id: "query-1", name: "call_api", arguments: JSON.stringify(args) },
      ],
    });
    useAgentStore.setState({
      chats: store.chats.map((chat) =>
        chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
      ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "query-1",
          prompt: "SQLService/Query",
          kind: "confirm",
          apiPermission: {
            actionId: "action-1",
            operationId: "SQLService/Query",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "SQLService/Query",
      body: JSON.stringify(args.body),
      expiresAt: Date.now() + 60_000,
    });
    mocks.resolvePendingApiCall.mockResolvedValue(result);
    mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executeTool, callbacks) => {
      callbacks.onText("The API denied the query.");
      return {
        kind: "completed",
        success: false,
        actions: [],
        totalTokensUsed: 0,
      };
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      const allow = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-allow"
      );
      allow?.click();
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chatId)
            .find((message) => message.content === "The API denied the query.")
        ).toBeDefined()
      );
    });

    expect(
      useAgentStore
        .getState()
        .getMessages(chatId)
        .find((message) => message.content === "The API denied the query.")
        ?.metadata?.accessRequestDraft
    ).toEqual(draft);
    expect(mocks.resolveDeniedQueryAccessDraft).toHaveBeenCalledWith(
      args,
      result
    );
    view.unmount();
  });

  test("does not offer access when another Query was attempted in the turn", async () => {
    const args = {
      operationId: "SQLService/Query",
      body: {
        name: "instances/prod/databases/app",
        statement: "SELECT * FROM orders",
      },
    };
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: '{"status":200,"response":{"results":[]}}',
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, executeTool, callbacks) => {
        for (const id of ["query-1", "query-2"]) {
          callbacks.onAssistantMessage({
            role: "assistant",
            toolCalls: [
              { id, name: "call_api", arguments: JSON.stringify(args) },
            ],
          });
          const execution = await executeTool("call_api", args, id);
          if (execution.kind === "tool_result") {
            callbacks.onToolResult(id, execution.result);
          }
        }
        callbacks.onText("The query failed.");
        return {
          kind: "completed",
          success: false,
          actions: [],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Query orders",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .find((message) => message.content === "The query failed.")
        ).toBeDefined()
      );
    });
    expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The query failed.")
        ?.metadata?.accessRequestDraft
    ).toBeUndefined();
    view.unmount();
  });

  test("does not attach a setup handoff from before a mutating API call", async () => {
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: '{"status":200}',
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, executeTool, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            databases: { status: "available", partial: false, databases: [] },
            setup: {
              isDefault: false,
              instance: "empty",
              transferableDatabase: "empty",
              canConnectInstance: true,
              canTransferDatabases: true,
              canCreateDatabase: true,
            },
          })
        );
        await executeTool(
          "call_api",
          { operationId: "DatabaseService/UpdateDatabase", body: {} },
          "write-1"
        );
        callbacks.onText("The database is ready.", {
          kind: "connect-project-instance",
          project: "projects/orders",
        });
        return {
          kind: "completed",
          success: true,
          actions: [
            {
              kind: "open",
              target: {
                kind: "connect-project-instance",
                project: "projects/orders",
              },
            },
          ],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Update the database",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    const metadata = useAgentStore
      .getState()
      .getMessages(chat.id)
      .find((message) => message.content === "The database is ready.")
      ?.metadata;
    expect(metadata?.handoff).toBeUndefined();
    expect(metadata?.actions).toBeUndefined();
    view.unmount();
  });

  test("refetches a referenced project overview after a mutating API call", async () => {
    const project = "projects/orders";
    const staleOverview = JSON.stringify({
      project,
      databases: { status: "available", partial: false, databases: [] },
      setup: {
        isDefault: false,
        instance: "empty",
        transferableDatabase: "empty",
        canConnectInstance: true,
        canTransferDatabases: true,
        canCreateDatabase: true,
      },
    });
    const freshOverview = JSON.stringify({
      project,
      databases: {
        status: "available",
        partial: false,
        databases: [{ name: "instances/prod/databases/orders" }],
      },
    });
    mocks.executeTool
      .mockResolvedValueOnce({ kind: "tool_result", result: staleOverview })
      .mockResolvedValueOnce({ kind: "tool_result", result: '{"status":200}' })
      .mockResolvedValueOnce({ kind: "tool_result", result: freshOverview });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, executeTool, callbacks) => {
        await executeTool(
          "call_api",
          { operationId: "DatabaseService/UpdateDatabase", body: {} },
          "write-1"
        );
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-2", name: "get_project_overview", arguments: "{}" },
          ],
        });
        const overview = await executeTool("get_project_overview", {}, "overview-2");
        callbacks.onToolResult("overview-2", overview.result);
        callbacks.onText("The database is ready.", {
          kind: "connect-project-instance",
          project,
        });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: project, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "@Orders update the database",
      page,
      resourceReference: {
        kind: "project",
        name: project,
        project,
        label: "Orders",
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(mocks.executeTool.mock.calls.map(([name]) => name)).toEqual([
      "get_project_overview",
      "call_api",
      "get_project_overview",
    ]);
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The database is ready.")
        ?.metadata?.handoff
    ).toBeUndefined();
    view.unmount();
  });

  test.each([false, true])(
    "drops pre-mutation database evidence and repopulates only from a later read: %s",
    async (readAgain) => {
      const project = "projects/orders";
      const databaseName = "instances/prod/databases/orders";
      const staleEvidence = {
        project,
        references: [{ kind: "database", name: databaseName, title: "Before update" }],
      };
      const freshEvidence = {
        project,
        references: [{ kind: "database", name: databaseName, title: "After update" }],
      };
      const overview = (evidence: typeof staleEvidence) =>
        JSON.stringify({
          project,
          database: { status: "available", name: databaseName, state: "ACTIVE" },
          evidence,
        });
      mocks.executeTool
        .mockResolvedValueOnce({ kind: "tool_result", result: overview(staleEvidence) })
        .mockResolvedValueOnce({ kind: "tool_result", result: '{"status":200}' });
      if (readAgain) {
        mocks.executeTool.mockResolvedValueOnce({
          kind: "tool_result",
          result: overview(freshEvidence),
        });
      }
      mocks.runAgentLoop.mockImplementation(
        async (_history, _tools, executeTool, callbacks) => {
          await executeTool(
            "call_api",
            { operationId: "DatabaseService/UpdateDatabase", body: {} },
            "write-1"
          );
          if (readAgain) {
            callbacks.onAssistantMessage({
              toolCalls: [
                {
                  id: "overview-2",
                  name: "get_database_overview",
                  arguments: JSON.stringify({ databaseName }),
                },
              ],
            });
            const result = await executeTool(
              "get_database_overview",
              { databaseName },
              "overview-2"
            );
            callbacks.onToolResult("overview-2", result.result);
          }
          callbacks.onText("The database has been updated.");
          return { kind: "completed", success: true, totalTokensUsed: 0 };
        }
      );
      const page = { path: "/projects/orders/databases", title: "Databases" };
      const chat = useAgentStore.getState().createChat({ context: project, page });
      useAgentStore.getState().queueUserMessage({
        chatId: chat.id,
        text: "@orders Update the database",
        page,
        resourceReference: {
          kind: "database",
          name: databaseName,
          project,
          label: "orders",
        },
      });

      const view = renderIntoContainer(<AgentInput />);
      view.render();
      await act(async () => {
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      expect(
        useAgentStore
          .getState()
          .getMessages(chat.id)
          .find((message) => message.content === "The database has been updated.")
          ?.metadata?.evidence
      ).toEqual(readAgain ? freshEvidence : undefined);
      view.unmount();
    }
  );

  test("does not attach an unproposed setup handoff to a completed answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/new-project-8i6r",
            personalIssues: { status: "available", partial: false, issues: [] },
            databases: { status: "available", partial: false, databases: [] },
            setup: {
              isDefault: false,
              instance: "empty",
              transferableDatabase: "empty",
              canConnectInstance: true,
              canTransferDatabases: true,
              canCreateDatabase: true,
            },
            openDatabaseChanges: {
              status: "available",
              partial: false,
              issues: [],
            },
          })
        );
        callbacks.onText("Connect an instance, then add its databases.");
        return { kind: "completed", text: "Connect an instance, then add its databases.", success: true, totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/new-project-8i6r/databases", title: "Databases" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/new-project-8i6r",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "What should I set up next in this project?",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Connect an instance, then add its databases.")
        ?.metadata?.handoff
    ).toBeUndefined();
    view.unmount();
  });

  test.each([
    {
      name: "an unusable model handoff",
      success: true,
      proposedHandoff: {
        kind: "review-project-issues" as const,
        project: "projects/orders",
      },
      expectedHandoff: undefined,
    },
    {
      name: "a matching model handoff",
      success: true,
      proposedHandoff: {
        kind: "connect-project-instance" as const,
        project: "projects/orders",
      },
      expectedHandoff: {
        kind: "connect-project-instance",
        project: "projects/orders",
      },
    },
    {
      name: "a failed answer",
      success: false,
      proposedHandoff: {
        kind: "connect-project-instance" as const,
        project: "projects/orders",
      },
      expectedHandoff: undefined,
    },
  ])("validates the setup handoff for $name", async ({ success, proposedHandoff, expectedHandoff }) => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "overview-1", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "overview-1",
          JSON.stringify({
            project: "projects/orders",
            databases: { status: "available", partial: false, databases: [] },
            setup: {
              isDefault: false,
              instance: "empty",
              transferableDatabase: "empty",
              canConnectInstance: true,
              canTransferDatabases: true,
              canCreateDatabase: true,
            },
          })
        );
        callbacks.onText("Set up an instance.", proposedHandoff);
        return {
          kind: "completed",
          text: "Set up an instance.",
          success,
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({ context: "projects/orders", page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "What should I set up next?",
      page,
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Set up an instance.")?.metadata
        ?.handoff
    ).toEqual(expectedHandoff);
    view.unmount();
  });

  test("attaches Issue overview evidence to the final answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "tool-issue-overview",
              name: "get_issue_overview",
              arguments: '{"issueName":"projects/orders/issues/42"}',
            },
          ],
        });
        callbacks.onToolResult(
          "tool-issue-overview",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "issue",
                  name: "projects/orders/issues/42",
                  title: "Add customer table",
                },
              ],
            },
          })
        );
        callbacks.onText("The Issue is awaiting approval.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Why is this Issue waiting?",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(mocks.extractIssueOverviewEvidence).toHaveBeenCalledOnce();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The Issue is awaiting approval.")
        ?.metadata?.evidence
    ).toEqual({
      project: "projects/orders",
      references: [
        {
          kind: "issue",
          name: "projects/orders/issues/42",
          title: "Add customer table",
        },
      ],
    });
    view.unmount();
  });

  test("accepts an exact failed Task link from the selected Issue overview", async () => {
    const issueName = "projects/orders/issues/42";
    const task =
      "projects/orders/plans/plan-1/rollout/stages/prod/tasks/task-1";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        issue: { status: "available", name: issueName },
        plan: {
          status: "available",
          name: "projects/orders/plans/plan-1",
          reviewOnPlan: true,
        },
        rollout: {
          status: "available",
          name: "projects/orders/plans/plan-1/rollout",
          failedTasks: [task],
          failedTasksPartial: false,
        },
      }),
    });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onText("The production task failed.");
      return {
        kind: "completed",
        text: "The production task failed.",
        success: true,
        actions: [
          { kind: "open", target: { kind: "view-failed-task", task } },
          { kind: "continue", prompt: "What caused the failure?" },
        ],
        totalTokensUsed: 0,
      };
    });
    const page = { path: "/projects/orders/issues/42", title: "Issue" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "@Change why did this rollout fail?",
      page,
      resourceReference: {
        kind: "issue",
        name: issueName,
        project: "projects/orders",
        label: "Change",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      store.getMessages(chat.id).find((message) => message.content === "The production task failed.")
        ?.metadata?.actions
    ).toContainEqual({ kind: "open", target: { kind: "view-failed-task", task } });
    view.unmount();
  });

  test("attaches an unsent review comment only for the selected Issue", async () => {
    const issueName = "projects/orders/issues/42";
    const action = {
      kind: "draft-issue-comment",
      issue: issueName,
      comment: "Please verify this column is masked before rollout.",
    };
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        issue: { status: "available", name: issueName },
      }),
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("I found a masking concern.");
        return {
          kind: "completed",
          success: true,
          actions: [action],
          totalTokensUsed: 0,
        };
      }
    );
    const page = { path: "/projects/orders/issues/42", title: "Issue" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "@Change any security concern?",
      page,
      resourceReference: {
        kind: "issue",
        name: issueName,
        project: "projects/orders",
        label: "Change",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(
          store
            .getMessages(chat.id)
            .find((message) => message.content === "I found a masking concern.")
        ).toBeDefined()
      );
    });
    expect(
      store
        .getMessages(chat.id)
        .find((message) => message.content === "I found a masking concern.")
        ?.metadata?.actions
    ).toContainEqual(action);
    view.unmount();
  });

  test("attaches database overview evidence to the final answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "tool-database-overview",
              name: "get_database_overview",
              arguments:
                '{"databaseName":"projects/orders/instances/prod/databases/application"}',
            },
          ],
        });
        callbacks.onToolResult(
          "tool-database-overview",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "database",
                  name: "projects/orders/instances/prod/databases/application",
                },
              ],
            },
          })
        );
        callbacks.onText("The database is synchronized.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Is the application database healthy?",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(mocks.extractDatabaseOverviewEvidence).toHaveBeenCalledOnce();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find(
          (message) => message.content === "The database is synchronized."
        )?.metadata?.evidence
    ).toEqual({
      project: "projects/orders",
      references: [
        {
          kind: "database",
          name: "projects/orders/instances/prod/databases/application",
        },
      ],
    });
    view.unmount();
  });

  test("retains deduplicated references from multiple overviews in one answer", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "project", name: "get_project_overview", arguments: "{}" },
            { id: "issue", name: "get_issue_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "project",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "database",
                  name: "projects/orders/instances/prod/databases/orders",
                },
                {
                  kind: "issue",
                  name: "projects/orders/issues/42",
                  title: "Add customer table",
                },
              ],
            },
          })
        );
        callbacks.onToolResult(
          "issue",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "issue",
                  name: "projects/orders/issues/42",
                  title: "Add customer table",
                },
                {
                  kind: "issue",
                  name: "projects/orders/issues/43",
                  title: "Add invoice table",
                },
              ],
            },
          })
        );
        callbacks.onText("Two changes are in progress.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "Summarize this project and its issues",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Two changes are in progress.")
        ?.metadata?.evidence
    ).toEqual({
      project: "projects/orders",
      references: [
        {
          kind: "database",
          name: "projects/orders/instances/prod/databases/orders",
        },
        {
          kind: "issue",
          name: "projects/orders/issues/42",
          title: "Add customer table",
        },
        {
          kind: "issue",
          name: "projects/orders/issues/43",
          title: "Add invoice table",
        },
      ],
    });
    view.unmount();
  });

  test("keeps distinct project and workspace evidence scopes separate", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            { id: "workspace", name: "get_workspace_overview", arguments: "{}" },
            { id: "project", name: "get_project_overview", arguments: "{}" },
          ],
        });
        callbacks.onToolResult(
          "workspace",
          JSON.stringify({
            evidence: {
              workspace: "workspaces/demo",
              references: [
                {
                  kind: "issue",
                  name: "projects/billing/issues/7",
                  title: "Billing approval",
                },
              ],
            },
          })
        );
        callbacks.onToolResult(
          "project",
          JSON.stringify({
            evidence: {
              project: "projects/orders",
              references: [
                {
                  kind: "database",
                  name: "projects/orders/instances/prod/databases/orders",
                },
              ],
            },
          })
        );
        callbacks.onText("Recent activity spans the workspace and project.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const page = { path: "/landing", title: "Workspace" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "Summarize recent work",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find(
          (message) =>
            message.content === "Recent activity spans the workspace and project."
        )?.metadata?.evidence
    ).toEqual([
      {
        workspace: "workspaces/demo",
        references: [
          {
            kind: "issue",
            name: "projects/billing/issues/7",
            title: "Billing approval",
          },
        ],
      },
      {
        project: "projects/orders",
        references: [
          {
            kind: "database",
            name: "projects/orders/instances/prod/databases/orders",
          },
        ],
      },
    ]);
    view.unmount();
  });

  test("does not special-case a typed project readiness question", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.landing-starters.project-next"
          ? "What can I do next?"
          : key,
      i18n: { language: "en-US" },
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "What can I do next?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });

    await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    expect(mocks.getToolDefinitions).toHaveBeenCalledWith({
      runMode: "default",
    });
    view.unmount();
  });

  test("runs a typed project recent-activity prompt directly in a project chat", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.landing-starters.project-open-recent"
          ? "What changed recently in this project?"
          : key,
      i18n: { language: "en-US" },
    });
    mocks.getRecentActivity.mockResolvedValue(
      JSON.stringify({ events: [], partial: false })
    );
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "What changed recently in this project?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });

    await vi.waitFor(() =>
      expect(mocks.getRecentActivity).toHaveBeenCalledWith("projects/orders")
    );
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("runs a typed workspace recent-activity prompt directly in a workspace chat", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.landing-starters.workspace-recent"
          ? "What changed recently in this workspace?"
          : key,
      i18n: { language: "en-US" },
    });
    mocks.currentContext = "workspaces/demo";
    mocks.getRecentActivity.mockResolvedValue(
      JSON.stringify({ events: [], partial: false })
    );
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "What changed recently in this workspace?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });

    await vi.waitFor(() =>
      expect(mocks.getRecentActivity).toHaveBeenCalledWith("projects/-")
    );
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("runs a typed workspace attention prompt directly in a workspace chat", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.landing-starters.workspace-attention"
          ? "What needs my attention?"
          : key,
      i18n: { language: "en-US" },
    });
    mocks.currentContext = "workspaces/demo";
    mocks.getWorkspaceAttention.mockResolvedValue(
      JSON.stringify({ issues: [], partial: false })
    );
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "What needs my attention?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });

    await vi.waitFor(() =>
      expect(mocks.getWorkspaceAttention).toHaveBeenCalledOnce()
    );
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("runs a typed project attention prompt directly in a project chat", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.landing-starters.project-open-blockers"
          ? "What needs my attention in this project?"
          : key,
      i18n: { language: "en-US" },
    });
    mocks.getProjectAttention.mockResolvedValue(
      JSON.stringify({ issues: [], partial: false })
    );
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "What needs my attention in this project?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });

    await vi.waitFor(() =>
      expect(mocks.getProjectAttention).toHaveBeenCalledOnce()
    );
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("finishes the workspace recent starter when its audit read fails", async () => {
    mocks.getRecentActivity.mockRejectedValue(new Error("network unavailable"));
    const page = { path: "/landing", title: "Workspace" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What changed recently in this workspace?",
      page,
      runMode: "read-only",
      starterId: "workspace-recent",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .some(
              (message) => message.content === "agent.recent-activity.unavailable"
            )
        ).toBe(true)
      );
    });

    expect(useAgentStore.getState().getMessages(chat.id)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: "agent.recent-activity.unavailable",
        }),
      ])
    );
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .some((message) => message.metadata?.pending)
    ).toBe(false);
    expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("idle");
    view.unmount();
  });

  test("runs the workspace attention starter without an activity lookup", async () => {
    mocks.getWorkspaceAttention.mockResolvedValue(
      JSON.stringify({
        issues: [
          {
            name: "projects/orders/issues/42",
            title: "Add customer table",
            relation: "review",
          },
        ],
        partial: false,
      })
    );
    const page = { path: "/landing", title: "Workspace" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What needs my attention?",
      page,
      runMode: "read-only",
      starterId: "workspace-attention",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(mocks.getWorkspaceAttention).toHaveBeenCalledOnce()
      );
    });
    expect(mocks.getRecentActivity).not.toHaveBeenCalled();
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getMessages(chat.id)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: expect.stringContaining("agent.workspace-attention.issue"),
          metadata: expect.objectContaining({
            handoff: {
              kind: "review-workspace-issues",
              workspace: "workspaces/demo",
            },
          }),
        }),
      ])
    );
    view.unmount();
  });

  test("does not call a partial workspace attention result empty", async () => {
    mocks.getWorkspaceAttention.mockResolvedValue(
      JSON.stringify({ issues: [], partial: true })
    );
    const page = { path: "/landing", title: "Workspace" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What needs my attention?",
      page,
      runMode: "read-only",
      starterId: "workspace-attention",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .some(
              (message) =>
                message.content === "agent.workspace-attention.partial"
            )
        ).toBe(true)
      );
    });

    view.unmount();
  });

  test("does not call a partial project attention result empty", async () => {
    mocks.getProjectAttention.mockResolvedValue(
      JSON.stringify({ issues: [], partial: true })
    );
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What needs my attention in this project?",
      page,
      runMode: "read-only",
      starterId: "project-open-blockers",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore
            .getState()
            .getMessages(chat.id)
            .some(
              (message) => message.content === "agent.project-attention.partial"
            )
        ).toBe(true)
      );
    });

    view.unmount();
  });

  test("returns to the default tool mode for a typed follow-up", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What is this Project ready for?",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Show me the databases");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });

    expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(
      expect.objectContaining({ runMode: "default" })
    );
    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "default",
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ runMode: "default" })
    );
    view.unmount();
  });

  test("keeps a read-only starter read-only after answering a pending question", async () => {
    mocks.runAgentLoop
      .mockResolvedValueOnce({
        kind: "awaiting_user",
        ask: {
          toolCallId: "ask-1",
          prompt: "Which database?",
          kind: "input",
        },
      })
      .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What should I review?",
      page,
      runMode: "read-only",
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe(
          "awaiting_user"
        )
      );
    });

    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Billing database");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });

    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "read-only",
    });
    view.unmount();
  });

  test("keeps a read-only starter read-only when retrying an interrupted run", async () => {
    mocks.runAgentLoop
      .mockResolvedValueOnce({ kind: "aborted" })
      .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What should I review?",
      page,
      runMode: "read-only",
    });

    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.interrupted).toBe(true)
      );
    });
    await act(async () => {
      retryActionRef.current?.();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });

    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "read-only",
    });
    view.unmount();
  });

  test("does not retry a chat after the account changes during configuration check", async () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.addMessage({ chatId, role: "user", content: "Original question" });
    store.interruptChatRun(chatId);

    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();

    let finishConfigurationCheck!: (setting: undefined) => void;
    mocks.getOrFetchSettingByName.mockImplementationOnce(
      () =>
        new Promise<undefined>((resolve) => {
          finishConfigurationCheck = resolve;
        })
    );
    act(() => retryActionRef.current?.());
    expect(finishConfigurationCheck).toBeDefined();

    act(() => {
      store.setWorkspaceName("workspaces/demo", "other@example.com");
    });
    await act(async () => finishConfigurationCheck(undefined));

    const current = useAgentStore.getState();
    expect(current.userEmail).toBe("other@example.com");
    expect(current.getMessages(current.currentChatId)).toHaveLength(0);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("does not leave composer actions when the user stops a running request", () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.startChatRun(chatId, { path: "/demo", title: "Demo" });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    act(() => {
      [...view.container.querySelectorAll("button")]
        .find((button) => button.textContent === "agent.stop")
        ?.click();
    });

    expect(view.container.textContent).not.toContain("agent.request-stopped-by-user");
    expect(view.container.textContent).not.toContain("agent.retry-last-chat-turn");
    expect(view.container.textContent).not.toContain("common.dismiss");
    view.unmount();
  });

  test("moves unexpected interruption feedback out of the composer", () => {
    const store = useAgentStore.getState();
    store.interruptChatRun(store.currentChatId!);

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    expect(view.container.textContent).not.toContain("agent.interrupted-retry-hint");
    expect(view.container.textContent).not.toContain("agent.retry-last-chat-turn");
    expect(view.container.textContent).not.toContain("common.dismiss");
    view.unmount();
  });

  test("keeps the queued message busy while AI configuration loads", async () => {
    let resolveSetting!: (value: undefined) => void;
    mocks.getOrFetchSettingByName.mockReturnValue(
      new Promise<undefined>((resolve) => { resolveSetting = resolve; })
    );
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: chat.id, text: "List billing databases", page });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => { await Promise.resolve(); });

    expect(mocks.getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.AI,
      true
    );
    expect(getQueuedUserMessage(chat.id)).toBeDefined();
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();

    await act(async () => { resolveSetting(undefined); });
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(getQueuedUserMessage(chat.id)).toBeUndefined();
    view.unmount();
  });

  test("does not send a queued message after the account changes during configuration check", async () => {
    let resolveSetting!: (value: undefined) => void;
    mocks.getOrFetchSettingByName.mockReturnValue(
      new Promise<undefined>((resolve) => {
        resolveSetting = resolve;
      })
    );
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: chat.id, text: "Private question", page });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => Promise.resolve());
    expect(getQueuedUserMessage(chat.id)).toBeDefined();

    act(() => {
      store.setWorkspaceName("workspaces/demo", "other@example.com");
    });
    await act(async () => resolveSetting(undefined));

    const current = useAgentStore.getState();
    expect(current.userEmail).toBe("other@example.com");
    expect(current.getMessages(current.currentChatId)).toHaveLength(0);
    expect(current.queuedUserMessagesByChatId).toEqual({});
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("does not overwrite a saved draft when a queued prompt is blocked", async () => {
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: { value: { case: "ai", value: { enabled: false } } },
    });
    const page = { path: "/projects/billing", title: "Billing" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/billing", page });
    store.setDraft(chat.id, { text: "My unsent question" });
    store.queueUserMessage({ chatId: chat.id, text: "Landing suggestion", page });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(getQueuedUserMessage(chat.id)).toBeUndefined());
    });
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe("My unsent question");
    expect(useAgentStore.getState().getMessages(chat.id)).toHaveLength(0);
    view.unmount();
  });

  test("restores both bound references when a queued follow-up is blocked", async () => {
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: { value: { case: "ai", value: { enabled: false } } },
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    const project = {
      kind: "project" as const,
      name: "projects/orders",
      project: "projects/orders",
      label: "Orders",
    };
    const database = {
      kind: "database" as const,
      name: "instances/prod/databases/orders",
      project: "projects/orders",
      label: "orders (prod)",
    };
    store.queueUserMessage({
      chatId: chat.id,
      text: "@Orders @orders (prod) What should I review next?",
      page,
      resourceReferences: [project, database],
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(getQueuedUserMessage(chat.id)).toBeUndefined());
    });
    expect(store.getDraft(chat.id)).toEqual({
      text: "@Orders @orders (prod) What should I review next?",
      mentions: [
        { resource: project, range: { start: 0, end: 7 } },
        { resource: database, range: { start: 8, end: 22 } },
      ],
    });
    expect(store.getMessages(chat.id)).toHaveLength(0);
    view.unmount();
  });

  test("binds an initial empty chat and uses its context for every run", async () => {
    mocks.runAgentLoop.mockResolvedValue({
      kind: "completed",
      totalTokensUsed: 0,
    });
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "First question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      (container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await Promise.resolve();
    });

    const chatId = useAgentStore.getState().currentChatId!;
    expect(useAgentStore.getState().getChat(chatId)?.context).toBe(
      "projects/orders"
    );
    expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(
      expect.objectContaining({ context: "projects/orders" })
    );

    mocks.currentContext = "projects/other";
    mocks.currentPath = "/projects/other/issues";
    await act(async () => {
      setTextareaValue(textarea, "Follow-up question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      (container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await Promise.resolve();
    });

    expect(mocks.buildSystemPrompt).toHaveBeenCalledTimes(2);
    expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(
      expect.objectContaining({
        context: "projects/orders",
        path: "/projects/other/issues",
      })
    );
    expect(useAgentStore.getState().getChat(chatId)?.context).toBe(
      "projects/orders"
    );
    unmount();
  });

  test("shows the project title while retaining its resource context", () => {
    useAgentStore.getState().ensureCurrentChat(undefined, "projects/orders");
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    expect(container.querySelector("[data-agent-context]")?.textContent).toBe(
      "Order service"
    );
    expect(useAgentStore.getState().chats[0].context).toBe("projects/orders");
    unmount();
  });

  test("fetches a project title when the bound project is not cached", () => {
    mocks.projectsByName = {};
    useAgentStore.getState().ensureCurrentChat(undefined, "projects/orders");
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith(
      "projects/orders",
      true
    );
    expect(container.querySelector("[data-agent-context]")?.textContent).toBe(
      "orders"
    );
    unmount();
  });

  test("shows the workspace title for a workspace-bound chat", () => {
    useAgentStore.getState().ensureCurrentChat(undefined, "workspaces/demo");
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    expect(container.querySelector("[data-agent-context]")?.textContent).toBe(
      "Demo workspace"
    );
    unmount();
  });

  test("changes the context of an empty chat from the composer", async () => {
    useAgentStore.getState().setWorkspaceName("workspaces/demo", "agent@example.com");
    useAgentStore.getState().ensureCurrentChat(undefined, "projects/orders");
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });
    await act(async () => {
      document
        .querySelector<HTMLButtonElement>(
          '[data-agent-context-option="workspaces/demo"]'
        )
        ?.click();
    });

    const state = useAgentStore.getState();
    expect(state.chats).toHaveLength(2);
    expect(state.chats.some((chat) => chat.context === "projects/orders")).toBe(true);
    expect(state.getChat(state.currentChatId)?.context).toBe("workspaces/demo");
    expect(container.querySelector("[data-agent-context]")?.textContent).toBe(
      "Demo workspace"
    );
    unmount();
  });

  test("starts a separate chat when selecting another context after a message", async () => {
    useAgentStore.getState().setWorkspaceName("workspaces/demo", "agent@example.com");
    const original = useAgentStore
      .getState()
      .ensureCurrentChat(undefined, "projects/orders", "workspaces/demo");
    act(() => {
      useAgentStore.getState().addMessage({
        chatId: original.id,
        role: "user",
        content: "What changed?",
      });
    });
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });
    expect(document.body.textContent).not.toContain("Start new chat in Billing");
    expect(document.body.textContent).toContain("Billing");
    await act(async () => {
      document
        .querySelector<HTMLButtonElement>(
          '[data-agent-context-option="projects/billing"]'
        )
        ?.click();
    });

    expect(useAgentStore.getState().chats).toHaveLength(2);
    expect(useAgentStore.getState().getChat(original.id)?.context).toBe(
      "projects/orders"
    );
    expect(useAgentStore.getState().getMessages(original.id)).toHaveLength(1);
    expect(useAgentStore.getState().currentChatId).not.toBe(original.id);
    expect(useAgentStore.getState().getChat(useAgentStore.getState().currentChatId)?.context)
      .toBe("projects/billing");

    mocks.runAgentLoop.mockResolvedValue({
      kind: "completed",
      totalTokensUsed: 0,
    });
    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "What about this project?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      (container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
      await Promise.resolve();
    });
    expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(
      expect.objectContaining({ context: "projects/billing" })
    );
    unmount();
  });

  test("keeps an inline mention in the source draft when switching contexts", async () => {
    const store = useAgentStore.getState();
    store.setWorkspaceName("workspaces/demo", "agent@example.com");
    const original = store.ensureCurrentChat(
      undefined,
      "projects/orders",
      "workspaces/demo"
    );
    act(() => {
      store.addMessage({
        chatId: original.id,
        role: "user",
        content: "Earlier question",
      });
      store.setDraft(original.id, {
        text: "Compare @Billing policies",
        mention: { start: 8, end: 16 },
        resource: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      });
    });
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });
    await act(async () => {
      document
        .querySelector<HTMLButtonElement>(
          '[data-agent-context-option="projects/orders"]'
        )
        ?.click();
    });
    expect(useAgentStore.getState().currentChatId).toBe(original.id);
    expect(store.getDraft(original.id).mention).toEqual({ start: 8, end: 16 });

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });
    await act(async () => {
      document
        .querySelector<HTMLButtonElement>(
          '[data-agent-context-option="workspaces/demo"]'
        )
        ?.click();
    });

    expect(useAgentStore.getState().currentChatId).not.toBe(original.id);
    expect(store.getDraft(original.id)).toMatchObject({
      text: "Compare @Billing policies",
      mention: { start: 8, end: 16 },
      resource: { name: "projects/billing" },
    });
    act(() => store.setCurrentChat(original.id));
    expect((container.querySelector("textarea") as HTMLTextAreaElement).value).toBe(
      "Compare @Billing policies"
    );
    expect(container.querySelector("[data-agent-inline-mention]")?.textContent).toBe(
      "@Billing"
    );
    unmount();
  });

  test("closes the context picker when an agent run starts", async () => {
    const chat = useAgentStore
      .getState()
      .ensureCurrentChat(undefined, "projects/orders");
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });
    expect(document.querySelector("[data-agent-context-options]")).not.toBeNull();

    await act(async () => {
      useAgentStore.getState().setChatStatus(chat.id, "running");
    });

    expect(
      container.querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.disabled
    ).toBe(true);
    expect(document.querySelector("[data-agent-context-options]")).toBeNull();
    unmount();
  });

  test("shows project titles without resource IDs in the context picker", async () => {
    mocks.availableProjects = [
      { name: "projects/billing-a", title: "Billing" },
      { name: "projects/billing-b", title: "Billing" },
      { name: "projects/orders", title: "Orders" },
    ];
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });

    expect(
      document.querySelector('[data-agent-context-option="projects/billing-a"]')
        ?.textContent
    ).toBe("Billing");
    expect(
      document.querySelector('[data-agent-context-option="projects/billing-b"]')
        ?.textContent
    ).toBe("Billing");
    expect(
      document.querySelector('[data-agent-context-option="projects/orders"]')
        ?.textContent
    ).toBe("Orders");
    unmount();
  });

  test("disables browser autocomplete for the context filter", async () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-context-trigger]")
        ?.click();
    });

    expect(
      document.querySelector<HTMLInputElement>(
        '[data-agent-context-options] input'
      )?.autocomplete
    ).toBe("off");
    unmount();
  });

  test("shows AI setup before the first prompt when AI is disabled", async () => {
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: {
        value: {
          case: "ai",
          value: { enabled: false },
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    await act(async () => {
      render();
      await Promise.resolve();
    });
    const chatId = useAgentStore.getState().currentChatId!;
    expect(mocks.getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.AI,
      true
    );
    expect(useAgentStore.getState().getChat(chatId)).toMatchObject({
      requiresAIConfiguration: true,
    });

    unmount();
  });

  test("React locale token usage labels include the token count", () => {
    for (const locale of [enUS, esES, jaJP, viVN, zhCN]) {
      expect(locale.agent["chat-total-tokens"]).toContain("{{count}}");
    }
  });

  test("localizes user-stopped request messaging", () => {
    expect(enUS.agent["request-stopped-by-user"]).toBe(
      "Request stopped by user"
    );
    expect(enUS.agent["request-stopped-retry-hint"]).toContain(
      "may still complete"
    );
    for (const locale of [enUS, esES, jaJP, viVN, zhCN]) {
      expect(locale.agent["request-stopped-by-user"].trim()).not.toBe("");
      expect(locale.agent["request-stopped-retry-hint"].trim()).not.toBe("");
    }
  });

  test("removes a cancelled @ trigger without loading page-element suggestions", async () => {
    const suggestions: DomRefSuggestion[] = [
      {
        ref: "button.submit",
        tag: "BUTTON",
        role: "button",
        label: "Submit",
        value: "",
      },
    ];
    mocks.lazyExtractDomRefSuggestions.mockResolvedValue(suggestions);
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    useAgentStore.getState().createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });

    const { render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = document.body.querySelector(
      "textarea"
    ) as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(mocks.lazyExtractDomRefSuggestions).not.toHaveBeenCalled();
    expect(document.body.querySelector("[data-agent-resource-picker]")).not.toBeNull();
    expect(document.body.querySelector("[data-agent-mention-list]")).toBeNull();

    const shiftEnter = new KeyboardEvent("keydown", {
      key: "Enter",
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    });
    await act(async () => {
      textarea.dispatchEvent(shiftEnter);
    });
    expect(shiftEnter.defaultPrevented).toBe(false);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();

    await act(async () => {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
      );
    });
    expect(document.body.querySelector("[data-agent-resource-picker]")).toBeNull();
    expect(textarea.value).toBe("");

    await act(async () => {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", bubbles: true })
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();

    unmount();
  });

  test("keeps a workspace chat and scopes only the mentioned turn to another project", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    store.setWorkspaceName("workspaces/demo", "agent@example.com");
    const original = store.createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
      page: { path: "/landing", title: "Demo workspace" },
    });
    store.addMessage({ chatId: original.id, role: "user", content: "Earlier work" });
    const destination = store.createChat({
      context: "projects/billing",
      workspaceName: "workspaces/demo",
      select: false,
    });
    store.setDraft(destination.id, { text: "Billing draft" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Tell me about @");
      textarea.setSelectionRange(15, 15);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="project"]'
      )?.click();
    });
    await vi.waitFor(() => expect(mocks.searchAgentProjects).toHaveBeenCalled());
    await vi.waitFor(() =>
      expect(document.querySelector('[data-agent-resource-project="projects/billing"]')).not.toBeNull()
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-project="projects/billing"]'
      )?.click();
    });

    expect(textarea.value).toBe("Tell me about @Billing ");
    expect(view.container.querySelector("[data-agent-context]")?.textContent).toBe("Demo workspace");
    expect(view.container.querySelector("[data-agent-project-reference]")).toBeNull();
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@Billing");
    expect(store.getChat(original.id)?.context).toBe("workspaces/demo");
    expect(store.getMessages(original.id)).toHaveLength(1);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();

    await act(async () => {
      setTextareaValue(textarea, "Tell me about @Billing's policies");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    const currentChatId = useAgentStore.getState().currentChatId!;
    expect(currentChatId).toBe(original.id);
    expect(useAgentStore.getState().getChat(currentChatId)?.context).toBe("workspaces/demo");
    expect(useAgentStore.getState().getMessages(currentChatId)[1]).toMatchObject({
      role: "user",
      content: "Tell me about @Billing's policies",
      metadata: {
        resourceReference: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      },
    });
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "workspaces/demo",
        referencedProject: "projects/billing",
        runMode: "read-only",
      })
    );
    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "workspaces/demo",
        referencedProject: "projects/billing",
        runMode: "read-only",
      })
    );
    expect(mocks.getToolDefinitions).toHaveBeenCalledWith({
      runMode: "read-only",
      referencedProject: "projects/billing",
    });
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_project_overview",
      {},
      expect.any(String)
    );
    const referenceHistory = mocks.runAgentLoop.mock.calls[0][0] as {
      role: string;
      toolCalls?: { name: string; id: string }[];
      toolCallId?: string;
    }[];
    expect(referenceHistory.slice(-2)).toMatchObject([
      {
        role: "assistant",
        toolCalls: [{ name: "get_project_overview" }],
      },
      { role: "tool", toolCallId: referenceHistory.at(-2)?.toolCalls?.[0].id },
    ]);
    expect(useAgentStore.getState().getDraft(original.id)).toEqual({ text: "" });
    expect(useAgentStore.getState().getDraft(destination.id).text).toBe("Billing draft");

    await act(async () => {
      setTextareaValue(textarea, "What about this workspace?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ context: "workspaces/demo", referencedProject: undefined })
    );
    expect(mocks.buildOutboundHistory).toHaveBeenLastCalledWith(
      expect.any(String),
      expect.any(Array),
      "workspaces/demo"
    );
    view.unmount();
  });

  test("does not cite unrelated project overview resources for a generic API answer", async () => {
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/billing",
        evidence: {
          project: "projects/billing",
          references: [
            {
              kind: "issue",
              name: "projects/billing/issues/42",
              title: "Unrelated access request",
            },
            {
              kind: "database",
              name: "projects/billing/instances/prod/databases/example",
            },
          ],
        },
      }),
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "tool-list-databases",
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "DatabaseService/ListDatabases",
                body: { parent: "projects/billing", pageSize: 100 },
              }),
            },
          ],
        });
        callbacks.onToolResult(
          "tool-list-databases",
          JSON.stringify({
            status: 200,
            response: {
              databases: Array.from({ length: 26 }, (_, index) => ({
                name: `instances/prod/databases/db-${index}`,
              })),
            },
          })
        );
        callbacks.onText("Billing has 26 active databases.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "@Billing how many dbs?",
      mention: { start: 0, end: 8 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_project_overview",
      {},
      expect.any(String)
    );
    expect(
      store.getMessages(chat.id).find(
        (message) => message.content === "Billing has 26 active databases."
      )?.metadata?.evidence
    ).toBeUndefined();
    view.unmount();
  });

  test("selects a database directly from workspace context without opening projects", async () => {
    mocks.canListWorkspaceDatabases = true;
    mocks.currentContext = "workspaces/demo";
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [
        {
          name: "instances/prod/databases/orders",
          project: "projects/orders",
          instanceResource: { title: "Production" },
        },
      ],
      nextPageToken: "",
    });
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        database: {
          status: "available",
          name: "instances/prod/databases/orders",
          state: "ACTIVE",
        },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="database"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        "workspaces/demo",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(document.querySelector('input[aria-label="agent.resource-picker.search-projects"]')).toBeNull();

    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-database="instances/prod/databases/orders"]'
      )?.click();
    });
    expect(textarea.value).toBe("@orders (Production) ");
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (Production)");
    expect(document.querySelector("[data-agent-resource-picker]")).toBeNull();
    expect(store.getChat(chat.id)?.context).toBe("workspaces/demo");

    expect(view.container.querySelector<HTMLButtonElement>("[data-agent-send]")).not.toBeDisabled();
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toEqual({
      kind: "database",
      name: "instances/prod/databases/orders",
      project: "projects/orders",
      label: "orders (Production)",
    });
    view.unmount();
  });

  test("selects an issue from the chat project and uses its fresh overview for one turn", async () => {
    const issueName = "projects/orders/issues/42";
    const overview = JSON.stringify({
      project: "projects/orders",
      issue: { status: "available", name: issueName, title: "Update orders schema" },
      evidence: {
        project: "projects/orders",
        references: [{ kind: "issue", name: issueName, title: "Update orders schema" }],
      },
    });
    mocks.executeTool.mockResolvedValueOnce({ kind: "tool_result", result: overview });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="issue"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
        "projects/orders", "", "", expect.any(AbortSignal)
      )
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        `[data-agent-resource-issue="${issueName}"]`
      )?.click();
    });
    expect(textarea.value).toBe("@Update orders schema ");
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@Update orders schema");

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toEqual({
      kind: "issue", name: issueName, project: "projects/orders", label: "Update orders schema",
    });
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_issue_overview", { issueName }, expect.any(String)
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/orders",
        referencedIssue: issueName,
      })
    );
    const history = mocks.runAgentLoop.mock.calls[0][0] as {
      role: string;
      toolCalls?: { name: string; arguments: string }[];
      content?: string;
    }[];
    expect(history.slice(-2)).toMatchObject([
      { role: "assistant", toolCalls: [{ name: "get_issue_overview", arguments: JSON.stringify({ issueName }) }] },
      { role: "tool", content: overview },
    ]);

    await act(async () => {
      setTextareaValue(textarea, "How many issues here?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ referencedProject: undefined, referencedIssue: undefined })
    );
    view.unmount();
  });

  test("selects a Plan from the chat project and uses its fresh overview for one turn", async () => {
    const planName = "projects/orders/plans/7";
    const overview = JSON.stringify({
      project: "projects/orders",
      plan: {
        status: "available",
        name: planName,
        title: "Change orders schema",
        state: "ACTIVE",
      },
      evidence: {
        project: "projects/orders",
        references: [
          { kind: "plan", name: planName, title: "Change orders schema" },
        ],
      },
    });
    mocks.executeTool.mockResolvedValueOnce({ kind: "tool_result", result: overview });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("Plan overview response");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="plan"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentPlans).toHaveBeenCalledWith(
        "projects/orders", "", "", expect.any(AbortSignal)
      )
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        `[data-agent-resource-plan="${planName}"]`
      )?.click();
    });
    expect(textarea.value).toBe("@Change orders schema (#7) ");

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toEqual({
      kind: "plan", name: planName, project: "projects/orders", label: "Change orders schema (#7)",
    });
    expect(mocks.extractPlanOverviewEvidence).toHaveBeenCalledWith(overview);
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_plan_overview", { planName }, expect.any(String)
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/orders",
        referencedPlan: planName,
      })
    );
    const history = mocks.runAgentLoop.mock.calls[0][0] as {
      role: string;
      toolCalls?: { name: string; arguments: string }[];
      content?: string;
    }[];
    expect(history.slice(-2)).toMatchObject([
      { role: "assistant", toolCalls: [{ name: "get_plan_overview", arguments: JSON.stringify({ planName }) }] },
      { role: "tool", content: overview },
    ]);
    expect(
      store
        .getMessages(chat.id)
        .find((message) => message.content === "Plan overview response")
        ?.metadata?.evidence
    ).toEqual({
      project: "projects/orders",
      references: [
        { kind: "plan", name: planName, title: "Change orders schema" },
      ],
    });

    await act(async () => {
      setTextareaValue(textarea, "How many Plans here?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ referencedProject: undefined, referencedPlan: undefined })
    );
    view.unmount();
  });

  test("does not send an unavailable selected Plan to the model", async () => {
    const planName = "projects/orders/plans/7";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({ project: "projects/orders", plan: { status: "unavailable" } }),
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Review @Change orders schema",
      mention: { start: 7, end: 28 },
      resource: { kind: "plan", name: planName, project: "projects/orders", label: "Change orders schema" },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() =>
        expect(store.getMessages(chat.id).some(
          (message) => message.content === "agent.plan-reference-unavailable"
        )).toBe(true)
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("uses an inline project with its Plan without changing the chat context", async () => {
    const planName = "projects/billing/plans/8";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/billing",
        plan: { status: "available", name: planName, state: "ACTIVE" },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Review @Billing @Change billing",
      mentions: [
        {
          resource: { kind: "project", name: "projects/billing", project: "projects/billing", label: "Billing" },
          range: { start: 7, end: 15 },
        },
        {
          resource: { kind: "plan", name: planName, project: "projects/billing", label: "Change billing" },
          range: { start: 16, end: 31 },
        },
      ],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getChat(chat.id)?.context).toBe("workspaces/demo");
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_plan_overview", { planName }, expect.any(String)
    );
    expect(mocks.executeTool).not.toHaveBeenCalledWith(
      "get_project_overview", {}, expect.any(String)
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ referencedProject: "projects/billing", referencedPlan: planName })
    );
    view.unmount();
  });

  test("selects a cross-project issue in workspace chat without changing its context", async () => {
    const issueName = "projects/orders/issues/42";
    mocks.currentContext = "workspaces/demo";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        issue: { status: "available", name: issueName },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="issue"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentIssues).toHaveBeenCalledWith(
        "projects/-", "", "", expect.any(AbortSignal)
      )
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        `[data-agent-resource-issue="${issueName}"]`
      )?.click();
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(store.getChat(chat.id)?.context).toBe("workspaces/demo");
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toEqual({
      kind: "issue", name: issueName, project: "projects/orders", label: "Update orders schema",
    });
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_issue_overview", { issueName }, expect.any(String)
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "workspaces/demo",
        referencedProject: "projects/orders",
        referencedIssue: issueName,
      })
    );
    view.unmount();
  });

  test("does not send an unavailable selected issue to the model", async () => {
    const issueName = "projects/orders/issues/42";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({ project: "projects/orders", issue: { status: "unavailable" } }),
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Review @Update orders schema",
      mention: { start: 7, end: 28 },
      resource: { kind: "issue", name: issueName, project: "projects/orders", label: "Update orders schema" },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() =>
        expect(store.getMessages(chat.id).some(
          (message) => message.content === "agent.issue-reference-unavailable"
        )).toBe(true)
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("uses an inline project with its Issue without changing the chat context", async () => {
    const issueName = "projects/billing/issues/7";
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/billing",
        issue: { status: "available", name: issueName },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Approve @Billing @Change billing",
      mentions: [
        {
          resource: { kind: "project", name: "projects/billing", project: "projects/billing", label: "Billing" },
          range: { start: 8, end: 16 },
        },
        {
          resource: { kind: "issue", name: issueName, project: "projects/billing", label: "Change billing" },
          range: { start: 17, end: 32 },
        },
      ],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_issue_overview", { issueName }, expect.any(String)
    );
    expect(mocks.executeTool).not.toHaveBeenCalledWith(
      "get_project_overview", {}, expect.any(String)
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ referencedProject: "projects/billing", referencedIssue: issueName })
    );
    view.unmount();
  });

  test("falls back to the workspace for database search in a contextless chat", async () => {
    mocks.currentContext = undefined;
    mocks.canListWorkspaceDatabases = true;
    const store = useAgentStore.getState();
    store.createChat({ workspaceName: "workspaces/demo" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="database"]'
      )?.click();
    });

    await vi.waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        "workspaces/demo",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    view.unmount();
  });

  test("uses an inline project as the database search parent and binds both references", async () => {
    const databaseName = "instances/prod/databases/invoices";
    mocks.searchAgentDatabases.mockResolvedValue({
      items: [{ name: databaseName, project: "projects/billing" }],
      nextPageToken: "",
    });
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/billing",
        database: { status: "available", name: databaseName, state: "ACTIVE" },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/orders",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "@");
      textarea.setSelectionRange(1, 1);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="project"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(document.querySelector('[data-agent-resource-project="projects/billing"]'))
        .not.toBeNull()
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-project="projects/billing"]'
      )?.click();
    });
    await act(async () => {
      setTextareaValue(textarea, "@Billing @");
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(document.querySelector('[data-agent-resource-category="project"]')).toBeNull();
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="database"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        "projects/billing",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        `[data-agent-resource-database="${databaseName}"]`
      )?.click();
    });
    expect(textarea.value).toBe("@Billing @invoices (prod) ");
    expect(view.container.querySelectorAll("[data-agent-inline-mention]")).toHaveLength(2);

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    expect(store.getMessages(chat.id)[0].metadata?.resourceReferences).toEqual([
      {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
      {
        kind: "database",
        name: databaseName,
        project: "projects/billing",
        label: "invoices (prod)",
      },
    ]);
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_database_overview",
      { databaseName },
      expect.any(String)
    );
    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/billing",
        referencedDatabase: databaseName,
      })
    );
    view.unmount();
  });

  test("deleting a database token keeps its preceding project binding", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });
    store.setDraft(chat.id, {
      text: "@Billing @invoices ",
      mentions: [
        {
          resource: {
            kind: "project",
            name: "projects/billing",
            project: "projects/billing",
            label: "Billing",
          },
          range: { start: 0, end: 8 },
        },
        {
          resource: {
            kind: "database",
            name: "instances/prod/databases/invoices",
            project: "projects/billing",
            label: "invoices",
          },
          range: { start: 9, end: 18 },
        },
      ],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    act(() => {
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new KeyboardEvent("keydown", {
        key: "Backspace",
        bubbles: true,
        cancelable: true,
      }));
    });

    expect(textarea.value).toBe("@Billing ");
    expect(store.getDraft(chat.id).mentions).toHaveLength(1);
    expect(store.getDraft(chat.id).mentions?.[0].resource.name).toBe("projects/billing");
    expect(view.container.querySelectorAll("[data-agent-inline-mention]")).toHaveLength(1);
    view.unmount();
  });

  test("selects a database inline and uses a fresh overview for only that turn", async () => {
    const databaseName = "instances/prod/databases/orders";
    const overview = JSON.stringify({
      project: "projects/orders",
      database: { status: "available", name: databaseName, state: "ACTIVE" },
      evidence: {
        project: "projects/orders",
        references: [{ kind: "database", name: databaseName }],
      },
    });
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: overview,
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/orders",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Check @");
      textarea.setSelectionRange(7, 7);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="database"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        "projects/orders",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        `[data-agent-resource-database="${databaseName}"]`
      )?.click();
    });
    expect(textarea.value).toBe("Check @orders (prod) ");
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (prod)");
    expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toEqual({
      kind: "database",
      name: databaseName,
      project: "projects/orders",
      label: "orders (prod)",
    });
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_database_overview",
      { databaseName },
      expect.any(String)
    );
    expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/orders",
        referencedDatabase: databaseName,
      })
    );
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/orders",
        referencedDatabase: databaseName,
      })
    );
    const history = mocks.runAgentLoop.mock.calls[0][0] as {
      role: string;
      toolCalls?: { name: string; arguments: string; id: string }[];
      content?: string;
      toolCallId?: string;
    }[];
    expect(history.slice(-2)).toMatchObject([
      {
        role: "assistant",
        toolCalls: [{
          name: "get_database_overview",
          arguments: JSON.stringify({ databaseName }),
        }],
      },
      { role: "tool", content: overview },
    ]);

    await act(async () => {
      setTextareaValue(textarea, "How many databases here?");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: undefined,
      })
    );
    view.unmount();
  });

  test("reads a selected database again after an API write in the same turn", async () => {
    const databaseName = "instances/prod/databases/orders";
    const beforeWrite = JSON.stringify({
      project: "projects/orders",
      database: {
        status: "available",
        name: databaseName,
        state: "ACTIVE",
        sync: { status: "available", syncStatus: "FAILED" },
      },
    });
    const afterWrite = JSON.stringify({
      project: "projects/orders",
      database: {
        status: "available",
        name: databaseName,
        state: "ACTIVE",
        sync: { status: "available", syncStatus: "OK" },
      },
    });
    mocks.executeTool
      .mockResolvedValueOnce({ kind: "tool_result", result: beforeWrite })
      .mockResolvedValueOnce({ kind: "tool_result", result: "{}" })
      .mockResolvedValueOnce({ kind: "tool_result", result: afterWrite });
    let postWriteResult: unknown;
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, execute) => {
      await execute(
        "call_api",
        {
          operationId: "DatabaseService/SyncDatabase",
          body: { name: databaseName },
        },
        "sync-call"
      );
      postWriteResult = await execute(
        "get_database_overview",
        { databaseName },
        "read-after-sync"
      );
      return { kind: "completed", totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Sync @orders and report its status",
      mention: { start: 5, end: 12 },
      resource: {
        kind: "database",
        name: databaseName,
        project: "projects/orders",
        label: "orders",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
    });

    const executorCalls = mocks.executeTool.mock.calls;
    view.unmount();
    expect(postWriteResult).toEqual({ kind: "tool_result", result: afterWrite });
    expect(executorCalls).toHaveLength(3);
    expect(mocks.executeTool).toHaveBeenNthCalledWith(
      3,
      "get_database_overview",
      { databaseName },
      "read-after-sync"
    );
  });

  test("attaches database Evidence when the model answers from preflight alone", async () => {
    const databaseName = "instances/prod/databases/orders";
    const evidence = {
      project: "projects/orders",
      references: [{ kind: "database", name: databaseName }],
    };
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        database: { status: "available", name: databaseName, state: "ACTIVE" },
        evidence,
      }),
    });
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("Orders is active.");
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Check @orders",
      mention: { start: 6, end: 13 },
      resource: {
        kind: "database",
        name: databaseName,
        project: "projects/orders",
        label: "orders",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
    });

    const answerEvidence = store
      .getMessages(chat.id)
      .find((message) => message.content === "Orders is active.")?.metadata
      ?.evidence;
    view.unmount();
    expect(mocks.executeTool).toHaveBeenCalledTimes(1);
    expect(answerEvidence).toEqual(evidence);
  });

  test.each([
    ["moved", { project: "projects/billing", database: { status: "available", name: "instances/prod/databases/orders", state: "ACTIVE" } }],
    ["deleted", { project: "projects/orders", database: { status: "available", name: "instances/prod/databases/orders", state: "DELETED" } }],
    ["inaccessible", { project: "projects/orders", database: { status: "unavailable" } }],
  ])("does not run the model for a %s selected database", async (_case, overview) => {
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify(overview),
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "Check @orders",
      mention: { start: 6, end: 13 },
      resource: {
        kind: "database",
        name: "instances/prod/databases/orders",
        project: "projects/orders",
        label: "orders",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() =>
        expect(store.getMessages(chat.id).some(
          (message) => message.content === "agent.database-reference-unavailable"
        )).toBe(true)
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    view.unmount();
  });

  test("binds a contextless chat to the selected database's owner project", async () => {
    const databaseName = "instances/prod/databases/billing";
    mocks.currentContext = undefined;
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/billing",
        database: { status: "available", name: databaseName, state: "ACTIVE" },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ workspaceName: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Inspect @billing",
      mention: { start: 8, end: 16 },
      resource: {
        kind: "database",
        name: databaseName,
        project: "projects/billing",
        label: "billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getChat(chat.id)?.context).toBe("projects/billing");
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toMatchObject({
      kind: "database",
      name: databaseName,
      project: "projects/billing",
    });
    view.unmount();
  });

  test("offers databases under a selected project, but not another project", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/orders",
      workspaceName: "workspaces/demo",
    });
    store.setDraft(chat.id, {
      text: "Ask @Billing and ",
      mention: { start: 4, end: 12 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Ask @Billing and @");
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(
      document.querySelector("[data-agent-resource-picker]")
    ).not.toBeNull();
    expect(
      document.querySelector('[data-agent-resource-category="project"]')
    ).toBeNull();
    expect(
      document.querySelector('[data-agent-resource-category="database"]')
    ).not.toBeNull();
    expect(
      document.querySelector('[data-agent-resource-picker] [role="status"]')
    ).toBeNull();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
    await act(async () => {
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-category="database"]'
      )?.click();
    });
    await vi.waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        "projects/billing",
        "",
        "",
        expect.any(AbortSignal)
      )
    );
    expect(view.container.querySelector('[role="alert"]')).toBeNull();
    expect(store.getDraft(chat.id).mentions?.[0].resource.name).toBe("projects/billing");
    expect(
      view.container.querySelector("[data-agent-send]")
    ).toBeDisabled();

    await act(async () => {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
      );
    });
    expect(document.querySelector("[data-agent-resource-picker]")).toBeNull();
    expect(view.container.querySelector('[role="alert"]')).toBeNull();
    expect(textarea.value).toBe("Ask @Billing and ");
    expect(store.getDraft(chat.id).mentions?.[0].resource.name).toBe("projects/billing");
    expect(view.container.querySelector("[data-agent-send]")).not.toBeDisabled();

    await act(async () => {
      setTextareaValue(textarea, "Ask @Billing and ed@example.com");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(
      view.container.querySelector("[data-agent-send]")
    ).not.toBeDisabled();

    await act(async () => {
      setTextareaValue(textarea, "Ask Billing and @");
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(store.getDraft(chat.id).mentions).toBeUndefined();
    expect(
      view.container.querySelector("[data-agent-send]")
    ).not.toBeDisabled();
    expect(
      document.querySelector('[data-agent-resource-category="project"]')
    ).not.toBeNull();
    view.unmount();
  });

  test("drops the selected resource when its visible mention is edited", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Ask @Billing about policy",
      mention: { start: 4, end: 12 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe(
      "@Billing"
    );

    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Ask @Bill about policy");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(store.getDraft(chat.id)).toEqual({ text: "Ask @Bill about policy" });
    expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toBeUndefined();
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ referencedProject: undefined, runMode: "default" })
    );
    view.unmount();
  });

  test("Backspace after the selected Project removes the mention and clears its reference", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Ask @Billing ",
      mention: { start: 4, end: 12 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe(
      "@Billing"
    );

    act(() => {
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Backspace",
          bubbles: true,
          cancelable: true,
        })
      );
    });

    expect(textarea.value).toBe("Ask ");
    expect(textarea.selectionStart).toBe(4);
    expect(store.getDraft(chat.id)).toEqual({ text: "Ask " });
    expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id)[0]).toMatchObject({ content: "Ask" });
    expect(store.getMessages(chat.id)[0].metadata?.resourceReference).toBeUndefined();
    view.unmount();
  });

  test("Delete inside the selected Project preserves surrounding draft text", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Ask @Billing about policy",
      mention: { start: 4, end: 12 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    act(() => {
      textarea.setSelectionRange(8, 8);
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Delete",
          bubbles: true,
          cancelable: true,
        })
      );
    });

    expect(textarea.value).toBe("Ask about policy");
    expect(textarea.selectionStart).toBe(4);
    expect(store.getDraft(chat.id)).toEqual({ text: "Ask about policy" });
    expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
    view.unmount();
  });

  test("keeps the selected resource when editing around its visible mention", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, {
      text: "Ask @Billing about policy",
      mention: { start: 4, end: 12 },
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Please ask @Billing about policy");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(store.getDraft(chat.id).mentions?.[0].range).toEqual({ start: 11, end: 19 });
    expect(store.getDraft(chat.id).mentions?.[0].resource.name).toBe("projects/billing");
    expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe(
      "@Billing"
    );
    view.unmount();
  });

  test("does not ask the model about a deleted or inaccessible mentioned project", async () => {
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({ error: "The referenced project is unavailable." }),
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, {
      text: "What changed in billing?",
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() =>
        expect(store.getMessages(chat.id).some(
          (message) => message.content === "agent.project-reference-unavailable"
        )).toBe(true)
      );
    });

    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    view.unmount();
  });

  test("does not reclassify a populated contextless chat when sending a project mention", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    mocks.currentContext = undefined;
    const store = useAgentStore.getState();
    const legacy = store.createChat({ workspaceName: "workspaces/demo" });
    store.addMessage({ chatId: legacy.id, role: "user", content: "Old question" });
    store.setDraft(legacy.id, {
      text: "Inspect billing",
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(store.getChat(legacy.id)?.context).toBeUndefined();
    expect(store.getMessages(legacy.id)).toHaveLength(1);
    const nextChatId = useAgentStore.getState().currentChatId!;
    expect(nextChatId).not.toBe(legacy.id);
    expect(store.getChat(nextChatId)?.context).toBe("projects/billing");
    expect(store.getMessages(nextChatId)[0].content).toBe("Inspect billing @Billing");
    expect(store.getMessages(nextChatId)[0].metadata?.resourceReference?.name).toBe(
      "projects/billing"
    );
    expect(mocks.executeTool).toHaveBeenCalledWith(
      "get_project_overview",
      {},
      expect.any(String)
    );
    view.unmount();
  });

  test("does not reuse a stale project chat when a contextless mention is unavailable", async () => {
    mocks.currentContext = undefined;
    mocks.executeTool.mockResolvedValueOnce({
      kind: "tool_result",
      result: JSON.stringify({ error: "The referenced project is unavailable." }),
    });
    const store = useAgentStore.getState();
    const legacy = store.createChat({ workspaceName: "workspaces/demo" });
    store.addMessage({ chatId: legacy.id, role: "user", content: "Old question" });
    store.setDraft(legacy.id, {
      text: "Inspect billing",
      resource: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getMessages(useAgentStore.getState().currentChatId!).some(
          (message) => message.content === "agent.project-reference-unavailable"
        )).toBe(true)
      );
    });

    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getChat(legacy.id)?.context).toBeUndefined();
    expect(store.getMessages(legacy.id)).toHaveLength(1);
    view.unmount();
  });

  test("does not run a chat-context starter fast path for a referenced project", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.queueUserMessage({
      chatId: chat.id,
      text: "What changed recently in this project?",
      page: { path: "/projects/orders", title: "Orders" },
      starterId: "project-open-recent",
      resourceReference: {
        kind: "project",
        name: "projects/billing",
        project: "projects/billing",
        label: "Billing",
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(mocks.getRecentActivity).not.toHaveBeenCalled();
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context: "projects/orders",
        referencedProject: "projects/billing",
      })
    );
    view.unmount();
  });

  test("sends the submitted draft without discarding edits made during the configuration check", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const chat = useAgentStore.getState().createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "First question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    let finishConfigurationCheck: (() => void) | undefined;
    mocks.getOrFetchSettingByName.mockImplementationOnce(
      () => new Promise<undefined>((resolve) => {
        finishConfigurationCheck = () => resolve(undefined);
      })
    );
    act(() => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });
    expect(finishConfigurationCheck).toBeDefined();

    await act(async () => {
      setTextareaValue(textarea, "Second question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await act(async () => {
      finishConfigurationCheck?.();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(useAgentStore.getState().getMessages(chat.id)[0]).toMatchObject({
      role: "user",
      content: "First question",
    });
    expect(textarea.value).toBe("Second question");
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe("Second question");
    view.unmount();
  });

  test("does not send an old account's draft after the account changes during configuration check", async () => {
    const store = useAgentStore.getState();
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Private question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    let finishConfigurationCheck!: (setting: undefined) => void;
    mocks.getOrFetchSettingByName.mockImplementationOnce(
      () =>
        new Promise<undefined>((resolve) => {
          finishConfigurationCheck = resolve;
        })
    );
    act(() => {
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });
    expect(finishConfigurationCheck).toBeDefined();

    act(() => {
      store.setWorkspaceName("workspaces/demo", "other@example.com");
    });
    await act(async () => finishConfigurationCheck(undefined));

    const current = useAgentStore.getState();
    expect(current.userEmail).toBe("other@example.com");
    expect(current.getMessages(current.currentChatId)).toHaveLength(0);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("does not open the resource picker for an email address", async () => {
    useAgentStore.getState().createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "ed@example.com");
      textarea.setSelectionRange(textarea.value.length, textarea.value.length);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(document.querySelector("[data-agent-resource-picker]")).toBeNull();
    view.unmount();
  });

  test.each([
    ["project", "projects/orders"],
    ["legacy", undefined],
  ])("offers the project picker in a %s chat", async (_kind, context) => {
    const store = useAgentStore.getState();
    store.createChat({ context, workspaceName: "workspaces/demo" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    await act(async () => {
      setTextareaValue(textarea, "Ask @ about policy");
      textarea.setSelectionRange(5, 5);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(textarea.value).toBe("Ask @ about policy");
    expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
    view.unmount();
  });

  test("closes the picker when the caret moves away from the mention", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    useAgentStore.getState().createChat({
      context: "workspaces/demo",
      workspaceName: "workspaces/demo",
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Ask @ about policy");
      textarea.setSelectionRange(5, 5);
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();

    textarea.setSelectionRange(0, 0);
    await act(async () => {
      textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(document.querySelector("[data-agent-resource-picker]")).toBeNull();
    expect(useAgentStore.getState().getMessages(useAgentStore.getState().currentChatId!))
      .toEqual(expect.arrayContaining([
        expect.objectContaining({ role: "user", content: "Ask @ about policy" }),
      ]));
    view.unmount();
  });

  test("does not send while Enter is part of IME composition", async () => {
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "A question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Enter",
          isComposing: true,
          bubbles: true,
        })
      );
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("restores an unsent question after switching chats and remounting", async () => {
    const store = useAgentStore.getState();
    const original = store.createChat({ context: "projects/orders" });
    const second = store.createChat({ context: "projects/billing", select: false });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => {
      setTextareaValue(textarea, "Unsent question");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(store.getDraft(original.id).text).toBe("Unsent question");
    await act(async () => {
      store.setCurrentChat(second.id);
    });
    expect(textarea.value).toBe("");
    await act(async () => {
      store.setCurrentChat(original.id);
    });
    expect(textarea.value).toBe("Unsent question");
    view.unmount();

    const remounted = renderIntoContainer(<AgentInput />);
    remounted.render();
    expect((remounted.container.querySelector("textarea") as HTMLTextAreaElement).value)
      .toBe("Unsent question");
    remounted.unmount();
  });

  test("shows token usage near the textarea", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, totalTokensUsed: 1234 } : chat
        ),
    });
    mocks.useTranslation.mockReturnValue({
      t: (key: string, values?: Record<string, unknown>) =>
        key === "agent.chat-total-tokens"
          ? `Tokens used: ${values?.count ?? ""}`
          : key,
      i18n: { language: "en-US" },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const inputFooter = container.querySelector("[data-agent-input-footer]");
    expect(inputFooter).toBeInstanceOf(HTMLDivElement);
    expect(inputFooter?.textContent).toContain("Tokens used: 1,234");

    unmount();
  });

  test("keeps token usage visible while awaiting confirmation", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId
            ? { ...chat, status: "awaiting_user", totalTokensUsed: 5678 }
            : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-call-confirm",
          prompt: "Continue?",
          kind: "confirm",
        },
      },
    });
    mocks.useTranslation.mockReturnValue({
      t: (key: string, values?: Record<string, unknown>) =>
        key === "agent.chat-total-tokens"
          ? `Tokens used: ${values?.count ?? ""}`
          : key,
      i18n: { language: "en-US" },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const inputFooter = container.querySelector("[data-agent-input-footer]");
    expect(inputFooter).toBeInstanceOf(HTMLDivElement);
    expect(inputFooter?.textContent).toContain("Tokens used: 5,678");
    expect(container.textContent).toContain("Continue?");
    expect(
      [...container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.confirm"
      )?.classList.contains("h-9")
    ).toBe(true);

    unmount();
  });

  test.each(["input", "choose", "confirm"] as const)(
    "cancels an unanswered %s request without replying or restarting",
    (kind) => {
      const store = useAgentStore.getState();
      const chatId = store.currentChatId!;
      store.addMessage({ chatId, role: "user", content: "Original question" });
      store.setDraft(chatId, { text: "Next question" });
      useAgentStore.setState({
        chats: useAgentStore.getState().chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
        pendingAskByChatId: {
          [chatId]: {
            toolCallId: "tool-ask",
            prompt: "Need more information",
            kind,
            ...(kind === "choose"
              ? { options: [{ label: "First option", value: "first" }] }
              : {}),
          },
        },
      });

      const view = renderIntoContainer(<AgentInput />);
      view.render();
      const cancel = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.cancel-request"
      );
      expect(cancel).toBeDefined();
      if (kind === "confirm") {
        expect(view.container.textContent).toContain("agent.cancel");
      }

      act(() => cancel?.click());

      expect(store.getPendingAsk(chatId)).toBeNull();
      expect(store.getChat(chatId)?.status).toBe("idle");
      expect(store.getChat(chatId)?.interrupted).toBe(false);
      expect(store.getDraft(chatId).text).toBe("Next question");
      expect(view.container.textContent).not.toContain("Need more information");
      expect(view.container.textContent).toContain("agent.send");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getMessages(chatId)).toEqual(
        expect.arrayContaining([
          expect.objectContaining({
            role: "tool",
            toolCallId: "tool-ask",
            content: JSON.stringify({ kind: "cancelled", reason: "user" }),
          }),
        ])
      );
      view.unmount();
    }
  );

  test("does not offer Cancel request for an API permission decision", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore.getState().chats.map((chat) =>
        chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
      ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "Update project",
          kind: "confirm",
          apiPermission: {
            actionId: "action-1",
            operationId: "ProjectService/UpdateProject",
          },
        },
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();

    expect(view.container.textContent).not.toContain("agent.cancel-request");
    expect(view.container.textContent).toContain("agent.api-permission-dismiss");
    view.unmount();
  });

  test("does not submit an answer when the request is canceled during configuration check", async () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.addMessage({ chatId, role: "user", content: "Original question" });
    useAgentStore.setState({
      chats: useAgentStore.getState().chats.map((chat) =>
        chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
      ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-ask",
          prompt: "Which database?",
          kind: "input",
        },
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    let finishConfigurationCheck!: (setting: undefined) => void;
    mocks.getOrFetchSettingByName.mockImplementationOnce(
      () =>
        new Promise<undefined>((resolve) => {
          finishConfigurationCheck = resolve;
        })
    );
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    act(() => {
      setTextareaValue(textarea, "Orders");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
    });
    expect(mocks.getOrFetchSettingByName).toHaveBeenCalled();
    act(() => {
      [...view.container.querySelectorAll("button")]
        .find((button) => button.textContent === "agent.cancel-request")
        ?.click();
    });
    await act(async () => finishConfigurationCheck(undefined));

    expect(store.getPendingAsk(chatId)).toBeNull();
    expect(store.getMessages(chatId).filter((message) => message.role === "tool"))
      .toHaveLength(1);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("localizes the separate Cancel request action and canceled status", () => {
    expect(enUS.agent["cancel-request"]).toBe("Cancel request");
    expect(enUS.agent["tool-request-canceled"]).toBe("Request canceled");
    for (const locale of [enUS, esES, jaJP, viVN, zhCN]) {
      expect(locale.agent["cancel-request"].trim()).not.toBe("");
      expect(locale.agent["tool-request-canceled"].trim()).not.toBe("");
    }
  });

  test("shows the exact pending API request and dispatches it only after approval", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-1",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders","title":"Orders"}}',
      expiresAt: Date.now() + 5 * 60_000,
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.textContent).toContain("bytebase.v1.ProjectService.UpdateProject");
    expect(view.container.textContent).toContain("projects/orders");
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    for (const label of ["agent.api-permission-allow", "agent.api-permission-deny"]) {
      const button = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === label
      );
      expect(button?.classList.contains("h-7")).toBe(true);
      expect(button?.classList.contains("text-xs")).toBe(true);
    }

    await act(async () => {
      const confirm = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-allow"
      );
      confirm?.click();
    });

    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith({
      actionId: "action-1",
      chatId,
      toolCallId: "tool-write",
      userName: "users/me@example.com",
      approved: true,
      signal: expect.any(AbortSignal),
    });
    expect(useAgentStore.getState().getMessages(chatId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-write",
          content: '{"status":200,"response":{}}',
        }),
      ])
    );
    expect(useAgentStore.getState().getPendingAsk(chatId)).toBeNull();
    view.unmount();
  });

  test("resumes after an approved delete without preflighting the now-deleted project", async () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.addMessage({
      chatId,
      role: "user",
      content: "@Order service Soft-delete this project",
      metadata: {
        resourceReference: {
          kind: "project",
          name: "projects/orders",
          project: "projects/orders",
          label: "Order service",
        },
      },
    });
    useAgentStore.setState({
      chats: store.chats.map((chat) =>
        chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
      ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-delete",
          prompt: "bytebase.v1.ProjectService.DeleteProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-delete",
            operationId: "bytebase.v1.ProjectService.DeleteProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.DeleteProject",
      body: '{"name":"projects/orders","purge":false}',
      expiresAt: Date.now() + 5 * 60_000,
    });
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: '{"error":"The referenced project is unavailable."}',
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      [...view.container.querySelectorAll("button")]
        .find((button) => button.textContent === "agent.api-permission-allow")
        ?.click();
    });

    expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
    expect(mocks.executeTool).not.toHaveBeenCalledWith(
      "get_project_overview",
      {},
      expect.any(String)
    );
    expect(store.getMessages(chatId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-delete",
          content: '{"status":200,"response":{}}',
        }),
      ])
    );
    view.unmount();
  });

  test("does not resume an approved API call after the user stops it", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().addMessage({
      chatId,
      role: "user",
      content: "Update the project",
    });
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-stop",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders"}}',
      expiresAt: Date.now() + 5 * 60_000,
    });
    let resolvePermission!: (result: string) => void;
    mocks.resolvePendingApiCall.mockImplementation(
      () =>
        new Promise<string>((resolve) => {
          resolvePermission = resolve;
        })
    );

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      const allow = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-allow"
      );
      allow?.click();
    });
    expect(useAgentStore.getState().getChat(chatId)?.status).toBe("running");

    await act(async () => {
      const stop = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.stop"
      );
      stop?.click();
    });
    expect(useAgentStore.getState().getChat(chatId)?.interruptionReason).toBe(
      "user"
    );

    await act(async () => {
      resolvePermission('{"status":200,"response":{}}');
    });
    expect(useAgentStore.getState().getChat(chatId)?.interruptionReason).toBe(
      "user"
    );
    expect(
      useAgentStore.getState().getMessages(chatId).some((message) =>
        message.role === "tool" && message.toolCallId === "tool-write"
      )
    ).toBe(false);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test.each([
    ["final-approval-with-rollout", "agent.permission.issue-approval-rollout"],
    ["final-approval-with-access-grant", "agent.permission.issue-approval-access-grant"],
    ["final-approval-with-role-grant", "agent.permission.issue-approval-role-grant"],
    ["final-approval", "agent.permission.issue-approval-final"],
    ["intermediate-approval", "agent.permission.issue-approval"],
    ["reject", "agent.permission.issue-rejection"],
  ])("shows the %s consequence beside an Issue permission request", (impact, messageKey) => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-issue-decision",
          prompt: "bytebase.v1.IssueService.ApproveIssue",
          kind: "confirm",
          apiPermission: {
            actionId: "issue-action",
            operationId: "bytebase.v1.IssueService.ApproveIssue",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.IssueService.ApproveIssue",
      body: '{"name":"projects/orders/issues/42"}',
      expiresAt: Date.now() + 5 * 60_000,
      issueDecision: {
        issueName: "projects/orders/issues/42",
        issueTitle: "Update orders schema",
        impact,
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.textContent).toContain("Update orders schema");
    expect(view.container.textContent).toContain("projects/orders/issues/42");
    expect(view.container.textContent).toContain(messageKey);
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    view.unmount();
  });

  test("does not allow approval after the pending API request is lost", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "lost-action",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.textContent).toContain("agent.api-permission-expired");
    const confirm = [...view.container.querySelectorAll("button")].find(
      (button) => button.textContent === "agent.api-permission-allow"
    );
    expect(confirm).toBeDisabled();
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    view.unmount();
  });

  test("counts down and disables approval as soon as a permission expires", () => {
    vi.useFakeTimers();
    vi.setSystemTime(100_000);
    const expiresAt = Date.now() + 5 * 60_000;
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-expiring",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockImplementation(() =>
      Date.now() < expiresAt
        ? {
            operationId: "bytebase.v1.ProjectService.UpdateProject",
            body: '{"project":{"name":"projects/orders"}}',
            expiresAt,
          }
        : undefined
    );
    mocks.useTranslation.mockReturnValue({
      t: (key: string, values?: Record<string, unknown>) =>
        key === "agent.api-permission-expires-in"
          ? `Expires in ${values?.time}`
          : key,
      i18n: { language: "en-US" },
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(view.container.textContent).toContain("Expires in 5:00");
    const allow = [...view.container.querySelectorAll("button")].find(
      (button) => button.textContent === "agent.api-permission-allow"
    );
    expect(allow).not.toBeDisabled();

    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(view.container.textContent).toContain("Expires in 4:59");

    act(() => {
      vi.setSystemTime(expiresAt - 1000);
      vi.advanceTimersByTime(1000);
    });
    expect(view.container.textContent).toContain("agent.api-permission-expired");
    expect(allow).toBeDisabled();
    expect(view.container.textContent).not.toContain("agent.api-permission-deny");
    expect(view.container.textContent).toContain("agent.api-permission-dismiss");
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    view.unmount();
  });

  test("dismisses an unavailable permission without restarting the agent", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "lost-action",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.resolvePendingApiCall.mockResolvedValue(
      '{"error":"Permission request expired."}'
    );

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      const dismiss = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-dismiss"
      );
      dismiss?.click();
    });

    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith({
      actionId: "lost-action",
      chatId,
      toolCallId: "tool-write",
      userName: "users/me@example.com",
      approved: false,
      signal: expect.any(AbortSignal),
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getPendingAsk(chatId)).toBeNull();
    expect(useAgentStore.getState().getMessages(chatId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-write",
          content: '{"error":"Permission request expired."}',
        }),
        expect.objectContaining({
          role: "assistant",
          content: "agent.api-permission-expired",
        }),
      ])
    );
    view.unmount();
  });

  test("does not restart the agent if permission expires during approval", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-expiring",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders"}}',
      expiresAt: Date.now() + 1000,
    });
    mocks.resolvePendingApiCall.mockResolvedValue(
      '{"error":"Permission request expired."}'
    );

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      const allow = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-allow"
      );
      allow?.click();
    });

    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith(
      expect.objectContaining({ approved: true })
    );
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getPendingAsk(chatId)).toBeNull();
    expect(useAgentStore.getState().getMessages(chatId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "assistant",
          content: "agent.api-permission-expired",
        }),
      ])
    );
    view.unmount();
  });

  test("returns a denied API request to the agent without dispatching it", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat
        ),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "tool-write",
          prompt: "bytebase.v1.ProjectService.UpdateProject",
          kind: "confirm",
          apiPermission: {
            actionId: "action-denied",
            operationId: "bytebase.v1.ProjectService.UpdateProject",
          },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders"}}',
      expiresAt: Date.now() + 5 * 60_000,
    });
    mocks.resolvePendingApiCall.mockResolvedValue(
      '{"error":"Permission request denied."}'
    );
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      const cancel = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.api-permission-deny"
      );
      cancel?.click();
    });

    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith({
      actionId: "action-denied",
      chatId,
      toolCallId: "tool-write",
      userName: "users/me@example.com",
      approved: false,
      signal: expect.any(AbortSignal),
    });
    expect(useAgentStore.getState().getMessages(chatId)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          role: "tool",
          toolCallId: "tool-write",
          content: '{"error":"Permission request denied."}',
        }),
      ])
    );
    view.unmount();
  });

  test("shows a single-line overlay placeholder and hides it when typing", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const placeholder = container.querySelector(
      "[data-agent-input-placeholder]"
    );
    expect(placeholder?.textContent).toBe(
      "agent.input-placeholder · agent.input-keyboard-hint"
    );
    expect(placeholder?.className).toContain("truncate");
    expect(placeholder?.className).toContain("pointer-events-none");
    expect(placeholder?.className).toContain("inset-y-0");
    expect(placeholder?.className).toContain("items-center");

    const textarea = container.querySelector("textarea");
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);

    act(() => {
      setTextareaValue(textarea as HTMLTextAreaElement, "hello");
      textarea?.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(
      container.querySelector("[data-agent-input-placeholder]")
    ).toBeNull();

    unmount();
  });

  test("uses a one-line textarea starting size beside the send button", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const row = container.querySelector("[data-agent-input-row]");
    expect(row?.className).toContain("items-end");
    const textarea = container.querySelector("textarea");
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);
    expect(textarea?.className).toContain("h-9");
    expect(textarea?.className).toContain("min-h-9");
    expect(textarea?.className).not.toContain("max-h-");
    expect(container.querySelector("[data-agent-send]")?.className).toContain(
      "h-9"
    );

    unmount();
  });

  test("keeps pasted line breaks in the textarea", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;

    act(() => {
      setTextareaValue(textarea, "first line\nsecond line");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(textarea.value).toContain("\n");
    unmount();
  });

  test("caps textarea growth at the available composer height", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    const composer = container.querySelector(
      "[data-agent-input-row]"
    )?.parentElement as HTMLDivElement;
    Object.defineProperties(textarea, {
      clientHeight: { configurable: true, value: 32 },
      offsetHeight: { configurable: true, value: 34 },
      scrollHeight: { configurable: true, value: 200 },
    });
    Object.defineProperties(composer, {
      offsetHeight: { configurable: true, value: 90 },
    });
    Object.defineProperties(container, {
      clientHeight: { configurable: true, value: 150 },
    });
    const getComputedStyleSpy = vi
      .spyOn(window, "getComputedStyle")
      .mockReturnValue({ minHeight: "34px" } as CSSStyleDeclaration);

    act(() => {
      setTextareaValue(textarea, "a\nb\nc\nd\ne\nf\ng");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(textarea.style.height).toBe("94px");
    expect(textarea.style.overflowY).toBe("auto");
    getComputedStyleSpy.mockRestore();
    unmount();
  });

  test("updates the textarea cap when the chat pane height changes", () => {
    let resizeCallback: ResizeObserverCallback | undefined;
    let observedElement: Element | undefined;
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(callback: ResizeObserverCallback) {
          resizeCallback = callback;
        }

        observe(element: Element) {
          observedElement = element;
        }

        disconnect() {}
      }
    );

    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    let chatPaneHeight = 300;
    Object.defineProperty(container, "clientHeight", {
      configurable: true,
      get: () => chatPaneHeight,
    });

    render();

    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    const composer = container.querySelector(
      "[data-agent-input-row]"
    )?.parentElement as HTMLDivElement;
    Object.defineProperties(textarea, {
      clientHeight: { configurable: true, value: 32 },
      offsetHeight: { configurable: true, value: 34 },
      scrollHeight: { configurable: true, value: 200 },
    });
    Object.defineProperties(composer, {
      offsetHeight: { configurable: true, value: 90 },
    });
    const getComputedStyleSpy = vi
      .spyOn(window, "getComputedStyle")
      .mockReturnValue({ minHeight: "34px" } as CSSStyleDeclaration);

    act(() => {
      setTextareaValue(textarea, "a\nb\nc\nd\ne\nf\ng");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(textarea.style.height).toBe("202px");
    expect(textarea.style.overflowY).toBe("hidden");
    expect(resizeCallback).toBeTypeOf("function");
    expect(observedElement).toBe(container);

    chatPaneHeight = 150;
    act(() => {
      resizeCallback?.([], {} as ResizeObserver);
    });

    expect(textarea.style.height).toBe("94px");
    expect(textarea.style.overflowY).toBe("auto");
    getComputedStyleSpy.mockRestore();
    unmount();
  });

  test("shrinks the textarea when composer chrome grows", () => {
    let resizeCallback: ResizeObserverCallback | undefined;
    const observedElements = new Set<Element>();
    const disconnect = vi.fn();
    vi.stubGlobal(
      "ResizeObserver",
      class {
        constructor(callback: ResizeObserverCallback) {
          resizeCallback = callback;
        }

        observe(element: Element) {
          observedElements.add(element);
        }

        disconnect() {
          disconnect();
        }
      }
    );

    const { container, render, unmount } = renderIntoContainer(<AgentInput />);
    Object.defineProperty(container, "clientHeight", {
      configurable: true,
      value: 300,
    });

    render();

    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    const composer = container.querySelector(
      "[data-agent-input-row]"
    )?.parentElement as HTMLDivElement;
    let composerHeight = 90;
    Object.defineProperties(textarea, {
      clientHeight: { configurable: true, value: 32 },
      offsetHeight: { configurable: true, value: 34 },
      scrollHeight: { configurable: true, value: 200 },
    });
    Object.defineProperty(composer, "offsetHeight", {
      configurable: true,
      get: () => composerHeight,
    });
    const getComputedStyleSpy = vi
      .spyOn(window, "getComputedStyle")
      .mockReturnValue({ minHeight: "34px" } as CSSStyleDeclaration);

    try {
      act(() => {
        setTextareaValue(textarea, "a\nb\nc\nd\ne\nf\ng");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });

      expect(textarea.style.height).toBe("202px");
      expect(textarea.style.overflowY).toBe("hidden");

      composerHeight = 240;
      act(() => {
        if (observedElements.has(composer)) {
          resizeCallback?.([], {} as ResizeObserver);
        }
      });

      expect(textarea.style.height).toBe("94px");
      expect(textarea.style.overflowY).toBe("auto");
    } finally {
      getComputedStyleSpy.mockRestore();
      unmount();
    }

    expect(disconnect).toHaveBeenCalledOnce();
  });
});
