import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_ISSUES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  SETTING_ROUTE_WORKSPACE_GENERAL,
} from "@/app/router/handles";
import type { RequestRoleDraft } from "../logic/deniedQueryAccess";
import { registerAgentExpressionTarget } from "../logic/expressionDraft";
import { acceptAgentResponseActions } from "../logic/responseActions";
import type { AgentHandoff, AgentMessageMetadata } from "../logic/types";
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
  routeListeners: [] as Array<() => void>,
  currentPath: "/initial",
  resolvedPath: undefined as string | undefined,
  currentRoute: undefined as
    | { name?: string; params?: Record<string, string> }
    | undefined,
  aiEnabled: undefined as boolean | undefined,
  getSettingByName: vi.fn(),
  notify: vi.fn(),
  writeTextToClipboard: vi.fn(),
  prepareAgentPlanDraftRoute: vi.fn(),
  applyPlanDraftToCurrent: vi.fn(),
  prepareAgentDeniedQueryRequestRoute: vi.fn(),
  prepareIssueCommentDraftRoute: vi.fn(),
  wasDeniedQueryRequestDraftDelivered: vi.fn(),
  wasIssueCommentDraftDelivered: vi.fn(),
  getMaskingRuleDraft: vi.fn(),
  getSemanticTypeDraft: vi.fn(),
  getMaskingRuleDraftSource: vi.fn(),
  revalidateMaskingRuleDraft: vi.fn(),
  consumeMaskingRuleDraft: vi.fn(),
  getMaskingExemptionDraft: vi.fn(),
  getMaskingExemptionDraftSource: vi.fn(),
  revalidateMaskingExemptionDraft: vi.fn(),
  consumeMaskingExemptionDraft: vi.fn(),
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

