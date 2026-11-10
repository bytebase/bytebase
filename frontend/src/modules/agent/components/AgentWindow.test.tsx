import { fireEvent, screen, within } from "@testing-library/react";
import type { ReactElement } from "react";
import { act, cloneElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import enUS from "@/locales/en-US.json";
import { State } from "@/types/proto-es/v1/common_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { createAgentStore, useAgentStore } from "../store/agent";
import { DEFAULT_AGENT_PREFERENCES, getAgentPreferencesKey } from "../store/preferences";
import type { AgentPageResource } from "../logic/chatContext";
import { openAgentPageHelp } from "../logic/contextualHelp";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  })),
  getOrFetchSettingByName: vi.fn(),
  aiEnabled: undefined as boolean | undefined,
  batchFetchProjects: vi.fn(),
  projectErrorsByName: {} as Record<string, Error | undefined>,
  currentContext: "projects/orders" as string | undefined,
  pageResource: undefined as AgentPageResource | undefined,
  getOrFetchDatabaseByName: vi.fn(),
  getIssue: vi.fn(),
  getPlan: vi.fn(),
  canListReferences: true,
  planPolicyKnown: true,
  planPermissions: {
    workspaceList: true,
    workspaceGet: true,
    projectList: true,
    projectGet: true,
  },
  projectsByName: {
    "projects/orders": { title: "Order service" },
    "projects/analytics": { title: "Analytics" },
  } as Record<string, { title: string }>,
  workspace: { name: "workspaces/acme", title: "Acme workspace" },
  currentUser: {
    name: "users/first@example.com",
    email: "first@example.com",
    workspace: "workspaces/acme",
  } as
    | { name: string; email: string; workspace: string }
    | undefined,
}));

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

vi.mock("@/stores/app", () => {
  const state = {
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    get serverInfo() {
      return { aiEnabled: mocks.aiEnabled };
    },
    batchFetchProjects: mocks.batchFetchProjects,
    get projectsByName() {
      return mocks.projectsByName;
    },
    get projectErrorsByName() {
      return mocks.projectErrorsByName;
    },
    get projectPoliciesByName() {
      return mocks.planPolicyKnown ? { "projects/orders": {} } : {};
    },
    get currentUser() {
      return mocks.currentUser;
    },
    roles: [{}],
    workspacePolicy: {},
    getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
    hasWorkspacePermission: (permission: string) =>
      permission === "bb.plans.list"
        ? mocks.planPermissions.workspaceList
        : permission === "bb.plans.get"
          ? mocks.planPermissions.workspaceGet
          : mocks.canListReferences,
    hasProjectPermission: (_project: unknown, permission: string) =>
      permission === "bb.plans.list"
        ? mocks.planPermissions.projectList
        : permission === "bb.plans.get"
          ? mocks.planPermissions.projectGet
          : mocks.canListReferences,
    get workspace() {
      return mocks.workspace;
    },
  };
  const useAppStore = <T,>(selector: (value: typeof state) => T) =>
    selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("../logic/chatContext", () => ({
  getCurrentChatContext: () => mocks.currentContext,
  getCurrentAgentPageResource: () => mocks.pageResource,
}));

vi.mock("@/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api")>();
  return {
    ...actual,
    issueServiceClientConnect: {
      ...actual.issueServiceClientConnect,
      getIssue: mocks.getIssue,
    },
    planServiceClientConnect: {
      ...actual.planServiceClientConnect,
      getPlan: mocks.getPlan,
    },
  };
});

vi.mock("./AgentChat", () => ({
  AgentChat: ({
    contextLabel,
    onSelectSuggestion,
    onContinueAction,
  }: {
    contextLabel?: string;
    onSelectSuggestion?: (starterId: "project-open-blockers") => void;
    onContinueAction?: (
      chatId: string,
      prompt: string,
      references: { kind: "project"; name: string; project: string; label: string }[]
    ) => void;
  }) => (
    <div data-testid="agent-chat" data-context-label={contextLabel}>
      <button
        type="button"
        data-testid="agent-empty-suggestion"
        onClick={() => onSelectSuggestion?.("project-open-blockers")}
      />
      <button
        type="button"
        data-testid="agent-continue-action"
        onClick={() =>
          onContinueAction?.(
            useAgentStore.getState().currentChatId ?? "",
            "What should I review next?",
            []
          )
        }
      />
      <button
        type="button"
        data-testid="agent-continue-with-project"
        onClick={() =>
          onContinueAction?.(
            useAgentStore.getState().currentChatId ?? "",
            "What should I review next?",
            [
              {
                kind: "project",
                name: "projects/orders",
                project: "projects/orders",
                label: "Orders",
              },
            ]
          )
        }
      />
    </div>
  ),
}));

vi.mock("./AgentInput", () => ({
  AgentInput: () => <div data-testid="agent-input" />,
}));

let AgentWindow: typeof import("./AgentWindow").AgentWindow;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(cloneElement(element));
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const getCurrentQueuedUserMessage = () => {
  const store = useAgentStore.getState();
  return store.currentChatId
    ? store.queuedUserMessagesByChatId[store.currentChatId]
    : undefined;
};

const openChatList = async (width: number, archived = false) => {
  if (width < 480 || (!archived && width < 750)) {
    await act(async () => {
      screen.getByRole("button", { name: "agent.chat-list-label" }).click();
      await Promise.resolve();
    });
  }
  if (archived) {
    await act(async () => {
      screen.getByRole("button", { name: "agent.archived-only-chats" }).click();
      await Promise.resolve();
    });
  }
};

const openChatMenu = async (chatId: string) => {
  const trigger = document.body.querySelector(
    `[data-agent-chat-row="${chatId}"] [data-agent-chat-menu]`
  ) as HTMLButtonElement;
  expect(trigger).toBeInstanceOf(HTMLButtonElement);
  await act(async () => {
    trigger.click();
    await Promise.resolve();
  });
  return trigger;
};

const beginChatRename = async (chatId: string) => {
  await openChatMenu(chatId);
  const rename = getLayerRoot("agent").querySelector(
    "[data-agent-rename-chat]"
  ) as HTMLElement;
  expect(rename).toHaveAttribute("role", "menuitem");
  await act(async () => {
    rename.click();
    await Promise.resolve();
  });
  const input = document.body.querySelector(
    `[data-agent-chat-row="${chatId}"] [data-agent-chat-title-input]`
  ) as HTMLInputElement;
  expect(input).toBeInstanceOf(HTMLInputElement);
  return input;
};

const selectChatMenuAction = async (chatId: string, selector: string) => {
  await openChatMenu(chatId);
  const action = getLayerRoot("agent").querySelector(selector) as HTMLElement;
  expect(action).toHaveAttribute("role", "menuitem");
  await act(async () => {
    action.click();
    await Promise.resolve();
  });
};

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      disconnect() {}
    }
  );
  vi.stubGlobal("PointerEvent", MouseEvent);
  vi.stubGlobal("innerWidth", 1600);
  vi.stubGlobal("innerHeight", 1000);
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: true,
      media: "",
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });

  const initialState = createAgentStore().getState();
  useAgentStore.setState({
    visible: initialState.visible,
    size: initialState.size,
    sidebarWidth: initialState.sidebarWidth,
    minimized: initialState.minimized,
    workspaceName: initialState.workspaceName,
    userEmail: initialState.userEmail,
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
    queuedUserMessagesByChatId: {},
    queuedUserMessageInFlightByChatId: {},
  });
  useAgentStore
    .getState()
    .setWorkspaceName(mocks.workspace.name, "first@example.com");
  useAgentStore.setState({
    visible: true,
    minimized: false,
    size: { width: 1200, height: 450 },
    sidebarWidth: 200,
  });

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  });
  mocks.getOrFetchSettingByName.mockReset();
  mocks.getOrFetchSettingByName.mockResolvedValue(undefined);
  mocks.aiEnabled = true;
  mocks.batchFetchProjects.mockReset();
  mocks.batchFetchProjects.mockResolvedValue([]);
  mocks.projectsByName = {
    "projects/orders": { title: "Order service" },
    "projects/analytics": { title: "Analytics" },
  };
  mocks.projectErrorsByName = {};
  mocks.currentContext = "projects/orders";
  mocks.pageResource = undefined;
  mocks.currentUser = {
    name: "users/first@example.com",
    email: "first@example.com",
    workspace: "workspaces/acme",
  };
  mocks.getOrFetchDatabaseByName.mockReset();
  mocks.getIssue.mockReset();
  mocks.getPlan.mockReset();
  mocks.canListReferences = true;
  mocks.planPolicyKnown = true;
  mocks.planPermissions = {
    workspaceList: true,
    workspaceGet: true,
    projectList: true,
    projectGet: true,
  };

  ({ AgentWindow } = await import("./AgentWindow"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  document.body.innerHTML = "";
});

test("opens preferences without unmounting the conversation and restores trigger focus", () => {
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    const input = document.body.querySelector('[data-testid="agent-input"]');
    const chatId = useAgentStore.getState().currentChatId;
    const trigger = screen.getByRole("button", { name: "agent.preferences.title" });
    act(() => trigger.click());

    expect(screen.getByRole("heading", { name: "agent.preferences.title" })).toBeInTheDocument();
    const header = document.body.querySelector<HTMLElement>("[data-agent-window-header]")!;
    expect(within(header).getByRole("heading", { name: "agent.assistant-title" })).toBeInTheDocument();
    expect(within(screen.getByRole("region", { name: "agent.preferences.title" })).getByRole("heading", { name: "agent.preferences.title" })).toBeInTheDocument();
    expect(trigger).not.toBeVisible();
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBe(input);
    expect(input?.closest("[inert]")).not.toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chatId);

    act(() => screen.getByRole("button", { name: "agent.preferences.back" }).click());
    expect(document.body.querySelector("[data-agent-preferences]")).toBeNull();
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBe(input);
    expect(trigger).toHaveFocus();
  } finally {
    unmount();
  }
});

test("keeps the preference description visible beside the body heading", () => {
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    act(() => screen.getByRole("button", { name: "agent.preferences.title" }).click());
    const preferences = screen.getByRole("region", { name: "agent.preferences.title" });
    expect(within(preferences).getByRole("heading", { name: "agent.preferences.title" })).toBeVisible();
    expect(within(preferences).getByText("agent.preferences.scope")).toBeVisible();
    expect(screen.queryByRole("button", { name: "agent.preferences.storage" })).toBeNull();
  } finally {
    unmount();
  }
});

test("continues the last eligible contextual chat on initial entry when preferred", () => {
  const store = useAgentStore.getState();
  const chat = store.createChat({
    context: "projects/orders", workspaceName: mocks.workspace.name,
    page: { path: "/projects/orders", title: "Orders" },
  });
  store.setDraft(chat.id, { text: "Keep this draft" });
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  useAgentStore.setState({
    preferences: { sendWithEnter: true, tokenLimitPerRun: 60000, openingBehavior: "continue" },
  });
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe("Keep this draft");
  } finally {
    unmount();
  }
});

test("uses persisted opening preferences on cold owner hydration", () => {
  const chat = useAgentStore.getState().createChat({
    context: "projects/orders", workspaceName: mocks.workspace.name,
    page: { path: "/projects/orders", title: "Orders" },
  });
  const key = getAgentPreferencesKey(mocks.workspace.name, "first@example.com");
  localStorage.setItem(key!, JSON.stringify({ ...DEFAULT_AGENT_PREFERENCES, openingBehavior: "continue" }));
  useAgentStore.setState({ workspaceName: null, userEmail: null, preferences: { ...DEFAULT_AGENT_PREFERENCES } });
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
  } finally {
    unmount();
  }
});

test("keeps a cold-hydrated queued request ahead of the opening preference", () => {
  const store = useAgentStore.getState();
  const chat = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
  store.queueUserMessage({ chatId: chat.id, text: "Queued question", page: { path: "/projects/orders", title: "Orders" } });
  store.saveState();
  useAgentStore.setState({ workspaceName: null, userEmail: null, chats: [], currentChatId: null, queuedUserMessagesByChatId: {} });
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
    expect(useAgentStore.getState().queuedUserMessagesByChatId[chat.id]?.text).toBe("Queued question");
  } finally {
    unmount();
  }
});

test("applies the next owner's opening preference while Agent remains open", () => {
  const second = createAgentStore();
  second.getState().setWorkspaceName(mocks.workspace.name, "second@example.com");
  const chat = second.getState().createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
  second.getState().updatePreferences({ openingBehavior: "continue" });
  second.getState().saveState();
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInTheDocument();
    mocks.currentUser = { name: "users/second@example.com", email: "second@example.com", workspace: mocks.workspace.name };
    render();
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
  } finally {
    unmount();
  }
});

test("applies a changed opening choice on the next open without disrupting the current view", () => {
  const store = useAgentStore.getState();
  const chat = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    act(() => store.updatePreferences({ openingBehavior: "continue" }));
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInTheDocument();
    act(() => store.toggle());
    act(() => store.open());
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
    act(() => store.updatePreferences({ openingBehavior: "landing" }));
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    act(() => store.toggle());
    act(() => store.open());
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInTheDocument();
  } finally {
    unmount();
  }
});

test("preserves a minimized view when the opening preference has changed", () => {
  const store = useAgentStore.getState();
  store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    act(() => store.updatePreferences({ openingBehavior: "continue" }));
    act(() => store.minimize());
    act(() => store.restore());
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInTheDocument();
    act(() => store.toggle());
    act(() => store.open());
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
  } finally {
    unmount();
  }
});

test("falls back to the landing view when the preferred last chat is unavailable", () => {
  mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
  useAgentStore.setState({
    preferences: { sendWithEnter: true, tokenLimitPerRun: 60000, openingBehavior: "continue" },
    chats: [], currentChatId: null,
  });
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    expect(document.body.querySelector('[data-agent-contextual-start="project"]')).toBeInTheDocument();
    expect(useAgentStore.getState().queuedUserMessagesByChatId).toEqual({});
  } finally {
    unmount();
  }
});

test("shows an empty start on general pages without discarding the previous discussion", () => {
  const store = useAgentStore.getState();
  const chat = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
  store.addMessage({ chatId: chat.id, role: "user", content: "Previous discussion" });
  const { render, unmount } = renderIntoContainer(<AgentWindow />);
  try {
    render();
    const state = useAgentStore.getState();
    expect(state.currentChatId).not.toBe(chat.id);
    expect(state.getMessages()).toEqual([]);
    expect(state.getMessages(chat.id).at(-1)?.content).toBe("Previous discussion");
  } finally {
    unmount();
  }
});

