import { fireEvent, within } from "@testing-library/react";
import { createInstance } from "i18next";
import type { ReactElement } from "react";
import { act, useState } from "react";
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
import enUS from "@/locales/en-US.json";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import { reactI18nextStub } from "@/test-utils/i18n";
import type { AgentApiAttempt } from "../logic/apiAttempt";
import { agentApiExecutionEvidence } from "../logic/apiExecution";
import type { RequestRoleDraft } from "../logic/deniedQueryAccess";
import { registerAgentExpressionTarget } from "../logic/expressionDraft";
import { buildOutboundHistory } from "../logic/outboundHistory";
import { acceptAgentResponseActions } from "../logic/responseActions";
import type {
  AgentActionReceipt,
  AgentChat as AgentChatRecord,
  AgentHandoff,
  AgentMessageMetadata,
} from "../logic/types";
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
  validateDeniedQueryRequestDraft: vi.fn(),
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

vi.mock("@/utils", async (importOriginal) => {
  const original = await importOriginal<typeof import("@/utils")>();
  return {
    ...original,
    extractDatabaseResourceName: (resource: string) =>
      original.extractDatabaseResourceName(resource),
    hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
  };
});

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/stores/app", () => {
  const state = {
    getSettingByName: mocks.getSettingByName,
    getWorkspaceProfile: () => ({}),
    currentUser: {
      name: "users/member@example.com",
      email: "member@example.com",
      workspace: "workspaces/demo",
    },
    workspace: { name: "workspaces/demo" },
    workspaceResourceName: () => "workspaces/demo",
    get serverInfo() {
      return { aiEnabled: mocks.aiEnabled };
    },
    notify: mocks.notify,
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  useAppStore.subscribe = () => () => {};
  return { useAppStore };
});

vi.mock("@/components/DatabaseSelect", () => ({
  DatabaseSelect: ({ value }: { value: string[] }) => (
    <div>{value.join(", ")}</div>
  ),
}));