vi.mock("@/app/router", async (importOriginal) => {
  let routeSource: typeof mocks.currentRoute;
  let routePath: string | undefined;
  let snapshot:
    | (NonNullable<typeof mocks.currentRoute> & { fullPath: string })
    | undefined;
  const getRouteSnapshot = () => {
    if (
      !snapshot ||
      routeSource !== mocks.currentRoute ||
      routePath !== mocks.currentPath
    ) {
      routeSource = mocks.currentRoute;
      routePath = mocks.currentPath;
      snapshot = {
        ...(mocks.currentRoute ?? {}),
        fullPath: mocks.currentPath,
      };
    }
    return snapshot;
  };
  return {
    ...(await importOriginal<typeof import("@/app/router")>()),
    router: {
      push: mocks.routerPush,
      afterEach: (listener: () => void) => {
        mocks.routeListeners.push(listener);
        return () => {
          mocks.routeListeners = mocks.routeListeners.filter(
            (registered) => registered !== listener
          );
        };
      },
      resolve: (to: unknown) => {
        const fullPath =
          mocks.resolvedPath ?? (typeof to === "object" &&
          to !== null &&
          "fullPath" in to &&
          typeof to.fullPath === "string"
            ? to.fullPath
            : JSON.stringify(to));
        return { href: fullPath, fullPath };
      },
      get currentRoute() {
        return { value: getRouteSnapshot() };
      },
    },
  };
});

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/stores/app", () => {
  const state = {
    getSettingByName: mocks.getSettingByName,
    currentUser: { name: "users/member@example.com" },
    workspaceResourceName: () => "workspaces/demo",
    get serverInfo() {
      return { aiEnabled: mocks.aiEnabled };
    },
    notify: mocks.notify,
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("../logic/semanticTypeDraft", () => ({
  getSemanticTypeDraft: mocks.getSemanticTypeDraft,
  getSemanticTypeDraftSource: () => undefined,
  getSafeSemanticTypeDraftSummary: (draft: unknown) => draft,
  revalidateSemanticTypeDraft: vi.fn(),
  consumeSemanticTypeDraft: vi.fn(),
}));
vi.mock("../logic/maskingRuleDraft", () => ({
  getMaskingRuleDraft: mocks.getMaskingRuleDraft,
  getMaskingRuleDrafts: (...args: unknown[]) => {
    const draft = mocks.getMaskingRuleDraft(...args);
    return draft ? [draft] : undefined;
  },
  getMaskingRuleDraftSource: mocks.getMaskingRuleDraftSource,
  revalidateMaskingRuleDraft: mocks.revalidateMaskingRuleDraft,
  consumeMaskingRuleDraft: mocks.consumeMaskingRuleDraft,
}));
vi.mock("../logic/maskingExemptionDraft", () => ({
  getMaskingExemptionDraft: mocks.getMaskingExemptionDraft,
  getMaskingExemptionDraftSource: mocks.getMaskingExemptionDraftSource,
  revalidateMaskingExemptionDraft: mocks.revalidateMaskingExemptionDraft,
  consumeMaskingExemptionDraft: mocks.consumeMaskingExemptionDraft,
}));

vi.mock("../logic/planDraft", () => ({
  prepareAgentPlanDraftRoute: mocks.prepareAgentPlanDraftRoute,
  PlanDraftStorageError: class PlanDraftStorageError extends Error {},
}));

vi.mock("@/lib/plan/agentDraftHandoff", () => ({
  dispatchAgentPlanDraftToCurrentPlan: mocks.applyPlanDraftToCurrent,
}));

vi.mock("../logic/deniedQueryAccess", () => ({
  prepareAgentDeniedQueryRequestRoute: mocks.prepareAgentDeniedQueryRequestRoute,
  DeniedQueryRequestDraftStorageError: class DeniedQueryRequestDraftStorageError extends Error {},
}));

vi.mock("@/modules/sql-editor/model/accessGrantDraft", () => ({
  ACCESS_GRANT_DRAFT_QUERY_KEY: "accessGrantDraftKey",
  wasDeniedQueryRequestDraftDelivered:
    mocks.wasDeniedQueryRequestDraftDelivered,
}));

vi.mock("@/lib/issueCommentDraft", () => ({
  prepareIssueCommentDraftRoute: mocks.prepareIssueCommentDraftRoute,
  wasIssueCommentDraftDelivered: mocks.wasIssueCommentDraftDelivered,
  ExistingIssueCommentDraftError: class ExistingIssueCommentDraftError extends Error {},
  IssueCommentDraftStorageError: class IssueCommentDraftStorageError extends Error {},
}));

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

const expectCompactPrimaryAction = (
  action: Element | null,
  label: string,
  icon = "arrow-up-right"
) => {
  expect(action).toHaveClass("h-7", "text-xs", "text-accent", "max-w-full");
  expect(action).not.toHaveClass("border", "h-auto", "whitespace-normal");
  expect(action).toHaveAttribute("title", label);
  expect(action?.querySelector(".truncate")?.textContent).toBe(label);
  expect(action?.lastElementChild).toHaveClass(`lucide-${icon}`);
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
    workspaceName: "workspaces/demo",
    userEmail: "member@example.com",
    abortControllersByChatId: {},
    queuedUserMessagesByChatId: {},
    queuedUserMessageInFlightByChatId: {},
  });

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  });
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.routerPush.mockReset();
  mocks.routeListeners = [];
  mocks.currentRoute = undefined;
  mocks.currentPath = "/initial";
  mocks.resolvedPath = undefined;
  mocks.aiEnabled = undefined;
  mocks.getSettingByName.mockReset();
  mocks.getSettingByName.mockReturnValue(undefined);
  mocks.writeTextToClipboard.mockReset();
  mocks.writeTextToClipboard.mockResolvedValue(true);
  mocks.notify.mockReset();
  mocks.prepareAgentPlanDraftRoute.mockReset();
  mocks.applyPlanDraftToCurrent.mockReset();
  mocks.prepareAgentDeniedQueryRequestRoute.mockReset();
  mocks.prepareIssueCommentDraftRoute.mockReset();
  mocks.wasDeniedQueryRequestDraftDelivered.mockReset();
  mocks.wasDeniedQueryRequestDraftDelivered.mockReturnValue(true);
  mocks.wasIssueCommentDraftDelivered.mockReset();
  mocks.wasIssueCommentDraftDelivered.mockReturnValue(true);
  mocks.getMaskingRuleDraft.mockReset();
  mocks.getMaskingRuleDraftSource.mockReset();
  mocks.revalidateMaskingRuleDraft.mockReset();
  mocks.consumeMaskingRuleDraft.mockReset();
  mocks.getMaskingExemptionDraft.mockReset();
  mocks.getMaskingExemptionDraftSource.mockReset();
  mocks.revalidateMaskingExemptionDraft.mockReset();
  mocks.consumeMaskingExemptionDraft.mockReset();

  ({ AgentChat } = await import("./AgentChat"));
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AgentChat", () => {
  test("copies each user message's original text without changing the draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const content = "@Orders select * from orders;\nKeep the original text.";
    store.addMessage({
      chatId: chat.id,
      role: "user",
      content,
      metadata: {
        resourceReference: { kind: "project", name: "projects/orders", project: "projects/orders", label: "Orders" },
      },
    });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "An answer" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Another request" });
    store.setDraft(chat.id, { text: "Unsent draft" });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const buttons = container.querySelectorAll<HTMLButtonElement>('button[aria-label="common.copy"]');
      expect(buttons).toHaveLength(2);
      await act(async () => buttons[0].click());
      await act(async () => buttons[1].click());
      expect(mocks.writeTextToClipboard.mock.calls).toEqual([[content], ["Another request"]]);
      expect(buttons[0].querySelector(".lucide-check")).not.toBeNull();
      expect(store.getDraft(chat.id).text).toBe("Unsent draft");
      expect(store.getMessages(chat.id)).toHaveLength(3);
    } finally {
      unmount();
    }
  });

  test("places Copy and Resend before the rightmost timestamp in a vertically centered row", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Original request" });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "An answer" });
    const { container, render, unmount } = renderIntoContainer(<AgentChat onResend={vi.fn()} />);
    render();
    try {
      const footer = container.querySelector("[data-agent-message-actions]");
      const copy = footer?.querySelector('button[aria-label="common.copy"]');
      const resend = footer?.querySelector('button[aria-label="agent.resend-message"]');
      const timestamp = footer?.querySelector("[data-agent-message-timestamp]");
      expect(copy).not.toBeNull();
      expect(resend).not.toBeNull();
      expect(footer).toHaveClass("mt-1", "flex", "items-center", "justify-end", "gap-0");
      expect(footer?.lastElementChild).toBe(timestamp);
      expect(copy?.compareDocumentPosition(resend!)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
      expect(copy).toHaveClass("h-6", "px-1.5");
      expect(resend).toHaveClass("h-6", "px-1.5");
      expect(timestamp).not.toHaveClass("mt-1");
      const assistantTimestamp = container.querySelectorAll("[data-agent-message-timestamp]")[1];
      expect(assistantTimestamp).toHaveClass("mt-1");
    } finally {
      unmount();
    }
  });

  test.each(["running", "awaiting_user", "archived", "queued"] as const)(
    "allows copying a user message while the chat is %s",
    async (state) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Original request" });
      if (state === "running") store.startChatRun(chat.id);
      if (state === "awaiting_user") store.awaitUser(chat.id, { toolCallId: "question", kind: "input", prompt: "Which database?" });
      if (state === "archived") store.archiveChat(chat.id);
      if (state === "queued") store.queueUserMessage({ chatId: chat.id, text: "Queued request", page: { path: "/demo", title: "Demo" } });
      const { container, render, unmount } = renderIntoContainer(<AgentChat onResend={vi.fn()} />);
      render();
      try {
        const copy = container.querySelector<HTMLButtonElement>('button[aria-label="common.copy"]');
        expect(copy).not.toBeNull();
        expect(copy).not.toBeDisabled();
        await act(async () => copy?.click());
        expect(mocks.writeTextToClipboard).toHaveBeenCalledWith("Original request");
      } finally {
        unmount();
      }
    }
  );

  test("resends each user message by its original ID without changing the draft", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const first = store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "First request",
    });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "First answer" });
    const second = store.addMessage({
      chatId: chat.id,
      role: "user",
      content: "Second request",
    });
    store.setDraft(chat.id, { text: "Unsent next question" });
    const onResend = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onResend={onResend} />
    );
    render();
    try {
      const resendButtons = container.querySelectorAll<HTMLButtonElement>(
        'button[aria-label="agent.resend-message"]'
      );
      expect(resendButtons).toHaveLength(2);
      expect(resendButtons[0].querySelector(".lucide-repeat-2")).not.toBeNull();
      act(() => resendButtons[0].click());
      act(() => resendButtons[1].click());
      expect(onResend.mock.calls).toEqual([[first?.id], [second?.id]]);
      expect(store.getDraft(chat.id)).toEqual({ text: "Unsent next question" });
      expect(store.getMessages(chat.id)).toHaveLength(3);
    } finally {
      unmount();
    }
  });

  test.each(["running", "awaiting_user", "queued"] as const)(
    "disables user-message resend while the chat is %s",
    (state) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      store.addMessage({ chatId: chat.id, role: "user", content: "Original request" });
      if (state === "running") store.startChatRun(chat.id);
      if (state === "awaiting_user") {
        store.awaitUser(chat.id, { toolCallId: "question", kind: "input", prompt: "Which database?" });
      }
      if (state === "queued") {
        store.queueUserMessage({ chatId: chat.id, text: "Queued request", page: { path: "/demo", title: "Demo" } });
      }
      const onResend = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onResend={onResend} />
      );
      render();
      try {
        const resend = container.querySelector<HTMLButtonElement>(
          'button[aria-label="agent.resend-message"]'
        );
        expect(resend).not.toBeNull();
        expect(resend).toBeDisabled();
        act(() => resend?.click());
        expect(onResend).not.toHaveBeenCalled();
      } finally {
        unmount();
      }
    }
  );

  test("hides resend and follow-up actions in archived transcripts while keeping Copy", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Original request" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "An answer",
      metadata: { actions: [{ kind: "continue", prompt: "What is the impact?" }] },
    });
    store.setDraft(chat.id, { text: "Unsent draft" });
    store.archiveChat(chat.id);
    const onResend = vi.fn();
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onResend={onResend} onContinueAction={onContinueAction} />
    );
    render();
    try {
      expect(container.textContent).toContain("Original request");
      expect(container.textContent).toContain("An answer");
      expect(
        container.querySelector('[aria-label="agent.resend-message"]')
      ).toBeNull();
      expect(container.querySelector("[data-agent-follow-up]")).toBeNull();
      const copy = container.querySelector<HTMLButtonElement>(
        '[aria-label="common.copy"]'
      );
      expect(copy).not.toBeNull();
      expect(copy).not.toBeDisabled();
      await act(async () => copy?.click());
      expect(mocks.writeTextToClipboard).toHaveBeenCalledWith("Original request");
      expect(store.getDraft(chat.id).text).toBe("Unsent draft");
      expect(onResend).not.toHaveBeenCalled();
      expect(onContinueAction).not.toHaveBeenCalled();

      act(() => store.unarchiveChat(chat.id));
      expect(
        container.querySelector('[aria-label="agent.resend-message"]')
      ).not.toBeNull();
      expect(container.querySelector("[data-agent-follow-up]")).not.toBeNull();
    } finally {
      unmount();
    }
  });

  test.each<{
    name: string;
    metadata: AgentMessageMetadata;
    selector: string;
  }>([
    {
      name: "navigation handoff",
      metadata: {
        handoff: { kind: "review-project-issues", project: "projects/sample" },
      },
      selector: "[data-agent-handoff]",
    },
    {
      name: "setup handoff",
      metadata: {
        actions: [{
          kind: "open",
          target: { kind: "transfer-project-databases", project: "projects/sample" },
        }],
      },
      selector: "[data-agent-action-open]",
    },
    {
      name: "plan draft",
      metadata: {
        actions: [{
          kind: "draft-plan",
          project: "projects/sample",
          database: "instances/prod/databases/app",
        }],
      },
      selector: "[data-agent-action-draft-plan]",
    },
    {
      name: "issue comment draft",
      metadata: {
        actions: [{
          kind: "draft-issue-comment",
          issue: "projects/sample/issues/7",
          comment: "Review this change.",
        }],
      },
      selector: "[data-agent-action-draft-issue-comment]",
    },
    {
      name: "access request draft",
      metadata: {
        accessRequestDraft: {
          project: "projects/sample",
          sourceDatabase: "instances/prod/databases/app",
          targets: ["instances/prod/databases/app"],
          statement: "SELECT * FROM orders",
        },
      },
      selector: "[data-agent-action-request-access]",
    },
  ])("hides the $name action when its transcript is archived", ({ metadata, selector }) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the answer.",
      metadata,
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      expect(container.querySelector(selector)).not.toBeNull();
      act(() => store.archiveChat(chat.id));
      expect(container.querySelector(selector)).toBeNull();
      expect(container.textContent).toContain("Review the answer.");
      act(() => store.unarchiveChat(chat.id));
      expect(container.querySelector(selector)).not.toBeNull();
    } finally {
      unmount();
    }
  });

  test("hides archived expression apply actions without removing evidence links", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const unregister = registerAgentExpressionTarget(
      chat.id,
      vi.fn(async () => true)
    );
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: 'Use this condition.\n```cel\nresource.environment == "prod"\n```',
      metadata: {
        evidence: {
          project: "projects/sample",
          references: [{
            kind: "issue",
            name: "projects/sample/issues/7",
            title: "Review changes",
          }],
        },
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      expect(
        container.querySelector("[data-agent-expression-draft]")
      ).not.toBeNull();
      act(() => store.archiveChat(chat.id));
      expect(container.querySelector("[data-agent-expression-draft]")).toBeNull();
      expect(container.querySelector("code")?.textContent).toContain(
        'resource.environment == "prod"'
      );
      const evidenceTrigger = Array.from(container.querySelectorAll("button")).find(
        (button) => button.textContent === "agent.evidence.references"
      );
      act(() => evidenceTrigger?.click());
      expect(container.querySelector("a")?.textContent).toBe("Review changes");
    } finally {
      unmount();
      unregister();
    }
  });

  test("offers Retry in the current failed turn's error banner", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Count databases" });
    store.finishChatRun(chat.id, { status: "error", lastError: "Connection failed" });
    const onRetry = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onRetry={onRetry} />
    );
    render();
    try {
      const retry = container.querySelector<HTMLButtonElement>(
        '[role="alert"] [data-agent-retry]'
      );
      expect(retry?.textContent).toBe("agent.retry-last-chat-turn");
      expect(retry?.querySelector(".lucide-rotate-ccw")).not.toBeNull();
      act(() => retry?.click());
      expect(onRetry).toHaveBeenCalledOnce();
    } finally {
      unmount();
    }
  });

  test.each(["no user", "idle", "running", "awaiting_user", "archived", "queued"] as const)(
    "does not offer error Retry for %s",
    (state) => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      if (state !== "no user") {
        store.addMessage({ chatId: chat.id, role: "user", content: "Count databases" });
      }
      store.finishChatRun(chat.id, { status: "error", lastError: "Connection failed" });
      if (state === "idle" || state === "running" || state === "awaiting_user") {
        store.setChatStatus(chat.id, state, { lastError: "Connection failed" });
      }
      if (state === "archived") store.archiveChat(chat.id);
      if (state === "queued") {
        useAgentStore.setState({ queuedUserMessagesByChatId: {
          [chat.id]: { id: "queued-message", chatId: chat.id, text: "Queued request", page: { path: "/demo", title: "Demo" } },
        } });
      }
      const onRetry = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onRetry={onRetry} />
      );
      render();
      try {
        expect(container.querySelector('[role="alert"] [data-agent-retry]')).toBeNull();
        expect(onRetry).not.toHaveBeenCalled();
      } finally {
        unmount();
      }
    }
  );

  test("renders the runtime semantic-type action without opening or writing automatically", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", workspaceName: "workspaces/demo" });
    mocks.getSemanticTypeDraft.mockReturnValue({ id: "new-type", title: "Same title", description: "Synthetic mask", algorithm: { kind: "full-mask", substitution: "hidden" } });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Review the semantic type.", metadata: { actions: acceptAgentResponseActions(undefined, [], undefined, [], [], ["semantic-1"]) } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelector<HTMLButtonElement>("[data-agent-action-draft-semantic-type]")?.disabled).toBe(false);
    expect(container.textContent).toContain("agent.semantic-type-draft.open");
    expect(container.textContent).toContain("Same title");
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test("renders a prepared exemption as a member-clicked prefill action, not only navigation", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({context:"projects/sample",workspaceName:"workspaces/demo"});
    mocks.getMaskingExemptionDraft.mockReturnValue({project:"projects/sample",scope:{kind:"database",database:"instances/primary/databases/app"},members:["user:member@example.com"],condition:'resource.column_name == "email"',description:"Temporary access",expirationTimestamp:null});
    store.addMessage({chatId:chat.id,role:"assistant",content:"Review the new exemption draft.",metadata:{actions:acceptAgentResponseActions(undefined,[],undefined,[],["exemption-1"])}});
    const {container,render,unmount}=renderIntoContainer(<AgentChat />);
    render();
    const button=container.querySelector<HTMLButtonElement>("[data-agent-action-draft-masking-exemption]");
    expect(button?.disabled).toBe(false);
    expect(button?.textContent).toContain("agent.masking-exemption-draft.open");
    expect(container.textContent).toContain("user:member@example.com");
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.revalidateMaskingExemptionDraft).not.toHaveBeenCalled();
    expect(mocks.consumeMaskingExemptionDraft).not.toHaveBeenCalled();
    unmount();
  });

  test("renders an actionable prepared masking draft when the model omits its action", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", workspaceName: "workspaces/demo" });
    const actions = acceptAgentResponseActions(undefined, [], undefined, ["runtime-draft"]);
    const draft = {
      title: "Mask email columns",
      condition: 'resource.column_name == "email"',
      semanticType: "bb.default",
      scope: { kind: "workspace" },
    };
    mocks.getMaskingRuleDraft.mockReturnValue(draft);
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the prepared masking rule draft.",
      metadata: { actions },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const button = container.querySelector<HTMLButtonElement>("[data-agent-action-draft-masking-rule]");
    expect(button?.textContent).toContain("agent.masking-rule-draft.open");
    expect(button?.disabled).toBe(false);
    expect(container.querySelector("code")?.textContent).toBe(draft.condition);
    expect(container.textContent).toContain("agent.masking-rule-draft.unsaved");
    expect(mocks.getMaskingRuleDraft).toHaveBeenCalledWith("runtime-draft", {
      workspaceName: "workspaces/demo", userName: "users/member@example.com", chatId: chat.id,
    });
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.revalidateMaskingRuleDraft).not.toHaveBeenCalled();
    expect(mocks.consumeMaskingRuleDraft).not.toHaveBeenCalled();
    unmount();
  });

  test("offers member-clicked apply actions for one CEL draft in its source editor", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const apply = vi.fn(async () => true);
    const unregister = registerAgentExpressionTarget(chat.id, apply);
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: 'This matches production.\n```cel\nresource.environment == "prod"\n```',
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const selectorButton = container.querySelector<HTMLButtonElement>(
      '[data-agent-expression-draft="selector"]'
    );
    expect(selectorButton?.textContent).toContain(
      "cel.agent-help.apply-selector"
    );
    expectCompactPrimaryAction(
      selectorButton,
      "cel.agent-help.apply-selector",
      "file-pen-line"
    );
    expectCompactPrimaryAction(
      container.querySelector('[data-agent-expression-draft="raw"]'),
      "cel.agent-help.apply-raw",
      "file-pen-line"
    );
    expect(selectorButton?.parentElement).toHaveClass("-ml-2", "gap-2");
    expect(selectorButton).not.toHaveClass("-ml-2");
    expect(
      container.querySelector('[data-agent-expression-draft="raw"]')
    ).not.toHaveClass("-ml-2");
    expect(
      container.querySelector('[data-agent-expression-draft="raw"]')
    ).not.toBeNull();

    await act(async () => selectorButton?.click());

    expect(apply).toHaveBeenCalledWith(
      'resource.environment == "prod"',
      "selector"
    );
    expect(selectorButton?.disabled).toBe(true);
    expect(useAgentStore.getState().minimized).toBe(true);
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
    expect(store.getMessages(chat.id)).toHaveLength(1);
    expectCompactPrimaryAction(
      selectorButton,
      "cel.agent-help.applied-selector",
      "check"
    );
    act(() => unregister());
    expect(
      container.querySelector('[data-agent-expression-draft="selector"]')
    ).toBeNull();
    unmount();
  });

  test("does not offer CEL apply actions outside an active editor chat", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: '```cel\nresource.environment == "prod"\n```',
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelector('[data-agent-expression-draft="selector"]')).toBeNull();
    unmount();
  });

  test("keeps the newly selected chat open when expression application finishes", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    let finishApply!: (result: boolean) => void;
    const unregister = registerAgentExpressionTarget(
      chat.id,
      () => new Promise((resolve) => { finishApply = resolve; })
    );
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: '```cel\nresource.environment == "prod"\n```',
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    await act(async () => {
      container.querySelector<HTMLButtonElement>('[data-agent-expression-draft="selector"]')?.click();
    });
    expect(finishApply).toBeDefined();
    let selectedChatId!: string;
    await act(async () => {
      selectedChatId = store.createChat({ context: "projects/another" }).id;
      finishApply(true);
    });
    expect(useAgentStore.getState().minimized).toBe(false);
    expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
    expect(store.getMessages(chat.id)).toHaveLength(1);
    act(() => unregister());
    unmount();
  });

  test("keeps a rejected selector draft available for a raw CEL retry", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const expression = 'resource.table_name.lowerAscii().contains("test")';
    const apply = vi.fn(async (_expression: string, mode: "selector" | "raw") =>
      mode === "raw"
    );
    const unregister = registerAgentExpressionTarget(chat.id, apply);
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: `\`\`\`cel\n${expression}\n\`\`\``,
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const selectorButton = container.querySelector<HTMLButtonElement>(
      '[data-agent-expression-draft="selector"]'
    );
    const rawButton = container.querySelector<HTMLButtonElement>(
      '[data-agent-expression-draft="raw"]'
    );

    await act(async () => selectorButton?.click());

    expect(apply).toHaveBeenNthCalledWith(1, expression, "selector");
    expect(selectorButton?.disabled).toBe(false);
    expect(useAgentStore.getState().minimized).toBe(false);
    expect(rawButton?.disabled).toBe(false);
    expect(container.querySelector('[role="alert"]')?.textContent).toBe(
      "cel.agent-help.invalid"
    );
    expect(selectorButton?.textContent).toBe("cel.agent-help.apply-selector");
    expect(container.textContent).not.toContain("cel.agent-help.applied-selector");

    await act(async () => rawButton?.click());

    expect(apply).toHaveBeenNthCalledWith(2, expression, "raw");
    expect(rawButton?.textContent).toBe("cel.agent-help.applied-raw");
    expectCompactPrimaryAction(rawButton, "cel.agent-help.applied-raw", "check");
    expect(rawButton?.disabled).toBe(true);
    expect(useAgentStore.getState().minimized).toBe(true);
    expect(selectorButton?.disabled).toBe(true);
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(container.textContent).not.toContain("cel.agent-help.applied-selector");
    act(() => unregister());
    unmount();
  });

  test("opens a denied Query as an unsent Access Grant draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const draft = {
      project: "projects/sample",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
    };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The API denied this query.",
      metadata: { accessRequestDraft: draft },
    });
    const route = {
      fullPath:
        "/sql-editor/sample/prod/app?accessGrantDraftKey=draft-1",
      query: { accessGrantDraftKey: "draft-1" },
    };
    mocks.prepareAgentDeniedQueryRequestRoute.mockResolvedValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = "/sql-editor/sample/prod/app";
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-request-access]"
    );
    expect(button?.textContent).toContain("agent.access-grant-draft.open");
    expectCompactPrimaryAction(button, "agent.access-grant-draft.open");
    expect(button).toHaveClass("-ml-2");
    expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
    await act(async () => button?.click());
    expect(mocks.prepareAgentDeniedQueryRequestRoute).toHaveBeenCalledWith(draft);
    expect(mocks.routerPush).toHaveBeenCalledWith(route);
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("opens a denied Query as an unsent Request Role draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const draft: RequestRoleDraft = {
      kind: "requestRole",
      project: "projects/sample",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      resources: ["instances/prod/databases/app/schemas/public/tables/orders"],
      statement: "SELECT * FROM orders",
      requiredPermissions: ["bb.sql.select"],
    };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The API denied this query.",
      metadata: { accessRequestDraft: draft },
    });
    const route = {
      fullPath: "/sql-editor/sample/prod/app?accessGrantDraftKey=role-1",
      query: { accessGrantDraftKey: "role-1" },
    };
    mocks.prepareAgentDeniedQueryRequestRoute.mockResolvedValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = "/sql-editor/sample/prod/app";
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-action-request-access]")
        ?.click();
    });

    expect(mocks.prepareAgentDeniedQueryRequestRoute).toHaveBeenCalledWith(
      draft
    );
    expect(mocks.routerPush).toHaveBeenCalledWith(route);
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("keeps the Agent visible when an existing access drawer rejects the draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The API denied this query.",
      metadata: {
        accessRequestDraft: {
          project: "projects/sample",
          sourceDatabase: "instances/prod/databases/app",
          targets: ["instances/prod/databases/app"],
          statement: "SELECT * FROM orders",
        },
      },
    });
    const route = {
      fullPath: "/sql-editor/sample/prod/app?accessGrantDraftKey=draft-2",
      query: { accessGrantDraftKey: "draft-2" },
    };
    mocks.prepareAgentDeniedQueryRequestRoute.mockResolvedValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = "/sql-editor/sample/prod/app";
    });
    mocks.wasDeniedQueryRequestDraftDelivered.mockReturnValue(false);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-action-request-access]")
        ?.click();
      await vi.waitFor(
        () =>
          expect(mocks.notify).toHaveBeenCalledWith(
            expect.objectContaining({
              style: "CRITICAL",
              title: "agent.access-grant-draft.unavailable",
            })
          ),
        { timeout: 5000 }
      );
    });

    expect(mocks.routerPush).toHaveBeenCalledWith(route);
    expect(useAgentStore.getState().minimized).toBe(false);
    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "CRITICAL",
        title: "agent.access-grant-draft.unavailable",
      })
    );
    unmount();
  });

  test("opens an Issue review comment draft without posting it", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const action = {
      kind: "draft-issue-comment" as const,
      issue: "projects/sample/issues/7",
      comment: "Please check exposure of the new column.",
    };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The change may expose personal data.",
      metadata: { actions: [action] },
    });
    const route = {
      name: "issue-detail",
      params: { issueId: "7" },
      query: { agentCommentDraft: "draft-7" },
    };
    mocks.prepareIssueCommentDraftRoute.mockReturnValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = JSON.stringify(route);
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-draft-issue-comment]"
    );
    expect(button?.textContent).toContain("agent.issue-comment-draft.open");
    expectCompactPrimaryAction(button, "agent.issue-comment-draft.open");
    expect(button).toHaveClass("-ml-2");
    await act(async () => button?.click());
    expect(mocks.prepareIssueCommentDraftRoute).toHaveBeenCalledWith(
      action.issue,
      action.comment
    );
    expect(mocks.routerPush).toHaveBeenCalledWith(route);
    expect(mocks.wasIssueCommentDraftDelivered).toHaveBeenCalledWith("draft-7");
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("minimizes after the Issue draft redirects to its Plan review", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the proposed change.",
      metadata: {
        actions: [{
          kind: "draft-issue-comment",
          issue: "projects/sample/issues/7",
          comment: "Please check exposure of the new column.",
        }],
      },
    });
    const route = {
      fullPath:
        "/projects/sample/issues/7?phase=review&agentCommentDraft=draft-7#issue-comment-editor",
      query: { agentCommentDraft: "draft-7" },
    };
    mocks.prepareIssueCommentDraftRoute.mockReturnValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath =
        "/projects/sample/plans/9?phase=review&agentCommentDraft=draft-7";
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>(
          "[data-agent-action-draft-issue-comment]"
        )
        ?.click();
    });

    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("keeps the Agent visible when the Issue comment editor never receives the draft", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review this Issue.",
      metadata: {
        actions: [{
          kind: "draft-issue-comment",
          issue: "projects/sample/issues/7",
          comment: "Check the security boundary.",
        }],
      },
    });
    const route = {
      fullPath: "/projects/sample/issues/7?agentCommentDraft=draft-7",
      query: { agentCommentDraft: "draft-7" },
    };
    mocks.prepareIssueCommentDraftRoute.mockReturnValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = route.fullPath;
    });
    mocks.wasIssueCommentDraftDelivered.mockReturnValue(false);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>("[data-agent-action-draft-issue-comment]")
        ?.click();
      await vi.waitFor(
        () =>
          expect(mocks.notify).toHaveBeenCalledWith(
            expect.objectContaining({
              style: "CRITICAL",
              title: "agent.issue-comment-draft.unavailable",
            })
          ),
        { timeout: 5000 }
      );
    });

    expect(useAgentStore.getState().minimized).toBe(false);
    expect(mocks.wasIssueCommentDraftDelivered).toHaveBeenCalledWith("draft-7");
    unmount();
  });

  test("opens a populated Plan draft only after the member clicks its answer action", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const content = "Review this DDL:\n\n```sql\nCREATE TABLE notes (id INT);\n```";
    const action = {
      kind: "draft-plan" as const,
      project: "projects/sample",
      database: "instances/prod/databases/app",
    };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content,
      metadata: { actions: [action] },
    });
    const route = { name: "plan-create", params: { projectId: "sample" } };
    mocks.prepareAgentPlanDraftRoute.mockResolvedValue(route);
    mocks.routerPush.mockImplementation(async () => {
      mocks.currentPath = JSON.stringify(route);
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-draft-plan]"
    );
    expect(button?.textContent).toContain("agent.plan-draft.open");
    expectCompactPrimaryAction(button, "agent.plan-draft.open");
    expect(button).toHaveClass("-ml-2");
    expect(button?.parentElement).toHaveClass("max-w-full");
    expect(mocks.prepareAgentPlanDraftRoute).not.toHaveBeenCalled();
    await act(async () => button?.click());
    expect(mocks.prepareAgentPlanDraftRoute).toHaveBeenCalledWith(
      action,
      content,
      "agent.plan-draft.default-title"
    );
    expect(mocks.routerPush).toHaveBeenCalledWith(route);
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("applies generated SQL to the open Plan draft without navigating", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const content = "```sql\nCREATE TABLE notes (id INT);\n```";
    const action = {
      kind: "draft-plan" as const,
      project: "projects/sample",
      database: "instances/prod/databases/app",
    };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content,
      metadata: { actions: [action] },
    });
    mocks.currentRoute = {
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "sample", planId: "create" },
    };
    let reportApplied!: () => void;
    mocks.applyPlanDraftToCurrent.mockImplementation((request) => {
      reportApplied = request.onApplied;
      return true;
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-draft-plan]"
    );
    expect(button?.textContent).toContain("agent.plan-draft.apply-to-current-plan");
    expectCompactPrimaryAction(
      button,
      "agent.plan-draft.apply-to-current-plan",
      "file-pen-line"
    );
    await act(async () => button?.click());

    expect(mocks.applyPlanDraftToCurrent).toHaveBeenCalledWith({
      project: action.project,
      database: action.database,
      sql: "CREATE TABLE notes (id INT);",
      onApplied: expect.any(Function),
    });
    expect(mocks.prepareAgentPlanDraftRoute).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(useAgentStore.getState().minimized).toBe(false);
    act(() => reportApplied());
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test.each([false, true])("keeps the Agent open before SQL is applied (handled=%s)", async (handled) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "```sql\nCREATE TABLE notes (id INT);\n```",
      metadata: { actions: [{
        kind: "draft-plan", project: "projects/sample", database: "instances/prod/databases/app",
      }] },
    });
    mocks.currentRoute = {
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "sample", planId: "create" },
    };
    mocks.applyPlanDraftToCurrent.mockReturnValue(handled);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    await act(async () => {
      container.querySelector<HTMLButtonElement>("[data-agent-action-draft-plan]")?.click();
    });
    expect(mocks.applyPlanDraftToCurrent).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().minimized).toBe(false);
    if (!handled) {
      expect(mocks.notify).toHaveBeenCalledWith(expect.objectContaining({
        title: "agent.plan-draft.current-plan-unavailable",
      }));
    } else {
      expect(mocks.notify).not.toHaveBeenCalled();
    }
    unmount();
  });

  test("updates the Plan draft action when navigation changes with the Agent open", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "```sql\nCREATE TABLE notes (id INT);\n```",
      metadata: {
        actions: [{
          kind: "draft-plan",
          project: "projects/sample",
          database: "instances/prod/databases/app",
        }],
      },
    });
    mocks.currentRoute = {
      name: "project.detail",
      params: { projectId: "sample" },
    };
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-draft-plan]"
    );
    expect(button?.textContent).toContain("agent.plan-draft.open");

    mocks.currentRoute = {
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "sample", planId: "create" },
    };
    mocks.currentPath = "/projects/sample/plans/create";
    act(() => {
      for (const listener of mocks.routeListeners) listener();
    });
    expect(button?.textContent).toContain(
      "agent.plan-draft.apply-to-current-plan"
    );

    mocks.currentRoute = {
      name: "project.detail",
      params: { projectId: "sample" },
    };
    mocks.currentPath = "/projects/sample";
    act(() => {
      for (const listener of mocks.routeListeners) listener();
    });
    expect(button?.textContent).toContain("agent.plan-draft.open");
    unmount();
  });

  test("keeps the Agent open when navigation to the Plan draft is blocked", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "```sql\nCREATE TABLE notes (id INT);\n```",
      metadata: {
        actions: [{
          kind: "draft-plan",
          project: "projects/sample",
          database: "instances/prod/databases/app",
        }],
      },
    });
    mocks.prepareAgentPlanDraftRoute.mockResolvedValue({ name: "plan-create" });
    mocks.routerPush.mockResolvedValue(undefined);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container.querySelector<HTMLButtonElement>("[data-agent-action-draft-plan]")?.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().minimized).toBe(false);
    unmount();
  });

  test("does not navigate when the Plan draft can no longer be prepared", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "```sql\nCREATE TABLE notes (id INT);\n```",
      metadata: {
        actions: [{
          kind: "draft-plan",
          project: "projects/sample",
          database: "instances/prod/databases/app",
        }],
      },
    });
    mocks.prepareAgentPlanDraftRoute.mockRejectedValue(new Error("moved"));
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-draft-plan]"
    );
    await act(async () => button?.click());
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({ title: "agent.plan-draft.unavailable" })
    );
    expect(button?.disabled).toBe(false);
    unmount();
  });

  test("keeps an Ask next prompt separate from the answer action", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the issue, then check its impact.",
      metadata: {
        actions: [
          {
            kind: "open",
            target: { kind: "review-project-issues", project: "projects/sample" },
          },
          { kind: "continue", prompt: "What is the impact of these issues?" },
        ],
      },
    });
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onContinueAction={onContinueAction} />
    );
    render();

    const open = container.querySelector<HTMLAnchorElement>("[data-agent-action-open]");
    expect(open?.getAttribute("href")).toContain(PROJECT_V1_ROUTE_ISSUES);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    const answer = container.querySelector("[data-agent-answer]");
    const followUpRow = container.querySelector("[data-agent-follow-up]");
    expect(answer?.contains(open)).toBe(true);
    expect(answer?.contains(followUpRow)).toBe(false);
    expect(followUpRow?.textContent).toContain("agent.follow-up.ask-next");
    expect(followUpRow?.textContent).toContain("What is the impact of these issues?");
    const followUp = followUpRow?.querySelector<HTMLButtonElement>(
      "[data-agent-action-continue]"
    );
    expect(followUp?.textContent).toContain("What is the impact of these issues?");
    expect(followUpRow?.textContent).not.toContain("agent.follow-up.add-to-message");
    expect(followUp?.className).not.toContain("border");
    const question = followUpRow?.querySelector<HTMLElement>(
      "[data-agent-follow-up-question]"
    );
    expect(question?.closest("button")).toBe(followUp);
    act(() => followUp?.click());
    expect(onContinueAction).toHaveBeenCalledWith(
      chat.id,
      "What is the impact of these issues?",
      []
    );
    expect(store.getMessages(chat.id)).toHaveLength(1);
    act(() => {
      store.queueUserMessage({
        chatId: chat.id,
        text: "What is the impact of these issues?",
        page: { path: "/projects/sample", title: "Sample" },
      });
    });
    expect(followUp?.disabled).toBe(true);
    unmount();
  });

  test("renders an exact failed Task Open action", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The production task failed.",
      metadata: {
        actions: [
          {
            kind: "open",
            target: {
              kind: "view-failed-task",
              task: "projects/sample/plans/7/rollout/stages/prod/tasks/t1",
            },
          },
        ],
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const link = container.querySelector<HTMLAnchorElement>(
      "[data-agent-action-open]"
    );
    expect(link?.textContent).toContain("agent.handoff.view-failed-task");
    expect(JSON.parse(link?.getAttribute("href") ?? "null")).toEqual({
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: {
        projectId: "sample",
        planId: "7",
        stageId: "prod",
        taskId: "t1",
      },
    });
    unmount();
  });

  test("shows three Ask next questions below the answer timestamp", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The plan is ready for review.",
      metadata: {
        actions: [
          { kind: "continue", prompt: "What will this change affect?" },
          { kind: "continue", prompt: "How should I validate it?" },
          { kind: "continue", prompt: "What are the rollback options?" },
        ],
      },
    });
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onContinueAction={onContinueAction} />
    );
    render();

    const followUp = container.querySelector("[data-agent-follow-up]");
    const timestamp = container.querySelector("[data-agent-message-timestamp]");
    const choices = Array.from(
      followUp?.querySelectorAll<HTMLButtonElement>("[data-agent-action-continue]") ?? []
    );
    expect(followUp?.previousElementSibling).toBe(timestamp);
    expect(choices.map((choice) => choice.textContent?.trim())).toEqual([
      "What will this change affect?",
      "How should I validate it?",
      "What are the rollback options?",
    ]);
    act(() => choices[1]?.click());
    expect(onContinueAction).toHaveBeenCalledWith(
      chat.id,
      "How should I validate it?",
      []
    );
    expect(store.getMessages(chat.id)).toHaveLength(1);
    unmount();
  });

  test("reveals a question added to the latest answer after completion", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const message = store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Transfer a database.",
      metadata: {
        actions: [{
          kind: "open",
          target: { kind: "transfer-project-databases", project: "projects/sample" },
        }],
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const scrollContainer = container.firstElementChild as HTMLElement;
    Object.defineProperty(scrollContainer, "scrollHeight", { value: 1100 });
    Object.defineProperty(scrollContainer, "clientHeight", { value: 500 });
    scrollContainer.scrollTop = 500;

    act(() => {
      store.appendContinueActions(
        chat.id,
        message!.id,
        [{ kind: "continue", prompt: "How do I connect a new instance?" }],
        []
      );
    });

    expect(container.querySelector("[data-agent-follow-up]")?.textContent).toContain(
      "How do I connect a new instance?"
    );
    expect(scrollContainer.scrollTop).toBe(1100);
    unmount();
  });

  test("reveals additional follow-up questions after the first one arrives", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const message = store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the proposed change.",
      metadata: {
        actions: [{ kind: "continue", prompt: "What will this change affect?" }],
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const scrollContainer = container.firstElementChild as HTMLElement;
    Object.defineProperty(scrollContainer, "scrollHeight", { value: 1100 });
    Object.defineProperty(scrollContainer, "clientHeight", { value: 500 });
    scrollContainer.scrollTop = 500;

    act(() => {
      useAgentStore.setState((state) => ({
        messagesByChatId: {
          ...state.messagesByChatId,
          [chat.id]: [
            {
              ...message!,
              metadata: {
                ...message!.metadata,
                actions: [
                  { kind: "continue", prompt: "What will this change affect?" },
                  { kind: "continue", prompt: "How should I validate it?" },
                ],
              },
            },
          ],
        },
      }));
    });

    expect(scrollContainer.scrollTop).toBe(1100);
    unmount();
  });

  test("disables the follow-up question when a draft would be replaced", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the issue.",
      metadata: {
        actions: [{ kind: "continue", prompt: "What is the impact?" }],
      },
    });
    store.setDraft(chat.id, { text: "My own question" });
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onContinueAction={onContinueAction} />
    );
    render();

    const question = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-continue]"
    );
    expect(question?.textContent).toContain("What is the impact?");
    expect(question?.disabled).toBe(true);
    act(() => question?.click());
    expect(onContinueAction).not.toHaveBeenCalled();
    unmount();
  });

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
    expectCompactPrimaryAction(link, "agent.handoff.review-database-changes");
    expect(link).toHaveClass("-ml-2");
    expect(link?.getAttribute("href")).toContain(PROJECT_V1_ROUTE_ISSUES);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test.each(["success", "blocked", "error", "chat switch"] as const)(
    "minimizes a primary Open only after verified navigation: %s",
    async (outcome) => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      store.addMessage({
        chatId: chat.id,
        role: "assistant",
        content: "Review the current change queue.",
        metadata: { handoff: { kind: "review-database-changes", project: "projects/sample" } },
      });
      let finishNavigation!: () => void;
      mocks.routerPush.mockImplementation(async (to) => {
        await new Promise<void>((resolve) => { finishNavigation = resolve; });
        if (outcome === "error") throw new Error("Navigation failed");
        if (outcome !== "blocked") mocks.currentPath = JSON.stringify(to);
      });
      useAgentStore.setState({ visible: true, minimized: false });
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      render();
      await act(async () => container.querySelector<HTMLAnchorElement>("[data-agent-handoff]")?.click());
      expect(useAgentStore.getState().minimized).toBe(false);
      expect(finishNavigation).toBeDefined();
      await act(async () => {
        if (outcome === "chat switch") store.createChat({ context: "projects/another" });
        finishNavigation();
      });
      expect(useAgentStore.getState().minimized).toBe(outcome === "success");
      expect(store.getMessages(chat.id)).toHaveLength(1);
      unmount();
    }
  );

  test.each<{
    handoff: AgentHandoff;
    targetPath: string;
    resultingPath: string;
    arrived: boolean;
  }>([
    {
      handoff: { kind: "review-issue-plan", plan: "projects/sample/plans/9" },
      targetPath: "/projects/sample/plans/9?phase=review",
      resultingPath: "/projects/sample/plans/9?phase=rollout",
      arrived: false,
    },
    {
      handoff: { kind: "review-issue-plan", plan: "projects/sample/plans/9" },
      targetPath: "/projects/sample/plans/9?phase=review",
      resultingPath: "/projects/sample/plans/9",
      arrived: false,
    },
    {
      handoff: { kind: "review-issue-plan", plan: "projects/sample/plans/9" },
      targetPath: "/projects/sample/plans/9?phase=review",
      resultingPath: "/projects/sample/plans/9?phase=review&phase=rollout",
      arrived: false,
    },
    {
      handoff: { kind: "open-database-catalog", project: "projects/sample", database: "instances/prod/databases/app" },
      targetPath: "/projects/sample/databases/app?parent=instances%2Fprod#catalog",
      resultingPath: "/projects/sample/databases/app?parent=instances%2Fprod#overview",
      arrived: false,
    },
    {
      handoff: { kind: "open-database-catalog", project: "projects/sample", database: "instances/prod/databases/app" },
      targetPath: "/projects/sample/databases/app?parent=instances%2Fprod#catalog",
      resultingPath: "/projects/sample/databases/app?parent=instances%2Fother#catalog",
      arrived: false,
    },
    {
      handoff: { kind: "review-issue-plan", plan: "projects/sample/plans/9" },
      targetPath: "/projects/sample/plans/9?phase=review",
      resultingPath: "/projects/sample/plans/9?view=list&phase=review",
      arrived: true,
    },
    {
      handoff: { kind: "open-global-masking" },
      targetPath: "/settings/global-masking",
      resultingPath: "/settings/global-masking?view=rules",
      arrived: true,
    },
    {
      handoff: { kind: "open-database-catalog", project: "projects/sample", database: "instances/prod/databases/app" },
      targetPath: "/projects/sample/databases/app?parent=instances%2Fprod#catalog",
      resultingPath: "/projects/sample/databases/app?schema=public&parent=instances%2Fprod#catalog",
      arrived: true,
    },
  ])("requires the primary Open destination state: $resultingPath", async ({ handoff, targetPath, resultingPath, arrived }) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Open the relevant page.", metadata: { handoff } });
    mocks.resolvedPath = targetPath;
    mocks.currentPath = arrived ? "/initial" : resultingPath;
    mocks.routerPush.mockImplementation(async () => {
      if (arrived) mocks.currentPath = resultingPath;
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    await act(async () => container.querySelector<HTMLAnchorElement>("[data-agent-handoff]")?.click());
    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().minimized).toBe(arrived);
    unmount();
  });

  test.each([
    { ctrlKey: true }, { metaKey: true }, { shiftKey: true }, { altKey: true }, { button: 1 },
  ])("preserves native modified primary Open clicks: %j", (init) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review the current change queue.",
      metadata: { handoff: { kind: "review-database-changes", project: "projects/sample" } },
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    let nativeNavigationAllowed = false;
    container.addEventListener("click", (click) => {
      nativeNavigationAllowed = !click.defaultPrevented;
      click.preventDefault();
    });
    const event = new MouseEvent("click", { bubbles: true, cancelable: true, ...init });
    act(() => { container.querySelector<HTMLAnchorElement>("[data-agent-handoff]")?.dispatchEvent(event); });
    expect(nativeNavigationAllowed).toBe(true);
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(useAgentStore.getState().minimized).toBe(false);
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
    expect(container.querySelector("[data-agent-follow-up]")).toBeNull();
    unmount();
  });

  test.each([
    "transfer-project-databases",
    "connect-project-instance",
    "create-project-database",
  ] as const)("opens %s in front of the Agent window", async (kind) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Continue project setup.",
      metadata: { handoff: { kind, project: "projects/sample" } },
    });
    mocks.routerPush.mockImplementation(async (to) => {
      mocks.currentPath = JSON.stringify(to);
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "button[data-agent-handoff]"
    );
    expectCompactPrimaryAction(button, button?.textContent ?? "");
    expect(button).toHaveClass("-ml-2");
    await act(async () => button?.click());

    expect(button).not.toBeNull();
    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  test("keeps the Agent visible when a setup route does not open", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Connect an instance.",
      metadata: {
        handoff: {
          kind: "connect-project-instance",
          project: "projects/sample",
        },
      },
    });
    mocks.routerPush.mockResolvedValue(undefined);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    await act(async () => {
      container.querySelector<HTMLButtonElement>("button[data-agent-handoff]")?.click();
    });

    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(useAgentStore.getState().minimized).toBe(false);
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

  test("shows AI setup and a settings link when ai_enabled is false", () => {
    mocks.aiEnabled = false;

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);

    render();

    expect(container.textContent).toContain("agent.ai-not-configured.title");
    const configureLink = Array.from(
      container.querySelectorAll<HTMLAnchorElement>("a")
    ).find((link) =>
      link.textContent?.includes("agent.ai-not-configured.configure")
    );
    expect(configureLink).toBeInstanceOf(HTMLAnchorElement);
    expect(JSON.parse(configureLink?.getAttribute("href") ?? "null")).toMatchObject({
      name: SETTING_ROUTE_WORKSPACE_GENERAL,
      hash: "#ai-assistant",
    });
    act(() => configureLink?.click());
    expect(mocks.routerPush).toHaveBeenCalledWith(
      expect.objectContaining({
        name: SETTING_ROUTE_WORKSPACE_GENERAL,
        hash: "#ai-assistant",
      })
    );

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
            {
              kind: "plan",
              name: "projects/orders/plans/123",
              title: "Update orders",
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
    const planLink = Array.from(container.querySelectorAll("a")).find(
      (link) => link.textContent === "Update orders"
    );
    expect(issueLink).toBeInstanceOf(HTMLElement);
    expect(databaseLink).toBeInstanceOf(HTMLElement);
    expect(planLink).toBeInstanceOf(HTMLElement);

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

    act(() => {
      planLink?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: { projectId: "orders", planId: "123" },
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
      new Date(userMessage!.createdTs).toISOString(),
      new Date(assistantMessage!.createdTs).toISOString(),
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

    expect(assistantMessage!.toolCalls).toHaveLength(2);
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

  test("shows one concise provider error and reveals its raw detail on demand", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const rawError =
      'OpenAI API error: {"error":{"message":"API key not valid. Please pass a valid API key."}}';
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: `Error: ${rawError}`,
      metadata: { error: rawError },
    });
    store.finishChatRun(chat.id, { status: "error", lastError: rawError });

    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelectorAll("[data-agent-error-summary]")).toHaveLength(1);
    expect(container.querySelector("[data-agent-error-summary]")?.textContent).toBe(
      "API key not valid. Please pass a valid API key."
    );
    expect(container.textContent).not.toContain("OpenAI API error:");

    const details = container.querySelector<HTMLButtonElement>(
      "[data-agent-error-details]"
    );
    expect(details?.getAttribute("aria-expanded")).toBe("false");
    act(() => details?.click());
    expect(container.textContent).toContain(rawError);

    unmount();
  });
});
