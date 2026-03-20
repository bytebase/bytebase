import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import {
  AGENT_STATE_KEY,
  LEGACY_AGENT_MESSAGES_KEY,
  useAgentStore,
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
  });

  test("migrates legacy flat messages into a thread-aware state", () => {
    localStorage.setItem(
      LEGACY_AGENT_MESSAGES_KEY,
      JSON.stringify([
        { role: "user", content: "Help me inspect this page" },
        { role: "assistant", content: "Sure" },
      ])
    );

    const store = createStore();

    expect(store.threads).toHaveLength(1);
    expect(store.currentThreadId).toBe(store.threads[0].id);
    expect(store.messages).toHaveLength(2);
    expect(store.messages[0].id).toBeTruthy();
    expect(store.messages[0].threadId).toBe(store.currentThreadId);
    expect(store.messages[0].createdTs).toBeTypeOf("number");
    expect(store.threads[0].title).toContain("Help me inspect this page");
    expect(localStorage.getItem(LEGACY_AGENT_MESSAGES_KEY)).toBeNull();
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
      })
    );

    const store = createStore();
    const thread = store.currentThread;

    expect(thread).not.toBeNull();
    expect(thread?.status).toBe("error");
    expect(thread?.interrupted).toBe(true);
    expect(store.loading).toBe(false);
  });

  test("persists the selected thread and resets thread messages in place", async () => {
    const store = createStore();
    const firstThreadId = store.currentThreadId!;

    store.addMessage({
      threadId: firstThreadId,
      role: "user",
      content: "Initial prompt",
    });
    const secondThread = store.createThread({ title: "Second" });
    store.setCurrentThread(secondThread.id);

    await nextTick();

    const rehydratedStore = createStore();
    expect(rehydratedStore.currentThreadId).toBe(secondThread.id);
    expect(rehydratedStore.getMessages(firstThreadId)).toHaveLength(1);

    rehydratedStore.clearConversation(secondThread.id);

    expect(rehydratedStore.getMessages(secondThread.id)).toEqual([]);
    expect(rehydratedStore.getThread(secondThread.id)?.status).toBe("idle");
  });
});
