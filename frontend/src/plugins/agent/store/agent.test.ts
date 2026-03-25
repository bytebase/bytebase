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
  test("creates a default thread when no persisted state exists", () => {
    const store = createStore();

    expect(store.threads).toHaveLength(1);
    expect(store.currentThreadId).toBe(store.threads[0].id);
    expect(store.messages).toEqual([]);
    expect(store.threads[0].totalTokensUsed).toBe(0);
  });

  test("loads persisted window state", () => {
    localStorage.setItem(
      AGENT_WINDOW_KEY,
      JSON.stringify({
        position: { x: 120, y: 240 },
        size: { width: 480, height: 640 },
      })
    );

    const store = createStore();
    store.loadWindowState();

    expect(store.position).toEqual({ x: 120, y: 240 });
    expect(store.size).toEqual({ width: 480, height: 640 });
    expect(localStorage.getItem(AGENT_WINDOW_KEY)).toContain('"width":480');
    expect(localStorage.getItem(AGENT_STATE_KEY)).toBeNull();
  });

  test("normalizes stale running threads on load", () => {
    localStorage.setItem(
      AGENT_STATE_KEY,
      JSON.stringify({
        currentThreadId: "thread-1",
        threads: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "running",
            runId: "run-1",
          },
        ],
        messagesByThreadId: {
          "thread-1": [
            {
              id: "msg-1",
              threadId: "thread-1",
              createdTs: 30,
              role: "user",
              content: "hello",
            },
          ],
        },
        pendingAskByThreadId: {},
      })
    );

    const store = createStore();
    const thread = store.currentThread;

    expect(thread).not.toBeNull();
    expect(thread?.status).toBe("idle");
    expect(thread?.interrupted).toBe(true);
    expect(thread?.runId).toBe("run-1");
    expect(store.loading).toBe(false);
  });

  test("clears interruption markers when a new run starts", () => {
    const store = createStore();
    const threadId = store.currentThreadId!;

    store.interruptRun(threadId);
    store.startRun(
      threadId,
      {
        path: "/projects/demo",
        title: "Demo",
      },
      {
        runId: "run-2",
      }
    );

    expect(store.getThread(threadId)?.status).toBe("running");
    expect(store.getThread(threadId)?.interrupted).toBe(false);
    expect(store.getThread(threadId)?.runId).toBe("run-2");
  });

  test("persists the selected thread and resets thread messages in place", async () => {
    const store = createStore();
    const firstThreadId = store.currentThreadId!;

    store.addMessage({
      threadId: firstThreadId,
      role: "user",
      content: "Initial prompt",
    });
    const secondThread = store.createThread({
      title: "Second",
      page: {
        path: "/projects/original",
        title: "Original Page",
      },
    });
    store.setCurrentThread(secondThread.id);

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.currentThreadId).toBe(secondThread.id);
    expect(rehydratedStore.getMessages(firstThreadId)).toHaveLength(1);

    rehydratedStore.clearConversation(secondThread.id);

    expect(rehydratedStore.getMessages(secondThread.id)).toEqual([]);
    expect(rehydratedStore.getThread(secondThread.id)?.status).toBe("idle");
    expect(rehydratedStore.getThread(secondThread.id)?.page).toBeUndefined();

    rehydratedStore.ensureCurrentThread({
      path: "/projects/current",
      title: "Current Page",
    });

    expect(rehydratedStore.getThread(secondThread.id)?.page).toEqual({
      path: "/projects/current",
      title: "Current Page",
    });
  });

  test("increments and resets thread token totals", async () => {
    const store = createStore();
    const threadId = store.currentThreadId!;

    store.incrementThreadTotalTokens(threadId, 120);
    store.incrementThreadTotalTokens(threadId, 30);

    expect(store.getThread(threadId)?.totalTokensUsed).toBe(150);

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.getThread(threadId)?.totalTokensUsed).toBe(150);

    rehydratedStore.clearConversation(threadId);
    expect(rehydratedStore.getThread(threadId)?.totalTokensUsed).toBe(0);
  });

  test("updates the thread page to the latest current page when starting a run", () => {
    const store = createStore();
    const threadId = store.currentThreadId!;

    store.updateThreadPage(threadId, {
      path: "/projects/original",
      title: "Original Page",
    });

    store.ensureCurrentThread({
      path: "/projects/other",
      title: "Other Page",
    });
    store.startRun(threadId, {
      path: "/projects/other",
      title: "Other Page",
    });

    expect(store.getThread(threadId)?.page).toEqual({
      path: "/projects/other",
      title: "Other Page",
    });
  });

  test("persists pending choose asks for awaiting-user threads", async () => {
    const store = createStore();
    const threadId = store.currentThreadId!;

    store.awaitUser(threadId, {
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
    expect(rehydratedStore.getThread(threadId)?.status).toBe("awaiting_user");
    expect(rehydratedStore.getPendingAsk(threadId)).toEqual({
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
        currentThreadId: "thread-1",
        threads: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "awaiting_user",
          },
        ],
        messagesByThreadId: {
          "thread-1": [],
        },
        pendingAskByThreadId: {
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
    const threadId = store.currentThreadId!;

    store.awaitUser(threadId, {
      toolCallId: "tool-1",
      prompt: "Proceed?",
      kind: "confirm",
      confirmLabel: "Proceed",
      cancelLabel: "Cancel",
    });

    store.answerPendingAsk(threadId, {
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });

    const messages = store.getMessages(threadId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-1");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "confirm",
      answer: "Proceed",
      confirmed: true,
    });
    expect(store.getPendingAsk(threadId)).toBeNull();
  });

  test("answerPendingAsk stores choose labels and values", () => {
    const store = createStore();
    const threadId = store.currentThreadId!;

    store.awaitUser(threadId, {
      toolCallId: "tool-choose",
      prompt: "Choose an environment",
      kind: "choose",
      options: [
        { label: "Production", value: "prod" },
        { label: "Staging", value: "staging" },
      ],
    });

    store.answerPendingAsk(threadId, {
      kind: "choose",
      answer: "Production",
      value: "prod",
    });

    const messages = store.getMessages(threadId);
    expect(messages).toHaveLength(1);
    expect(messages[0].role).toBe("tool");
    expect(messages[0].toolCallId).toBe("tool-choose");
    expect(JSON.parse(messages[0].content ?? "{}")).toEqual({
      kind: "choose",
      answer: "Production",
      value: "prod",
    });
    expect(store.getPendingAsk(threadId)).toBeNull();
  });

  test("migrates persisted threads without archived state", () => {
    localStorage.setItem(
      AGENT_STATE_KEY,
      JSON.stringify({
        currentThreadId: "thread-1",
        threads: [
          {
            id: "thread-1",
            title: "Existing thread",
            createdTs: 10,
            updatedTs: 20,
            status: "idle",
          },
        ],
        messagesByThreadId: {
          "thread-1": [],
        },
        pendingAskByThreadId: {},
      })
    );

    const store = createStore();

    expect(store.getThread("thread-1")?.archived).toBe(false);
  });

  test("supports renaming, archiving, unarchiving, and deleting threads", () => {
    const store = createStore();
    const firstThreadId = store.currentThreadId!;
    const secondThread = store.createThread({ title: "Second thread" });

    store.renameThread(firstThreadId, "  Renamed thread  ");
    store.archiveThread(firstThreadId);

    expect(store.getThread(firstThreadId)?.title).toBe("Renamed thread");
    expect(store.getThread(firstThreadId)?.archived).toBe(true);

    store.unarchiveThread(firstThreadId);
    expect(store.getThread(firstThreadId)?.archived).toBe(false);

    store.deleteThread(secondThread.id);
    expect(store.getThread(secondThread.id)).toBeNull();
    expect(store.currentThreadId).toBe(firstThreadId);
  });

  test("tracks abort controllers per thread and cancels only the requested thread", () => {
    const store = createStore();
    const firstThreadId = store.currentThreadId!;
    const secondThread = store.createThread({ title: "Second thread" });
    const firstController = new AbortController();
    const secondController = new AbortController();

    store.startRun(firstThreadId, { path: "/projects/one", title: "One" });
    store.startRun(secondThread.id, { path: "/projects/two", title: "Two" });
    store.setAbortController(firstThreadId, firstController);
    store.setAbortController(secondThread.id, secondController);

    store.cancel(firstThreadId);

    expect(firstController.signal.aborted).toBe(true);
    expect(secondController.signal.aborted).toBe(false);
    expect(store.isThreadRunning(firstThreadId)).toBe(false);
    expect(store.isThreadRunning(secondThread.id)).toBe(true);
  });
});
