import { fireEvent, screen } from "@testing-library/react";
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
import type { AgentMaskingPageResource } from "../logic/chatContext";
import type { AgentCallbacks, AgentLoopOptions } from "../logic/agentLoop";
import type { createToolExecutor } from "../logic/tools";
import type { AgentPendingAsk, ToolExecutor } from "../logic/types";
import { withAgentRunLease } from "../logic/runOwnership";
import { createAgentStore, getAgentStateKey, selectError, useAgentStore } from "../store/agent";
import {
  AgentDialog,
  AgentDialogContent,
  AgentDialogTitle,
} from "./ui/AgentDialog";

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
  aiEnabled: undefined as boolean | undefined,
  currentContext: "projects/orders" as string | undefined,
  currentPath: "/demo",
  currentPageResource: undefined as AgentMaskingPageResource | undefined,
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
    get serverInfo() {
      return { aiEnabled: mocks.aiEnabled };
    },
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
    workspaceResourceName: () => mocks.workspace?.name ?? "",
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

vi.mock("../logic/outboundHistory", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../logic/outboundHistory")>()),
  buildOutboundHistory: mocks.buildOutboundHistory,
}));

vi.mock("../logic/prompt", () => ({
  buildSystemPrompt: mocks.buildSystemPrompt,
}));