describe("AgentWindow", () => {
  test.each([
    { width: 1200, action: "change" },
    { width: 390, action: "change" },
    { width: 1200, action: "reset" },
    { width: 390, action: "reset" },
  ])("keeps preferences open after $action with current page help at $width px", async ({ width, action }) => {
    vi.stubGlobal("innerWidth", width);
    useAgentStore.setState({ size: { width, height: 600 } });
    const store = useAgentStore.getState();
    store.updatePreferences({ openingBehavior: action === "reset" ? "continue" : "landing" });
    const chat = store.createChat({ context: mocks.workspace.name, workspaceName: mocks.workspace.name });
    store.setDraft(chat.id, { text: "Keep this draft" });
    store.addMessage({ chatId: chat.id, role: "user", content: "Existing discussion" });
    const onStart = vi.fn(() => true);
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" }, maskingRuleEditor: true, onStart }); });
      const chatCount = useAgentStore.getState().chats.length;
      act(() => screen.getByRole("button", { name: "agent.preferences.title" }).click());

      if (action === "change") {
        act(() => screen.getByRole("radio", { name: "agent.preferences.continue-last-chat" }).click());
      } else {
        await act(async () => screen.getByRole("button", { name: "agent.preferences.reset" }).click());
        const confirmation = screen.getByRole("dialog", { name: "agent.preferences.reset-confirmation" });
        await act(async () => within(confirmation).getByRole("button", { name: "common.reset" }).click());
      }

      expect(screen.getByRole("region", { name: "agent.preferences.title" })).toBeVisible();
      expect(screen.getByRole("radio", { name: action === "change" ? "agent.preferences.continue-last-chat" : "agent.preferences.show-landing" })).toBeChecked();
      if (action === "reset") {
        expect(useAgentStore.getState().preferences).toEqual(DEFAULT_AGENT_PREFERENCES);
      }
      act(() => screen.getByRole("button", { name: "agent.preferences.back" }).click());
      expect(document.body.querySelector('[data-agent-contextual-start="global-masking"]')).toBeVisible();
      expect(useAgentStore.getState().currentChatId).toBe(chat.id);
      expect(useAgentStore.getState().chats).toHaveLength(chatCount);
      expect(store.getDraft(chat.id).text).toBe("Keep this draft");
      expect(store.getMessages(chat.id)).toHaveLength(1);
      expect(onStart).not.toHaveBeenCalled();
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
    } finally {
      act(() => revoke());
      unmount();
    }
  });

  test.each(["reopen", "new help"])("dismisses preferences on %s while page help is current", (action) => {
    const store = useAgentStore.getState();
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" } }); });
      act(() => screen.getByRole("button", { name: "agent.preferences.title" }).click());
      expect(screen.getByRole("region", { name: "agent.preferences.title" })).toBeVisible();

      if (action === "reopen") {
        act(() => store.toggle());
        act(() => store.open());
      } else {
        act(() => { revoke = openAgentPageHelp({ page: { kind: "semantic-types" } }); });
      }

      expect(screen.queryByRole("region", { name: "agent.preferences.title" })).toBeNull();
      expect(document.body.querySelector(`[data-agent-contextual-start="${action === "reopen" ? "global-masking" : "semantic-types"}"]`)).toBeVisible();
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
    } finally {
      act(() => revoke());
      unmount();
    }
  });

  test.each([false, true])("opens editor help without submitting and dismisses only when started=%s", (started) => {
    const store = useAgentStore.getState();
    const owner = store.createChat({ context: mocks.workspace.name, workspaceName: mocks.workspace.name });
    store.addMessage({ chatId: owner.id, role: "user", content: "Existing rule discussion" });
    const onStart = vi.fn(() => started);
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      const chatCount = useAgentStore.getState().chats.length;
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" }, maskingRuleEditor: true, onStart }); });
      expect(document.body.querySelector('[data-agent-contextual-start="global-masking"]')).toBeInstanceOf(HTMLElement);
      const suggestions = [...document.body.querySelectorAll<HTMLButtonElement>("[data-agent-contextual-suggestion]")];
      expect(suggestions.map((button) => button.textContent)).toEqual([
        "agent.contextual-start.masking-rule-configure",
        "agent.contextual-start.masking-rule-effect",
        "agent.contextual-start.masking-rule-review",
      ]);
      expect(document.body.querySelector("[data-agent-contextual-continue]")).toBeNull();
      expect(onStart).not.toHaveBeenCalled();
      expect(useAgentStore.getState().chats).toHaveLength(chatCount);
      expect(store.getMessages(owner.id)).toHaveLength(1);
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
      act(() => suggestions[0].click());
      expect(onStart).toHaveBeenCalledWith("agent.contextual-start.masking-rule-configure");
      expect(Boolean(document.body.querySelector("[data-agent-contextual-start]"))).toBe(!started);
      expect(store.getMessages(owner.id)).toHaveLength(1);
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
    } finally {
      act(() => revoke());
      unmount();
    }
  });

  test("updates busy editor help starters when its source becomes ready", () => {
    let ready = false;
    const readinessListeners = new Set<() => void>();
    const onStart = vi.fn(() => true);
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" }, maskingRuleEditor: true, onStart, canStart: () => ready, subscribeCanStart: (listener) => {
        readinessListeners.add(listener);
        return () => readinessListeners.delete(listener);
      } }); });
      let starter = document.body.querySelector<HTMLButtonElement>("[data-agent-contextual-suggestion]");
      expect(starter?.disabled).toBe(true);
      act(() => starter?.click());
      expect(onStart).not.toHaveBeenCalled();
      act(() => {
        ready = true;
        for (const listener of readinessListeners) listener();
      });
      starter = document.body.querySelector<HTMLButtonElement>("[data-agent-contextual-suggestion]");
      expect(starter?.disabled).toBe(false);
      ready = false;
      act(() => starter?.click());
      expect(onStart).not.toHaveBeenCalled();
      act(() => {
        for (const listener of readinessListeners) listener();
      });
      expect(starter?.disabled).toBe(true);
      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInstanceOf(HTMLElement);
    } finally {
      act(() => revoke());
      expect(readinessListeners.size).toBe(0);
      unmount();
    }
  });

  test.each(["source", "member"] as const)("does not start stale editor help after its %s changes", (change) => {
    let available = true;
    const onStart = vi.fn(() => true);
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" }, maskingRuleEditor: true, onStart, isAvailable: () => available }); });
      const starter = document.body.querySelector<HTMLButtonElement>("[data-agent-contextual-suggestion]");
      if (change === "source") available = false;
      else mocks.currentUser = { name: "users/second@example.com", email: "second@example.com", workspace: mocks.workspace.name };
      act(() => starter?.click());
      expect(onStart).not.toHaveBeenCalled();
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
      act(() => revoke());
      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    } finally {
      act(() => revoke());
      unmount();
    }
  });

  test("keeps explicitly opened list-page help in its captured workspace after navigation", () => {
    mocks.currentContext = mocks.workspace.name;
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    let revoke = () => {};
    try {
      render();
      const count = useAgentStore.getState().chats.length;
      act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" } }); });
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
      expect(useAgentStore.getState().chats).toHaveLength(count);
      const starter = document.body.querySelector<HTMLButtonElement>("[data-agent-contextual-suggestion]");
      expect(starter?.textContent).toBe("agent.contextual-start.masking-global-effect");
      mocks.currentContext = "projects/analytics";
      mocks.pageResource = { kind: "project", name: "projects/analytics", project: "projects/analytics" };
      act(() => starter?.click());
      expect(getCurrentQueuedUserMessage()).toMatchObject({ text: "agent.contextual-start.masking-global-effect", runMode: "read-only" });
      expect(useAgentStore.getState().chats).toHaveLength(count + 1);
      expect(useAgentStore.getState().getChat(useAgentStore.getState().currentChatId)?.context).toBe(mocks.workspace.name);
      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    } finally {
      act(() => revoke());
      unmount();
    }
  });

  test("uses the header bot icon for the minimized launcher", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => useAgentStore.getState().minimize());

    expect(
      document.body.querySelector("[data-agent-window] .lucide-bot")
    ).toBeInstanceOf(SVGElement);
    unmount();
  });

  test.each([
    { width: 1600, height: 1000, footerHeight: 72 },
    { width: 390, height: 844, footerHeight: 120 },
  ])(
    "keeps the minimized launcher above the sticky action footer at $width px",
    ({ width, height, footerHeight }) => {
      vi.stubGlobal("innerWidth", width);
      vi.stubGlobal("innerHeight", height);
      const footer = document.createElement("footer");
      footer.dataset.slot = "sticky-action-footer";
      document.body.appendChild(footer);
      const rectSpy = vi
        .spyOn(Element.prototype, "getBoundingClientRect")
        .mockImplementation(function (this: Element) {
          if (this === footer) {
            return new DOMRect(0, height - footerHeight, width, footerHeight);
          }
          if (
            this instanceof HTMLButtonElement &&
            this.hasAttribute("data-agent-window")
          ) {
            const bottom = Number.parseFloat(this.style.bottom) || 16;
            return new DOMRect(width - 56, height - bottom - 40, 40, 40);
          }
          return new DOMRect();
        });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        act(() => useAgentStore.getState().minimize());
        const launcher = document.body.querySelector<HTMLButtonElement>(
          "button[data-agent-window]"
        );
        expect(launcher).toBeInstanceOf(HTMLButtonElement);
        expect(launcher?.parentElement).toBe(getLayerRoot("agent"));
        expect(launcher!.getBoundingClientRect().bottom).toBeLessThanOrEqual(
          height - footerHeight - 16
        );
      } finally {
        unmount();
        rectSpy.mockRestore();
        footer.remove();
      }
    }
  );

  test("opens the selected chat from the minimized Agent button", () => {
    const store = useAgentStore.getState();
    const chatId = store.currentChatId;
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      act(() => store.minimize());
      const launcher = document.body.querySelector(
        'button[data-agent-window][aria-label="agent.self"]'
      ) as HTMLButtonElement;
      expect(launcher).toBeInstanceOf(HTMLButtonElement);
      act(() => launcher.click());

      expect(useAgentStore.getState().visible).toBe(true);
      expect(useAgentStore.getState().minimized).toBe(false);
      expect(useAgentStore.getState().currentChatId).toBe(chatId);
      expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);
    } finally {
      unmount();
    }
  });

  test.each([
    { width: 1600, height: 1000 },
    { width: 390, height: 844 },
  ])(
    "remembers the dragged launcher position across reopening and minimizing at $width px",
    ({ width, height }) => {
      vi.stubGlobal("innerWidth", width);
      vi.stubGlobal("innerHeight", height);
      const rectSpy = vi
        .spyOn(Element.prototype, "getBoundingClientRect")
        .mockImplementation(function (this: Element) {
          if (
            this instanceof HTMLButtonElement &&
            this.hasAttribute("data-agent-window")
          ) {
            const right = Number.parseFloat(this.style.right) || 16;
            const bottom = Number.parseFloat(this.style.bottom) || 16;
            return new DOMRect(
              window.innerWidth - right - 40,
              window.innerHeight - bottom - 40,
              40,
              40
            );
          }
          return new DOMRect();
        });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      const getLauncher = () =>
        document.body.querySelector<HTMLButtonElement>(
          "button[data-agent-window]"
        )!;
      try {
        render();
        act(() => useAgentStore.getState().minimize());
        const launcher = getLauncher();
        launcher.setPointerCapture = vi.fn();
        launcher.releasePointerCapture = vi.fn();
        const pointer = (type: string, x: number, y: number) => {
          const event = new MouseEvent(type, {
            bubbles: true,
            button: 0,
            clientX: x,
            clientY: y,
          });
          Object.defineProperties(event, {
            pointerId: { value: 1 },
            isPrimary: { value: true },
          });
          act(() => launcher.dispatchEvent(event));
        };
        pointer("pointerdown", width - 36, height - 36);
        pointer("pointermove", width - 136, height - 136);
        pointer("pointerup", width - 136, height - 136);
        expect(launcher.style.right).toBe("116px");
        expect(launcher.style.bottom).toBe("116px");

        for (let i = 0; i < 2; i++) {
          act(() => getLauncher().click());
          const chatWindow = document.body.querySelector<HTMLElement>(
            "div[data-agent-window]"
          )!;
          expect(chatWindow.style.right).toBe("16px");
          expect(chatWindow.style.bottom).toBe("16px");
          act(() => useAgentStore.getState().minimize());
          expect(getLauncher().style.right).toBe("116px");
          expect(getLauncher().style.bottom).toBe("116px");
        }
      } finally {
        unmount();
        rectSpy.mockRestore();
      }
    }
  );

  test("hides the Agent window while the authenticated member is unavailable", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();
    expect(document.body.querySelector("[data-agent-window]")).toBeInstanceOf(
      HTMLElement
    );

    mocks.currentUser = undefined;
    render();
    expect(document.body.querySelector("[data-agent-window]")).toBeNull();
    expect(useAgentStore.getState().workspaceName).toBeNull();
    unmount();
  });

  test("switches visible chats when another member signs in to the same workspace", () => {
    const store = useAgentStore.getState();
    const firstChatId = store.currentChatId!;
    store.addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Private first-member question",
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    mocks.currentUser = {
      name: "users/second@example.com",
      email: "second@example.com",
      workspace: mocks.workspace.name,
    };
    render();
    expect(useAgentStore.getState().userEmail).toBe("second@example.com");
    expect(useAgentStore.getState().getChat(firstChatId)).toBeNull();
    expect(document.body.querySelector("[data-agent-window]")).toBeInstanceOf(
      HTMLElement
    );
    unmount();
  });

  test("shows a chat queued from the page composer instead of the contextual start", () => {
    const store = useAgentStore.getState();
    const page = { path: "/projects/orders", title: "Order service" };
    const chat = store.selectOrCreateChatForContext(
      "projects/orders",
      page,
      mocks.workspace.name
    );
    expect(chat).not.toBeNull();
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    useAgentStore.setState({ visible: false });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      store.queueUserMessage({
        chatId: chat!.id,
        text: "What should I do next?",
        page,
      });
      store.open();
    });

    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(document.body.querySelector('[data-testid="agent-chat"]')).toBeInstanceOf(HTMLElement);
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);
    expect(useAgentStore.getState().currentChatId).toBe(chat!.id);
    unmount();
  });

  test("keeps the new member's selected chat after switching members while open", () => {
    mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      mocks.currentUser = {
        name: "users/second@example.com",
        email: "second@example.com",
        workspace: mocks.workspace.name,
      };
      render();
      let chatId = "";
      act(() => {
        const store = useAgentStore.getState();
        const chat = store.createChat({ context: "projects/analytics", workspaceName: mocks.workspace.name });
        chatId = chat.id;
        store.addMessage({ chatId, role: "user", content: "Second member's question" });
      });
      act(() => {
        (document.body.querySelector(`[data-agent-chat-row="${chatId}"] button`) as HTMLButtonElement).click();
      });
      act(() => useAgentStore.getState().toggle());
      act(() => useAgentStore.getState().open());

      expect(useAgentStore.getState().userEmail).toBe("second@example.com");
      expect(useAgentStore.getState().currentChatId).toBe(chatId);
      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
      expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);
    } finally {
      unmount();
    }
  });

  test.each([
    { width: 1200, action: "close" },
    { width: 700, action: "close" },
    { width: 390, action: "close" },
    { width: 1200, action: "minimize" },
    { width: 700, action: "minimize" },
    { width: 390, action: "minimize" },
  ])("keeps the selected conversation after $action at $width px", ({ width, action }) => {
    const store = useAgentStore.getState();
    const chat = store.createChat({
      context: "projects/orders",
      workspaceName: mocks.workspace.name,
    });
    store.addMessage({ chatId: chat.id, role: "user", content: "Earlier question" });
    store.setDraft(chat.id, { text: "Unsent follow-up" });
    useAgentStore.setState({ size: { width, height: 450 } });
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      act(() => {
        (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click();
      });
      const before = store.getChat(chat.id);
      act(() => {
        if (action === "close") store.toggle();
        else store.minimize();
      });
      mocks.currentContext = "projects/analytics";
      mocks.pageResource = {
        kind: "project",
        name: "projects/analytics",
        project: "projects/analytics",
      };
      act(() => {
        if (action === "close") store.open();
        else store.restore();
      });

      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
      expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);
      expect(useAgentStore.getState().currentChatId).toBe(chat.id);
      expect(store.getChat(chat.id)).toEqual(before);
      expect(store.getDraft(chat.id).text).toBe("Unsent follow-up");
      if (width === 390) {
        expect(document.body.querySelector("[data-agent-mobile-conversation]")).toBeInstanceOf(HTMLElement);
      }
    } finally {
      unmount();
    }
  });

  test.each([1200, 700, 390])("opens the contextual landing for a new empty chat at %ipx", (width) => {
    const store = useAgentStore.getState();
    const previous = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
    store.addMessage({ chatId: previous.id, role: "user", content: "Earlier question" });
    const chatCount = useAgentStore.getState().chats.length;
    useAgentStore.setState({ size: { width, height: 450 } });
    mocks.pageResource = { kind: "project", name: "projects/orders", project: "projects/orders" };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      act(() => {
        (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click();
      });
      if (width === 390) {
        act(() => {
          (document.body.querySelector('[aria-label="agent.chat-list-label"]') as HTMLButtonElement).click();
        });
      }
      act(() => {
        (document.body.querySelector('[aria-label="agent.new-chat"]') as HTMLButtonElement).click();
      });

      const chatId = useAgentStore.getState().currentChatId!;
      expect(chatId).not.toBe(previous.id);
      expect(store.getMessages(chatId)).toEqual([]);
      expect(document.body.querySelector('[data-agent-contextual-start="project"]')).toBeInstanceOf(HTMLElement);
      expect(document.body.querySelector('[data-testid="agent-input"]')).toBeNull();
      expect(useAgentStore.getState().chats).toHaveLength(chatCount);
      act(() => {
        (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
      });
      expect(useAgentStore.getState().chats).toHaveLength(chatCount + 1);
      expect(store.getChat(useAgentStore.getState().currentChatId)?.context).toBe("projects/orders");
      expect(getCurrentQueuedUserMessage()?.chatId).toBe(useAgentStore.getState().currentChatId);
    } finally {
      unmount();
    }
  });

  test("shows the project start view first without replacing the selected draft", () => {
    const store = useAgentStore.getState();
    const existing = store.createChat({ context: "projects/analytics" });
    store.setDraft(existing.id, { text: "Unfinished analytics question" });
    const count = useAgentStore.getState().chats.length;
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(document.body.querySelector('[data-agent-contextual-start="project"]')).toBeInstanceOf(HTMLElement);
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(existing.id);
    expect(useAgentStore.getState().chats).toHaveLength(count);
    expect(store.getDraft(existing.id).text).toBe("Unfinished analytics question");
    expect(
      document.body.querySelector(
        `[data-agent-chat-row="${existing.id}"] [aria-current]`
      )
    ).toBeNull();

    act(() => {
      (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
    });
    const next = store.getChat(useAgentStore.getState().currentChatId);
    expect(next?.id).not.toBe(existing.id);
    expect(next?.context).toBe("projects/orders");
    expect(store.getDraft(next?.id).text).toBe("");
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: next?.id,
      text: "agent.landing-starters.project-next",
      runMode: "read-only",
    });
    expect(store.getDraft(existing.id).text).toBe("Unfinished analytics question");
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);
    unmount();
  });

  describe("contextual chat continuation", () => {
    beforeEach(() => {
      mocks.pageResource = {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
      };
    });

    test.each([1200, 700, 390])(
      "continues the newest matching chat at %ipx without changing chat data",
      async (width) => {
        const store = useAgentStore.getState();
        const older = store.createChat({
          context: "projects/orders",
          workspaceName: mocks.workspace.name,
        });
        const matching = store.createChat({
          context: "projects/orders",
          workspaceName: mocks.workspace.name,
        });
        const unrelated = store.createChat({
          context: "projects/analytics",
          workspaceName: mocks.workspace.name,
        });
        useAgentStore.setState((state) => ({
          size: { width, height: 450 },
          chats: state.chats.map((chat) => ({
            ...chat,
            updatedTs: chat.id === matching.id ? 20 : 10,
          })),
        }));
        const chats = useAgentStore.getState().chats;
        const { render, unmount } = renderIntoContainer(<AgentWindow />);
        try {
          render();
          expect(useAgentStore.getState().currentChatId).toBe(unrelated.id);
          const button = document.body.querySelector(
            "[data-agent-contextual-continue]"
          ) as HTMLButtonElement;
          expect(button).toBeInstanceOf(HTMLButtonElement);
          if (width === 1200) {
            const group = document.body.querySelector(
              '[data-agent-chat-group-toggle="projects/orders"]'
            ) as HTMLButtonElement;
            act(() => group.click());
            expect(group).toHaveAttribute("aria-expanded", "false");
          } else if (width === 700) {
            await act(async () => {
              (document.body.querySelector(
                '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
              ) as HTMLButtonElement).click();
              await Promise.resolve();
            });
            expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeInstanceOf(HTMLElement);
          }
          act(() => button.click());

          expect(useAgentStore.getState().currentChatId).toBe(matching.id);
          expect(useAgentStore.getState().currentChatId).not.toBe(older.id);
          expect(useAgentStore.getState().chats).toEqual(chats);
          expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
          if (width === 390) {
            expect(document.body.querySelector("[data-agent-mobile-conversation]")).toBeInstanceOf(HTMLElement);
          } else if (width === 700) {
            expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
          } else {
            expect(document.body.querySelector(
              '[data-agent-chat-group-toggle="projects/orders"]'
            )).toHaveAttribute("aria-expanded", "true");
          }
        } finally {
          unmount();
        }
      }
    );

    test("hides Continue without a matching chat and preserves the selected draft", () => {
      const store = useAgentStore.getState();
      const unrelated = store.createChat({
        context: "projects/analytics",
        workspaceName: mocks.workspace.name,
      });
      store.setDraft(unrelated.id, { text: "Keep this draft" });
      const chats = useAgentStore.getState().chats;
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        expect(document.body.querySelector("[data-agent-contextual-continue]")).toBeNull();
        expect(document.body.querySelector("[data-agent-contextual-suggestion]")).toBeInstanceOf(HTMLButtonElement);
        expect(useAgentStore.getState().currentChatId).toBe(unrelated.id);
        expect(useAgentStore.getState().chats).toEqual(chats);
        expect(store.getDraft(unrelated.id).text).toBe("Keep this draft");
      } finally {
        unmount();
      }
    });

    test("uses the captured Workspace scope after navigating to a Project", () => {
      const store = useAgentStore.getState();
      const matching = store.createChat({
        context: mocks.workspace.name,
        workspaceName: mocks.workspace.name,
      });
      store.createChat({
        context: "projects/analytics",
        workspaceName: mocks.workspace.name,
      });
      mocks.pageResource = { kind: "global-masking" };
      mocks.currentContext = mocks.workspace.name;
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        mocks.currentContext = "projects/analytics";
        mocks.pageResource = {
          kind: "project",
          name: "projects/analytics",
          project: "projects/analytics",
        };
        render();
        expect(document.body.querySelector('[data-agent-contextual-start="global-masking"]')).toBeInstanceOf(HTMLElement);
        act(() => {
          (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click();
        });
        expect(useAgentStore.getState().currentChatId).toBe(matching.id);
      } finally {
        unmount();
      }
    });

    test("invalidates the captured scope when the Workspace changes", () => {
      const originalWorkspace = mocks.workspace;
      const originalUser = mocks.currentUser;
      const store = useAgentStore.getState();
      store.createChat({
        context: "projects/orders",
        workspaceName: mocks.workspace.name,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        expect(document.body.querySelector("[data-agent-contextual-continue]")).toBeInstanceOf(HTMLButtonElement);
        mocks.workspace = { name: "workspaces/other", title: "Other Workspace" };
        mocks.currentUser = { ...originalUser!, workspace: mocks.workspace.name };
        render();
        expect(document.body.querySelector("[data-agent-contextual-continue]")).toBeNull();
        let nextChatId: string;
        act(() => {
          nextChatId = useAgentStore.getState().createChat({
            context: "projects/orders",
            workspaceName: mocks.workspace.name,
          }).id;
        });
        expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInstanceOf(HTMLElement);
        act(() => (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click());
        expect(useAgentStore.getState().currentChatId).toBe(nextChatId!);
        expect(useAgentStore.getState().getChat(nextChatId!)?.workspaceName).toBe("workspaces/other");
      } finally {
        unmount();
        mocks.workspace = originalWorkspace;
        mocks.currentUser = originalUser;
      }
    });

    test("does not enrich the resumed chat after navigating to a non-contextual page", () => {
      const store = useAgentStore.getState();
      const matching = store.createChat({
        context: "projects/orders",
        workspaceName: mocks.workspace.name,
      });
      store.createChat({ context: "projects/analytics" });
      const before = store.getChat(matching.id);
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        mocks.pageResource = undefined;
        mocks.currentContext = mocks.workspace.name;
        render();
        act(() => {
          (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click();
        });
        expect(useAgentStore.getState().currentChatId).toBe(matching.id);
        expect(store.getChat(matching.id)).toEqual(before);
      } finally {
        unmount();
      }
    });

    test.each(["archive", "delete"] as const)(
      "rechecks the candidate when it is removed by %s before the click",
      (operation) => {
        const store = useAgentStore.getState();
        const matching = store.createChat({
          context: "projects/orders",
          workspaceName: mocks.workspace.name,
        });
        const unrelated = store.createChat({ context: "projects/analytics" });
        const { render, unmount } = renderIntoContainer(<AgentWindow />);
        try {
          render();
          const button = document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement;
          act(() => {
            if (operation === "archive") store.archiveChat(matching.id);
            else store.deleteChat(matching.id);
            button.click();
          });
          expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInstanceOf(HTMLElement);
          expect(useAgentStore.getState().currentChatId).toBe(unrelated.id);
        } finally {
          unmount();
        }
      }
    );

    test("falls back to the next matching chat when the newest is archived before the click", () => {
      const store = useAgentStore.getState();
      const older = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
      const newest = store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
      store.createChat({ context: "projects/analytics" });
      useAgentStore.setState((state) => ({
        chats: state.chats.map((chat) => ({ ...chat, updatedTs: chat.id === newest.id ? 20 : 10 })),
      }));
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        const button = document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement;
        act(() => {
          store.archiveChat(newest.id);
          button.click();
        });
        expect(useAgentStore.getState().currentChatId).toBe(older.id);
        expect(store.getChat(newest.id)?.archived).toBe(true);
      } finally {
        unmount();
      }
    });

    test("keeps the landing visible when selection is rejected", () => {
      const store = useAgentStore.getState();
      store.createChat({ context: "projects/orders", workspaceName: mocks.workspace.name });
      const unrelated = store.createChat({ context: "projects/analytics" });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        const button = document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement;
        act(() => {
          useAgentStore.setState({ setCurrentChat: () => {} });
          button.click();
        });
        expect(useAgentStore.getState().currentChatId).toBe(unrelated.id);
        expect(document.body.querySelector("[data-agent-contextual-start]")).toBeInstanceOf(HTMLElement);
      } finally {
        act(() => useAgentStore.setState({ setCurrentChat: store.setCurrentChat }));
        unmount();
      }
    });

    test.each(["idle", "running", "awaiting_user", "error"] as const)(
      "preserves the matching %s chat's conversation and execution state",
      (status) => {
        const store = useAgentStore.getState();
        const matching = store.createChat({
          context: "projects/orders",
          workspaceName: mocks.workspace.name,
        });
        store.addMessage({ chatId: matching.id, role: "user", content: "Earlier question" });
        store.setDraft(matching.id, { text: "Unsent follow-up" });
        const page = { path: "/projects/orders", title: "Orders" };
        const controller = new AbortController();
        if (status === "running") {
          store.startChatRun(matching.id, page);
          store.setAbortController(matching.id, controller);
        } else if (status === "awaiting_user") {
          store.awaitUser(matching.id, { toolCallId: "ask", prompt: "Proceed?", kind: "confirm" });
        } else if (status === "error") {
          store.setChatStatus(matching.id, "error", { lastError: "Request failed" });
        }
        store.createChat({ context: "projects/analytics" });
        const before = useAgentStore.getState();
        const { render, unmount } = renderIntoContainer(<AgentWindow />);
        try {
          render();
          act(() => {
            (document.body.querySelector("[data-agent-contextual-continue]") as HTMLButtonElement).click();
          });
          const after = useAgentStore.getState();
          expect(after.currentChatId).toBe(matching.id);
          expect(after.chats).toEqual(before.chats);
          expect(after.messagesByChatId).toEqual(before.messagesByChatId);
          expect(after.draftsByChatId).toEqual(before.draftsByChatId);
          expect(after.pendingAskByChatId).toEqual(before.pendingAskByChatId);
          expect(after.queuedUserMessagesByChatId).toEqual(before.queuedUserMessagesByChatId);
          expect(after.getAbortController(matching.id)).toBe(before.getAbortController(matching.id));
          expect(controller.signal.aborted).toBe(false);
        } finally {
          unmount();
        }
      }
    );
  });

  test("shows AI setup on the page-detected start when ai_enabled is false", () => {
    mocks.aiEnabled = false;
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const start = document.body.querySelector(
      '[data-agent-contextual-start="project"]'
    );
    expect(start?.textContent).toContain("agent.ai-not-configured.title");
    unmount();
  });

  test("sends a contextual project starter through its direct read-only path", () => {
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const suggestions = document.body.querySelectorAll<HTMLButtonElement>(
      "[data-agent-contextual-suggestion]"
    );
    act(() => suggestions[1].click());

    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: useAgentStore.getState().currentChatId,
      text: "agent.landing-starters.project-open-blockers",
      runMode: "read-only",
      starterId: "project-open-blockers",
    });
    unmount();
  });

  test.each([
    {
      kind: "plan-create" as const,
      page: {
        kind: "plan-create" as const,
        project: "projects/orders",
      },
      context: "projects/orders",
      heading: "agent.contextual-start.plan-create-heading",
      suggestions: ["agent.contextual-start.plan-create-sql"],
    },
    {
      kind: "instance" as const,
      page: { kind: "instance" as const, name: "instances/primary" },
      context: "workspaces/acme",
      heading: "agent.contextual-start.instance-heading",
      suggestions: [
        "agent.contextual-start.instance-sync",
        "agent.contextual-start.instance-connection-permissions",
      ],
    },
    {
      kind: "instance-create" as const,
      page: {
        kind: "instance-create" as const,
        project: "projects/orders",
      },
      context: "projects/orders",
      heading: "agent.contextual-start.instance-create-heading",
      suggestions: [
        "agent.contextual-start.instance-create-details",
        "agent.contextual-start.instance-create-transport",
        "agent.contextual-start.instance-create-troubleshoot",
      ],
    },
    {
      kind: "instance" as const,
      page: {
        kind: "instance" as const,
        name: "projects/orders/instances/primary",
        project: "projects/orders",
      },
      context: "projects/orders",
      heading: "agent.contextual-start.instance-heading",
      suggestions: [
        "agent.contextual-start.instance-sync",
        "agent.contextual-start.instance-connection-permissions",
      ],
    },
  ])("starts an instance contextual chat for $kind", ({
    kind,
    page,
    context,
    heading,
    suggestions,
  }) => {
    mocks.currentContext = context;
    mocks.pageResource = page;

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const start = document.body.querySelector(
      `[data-agent-contextual-start="${kind}"]`
    );
    expect(start?.textContent).toContain(heading);
    expect(start?.querySelector("[data-agent-contextual-mention]")).toBeNull();
    expect(
      Array.from(
        start?.querySelectorAll("[data-agent-contextual-suggestion]") ?? [],
        (element) => element.textContent
      )
    ).toEqual(suggestions);

    act(() => {
      (start?.querySelector("[data-agent-contextual-suggestion]") as HTMLButtonElement).click();
    });

    const chat = useAgentStore
      .getState()
      .getChat(useAgentStore.getState().currentChatId);
    expect(chat?.context).toBe(context);
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat?.id,
      text: suggestions[0],
      runMode: "read-only",
    });
    expect(getCurrentQueuedUserMessage()).not.toHaveProperty("starterId");
    unmount();
  });

  test("prefills a bound database mention from the detail page", async () => {
    mocks.pageResource = {
      kind: "database",
      project: "projects/orders",
      names: ["projects/orders/instances/primary/databases/app"],
    };
    mocks.getOrFetchDatabaseByName.mockResolvedValue({
      name: "projects/orders/instances/primary/databases/app",
      project: "projects/orders",
      instanceResource: { title: "Primary" },
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    const mentionButton = document.body.querySelector('[data-agent-contextual-mention]') as HTMLButtonElement;
    expect(mentionButton).toBeInstanceOf(HTMLButtonElement);
    expect(mentionButton.disabled).toBe(false);
    act(() => mentionButton.click());

    const store = useAgentStore.getState();
    const chat = store.getChat(useAgentStore.getState().currentChatId);
    expect(chat?.context).toBe("projects/orders");
    expect(store.getDraft(chat?.id)).toEqual({
      text: "@app (Primary) ",
      mentions: [{
        resource: {
          kind: "database",
          name: "projects/orders/instances/primary/databases/app",
          project: "projects/orders",
          label: "app (Primary)",
        },
        range: { start: 0, end: 14 },
      }],
    });
    expect(store.getMessages(chat?.id)).toEqual([]);
    unmount();
  });

  test("prefills a bound database mention from the database catalog page", async () => {
    mocks.pageResource = {
      kind: "database-catalog",
      project: "projects/orders",
      names: ["projects/orders/instances/primary/databases/app"],
    };
    mocks.getOrFetchDatabaseByName.mockResolvedValue({
      name: "projects/orders/instances/primary/databases/app",
      project: "projects/orders",
      instanceResource: { title: "Primary" },
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    const mentionButton = document.body.querySelector(
      '[data-agent-contextual-mention]'
    ) as HTMLButtonElement;
    expect(mentionButton).toBeInstanceOf(HTMLButtonElement);
    expect(mentionButton.disabled).toBe(false);
    expect(mentionButton.textContent).toContain("@app (Primary)");

    act(() => mentionButton.click());
    const store = useAgentStore.getState();
    const chat = useAgentStore
      .getState()
      .getChat(useAgentStore.getState().currentChatId);
    expect(store.getDraft(chat?.id)).toMatchObject({
      text: "@app (Primary) ",
      mentions: [
        {
          resource: {
            kind: "database",
            name: "projects/orders/instances/primary/databases/app",
            project: "projects/orders",
            label: "app (Primary)",
          },
        },
      ],
    });
    unmount();
  });

  test("binds the Plan-create starter to its selected database", async () => {
    mocks.pageResource = {
      kind: "plan-create",
      project: "projects/orders",
      databaseName: "instances/primary/databases/app",
    };
    mocks.getOrFetchDatabaseByName.mockResolvedValue({
      name: "instances/primary/databases/app",
      project: "projects/orders",
      instanceResource: { title: "Primary" },
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    const start = document.body.querySelector(
      '[data-agent-contextual-start="plan-create"]'
    );
    const mention = start?.querySelector(
      "[data-agent-contextual-mention]"
    ) as HTMLButtonElement;
    expect(mention).toBeInstanceOf(HTMLButtonElement);
    expect(mention.textContent).toContain("@app (Primary)");

    act(() => {
      (
        start?.querySelector(
          "[data-agent-contextual-suggestion]"
        ) as HTMLButtonElement
      ).click();
    });

    expect(getCurrentQueuedUserMessage()).toMatchObject({
      text: "@app (Primary) agent.contextual-start.plan-create-sql",
      resourceReference: {
        kind: "database",
        name: "instances/primary/databases/app",
        project: "projects/orders",
        label: "app (Primary)",
      },
      runMode: "read-only",
    });
    unmount();
  });

  test("does not offer a database mention without list permission", () => {
    mocks.canListReferences = false;
    mocks.pageResource = {
      kind: "database",
      project: "projects/orders",
      names: ["instances/primary/databases/app"],
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeNull();
    expect(mocks.getOrFetchDatabaseByName).not.toHaveBeenCalled();
    expect(document.body.querySelector('[data-agent-contextual-start="database"]')?.textContent).toContain(
      "agent.contextual-start.resource-unavailable"
    );
    unmount();
  });

  test("prefills a bound issue mention and continues a matching Project chat", async () => {
    const store = useAgentStore.getState();
    const existing = store.createChat({
      context: "projects/orders",
      workspaceName: mocks.workspace.name,
    });
    store.createChat({ context: "projects/analytics" });
    mocks.pageResource = {
      kind: "issue",
      name: "projects/orders/issues/42",
      project: "projects/orders",
    };
    mocks.getIssue.mockResolvedValue({
      name: "projects/orders/issues/42",
      title: "Review database change",
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });
    expect(
      Array.from(document.body.querySelectorAll("[data-agent-contextual-suggestion]"), (element) =>
        element.textContent
      )
    ).toEqual([
      "agent.contextual-start.issue-summary",
      "agent.contextual-start.issue-changes",
      "agent.contextual-start.issue-security",
    ]);

    act(() => {
      (document.body.querySelector('[data-agent-contextual-continue]') as HTMLButtonElement).click();
    });
    expect(useAgentStore.getState().currentChatId).toBe(existing.id);
    expect(document.body.querySelector('[data-testid="agent-input"]')).toBeInstanceOf(HTMLElement);

    act(() => {
      (document.body.querySelector('[aria-label="agent.new-chat"]') as HTMLButtonElement).click();
    });
    await act(async () => Promise.resolve());
    const suggestion = document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement;
    act(() => suggestion.click());
    const chat = store.getChat(useAgentStore.getState().currentChatId);
    expect(chat?.context).toBe("projects/orders");
    expect(store.getDraft(chat?.id).text).toBe(
      ""
    );
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat?.id,
      text: "@Review database change agent.contextual-start.issue-summary",
      runMode: "read-only",
      resourceReference: {
        kind: "issue",
        name: "projects/orders/issues/42",
      },
    });
    unmount();
  });

  test("prefills a bound Plan mention from the detail page", async () => {
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Review schema",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    expect(document.body.querySelector('[data-agent-contextual-start="plan"]')?.textContent).toContain(
      "Review schema (#42)"
    );
    expect(
      Array.from(document.body.querySelectorAll("[data-agent-contextual-suggestion]"), (element) =>
        element.textContent
      )
    ).toEqual([
      "agent.contextual-start.plan-changes",
      "agent.contextual-start.plan-risks",
      "agent.contextual-start.plan-status",
    ]);
    act(() => {
      (document.body.querySelector('[data-agent-contextual-mention]') as HTMLButtonElement).click();
    });

    const store = useAgentStore.getState();
    const chat = useAgentStore.getState().getChat(useAgentStore.getState().currentChatId);
    expect(chat?.context).toBe("projects/orders");
    expect(store.getDraft(chat?.id)).toEqual({
      text: "@Review schema (#42) ",
      mentions: [{
        resource: {
          kind: "plan",
          name: "projects/orders/plans/42",
          project: "projects/orders",
          label: "Review schema (#42)",
        },
        range: { start: 0, end: 20 },
      }],
    });
    expect(store.getMessages(chat?.id)).toEqual([]);
    unmount();
  });

  test("sends a Plan question with its exact bound reference", async () => {
    const store = useAgentStore.getState();
    const previous = store.createChat({ context: "projects/analytics" });
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Review schema",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    expect(document.body.querySelector('[data-agent-contextual-continue]')).toBeNull();
    act(() => {
      (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
    });
    const chat = store.getChat(useAgentStore.getState().currentChatId);
    expect(chat?.id).not.toBe(previous.id);
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat?.id,
      text: "@Review schema (#42) agent.contextual-start.plan-changes",
      runMode: "read-only",
      resourceReference: {
        kind: "plan",
        name: "projects/orders/plans/42",
        project: "projects/orders",
      },
    });
    unmount();
  });

  test("offers a page-detected Plan mention with get permission only", async () => {
    mocks.planPermissions = {
      workspaceList: false,
      workspaceGet: true,
      projectList: false,
      projectGet: true,
    };
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Review schema",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    expect(mocks.getPlan).toHaveBeenCalledOnce();
    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeInstanceOf(HTMLButtonElement);
    unmount();
  });

  test("offers a page-detected Plan mention with project-only get permission", async () => {
    mocks.planPermissions.workspaceList = false;
    mocks.planPermissions.workspaceGet = false;
    mocks.planPermissions.projectList = false;
    mocks.planPermissions.projectGet = true;
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Review schema",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeInstanceOf(HTMLButtonElement);
    unmount();
  });

  test("lets the backend decide when the Plan policy has not loaded", async () => {
    mocks.planPermissions = {
      workspaceList: false,
      workspaceGet: false,
      projectList: false,
      projectGet: false,
    };
    mocks.planPolicyKnown = false;
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Review schema",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });

    expect(mocks.getPlan).toHaveBeenCalledOnce();
    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeInstanceOf(HTMLButtonElement);
    unmount();
  });

  test("does not bind a Plan when the detail fetch returns another or deleted Plan", async () => {
    mocks.pageResource = {
      kind: "plan",
      name: "projects/orders/plans/42",
      project: "projects/orders",
    };
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/43",
      title: "Other Plan",
      state: State.ACTIVE,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    await act(async () => {
      render();
      await Promise.resolve();
    });
    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeNull();

    act(() => useAgentStore.getState().toggle());
    mocks.getPlan.mockResolvedValue({
      name: "projects/orders/plans/42",
      title: "Deleted Plan",
      state: State.DELETED,
    });
    act(() => useAgentStore.getState().toggle());
    await act(async () => Promise.resolve());
    expect(document.body.querySelector('[data-agent-contextual-mention]')).toBeNull();
    unmount();
  });

  test("continues an active matching chat instead of the archived selection", () => {
    const store = useAgentStore.getState();
    const active = store.createChat({
      context: "projects/orders",
      workspaceName: mocks.workspace.name,
    });
    const archived = store.createChat({
      context: "projects/orders",
      archived: true,
    });
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      expect(useAgentStore.getState().currentChatId).toBe(archived.id);
      expect(
        document.body.querySelector(`[data-agent-chat-row="${archived.id}"]`)
      ).toBeNull();
      expect(
        document.body.querySelector("[data-agent-contextual-continue]")
      ).toBeInstanceOf(HTMLButtonElement);

      act(() => {
        (
          document.body.querySelector(
            "[data-agent-contextual-continue]"
          ) as HTMLButtonElement
        ).click();
      });
      expect(useAgentStore.getState().currentChatId).toBe(active.id);
      expect(
        document.body.querySelector("[data-agent-contextual-start]")
      ).toBeNull();
      expect(
        document.body.querySelector('[data-testid="agent-input"]')
      ).toBeInstanceOf(HTMLElement);
    } finally {
      unmount();
    }
  });

  test("does not offer Continue chat when no active chat can be selected", () => {
    const store = useAgentStore.getState();
    const archived = store.createChat({ archived: true });
    store.archiveAllActiveChats();
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();
      expect(useAgentStore.getState().currentChatId).toBe(archived.id);
      expect(
        document.body.querySelector("[data-agent-contextual-continue]")
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("keeps the start view when the Agent is reopened and refreshes it for a new chat", () => {
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();
    mocks.pageResource = {
      kind: "project",
      name: "projects/analytics",
      project: "projects/analytics",
    };
    mocks.currentContext = "projects/analytics";
    render();

    expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).toContain("Order service");
    expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).not.toContain("Analytics");

    act(() => useAgentStore.getState().toggle());
    act(() => useAgentStore.getState().toggle());
    expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).toContain("Order service");
    act(() => {
      (document.body.querySelector('[aria-label="agent.new-chat"]') as HTMLButtonElement).click();
    });
    expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).toContain("Analytics");
    unmount();
  });

  test("opens the selected sidebar chat without entering rename mode", () => {
    const chat = useAgentStore.getState().createChat({
      context: "projects/analytics",
    });
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      (
        document.body.querySelector(
          `[data-agent-chat-row="${chat.id}"] button`
        ) as HTMLButtonElement
      ).click();
    });
    expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    expect(document.body.querySelector("[data-agent-inline-rename-input]")).toBeNull();
    expect(useAgentStore.getState().currentChatId).toBe(chat.id);
    unmount();
  });

  test.each([1200, 700, 390])(
    "selects chats by title without entering rename mode at %ipx",
    async (width) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const chat = useAgentStore.getState().createChat({
        title: "Inspect the database settings",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        const title = document.body.querySelector(
          `[data-agent-chat-row="${chat.id}"] button`
        ) as HTMLButtonElement;
        act(() => title.click());
        expect(useAgentStore.getState().currentChatId).toBe(chat.id);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();

        if (width < 750) await openChatList(width);
        const selectedTitle = document.body.querySelector(
          `[data-agent-chat-row="${chat.id}"] button`
        ) as HTMLButtonElement;
        act(() => selectedTitle.click());
        expect(useAgentStore.getState().currentChatId).toBe(chat.id);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        expect(document.body.querySelector("[data-agent-inline-rename-input]")).toBeNull();
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390].flatMap((width) =>
    [false, true].map((archived) => ({ width, archived }))
  ))(
    "renames a row without selecting or restoring it at $width px when archived=$archived",
    async ({ width, archived }) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const selectedChatId = store.currentChatId;
      const selectedChat = store.getChat(selectedChatId);
      const target = store.createChat({
        title: "Saved database discussion",
        context: "projects/orders",
        archived,
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width, archived);
        const trigger = await openChatMenu(target.id);
        expect(trigger).toHaveAccessibleName("common.more: Saved database discussion");
        const menu = getLayerRoot("agent").querySelector("[data-agent-dropdown-menu-content]");
        expect(menu).toHaveAttribute("role", "menu");
        expect(menu?.querySelector("[data-agent-rename-chat]")).toBeInstanceOf(HTMLElement);
        expect(menu?.querySelector(archived ? "[data-agent-unarchive-chat]" : "[data-agent-archive-chat]")).toBeInstanceOf(HTMLElement);
        expect(menu?.querySelector("[data-agent-delete-chat]") !== null).toBe(archived);
        expect(getLayerRoot("overlay").querySelector("[data-agent-rename-chat]")).toBeNull();
        await act(async () => {
          (menu?.querySelector("[data-agent-rename-chat]") as HTMLElement).click();
          await Promise.resolve();
        });
        const row = document.body.querySelector(`[data-agent-chat-row="${target.id}"]`);
        const input = row?.querySelector("[data-agent-chat-title-input]") as HTMLInputElement;
        expect(input).toBeInstanceOf(HTMLInputElement);
        expect(input).toHaveValue("Saved database discussion");
        expect(document.activeElement).toBe(input);
        expect([input.selectionStart, input.selectionEnd]).toEqual([0, input.value.length]);
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
        expect(store.getChat(target.id)?.archived).toBe(archived);
        act(() => fireEvent.change(input, { target: { value: "  Renamed discussion  " } }));
        const save = row?.querySelector("[data-agent-save-chat-title]") as HTMLButtonElement;
        expect(save).toHaveAccessibleName("common.save");
        act(() => save.click());
        expect(store.getChat(target.id)?.title).toBe("Renamed discussion");
        expect(store.getChat(target.id)?.archived).toBe(archived);
        expect(store.getChat(selectedChatId)?.title).toBe(selectedChat?.title);
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        expect(document.body.querySelector(`[data-agent-chat-row="${target.id}"] button`)).toHaveAccessibleName("Renamed discussion");
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390])(
    "discards canceled rename edits and ignores composing Enter at %ipx",
    async (width) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const selectedChatId = store.currentChatId;
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        let input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "Uncommitted name" } }));
        act(() => fireEvent.keyDown(input, { key: "Escape" }));
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();

        input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "Canceled name" } }));
        const cancel = document.body.querySelector("[data-agent-cancel-chat-title]") as HTMLButtonElement;
        expect(cancel).toHaveAccessibleName("common.cancel");
        act(() => cancel.click());
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();

        input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "  Enter-saved name  " } }));
        act(() => fireEvent.keyDown(input, { key: "Enter", isComposing: true }));
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(input).toHaveFocus();
        expect(input.isConnected).toBe(true);
        act(() => fireEvent.keyDown(input, { key: "Enter" }));
        expect(store.getChat(target.id)?.title).toBe("Enter-saved name");
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390].flatMap((width) =>
    [
      { value: "  Renamed discussion  ", expected: "Renamed discussion" },
      { value: "   ", expected: "Original discussion" },
      { value: "  Original discussion  ", expected: "Original discussion" },
    ].map((rename) => ({ width, ...rename }))
  ))(
    "finishes rename on blur without moving focus at $width px for '$value'",
    async ({ width, value, expected }) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const selectedChatId = store.currentChatId;
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const destination = store.createChat({
        title: "Destination discussion",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        const input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value } }));
        const destinationButton = document.body.querySelector(
          `[data-agent-chat-row="${destination.id}"] button`
        ) as HTMLButtonElement;
        act(() => destinationButton.focus());
        expect(store.getChat(target.id)?.title).toBe(expected);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        expect(destinationButton).toHaveFocus();
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390].flatMap((width) =>
    ["   ", "  Original discussion  "].map((value) => ({ width, value }))
  ))(
    "finishes unchanged rename on Enter at $width px for '$value'",
    async ({ width, value }) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        const input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value } }));
        expect(document.body.querySelector("[data-agent-save-chat-title]")).toBeDisabled();
        act(() => fireEvent.keyDown(input, { key: "Enter" }));
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        expect(document.body.querySelector(
          `[data-agent-chat-row="${target.id}"] [data-agent-chat-menu]`
        )).toHaveFocus();
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390].flatMap((width) =>
    ["pointer", "keyboard"].map((interaction) => ({ width, interaction }))
  ))(
    "does not save before $interaction Cancel at $width px",
    async ({ width, interaction }) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        const input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "Canceled discussion" } }));
        const cancel = document.body.querySelector(
          "[data-agent-cancel-chat-title]"
        ) as HTMLButtonElement;
        if (interaction === "pointer") {
          act(() => {
            const shouldFocus = fireEvent.pointerDown(cancel, {
              button: 0,
              clientX: 100,
              clientY: 100,
            });
            if (shouldFocus) cancel.focus();
          });
        } else {
          const save = document.body.querySelector(
            "[data-agent-save-chat-title]"
          ) as HTMLButtonElement;
          act(() => save.focus());
          expect(save).toHaveFocus();
          act(() => cancel.focus());
          expect(cancel).toHaveFocus();
        }
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(cancel.isConnected).toBe(true);
        act(() => cancel.click());
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
      } finally {
        unmount();
      }
    }
  );

  test.each([1200, 700, 390].flatMap((width) =>
    [false, true].map((selected) => ({ width, selected }))
  ))(
    "finishes rename when clicking a chat at $width px (already selected: $selected)",
    async ({ width, selected }) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const destination = store.createChat({
        title: "Destination discussion",
        context: "projects/orders",
        select: selected,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        await openChatList(width);
        const input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "Renamed discussion" } }));
        const destinationButton = document.body.querySelector(
          `[data-agent-chat-row="${destination.id}"] button`
        ) as HTMLButtonElement;
        act(() => {
          const shouldFocus = fireEvent.mouseDown(destinationButton, { button: 0 });
          expect(shouldFocus).toBe(false);
          if (shouldFocus) destinationButton.focus();
        });
        expect(input).toHaveFocus();
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        act(() => destinationButton.click());
        expect(store.getChat(target.id)?.title).toBe("Renamed discussion");
        expect(store.getChat(destination.id)?.title).toBe("Destination discussion");
        expect(useAgentStore.getState().currentChatId).toBe(destination.id);
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        if (width >= 750) expect(destinationButton).toHaveFocus();
      } finally {
        unmount();
      }
    }
  );

  test.each(["close/open", "minimize/restore"])(
    "refocuses the unsaved rename draft after %s",
    async (transition) => {
      const store = useAgentStore.getState();
      const selectedChatId = store.currentChatId;
      const target = store.createChat({
        title: "Original discussion",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        const input = await beginChatRename(target.id);
        act(() => fireEvent.change(input, { target: { value: "Unsaved discussion" } }));
        act(() => {
          if (transition === "close/open") store.toggle();
          else store.minimize();
        });
        expect(document.body.querySelector("[data-agent-chat-title-input]")).toBeNull();
        act(() => {
          if (transition === "close/open") store.open();
          else store.restore();
        });

        const reopenedInput = document.body.querySelector(
          `[data-agent-chat-row="${target.id}"] [data-agent-chat-title-input]`
        ) as HTMLInputElement;
        expect(reopenedInput).toHaveValue("Unsaved discussion");
        expect(document.activeElement).toBe(reopenedInput);
        expect([reopenedInput.selectionStart, reopenedInput.selectionEnd]).toEqual([
          0,
          reopenedInput.value.length,
        ]);
        expect(store.getChat(target.id)?.title).toBe("Original discussion");
        expect(useAgentStore.getState().currentChatId).toBe(selectedChatId);
      } finally {
        unmount();
      }
    }
  );

  test("allows a contextual chat to start while another chat is running", () => {
    const store = useAgentStore.getState();
    const running = store.createChat({ context: "projects/analytics" });
    store.setChatStatus(running.id, "running");
    mocks.pageResource = {
      kind: "project",
      name: "projects/orders",
      project: "projects/orders",
    };
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(document.body.querySelector('[data-agent-contextual-start="project"]')).toBeInstanceOf(HTMLElement);
    const suggestion = document.body.querySelector(
      '[data-agent-contextual-suggestion]'
    ) as HTMLButtonElement;
    expect(suggestion.disabled).toBe(false);
    act(() => suggestion.click());
    const current = useAgentStore.getState().getChat(
      useAgentStore.getState().currentChatId
    );
    expect(current?.id).not.toBe(running.id);
    expect(current?.context).toBe("projects/orders");
    expect(useAgentStore.getState().getChat(running.id)?.status).toBe("running");
    unmount();
  });

  test("allows a new chat and switching back while a chat is running", () => {
    const store = useAgentStore.getState();
    const running = store.createChat({ context: "projects/orders" });
    store.setChatStatus(running.id, "running");
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const newChat = document.body.querySelector(
      '[data-agent-chat-sidebar-actions] [aria-label="agent.new-chat"]'
    ) as HTMLButtonElement;
    expect(newChat.disabled).toBe(false);
    act(() => newChat.click());
    const nextChatId = useAgentStore.getState().currentChatId;
    expect(nextChatId).not.toBe(running.id);
    expect(useAgentStore.getState().getChat(running.id)?.status).toBe("running");

    const runningChatRow = document.body.querySelector(
      `[data-agent-chat-row="${running.id}"] button`
    ) as HTMLButtonElement;
    expect(runningChatRow.disabled).toBe(false);
    expect(
      runningChatRow.querySelector('[aria-label="common.running"]')
    ).toBeInstanceOf(SVGElement);
    act(() => runningChatRow.click());
    expect(useAgentStore.getState().currentChatId).toBe(running.id);
    unmount();
  });

  test("sends an empty-chat suggestion for the current context", () => {
    const chat = useAgentStore.getState().ensureCurrentChat(
      undefined,
      "projects/orders"
    );
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(
      document.body.querySelector('[data-testid="agent-chat"]')
    ).toHaveAttribute("data-context-label", "Order service");
    act(() => {
      (document.body.querySelector('[data-testid="agent-empty-suggestion"]') as HTMLButtonElement).click();
    });
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe("");
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat.id,
      text: "agent.landing-starters.project-open-blockers",
      runMode: "read-only",
      starterId: "project-open-blockers",
    });
    expect(useAgentStore.getState().getMessages(chat.id)).toEqual([]);
    unmount();
  });

  test("does not replace an existing draft with a suggested prompt", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.setDraft(chat.id, { text: "Unfinished question" });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      (document.body.querySelector('[data-testid="agent-empty-suggestion"]') as HTMLButtonElement).click();
    });
    expect(store.getDraft(chat.id).text).toBe("Unfinished question");
    unmount();
  });

  test("queues a response follow-up to send immediately", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      document.body
        .querySelector<HTMLButtonElement>('[data-testid="agent-continue-action"]')
        ?.click();
    });
    expect(store.getDraft(chat.id).text).toBe("");
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat.id,
      text: "What should I review next?",
      runMode: "default",
    });
    expect(store.getMessages(chat.id)).toEqual([]);
    unmount();
  });

  test("keeps the referenced project bound when sending a follow-up", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      document.body
        .querySelector<HTMLButtonElement>(
          '[data-testid="agent-continue-with-project"]'
        )
        ?.click();
    });
    expect(store.getDraft(chat.id).text).toBe("");
    expect(getCurrentQueuedUserMessage()).toMatchObject({
      chatId: chat.id,
      text: "@Orders What should I review next?",
      runMode: "default",
      resourceReferences: [
        {
          kind: "project",
          name: "projects/orders",
          project: "projects/orders",
          label: "Orders",
        },
      ],
    });
    expect(store.getMessages(chat.id)).toEqual([]);
    unmount();
  });

  test("does not replace an existing draft with a response follow-up", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    store.setDraft(chat.id, { text: "Unfinished question" });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      document.body
        .querySelector<HTMLButtonElement>(
          '[data-testid="agent-continue-action"]'
        )
        ?.click();
    });
    expect(store.getDraft(chat.id).text).toBe("Unfinished question");
    expect(getCurrentQueuedUserMessage()).toBeUndefined();
    unmount();
  });

  test("does not replace a queued response follow-up", () => {
    const store = useAgentStore.getState();
    const chat = store.ensureCurrentChat(undefined, "projects/orders");
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    act(() => {
      document.body
        .querySelector<HTMLButtonElement>('[data-testid="agent-continue-action"]')
        ?.click();
    });
    const queued = getCurrentQueuedUserMessage();
    act(() => {
      document.body
        .querySelector<HTMLButtonElement>(
          '[data-testid="agent-continue-with-project"]'
        )
        ?.click();
    });
    expect(getCurrentQueuedUserMessage()).toEqual(queued);
    unmount();
  });

  test("uses the compact rail below the roomy agent-window width", () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(
      document.body.querySelector("[data-agent-compact-rail]")
    ).toBeInstanceOf(HTMLElement);
    expect(
      document.body.querySelector("[data-agent-chat-sidebar]")
    ).toBeNull();

    unmount();
  });

  test("does not render a sidebar resize handle in compact view", () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(
      document.body.querySelector("[data-agent-sidebar-resize]")
    ).toBeNull();

    unmount();
  });

  test("keeps the compact rail at the 480px agent-window boundary", () => {
    useAgentStore.setState({ size: { width: 480, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      expect(
        document.body.querySelector("[data-agent-compact-rail]")
      ).toBeInstanceOf(HTMLElement);
      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("shows navigation rail, chat list, and conversation together when the agent window has room", () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const body = document.body.querySelector("[data-agent-window-body]");
      const rail = body?.querySelector("[data-agent-compact-rail]");
      const list = body?.querySelector("[data-agent-chat-sidebar]");
      const panels = body?.querySelector("[data-agent-chat-panels]");
      expect(rail).toBeInstanceOf(HTMLElement);
      expect(list).toBeInstanceOf(HTMLElement);
      expect(panels).toBeInstanceOf(HTMLElement);
      expect(rail).not.toContainElement(list as HTMLElement);
      expect(body?.querySelector("[data-agent-compact-drawer]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test("switches the persistent chat list between active and archived chats from the rail", () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });
    const activeChatId = useAgentStore.getState().currentChatId!;

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      const archiveButton = rail?.querySelector(
        '[aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      const chatsButton = rail?.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
      expect(chatsButton).toBeInstanceOf(HTMLButtonElement);

      act(() => archiveButton.click());
      const archivedList = document.body.querySelector(
        "[data-agent-chat-sidebar]"
      );
      expect(archivedList?.textContent).toContain("agent.archived-only-chats");
      expect(
        archivedList?.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`)
      ).toBeInstanceOf(HTMLElement);
      expect(
        archivedList?.querySelector(`[data-agent-chat-row="${activeChatId}"]`)
      ).toBeNull();
      expect(archivedList?.querySelector('[aria-label="agent.new-chat"]')).toBeNull();
      expect(archivedList?.querySelector("[data-agent-new-chat-for-context]")).toBeNull();

      act(() => chatsButton.click());
      const activeList = document.body.querySelector("[data-agent-chat-sidebar]");
      expect(activeList?.querySelector(`[data-agent-chat-row="${activeChatId}"]`)).toBeInstanceOf(
        HTMLElement
      );
      expect(activeList?.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`)).toBeNull();
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test.each([
    ["large", 750],
    ["compact", 700],
  ])("highlights the selected %s rail destination", async (_name, width) => {
    useAgentStore.setState({ size: { width, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      const chatsButton = rail?.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      const archiveButton = rail?.querySelector(
        '[aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;

      if (width === 750) {
        expect(chatsButton).toHaveAttribute("aria-current", "page");
      } else {
        expect(chatsButton).not.toHaveAttribute("aria-current");
      }
      expect(archiveButton).not.toHaveAttribute("aria-current");

      act(() => archiveButton.click());
      expect(chatsButton).not.toHaveAttribute("aria-current");
      expect(archiveButton).toHaveAttribute("aria-current", "page");

      if (width === 700) {
        const closeButton = document.body.querySelector(
          '[data-agent-compact-drawer] [aria-label="agent.close"]'
        ) as HTMLButtonElement;
        await act(async () => {
          closeButton.click();
          await Promise.resolve();
        });
        expect(archiveButton).not.toHaveAttribute("aria-current");
      }
    } finally {
      unmount();
    }
  });

  test.each([
    ["large", 750],
    ["compact", 700],
  ])("creates an active chat from the archived %s rail", async (_name, width) => {
    useAgentStore.setState({ size: { width, height: 450 } });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });
    const previousChatId = useAgentStore.getState().currentChatId;

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      const archiveButton = rail?.querySelector(
        '[aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      const newChatButton = rail?.querySelector(
        '[aria-label="agent.new-chat"]'
      ) as HTMLButtonElement;
      expect(newChatButton).toBeInstanceOf(HTMLButtonElement);

      act(() => archiveButton.click());
      expect(rail?.querySelector('[aria-label="agent.new-chat"]')).toBe(
        newChatButton
      );

      await act(async () => {
        newChatButton.click();
        await Promise.resolve();
      });

      expect(document.body.querySelector('[data-agent-contextual-start="project"]')).toBeInstanceOf(HTMLElement);
      act(() => {
        (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
      });

      const currentChatId = useAgentStore.getState().currentChatId;
      expect(currentChatId).not.toBe(previousChatId);
      expect(useAgentStore.getState().getChat(currentChatId!)?.archived).toBe(
        false
      );
      if (width === 750) {
        const list = document.body.querySelector("[data-agent-chat-sidebar]");
        expect(list?.querySelector(`[data-agent-chat-row="${currentChatId}"]`)).toBeInstanceOf(
          HTMLElement
        );
        expect(list?.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`)).toBeNull();
      } else {
        expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
        const chatsButton = rail?.querySelector(
          '[aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        await act(async () => {
          chatsButton.click();
          await Promise.resolve();
        });
        expect(
          document.body.querySelector(
            `[data-agent-compact-drawer] [data-agent-chat-row="${currentChatId}"]`
          )
        ).toBeInstanceOf(HTMLElement);
      }
    } finally {
      unmount();
    }
  });

  test("keeps the current conversation selected while the large rail switches list modes", () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const activeChatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      const archiveButton = rail?.querySelector(
        '[aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      const chatsButton = rail?.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;

      act(() => archiveButton.click());
      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);
      expect(document.body.querySelector("[data-agent-chat-panels]")).not.toHaveAttribute(
        "aria-hidden"
      );
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();

      act(() => chatsButton.click());
      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);
      expect(
        document.body.querySelector(
          `[data-agent-chat-row="${activeChatId}"] [aria-current="true"]`
        )
      ).toBeInstanceOf(HTMLButtonElement);
    } finally {
      unmount();
    }
  });

  test.each([1200, 700, 390])("restores the viewed archived chat into active chats at %ipx", async (width) => {
    vi.stubGlobal("innerWidth", Math.max(width, 390));
    useAgentStore.setState({ size: { width, height: 600 } });
    const store = useAgentStore.getState();
    const archivedChat = store.createChat({
      title: "Saved discussion",
      context: "projects/orders",
      workspaceName: mocks.workspace.name,
      archived: true,
      select: false,
    });
    const otherArchivedChat = store.createChat({
      title: "Other archived discussion",
      context: "projects/orders",
      workspaceName: mocks.workspace.name,
      archived: true,
      select: false,
    });
    store.addMessage({ chatId: archivedChat.id, role: "user", content: "Review the database settings" });
    store.setDraft(archivedChat.id, { text: "Saved unsent question" });
    const messages = store.getMessages(archivedChat.id);
    const chatCount = useAgentStore.getState().chats.length;
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      if (width === 390) {
        act(() => screen.getByRole("button", { name: "agent.chat-list-label" }).click());
      }
      await act(async () => {
        screen.getByRole("button", { name: "agent.archived-only-chats" }).click();
        await Promise.resolve();
      });
      const row = document.body.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`);
      const chatButton = row?.querySelector<HTMLButtonElement>("button");
      expect.soft(chatButton).toHaveAccessibleName("Saved discussion");
      expect.soft(chatButton).toHaveAttribute("title", "Saved discussion");
      act(() => row?.querySelector<HTMLButtonElement>("button")?.click());

      expect(useAgentStore.getState().currentChatId).toBe(archivedChat.id);
      const notice = screen.getByRole("status");
      expect(notice).toHaveTextContent("agent.chat-archived-notice");
      const restoreButton = notice.parentElement?.querySelector<HTMLButtonElement>("button");
      expect(restoreButton).toHaveAccessibleName("agent.unarchive-chat");
      expect.soft(restoreButton?.querySelector("svg")).toHaveClass("size-4");
      expect(store.getMessages(archivedChat.id)).toEqual(messages);
      expect(store.getDraft(archivedChat.id)?.text).toBe("Saved unsent question");

      act(() => restoreButton?.click());

      expect(useAgentStore.getState().currentChatId).toBe(archivedChat.id);
      expect(store.getChat(archivedChat.id)?.archived).toBe(false);
      expect(store.getChat(otherArchivedChat.id)?.archived).toBe(true);
      expect(store.getMessages(archivedChat.id)).toEqual(messages);
      expect(store.getDraft(archivedChat.id)?.text).toBe("Saved unsent question");
      expect(useAgentStore.getState().chats).toHaveLength(chatCount);
      expect(getCurrentQueuedUserMessage()).toBeUndefined();
      expect(screen.queryByRole("status")).toBeNull();

      if (width < 750) {
        await act(async () => {
          screen.getByRole("button", { name: "agent.chat-list-label" }).click();
          await Promise.resolve();
        });
      }
      const restoredRow = document.body.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`);
      expect(restoredRow?.querySelector('[aria-current="true"]')).toBeInstanceOf(HTMLButtonElement);
      expect(document.body.querySelector(`[data-agent-chat-row="${otherArchivedChat.id}"]`)).toBeNull();
    } finally {
      unmount();
    }
  });

  describe.each(["page help", "queued message"] as const)("after %s", (source) => {
    test.each([1200, 700, 390])("keeps the archived list when selecting an archived chat at %ipx", async (width) => {
      vi.stubGlobal("innerWidth", width === 390 ? 390 : 1600);
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const archivedChat = store.createChat({
        title: "Saved discussion",
        context: "projects/orders",
        workspaceName: mocks.workspace.name,
        select: false,
      });
      store.addMessage({ chatId: archivedChat.id, role: "user", content: "Review the database settings" });
      if (source === "queued message") {
        store.queueUserMessage({
          chatId: archivedChat.id,
          text: "Saved queued question",
          page: { path: "/projects/orders", title: "Order service" },
        });
      }
      store.archiveChat(archivedChat.id);
      const messages = store.getMessages(archivedChat.id);
      const queuedMessage = useAgentStore.getState().queuedUserMessagesByChatId[archivedChat.id];
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      let revoke = () => {};
      render();

      try {
        if (source === "page help") {
          act(() => { revoke = openAgentPageHelp({ page: { kind: "global-masking" } }); });
        }
        if (width === 390) {
          act(() => screen.getByRole("button", { name: "agent.chat-list-label" }).click());
        }
        await act(async () => {
          screen.getByRole("button", { name: "agent.archived-only-chats" }).click();
          await Promise.resolve();
        });
        const row = document.body.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`);
        expect(row).toBeInstanceOf(HTMLElement);
        act(() => row?.querySelector<HTMLButtonElement>("button")?.click());

        expect(useAgentStore.getState().currentChatId).toBe(archivedChat.id);
        expect(screen.getByRole("status")).toHaveTextContent("agent.chat-archived-notice");
        if (width === 390) {
          act(() => screen.getByRole("button", { name: "agent.chat-list-label" }).click());
        } else if (width === 700) {
          act(() => useAgentStore.setState({ size: { width: 1200, height: 600 } }));
        }

        const expectArchivedSelection = () => {
          const selectedRow = document.body.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`);
          expect(selectedRow?.querySelector('[aria-current="true"]')).toBeInstanceOf(HTMLButtonElement);
          expect(store.getChat(archivedChat.id)?.archived).toBe(true);
          expect(store.getMessages(archivedChat.id)).toEqual(messages);
          expect(useAgentStore.getState().queuedUserMessagesByChatId[archivedChat.id]).toEqual(queuedMessage);
        };
        expectArchivedSelection();

        act(() => store.toggle());
        act(() => store.open());
        if (width === 390) {
          act(() => screen.getByRole("button", { name: "agent.chat-list-label" }).click());
        }
        expectArchivedSelection();
      } finally {
        act(() => revoke());
        unmount();
      }
    });
  });

  test("keeps an unrelated active conversation selected when restoring and deleting archived chats", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const activeChatId = useAgentStore.getState().currentChatId!;
    const restoredChat = useAgentStore.getState().createChat({
      title: "Restore analytics",
      archived: true,
      select: false,
    });
    const deletedChat = useAgentStore.getState().createChat({
      title: "Delete analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      act(() =>
        (
          rail?.querySelector(
            '[aria-label="agent.archived-only-chats"]'
          ) as HTMLButtonElement
        ).click()
      );

      const openRowMenu = async (chatId: string) => {
        const menuButton = document.body.querySelector(
          `[data-agent-chat-row="${chatId}"] [data-agent-archived-chat-menu]`
        ) as HTMLButtonElement;
        await act(async () => {
          menuButton.click();
          await Promise.resolve();
        });
      };

      await openRowMenu(restoredChat.id);
      act(() =>
        (
          getLayerRoot("agent").querySelector(
            "[data-agent-unarchive-chat]"
          ) as HTMLElement
        ).click()
      );
      expect(useAgentStore.getState().getChat(restoredChat.id)?.archived).toBe(
        false
      );
      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);

      await openRowMenu(deletedChat.id);
      act(() =>
        (
          getLayerRoot("agent").querySelector(
            "[data-agent-delete-chat]"
          ) as HTMLElement
        ).click()
      );
      const confirmation = getLayerRoot("agent").querySelector(
        "[data-agent-dialog-content]"
      ) as HTMLElement;
      const confirmButton = Array.from(
        confirmation.querySelectorAll("button")
      ).find((button) => button.textContent === "common.confirm");
      act(() => confirmButton?.click());

      expect(useAgentStore.getState().getChat(deletedChat.id)).toBeNull();
      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);

      act(() =>
        (
          rail?.querySelector('[aria-label="agent.chat-list-label"]') as HTMLButtonElement
        ).click()
      );
      expect(
        document.body.querySelector(
          `[data-agent-chat-row="${activeChatId}"] [aria-current="true"]`
        )
      ).toBeInstanceOf(HTMLButtonElement);
    } finally {
      unmount();
    }
  });

  test("keeps an unrelated active conversation selected when unarchiving all chats", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const activeChatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().createChat({
      title: "Archived analytics",
      archived: true,
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      act(() =>
        (
          document.body.querySelector(
            '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
          ) as HTMLButtonElement
        ).click()
      );
      const menuButton = document.body.querySelector(
        '[data-agent-chat-sidebar-actions] [aria-label="common.more"]'
      ) as HTMLButtonElement;
      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });
      act(() =>
        (
          getLayerRoot("agent").querySelector(
            "[data-agent-unarchive-all-chats]"
          ) as HTMLElement
        ).click()
      );

      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);
    } finally {
      unmount();
    }
  });

  test("keeps an unrelated active conversation selected when deleting all archived chats", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const activeChatId = useAgentStore.getState().currentChatId!;
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      archived: true,
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      act(() =>
        (
          document.body.querySelector(
            '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
          ) as HTMLButtonElement
        ).click()
      );
      const menuButton = document.body.querySelector(
        '[data-agent-chat-sidebar-actions] [aria-label="common.more"]'
      ) as HTMLButtonElement;
      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });
      act(() =>
        (
          getLayerRoot("agent").querySelector(
            "[data-agent-delete-all-chats]"
          ) as HTMLElement
        ).click()
      );
      const confirmation = getLayerRoot("agent").querySelector(
        "[data-agent-dialog-content]"
      ) as HTMLElement;
      const confirmButton = Array.from(
        confirmation.querySelectorAll("button")
      ).find((button) => button.textContent === "common.confirm");
      act(() => confirmButton?.click());

      expect(useAgentStore.getState().getChat(archivedChat.id)).toBeNull();
      expect(useAgentStore.getState().currentChatId).toBe(activeChatId);
    } finally {
      unmount();
    }
  });

  test("keeps an unrelated archived conversation selected when archiving active chats", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const firstActiveChatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().createChat({
      title: "Second active",
      select: false,
    });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      archived: true,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      await selectChatMenuAction(firstActiveChatId, "[data-agent-archive-chat]");
      const confirmation = getLayerRoot("agent").querySelector(
        "[data-agent-archive-chat-confirmation]"
      ) as HTMLElement;
      await act(async () => {
        Array.from(confirmation.querySelectorAll("button"))
          .find((button) => button.textContent === "common.archive")!
          .click();
      });
      expect(useAgentStore.getState().currentChatId).toBe(archivedChat.id);

      const menuButton = document.body.querySelector(
        '[data-agent-chat-sidebar-actions] [aria-label="common.more"]'
      ) as HTMLButtonElement;
      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });
      act(() =>
        (
          getLayerRoot("agent").querySelector(
            "[data-agent-archive-all-chats]"
          ) as HTMLElement
        ).click()
      );
      expect(useAgentStore.getState().currentChatId).toBe(archivedChat.id);
    } finally {
      unmount();
    }
  });

  test("does not reveal the compact drawer when resizing an archived large list", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      act(() => archiveButton.click());
      expect(document.body.querySelector("[data-agent-chat-sidebar]")).toBeInstanceOf(
        HTMLElement
      );

      act(() => useAgentStore.setState({ size: { width: 700, height: 450 } }));
      expect(document.body.querySelector("[data-agent-chat-sidebar]")).toBeNull();
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
      expect(document.body.querySelector("[data-agent-chat-panels]")).not.toHaveAttribute(
        "aria-hidden"
      );

      await act(async () => {
        archiveButton.click();
        await Promise.resolve();
      });
      expect(
        document.body.querySelector(
          `[data-agent-compact-drawer] [data-agent-chat-row="${archivedChat.id}"]`
        )
      ).toBeInstanceOf(HTMLElement);
    } finally {
      unmount();
    }
  });

  test("offers Rename, Restore, and Delete in one archived chat row menu", async () => {
    mocks.useTranslation.mockReturnValue({
      t: (key: string) =>
        key === "agent.unarchive-chat" ? enUS.agent["unarchive-chat"] : key,
      i18n: { language: "en-US" },
    });
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
      act(() => archiveButton.click());

      const list = document.body.querySelector("[data-agent-chat-sidebar]");
      const row = list?.querySelector(
        `[data-agent-chat-row="${archivedChat.id}"]`
      );
      const menuButton = row?.querySelector(
        "[data-agent-archived-chat-menu]"
      ) as HTMLButtonElement;
      expect(menuButton).toBeInstanceOf(HTMLButtonElement);
      expect(row?.querySelector("[data-agent-unarchive-chat]")).toBeNull();
      expect(row?.querySelector("[data-agent-delete-chat]")).toBeNull();
      expect(list?.querySelector('[aria-label="agent.new-chat"]')).toBeNull();
      expect(list?.querySelector("[data-agent-new-chat-for-context]")).toBeNull();

      await act(async () => {
        menuButton.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
        await Promise.resolve();
      });

      const agentRoot = getLayerRoot("agent");
      expect(agentRoot.querySelector("[data-agent-rename-chat]")).toBeInstanceOf(
        HTMLElement
      );
      expect(
        agentRoot.querySelector("[data-agent-unarchive-chat]")
      ).toBeInstanceOf(HTMLElement);
      expect(
        agentRoot.querySelector("[data-agent-unarchive-chat]")?.textContent
      ).toBe("Restore chat");
      expect(agentRoot.querySelector("[data-agent-delete-chat]")).toBeInstanceOf(
        HTMLElement
      );
    } finally {
      unmount();
    }
  });

  test("deletes an archived chat from the compact drawer row menu", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    const archivedChat = useAgentStore.getState().createChat({
      title: "Delete archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      await act(async () => {
        archiveButton.click();
        await Promise.resolve();
      });

      const drawer = document.body.querySelector(
        "[data-agent-compact-drawer]"
      ) as HTMLElement;
      const row = drawer.querySelector(
        `[data-agent-chat-row="${archivedChat.id}"]`
      ) as HTMLElement;
      expect(row).toBeInstanceOf(HTMLElement);
      const menuButton = row.querySelector(
        "[data-agent-archived-chat-menu]"
      ) as HTMLButtonElement;
      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });

      const deleteButton = getLayerRoot("agent").querySelector(
        "[data-agent-delete-chat]"
      ) as HTMLElement;
      expect(deleteButton).toBeInstanceOf(HTMLElement);
      await act(async () => {
        deleteButton.click();
        await Promise.resolve();
      });

      const confirmation = getLayerRoot("agent").querySelector(
        "[data-agent-dialog-content]"
      ) as HTMLElement;
      expect(confirmation?.textContent).toContain(
        "agent.delete-chat-confirmation"
      );
      const confirmButton = Array.from(confirmation.querySelectorAll("button")).find(
        (button) => button.textContent === "common.confirm"
      );
      act(() => confirmButton?.click());

      expect(useAgentStore.getState().getChat(archivedChat.id)).toBeNull();
    } finally {
      unmount();
    }
  });

  test("renders the compact drawer as a bounded dialog", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const listButton = document.body.querySelector(
        'button[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      await act(async () => {
        listButton.click();
        await Promise.resolve();
      });

      const drawer = document.body.querySelector(
        "[data-agent-compact-drawer]"
      ) as HTMLElement;
      const body = document.body.querySelector(
        "[data-agent-window-body]"
      ) as HTMLDivElement;
      expect(body).toContainElement(drawer);
      expect(drawer).toHaveAttribute("role", "dialog");
      expect(
        document.body.querySelector("[data-agent-chat-panels]")
      ).toHaveAttribute("aria-hidden", "true");
      expect(
        document.body.querySelector("[data-agent-chat-panels]")
      ).toHaveAttribute("inert");
      expect(
        document.body.querySelector("[data-agent-window-header]")
      ).toHaveAttribute("inert");
      expect(
        document.body.querySelector("[data-agent-compact-header-overlay]")
      ).toBeNull();
      expect(
        document.body.querySelector("[data-agent-window-resize-zone]")
      ).toBeNull();
      expect(
        document.body.querySelector("[data-agent-compact-rail]")
      ).toBeInstanceOf(HTMLElement);
    } finally {
      unmount();
    }
  });

  test("keeps an active critical layer accessible while the compact drawer is open", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    const criticalRoot = getLayerRoot("critical");
    const criticalAction = document.createElement("button");
    criticalAction.textContent = "Critical action";
    criticalRoot.appendChild(criticalAction);

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const listButton = document.body.querySelector(
        'button[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      await act(async () => {
        listButton.click();
        await Promise.resolve();
      });

      await vi.waitFor(() => {
        expect(criticalRoot).not.toHaveAttribute("aria-hidden");
        expect(criticalRoot).not.toHaveAttribute("inert");
        expect(criticalRoot).not.toHaveAttribute("data-base-ui-inert");
      });
    } finally {
      unmount();
    }
  });

  test("returns focus to the compact rail after Escape closes its drawer", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const listButton = document.body.querySelector(
      'button[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    listButton.focus();
    await act(async () => {
      listButton.click();
      await Promise.resolve();
    });

    try {
      const drawer = document.body.querySelector(
        "[data-agent-compact-drawer]"
      ) as HTMLElement;
      expect(listButton.isConnected).toBe(true);
      expect(drawer).toContainElement(document.activeElement as HTMLElement);

      await act(async () => {
        document.dispatchEvent(
          new KeyboardEvent("keydown", { key: "Escape", bubbles: true })
        );
        await Promise.resolve();
      });

      expect(
        document.body.querySelector("[data-agent-compact-drawer]")
      ).toBeNull();
      expect(document.activeElement).toBe(listButton);
    } finally {
      unmount();
    }
  });

  test("closes the compact chat list after selecting another conversation", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    const otherChat = useAgentStore.getState().createChat({
      title: "Review analytics",
      context: "projects/analytics",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const listButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      await act(async () => {
        listButton.click();
        await Promise.resolve();
      });
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeInstanceOf(
        HTMLElement
      );

      const rowButton = document.body.querySelector(
        `[data-agent-compact-drawer] [data-agent-chat-row="${otherChat.id}"] button`
      ) as HTMLButtonElement;
      act(() => rowButton.click());

      expect(useAgentStore.getState().currentChatId).toBe(otherChat.id);
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
      expect(document.body.querySelector("[data-agent-chat-panels]")).toBeInstanceOf(
        HTMLElement
      );
    } finally {
      unmount();
    }
  });

  test("closes the compact chat list after creating a chat in a context group", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    useAgentStore.getState().createChat({
      title: "Previous analytics",
      context: "projects/analytics",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const listButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      await act(async () => {
        listButton.click();
        await Promise.resolve();
      });

      const newChatButton = document.body.querySelector(
        '[data-agent-compact-drawer] [data-agent-new-chat-for-context="projects/analytics"]'
      ) as HTMLButtonElement;
      act(() => newChatButton.click());

      expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).toContain("Analytics");
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
      act(() => {
        (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
      });

      const chatId = useAgentStore.getState().currentChatId!;
      expect(useAgentStore.getState().getChat(chatId)?.context).toBe(
        "projects/analytics"
      );
      expect(useAgentStore.getState().getMessages(chatId)).toEqual([]);
      expect(document.body.querySelector("[data-agent-compact-drawer]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test("opens the mobile agent directly in its current conversation", async () => {
    useAgentStore.setState({
      visible: false,
      minimized: false,
      size: { width: 479, height: 450 },
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      await act(async () => {
        useAgentStore.getState().open();
        await Promise.resolve();
      });

      expect(
        document.body.querySelector("[data-agent-mobile-conversation]")
      ).toBeInstanceOf(HTMLDivElement);
      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeNull();
      expect(
        document.body.querySelector("[data-agent-compact-rail]")
      ).toBeNull();

      const back = document.body.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      act(() => back.click());
      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeInstanceOf(HTMLDivElement);
    } finally {
      unmount();
    }
  });

  test("keeps the mobile chat list after restoring on a contextual page", () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();

      const back = document.body.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      act(() => back.click());
      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeInstanceOf(HTMLDivElement);

      mocks.pageResource = {
        kind: "project",
        name: "projects/orders",
        project: "projects/orders",
      };
      act(() => useAgentStore.getState().minimize());
      act(() => useAgentStore.getState().restore());
      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeInstanceOf(HTMLDivElement);
      expect(document.body.querySelector("[data-agent-contextual-start]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test("restores the mobile chat list when there is no contextual resource", () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    try {
      render();

      const back = document.body.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      act(() => back.click());
      act(() => useAgentStore.getState().minimize());
      act(() => useAgentStore.getState().restore());

      expect(
        document.body.querySelector("[data-agent-mobile-chat-list]")
      ).toBeInstanceOf(HTMLDivElement);
      expect(
        document.body.querySelector("[data-agent-contextual-start]")
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("keeps title selection separate from rename in the mobile chat list", () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const chatId = useAgentStore.getState().currentChatId!;

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const back = document.body.querySelector(
      '[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    act(() => back.click());

    const row = document.body.querySelector(
      `[data-agent-chat-row="${chatId}"] button`
    ) as HTMLButtonElement;
    act(() => row.click());

    expect(
      document.body.querySelector("[data-agent-inline-rename-input]")
    ).toBeNull();
    expect(
      document.body.querySelector("[data-agent-mobile-conversation]")
    ).toBeInstanceOf(HTMLDivElement);

    unmount();
  });

  test("returns an archived mobile conversation to archived chats", () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const activeChatId = useAgentStore.getState().currentChatId!;
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived mobile chat",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const conversationBack = document.body.querySelector(
      '[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    act(() => conversationBack.click());

    const archiveButton = document.body.querySelector(
      '[aria-label="agent.archived-only-chats"]'
    ) as HTMLButtonElement;
    act(() => archiveButton.click());

    const archivedChatButton = document.body.querySelector(
      `[data-agent-chat-row="${archivedChat.id}"] button`
    ) as HTMLButtonElement;
    act(() => archivedChatButton.click());

    const archivedConversationBack = document.body.querySelector(
      '[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    act(() => archivedConversationBack.click());

    expect(
      document.body.querySelector("[data-agent-mobile-chat-list]")
    ).toBeInstanceOf(HTMLDivElement);
    expect(
      document.body.querySelector(`[data-agent-chat-row="${archivedChat.id}"]`)
    ).toBeInstanceOf(HTMLDivElement);
    expect(
      document.body.querySelector(`[data-agent-chat-row="${activeChatId}"]`)
    ).toBeNull();

    unmount();
  });

  test("restores and deletes archived chats from the mobile archive view", async () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const store = useAgentStore.getState();
    const restoredChat = store.createChat({
      title: "Restore me",
      archived: true,
      select: false,
    });
    const deletedChat = store.createChat({
      title: "Delete me",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const back = document.body.querySelector(
      '[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    act(() => back.click());

    const archiveButton = document.body.querySelector(
      '[aria-label="agent.archived-only-chats"]'
    ) as HTMLButtonElement;
    act(() => archiveButton.click());

    await selectChatMenuAction(restoredChat.id, "[data-agent-unarchive-chat]");
    expect(useAgentStore.getState().getChat(restoredChat.id)?.archived).toBe(
      false
    );

    await selectChatMenuAction(deletedChat.id, "[data-agent-delete-chat]");
    const confirmButton = Array.from(
      document.body.querySelectorAll("button")
    ).find((button) => button.textContent === "common.confirm");
    act(() => confirmButton?.click());
    expect(useAgentStore.getState().getChat(deletedChat.id)).toBeNull();

    unmount();
  });

  test("opens the compact chat list before renaming the selected chat", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    const chatId = useAgentStore.getState().currentChatId!;

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const listButton = document.body.querySelector(
      'button[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    await act(async () => {
      listButton.click();
      await Promise.resolve();
    });

    const input = await beginChatRename(chatId);
    expect(input).toBeInstanceOf(HTMLInputElement);
    expect(useAgentStore.getState().currentChatId).toBe(chatId);

    unmount();
  });

  test("binds the initial empty chat when opened under a project", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    expect(useAgentStore.getState().getChat(chatId)?.context).toBeUndefined();

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(useAgentStore.getState().getChat(chatId)?.context).toBe(
      "projects/orders"
    );
    unmount();
  });

  test("binds a new empty chat to the current resource context", async () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const trigger = document.body.querySelector(
      "[aria-label='agent.new-chat']"
    ) as HTMLButtonElement | null;
    await act(async () => trigger?.click());

    expect(useAgentStore.getState().currentChatId).toBeNull();
    act(() => {
      (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
    });

    const chatId = useAgentStore.getState().currentChatId!;
    expect(useAgentStore.getState().getChat(chatId)?.context).toBe(
      "projects/orders"
    );
    expect(useAgentStore.getState().getMessages(chatId)).toEqual([]);
    unmount();
  });

  test("keeps context groups in a stable workspace-first order", () => {
    const store = useAgentStore.getState();
    const legacyChat = store.createChat({
      title: "Legacy chat",
      select: false,
    });
    const workspaceChat = store.createChat({
      title: "List databases",
      context: "workspaces/acme",
      select: false,
    });
    const analyticsChat = store.createChat({
      title: "Investigate analytics",
      context: "projects/analytics",
      select: false,
    });
    useAgentStore.setState((state) => ({
      chats: state.chats.map((chat) => {
        if (chat.id === legacyChat.id) return { ...chat, updatedTs: 4_000 };
        if (chat.id === analyticsChat.id) return { ...chat, updatedTs: 3_000 };
        if (chat.id === workspaceChat.id) return { ...chat, updatedTs: 2_000 };
        return { ...chat, updatedTs: 1_000 };
      }),
    }));

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const groups = Array.from(
      document.body.querySelectorAll("[data-agent-chat-group]")
    ) as HTMLDivElement[];
    expect(groups.map((group) => group.dataset.agentChatGroup)).toEqual([
      "workspaces/acme",
      "projects/orders",
      "projects/analytics",
      "unscoped",
    ]);
    expect(groups[0]?.textContent).toContain("Acme workspace");
    expect(groups[1]?.textContent).toContain("Order service");
    expect(groups[2]?.textContent).toContain("Analytics");
    expect(groups[3]?.textContent).toContain("agent.other-chats");
    expect(groups[3]?.textContent).toContain("Legacy chat");
    expect(
      document.body.querySelector("[data-agent-chat-updated-ts]")
    ).toBeNull();

    unmount();
  });

  test.each([
    { name: "large sidebar", width: 750, listSelector: "[data-agent-chat-sidebar]" },
    { name: "compact drawer", width: 700, listSelector: "[data-agent-compact-drawer]" },
    { name: "mobile chat list", width: 479, listSelector: "[data-agent-mobile-chat-list]" },
  ])("exposes full clipped group and chat titles in the $name", async ({
    width,
    listSelector,
  }) => {
    useAgentStore.setState({ size: { width, height: 450 } });
    const longTitle =
      "How do I connect the orders database and review every pending change?";
    const chat = useAgentStore.getState().createChat({
      title: longTitle,
      context: "projects/orders",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      if (width === 700) {
        const listButton = document.body.querySelector(
          '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        await act(async () => {
          listButton.click();
          await Promise.resolve();
        });
      } else if (width === 479) {
        const backButton = document.body.querySelector(
          '[aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        act(() => backButton.click());
      }

      const list = document.body.querySelector(listSelector) as HTMLElement;
      const groupToggle = list.querySelector(
        '[data-agent-chat-group-toggle="projects/orders"]'
      ) as HTMLButtonElement;
      const chatButton = list.querySelector(
        `[data-agent-chat-row="${chat.id}"] button`
      ) as HTMLButtonElement;
      expect(groupToggle).toHaveAttribute("title", "Order service");
      expect(chatButton).toHaveAttribute("title", longTitle);
    } finally {
      unmount();
    }
  });

  test.each([
    { name: "large sidebar", width: 750, listSelector: "[data-agent-chat-sidebar]" },
    { name: "compact drawer", width: 700, listSelector: "[data-agent-compact-drawer]" },
    { name: "mobile chat list", width: 479, listSelector: "[data-agent-mobile-chat-list]" },
  ])("uses the full chat-list width for group and chat rows in the $name", async ({
    width,
    listSelector,
  }) => {
    useAgentStore.setState({ size: { width, height: 450 } });
    const chat = useAgentStore.getState().createChat({
      title: "Inspect analytics",
      context: "projects/analytics",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      if (width === 700) {
        const listButton = document.body.querySelector(
          '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        await act(async () => {
          listButton.click();
          await Promise.resolve();
        });
      } else if (width === 479) {
        const backButton = document.body.querySelector(
          '[aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        act(() => backButton.click());
      }

      const list = document.body.querySelector(listSelector) as HTMLElement;
      const header = list.querySelector(
        '[data-agent-chat-group-header="projects/analytics"]'
      ) as HTMLElement;
      const children = list.querySelector(
        '[data-agent-chat-group-children="projects/analytics"]'
      ) as HTMLElement;
      const row = children.querySelector(
        `[data-agent-chat-row="${chat.id}"]`
      ) as HTMLElement;

      expect(header).toHaveClass("w-full");
      expect(header).not.toHaveClass("px-2");
      expect(children).not.toHaveClass("pl-7");
      expect(row).toHaveClass("w-full");
      expect(row).toHaveClass("pl-5");
    } finally {
      unmount();
    }
  });

  test("toggles a group in the full chat sidebar", () => {
    const chat = useAgentStore.getState().createChat({
      title: "Investigate analytics",
      context: "projects/analytics",
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const toggle = document.body.querySelector(
        '[data-agent-chat-group-toggle="projects/analytics"]'
      ) as HTMLButtonElement;
      const header = document.body.querySelector(
        '[data-agent-chat-group-header="projects/analytics"]'
      ) as HTMLDivElement;
      expect(toggle).toHaveAttribute("aria-expanded", "true");
      const children = document.body.querySelector(
        '[data-agent-chat-group-children="projects/analytics"]'
      ) as HTMLDivElement;
      const chatRow = document.body.querySelector(
        `[data-agent-chat-row="${chat.id}"]`
      ) as HTMLDivElement;
      expect(header).toContainElement(toggle);
      expect(header).not.toContainElement(children);
      expect(children).not.toHaveClass("ml-8", "pl-4");
      expect(children).toContainElement(chatRow);
      expect(chatRow).toHaveClass("w-full", "min-h-10");
      expect(toggle.querySelector("svg.lucide-folder-open")).toBeInstanceOf(
        SVGElement
      );
      expect(toggle.querySelector("svg.lucide-chevron-right")).toBeNull();

      act(() => toggle.click());
      expect(toggle).toHaveAttribute("aria-expanded", "false");
      expect(toggle.querySelector("svg.lucide-folder")).toBeInstanceOf(
        SVGElement
      );
      expect(toggle.querySelector("svg.lucide-folder-open")).toBeNull();
      expect(
        document.body.querySelector(`[data-agent-chat-row="${chat.id}"]`)
      ).toBeNull();

      act(() => toggle.click());
      expect(toggle).toHaveAttribute("aria-expanded", "true");
      expect(toggle.querySelector("svg.lucide-folder-open")).toBeInstanceOf(
        SVGElement
      );
      expect(
        document.body.querySelector(`[data-agent-chat-row="${chat.id}"]`)
      ).toBeInstanceOf(HTMLDivElement);
    } finally {
      unmount();
    }
  });

  test("toggles a group in the compact chat drawer", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    const chat = useAgentStore.getState().createChat({
      title: "Investigate analytics",
      context: "projects/analytics",
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const listButton = document.body.querySelector(
        'button[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      await act(async () => {
        listButton.click();
        await Promise.resolve();
      });

      const drawer = document.body.querySelector(
        "[data-agent-compact-drawer]"
      ) as HTMLElement;
      expect(drawer).toBeInstanceOf(HTMLElement);
      const toggle = drawer.querySelector(
        '[data-agent-chat-group-toggle="projects/analytics"]'
      ) as HTMLButtonElement;
      expect(toggle).toHaveAttribute("aria-expanded", "true");

      act(() => toggle.click());
      expect(toggle).toHaveAttribute("aria-expanded", "false");
      expect(drawer.querySelector(`[data-agent-chat-row="${chat.id}"]`)).toBeNull();
    } finally {
      unmount();
    }
  });

  test("toggles a group in the mobile chat list", () => {
    useAgentStore.setState({ size: { width: 479, height: 450 } });
    const chat = useAgentStore.getState().createChat({
      title: "Investigate analytics",
      context: "projects/analytics",
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const back = document.body.querySelector(
        '[aria-label="agent.chat-list-label"]'
      ) as HTMLButtonElement;
      act(() => back.click());

      const list = document.body.querySelector(
        "[data-agent-mobile-chat-list]"
      ) as HTMLDivElement;
      expect(list).toBeInstanceOf(HTMLDivElement);
      const toggle = list.querySelector(
        '[data-agent-chat-group-toggle="projects/analytics"]'
      ) as HTMLButtonElement;
      const header = list.querySelector(
        '[data-agent-chat-group-header="projects/analytics"]'
      ) as HTMLDivElement;
      expect(toggle).toHaveAttribute("aria-expanded", "true");
      const children = list.querySelector(
        '[data-agent-chat-group-children="projects/analytics"]'
      ) as HTMLDivElement;
      const chatRow = list.querySelector(
        `[data-agent-chat-row="${chat.id}"]`
      ) as HTMLDivElement;
      expect(header).toContainElement(toggle);
      expect(header).not.toContainElement(children);
      expect(children).not.toHaveClass("ml-8", "pl-4");
      expect(children).toContainElement(chatRow);
      expect(chatRow).toHaveClass("w-full", "min-h-10");

      act(() => toggle.click());
      expect(toggle).toHaveAttribute("aria-expanded", "false");
      expect(list.querySelector(`[data-agent-chat-row="${chat.id}"]`)).toBeNull();
    } finally {
      unmount();
    }
  });

  test("creates a new chat in the selected context group", async () => {
    const store = useAgentStore.getState();
    store.createChat({
      title: "Workspace chat",
      context: "workspaces/acme",
      select: false,
    });
    store.createChat({
      title: "Analytics chat",
      context: "projects/analytics",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const workspaceToggle = document.body.querySelector(
      '[data-agent-chat-group-toggle="workspaces/acme"]'
    ) as HTMLButtonElement;
    act(() => workspaceToggle.click());
    expect(workspaceToggle).toHaveAttribute("aria-expanded", "false");

    const workspaceButton = document.body.querySelector(
      '[data-agent-new-chat-for-context="workspaces/acme"]'
    ) as HTMLButtonElement | null;
    expect(workspaceButton).toBeInstanceOf(HTMLButtonElement);
    await act(async () => workspaceButton?.click());

    const workspaceChatId = useAgentStore.getState().currentChatId!;
    expect(useAgentStore.getState().getChat(workspaceChatId)?.context).toBe(
      "workspaces/acme"
    );
    expect(useAgentStore.getState().getMessages(workspaceChatId)).toEqual([]);
    expect(
      document.body.querySelector(
        '[data-agent-chat-group-toggle="workspaces/acme"]'
      )
    ).toHaveAttribute("aria-expanded", "true");
    expect(
      document.body.querySelector(
        `[data-agent-chat-row="${workspaceChatId}"]`
      )
    ).toBeInstanceOf(HTMLDivElement);

    const projectButton = document.body.querySelector(
      '[data-agent-new-chat-for-context="projects/analytics"]'
    ) as HTMLButtonElement | null;
    expect(projectButton).toBeInstanceOf(HTMLButtonElement);
    await act(async () => projectButton?.click());
    expect(document.body.querySelector('[data-agent-contextual-start="project"]')?.textContent).toContain("Analytics");
    act(() => {
      (document.body.querySelector('[data-agent-contextual-suggestion]') as HTMLButtonElement).click();
    });

    const projectChatId = useAgentStore.getState().currentChatId!;
    expect(useAgentStore.getState().getChat(projectChatId)?.context).toBe(
      "projects/analytics"
    );
    expect(useAgentStore.getState().getMessages(projectChatId)).toEqual([]);
    expect(
      document.body.querySelector('[data-agent-new-chat-for-context="unscoped"]')
    ).toBeNull();

    unmount();
  });

  test.each([
    ["large", 750],
    ["mobile", 470],
  ])("gives the %s group new-chat button a stronger hover background", (_layout, width) => {
    useAgentStore.setState({ size: { width, height: 450 } });
    useAgentStore.getState().createChat({
      title: "Analytics chat",
      context: "projects/analytics",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      if (width === 470) {
        const backButton = document.body.querySelector(
          '[aria-label="agent.chat-list-label"]'
        ) as HTMLButtonElement;
        act(() => backButton.click());
      }

      const groupHeader = document.body.querySelector(
        '[data-agent-chat-group-header="projects/analytics"]'
      );
      const newChatButton = groupHeader?.querySelector(
        '[data-agent-new-chat-for-context="projects/analytics"]'
      );
      expect(newChatButton).toHaveClass("hover:bg-control/25");
    } finally {
      unmount();
    }
  });

  test("hides new-chat actions from the archived full sidebar", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const archiveButton = document.body.querySelector(
      '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
    ) as HTMLButtonElement;
    expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
    act(() => archiveButton.click());

    try {
      const actions = document.body.querySelector(
        "[data-agent-chat-sidebar-actions]"
      );
      expect(
        actions?.querySelector("[aria-label='agent.new-chat']")
      ).toBeNull();
      expect(
        actions?.querySelector("[aria-label='common.more']")
      ).toBeInstanceOf(HTMLButtonElement);
      expect(
        document.body.querySelector(
          '[data-agent-new-chat-for-context="projects/analytics"]'
        )
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("only offers the destructive Archive all action in the active list menu", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const list = document.body.querySelector("[data-agent-chat-sidebar]");
      const menuButton = list?.querySelector(
        '[data-agent-chat-sidebar-actions] [aria-label="common.more"]'
      ) as HTMLButtonElement;
      expect(menuButton).toBeInstanceOf(HTMLButtonElement);

      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });

      const menu = getLayerRoot("agent").querySelector(
        "[data-agent-dropdown-menu-content]"
      ) as HTMLElement;
      expect(menu).toBeInstanceOf(HTMLElement);
      expect(menu.querySelectorAll('[role="menuitem"]')).toHaveLength(1);
      expect(menu.querySelector("[data-agent-archive-all-chats]")).toHaveClass(
        "text-error"
      );
      expect(menu.querySelector("[data-agent-chat-list-mode]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test("only offers Unarchive all and Delete all in the archived list menu", async () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });
    useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      act(() => archiveButton.click());
      const list = document.body.querySelector("[data-agent-chat-sidebar]");
      const menuButton = list?.querySelector(
        '[data-agent-chat-sidebar-actions] [aria-label="common.more"]'
      ) as HTMLButtonElement;
      expect(menuButton).toBeInstanceOf(HTMLButtonElement);

      await act(async () => {
        menuButton.click();
        await Promise.resolve();
      });

      const menu = getLayerRoot("agent").querySelector(
        "[data-agent-dropdown-menu-content]"
      ) as HTMLElement;
      expect(menu).toBeInstanceOf(HTMLElement);
      expect(menu.querySelectorAll('[role="menuitem"]')).toHaveLength(2);
      expect(menu.querySelector("[data-agent-unarchive-all-chats]")).toBeInstanceOf(
        HTMLElement
      );
      expect(menu.querySelector("[data-agent-delete-all-chats]")).toHaveClass(
        "text-error"
      );
      expect(menu.querySelector("[data-agent-chat-list-mode]")).toBeNull();
    } finally {
      unmount();
    }
  });

  test("toggles a group in archived chats", async () => {
    const chat = useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
      act(() => archiveButton.click());

      const toggle = document.body.querySelector(
        '[data-agent-chat-group-toggle="projects/analytics"]'
      ) as HTMLButtonElement;
      expect(toggle).toHaveAttribute("aria-expanded", "true");

      act(() => toggle.click());
      expect(toggle).toHaveAttribute("aria-expanded", "false");
      expect(
        document.body.querySelector(`[data-agent-chat-row="${chat.id}"]`)
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("keeps the compact rail new-chat action while viewing archived chats", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });
    useAgentStore.getState().createChat({
      title: "Archived analytics",
      context: "projects/analytics",
      archived: true,
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const archiveButton = document.body.querySelector(
      "[data-agent-compact-rail] [aria-label='agent.archived-only-chats']"
    ) as HTMLButtonElement;
    await act(async () => {
      archiveButton.click();
      await Promise.resolve();
    });

    try {
      const rail = document.body.querySelector("[data-agent-compact-rail]");
      expect(
        rail?.querySelector("[aria-label='agent.new-chat']")
      ).toBeInstanceOf(HTMLButtonElement);
      expect(
        document.body.querySelector(
          '[data-agent-new-chat-for-context="projects/analytics"]'
        )
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("hydrates uncached project chat contexts before rendering their labels", async () => {
    mocks.projectsByName = {};
    mocks.batchFetchProjects.mockImplementation(async () => {
      mocks.projectsByName = {
        "projects/new-project-8i6r": { title: "New project" },
      };
      return [];
    });
    useAgentStore.getState().createChat({
      title: "Set up collaboration",
      context: "projects/new-project-8i6r",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(mocks.batchFetchProjects).toHaveBeenCalledWith(
          ["projects/new-project-8i6r"],
          true
        )
      );
    });
    render();
    expect(document.body.textContent).toContain("New project");
    unmount();
  });

  test("keeps the project resource id when an old chat's project is unavailable", async () => {
    mocks.projectsByName = {};
    useAgentStore.getState().createChat({
      title: "Set up collaboration",
      context: "projects/deleted-project",
      select: false,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    await act(async () => {
      await vi.waitFor(() =>
        expect(mocks.batchFetchProjects).toHaveBeenCalledWith(
          ["projects/deleted-project"],
          true
        )
      );
    });
    expect(document.body.textContent).toContain("deleted-project");
    unmount();
  });

  test("marks the active chat when ai_enabled is false", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    mocks.aiEnabled = false;
    useAgentStore.setState({ visible: false });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();
    await act(async () => {
      useAgentStore.getState().toggle();
      await Promise.resolve();
    });

    expect(useAgentStore.getState().getChat(chatId)).toMatchObject({
      requiresAIConfiguration: true,
    });

    unmount();
  });

  test("falls back to the AI setting when ai_enabled is unavailable", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    mocks.aiEnabled = undefined;
    mocks.getOrFetchSettingByName.mockResolvedValue({
      value: {
        value: {
          case: "ai",
          value: { enabled: false },
        },
      },
    });
    useAgentStore.setState({ visible: false });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();
    await act(async () => {
      useAgentStore.getState().toggle();
      await Promise.resolve();
    });

    expect(mocks.getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.AI,
      true
    );
    expect(useAgentStore.getState().getChat(chatId)).toMatchObject({
      requiresAIConfiguration: true,
    });
    unmount();
  });

  test("uses an accessible panel separator for the chat sidebar", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const separator = document.body.querySelector(
      "[data-agent-sidebar-resize]"
    );

    expect(separator).toHaveAttribute("role", "separator");
    expect(separator).toHaveAttribute("data-separator");
    expect(separator).not.toBeInstanceOf(HTMLButtonElement);

    unmount();
  });

  test("uses compact borderless controls in the chat sidebar header", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const actions = document.body.querySelector(
      "[data-agent-chat-sidebar-actions]"
    ) as HTMLDivElement | null;
    const controls = actions?.querySelectorAll("button");

    expect(controls).toHaveLength(2);
    expect(controls?.[0]?.className).toContain("h-6");
    expect(controls?.[0]?.className).not.toContain("border");
    expect(controls?.[0]?.querySelector("svg")?.className.baseVal).toContain(
      "size-3.5"
    );
    expect(controls?.[1]?.className).toContain("size-6");
    expect(controls?.[1]?.className).not.toContain("border");
    expect(controls?.[1]?.querySelector("svg")?.className.baseVal).toContain(
      "size-3.5"
    );

    unmount();
  });

  test("keeps the selected chat row stable while its actions are revealed", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const title = document.body.querySelector("[data-agent-chat-title]");
    const titleButton = title?.closest("button");

    expect(titleButton?.className).toContain("hover:bg-transparent");

    unmount();
  });

  test("mounts the agent shell into the agent layer root", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const agentRoot = document.getElementById("bb-react-layer-agent");
    expect(agentRoot).toBeInstanceOf(HTMLDivElement);
    expect(agentRoot?.querySelector("[data-agent-window]")).toBeInstanceOf(
      HTMLDivElement
    );

    unmount();
  });

  test("keeps chat status and token usage out of the window header", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.setState({
      chats: useAgentStore
        .getState()
        .chats.map((chat) =>
          chat.id === chatId
            ? { ...chat, status: "running", totalTokensUsed: 1234 }
            : chat
        ),
    });

    mocks.useTranslation.mockReturnValue({
      t: (key: string, values?: Record<string, unknown>) =>
        key === "agent.chat-total-tokens"
          ? `Tokens used: ${values?.count ?? ""}`
          : key,
      i18n: { language: "en-US" },
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const header = document.body.querySelector(
      "[data-agent-window-header]"
    ) as HTMLDivElement | null;

    expect(header).toBeInstanceOf(HTMLDivElement);
    expect(header?.textContent).toContain("agent.assistant-title");
    expect(header?.textContent).not.toContain("agent.chat-status-running");
    expect(header?.textContent).not.toContain("Tokens used");

    unmount();
  });

  test("mounts agent menu and delete dialog into the agent layer root", async () => {
    const archivedChat = useAgentStore.getState().createChat({
      title: "Archived chat",
      archived: true,
      select: false,
    });

    useAgentStore.setState({
      chats: useAgentStore.getState().chats.map((chat) => {
        if (chat.id === archivedChat.id) {
          return { ...chat, updatedTs: 2000 };
        }
        return chat;
      }),
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
      act(() => archiveButton.click());

      const row = document.body.querySelector(
        `[data-agent-chat-row="${archivedChat.id}"]`
      );
      const menuButton = row?.querySelector(
        "[data-agent-archived-chat-menu]"
      ) as HTMLButtonElement;
      expect(menuButton).toBeInstanceOf(HTMLButtonElement);
      await act(async () => {
        menuButton.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
        await Promise.resolve();
      });

      const agentRoot = document.getElementById("bb-react-layer-agent");
      const overlayRoot = document.getElementById("bb-react-layer-overlay");
      expect(
        agentRoot?.querySelector("[data-agent-dropdown-menu-content]")
      ).toBeInstanceOf(HTMLDivElement);
      expect(agentRoot?.querySelector("[data-agent-delete-chat]")).toBeInstanceOf(
        HTMLElement
      );
      expect(
        overlayRoot?.querySelector("[data-agent-dropdown-menu-content]") ?? null
      ).toBeNull();

      const deleteButton = agentRoot?.querySelector(
        "[data-agent-delete-chat]"
      ) as HTMLElement;
      await act(async () => {
        deleteButton.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
        await Promise.resolve();
      });

      expect(
        agentRoot?.querySelector("[data-agent-dialog-content]")
      ).toBeInstanceOf(HTMLDivElement);
      expect(agentRoot?.textContent?.includes("agent.delete-chat-confirmation")).toBe(
        true
      );
      expect(
        overlayRoot?.querySelector("[data-agent-dialog-content]") ?? null
      ).toBeNull();
    } finally {
      unmount();
    }
  });

  test("mounts agent tooltip content into the agent layer root", async () => {
    vi.useFakeTimers();

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const trigger = document.body.querySelector(
      "[aria-label='agent.new-chat']"
    ) as HTMLButtonElement | null;

    expect(trigger).toBeInstanceOf(HTMLButtonElement);

    await act(async () => {
      trigger?.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      vi.advanceTimersByTime(100);
    });

    const agentRoot = document.getElementById("bb-react-layer-agent");
    const overlayRoot = document.getElementById("bb-react-layer-overlay");

    expect(
      agentRoot?.querySelector("[data-agent-tooltip-content]")
    ).toBeInstanceOf(HTMLDivElement);
    expect(agentRoot?.textContent).toContain("agent.new-chat");
    expect(
      overlayRoot?.querySelector("[data-agent-tooltip-content]") ?? null
    ).toBeNull();

    unmount();
  });

  test("selects the next archived chat after deleting the current archived chat", async () => {
    const activeChatId = useAgentStore.getState().currentChatId!;
    const firstArchivedChat = useAgentStore.getState().createChat({
      title: "First archived",
      archived: true,
      select: false,
    });
    const secondArchivedChat = useAgentStore.getState().createChat({
      title: "Second archived",
      archived: true,
      select: false,
    });

    useAgentStore.setState({
      chats: useAgentStore.getState().chats.map((chat) => {
        if (chat.id === activeChatId) {
          return { ...chat, updatedTs: 3000 };
        }
        if (chat.id === firstArchivedChat.id) {
          return { ...chat, updatedTs: 2000 };
        }
        if (chat.id === secondArchivedChat.id) {
          return { ...chat, updatedTs: 1000 };
        }
        return chat;
      }),
      currentChatId: firstArchivedChat.id,
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    try {
      const archiveButton = document.body.querySelector(
        '[data-agent-compact-rail] [aria-label="agent.archived-only-chats"]'
      ) as HTMLButtonElement;
      expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
      act(() => archiveButton.click());

      const firstRow = document.body.querySelector(
        `[data-agent-chat-row="${firstArchivedChat.id}"]`
      );
      const menuButton = firstRow?.querySelector(
        "[data-agent-archived-chat-menu]"
      ) as HTMLButtonElement;
      expect(menuButton).toBeInstanceOf(HTMLButtonElement);
      await act(async () => {
        menuButton.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
        await Promise.resolve();
      });

      const deleteButton = document.body.querySelector(
        "[data-agent-delete-chat]"
      ) as HTMLElement;
      expect(deleteButton).toBeInstanceOf(HTMLElement);
      act(() => {
        deleteButton.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
      });

      const confirmButton = Array.from(
        document.body.querySelectorAll("button")
      ).find((button) => button.textContent === "common.confirm");

      act(() => {
        confirmButton?.dispatchEvent(
          new MouseEvent("click", { bubbles: true, cancelable: true })
        );
      });

      expect(useAgentStore.getState().getChat(firstArchivedChat.id)).toBeNull();
      expect(useAgentStore.getState().currentChatId).toBe(secondArchivedChat.id);
    } finally {
      unmount();
    }
  });

  test.each([1200, 700, 390])(
    "confirms archiving the chosen chat from its Agent menu at width %s",
    async (width) => {
      useAgentStore.setState({ size: { width, height: 600 } });
      const store = useAgentStore.getState();
      const currentChatId = store.currentChatId!;
      const targetChat = store.createChat({
        title: "Archive this conversation",
        context: "projects/orders",
        select: false,
      });
      const { render, unmount } = renderIntoContainer(<AgentWindow />);
      try {
        render();
        if (width < 480) {
          act(() => {
            (
              document.body.querySelector(
                '[aria-label="agent.chat-list-label"]'
              ) as HTMLButtonElement
            ).click();
          });
        } else if (width < 720) {
          await act(async () => {
            (
              document.body.querySelector(
                '[data-agent-compact-rail] [aria-label="agent.chat-list-label"]'
              ) as HTMLButtonElement
            ).click();
          });
        }

        await selectChatMenuAction(targetChat.id, "[data-agent-archive-chat]");

        expect(store.getChat(targetChat.id)?.archived).toBe(false);
        expect(useAgentStore.getState().currentChatId).toBe(currentChatId);
        const agentRoot = getLayerRoot("agent");
        let confirmation = agentRoot.querySelector(
          "[data-agent-archive-chat-confirmation]"
        ) as HTMLElement;
        expect(confirmation).toHaveAttribute("role", "dialog");
        expect(confirmation.textContent).toContain("agent.archive-chat-confirmation");
        expect(
          getLayerRoot("overlay").querySelector(
            "[data-agent-archive-chat-confirmation]"
          )
        ).toBeNull();

        await act(async () => {
          Array.from(confirmation.querySelectorAll("button"))
            .find((button) => button.textContent === "common.cancel")!
            .click();
        });
        expect(store.getChat(targetChat.id)?.archived).toBe(false);
        expect(agentRoot.querySelector("[data-agent-archive-chat-confirmation]")).toBeNull();

        await selectChatMenuAction(targetChat.id, "[data-agent-archive-chat]");
        confirmation = agentRoot.querySelector(
          "[data-agent-archive-chat-confirmation]"
        ) as HTMLElement;
        await act(async () => {
          Array.from(confirmation.querySelectorAll("button"))
            .find((button) => button.textContent === "common.archive")!
            .click();
        });
        expect(store.getChat(targetChat.id)?.archived).toBe(true);
        expect(store.getChat(currentChatId)?.archived).toBe(false);
        expect(useAgentStore.getState().currentChatId).toBe(currentChatId);
      } finally {
        unmount();
      }
    }
  );

  test("keeps the running chat selected when archiving it from the active list", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().startChatRun(chatId, {
      path: "/projects/demo",
      title: "Demo",
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    await selectChatMenuAction(chatId, "[data-agent-archive-chat]");

    const confirmation = getLayerRoot("agent").querySelector(
      "[data-agent-archive-chat-confirmation]"
    ) as HTMLElement;
    expect(useAgentStore.getState().getChat(chatId)?.archived).toBe(false);
    await act(async () => {
      Array.from(confirmation.querySelectorAll("button"))
        .find((button) => button.textContent === "common.archive")!
        .click();
    });

    expect(useAgentStore.getState().getChat(chatId)?.archived).toBe(true);
    expect(useAgentStore.getState().currentChatId).toBe(chatId);

    unmount();
  });

  test("vertically aligns each chat title with its action menu", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const menuButton = document.body.querySelector(
      "[data-agent-chat-menu]"
    ) as HTMLButtonElement | null;
    const title = document.body.querySelector(
      "[data-agent-chat-title]"
    ) as HTMLDivElement | null;
    const rowContent = title?.parentElement?.parentElement;

    expect(menuButton).toBeInstanceOf(HTMLButtonElement);
    expect(rowContent).toBeInstanceOf(HTMLDivElement);
    expect(rowContent?.className).toContain("items-center");

    unmount();
  });

  test("anchors the window at the bottom right without a drag affordance", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const windowEl = document.body.querySelector(
      "[data-agent-window]"
    ) as HTMLDivElement | null;
    const header = document.body.querySelector(
      "[data-agent-window-header]"
    ) as HTMLDivElement | null;

    expect(windowEl).toBeInstanceOf(HTMLDivElement);
    expect(header).toBeInstanceOf(HTMLDivElement);
    expect(windowEl?.style.right).toBe("16px");
    expect(windowEl?.style.bottom).toBe("16px");
    expect(windowEl?.style.left).toBe("");
    expect(windowEl?.style.top).toBe("");
    expect(header?.className).not.toContain("cursor-move");
    expect(
      Array.from(
        document.body.querySelectorAll("[data-agent-window-resize-zone]"),
        (zone) => zone.getAttribute("data-agent-window-resize-zone")
      )
    ).toEqual(["n", "w", "nw"]);

    act(() => {
      header?.dispatchEvent(
        new PointerEvent("pointerdown", {
          bubbles: true,
          button: 0,
          clientX: 150,
          clientY: 150,
        })
      );
      document.dispatchEvent(
        new PointerEvent("pointermove", {
          bubbles: true,
          clientX: 180,
          clientY: 190,
        })
      );
      document.dispatchEvent(
        new PointerEvent("pointerup", {
          bubbles: true,
          clientX: 180,
          clientY: 190,
        })
      );
    });

    expect(windowEl?.style.right).toBe("16px");
    expect(windowEl?.style.bottom).toBe("16px");
    expect(windowEl?.style.left).toBe("");
    expect(windowEl?.style.top).toBe("");

    unmount();
  });

});
