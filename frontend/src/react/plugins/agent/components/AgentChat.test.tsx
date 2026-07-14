import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { createAgentStore, useAgentStore } from "../store/agent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  hasWorkspacePermissionV2: vi.fn(() => true),
  routerPush: vi.fn(),
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

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    push: mocks.routerPush,
    resolve: (to: unknown) => ({ href: JSON.stringify(to) }),
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
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

beforeEach(async () => {
  vi.stubGlobal("localStorage", createMockStorage());
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

  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.routerPush.mockReset();

  ({ AgentChat } = await import("./AgentChat"));
});

describe("AgentChat", () => {
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

  test("routes AI configuration recovery to general settings with intro", () => {
    const chat = useAgentStore.getState().ensureCurrentChat();
    useAgentStore.setState((state) => ({
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

    unmount();
  });
});
