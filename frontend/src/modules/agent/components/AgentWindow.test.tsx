import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
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
  getOrFetchSettingByName: vi.fn(),
  batchFetchProjects: vi.fn(),
  projectErrorsByName: {} as Record<string, Error | undefined>,
  currentContext: "projects/orders" as string | undefined,
  projectsByName: {
    "projects/orders": { title: "Order service" },
    "projects/analytics": { title: "Analytics" },
  } as Record<string, { title: string }>,
  workspace: { name: "workspaces/acme", title: "Acme workspace" },
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
    batchFetchProjects: mocks.batchFetchProjects,
    get projectsByName() {
      return mocks.projectsByName;
    },
    get projectErrorsByName() {
      return mocks.projectErrorsByName;
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

vi.mock("../logic/chatContext", () => ({
  getCurrentChatContext: () => mocks.currentContext,
}));

vi.mock("./AgentChat", () => ({
  AgentChat: () => <div data-testid="agent-chat" />,
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
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
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
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
  });
  useAgentStore.getState().setWorkspaceName(mocks.workspace.name);
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
  mocks.batchFetchProjects.mockReset();
  mocks.batchFetchProjects.mockResolvedValue([]);
  mocks.projectsByName = {
    "projects/orders": { title: "Order service" },
    "projects/analytics": { title: "Analytics" },
  };
  mocks.projectErrorsByName = {};
  mocks.currentContext = "projects/orders";

  ({ AgentWindow } = await import("./AgentWindow"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  document.body.innerHTML = "";
});

describe("AgentWindow", () => {
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

  test("keeps the full chat sidebar when the agent window has room for it", () => {
    useAgentStore.setState({ size: { width: 750, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    expect(
      document.body.querySelector("[data-agent-chat-sidebar]")
    ).toBeInstanceOf(HTMLElement);
    expect(
      document.body.querySelector("[data-agent-compact-rail]")
    ).toBeNull();

    unmount();
  });

  test("renders the compact drawer outside the clipped rail panel", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const listButton = document.body.querySelector(
      'button[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    await act(async () => {
      listButton.click();
      await Promise.resolve();
    });

    const drawer = document.body.querySelector(
      "[data-agent-compact-drawer]"
    );
    const body = document.body.querySelector("[data-agent-window-body]");
    expect(drawer?.parentElement).toBe(body);
    expect(
      document.body.querySelector("[data-agent-compact-rail]")
    ).toBeInstanceOf(HTMLElement);

    unmount();
  });

  test("layers the compact drawer above the chat panel and composer", async () => {
    useAgentStore.setState({ size: { width: 700, height: 450 } });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const listButton = document.body.querySelector(
      'button[aria-label="agent.chat-list-label"]'
    ) as HTMLButtonElement;
    await act(async () => {
      listButton.click();
      await Promise.resolve();
    });

    try {
      const body = document.body.querySelector("[data-agent-window-body]");
      const panels = document.body.querySelector("[data-agent-chat-panels]");
      const drawer = document.body.querySelector(
        "[data-agent-compact-drawer]"
      );
      const overlay = document.body.querySelector(
        "[data-agent-compact-overlay]"
      );
      const input = document.body.querySelector('[data-testid="agent-input"]');

      expect(panels?.parentElement).toBe(body);
      expect(panels?.contains(input)).toBe(true);
      expect(panels?.className).toContain("relative");
      expect(panels?.className).toContain("z-0");
      expect(overlay?.parentElement).toBe(body);
      expect(overlay?.className).toContain("absolute");
      expect(overlay?.className).toContain("inset-0");
      expect(overlay?.className).toContain("z-1");
      expect(drawer?.parentElement).toBe(body);
      expect(drawer?.className).toContain("z-2");
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

  test("keeps rename out of the mobile chat list", () => {
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

  test("restores and deletes archived chats from the mobile archive view", () => {
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

    const restoreButton = document.body.querySelector(
      `[data-agent-chat-row="${restoredChat.id}"] [data-agent-unarchive-chat]`
    ) as HTMLButtonElement;
    act(() => restoreButton.click());
    expect(useAgentStore.getState().getChat(restoredChat.id)?.archived).toBe(
      false
    );

    const deleteButton = document.body.querySelector(
      `[data-agent-chat-row="${deletedChat.id}"] [data-agent-delete-chat]`
    ) as HTMLButtonElement;
    act(() => deleteButton.click());
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

    const selectedChat = document.body.querySelector(
      `[data-agent-chat-row="${chatId}"] button`
    ) as HTMLButtonElement;
    expect(selectedChat).toBeInstanceOf(HTMLButtonElement);
    await act(async () => {
      selectedChat.click();
      await Promise.resolve();
    });

    expect(
      document.body.querySelector("[data-agent-inline-rename-input]")
    ).toBeInstanceOf(HTMLInputElement);

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

    const projectButton = document.body.querySelector(
      '[data-agent-new-chat-for-context="projects/analytics"]'
    ) as HTMLButtonElement | null;
    expect(projectButton).toBeInstanceOf(HTMLButtonElement);
    await act(async () => projectButton?.click());

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

    const moreButton = document.body.querySelector(
      "[aria-label='common.more']"
    ) as HTMLButtonElement;
    act(() => {
      moreButton.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    const archivedModeButton = document.body.querySelector(
      "[data-agent-chat-list-mode]"
    ) as HTMLDivElement;
    await act(async () => {
      archivedModeButton.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

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

  test("hides the compact rail new-chat action while viewing archived chats", async () => {
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
      ).toBeNull();
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

  test("checks AI configuration when opening an existing chat", async () => {
    const chatId = useAgentStore.getState().currentChatId!;
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

    const moreButton = document.body.querySelector(
      "[aria-label='common.more']"
    ) as HTMLButtonElement | null;

    act(() => {
      moreButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    const agentRoot = document.getElementById("bb-react-layer-agent");
    const overlayRoot = document.getElementById("bb-react-layer-overlay");

    expect(
      agentRoot?.querySelector("[data-agent-dropdown-menu-content]")
    ).toBeInstanceOf(HTMLDivElement);
    expect(
      agentRoot?.querySelector("[data-agent-chat-list-mode]")
    ).toBeInstanceOf(HTMLDivElement);
    expect(
      overlayRoot?.querySelector("[data-agent-dropdown-menu-content]") ?? null
    ).toBeNull();

    const archivedModeButton = document.body.querySelector(
      "[data-agent-chat-list-mode]"
    ) as HTMLDivElement | null;

    await act(async () => {
      archivedModeButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    const deleteButton = document.body.querySelector(
      "[data-agent-delete-chat]"
    ) as HTMLButtonElement | null;

    await act(async () => {
      deleteButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    expect(
      agentRoot?.querySelector("[data-agent-dialog-content]")
    ).toBeInstanceOf(HTMLDivElement);
    expect(
      agentRoot?.textContent?.includes("agent.delete-chat-confirmation")
    ).toBe(true);
    expect(
      overlayRoot?.querySelector("[data-agent-dialog-content]") ?? null
    ).toBeNull();

    unmount();
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

    const moreButton = document.body.querySelector(
      "[aria-label='common.more']"
    ) as HTMLButtonElement | null;

    act(() => {
      moreButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    const archivedModeButton = document.body.querySelector(
      "[data-agent-chat-list-mode]"
    ) as HTMLDivElement | null;

    await act(async () => {
      archivedModeButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
      await Promise.resolve();
    });

    const deleteButtons = Array.from(
      document.body.querySelectorAll("[data-agent-delete-chat]")
    ) as HTMLButtonElement[];

    expect(deleteButtons).toHaveLength(2);

    act(() => {
      deleteButtons[0]?.dispatchEvent(
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

    unmount();
  });

  test("keeps the running chat selected when archiving it from the active list", () => {
    const chatId = useAgentStore.getState().currentChatId!;
    useAgentStore.getState().startChatRun(chatId, {
      path: "/projects/demo",
      title: "Demo",
    });

    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const archiveButton = document.body.querySelector(
      "[data-agent-archive-chat]"
    ) as HTMLButtonElement | null;

    expect(archiveButton).toBeInstanceOf(HTMLButtonElement);

    act(() => {
      archiveButton?.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true })
      );
    });

    expect(useAgentStore.getState().getChat(chatId)?.archived).toBe(true);
    expect(useAgentStore.getState().currentChatId).toBe(chatId);

    unmount();
  });

  test("vertically aligns each chat title with its archive action", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);
    render();

    const archiveButton = document.body.querySelector(
      "[data-agent-archive-chat]"
    ) as HTMLButtonElement | null;
    const title = document.body.querySelector(
      "[data-agent-chat-title]"
    ) as HTMLDivElement | null;
    const rowContent = title?.parentElement?.parentElement;

    expect(archiveButton).toBeInstanceOf(HTMLButtonElement);
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