vi.mock("@/components/monaco/MonacoEditor", () => ({
  MonacoEditor: ({ content }: { content: string }) => <pre>{content}</pre>,
}));

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
  validateDeniedQueryRequestDraft: mocks.validateDeniedQueryRequestDraft,
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
  expect(action).toHaveClass("min-h-7", "h-auto", "whitespace-normal", "text-xs", "text-accent", "max-w-full");
  expect(action).not.toHaveClass("border", "h-7");
  expect(action).toHaveAttribute("title", label);
  expect(action?.firstElementChild?.textContent).toBe(label);
  expect(action?.firstElementChild).toHaveClass("wrap-anywhere");
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
    landingDraft: initialState.landingDraft,
    currentChatId: initialState.currentChatId,
    workspaceName: "workspaces/demo",
    userEmail: "member@example.com",
    abortControllersByChatId: {},
    storagePressure: undefined,
    queuedUserMessagesByChatId: {},
    queuedUserMessageInFlightByChatId: {},
    sqlFixSourcesByChatId: {},
    sqlFixDraftsById: {},
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
  mocks.validateDeniedQueryRequestDraft.mockReset().mockResolvedValue(true);
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
  test.each([1, 2])("keeps %i saved receipts in the action group instead of reopening their drafts", (count) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "workspaces/demo", workspaceName: "workspaces/demo" });
    const receipts: Extract<AgentActionReceipt, { kind: "masking-rule" }>[] = Array.from({ length: count }, (_, index) => ({
      kind: "masking-rule",
      policy: "workspaces/demo/policies/MASKING_RULE",
      draftId: `saved-draft-${index}`,
      rule: {
        id: `saved-rule-${index}`,
        semanticType: "ssn",
        condition: { title: `Saved masking rule ${index + 1}`, expression: "true" },
      },
      savedAt: 100,
    }));
    mocks.getMaskingRuleDraft.mockReturnValue({
      title: "Old unsaved draft",
      condition: "true",
      semanticType: "ssn",
      scope: { kind: "workspace" },
    });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The masking rules were saved.",
      metadata: {
        actions: receipts.map(({ draftId }) => ({ kind: "draft-masking-rule", draftId })),
        actionReceipts: receipts,
      },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const group = container.querySelector("[data-agent-action-group]");
    expect(group?.querySelectorAll("[data-agent-action-receipt]")).toHaveLength(count);
    expect(group?.querySelectorAll("[data-agent-action-check-receipt]")).toHaveLength(count);
    for (let index = 0; index < count; index++) {
      expect(group).toHaveTextContent(`Saved masking rule ${index + 1}`);
    }
    expect(container.querySelector("[data-agent-action-draft-masking-rule]")).toBeNull();
    expect(container).not.toHaveTextContent("Old unsaved draft");
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test("offers a diagnostic SQL fix review without preparing or applying on render", async () => {
    const store = useAgentStore.getState();
    const resource = { kind: "plan" as const, name: "projects/sample/plans/9", project: "projects/sample", label: "projects/sample/plans/9" };
    const chat = store.createChat({ context: resource.project, capability: { kind: "diagnostic", resource } });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Review the SQL suggestion.", metadata: { actions: [{ kind: "review-sql-fix", draftId: "missing-local-source" }] } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    const button = container.querySelector<HTMLButtonElement>("[data-agent-action-review-sql-fix]");
    expectCompactPrimaryAction(button, "agent.sql-fix.review");
    expect(document.querySelector("[role=dialog]")).toBeNull();
    await act(async () => button?.click());
    expect(document.querySelector("[role=dialog]")?.textContent).toContain("agent.sql-fix.original-unavailable");
    expect(mocks.applyPlanDraftToCurrent).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    unmount();
  });

  test("does not turn a forged SQL fix action in a normal chat into a handoff", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "A normal answer.", metadata: { actions: [{ kind: "review-sql-fix", draftId: "forged" }] } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    expect(container.querySelector("[data-agent-action-review-sql-fix]")).toBeNull();
    unmount();
  });

  test("hides application preflight rows without deleting canonical outbound tool groups", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Inspect workspace metadata" });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Internal startup evidence", toolCalls: [{ id: "preflight-read", name: "get_page_state", arguments: "{}", metadata: "opaque-provider-metadata" }], metadata: { runPreflight: true, hidden: true } });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: "preflight-read", content: '{"result":"Canonical startup result"}', metadata: { runPreflight: true, hidden: true } });
    store.addMessage({ chatId: chat.id, role: "assistant", content: "Visible reviewed answer" });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      expect(container.textContent).not.toContain("Internal startup evidence");
      expect(container.textContent).not.toContain("Canonical startup result");
      expect(container.textContent).toContain("Visible reviewed answer");
      expect(store.getMessages(chat.id)).toHaveLength(4);
      const outbound = buildOutboundHistory("System instruction", store.getMessages(chat.id));
      expect(outbound.find((row) => row.toolCalls?.some((call) => call.id === "preflight-read"))?.toolCalls?.[0].metadata).toBe("opaque-provider-metadata");
      expect(outbound.find((row) => row.toolCallId === "preflight-read")?.content).toContain("Canonical startup result");
    } finally {
      unmount();
    }
  });

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

  test.each(["running", "awaiting_user", "queued", "remote", "controller", "pending answer", "queue in flight"] as const)(
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
      if (state === "remote") useAgentStore.setState({ remoteBusyChatIds: { [chat.id]: true } });
      if (state === "controller") store.setAbortController(chat.id, new AbortController());
      if (state === "pending answer") {
        store.awaitUser(chat.id, { toolCallId: "question", kind: "input", prompt: "Which database?" });
        useAgentStore.setState((current) => ({ chats: current.chats.map((candidate) => candidate.id === chat.id ? { ...candidate, status: "error", lastError: "Saving the question failed" } : candidate) }));
      }
      if (state === "queue in flight") useAgentStore.setState({ queuedUserMessageInFlightByChatId: { [chat.id]: true } });
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

  test.each(
    (["idle", "error", "paused"] as const).flatMap((status) =>
      ([undefined, "budget", "quota", "unavailable"] as const).map((reason) => ({ status, reason }))
    )
  )("allows Resend for $status with storage pressure $reason", ({ status, reason }) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const message = store.addMessage({ chatId: chat.id, role: "user", content: "Original request" })!;
    store.setChatStatus(chat.id, status, { lastError: status === "error" ? "The previous request failed" : null });
    useAgentStore.setState({ storagePressure: reason ? { reason, phase: "saving", bytes: 1024, limitBytes: 3 * 1024 * 1024 } : undefined });
    const onResend = vi.fn();
    const { container, render, unmount } = renderIntoContainer(<AgentChat onResend={onResend} />);
    render();
    try {
      const resend = container.querySelector<HTMLButtonElement>('button[aria-label="agent.resend-message"]');
      expect(resend).toBeEnabled();
      act(() => resend?.click());
      expect(onResend).toHaveBeenCalledExactlyOnceWith(message.id);
    } finally {
      unmount();
      useAgentStore.setState({ storagePressure: undefined });
    }
  });

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
      expect(retry).toHaveClass("max-w-full", "whitespace-normal", "wrap-anywhere");
      const footer = container.querySelector("[data-agent-error-actions]");
      expect(footer).toContainElement(retry);
      expect(footer).not.toContainElement(container.querySelector("[data-agent-error-details]"));
      expect(footer).not.toContainElement(container.querySelector("[data-agent-error-summary]"));
      act(() => retry?.click());
      expect(onRetry).toHaveBeenCalledOnce();
    } finally {
      unmount();
    }
  });

  test.each(["budget", "quota", "unavailable"] as const)("shows one %s storage failure in the transcript while preserving its diagnostic error", (reason) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" });
    store.finishChatRun(chat.id, { status: "error", lastError: "QuotaExceededError: Result not saved" });
    useAgentStore.setState({ storagePressure: { reason, phase: "saving", bytes: 1024, limitBytes: 3 * 1024 * 1024 } });
    const onRetry = vi.fn();
    const { container, render, unmount } = renderIntoContainer(<AgentChat onRetry={onRetry} />);
    render();
    try {
      const storage = container.querySelector("[data-agent-storage-status]");
      expect(storage).not.toBeNull();
      expect(container.querySelectorAll('[role="alert"]')).toHaveLength(1);
      expect(storage).toHaveTextContent(reason === "unavailable" ? "agent.storage.save-failed" : "agent.storage.full");
      expect(container.querySelector("[data-agent-error-summary]")).toBeNull();
      expect(container.querySelector("[data-agent-retry]")).toBeNull();
      act(() => container.querySelector<HTMLButtonElement>("[data-agent-error-details]")?.click());
      expect(storage).toHaveTextContent("QuotaExceededError: Result not saved");
      expect(store.getChat(chat.id)?.lastError).toBe("QuotaExceededError: Result not saved");
      expect(onRetry).not.toHaveBeenCalled();
    } finally {
      unmount();
      useAgentStore.setState({ storagePressure: undefined });
    }
  });

  test("retries transcript save failures through the store instead of retrying execution", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Keep this question" });
    const retrySaving = vi.fn(async () => true);
    const onRetry = vi.fn();
    const {normalizeAgentApiAttempts} = await import("../logic/apiAttempt");
    const [attempt] = normalizeAgentApiAttempts([null], chat.id)!;
    useAgentStore.setState(s => {s.chats.find(c => c.id === chat.id)!.apiAttempts=[attempt];});
    useAgentStore.setState({ flushRunPersistence: retrySaving, storagePressure: { reason: "unavailable", phase: "saving", bytes: 1024, limitBytes: 3 * 1024 * 1024 } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat onRetry={onRetry} />);
    render();
    try {
      await act(async () => within(container).getByRole("button", { name: "agent.storage.retry" }).click());
      expect(retrySaving).toHaveBeenCalledOnce();
      expect(onRetry).not.toHaveBeenCalled();
      expect(container).toHaveTextContent("Keep this question");
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      expect(container.querySelector("[data-agent-retry]")).toBeNull();
      expect(store.getChat(chat.id)?.apiAttempts).toEqual([attempt]);
    } finally {
      unmount();
      useAgentStore.setState({ flushRunPersistence: store.flushRunPersistence, storagePressure: undefined });
    }
  });

  test("keeps an uncertain result inline instead of repeating a chat-wide warning", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const execution = agentApiExecutionEvidence("fetch-failed", { attemptId: "lost-response" });
    const attempt: AgentApiAttempt = {
      version: 1, id: "lost-response", ordinal: 1, startedAt: Date.now(),
      scopeKey: "workspaces/demo", userName: "me@example.com", chatId: chat.id,
      runId: "update-run", rootUserMessageId: root.id, toolCallId: "update-project",
      logicalActionId: "update-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "settled", settledAt: Date.now(), execution,
    };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId }) }] });
    useAgentStore.setState(s => {
      s.messagesByChatId[chat.id].push({ id: "lost-result", createdTs: Date.now(), chatId: chat.id, role: "tool", toolCallId: attempt.toolCallId, content: JSON.stringify({ error: "Response lost", agentExecution: execution }), metadata: { apiAttemptId: attempt.id, runId: attempt.runId } });
      s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt];
    });
    expect(store.getMessages(chat.id)).toEqual(expect.arrayContaining([expect.objectContaining({ role: "tool", metadata: expect.objectContaining({ apiAttemptId: attempt.id }) })]));
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      expect(container).toHaveTextContent("agent.api-recovery.unknown");
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      act(() => { store.addMessage({ chatId: chat.id, role: "user", content: "List databases" }); });
      expect(container).toHaveTextContent("agent.api-recovery.unknown");
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
    } finally { unmount(); }
  });

  test("warns for a missing current result but not after a new user turn", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" })!;
    const attempt: AgentApiAttempt = {
      version: 1, id: "unfinished", ordinal: 1, startedAt: Date.now(),
      scopeKey: "workspaces/demo", userName: "me@example.com", chatId: chat.id,
      runId: "update-run", rootUserMessageId: root.id, toolCallId: "update-project",
      logicalActionId: "update-approval", operationId: "bytebase.v1.ProjectService.UpdateProject",
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "b".repeat(64),
      phase: "dispatching",
    };
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt]; });
    store.setChatStatus(chat.id, "running");
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    await act(async () => render());
    try {
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      act(() => store.setChatStatus(chat.id, "idle"));
      expect(container.querySelector("[data-agent-api-recovery]")).not.toBeNull();
      act(() => { store.addMessage({ chatId: chat.id, role: "user", content: "List databases" }); });
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      expect(store.getChat(chat.id)?.apiAttempts).toEqual([attempt]);
    } finally { unmount(); }
  });

  test("keeps a receipt-backed missing result inline after a later user turn", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const attempt: AgentApiAttempt = {
      version: 1, id: "unfinished", ordinal: 1, startedAt: Date.now(),
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "update-run", rootUserMessageId: root.id, toolCallId: "update-project",
      logicalActionId: "update-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "b".repeat(64),
      phase: "dispatching",
    };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId }) }], metadata: { runId: attempt.runId } });
    store.addMessage({ chatId: chat.id, role: "user", content: "List databases" });
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt]; });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const card = within(container).getByRole("button", { name: /call_api/ });
      expect(card).toHaveTextContent("agent.api-recovery.unknown");
      expect(card.querySelector(".animate-pulse")).toBeNull();
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      act(() => card.click());
      expect(container).toHaveTextContent("agent.api-recovery.no-replay");
      expect(store.getMessages(chat.id).filter(m => m.role === "tool")).toHaveLength(0);
    } finally { unmount(); }
  });

  test.each(["bytebase.v1.ProjectService.UpdateProject", "ProjectService/UpdateProject"])("does not reuse an earlier acknowledged receipt for a repeated tool call id: %s", (callOperationId) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update both projects" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const execution = agentApiExecutionEvidence("rpc-success", { attemptId: "first-write", httpStatus: 200 });
    const first: AgentApiAttempt = {
      version: 1, id: "first-write", ordinal: 1, startedAt: 1,
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "update-run", rootUserMessageId: root.id, toolCallId: "reused-id",
      logicalActionId: "first-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "settled", settledAt: 2, execution,
    };
    const second: AgentApiAttempt = {
      ...first, id: "second-write", logicalActionId: "second-approval", startedAt: 3,
      fingerprint: "b".repeat(64), phase: "dispatching", settledAt: undefined, execution: undefined,
    };
    const call = { id: first.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId: callOperationId }) };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: first.runId } });
    useAgentStore.setState(s => {
      s.chats.find(c => c.id === chat.id)!.apiAttempts = [first];
      s.messagesByChatId[chat.id].push({ id: "first-result", createdTs: Date.now(), chatId: chat.id, role: "tool", toolCallId: call.id, content: JSON.stringify({ status: 200, response: {}, agentExecution: execution }), metadata: { runId: first.runId, apiAttemptId: first.id } });
    });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: second.runId } });
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = [first, second]; });
    expect(store.getMessages(chat.id).map(message => [message.role, message.metadata?.apiAttemptId])).toEqual([["user", undefined], ["assistant", undefined], ["tool", first.id], ["assistant", undefined]]);
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const cards = within(container).getAllByRole("button", { name: /call_api/ });
      expect(cards[0]).toHaveTextContent("agent.api-recovery.acknowledged");
      expect(cards[1]).toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1]).not.toHaveTextContent("agent.api-recovery.acknowledged");
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
      act(() => cards[1].click());
      expect(container).toHaveTextContent("agent.api-recovery.no-replay");
      expect(store.getChat(chat.id)?.apiAttempts).toEqual([first, second]);
      expect(store.getMessages(chat.id).filter(m => m.role === "tool")).toHaveLength(1);
    } finally { unmount(); }
  });

  test("does not reuse a receipt linked outside the current result span", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the projects" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const execution = agentApiExecutionEvidence("fetch-failed", { attemptId: "earlier-write" });
    const attempt: AgentApiAttempt = {
      version: 1, id: "earlier-write", ordinal: 1, startedAt: 1,
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "write-run", rootUserMessageId: root.id, toolCallId: "reused-id",
      logicalActionId: "old-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "settled", settledAt: 2, execution,
    };
    const call = { id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId }) };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: attempt.runId } });
    useAgentStore.setState(s => {
      s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt];
      s.messagesByChatId[chat.id].push({ id: "earlier-result", createdTs: Date.now(), chatId: chat.id, role: "tool", toolCallId: call.id, content: JSON.stringify({ error: "Response lost", agentExecution: execution }), metadata: { runId: attempt.runId, apiAttemptId: attempt.id } });
    });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: attempt.runId } });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const cards = within(container).getAllByRole("button", { name: /call_api/ });
      expect(cards[0]).toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1]).not.toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1].querySelector(".text-warning")).toBeNull();
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
    } finally { unmount(); }
  });

  test.each(["assistant", "user"] as const)("does not borrow a later read result across a %s boundary", (boundary) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const attempt: AgentApiAttempt = {
      version: 1, id: "missing-write", ordinal: 1, startedAt: 1,
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "write-run", rootUserMessageId: root.id, toolCallId: "reused-id",
      logicalActionId: "write-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "dispatching",
    };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId }) }], metadata: { runId: attempt.runId } });
    if (boundary === "user") store.addMessage({ chatId: chat.id, role: "user", content: "List the projects" });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [{ id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId: "bytebase.v1.ProjectService.ListProjects" }) }], metadata: { runId: "read-run" } });
    store.addMessage({ chatId: chat.id, role: "tool", toolCallId: attempt.toolCallId, content: JSON.stringify({ status: 200, response: { projects: [] } }), metadata: { runId: "read-run" } });
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt]; });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const cards = within(container).getAllByRole("button", { name: /call_api/ });
      expect(cards[0]).toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1]).not.toHaveTextContent("agent.api-recovery.unknown");
      act(() => cards[0].click());
      expect(container).not.toHaveTextContent('"projects": []');
      act(() => cards[1].click());
      expect(container).toHaveTextContent('"projects": []');
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
    } finally { unmount(); }
  });

  test.each(["run", "operation", "multiple receipts", "legacy duplicate", "batch duplicate"] as const)("keeps %s receipt matches out of unrelated or ambiguous tool cards", (mismatch) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update the project" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const attempt: AgentApiAttempt = {
      version: 1, id: "missing-write", ordinal: 1, startedAt: 1,
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "write-run", rootUserMessageId: root.id, toolCallId: "reused-id",
      logicalActionId: "write-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "dispatching",
    };
    const call = { id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId: mismatch === "operation" ? "bytebase.v1.ProjectService.ListProjects" : operationId }) };
    const metadata = mismatch === "legacy duplicate" ? undefined : { runId: mismatch === "run" ? "another-run" : attempt.runId };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: mismatch === "batch duplicate" ? [call, call] : [call], metadata });
    if (mismatch === "legacy duplicate") store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call] });
    const attempts = mismatch === "multiple receipts" ? [attempt, { ...attempt, id: "another-write", logicalActionId: "another-approval" }] : [attempt];
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = attempts; });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      for (const card of within(container).getAllByRole("button", { name: /call_api/ })) {
        expect(card).not.toHaveTextContent("agent.api-recovery.unknown");
        expect(card).not.toHaveTextContent("agent.api-recovery.acknowledged");
      }
      expect(container.querySelectorAll("[data-agent-api-recovery]")).toHaveLength(attempts.length);
      expect(store.getChat(chat.id)?.apiAttempts).toEqual(attempts);
    } finally { unmount(); }
  });

  test("does not give a new pending permission the receipt of an earlier missing call", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    const root = store.addMessage({ chatId: chat.id, role: "user", content: "Update both projects" })!;
    const operationId = "bytebase.v1.ProjectService.UpdateProject";
    const attempt: AgentApiAttempt = {
      version: 1, id: "missing-write", ordinal: 1, startedAt: 1,
      scopeKey: "workspaces/demo", userName: "member@example.com", chatId: chat.id,
      runId: "write-run", rootUserMessageId: root.id, toolCallId: "reused-id",
      logicalActionId: "old-approval", operationId,
      path: "/bytebase.v1.ProjectService/UpdateProject", fingerprint: "a".repeat(64),
      phase: "dispatching",
    };
    const call = { id: attempt.toolCallId, name: "call_api", arguments: JSON.stringify({ operationId }) };
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: attempt.runId } });
    store.addMessage({ chatId: chat.id, role: "assistant", toolCalls: [call], metadata: { runId: attempt.runId } });
    store.awaitUser(chat.id, { kind: "confirm", toolCallId: call.id, prompt: operationId, apiPermission: { actionId: "new-approval", operationId } });
    useAgentStore.setState(s => { s.chats.find(c => c.id === chat.id)!.apiAttempts = [attempt]; });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const cards = within(container).getAllByRole("button", { name: /call_api/ });
      expect(cards[0]).toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1]).not.toHaveTextContent("agent.api-recovery.unknown");
      expect(cards[1].querySelector(".text-warning")).toBeNull();
      expect(container.querySelector("[data-agent-api-recovery]")).toBeNull();
    } finally { unmount(); }
  });

  test("reveals a new transcript storage failure without forcing scroll for pressure updates", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    for (let index = 0; index < 12; index++) {
      store.addMessage({ chatId: chat.id, role: "user", content: `Earlier question ${index}` });
      store.addMessage({ chatId: chat.id, role: "assistant", content: `Earlier answer ${index}` });
    }
    const messageCount = store.getMessages(chat.id).length;
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const transcript = container.querySelector<HTMLDivElement>("[data-agent-chat-scroll-area]")!;
      Object.defineProperty(transcript, "scrollHeight", { value: 1400 });
      transcript.scrollTop = 100;
      const pressure = { reason: "budget" as const, phase: "admission" as const, bytes: 3 * 1024 * 1024, limitBytes: 2 * 1024 * 1024 };
      act(() => useAgentStore.setState({ storagePressure: pressure }));
      expect(useAgentStore.getState().getMessages(chat.id)).toHaveLength(messageCount);
      expect(container.querySelector("[data-agent-storage-status]")).not.toBeNull();
      expect(transcript.scrollTop).toBe(1400);

      transcript.scrollTop = 250;
      act(() => useAgentStore.setState({ storagePressure: { ...pressure, bytes: pressure.bytes + 1024 } }));
      expect(transcript.scrollTop).toBe(250);
    } finally {
      unmount();
      useAgentStore.setState({ storagePressure: undefined });
    }
  });

  test("delegates confirmed transcript storage recovery to the supplied action", async () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.addMessage({ chatId: chat.id, role: "user", content: "Keep this question" });
    useAgentStore.setState({ storagePressure: { reason: "budget", phase: "saving", bytes: 1024, limitBytes: 3 * 1024 * 1024 } });
    const onFreeSpaceAndRetry = vi.fn(async () => true);
    const { container, render, unmount } = renderIntoContainer(<AgentChat onFreeSpaceAndRetry={onFreeSpaceAndRetry} />);
    document.body.appendChild(container);
    render();
    try {
      await act(async () => within(container).getByRole("button", { name: "agent.storage.free-and-retry" }).click());
      const dialog = await within(document.body).findByRole("dialog", { name: "agent.storage.free-title" });
      expect(onFreeSpaceAndRetry).not.toHaveBeenCalled();
      await act(async () => within(dialog).getByRole("button", { name: "agent.storage.free-and-retry" }).click());
      expect(onFreeSpaceAndRetry).toHaveBeenCalledOnce();
      expect(container.querySelectorAll("[data-agent-storage-status]")).toHaveLength(1);
    } finally {
      unmount();
      container.remove();
      useAgentStore.setState({ storagePressure: undefined });
    }
  });

  test.each(["workspace", "user", "chat"] as const)("resets storage diagnostic disclosure after the current %s changes", (scope) => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat();
    store.finishChatRun(chat.id, { status: "error", lastError: "Storage diagnostic" });
    const pressure = { reason: "unavailable" as const, phase: "saving" as const, bytes: 1024, limitBytes: 3 * 1024 * 1024 };
    useAgentStore.setState({ storagePressure: pressure });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      act(() => container.querySelector<HTMLButtonElement>("[data-agent-error-details]")?.click());
      expect(container.querySelector("[data-agent-error-details]")).toHaveAttribute("aria-expanded", "true");
      act(() => {
        if (scope === "workspace") useAgentStore.setState({ workspaceName: "workspaces/another" });
        else if (scope === "user") useAgentStore.setState({ userEmail: "another@example.com" });
        else store.createChat({ title: "Another chat" });
        useAgentStore.setState({ storagePressure: pressure });
      });
      expect(container.querySelectorAll("[data-agent-storage-status]")).toHaveLength(1);
      if (scope === "chat") expect(container).not.toHaveTextContent("Storage diagnostic");
      else expect(container.querySelector("[data-agent-error-details]")).toHaveAttribute("aria-expanded", "false");
    } finally {
      unmount();
      useAgentStore.setState({ storagePressure: undefined });
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
    expect(selectorButton?.parentElement).toHaveClass("justify-end", "flex-wrap", "w-full", "gap-2");
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

  describe("Access Grant recovery status", () => {
    const draft = {
      project: "projects/sample",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
    };
    const addAttempt = (runId?: string) => {
      const store = useAgentStore.getState();
      store.addMessage({
        role: "user",
        content: "Query orders",
        metadata: { runId },
      });
      store.addMessage({
        role: "assistant",
        toolCalls: [
          {
            id: "query",
            name: "call_api",
            arguments: JSON.stringify({
              operationId: "SQLService/Query",
              body: {
                name: draft.sourceDatabase,
                statement: draft.statement,
              },
            }),
          },
          {
            id: "prepare",
            name: "prepare_access_grant_draft",
            arguments: "{}",
          },
          {
            id: "done",
            name: "done",
            arguments: JSON.stringify({
              text: "The query was denied. An access request draft is ready.",
              success: false,
              hasRecoveryDraft: true,
            }),
          },
        ],
        metadata: { runId },
      });
      store.addMessage({
        role: "tool",
        toolCallId: "query",
        content: JSON.stringify({
          status: 403,
          error: "Permission denied",
          response: null,
        }),
        metadata: { runId },
      });
      store.addMessage({
        role: "tool",
        toolCallId: "prepare",
        content: JSON.stringify({ status: "unsaved", draft }),
        metadata: { runId },
      });
      return store.addMessage({
        role: "tool",
        toolCallId: "done",
        content: JSON.stringify({
          text: "The query was denied. An access request draft is ready.",
          success: false,
        }),
        metadata: { runId },
      })!;
    };

    test.each([undefined, "run-1"])(
      "shows a warning for a failed query with a runtime-prepared draft, run %s",
      (runId) => {
        const store = useAgentStore.getState();
        const result = addAttempt(runId);
        store.addMessage({
          role: "assistant",
          content: "The query was denied. Review the draft before submitting.",
          metadata: { runId, accessRequestDraft: draft },
        });
        const messages = store.getMessages(result.chatId);
        expect(messages).toHaveLength(6);
        const { container, render, unmount } = renderIntoContainer(<AgentChat />);
        try {
          render();
          expect(
            within(container).getByRole("button", { name: /^call_api/ })
          ).toHaveTextContent("agent.tool-failed");
          expect(
            within(container).getByRole("button", { name: /^done/ })
          ).toHaveTextContent("agent.tool-completed-with-warnings");
          expect(
            container.querySelector("[data-agent-action-request-access]")
          ).not.toBeNull();
          expect(store.getMessages(result.chatId)).toEqual(messages);
          expect(JSON.parse(result.content!)).toMatchObject({ success: false });
          expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
          expect(mocks.routerPush).not.toHaveBeenCalled();
        } finally {
          unmount();
        }
      }
    );

    test.each([
      {
        name: "a model claim without runtime metadata",
        metadata: { runId: "run-1" },
      },
      {
        name: "another run's draft",
        metadata: { runId: "run-2", accessRequestDraft: draft },
      },
      {
        name: "a pending answer",
        metadata: { runId: "run-1", pending: true, accessRequestDraft: draft },
      },
      {
        name: "an error answer",
        metadata: {
          runId: "run-1",
          error: "Preparation failed",
          accessRequestDraft: draft,
        },
      },
      {
        name: "a Request Role draft",
        metadata: {
          runId: "run-1",
          accessRequestDraft: {
            ...draft,
            kind: "requestRole",
            resources: [],
            requiredPermissions: [],
          },
        },
      },
    ] satisfies { name: string; metadata: AgentMessageMetadata }[])(
      "does not recover done from $name",
      ({ metadata }) => {
        addAttempt("run-1");
        useAgentStore.getState().addMessage({
          role: "assistant",
          content: "An access request draft is ready.",
          metadata,
        });
        const { container, render, unmount } = renderIntoContainer(<AgentChat />);
        try {
          render();
          const done = within(container).getByRole("button", { name: /^done/ });
          expect(done).toHaveTextContent("agent.tool-failed");
          expect(done).not.toHaveTextContent("agent.tool-completed-with-warnings");
        } finally {
          unmount();
        }
      }
    );

    test("does not apply a later turn's draft to an earlier done with reused call IDs", () => {
      const store = useAgentStore.getState();
      addAttempt();
      store.addMessage({ role: "assistant", content: "The query was denied." });
      addAttempt();
      store.addMessage({
        role: "assistant",
        content: "Review the access request draft.",
        metadata: { accessRequestDraft: draft },
      });
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      try {
        render();
        const done = within(container).getAllByRole("button", { name: /^done/ });
        expect(done[0]).toHaveTextContent("agent.tool-failed");
        expect(done[1]).toHaveTextContent(
          "agent.tool-completed-with-warnings"
        );
      } finally {
        unmount();
      }
    });
  });

  test("opens a freshly validated Access Grant draft inline without navigating", async () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const draft = {
      project: "projects/sample",
      sourceDatabase: "instances/prod/databases/app",
      targets: ["instances/prod/databases/app"],
      statement: "SELECT * FROM orders",
      schema: "",
      container: "",
      unmask: true,
    };
    const source = store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The API denied this query.",
      metadata: { accessRequestDraft: draft },
    });
    useAgentStore.setState({ visible: true, minimized: false });
    const onOpenAccessGrantDraft = vi.fn(() => true);
    const { container, render, unmount } = renderIntoContainer(<AgentChat onOpenAccessGrantDraft={onOpenAccessGrantDraft} />);
    render();

    const button = container.querySelector<HTMLButtonElement>(
      "[data-agent-action-request-access]"
    );
    expect(button?.textContent).toContain("agent.access-grant-draft.open");
    expectCompactPrimaryAction(button, "agent.access-grant-draft.open");
    expect(button).not.toHaveClass("-ml-2");
    expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
    expect(mocks.validateDeniedQueryRequestDraft).not.toHaveBeenCalled();
    expect(onOpenAccessGrantDraft).not.toHaveBeenCalled();
    await act(async () => button?.click());
    expect(mocks.validateDeniedQueryRequestDraft).toHaveBeenCalledWith(draft);
    expect(onOpenAccessGrantDraft).toHaveBeenCalledWith({
      ...draft,
      reason: "Request temporary query access to app with unmasked results.",
    }, {chatId: chat.id, messageId: source!.id});
    expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(useAgentStore.getState().minimized).toBe(true);
    unmount();
  });

  describe.each([false, true])("Access Grant Reason with Unmask=%s", (unmask) => {
    test.each([undefined, "", " \n ", "  Investigate the reported discrepancy.\n"])(
      "prefills an editable Reason from %j without changing the draft's scope",
      async (reason) => {
        mocks.useTranslation.mockReturnValue(reactI18nextStub().useTranslation());
        const store = useAgentStore.getState();
        const chat = store.createChat({ context: "projects/sample" });
        const draft = {
          project: "projects/sample",
          sourceDatabase: "projects/sample/instances/prod/databases/app",
          targets: ["projects/sample/instances/prod/databases/app"],
          statement: "SELECT id FROM orders",
          unmask,
          reason,
        };
        store.addMessage({
          chatId: chat.id,
          role: "assistant",
          content: "Review the access request draft.",
          metadata: { accessRequestDraft: draft },
        });
        const onOpenAccessGrantDraft = vi.fn(() => true);
        const { container, render, unmount } = renderIntoContainer(
          <AgentChat onOpenAccessGrantDraft={onOpenAccessGrantDraft} />
        );
        try {
          render();
          await act(async () => {
            container.querySelector<HTMLButtonElement>(
              "[data-agent-action-request-access]"
            )?.click();
          });
          expect(onOpenAccessGrantDraft).toHaveBeenCalledWith({
            ...draft,
            reason: reason?.trim()
              ? reason
              : unmask
                ? "Request temporary query access to app with unmasked results."
                : "Request temporary query access to app.",
          }, {chatId: chat.id, messageId: store.getMessages(chat.id).at(-1)!.id});
          expect(draft.reason).toBe(reason);
          expect(mocks.validateDeniedQueryRequestDraft).toHaveBeenCalledWith(draft);
          expect(mocks.routerPush).not.toHaveBeenCalled();
          expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
        } finally {
          unmount();
        }
      }
    );

    test.each([undefined, "", " \n ", "  Investigate the reported discrepancy.\n"])(
      "opens the actual request form with an editable English Reason from %j",
      async (reason) => {
        const actualI18next = await vi.importActual<typeof import("react-i18next")>(
          "react-i18next"
        );
        const i18n = createInstance();
        await i18n.init({
          lng: "en-US",
          fallbackLng: "en-US",
          resources: { "en-US": { translation: enUS } },
          interpolation: { escapeValue: false },
        });
        mocks.useTranslation.mockImplementation(() => {
          const translation = actualI18next.useTranslation();
          return { t: translation.t, i18n: translation.i18n };
        });
        const { AccessGrantRequestDrawer } = await import(
          "@/modules/access-control/AccessGrantRequestDrawer"
        );
        const { accessGrantServiceClientConnect } = await import("@/api");
        const createGrant = vi.spyOn(
          accessGrantServiceClientConnect,
          "createAccessGrant"
        );
        const draft: AccessGrantDraft = {
          project: "projects/sample",
          sourceDatabase: "projects/sample/instances/prod/databases/app",
          targets: ["projects/sample/instances/prod/databases/app"],
          statement: "SELECT id FROM orders",
          unmask,
          reason,
        };
        const store = useAgentStore.getState();
        const chat = store.createChat({ context: draft.project });
        store.addMessage({
          chatId: chat.id,
          role: "assistant",
          content: "Review the access request draft.",
          metadata: { accessRequestDraft: draft },
        });
        function DraftHost() {
          const [opened, setOpened] = useState<AccessGrantDraft>();
          return (
            <>
              <AgentChat
                onOpenAccessGrantDraft={(next) => {
                  setOpened(next);
                  return true;
                }}
              />
              {opened && (
                <AccessGrantRequestDrawer
                  project={opened.project}
                  targets={opened.targets}
                  query={opened.statement}
                  unmask={opened.unmask}
                  reason={opened.reason}
                  onClose={() => setOpened(undefined)}
                />
              )}
            </>
          );
        }
        const { container, render, unmount } = renderIntoContainer(
          <actualI18next.I18nextProvider i18n={i18n}>
            <DraftHost />
          </actualI18next.I18nextProvider>
        );
        try {
          render();
          await act(async () => {
            container.querySelector<HTMLButtonElement>(
              "[data-agent-action-request-access]"
            )?.click();
          });
          const dialog = within(document.body).getByRole("dialog");
          const reasonInput = within(dialog).getByRole<HTMLTextAreaElement>(
            "textbox"
          );
          const submit = within(dialog).getByRole("button", { name: "Submit" });
          const expectedReason = reason?.trim()
            ? reason
            : unmask
              ? "Request temporary query access to app with unmasked results."
              : "Request temporary query access to app.";
          expect(reasonInput).toHaveValue(expectedReason);
          expect(submit).toBeEnabled();
          act(() => {
            fireEvent.change(reasonInput, {
              target: { value: "Investigate a reported discrepancy." },
            });
          });
          expect(reasonInput).toHaveValue("Investigate a reported discrepancy.");
          expect(submit).toBeEnabled();
          act(() => {
            fireEvent.change(reasonInput, { target: { value: "" } });
          });
          expect(reasonInput).toHaveValue("");
          expect(submit).toBeDisabled();
          expect(createGrant).not.toHaveBeenCalled();
          expect(mocks.routerPush).not.toHaveBeenCalled();
          expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
        } finally {
          unmount();
          createGrant.mockRestore();
        }
      }
    );
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

  test.each(["invalid", "unavailable", "rejected"])("keeps the Agent visible when the inline draft is %s", async (failure) => {
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
    mocks.validateDeniedQueryRequestDraft.mockResolvedValue(failure !== "invalid");
    const onOpenAccessGrantDraft = vi.fn(() => false);
    useAgentStore.setState({ visible: true, minimized: false });
    const { container, render, unmount } = renderIntoContainer(<AgentChat onOpenAccessGrantDraft={failure === "unavailable" ? undefined : onOpenAccessGrantDraft} />);
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

    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.prepareAgentDeniedQueryRequestRoute).not.toHaveBeenCalled();
    if (failure === "invalid") expect(onOpenAccessGrantDraft).not.toHaveBeenCalled();
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
    expect(button).not.toHaveClass("-ml-2");
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
    expect(button).not.toHaveClass("-ml-2");
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
      [],
      store.getMessages(chat.id)[0].id
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

  describe("runtime-owned query follow-ups", () => {
    beforeEach(() => {
      mocks.useTranslation.mockReturnValue({
        t: (key: string) => {
          const prefix = "agent.follow-up.";
          return key.startsWith(prefix)
            ? (enUS.agent["follow-up"] as Record<string, string>)[
                key.slice(prefix.length)
              ] ?? key
            : key;
        },
        i18n: { language: "en-US" },
      });
    });

    const database = "instances/prod/databases/app";
    const references = [
      {
        kind: "database" as const,
        name: database,
        project: "projects/sample",
        label: "app",
      },
    ];
    const expiryQuestion =
      "When does my masking exemption for private_value in app expire?";
    const countQuestion = "How many records match this query?";
    const groupQuestion = "Can you group these results by private_value?";
    const ordinaryQuestions = [countQuestion, groupQuestion];
    const maskQuestion = enUS.agent["follow-up"]["query-mask"];
    const unmaskQuestion = enUS.agent["follow-up"]["query-unmask"];
    const addQueryTurn = (
      chatId: string,
      id: string,
      masked: boolean,
      questions = [countQuestion, expiryQuestion, groupQuestion],
      queryMaskingEligibility?: { database: string; allowed: boolean }
    ) => {
      const store = useAgentStore.getState();
      store.addMessage({
        chatId,
        role: "user",
        content: "Query private_value from records.",
        metadata: { resourceReferences: references },
      });
      store.addMessage({
        chatId,
        role: "assistant",
        toolCalls: [
          {
            id,
            name: "call_api",
            arguments: JSON.stringify({
              operationId: "SQLService/Query",
              body: {
                name: database,
                statement: "SELECT private_value FROM records LIMIT 1",
              },
            }),
          },
        ],
      });
      store.addMessage({
        chatId,
        role: "tool",
        toolCallId: id,
        content: JSON.stringify({
          status: 200,
          response: {
            results: [
              {
                columnNames: ["private_value"],
                columnTypeNames: ["text"],
                rows: [{ values: [{ stringValue: masked ? "******" : "visible" }] }],
                rowsCount: "1",
                masked: [masked ? { semanticTypeId: "bb.default" } : {}],
              },
            ],
          },
        }),
      });
      return store.addMessage({
        chatId,
        role: "assistant",
        content: "The query returned one row.",
        metadata: {
          resourceReferences: references,
          ...(queryMaskingEligibility ? { queryMaskingEligibility } : {}),
          actions: questions.map((prompt) => ({ kind: "continue" as const, prompt })),
        },
      })!;
    };

    test("preserves ordinary saved query questions while removing unsupported exemptions", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      const answer = addQueryTurn(chat.id, "query-visible", false);
      const savedMetadata = structuredClone(answer.metadata);
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      render();

      try {
        const questions = Array.from(
          container.querySelectorAll("[data-agent-follow-up-question]"),
          (question) => question.textContent
        );
        expect(questions).toEqual(ordinaryQuestions);
        expect(container.textContent).not.toContain(expiryQuestion);
        expect(container.textContent).not.toContain(
          "How can I filter or sort these results?"
        );
        expect(
          store.getMessages(chat.id).find((message) => message.id === answer.id)
            ?.metadata
        ).toEqual(savedMetadata);
      } finally {
        unmount();
      }
    });

    test("does not invent a fallback when saved questions lack governance evidence", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      const answer = addQueryTurn(chat.id, "query-visible", false, [
        expiryQuestion,
        "How can I mask these columns?",
      ]);
      const savedMetadata = structuredClone(answer.metadata);
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      render();

      try {
        expect(container.querySelector("[data-agent-follow-up]")).toBeNull();
        expect(container.textContent).not.toContain(expiryQuestion);
        expect(container.textContent).not.toContain(
          "How can I filter or sort these results?"
        );
        expect(
          store.getMessages(chat.id).find((message) => message.id === answer.id)
            ?.metadata
        ).toEqual(savedMetadata);
      } finally {
        unmount();
      }
    });

    test("keeps later masking evidence out of an earlier visible query's follow-ups", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      const visibleAnswer = addQueryTurn(chat.id, "query-visible", false);
      const maskedAnswer = addQueryTurn(chat.id, "query-masked", true);
      const savedMetadata = [visibleAnswer, maskedAnswer].map((message) =>
        structuredClone(message.metadata)
      );
      const onContinueAction = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onContinueAction={onContinueAction} />
      );
      render();

      try {
        const followUps = container.querySelectorAll("[data-agent-follow-up]");
        expect(followUps).toHaveLength(2);
        expect(
          Array.from(
            followUps[0]!.querySelectorAll("[data-agent-follow-up-question]"),
            (question) => question.textContent
          )
        ).toEqual(ordinaryQuestions);
        expect(
          Array.from(
            followUps[1]!.querySelectorAll("[data-agent-follow-up-question]"),
            (question) => question.textContent
          )
        ).toEqual([...ordinaryQuestions, unmaskQuestion]);
        expect(unmaskQuestion).toMatch(/\?$/);
        expect(container.textContent).not.toContain(expiryQuestion);
        expect(container.textContent).not.toContain(
          "How can I filter or sort these results?"
        );
        const unmask = Array.from(
          followUps[1]!.querySelectorAll<HTMLButtonElement>(
            "[data-agent-action-continue]"
          )
        ).find((button) => button.textContent === unmaskQuestion);
        act(() => unmask?.click());
        expect(onContinueAction).toHaveBeenCalledWith(
          chat.id,
          unmaskQuestion,
          references,
          maskedAnswer.id
        );
        expect(
          [visibleAnswer, maskedAnswer].map(
            (answer) =>
              store.getMessages(chat.id).find((message) => message.id === answer.id)
                ?.metadata
          )
        ).toEqual(savedMetadata);
      } finally {
        unmount();
      }
    });

    test("binds an older answer's unmask question to its own source message", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      const contactsAnswer = addQueryTurn(chat.id, "contacts-query", true);
      const employeesAnswer = addQueryTurn(chat.id, "employees-query", true);
      const onContinueAction = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onContinueAction={onContinueAction} />
      );
      render();

      try {
        const followUps = container.querySelectorAll("[data-agent-follow-up]");
        expect(followUps).toHaveLength(2);
        const olderUnmask = Array.from(followUps[0]!.querySelectorAll<HTMLButtonElement>(
          "[data-agent-action-continue]"
        )).find((button) => button.textContent === unmaskQuestion);
        act(() => olderUnmask?.click());
        expect(onContinueAction).toHaveBeenCalledExactlyOnceWith(
          chat.id,
          unmaskQuestion,
          references,
          contactsAnswer.id
        );
        expect(contactsAnswer.id).not.toBe(employeesAnswer.id);
      } finally {
        unmount();
      }
    });

    test("renders each saved query's runtime masking eligibility without reading policies", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      const allowedAnswer = addQueryTurn(chat.id, "query-allowed", false, undefined, {
        database,
        allowed: true,
      });
      const deniedAnswer = addQueryTurn(chat.id, "query-denied", false, undefined, {
        database,
        allowed: false,
      });
      const savedMetadata = [allowedAnswer, deniedAnswer].map((message) =>
        structuredClone(message.metadata)
      );
      const onContinueAction = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onContinueAction={onContinueAction} />
      );
      render();

      try {
        const followUps = container.querySelectorAll("[data-agent-follow-up]");
        expect(followUps).toHaveLength(2);
        expect(Array.from(
          followUps[0]!.querySelectorAll("[data-agent-follow-up-question]"),
          (question) => question.textContent
        )).toEqual([...ordinaryQuestions, maskQuestion]);
        expect(Array.from(
          followUps[1]!.querySelectorAll("[data-agent-follow-up-question]"),
          (question) => question.textContent
        )).toEqual(ordinaryQuestions);
        const mask = Array.from(
          followUps[0]!.querySelectorAll<HTMLButtonElement>("[data-agent-action-continue]")
        ).find((button) => button.textContent === maskQuestion);
        act(() => mask?.click());
        expect(onContinueAction).toHaveBeenCalledWith(chat.id, maskQuestion, references, allowedAnswer.id);
        expect([allowedAnswer, deniedAnswer].map((answer) =>
          store.getMessages(chat.id).find((message) => message.id === answer.id)?.metadata
        )).toEqual(savedMetadata);
      } finally {
        unmount();
      }
    });
  });

  test.each(
    (["issue", "plan"] as const).flatMap((kind) =>
      (["actions", "handoff", "both"] as const).map((format) => ({ kind, format }))
    )
  )("hides redundant Issue navigation in a $kind diagnostic with $format metadata", ({ kind, format }) => {
    mocks.currentPath = "/projects/sample/issues/7?panel=checks#comments";
    mocks.resolvedPath = "/projects/sample/issues/7";
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/sample",
      capability: {
        kind: "diagnostic",
        resource: {
          kind,
          name: `projects/sample/${kind}s/7`,
          project: "projects/sample",
          label: "Selected problem",
        },
      },
    });
    const handoff = { kind: "open-issue" as const, issue: "projects/sample/issues/7" };
    const prompt = "How can I verify the corrected SQL?";
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "The diagnostic answer is still available.",
      metadata: {
        ...(format !== "actions" ? { handoff } : {}),
        actions: [
          ...(format !== "handoff" ? [{ kind: "open" as const, target: handoff }] : []),
          { kind: "continue", prompt },
        ],
      },
    });
    const metadata = structuredClone(store.getMessages(chat.id)[0].metadata);
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onContinueAction={onContinueAction} />
    );
    render();

    expect(container.querySelector("[data-agent-action-open]")).toBeNull();
    expect(container.querySelector("[data-agent-answer]")?.textContent).toContain(
      "The diagnostic answer is still available."
    );
    const followUp = container.querySelector<HTMLButtonElement>("[data-agent-action-continue]");
    expect(followUp?.textContent).toContain(prompt);
    act(() => followUp?.click());
    expect(onContinueAction).toHaveBeenCalledWith(chat.id, prompt, [], store.getMessages(chat.id)[0].id);
    expect(store.getMessages(chat.id)[0].metadata).toEqual(metadata);
    unmount();
  });

  test.each(["issue", "plan"] as const)("keeps navigation to another Issue in a %s diagnostic", (kind) => {
    mocks.currentPath = `/projects/sample/${kind}s/7`;
    mocks.resolvedPath = "/projects/sample/issues/8";
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/sample",
      capability: {
        kind: "diagnostic",
        resource: { kind, name: `projects/sample/${kind}s/7`, project: "projects/sample", label: "Selected problem" },
      },
    });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Inspect the related issue.",
      metadata: { actions: [{ kind: "open", target: { kind: "open-issue", issue: "projects/sample/issues/8" } }] },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-action-open]")?.getAttribute("href")).toBe(
      "/projects/sample/issues/8"
    );
    unmount();
  });

  test("restores the diagnostic Issue handoff after navigating away", () => {
    mocks.currentPath = "/projects/sample/issues/7";
    mocks.resolvedPath = "/projects/sample/issues/7";
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/sample",
      capability: {
        kind: "diagnostic",
        resource: { kind: "issue", name: "projects/sample/issues/7", project: "projects/sample", label: "Selected issue" },
      },
    });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Return to the issue.",
      metadata: { handoff: { kind: "open-issue", issue: "projects/sample/issues/7" } },
    });
    const metadata = structuredClone(store.getMessages(chat.id)[0].metadata);
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-action-open]")).toBeNull();
    act(() => {
      mocks.currentPath = "/projects/sample/plans/9";
      for (const listener of mocks.routeListeners) listener();
    });
    expect(container.querySelector("[data-agent-action-open]")?.getAttribute("href")).toBe(
      "/projects/sample/issues/7"
    );
    act(() => {
      mocks.currentPath = "/projects/sample/issues/7?panel=checks";
      for (const listener of mocks.routeListeners) listener();
    });
    expect(container.querySelector("[data-agent-action-open]")).toBeNull();
    expect(store.getMessages(chat.id)[0].metadata).toEqual(metadata);
    unmount();
  });

  test.each(["actions", "handoff"] as const)("keeps normal Issue navigation from %s metadata", (format) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({ context: "projects/sample" });
    const handoff = { kind: "open-issue" as const, issue: "projects/sample/issues/7" };
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Review this issue.",
      metadata: format === "actions" ? { actions: [{ kind: "open", target: handoff }] } : { handoff },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-action-open]")?.textContent).toContain(
      "agent.handoff.open-issue"
    );
    unmount();
  });

  test("keeps useful task navigation in a diagnostic chat", () => {
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/sample",
      capability: {
        kind: "diagnostic",
        resource: { kind: "plan", name: "projects/sample/plans/7", project: "projects/sample", label: "Selected Plan" },
      },
    });
    store.addMessage({
      chatId: chat.id,
      role: "assistant",
      content: "Inspect the failed task.",
      metadata: { actions: [{ kind: "open", target: { kind: "view-failed-task", task: "projects/sample/plans/7/rollout/stages/prod/tasks/t1" } }] },
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();

    expect(container.querySelector("[data-agent-action-open]")?.textContent).toContain(
      "agent.handoff.view-failed-task"
    );
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
    for (const choice of choices) {
      expect(choice).toHaveClass("h-auto", "px-2", "py-1.5", "text-xs", "whitespace-normal");
      expect(choice).not.toHaveClass("py-2", "text-sm");
    }
    act(() => choices[1]?.click());
    expect(onContinueAction).toHaveBeenCalledWith(
      chat.id,
      "How should I validate it?",
      [],
      store.getMessages(chat.id)[0].id
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

  test("keeps the follow-up question enabled without replacing the composer draft", () => {
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
    const draft = structuredClone(store.getDraft(chat.id));
    const onContinueAction = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <AgentChat onContinueAction={onContinueAction} />
    );
    render();

    try {
      const question = container.querySelector<HTMLButtonElement>(
        "[data-agent-action-continue]"
      );
      expect(question?.textContent).toContain("What is the impact?");
      expect(question?.disabled).toBe(false);
      act(() => question?.click());
      expect(onContinueAction).toHaveBeenCalledWith(chat.id, "What is the impact?", [], store.getMessages(chat.id)[0].id);
      expect(store.getDraft(chat.id)).toEqual(draft);
    } finally {
      unmount();
    }
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
    expect(link).not.toHaveClass("-ml-2");
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
    expect(button).not.toHaveClass("-ml-2");
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
  describe("landing without a selected chat", () => {
    test.each(["no chats", "all chats archived"] as const)(
      "shows existing workspace starters with %s without creating a chat",
      (state) => {
        useAgentStore.setState({ chats: [], currentChatId: null, messagesByChatId: {} });
        const store = useAgentStore.getState();
        if (state === "all chats archived") {
          const chat = store.createChat({ context: "projects/sample" });
          store.archiveChat(chat.id);
        }
        useAgentStore.setState({ currentChatId: null });
        const chats = useAgentStore.getState().chats;
        const messages = useAgentStore.getState().messagesByChatId;
        const onSelectSuggestion = vi.fn();
        const { container, render, unmount } = renderIntoContainer(
          <AgentChat
            contextLabel="Example workspace"
            onSelectSuggestion={onSelectSuggestion}
          />
        );
        render();
        try {
          expect(container.querySelector("[data-agent-empty-chat]")?.textContent)
            .toContain("Example workspace");
          expect(container.querySelector("h2")?.textContent)
            .toBe("agent.empty-chat.workspace-heading");
          expect(
            Array.from(container.querySelectorAll("[data-agent-empty-suggestion]"))
              .map((item) => item.getAttribute("data-starter-id"))
          ).toEqual([
            "workspace-attention",
            "workspace-recent",
            "workspace-governance",
          ]);
          act(() => {
            container.querySelector<HTMLButtonElement>(
              '[data-starter-id="workspace-attention"]'
            )?.click();
          });
          expect(onSelectSuggestion).toHaveBeenCalledExactlyOnceWith("workspace-attention");
          expect(useAgentStore.getState().chats).toEqual(chats);
          expect(useAgentStore.getState().currentChatId).toBeNull();
          expect(useAgentStore.getState().messagesByChatId).toEqual(messages);
        } finally {
          unmount();
        }
      }
    );

    test("uses explicit project context and its existing starters", () => {
      useAgentStore.setState({ chats: [], currentChatId: null });
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat
          context="projects/sample"
          contextLabel="Sample project"
          onSelectSuggestion={vi.fn()}
        />
      );
      render();
      try {
        expect(container.querySelector("[data-agent-empty-chat]")?.textContent)
          .toContain("Sample project");
        expect(container.querySelector("h2")?.textContent)
          .toBe("agent.empty-chat.project-heading");
        expect(
          Array.from(container.querySelectorAll("[data-agent-empty-suggestion]"))
            .map((item) => item.getAttribute("data-starter-id"))
        ).toEqual([
          "project-open-blockers",
          "project-open-recent",
          "project-propose-change",
          "project-next",
        ]);
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(useAgentStore.getState().currentChatId).toBeNull();
      } finally {
        unmount();
      }
    });

    test("keeps a selected unscoped chat in workspace scope despite the landing context", () => {
      const chat = useAgentStore.getState().createChat({ context: undefined });
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat context="projects/other" onSelectSuggestion={vi.fn()} />
      );
      render();
      try {
        expect(container.querySelector("h2")?.textContent)
          .toBe("agent.empty-chat.workspace-heading");
        expect(
          Array.from(container.querySelectorAll("[data-agent-empty-suggestion]"))
            .map((item) => item.getAttribute("data-starter-id"))
        ).toEqual([
          "workspace-attention",
          "workspace-recent",
          "workspace-governance",
        ]);
        expect(useAgentStore.getState().currentChatId).toBe(chat.id);
        expect(useAgentStore.getState().getChat(chat.id)?.context).toBeUndefined();
      } finally {
        unmount();
      }
    });

    test("keeps the landing but hides starters for an unsent landing draft", () => {
      useAgentStore.setState({
        chats: [],
        currentChatId: null,
        landingDraft: { text: "My unfinished question" },
      });
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onSelectSuggestion={vi.fn()} />
      );
      render();
      try {
        expect(container.querySelector("[data-agent-empty-chat]")).not.toBeNull();
        expect(container.querySelector("[data-agent-empty-suggestion]")).toBeNull();
        expect(useAgentStore.getState().landingDraft.text).toBe("My unfinished question");
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(useAgentStore.getState().currentChatId).toBeNull();
      } finally {
        unmount();
      }
    });

    test("does not infer a virtual landing without a suggestion callback", () => {
      useAgentStore.setState({ chats: [], currentChatId: null });
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      render();
      try {
        expect(container.querySelector("[data-agent-empty-chat]")).toBeNull();
        expect(container.querySelector("[data-agent-empty-suggestion]")).toBeNull();
        expect(useAgentStore.getState().chats).toEqual([]);
        expect(useAgentStore.getState().currentChatId).toBeNull();
      } finally {
        unmount();
      }
    });

    test("keeps an explicitly selected archived transcript instead of the landing", () => {
      const store = useAgentStore.getState();
      const chat = store.createChat({ context: "projects/sample" });
      store.addMessage({ chatId: chat.id, role: "user", content: "Archived question" });
      store.addMessage({ chatId: chat.id, role: "assistant", content: "Archived answer" });
      store.archiveChat(chat.id);
      useAgentStore.setState({ currentChatId: chat.id });
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onSelectSuggestion={vi.fn()} />
      );
      render();
      try {
        expect(container.textContent).toContain("Archived question");
        expect(container.textContent).toContain("Archived answer");
        expect(container.querySelector("[data-agent-empty-chat]")).toBeNull();
        expect(container.querySelector("[data-agent-empty-suggestion]")).toBeNull();
        expect(useAgentStore.getState().currentChatId).toBe(chat.id);
      } finally {
        unmount();
      }
    });
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
    for (const button of container.querySelectorAll("[data-agent-empty-suggestion]")) {
      expect(button).toHaveClass("h-auto", "px-2", "py-1.5", "text-xs", "whitespace-normal");
      expect(button).not.toHaveClass("py-2", "text-sm");
      expect(button.querySelector("svg")).toHaveClass("size-4");
    }

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

  describe("user-stopped managed run Retry", () => {
    const prepareStoppedRun = () => {
      const store = useAgentStore.getState();
      const chat = store.ensureCurrentChat();
      const user = store.addMessage({
        chatId: chat.id,
        role: "user",
        content: "Count databases",
      })!;
      store.startChatRun(chat.id, undefined, { runId: "run-1" });
      useAgentStore.setState((state) => ({
        chats: state.chats.map((current): AgentChatRecord => current.id === chat.id
          ? {
              ...current,
              runBudget: {
                version: 1,
                runId: "run-1",
                limit: 1000,
                totalTokensUsed: 100,
                hasUnreportedUsage: false,
                providerAttemptCount: 1,
                revision: 1,
              },
              runCheckpoint: {
                version: 1,
                runId: "run-1",
                suspension: "interrupted",
                rootUserMessageId: user.id,
                throughMessageId: user.id,
                bindings: {
                  page: { path: "/projects/sample", title: "Sample" },
                  references: [],
                  runMode: "default",
                },
                maskingSections: [],
                phase: "working",
                startupComplete: true,
              },
            }
          : current),
      }));
      store.interruptChatRun(chat.id, undefined, "user");
      return { store, chatId: chat.id };
    };

    test.each(["idle", "error"] as const)(
      "offers Retry for a recoverable stopped run in %s without replacing its budget",
      (status) => {
        const { store, chatId } = prepareStoppedRun();
        if (status === "error") {
          store.setChatStatus(chatId, status, {
            interrupted: true,
            interruptionReason: "user",
            lastError: "Request stopped",
          });
        }
        const runBeforeRetry = store.getChat(chatId);
        const onRetry = vi.fn();
        const { container, render, unmount } = renderIntoContainer(
          <AgentChat onRetry={onRetry} />
        );
        render();
        try {
          const result = container.querySelector(status === "error" ? '[role="alert"]' : "[data-agent-interruption-status]");
          expect(result?.textContent).toContain(status === "error" ? "Request stopped" : "agent.request-stopped-by-user");
          expect(container.querySelector("[data-agent-interruption-status]") === null).toBe(status === "error");
          const retry = Array.from(result?.querySelectorAll("button") ?? []).find(
            (button) => button.textContent === "agent.retry-last-chat-turn"
          );
          expect(retry).toBeDefined();
          act(() => retry?.click());
          expect(onRetry).toHaveBeenCalledOnce();
          expect(store.getChat(chatId)?.runId).toBe("run-1");
          expect(store.getChat(chatId)?.runBudget).toEqual(runBeforeRetry?.runBudget);
          expect(store.getMessages(chatId)).toHaveLength(1);
        } finally {
          unmount();
        }
      }
    );

    test.each([
      "missing checkpoint", "missing budget", "incomplete startup", "stale cursor",
      "different root", "mismatched budget", "token limit", "archived",
      "running", "awaiting_user", "paused", "queued", "remote busy",
    ] as const)("does not offer Retry for a %s stopped run", (state) => {
      const { store, chatId } = prepareStoppedRun();
      if (state === "different root") {
        store.addMessage({ chatId, role: "user", content: "A later request" });
        store.interruptChatRun(chatId, undefined, "user");
      }
      useAgentStore.setState((current) => ({
        chats: current.chats.map((chat): AgentChatRecord => {
          if (chat.id !== chatId) return chat;
          switch (state) {
            case "missing checkpoint": return { ...chat, runCheckpoint: undefined };
            case "missing budget": return { ...chat, runBudget: undefined };
            case "incomplete startup": return { ...chat, runCheckpoint: { ...chat.runCheckpoint!, startupComplete: false } };
            case "stale cursor": return { ...chat, runCheckpoint: { ...chat.runCheckpoint!, throughMessageId: "missing" } };
            case "mismatched budget": return { ...chat, runBudget: { ...chat.runBudget!, runId: "other-run" } };
            case "token limit": return { ...chat, runCheckpoint: { ...chat.runCheckpoint!, suspension: "token-limit" } };
            case "archived": return { ...chat, archived: true };
            case "running":
            case "awaiting_user":
            case "paused": return { ...chat, status: state };
            default: return chat;
          }
        }),
        ...(state === "queued" ? {
          queuedUserMessagesByChatId: {
            [chatId]: { id: "queued-message", chatId, text: "Queued request", page: { path: "/demo", title: "Demo" } },
          },
        } : {}),
      }));
      if (state === "remote busy") {
        useAgentStore.setState({ remoteBusyChatIds: { [chatId]: true } });
      }
      const onRetry = vi.fn();
      const { container, render, unmount } = renderIntoContainer(
        <AgentChat onRetry={onRetry} />
      );
      render();
      try {
        expect(container.querySelector("[data-agent-interruption-status]") === null).toBe(state === "token limit");
        expect(container.textContent).not.toContain("agent.retry-last-chat-turn");
        expect(onRetry).not.toHaveBeenCalled();
      } finally {
        unmount();
        useAgentStore.setState({ remoteBusyChatIds: {} });
      }
    });
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
      expect(retry?.parentElement).toHaveAttribute("data-agent-interruption-actions");
      expect(retry?.parentElement).toHaveClass("flex-wrap", "justify-end", "w-full");
      expect(retry).toHaveClass("max-w-full", "whitespace-normal", "wrap-anywhere");
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

  test("keeps AI setup visible without a dismiss action when ai_enabled is false", () => {
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
    expect(container.textContent).not.toContain("common.dismiss");
    expect(configureLink?.parentElement?.children).toHaveLength(1);
    act(() => configureLink?.click());
    expect(mocks.routerPush).toHaveBeenCalledWith(
      expect.objectContaining({
        name: SETTING_ROUTE_WORKSPACE_GENERAL,
        hash: "#ai-assistant",
      })
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

  test.each(["user", "assistant"] as const)(
    "wraps unbroken %s message text without changing its content",
    (role) => {
      const content =
        'Diagnostic snapshot: {"sha256":"354b7196c9ba5fb4b21cf615bb6ec4cd5c3c0c26c8f296b0f42b0f8a1d4e9abc"}';
      useAgentStore.getState().addMessage({ role, content });
      const { container, render, unmount } = renderIntoContainer(<AgentChat />);
      render();
      try {
        const bubble =
          role === "user"
            ? Array.from(container.querySelectorAll("div")).find((element) =>
                element.classList.contains("bg-info/10")
              )
            : container.querySelector("[data-agent-answer]");
        expect(bubble?.textContent).toBe(content);
        expect(bubble).toHaveClass("w-fit", "max-w-full", "wrap-anywhere");
        expect(bubble).not.toHaveClass("truncate", "overflow-hidden", "break-all");
      } finally {
        unmount();
      }
    }
  );

  test("keeps wide fenced SQL in its own horizontal scroller", () => {
    const sql = `SELECT '${"a".repeat(256)}';`;
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: `\`\`\`sql\n${sql}\n\`\`\``,
    });
    const { container, render, unmount } = renderIntoContainer(<AgentChat />);
    render();
    try {
      const pre = container.querySelector("[data-agent-answer] pre");
      expect(pre?.textContent).toBe(`${sql}\n`);
      expect(pre).toHaveClass("overflow-x-auto");
      expect(pre).not.toHaveClass("whitespace-normal", "whitespace-pre-wrap");
    } finally {
      unmount();
    }
  });

  test.each([
    {
      name: "multiline JSON",
      content: JSON.stringify({
        database: `instances/test-instance/databases/${"orders_".repeat(100)}`,
        policy: "x".repeat(1000),
      }, null, 2),
    },
    { name: "unbroken text", content: "x".repeat(2000) },
    {
      name: "canonical resource path",
      content: `Inspect instances/test-instance/databases/${"orders_".repeat(100)}`,
    },
    {
      name: "bound resource mention",
      content: `Inspect @${"orders_".repeat(100)}`,
      metadata: {
        resourceReference: {
          kind: "database",
          name: `instances/test-instance/databases/${"orders_".repeat(100)}`,
          project: "projects/demo",
          label: "orders_".repeat(100),
        },
      } satisfies AgentMessageMetadata,
    },
  ])("bounds $name in a user bubble without changing its text", ({ content, metadata }) => {
    const message = useAgentStore.getState().addMessage({
      role: "user",
      content,
      metadata,
    });
    const { container, render, unmount } = renderIntoContainer(
      <div style={{ width: 280 }}><AgentChat /></div>
    );
    try {
      render();
      const bubble = Array.from(container.querySelectorAll("div")).find(
        (element) => element.classList.contains("bg-info/10")
      );
      expect(bubble?.textContent).toBe(content);
      expect(bubble).toHaveClass(
        "w-fit", "min-w-0", "max-w-full", "wrap-anywhere", "whitespace-pre-wrap"
      );
      if (metadata) {
        expect(bubble?.querySelector("[data-agent-message-mention]")?.textContent).toBe(
          `@${metadata.resourceReference.label}`
        );
      }
      expect(useAgentStore.getState().getMessages(
        useAgentStore.getState().currentChatId
      ).find((current) => current.id === message?.id)?.content).toBe(content);
    } finally {
      unmount();
    }
  });

  test("wraps long assistant prose within its content-sized bubble", () => {
    const content = `Resource ${"orders_".repeat(200)} is unavailable.`;
    useAgentStore.getState().addMessage({ role: "assistant", content });
    const { container, render, unmount } = renderIntoContainer(
      <div style={{ width: 280 }}><AgentChat /></div>
    );
    try {
      render();
      const bubble = container.querySelector("[data-agent-answer]");
      expect(bubble?.textContent).toBe(content);
      expect(bubble).toHaveClass("w-fit", "min-w-0", "max-w-full", "wrap-anywhere");
    } finally {
      unmount();
    }
  });

  test("keeps long fenced code unwrapped in its own horizontal scroll area", () => {
    const sql = `SELECT '${"x".repeat(2000)}' AS value;`;
    useAgentStore.getState().addMessage({
      role: "assistant",
      content: `Review the statement:\n\n\`\`\`sql\n${sql}\n\`\`\``,
    });
    const { container, render, unmount } = renderIntoContainer(
      <div style={{ width: 280 }}><AgentChat /></div>
    );
    try {
      render();
      const bubble = container.querySelector("[data-agent-answer]");
      const pre = bubble?.querySelector("pre");
      expect(bubble).toHaveClass("min-w-0", "max-w-full", "wrap-anywhere");
      expect(pre?.textContent).toBe(`${sql}\n`);
      expect(pre).toHaveClass("overflow-x-auto", "whitespace-pre");
      expect(pre).not.toHaveClass("whitespace-pre-wrap", "break-all");
    } finally {
      unmount();
    }
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