vi.mock("../logic/chatContext", () => ({
  getCurrentAgentPageResource: () => mocks.currentPageResource,
  getCurrentChatContext: () => mocks.currentContext,
  isAgentMaskingPageResource: (page: unknown) =>
    page !== undefined && page === mocks.currentPageResource,
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
    visible: initialState.visible,
    minimized: initialState.minimized,
    abortControllersByChatId: {},
    remoteBusyChatIds: {},
    sqlFixSourcesByChatId: {},
    sqlFixDraftsById: {},
    preferences: {
      sendWithEnter: true,
      tokenLimitPerRun: 60_000,
      openingBehavior: "landing",
    },
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
  mocks.currentPageResource = undefined;
  mocks.aiEnabled = undefined;

  ({ AgentInput } = await import("./AgentInput"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("AgentInput", () => {
  const openRunLimitIncrease = async (container: HTMLElement, limit: number) => {
    const increase = [...container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.increase");
    expect(increase).toBeDefined();
    await act(async () => { increase?.click(); });
    fireEvent.change(screen.getByRole("spinbutton", { name: "agent.token-budget.run-limit" }), { target: { value: String(limit) } });
    return screen.getByRole("button", { name: "agent.token-budget.increase-and-resume" });
  };

  const increaseManagedRunTokenLimit = async (chatId: string, limit: number) => {
    const store = useAgentStore.getState();
    const key = getAgentStateKey(store.workspaceName!, store.userEmail!);
    if (!key) throw new Error("Agent test scope is unavailable");
    const result = await withAgentRunLease(key, chatId, () =>
      store.increaseRunTokenLimit(chatId, limit)
    );
    expect(result.acquired).toBe(true);
    return result.value;
  };

  const enableRunRecovery = () => {
    vi.stubGlobal("navigator", {
      ...navigator,
      locks: {
        request: vi.fn(async (name: string, _options: unknown, callback: (lock: { name: string }) => unknown) => callback({ name })),
      },
    });
    vi.stubGlobal("sessionStorage", createMockStorage());
  };

  const seedUnrecoverableRun = async (kind: "legacy" | "incomplete") => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "projects/orders", page });
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
    if (kind === "legacy") {
      store.startChatRun(chat.id, page, { runId: "legacy-run" });
    } else {
      const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
      await withAgentRunLease(key, chat.id, () => store.beginChatRun(chat.id, root.id, {
        page, context: chat.context, references: [], runMode: "default",
      }));
    }
    store.setDraft(chat.id, { text: "Keep my next question" });
    await store.flushRunPersistence();
    return { store, chat };
  };

  test("does not show a continuation failure on an empty landing composer", () => {
    useAgentStore.getState().clearCurrentChat();
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
    } finally {
      view.unmount();
    }
  });

  test("clears unavailable continuation feedback after canceling a legacy question", async () => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
    store.awaitUser(chat.id, { kind: "input", toolCallId: "legacy-question", prompt: "Which database?" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      fireEvent.change(view.container.querySelector("textarea")!, { target: { value: "Orders" } });
      await act(async () => { view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click(); });
      expect(view.container.textContent).toContain("agent.token-budget.continuation-unavailable");
      await act(async () => { screen.getByRole("button", { name: "agent.cancel-request" }).click(); });
      expect(store.getPendingAsk(chat.id)).toBeNull();
      expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
    } finally {
      view.unmount();
    }
  });

  test.each(["legacy", "incomplete"] as const)("keeps the error and history when Retry cannot recover a %s run", async kind => {
    const { store, chat } = await seedUnrecoverableRun(kind);
    store.setChatStatus(chat.id, "error", { lastError: "Connection lost" });
    const before = store.getChat(chat.id);
    const history = store.getMessages(chat.id);
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      await act(async () => { retryActionRef.current?.(); });
      expect(store.getChat(chat.id)).toEqual(before);
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(view.container.textContent).toContain("agent.token-budget.continuation-unavailable");
    } finally {
      view.unmount();
    }
  });

  test.each([
    ["legacy", "input"], ["legacy", "confirm"], ["legacy", "choose"],
    ["incomplete", "input"], ["incomplete", "confirm"], ["incomplete", "choose"],
  ] as const)("preserves a %s run's %s question when its continuation is unavailable", async (kind, askKind) => {
    const { store, chat } = await seedUnrecoverableRun(kind);
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{
      id: "unrecoverable-ask", name: "ask_user", arguments: JSON.stringify({ kind: askKind, prompt: "Which database?" }),
    }] });
    const ask = { kind: askKind, toolCallId: "unrecoverable-ask", prompt: "Which database?", ...(askKind === "choose" ? { options: [{ value: "orders", label: "Orders" }] } : {}) };
    store.awaitUser(chat.id, ask);
    const history = store.getMessages(chat.id);
    const before = store.getChat(chat.id);
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      if (askKind === "input") {
        fireEvent.change(view.container.querySelector("textarea")!, { target: { value: "Use orders" } });
        await act(async () => { view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click(); });
        expect(view.container.querySelector("textarea")?.value).toBe("Use orders");
      } else if (askKind === "confirm") {
        await act(async () => { screen.getByRole("button", { name: "agent.confirm" }).click(); });
      } else {
        fireEvent.click(screen.getByRole("radio", { name: "Orders" }));
        await act(async () => { screen.getByRole("button", { name: "agent.reply" }).click(); });
        expect(screen.getByRole("radio", { name: "Orders" })).toBeChecked();
      }
      expect(store.getPendingAsk(chat.id)).toEqual(ask);
      expect(store.getChat(chat.id)).toEqual(before);
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(view.container.textContent).toContain("agent.token-budget.continuation-unavailable");
    } finally {
      view.unmount();
    }
  });

  test.each([
    ["legacy", true], ["legacy", false],
    ["incomplete", true], ["incomplete", false],
  ] as const)("handles a %s run's available API permission with approved=%s without unsafe continuation", async (kind, approved) => {
    const { store, chat } = await seedUnrecoverableRun(kind);
    const ask: AgentPendingAsk = {
      kind: "confirm", toolCallId: "unrecoverable-permission", prompt: "Update the project title?",
      apiPermission: { actionId: "unrecoverable-action", operationId: "bytebase.v1.ProjectService.UpdateProject" },
    };
    mocks.getPendingApiCallPreview.mockReturnValue({ operationId: ask.apiPermission!.operationId, body: "{}", expiresAt: Date.now() + 300_000 });
    const result = '{"error":"API request denied by user."}';
    mocks.resolvePendingApiCall.mockResolvedValue(result);
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "call_api", arguments: "{}" }] });
    store.awaitUser(chat.id, ask);
    const before = store.getChat(chat.id);
    const history = store.getMessages(chat.id);
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { screen.getByRole("button", { name: approved ? "agent.api-permission-allow" : "agent.api-permission-deny" }).click(); });
      if (approved) {
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
        expect(store.getChat(chat.id)).toEqual(before);
        expect(store.getPendingAsk(chat.id)).toEqual(ask);
        expect(store.getMessages(chat.id)).toEqual(history);
      } else {
        expect(mocks.resolvePendingApiCall).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ actionId: "unrecoverable-action", approved: false }));
        expect(store.getChat(chat.id)).toMatchObject({ status: "idle", runId: null });
        expect(store.getChat(chat.id)?.runBudget).toEqual(before?.runBudget);
        expect(store.getChat(chat.id)?.runCheckpoint).toBeUndefined();
        expect(store.getPendingAsk(chat.id)).toBeNull();
        expect(store.getMessages(chat.id).filter(message => message.toolCallId === ask.toolCallId)).toEqual([
          expect.objectContaining({ role: "tool", content: result, metadata: expect.objectContaining({ runId: before?.runId }) }),
        ]);
      }
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getAbortController(chat.id)).toBeNull();
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
    } finally {
      view.unmount();
    }
  });

  test("does not report missing usage while its locally owned provider attempt is pending", async () => {
    enableRunRecovery();
    let finish: (() => void) | undefined;
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, _callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await new Promise<void>((resolve) => { finish = resolve; });
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 10 },
      });
      return { kind: "completed", success: true, totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Check the page.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(finish).toBeTypeOf("function")); });
      expect(store.getChat(chat.id)?.runBudget?.hasUnreportedUsage).toBe(true);
      expect(view.container.textContent).not.toContain("agent.token-budget.partial");
      view.render(<div />);
      view.render(<AgentInput />);
      expect(view.container.textContent).not.toContain("agent.token-budget.partial");
    } finally {
      await act(async () => { finish?.(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      view.unmount();
    }
  });

  test("disables budget recovery controls when durable execution becomes unavailable", async () => {
    enableRunRecovery();
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, _callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 60_000 },
      });
      await options.onCheckpoint({ phase: "working" });
      return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Check the page.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      expect(await store.flushRunPersistence()).toBe(true);
      vi.stubGlobal("navigator", { ...navigator, locks: undefined });
      act(() => store.setDraft(chat.id, { text: "Keep this draft." }));
      expect(view.container.textContent).toContain("agent.token-budget.recovery-unavailable");
      for (const label of ["increase", "end-run"]) {
        const button = [...view.container.querySelectorAll("button")].find(candidate => candidate.textContent === `agent.token-budget.${label}`);
        expect(button).toBeDefined();
        expect(button?.disabled).toBe(true);
      }
      expect(view.container.textContent).not.toContain("agent.token-budget.resume");
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
    } finally {
      view.unmount();
    }
  });

  test("increases and resumes the same logical run without clearing its draft or repeating startup", async () => {
    enableRunRecovery();
    let logicalRunId: string | undefined;
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      expect(options.reserveProviderAttempt).toBeTypeOf("function");
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      logicalRunId ??= options.runId;
      expect(options.runId).toBe(logicalRunId);
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, runId: options.runId, ordinal: reservation.ordinal,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 60_000 },
      });
      await options.onCheckpoint({ phase: "working" });
      if (reservation.ordinal === 1) return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      callbacks.onText("The review is complete.");
      return { kind: "completed", success: true, totalTokensUsed: 60_000 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/billing", title: "Billing" };
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review this project.", page, resourceReferences: [{ kind: "project", name: "projects/billing", project: "projects/billing", label: "Billing" }] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
    expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
    expect(store.getMessages(chat.id).some(message => message.content?.includes("usage limit"))).toBe(false);
    const startupCalls = mocks.executeTool.mock.calls.length;
    act(() => store.setDraft(chat.id, { text: "Keep this draft." }));
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    act(() => {
      textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    });
    expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
    expect(view.container.textContent).not.toContain("agent.token-budget.resume");
    const confirm = await openRunLimitIncrease(view.container, 150);
    await act(async () => { confirm.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
    expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
    expect(mocks.executeTool).toHaveBeenCalledTimes(startupCalls);
    expect(store.getDraft(chat.id).text).toBe("Keep this draft.");
    expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
    expect(store.getChat(chat.id)?.runBudget?.totalTokensUsed).toBe(120_000);
    expect(store.getChat(chat.id)?.runBudget?.limit).toBe(150_000);
    expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(60_000);
    view.unmount();
  });

  test.each(["persistence", "lease", "resume-lease", "archived"] as const)("does not resume after a combined increase is blocked by %s", async failure => {
    enableRunRecovery();
    mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, _callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 60_000 },
      });
      await options.onCheckpoint({ phase: "working" });
      return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
    });
    const store = useAgentStore.getState();
    const flushRunPersistence = store.flushRunPersistence;
    const increaseRunTokenLimit = store.increaseRunTokenLimit;
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review the project.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      act(() => store.setDraft(chat.id, { text: "Keep this draft." }));
      const confirm = await openRunLimitIncrease(view.container, 100);
      if (failure === "persistence") {
        useAgentStore.setState({ flushRunPersistence: async () => false });
      } else if (failure === "lease") {
        vi.stubGlobal("navigator", {
          ...navigator,
          locks: { request: vi.fn(async (name: string, options: { ifAvailable?: boolean }, callback: (lock: { name: string } | null) => unknown) => callback(options.ifAvailable ? null : { name })) },
        });
      } else {
        useAgentStore.setState({ increaseRunTokenLimit: async (chatId, limit) => {
          const saved = await increaseRunTokenLimit(chatId, limit);
          if (failure === "archived") store.archiveChat(chatId);
          else vi.stubGlobal("navigator", {
            ...navigator,
            locks: { request: vi.fn(async (name: string, options: { ifAvailable?: boolean }, callback: (lock: { name: string } | null) => unknown) => callback(options.ifAvailable ? null : { name })) },
          });
          return saved;
        } });
      }
      await act(async () => { confirm.click(); });
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(store.getDraft(chat.id).text).toBe("Keep this draft.");
      expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
      expect(useAgentStore.getState().preferences.tokenLimitPerRun).toBe(60_000);
      if (failure !== "archived") {
        expect(store.getChat(chat.id)?.status).toBe("paused");
        if (failure !== "resume-lease") expect(screen.getByRole("spinbutton")).toHaveValue(100);
        else expect(screen.queryByRole("spinbutton")).toBeNull();
        expect(screen.getByText("agent.token-budget.action-failed")).toBeVisible();
      } else {
        expect(store.getChat(chat.id)?.archived).toBe(true);
      }
    } finally {
      useAgentStore.setState({ flushRunPersistence, increaseRunTokenLimit });
      view.unmount();
    }
  });

  test("keeps clarification and Retry in the same logical run", async () => {
    enableRunRecovery();
    const runIds: string[] = [];
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      runIds.push(options.runId);
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.ordinal).toBe(runIds.length);
      await options.onAttemptSettled({ attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 10 } });
      if (runIds.length === 1) {
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: "clarification", name: "ask_user", arguments: '{"kind":"input","prompt":"Which database?"}' }] });
        await options.onCheckpoint({ phase: "working" });
        return { kind: "awaiting_user", ask: { kind: "input", toolCallId: "clarification", prompt: "Which database?" }, totalTokensUsed: 10 };
      }
      await options.onCheckpoint({ phase: "working" });
      if (runIds.length === 2) return { kind: "error", error: new Error("temporary failure"), totalTokensUsed: 10 };
      return { kind: "completed", success: true, totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Check the database.", page });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    await act(async () => { setTextareaValue(textarea, "orders"); textarea.dispatchEvent(new Event("input", { bubbles: true })); });
    const send = view.container.querySelector("button[data-agent-send]") as HTMLButtonElement | null;
    await act(async () => {
      if (send) send.click();
      else textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
      await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error"));
    });
    await act(async () => { retryActionRef.current?.(); await vi.waitFor(() => expect(runIds).toHaveLength(3)); });
    expect(new Set(runIds).size).toBe(1);
    expect(store.getChat(chat.id)?.runBudget?.providerAttemptCount).toBe(3);
    view.unmount();
  });

  test("refines a recovered clarification with its explicit database without rebinding to navigation", async () => {
    enableRunRecovery();
    const database = { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders" };
    const ask = { kind: "input" as const, toolCallId: "choose-database", prompt: "Which database?", defaultValue: database.name, defaultResource: database };
    let runId: string | undefined;
    let resumedCheckpoint: unknown;
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: database.project, database: { name: database.name, status: "available", state: "ACTIVE" } }) });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      if (!runId) {
        runId = options.runId;
        const reservation = await options.reserveProviderAttempt();
        await options.onAttemptSettled({ attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 40 } });
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "ask_user", arguments: JSON.stringify({ kind: "input", prompt: ask.prompt }) }] });
        return { kind: "awaiting_user", ask, totalTokensUsed: 40 };
      }
      resumedCheckpoint = options.checkpoint;
      callbacks.onText("Ready to inspect orders.");
      return { kind: "completed", success: true, totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({ chatId: chat.id, text: "Help me query a database", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      mocks.currentPath = "/projects/billing";
      await act(async () => {
        view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
      expect(resumedCheckpoint).toMatchObject({ runId, bindings: { page, context: "workspaces/demo", references: [database], runMode: "read-only" } });
      expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ pagePath: page.path, context: "workspaces/demo", referencedDatabase: database.name, referencedProject: database.project, runMode: "read-only" }));
      expect(mocks.executeTool).toHaveBeenCalledExactlyOnceWith("get_database_overview", { databaseName: database.name }, expect.any(String));
      expect(store.getMessages(chat.id).filter(message => message.metadata?.runPreflight)).toHaveLength(2);
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ runId, totalTokensUsed: 40, providerAttemptCount: 1 });
      expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
    } finally {
      view.unmount();
    }
  });

  test("revalidates a clarified database on Retry when continuing could not persist", async () => {
    enableRunRecovery();
    const database = { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders" };
    const ask = { kind: "input" as const, toolCallId: "choose-database", prompt: "Which database?", defaultValue: database.name, defaultResource: database };
    const store = useAgentStore.getState();
    const flushRunPersistence = store.flushRunPersistence;
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: database.project, database: { name: database.name, status: "available", state: "ACTIVE" } }) });
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
        const reservation = await options.reserveProviderAttempt();
        await options.onAttemptSettled({ attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 40 } });
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "ask_user", arguments: JSON.stringify({ kind: "input", prompt: ask.prompt }) }] });
        return { kind: "awaiting_user", ask, totalTokensUsed: 40 };
      })
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("The chosen database is available.");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      });
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({ chatId: chat.id, text: "Help me query a database", page });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      const runId = store.getChat(chat.id)?.runId;
      useAgentStore.setState({ flushRunPersistence: vi.fn(async () => false) });
      await act(async () => {
        view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error"));
      });
      expect(store.getChat(chat.id)?.runCheckpoint?.bindings.references).toEqual([database]);
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      useAgentStore.setState({ flushRunPersistence });
      await act(async () => {
        retryActionRef.current?.();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(mocks.executeTool).toHaveBeenCalledExactlyOnceWith("get_database_overview", { databaseName: database.name }, expect.any(String));
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
      expect(mocks.executeTool.mock.invocationCallOrder[0]).toBeLessThan(mocks.runAgentLoop.mock.invocationCallOrder[1]);
      expect(store.getMessages(chat.id).filter(message => message.metadata?.runPreflight)).toHaveLength(2);
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ runId, totalTokensUsed: 40, providerAttemptCount: 1 });
    } finally {
      useAgentStore.setState({ flushRunPersistence });
      view.unmount();
    }
  });

  test.each([enUS, zhCN, esES, jaJP, viVN])("localizes a provider-attempt stop at the UI boundary", async locale => {
    mocks.useTranslation.mockReturnValue({ t: key => key === "agent.provider-attempt-limit" ? locale.agent["provider-attempt-limit"] : key, i18n: { language: "test" } });
    mocks.runAgentLoop.mockResolvedValue({ kind: "stopped", reason: "provider-attempt-limit", totalTokensUsed: 10 });
    const store = useAgentStore.getState();
    const page = { path: "/demo", title: "Demo" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Check orders", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      expect(store.getMessages(chat.id).at(-1)).toMatchObject({ role: "assistant", content: locale.agent["provider-attempt-limit"] });
    } finally {
      view.unmount();
    }
  });

  test("requires explicit API Allow at the run limit and keeps budget Resume from approving again", async () => {
    enableRunRecovery();
    const ask: AgentPendingAsk = {
      kind: "confirm",
      toolCallId: "budget-write",
      prompt: "Update the project title?",
      apiPermission: { actionId: "budget-action", operationId: "bytebase.v1.ProjectService.UpdateProject" },
    };
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders","title":"Orders"}}',
      expiresAt: Date.now() + 300_000,
    });
    const runIds: string[] = [];
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      runIds.push(options.runId);
      const reservation = await options.reserveProviderAttempt();
      if (runIds.length === 2) {
        expect(reservation.kind).toBe("token-limit");
        return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      }
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: runIds.length === 1 ? 60_000 : 10 },
      });
      if (runIds.length === 1) {
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "call_api", arguments: "{}" }] });
        await options.onCheckpoint({ phase: "working" });
        return { kind: "awaiting_user", ask, totalTokensUsed: 60_000 };
      }
      callbacks.onText("The project was updated.");
      return { kind: "completed", success: true, totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Update the project title.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(view.container.textContent).not.toContain("agent.token-budget.resume");
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ totalTokensUsed: 60_000, providerAttemptCount: 1 });
      const allow = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.api-permission-allow");
      expect(allow).toBeDefined();
      await act(async () => { allow?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ actionId: "budget-action", approved: true }));
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ totalTokensUsed: 60_000, providerAttemptCount: 1 });
      expect(store.getPendingAsk(chat.id)).toBeNull();
      await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.resume");
      expect(resume).toBeDefined();
      await act(async () => { resume?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      expect(new Set(runIds).size).toBe(1);
      expect(runIds).toHaveLength(3);
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
      expect(store.getMessages(chat.id).filter(message => message.toolCallId === ask.toolCallId)).toHaveLength(1);
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ totalTokensUsed: 60_010, providerAttemptCount: 2 });
    } finally {
      view.unmount();
    }
  });

  test("retains an approved API result arriving after Stop without automatically replaying it", async () => {
    enableRunRecovery();
    const ask: AgentPendingAsk = {
      kind: "confirm", toolCallId: "stopped-write", prompt: "Update the project title?",
      apiPermission: { actionId: "stopped-action", operationId: "bytebase.v1.ProjectService.UpdateProject" },
    };
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject", body: "{}", expiresAt: Date.now() + 300_000,
    });
    mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
      callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "call_api", arguments: "{}" }] });
      await options.onCheckpoint({ phase: "working" });
      return { kind: "awaiting_user", ask, totalTokensUsed: 0 };
    });
    let settleApproval!: (result: string) => void;
    mocks.resolvePendingApiCall.mockImplementationOnce(() => new Promise<string>(resolve => { settleApproval = resolve; }));
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Update the project title.", page });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      const runId = store.getChat(chat.id)?.runId;
      await act(async () => {
        [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.api-permission-allow")?.click();
        await vi.waitFor(() => expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce());
      });
      act(() => { [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.stop")?.click(); });
      expect(store.getChat(chat.id)?.interrupted).toBe(true);
      expect(store.getChat(chat.id)?.interruptionReason).toBe("user");
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      const result = '{"status":200,"response":{"name":"projects/orders","title":"Updated"}}';
      await act(async () => { settleApproval(result); });
      expect(store.getMessages(chat.id)).toEqual(expect.arrayContaining([
        expect.objectContaining({ role: "tool", toolCallId: ask.toolCallId, content: result, metadata: expect.objectContaining({ runId }) }),
      ]));
      expect(store.getChat(chat.id)?.interrupted).toBe(true);
      expect(store.getChat(chat.id)?.status).toBe("idle");
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
      expect(await store.flushRunPersistence()).toBe(true);
      expect(store.getPendingAsk(chat.id)).toBeNull();
      mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
        expect(options.runId).toBe(runId);
        expect(options.checkpoint.runId).toBe(runId);
        expect(store.getMessages(chat.id).filter(message => message.toolCallId === ask.toolCallId)).toHaveLength(1);
        callbacks.onText("The recorded update succeeded.");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      });
      await act(async () => {
        retryActionRef.current?.();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
      });
      expect(store.getChat(chat.id)?.status).toBe("idle");
      expect(store.getChat(chat.id)?.interrupted).toBe(false);
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
    } finally {
      view.unmount();
    }
  });

  test("ends a paused run without losing its draft and gives the next Send a fresh budget", async () => {
    enableRunRecovery();
    const runIds: string[] = [];
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      runIds.push(options.runId);
      const reservation = await options.reserveProviderAttempt();
      expect(reservation).toMatchObject({ kind: "allowed", ordinal: 1 });
      expect(useAgentStore.getState().getChat(chat.id)?.runBudget?.totalTokensUsed).toBe(0);
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: runIds.length === 1 ? 60_000 : 12 },
      });
      await options.onCheckpoint({ phase: "working" });
      if (runIds.length === 1) return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      callbacks.onText("This is a new request.");
      return { kind: "completed", success: true, totalTokensUsed: 12 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review the project.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      act(() => store.setDraft(chat.id, { text: "Start a separate review." }));
      const end = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.end-run");
      expect(end).toBeDefined();
      await act(async () => { end?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.runId).toBeNull()); });
      expect(store.getDraft(chat.id).text).toBe("Start a separate review.");
      expect(store.getChat(chat.id)?.runCheckpoint).toBeUndefined();
      expect(store.getChat(chat.id)?.runBudget?.totalTokensUsed).toBe(60_000);
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      await act(async () => {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
      });
      expect(new Set(runIds).size).toBe(2);
      expect(store.getMessages(chat.id).filter(message => message.role === "user").map(message => message.content)).toEqual(["Review the project.", "Start a separate review."]);
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ runId: runIds[1], limit: 60_000, totalTokensUsed: 12, providerAttemptCount: 1 });
      expect(store.getChat(chat.id)?.totalTokensUsed).toBe(60_012);
    } finally {
      view.unmount();
    }
  });

  test("resumes finalization with original resource bindings after navigation without repeating startup", async () => {
    enableRunRecovery();
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: reservation.ordinal === 1 ? 60_000 : 10 },
      });
      if (reservation.ordinal === 1) {
        await options.onCheckpoint({ phase: "finalizing", finalizationText: "The original project review." });
        return { kind: "paused", reason: "token-limit", phase: "finalizing", finalizationText: "The original project review.", totalTokensUsed: 60_000 };
      }
      expect(options.checkpoint).toMatchObject({ phase: "finalizing", finalizationText: "The original project review.", bindings: { page, context: "projects/billing", references: [reference], runMode: "read-only" } });
      callbacks.onText("The original project review.");
      return { kind: "completed", success: true, totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/billing", title: "Billing" };
    const reference = { kind: "project" as const, name: "projects/billing", project: "projects/billing", label: "Billing" };
    const chat = store.createChat({ context: "projects/billing", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review the project.", page, resourceReferences: [reference] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      const startupCalls = mocks.executeTool.mock.calls.length;
      expect(startupCalls).toBeGreaterThan(0);
      mocks.currentPath = "/projects/orders";
      mocks.currentContext = "projects/orders";
      act(() => store.updateChatPage(chat.id, { path: mocks.currentPath, title: "Orders" }));
      await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.resume");
      expect(resume).toBeDefined();
      await act(async () => { resume?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
      expect(mocks.executeTool).toHaveBeenCalledTimes(startupCalls);
      expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(expect.objectContaining({ path: page.path, context: "projects/billing", referencedProject: "projects/billing", runMode: "read-only" }));
      expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ context: "projects/billing", referencedProject: "projects/billing", pagePath: page.path }));
      expect(store.getMessages(chat.id).find(message => message.content === "The original project review.")?.metadata?.route).toBe(page.path);
    } finally {
      view.unmount();
    }
  });

  test("blocks explicit API approval when its durable recovery state cannot be flushed", async () => {
    enableRunRecovery();
    const ask: AgentPendingAsk = {
      kind: "confirm", toolCallId: "durable-write", prompt: "Update the project title?",
      apiPermission: { actionId: "durable-action", operationId: "bytebase.v1.ProjectService.UpdateProject" },
    };
    mocks.getPendingApiCallPreview.mockReturnValue({ operationId: "bytebase.v1.ProjectService.UpdateProject", body: "{}", expiresAt: Date.now() + 300_000 });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "call_api", arguments: "{}" }] });
      await options.onCheckpoint({ phase: "working" });
      return { kind: "awaiting_user", ask, totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Update the project title.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      vi.spyOn(localStorage, "setItem").mockImplementation(() => { throw new Error("Storage unavailable"); });
      act(() => store.updateChatPage(chat.id, { ...page, title: "Updated page title" }));
      expect(await store.flushRunPersistence()).toBe(false);
      await act(async () => { [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.api-permission-allow")?.click(); });
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
      expect(store.getChat(chat.id)?.status).toBe("awaiting_user");
    } finally {
      view.unmount();
    }
  });

  test("retains the prepared masking draft action across budget Resume without preparing it again", async () => {
    enableRunRecovery();
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: reservation.ordinal === 1 ? 60_000 : 10 },
      });
      if (reservation.ordinal === 1) {
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: "prepared-mask", name: "prepare_masking_rule_draft", arguments: "{}" }] });
        await callbacks.onToolResult("prepared-mask", '{"draftId":"prepared-before-pause"}');
        await options.onCheckpoint({ phase: "working" });
        return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      }
      callbacks.onText("Review the prepared masking draft.");
      return { kind: "completed", success: true, actions: [], totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/settings/global-masking", title: "Global Masking" };
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({ chatId: chat.id, text: "Prepare an email masking rule.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      const executionCount = mocks.executeTool.mock.calls.length;
      await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.resume");
      expect(resume).toBeDefined();
      await act(async () => { resume?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      expect(store.getMessages(chat.id).find(message => message.content === "Review the prepared masking draft.")?.metadata?.actions).toEqual([
        { kind: "draft-masking-rule", draftId: "prepared-before-pause" },
      ]);
      expect(store.getMessages(chat.id).filter(message => message.toolCallId === "prepared-mask")).toHaveLength(1);
      expect(mocks.executeTool).toHaveBeenCalledTimes(executionCount);
      expect(mocks.routerPush).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("does not restore evidence or actions invalidated by a later write in the same tool group", async () => {
    enableRunRecovery();
    const issue = "projects/orders/issues/42";
    const writeArgs = { operationId: "bytebase.v1.IssueService.UpdateIssue", body: { issue: { name: issue, title: "Updated" } } };
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, execute, callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: reservation.ordinal === 1 ? 60_000 : 10 },
      });
      if (reservation.ordinal === 1) {
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [
          { id: "before-write-overview", name: "get_issue_overview", arguments: JSON.stringify({ issue }) },
          { id: "before-write-draft", name: "prepare_masking_rule_draft", arguments: "{}" },
          { id: "invalidating-write", name: "call_api", arguments: JSON.stringify(writeArgs) },
        ] });
        await callbacks.onToolResult("before-write-overview", JSON.stringify({
          project: "projects/orders", issue: { status: "available", name: issue },
          evidence: { project: "projects/orders", references: [{ kind: "issue", name: issue }] },
        }));
        await callbacks.onToolResult("before-write-draft", '{"draftId":"stale-before-write"}');
        await execute("call_api", writeArgs, "invalidating-write");
        await callbacks.onToolResult("invalidating-write", '{"status":200,"response":{}}');
        await options.onCheckpoint({ phase: "working" });
        return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      }
      callbacks.onText("The update is complete.", { kind: "open-issue", issue });
      return { kind: "completed", success: true, actions: [
        { kind: "open", target: { kind: "open-issue", issue } },
        { kind: "draft-masking-rule", draftId: "stale-before-write" },
      ], totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review and update the issue.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      const executionCount = mocks.executeTool.mock.calls.length;
      await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.resume");
      expect(resume).toBeDefined();
      await act(async () => { resume?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      const answer = store.getMessages(chat.id).find(message => message.content === "The update is complete.");
      expect(answer).toBeDefined();
      expect(answer?.metadata?.evidence).toBeUndefined();
      expect(answer?.metadata?.handoff).toBeUndefined();
      expect(answer?.metadata?.actions).toBeUndefined();
      expect(mocks.executeTool).toHaveBeenCalledTimes(executionCount);
      expect(mocks.routerPush).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("shows a recoverable error when beginning the run cannot persist instead of leaving it running", async () => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const flushRunPersistence = store.flushRunPersistence;
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.setDraft(chat.id, { text: "Keep my next question." });
    store.queueUserMessage({ chatId: chat.id, text: "Review the project.", page });
    useAgentStore.setState({ flushRunPersistence: vi.fn(async () => false) });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
      expect(store.getChat(chat.id)?.lastError).toBe("agent.token-budget.action-failed");
      expect(selectError(useAgentStore.getState())).toBe("agent.token-budget.action-failed");
      expect(store.getAbortController(chat.id)).toBeNull();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(store.getMessages(chat.id).filter(message => message.role === "user").map(message => message.content)).toEqual(["Review the project."]);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question.");
    } finally {
      useAgentStore.setState({ flushRunPersistence });
      view.unmount();
    }
  });

  test.each(["continue", "pause"] as const)("shows a recoverable error when %s cannot persist and keeps the known transcript", async failurePoint => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const flushRunPersistence = store.flushRunPersistence;
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 60_000 },
      });
      callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: "known-before-failure", name: "read_file", arguments: "{}" }] });
      await callbacks.onToolResult("known-before-failure", '{"known":"retained"}');
      await options.onCheckpoint({ phase: "working" });
      store.setDraft(chat.id, { text: "Keep my next question." });
      if (failurePoint === "pause") useAgentStore.setState({ flushRunPersistence: vi.fn(async () => false) });
      return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
    });
    store.queueUserMessage({ chatId: chat.id, text: "Review the project.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      if (failurePoint === "continue") {
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
        await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
        useAgentStore.setState({ flushRunPersistence: vi.fn(async () => false) });
        const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.token-budget.resume");
        expect(resume).toBeDefined();
        await act(async () => { resume?.click(); });
      }
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
      expect(store.getChat(chat.id)?.lastError).toBe("agent.token-budget.action-failed");
      expect(selectError(useAgentStore.getState())).toBe("agent.token-budget.action-failed");
      expect(store.getAbortController(chat.id)).toBeNull();
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(store.getMessages(chat.id).filter(message => message.toolCallId === "known-before-failure")).toEqual([
        expect.objectContaining({ role: "tool", content: '{"known":"retained"}' }),
      ]);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question.");
      expect(store.getChat(chat.id)?.runBudget?.totalTokensUsed).toBe(60_000);
    } finally {
      useAgentStore.setState({ flushRunPersistence });
      view.unmount();
    }
  });

  test("does not cancel a clarification when another tab owns the execution lease", async () => {
    enableRunRecovery();
    const ask: AgentPendingAsk = { kind: "input", toolCallId: "lease-clarification", prompt: "Which database?" };
    mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
      callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "ask_user", arguments: JSON.stringify({ kind: "input", prompt: ask.prompt }) }] });
      await options.onCheckpoint({ phase: "working" });
      return { kind: "awaiting_user", ask, totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Review the database.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
      const runId = store.getChat(chat.id)?.runId;
      vi.stubGlobal("navigator", {
        ...navigator,
        locks: { request: vi.fn(async (name: string, options: { ifAvailable?: boolean }, callback: (lock: { name: string } | null) => unknown) => callback(options.ifAvailable ? null : { name })) },
      });
      await act(async () => { [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.cancel-request")?.click(); });
      expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
      expect(store.getChat(chat.id)?.status).toBe("awaiting_user");
      expect(store.getChat(chat.id)?.runId).toBe(runId);
      expect(store.getMessages(chat.id).some(message => message.toolCallId === ask.toolCallId)).toBe(false);
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      enableRunRecovery();
      await act(async () => { [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.cancel-request")?.click(); });
      expect(store.getPendingAsk(chat.id)).toBeNull();
      expect(store.getMessages(chat.id).filter(message => message.toolCallId === ask.toolCallId)).toEqual([
        expect.objectContaining({ role: "tool", content: JSON.stringify({ kind: "cancelled", reason: "user" }) }),
      ]);
    } finally {
      view.unmount();
    }
  });

  test("resolves masking tools and registered guidance after a trusted skill load", async () => {
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _execute, _callbacks, _signal, options) => {
        expect(options.resolveRequestContext().systemGuidance).toBe("");
        const executorCalls = mocks.createToolExecutor.mock.calls as unknown as
          Parameters<typeof createToolExecutor>[];
        const executorOptions = executorCalls.at(-1)?.[1];
        expect(executorOptions?.onSkillLoaded).toBeTypeOf("function");
        executorOptions?.onSkillLoaded?.({
          kind: "loaded",
          name: "data-masking",
          section: "rule",
          content: "ignored result text",
        });
        const context = options.resolveRequestContext();
        expect(context.systemGuidance).toContain("# Data Masking");
        expect(context.systemGuidance).not.toContain("ignored result text");
        expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith(
          expect.objectContaining({ maskingSections: ["rule"] })
        );
        return { kind: "completed", totalTokensUsed: 0 };
      }
    );
    const store = useAgentStore.getState();
    store.setDraft(store.currentChatId!, { text: "Help with masking" });
    useAgentStore.setState({
      preferences: { ...store.preferences, sendWithEnter: true },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        view.container.querySelector("textarea")!.dispatchEvent(
          new KeyboardEvent("keydown", {
            key: "Enter",
            bubbles: true,
            cancelable: true,
          })
        );
      });
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      expect(
        useAgentStore.getState().getChat(store.currentChatId!)?.status
      ).not.toBe("error");
    } finally {
      view.unmount();
    }
  });

  test.each([
    { sendWithEnter: true, key: "Enter", sends: true },
    { sendWithEnter: true, key: "Enter", shiftKey: true, sends: false },
    { sendWithEnter: true, key: "Enter", ctrlKey: true, sends: false },
    { sendWithEnter: false, key: "Enter", sends: false },
    { sendWithEnter: false, key: "Enter", shiftKey: true, sends: false },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, sends: true },
    { sendWithEnter: false, key: "Enter", metaKey: true, sends: true },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, altKey: true, sends: false },
    { sendWithEnter: false, key: "Enter", metaKey: true, shiftKey: true, sends: false },
    { sendWithEnter: true, key: "Enter", keyCode: 229, sends: false },
    { sendWithEnter: false, key: "Enter", ctrlKey: true, isComposing: true, sends: false },
    { sendWithEnter: false, key: "Enter", metaKey: true, keyCode: 229, sends: false },
  ])("honors the configured send shortcut for %j", async ({ sendWithEnter, sends, ...event }) => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    store.setDraft(store.currentChatId!, { text: "A question" });
    useAgentStore.setState({ preferences: { ...store.preferences, sendWithEnter } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      const textarea = view.container.querySelector("textarea")!;
      let allowsDefault = false;
      await act(async () => {
        allowsDefault = textarea.dispatchEvent(new KeyboardEvent("keydown", { ...event, bubbles: true, cancelable: true }));
      });
      if (sends) await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());

      expect(allowsDefault).toBe(!sends);
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(sends ? 1 : 0);
      expect(useAgentStore.getState().getMessages(store.currentChatId!)).toEqual(
        sends ? expect.arrayContaining([expect.objectContaining({ role: "user", content: "A question" })]) : []
      );
    } finally {
      view.unmount();
    }
  });

  test("updates the keyboard hint when the send preference changes", async () => {
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      expect(view.container.querySelector("[data-agent-input-placeholder]")).toHaveTextContent("agent.input-keyboard-hint");
      await act(async () => useAgentStore.setState({ preferences: { ...useAgentStore.getState().preferences, sendWithEnter: false } }));
      expect(view.container.querySelector("[data-agent-input-placeholder]")).toHaveTextContent("agent.input-keyboard-hint-newline");
    } finally {
      view.unmount();
    }
  });

  test("uses the configured shortcut when replying to a clarification", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.awaitUser(chatId, { toolCallId: "clarification", kind: "input", prompt: "Which database?" });
    useAgentStore.setState({ preferences: { ...store.preferences, sendWithEnter: false } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      const textarea = view.container.querySelector("textarea")!;
      await act(async () => {
        setTextareaValue(textarea, "The orders database");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await act(async () => textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true })));
      expect(useAgentStore.getState().pendingAskByChatId[chatId]).toBeDefined();
      await act(async () => textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", ctrlKey: true, bubbles: true, cancelable: true })));
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      expect(useAgentStore.getState().pendingAskByChatId[chatId]).toBeUndefined();
    } finally {
      view.unmount();
    }
  });

  test("snapshots the token limit for each run and applies changes only to the next run", async () => {
    let finishRun!: (outcome: { kind: "completed"; totalTokensUsed: number }) => void;
    mocks.runAgentLoop.mockImplementationOnce(() => new Promise((resolve) => { finishRun = resolve; }));
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.setDraft(chatId, { text: "First question" });
    useAgentStore.setState({ preferences: { ...store.preferences, tokenLimitPerRun: 24_000 } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click());
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      expect(mocks.runAgentLoop.mock.calls[0][5]).toMatchObject({ maxTotalTokens: 24_000 });

      await act(async () => useAgentStore.setState({ preferences: { ...useAgentStore.getState().preferences, tokenLimitPerRun: 36_000 } }));
      expect(mocks.runAgentLoop.mock.calls[0][5]).toMatchObject({ maxTotalTokens: 24_000 });
      await act(async () => finishRun({ kind: "completed", totalTokensUsed: 0 }));
      await act(async () => store.setDraft(chatId, { text: "Second question" }));
      await act(async () => (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click());
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
      expect(mocks.runAgentLoop.mock.calls[1][5]).toMatchObject({ maxTotalTokens: 36_000 });
    } finally {
      view.unmount();
    }
  });

  test.each([undefined, "", " \n\t "])(
    "uses visible fallback confirmation labels for %j custom labels",
    async (label) => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat(undefined, "projects/orders");
      const view = renderIntoContainer(<AgentInput />);
      try {
        for (const confirmed of [true, false]) {
          act(() => store.awaitUser(chat.id, {
            toolCallId: `confirm-${confirmed}`,
            kind: "confirm",
            prompt: "Prepare this unsaved draft?",
            confirmLabel: label,
            cancelLabel: label,
          }));
          await act(async () => view.render());
          const buttons = [...view.container.querySelectorAll("button")];
          const confirm = buttons.find((button) => button.textContent === "agent.confirm");
          const cancel = buttons.find((button) => button.textContent === "agent.cancel");
          expect(confirm).toHaveAccessibleName("agent.confirm");
          expect(cancel).toHaveAccessibleName("agent.cancel");
          expect(store.getPendingAsk(chat.id)?.toolCallId).toBe(`confirm-${confirmed}`);
          await act(async () => (confirmed ? confirm : cancel)?.click());
          const response = store.getMessages(chat.id).find((message) => message.toolCallId === `confirm-${confirmed}`);
          expect(JSON.parse(response?.content ?? "null")).toMatchObject({
            kind: "confirm",
            confirmed,
            answer: confirmed ? "agent.confirm" : "agent.cancel",
          });
        }
      } finally {
        view.unmount();
      }
    }
  );

  test("keeps nonblank custom confirmation labels", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.awaitUser(chat.id, {
      toolCallId: "confirm-custom",
      kind: "confirm",
      prompt: "Prepare this unsaved draft?",
      confirmLabel: "  Prepare draft  ",
      cancelLabel: "  Not now  ",
    });
    const view = renderIntoContainer(<AgentInput />);
    try {
      await act(async () => view.render());
      const buttons = [...view.container.querySelectorAll("button")];
      expect(buttons.find((button) => button.textContent === "Prepare draft")).toHaveAccessibleName("Prepare draft");
      expect(buttons.find((button) => button.textContent === "Not now")).toHaveAccessibleName("Not now");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("passes a historical masking database candidate without selecting it", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Query @orders",
      metadata: {
        resourceReference: {
          kind: "database",
          name: "instances/prod/databases/orders",
          project: "projects/orders",
          label: "orders",
        },
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      await act(async () => {
        setTextareaValue(textarea, "Help me mask email");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await act(async () => {
        view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });

      expect(mocks.createToolExecutor).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
          historicalDatabase: {
            database: "instances/prod/databases/orders",
            project: "projects/orders",
          },
          referencedDatabase: undefined,
          referencedProject: undefined,
        })
      );
      expect(mocks.executeTool).not.toHaveBeenCalledWith(
        "get_database_overview",
        expect.anything(),
        expect.anything()
      );
      expect(store.getMessages(chat.id).at(-1)?.metadata?.resourceReference).toBeUndefined();
      expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    } finally {
      view.unmount();
    }
  });

  test("keeps the mention picker open when typing creates the first selected chat", async () => {
    useAgentStore.getState().clearCurrentChat();
    mocks.currentContext = undefined;
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      await act(async () => {
        setTextareaValue(textarea, "@");
        textarea.setSelectionRange(1, 1);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(useAgentStore.getState().currentChatId).not.toBeNull();
      expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
    } finally {
      view.unmount();
    }
  });

  test("binds a database mention in a clarification reply without changing the saved draft", async () => {
    const databaseName = "instances/prod/databases/orders";
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({
        project: "projects/orders",
        database: { status: "available", name: databaseName, state: "ACTIVE" },
      }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Unsent next question" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help me with this database" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      toolCalls: [{ id: "ask-database", name: "ask_user", arguments: '{"kind":"input","prompt":"Which database?"}' }],
    });
    store.awaitUser(chat.id, { toolCallId: "ask-database", kind: "input", prompt: "Which database?" });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      await act(async () => {
        setTextareaValue(textarea, "Use @");
        textarea.setSelectionRange(5, 5);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
      await act(async () => {
        document.querySelector<HTMLButtonElement>('[data-agent-resource-category="database"]')?.click();
      });
      await vi.waitFor(() => expect(document.querySelector(`[data-agent-resource-database="${databaseName}"]`)).not.toBeNull());
      await act(async () => {
        document.querySelector<HTMLButtonElement>(`[data-agent-resource-database="${databaseName}"]`)?.click();
      });

      expect(textarea.value).toBe("Use @orders (prod) ");
      expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (prod)");
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
      const send = view.container.querySelector("[data-agent-send]") as HTMLButtonElement;
      await act(async () => {
        setTextareaValue(textarea, "Use @orders (prod) @unknown");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(send).toBeDisabled();
      await act(async () => {
        setTextareaValue(textarea, "Use @orders (prod) ");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await act(async () => {
        send.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });

      const reply = store.getMessages(chat.id).find((message) => message.toolCallId === "ask-database");
      expect(JSON.parse(reply?.content ?? "{}")).toEqual({ kind: "input", answer: "Use @orders (prod)" });
      expect(reply?.metadata?.resourceReference).toEqual({
        kind: "database", name: databaseName, project: "projects/orders", label: "orders (prod)",
      });
      expect(mocks.executeTool).toHaveBeenCalledWith("get_database_overview", { databaseName }, expect.any(String));
      expect(mocks.createToolExecutor).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
        context: "projects/orders", referencedDatabase: databaseName, referencedProject: "projects/orders", runMode: "read-only",
      }));
      expect(store.getPendingAsk(chat.id)).toBeNull();
      expect(store.getChat(chat.id)?.context).toBe("projects/orders");
      expect(store.getMessages(chat.id).filter((message) => message.role === "user")).toHaveLength(1);
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
    } finally {
      view.unmount();
    }
  });

  test.each([false, true])("prefills a verified database as a bound mention and revalidates it only after Reply (referenced project: %s)", async (referencedProject) => {
    const databaseName = "instances/prod/databases/orders";
    const project = referencedProject ? "projects/billing" : "projects/orders";
    const resource = { kind: "database" as const, name: databaseName, project, label: "orders (Production)" };
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({ project: resource.project, database: { status: "available", name: databaseName, state: "ACTIVE" } }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Unsent next question" });
    store.addMessage({ chatId: chat.id, role: "user", content: referencedProject ? "@Billing Retry my query" : "Retry my query", metadata: referencedProject ? { resourceReference: { kind: "project", name: project, project, label: "Billing" } } : undefined });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "ask-prefilled-database", name: "ask_user", arguments: JSON.stringify({ kind: "input", resourceKind: "database", prompt: "Which database?", defaultValue: databaseName }) }] });
    store.awaitUser(chat.id, { toolCallId: "ask-prefilled-database", kind: "input", prompt: "Which database?", defaultValue: databaseName, defaultResource: resource });
    useAgentStore.setState({ visible: true, minimized: false });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      expect(textarea.value).toBe("@orders (Production) ");
      expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (Production)");
      expect(document.activeElement).toBe(textarea);
      expect(textarea.selectionStart).toBe(textarea.value.length);
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();

      await act(async () => {
        (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      const reply = store.getMessages(chat.id).find((message) => message.toolCallId === "ask-prefilled-database");
      expect(JSON.parse(reply?.content ?? "{}")).toEqual({ kind: "input", answer: "@orders (Production)" });
      expect(reply?.metadata?.resourceReference).toEqual(resource);
      expect(mocks.executeTool).toHaveBeenCalledWith("get_database_overview", { databaseName }, expect.any(String));
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
    } finally {
      view.unmount();
    }
  });

  test("lets the member edit and delete a prefilled database mention without resetting their answer", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Saved question" });
    store.awaitUser(chat.id, {
      toolCallId: "ask-prefilled-database", kind: "input", prompt: "Which database?",
      defaultValue: "instances/prod/databases/orders",
      defaultResource: { kind: "database", name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" },
    });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      expect(textarea.value).toBe("@orders (prod) ");
      await act(async () => {
        setTextareaValue(textarea, "Use @orders (prod) ");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      act(() => store.incrementChatTotalTokens(chat.id, 1));
      expect(textarea.value).toBe("Use @orders (prod) ");
      expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (prod)");
      act(() => {
        textarea.setSelectionRange(textarea.value.length, textarea.value.length);
        textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Backspace", bubbles: true, cancelable: true }));
      });
      expect(textarea.value).toBe("Use ");
      expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
      act(() => store.awaitUser(chat.id, { toolCallId: "ask-schema", kind: "input", prompt: "Which schema?", defaultValue: "public" }));
      expect(textarea.value).toBe("public");
      expect(store.getDraft(chat.id)).toEqual({ text: "Saved question" });
    } finally {
      view.unmount();
    }
  });

  test.each([
    { defaultValue: "instances/prod/databases/orders" },
    { defaultValue: "Use instances/prod/databases/orders" },
    { defaultValue: "instances/prod/databases/another", defaultResource: { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" } },
    { defaultValue: "instances/prod/databases/orders", defaultResource: { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/billing", label: "orders (prod)" } },
  ])("does not fabricate a mention from an unbound or mismatched default: %j", (prefill) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.awaitUser(chat.id, { toolCallId: "ask-database", kind: "input", prompt: "Which database?", ...prefill });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      expect(view.container.querySelector("textarea")?.value).toBe(prefill.defaultValue);
      expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("blocks a prefilled database reply when fresh access verification fails", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Retry my query" });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "ask-database", name: "ask_user", arguments: '{"kind":"input","prompt":"Which database?"}' }] });
    store.awaitUser(chat.id, {
      toolCallId: "ask-database", kind: "input", prompt: "Which database?", defaultValue: "instances/prod/databases/orders",
      defaultResource: { kind: "database", name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" },
    });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: "projects/orders", database: { status: "unavailable" } }) });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      await act(async () => {
        (view.container.querySelector("[data-agent-send]") as HTMLButtonElement).click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(mocks.executeTool).toHaveBeenCalledWith("get_database_overview", { databaseName: "instances/prod/databases/orders" }, expect.any(String));
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getMessages(chat.id).at(-1)?.content).toBe("agent.database-reference-unavailable");
    } finally {
      view.unmount();
    }
  });

  test("edits reply mentions and clears them for the next question without touching the saved draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Saved question" });
    store.awaitUser(chat.id, { toolCallId: "first-question", kind: "input", prompt: "Which project?" });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
      const selectProject = async () => {
        await act(async () => {
          setTextareaValue(textarea, "@");
          textarea.setSelectionRange(1, 1);
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () => {
          document.querySelector<HTMLButtonElement>('[data-agent-resource-category="project"]')?.click();
        });
        await vi.waitFor(() => expect(document.querySelector('[data-agent-resource-project="projects/billing"]')).not.toBeNull());
        await act(async () => {
          document.querySelector<HTMLButtonElement>('[data-agent-resource-project="projects/billing"]')?.click();
        });
      };
      await selectProject();
      expect(textarea.value).toBe("@Billing ");
      expect(view.container.querySelector("[data-agent-inline-mention]")).not.toBeNull();

      act(() => {
        textarea.setSelectionRange(textarea.value.length, textarea.value.length);
        textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Backspace", bubbles: true, cancelable: true }));
      });
      expect(textarea.value).toBe("");
      expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
      await selectProject();
      act(() => store.awaitUser(chat.id, { toolCallId: "second-question", kind: "input", prompt: "Which schema?", defaultValue: "public" }));
      expect(textarea.value).toBe("public");
      expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
      expect(document.querySelector("[data-agent-resource-picker]")).toBeNull();
      expect(store.getDraft(chat.id)).toEqual({ text: "Saved question" });
      expect(store.getChat(chat.id)?.context).toBe("projects/orders");
    } finally {
      view.unmount();
    }
  });

  test("focuses a new free-text question once with its default value and caret at the end", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    useAgentStore.setState({ visible: true, minimized: false });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;

    act(() =>
      store.awaitUser(chat.id, {
        toolCallId: "question-1",
        kind: "input",
        prompt: "Which database?",
        defaultValue: "orders",
      })
    );

    expect(textarea.value).toBe("orders");
    expect(document.activeElement).toBe(textarea);
    expect(textarea.selectionStart).toBe(6);
    expect(textarea.selectionEnd).toBe(6);
    act(() => {
      setTextareaValue(textarea, "orders_archive");
      textarea.dispatchEvent(new Event("input", { bubbles: true }));
      textarea.setSelectionRange(2, 2);
      store.incrementChatTotalTokens(chat.id, 10);
    });
    view.render();
    expect(textarea.value).toBe("orders_archive");
    expect(textarea.selectionStart).toBe(2);
    const send = view.container.querySelector(
      "[data-agent-send]"
    ) as HTMLButtonElement;
    send.focus();
    act(() => store.incrementChatTotalTokens(chat.id, 10));
    view.render();
    expect(document.activeElement).toBe(send);

    act(() =>
      store.awaitUser(chat.id, {
        toolCallId: "question-2",
        kind: "input",
        prompt: "Which schema?",
        defaultValue: "public",
      })
    );
    expect(textarea.value).toBe("public");
    expect(document.activeElement).toBe(textarea);
    expect(textarea.selectionStart).toBe(6);
    view.unmount();
  });

  test("aligns clarification radios with the first title line above descriptions", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.awaitUser(chat.id, {
      toolCallId: "scope",
      kind: "choose",
      prompt: "Which databases should this rule cover?",
      options: [
        { label: "Only this database", value: "database", description: "Keep the rule limited to the selected database, even when this description wraps onto multiple lines." },
        { label: "All databases in this project", value: "project", description: "Apply the same policy to the project." },
        { label: "All workspace databases", value: "workspace" },
      ],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
    expect(choices).toHaveLength(3);
    for (const choice of choices) {
      expect(choice.closest("label")?.classList.contains("items-start")).toBe(true);
      expect(choice.classList.contains("mt-0.5")).toBe(true);
    }
    view.unmount();
  });

  test("selects a clarification option without sending until Reply", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.setDraft(chat.id, { text: "Keep my draft" });
    store.awaitUser(chat.id, {
      toolCallId: "visibility",
      kind: "choose",
      prompt: "How should emails appear?",
      defaultValue: "full",
      options: [
        { label: "Fully hidden", value: "full" },
        { label: "Partially masked", value: "partial" },
      ],
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const reply = [...view.container.querySelectorAll("button")].find(
      (button) => button.textContent === "agent.reply"
    );
    expect(reply).toBeDefined();
    expect(reply?.disabled).toBe(true);
    const choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
    expect(choices).toHaveLength(2);
    expect([...choices].every((choice) => choice.getAttribute("aria-checked") === "false")).toBe(true);
    act(() => choices[0].click());
    expect(choices[0].getAttribute("aria-checked")).toBe("true");
    expect(reply?.disabled).toBe(false);
    act(() => choices[1].click());
    expect(choices[1].getAttribute("aria-checked")).toBe("true");
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getPendingAsk(chat.id)?.toolCallId).toBe("visibility");
    await act(async () => {
      reply?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "visibility")?.content ?? "{}")).toEqual({
      kind: "choose", answer: "Partially masked", value: "partial",
    });
    expect(store.getDraft(chat.id).text).toBe("Keep my draft");
    view.unmount();
  });

  test("replaces exemption scope choices with separate unselected expiration choices after Reply", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Prepare a masking exemption for me" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      toolCalls: [{ id: "exemption-scope", name: "ask_masking_exemption_scope", arguments: "{}" }],
    });
    store.awaitUser(chat.id, {
      toolCallId: "exemption-scope",
      kind: "choose",
      prompt: "Which databases should this exemption cover?",
      options: [
        { label: "Only orders (Production)", value: "instances/prod/databases/orders" },
        { label: "All databases in this project", value: "projects/orders" },
        { label: "Another database", value: "another-database" },
      ],
    });
    const expiration = {
      kind: "choose" as const,
      prompt: "When should this exemption expire?",
      options: [
        { label: "After 24 hours", value: "24-hours" },
        { label: "No expiration", value: "no-expiration" },
      ],
    };
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          role: "assistant",
          toolCalls: [{ id: "exemption-expiration", name: "ask_user", arguments: JSON.stringify(expiration) }],
        });
        return { kind: "awaiting_user", ask: { toolCallId: "exemption-expiration", ...expiration }, totalTokensUsed: 0 };
      })
      .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const reply = () => [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply");
      let choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
      expect(choices).toHaveLength(3);
      expect([...choices].every((choice) => choice.getAttribute("aria-checked") === "false")).toBe(true);
      expect(reply()?.disabled).toBe(true);
      expect(store.getPendingAsk(chat.id)).not.toHaveProperty("defaultValue");
      act(() => choices[1].click());
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getPendingAsk(chat.id)?.toolCallId).toBe("exemption-scope");

      await act(async () => {
        reply()?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });

      expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "exemption-scope")?.content ?? "{}")).toEqual({
        kind: "choose", answer: "All databases in this project", value: "projects/orders",
      });
      expect(view.container.textContent).toContain("When should this exemption expire?");
      expect(view.container.textContent).not.toContain("Which databases should this exemption cover?");
      expect(view.container.textContent).not.toContain("Only orders (Production)");
      expect(view.container.textContent).not.toContain("All databases in this project");
      expect(view.container.textContent).not.toContain("Another database");
      expect(view.container.textContent).toContain("After 24 hours");
      expect(view.container.textContent).toContain("No expiration");
      choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
      expect(choices).toHaveLength(2);
      expect([...choices].every((choice) => choice.getAttribute("aria-checked") === "false")).toBe(true);
      expect(reply()?.disabled).toBe(true);
      expect(store.getPendingAsk(chat.id)).not.toHaveProperty("defaultValue");
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();

      act(() => choices[1].click());
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      await act(async () => {
        reply()?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
      });
      expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "exemption-expiration")?.content ?? "{}")).toEqual({
        kind: "choose", answer: "No expiration", value: "no-expiration",
      });
    } finally {
      view.unmount();
    }
  });

  test("does not reuse a choice across questions or chats", () => {
    const store = useAgentStore.getState();
    const first = store.createChat({ context: "projects/orders" });
    const ask = { toolCallId: "scope", kind: "choose" as const, prompt: "Which scope?", options: [{ label: "Workspace", value: "workspace" }] };
    store.awaitUser(first.id, ask);
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const reply = () => [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply");
    act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
    expect(reply()?.disabled).toBe(false);
    act(() => store.awaitUser(first.id, { ...ask, toolCallId: "next-scope" }));
    expect(reply()?.disabled).toBe(true);
    act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
    expect(reply()?.disabled).toBe(false);
    act(() => {
      const second = store.createChat({ context: "projects/billing" });
      store.awaitUser(second.id, { ...ask, toolCallId: "next-scope" });
    });
    expect(reply()?.disabled).toBe(true);
    act(() => store.setCurrentChat(first.id));
    expect(reply()?.disabled).toBe(true);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test.each(["chat", "question", "account"] as const)(
    "does not submit a stale choice after the %s changes during configuration refresh",
    async (change) => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders" });
      const ask = { toolCallId: "visibility", kind: "choose" as const, prompt: "Which visibility?", options: [{ label: "Fully hidden", value: "full" }] };
      store.awaitUser(chat.id, ask);
      const view = renderIntoContainer(<AgentInput />);
      await act(async () => view.render());
      let finishRefresh!: () => void;
      mocks.getOrFetchSettingByName.mockImplementationOnce(
        () => new Promise((resolve) => { finishRefresh = () => resolve(undefined); })
      );
      act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
      await act(async () => {
        [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
      });
      expect(finishRefresh).toBeDefined();
      await act(async () => {
        if (change === "chat") store.createChat({ context: "projects/billing" });
        if (change === "question") store.awaitUser(chat.id, { ...ask, toolCallId: "replacement" });
        if (change === "account") useAgentStore.setState({ userEmail: "another@example.com" });
        finishRefresh();
      });
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getMessages(chat.id).filter((message) => message.role === "tool")).toHaveLength(0);
      view.unmount();
    }
  );

  test.each<AgentPendingAsk>([
    {
      toolCallId: "choose",
      kind: "choose",
      prompt: "Which database?",
      options: [{ label: "Orders", value: "orders" }],
    },
    { toolCallId: "confirm", kind: "confirm", prompt: "Continue?" },
    {
      toolCallId: "api",
      kind: "input",
      prompt: "Approve?",
      apiPermission: {
        actionId: "action-1",
        operationId: "ProjectService/UpdateProject",
      },
    },
  ])("does not autofocus a $toolCallId request", (pendingAsk) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    useAgentStore.setState({ visible: true, minimized: false });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const button = document.createElement("button");
    view.container.appendChild(button);
    button.focus();
    act(() => store.awaitUser(chat.id, pendingAsk));
    expect(document.activeElement).toBe(button);
    view.unmount();
  });

  test("does not focus for another chat's question", () => {
    const store = useAgentStore.getState();
    const background = store.createChat({ context: "projects/billing" });
    store.createChat({ context: "projects/orders" });
    useAgentStore.setState({ visible: true, minimized: false });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const button = document.createElement("button");
    view.container.appendChild(button);
    button.focus();
    act(() =>
      store.awaitUser(background.id, {
        toolCallId: "background",
        kind: "input",
        prompt: "Which database?",
      })
    );
    expect(document.activeElement).toBe(button);
    act(() => store.setCurrentChat(background.id));
    expect(document.activeElement).toBe(view.container.querySelector("textarea"));
    view.unmount();
  });

  test.each([
    { tool: "prepare_masking_rule_drafts", result: { draftId: "batch-1" }, kind: "draft-masking-rule", id: "batch-1" },
    { tool: "prepare_semantic_type_draft", result: { draftId: "semantic-1" }, kind: "draft-semantic-type", id: "semantic-1" },
    { tool: "prepare_semantic_type_draft", result: { error: "Permission denied" }, kind: undefined, id: undefined },
    { tool: "prepare_masking_exemption_draft", result: { draftId: "exemption-1" }, kind: "draft-masking-exemption", id: "exemption-1" },
    { tool: "prepare_masking_exemption_draft", result: { error: "Permission denied" }, kind: undefined, id: undefined },
    { tool: "prepare_masking_rule_drafts", result: { error: "Invalid second rule" }, kind: undefined, id: undefined },
  ])("uses only the latest successful $tool handoff across masking flows", async ({tool, result, kind, id}) => {
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({toolCalls:[
        {id:"old", name:"prepare_masking_rule_draft", arguments:"{}"},
        {id:"new", name:tool, arguments:"{}"},
      ]});
      callbacks.onToolResult("old", JSON.stringify({draftId:"old-rule"}));
      callbacks.onToolResult("new", JSON.stringify(result));
      callbacks.onText("Review the new masking proposal.");
      return {kind:"completed", success:true, actions:[], totalTokensUsed:0};
    });
    const store = useAgentStore.getState();
    const page = {path:"/projects/orders", title:"Orders"};
    const chat = store.createChat({context:"projects/orders",page});
    store.queueUserMessage({chatId:chat.id,text:"Prepare the masking drafts.",page});
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => { await vi.waitFor(() => expect(store.getMessages(chat.id).some(message => message.content === "Review the new masking proposal.")).toBe(true)); });
    expect(store.getMessages(chat.id).find(message => message.content === "Review the new masking proposal.")?.metadata?.actions).toEqual(id ? [{kind, draftId:id}] : undefined);
    view.unmount();
  });

  test.each([
    { latest: JSON.stringify({ draftId: "draft-new" }), expectedId: "draft-new" },
    { latest: JSON.stringify({ error: "Unsupported selector condition" }), expectedId: undefined },
    { latest: JSON.stringify({ draftId: "draft-old" }), expectedId: "draft-old" },
  ])("offers only the latest successfully prepared masking draft: $latest", async ({ latest, expectedId }) => {
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [
        { id: "prepare-1", name: "prepare_masking_rule_draft", arguments: "{}" },
        { id: "prepare-2", name: "prepare_masking_rule_draft", arguments: "{}" },
      ] });
      callbacks.onToolResult("prepare-1", JSON.stringify({ draftId: "draft-old" }));
      callbacks.onToolResult("prepare-2", latest);
      callbacks.onText("Review the masking draft.");
      return {
        kind: "completed", success: true,
        actions: [{ kind: "draft-masking-rule", draftId: "draft-old" }],
        totalTokensUsed: 0,
      };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Prepare a global email masking rule.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the masking draft.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "Review the masking draft.");
    expect(answer?.metadata?.actions).toEqual(expectedId ? [{ kind: "draft-masking-rule", draftId: expectedId }] : undefined);
    view.unmount();
  });

  test.each([
    { name: "omitted actions", actions: undefined, success: true },
    { name: "empty actions", actions: [], success: true },
    { name: "invented draft ID", actions: [{ kind: "draft-masking-rule", draftId: "invented" }], success: true },
    { name: "competing settings link", actions: [{ kind: "open", target: { kind: "open-global-masking" } }], success: true },
    { name: "unsuccessful completion", actions: [], success: false },
  ])("attaches the runtime masking draft action after same-batch completion with $name", async ({ actions, success }) => {
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [
        { id: "overview", name: "get_masking_overview", arguments: "{}" },
        { id: "prepare", name: "prepare_masking_rule_draft", arguments: "{}" },
        { id: "done", name: "done", arguments: JSON.stringify({ success, text: "Review the prepared draft.", actions }) },
      ] });
      callbacks.onToolResult("overview", JSON.stringify({ openTargets: [{ kind: "open-global-masking" }] }));
      callbacks.onToolResult("prepare", JSON.stringify({ draftId: "runtime-draft" }));
      callbacks.onText("Review the prepared draft.");
      return { kind: "completed", success, actions, totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/settings/global-masking", title: "Global Masking" };
    const chat = store.createChat({ context: "workspaces/demo", page });
    store.queueUserMessage({ chatId: chat.id, text: "Prepare an email masking rule.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the prepared draft.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "Review the prepared draft.");
    expect(answer?.metadata?.actions).toEqual(success ? [{ kind: "draft-masking-rule", draftId: "runtime-draft" }] : undefined);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.executeTool).not.toHaveBeenCalledWith("prepare_masking_rule_draft", expect.anything(), expect.anything());
    view.unmount();
  });

  test("does not revive a masking draft from an earlier run when a clarification resumes", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Prepare an email mask." });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "old-prepare", name: "prepare_masking_rule_draft", arguments: "{}" }] });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "old-prepare", content: '{"draftId":"previous-run"}' });
    store.awaitUser(chat.id, { toolCallId: "visibility", kind: "choose", prompt: "Which visibility?", options: [{ label: "Fully hidden", value: "full" }] });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onText("Review this draft.");
      return { kind: "completed", success: true, actions: [{ kind: "draft-masking-rule", draftId: "previous-run" }], totalTokensUsed: 0 };
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chat.id).find((message) => message.content === "Review this draft.")?.metadata?.actions).toBeUndefined();
    view.unmount();
  });

  test.each([
    { visible: false, minimized: false },
    { visible: true, minimized: true },
  ])(
    "does not autofocus while visible=$visible and minimized=$minimized",
    (windowState) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat(undefined, "projects/orders");
      useAgentStore.setState(windowState);
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      const button = document.createElement("button");
      view.container.appendChild(button);
      button.focus();
      act(() =>
        store.awaitUser(chat.id, {
          toolCallId: "hidden",
          kind: "input",
          prompt: "Which database?",
        })
      );
      expect(document.activeElement).toBe(button);
      view.unmount();
    }
  );

  test.each(["inert", "data-base-ui-inert", "aria-hidden"])(
    "does not autofocus under an %s ancestor or after it is revealed",
    (attribute) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat(undefined, "projects/orders");
      useAgentStore.setState({ visible: true, minimized: false });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      const button = document.createElement("button");
      document.body.appendChild(button);
      button.focus();
      view.container.setAttribute(
        attribute,
        attribute === "aria-hidden" ? "true" : ""
      );
      act(() =>
        store.awaitUser(chat.id, {
          toolCallId: "obstructed",
          kind: "input",
          prompt: "Which database?",
          defaultValue: "orders",
        })
      );
      expect(document.activeElement).toBe(button);
      view.container.removeAttribute(attribute);
      view.render();
      expect(document.activeElement).toBe(button);
      button.remove();
      view.unmount();
    }
  );

  test("does not take focus from an open agent dialog", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    useAgentStore.setState({ visible: true, minimized: false });
    const view = renderIntoContainer(
      <>
        <AgentInput />
        <AgentDialog open>
          <AgentDialogContent>
            <AgentDialogTitle>Delete chat</AgentDialogTitle>
            <button type="button">Keep chat</button>
          </AgentDialogContent>
        </AgentDialog>
      </>
    );
    await act(async () => view.render());
    const button = document.querySelector(
      "[data-agent-dialog-content] button"
    ) as HTMLButtonElement;
    button.focus();
    act(() =>
      store.awaitUser(chat.id, {
        toolCallId: "modal",
        kind: "input",
        prompt: "Which database?",
        defaultValue: "orders",
      })
    );
    expect(document.activeElement).toBe(button);
    expect(view.container.querySelector("textarea")?.value).toBe("orders");
    view.unmount();
  });

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

  test("shows only a processing placeholder while running and restores the idle hint afterward", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    act(() => store.setChatStatus(chat.id, "running"));

    expect(
      view.container.querySelector("[data-agent-input-placeholder]")?.textContent
    ).toBe("agent.input-processing-placeholder");
    expect(view.container.textContent).not.toContain("agent.input-keyboard-hint");
    expect(view.container.querySelector("textarea")?.disabled).toBe(true);
    expect(
      view.container.querySelector("textarea")?.classList.contains("disabled:bg-transparent")
    ).toBe(true);
    expect(view.container.textContent).toContain("agent.stop");

    act(() => store.setChatStatus(chat.id, "idle"));

    expect(
      view.container.querySelector("[data-agent-input-placeholder]")?.textContent
    ).toBe("agent.input-placeholder · agent.input-keyboard-hint");
    expect(view.container.querySelector("textarea")?.disabled).toBe(false);
    view.unmount();
  });

  test("keeps a retained draft visible while running", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.setDraft(chat.id, { text: "List databases" });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    act(() => store.setChatStatus(chat.id, "running"));

    expect(view.container.querySelector("textarea")?.value).toBe("List databases");
    expect(view.container.querySelector("[data-agent-input-placeholder]")).toBeNull();
    view.unmount();
  });

  test("does not execute queued work or stop a run owned by another tab", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.saveState();
    const sibling = createAgentStore();
    sibling.getState().setWorkspaceName(store.workspaceName!, store.userEmail!);
    const view = renderIntoContainer(<AgentInput />);

    try {
      view.render();
      expect(view.container.querySelector("textarea")).not.toBeDisabled();

      sibling.getState().queueUserMessage({
        chatId: chat.id,
        text: "List databases",
        page: { path: "/demo", title: "Demo" },
      });
      await act(async () => useAgentStore.getState().syncState());

      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(sibling.getState().getMessages(chat.id)).toEqual([]);
      expect(sibling.getState().takeQueuedUserMessage(chat.id)).not.toBeNull();
      sibling.getState().commitQueuedUserMessage(chat.id);
      sibling.getState().startChatRun(chat.id, undefined, {
        runId: "sibling-run",
      });
      await act(async () => useAgentStore.getState().syncState());

      const stop = [...view.container.querySelectorAll("button")].find(
        (button) => button.textContent === "agent.stop"
      );
      expect(stop).toBeDisabled();
      expect(view.container.querySelector("textarea")).toBeDisabled();
      await act(async () => stop?.click());

      expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("running");
      expect(sibling.getState().getChat(chat.id)?.status).toBe("running");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("disables permission confirmation owned by another tab", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.saveState();
    const sibling = createAgentStore();
    sibling.getState().setWorkspaceName(store.workspaceName!, store.userEmail!);
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders","title":"Orders"}}',
      expiresAt: Date.now() + 5 * 60_000,
    });
    const view = renderIntoContainer(<AgentInput />);

    try {
      view.render();
      sibling.getState().startChatRun(chat.id, undefined, {
        runId: "sibling-run",
      });
      sibling.getState().awaitUser(chat.id, {
        kind: "confirm",
        toolCallId: "sibling-write",
        prompt: "Update this project?",
        apiPermission: {
          actionId: "sibling-action",
          operationId: "bytebase.v1.ProjectService.UpdateProject",
        },
      });
      await act(async () => useAgentStore.getState().syncState());

      for (const label of [
        "agent.api-permission-allow",
        "agent.api-permission-deny",
      ]) {
        const button = [...view.container.querySelectorAll("button")].find(
          (candidate) => candidate.textContent === label
        );
        expect(button).toBeDisabled();
        await act(async () => button?.click());
      }

      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(useAgentStore.getState().getChat(chat.id)?.status).toBe(
        "awaiting_user"
      );
      expect(useAgentStore.getState().getPendingAsk(chat.id)).toEqual(
        sibling.getState().getPendingAsk(chat.id)
      );
      expect(useAgentStore.getState().getMessages(chat.id)).toEqual([]);
    } finally {
      view.unmount();
    }
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
      maskingSections: [],
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
      maskingSections: [],
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

  test.each([
    {
      name: "a workspace-selected project",
      context: "workspaces/demo",
      referencedProject: undefined,
      accepted: true,
    },
    {
      name: "a referenced project",
      context: "projects/orders",
      referencedProject: "projects/billing",
      accepted: true,
    },
    {
      name: "a different project-bound chat",
      context: "projects/orders",
      referencedProject: undefined,
      accepted: false,
    },
  ])("scopes masking handoffs to $name instead of the captured page", async ({
    context,
    referencedProject,
    accepted,
  }) => {
    const target = {
      kind: "open-project-masking-exemptions" as const,
      project: "projects/billing",
    };
    mocks.currentPageResource = {
      kind: "masking-exemption",
      project: "projects/orders",
    };
    mocks.currentContext = context;
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, _executor, callbacks) => {
        callbacks.onAssistantMessage({
          toolCalls: [
            {
              id: "masking-overview",
              name: "get_masking_overview",
              arguments: JSON.stringify({ project: "projects/billing" }),
            },
          ],
        });
        callbacks.onToolResult(
          "masking-overview",
          JSON.stringify({
            page: "workspace",
            project: "projects/billing",
            openTargets: [target],
          })
        );
        callbacks.onText("Review the Billing masking exemptions.", target);
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      }
    );
    const page = {
      path: "/projects/orders/masking-exemption",
      title: "Masking exemptions",
    };
    const chat = useAgentStore.getState().createChat({ context, page });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Help me configure masking in Billing.",
      page,
      resourceReference: referencedProject
        ? {
            kind: "project",
            name: referencedProject,
            project: referencedProject,
            label: "Billing",
          }
        : undefined,
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(
          useAgentStore.getState().getMessages(chat.id).some((message) =>
            message.content === "Review the Billing masking exemptions."
          )
        ).toBe(true)
      );
    });

    const answer = useAgentStore.getState().getMessages(chat.id).find(
      (message) => message.content === "Review the Billing masking exemptions."
    );
    expect(answer?.metadata?.handoff).toEqual(accepted ? target : undefined);
    expect(mocks.createToolExecutor).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({
        context,
        referencedProject,
        maskingPage: mocks.currentPageResource,
      })
    );
    view.unmount();
  });

  test.each([
    { name: "denied", latestResult: JSON.stringify({ project: "projects/orders", openTargets: [] }) },
    { name: "failed", latestResult: JSON.stringify({ error: "Current permissions could not be verified." }) },
  ])("replaces stale masking handoffs with the latest $name snapshot while keeping other destinations", async ({ latestResult }) => {
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [
        { id: "project-overview", name: "get_project_overview", arguments: "{}" },
        { id: "masking-allowed", name: "get_masking_overview", arguments: "{}" },
        { id: "masking-latest", name: "get_masking_overview", arguments: "{}" },
      ] });
      callbacks.onToolResult("project-overview", JSON.stringify({
        project: "projects/orders",
        personalIssues: { status: "available", issues: [{ name: "projects/orders/issues/42", relation: "review" }] },
      }));
      callbacks.onToolResult("masking-allowed", JSON.stringify({ project: "projects/orders", openTargets: [{ kind: "open-global-masking" }] }));
      callbacks.onToolResult("masking-latest", latestResult);
      callbacks.onText("Review the issue; masking configuration remains unavailable.", { kind: "review-project-issues", project: "projects/orders" });
      return {
        kind: "completed",
        success: true,
        actions: [{ kind: "open", target: { kind: "open-global-masking" } }],
        totalTokensUsed: 0,
      };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "What should I review about masking?", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the issue; masking configuration remains unavailable.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "Review the issue; masking configuration remains unavailable.");
    expect(answer?.metadata?.actions).toBeUndefined();
    expect(answer?.metadata?.handoff).toEqual({ kind: "review-project-issues", project: "projects/orders" });
    view.unmount();
  });

  test.each([
    { name: "available", openTargets: [{ kind: "open-global-masking" }], fails: false },
    { name: "permission denied", openTargets: [], fails: false },
    { name: "failed", openTargets: [], fails: true },
  ])("refreshes masking handoffs after clarification when the overview is $name", async ({
    openTargets,
    fails,
  }) => {
    const target = { kind: "open-global-masking" as const };
    const args = { project: "projects/orders" };
    const freshResult = JSON.stringify({
      project: "projects/orders",
      openTargets,
      maskingPolicy: { status: openTargets.length ? "available" : "unavailable" },
    });
    if (fails) mocks.executeTool.mockRejectedValueOnce(new Error("unavailable"));
    else mocks.executeTool.mockResolvedValueOnce({ kind: "tool_result", result: freshResult });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onText("Open Global Masking, then select Create rule.");
      return {
        kind: "completed",
        success: true,
        actions: [{ kind: "open", target }],
        totalTokensUsed: 0,
      };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help me mask email addresses." });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "masking-before-ask", name: "get_masking_overview", arguments: JSON.stringify(args) }] });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "masking-before-ask", content: JSON.stringify({ project: "projects/orders", openTargets: [target] }) });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "masking-ask", name: "ask_user", arguments: '{"prompt":"Which visibility?"}' }] });
    store.awaitUser(chat.id, {
      toolCallId: "masking-ask",
      prompt: "Which visibility?",
      kind: "choose",
      options: [{ label: "Fully hidden", value: "full" }],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    const choice = view.container.querySelector<HTMLElement>('[role="radio"]');
    expect(choice).not.toBeNull();
    act(() => choice?.click());
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(mocks.executeTool).toHaveBeenCalledWith("get_masking_overview", args, expect.any(String));
    const history = mocks.runAgentLoop.mock.calls[0][0];
    const refresh = history.find((message: { role: string }) => message.role === "tool");
    if (fails) expect(JSON.parse(refresh.content)).toMatchObject({ error: expect.any(String) });
    else expect(refresh.content).toBe(freshResult);
    expect(store.getMessages(chat.id).find((message) => message.content === "Open Global Masking, then select Create rule.")?.metadata?.actions).toEqual(openTargets.length ? [{ kind: "open", target }] : undefined);
    view.unmount();
  });

  test.each([
    {
      name: "a workspace chat",
      context: "workspaces/demo",
      referencedProject: undefined,
      overviewArgs: { project: "projects/orders" },
    },
    {
      name: "a project-bound chat",
      context: "projects/orders",
      referencedProject: undefined,
      overviewArgs: { project: "projects/orders" },
    },
    {
      name: "a referenced-project turn",
      context: "workspaces/demo",
      referencedProject: "projects/orders",
      overviewArgs: { project: "projects/orders" },
    },
    {
      name: "a workspace chat after an exact semantic type lookup",
      context: "workspaces/demo",
      referencedProject: undefined,
      overviewArgs: { project: "projects/orders", semanticTypeId: "custom-email" },
    },
  ])(
    "keeps a resumed masking selection local to database choices in $name",
    async ({ context, referencedProject, overviewArgs }) => {
      mocks.executeTool.mockResolvedValue({
        kind: "tool_result",
        result: '{"project":"projects/orders"}',
      });
      mocks.runAgentLoop.mockResolvedValue({
        kind: "completed",
        success: true,
        totalTokensUsed: 0,
      });
      const store = useAgentStore.getState();
      const page = { path: "/settings/global-masking", title: "Global Masking" };
      const chat = store.createChat({ context, page });
      store.addMessage({
        chatId: chat.id,
        role: "user",
        content: "Help me mask email addresses.",
        ...(referencedProject
          ? {
              metadata: {
                resourceReference: {
                  kind: "project" as const,
                  name: referencedProject,
                  project: referencedProject,
                  label: "Order service",
                },
              },
            }
          : {}),
      });
      store.addMessage({
        chatId: chat.id,
        role: "assistant",
        toolCalls: [{
          id: "masking-overview",
          name: "get_masking_overview",
          arguments: JSON.stringify(overviewArgs),
        }],
      });
      store.addMessage({
        chatId: chat.id,
        role: "tool",
        toolCallId: "masking-overview",
        content: '{"project":"projects/orders"}',
      });
      store.addMessage({
        chatId: chat.id,
        role: "assistant",
        toolCalls: [{
          id: "rule-scope",
          name: "ask_masking_rule_scope",
          arguments: '{"project":"projects/orders"}',
        }],
      });
      store.awaitUser(chat.id, {
        toolCallId: "rule-scope",
        kind: "choose",
        prompt: "Which databases should the rule cover?",
        options: [{ label: "Another database", value: "another-database" }],
      });
      const view = renderIntoContainer(<AgentInput />);
      try {
        view.render();
        const choice = view.container.querySelector<HTMLElement>('[role="radio"]');
        expect(choice).not.toBeNull();
        act(() => choice?.click());
        await act(async () => {
          [...view.container.querySelectorAll("button")]
            .find((button) => button.textContent === "agent.reply")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });

        const answer = store.getMessages(chat.id)
          .find((message) => message.toolCallId === "rule-scope");
        expect(JSON.parse(answer?.content ?? "null")).toEqual({
          kind: "choose",
          answer: "Another database",
          value: "another-database",
        });
        expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
          expect.anything(),
          expect.objectContaining({
            context,
            referencedProject,
            databaseChoiceProject: "projects/orders",
          })
        );
        expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(
          expect.objectContaining({ context, referencedProject })
        );
        expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith(
          expect.objectContaining({ referencedProject })
        );
        expect(store.getChat(chat.id)?.context).toBe(context);

        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "What needs my attention?");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
        });
        expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
          expect.anything(),
          expect.objectContaining({
            context,
            referencedProject: undefined,
            databaseChoiceProject: undefined,
          })
        );
        expect(store.getChat(chat.id)?.context).toBe(context);
      } finally {
        view.unmount();
      }
    }
  );

  test.each(["abort", "account change"])("does not continue a masking refresh after %s", async (change) => {
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help with masking." });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "overview", name: "get_masking_overview", arguments: "{}" }] });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "overview", content: '{"project":"projects/orders","openTargets":[{"kind":"open-global-masking"}]}' });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "ask", name: "ask_user", arguments: "{}" }] });
    store.awaitUser(chat.id, { toolCallId: "ask", kind: "choose", prompt: "Which visibility?", options: [{ label: "Fully hidden", value: "full" }] });
    let finishRefresh!: (result: { kind: "tool_result"; result: string }) => void;
    mocks.executeTool.mockImplementationOnce(() => new Promise((resolve) => { finishRefresh = resolve; }));
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
      await vi.waitFor(() => expect(mocks.executeTool).toHaveBeenCalledOnce());
    });
    await act(async () => {
      if (change === "abort") store.getAbortController(chat.id)?.abort();
      else useAgentStore.setState({ userEmail: "another@example.com" });
      finishRefresh({ kind: "tool_result", result: '{"project":"projects/orders","openTargets":[{"kind":"open-global-masking"}]}' });
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getMessages(chat.id).filter((message) => message.toolCalls?.[0]?.name === "get_masking_overview")).toHaveLength(1);
    view.unmount();
  });

  test("does not refresh masking from a previous completed user turn", async () => {
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help with masking." });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "old-overview", name: "get_masking_overview", arguments: "{}" }] });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "old-overview", content: '{"openTargets":[{"kind":"open-global-masking"}]}' });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Masking explanation." });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help me connect an instance." });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "connect-ask", name: "ask_user", arguments: "{}" }] });
    store.awaitUser(chat.id, { toolCallId: "connect-ask", kind: "choose", prompt: "Which engine?", options: [{ label: "PostgreSQL", value: "postgres" }] });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(mocks.executeTool).not.toHaveBeenCalled();
    view.unmount();
  });

  test("keeps an accepted action without generating Continue questions", async () => {
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
    expect(answer?.metadata?.actions).toBeUndefined();
    expect(mocks.aiChat).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(0);
    view.unmount();
  });

  test("records the main response once without a follow-up provider call", async () => {
    mocks.aiChat.mockResolvedValue({
      content: '["What should I do next?", "How can I verify this?"]',
      toolCalls: [],
      stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
      usage: { totalTokens: 30, inputTokens: 20, outputTokens: 10 },
    });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onUsage({ purpose: "agent", toolNames: ["done"], totalTokens: 120, inputTokens: 100, outputTokens: 20 });
      callbacks.onText("Two changes need your review.");
      return { kind: "completed", success: true, totalTokensUsed: 120 };
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "What should I review?", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
    });
    expect(store.getChat(chat.id)?.totalTokensUsed).toBe(120);
    expect(store.getChat(chat.id)?.lastRunUsage?.calls).toEqual([
      { purpose: "agent", toolNames: ["done"], totalTokens: 120, inputTokens: 100, outputTokens: 20 },
    ]);
    expect(store.getMessages(chat.id).find(message => message.content === "Two changes need your review.")?.metadata?.actions).toBeUndefined();
    expect(mocks.aiChat).not.toHaveBeenCalled();
    view.unmount();
  });

  test("records a returned response even when its run was canceled", async () => {
    let reportUsage!: () => void;
    let resolveRun!: (result: { kind: "aborted"; totalTokensUsed: number }) => void;
    mocks.runAgentLoop.mockImplementation((_history, _tools, _executor, callbacks) => {
      reportUsage = () => callbacks.onUsage({ purpose: "agent", toolNames: [], totalTokens: 40 });
      return new Promise(resolve => { resolveRun = resolve; });
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "What should I review?", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      store.cancel(chat.id);
      reportUsage();
      resolveRun({ kind: "aborted", totalTokensUsed: 40 });
    });
    expect(store.getChat(chat.id)?.totalTokensUsed).toBe(40);
    expect(store.getChat(chat.id)?.lastRunUsage?.calls).toHaveLength(1);
    view.unmount();
  });

  test("keeps a successful answer without a primary action or generated questions", async () => {
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
          )?.content
        ).toBe("Two changes need your review.")
      );
    });
    expect(mocks.aiChat).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(0);
    view.unmount();
  });

  test("releases the completed answer's controller without a pending suggestion request", async () => {
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
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("idle")
      );
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
    expect(useAgentStore.getState().getAbortController(chat.id)).toBeNull();
    expect(mocks.aiChat).not.toHaveBeenCalled();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.id === completedAnswer?.id)?.metadata?.actions
    ).toBeUndefined();
    expect(useAgentStore.getState().getChat(chat.id)?.interrupted).toBe(false);
    view.unmount();
  });

  test("starts the next turn without leaving suggestion work on the completed answer", async () => {
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
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("idle")
      );
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
    expect(mocks.aiChat).not.toHaveBeenCalled();
    expect(useAgentStore.getState().getChat(chat.id)?.totalTokensUsed).toBe(0);
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

  test.each([0, 1, 2, 3])("preserves %i validated Continue questions without a post-answer provider call", async (questionCount) => {
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
            ...[
              { kind: "continue" as const, prompt: "What should I configure first?" },
              { kind: "continue" as const, prompt: "How can I test the connection?" },
              { kind: "continue" as const, prompt: "How do I assign a database afterward?" },
            ].slice(0, questionCount),
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
      ...[
        { kind: "continue", prompt: "What should I configure first?" },
        { kind: "continue", prompt: "How can I test the connection?" },
        { kind: "continue", prompt: "How do I assign a database afterward?" },
      ].slice(0, questionCount),
    ]);
    expect(mocks.aiChat).not.toHaveBeenCalled();
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "Set up an instance.")?.metadata
        ?.resourceReferences
    ).toEqual(questionCount > 0 ? [
      {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      },
    ] : undefined);
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

  test.each([false, true])("keeps the Plan draft action after a schema read and clarification reply (recovery: %s)", async recovery => {
    if (recovery) enableRunRecovery();
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
        await callbacks.onToolResult("schema-1", result.result);
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

  test.each([false, true])("does not restore Plan draft evidence after an intervening write (recovery: %s)", async recovery => {
    if (recovery) enableRunRecovery();
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
          await callbacks.onToolResult(id, result.result);
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

  describe("managed Plan draft evidence", () => {
    const database = "instances/prod/databases/orders";
    const project = "projects/orders";
    const page = { path: "/projects/orders", title: "Orders" };
    const answer = "```sql\nALTER TABLE orders ADD COLUMN note TEXT;\n```";
    const overview = { project, database: { status: "available", name: database, state: "ACTIVE" } };
    const schemaArgs = { operationId: "DatabaseService/GetDatabaseSchema", body: { name: `${database}/schema` } };

    beforeEach(() => {
      enableRunRecovery();
      mocks.executeTool.mockImplementation(async (name, args) => ({
        kind: "tool_result",
        result: JSON.stringify(name === "get_database_overview" ? overview : {
          status: 200,
          response: args.operationId === schemaArgs.operationId ? { schema: "CREATE TABLE orders (id INT);" } : {},
        }),
      }));
    });

    const queueQuestion = () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: project, page });
      store.queueUserMessage({
        chatId: chat.id, text: "@orders Add a note column", page,
        resourceReference: { kind: "database", name: database, project, label: "orders" },
      });
      return { store, chat };
    };

    const recordTool = async (executor: ToolExecutor, callbacks: AgentCallbacks, id: string, name: string, args: Record<string, unknown>) => {
      callbacks.onAssistantMessage!({ toolCalls: [{ id, name, arguments: JSON.stringify(args) }] });
      const result = await executor(name, args, id);
      if (result.kind !== "tool_result") throw new Error("Expected a completed tool result");
      await callbacks.onToolResult!(id, result.result);
    };

    const settleAttempt = async (options: AgentLoopOptions, totalTokens: number) => {
      const reservation = await options.reserveProviderAttempt!();
      if (reservation.kind !== "allowed") throw new Error("Expected an allowed provider attempt");
      expect(await options.onAttemptSettled!({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId!,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens },
      })).toBe(true);
      return reservation.ordinal;
    };

    test.each(["working", "finalizing", "Retry"] as const)("retains verified Plan actions across %s without repeating startup", async suspension => {
      const runIds: string[] = [];
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, executor, callbacks, _signal, options) => {
        runIds.push(options.runId);
        const totalTokens = runIds.length === 1 && suspension !== "Retry" ? 60_000 : 10;
        const ordinal = await settleAttempt(options, totalTokens);
        if (ordinal === 1) {
          await recordTool(executor, callbacks, "schema-1", "call_api", schemaArgs);
          const phase = suspension === "finalizing" ? "finalizing" : "working";
          const progress = { phase, ...(phase === "finalizing" ? { finalizationText: answer } : {}) };
          expect(await options.onCheckpoint(progress)).toBe(true);
          return suspension === "Retry"
            ? { kind: "error", error: new Error("Temporary provider failure"), totalTokensUsed: totalTokens }
            : { kind: "paused", reason: "token-limit", ...progress, totalTokensUsed: totalTokens };
        }
        expect(options.checkpoint).toMatchObject({ phase: suspension === "finalizing" ? "finalizing" : "working" });
        if (suspension === "finalizing") expect(options.checkpoint.finalizationText).toBe(answer);
        callbacks.onText(answer);
        return { kind: "completed", success: true, actions: [], totalTokensUsed: totalTokens };
      });
      const { store, chat } = queueQuestion();
      const retryActionRef = createRef<(() => void) | null>();
      const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
      view.render();
      try {
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe(suspension === "Retry" ? "error" : "paused")); });
        if (suspension === "Retry") {
          await act(async () => { retryActionRef.current?.(); });
        } else {
          const resume = await openRunLimitIncrease(view.container, 100);
          await act(async () => { resume.click(); });
        }
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
        expect(runIds).toHaveLength(2);
        expect(new Set(runIds).size).toBe(1);
        expect(store.getChat(chat.id)?.runBudget).toMatchObject({
          runId: runIds[0], providerAttemptCount: 2, totalTokensUsed: suspension === "Retry" ? 20 : 60_010,
        });
        expect(store.getMessages(chat.id).find(message => message.content === answer)?.metadata?.actions).toEqual([{ kind: "draft-plan", project, database }]);
        expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
        expect(mocks.executeTool).toHaveBeenCalledTimes(2);
        expect(mocks.executeTool).toHaveBeenNthCalledWith(1, "get_database_overview", { databaseName: database }, expect.any(String));
        expect(mocks.executeTool).toHaveBeenNthCalledWith(2, "call_api", schemaArgs, "schema-1");
      } finally {
        view.unmount();
      }
    });

    test.each([
      { name: "another project", result: { ...overview, project: "projects/other" } },
      { name: "another database", result: { ...overview, database: { ...overview.database, name: "instances/prod/databases/other" } } },
      { name: "unavailable", result: { project, database: { status: "unavailable" } } },
      { name: "archived", result: { ...overview, database: { ...overview.database, state: "DELETED" } } },
    ])("rejects Plan actions after a canonical overview for $name", async ({ result }) => {
      mocks.executeTool
        .mockResolvedValueOnce({ kind: "tool_result", result: JSON.stringify(overview) })
        .mockResolvedValueOnce({ kind: "tool_result", result: JSON.stringify(result) });
      mocks.runAgentLoop
        .mockImplementationOnce(async (_history, _tools, executor, callbacks, _signal, options) => {
          await settleAttempt(options, 10);
          await recordTool(executor, callbacks, "overview-1", "get_database_overview", { databaseName: database });
          await recordTool(executor, callbacks, "schema-1", "call_api", schemaArgs);
          expect(await options.onCheckpoint({ phase: "working" })).toBe(true);
          return { kind: "error", error: new Error("Temporary provider failure"), totalTokensUsed: 10 };
        })
        .mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
          await settleAttempt(options, 10);
          callbacks.onText(answer);
          return { kind: "completed", success: true, actions: [{ kind: "draft-plan", project, database }], totalTokensUsed: 10 };
        });
      const { store, chat } = queueQuestion();
      const retryActionRef = createRef<(() => void) | null>();
      const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
      view.render();
      try {
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
        await act(async () => {
          retryActionRef.current?.();
          await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
        });
        expect(store.getMessages(chat.id).find(message => message.content === answer)).toMatchObject({ role: "assistant" });
        expect(store.getMessages(chat.id).find(message => message.content === answer)?.metadata?.actions).toBeUndefined();
        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
        expect(mocks.executeTool).toHaveBeenCalledTimes(3);
      } finally {
        view.unmount();
      }
    });

    test("does not restore verification from pre-write reads whose results arrive after the write", async () => {
      mocks.runAgentLoop
        .mockImplementationOnce(async (_history, _tools, executor, callbacks, _signal, options) => {
          await settleAttempt(options, 10);
          const reads = [
            { id: "late-overview", name: "get_database_overview", args: { databaseName: database } },
            { id: "late-schema", name: "call_api", args: schemaArgs },
          ];
          callbacks.onAssistantMessage({ toolCalls: reads.map(({ id, name, args }) => ({ id, name, arguments: JSON.stringify(args) })) });
          const results = reads.map(({ id, name, args }) => executor(name, args, id));
          await recordTool(executor, callbacks, "write-1", "call_api", { operationId: "PlanService/CreatePlan", body: {} });
          for (const [index, pending] of results.entries()) {
            const result = await pending;
            if (result.kind !== "tool_result") throw new Error("Expected a completed read");
            await callbacks.onToolResult(reads[index].id, result.result);
          }
          expect(await options.onCheckpoint({ phase: "working" })).toBe(true);
          return { kind: "error", error: new Error("Temporary provider failure"), totalTokensUsed: 10 };
        })
        .mockImplementationOnce(async (_history, _tools, _executor, callbacks, _signal, options) => {
          await settleAttempt(options, 10);
          callbacks.onText(answer);
          return { kind: "completed", success: true, actions: [{ kind: "draft-plan", project, database }], totalTokensUsed: 10 };
        });
      const { store, chat } = queueQuestion();
      const retryActionRef = createRef<(() => void) | null>();
      const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
      view.render();
      try {
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
        const resultOrder = store.getMessages(chat.id).filter(message => message.role === "tool" && !message.metadata?.runPreflight).map(message => message.toolCallId);
        expect(resultOrder).toEqual(["write-1", "late-overview", "late-schema"]);
        await act(async () => {
          retryActionRef.current?.();
          await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
        });
        expect(store.getMessages(chat.id).find(message => message.content === answer)).toMatchObject({ role: "assistant" });
        expect(store.getMessages(chat.id).find(message => message.content === answer)?.metadata?.actions).toBeUndefined();
        expect(mocks.executeTool).toHaveBeenCalledTimes(4);
      } finally {
        view.unmount();
      }
    });
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
    expect(mocks.aiChat).not.toHaveBeenCalled();
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

  test("stores a failed provider request once instead of duplicating it in chat history", async () => {
    const providerError = new Error(
      'OpenAI API error: {"error":{"message":"API key not valid. Please pass a valid API key."}}'
    );
    mocks.runAgentLoop.mockResolvedValue({
      kind: "error",
      error: providerError,
      totalTokensUsed: 0,
    });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = useAgentStore.getState().createChat({
      context: "projects/orders",
      page,
    });
    useAgentStore.getState().queueUserMessage({
      chatId: chat.id,
      text: "Check the API configuration",
      page,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.status).toBe("error")
      );
    });

    expect(useAgentStore.getState().getChat(chat.id)?.lastError).toBe(
      providerError.message
    );
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .filter((message) => message.role === "assistant")
    ).toHaveLength(0);
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
      maskingSections: [],
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
      maskingSections: [],
    });
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
      expect.anything(),
      expect.objectContaining({ runMode: "default" })
    );
    view.unmount();
  });

  test.each(["typed follow-up", "Continue", "Resend", "Retry"] as const)(
    "keeps diagnostic capability and resource on %s after navigation",
    async (action) => {
      const resource = {
        kind: "plan" as const,
        name: "projects/orders/plans/7",
        project: "projects/orders",
        label: "projects/orders/plans/7",
      };
      const capability = { kind: "diagnostic" as const, resource };
      mocks.executeTool.mockResolvedValue({
        kind: "tool_result",
        result: JSON.stringify({
          project: resource.project,
          plan: { status: "available", name: resource.name, state: "ACTIVE" },
        }),
      });
      mocks.runAgentLoop
        .mockResolvedValueOnce(action === "Retry"
          ? { kind: "error", error: new Error("Connection lost") }
          : { kind: "completed", totalTokensUsed: 0 })
        .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
      const page = { path: "/projects/orders/plans/7", title: "Orders Plan" };
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders", page, capability });
      store.queueUserMessage({ chatId: chat.id, text: "Explain this SQL error", page });
      const retryActionRef = createRef<(() => void) | null>();
      const resendActionRef = createRef<((id: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} resendActionRef={resendActionRef} />);
      view.render();
      await act(async () => {
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
        expect.anything(), expect.objectContaining({ capability, referencedPlan: resource.name })
      );

      mocks.currentPath = "/projects/billing/plans/99";
      mocks.currentContext = "projects/billing";
      await act(async () => {
        if (action === "Retry") {
          retryActionRef.current?.();
        } else if (action === "Resend") {
          resendActionRef.current?.(store.getMessages(chat.id).find((message) => message.role === "user")!.id);
        } else if (action === "Continue") {
          store.queueUserMessage({ chatId: chat.id, text: "What caused this failure?", page: { path: mocks.currentPath, title: "Billing" }, runMode: "default" });
        } else {
          const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
          setTextareaValue(textarea, "What caused this failure?");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        }
      });
      if (action === "typed follow-up") {
        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        });
      }
      await act(async () => {
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
      });
      expect(mocks.buildSystemPrompt).toHaveBeenLastCalledWith(expect.objectContaining({ capability, referencedProject: resource.project, referencedPlan: resource.name }));
      expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith(expect.objectContaining({ capability }));
      expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ capability, referencedPlan: resource.name }));
      expect(store.getChat(chat.id)?.capability).toEqual(capability);
      expect(mocks.executeTool.mock.calls.every(([name, args]) => name !== "get_plan_overview" || args.planName === resource.name)).toBe(true);
      view.unmount();
    }
  );

  test("does not replace diagnostic scope with another explicit resource", async () => {
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "projects/orders/plans/7" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: resource.project, capability: { kind: "diagnostic", resource } });
    store.queueUserMessage({
      chatId: chat.id,
      text: "Explain another Plan",
      page: { path: "/projects/orders/plans/8", title: "Another Plan" },
      resourceReferences: [{ ...resource, name: "projects/orders/plans/8" }],
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "agent.diagnostic-help.scope-mismatch")).toBe(true));
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(mocks.createToolExecutor).not.toHaveBeenCalled();
    expect(store.getChat(chat.id)?.capability?.resource.name).toBe(resource.name);
    view.unmount();
  });

  test.each([false, true])(
    "diagnostic DDL answers cannot synthesize Plan drafts with Continue=%s",
    async (withContinue) => {
      const resource = {
        kind: "database" as const,
        name: "instances/prod/databases/orders",
        project: "projects/orders",
        label: "instances/prod/databases/orders",
      };
      const answer =
        "Proposed change:\n\n```sql\nCREATE TABLE audit_log (id INT PRIMARY KEY);\n```";
      const actions = withContinue
        ? [{ kind: "continue", prompt: "Why did this check fail?" }]
        : [];
      mocks.executeTool.mockImplementation(async (name) => ({
        kind: "tool_result",
        result: JSON.stringify(
          name === "call_api"
            ? { status: 200, response: { schema: "CREATE TABLE orders (id INT);" } }
            : {
                project: resource.project,
                database: {
                  status: "available",
                  name: resource.name,
                  state: "ACTIVE",
                },
              }
        ),
      }));
      mocks.runAgentLoop.mockImplementation(
        async (_history, _tools, executor, callbacks) => {
          await executor(
            "call_api",
            {
              operationId: "DatabaseService/GetDatabaseSchema",
              body: { name: `${resource.name}/schema` },
            },
            "schema-1"
          );
          callbacks.onText(answer);
          return {
            kind: "completed",
            success: true,
            totalTokensUsed: 0,
            actions,
          };
        }
      );
      const store = useAgentStore.getState();
      const page = { path: "/projects/orders", title: "Orders" };
      const chat = store.createChat({
        context: resource.project,
        capability: { kind: "diagnostic", resource },
      });
      store.queueUserMessage({
        chatId: chat.id,
        text: "Explain this database check",
        page,
        resourceReferences: [resource],
      });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      await act(async () => {
        await vi.waitFor(() =>
          expect(
            store.getMessages(chat.id).some((message) => message.content === answer)
          ).toBe(true)
        );
      });
      const finalMessage = store
        .getMessages(chat.id)
        .find((message) => message.content === answer);
      expect(finalMessage?.metadata?.actions).toEqual(
        withContinue ? actions : undefined
      );
      expect(finalMessage?.metadata?.resourceReferences).toEqual(
        withContinue ? [resource] : undefined
      );
      view.unmount();
    }
  );

  test.each([false, true].flatMap((recovery) => [
    { trusted: true, success: true, background: false },
    { trusted: false, success: true, background: false },
    { trusted: true, success: false, background: false },
    { trusted: true, success: true, background: true },
  ].map((scenario) => ({ ...scenario, recovery }))))("synthesizes SQL fix actions only for a trusted successful answer: $trusted/$success/$background (recovery: $recovery)", async ({ trusted, success, background, recovery }) => {
    if (recovery) enableRunRecovery();
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "projects/orders/plans/7" };
    const answer = "Complete replacement:\n```sql\nUPDATE books SET price = 2;\n```";
    const followUp = { kind: "continue", prompt: "Why is this predicate needed?" };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: resource.project, plan: { status: "available", name: resource.name, state: "ACTIVE" } }) });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      if (background) useAgentStore.getState().createChat({ context: "projects/other" });
      callbacks.onText(answer);
      return { kind: "completed", success, totalTokensUsed: 0, actions: [{ kind: "review-sql-fix", draftId: "model-invented" }, followUp] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders/plans/7", title: "Plan" };
    const chat = store.createChat({ context: resource.project, workspaceName: "workspaces/demo", capability: { kind: "diagnostic", resource } });
    if (trusted) store.setSqlFixSource(chat.id, {
      identity: JSON.stringify(["workspaces/demo", "users/me@example.com"]),
      pageKey: "orders/7", resource, sourceId: `${resource.name}/planCheckRun`, plan: resource.name,
      specId: "spec-1", sheet: "projects/orders/sheets/9", targets: ["instances/prod/databases/orders"], sha256: "a".repeat(64), sql: "UPDATE books SET price = 1;",
    });
    store.queueUserMessage({ chatId: chat.id, text: "Fix this SQL check", page, resourceReferences: [resource] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === answer)).toBe(true));
    });
    const actions = store.getMessages(chat.id).find((message) => message.content === answer)?.metadata?.actions ?? [];
    expect(actions.some((action) => action.kind === "review-sql-fix")).toBe(trusted && success);
    expect(actions).not.toContainEqual({ kind: "review-sql-fix", draftId: "model-invented" });
    if (trusted && success) {
      const action = actions.find((candidate) => candidate.kind === "review-sql-fix")!;
      expect(useAgentStore.getState().sqlFixDraftsById[action.draftId].chatId).toBe(chat.id);
      expect(actions.filter((candidate) => candidate.kind !== "continue")).toHaveLength(1);
    }
    expect(actions.filter((action) => action.kind === "continue")).toEqual(success ? [followUp] : []);
    view.unmount();
  });

  test.each([true, false])("resumes diagnostic SQL fix synthesis only with its local source (available: %s)", async (sourceAvailable) => {
    enableRunRecovery();
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "projects/orders/plans/7" };
    const page = { path: `/${resource.name}`, title: "Plan" };
    const answer = "Complete replacement:\n```sql\nUPDATE books SET price = 2;\n```";
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: resource.project, plan: { status: "available", name: resource.name, state: "ACTIVE" } }) });
    const runIds: string[] = [];
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      runIds.push(options.runId);
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: runIds.length === 1 ? 60_000 : 10 },
      });
      callbacks.onText(answer);
      if (runIds.length === 1) {
        expect(await options.onCheckpoint({ phase: "working" })).toBe(true);
        return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      }
      return { kind: "completed", success: true, actions: [], totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: resource.project, workspaceName: "workspaces/demo", capability: { kind: "diagnostic", resource } });
    store.setSqlFixSource(chat.id, {
      identity: JSON.stringify(["workspaces/demo", "users/me@example.com"]),
      pageKey: "orders/7", resource, sourceId: `${resource.name}/planCheckRun`, plan: resource.name,
      specId: "spec-1", sheet: "projects/orders/sheets/9", targets: ["instances/prod/databases/orders"], sha256: "a".repeat(64), sql: "UPDATE books SET price = 1;",
    });
    store.queueUserMessage({ chatId: chat.id, text: "Fix this SQL check", page, resourceReferences: [resource] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused"));
      });
      expect(useAgentStore.getState().sqlFixDraftsById).toEqual({});
      expect(store.getMessages(chat.id).some((message) => message.metadata?.actions?.some((action) => action.kind === "review-sql-fix"))).toBe(false);
      const startupCalls = mocks.executeTool.mock.calls.length;
      if (!sourceAvailable) useAgentStore.setState({ sqlFixSourcesByChatId: {} });
      const resume = await openRunLimitIncrease(view.container, 100);
      await act(async () => {
        resume.click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(runIds).toHaveLength(2);
      expect(new Set(runIds).size).toBe(1);
      expect(mocks.executeTool).toHaveBeenCalledTimes(startupCalls);
      const actions = store.getMessages(chat.id).find((message) => message.content === answer)?.metadata?.actions ?? [];
      expect(actions.some((action) => action.kind === "review-sql-fix")).toBe(sourceAvailable);
      expect(Object.values(useAgentStore.getState().sqlFixDraftsById)).toHaveLength(sourceAvailable ? 1 : 0);
      if (sourceAvailable) {
        expect(Object.values(useAgentStore.getState().sqlFixDraftsById)[0]).toMatchObject({ chatId: chat.id, sql: "UPDATE books SET price = 2;" });
      } else {
        expect(useAgentStore.getState().sqlFixSourcesByChatId).toEqual({});
      }
    } finally {
      view.unmount();
    }
  });

  test("diagnostic answers cannot stage Issue comment drafts", async () => {
    const resource = { kind: "issue" as const, name: "projects/orders/issues/42", project: "projects/orders", label: "projects/orders/issues/42" };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: resource.project, issue: { status: "available", name: resource.name } }) });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onText("The SQL needs a bounded predicate.");
      return { kind: "completed", success: true, totalTokensUsed: 0, actions: [
        { kind: "draft-issue-comment", issue: resource.name, comment: "Add a WHERE clause." },
        { kind: "continue", prompt: "Which predicate would be appropriate?" },
      ] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders/issues/42", title: "Issue" };
    const chat = store.createChat({ context: resource.project, capability: { kind: "diagnostic", resource } });
    store.queueUserMessage({ chatId: chat.id, text: "Explain the warning", page, resourceReferences: [resource] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "The SQL needs a bounded predicate.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "The SQL needs a bounded predicate.");
    expect(answer?.metadata?.actions).toEqual([{ kind: "continue", prompt: "Which predicate would be appropriate?" }]);
    expect(answer?.metadata?.resourceReferences).toEqual([resource]);
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
      maskingSections: [],
    });
    view.unmount();
  });

  test("retries the latest failed turn without losing completed tools, human answers, or the draft", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.addMessage({ chatId, role: "user", content: "Inspect orders" });
    store.addMessage({ chatId, role: "assistant", toolCalls: [{ id: "read-1", name: "call_api", arguments: "{}" }], metadata: { runId: "failed-run" } });
    store.addMessage({ chatId, role: "tool", toolCallId: "read-1", content: '{"status":200}', metadata: { runId: "failed-run" } });
    store.addMessage({ chatId, role: "assistant", toolCalls: [{ id: "ask-1", name: "ask_user", arguments: '{"kind":"input","prompt":"Which database?"}' }], metadata: { runId: "failed-run" } });
    store.addMessage({ chatId, role: "tool", toolCallId: "ask-1", content: '{"kind":"input","answer":"Orders"}', metadata: { runId: "failed-run" } });
    store.setDraft(chatId, { text: "My unsent question" });
    store.startChatRun(chatId, { path: "/demo", title: "Demo" }, { runId: "failed-run", runMode: "read-only" });
    store.setChatStatus(chatId, "error", { lastError: "Connection lost" });
    const history = store.getMessages(chatId);
    const draft = store.getDraft(chatId);
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    textarea.setSelectionRange(3, 3);

    await act(async () => {
      retryActionRef.current?.();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    expect(store.getMessages(chatId)).toEqual(history);
    expect(mocks.buildOutboundHistory).toHaveBeenLastCalledWith("system", history, undefined);
    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "read-only",
      maskingSections: [],
    });
    expect(store.getDraft(chatId)).toEqual(draft);
    expect(textarea.value).toBe(draft.text);
    expect(textarea.selectionStart).toBe(3);
    view.unmount();
  });

  test("keeps failed run mode so Retry cannot widen a read-only request", async () => {
    mocks.runAgentLoop
      .mockResolvedValueOnce({ kind: "error", error: new Error("Connection lost") })
      .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.queueUserMessage({ chatId, text: "Review orders", page: { path: "/demo", title: "Demo" }, runMode: "read-only" });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getChat(chatId)?.status).toBe("error"));
    });
    await act(async () => {
      retryActionRef.current?.();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "read-only",
      maskingSections: [],
    });
    expect(store.getMessages(chatId).filter((message) => message.role === "user")).toHaveLength(1);
    view.unmount();
  });

  test("directly resends an older user message with its bound resources and preserves the draft and caret", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: '{"project":"projects/orders"}' });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const resource = { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" };
    const draftResource = { kind: "project" as const, name: "projects/billing", project: "projects/billing", label: "Billing" };
    const original = store.addMessage({ chatId, role: "user", content: "@Orders Review databases", metadata: { resourceReference: resource, resourceReferences: [resource], route: "/old-page", runId: "old-run" } })!;
    store.addMessage({ chatId, role: "assistant", content: "First answer" });
    store.addMessage({ chatId, role: "user", content: "Another question" });
    store.addMessage({ chatId, role: "assistant", content: "Another answer" });
    store.setDraft(chatId, { text: "@Billing unsent question", mentions: [{ resource: draftResource, range: { start: 0, end: 8 } }] });
    const draft = store.getDraft(chatId);
    const history = store.getMessages(chatId);
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
    view.render();
    const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
    textarea.setSelectionRange(12, 12);

    expect(resendActionRef.current).toBeTypeOf("function");
    await act(async () => {
      resendActionRef.current?.(original.id);
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });

    const messages = store.getMessages(chatId);
    expect(messages.slice(0, history.length)).toEqual(history);
    expect(messages.filter((message) => message.role === "user")).toHaveLength(3);
    const resent = messages[history.length];
    expect(resent.id).not.toBe(original.id);
    expect(resent).toMatchObject({ role: "user", content: original.content, metadata: { route: "/demo", resourceReference: resource } });
    expect(resent.metadata?.runId).toBeUndefined();
    expect(resent.metadata?.resourceReferences).toBeUndefined();
    expect(store.getDraft(chatId)).toEqual(draft);
    expect(textarea.value).toBe(draft.text);
    expect(textarea.selectionStart).toBe(12);
    expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ referencedProject: "projects/orders", runMode: "read-only" }));
    view.unmount();
  });

  test.each(["running", "awaiting_user", "archived", "queued"] as const)("does not retry or resend while %s", async (blocked) => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const message = store.addMessage({ chatId, role: "user", content: "Original question" })!;
    store.setChatStatus(chatId, "error", { lastError: "Connection lost" });
    if (blocked === "running") store.startChatRun(chatId, { path: "/demo", title: "Demo" });
    if (blocked === "awaiting_user") store.awaitUser(chatId, { toolCallId: "ask-1", kind: "input", prompt: "Which database?" });
    if (blocked === "archived") store.archiveChat(chatId);
    const retryActionRef = createRef<(() => void) | null>();
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} resendActionRef={resendActionRef} />);
    view.render();
    if (blocked === "queued") {
      act(() => useAgentStore.setState({ queuedUserMessageInFlightByChatId: { [chatId]: true } }));
    }
    await act(async () => {
      retryActionRef.current?.();
      resendActionRef.current?.(message.id);
    });
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getMessages(chatId)).toHaveLength(1);
    view.unmount();
  });

  test("allows Resend once the answer completes without a pending follow-up request", async () => {
    mocks.runAgentLoop
      .mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("Hello");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      })
      .mockResolvedValueOnce({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const message = store.addMessage({ chatId, role: "user", content: "Say hello" })!;
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
    view.render();
    await act(async () => {
      resendActionRef.current?.(message.id);
      await vi.waitFor(() => expect(store.getChat(chatId)?.status).toBe("idle"));
    });
    expect(store.getChat(chatId)?.status).toBe("idle");
    expect(store.getAbortController(chatId)).toBeNull();
    expect(mocks.aiChat).not.toHaveBeenCalled();
    await act(async () => {
      resendActionRef.current?.(message.id);
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });
    view.unmount();
  });

  test.each(["retry", "resend"] as const)("deduplicates %s while configuration loads", async (action) => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const message = store.addMessage({ chatId, role: "user", content: "Original question" })!;
    store.setChatStatus(chatId, "error", { lastError: "Connection lost" });
    const retryActionRef = createRef<(() => void) | null>();
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} resendActionRef={resendActionRef} />);
    view.render();
    let resolveSetting!: (setting: undefined) => void;
    mocks.getOrFetchSettingByName.mockImplementationOnce(() => new Promise<undefined>((resolve) => { resolveSetting = resolve; }));
    const trigger = () => action === "retry" ? retryActionRef.current?.() : resendActionRef.current?.(message.id);
    act(() => { trigger(); trigger(); });
    expect(resolveSetting).toBeTypeOf("function");
    await act(async () => {
      resolveSetting(undefined);
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(store.getMessages(chatId).filter((item) => item.role === "user")).toHaveLength(action === "retry" ? 1 : 2);
    view.unmount();
  });

  test.each(
    (["retry", "resend"] as const).flatMap((action) =>
      (["chat", "account", "workspace", "page", "history", "running", "awaiting_user", "archived"] as const).map((change) => [action, change] as const)
    )
  )("does not %s when %s changes during configuration check", async (action, change) => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const message = store.addMessage({ chatId, role: "user", content: "Original question" })!;
    store.setChatStatus(chatId, "error", { lastError: "Connection lost" });
    const retryActionRef = createRef<(() => void) | null>();
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} resendActionRef={resendActionRef} />);
    view.render();
    let resolveSetting!: (setting: undefined) => void;
    mocks.getOrFetchSettingByName.mockImplementationOnce(() => new Promise<undefined>((resolve) => { resolveSetting = resolve; }));
    act(() => action === "retry" ? retryActionRef.current?.() : resendActionRef.current?.(message.id));
    expect(resolveSetting).toBeTypeOf("function");
    act(() => {
      if (change === "chat") store.createChat();
      if (change === "account") store.setWorkspaceName("workspaces/demo", "other@example.com");
      if (change === "workspace") store.setWorkspaceName("workspaces/other", "agent@example.com");
      if (change === "page") mocks.currentPath = "/another-page";
      if (change === "history") store.addMessage({ chatId, role: "user", content: "A newer question" });
      if (change === "running") store.startChatRun(chatId, { path: "/demo", title: "Demo" });
      if (change === "awaiting_user") store.awaitUser(chatId, { toolCallId: "ask-1", kind: "input", prompt: "Which database?" });
      if (change === "archived") store.archiveChat(chatId);
    });
    await act(async () => resolveSetting(undefined));
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    expect(store.getMessages(chatId).filter((item) => item.content === message.content)).toHaveLength(change === "account" || change === "workspace" ? 0 : 1);
    view.unmount();
  });

  test.each(["", "Unsent question"])("preserves draft %j when direct Resend is blocked by configuration", async (draft) => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const message = store.addMessage({ chatId, role: "user", content: "Original question" })!;
    store.setDraft(chatId, { text: draft });
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
    view.render();
    mocks.getOrFetchSettingByName.mockResolvedValue({ value: { value: { case: "ai", value: { enabled: false } } } });
    expect(resendActionRef.current).toBeTypeOf("function");
    await act(async () => resendActionRef.current?.(message.id));
    expect(store.getDraft(chatId).text).toBe(draft);
    expect(store.getMessages(chatId)).toHaveLength(1);
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    view.unmount();
  });

  test("resends both original bound references and revalidates the database", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const project = { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" };
    const database = { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: project.name, database: { status: "available", name: database.name, state: "ACTIVE" } }) });
    const original = store.addMessage({ chatId, role: "user", content: "@Orders @orders (prod) Inspect schema", metadata: { resourceReference: project, resourceReferences: [project, database] } })!;
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
    view.render();
    await act(async () => {
      resendActionRef.current?.(original.id);
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    const resent = store.getMessages(chatId).filter((message) => message.role === "user")[1];
    expect(resent).toMatchObject({ content: original.content, metadata: { resourceReference: project, resourceReferences: [project, database] } });
    expect(mocks.executeTool).toHaveBeenCalledWith("get_database_overview", { databaseName: database.name }, expect.any(String));
    expect(store.getDraft(chatId).text).toBe("");
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
      maskingSections: [],
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

  test.each(
    (["before", "during"] as const).flatMap((timing) =>
      [false, true].map((hasNewerDraft) => [timing, hasNewerDraft] as const)
    )
  )(
    "keeps an archived queued message unsent when archived %s configuration loading with newer draft %s",
    async (timing, hasNewerDraft) => {
      mocks.runAgentLoop.mockResolvedValue({
        kind: "completed",
        totalTokensUsed: 0,
      });
      const page = { path: "/projects/billing", title: "Billing" };
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/billing", page });
      store.queueUserMessage({ chatId: chat.id, text: "Queued question", page });
      if (hasNewerDraft) {
        store.setDraft(chat.id, { text: "Newer unsent question" });
      }
      let resolveSetting!: (value: undefined) => void;
      if (timing === "before") {
        store.archiveChat(chat.id);
      } else {
        mocks.getOrFetchSettingByName.mockReturnValue(
          new Promise<undefined>((resolve) => {
            resolveSetting = resolve;
          })
        );
      }
      const view = renderIntoContainer(<AgentInput />);
      await act(async () => view.render());
      if (timing === "during") {
        act(() => {
          store.archiveChat(chat.id);
        });
      }
      view.unmount();
      if (timing === "during") {
        await act(async () => {
          resolveSetting(undefined);
        });
      }

      const messages = store.getMessages(chat.id);
      const draft = store.getDraft(chat.id);
      const queued = getQueuedUserMessage(chat.id);
      store.unarchiveChat(chat.id);
      const restored = renderIntoContainer(<AgentInput />);
      await act(async () => restored.render());
      restored.unmount();

      expect(messages).toEqual([]);
      expect(draft.text).toBe(
        hasNewerDraft ? "Newer unsent question" : "Queued question"
      );
      expect(queued).toBeUndefined();
      expect(store.getMessages(chat.id)).toEqual([]);
      expect(store.getChat(chat.id)?.status).toBe("idle");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    }
  );

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

  test("restores resource references when the queued prompt has no mention prefix", async () => {
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: { value: { case: "ai", value: { enabled: false } } },
    });
    const page = { path: "/projects/orders/plans/123", title: "Plan" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    const plan = {
      kind: "plan" as const,
      name: "projects/orders/plans/123",
      project: "projects/orders",
      label: "Add event tracking (#123)",
    };
    store.queueUserMessage({
      chatId: chat.id,
      text: "Review this Plan.",
      page,
      resourceReferences: [plan],
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(getQueuedUserMessage(chat.id)).toBeUndefined());
    });

    expect(store.getDraft(chat.id)).toEqual({
      text: "@Add event tracking (#123) Review this Plan.",
      mentions: [
        { resource: plan, range: { start: 0, end: 26 } },
      ],
    });
    view.unmount();
  });

  test("restores a singular bound resource from its existing mention", async () => {
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: { value: { case: "ai", value: { enabled: false } } },
    });
    const page = { path: "/projects/orders/plans/create", title: "Plan" };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders", page });
    const database = {
      kind: "database" as const,
      name: "instances/primary/databases/app",
      project: "projects/orders",
      label: "app (Primary)",
    };
    store.queueUserMessage({
      chatId: chat.id,
      text: "@app (Primary) Draft SQL for a new table.",
      page,
      resourceReference: database,
    });

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(getQueuedUserMessage(chat.id)).toBeUndefined());
    });

    expect(store.getDraft(chat.id)).toEqual({
      text: "@app (Primary) Draft SQL for a new table.",
      resource: database,
      mention: { start: 0, end: 14 },
    });
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

  test("blocks chat input when ai_enabled is false", async () => {
    mocks.aiEnabled = false;

    const { render, unmount } = renderIntoContainer(<AgentInput />);

    await act(async () => {
      render();
      await Promise.resolve();
    });

    const chatId = useAgentStore.getState().currentChatId!;
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
      maskingSections: [],
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

  test("disables Send for an archived chat", () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.setDraft(chatId, { text: "Unsent question" });
    store.archiveChat(chatId);
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    const sendButton = view.container.querySelector("[data-agent-send]");
    view.unmount();
    expect(sendButton).toBeDisabled();
  });

  test("does not send an archived chat with Enter", async () => {
    mocks.runAgentLoop.mockResolvedValue({
      kind: "completed",
      totalTokensUsed: 0,
    });
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.setDraft(chatId, { text: "Unsent question" });
    store.archiveChat(chatId);
    const draft = store.getDraft(chatId);
    const messages = store.getMessages(chatId);
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    await act(async () => {
      view.container
        .querySelector("textarea")
        ?.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Enter", bubbles: true })
        );
    });

    view.unmount();
    expect(store.getMessages(chatId)).toEqual(messages);
    expect(store.getDraft(chatId)).toEqual(draft);
    expect(store.getChat(chatId)?.status).toBe("idle");
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
  });

  test.each([false, true])(
    "does not send after a chat is archived during configuration loading with pending input %s",
    async (hasPendingInput) => {
      mocks.runAgentLoop.mockResolvedValue({
        kind: "completed",
        totalTokensUsed: 0,
      });
      const store = useAgentStore.getState();
      const chatId = store.currentChatId!;
      store.setDraft(chatId, { text: "Unsent question" });
      if (hasPendingInput) {
        store.awaitUser(chatId, {
          toolCallId: "ask-1",
          kind: "input",
          prompt: "Which database?",
        });
      }
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      const textarea = view.container.querySelector(
        "textarea"
      ) as HTMLTextAreaElement;
      if (hasPendingInput) {
        act(() => {
          setTextareaValue(textarea, "Orders");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
      }
      const draft = store.getDraft(chatId);
      const messages = store.getMessages(chatId);
      const pendingAsk = store.getPendingAsk(chatId);
      let resolveSetting!: (value: undefined) => void;
      mocks.getOrFetchSettingByName.mockImplementationOnce(
        () =>
          new Promise<undefined>((resolve) => {
            resolveSetting = resolve;
          })
      );

      act(() => {
        (
          view.container.querySelector("[data-agent-send]") as HTMLButtonElement
        ).click();
      });
      expect(resolveSetting).toBeTypeOf("function");
      act(() => {
        store.archiveChat(chatId);
      });
      await act(async () => {
        resolveSetting(undefined);
      });

      view.unmount();
      expect(store.getMessages(chatId)).toEqual(messages);
      expect(store.getDraft(chatId)).toEqual(draft);
      expect(store.getPendingAsk(chatId)).toEqual(pendingAsk);
      expect(store.getChat(chatId)?.status).toBe(
        hasPendingInput ? "awaiting_user" : "idle"
      );
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    }
  );

  test.each(
    (["api permission", "confirmation", "choice"] as const).flatMap((action) =>
      (["before", "during"] as const).map((timing) => [action, timing] as const)
    )
  )(
    "does not submit %s when archived %s configuration loading",
    async (action, timing) => {
      mocks.runAgentLoop.mockResolvedValue({
        kind: "completed",
        totalTokensUsed: 0,
      });
      const store = useAgentStore.getState();
      const chatId = store.currentChatId!;
      store.setDraft(chatId, { text: "Unsent question" });
      store.awaitUser(
        chatId,
        action === "choice"
          ? {
              toolCallId: "ask-1",
              kind: "choose",
              prompt: "Which database?",
              options: [{ label: "Orders", value: "orders" }],
            }
          : {
              toolCallId: "ask-1",
              kind: "confirm",
              prompt: "Continue?",
              ...(action === "api permission"
                ? {
                    apiPermission: {
                      actionId: "action-1",
                      operationId: "bytebase.v1.ProjectService.UpdateProject",
                    },
                  }
                : {}),
            }
      );
      if (action === "api permission") {
        mocks.getPendingApiCallPreview.mockReturnValue({
          operationId: "bytebase.v1.ProjectService.UpdateProject",
          body: '{"project":{"name":"projects/orders"}}',
          expiresAt: Date.now() + 5 * 60_000,
        });
      }
      const view = renderIntoContainer(<AgentInput />);
      await act(async () => view.render());
      if (action === "choice") {
        act(() => {
          view.container.querySelector<HTMLElement>('[role="radio"]')?.click();
        });
      }
      const draft = store.getDraft(chatId);
      const messages = store.getMessages(chatId);
      const pendingAsk = store.getPendingAsk(chatId);
      let resolveSetting!: (value: undefined) => void;
      if (timing === "before") {
        act(() => {
          store.archiveChat(chatId);
        });
      } else {
        mocks.getOrFetchSettingByName.mockImplementationOnce(
          () =>
            new Promise<undefined>((resolve) => {
              resolveSetting = resolve;
            })
        );
      }
      const label =
        action === "api permission"
          ? "agent.api-permission-allow"
          : action === "choice"
            ? "agent.reply"
            : "agent.confirm";
      await act(async () => {
        [...view.container.querySelectorAll("button")]
          .find((button) => button.textContent === label)
          ?.click();
      });
      if (timing === "during") {
        expect(resolveSetting).toBeTypeOf("function");
        act(() => {
          store.archiveChat(chatId);
        });
      }
      view.unmount();
      if (timing === "during") {
        await act(async () => {
          resolveSetting(undefined);
        });
      }

      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(store.getMessages(chatId)).toEqual(messages);
      expect(store.getDraft(chatId)).toEqual(draft);
      expect(store.getPendingAsk(chatId)).toEqual(pendingAsk);
      expect(store.getChat(chatId)?.status).toBe("awaiting_user");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    }
  );

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

  test("passes durable diagnostic capability to pending API resolution", async () => {
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "projects/orders/plans/7" };
    const capability = { kind: "diagnostic" as const, resource };
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: resource.project, capability });
    store.awaitUser(chat.id, {
      toolCallId: "tool-write",
      prompt: "bytebase.v1.ProjectService.UpdateProject",
      kind: "confirm",
      apiPermission: { actionId: "action-diagnostic", operationId: "bytebase.v1.ProjectService.UpdateProject" },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.ProjectService.UpdateProject",
      body: '{"project":{"name":"projects/orders"}}',
      expiresAt: Date.now() + 60_000,
    });
    mocks.resolvePendingApiCall.mockResolvedValue('{"error":"Diagnostic request denied."}');
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.api-permission-allow")?.click();
    });
    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith(expect.objectContaining({ capability }));
    expect(store.getMessages(chat.id)).toEqual(expect.arrayContaining([
      expect.objectContaining({ role: "tool", content: '{"error":"Diagnostic request denied."}' }),
    ]));
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

  test("records an approved API result and settles the run if archived during execution", async () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.startChatRun(
      chatId,
      { path: "/demo", title: "Demo" },
      { runId: "approved-run" }
    );
    store.awaitUser(chatId, {
      toolCallId: "tool-write",
      prompt: "bytebase.v1.ProjectService.UpdateProject",
      kind: "confirm",
      apiPermission: {
        actionId: "action-archive",
        operationId: "bytebase.v1.ProjectService.UpdateProject",
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
      [...view.container.querySelectorAll("button")]
        .find((button) => button.textContent === "agent.api-permission-allow")
        ?.click();
    });
    expect(resolvePermission).toBeTypeOf("function");
    expect(store.getChat(chatId)?.status).toBe("running");
    expect(store.getChat(chatId)?.runId).toBe("approved-run");
    act(() => {
      store.archiveChat(chatId);
    });
    view.unmount();

    await act(async () => {
      resolvePermission('{"status":200,"response":{}}');
    });

    expect(store.getMessages(chatId)).toEqual([
      expect.objectContaining({
        role: "tool",
        toolCallId: "tool-write",
        content: '{"status":200,"response":{}}',
        metadata: { route: "/demo", runId: "approved-run" },
      }),
    ]);
    expect(store.getChat(chatId)).toMatchObject({
      archived: true,
      status: "idle",
      runId: null,
    });
    expect(store.getAbortController(chatId)).toBeNull();
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();
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

  test("keeps the input above the context, token usage, and send action in one composer", () => {
    const { container, render, unmount } = renderIntoContainer(<AgentInput />);

    render();

    const composer = container.querySelector("[data-agent-composer]");
    const body = composer?.querySelector("[data-agent-composer-body]");
    const footer = composer?.querySelector("[data-agent-composer-footer]");
    const textarea = container.querySelector("textarea");
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);
    expect(body?.contains(textarea)).toBe(true);
    expect(footer?.contains(textarea)).toBe(false);
    expect(footer?.querySelector("[data-agent-input-footer]")).not.toBeNull();
    expect(footer?.textContent).toContain("agent.context");
    expect(footer?.textContent).toContain("agent.chat-total-tokens");
    expect(footer?.querySelector("[data-agent-send]")).not.toBeNull();
    expect(footer?.querySelector(".lucide-send")).not.toBeNull();
    expect(textarea?.className).toContain("h-9");
    expect(textarea?.className).toContain("min-h-9");
    expect(textarea?.className).not.toContain("max-h-");

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
    const composer = container.firstElementChild as HTMLDivElement;
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
    const composer = container.firstElementChild as HTMLDivElement;
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
    const composer = container.firstElementChild as HTMLDivElement;
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
