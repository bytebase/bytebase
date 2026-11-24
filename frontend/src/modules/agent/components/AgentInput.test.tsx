import { fireEvent, screen, within } from "@testing-library/react";
import type { ComponentProps, ReactElement } from "react";
import { act, createRef } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import {
  type AIChatRequest,
  AIChatResponse_StopReason,
} from "@/types/proto-es/v1/ai_service_pb";
import type { DomRefSuggestion } from "../dom";
import type { AgentDomContext } from "../dom/interactionRegistry";
import type { AgentMaskingPageResource } from "../logic/chatContext";
import type { AgentCallbacks, AgentLoopOptions } from "../logic/agentLoop";
import type { createToolExecutor } from "../logic/tools";
import type {
  AgentChat as AgentChatRecord,
  AgentLoopOutcome,
  AgentPendingAsk,
  ToolExecutor,
} from "../logic/types";
import {
  withAgentPersistenceLock,
  withAgentRunLease,
} from "../logic/runOwnership";
import { getAgentRunStartup } from "../logic/runCheckpoint";
import { boundAgentQueryResult } from "../logic/queryResultRetention";
import { createAgentStore, getAgentStateKey, selectError, useAgentStore } from "../store/agent";
import { AGENT_STORAGE_LIMIT_BYTES, AgentStorageFullError } from "../store/storageBudget";
import * as storageStatusModule from "./AgentStorageStatus";
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
  routeChanged: () => {},
  lazyExtractDomRefSuggestions: vi.fn<() => Promise<DomRefSuggestion[]>>(
    async () => []
  ),
  runAgentLoop: vi.fn(),
  isAgentAIConfigurationError: vi.fn(() => false),
  buildOutboundHistory: vi.fn<typeof import("../logic/outboundHistory").buildOutboundHistory>(() => []),
  buildSystemPrompt: vi.fn<typeof import("../logic/prompt").buildSystemPrompt>(
    () => "system"
  ),
  executeTool: vi.fn(),
  createToolExecutor: vi.fn(() => mocks.executeTool),
  getToolDefinitions: vi.fn<typeof import("../logic/tools").getToolDefinitions>(
    () => []
  ),
  getPendingApiCallPreview: vi.fn(),
  resolvePendingApiCall: vi.fn(),
  resolveDeniedQueryAccessDraft: vi.fn(),
  resolveQueryMaskingEligibility: vi.fn(),
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
  currentUserEmail: "agent@example.com",
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
let AgentChat: typeof import("./AgentChat").AgentChat;

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

vi.mock("@tanstack/react-virtual", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-virtual")>()),
  useVirtualizer: (options: { count: number; estimateSize: (index: number) => number }) => ({
    getVirtualItems: () => Array.from({ length: options.count }, (_, index) => ({ index, key: index, start: index * options.estimateSize(index) })),
    getTotalSize: () => options.count * options.estimateSize(0),
    measureElement: () => undefined,
    scrollToIndex: vi.fn(),
  }),
}));

vi.mock("@/hooks/useReactiveRoute", () => ({
  useReactiveRoute: () => ({ fullPath: mocks.currentPath }),
}));

vi.mock("@/utils", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/utils")>();
  return {
    ...original,
    extractDatabaseResourceName: (resource: string) => original.extractDatabaseResourceName(resource),
    hasWorkspacePermissionV2: () => true,
  };
});

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
    afterEach: (listener: () => void) => {
      mocks.routeChanged = listener;
      return () => {};
    },
    resolve: () => ({ href: "/settings/general", fullPath: "/settings/general" }),
  },
}));

