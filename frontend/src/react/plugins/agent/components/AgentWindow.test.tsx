import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  })),
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
  useTranslation: mocks.useTranslation,
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
    position: initialState.position,
    size: initialState.size,
    sidebarWidth: initialState.sidebarWidth,
    minimized: initialState.minimized,
    chats: initialState.chats,
    messagesByChatId: initialState.messagesByChatId,
    pendingAskByChatId: initialState.pendingAskByChatId,
    currentChatId: initialState.currentChatId,
    abortControllersByChatId: {},
  });
  useAgentStore.setState({
    visible: true,
    minimized: false,
    position: { x: 100, y: 120 },
    size: { width: 500, height: 450 },
    sidebarWidth: 200,
  });

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  });

  ({ AgentWindow } = await import("./AgentWindow"));
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  document.body.innerHTML = "";
});

describe("AgentWindow", () => {
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

  test("commits drag position only on pointerup", () => {
    const { render, unmount } = renderIntoContainer(<AgentWindow />);

    render();

    const windowEl = document.body.querySelector(
      "[data-agent-window]"
    ) as HTMLDivElement | null;
    const header = windowEl?.querySelector(
      ".cursor-move"
    ) as HTMLDivElement | null;

    expect(windowEl).toBeInstanceOf(HTMLDivElement);
    expect(header).toBeInstanceOf(HTMLDivElement);

    Object.defineProperty(windowEl, "offsetWidth", {
      configurable: true,
      get: () => 500,
    });
    Object.defineProperty(windowEl, "offsetHeight", {
      configurable: true,
      get: () => 450,
    });

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
    });

    expect(windowEl?.style.left).toBe("130px");
    expect(windowEl?.style.top).toBe("160px");
    expect(useAgentStore.getState().position).toEqual({ x: 100, y: 120 });

    act(() => {
      document.dispatchEvent(
        new PointerEvent("pointerup", {
          bubbles: true,
          clientX: 180,
          clientY: 190,
        })
      );
    });

    expect(useAgentStore.getState().position).toEqual({ x: 130, y: 160 });

    unmount();
  });
});
