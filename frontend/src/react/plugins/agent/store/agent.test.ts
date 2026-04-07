import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { StoreApi } from "zustand";
import type { AgentState } from "./agent";
import {
  AGENT_STATE_KEY,
  AGENT_WINDOW_KEY,
  createAgentStore,
  selectLoading,
  selectOrderedChats,
} from "./agent";

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

let mockStorage: Storage;

beforeEach(() => {
  mockStorage = createMockStorage();
  vi.stubGlobal("localStorage", mockStorage);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const s = (store: StoreApi<AgentState>) => store.getState();

describe("useAgentStore (Zustand)", () => {
  test("creates a default chat when no persisted state exists", () => {
    const store = createAgentStore();

    expect(s(store).chats).toHaveLength(1);
    expect(s(store).currentChatId).toBe(s(store).chats[0].id);
    expect(s(store).getMessages(s(store).currentChatId)).toEqual([]);
    expect(s(store).chats[0].totalTokensUsed).toBe(0);
  });

  test("loads persisted window state", () => {
    localStorage.setItem(
      AGENT_WINDOW_KEY,
      JSON.stringify({
        position: { x: 120, y: 240 },
        size: { width: 480, height: 640 },
        sidebarWidth: 280,
      })
    );

    const store = createAgentStore();
    s(store).loadWindowState();

    expect(s(store).sidebarWidth).toBe(280);
    expect(s(store).position).toEqual({ x: 120, y: 240 });
    expect(s(store).size).toEqual({ width: 480, height: 640 });
    expect(localStorage.getItem(AGENT_WINDOW_KEY)).toContain('"width":480');
    expect(localStorage.getItem(AGENT_WINDOW_KEY)).toContain(
      '"sidebarWidth":280'
    );
    expect(localStorage.getItem(AGENT_STATE_KEY)).not.toBeNull();
  });

  test("normalizes stale running chats on load", () => {
    localStorage.setItem(
      AGENT_STATE_KEY,
      JSON.stringify({
        currentChatId: "thread-1",
        chats: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "running",
            runId: "run-1",
          },
        ],
        messagesByChatId: {
          "thread-1": [
            {
              id: "msg-1",
              chatId: "thread-1",
              createdTs: 30,
              role: "user",
              content: "hello",
            },
          ],
        },
        pendingAskByChatId: {},
      })
    );

    const store = createAgentStore();
    const chat = s(store).getChat(s(store).currentChatId);

    expect(chat).not.toBeNull();
    expect(chat?.status).toBe("idle");
    expect(chat?.interrupted).toBe(true);
    expect(chat?.runId).toBe("run-1");
    expect(selectLoading(s(store))).toBe(false);
  });

  test("clears interruption markers when a new run starts", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).interruptChatRun(chatId);
    s(store).startChatRun(
      chatId,
      {
        path: "/projects/demo",
        title: "Demo",
      },
      {
        runId: "run-2",
      }
    );

    expect(s(store).getChat(chatId)?.status).toBe("running");
    expect(s(store).getChat(chatId)?.interrupted).toBe(false);
    expect(s(store).getChat(chatId)?.runId).toBe("run-2");
  });

  test("persists the selected chat across store reloads", () => {
    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;

    s(store).addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Initial prompt",
    });
    const secondChat = s(store).createChat({
      title: "Second",
      page: {
        path: "/projects/original",
        title: "Original Page",
      },
    });
    s(store).setCurrentChat(secondChat.id);

    const rehydratedStore = createAgentStore();
    expect(s(rehydratedStore).currentChatId).toBe(secondChat.id);
    expect(s(rehydratedStore).getMessages(firstChatId)).toHaveLength(1);
    expect(s(rehydratedStore).getChat(secondChat.id)?.page).toEqual({
      path: "/projects/original",
      title: "Original Page",
    });
  });

  test("increments and persists chat token totals", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).incrementChatTotalTokens(chatId, 120);
    s(store).incrementChatTotalTokens(chatId, 30);

    expect(s(store).getChat(chatId)?.totalTokensUsed).toBe(150);

    const rehydratedStore = createAgentStore();
    expect(s(rehydratedStore).getChat(chatId)?.totalTokensUsed).toBe(150);
  });

  test("updates the chat page to the latest current page when starting a run", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });

    s(store).ensureCurrentChat({
      path: "/projects/other",
      title: "Other Page",
    });
    s(store).startChatRun(chatId, {
      path: "/projects/other",
      title: "Other Page",
    });

    expect(s(store).getChat(chatId)?.page).toEqual({
      path: "/projects/other",
      title: "Other Page",
    });
  });

  test("uses the first user message as the generated chat title", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).addMessage({
      chatId,
      role: "user",
      content: "Investigate unexpected production migration failures now",
    });

    expect(s(store).getChat(chatId)?.title).toBe(
      "Investigate unexpected production migration f..."
    );
  });

  test("orders equally updated chats by recency of creation", () => {
    const dateNow = vi.spyOn(Date, "now");
    dateNow.mockReturnValue(1000);

    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;

    dateNow.mockReturnValue(2000);
    s(store).addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Summarize the production incident timeline",
    });

    dateNow.mockReturnValue(2000);
    const secondChat = s(store).createChat({
      title: "Renamed thread",
      select: false,
    });

    expect(selectOrderedChats(s(store)).map((chat) => chat.id)).toEqual([
      secondChat.id,
      firstChatId,
    ]);

    dateNow.mockRestore();
  });

  test("does not switch chats while another chat is running", () => {
    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;
    const secondChat = s(store).createChat({
      title: "Second",
      select: false,
    });

    s(store).startChatRun(firstChatId, {
      path: "/projects/demo",
      title: "Demo",
    });
    s(store).setCurrentChat(secondChat.id);

    expect(s(store).currentChatId).toBe(firstChatId);
    expect(s(store).canSelectChat(secondChat.id)).toBe(false);
    expect(s(store).canSelectChat(firstChatId)).toBe(true);
  });

  test("creates a new chat without switching while a chat is running", () => {
    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;

    s(store).startChatRun(firstChatId, {
      path: "/projects/demo",
      title: "Demo",
    });
    const secondChat = s(store).createChat({
      title: "Second",
    });

    expect(s(store).currentChatId).toBe(firstChatId);
    expect(s(store).getChat(secondChat.id)?.title).toBe("Second");
  });

  test("persists pending choose asks for awaiting-user chats", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).awaitUser(chatId, {
      toolCallId: "tool-1",
      prompt: "Which database should I use?",
      kind: "choose",
      defaultValue: "prod-db",
      options: [
        {
          label: "Production",
          value: "prod-db",
          description: "Primary production database",
        },
        {
          label: "Staging",
          value: "staging-db",
        },
      ],
    });

    const rehydratedStore = createAgentStore();
    expect(s(rehydratedStore).getChat(chatId)?.status).toBe("awaiting_user");
    expect(s(rehydratedStore).getPendingAsk(chatId)).toEqual({
      toolCallId: "tool-1",
      prompt: "Which database should I use?",
      kind: "choose",
      defaultValue: "prod-db",
      confirmLabel: undefined,
      cancelLabel: undefined,
      options: [
        {
          label: "Production",
          value: "prod-db",
          description: "Primary production database",
        },
        {
          label: "Staging",
          value: "staging-db",
        },
      ],
    });
  });

  test("normalizes invalid persisted choose asks to input", () => {
    localStorage.setItem(
      AGENT_STATE_KEY,
      JSON.stringify({
        currentChatId: "thread-1",
        chats: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "awaiting_user",
          },
        ],
        messagesByChatId: {
          "thread-1": [],
        },
        pendingAskByChatId: {
          "thread-1": {
            toolCallId: "tool-1",
            prompt: "Choose a database",
            kind: "choose",
            options: [{ label: "Broken option" }],
          },
        },
      })
    );

    const store = createAgentStore();

    expect(s(store).getPendingAsk("thread-1")).toEqual({
      toolCallId: "tool-1",
      prompt: "Choose a database",
      kind: "input",
      defaultValue: undefined,
      confirmLabel: undefined,
      cancelLabel: undefined,
      options: undefined,
    });
  });

  test("answerPendingAsk appends a synthetic tool result and clears pending state", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).awaitUser(chatId, {
      toolCallId: "tool-1",
      prompt: "Proceed?",
      kind: "confirm",
      confirmLabel: "Proceed",
      cancelLabel: "Cancel",
    });

    s(store).answerPendingAsk(chatId, {
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });

    const messages = s(store).getMessages(chatId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-1");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });
    expect(s(store).getPendingAsk(chatId)).toBeNull();
  });

  test("answerPendingAsk stores choose labels and values", () => {
    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).awaitUser(chatId, {
      toolCallId: "tool-choose",
      prompt: "Choose an environment",
      kind: "choose",
      options: [
        { label: "Production", value: "prod" },
        { label: "Staging", value: "staging" },
      ],
    });

    s(store).answerPendingAsk(chatId, {
      kind: "choose",
      answer: "Production",
      value: "prod",
    });

    const messages = s(store).getMessages(chatId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-choose");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "choose",
      answer: "Production",
      value: "prod",
    });
    expect(s(store).getPendingAsk(chatId)).toBeNull();
  });

  test("migrates persisted chats without archived state", () => {
    localStorage.setItem(
      AGENT_STATE_KEY,
      JSON.stringify({
        currentChatId: "thread-1",
        chats: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "idle",
          },
        ],
        messagesByChatId: {
          "thread-1": [],
        },
        pendingAskByChatId: {},
      })
    );

    const store = createAgentStore();

    expect(s(store).getChat("thread-1")?.archived).toBe(false);
  });

  test("does not update updatedTs when rename normalization keeps the same title", () => {
    const dateNow = vi.spyOn(Date, "now");
    dateNow.mockReturnValue(1000);

    const store = createAgentStore();
    const chatId = s(store).currentChatId!;

    s(store).renameChat(chatId, "Renamed thread");
    const updatedTs = s(store).getChat(chatId)?.updatedTs;

    dateNow.mockReturnValue(2000);
    s(store).renameChat(chatId, "  Renamed thread  ");

    expect(s(store).getChat(chatId)?.title).toBe("Renamed thread");
    expect(s(store).getChat(chatId)?.updatedTs).toBe(updatedTs);

    dateNow.mockRestore();
  });

  test("supports renaming, archiving, unarchiving, and deleting chats", () => {
    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;
    const secondChat = s(store).createChat({ title: "Second thread" });

    s(store).renameChat(firstChatId, "  Renamed thread  ");
    s(store).archiveChat(firstChatId);

    expect(s(store).getChat(firstChatId)?.title).toBe("Renamed thread");
    expect(s(store).getChat(firstChatId)?.archived).toBe(true);

    s(store).unarchiveChat(firstChatId);
    expect(s(store).getChat(firstChatId)?.archived).toBe(false);

    s(store).deleteChat(secondChat.id);
    expect(s(store).getChat(secondChat.id)).toBeNull();
    expect(s(store).currentChatId).toBe(firstChatId);
  });

  test("tracks abort controllers per chat and cancels only the requested chat", () => {
    const store = createAgentStore();
    const firstChatId = s(store).currentChatId!;
    const secondChat = s(store).createChat({ title: "Second thread" });
    const firstController = new AbortController();
    const secondController = new AbortController();

    s(store).startChatRun(firstChatId, {
      path: "/projects/one",
      title: "One",
    });
    s(store).startChatRun(secondChat.id, {
      path: "/projects/two",
      title: "Two",
    });
    s(store).setAbortController(firstChatId, firstController);
    s(store).setAbortController(secondChat.id, secondController);

    s(store).cancel(firstChatId);

    expect(firstController.signal.aborted).toBe(true);
    expect(secondController.signal.aborted).toBe(false);
    expect(s(store).isChatRunning(firstChatId)).toBe(false);
    expect(s(store).isChatRunning(secondChat.id)).toBe(true);
  });
});
