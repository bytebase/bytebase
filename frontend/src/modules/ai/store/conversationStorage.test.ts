import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Conversation, Message } from "../types";
import {
  loadConversationHistory,
  MAX_CONVERSATION_HISTORY_BYTES,
  mutateConversationHistory,
  saveConversationHistory,
} from "./conversationStorage";

const STORAGE_KEY = "bb.ai.conversations.workspaces.test.user@example.com";

const conversation = (
  id: string,
  createdTs: number,
  contents: string[] = ["SELECT 1"]
): Conversation => {
  const result: Conversation = {
    id,
    created_ts: createdTs,
    name: id,
    instance: "instances/test",
    database: "instances/test/databases/demo",
    messageList: [],
  };
  result.messageList = contents.map<Message>((content, index) => ({
    id: `${id}-message-${index}`,
    created_ts: createdTs + index,
    author: index % 2 === 0 ? "USER" : "AI",
    content,
    status: "DONE",
    error: "",
    conversation: result,
  }));
  return result;
};

beforeEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
  Object.defineProperty(navigator, "locks", {
    configurable: true,
    value: undefined,
  });
});

describe("AI conversation local storage", () => {
  test("restores visible history with message back-references", () => {
    const original = conversation("conversation-1", 1, ["visible question"]);

    expect(saveConversationHistory(STORAGE_KEY, [original])).toBe(true);
    const [restored] = loadConversationHistory(STORAGE_KEY);
    expect(restored).toMatchObject({
      id: "conversation-1",
      messageList: [
        expect.objectContaining({
          content: "visible question",
        }),
      ],
    });
    expect(restored?.messageList[0]?.conversation).toBe(restored);
  });

  test("turns an interrupted response into a failed message on reload", () => {
    const original = conversation("conversation-1", 1);
    original.messageList[0]!.author = "AI";
    original.messageList[0]!.status = "LOADING";
    original.messageList[0]!.content = "";

    saveConversationHistory(STORAGE_KEY, [original]);

    expect(
      loadConversationHistory(STORAGE_KEY)[0]?.messageList[0]
    ).toMatchObject({
      status: "FAILED",
      error: "Request timeout",
    });
  });

  test("keeps only the 20 most recent conversations", () => {
    const conversations = Array.from({ length: 25 }, (_, index) =>
      conversation(`conversation-${index}`, index)
    );

    saveConversationHistory(STORAGE_KEY, conversations);

    const restored = loadConversationHistory(STORAGE_KEY);
    expect(restored).toHaveLength(20);
    expect(restored[0]?.id).toBe("conversation-5");
    expect(restored.at(-1)?.id).toBe("conversation-24");
  });

  test("does not let an empty draft evict saved conversations", () => {
    const conversations = Array.from({ length: 20 }, (_, index) =>
      conversation(`conversation-${index}`, index)
    );
    conversations.push(conversation("empty-draft", 21, []));

    saveConversationHistory(STORAGE_KEY, conversations);

    const restored = loadConversationHistory(STORAGE_KEY);
    expect(restored).toHaveLength(20);
    expect(restored.some(({ id }) => id === "empty-draft")).toBe(false);
    expect(restored[0]?.id).toBe("conversation-0");
  });

  test("keeps serialized history within one MiB", () => {
    const original = conversation("conversation-1", 1, [
      "old".repeat(MAX_CONVERSATION_HISTORY_BYTES),
      "answer",
      "recent question",
    ]);

    saveConversationHistory(STORAGE_KEY, [original]);

    const raw = localStorage.getItem(STORAGE_KEY) ?? "";
    expect(new TextEncoder().encode(raw).byteLength).toBeLessThanOrEqual(
      MAX_CONVERSATION_HISTORY_BYTES
    );
    expect(loadConversationHistory(STORAGE_KEY)[0]?.messageList).toEqual([
      expect.objectContaining({ content: "recent question" }),
    ]);
  });

  test("drops a conversation whose metadata alone exceeds the byte cap", () => {
    const original = conversation("conversation-1", 1, []);
    original.name = "x".repeat(MAX_CONVERSATION_HISTORY_BYTES + 1);

    saveConversationHistory(STORAGE_KEY, [original]);

    const raw = localStorage.getItem(STORAGE_KEY) ?? "";
    expect(new TextEncoder().encode(raw).byteLength).toBeLessThanOrEqual(
      MAX_CONVERSATION_HISTORY_BYTES
    );
    expect(loadConversationHistory(STORAGE_KEY)).toEqual([]);
  });

  test("physically removes conversations omitted by the next write", () => {
    saveConversationHistory(STORAGE_KEY, [
      conversation("deleted", 1),
      conversation("kept", 2),
    ]);

    saveConversationHistory(STORAGE_KEY, [conversation("kept", 2)]);

    expect(loadConversationHistory(STORAGE_KEY).map(({ id }) => id)).toEqual([
      "kept",
    ]);
    expect(localStorage.getItem(STORAGE_KEY)).not.toContain("deleted");
  });

  test("ignores malformed stored history", () => {
    localStorage.setItem(STORAGE_KEY, '{"version":1,"conversations":"bad"}');

    expect(loadConversationHistory(STORAGE_KEY)).toEqual([]);
  });

  test("prunes and retries once after a storage write failure", () => {
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementationOnce(() => {
        throw new DOMException("Storage quota exceeded", "QuotaExceededError");
      })
      .mockImplementationOnce(() => undefined);

    const saved = saveConversationHistory(
      STORAGE_KEY,
      Array.from({ length: 4 }, (_, index) =>
        conversation(`conversation-${index}`, index, ["x".repeat(100)])
      )
    );

    expect(saved).toBe(true);
    expect(setItem).toHaveBeenCalledTimes(2);
    expect(String(setItem.mock.calls[1]?.[1]).length).toBeLessThan(
      String(setItem.mock.calls[0]?.[1]).length
    );
  });

  test("does not throw when both storage write attempts fail", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("Storage quota exceeded", "QuotaExceededError");
    });

    expect(
      saveConversationHistory(STORAGE_KEY, [conversation("conversation-1", 1)])
    ).toBe(false);
  });

  test("coordinates history mutations with a cross-tab lock", async () => {
    const request = vi.fn(async (_name: string, callback: () => boolean) =>
      callback()
    );
    Object.defineProperty(navigator, "locks", {
      configurable: true,
      value: { request },
    });

    await mutateConversationHistory(STORAGE_KEY, (conversations) => [
      ...conversations,
      conversation("conversation-1", 1),
    ]);

    expect(request).toHaveBeenCalledWith(
      `${STORAGE_KEY}.write`,
      expect.any(Function)
    );
    expect(loadConversationHistory(STORAGE_KEY)).toHaveLength(1);
  });
});
