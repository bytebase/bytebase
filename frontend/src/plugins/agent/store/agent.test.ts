import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import { AGENT_STATE_KEY, AGENT_WINDOW_KEY, useAgentStore } from "./agent";

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

const createStore = () => {
  setActivePinia(createPinia());
  return useAgentStore();
};

let mockStorage: Storage;

beforeEach(() => {
  mockStorage = createMockStorage();
  vi.stubGlobal("localStorage", mockStorage);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useAgentStore", () => {
  test("creates a default chat when no persisted state exists", () => {
    const store = createStore();

    expect(store.chats).toHaveLength(1);
    expect(store.currentChatId).toBe(store.chats[0].id);
    expect(store.messages).toEqual([]);
    expect(store.chats[0].totalTokensUsed).toBe(0);
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

    const store = createStore();
    store.loadWindowState();

    expect(store.sidebarWidth).toBe(280);
    expect(store.position).toEqual({ x: 120, y: 240 });
    expect(store.size).toEqual({ width: 480, height: 640 });
    expect(localStorage.getItem(AGENT_WINDOW_KEY)).toContain('"width":480');
    expect(localStorage.getItem(AGENT_WINDOW_KEY)).toContain(
      '"sidebarWidth":280'
    );
    expect(localStorage.getItem(AGENT_STATE_KEY)).toBeNull();
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

    const store = createStore();
    const chat = store.currentChat;

    expect(chat).not.toBeNull();
    expect(chat?.status).toBe("idle");
    expect(chat?.interrupted).toBe(true);
    expect(chat?.runId).toBe("run-1");
    expect(store.loading).toBe(false);
  });

  test("clears interruption markers when a new run starts", () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.interruptChatRun(chatId);
    store.startChatRun(
      chatId,
      {
        path: "/projects/demo",
        title: "Demo",
      },
      {
        runId: "run-2",
      }
    );

    expect(store.getChat(chatId)?.status).toBe("running");
    expect(store.getChat(chatId)?.interrupted).toBe(false);
    expect(store.getChat(chatId)?.runId).toBe("run-2");
  });

  test("persists the selected chat across store reloads", async () => {
    const store = createStore();
    const firstChatId = store.currentChatId!;

    store.addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Initial prompt",
    });
    const secondChat = store.createChat({
      title: "Second",
      page: {
        path: "/projects/original",
        title: "Original Page",
      },
    });
    store.setCurrentChat(secondChat.id);

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.currentChatId).toBe(secondChat.id);
    expect(rehydratedStore.getMessages(firstChatId)).toHaveLength(1);
    expect(rehydratedStore.getChat(secondChat.id)?.page).toEqual({
      path: "/projects/original",
      title: "Original Page",
    });
  });

  test("increments and persists chat token totals", async () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.incrementChatTotalTokens(chatId, 120);
    store.incrementChatTotalTokens(chatId, 30);

    expect(store.getChat(chatId)?.totalTokensUsed).toBe(150);

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.getChat(chatId)?.totalTokensUsed).toBe(150);
  });

  test("updates the chat page to the latest current page when starting a run", () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.updateChatPage(chatId, {
      path: "/projects/original",
      title: "Original Page",
    });

    store.ensureCurrentChat({
      path: "/projects/other",
      title: "Other Page",
    });
    store.startChatRun(chatId, {
      path: "/projects/other",
      title: "Other Page",
    });

    expect(store.getChat(chatId)?.page).toEqual({
      path: "/projects/other",
      title: "Other Page",
    });
  });

  test("uses the first user message as the generated chat title", () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.addMessage({
      chatId,
      role: "user",
      content: "Investigate unexpected production migration failures now",
    });

    expect(store.getChat(chatId)?.title).toBe(
      "Investigate unexpected production migration f..."
    );
  });

  test("orders equally updated chats by recency of creation", () => {
    const dateNow = vi.spyOn(Date, "now");
    dateNow.mockReturnValue(1000);

    const store = createStore();
    const firstChatId = store.currentChatId!;

    dateNow.mockReturnValue(2000);
    store.addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Summarize the production incident timeline",
    });

    dateNow.mockReturnValue(2000);
    const secondChat = store.createChat({
      title: "Renamed thread",
      select: false,
    });

    expect(store.orderedChats.map((chat) => chat.id)).toEqual([
      secondChat.id,
      firstChatId,
    ]);

    dateNow.mockRestore();
  });

  test("does not switch chats while another chat is running", () => {
    const store = createStore();
    const firstChatId = store.currentChatId!;
    const secondChat = store.createChat({
      title: "Second",
      select: false,
    });

    store.startChatRun(firstChatId, {
      path: "/projects/demo",
      title: "Demo",
    });
    store.setCurrentChat(secondChat.id);

    expect(store.currentChatId).toBe(firstChatId);
    expect(store.canSelectChat(secondChat.id)).toBe(false);
    expect(store.canSelectChat(firstChatId)).toBe(true);
  });

  test("creates a new chat without switching while a chat is running", () => {
    const store = createStore();
    const firstChatId = store.currentChatId!;

    store.startChatRun(firstChatId, {
      path: "/projects/demo",
      title: "Demo",
    });
    const secondChat = store.createChat({
      title: "Second",
    });

    expect(store.currentChatId).toBe(firstChatId);
    expect(store.getChat(secondChat.id)?.title).toBe("Second");
  });

  test("persists pending choose asks for awaiting-user chats", async () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.awaitUser(chatId, {
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

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.getChat(chatId)?.status).toBe("awaiting_user");
    expect(rehydratedStore.getPendingAsk(chatId)).toEqual({
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

    const store = createStore();

    expect(store.getPendingAsk("thread-1")).toEqual({
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
    const store = createStore();
    const chatId = store.currentChatId!;

    store.awaitUser(chatId, {
      toolCallId: "tool-1",
      prompt: "Proceed?",
      kind: "confirm",
      confirmLabel: "Proceed",
      cancelLabel: "Cancel",
    });

    store.answerPendingAsk(chatId, {
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });

    const messages = store.getMessages(chatId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-1");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });
    expect(store.getPendingAsk(chatId)).toBeNull();
  });

  test("answerPendingAsk stores choose labels and values", () => {
    const store = createStore();
    const chatId = store.currentChatId!;

    store.awaitUser(chatId, {
      toolCallId: "tool-choose",
      prompt: "Choose an environment",
      kind: "choose",
      options: [
        { label: "Production", value: "prod" },
        { label: "Staging", value: "staging" },
      ],
    });

    store.answerPendingAsk(chatId, {
      kind: "choose",
      answer: "Production",
      value: "prod",
    });

    const messages = store.getMessages(chatId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-choose");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "choose",
      answer: "Production",
      value: "prod",
    });
    expect(store.getPendingAsk(chatId)).toBeNull();
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

    const store = createStore();

    expect(store.getChat("thread-1")?.archived).toBe(false);
  });

  test("does not update updatedTs when rename normalization keeps the same title", () => {
    const dateNow = vi.spyOn(Date, "now");
    dateNow.mockReturnValue(1000);

    const store = createStore();
    const chatId = store.currentChatId!;

    store.renameChat(chatId, "Renamed thread");
    const updatedTs = store.getChat(chatId)?.updatedTs;

    dateNow.mockReturnValue(2000);
    store.renameChat(chatId, "  Renamed thread  ");

    expect(store.getChat(chatId)?.title).toBe("Renamed thread");
    expect(store.getChat(chatId)?.updatedTs).toBe(updatedTs);

    dateNow.mockRestore();
  });

  test("supports renaming, archiving, unarchiving, and deleting chats", () => {
    const store = createStore();
    const firstChatId = store.currentChatId!;
    const secondChat = store.createChat({ title: "Second thread" });

    store.renameChat(firstChatId, "  Renamed thread  ");
    store.archiveChat(firstChatId);

    expect(store.getChat(firstChatId)?.title).toBe("Renamed thread");
    expect(store.getChat(firstChatId)?.archived).toBe(true);

    store.unarchiveChat(firstChatId);
    expect(store.getChat(firstChatId)?.archived).toBe(false);

    store.deleteChat(secondChat.id);
    expect(store.getChat(secondChat.id)).toBeNull();
    expect(store.currentChatId).toBe(firstChatId);
  });

  test("tracks abort controllers per chat and cancels only the requested chat", () => {
    const store = createStore();
    const firstChatId = store.currentChatId!;
    const secondChat = store.createChat({ title: "Second thread" });
    const firstController = new AbortController();
    const secondController = new AbortController();

    store.startChatRun(firstChatId, { path: "/projects/one", title: "One" });
    store.startChatRun(secondChat.id, { path: "/projects/two", title: "Two" });
    store.setAbortController(firstChatId, firstController);
    store.setAbortController(secondChat.id, secondController);

    store.cancel(firstChatId);

    expect(firstController.signal.aborted).toBe(true);
    expect(secondController.signal.aborted).toBe(false);
    expect(store.isChatRunning(firstChatId)).toBe(false);
    expect(store.isChatRunning(secondChat.id)).toBe(true);
  });
});
