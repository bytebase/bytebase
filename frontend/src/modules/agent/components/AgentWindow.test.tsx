import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
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
  AgentChat: ({
    contextLabel,
    onSelectSuggestion,
  }: {
    contextLabel?: string;
    onSelectSuggestion?: (starterId: "project-open-blockers") => void;
  }) => (
    <div data-testid="agent-chat" data-context-label={contextLabel}>
      <button
        type="button"
        data-testid="agent-empty-suggestion"
        onClick={() => onSelectSuggestion?.("project-open-blockers")}
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
  test("prefills a landing prompt for the current context without sending it", () => {
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
    expect(useAgentStore.getState().getDraft(chat.id).text).toBe(
      "agent.landing-starters.project-open-blockers"
    );
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
      const archiveButton = document.body.querySelector(
        `[data-agent-chat-row="${firstActiveChatId}"] [data-agent-archive-chat]`
      ) as HTMLButtonElement;
      act(() => archiveButton.click());
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

  test("offers Restore and Delete in one archived chat row menu", async () => {
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
      expect(
        agentRoot.querySelector("[data-agent-unarchive-chat]")
      ).toBeInstanceOf(HTMLElement);
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
      expect(chatRow).toHaveClass("w-full", "py-2");
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
      expect(chatRow).toHaveClass("w-full", "py-2");

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
