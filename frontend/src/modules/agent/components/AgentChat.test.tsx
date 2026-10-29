import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
} from "@/app/router/handles";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  })),
  hasWorkspacePermissionV2: vi.fn(() => true),
  routerPush: vi.fn(),
  getSettingByName: vi.fn(),
}));

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

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: mocks.useTranslation,
}));

vi.mock("@tanstack/react-virtual", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-virtual")>()),
  useVirtualizer: (options: {
    count: number;
    estimateSize: (index: number) => number;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: index,
        start: index * options.estimateSize(index),
      })),
    getTotalSize: () => options.count * options.estimateSize(0),
    measureElement: () => undefined,
    scrollToIndex: vi.fn(),
  }),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: JSON.stringify(to) }),
  },
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/stores/app", () => {
  const state = { getSettingByName: mocks.getSettingByName };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
  const initialState = createAgentStore().getState();
  useAgentStore.setState({
    visible: initialState.visible,
    size: initialState.size,
    sidebarWidth: initialState.sidebarWidth,
    minimized: initialState.minimized,
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    draftsByChatId: initialState.draftsByChatId,
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
  });

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  });
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.routerPush.mockReset();
  mocks.getSettingByName.mockReset();
  mocks.getSettingByName.mockReturnValue(undefined);

  ({ AgentChat } = await import("./AgentChat"));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AgentChat", () => {
  test("renders a member-clicked handoff only on a final answer", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the current change queue.",
      metadata: {
        handoff: { kind: "review-database-changes", project: "projects/sample" },
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const link = container.querySelector<HTMLAnchorElement>("[data-agent-handoff]");
    expect(link?.textContent).toContain("agent.handoff.review-database-changes");
    expect(link?.getAttribute("href")).toContain(PROJECT_V1_ROUTE_ISSUES);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test("uses the same setup action label as Project Home", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Transfer an eligible database.",
      metadata: {
        handoff: { kind: "transfer-project-databases", project: "projects/sample" },
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelector("[data-agent-handoff]")?.textContent).toContain(
      "project.home.transfer-databases"
    );
    unmount();
  });

  test("does not render a handoff from pending, errored, or malformed messages", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const handoff = { kind: "review-project-issues" as const, project: "projects/sample" };
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Loading", metadata: { pending: true, handoff } });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Failed", metadata: { error: "failed", handoff } });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Old", metadata: { handoff: { kind: "review-project-issues", project: "projects/-" } } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelectorAll("[data-agent-handoff]")).toHaveLength(0);
    unmount();
  });
  test("shows project landing prompts in an empty project chat and prefills without sending", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const onSelectSuggestion = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat
        contextLabel="Sample project"
        onSelectSuggestion={onSelectSuggestion}
      />
    );
    render();

    expect(container.querySelector("[data-agent-empty-chat]")?.textContent).toContain(
      "Sample project"
    );
    expect(
      Array.from(container.querySelectorAll("[data-agent-empty-suggestion]"))
        .map((item) => item.getAttribute("data-starter-id"))
    ).toEqual([
      "project-open-blockers",
      "project-open-recent",
      "project-propose-change",
      "project-next",
    ]);

    act(() => {
      (container.querySelector('[data-starter-id="project-open-blockers"]') as HTMLButtonElement).click();
    });
    expect(onSelectSuggestion).toHaveBeenCalledWith("project-open-blockers");
    expect(store.getMessages(chat.id)).toEqual([]);
    unmount();
  });

  test("uses workspace landing prompts and hides the empty state after a message", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/example" });
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat contextLabel="Example workspace" onSelectSuggestion={vi.fn()} />
    );
    render();
    expect(
      Array.from(container.querySelectorAll("[data-agent-empty-suggestion]"))
        .map((item) => item.getAttribute("data-starter-id"))
    ).toEqual([
      "workspace-attention",
      "workspace-recent",
      "workspace-governance",
    ]);

    act(() => {
      store.addMessage({ chatId: chat.id, role: "user", content: "Hello" });
    });
    expect(container.querySelector("[data-agent-empty-chat]")).toBeNull();
    unmount();
  });

  test("does not offer prompts that would replace an unsent draft", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/example" });
    store.setDraft(chat.id, { text: "My unfinished question" });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-empty-chat]")).not.toBeNull();
    expect(container.querySelector("[data-agent-empty-suggestion]")).toBeNull();
    unmount();
  });

  test("does not show starter prompts while a new chat is running", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.setChatStatus(chat.id, "running");
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelector("[data-agent-empty-chat]")).toBeNull();
    unmount();
  });

  test("starts a new empty chat at the top so its context and heading stay visible", () => {
    const store = useAgentStore.getState();
    store.addMessage({ role: "user", content: "Previous question" });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const chatPanel = container.firstElementChild as HTMLDivElement;
    Object.defineProperty(chatPanel, "scrollHeight", { value: 500 });
    chatPanel.scrollTop = 250;

    act(() => {
      store.createChat({ context: "projects/sample" });
    });

    expect(chatPanel.scrollTop).toBe(0);
    expect(container.querySelector("[data-agent-empty-chat]")).not.toBeNull();
    unmount();
  });

  test("shows a canceled clarification without the running-request warning", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Reject the issue",
    });
    store.startChatRun(chat.id, undefined, { runId: "run-1" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      toolCalls: [
        { id: "ask-1", name: "ask_user", arguments: '{"prompt":"Why?"}' },
      ],
      metadata: { runId: "run-1" },
    });
    store.awaitUser(chat.id, {
      toolCallId: "ask-1",
      prompt: "Why?",
      kind: "input",
    });
    store.cancelPendingAsk(chat.id);

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      expect(container.textContent).toContain("agent.tool-request-canceled");
      expect(
        container.querySelector("[data-agent-interruption-status]")
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("renders a user-stopped result after the interrupted turn's tool calls without actions", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Count databases",
    });
    store.startChatRun(chat.id, undefined, { runId: "run-1" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      toolCalls: [{ id: "tool-1", name: "get_page_state", arguments: "{}" }],
      metadata: { runId: "run-1" },
    });
    store.addMessage({
      chatId: chat.id,
      role: "tool",
      toolCallId: "tool-1",
      content: "{}",
      metadata: { runId: "run-1" },
    });
    store.interruptChatRun(chat.id, undefined, "user");

    const onRetry = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onRetry={onRetry} />
    );
    render();
    try {
      const result = container.querySelector(
        "[data-agent-interruption-status]"
      );
      const toolRow = [...container.querySelectorAll(".font-mono")].find(
        (element) => element.textContent === "get_page_state"
      );
      expect(result).not.toBeNull();
      expect(result?.textContent).toContain("agent.request-stopped-by-user");
      expect(result?.textContent).toContain("agent.request-stopped-retry-hint");
      expect(toolRow).toBeDefined();
      expect(
        toolRow &&
          result &&
          (toolRow.compareDocumentPosition(result) &
            Node.DOCUMENT_POSITION_FOLLOWING)
      ).toBeTruthy();
      expect(result?.textContent).not.toContain("agent.retry-last-chat-turn");
      expect(result?.textContent).not.toContain("common.dismiss");
      expect(onRetry).not.toHaveBeenCalled();
    } finally {
      unmount();
    }
  });

  test("offers Retry for the current unexpected interruption", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Count databases",
    });
    store.startChatRun(chat.id, undefined, { runId: "run-1" });
    store.interruptChatRun(chat.id);

    const onRetry = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onRetry={onRetry} />
    );
    render();
    try {
      const result = container.querySelector(
        "[data-agent-interruption-status]"
      );
      expect(result).not.toBeNull();
      expect(result?.textContent).toContain("agent.interrupted");
      expect(result?.textContent).toContain("agent.interrupted-retry-hint");
      const retry = [...(result?.querySelectorAll("button") ?? [])].find(
        (button) => button.textContent === "agent.retry-last-chat-turn"
      );
      expect(retry).toBeDefined();
      act(() => retry?.click());
      expect(onRetry).toHaveBeenCalledOnce();
      expect(result?.textContent).not.toContain("common.dismiss");
    } finally {
      unmount();
    }
  });

  test("scrolls to the interruption result when no message was added", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Count databases",
    });
    store.startChatRun(chat.id, undefined, { runId: "run-1" });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const transcript = container.firstElementChild as HTMLDivElement;
      Object.defineProperty(transcript, "scrollHeight", { value: 500 });
      transcript.scrollTop = 0;

      act(() => store.interruptChatRun(chat.id, undefined, "user"));

      expect(
        container.querySelector("[data-agent-interruption-status]")
      ).not.toBeNull();
      expect(transcript.scrollTop).toBe(500);
    } finally {
      unmount();
    }
  });

  test("keeps an old interruption result in the transcript without a stale Retry action", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "First request",
    });
    store.startChatRun(chat.id, undefined, { runId: "run-1" });
    store.interruptChatRun(chat.id);
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Second request",
    });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Second answer",
    });

    const onRetry = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onRetry={onRetry} />
    );
    render();
    try {
      const result = container.querySelector(
        "[data-agent-interruption-status]"
      );
      expect(result).not.toBeNull();
      expect(result?.textContent).toContain("agent.interrupted");
      expect(result?.querySelector("button")).toBeNull();
      expect(container.textContent?.indexOf("First request")).toBeLessThan(
        container.textContent?.indexOf("agent.interrupted") ?? -1
      );
      expect(container.textContent?.indexOf("agent.interrupted")).toBeLessThan(
        container.textContent?.indexOf("Second request") ?? -1
      );
      expect(onRetry).not.toHaveBeenCalled();
    } finally {
      unmount();
    }
  });

  test("shows the selected Project separately from the plain user question", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "Compare its policies",
      metadata: {
        resourceReference: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-resource-reference]")?.textContent).toBe(
      "common.project: Billing"
    );
    expect(container.textContent).toContain("Compare its policies");
    expect(
      useAgentStore
        .getState()
        .getMessages(useAgentStore.getState().currentChatId)[0].content
    ).toBe(
      "Compare its policies"
    );

    unmount();
  });

  test("highlights an inline resource mention without repeating the project label", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "Compare @Billing policies",
      metadata: {
        resourceReference: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-message-mention]")?.textContent).toBe(
      "@Billing"
    );
    expect(container.querySelector("[data-agent-resource-reference]")).toBeNull();
    expect(container.textContent).toContain("Compare @Billing policies");
    unmount();
  });

  test("highlights a bound Project and Database together", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "How many tables in @Billing @orders?",
      metadata: {
        resourceReferences: [
          {
            kind: "project",
            name: "projects/billing",
            project: "projects/billing",
            label: "Billing",
          },
          {
            kind: "database",
            name: "instances/prod/databases/orders",
            project: "projects/billing",
            label: "orders",
          },
        ],
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(
      Array.from(container.querySelectorAll("[data-agent-message-mention]")).map(
        (element) => element.textContent
      )
    ).toEqual(["@Billing", "@orders"]);
    expect(container.querySelector("[data-agent-resource-reference]")).toBeNull();
    expect(container.textContent).toContain("How many tables in @Billing @orders?");
    unmount();
  });

  test("does not guess which repeated mention has a single binding", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "Compare @orders with @orders in @Billing",
      metadata: {
        resourceReferences: [
          {
            kind: "project",
            name: "projects/billing",
            project: "projects/billing",
            label: "Billing",
          },
          {
            kind: "database",
            name: "instances/prod/databases/orders",
            project: "projects/billing",
            label: "orders",
          },
        ],
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(
      Array.from(container.querySelectorAll("[data-agent-message-mention]")).map(
        (element) => element.textContent
      )
    ).toEqual(["@Billing"]);
    unmount();
  });

  test("highlights both bindings when the Project and Database share a label", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "Compare @orders and @orders",
      metadata: {
        resourceReferences: [
          {
            kind: "project",
            name: "projects/orders",
            project: "projects/orders",
            label: "orders",
          },
          {
            kind: "database",
            name: "instances/prod/databases/orders",
            project: "projects/orders",
            label: "orders",
          },
        ],
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(
      Array.from(container.querySelectorAll("[data-agent-message-mention]")).map(
        (element) => element.textContent
      )
    ).toEqual(["@orders", "@orders"]);
    unmount();
  });

  test("does not highlight an arbitrary copy when the mention text occurs twice", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "Compare @Billing with @Billing",
      metadata: {
        resourceReference: {
          kind: "project",
          name: "projects/billing",
          project: "projects/billing",
          label: "Billing",
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-message-mention]")).toBeNull();
    expect(container.querySelector("[data-agent-resource-reference]")).toBeNull();
    expect(container.textContent).toContain("Compare @Billing with @Billing");
    unmount();
  });

  test("shows AI setup when the cached AI setting is disabled", () => {
    mocks.getSettingByName.mockImplementation((name) => {
      if (name !== Setting_SettingName.AI) return undefined;
      return {
        value: {
          value: {
            case: "ai",
            value: { enabled: false },
          },
        },
      };
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    expect(container.textContent).toContain("agent.ai-not-configured.title");

    const dismissButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.dismiss"
    );
    act(() => {
      dismissButton?.click();
    });
    expect(container.textContent).not.toContain(
      "agent.ai-not-configured.title"
    );

    unmount();
  });

  test("keeps assistant links within the chat bubble", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content:
        "Configure billing at https://platform.openai.com/settings/organization/billing/.",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    const link = container.querySelector("a");
    expect(link).toBeInstanceOf(HTMLElement);
    expect(link?.className).toContain("break-all");

    unmount();
  });

  test("shows member-clicked evidence links below an evidence-backed answer", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "The project is ready for its next governed change.",
      metadata: {
        evidence: {
          project: "projects/orders",
          references: [
            {
              kind: "issue",
              name: "projects/orders/issues/42",
              title: "Add customer table",
            },
            {
              kind: "database",
              name: "projects/orders/instances/prod/databases/orders",
            },
          ],
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const evidenceTrigger = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "agent.evidence.references"
    );
    expect(evidenceTrigger).toBeInstanceOf(HTMLElement);

    act(() => {
      evidenceTrigger?.click();
    });

    const issueLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "Add customer table"
    );
    const databaseLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "orders"
    );
    expect(issueLink).toBeInstanceOf(HTMLElement);
    expect(databaseLink).toBeInstanceOf(HTMLElement);

    act(() => {
      issueLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: "orders", issueId: "42" },
    });

    act(() => {
      databaseLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "orders",
        instanceId: "prod",
        databaseName: "orders",
      },
      query: { parent: "projects/orders/instances/prod" },
    });

    unmount();
  });

  test("preserves the workspace-instance parent in a database evidence link", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "The database is ready for its next governed change.",
      metadata: {
        evidence: {
          project: "projects/orders",
          references: [
            {
              kind: "database",
              name: "instances/prod/databases/orders",
            },
          ],
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const evidenceTrigger = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "agent.evidence.references"
    );
    act(() => {
      evidenceTrigger?.click();
    });

    const databaseLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "orders"
    );
    expect(databaseLink).toBeInstanceOf(HTMLElement);

    act(() => {
      databaseLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "orders",
        instanceId: "prod",
        databaseName: "orders",
      },
      query: { parent: "instances/prod" },
    });

    unmount();
  });

  test("omits evidence references outside the evidence project", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "The project is ready for its next governed change.",
      metadata: {
        evidence: {
          project: "projects/orders",
          references: [
            {
              kind: "issue",
              name: "projects/payroll/issues/42",
              title: "Update payroll schema",
            },
            {
              kind: "database",
              name: "projects/orders/instances/prod/databases/orders",
            },
          ],
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const evidenceTrigger = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "agent.evidence.references"
    );
    act(() => {
      evidenceTrigger?.click();
    });

    expect(container.textContent).toContain("orders");
    expect(container.textContent).not.toContain("Update payroll schema");

    unmount();
  });

  test("links workspace issue evidence to its project", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "One issue needs your review.",
      metadata: {
        evidence: {
          workspace: "workspaces/default",
          references: [
            {
              kind: "issue",
              name: "projects/orders/issues/42",
              title: "Add customer table",
            },
            {
              kind: "database",
              name: "projects/orders/instances/prod/databases/orders",
            },
          ],
        },
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const evidenceTrigger = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "agent.evidence.references"
    );
    act(() => {
      evidenceTrigger?.click();
    });

    const issueLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "Add customer table"
    );
    expect(issueLink).toBeInstanceOf(HTMLElement);
    expect(container.textContent).not.toContain("database: orders");

    act(() => {
      issueLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: { projectId: "orders", issueId: "42" },
    });

    unmount();
  });

  test("renders deduplicated references from separately scoped overviews", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "The workspace and project both have relevant activity.",
      metadata: {
        evidence: [
          {
            workspace: "workspaces/default",
            references: [
              {
                kind: "issue",
                name: "projects/billing/issues/7",
                title: "Billing approval",
              },
              {
                kind: "database",
                name: "projects/orders/instances/prod/databases/ignored",
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
              {
                kind: "issue",
                name: "projects/billing/issues/7",
                title: "Out-of-scope issue",
              },
            ],
          },
        ],
      },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const evidenceTrigger = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "agent.evidence.references"
    );
    act(() => {
      evidenceTrigger?.click();
    });

    expect(container.textContent).toContain("Billing approval");
    expect(container.textContent).toContain("database: orders");
    expect(container.textContent).not.toContain("Out-of-scope issue");
    expect(container.textContent).not.toContain("database: ignored");
    expect(
      Array.from(container.querySelectorAll("a")).filter(
        (link) => link.textContent === "Billing approval"
      )
    ).toHaveLength(1);
    unmount();
  });

  test("sizes user and assistant message bubbles to their content", () => {
    useAgentStore.getState().addMessage({
      role: "user",
      content: "List databases",
    });
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "No databases found.",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const bubbles = Array.from(container.querySelectorAll("div")).filter(
      (element) =>
        element.className.includes("bg-info/10") ||
        element.className.includes("bg-control-bg px-3 py-2")
    );
    expect(bubbles).toHaveLength(2);
    for (const bubble of bubbles) {
      const classes = bubble.className.split(/\s+/);
      expect(classes).toContain("w-fit");
      expect(classes).toContain("max-w-full");
      expect(classes).not.toContain("w-full");
      expect(classes).not.toContain("max-w-[80%]");
    }

    unmount();
  });

  test("shows timestamps for user messages and final assistant replies", () => {
    const userMessage = useAgentStore.getState().addMessage({
      role: "user",
      content: "List databases",
    });
    useAgentStore.getState().addMessage({
      role: "assistant",
      toolCalls: [
        {
          id: "call_list_databases",
          name: "list_databases",
          arguments: "{}",
        },
      ],
    });
    const assistantMessage = useAgentStore.getState().addMessage({
      role: "assistant",
      content: "You can find databases from the Database section.",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const timestamps = Array.from(
      container.querySelectorAll("[data-agent-message-timestamp]")
    ) as HTMLTimeElement[];
    expect(timestamps).toHaveLength(2);
    expect(timestamps.map((timestamp) => timestamp.dateTime)).toEqual([
      new Date(userMessage.createdTs).toISOString(),
      new Date(assistantMessage.createdTs).toISOString(),
    ]);
    expect(timestamps[1]?.parentElement?.className).not.toContain("gap-y-2");
    expect(timestamps[1]?.previousElementSibling?.className).toContain(
      "gap-y-2"
    );
    expect(container.querySelector("[data-agent-message-day]")).toBeNull();

    unmount();
  });

  test("shows yesterday and the local time for a message from yesterday", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-22T04:03:00Z"));
    useAgentStore.getState().addMessage({
      role: "user",
      content: "How do I connect a database?",
    });
    vi.setSystemTime(new Date("2026-10-23T04:03:00Z"));

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const timestamp = container.querySelector("[data-agent-message-timestamp]");
    expect(timestamp?.textContent).toBe("Yesterday 12:03 PM");

    unmount();
  });

  test("shows today and the local time for a recent message", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-10-23T03:03:00Z"));
    useAgentStore.getState().addMessage({
      role: "user",
      content: "How do I connect a database?",
    });
    vi.setSystemTime(new Date("2026-10-23T04:03:00Z"));

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const timestamp = container.querySelector("[data-agent-message-timestamp]");
    expect(timestamp?.textContent).toBe("Today 11:03 AM");

    unmount();
  });

  test("renders a pending starter as one compact status instead of progress cards", () => {
    const chat = useAgentStore.getState().ensureCurrentChat();
    useAgentStore.getState().setChatStatus(chat.id, "running");
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "Checking recent activity...",
      metadata: { pending: true },
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-pending-message]")?.textContent).toContain(
      "Checking recent activity..."
    );
    expect(container.querySelector("[data-agent-message-timestamp]")).toBeNull();
    expect(container.textContent).not.toContain("common.loading");

    unmount();
  });

  test("renders inline code with wrap-safe styling", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      content:
        "Result: `354b7196c9ba5fb4b21cf615bb6ec4cd5c3c0c26c8f296b0f42b0f8a1d4e9abc`",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    const code = container.querySelector("code");
    expect(code).toBeInstanceOf(HTMLElement);
    expect(code?.textContent).toContain("354b7196");
    expect(code?.className).toContain("break-all");

    unmount();
  });

  test("renders duplicate tool call ids without React key warnings", () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    useAgentStore.getState().addMessage({
      role: "assistant",
      toolCalls: [
        {
          id: "call_search_api",
          name: "search_api",
          arguments: JSON.stringify({ service: "SQLService" }),
        },
        {
          id: "call_search_api",
          name: "search_api",
          arguments: JSON.stringify({ operationId: "SQLService/Query" }),
        },
      ],
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    expect(container.textContent).toContain("search_api");
    expect(container.querySelectorAll(".font-mono")).toHaveLength(2);
    expect(consoleError).not.toHaveBeenCalledWith(
      expect.stringContaining("Encountered two children with the same key")
    );

    consoleError.mockRestore();
    unmount();
  });

  test("matches duplicate tool call ids to results by occurrence order", () => {
    const assistantMessage = useAgentStore.getState().addMessage({
      role: "assistant",
      toolCalls: [
        {
          id: "call_search_api",
          name: "search_api",
          arguments: JSON.stringify({ service: "SQLService" }),
        },
        {
          id: "call_search_api",
          name: "search_api",
          arguments: JSON.stringify({ operationId: "SQLService/Query" }),
        },
      ],
    });

    useAgentStore.getState().addMessage({
      role: "tool",
      toolCallId: "call_search_api",
      content: "first-result",
    });
    useAgentStore.getState().addMessage({
      role: "tool",
      toolCallId: "call_search_api",
      content: "second-result",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    const cardHeaders = container.querySelectorAll(".cursor-pointer");
    expect(cardHeaders).toHaveLength(2);

    act(() => {
      for (const header of cardHeaders) {
        header.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      }
    });

    expect(assistantMessage.toolCalls).toHaveLength(2);
    expect(container.textContent).toContain("first-result");
    expect(container.textContent).toContain("second-result");

    unmount();
  });

  test("shows the SQL query result for its call without exposing other call results", () => {
    useAgentStore.getState().addMessage({
      role: "assistant",
      toolCalls: [
        {
          id: "list-1",
          name: "call_api",
          arguments: JSON.stringify({
            operationId: "DatabaseService/ListDatabases",
          }),
        },
        {
          id: "query-1",
          name: "call_api",
          arguments: JSON.stringify({ operationId: "SQLService/Query" }),
        },
      ],
    });
    useAgentStore.getState().addMessage({
      role: "tool",
      toolCallId: "list-1",
      content: JSON.stringify({ status: 200, response: { databases: [] } }),
    });
    useAgentStore.getState().addMessage({
      role: "tool",
      toolCallId: "query-1",
      content: JSON.stringify({
        status: 200,
        response: {
          results: [
            {
              columnNames: ["value"],
              rows: [{ values: [{ stringValue: "query-cell" }] }],
              rowsCount: "1",
            },
          ],
        },
      }),
    });
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: "The query returned one row.",
    });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const grids = container.querySelectorAll(
      "[data-agent-query-result] [data-query-result-table-grid]"
    );
    expect(grids).toHaveLength(1);
    expect(grids[0]?.textContent).toContain("query-cell");
    expect(container.textContent).toContain("The query returned one row.");

    unmount();
  });

  test("routes AI configuration recovery to general settings with intro", () => {
    const chat = useAgentStore.getState().ensureCurrentChat();
    useAgentStore.setState((state) => ({
      visible: true,
      chats: state.chats.map((item) =>
        item.id === chat.id ? { ...item, requiresAIConfiguration: true } : item
      ),
    }));

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    const configureLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "agent.ai-not-configured.configure"
    ) as HTMLAnchorElement;

    expect(configureLink).toBeTruthy();

    act(() => {
      configureLink.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "setting.workspace.general",
      hash: "#ai-assistant",
      query: { intro: "ai-assistant" },
    });
    expect(useAgentStore.getState().visible).toBe(false);

    unmount();
  });
});
