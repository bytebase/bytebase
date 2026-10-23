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
  createToolExecutor: vi.fn(() => ({})),
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
  mocks.createToolExecutor.mockReturnValue({});
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

  test("changes the current chat context after a message", async () => {
    const original = useAgentStore
      .getState()
      .ensureCurrentChat(undefined, "projects/orders");
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

    expect(useAgentStore.getState().chats).toHaveLength(1);
    expect(useAgentStore.getState().getChat(original.id)?.context).toBe(
      "projects/billing"
    );
    expect(useAgentStore.getState().getMessages(original.id)).toHaveLength(1);
    expect(useAgentStore.getState().currentChatId).toBe(original.id);

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

  test("treats @ as ordinary text without loading page-element suggestions", async () => {
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
    expect(document.body.querySelector("[data-agent-mention-list]")).toBeNull();

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
