import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import type { DomRefSuggestion } from "../dom";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
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
  extractDatabaseOverviewEvidence: vi.fn((result: string) => {
    const payload = JSON.parse(result) as { evidence?: unknown };
    return payload.evidence;
  }),
  getOrFetchSettingByName: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
  searchAgentProjects: vi.fn(),
  projectsByName: {} as Record<string, { title: string }>,
  availableProjects: [] as { name: string; title: string }[],
  workspace: { name: "workspaces/demo", title: "Demo workspace" },
  currentContext: "projects/orders" as string | undefined,
  currentPath: "/demo",
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
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    getSettingByName: () => undefined,
    get projectsByName() {
      return mocks.projectsByName;
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

vi.mock("../logic/tools/databaseOverview", () => ({
  extractDatabaseOverviewEvidence: mocks.extractDatabaseOverviewEvidence,
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

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
  const initialState = createAgentStore().getState();
  useAgentStore.setState({
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    draftsByChatId: initialState.draftsByChatId,
    queuedUserMessage: null,
    queuedUserMessageInFlight: false,
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
  });

  mocks.useTranslation.mockReset();
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
  mocks.getRecentActivity.mockReset();
  mocks.getWorkspaceAttention.mockReset();
  mocks.getProjectAttention.mockReset();
  mocks.extractIssueOverviewEvidence.mockReset();
  mocks.extractIssueOverviewEvidence.mockImplementation((result: string) => {
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
  mocks.projectsByName = { "projects/orders": { title: "Order service" } };
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
  vi.unstubAllGlobals();
});

describe("AgentInput", () => {
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
    expect(useAgentStore.getState().queuedUserMessage).toBeNull();
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

    const view = renderIntoContainer(<AgentInput />);
    view.render();
    await act(async () => {
      await vi.waitFor(() =>
        expect(useAgentStore.getState().getChat(chat.id)?.interrupted).toBe(true)
      );
    });
    await act(async () => {
      [...view.container.querySelectorAll("button")]
        .find((button) => button.textContent === "agent.retry-last-chat-turn")
        ?.click();
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledTimes(2));
    });

    expect(mocks.getToolDefinitions).toHaveBeenLastCalledWith({
      runMode: "read-only",
    });
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
    expect(useAgentStore.getState().queuedUserMessage).not.toBeNull();
    expect(mocks.runAgentLoop).not.toHaveBeenCalled();

    await act(async () => { resolveSetting(undefined); });
    await act(async () => {
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(useAgentStore.getState().queuedUserMessage).toBeNull();
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
      await vi.waitFor(() => expect(useAgentStore.getState().queuedUserMessage).toBeNull());
    });
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe("My unsent question");
    expect(useAgentStore.getState().getMessages(chat.id)).toHaveLength(0);
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

    expect(useAgentStore.getState().chats).toHaveLength(1);
    expect(useAgentStore.getState().chats[0].context).toBe("workspaces/demo");
    expect(container.querySelector("[data-agent-context]")?.textContent).toBe(
      "Demo workspace"
    );
    unmount();
  });

  test("starts a separate chat when selecting another context after a message", async () => {
    useAgentStore.getState().setWorkspaceName("workspaces/demo");
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
    store.setWorkspaceName("workspaces/demo");
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

  test("keeps a cancelled @ literal without loading page-element suggestions", async () => {
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

    await act(async () => {
      textarea.dispatchEvent(
        new KeyboardEvent("keydown", { key: "Enter", bubbles: true })
      );
      await vi.waitFor(() => expect(mocks.runAgentLoop).toHaveBeenCalledOnce());
    });
    expect(
      useAgentStore
        .getState()
        .getMessages(useAgentStore.getState().currentChatId!)
    ).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ role: "user", content: "@" }),
      ])
    );

    unmount();
  });

  test("keeps a workspace chat and scopes only the mentioned turn to another project", async () => {
    mocks.runAgentLoop.mockResolvedValue({ kind: "completed", totalTokensUsed: 0 });
    const store = useAgentStore.getState();
    store.setWorkspaceName("workspaces/demo");
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

  test("stops a second reference until the first mention is removed", async () => {
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
      document.querySelector("[data-agent-resource-category]")
    ).toBeNull();
    expect(
      document.querySelector('[data-agent-resource-picker] [role="status"]')
    ).toHaveTextContent(
      "agent.resource-picker.one-reference-per-message"
    );
    expect(view.container.querySelector('[role="alert"]')).toBeNull();
    expect(store.getDraft(chat.id).resource?.name).toBe("projects/billing");
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
    expect(view.container.querySelector("[data-agent-send]")).toBeDisabled();

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
    expect(store.getDraft(chat.id).resource).toBeUndefined();
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

    expect(store.getDraft(chat.id).mention).toEqual({ start: 11, end: 19 });
    expect(store.getDraft(chat.id).resource?.name).toBe("projects/billing");
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

    unmount();
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