vi.mock("@/stores/app", () => {
  const state = {
    currentUser: {
      name: "users/me@example.com",
      get email() { return mocks.currentUserEmail; },
      get workspace() { return mocks.workspace?.name; },
    },
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
  useAppStore.subscribe = (_listener: () => void) => () => {};
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

vi.mock("../logic/chatContext", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../logic/chatContext")>()),
  getCurrentAgentPageResource: () => mocks.currentPageResource,
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

vi.mock("../logic/deniedQueryAccess", async (importOriginal) => ({
  ...(await importOriginal<Record<string, unknown>>()),
  resolveDeniedQueryAccessDraft: mocks.resolveDeniedQueryAccessDraft,
}));

vi.mock("../logic/queryMaskingEligibility", () => ({
  resolveQueryMaskingEligibility: mocks.resolveQueryMaskingEligibility,
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
  mocks.currentUserEmail = "agent@example.com";
  const initialState = createAgentStore().getState();
  useAgentStore.setState({
    workspaceName: "workspaces/demo",
    userEmail: "agent@example.com",
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    draftsByChatId: initialState.draftsByChatId,
    landingDraft: initialState.landingDraft,
    landingContext: initialState.landingContext,
    queuedUserMessagesByChatId: {},
    queuedUserMessageInFlightByChatId: {},
    followUpQueuesByChatId: {},
    followUpDrainChatIds: {},
    currentChatId: initialState.currentChatId,
    visible: initialState.visible,
    minimized: initialState.minimized,
    abortControllersByChatId: {},
    remoteBusyChatIds: {},
    storagePressure: undefined,
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
  mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(undefined);
  mocks.resolveQueryMaskingEligibility.mockReset();
  mocks.resolveQueryMaskingEligibility.mockImplementation(async (_workspace, database) => ({
    database,
    allowed: false,
  }));
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
  ({ AgentChat } = await import("./AgentChat"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("AgentInput", () => {
  describe("landing draft", () => {
    beforeEach(() => {
      useAgentStore.setState({
        currentChatId: null,
        chats: [],
        messagesByChatId: {},
        draftsByChatId: {},
      });
    });

    test("keeps unsent text outside chat history across composer remounts", async () => {
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "Review the database policy");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });

        expect(textarea.value).toBe("Review the database policy");
        expect(useAgentStore.getState().currentChatId).toBeNull();
        expect(useAgentStore.getState().chats).toEqual([]);

        view.render(<div />);
        view.render(<AgentInput />);
        expect(view.container.querySelector("textarea")?.value).toBe("Review the database policy");
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(localStorage.getItem(getAgentStateKey("workspaces/demo", "agent@example.com")!) ?? "").not.toContain("Review the database policy");
      } finally {
        view.unmount();
      }
    });

    test("stages a project mention without creating history and transfers it on Send", async () => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "Review @");
          textarea.setSelectionRange(8, 8);
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () => {
          document.querySelector<HTMLButtonElement>('[data-agent-resource-category="project"]')?.click();
        });
        await vi.waitFor(() => expect(document.querySelector('[data-agent-resource-project="projects/billing"]')).not.toBeNull());
        await act(async () => {
          document.querySelector<HTMLButtonElement>('[data-agent-resource-project="projects/billing"]')?.click();
        });

        expect(textarea.value).toBe("Review @Billing ");
        expect(useAgentStore.getState().currentChatId).toBeNull();
        expect(useAgentStore.getState().chats).toEqual([]);
        view.render(<div />);
        view.render(<AgentInput />);
        expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@Billing");

        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        const store = useAgentStore.getState();
        expect(store.chats).toHaveLength(1);
        expect(store.getMessages(store.currentChatId)[0]).toMatchObject({
          role: "user",
          content: "Review @Billing",
          metadata: {
            resourceReference: {
              kind: "project",
              name: "projects/billing",
              project: "projects/billing",
              label: "Billing",
            },
          },
        });
      } finally {
        view.unmount();
      }
    });

    test.each(["Explain this policy", "Explain @x literally"])("creates one chat only when submitting %s", async (text) => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, text);
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        expect(useAgentStore.getState().chats).toHaveLength(0);

        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        const store = useAgentStore.getState();
        expect(store.chats).toHaveLength(1);
        expect(store.currentChatId).toBe(store.chats[0].id);
        expect(store.getMessages(store.currentChatId).filter(message => message.role === "user")).toMatchObject([
          { content: text },
        ]);
        expect(textarea.value).toBe("");
      } finally {
        view.unmount();
      }
    });

    test("stages a context selection without creating a chat until Send", async () => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-context-trigger]")?.click();
        });
        await act(async () => {
          document.querySelector<HTMLButtonElement>('[data-agent-context-option="projects/billing"]')?.click();
        });
        expect(useAgentStore.getState().currentChatId).toBeNull();
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(view.container.querySelector("[data-agent-context]")?.textContent).toBe("Billing");

        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "Review this project");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        const store = useAgentStore.getState();
        expect(store.chats).toHaveLength(1);
        expect(store.getChat(store.currentChatId)?.context).toBe("projects/billing");
      } finally {
        view.unmount();
      }
    });

    test("does not create a chat for a whitespace-only prompt", async () => {
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "  \n ");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
          textarea.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
        });

        expect(useAgentStore.getState().currentChatId).toBeNull();
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      } finally {
        view.unmount();
      }
    });

    test("preserves landing edits made while the submitted prompt checks AI configuration", async () => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "First question");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        let finishConfigurationCheck!: (setting: undefined) => void;
        mocks.getOrFetchSettingByName.mockImplementationOnce(() => new Promise<undefined>((resolve) => {
          finishConfigurationCheck = resolve;
        }));
        act(() => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        });
        expect(finishConfigurationCheck).toBeDefined();

        await act(async () => {
          setTextareaValue(textarea, "Second question");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
          finishConfigurationCheck(undefined);
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        const store = useAgentStore.getState();
        expect(store.chats).toHaveLength(1);
        expect(store.getMessages(store.currentChatId)[0]).toMatchObject({ role: "user", content: "First question" });
        expect(view.container.querySelector("textarea")?.value).toBe("Second question");
        expect(store.getDraft(store.currentChatId).text).toBe("Second question");
      } finally {
        view.unmount();
      }
    });

    test("does not submit the prior owner's landing draft after the owner changes during configuration checking", async () => {
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        const textarea = view.container.querySelector("textarea") as HTMLTextAreaElement;
        await act(async () => {
          setTextareaValue(textarea, "Private question");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        let finishConfigurationCheck!: (setting: undefined) => void;
        mocks.getOrFetchSettingByName.mockImplementationOnce(() => new Promise<undefined>((resolve) => {
          finishConfigurationCheck = resolve;
        }));
        act(() => {
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        });
        expect(finishConfigurationCheck).toBeDefined();

        act(() => useAgentStore.getState().setWorkspaceName("workspaces/demo", "other@example.com"));
        const selectedChatAfterOwnerChange = useAgentStore.getState().currentChatId;
        await act(async () => finishConfigurationCheck(undefined));
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatAfterOwnerChange);
        expect(useAgentStore.getState().getMessages(selectedChatAfterOwnerChange)).toEqual([]);
        expect(view.container.querySelector("textarea")?.value).toBe("");
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      } finally {
        view.unmount();
      }
    });
  });

  const renderChatWithInput = (inputProps: ComponentProps<typeof AgentInput> = {}) => {
    const recoveryRef = createRef<(() => Promise<boolean>) | null>();
    const retryRef = inputProps.retryActionRef ?? createRef<(() => void) | null>();
    return renderIntoContainer(<>
      <div data-testid="transcript"><AgentChat onRetry={() => retryRef.current?.()} onFreeSpaceAndRetry={() => recoveryRef.current?.() ?? Promise.resolve(false)} /></div>
      <div data-testid="composer"><AgentInput {...inputProps} retryActionRef={retryRef} freeSpaceAndRetryActionRef={recoveryRef} /></div>
    </>);
  };

  test("shows storage failure only in the transcript, not above the chatbox", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
    store.setChatStatus(chat.id, "error", { lastError: "Agent storage is full" });
    useAgentStore.setState({ storagePressure: { reason: "budget", phase: "admission", bytes: 2_100_000, limitBytes: 2_000_000 } });
    const view = renderChatWithInput();
    view.render();
    try {
      const notice = view.container.querySelector("[data-agent-storage-status]");
      expect(notice).not.toBeNull();
      expect(screen.getByTestId("transcript")).toContainElement(notice as HTMLElement);
      expect(within(screen.getByTestId("composer")).queryByRole("alert")).toBeNull();
      expect(view.container.querySelectorAll("[role=alert]")).toHaveLength(1);
      expect(within(notice as HTMLElement).getByRole("button", { name: "agent.storage.free-and-retry" })).toBeEnabled();
    } finally { view.unmount(); }
  });

  test("retries storage saving without replaying a returned operation or clearing the composer draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Update the project title" });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "completed-write", content: '{"status":200,"response":{"title":"Updated"}}' });
    store.setChatStatus(chat.id, "error", { lastError: "SecurityError: Browser storage is unavailable" });
    store.setDraft(chat.id, { text: "Keep my next question" });
    const flushRunPersistence = store.flushRunPersistence;
    const flush = vi.fn(async () => true);
    useAgentStore.setState({ flushRunPersistence: flush, storagePressure: { reason: "unavailable", phase: "saving", bytes: 1024, limitBytes: 3 * 1024 * 1024 } });
    const messages = store.getMessages(chat.id);
    const view = renderChatWithInput();
    view.render();
    try {
      const status = view.container.querySelector("[data-agent-storage-status]");
      expect(status).not.toBeNull();
      expect(status?.closest("[data-agent-composer]")).toBeNull();
      expect(view.container.querySelector("textarea")).not.toHaveAttribute("aria-invalid");
      expect(screen.queryByRole("button", { name: "agent.storage.manage-history" })).toBeNull();
      await act(async () => screen.getByRole("button", { name: "agent.storage.retry" }).click());
      expect(flush).toHaveBeenCalledOnce();
      expect(store.getMessages(chat.id)).toEqual(messages);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(selectError(useAgentStore.getState())).toContain("SecurityError");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.aiChat).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ flushRunPersistence, storagePressure: undefined });
    }
  });

  test("clears storage status on an identity change without changing the prior account draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.renameChat(chat.id, "Current account history");
    store.setDraft(chat.id, { text: "Keep this account's draft" });
    useAgentStore.setState({ storagePressure: { reason: "budget", phase: "admission", bytes: 2 * 1024 * 1024 + 1, limitBytes: 2 * 1024 * 1024 } });
    const view = renderChatWithInput();
    view.render();
    try {
      expect(screen.getByRole("alert", { name: "agent.storage.title" })).toBeVisible();
      expect(screen.queryByRole("button", { name: "agent.storage.manage-history" })).toBeNull();
      act(() => useAgentStore.setState({ userEmail: "other@example.com", storagePressure: undefined }));
      expect(screen.queryByRole("alert", { name: "agent.storage.title" })).toBeNull();
      act(() => useAgentStore.setState({ userEmail: "agent@example.com" }));
      expect(screen.queryByRole("alert", { name: "agent.storage.title" })).toBeNull();
      expect(store.getDraft(chat.id).text).toBe("Keep this account's draft");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ storagePressure: undefined });
    }
  });

  test("blocks explicit API Allow when storage admission fails without consuming its approval", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.awaitUser(chat.id, { kind: "confirm", toolCallId: "admission-write", prompt: "Update the project?", apiPermission: { actionId: "admission-action", operationId: "bytebase.v1.ProjectService.UpdateProject" } });
    mocks.getPendingApiCallPreview.mockReturnValue({ operationId: "bytebase.v1.ProjectService.UpdateProject", body: "{}", expiresAt: Date.now() + 300_000 });
    const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
    const admission = vi.fn(async () => false);
    useAgentStore.setState({ checkAgentStorageAdmission: admission });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => screen.getByRole("button", { name: "agent.api-permission-allow" }).click());
      expect(admission).toHaveBeenCalledOnce();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getPendingAsk(chat.id)?.apiPermission?.actionId).toBe("admission-action");
      expect(store.getChat(chat.id)?.status).toBe("awaiting_user");
    } finally {
      view.unmount();
      useAgentStore.setState({ checkAgentStorageAdmission });
    }
  });

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

  const enableAsynchronousRunRecovery = (leaseAvailable = () => true) => {
    const pendingLocks = new Map<string, Promise<void>>();
    const heldLocks = new Set<string>();
    vi.stubGlobal("navigator", {
      ...navigator,
      locks: {
        request: vi.fn(async (name: string, options: { ifAvailable?: boolean }, callback: (lock: { name: string } | null) => unknown) => {
          await new Promise<void>((resolve) => setTimeout(resolve, 0));
          if (options.ifAvailable && (!leaseAvailable() || heldLocks.has(name))) return callback(null);
          const previous = pendingLocks.get(name) ?? Promise.resolve();
          let release!: () => void;
          const current = new Promise<void>((resolve) => { release = resolve; });
          const tail = previous.then(() => current);
          pendingLocks.set(name, tail);
          await previous;
          heldLocks.add(name);
          try {
            return await callback({ name });
          } finally {
            heldLocks.delete(name);
            release();
            if (pendingLocks.get(name) === tail) pendingLocks.delete(name);
          }
        }),
      },
    });
    vi.stubGlobal("sessionStorage", createMockStorage());
  };

  const waitForLeaseRelease = async (chatId: string) => {
    const { workspaceName, userEmail } = useAgentStore.getState();
    const key = getAgentStateKey(workspaceName!, userEmail!)!;
    await act(async () => {
      await vi.waitFor(async () => {
        expect(
          (await withAgentRunLease(key, chatId, async () => true)).acquired
        ).toBe(true);
      });
    });
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
      await withAgentRunLease(key, chat.id, async () => {
        const runId = await store.beginChatRun(chat.id, root.id, {
          page, context: chat.context, references: [], runMode: "default",
        });
        if (runId) await store.reserveChatRunAttempt(chat.id, runId);
      });
    }
    store.setDraft(chat.id, { text: "Keep my next question" });
    await store.flushRunPersistence();
    return { store, chat };
  };

  const seedPausedRun = async () => {
    const { store, chat } = await seedUnrecoverableRun("incomplete");
    await withAgentRunLease(getAgentStateKey(store.workspaceName!, store.userEmail!)!, chat.id, () => store.pauseChatRun(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true, suspension: "token-limit" }));
    return { store, chat };
  };

  describe("confirmed storage retry", () => {
    const pressure = { reason: "quota" as const, phase: "saving" as const, bytes: 1024, limitBytes: 3 * 1024 * 1024 };
    const confirmFreeSpace = async () => {
      await act(async () => screen.getByRole("button", { name: "agent.storage.free-and-retry" }).click());
      const dialog = await screen.findByRole("dialog", { name: "agent.storage.free-title" });
      await act(async () => within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" }).click());
    };

    const captureStorageRetry = () => {
      let retry: (() => Promise<boolean>) | undefined;
      const original = storageStatusModule.AgentStorageStatus;
      vi.spyOn(storageStatusModule, "AgentStorageStatus").mockImplementation((props) => {
        retry = props.onFreeSpaceAndRetry;
        return original(props);
      });
      return () => retry!();
    };

    const seedZeroProgressStartup = async () => {
      enableRunRecovery();
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders" });
      const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders", metadata: { resourceReferences: [{ kind: "project", name: "projects/orders", project: "projects/orders", label: "Orders" }] } })!;
      await withAgentRunLease(getAgentStateKey(store.workspaceName!, store.userEmail!)!, chat.id, () => store.beginChatRun(chat.id, root.id, { page: { path: "/demo", title: "Demo" }, references: [{ kind: "project", name: "projects/orders", project: "projects/orders", label: "Orders" }], runMode: "read-only" }));
      store.setChatStatus(chat.id, "error", { lastError: "Storage admission failed" });
      await store.flushRunPersistence();
      return { store, chat };
    };

    test("retries a repaired zero-progress startup despite stale local continuation failure", async () => {
      const { store, chat } = await seedZeroProgressStartup();
      const checkpoint = store.getChat(chat.id)!.runCheckpoint!;
      useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runCheckpoint: { ...checkpoint, rootUserMessageId: "missing-root" } } : candidate) }));
      const retryActionRef = createRef<(() => void) | null>();
      const view = renderIntoContainer(<><AgentChat onRetry={() => retryActionRef.current?.()} /><AgentInput retryActionRef={retryActionRef} /></>);
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: "projects/orders" }) });
      view.render();
      try {
        await act(async () => retryActionRef.current?.());
        expect(view.container.textContent).toContain("agent.token-budget.continuation-unavailable");
        act(() => useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runCheckpoint: checkpoint } : candidate) })));
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
        expect(view.container.querySelector("[data-agent-error-summary]")).not.toBeNull();
        expect(view.container.querySelector("[data-agent-retry]")).not.toBeNull();
        await act(async () => retryActionRef.current?.());
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
      } finally { view.unmount(); }
    });

    test("reports failed recovery when storage fills again before startup preflight admission", async () => {
      const { store, chat } = await seedZeroProgressStartup();
      const retry = captureStorageRetry();
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(async () => {
        localStorage.setItem("bb-agent-state-v3.racing-scope", "x".repeat(1_050_000));
        useAgentStore.setState({ storagePressure: undefined });
        return true;
      }) });
      const view = renderChatWithInput();
      view.render();
      try {
        let recovered: boolean | undefined;
        await act(async () => { recovered = await retry(); });
        expect(recovered).toBe(false);
        expect(useAgentStore.getState().storagePressure).toMatchObject({ reason: "budget", phase: "admission" });
        expect(store.getChat(chat.id)?.status).toBe("error");
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
      } finally {
        view.unmount();
        localStorage.removeItem("bb-agent-state-v3.racing-scope");
        useAgentStore.setState({ freeAgentStorage });
      }
    });

    test("does not report failed startup persistence into a colliding chat after account and workspace change", async () => {
      const { store, chat } = await seedZeroProgressStartup();
      const retry = captureStorageRetry();
      const freeAgentStorage = store.freeAgentStorage;
      const retryChatRunStartup = store.retryChatRunStartup;
      let release!: (saved: boolean) => void;
      const startup = vi.fn(() => new Promise<boolean>((resolve) => { release = resolve; }));
      useAgentStore.setState({ storagePressure: pressure, retryChatRunStartup: startup, freeAgentStorage: vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined }); return true; }) });
      const view = renderChatWithInput();
      view.render();
      try {
        let recovery: Promise<boolean> | undefined;
        await act(async () => { recovery = retry(); await vi.waitFor(() => expect(startup).toHaveBeenCalledOnce()); });
        act(() => useAgentStore.setState((state) => ({ workspaceName: "workspaces/other", userEmail: "other@example.com", chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runId: "other-run", runCheckpoint: undefined, runBudget: undefined, lastError: "Other account request failed" } : candidate) })));
        const other = store.getChat(chat.id);
        await act(async () => { release(false); await recovery; });
        expect(store.getChat(chat.id)).toEqual(other);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage, retryChatRunStartup }); }
    });

    test("restarts a persisted zero-progress startup after admission recovers using its original reference", async () => {
      enableRunRecovery();
      const store = useAgentStore.getState();
      const resource = { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" };
      const page = { path: "/demo", title: "Demo" };
      const chat = store.createChat({ context: resource.name, page });
      store.setDraft(chat.id, { text: "Keep my next request" });
      const freeAgentStorage = store.freeAgentStorage;
      const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
      const admission = vi.fn(async () => false);
      useAgentStore.setState({ checkAgentStorageAdmission: admission, freeAgentStorage: vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined, checkAgentStorageAdmission }); return true; }) });
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: resource.name }) });
      store.queueUserMessage({ chatId: chat.id, text: "Inspect orders", page, resourceReferences: [resource] });
      const view = renderChatWithInput();
      view.render();
      try {
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
        expect(admission).toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        const root = store.getMessages(chat.id).find((message) => message.role === "user")!;
        const runId = store.getChat(chat.id)!.runId;
        const budget = store.getChat(chat.id)!.runBudget;
        expect(store.getChat(chat.id)?.runCheckpoint?.startupComplete).toBe(false);
        expect(getAgentRunStartup(store.getChat(chat.id), store.getMessages(chat.id))).toBeDefined();
        await store.flushRunPersistence();
        act(() => useAgentStore.setState({ storagePressure: pressure }));
        await confirmFreeSpace();
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(mocks.runAgentLoop.mock.calls[0][5]).toMatchObject({ runId });
        expect(mocks.executeTool).toHaveBeenCalledWith("get_project_overview", {}, expect.any(String));
        expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ referencedProject: resource.name }));
        expect(store.getMessages(chat.id).filter((message) => message.role === "user").map((message) => message.id)).toEqual([root.id]);
        expect(store.getChat(chat.id)?.runBudget).toEqual(budget);
        expect(store.getDraft(chat.id).text).toBe("Keep my next request");
        expect(store.getChat(chat.id)?.lastError).toBeNull();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage, checkAgentStorageAdmission }); }
    });

    test.each(["corrupt", "provider-progress"] as const)("does not report a successful retry for %s recovery state", async (kind) => {
      const { store, chat } = await seedUnrecoverableRun("incomplete");
      store.setChatStatus(chat.id, "error", { lastError: "AgentStorageFullError: Agent storage is full." });
      useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, ...(kind === "corrupt" ? { runCheckpoint: { ...candidate.runCheckpoint!, rootUserMessageId: "missing-root" } } : { runBudget: { ...candidate.runBudget!, providerAttemptCount: 1 } }) } : candidate) }));
      await store.flushRunPersistence();
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined }); return true; }) });
      const view = renderChatWithInput();
      view.render();
      try {
        await confirmFreeSpace();
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(store.getChat(chat.id)?.lastError).toContain("AgentStorageFullError");
        expect(screen.queryByRole("button", { name: "agent.token-budget.end-run" })).toBeNull();
        const notice = screen.getByRole("alert");
        expect(notice).not.toHaveClass("items-center");
        const details = within(notice).getByRole("button", { name: "error-page.error-details" });
        const header = details.parentElement;
        expect(header).toHaveTextContent("agent.token-budget.continuation-unavailable");
        await act(async () => details.click());
        expect(notice).toHaveTextContent("AgentStorageFullError");
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });

    test("reuses a blocked user's message without duplicating it or clearing the draft", async () => {
      vi.useFakeTimers();
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      store.setDraft(chat.id, { text: "Keep my next request" });
      store.setChatStatus(chat.id, "error", { lastError: "Browser storage is full" });
      const freeAgentStorage = store.freeAgentStorage;
      const free = vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined }); return true; });
      useAgentStore.setState({ freeAgentStorage: free, storagePressure: pressure });
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      const view = renderChatWithInput();
      view.render();
      try {
        await act(async () => { await vi.advanceTimersByTimeAsync(250); });
        expect(useAgentStore.getState().storagePressure).toEqual(pressure);
        vi.useRealTimers();
        await confirmFreeSpace();
        expect(free).toHaveBeenCalledExactlyOnceWith();
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(store.getMessages(chat.id).filter((message) => message.role === "user")).toEqual([root]);
        expect(store.getDraft(chat.id).text).toBe("Keep my next request");
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });

    test("resumes a valid saved checkpoint rather than starting another request", async () => {
      const { store, chat } = await seedUnrecoverableRun("incomplete");
      await withAgentRunLease(getAgentStateKey(store.workspaceName!, store.userEmail!)!, chat.id, async () => {
        await store.saveChatRunCheckpoint(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true });
        store.interruptChatRun(chat.id);
      });
      const runId = store.getChat(chat.id)!.runId;
      const history = store.getMessages(chat.id);
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined }); return true; }) });
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      const view = renderChatWithInput();
      view.render();
      try {
        await confirmFreeSpace();
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(mocks.runAgentLoop.mock.calls[0][5]).toMatchObject({ runId });
        expect(store.getMessages(chat.id).map(({ id, content }) => ({ id, content }))).toEqual(history.map(({ id, content }) => ({ id, content })));
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });

    test("restores pending API confirmation without approving or running it", async () => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" });
      const ask: AgentPendingAsk = { kind: "confirm", toolCallId: "confirm-write", prompt: "Update the project?", apiPermission: { actionId: "saved-write", operationId: "bytebase.v1.ProjectService.UpdateProject" } };
      store.awaitUser(chat.id, ask);
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(async () => { useAgentStore.setState({ storagePressure: undefined }); return true; }) });
      const view = renderChatWithInput();
      view.render();
      try {
        await confirmFreeSpace();
        expect(store.getPendingAsk(chat.id)).toEqual(ask);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });

    test("keeps the request blocked when freeing storage fails", async () => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
      store.setChatStatus(chat.id, "error");
      const history = store.getMessages(chat.id);
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(async () => false) });
      const view = renderChatWithInput();
      view.render();
      try {
        await confirmFreeSpace();
        expect(screen.getByRole("dialog", { name: "agent.storage.free-title" })).toHaveTextContent("agent.storage.free-failed");
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });

    test.each(["controller", "queued", "in-flight"] as const)("does not start work while %s owns the request", async (owner) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
      store.setChatStatus(chat.id, "error");
      const queued = { id: "queued-recovery", chatId: chat.id, text: "Next request", page: { path: "/demo", title: "Demo" } };
      const freeAgentStorage = store.freeAgentStorage;
      const takeQueuedUserMessage = store.takeQueuedUserMessage;
      const view = renderChatWithInput();
      view.render();
      act(() => useAgentStore.setState({
        freeAgentStorage: vi.fn(async () => true),
        takeQueuedUserMessage: vi.fn(() => null),
        ...(owner === "controller" ? { abortControllersByChatId: { [chat.id]: new AbortController() } } : {}),
        ...(owner === "queued" ? { queuedUserMessagesByChatId: { [chat.id]: queued } } : {}),
        ...(owner === "in-flight" ? { queuedUserMessageInFlightByChatId: { [chat.id]: true } } : {}),
      }));
      act(() => useAgentStore.setState({ storagePressure: pressure }));
      try {
        await act(async () => screen.getByRole("button", { name: "agent.storage.free-and-retry" }).click());
        const dialog = screen.getByRole("dialog", { name: "agent.storage.free-title" });
        expect(within(dialog).queryByRole("checkbox")).toBeNull();
        await act(async () => within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" }).click());
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage, takeQueuedUserMessage }); }
    });

    test.each(["identity", "selection"] as const)("ignores successful async recovery after %s changes", async (change) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
      store.setChatStatus(chat.id, "error");
      const next = store.createChat({ select: false });
      let finish!: (saved: boolean) => void;
      const freeAgentStorage = store.freeAgentStorage;
      useAgentStore.setState({ storagePressure: pressure, freeAgentStorage: vi.fn(() => new Promise<boolean>((resolve) => { finish = resolve; })) });
      const view = renderChatWithInput();
      view.render();
      try {
        await confirmFreeSpace();
        act(() => { if (change === "identity") useAgentStore.setState({ userEmail: "other@example.com" }); else store.setCurrentChat(next.id); });
        await act(async () => { finish(true); });
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(screen.queryByRole("dialog", { name: "agent.storage.free-title" })).toBeNull();
      } finally { view.unmount(); useAgentStore.setState({ freeAgentStorage }); }
    });
  });

  describe("combined current feedback", () => {
    const rawError = "Connection failed while saving the returned operation";
    const pressure = { reason: "quota" as const, phase: "saving" as const, bytes: 1024, limitBytes: 3 * 1024 * 1024 };
    const seedFeedbackRun = (paused = false) => {
      enableRunRecovery();
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders" });
      const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      store.setDraft(chat.id, { text: "Keep my next question" });
      useAgentStore.setState((state) => ({
        chats: state.chats.map((candidate): AgentChatRecord => candidate.id === chat.id ? {
          ...candidate,
          status: paused ? "paused" : "error",
          runId: "feedback-run",
          lastError: rawError,
          runBudget: { version: 1, runId: "feedback-run", limit: 60_000, totalTokensUsed: paused ? 60_000 : 100, hasUnreportedUsage: false, providerAttemptCount: 1, revision: 1 },
          runCheckpoint: { version: 1, runId: "feedback-run", suspension: paused ? "token-limit" : "interrupted", rootUserMessageId: root.id, throughMessageId: root.id, bindings: { page: { path: "/demo", title: "Demo" }, references: [], runMode: "default" }, maskingSections: [], phase: "working", startupComplete: true },
        } : candidate),
      }));
      return { store, chat };
    };
    const renderFeedback = () => {
      const retryActionRef = createRef<(() => void) | null>();
      return renderIntoContainer(<><AgentChat onRetry={() => retryActionRef.current?.()} /><AgentInput retryActionRef={retryActionRef} /></>);
    };
    const expectOneCurrentFeedback = (container: HTMLElement, text: string) => {
      const selector = '[role="alert"], [data-agent-storage-status], [data-agent-token-budget]';
      const surfaces = [...container.querySelectorAll(selector)].filter((surface) => !surface.parentElement?.closest(selector));
      expect(surfaces).toHaveLength(1);
      expect(surfaces[0]?.textContent).toContain(text);
    };
    const expectNoExecution = () => {
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.aiChat).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    };

    test.each([
      { name: "storage and an ordinary error", paused: false, storage: true, unavailable: false, configuration: false, continuation: false, title: "agent.storage.full" },
      { name: "storage and a token pause", paused: true, storage: true, unavailable: false, configuration: false, continuation: false, title: "agent.storage.full" },
      { name: "unavailable execution, storage, and a token pause", paused: true, storage: true, unavailable: true, configuration: false, continuation: false, title: "agent.token-budget.recovery-unavailable" },
      { name: "AI configuration and a token pause", paused: true, storage: false, unavailable: false, configuration: true, continuation: false, title: "agent.ai-not-configured.title" },
      { name: "unrecoverable continuation and an ordinary error", paused: false, storage: false, unavailable: false, configuration: false, continuation: true, title: "agent.token-budget.continuation-unavailable" },
      { name: "a token pause and an ordinary error", paused: true, storage: false, unavailable: false, configuration: false, continuation: false, title: "agent.token-budget.paused" },
    ])("selects one current feedback for $name without executing or changing history", ({ paused, storage, unavailable, configuration, continuation, title }) => {
      const { store, chat } = seedFeedbackRun(paused);
      const canUseAgentExecution = store.canUseAgentExecution;
      useAgentStore.setState((state) => ({
        storagePressure: storage ? pressure : undefined,
        canUseAgentExecution: unavailable ? () => false : canUseAgentExecution,
        chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, requiresAIConfiguration: configuration, ...(continuation ? { runCheckpoint: undefined } : {}) } : candidate),
      }));
      const history = store.getMessages(chat.id);
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, title);
        expect(view.container.querySelector("[data-agent-error-summary]")).toBeNull();
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(store.getChat(chat.id)?.lastError).toBe(rawError);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
        expectNoExecution();
        if (!configuration) {
          const details = view.container.querySelector<HTMLButtonElement>("[data-agent-error-details]");
          expect(details).not.toBeNull();
          expect(view.container.textContent).not.toContain(rawError);
          act(() => details?.click());
          expect(view.container.textContent).toContain(rawError);
        }
      } finally {
        view.unmount();
        useAgentStore.setState({ canUseAgentExecution, storagePressure: undefined });
      }
    });

    test("reveals the next current feedback as execution, storage, configuration, and token blockers clear", () => {
      const { store, chat } = seedFeedbackRun(true);
      const canUseAgentExecution = store.canUseAgentExecution;
      useAgentStore.setState((state) => ({ canUseAgentExecution: () => false, storagePressure: pressure, chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, requiresAIConfiguration: true } : candidate) }));
      const history = store.getMessages(chat.id);
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, "agent.token-budget.recovery-unavailable");
        act(() => useAgentStore.setState({ canUseAgentExecution }));
        expectOneCurrentFeedback(view.container, "agent.storage.full");
        act(() => useAgentStore.setState({ storagePressure: undefined }));
        expectOneCurrentFeedback(view.container, "agent.ai-not-configured.title");
        act(() => useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, requiresAIConfiguration: false } : candidate) })));
        expectOneCurrentFeedback(view.container, "agent.token-budget.paused");
        act(() => useAgentStore.setState((state) => ({ chats: state.chats.map((candidate): AgentChatRecord => candidate.id === chat.id ? { ...candidate, status: "error", runCheckpoint: { ...candidate.runCheckpoint!, suspension: "interrupted" } } : candidate) })));
        expectOneCurrentFeedback(view.container, rawError);
        expect(view.container.querySelector("[data-agent-error-summary]")).not.toBeNull();
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
        expectNoExecution();
      } finally {
        view.unmount();
        useAgentStore.setState({ canUseAgentExecution, storagePressure: undefined });
      }
    });

    test("keeps a legacy failed turn without a run identifier as an ordinary retryable error", () => {
      enableRunRecovery();
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders" });
      store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
      store.finishChatRun(chat.id, { status: "error", lastError: rawError });
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, rawError);
        expect(view.container.querySelector("[data-agent-retry]")).not.toBeNull();
        expect(screen.queryByRole("button", { name: "agent.token-budget.end-run" })).toBeNull();
        expect(store.getChat(chat.id)?.lastError).toBe(rawError);
        expectNoExecution();
      } finally {
        view.unmount();
      }
    });

    test("keeps token recovery after End run fails without changing a valid paused run", async () => {
      const { store, chat } = seedFeedbackRun(true);
      await store.flushRunPersistence();
      const endPausedRun = store.endPausedRun;
      const end = vi.fn(async () => false);
      useAgentStore.setState({ endPausedRun: end });
      const history = store.getMessages(chat.id);
      const checkpoint = store.getChat(chat.id)?.runCheckpoint;
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, "agent.token-budget.paused");
        await act(async () => screen.getByRole("button", { name: "agent.token-budget.end-run" }).click());
        expect(end).toHaveBeenCalledExactlyOnceWith(chat.id);
        expectOneCurrentFeedback(view.container, "agent.token-budget.paused");
        expect(view.container.textContent).toContain("agent.token-budget.action-failed");
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
        const tokenStatus = view.container.querySelector<HTMLElement>("[data-agent-token-budget]");
        expect(tokenStatus).not.toBeNull();
        expect(tokenStatus && (within(tokenStatus).queryByRole("button", { name: "agent.token-budget.increase" }) ?? within(tokenStatus).queryByRole("button", { name: "agent.retry-last-chat-turn" }))).toBeEnabled();
        expect(store.getChat(chat.id)?.runCheckpoint).toEqual(checkpoint);
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
        expectNoExecution();
      } finally {
        view.unmount();
        useAgentStore.setState({ endPausedRun });
      }
    });

    test("clears stale local continuation feedback after the same run checkpoint is repaired", async () => {
      const { store, chat } = seedFeedbackRun();
      const checkpoint = store.getChat(chat.id)!.runCheckpoint!;
      useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runCheckpoint: { ...checkpoint, startupComplete: false } } : candidate) }));
      await store.flushRunPersistence();
      const retryActionRef = createRef<(() => void) | null>();
      const history = store.getMessages(chat.id);
      const view = renderIntoContainer(<><AgentChat onRetry={() => retryActionRef.current?.()} /><AgentInput retryActionRef={retryActionRef} /></>);
      try {
        view.render();
        expectOneCurrentFeedback(view.container, "agent.token-budget.continuation-unavailable");
        await act(async () => retryActionRef.current?.());
        expectOneCurrentFeedback(view.container, "agent.token-budget.continuation-unavailable");
        act(() => useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runCheckpoint: checkpoint } : candidate) })));
        expectOneCurrentFeedback(view.container, rawError);
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
        expect(view.container.querySelector("[data-agent-error-summary]")).not.toBeNull();
        expect(store.getChat(chat.id)?.runId).toBe(checkpoint.runId);
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
        expectNoExecution();
      } finally {
        view.unmount();
      }
    });

    test("keeps AI configuration feedback instead of token or error recovery while AI stays disabled", () => {
      const { store, chat } = seedFeedbackRun(true);
      mocks.aiEnabled = false;
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, "agent.ai-not-configured.title");
        expect(screen.queryByRole("button", { name: "common.dismiss" })).toBeNull();
        expect(view.container.querySelector("[data-agent-token-budget], [data-agent-error-summary]")).toBeNull();
        expect(view.container.querySelector<HTMLButtonElement>("[data-agent-send]")).toBeDisabled();
        expect(store.getChat(chat.id)?.runCheckpoint?.suspension).toBe("token-limit");
        expectNoExecution();
      } finally {
        view.unmount();
      }
    });

    test.each([false, true])("does not offer local recovery for a remotely owned paused=%s run", async (paused) => {
      const { store, chat } = seedFeedbackRun(paused);
      await store.flushRunPersistence();
      enableAsynchronousRunRecovery(() => false);
      useAgentStore.setState({ remoteBusyChatIds: { [chat.id]: true } });
      const view = renderFeedback();
      try {
        view.render();
        expect(view.container.querySelector('[role="alert"], [data-agent-token-budget]')).toBeNull();
        expect(view.container.querySelector("[data-agent-retry]")).toBeNull();
        expect(store.getChat(chat.id)?.lastError).toBe(rawError);
        expectNoExecution();
      } finally {
        view.unmount();
        useAgentStore.setState({ remoteBusyChatIds: {} });
      }
    });

    test("keeps historical interruptions while suppressing the current interrupted turn beneath storage feedback", () => {
      const { store, chat } = seedFeedbackRun();
      store.interruptChatRun(chat.id, undefined, "user");
      store.addMessage({ chatId: chat.id, role: "user", content: "Inspect the next database" });
      store.interruptChatRun(chat.id, undefined, "unexpected");
      useAgentStore.setState({ storagePressure: pressure });
      const history = store.getMessages(chat.id);
      const view = renderFeedback();
      try {
        view.render();
        expectOneCurrentFeedback(view.container, "agent.storage.full");
        const interruptions = view.container.querySelectorAll("[data-agent-interruption-status]");
        expect(interruptions).toHaveLength(1);
        expect(interruptions[0]?.textContent).toContain("agent.request-stopped-by-user");
        expect(view.container.textContent).not.toContain("agent.interrupted-retry-hint");
        expect(store.getMessages(chat.id)).toEqual(history);
        expectNoExecution();
      } finally {
        view.unmount();
        useAgentStore.setState({ storagePressure: undefined });
      }
    });
  });

  test.each([true, false])("checks storage at the managed=%s model and tool boundary before more work", async (managed) => {
    if (managed) enableRunRecovery();
    const store = useAgentStore.getState();
    const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
    const admission = vi.fn(async () => true);
    useAgentStore.setState({ checkAgentStorageAdmission: admission });
    let allowed: boolean | undefined;
    let boundaryAvailable = false;
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, _callbacks, _signal, options) => {
      admission.mockResolvedValue(false);
      const boundary = managed ? options.beforeToolExecution : options.onCheckpoint;
      boundaryAvailable = typeof boundary === "function";
      allowed = boundary ? await boundary({ phase: "working" }) : undefined;
      return { kind: "completed", success: false, totalTokensUsed: 0 };
    });
    const chat = store.createChat({ context: "projects/orders" });
    store.queueUserMessage({ chatId: chat.id, text: "Inspect orders", page: { path: "/demo", title: "Demo" } });
    if (managed) expect(await store.flushRunPersistence()).toBe(true);
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => {
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(store.getChat(chat.id)?.status).toBe("idle");
      }); });
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(boundaryAvailable).toBe(true);
      expect(allowed).toBe(false);
      expect(admission).toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.aiChat).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ checkAgentStorageAdmission });
    }
  });

  test("checks storage before executing a reference preflight", async () => {
    const store = useAgentStore.getState();
    const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
    const admission = vi.fn(async () => false);
    useAgentStore.setState({ checkAgentStorageAdmission: admission });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
    const chat = store.createChat({ context: "projects/orders" });
    store.queueUserMessage({ chatId: chat.id, text: "Inspect billing", resourceReference: { kind: "project", name: "projects/billing", project: "projects/billing", label: "Billing" }, page: { path: "/demo", title: "Demo" } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await new Promise((resolve) => setTimeout(resolve, 0)); });
      expect(admission).toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ checkAgentStorageAdmission });
    }
  });

  test.each(["workspace-recent", "project-open-recent", "workspace-attention", "project-open-blockers"] as const)("checks storage before the %s read-only shortcut", async (starterId) => {
    const store = useAgentStore.getState();
    const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
    const admission = vi.fn(async () => false);
    useAgentStore.setState({ checkAgentStorageAdmission: admission });
    const chat = store.createChat({ context: starterId.startsWith("workspace") ? "workspaces/demo" : "projects/orders" });
    store.queueUserMessage({ chatId: chat.id, text: "Inspect recent work", starterId, page: { path: "/demo", title: "Demo" } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await new Promise((resolve) => setTimeout(resolve, 0)); });
      expect(admission).toHaveBeenCalledOnce();
      expect(mocks.getRecentActivity).not.toHaveBeenCalled();
      expect(mocks.getWorkspaceAttention).not.toHaveBeenCalled();
      expect(mocks.getProjectAttention).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ checkAgentStorageAdmission });
    }
  });

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

  test.each([0, 41_266])("keeps a failed continuation using %s tokens without offering End run or replaying work", async (totalTokensUsed) => {
    const { store, chat } = await seedUnrecoverableRun("incomplete");
    const runId = store.getChat(chat.id)!.runId!;
    if (totalTokensUsed) {
      await withAgentRunLease(getAgentStateKey(store.workspaceName!, store.userEmail!)!, chat.id, async () => {
        await store.reserveChatRunAttempt(chat.id, runId);
        await store.recordChatModelAttempt(chat.id, {
          runId, attemptId: `${runId}:1`, ordinal: 1, outcome: "response",
          usage: { purpose: "agent", toolNames: ["ask_masking_exemption_scope"], totalTokens: totalTokensUsed },
        });
      });
    }
    store.setChatStatus(chat.id, "error", { lastError: "Couldn't update this run. Try again." });
    const history = store.getMessages(chat.id);
    const budget = store.getChat(chat.id)?.runBudget;
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      expect(screen.queryByRole("button", { name: "agent.token-budget.end-run" })).toBeNull();
      expect(view.container.textContent).toContain("agent.token-budget.continuation-unavailable");
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(store.getChat(chat.id)).toMatchObject({ status: "error", runId, totalTokensUsed, runBudget: budget });
      expect(store.getChat(chat.id)?.runCheckpoint).toBeDefined();
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(view.container.querySelector<HTMLButtonElement>("[data-agent-send]")).toBeEnabled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test("keeps network failures in the message error without storage or token notices", async () => {
    enableRunRecovery();
    const { runAgentLoop } = await vi.importActual<typeof import("../logic/agentLoop")>("../logic/agentLoop");
    mocks.runAgentLoop.mockImplementation(runAgentLoop);
    mocks.aiChat.mockRejectedValue(new TypeError("Failed to fetch"));
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.queueUserMessage({ chatId: chat.id, text: "Inspect orders", page: { path: "/demo", title: "Demo" } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error"), { timeout: 5000 }); });
      expect(store.getChat(chat.id)?.lastError).toBe("Failed to fetch");
      expect(useAgentStore.getState().storagePressure).toBeUndefined();
      expect(store.getChat(chat.id)?.modelAttempts).toHaveLength(3);
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(view.container.querySelector("[data-agent-storage-status]")).toBeNull();
      expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
    } finally {
      view.unmount();
    }
  });

  test("blocks work on the real Agent storage budget without a token notice", async () => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Keep my next question" });
    await store.flushRunPersistence();
    localStorage.setItem("bb-agent-state-v3.foreign-scope", "x".repeat(AGENT_STORAGE_LIMIT_BYTES / 2));
    store.queueUserMessage({ chatId: chat.id, text: "Inspect orders", page: { path: "/demo", title: "Demo" } });
    const view = renderChatWithInput();
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
      expect(store.getRunPersistenceError()).toBeInstanceOf(AgentStorageFullError);
      expect(store.getChat(chat.id)?.lastError).toContain("AgentStorageFullError: Agent storage is full.");
      expect(useAgentStore.getState().storagePressure?.reason).toBe("budget");
      expect(view.container.querySelector("[data-agent-storage-status]")).not.toBeNull();
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.aiChat).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      localStorage.removeItem("bb-agent-state-v3.foreign-scope");
      await store.flushRunPersistence();
    }
  });

  test("offers End run only after storage recovers and restores storage feedback when ending cannot persist", async () => {
    const { store, chat } = await seedPausedRun();
    localStorage.setItem("bb-agent-state-v3.foreign-scope", "x".repeat(AGENT_STORAGE_LIMIT_BYTES / 2));
    store.setChatStatus(chat.id, "error", { lastError: "Agent storage is full." });
    const history = store.getMessages(chat.id);
    expect(await store.flushRunPersistence()).toBe(false);
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderChatWithInput({ retryActionRef });
    view.render();
    let write: ReturnType<typeof vi.spyOn> | undefined;
    try {
      await act(async () => { retryActionRef.current?.(); });
      expect(screen.queryByRole("button", { name: "agent.token-budget.end-run" })).toBeNull();
      expect(view.container.querySelector("[data-agent-storage-status]")).not.toBeNull();
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      localStorage.removeItem("bb-agent-state-v3.foreign-scope");
      await act(async () => { await store.flushRunPersistence(); });
      expect(view.container.querySelector("[data-agent-storage-status]")).toBeNull();
      expect(screen.getByRole("button", { name: "agent.token-budget.end-run" })).toBeEnabled();
      write = vi.spyOn(localStorage, "setItem").mockImplementation(() => {
        throw new AgentStorageFullError("budget", AGENT_STORAGE_LIMIT_BYTES + 1);
      });
      await act(async () => { screen.getByRole("button", { name: "agent.token-budget.end-run" }).click(); });
      expect(view.container.querySelector("[data-agent-storage-status]")).not.toBeNull();
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      write?.mockRestore();
      localStorage.removeItem("bb-agent-state-v3.foreign-scope");
      await store.flushRunPersistence();
    }
  });

  test.each(["identity", "selection", "run", "remote"] as const)("does not end a stale run when %s changes before acquiring its lease", async (changed) => {
    const { store, chat } = await seedPausedRun();
    store.setChatStatus(chat.id, "error", { lastError: "Connection lost" });
    const other = store.createChat();
    store.setCurrentChat(chat.id);
    await store.flushRunPersistence();
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    const endPausedRun = vi.fn(store.endPausedRun);
    useAgentStore.setState({ endPausedRun });
    try {
      await act(async () => { retryActionRef.current?.(); });
      enableAsynchronousRunRecovery();
      await act(async () => {
        screen.getByRole("button", { name: "agent.token-budget.end-run" }).click();
        if (changed === "identity") useAgentStore.setState({ userEmail: "other@example.com" });
        else if (changed === "selection") store.setCurrentChat(other.id);
        else if (changed === "remote") useAgentStore.setState({ remoteBusyChatIds: { [chat.id]: true } });
        else useAgentStore.setState((state) => ({ chats: state.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, runId: "new-run" } : candidate) }));
        await new Promise((resolve) => setTimeout(resolve, 20));
      });
      expect(endPausedRun).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ endPausedRun: store.endPausedRun });
    }
  });

  test("disables duplicate End run actions and keeps failed-action feedback", async () => {
    const { store, chat } = await seedPausedRun();
    store.setChatStatus(chat.id, "error", { lastError: "Connection lost" });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    let release!: (saved: boolean) => void;
    const endPausedRun = vi.fn(() => new Promise<boolean>((resolve) => { release = resolve; }));
    useAgentStore.setState({ endPausedRun });
    const history = store.getMessages(chat.id);
    try {
      await act(async () => { retryActionRef.current?.(); });
      const end = screen.getByRole("button", { name: "agent.token-budget.end-run" });
      await act(async () => { end.click(); end.click(); });
      expect(end).toBeDisabled();
      expect(endPausedRun).toHaveBeenCalledOnce();
      await act(async () => { release(false); });
      expect(end).toBeEnabled();
      expect(view.container.textContent).toContain("agent.token-budget.action-failed");
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      view.unmount();
      useAgentStore.setState({ endPausedRun: store.endPausedRun });
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

  test("keeps incomplete usage at 24860 of 100000 tokens in details instead of a banner", async () => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" });
    useAgentStore.setState((state) => ({
      chats: state.chats.map((candidate): AgentChatRecord => candidate.id === chat.id ? {
        ...candidate, status: "running", runId: "partial-run", totalTokensUsed: 24_860, hasUnreportedUsage: true,
        legacyUsage: { totalTokensUsed: 0, hasUnreportedUsage: false },
        runBudget: { version: 1, runId: "partial-run", limit: 100_000, totalTokensUsed: 24_860, hasUnreportedUsage: true, providerAttemptCount: 2, revision: 2 },
        modelAttempts: [
          { attemptId: "partial-run:1", runId: "partial-run", ordinal: 1, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 24_860 } },
          { attemptId: "partial-run:2", runId: "partial-run", ordinal: 2, outcome: "error" },
        ],
      } : candidate),
      abortControllersByChatId: { [chat.id]: new AbortController() },
    }));
    const before = store.getChat(chat.id);
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(view.container.textContent).not.toContain("agent.token-budget.partial");
      await act(async () => screen.getByRole("button", { name: "agent.chat-total-tokens" }).click());
      const details = document.querySelector("[data-agent-token-usage]");
      expect(details).toHaveTextContent("agent.token-budget.partial");
      expect(details).toHaveTextContent("agent.token-usage.at-least");
      expect(details).toHaveTextContent("100,000");
      expect(store.getChat(chat.id)).toEqual(before);
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally { view.unmount(); }
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

  test("replaces budget recovery controls when durable execution becomes unavailable", async () => {
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
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      expect(screen.queryByRole("button", { name: "agent.token-budget.increase" })).toBeNull();
      expect(screen.queryByRole("button", { name: "agent.token-budget.end-run" })).toBeNull();
      expect(view.container.textContent).not.toContain("agent.token-budget.resume");
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
    } finally {
      view.unmount();
    }
  });

  test("increases and resumes the same logical run without clearing its draft or repeating startup", async () => {
    enableRunRecovery();
    const { buildOutboundHistory } = await vi.importActual<typeof import("../logic/outboundHistory")>("../logic/outboundHistory");
    mocks.buildOutboundHistory.mockImplementation(buildOutboundHistory);
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
    for (const [history] of mocks.runAgentLoop.mock.calls) {
      expect(history.flatMap((message: { toolCalls?: unknown[] }) => message.toolCalls ?? [])).toContainEqual(expect.objectContaining({
        name: "get_project_overview",
        origin: "runtime",
      }));
    }
    expect(await store.flushRunPersistence()).toBe(true);
    const reloaded = createAgentStore();
    reloaded.getState().setWorkspaceName("workspaces/demo", "agent@example.com");
    expect(reloaded.getState().getMessages(chat.id).flatMap(message => message.toolCalls ?? [])).toContainEqual(expect.objectContaining({
      name: "get_project_overview",
      origin: "runtime",
    }));
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

  test.each([
    [true, "api"], [false, "api"], [true, "dom"], [true, "dom-navigation"],
  ] as const)("keeps a below-limit API permission interactive after asynchronous persistence and focus with approved=%s and origin %s", async (approved, origin) => {
    enableAsynchronousRunRecovery();
    const toolName = origin === "api" ? "call_api" : "dom_action";
    const operationId = origin === "api" ? "bytebase.v1.SQLService.Query" : "bytebase.v1.DatabaseService.UpdateDatabase";
    const ask: AgentPendingAsk = {
      kind: "confirm", toolCallId: "query-permission", prompt: "Run this query?",
      apiPermission: { actionId: "query-action", operationId },
    };
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.SQLService.Query", body: '{"name":"instances/example/databases/orders","statement":"SELECT 1","limit":100}', expiresAt: Date.now() + 300_000,
    });
    const runIds: string[] = [];
    let activeDom: AgentDomContext | undefined;
    if (origin !== "api") {
      mocks.getPendingApiCallPreview.mockImplementation(({ domContext }) =>
        domContext?.isCurrent() && domContext.pageGeneration === activeDom?.pageGeneration
          ? { operationId, body: "{}", dom: { kind: "database-labels", changes: [], resourceName: "instances/i/databases/db" }, expiresAt: Date.now() + 300_000 }
          : undefined
      );
      mocks.resolvePendingApiCall.mockImplementation(async ({ domContext, approved: allowed, signal, context }) => {
        expect(activeDom?.isCurrent()).toBe(false);
        expect(domContext?.runId).toBe(runIds[0]);
        expect(domContext?.signal).toBe(signal);
        expect(domContext?.isCurrent()).toBe(true);
        expect(context).toMatchObject({ runId: runIds[0], toolCallId: ask.toolCallId, logicalActionId: ask.apiPermission!.actionId });
        expect(context?.isCurrent()).toBe(true);
        expect(allowed).toBe(origin === "dom");
        return allowed ? '{"status":200,"response":{}}' : '{"called":false}';
      });
    }
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks, _signal, options) => {
      runIds.push(options.runId);
      if (runIds.length === 1) {
        if (origin !== "api") {
          const [, executorOptions] = mocks.createToolExecutor.mock.lastCall as unknown as Parameters<typeof createToolExecutor>;
          activeDom = executorOptions?.getDomContext?.();
          expect(activeDom?.isCurrent()).toBe(true);
          expect(executorOptions?.priorAttemptIds?.("f".repeat(64))).toEqual([]);
          expect(executorOptions?.isApiScopeCurrent?.()).toBe(true);
        }
        const reservation = await options.reserveProviderAttempt();
        await options.onAttemptSettled({
          attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId, outcome: "response",
          usage: { purpose: "agent", toolNames: [toolName], totalTokens: 24_338 },
        });
        callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: toolName, arguments: "{}" }] });
        await options.onCheckpoint({ phase: "working" });
        return { kind: "awaiting_user", ask, totalTokensUsed: 24_338 };
      }
      callbacks.onText(approved ? "The query finished." : "The query was not run.");
      return { kind: "completed", success: true, totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    useAgentStore.setState({ preferences: { ...store.preferences, tokenLimitPerRun: 100_000 } });
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    let focusCount = 0;
    const unsubscribe = useAgentStore.subscribe((state, previous) => {
      const awaiting = state.getChat(chat.id)?.status === "awaiting_user";
      if (awaiting && (previous.chats.find(candidate => candidate.id === chat.id)?.status !== "awaiting_user" || (previous.abortControllersByChatId[chat.id] && !state.abortControllersByChatId[chat.id]))) {
        queueMicrotask(() => {
          focusCount++;
          window.dispatchEvent(new Event("focus"));
        });
      }
    });
    store.queueUserMessage({ chatId: chat.id, text: "Run SELECT 1.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => {
          expect(store.getChat(chat.id)?.status).toBe("awaiting_user");
          expect(store.getAbortController(chat.id)).toBeNull();
        });
        expect(await store.flushRunPersistence()).toBe(true);
        window.dispatchEvent(new Event("focus"));
        expect(await store.flushRunPersistence()).toBe(true);
        store.syncState();
      });
      expect(focusCount).toBe(2);
      expect(store.getChat(chat.id)?.runBudget).toMatchObject({ limit: 100_000, totalTokensUsed: 24_338 });
      expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBeUndefined();
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      const allow = screen.getByRole("button", { name: "agent.api-permission-allow" });
      const deny = screen.getByRole("button", { name: "agent.api-permission-deny" });
      expect(allow).toBeEnabled();
      expect(deny).toBeEnabled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      if (origin === "dom-navigation") {
        const admit = useAgentStore.getState().checkAgentStorageAdmission;
        useAgentStore.setState({
          checkAgentStorageAdmission: async (...args) => {
            const previous = mocks.currentPath;
            mocks.currentPath = "/another-page";
            mocks.routeChanged();
            mocks.currentPath = previous;
            mocks.routeChanged();
            return admit(...args);
          },
        });
      }
      await act(async () => {
        (approved ? allow : deny).click();
        (approved ? allow : deny).click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ actionId: "query-action", approved: approved && origin !== "dom-navigation" }));
      expect(store.getMessages(chat.id).filter(message => message.role === "tool" && message.toolCallId === ask.toolCallId)).toHaveLength(1);
      expect(store.getPendingAsk(chat.id)).toBeNull();
      expect(new Set(runIds).size).toBe(1);
    } finally {
      unsubscribe();
      await store.flushRunPersistence();
      view.unmount();
    }
  });

  test.each(["available", "expired", "owned-elsewhere", "temporarily-blocked", "no-local-ticket", "live-without-preview"] as const)("automatically restores a pending API permission without approving it when its ticket is %s", async (ticket) => {
    let leaseAvailable = true;
    const checkLease = vi.fn(() => leaseAvailable);
    enableAsynchronousRunRecovery(checkLease);
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Run SELECT 1." })!;
    const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
    const noPreview = ticket === "no-local-ticket" || ticket === "live-without-preview";
    const apiPermission = {
      actionId: "restored-query-action",
      operationId: "bytebase.v1.SQLService.Query",
      ...(ticket === "live-without-preview" ? { expiresAt: Date.now() + 300_000 } : {}),
    };
    const ask: AgentPendingAsk = {
      kind: "confirm", toolCallId: "restored-query", prompt: "Run this query?",
      apiPermission,
    };
    const result = await withAgentRunLease(key, chat.id, async () => {
      const runId = await store.beginChatRun(chat.id, root.id, { page, context: "projects/orders", references: [], runMode: "default" });
      expect(runId).toBeTruthy();
      if (ticket === "live-without-preview") {
        expect(store.enqueueFollowUp({ chatId: chat.id, text: "List open issues", page })).toBeTruthy();
      }
      const assistant = store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "call_api", arguments: "{}" }], metadata: { runId: runId! } })!;
      expect(await store.saveChatRunCheckpoint(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true, throughMessageId: assistant.id, suspension: "awaiting-user" })).toBe(true);
      store.awaitUser(chat.id, ask);
      expect(await store.flushRunPersistence()).toBe(true);
      return runId;
    });
    expect(result.acquired).toBe(true);
    store.loadState();
    expect(await store.flushRunPersistence()).toBe(true);
    if (ticket !== "expired") useAgentStore.setState({ remoteBusyChatIds: { [chat.id]: true } });
    if (ticket !== "expired" && !noPreview) {
      mocks.getPendingApiCallPreview.mockReturnValue({
        operationId: "bytebase.v1.SQLService.Query", body: '{"name":"instances/example/databases/orders","statement":"SELECT 1","limit":100}', expiresAt: Date.now() + 300_000,
      });
    }
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
    leaseAvailable = ticket !== "owned-elsewhere" && ticket !== "temporarily-blocked";
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      const decisionLabel = ticket === "expired" || noPreview ? "agent.api-permission-dismiss" : "agent.api-permission-deny";
      const priorLeaseChecks = checkLease.mock.calls.length;
      await act(async () => {
        if (ticket === "owned-elsewhere" || ticket === "temporarily-blocked") {
          await vi.waitFor(() => expect(checkLease.mock.calls.length).toBeGreaterThan(priorLeaseChecks));
          expect(screen.getByRole("button", { name: "agent.api-permission-allow" })).toBeDisabled();
          expect(screen.getByRole("button", { name: decisionLabel })).toBeDisabled();
          if (ticket === "temporarily-blocked") leaseAvailable = true;
        }
        if (noPreview) {
          expect(await store.flushRunPersistence()).toBe(true);
          expect(checkLease).toHaveBeenCalledTimes(priorLeaseChecks);
        } else if (ticket !== "owned-elsewhere") {
          await vi.waitFor(() => expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBeUndefined());
        }
      });
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getChat(chat.id)).toMatchObject({ runId: result.value, status: "awaiting_user" });
      expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
      expect(screen.queryByRole("button", { name: "agent.token-budget.resume" })).toBeNull();
      if (ticket === "owned-elsewhere" || noPreview) {
        expect(screen.queryByText("agent.token-budget.action-failed")).toBeNull();
        expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);
        expect(screen.getByRole("button", { name: "agent.api-permission-allow" })).toBeDisabled();
        expect(screen.getByRole("button", { name: decisionLabel })).toBeDisabled();
        if (ticket === "live-without-preview") {
          expect(useAgentStore.getState().followUpQueuesByChatId[chat.id]?.messages.map(({ text }) => text)).toEqual(["List open issues"]);
        }
        return;
      }
      await waitForLeaseRelease(chat.id);
      expect(view.container.textContent).not.toContain("agent.token-budget.remote-busy");
      if (ticket !== "expired") {
        expect(screen.getByRole("button", { name: "agent.api-permission-allow" })).toBeEnabled();
        expect(screen.getByRole("button", { name: decisionLabel })).toBeEnabled();
      } else {
        expect(screen.getByRole("button", { name: "agent.api-permission-allow" })).toBeDisabled();
        expect(screen.getByRole("button", { name: decisionLabel })).toBeEnabled();
      }
      await act(async () => {
        screen.getByRole("button", { name: ticket !== "expired" ? "agent.api-permission-allow" : decisionLabel }).click();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
      });
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({ actionId: "restored-query-action", approved: ticket !== "expired" }));
      expect(store.getPendingAsk(chat.id)).toBeNull();
      if (ticket !== "expired") expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      else expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      await store.flushRunPersistence();
      view.unmount();
    }
  });

  test("restores an orphaned running checkpoint as interrupted without executing before Retry", async () => {
    enableAsynchronousRunRecovery();
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect this project." })!;
    const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
    const result = await withAgentRunLease(key, chat.id, async () => {
      const runId = await store.beginChatRun(chat.id, root.id, { page, context: "projects/orders", references: [], runMode: "default" });
      expect(runId).toBeTruthy();
      expect(await store.saveChatRunCheckpoint(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true })).toBe(true);
      return runId;
    });
    const persisted = localStorage.getItem(key)!;
    store.loadState();
    expect(await store.flushRunPersistence()).toBe(true);
    localStorage.setItem(key, persisted);
    store.syncState();
    expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);
    expect(store.getChat(chat.id)?.status).toBe("running");
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => expect(store.getChat(chat.id)).toMatchObject({ status: "idle", interrupted: true, interruptionReason: "unexpected" }));
      });
      expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBeUndefined();
      expect(store.getChat(chat.id)?.runId).toBe(result.value);
      expect(store.getChat(chat.id)?.runCheckpoint?.suspension).toBe("interrupted");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      expect(view.container.querySelector("[data-agent-token-budget]")).toBeNull();
      await act(async () => {
        retryActionRef.current?.();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.runId).toBeNull());
      });
      expect(mocks.runAgentLoop).toHaveBeenCalledExactlyOnceWith(expect.anything(), expect.anything(), expect.anything(), expect.anything(), expect.anything(), expect.objectContaining({ runId: result.value }));
    } finally {
      await store.flushRunPersistence();
      view.unmount();
    }
  });

  test.each(["paused", "error", "user-stopped", "completed"] as const)("clears only a stale busy marker for %s work without executing it", async (status) => {
    let leaseAvailable = true;
    const checkLease = vi.fn(() => leaseAvailable);
    enableAsynchronousRunRecovery(checkLease);
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect this project." })!;
    const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
    await withAgentRunLease(key, chat.id, async () => {
      const runId = await store.beginChatRun(chat.id, root.id, { page, context: "projects/orders", references: [], runMode: "default" });
      expect(runId).toBeTruthy();
      expect(await store.saveChatRunCheckpoint(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true })).toBe(true);
      const attempt = await store.reserveChatRunAttempt(chat.id, runId!);
      expect(attempt.kind).toBe("allowed");
      if (attempt.kind !== "allowed") throw new Error("Expected a provider attempt");
      expect(await store.recordChatModelAttempt(chat.id, { ...attempt, runId: runId!, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 123 } })).toBe(true);
      expect(store.getChat(chat.id)?.totalTokensUsed).toBe(123);
      if (status === "paused") expect(await store.pauseChatRun(chat.id, store.getChat(chat.id)!.runCheckpoint!)).toBe(true);
      else if (status === "error") store.setChatStatus(chat.id, "error", { lastError: "Provider unavailable" });
      else if (status === "user-stopped") store.interruptChatRun(chat.id, undefined, "user");
      else store.finishChatRun(chat.id);
      expect(await store.flushRunPersistence()).toBe(true);
    });
    const persisted = JSON.parse(localStorage.getItem(key)!);
    persisted.activeChatIds = [chat.id];
    localStorage.setItem(key, JSON.stringify(persisted));
    store.syncState();
    expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);
    const before = store.getChat(chat.id);
    const history = store.getMessages(chat.id);
    leaseAvailable = false;
    const priorLeaseChecks = checkLease.mock.calls.length;
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => expect(checkLease.mock.calls.length).toBeGreaterThan(priorLeaseChecks));
      });
      expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);
      expect(store.getChat(chat.id)).toEqual(before);
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
      await act(async () => {
        leaseAvailable = true;
        window.dispatchEvent(new Event("focus"));
        await vi.waitFor(() => expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBeUndefined());
      });
      expect(store.getChat(chat.id)).toEqual(before);
      expect(store.getChat(chat.id)?.totalTokensUsed).toBe(123);
      expect(store.getMessages(chat.id)).toEqual(history);
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    } finally {
      await store.flushRunPersistence();
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
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.retry-last-chat-turn");
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
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.retry-last-chat-turn");
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
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.retry-last-chat-turn");
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
      const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.retry-last-chat-turn");
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

  test("explains a failed question checkpoint and retries only after storage recovers", async () => {
    enableRunRecovery();
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    const ask: AgentPendingAsk = {
      kind: "choose", toolCallId: "exemption-scope", prompt: "Which databases?",
      options: [{ value: "project", label: "All databases in this project" }],
    };
    let write: ReturnType<typeof vi.spyOn> | undefined;
    mocks.runAgentLoop.mockImplementationOnce(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ role: "assistant", toolCalls: [{ id: ask.toolCallId, name: "ask_masking_exemption_scope", arguments: "{}" }] });
      write = vi.spyOn(localStorage, "setItem").mockImplementation(() => {
        throw new DOMException("Browser storage is full", "QuotaExceededError");
      });
      return { kind: "awaiting_user", ask, totalTokensUsed: 0 };
    }).mockResolvedValueOnce({ kind: "awaiting_user", ask, totalTokensUsed: 0 });
    store.queueUserMessage({ chatId: chat.id, text: "Let me access unmasked email.", page });
    const retryActionRef = createRef<(() => void) | null>();
    const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
      expect(store.getChat(chat.id)?.lastError).toContain("QuotaExceededError: Browser storage is full");
      const runId = store.getChat(chat.id)?.runId;
      await act(async () => { retryActionRef.current?.(); });
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(store.getChat(chat.id)?.lastError).toContain("QuotaExceededError: Browser storage is full");
      write?.mockRestore();
      await act(async () => { await store.flushRunPersistence(); });
      expect(useAgentStore.getState().storagePressure).toBeUndefined();
      expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      expect(mocks.executeTool).not.toHaveBeenCalled();
      await act(async () => {
        retryActionRef.current?.();
        await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user"));
      });
      expect(store.getChat(chat.id)?.runId).toBe(runId);
      expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
      expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
      expect(store.getMessages(chat.id).filter(message => message.role === "user")).toHaveLength(1);
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      write?.mockRestore();
      view.unmount();
    }
  });

  test("retains the cause of an unexpected managed run failure", async () => {
    enableRunRecovery();
    mocks.runAgentLoop.mockRejectedValueOnce(new Error("Unexpected run state"));
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.queueUserMessage({ chatId: chat.id, text: "Inspect orders.", page: { path: "/demo", title: "Demo" } });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("error")); });
      expect(store.getChat(chat.id)?.lastError).toContain("Error: Unexpected run state");
      expect(store.getChat(chat.id)?.runId).toBeTruthy();
    } finally {
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
        const resume = [...view.container.querySelectorAll("button")].find(button => button.textContent === "agent.retry-last-chat-turn");
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

  test.each([true, false])(
    "wraps full custom confirmation labels without changing the answer, confirmed=%s",
    async (confirmed) => {
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat(undefined, "projects/orders");
      const confirmLabel = "Prepare the unsaved database access request for review before submitting";
      const cancelLabel = "KeepTheCurrentDatabaseAccessUnchangedAndDoNotPrepareAnyRequest";
      store.awaitUser(chat.id, {
        toolCallId: "confirm-long-labels",
        kind: "confirm",
        prompt: "Prepare this unsaved draft?",
        confirmLabel,
        cancelLabel,
      });
      const view = renderIntoContainer(<AgentInput />);
      try {
        await act(async () => view.render());
        const scope = within(view.container);
        const confirm = scope.getByRole("button", { name: confirmLabel });
        const cancel = scope.getByRole("button", { name: cancelLabel });
        expect(view.container.querySelector("[data-agent-pending-preview]")).toHaveClass("min-h-0", "overflow-y-auto");
        const actions = confirm.closest("[data-agent-confirmation-actions]");
        expect(actions).toHaveClass("flex-wrap", "justify-end", "w-full");
        expect(actions).toContainElement(cancel);
        expect(cancel.compareDocumentPosition(confirm) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
        for (const button of [confirm, cancel]) {
          expect(button).toHaveClass("max-w-full", "min-w-0", "whitespace-normal", "wrap-anywhere", "h-auto", "min-h-7", "py-1");
          expect(button).not.toHaveClass("whitespace-nowrap", "h-7", "truncate");
        }
        expect(confirm).toHaveTextContent(confirmLabel);
        expect(cancel).toHaveTextContent(cancelLabel);
        expect(store.getPendingAsk(chat.id)?.toolCallId).toBe("confirm-long-labels");
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();

        await act(async () => (confirmed ? confirm : cancel).click());

        const response = store.getMessages(chat.id).find((message) => message.toolCallId === "confirm-long-labels");
        expect(JSON.parse(response?.content ?? "null")).toMatchObject({
          kind: "confirm",
          confirmed,
          answer: confirmed ? confirmLabel : cancelLabel,
        });
      } finally {
        view.unmount();
      }
    }
  );

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

  test("keeps the mention picker open while typing without a selected chat", async () => {
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
      expect(useAgentStore.getState().currentChatId).toBeNull();
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

  test.each(["typed", "mention"] as const)("collects a project %s immediately after selecting Choose a project", async (inputKind) => {
    mocks.currentContext = "workspaces/demo";
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: "projects/orders" }) });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo" });
    store.setDraft(chat.id, { text: "Unsent next question" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help me mask email" });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "scope-overview", name: "get_masking_overview", arguments: "{}" }] });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "scope-overview", content: "{}" });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "project-scope", name: "ask_masking_rule_scope", arguments: "{}" }] });
    store.awaitUser(chat.id, { toolCallId: "project-scope", kind: "choose", prompt: "Which databases?", options: [
      { label: "All workspace databases", value: "workspace" },
      { label: "Choose a project", value: "choose-project", kind: "custom-input" },
    ] });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      expect(view.container.querySelector("[data-agent-composer]")).toBeNull();
      act(() => screen.getByRole("radio", { name: "Choose a project" }).click());
      const textarea = screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" });
      expect(textarea).toHaveFocus();
      const reply = view.container.querySelector<HTMLButtonElement>("[data-agent-send]")!;
      expect(reply).toBeDisabled();
      await act(async () => {
        setTextareaValue(textarea, inputKind === "typed" ? "Order service" : "@");
        textarea.setSelectionRange(textarea.value.length, textarea.value.length);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      if (inputKind === "mention") {
        await act(async () => document.querySelector<HTMLButtonElement>('[data-agent-resource-category="project"]')?.click());
        await vi.waitFor(() => expect(document.querySelector('[data-agent-resource-project="projects/orders"]')).not.toBeNull());
        await act(async () => document.querySelector<HTMLButtonElement>('[data-agent-resource-project="projects/orders"]')?.click());
        expect(textarea.value).toBe("@Order service ");
      }
      expect(reply).toBeEnabled();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      await act(async () => { reply.click(); await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce()); });
      const response = store.getMessages(chat.id).find((message) => message.toolCallId === "project-scope");
      expect(JSON.parse(response?.content ?? "{}")).toEqual({ kind: "choose", value: "choose-project", answer: `Choose a project: ${inputKind === "typed" ? "Order service" : "@Order service"}`, input: inputKind === "typed" ? "Order service" : "@Order service" });
      expect(response?.metadata?.resourceReference).toEqual(inputKind === "mention" ? { kind: "project", name: "projects/orders", project: "projects/orders", label: "Order service" } : undefined);
      expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), inputKind === "mention"
        ? expect.objectContaining({ referencedProject: "projects/orders" })
        : expect.not.objectContaining({ referencedProject: expect.any(String) }));
      expect(mocks.executeTool).toHaveBeenCalledWith("get_masking_overview", {}, expect.any(String));
      expect(store.getChat(chat.id)?.context).toBe("workspaces/demo");
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
    } finally { view.unmount(); }
  });

  test.each(["project", "workspace-mention", "workspace-scope"] as const)("binds a database mention in a custom choice reply (%s)", async (scope) => {
    const databaseName = "instances/prod/databases/orders";
    mocks.executeTool.mockResolvedValue({
      kind: "tool_result",
      result: JSON.stringify({ project: "projects/orders", database: { status: "available", name: databaseName, state: "ACTIVE" } }),
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: scope === "project" ? "projects/orders" : "workspaces/demo" });
    store.setDraft(chat.id, { text: "Unsent next question" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Help me mask email in Order service", ...(scope === "workspace-mention" ? { metadata: { resourceReference: { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Order service" } } } : {}) });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "ask-scope", name: "ask_masking_rule_scope", arguments: '{"project":"projects/orders"}' }] });
    store.awaitUser(chat.id, {
      toolCallId: "ask-scope", kind: "choose", prompt: "Which databases?",
      ...(scope === "workspace-scope" ? { resourceProject: "projects/orders" } : {}),
      options: [
        { label: "All project databases", value: "projects/orders" },
        { label: "Another database", value: "another-database", kind: "custom-input" },
      ],
    });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      act(() => view.container.querySelectorAll<HTMLElement>('[role="radio"]')[1].click());
      let textarea = screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" });
      expect(textarea.closest("[data-agent-composer]")).not.toBeNull();
      await act(async () => {
        setTextareaValue(textarea, "Use @");
        textarea.setSelectionRange(5, 5);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
      await act(async () => document.querySelector<HTMLButtonElement>('[data-agent-resource-category="database"]')?.click());
      await vi.waitFor(() => expect(document.querySelector(`[data-agent-resource-database="${databaseName}"]`)).not.toBeNull());
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith("projects/orders", "", "", expect.any(AbortSignal));
      await act(async () => document.querySelector<HTMLButtonElement>(`[data-agent-resource-database="${databaseName}"]`)?.click());
      expect(textarea.value).toBe("Use @orders (prod) ");
      expect(view.container.querySelector("[data-agent-inline-mention]")?.textContent).toBe("@orders (prod)");
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
      act(() => view.container.querySelectorAll<HTMLElement>('[role="radio"]')[0].click());
      expect(view.container.querySelector("[data-agent-composer]")).toBeNull();
      act(() => view.container.querySelectorAll<HTMLElement>('[role="radio"]')[1].click());
      textarea = screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" });
      expect(textarea.value).toBe("Use @orders (prod) ");
      expect(view.container.querySelector("[data-agent-inline-mention]")).not.toBeNull();
      const reply = view.container.querySelector<HTMLButtonElement>("[data-agent-send]")!;
      await act(async () => {
        setTextareaValue(textarea, "Use @orders (prod) @unknown");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(reply).toBeDisabled();
      await act(async () => {
        setTextareaValue(textarea, "Use @orders (prod) ");
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      await act(async () => {
        reply.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      const response = store.getMessages(chat.id).find((message) => message.toolCallId === "ask-scope");
      expect(JSON.parse(response?.content ?? "{}")).toEqual({ kind: "choose", answer: "Another database: Use @orders (prod)", value: "another-database", input: "Use @orders (prod)" });
      expect(response?.metadata?.resourceReference).toEqual({ kind: "database", name: databaseName, project: "projects/orders", label: "orders (prod)" });
      expect(mocks.createToolExecutor).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ referencedDatabase: databaseName, referencedProject: "projects/orders" }));
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
    } finally {
      view.unmount();
    }
  });

  test("does not submit another custom choice's resource mention", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    const options = [
      { label: "First", value: "first", kind: "custom-input" as const },
      { label: "Second", value: "second", kind: "custom-input" as const },
    ];
    store.addMessage({ chatId: chat.id, role: "user", content: "Choose a value" });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: "custom", name: "ask_user", arguments: JSON.stringify({ kind: "choose", prompt: "Which value?", options }) }] });
    store.awaitUser(chat.id, { toolCallId: "custom", kind: "choose", prompt: "Which value?", options });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      act(() => view.container.querySelectorAll<HTMLElement>('[role="radio"]')[0].click());
      const textarea = screen.getByRole<HTMLTextAreaElement>("textbox");
      await act(async () => {
        setTextareaValue(textarea, "@");
        textarea.setSelectionRange(1, 1);
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      expect(document.querySelector("[data-agent-resource-picker]")).not.toBeNull();
      await act(async () => document.querySelector<HTMLButtonElement>('[data-agent-resource-category="database"]')?.click());
      await vi.waitFor(() => expect(document.querySelector('[data-agent-resource-database="instances/prod/databases/orders"]')).not.toBeNull());
      await act(async () => document.querySelector<HTMLButtonElement>('[data-agent-resource-database="instances/prod/databases/orders"]')?.click());
      act(() => view.container.querySelectorAll<HTMLElement>('[role="radio"]')[1].click());
      expect(view.container.querySelector("[data-agent-inline-mention]")).toBeNull();
      act(() => fireEvent.change(screen.getByRole("textbox"), { target: { value: "Plain second value" } }));
      await act(async () => {
        view.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      const response = store.getMessages(chat.id).find((message) => message.toolCallId === "custom");
      expect(JSON.parse(response?.content ?? "{}")).toMatchObject({ value: "second", input: "Plain second value" });
      expect(response?.metadata?.resourceReference).toBeUndefined();
      expect(response?.metadata?.resourceReferences).toBeUndefined();
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

  test("collects custom choice details inline and sends them with one Reply", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.setDraft(chat.id, { text: "Keep my draft" });
    store.awaitUser(chat.id, {
      toolCallId: "delivery",
      kind: "choose",
      prompt: "Which delivery method?",
      options: [
        { label: "Default", value: "default" },
        { label: "Another destination", value: "destination", kind: "custom-input", placeholder: "Enter a destination" },
        { label: "Another schedule", value: "schedule", kind: "custom-input" },
      ],
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const reply = () => [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply");
      const choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
      expect(view.container.querySelector("textarea")).toBeNull();
      act(() => choices[1].click());
      const details = screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" });
      expect(view.container.querySelector("[data-agent-input-placeholder]")?.textContent).toContain("Enter a destination");
      expect(details.closest("label")).toBeNull();
      expect(document.activeElement).toBe(details);
      const expectNeutralDetails = () => {
        expect(details.hasAttribute("required")).toBe(false);
        expect(details.hasAttribute("aria-invalid")).toBe(false);
        expect(details.hasAttribute("aria-describedby")).toBe(false);
        expect(screen.queryByRole("alert")).toBeNull();
        expect(screen.queryByText("agent.custom-input-required")).toBeNull();
      };
      expectNeutralDetails();
      expect(reply()?.disabled).toBe(true);
      act(() => fireEvent.blur(details));
      expectNeutralDetails();
      act(() => fireEvent.change(details, { target: { value: "   " } }));
      expect(reply()?.disabled).toBe(true);
      expectNeutralDetails();
      act(() => fireEvent.keyDown(details, { key: "Enter" }));
      expectNeutralDetails();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getMessages(chat.id).some((message) => message.toolCallId === "delivery")).toBe(false);
      act(() => fireEvent.change(details, { target: { value: "  audit@example.com  " } }));
      expect(reply()?.disabled).toBe(false);
      expectNeutralDetails();
      act(() => fireEvent.change(details, { target: { value: "" } }));
      expect(reply()?.disabled).toBe(true);
      act(() => fireEvent.keyDown(details, { key: "Enter" }));
      expectNeutralDetails();
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      expect(store.getPendingAsk(chat.id)?.toolCallId).toBe("delivery");
      expect(store.getMessages(chat.id).some((message) => message.toolCallId === "delivery")).toBe(false);
      act(() => fireEvent.change(details, { target: { value: "  audit@example.com  " } }));
      expectNeutralDetails();
      act(() => choices[2].click());
      const schedule = screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" });
      expect(schedule.value).toBe("");
      expect(view.container.querySelector("[data-agent-input-placeholder]")?.textContent).toContain("agent.custom-input-placeholder");
      act(() => fireEvent.change(schedule, { target: { value: "Every Monday" } }));
      act(() => choices[0].click());
      expect(view.container.querySelector("textarea")).toBeNull();
      expect(reply()?.disabled).toBe(false);
      act(() => choices[1].click());
      expect(screen.getByRole<HTMLTextAreaElement>("textbox", { name: "agent.custom-input-label" }).value).toBe("  audit@example.com  ");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      await act(async () => {
        reply()?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "delivery")?.content ?? "{}")).toEqual({
        kind: "choose", answer: "Another destination: audit@example.com", value: "destination", input: "audit@example.com",
      });
      expect(store.getDraft(chat.id).text).toBe("Keep my draft");
    } finally {
      view.unmount();
    }
  });

  test("does not attach inactive custom details to an ordinary choice", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    store.awaitUser(chat.id, {
      toolCallId: "method", kind: "choose", prompt: "Which method?",
      options: [
        { label: "Standard", value: "standard" },
        { label: "Other", value: "other", kind: "custom-input" },
      ],
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      const choices = view.container.querySelectorAll<HTMLElement>('[role="radio"]');
      act(() => choices[1].click());
      act(() => fireEvent.change(screen.getByRole("textbox"), { target: { value: "Unused custom details" } }));
      act(() => choices[0].click());
      await act(async () => {
        [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.reply")?.click();
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "method")?.content ?? "{}")).toEqual({ kind: "choose", answer: "Standard", value: "standard" });
    } finally {
      view.unmount();
    }
  });

  test.each([true, false])("custom choice input honors sendWithEnter=%s without sending during composition", async (sendWithEnter) => {
    const store = useAgentStore.getState();
    useAgentStore.setState({ preferences: { ...store.preferences, sendWithEnter } });
    const chat = store.createChat({ context: "projects/orders" });
    store.awaitUser(chat.id, {
      toolCallId: "custom", kind: "choose", prompt: "Which value?",
      options: [{ label: "Other", value: "other", kind: "custom-input" }],
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
      const details = screen.getByRole("textbox");
      act(() => fireEvent.change(details, { target: { value: "A custom value" } }));
      act(() => fireEvent.keyDown(details, { key: "Enter", isComposing: true, ctrlKey: !sendWithEnter }));
      act(() => fireEvent.keyDown(details, { key: "Enter", shiftKey: true }));
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      await act(async () => {
        fireEvent.keyDown(details, { key: "Enter", ctrlKey: !sendWithEnter });
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      expect(JSON.parse(store.getMessages(chat.id).find((message) => message.toolCallId === "custom")?.content ?? "{}").input).toBe("A custom value");
    } finally {
      view.unmount();
    }
  });

  test.each(["question", "chat", "account"] as const)("does not reuse custom details after the %s changes", (change) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    const ask: AgentPendingAsk = {
      toolCallId: "custom", kind: "choose", prompt: "Which value?",
      options: [{ label: "Other", value: "other", kind: "custom-input" }],
    };
    store.awaitUser(chat.id, ask);
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
      act(() => fireEvent.change(screen.getByRole("textbox"), { target: { value: "Private old details" } }));
      act(() => {
        if (change === "question") store.awaitUser(chat.id, { ...ask, toolCallId: "next" });
        if (change === "chat") {
          const other = store.createChat({ context: "projects/billing" });
          store.awaitUser(other.id, ask);
        }
        if (change === "account") useAgentStore.setState({ userEmail: "another@example.com" });
      });
      expect(view.container.querySelector("textarea")).toBeNull();
      act(() => view.container.querySelector<HTMLElement>('[role="radio"]')?.click());
      expect(screen.getByRole<HTMLTextAreaElement>("textbox").value).toBe("");
      expect(mocks.runAgentLoop).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
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

  test("shows a follow-up placeholder while locally running and restores the idle hint afterward", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    act(() => store.setChatStatus(chat.id, "running"));

    expect(
      view.container.querySelector("[data-agent-input-placeholder]")?.textContent
    ).toContain("agent.follow-up-queue.input-placeholder");
    expect(view.container.querySelector("textarea")).toBeEnabled();
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

  describe("pending follow-ups", () => {
    const success: AgentLoopOutcome = {
      kind: "completed",
      success: true,
      text: "Done",
      totalTokensUsed: 0,
    };

    const deferRun = (beforeFinish?: (callbacks: AgentCallbacks) => void) => {
      let settle!: (outcome: AgentLoopOutcome) => void;
      mocks.runAgentLoop.mockImplementationOnce(
        (_history, _tools, _executor, callbacks: AgentCallbacks) =>
          new Promise<AgentLoopOutcome>((resolve) => {
            settle = (outcome) => {
              beforeFinish?.(callbacks);
              if (outcome.kind === "completed")
                callbacks.onText?.(outcome.text);
              resolve(outcome);
            };
          })
      );
      return (outcome: AgentLoopOutcome) => settle(outcome);
    };

    const mountInput = (context = "projects/orders") => {
      const store = useAgentStore.getState();
      const chat = store.createChat({
        context,
        page: { path: "/demo", title: "Demo" },
      });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      return {
        ...view,
        chatId: chat.id,
        textarea: within(view.container).getByRole<HTMLTextAreaElement>(
          "textbox"
        ),
        userMessages: () =>
          store
            .getMessages(chat.id)
            .filter((message) => message.role === "user"),
      };
    };

    const submit = async (
      view: ReturnType<typeof mountInput>,
      text: string
    ) => {
      expect(view.textarea).toBeEnabled();
      act(() => {
        setTextareaValue(view.textarea, text);
        view.textarea.dispatchEvent(new Event("input", { bubbles: true }));
      });
      const send =
        view.container.querySelector<HTMLButtonElement>("[data-agent-send]");
      expect(send).toBeEnabled();
      await act(async () => send?.click());
    };

    const waitForRuns = async (count: number) => {
      await act(async () => {
        await vi.waitFor(() =>
          expect(mocks.runAgentLoop).toHaveBeenCalledTimes(count)
        );
      });
    };

    const getQueue = (view: ReturnType<typeof mountInput>) =>
      within(view.container).getByRole("list", {
        name: "agent.follow-up-queue.queued-count",
      });

    test.each(
      ["projects/orders", "workspaces/demo"].flatMap((context) =>
        [false, true].map((explicitDatabase) => ({ context, explicitDatabase }))
      )
    )(
      "preserves query context for a queued table question in $context, explicit database: $explicitDatabase",
      async ({ context, explicitDatabase }) => {
        enableAsynchronousRunRecovery();
        const { buildOutboundHistory } = await vi.importActual<
          typeof import("../logic/outboundHistory")
        >("../logic/outboundHistory");
        mocks.buildOutboundHistory.mockImplementation(buildOutboundHistory);
        const finishFirst = deferRun();
        const finishFollowUp = deferRun();
        const view = mountInput(context);
        const store = useAgentStore.getState();
        const database = "instances/main/databases/orders";
        const otherDatabase = "instances/main/databases/archive";
        const statement = "SELECT * FROM employee";
        mocks.executeTool.mockImplementation(async (_name, args) => ({
          kind: "tool_result",
          result: JSON.stringify({
            project: "projects/orders",
            database: {
              status: "available",
              state: "ACTIVE",
              name: args.databaseName,
            },
          }),
        }));
        const firstText = "@Orders select * from employee";
        const followUpText = explicitDatabase
          ? "@Archive how many data in this table"
          : "how many data in this table";
        let firstFinished = false;
        let followUpFinished = false;
        try {
          act(() =>
            store.setDraft(view.chatId, {
              text: firstText,
              mentions: [
                {
                  resource: {
                    kind: "database",
                    name: database,
                    project: "projects/orders",
                    label: "Orders",
                  },
                  range: { start: 0, end: 7 },
                },
              ],
            })
          );
          await submit(view, firstText);
          await waitForRuns(1);
          if (explicitDatabase)
            act(() =>
              store.setDraft(view.chatId, {
                text: followUpText,
                mentions: [
                  {
                    resource: {
                      kind: "database",
                      name: otherDatabase,
                      project: "projects/orders",
                      label: "Archive",
                    },
                    range: { start: 0, end: 8 },
                  },
                ],
              })
            );
          await submit(view, followUpText);
          expect(getQueue(view)).toHaveTextContent(followUpText);
          const callbacks = mocks.runAgentLoop.mock.calls[0][3] as AgentCallbacks;
          await act(async () => {
            callbacks.onAssistantMessage?.({
              toolCalls: [
                {
                  id: "originating-query",
                  name: "call_api",
                  arguments: JSON.stringify({
                    operationId: "bytebase.v1.SQLService.Query",
                    body: { name: database, statement },
                  }),
                },
              ],
            });
            await callbacks.onToolResult?.(
              "originating-query",
              JSON.stringify({
                status: 200,
                response: {
                  results: [
                    {
                      columnNames: ["id"],
                      rows: [{ values: [{ stringValue: "PRIVATE_ROW_VALUE" }] }],
                    },
                  ],
                },
              })
            );
            firstFinished = true;
            finishFirst({
              ...success,
              text: "The query returned employee rows.",
            });
          });
          await waitForRuns(2);

          const messages = store.getMessages(view.chatId);
          const answer = messages.find(
            (message) =>
              message.role === "assistant" &&
              message.content === "The query returned employee rows."
          );
          expect(answer).toBeDefined();
          const followUp = view.userMessages().at(-1)!;
          const history = mocks.runAgentLoop.mock.calls[1][0];
          const system =
            history.find(
              (message: { role: string }) => message.role === "system"
            )?.content ?? "";
          if (explicitDatabase) {
            expect(followUp.metadata?.followUpSourceId).toBeUndefined();
            expect(mocks.buildSystemPrompt).toHaveBeenNthCalledWith(
              2,
              expect.objectContaining({ referencedDatabase: otherDatabase })
            );
            expect(system).not.toContain("Historical database context");
          } else {
            expect(system).toContain("Historical database context");
            expect(system).toContain(database);
            expect(system).toContain(JSON.stringify({ statement }));
            expect(system).toContain(
              "not an active selection and not authorization"
            );
            expect(followUp.metadata?.followUpSourceId).toBe(answer!.id);
            expect(followUp.metadata?.resourceReference).toBeUndefined();
            expect(followUp.metadata?.resourceReferences).toBeUndefined();
            expect(mocks.buildSystemPrompt).toHaveBeenNthCalledWith(
              2,
              expect.objectContaining({ referencedDatabase: undefined })
            );
          }
          expect(JSON.stringify(history)).not.toContain("PRIVATE_ROW_VALUE");
          await act(async () => {
            followUpFinished = true;
            finishFollowUp(success);
          });
          await waitForLeaseRelease(view.chatId);
        } finally {
          await act(async () => {
            if (mocks.runAgentLoop.mock.calls.length > 0 && !firstFinished)
              finishFirst({ kind: "aborted", totalTokensUsed: 0 });
            if (mocks.runAgentLoop.mock.calls.length > 1 && !followUpFinished)
              finishFollowUp({ kind: "aborted", totalTokensUsed: 0 });
          });
          if (mocks.runAgentLoop.mock.calls.length > 0)
            await waitForLeaseRelease(view.chatId);
          view.unmount();
        }
      }
    );

    const holdPersistence = async () => {
      const store = useAgentStore.getState();
      const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
      let entered!: () => void;
      let release!: () => void;
      const locked = new Promise<void>((resolve) => {
        entered = resolve;
      });
      const gate = new Promise<void>((resolve) => {
        release = resolve;
      });
      const holding = withAgentPersistenceLock(key, () => {
        entered();
        return gate;
      });
      await locked;
      return async () => {
        release();
        await holding;
      };
    };

    const observeNextStateWrite = () => {
      const store = useAgentStore.getState();
      const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
      const setItem = localStorage.setItem.bind(localStorage);
      let finish!: () => void;
      const written = new Promise<void>((resolve) => { finish = resolve; });
      const spy = vi.spyOn(localStorage, "setItem").mockImplementation((name, value) => {
        setItem(name, value);
        if (name === key) finish();
      });
      return { written, restore: () => spy.mockRestore() };
    };

    test("keeps the composer cleared after the queued follow-up is persisted", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      const view = mountInput();
      let release: (() => Promise<void>) | undefined;
      let restoreWrite: (() => void) | undefined;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await act(async () => {
          expect(await useAgentStore.getState().flushRunPersistence()).toBe(true);
        });
        release = await holdPersistence();
        await submit(view, "List database owners");
        expect(getQueue(view)).toHaveTextContent("List database owners");
        expect(view.textarea).toHaveValue("");
        const write = observeNextStateWrite();
        restoreWrite = write.restore;
        await act(async () => {
          await release?.();
          await write.written;
        });

        expect(view.textarea).toHaveValue("");
        expect(useAgentStore.getState().getDraft(view.chatId)).toEqual({
          text: "",
        });
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
        ]);
      } finally {
        await release?.();
        restoreWrite?.();
        await act(async () =>
          finishFirst({ kind: "aborted", totalTokensUsed: 0 })
        );
        await waitForLeaseRelease(view.chatId);
        view.unmount();
      }
    });

    test("preserves follow-up typing across repeated run progress writes", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      const view = mountInput();
      let release: (() => Promise<void>) | undefined;
      let restoreWrite: (() => void) | undefined;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await act(async () => { expect(await useAgentStore.getState().flushRunPersistence()).toBe(true); });
        const callbacks = mocks.runAgentLoop.mock.calls[0][3] as AgentCallbacks;
        for (const text of ["Check", "Check grants", "Check grants for this database"]) {
          release = await holdPersistence();
          act(() => {
            callbacks.onAssistantMessage?.({ content: `Progress: ${text}` });
            setTextareaValue(view.textarea, text);
            view.textarea.dispatchEvent(new Event("input", { bubbles: true }));
          });
          expect(view.textarea).toHaveValue(text);
          const write = observeNextStateWrite();
          restoreWrite = write.restore;
          await act(async () => {
            await release?.();
            await write.written;
          });
          write.restore();
          restoreWrite = undefined;
          expect(view.textarea).toHaveValue(text);
          expect(useAgentStore.getState().getDraft(view.chatId).text).toBe(
            text
          );
        }
      } finally {
        await release?.();
        restoreWrite?.();
        await act(async () =>
          finishFirst({ kind: "aborted", totalTokensUsed: 0 })
        );
        await waitForLeaseRelease(view.chatId);
        view.unmount();
      }
    });

    test("preserves a newer composer draft written synchronously during enqueue", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      const view = mountInput();
      let unsubscribe: (() => void) | undefined;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        unsubscribe = useAgentStore.subscribe((state, previous) => {
          if (
            state.followUpQueuesByChatId[view.chatId]?.messages.length &&
            !previous.followUpQueuesByChatId[view.chatId]?.messages.length
          ) {
            state.setDraft(view.chatId, { text: "Keep this newer question" });
          }
        });
        await submit(view, "List database owners");

        expect(getQueue(view)).toHaveTextContent("List database owners");
        expect(view.textarea).toHaveValue("Keep this newer question");
        await act(async () => {
          expect(await useAgentStore.getState().flushRunPersistence()).toBe(true);
        });
        expect(useAgentStore.getState().getDraft(view.chatId).text).toBe(
          "Keep this newer question"
        );
      } finally {
        unsubscribe?.();
        await act(async () =>
          finishFirst({ kind: "aborted", totalTokensUsed: 0 })
        );
        await waitForLeaseRelease(view.chatId);
        view.unmount();
      }
    });

    test("never starts a follow-up deleted while checking its AI configuration", async () => {
      const finishFirst = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = mountInput();
      let resolveConfiguration!: (value: undefined) => void;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await submit(view, "Delete this queued question");
        await submit(view, "List open issues");
        mocks.getOrFetchSettingByName.mockImplementationOnce(
          () =>
            new Promise<undefined>((resolve) => {
              resolveConfiguration = resolve;
            })
        );
        await act(async () => {
          finishFirst(success);
          await vi.waitFor(() =>
            expect(resolveConfiguration).toBeTypeOf("function")
          );
        });

        const items = within(getQueue(view)).getAllByRole("listitem");
        expect(items[0]).toHaveTextContent("Delete this queued question");
        const remove = within(items[0]).getByRole("button", {
          name: "agent.follow-up-queue.remove",
        });
        await act(async () => remove.click());
        expect(getQueue(view)).not.toHaveTextContent(
          "Delete this queued question"
        );

        await act(async () => resolveConfiguration(undefined));
        await waitForRuns(2);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
          "List open issues",
        ]);
      } finally {
        await act(async () => resolveConfiguration?.(undefined));
        view.unmount();
      }
    });

    test.each([
      { ...success, success: false, text: "Could not finish" },
      {
        kind: "error",
        error: new Error("Provider unavailable"),
        totalTokensUsed: 0,
      },
    ] satisfies AgentLoopOutcome[])(
      "pauses a follow-up after an unsuccessful $kind until Resume",
      async (outcome) => {
        enableAsynchronousRunRecovery();
        const finishFirst = deferRun();
        mocks.runAgentLoop.mockResolvedValue(success);
        const view = mountInput();
        try {
          await submit(view, "List databases");
          await waitForRuns(1);
          await submit(view, "List database owners");
          await act(async () => {
            finishFirst(outcome);
            await vi.waitFor(() => {
              const state = useAgentStore.getState();
              expect(state.followUpQueuesByChatId[view.chatId]?.paused).toBe(
                true
              );
              expect(state.getAbortController(view.chatId)).toBeNull();
            });
          });
          await waitForLeaseRelease(view.chatId);

          expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
          expect(view.userMessages().map(({ content }) => content)).toEqual([
            "List databases",
          ]);
          expect(getQueue(view)).toHaveTextContent("List database owners");
          const resume = within(view.container).getByRole("button", {
            name: "agent.follow-up-queue.resume",
          });
          expect(resume).toBeEnabled();
          await act(async () => resume.click());
          await waitForRuns(2);
          await waitForLeaseRelease(view.chatId);
          expect(view.userMessages().map(({ content }) => content)).toEqual([
            "List databases",
            "List database owners",
          ]);
        } finally {
          view.unmount();
        }
      }
    );

    test("keeps the queue paused if the stopped provider later reports success", async () => {
      const finishFirst = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = mountInput();
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await submit(view, "List database owners");
        await act(async () =>
          within(view.container)
            .getByRole("button", { name: "agent.stop" })
            .click()
        );
        await act(async () => finishFirst(success));

        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
        ]);
        expect(getQueue(view)).toHaveTextContent("List database owners");
        expect(
          within(view.container).getByRole("button", {
            name: "agent.follow-up-queue.resume",
          })
        ).toBeEnabled();
      } finally {
        view.unmount();
      }
    });

    test("advances pending follow-ups after the composer unmounts", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = mountInput();
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await submit(view, "List database owners");
      } finally {
        view.unmount();
      }

      await act(async () => finishFirst(success));
      await waitForRuns(2);
      await waitForLeaseRelease(view.chatId);
      expect(view.userMessages().map(({ content }) => content)).toEqual([
        "List databases",
        "List database owners",
      ]);
    });

    test("does not start a promoted follow-up stopped before its initial checkpoint is durable", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const store = useAgentStore.getState();
      const view = mountInput();
      let releaseCheckpoint: (() => void) | undefined;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await submit(view, "List database owners");
        await submit(view, "List open issues");

        const flush = async () => {
          if (
            !releaseCheckpoint &&
            store.getChat(view.chatId)?.status === "running" &&
            view.userMessages().at(-1)?.content === "List database owners" &&
            !store.getAbortController(view.chatId)
          ) {
            await new Promise<void>((resolve) => {
              releaseCheckpoint = resolve;
            });
          }
          return store.flushRunPersistence();
        };
        act(() => useAgentStore.setState({ flushRunPersistence: flush }));
        await act(async () => {
          finishFirst(success);
          await vi.waitFor(() =>
            expect(releaseCheckpoint).toBeTypeOf("function")
          );
        });
        const stop = within(view.container).getByRole("button", {
          name: "agent.stop",
        });
        await act(async () => stop.click());
        await act(async () => releaseCheckpoint?.());
        await waitForLeaseRelease(view.chatId);

        expect(mocks.createToolExecutor).toHaveBeenCalledOnce();
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(store.getChat(view.chatId)).toMatchObject({
          status: "idle",
          interrupted: true,
          interruptionReason: "user",
          lastError: null,
        });
        expect(getQueue(view)).toHaveTextContent("List open issues");
        expect(
          within(view.container).getByRole("button", {
            name: "agent.follow-up-queue.resume",
          })
        ).toBeEnabled();
      } finally {
        await act(async () => releaseCheckpoint?.());
        act(() =>
          useAgentStore.setState({
            flushRunPersistence: store.flushRunPersistence,
          })
        );
        view.unmount();
      }
    });

    test.each(["same", "peer"])("a new request runs with its normal tools after unresolved history in the %s chat", async (owner) => {
      enableAsynchronousRunRecovery();
      const store = useAgentStore.getState();
      const view = mountInput();
      try {
        const { normalizeAgentApiAttempts } = await import("../logic/apiAttempt");
        const chatId = owner === "same" ? view.chatId : store.createChat({ select: false }).id;
        act(() => useAgentStore.setState(s => {
          s.chats.find(c => c.id === chatId)!.apiAttempts = normalizeAgentApiAttempts([null], chatId);
        }));
        await store.flushRunPersistence();
        mocks.runAgentLoop.mockResolvedValue(success);
        await submit(view, "Query the referenced database");
        await waitForRuns(1);
        await waitForLeaseRelease(view.chatId);
        expect(view.userMessages()).toHaveLength(1);
        expect(mocks.getToolDefinitions).not.toHaveBeenCalledWith(expect.objectContaining({ apiRecoveryOnly: true }));
      } finally { view.unmount(); }
    });

    test("optional history sync failure does not create a repeat warning or restrict execution", async () => {
      enableAsynchronousRunRecovery();
      const view = mountInput();
      const finish = deferRun();
      const syncState = useAgentStore.getState().syncState;
      try {
        await submit(view, "Change the project title");
        await waitForRuns(1);
        const calls = mocks.createToolExecutor.mock.calls as unknown as Parameters<typeof createToolExecutor>[];
        act(() => useAgentStore.setState({ syncState: () => false }));
        expect(calls.at(-1)?.[1]?.priorAttemptIds?.("f".repeat(64))).toEqual([]);
        expect(calls.at(-1)?.[1]?.isApiScopeCurrent?.()).toBe(true);
      } finally {
        act(() => useAgentStore.setState({ syncState }));
        await act(async () => finish(success));
        await waitForLeaseRelease(view.chatId);
        view.unmount();
      }
    });
    test("approval response loss continues the explanation and ordinary queued follow-up without replaying the request", async () => {
      enableAsynchronousRunRecovery();
      const state = useAgentStore.getState();
      mocks.currentUserEmail = "me@example.com";
      state.setWorkspaceName("workspaces/demo", "me@example.com");
      const call = { operationId: "bytebase.v1.ProjectService.UpdateProject", path: "/bytebase.v1.ProjectService/UpdateProject", body: '{"project":{"name":"projects/orders","title":"Changed"}}' };
      const ask: AgentPendingAsk = { kind:"confirm", toolCallId:"write", prompt:call.operationId,
        apiPermission: { actionId:"exact-ticket", operationId:call.operationId } };
      mocks.getPendingApiCallPreview.mockReturnValue({ ...call, expiresAt: Date.now() + 300_000 });
      const { dispatchApprovedApiCall } = await import("../logic/tools/callApi");
      const fetch = vi.fn().mockRejectedValue(new Error("response lost"));
      vi.stubGlobal("fetch", fetch);
      mocks.resolvePendingApiCall.mockImplementation(({context, signal}) => dispatchApprovedApiCall(call, { context, signal }));
      const first = deferRun(callbacks => callbacks.onAssistantMessage?.({ toolCalls:[{id:"write", name:"call_api", arguments:JSON.stringify({operationId:call.operationId})}] }));
      const continuation = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = mountInput();
      try {
        await submit(view, "Change title"); await waitForRuns(1);
        await submit(view, "dependent follow-up");
        await act(async () => { first({kind:"awaiting_user", ask}); });
        await waitForLeaseRelease(view.chatId);
        await act(async () => within(view.container).getByRole("button", {name:"agent.api-permission-allow"}).click());
        await waitForRuns(2);
        expect(fetch).toHaveBeenCalledTimes(1);
        const [attempt] = state.getChat(view.chatId)!.apiAttempts!;
        expect(attempt).toMatchObject({ phase:"settled", toolCallId:"write", logicalActionId:"exact-ticket", execution:{result:"unknown", effects:"possible"} });
        expect(state.getMessages(view.chatId).filter(m => m.toolCallId === "write")).toHaveLength(1);
        expect(state.getMessages(view.chatId)).toEqual(expect.arrayContaining([expect.objectContaining({ role: "tool", content: expect.stringContaining('"result":"unknown"') })]));
        expect(state.getChat(view.chatId)?.status).toBe("running");
        expect(useAgentStore.getState().followUpQueuesByChatId[view.chatId]?.messages.map(m => m.text)).toEqual(["dependent follow-up"]);
        await act(async () => continuation(success));
        await waitForRuns(3);
        await waitForLeaseRelease(view.chatId);
        expect(view.userMessages().map(m => m.content)).toEqual(["Change title", "dependent follow-up"]);
        expect(fetch).toHaveBeenCalledTimes(1);
        expect(state.getChat(view.chatId)?.status).toBe("idle");
      } finally { view.unmount(); }
    });
    test("keeps and observes a later READ result with a settled write's provider ID", async () => {
      enableAsynchronousRunRecovery();
      const state = useAgentStore.getState();
      mocks.currentUserEmail = "me@example.com";
      state.setWorkspaceName("workspaces/demo", "me@example.com");
      const observe = vi.spyOn(
        await import("../logic/handoff"),
        "collectAgentHandoffs"
      );
      const call = {
        operationId: "bytebase.v1.ProjectService.UpdateProject",
        path: "/bytebase.v1.ProjectService/UpdateProject",
        body: '{"project":{"name":"projects/orders","title":"Changed"}}',
      };
      const { agentApiExecutionEvidence } = await import("../logic/apiExecution");
      const readResult = JSON.stringify({
        status: 200,
        response: { name: "projects/orders", title: "Changed" },
        agentExecution: agentApiExecutionEvidence("rpc-success", {
          httpStatus: 200,
        }),
      });
      const fetch = vi.fn(
        async () =>
          new Response('{"name":"projects/orders","title":"Changed"}', {
            status: 200,
            headers: { "content-type": "application/json" },
          })
      );
      vi.stubGlobal("fetch", fetch);
      const { dispatchApprovedApiCall } = await import("../logic/tools/callApi");
      let writeResult = "";
      const view = mountInput();
      mocks.runAgentLoop.mockImplementationOnce(
        async (_history, _tools, _executor, callbacks, signal, options) => {
          const toolCallId = "reused-api-id";
          callbacks.onAssistantMessage({
            toolCalls: [{
              id: toolCallId,
              name: "call_api",
              arguments: JSON.stringify({ operationId: call.operationId }),
            }],
          });
          const context = state.createApiDispatchContext({
            chatId: view.chatId,
            runId: options.runId!,
            rootUserMessageId: view.userMessages()[0].id,
            toolCallId,
            logicalActionId: "approved-title-change",
          });
          writeResult = await dispatchApprovedApiCall(call, {
            context: context!,
            signal,
          });
          await callbacks.onToolResult(toolCallId, writeResult);
          await callbacks.onToolResult(toolCallId, writeResult);
          callbacks.onAssistantMessage({
            toolCalls: [{
              id: toolCallId,
              name: "call_api",
              arguments: JSON.stringify({
                operationId: "bytebase.v1.ProjectService.GetProject",
              }),
            }],
          });
          await callbacks.onToolResult(toolCallId, readResult);
          callbacks.onText("The project title was changed and verified.");
          return success;
        }
      );
      try {
        await submit(view, "Change and verify the project title");
        await waitForRuns(1);
        await waitForLeaseRelease(view.chatId);
        const [attempt] = state.getChat(view.chatId)!.apiAttempts!;
        expect(attempt).toMatchObject({
          phase: "settled",
          toolCallId: "reused-api-id",
        });
        expect(
          state.getApiAttemptToolResult(view.chatId, attempt.id)?.content
        ).toBe(writeResult);
        const results = state.getMessages(view.chatId).filter(
          (message) =>
            message.role === "tool" &&
            message.toolCallId === "reused-api-id"
        );
        expect(results.map((message) => message.content)).toEqual([
          writeResult,
          readResult,
        ]);
        expect(observe).not.toHaveBeenCalledWith(
          "call_api",
          writeResult,
          expect.anything()
        );
        expect(observe).toHaveBeenCalledWith(
          "call_api",
          readResult,
          expect.objectContaining({ projectName: "projects/orders" })
        );
        expect(fetch).toHaveBeenCalledOnce();
      } finally {
        view.unmount();
      }
    });

    test("holds follow-ups for an API approval until its own turn finishes", async () => {
      enableAsynchronousRunRecovery();
      const ask: AgentPendingAsk = {
        kind: "confirm",
        toolCallId: "write-project",
        prompt: "Update the project title?",
        apiPermission: {
          actionId: "approval-action",
          operationId: "bytebase.v1.ProjectService.UpdateProject",
        },
      };
      mocks.getPendingApiCallPreview.mockReturnValue({
        ...ask.apiPermission,
        body: '{"project":{"name":"projects/orders","title":"Orders"}}',
        priorAttemptIds: ["externally-reviewed-attempt"],
        expiresAt: Date.now() + 300_000,
      });
      const finishFirst = deferRun((callbacks) => {
        callbacks.onAssistantMessage?.({
          toolCalls: [
            { id: ask.toolCallId, name: "call_api", arguments: "{}" },
          ],
        });
      });
      const finishContinuation = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = mountInput();
      try {
        await submit(view, "Update the project title");
        await waitForRuns(1);
        await submit(view, "List database owners");
        await act(async () => {
          finishFirst({ kind: "awaiting_user", ask, totalTokensUsed: 0 });
          await vi.waitFor(() => {
            const state = useAgentStore.getState();
            expect(state.getChat(view.chatId)?.status).toBe("awaiting_user");
            expect(state.getAbortController(view.chatId)).toBeNull();
          });
        });
        await waitForLeaseRelease(view.chatId);

        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
        expect(getQueue(view)).toHaveTextContent("List database owners");
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "Update the project title",
        ]);

        const approve = within(view.container).getByRole("button", {
          name: "agent.api-permission-allow",
        });
        expect(view.container).toHaveTextContent("agent.api-recovery.repeat-risk");
        await act(async () => approve.click());
        await waitForRuns(2);
        expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
        expect(mocks.runAgentLoop.mock.calls[1][5].runId).toBe(
          mocks.runAgentLoop.mock.calls[0][5].runId
        );
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "Update the project title",
        ]);

        await act(async () => finishContinuation(success));
        await waitForRuns(3);
        await waitForLeaseRelease(view.chatId);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "Update the project title",
          "List database owners",
        ]);
      } finally {
        view.unmount();
      }
    });

    test("pauses follow-ups when the owner dismisses an expired API approval", async () => {
      enableAsynchronousRunRecovery();
      const apiPermission = {
        actionId: "expiring-project-action",
        operationId: "bytebase.v1.ProjectService.UpdateProject",
        expiresAt: Date.now() + 300_000,
      };
      const ask: AgentPendingAsk = {
        kind: "confirm",
        toolCallId: "expiring-project-call",
        prompt: "Update the project title?",
        apiPermission,
      };
      mocks.getPendingApiCallPreview.mockReturnValue({
        ...apiPermission,
        body: '{"project":{"name":"projects/orders","title":"Orders"}}',
      });
      const finishFirst = deferRun((callbacks) => {
        callbacks.onAssistantMessage?.({
          toolCalls: [
            { id: ask.toolCallId, name: "call_api", arguments: "{}" },
          ],
        });
      });
      mocks.runAgentLoop.mockResolvedValue(success);
      const store = useAgentStore.getState();
      const view = mountInput();
      const scope = within(view.container);
      try {
        await submit(view, "Update the project title");
        await waitForRuns(1);
        await submit(view, "List open issues");
        await act(async () => {
          finishFirst({ kind: "awaiting_user", ask, totalTokensUsed: 0 });
          await vi.waitFor(() =>
            expect(store.getChat(view.chatId)?.status).toBe("awaiting_user")
          );
        });
        await waitForLeaseRelease(view.chatId);
        expect(
          scope.getByRole("button", { name: "agent.api-permission-allow" })
        ).toBeEnabled();

        vi.spyOn(Date, "now").mockReturnValue(apiPermission.expiresAt + 1);
        mocks.getPendingApiCallPreview.mockReturnValue(undefined);
        view.render(<AgentInput />);
        await act(async () => {
          scope
            .getByRole("button", { name: "agent.api-permission-dismiss" })
            .click();
          await vi.waitFor(() =>
            expect(store.getChat(view.chatId)?.status).toBe("idle")
          );
        });
        await waitForLeaseRelease(view.chatId);

        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalledWith(
          expect.objectContaining({ approved: true })
        );
        expect(getQueue(view)).toHaveTextContent("List open issues");
        expect(
          scope.getByRole("button", { name: "agent.follow-up-queue.resume" })
        ).toBeEnabled();
      } finally {
        view.unmount();
      }
    });

    test("restores a clarification without replaying its pending follow-up", async () => {
      let leaseAvailable = true;
      const checkLease = vi.fn(() => leaseAvailable);
      enableAsynchronousRunRecovery(checkLease);
      const store = useAgentStore.getState();
      const page = { path: "/demo", title: "Demo" };
      const chat = store.createChat({ context: "projects/orders", page });
      const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
      const ask = {
        kind: "input",
        toolCallId: "which-database",
        prompt: "Which database?",
      } as const;
      const first = store.addMessage({
        chatId: chat.id,
        role: "user",
        content: "Inspect a database",
      })!;
      const run = await withAgentRunLease(key, chat.id, async () => {
        const runId = await store.beginChatRun(chat.id, first.id, {
          page,
          context: "projects/orders",
          references: [],
          runMode: "default",
        });
        expect(runId).toBeTruthy();
        expect(
          store.enqueueFollowUp({
            chatId: chat.id,
            text: "List database owners",
            page,
          })
        ).toBeTruthy();
        const assistant = store.addMessage({
          chatId: chat.id,
          role: "assistant",
          toolCalls: [
            {
              id: ask.toolCallId,
              name: "ask_user",
              arguments: JSON.stringify({ kind: ask.kind, prompt: ask.prompt }),
            },
          ],
          metadata: { runId: runId! },
        })!;
        expect(
          await store.saveChatRunCheckpoint(chat.id, {
            ...store.getChat(chat.id)!.runCheckpoint!,
            startupComplete: true,
            throughMessageId: assistant.id,
            suspension: "awaiting-user",
          })
        ).toBe(true);
        store.awaitUser(chat.id, ask);
        expect(await store.flushRunPersistence()).toBe(true);
        return runId;
      });
      expect(run.acquired).toBe(true);
      store.loadState();
      expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);

      leaseAvailable = false;
      checkLease.mockClear();
      mocks.runAgentLoop.mockResolvedValue(success);
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      const scope = within(view.container);
      try {
        await act(async () => {
          await vi.waitFor(() => expect(checkLease).toHaveBeenCalled());
        });
        expect(scope.getByRole("textbox", { name: ask.prompt })).toBeDisabled();
        expect(
          scope.getByRole("button", { name: "agent.follow-up-queue.remove" })
        ).toBeDisabled();
        expect(useAgentStore.getState().remoteBusyChatIds[chat.id]).toBe(true);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();

        await act(async () => {
          leaseAvailable = true;
          window.dispatchEvent(new Event("focus"));
          await vi.waitFor(() =>
            expect(
              useAgentStore.getState().remoteBusyChatIds[chat.id]
            ).toBeUndefined()
          );
        });
        await waitForLeaseRelease(chat.id);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
        expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
        expect(
          useAgentStore.getState().followUpQueuesByChatId[chat.id]?.paused
        ).toBe(true);
        const textarea = scope.getByRole<HTMLTextAreaElement>("textbox", {
          name: ask.prompt,
        });
        expect(textarea).toBeEnabled();
        act(() => {
          setTextareaValue(textarea, "orders");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () => {
          scope.getByRole("button", { name: "agent.reply" }).click();
          await vi.waitFor(() => {
            expect(store.getChat(chat.id)?.status).toBe("idle");
            expect(store.getAbortController(chat.id)).toBeNull();
          });
        });
        await waitForLeaseRelease(chat.id);
        expect(mocks.runAgentLoop).toHaveBeenCalledExactlyOnceWith(
          expect.anything(),
          expect.anything(),
          expect.anything(),
          expect.anything(),
          expect.anything(),
          expect.objectContaining({ runId: run.value })
        );
        expect(
          store
            .getMessages(chat.id)
            .find(({ toolCallId }) => toolCallId === ask.toolCallId)?.content
        ).toBe('{"kind":"input","answer":"orders"}');

        const resume = scope.getByRole("button", {
          name: "agent.follow-up-queue.resume",
        });
        expect(resume).toBeEnabled();
        await act(async () => resume.click());
        await waitForRuns(2);
        await waitForLeaseRelease(chat.id);
        expect(
          store
            .getMessages(chat.id)
            .filter(({ role }) => role === "user")
            .map(({ content }) => content)
        ).toEqual(["Inspect a database", "List database owners"]);
      } finally {
        view.unmount();
        await store.flushRunPersistence();
      }
    });

    test("uses the queued page and resource reference after navigation", async () => {
      enableAsynchronousRunRecovery();
      const ask = {
        kind: "input",
        toolCallId: "queued-clarification",
        prompt: "What should I look for?",
      } as const;
      const finishFirst = deferRun();
      const finishQueued = deferRun((callbacks) => {
        callbacks.onAssistantMessage?.({
          toolCalls: [
            {
              id: ask.toolCallId,
              name: "ask_user",
              arguments: JSON.stringify({ kind: ask.kind, prompt: ask.prompt }),
            },
          ],
        });
      });
      mocks.runAgentLoop.mockResolvedValue(success);
      const store = useAgentStore.getState();
      const view = mountInput();
      const reference = {
        kind: "project" as const,
        name: "projects/orders",
        project: "projects/orders",
        label: "Orders",
      };
      const maskingPage: AgentMaskingPageResource = {
        kind: "masking-exemption",
        project: "projects/orders",
      };
      mocks.executeTool.mockResolvedValue({
        kind: "tool_result",
        result: JSON.stringify({ project: "projects/orders" }),
      });
      let queuedSettled = false;
      try {
        let sourceId: string | undefined;
        act(() => {
          sourceId = useAgentStore.getState().addMessage({
            chatId: view.chatId,
            role: "assistant",
            content: "I can also review masking exemptions.",
          })?.id;
        });
        expect(sourceId).toBeDefined();
        await submit(view, "List databases");
        await waitForRuns(1);
        mocks.currentPath = "/projects/orders/masking-exemption";
        mocks.currentPageResource = maskingPage;
        act(() =>
          useAgentStore.getState().setDraft(view.chatId, {
            text: "@Orders List masking exemptions",
            mentions: [{ resource: reference, range: { start: 0, end: 7 } }],
            followUpSourceId: sourceId,
          })
        );
        const send =
          view.container.querySelector<HTMLButtonElement>("[data-agent-send]");
        expect(send).toBeEnabled();
        await act(async () => send?.click());

        mocks.currentPath = "/projects/billing/masking-exemption";
        mocks.currentPageResource = {
          kind: "masking-exemption",
          project: "projects/billing",
        };
        await act(async () => finishFirst(success));
        await waitForRuns(2);
        expect(view.userMessages().at(-1)?.metadata?.followUpSourceId).toBe(
          sourceId
        );

        expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(
          expect.anything(),
          expect.objectContaining({
            context: "projects/orders",
            pagePath: "/projects/orders/masking-exemption",
            referencedProject: "projects/orders",
            maskingPage,
            runMode: "read-only",
          })
        );
        const [, executor] = mocks.createToolExecutor.mock
          .lastCall as unknown as Parameters<typeof createToolExecutor>;
        expect(executor?.canUseCurrentPage?.()).toBe(false);
        mocks.currentPath = "/projects/orders/masking-exemption";
        expect(executor?.canUseCurrentPage?.()).toBe(true);

        const toolDestination = "/projects/orders/plans";
        const options = mocks.runAgentLoop.mock.calls[1][5] as AgentLoopOptions;
        await act(async () => {
          mocks.currentPath = toolDestination;
          executor?.onNavigate?.();
          expect(await options.onCheckpoint?.({ phase: "working" })).toBe(true);
        });
        expect(executor?.canUseCurrentPage?.()).toBe(true);
        expect(store.getChat(view.chatId)).toMatchObject({
          page: { path: toolDestination },
          runCheckpoint: {
            pageToolPath: toolDestination,
            bindings: {
              page: { path: "/projects/orders/masking-exemption" },
              pageToolPath: "/projects/orders/masking-exemption",
            },
          },
        });

        await act(async () => {
          queuedSettled = true;
          finishQueued({ kind: "awaiting_user", ask, totalTokensUsed: 0 });
          await vi.waitFor(() => {
            expect(store.getChat(view.chatId)?.status).toBe("awaiting_user");
            expect(store.getAbortController(view.chatId)).toBeNull();
          });
        });
        await waitForLeaseRelease(view.chatId);
        await act(async () => {
          mocks.currentPath = "/projects/billing/issues";
          store.updateChatPage(view.chatId, {
            path: mocks.currentPath,
            title: "Billing issues",
          });
          expect(await store.flushRunPersistence()).toBe(true);
          store.loadState();
        });
        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
        const scope = within(view.container);
        const textarea = scope.getByRole<HTMLTextAreaElement>("textbox", {
          name: ask.prompt,
        });
        act(() => {
          setTextareaValue(textarea, "Review the active exemptions");
          textarea.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await act(async () =>
          scope.getByRole("button", { name: "agent.reply" }).click()
        );
        await waitForRuns(3);
        const [, resumedExecutor] = mocks.createToolExecutor.mock
          .lastCall as unknown as Parameters<typeof createToolExecutor>;
        expect(resumedExecutor?.canUseCurrentPage?.()).toBe(false);
        mocks.currentPath = toolDestination;
        expect(resumedExecutor?.canUseCurrentPage?.()).toBe(true);
        await waitForLeaseRelease(view.chatId);
      } finally {
        if (!queuedSettled && mocks.runAgentLoop.mock.calls.length > 1)
          await act(async () => finishQueued(success));
        view.unmount();
      }
    });

    test("queues composer submissions outside history and starts them in FIFO order", async () => {
      enableAsynchronousRunRecovery();
      const finishFirst = deferRun();
      const finishSecond = deferRun();
      mocks.runAgentLoop.mockResolvedValue(success);
      const store = useAgentStore.getState();
      const view = mountInput();
      let releaseFlush: (() => void) | undefined;
      try {
        await submit(view, "List databases");
        await waitForRuns(1);
        await submit(view, "List database owners");
        await submit(view, "What does SELECT @x mean?");
        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
        ]);
        expect(view.textarea).toHaveValue("");

        const firstController = store.getAbortController(view.chatId);
        const flushGate = new Promise<void>((resolve) => {
          releaseFlush = resolve;
        });
        const flush = vi
          .fn(store.flushRunPersistence)
          .mockImplementationOnce(async () => {
            await flushGate;
            return store.flushRunPersistence();
          });
        act(() => useAgentStore.setState({ flushRunPersistence: flush }));
        await act(async () => {
          finishFirst(success);
          await vi.waitFor(() => expect(flush).toHaveBeenCalled());
        });
        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(1);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
        ]);

        await act(async () => releaseFlush?.());
        await waitForRuns(2);
        expect(store.getAbortController(view.chatId)).not.toBe(firstController);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
          "List database owners",
        ]);

        await act(async () => finishSecond(success));
        await waitForRuns(3);
        await waitForLeaseRelease(view.chatId);
        expect(view.userMessages().map(({ content }) => content)).toEqual([
          "List databases",
          "List database owners",
          "What does SELECT @x mean?",
        ]);
        expect(
          new Set(mocks.runAgentLoop.mock.calls.map((call) => call[5].runId))
            .size
        ).toBe(3);
      } finally {
        await act(async () => releaseFlush?.());
        act(() =>
          useAgentStore.setState({
            flushRunPersistence: store.flushRunPersistence,
          })
        );
        view.unmount();
      }
    });
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
    expect(history.find((message: { role: string }) => message.role === "assistant")?.toolCalls).toMatchObject([
      { name: "get_masking_overview", origin: "runtime" },
    ]);
    expect(store.getMessages(chat.id).findLast(message => message.toolCalls?.some(call => call.id === refresh.toolCallId))?.toolCalls).toMatchObject([
      { name: "get_masking_overview", origin: "runtime" },
    ]);
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

  test.each([
    { name: "successful preparation", prepared: true, success: true, expected: true },
    { name: "failed preparation", prepared: false, success: true, expected: false },
    { name: "unsuccessful completion", prepared: true, success: false, expected: false },
    { name: "model-invented action", prepared: undefined, success: true, expected: false },
  ])("attaches only a runtime-prepared Access Grant draft after $name", async ({ prepared, success, expected }) => {
    const draft = {
      kind: "accessGrant" as const,
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT email FROM public.users WHERE id = 42;",
      schema: "public",
      unmask: true,
    };
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      if (prepared !== undefined) {
        callbacks.onAssistantMessage({ toolCalls: [{ id: "jit-prepare", name: "prepare_access_grant_draft", arguments: "{}" }] });
        await callbacks.onToolResult("jit-prepare", JSON.stringify(prepared ? { status: "unsaved", draft } : { error: "Permission denied" }));
      }
      callbacks.onText("Review the access request draft.");
      return {
        kind: "completed", success, totalTokensUsed: 0,
        actions: [{ kind: "draft-access-grant", draft }],
      };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Request unmasked email access.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the access request draft.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "Review the access request draft.");
    expect(answer?.metadata?.accessRequestDraft).toEqual(expected ? {
      ...draft,
      reason: "Request temporary query access to app with unmasked results.",
    } : undefined);
    expect(answer?.metadata?.actions).toBeUndefined();
    expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
    expect(mocks.executeTool).not.toHaveBeenCalledWith("call_api", expect.anything(), expect.anything());
    expect(mocks.routerPush).not.toHaveBeenCalled();
    view.unmount();
  });

  test.each(
    ["prepared", "prepared-after-denial"].flatMap((source) =>
      [false, true].flatMap((unmask) =>
        [
          { name: "omitted", reason: undefined },
          { name: "empty", reason: "" },
          { name: "whitespace", reason: " \n\t " },
          { name: "supplied", reason: "  Investigate the reported order issue  " },
        ].map((reason) => ({ source, unmask, ...reason }))
      )
    )
  )(
    "persists a nonblank reason for a $source Access Grant with $name reason and Unmask=$unmask",
    async ({ source, unmask, reason }) => {
      const draft = {
        kind: "accessGrant" as const,
        project: "projects/orders",
        sourceDatabase: "instances/prod/databases/app",
        targets: ["instances/prod/databases/app"],
        statement: "SELECT email FROM public.users WHERE id = 42;",
        schema: "public",
        unmask,
        ...(reason !== undefined ? { reason } : {}),
      };
      const queryArgs = {
        operationId: "SQLService/Query",
        body: { name: draft.sourceDatabase, statement: draft.statement },
      };
      const queryResult = JSON.stringify({
        status: 200,
        response: { results: [{
          error: "Permission denied",
          permissionDenied: { requiredPermissions: ["bb.sql.select"], resources: draft.targets },
        }] },
      });
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: queryResult });
      mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, executeTool, callbacks) => {
        callbacks.onAssistantMessage({ toolCalls: [source === "prepared"
          ? { id: "jit-reason", name: "prepare_access_grant_draft", arguments: "{}" }
          : { id: "jit-reason", name: "call_api", arguments: JSON.stringify(queryArgs) },
        ] });
        if (source === "prepared") {
          await callbacks.onToolResult("jit-reason", JSON.stringify({ status: "unsaved", draft }));
        } else {
          const execution = await executeTool("call_api", queryArgs, "jit-reason");
          if (execution.kind === "tool_result") {
            await callbacks.onToolResult("jit-reason", execution.result);
          }
          callbacks.onAssistantMessage({ toolCalls: [{ id: "jit-reason-prepared", name: "prepare_access_grant_draft", arguments: "{}" }] });
          await callbacks.onToolResult("jit-reason-prepared", JSON.stringify({ status: "unsaved", draft }));
        }
        callbacks.onText("Review the request with its default reason.");
        return { kind: "completed", success: true, totalTokensUsed: 0, actions: [] };
      });
      const store = useAgentStore.getState();
      const page = { path: "/projects/orders", title: "Orders" };
      const chat = store.createChat({ context: "projects/orders", page });
      store.queueUserMessage({ chatId: chat.id, text: "Request temporary query access.", page });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        await act(async () => {
          await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the request with its default reason.")).toBe(true));
        });
        const answer = store.getMessages(chat.id).find((message) => message.content === "Review the request with its default reason.");
        expect(answer?.metadata?.accessRequestDraft).toEqual({
          ...draft,
          reason: reason?.trim()
            ? reason
            : unmask
              ? "Request temporary query access to app with unmasked results."
              : "Request temporary query access to app.",
        });
        expect(mocks.routerPush).not.toHaveBeenCalled();
      } finally {
        view.unmount();
      }
    }
  );

  test("preserves a Request Role draft without adding an Access Grant reason", async () => {
    const draft = {
      kind: "requestRole" as const,
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      resources: ["instances/prod/databases/app/schemas/public/tables/users"],
      statement: "SELECT email FROM public.users;",
      requiredPermissions: ["bb.sql.select"],
    };
    const queryArgs = {
      operationId: "SQLService/Query",
      body: { name: draft.sourceDatabase, statement: "SELECT email FROM public.users;" },
    };
    const result = JSON.stringify({ status: 403, error: "Permission denied" });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result });
    mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, executeTool, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [{ id: "role-reason", name: "call_api", arguments: JSON.stringify(queryArgs) }] });
      const execution = await executeTool("call_api", queryArgs, "role-reason");
      if (execution.kind === "tool_result") {
        await callbacks.onToolResult("role-reason", execution.result);
      }
      callbacks.onText("Request the required role.");
      return { kind: "completed", success: false, totalTokensUsed: 0, actions: [] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Query email.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Request the required role.")).toBe(true));
      });
      const answer = store.getMessages(chat.id).find((message) => message.content === "Request the required role.");
      expect(answer?.metadata?.accessRequestDraft).toEqual(draft);
    } finally {
      view.unmount();
    }
  });

  test.each(
    ["prepared", "prepared-after-denial"].flatMap((source) =>
      ["open", "draft-issue-comment", "handoff"].map((primary) => ({ source, primary }))
    )
  )(
    "prioritizes a $source Access Grant over an evidence-backed $primary while preserving Continue",
    async ({ source, primary }) => {
      const issue = "projects/orders/issues/42";
      const target = { kind: "open-issue", issue };
      const followUp = { kind: "continue", prompt: "How long will this access last?" };
      const draft = {
        kind: "accessGrant" as const,
        project: "projects/orders",
        sourceDatabase: "instances/prod/databases/app",
        targets: ["instances/prod/databases/app"],
        statement: "SELECT email FROM users;",
        unmask: true,
      };
      const queryArgs = {
        operationId: "SQLService/Query",
        body: { name: draft.sourceDatabase, statement: draft.statement },
      };
      const queryResult = JSON.stringify({
        status: 200,
        response: { results: [{
          error: "Permission denied",
          permissionDenied: { requiredPermissions: ["bb.sql.select"], resources: draft.targets },
        }] },
      });
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: queryResult });
      mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, executeTool, callbacks) => {
        callbacks.onAssistantMessage({ toolCalls: [
          { id: "jit-related-issue", name: "get_issue_overview", arguments: JSON.stringify({ issue }) },
          source === "prepared"
            ? { id: "jit-priority", name: "prepare_access_grant_draft", arguments: "{}" }
            : { id: "jit-priority", name: "call_api", arguments: JSON.stringify(queryArgs) },
        ] });
        await callbacks.onToolResult("jit-related-issue", JSON.stringify({ project: "projects/orders", issue: { status: "available", name: issue } }));
        if (source === "prepared") {
          await callbacks.onToolResult("jit-priority", JSON.stringify({ status: "unsaved", draft }));
        } else {
          const execution = await executeTool("call_api", queryArgs, "jit-priority");
          if (execution.kind === "tool_result") {
            await callbacks.onToolResult("jit-priority", execution.result);
          }
          callbacks.onAssistantMessage({ toolCalls: [{ id: "jit-priority-prepared", name: "prepare_access_grant_draft", arguments: "{}" }] });
          await callbacks.onToolResult("jit-priority-prepared", JSON.stringify({ status: "unsaved", draft }));
        }
        callbacks.onText("Review the prepared access request first.", target);
        return {
          kind: "completed", success: true, totalTokensUsed: 0,
          actions: [
            ...(primary === "open" ? [{ kind: "open", target }] : primary === "draft-issue-comment" ? [{ kind: "draft-issue-comment", issue, comment: "Request temporary unmasked access." }] : []),
            followUp,
          ],
        };
      });
      const store = useAgentStore.getState();
      const page = { path: "/projects/orders", title: "Orders" };
      const chat = store.createChat({ context: "projects/orders", page });
      store.queueUserMessage({ chatId: chat.id, text: "Prepare an unmasked query request.", page });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        await act(async () => {
          await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the prepared access request first.")).toBe(true));
        });
        const answer = store.getMessages(chat.id).find((message) => message.content === "Review the prepared access request first.");
        expect(answer?.metadata?.accessRequestDraft).toEqual({
          ...draft,
          reason: "Request temporary query access to app with unmasked results.",
        });
        expect(answer?.metadata?.actions).toEqual([followUp]);
        expect(answer?.metadata?.handoff).toBeUndefined();
      } finally {
        view.unmount();
      }
    }
  );

  test.each([
    { tool: "prepare_access_grant_draft", successful: true, expected: "jit" },
    { tool: "prepare_access_grant_draft", successful: false, expected: undefined },
    { tool: "prepare_masking_rule_draft", successful: true, expected: "masking" },
    { tool: "prepare_masking_exemption_draft", successful: false, expected: undefined },
    { tool: "prepare_semantic_type_draft", successful: false, expected: undefined },
  ])("keeps only the latest Access Grant or masking handoff: $tool/$successful", async ({ tool, successful, expected }) => {
    const draft = {
      kind: "accessGrant" as const,
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT email FROM public.users;",
      unmask: true,
    };
    const newestDraft = { ...draft, statement: "SELECT email FROM public.users WHERE id = 42;" };
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [
        { id: "masking-old", name: "prepare_masking_rule_draft", arguments: "{}" },
        { id: "jit-old", name: "prepare_access_grant_draft", arguments: "{}" },
        { id: "latest", name: tool, arguments: "{}" },
      ] });
      await callbacks.onToolResult("masking-old", JSON.stringify({ draftId: "old-mask" }));
      await callbacks.onToolResult("jit-old", JSON.stringify({ status: "unsaved", draft }));
      await callbacks.onToolResult("latest", JSON.stringify(successful ? tool === "prepare_access_grant_draft" ? { status: "unsaved", draft: newestDraft } : { draftId: "new-mask" } : { error: "Permission denied" }));
      callbacks.onText("Review the latest request.");
      return { kind: "completed", success: true, totalTokensUsed: 0, actions: [] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Prepare unmasked access.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the latest request.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "Review the latest request.");
    expect(answer?.metadata?.accessRequestDraft).toEqual(expected === "jit" ? {
      ...newestDraft,
      reason: "Request temporary query access to app with unmasked results.",
    } : undefined);
    expect(answer?.metadata?.actions).toEqual(expected === "masking" ? [{ kind: "draft-masking-rule", draftId: "new-mask" }] : undefined);
    view.unmount();
  });

  test.each([
    { tool: "prepare_masking_rule_draft", kind: "draft-masking-rule" },
    { tool: "prepare_masking_exemption_draft", kind: "draft-masking-exemption" },
    { tool: "prepare_semantic_type_draft", kind: "draft-semantic-type" },
  ].flatMap((draft) => [false, true].map((resumed) => ({ ...draft, resumed }))))(
    "prioritizes a prepared $kind over an earlier denied Query with Resume=$resumed",
    async ({ tool, kind, resumed }) => {
      if (resumed) enableRunRecovery();
      const queryArgs = {
        operationId: "SQLService/Query",
        body: { name: "instances/prod/databases/app", statement: "SELECT email FROM users;" },
      };
      const queryResult = JSON.stringify({
        status: 200,
        response: { results: [{
          error: "Permission denied",
          permissionDenied: { requiredPermissions: ["bb.sql.select"], resources: [queryArgs.body.name] },
        }] },
      });
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: queryResult });
      mocks.resolveDeniedQueryAccessDraft.mockResolvedValue({
        project: "projects/orders",
        sourceDatabase: queryArgs.body.name,
        targets: [queryArgs.body.name],
        statement: queryArgs.body.statement,
      });
      let invocation = 0;
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, execute, callbacks, _signal, options) => {
        invocation++;
        if (resumed) {
          const reservation = await options.reserveProviderAttempt();
          expect(reservation.kind).toBe("allowed");
          await options.onAttemptSettled({
            attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
            outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: invocation === 1 ? 60_000 : 10 },
          });
        }
        if (invocation === 1) {
          callbacks.onAssistantMessage({ toolCalls: [
            { id: "denied-before-draft", name: "call_api", arguments: JSON.stringify(queryArgs) },
            { id: "prepared-after-denial", name: tool, arguments: "{}" },
          ] });
          const execution = await execute("call_api", queryArgs, "denied-before-draft");
          if (execution.kind === "tool_result") {
            await callbacks.onToolResult("denied-before-draft", execution.result);
          }
          await callbacks.onToolResult("prepared-after-denial", '{"draftId":"prepared-after-denial"}');
          if (resumed) {
            await options.onCheckpoint({ phase: "working" });
            return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
          }
        }
        callbacks.onText("Review the explicit prepared draft.");
        return { kind: "completed", success: true, actions: [], totalTokensUsed: 10 };
      });
      const store = useAgentStore.getState();
      const page = { path: "/projects/orders", title: "Orders" };
      const chat = store.createChat({ context: "projects/orders", page });
      store.queueUserMessage({ chatId: chat.id, text: "Query email, then prepare its masking policy.", page });
      const view = renderIntoContainer(<AgentInput />);
      view.render();
      try {
        if (resumed) {
          await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
          await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
          const resume = [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.retry-last-chat-turn");
          expect(resume).toBeDefined();
          await act(async () => { resume?.click(); });
        }
        await act(async () => {
          await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the explicit prepared draft.")).toBe(true));
        });
        const answer = store.getMessages(chat.id).find((message) => message.content === "Review the explicit prepared draft.");
        expect(answer?.metadata?.actions).toEqual([{ kind, draftId: "prepared-after-denial" }]);
        expect(answer?.metadata?.accessRequestDraft).toBeUndefined();
        expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
        expect(store.getMessages(chat.id).filter((message) => message.toolCallId === "prepared-after-denial")).toHaveLength(1);
      } finally {
        view.unmount();
      }
    }
  );

  test.each([false, true])("restores a prepared Access Grant across budget Resume unless a write invalidates it: %s", async (withWrite) => {
    enableRunRecovery();
    const draft = {
      kind: "accessGrant" as const,
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT email FROM public.users;",
      unmask: true,
    };
    const writeArgs = { operationId: "bytebase.v1.IssueService.UpdateIssue", body: { issue: { name: "projects/orders/issues/42", title: "Updated" } } };
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, execute, callbacks, _signal, options) => {
      const reservation = await options.reserveProviderAttempt();
      expect(reservation.kind).toBe("allowed");
      await options.onAttemptSettled({
        attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId,
        outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: reservation.ordinal === 1 ? 60_000 : 10 },
      });
      if (reservation.ordinal === 1) {
        callbacks.onAssistantMessage({ toolCalls: [
          { id: "prepared-jit", name: "prepare_access_grant_draft", arguments: "{}" },
          ...(withWrite ? [{ id: "invalidate-jit", name: "call_api", arguments: JSON.stringify(writeArgs) }] : []),
        ] });
        await callbacks.onToolResult("prepared-jit", JSON.stringify({ status: "unsaved", draft }));
        if (withWrite) {
          await execute("call_api", writeArgs, "invalidate-jit");
          await callbacks.onToolResult("invalidate-jit", '{"status":200,"response":{}}');
        }
        await options.onCheckpoint({ phase: "working" });
        return { kind: "paused", reason: "token-limit", phase: "working", totalTokensUsed: 60_000 };
      }
      callbacks.onText("Review the resumed access request.");
      return { kind: "completed", success: true, actions: [], totalTokensUsed: 10 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Prepare an unmasked query request.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("paused")); });
      const executionCount = mocks.executeTool.mock.calls.length;
      await act(async () => { expect(await increaseManagedRunTokenLimit(chat.id, 100_000)).toBe(true); });
      const resume = [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.retry-last-chat-turn");
      await act(async () => { resume?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
      const answer = store.getMessages(chat.id).find((message) => message.content === "Review the resumed access request.");
      expect(answer?.metadata?.accessRequestDraft).toEqual(withWrite ? undefined : {
        ...draft,
        reason: "Request temporary query access to app with unmasked results.",
      });
      expect(store.getMessages(chat.id).filter((message) => message.toolCallId === "prepared-jit")).toHaveLength(1);
      expect(mocks.executeTool).toHaveBeenCalledTimes(executionCount);
    } finally {
      view.unmount();
    }
  });

  test("does not attach a prepared Access Grant draft in a diagnostic chat", async () => {
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "Plan 7" };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: resource.project, plan: { status: "available", name: resource.name, state: "ACTIVE" } }) });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [{ id: "diagnostic-jit", name: "prepare_access_grant_draft", arguments: "{}" }] });
      await callbacks.onToolResult("diagnostic-jit", JSON.stringify({ status: "unsaved", draft: {
        kind: "accessGrant", project: resource.project, sourceDatabase: "instances/prod/databases/app", targets: ["instances/prod/databases/app"], statement: "SELECT email FROM users;", unmask: true,
      } }));
      callbacks.onText("This diagnostic remains read-only.");
      return { kind: "completed", success: true, totalTokensUsed: 0, actions: [] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders/plans/7", title: "Plan" };
    const chat = store.createChat({ context: resource.project, capability: { kind: "diagnostic", resource } });
    store.queueUserMessage({ chatId: chat.id, text: "Review this plan.", page, resourceReferences: [resource] });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "This diagnostic remains read-only.")).toBe(true));
    });
    const answer = store.getMessages(chat.id).find((message) => message.content === "This diagnostic remains read-only.");
    expect(answer?.metadata?.accessRequestDraft).toBeUndefined();
    expect(answer?.metadata?.actions).toBeUndefined();
    expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
    view.unmount();
  });

  test("regenerates the default reason when an unmasked preparation falls back to ordinary query access", async () => {
    const draft = {
      kind: "accessGrant" as const,
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT email FROM public.users WHERE id = 42;",
      schema: "public",
      unmask: false,
    };
    const preparedDraft = {
      ...draft,
      unmask: true,
      reason: "Request temporary query access to app with unmasked results.",
    };
    const args = {
      operationId: "SQLService/Query",
      body: { name: draft.sourceDatabase, statement: draft.statement },
    };
    const result = JSON.stringify({
      status: 200,
      response: { results: [{
        error: "Permission denied",
        permissionDenied: { requiredPermissions: ["bb.sql.select"], resources: draft.targets },
      }] },
    });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result });
    mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(draft);
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, executeTool, callbacks) => {
      callbacks.onAssistantMessage({ toolCalls: [{ id: "fallback-query", name: "call_api", arguments: JSON.stringify(args) }] });
      const execution = await executeTool("call_api", args, "fallback-query");
      if (execution.kind === "tool_result") {
        await callbacks.onToolResult("fallback-query", execution.result);
      }
      callbacks.onAssistantMessage({ toolCalls: [{ id: "unmasked-prepare", name: "prepare_access_grant_draft", arguments: "{}" }] });
      await callbacks.onToolResult("unmasked-prepare", JSON.stringify({ status: "unsaved", draft: preparedDraft }));
      callbacks.onText("Review the ordinary query access request.");
      return { kind: "completed", success: false, totalTokensUsed: 0, actions: [] };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Query email, then request unmasked access.", page });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    try {
      await act(async () => {
        await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "Review the ordinary query access request.")).toBe(true));
      });
      const answer = store.getMessages(chat.id).find((message) => message.content === "Review the ordinary query access request.");
      expect(answer?.metadata?.accessRequestDraft).toEqual({
        ...draft,
        reason: "Request temporary query access to app.",
      });
      expect(mocks.resolveDeniedQueryAccessDraft).toHaveBeenCalledWith(args, boundAgentQueryResult(result));
    } finally {
      view.unmount();
    }
  });

  test("supplies common paid-feature recovery guidance without replaying the API", async () => {
    const args = { operationId: "SettingService/UpdateSetting", body: { setting: { name: "settings/DATA_CLASSIFICATION" } } };
    const result = JSON.stringify({ status: 403, response: { code: "permission_denied", message: "feature FEATURE_DATA_CLASSIFICATION is a ENTERPRISE feature, please upgrade to access it" } });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result });
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, execute, callbacks, _signal, options) => {
      callbacks.onAssistantMessage({ toolCalls: [{ id: "paid-api", name: "call_api", arguments: JSON.stringify(args) }] });
      const execution = await execute("call_api", args, "paid-api");
      if (execution.kind !== "tool_result") throw new Error("Expected API result");
      await callbacks.onToolResult("paid-api", execution.result);
      expect(options.resolveRequestContext().systemGuidance).toContain("required plan");
      expect(options.resolveRequestContext().systemGuidance).not.toContain("Request temporary access");
      callbacks.onText("The backend requires Enterprise for data classification.");
      return { kind: "completed", success: false, actions: [], totalTokensUsed: 0 };
    });
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Orders" };
    const chat = store.createChat({ context: "projects/orders", page });
    store.queueUserMessage({ chatId: chat.id, text: "Update data classification", page });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      await act(async () => {
        await vi.waitFor(() => expect(store.getMessages(chat.id).some((message) => message.content === "The backend requires Enterprise for data classification.")).toBe(true));
      });
      expect(mocks.executeTool.mock.calls.filter(([name]) => name === "call_api")).toHaveLength(1);
      expect(store.getPendingAsk(chat.id)).toBeNull();
      expect(store.getMessages(chat.id).some((message) => message.metadata?.accessRequestDraft)).toBe(false);
    } finally {
      view.unmount();
    }
  });

  test.each([false, true].flatMap((managed) => [false, true].map((confirmed) => ({ managed, confirmed }))))(
    "waits for denied-query confirmation before the normal JIT skill, managed=$managed confirmed=$confirmed",
    async ({ managed, confirmed }) => {
      if (managed) enableRunRecovery();
      const { buildOutboundHistory } = await vi.importActual<typeof import("../logic/outboundHistory")>("../logic/outboundHistory");
      mocks.buildOutboundHistory.mockImplementation(buildOutboundHistory);
      const database = "instances/prod/databases/app";
      const statement = "SELECT id FROM orders WHERE id = 42;";
      const query = { operationId: "SQLService/Query", body: { name: database, statement } };
      const denied = JSON.stringify({ status: 200, response: { results: [{ error: "Permission denied", permissionDenied: { requiredPermissions: ["bb.sql.select"], resources: [database] } }] } });
      const ask: AgentPendingAsk = {
        kind: "confirm",
        toolCallId: "request-query-access",
        prompt: "The query failed because you lack bb.sql.select on app. You can request temporary access. Would you like to request it for this query?",
        confirmLabel: "Request temporary access",
        cancelLabel: "Not now",
      };
      const draft = { kind: "accessGrant", project: "projects/orders", sourceDatabase: database, targets: [database], statement, unmask: false };
      mocks.executeTool.mockImplementation(async (name) => {
        if (name === "call_api") return { kind: "tool_result", result: denied };
        if (name === "ask_user") return { kind: "ask_user", ask };
        if (name === "prepare_access_grant_draft") return { kind: "tool_result", result: JSON.stringify({ status: "unsaved", draft }) };
        return { kind: "tool_result", result: JSON.stringify({ justInTime: { eligible: true } }) };
      });
      let invocation = 0;
      mocks.runAgentLoop.mockImplementation(async (history: ReturnType<typeof buildOutboundHistory>, _tools, execute, callbacks, _signal, options) => {
        invocation++;
        if (managed) {
          const reservation = await options.reserveProviderAttempt();
          expect(reservation.kind).toBe("allowed");
          await options.onAttemptSettled({ attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 10 } });
        }
        const call = async (name: string, args: Record<string, unknown>, id: string) => {
          callbacks.onAssistantMessage({ toolCalls: [{ id, name, arguments: JSON.stringify(args) }] });
          const result = await execute(name, args, id);
          if (result.kind === "tool_result") await callbacks.onToolResult(id, result.result);
          return result;
        };
        if (invocation === 1) {
          await call("call_api", query, "denied-query");
          await call("get_just_in_time_overview", { database }, "jit-eligibility");
          const result = await call("ask_user", { kind: ask.kind, prompt: ask.prompt, confirmLabel: ask.confirmLabel, cancelLabel: ask.cancelLabel }, ask.toolCallId);
          if (result.kind !== "ask_user") throw new Error("Expected access confirmation");
          return { kind: "awaiting_user", ask: result.ask, totalTokensUsed: 10 };
        }
        expect(JSON.stringify(history)).toContain(statement);
        expect(JSON.stringify(history)).toContain(database);
        const response = history.findLast((message) => message.role === "tool" && message.toolCallId === ask.toolCallId);
        expect(JSON.parse(response?.content ?? "null")).toMatchObject({ kind: "confirm", confirmed });
        if (confirmed) {
          await call("get_skill", { name: "just-in-time" }, "jit-skill");
          await call("get_just_in_time_overview", { database }, "fresh-jit-eligibility");
          await call("prepare_access_grant_draft", { project: draft.project, database, statement, unmask: false }, "confirmed-jit-draft");
        }
        callbacks.onText(confirmed ? "Review the access request draft." : "No access request was prepared.");
        return { kind: "completed", success: confirmed, actions: [], totalTokensUsed: 10 };
      });
      const store = useAgentStore.getState();
      const page = { path: "/projects/orders", title: "Orders" };
      const chat = store.createChat({ context: "projects/orders", page });
      store.queueUserMessage({ chatId: chat.id, text: "Query the order with ID 42.", page });
      const view = renderIntoContainer(<AgentInput />);
      try {
        view.render();
        await act(async () => { await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("awaiting_user")); });
        expect(store.getPendingAsk(chat.id)).toMatchObject(ask);
        expect(view.container).toHaveTextContent(ask.prompt);
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        expect(mocks.executeTool.mock.calls.map(([name]) => name)).not.toContain("get_skill");
        expect(mocks.executeTool.mock.calls.map(([name]) => name)).not.toContain("prepare_access_grant_draft");
        expect(store.getMessages(chat.id).some((message) => message.metadata?.accessRequestDraft)).toBe(false);
        expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
        const label = confirmed ? ask.confirmLabel : ask.cancelLabel;
        const button = [...view.container.querySelectorAll("button")].find((item) => item.textContent === label);
        expect(button).toBeDefined();
        await act(async () => { button?.click(); await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle")); });
        expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2);
        expect(mocks.executeTool.mock.calls.filter(([name]) => name === "call_api")).toHaveLength(1);
        expect(mocks.executeTool.mock.calls.filter(([name]) => name === "prepare_access_grant_draft")).toHaveLength(confirmed ? 1 : 0);
        expect(store.getPendingAsk(chat.id)).toBeNull();
        const answer = store.getMessages(chat.id).find((message) => message.content === (confirmed ? "Review the access request draft." : "No access request was prepared."));
        expect(answer?.metadata?.accessRequestDraft).toEqual(confirmed ? { ...draft, reason: "Request temporary query access to app." } : undefined);
      } finally {
        view.unmount();
      }
    }
  );

  test.each([
    { name: "an unsuccessful completion", success: false, denied: true, prepared: false },
    { name: "a successful completion", success: true, denied: true, prepared: false },
    { name: "a successful Query", success: true, denied: false, prepared: false },
    { name: "an explicit preparation", success: true, denied: true, prepared: true },
    { name: "an explicit preparation after an unsuccessful query", success: false, denied: true, prepared: true },
  ])("resolves one actual denied Query independently of $name", async ({ success, denied, prepared }) => {
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
        results: denied ? [
          {
            error: "Permission denied",
            permissionDenied: {
              requiredPermissions: ["bb.sql.select"],
              resources: ["instances/prod/databases/app"],
            },
          },
        ] : [],
      },
    });
    const draft = {
      project: "projects/orders",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
    };
    const preparedDraft = {
      ...draft,
      kind: "accessGrant",
      unmask: false,
      reason: "Investigate the reported order issue",
    };
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result });
    mocks.resolveDeniedQueryAccessDraft.mockResolvedValue(denied ? draft : undefined);
    mocks.runAgentLoop.mockImplementation(
      async (_history, _tools, executeTool, callbacks, _signal, options) => {
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
          await callbacks.onToolResult("query-1", execution.result);
        }
        expect(options.resolveRequestContext().systemGuidance.includes("Request temporary access")).toBe(denied);
        if (prepared) {
          callbacks.onAssistantMessage({
            toolCalls: [{ id: "prepared-jit", name: "prepare_access_grant_draft", arguments: "{}" }],
          });
          await callbacks.onToolResult("prepared-jit", JSON.stringify({ status: "unsaved", draft: preparedDraft }));
        }
        callbacks.onText("The API denied the query.");
        return {
          kind: "completed",
          success,
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
    if (prepared && success) {
      expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
    } else {
      expect(mocks.resolveDeniedQueryAccessDraft).toHaveBeenCalledWith(
        args,
        boundAgentQueryResult(result)
      );
    }
    expect(
      useAgentStore
        .getState()
        .getMessages(chat.id)
        .find((message) => message.content === "The API denied the query.")
        ?.metadata?.accessRequestDraft
    ).toEqual(
      prepared && success
        ? preparedDraft
        : denied && prepared
          ? { ...draft, reason: preparedDraft.reason }
          : undefined
    );
    expect(useAgentStore.getState().getPendingAsk(chat.id)).toBeNull();
    view.unmount();
  });

  test.each([
    { masked: false, allowed: false, invalidation: "none" },
    { masked: false, allowed: true, invalidation: "none" },
    { masked: true, allowed: false, invalidation: "none" },
    { masked: false, allowed: true, invalidation: "run" },
    { masked: false, allowed: true, invalidation: "archive" },
  ])("grounds follow-ups in a resumed approved Query, masked=$masked allowed=$allowed invalidation=$invalidation", async ({ masked, allowed, invalidation }) => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) => key.startsWith("agent.follow-up.query-")
        ? enUS.agent["follow-up"][key.slice("agent.follow-up.".length) as keyof typeof enUS.agent["follow-up"]]
        : key,
      i18n: { language: "en-US" },
    });
    const countQuestion = "How many records match this query?";
    const summarizeQuestion = "Can you summarize these results?";
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    const database = "instances/prod/databases/app";
    let resolveEligibility: ((value: { database: string; allowed: boolean }) => void) | undefined;
    if (invalidation === "none") {
      mocks.resolveQueryMaskingEligibility.mockResolvedValue({ database, allowed });
    } else {
      mocks.resolveQueryMaskingEligibility.mockImplementation(() => new Promise((resolve) => {
        resolveEligibility = resolve;
      }));
    }
    const args = {
      operationId: "SQLService/Query",
      body: { name: database, statement: "SELECT private_value FROM records" },
    };
    const result = JSON.stringify({
      status: 200,
      response: {
        appliedAccessGrant: "projects/orders/accessGrants/123",
        results: [{
          columnNames: ["private_value"],
          rows: [{ values: [{ stringValue: masked ? "******" : "Visible" }] }],
          masked: [masked ? { algorithm: "full-mask" } : {}],
        }],
      },
    });
    store.addMessage({
      chatId,
      role: "user",
      content: "Query the records table",
      metadata: { resourceReferences: [{ kind: "database", name: database, project: "projects/orders", label: "app" }] },
    });
    store.addMessage({ chatId, role: "assistant", toolCalls: [{ id: "query-1", name: "call_api", arguments: JSON.stringify(args) }] });
    useAgentStore.setState({
      chats: store.chats.map((chat) => chat.id === chatId ? { ...chat, status: "awaiting_user" } : chat),
      pendingAskByChatId: {
        [chatId]: {
          toolCallId: "query-1", prompt: "SQLService/Query", kind: "confirm",
          apiPermission: { actionId: "action-1", operationId: "SQLService/Query" },
        },
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({ operationId: "SQLService/Query", body: JSON.stringify(args.body), expiresAt: Date.now() + 60_000 });
    mocks.resolvePendingApiCall.mockResolvedValue(result);
    mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executeTool, callbacks) => {
      callbacks.onText("The query returned a preview.");
      return {
        kind: "completed", success: true, totalTokensUsed: 0,
        actions: [
          { kind: "continue", prompt: countQuestion },
          { kind: "continue", prompt: "When does my masking exemption for private_value in app expire?" },
          { kind: "continue", prompt: summarizeQuestion },
        ],
      };
    });
    const view = renderIntoContainer(<AgentInput />);
    try {
      view.render();
      await act(async () => {
        const allow = [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.api-permission-allow");
        expect(allow).toBeDefined();
        allow?.click();
        if (invalidation === "none") {
          await vi.waitFor(() => expect(store.getMessages(chatId).some((message) => message.content === "The query returned a preview.")).toBe(true));
        } else {
          await vi.waitFor(() => expect(mocks.resolveQueryMaskingEligibility).toHaveBeenCalledOnce());
        }
      });
      if (invalidation !== "none") {
        await act(async () => {
          useAgentStore.setState((state) => ({
            chats: state.chats.map((chat) => chat.id !== chatId
              ? chat
              : invalidation === "run"
                ? { ...chat, runId: "replacement-run" }
                : { ...chat, archived: true }),
          }));
          resolveEligibility?.({ database, allowed });
          await vi.waitFor(() => expect(useAgentStore.getState().getAbortController(chatId)).toBeNull());
        });
        expect(store.getMessages(chatId).find((message) => message.content === "The query returned a preview.")).toBeUndefined();
        expect(store.getMessages(chatId).some((message) => message.metadata?.queryMaskingEligibility)).toBe(false);
        expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
        expect(mocks.executeTool).not.toHaveBeenCalled();
        return;
      }
      const answer = store.getMessages(chatId).find((message) => message.content === "The query returned a preview.");
      expect(answer?.metadata?.actions).toEqual([
        { kind: "continue", prompt: countQuestion },
        { kind: "continue", prompt: summarizeQuestion },
        ...(masked
          ? [{ kind: "continue", prompt: "How can I request access to unmasked data?" }]
          : allowed
            ? [{ kind: "continue", prompt: "How can I mask these columns?" }]
            : []),
      ]);
      const expectedEligibility = masked ? undefined : { database, allowed };
      expect(answer?.metadata?.queryMaskingEligibility).toEqual(expectedEligibility);
      if (masked) {
        expect(mocks.resolveQueryMaskingEligibility).not.toHaveBeenCalled();
      } else {
        expect(mocks.resolveQueryMaskingEligibility).toHaveBeenCalledExactlyOnceWith(
          "workspaces/demo",
          database
        );
      }
      expect(await store.flushRunPersistence()).toBe(true);
      const reloaded = createAgentStore();
      reloaded.getState().setWorkspaceName("workspaces/demo", "agent@example.com");
      expect(
        reloaded.getState().getMessages(chatId).find((message) => message.id === answer?.id)
          ?.metadata?.queryMaskingEligibility
      ).toEqual(expectedEligibility);
      expect(mocks.resolvePendingApiCall).toHaveBeenCalledOnce();
      expect(mocks.executeTool).not.toHaveBeenCalled();
    } finally {
      view.unmount();
    }
  });

  test.each([false, true])("does not auto-prepare access when an approved Query is denied and completion success is %s", async (success) => {
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
        success,
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
    ).toBeUndefined();
    expect(mocks.resolveDeniedQueryAccessDraft).toHaveBeenCalledWith(
      args,
      boundAgentQueryResult(result)
    );
    view.unmount();
  });

  test.each([false, true])("does not offer access when another Query was attempted in the turn and completion success is %s", async (success) => {
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
          success,
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
      expect(mocks.buildOutboundHistory.mock.calls.at(-1)?.[4]).toEqual({
        includeActionReceipts: false,
        ownerEmail: "agent@example.com",
      });
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

  test.each([
    { outcome: "a failed", failed: true },
    { outcome: "a successful", failed: false },
  ])(
    "sends explanation-only guidance only after $outcome diagnostic READ",
    async ({ failed }) => {
      const { runAgentLoop } = await vi.importActual<
        typeof import("../logic/agentLoop")
      >("../logic/agentLoop");
      const { buildSystemPrompt } = await vi.importActual<
        typeof import("../logic/prompt")
      >("../logic/prompt");
      const { buildOutboundHistory } = await vi.importActual<
        typeof import("../logic/outboundHistory")
      >("../logic/outboundHistory");
      const { getToolDefinitions } = await vi.importActual<
        typeof import("../logic/tools")
      >("../logic/tools");
      mocks.runAgentLoop.mockImplementation(runAgentLoop);
      mocks.buildSystemPrompt.mockImplementation(buildSystemPrompt);
      mocks.buildOutboundHistory.mockImplementation(buildOutboundHistory);
      mocks.getToolDefinitions.mockImplementation(getToolDefinitions);

      const resource = {
        kind: "plan" as const,
        name: "projects/orders/plans/7",
        project: "projects/orders",
        label: "Plan 7",
      };
      const plan = { status: "available", name: resource.name, state: "ACTIVE" };
      const args = {
        operationId: "PlanService/GetPlan",
        body: { name: resource.name },
      };
      const result = failed
        ? { status: 409, response: { code: "aborted" } }
        : { status: 200, response: plan };
      const answer = failed ? "The Plan read failed." : "The Plan is active.";
      mocks.executeTool.mockImplementation(async (name) => {
        if (name === "get_plan_overview")
          return {
            kind: "tool_result",
            result: JSON.stringify({ project: resource.project, plan }),
          };
        if (name === "call_api")
          return { kind: "tool_result", result: JSON.stringify(result) };
        if (name === "done")
          return { kind: "done", text: answer, success: !failed };
        throw new Error(`Unexpected tool: ${name}`);
      });
      mocks.aiChat
        .mockResolvedValueOnce({
          content: "",
          toolCalls: [
            {
              id: "diagnostic-read",
              name: "call_api",
              arguments: JSON.stringify(args),
            },
          ],
        })
        .mockResolvedValueOnce({
          content: "",
          toolCalls: [
            {
              id: "diagnostic-done",
              name: "done",
              arguments: JSON.stringify({
                text: answer,
                success: !failed,
                actions: [],
              }),
            },
          ],
          stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
        });
      mocks.currentPageResource = {
        kind: "masking-exemption",
        project: "projects/other",
      };
      const page = { path: "/projects/other/masking", title: "Other project" };
      const store = useAgentStore.getState();
      const capability = { kind: "diagnostic" as const, resource };
      const chat = store.createChat({
        context: resource.project,
        page,
        capability,
      });
      store.queueUserMessage({
        chatId: chat.id,
        text: "Explain this Plan.",
        page,
      });

      const view = renderIntoContainer(<AgentInput />);
      try {
        view.render();
        await act(async () => {
          await vi.waitFor(() => {
            expect(store.getMessages(chat.id)).toContainEqual(
              expect.objectContaining({ role: "assistant", content: answer })
            );
          });
        });
        expect(mocks.aiChat).toHaveBeenCalledTimes(2);
        const [first, followUp] = mocks.aiChat.mock.calls.map(
          ([request]) => request as AIChatRequest
        );
        const initialPrompt = first.messages[0].content;
        const prompt = followUp.messages[0].content;
        expect(initialPrompt).not.toContain("Explain the observed failure");
        if (failed) {
          expect(prompt).toContain("Explain the observed failure");
          expect(prompt).toContain("resource conflict");
          expect(prompt).toContain(
            "Recovery guidance is not authorization"
          );
        } else {
          expect(prompt).toBe(initialPrompt);
        }
        expect(prompt).toContain(`Diagnostic resource: plan ${resource.name}`);
        expect(prompt).not.toContain("projects/other");
        expect(prompt).not.toMatch(
          /# Data Masking|Request temporary access|prepare_access_grant_draft/
        );
        expect(followUp.toolDefinitions).toEqual(first.toolDefinitions);
        expect(
          followUp.toolDefinitions.find(({ name }) => name === "call_api")
            ?.description
        ).toContain(`Call authorized APIs in ${resource.project}`);
        expect(
          followUp.toolDefinitions.filter(({ name }) =>
            /^(navigate|dom_action|get_page_state|get_just_in_time_overview|prepare_)/.test(
              name
            )
          )
        ).toEqual([]);
        expect(
          mocks.executeTool.mock.calls.filter(([name]) => name === "call_api")
        ).toEqual([["call_api", args, "diagnostic-read"]]);
        const message = store
          .getMessages(chat.id)
          .find(
            (message) => message.role === "assistant" && message.content === answer
          );
        expect(message?.metadata?.actions).toBeUndefined();
        expect(message?.metadata?.accessRequestDraft).toBeUndefined();
        expect(mocks.resolveDeniedQueryAccessDraft).not.toHaveBeenCalled();
      } finally {
        view.unmount();
      }
    }
  );

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
    expect(mocks.buildOutboundHistory).toHaveBeenLastCalledWith("system", history, undefined, undefined, {
      includeActionReceipts: true,
      ownerEmail: "agent@example.com",
    });
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

  test("resends a follow-up with its original source independently of the composer provenance", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: '{"project":"projects/orders"}' });
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/orders" });
    const reference = { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" };
    store.addMessage({ chatId: chat.id, role: "user", content: "Query contacts" });
    const source = store.addMessage({ chatId: chat.id, role: "assistant", content: "Contacts returned." })!;
    const original = store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "@Orders How can I request access to unmasked data?",
      metadata: { resourceReferences: [reference], followUpSourceId: source.id },
    })!;
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Review the original query." });
    store.addMessage({ chatId: chat.id, role: "user", content: "Query employees" });
    const newerSource = store.addMessage({ chatId: chat.id, role: "assistant", content: "Employees returned." })!;
    store.setDraft(chat.id, {
      text: "@Billing Keep this unsent question.",
      mentions: [{
        resource: { kind: "project", name: "projects/billing", project: "projects/billing", label: "Billing" },
        range: { start: 0, end: 8 },
      }],
      followUpSourceId: newerSource.id,
    });
    const draft = structuredClone(store.getDraft(chat.id));
    const resendActionRef = createRef<((messageId: string) => void) | null>();
    const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
    view.render();

    try {
      await act(async () => {
        resendActionRef.current?.(original.id);
        await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
      });
      const resent = store.getMessages(chat.id).findLast((message) => message.role === "user");
      expect(resent?.id).not.toBe(original.id);
      expect(resent).toMatchObject({
        content: original.content,
        metadata: { resourceReference: reference, followUpSourceId: source.id },
      });
      expect(resent?.metadata?.followUpSourceId).not.toBe(newerSource.id);
      expect(store.getDraft(chat.id)).toEqual(draft);
    } finally {
      view.unmount();
    }
  });

  describe("Resend admission", () => {
    test.each(["idle", "error"] as const)("rechecks admission for an %s chat despite prior storage pressure", async (status) => {
      enableRunRecovery();
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, _callbacks, _signal, options) => {
        const reservation = await options.reserveProviderAttempt();
        expect(reservation.kind).toBe("allowed");
        await options.onAttemptSettled({ attemptId: reservation.attemptId, ordinal: reservation.ordinal, runId: options.runId, outcome: "response", usage: { purpose: "agent", toolNames: [], totalTokens: 0 } });
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      });
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const original = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      store.setChatStatus(chat.id, status, { lastError: status === "error" ? "Previous request failed" : null });
      store.setDraft(chat.id, { text: "Keep my next question" });
      await store.flushRunPersistence();
      const checkAgentStorageAdmission = store.checkAgentStorageAdmission;
      const admission = vi.fn(checkAgentStorageAdmission);
      useAgentStore.setState({ checkAgentStorageAdmission: admission, storagePressure: { reason: "quota", phase: "saving", bytes: 1024, limitBytes: AGENT_STORAGE_LIMIT_BYTES } });
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
      try {
        view.render();
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(admission).toHaveBeenCalled());
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        expect(useAgentStore.getState().storagePressure).toBeUndefined();
        expect(store.getMessages(chat.id).filter((message) => message.role === "user")).toHaveLength(2);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      } finally {
        view.unmount();
        useAgentStore.setState({ checkAgentStorageAdmission, storagePressure: undefined });
      }
    });

    test("reports a current storage admission failure on Resend and runs after storage recovers", async () => {
      enableRunRecovery();
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const original = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      store.setDraft(chat.id, { text: "Keep my next question" });
      await store.flushRunPersistence();
      localStorage.setItem("bb-agent-state-v3.foreign-scope", "x".repeat(AGENT_STORAGE_LIMIT_BYTES / 2));
      expect(await store.checkAgentStorageAdmission()).toBe(false);
      const beginChatRun = store.beginChatRun;
      const begin = vi.fn(beginChatRun);
      useAgentStore.setState({ beginChatRun: begin });
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderChatWithInput({ resendActionRef });
      try {
        view.render();
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(begin).toHaveBeenCalled());
        });
        expect(useAgentStore.getState().storagePressure?.reason).toBe("budget");
        expect(store.getChat(chat.id)?.status).toBe("error");
        expect(store.getChat(chat.id)?.lastError).toBeTruthy();
        expect(view.container.querySelector("[data-agent-storage-status]")).not.toBeNull();
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.aiChat).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
        localStorage.removeItem("bb-agent-state-v3.foreign-scope");
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        expect(useAgentStore.getState().storagePressure).toBeUndefined();
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      } finally {
        view.unmount();
        localStorage.removeItem("bb-agent-state-v3.foreign-scope");
        useAgentStore.setState({ beginChatRun, storagePressure: undefined });
        await store.flushRunPersistence();
      }
    });

    const prepareManagedResend = async (status: "error" | "paused") => {
      enableRunRecovery();
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const project = { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" };
      const original = store.addMessage({ chatId: chat.id, role: "user", content: "@Orders Inspect orders", metadata: { resourceReference: project } })!;
      const latest = store.addMessage({ chatId: chat.id, role: "user", content: "A later request" })!;
      const key = getAgentStateKey(store.workspaceName!, store.userEmail!)!;
      const result = await withAgentRunLease(key, chat.id, async () => {
        const runId = await store.beginChatRun(chat.id, latest.id, { page: { path: "/demo", title: "Demo" }, context: chat.context, references: [], runMode: "default" });
        expect(runId).toBeTruthy();
        await store.saveChatRunCheckpoint(chat.id, { ...store.getChat(chat.id)!.runCheckpoint!, startupComplete: true, suspension: status === "paused" ? "token-limit" : "interrupted" });
        store.setChatStatus(chat.id, status, { lastError: status === "error" ? "Previous request failed" : null });
        await store.flushRunPersistence();
        return runId;
      });
      expect(result.acquired).toBe(true);
      store.setDraft(chat.id, { text: "@Orders Keep my next question", mentions: [{ resource: project, range: { start: 0, end: 7 } }] });
      await store.flushRunPersistence();
      return { store, chat, original, project, runId: result.value! };
    };

    test.each(["error", "paused"] as const)("resends an older message from a managed %s run as a new request", async (status) => {
      const { store, chat, original, project, runId } = await prepareManagedResend(status);
      mocks.executeTool.mockResolvedValue({ kind: "tool_result", result: JSON.stringify({ project: project.name }) });
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      const endPausedRun = store.endPausedRun;
      const end = vi.fn(endPausedRun);
      const beginChatRun = store.beginChatRun;
      const begin = vi.fn(beginChatRun);
      useAgentStore.setState({ endPausedRun: end, beginChatRun: begin });
      const history = store.getMessages(chat.id);
      const draft = store.getDraft(chat.id);
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
      try {
        view.render();
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        expect(end).toHaveBeenCalledExactlyOnceWith(chat.id);
        expect(mocks.runAgentLoop.mock.calls[0][5].runId).not.toBe(runId);
        const messages = store.getMessages(chat.id);
        expect(messages.slice(0, history.length)).toEqual(history);
        expect(messages[history.length]).toMatchObject({ role: "user", content: original.content, metadata: { resourceReference: project } });
        expect(messages[history.length].id).not.toBe(original.id);
        expect(begin).toHaveBeenCalledOnce();
        expect(begin.mock.calls[0][1]).toBe(messages[history.length].id);
        expect(store.getDraft(chat.id)).toEqual(draft);
        expect(mocks.createToolExecutor).toHaveBeenLastCalledWith(expect.anything(), expect.objectContaining({ referencedProject: project.name }));
      } finally {
        view.unmount();
        useAgentStore.setState({ endPausedRun, beginChatRun });
      }
    });

    test("reports a failed durable run ending on Resend without starting model or tool work", async () => {
      const { store, chat, original } = await prepareManagedResend("paused");
      const history = store.getMessages(chat.id);
      const endPausedRun = store.endPausedRun;
      const end = vi.fn(endPausedRun);
      useAgentStore.setState({ endPausedRun: end });
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderChatWithInput({ resendActionRef });
      let write: ReturnType<typeof vi.spyOn> | undefined;
      try {
        view.render();
        write = vi.spyOn(localStorage, "setItem").mockImplementation(() => { throw new DOMException("Browser storage is full", "QuotaExceededError"); });
        act(() => useAgentStore.setState({ storagePressure: { reason: "quota", phase: "saving", bytes: 1024, limitBytes: AGENT_STORAGE_LIMIT_BYTES } }));
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(end).toHaveBeenCalledOnce());
        });
        expect(await end.mock.results[0].value).toBe(false);
        expect(useAgentStore.getState().storagePressure).toBeDefined();
        expect(store.getChat(chat.id)?.status).toBe("error");
        expect(store.getChat(chat.id)?.lastError).toContain("QuotaExceededError");
        expect(view.container.querySelector("[data-agent-storage-status]")).not.toBeNull();
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.aiChat).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
        expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(store.getDraft(chat.id).text).toBe("@Orders Keep my next question");
      } finally {
        view.unmount();
        write?.mockRestore();
        useAgentStore.setState({ endPausedRun, storagePressure: undefined });
        await store.flushRunPersistence();
      }
    });

    test("does not Resend while a local execution controller still owns an idle chat", async () => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const original = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      store.setAbortController(chat.id, new AbortController());
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
      try {
        view.render();
        await act(async () => resendActionRef.current?.(original.id));
        expect(store.getMessages(chat.id)).toHaveLength(1);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
      } finally {
        view.unmount();
        store.setAbortController(chat.id, null);
      }
    });

    test("does not Resend a run owned by another tab", async () => {
      const { store, chat, original } = await prepareManagedResend("error");
      enableAsynchronousRunRecovery(() => false);
      useAgentStore.setState({ remoteBusyChatIds: { [chat.id]: true } });
      const history = store.getMessages(chat.id);
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput resendActionRef={resendActionRef} />);
      try {
        view.render();
        await act(async () => resendActionRef.current?.(original.id));
        expect(store.getMessages(chat.id)).toEqual(history);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(mocks.executeTool).not.toHaveBeenCalled();
      } finally {
        view.unmount();
        useAgentStore.setState({ remoteBusyChatIds: {} });
      }
    });

    test.each([false, true])("keeps explicit Retry and Resend usable, prior unknown=%s", async (priorUnknown) => {
      enableRunRecovery();
      mocks.runAgentLoop.mockResolvedValue({ kind: "completed", success: true, totalTokensUsed: 0 });
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const original = store.addMessage({ chatId: chat.id, role: "user", content: "Inspect orders" })!;
      if (priorUnknown) {
        const { normalizeAgentApiAttempts } = await import("../logic/apiAttempt");
        useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = normalizeAgentApiAttempts([null], chat.id); });
      }
      store.setChatStatus(chat.id, "error", { lastError: "Previous request failed" });
      store.setDraft(chat.id, { text: "Keep my next question" });
      await store.flushRunPersistence();
      const retryActionRef = createRef<(() => void) | null>();
      const resendActionRef = createRef<((messageId: string) => void) | null>();
      const view = renderIntoContainer(<AgentInput retryActionRef={retryActionRef} resendActionRef={resendActionRef} />);
      try {
        view.render();
        await act(async () => retryActionRef.current?.());
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
        await act(async () => {
          resendActionRef.current?.(original.id);
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
          await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
        });
        expect(view.container.textContent).not.toContain("agent.token-budget.continuation-unavailable");
        expect(store.getMessages(chat.id)[0]).toMatchObject({ id: original.id, content: original.content });
        expect(store.getMessages(chat.id).filter((message) => message.role === "user")).toHaveLength(2);
        expect(store.getDraft(chat.id).text).toBe("Keep my next question");
      } finally {
        view.unmount();
      }
    });
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

  test.each([false, true])(
    "sends a queued follow-up independently of the composer draft, edited during configuration=%s",
    async (editDuringConfiguration) => {
      const { buildOutboundHistory } = await vi.importActual<typeof import("../logic/outboundHistory")>("../logic/outboundHistory");
      mocks.buildOutboundHistory.mockImplementation(buildOutboundHistory);
      let resolveConfiguration!: (setting: undefined) => void;
      mocks.getOrFetchSettingByName.mockReturnValue(new Promise<undefined>((resolve) => {
        resolveConfiguration = resolve;
      }));
      let finishRun!: () => void;
      mocks.runAgentLoop.mockImplementation((_history, _tools, _executor, callbacks: AgentCallbacks) =>
        new Promise((resolve) => {
          finishRun = () => {
            callbacks.onText?.("The queued follow-up completed.");
            resolve({ kind: "completed", success: true, totalTokensUsed: 0 });
          };
        })
      );
      const page = { path: "/projects/orders", title: "Orders" };
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders", page });
      const references = [
        { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" },
        { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" },
      ];
      mocks.executeTool.mockResolvedValue({
        kind: "tool_result",
        result: JSON.stringify({
          project: "projects/orders",
          database: { status: "available", name: references[1].name, state: "ACTIVE" },
        }),
      });
      const makeDraft = (databaseName: string, label: string, question: string) => ({
        text: `@Billing @${label} ${question}`,
        mentions: [
          {
            resource: { kind: "project" as const, name: "projects/billing", project: "projects/billing", label: "Billing" },
            range: { start: 0, end: 8 },
          },
          {
            resource: { kind: "database" as const, name: databaseName, project: "projects/billing", label },
            range: { start: 9, end: 10 + label.length },
          },
        ],
      });
      store.setDraft(chat.id, makeDraft("instances/test/databases/billing", "billing (test)", "Keep my unsent question."));
      let expectedDraft = structuredClone(store.getDraft(chat.id));
      const question = "Can you summarize these results?";
      store.queueUserMessage({ chatId: chat.id, text: question, page, resourceReferences: references });
      expect(store.getDraft(chat.id)).toEqual(expectedDraft);

      const view = renderIntoContainer(<AgentInput />);
      try {
        view.render();
        await act(async () => { await Promise.resolve(); });
        expect(resolveConfiguration).toBeTypeOf("function");
        expect(getQueuedUserMessage(chat.id)?.text).toBe(question);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
        expect(store.getDraft(chat.id)).toEqual(expectedDraft);

        if (editDuringConfiguration) {
          act(() => store.setDraft(chat.id, makeDraft("instances/test/databases/archive", "archive (test)", "Keep this newer question.")));
          expectedDraft = structuredClone(store.getDraft(chat.id));
        }
        expect(view.container.querySelector<HTMLTextAreaElement>("textarea")?.value).toBe(expectedDraft.text);

        await act(async () => {
          resolveConfiguration(undefined);
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        expect(store.getDraft(chat.id)).toEqual(expectedDraft);
        expect(view.container.querySelector<HTMLTextAreaElement>("textarea")?.value).toBe(expectedDraft.text);
        const users = store.getMessages(chat.id).filter((message) => message.role === "user");
        expect(users).toHaveLength(1);
        expect(users[0]).toMatchObject({ content: question, metadata: { resourceReferences: references } });
        const modelUsers = mocks.runAgentLoop.mock.calls[0][0].filter((message: { role: string }) => message.role === "user");
        expect(modelUsers).toHaveLength(1);
        expect(modelUsers[0].content).toMatch(/^Can you summarize these results\?\n\n/);
        for (const { kind, name, project } of references) {
          expect(modelUsers[0].content).toContain(JSON.stringify({ kind, name, project }));
        }
        expect(modelUsers[0].content).not.toContain("projects/billing");
        expect(modelUsers[0].content).not.toContain("instances/test/databases/");
        expect(modelUsers[0].content).not.toContain("Keep my unsent question.");
        expect(modelUsers[0].content).not.toContain("Keep this newer question.");
        expect(mocks.buildSystemPrompt).toHaveBeenCalledWith(expect.objectContaining({
          referencedProject: "projects/orders", referencedDatabase: references[1].name,
        }));
        expect(mocks.createToolExecutor).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
          referencedProject: "projects/orders", referencedDatabase: references[1].name,
        }));
        view.render();
        await act(async () => { await Promise.resolve(); });
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();

        await act(async () => {
          finishRun();
          await vi.waitFor(() => expect(store.getChat(chat.id)?.status).toBe("idle"));
        });
        expect(store.getMessages(chat.id).filter((message) => message.role === "user")).toEqual(users);
        expect(store.getMessages(chat.id).some((message) => message.content === "The queued follow-up completed.")).toBe(true);
        expect(store.getDraft(chat.id)).toEqual(expectedDraft);
        expect(view.container.querySelector<HTMLTextAreaElement>("textarea")?.value).toBe(expectedDraft.text);
        expect(getQueuedUserMessage(chat.id)).toBeUndefined();
        expect(mocks.runAgentLoop).toHaveBeenCalledOnce();
      } finally {
        view.unmount();
      }
    }
  );

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

  test.each(["unchanged", "text", "reference"] as const)(
    "preserves a blocked follow-up source through manual send only while its draft is unchanged: %s",
    async (edit) => {
      mocks.getOrFetchSettingByName.mockResolvedValue({
        value: { value: { case: "ai", value: { enabled: false } } },
      });
      mocks.runAgentLoop.mockImplementation(async (_history, _tools, _executor, callbacks) => {
        callbacks.onText("The restored follow-up completed.");
        return { kind: "completed", success: true, totalTokensUsed: 0 };
      });
      mocks.executeTool.mockImplementation(async (name, args) => ({
        kind: "tool_result",
        result: JSON.stringify({
          project: "projects/orders",
          ...(name === "get_database_overview" ? {
            database: { status: "available", name: args.databaseName, state: "ACTIVE" },
          } : {}),
        }),
      }));
      const page = { path: "/projects/orders", title: "Orders" };
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/orders", page });
      const references = [
        { kind: "project" as const, name: "projects/orders", project: "projects/orders", label: "Orders" },
        { kind: "database" as const, name: "instances/prod/databases/orders", project: "projects/orders", label: "orders (prod)" },
      ];
      store.addMessage({ chatId: chat.id, role: "user", content: "Query contacts" });
      const source = store.addMessage({ chatId: chat.id, role: "assistant", content: "Contacts returned." })!;
      const question = "@Orders @orders (prod) How can I request access to unmasked data?";
      store.queueUserMessage({
        chatId: chat.id,
        text: question,
        page,
        resourceReferences: references,
        followUpSourceId: source.id,
      });
      const blocked = renderIntoContainer(<AgentInput />);
      blocked.render();
      try {
        await act(async () => {
          await vi.waitFor(() => expect(getQueuedUserMessage(chat.id)).toBeUndefined());
        });
        expect(store.getDraft(chat.id)).toMatchObject({ text: question, followUpSourceId: source.id });
        expect(store.getDraft(chat.id).mentions?.map(({ resource }) => resource)).toEqual(references);
        expect(mocks.runAgentLoop).not.toHaveBeenCalled();
      } finally {
        blocked.unmount();
      }

      const restored = store.getDraft(chat.id);
      if (edit === "text") {
        store.setDraft(chat.id, { ...restored, text: `${restored.text} Explain the options.` });
      } else if (edit === "reference") {
        store.setDraft(chat.id, {
          ...restored,
          mentions: restored.mentions?.map((mention) => mention.resource.kind === "database"
            ? { ...mention, resource: { ...mention.resource, name: "instances/prod/databases/other" } }
            : mention),
        });
      }
      const submitted = structuredClone(store.getDraft(chat.id));
      expect(submitted.followUpSourceId).toBe(edit === "unchanged" ? source.id : undefined);
      mocks.getOrFetchSettingByName.mockResolvedValue(undefined);
      mocks.aiEnabled = true;
      const ready = renderIntoContainer(<AgentInput />);
      ready.render();
      try {
        await act(async () => {
          await vi.waitFor(() => expect(ready.container.querySelector<HTMLButtonElement>("[data-agent-send]")).not.toBeDisabled());
          ready.container.querySelector<HTMLButtonElement>("[data-agent-send]")?.click();
          await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
        });
        const sent = store.getMessages(chat.id).findLast((message) => message.role === "user");
        expect(sent?.content).toBe(submitted.text);
        expect(sent?.metadata?.followUpSourceId).toBe(edit === "unchanged" ? source.id : undefined);
        expect(sent?.metadata?.resourceReferences).toEqual(submitted.mentions?.map(({ resource }) => resource));
      } finally {
        ready.unmount();
      }
    }
  );

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
        actions: "manual-required",
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
        toolCalls: [{ name: "get_project_overview", origin: "runtime" }],
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
      "workspaces/demo",
      undefined,
      { includeActionReceipts: true, ownerEmail: "agent@example.com" }
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
      { role: "assistant", toolCalls: [{ name: "get_issue_overview", arguments: JSON.stringify({ issueName }), origin: "runtime" }] },
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
      { role: "assistant", toolCalls: [{ name: "get_plan_overview", arguments: JSON.stringify({ planName }), origin: "runtime" }] },
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
          origin: "runtime",
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
      expect(store.getPendingAsk(chatId)).toEqual(
        action === "api permission" ? null : pendingAsk
      );
      expect(store.getChat(chatId)?.status).toBe(
        action === "api permission" ? "idle" : "awaiting_user"
      );
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
    for (const label of ["agent.confirm", "agent.cancel"]) {
      const button = [...container.querySelectorAll("button")].find(
        (button) => button.textContent === label
      );
      expect(button).toHaveClass("h-auto", "min-h-7", "px-2", "text-xs");
      expect(button).not.toHaveClass("h-9", "h-7");
    }

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
      expect(button).toHaveClass("h-auto", "min-h-7", "text-xs");
      expect(button).not.toHaveClass("h-7");
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

  test.each([
    ["database-labels", "agent.permission.database-labels"],
    ["database-environment", "agent.permission.database-environment"],
  ] as const)("shows the runtime-owned %s diff in the existing API permission", (kind, title) => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.awaitUser(chatId, {
      toolCallId: "dom-write",
      prompt: "Untrusted model description",
      kind: "confirm",
      apiPermission: {
        actionId: "dom-permission",
        operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      },
    });
    const resourceName = "instances/example/databases/app";
    const longField = `label-${"x".repeat(160)}`;
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      body: '{"database":{"name":"instances/example/databases/app"},"updateMask":"labels"}',
      priorAttemptIds: ["uncertain-dom-attempt"],
      expiresAt: Date.now() + 60_000,
      dom: {
        kind,
        resourceName,
        changes: kind === "database-labels" ? [
          { field: longField, before: null, after: "<script>new value</script>" },
          { field: "removed-label", before: "old value", after: null },
          { field: "empty-label", before: "", after: "filled value" },
        ] : [{ field: "environment", before: "environments/dev", after: "environments/prod" }],
      },
    });
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const focusTarget = document.createElement("button");
    document.body.appendChild(focusTarget);
    focusTarget.focus();
    const view = renderIntoContainer(<AgentInput />);
    view.render();

    expect(screen.getByText(title)).toBeInTheDocument();
    expect(screen.getByText(resourceName)).toBeInTheDocument();
    const changes = screen.getByRole("list", { name: title });
    const rows = within(changes).getAllByRole("listitem");
    expect(rows).toHaveLength(kind === "database-labels" ? 3 : 1);
    if (kind === "database-labels") {
      expect(within(rows[0]).getByText(longField)).toBeInTheDocument();
      expect(within(rows[0]).getByText("agent.permission.not-set")).toBeInTheDocument();
      expect(within(rows[0]).getByText("<script>new value</script>")).toBeInTheDocument();
      expect(within(rows[1]).getByText("old value")).toBeInTheDocument();
      expect(within(rows[1]).getByText("agent.permission.not-set")).toBeInTheDocument();
      expect(within(rows[2]).getByText("common.empty")).toBeInTheDocument();
    } else {
      expect(within(rows[0]).getByText("common.environment")).toBeInTheDocument();
      expect(within(rows[0]).getByText("environments/dev")).toBeInTheDocument();
      expect(within(rows[0]).getByText("environments/prod")).toBeInTheDocument();
    }
    for (const row of rows) {
      expect(within(row).getByText("agent.permission.before")).toBeInTheDocument();
      expect(within(row).getByText("agent.permission.after")).toBeInTheDocument();
    }
    expect(view.container.querySelector("script")).toBeNull();
    expect(view.container.textContent).not.toContain("Untrusted model description");
    const details = view.container.querySelector("details");
    expect(details).not.toHaveAttribute("open");
    expect(details?.querySelector("summary")?.textContent).toBe("agent.permission.exact-request");
    expect(details?.textContent).toContain('"updateMask": "labels"');
    expect(screen.getAllByRole("button", { name: "agent.api-permission-allow" })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: "agent.api-permission-deny" })).toHaveLength(1);
    expect(view.container).toHaveTextContent("agent.api-recovery.repeat-risk");
    expect(screen.getByRole("timer")).toBeInTheDocument();
    expect(document.activeElement).toBe(focusTarget);
    expect(fetch).not.toHaveBeenCalled();
    expect(mocks.executeTool).not.toHaveBeenCalled();
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    view.unmount();
    focusTarget.remove();
  });

  test("removes a stale DOM preview instead of retaining its diff or offering approval", () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId!;
    store.awaitUser(chatId, {
      toolCallId: "dom-write",
      prompt: "Untrusted stale change",
      kind: "confirm",
      apiPermission: {
        actionId: "dom-permission",
        operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      },
    });
    mocks.getPendingApiCallPreview.mockReturnValue({
      operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
      body: "{}",
      expiresAt: Date.now() + 60_000,
      dom: {
        kind: "database-environment",
        resourceName: "instances/example/databases/app",
        changes: [{ field: "environment", before: "environments/dev", after: "environments/prod" }],
      },
    });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    expect(screen.getByText("environments/prod")).toBeInTheDocument();

    mocks.getPendingApiCallPreview.mockReturnValue(undefined);
    view.render(<AgentInput />);
    expect(screen.queryByText("environments/prod")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "agent.api-permission-allow" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "agent.api-permission-dismiss" })).toBeEnabled();
    expect(mocks.resolvePendingApiCall).not.toHaveBeenCalled();
    view.unmount();
  });

  test("resumes an approved diagnostic API call in the same bound chat", async () => {
    const resource = { kind: "plan" as const, name: "projects/orders/plans/7", project: "projects/orders", label: "projects/orders/plans/7" };
    const capability = { kind: "diagnostic" as const, resource };
    const store = useAgentStore.getState();
    mocks.currentUserEmail = "me@example.com";
    store.setWorkspaceName("workspaces/demo", "me@example.com");
    const chat = store.createChat({ context: resource.project, capability });
    store.addMessage({ chatId: chat.id, role: "user", content: "Update this project title." });
    store.startChatRun(chat.id, undefined, { runId: "diagnostic-run" });
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
    let scopeWasCurrent = false;
    mocks.resolvePendingApiCall.mockImplementationOnce(async (options) => {
      scopeWasCurrent = options.context?.isCurrent() ?? false;
      return '{"status":200,"response":{"name":"projects/orders"}}';
    });
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      [...view.container.querySelectorAll("button")].find((button) => button.textContent === "agent.api-permission-allow")?.click();
    });
    expect(mocks.resolvePendingApiCall).toHaveBeenCalledWith(expect.objectContaining({ approved: true, capability, context: expect.objectContaining({ chatId: chat.id, logicalActionId: "action-diagnostic" }) }));
    expect(scopeWasCurrent).toBe(true);
    expect(store.getMessages(chat.id)).toEqual(expect.arrayContaining([
      expect.objectContaining({ role: "tool", content: '{"status":200,"response":{"name":"projects/orders"}}' }),
    ]));
    expect(store.getChat(chat.id)?.capability).toEqual(capability);
    expect(store.getPendingAsk(chat.id)).toBeNull();
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
