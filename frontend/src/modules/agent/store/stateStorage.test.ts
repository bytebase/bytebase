import * as LZString from "lz-string";
import { describe, expect, it, vi } from "vitest";
import {
  encodeAgentStoredState,
  isCompressedAgentState,
  parseAgentStoredState,
  writeAgentStoredState,
} from "./stateStorage";

vi.mock("lz-string", async (importOriginal) => {
  const imported = await importOriginal<
    typeof LZString & { default?: typeof LZString }
  >();
  const actual = imported.default ?? imported;
  return {
    ...actual,
    compressToUTF16: vi.fn(actual.compressToUTF16),
  };
});

const PLACEHOLDER = "[bb-agent-compressed-content-v1]";
const STATE_KEY = "bb-agent-state-v3.example";

function createCapacityStorage(
  capacity = Number.POSITIVE_INFINITY,
  quotaName = "QuotaExceededError"
): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    key: (index) => [...values.keys()][index] ?? null,
    getItem: (key) => values.get(key) ?? null,
    setItem: vi.fn((key, value) => {
      const next = new Map(values).set(key, value);
      const bytes = [...next].reduce(
        (total, [entryKey, entryValue]) =>
          total + 2 * (entryKey.length + entryValue.length),
        0
      );
      if (bytes > capacity) {
        throw new DOMException("Storage quota exceeded", quotaName);
      }
      values.set(key, value);
    }),
    removeItem: vi.fn((key) => values.delete(key)),
    clear: vi.fn(() => values.clear()),
  };
}

const content =
  "SELECT email, '\u90ae\u7bb1', '\u00e9', '\ud83d\ude80' FROM example_table;\n".repeat(
    70
  ) + 'Quoted "SQL", newline\n\ud800';
const snapshot = {
  version: 3,
  currentChatId: "chat-1",
  chats: [
    { id: "chat-1", checkpoint: { runId: "run-1", status: "awaiting_user" } },
  ],
  messagesByChatId: {
    "chat-1": [
      {
        id: "message-1",
        role: "tool",
        content,
        metadata: { runId: "run-1" },
      },
      { id: "message-2", role: "assistant", content: "Result" },
    ],
  },
  pendingAskByChatId: {
    "chat-1": { id: "ask-1", kind: "choose", options: ["database"] },
  },
};
const json = JSON.stringify(snapshot);

function encodedJson() {
  return JSON.stringify({
    ...snapshot,
    messagesByChatId: {
      "chat-1": [
        {
          ...snapshot.messagesByChatId["chat-1"][0],
          content: PLACEHOLDER,
          metadata: {
            runId: "run-1",
            agentCompressedContent: {
              version: 1,
              value: LZString.compressToUTF16(content),
            },
          },
        },
        snapshot.messagesByChatId["chat-1"][1],
      ],
    },
  });
}

describe("agent state storage", () => {
  it("measures exactly the representation selected for an atomic write", () => {
    const storage = createCapacityStorage();
    expect(encodeAgentStoredState(json, null)).toBe(json);
    const compressed = encodeAgentStoredState(json, null, true);
    expect(compressed.length).toBeLessThan(json.length);
    expect(parseAgentStoredState(compressed)).toEqual(snapshot);
    expect(writeAgentStoredState(storage, STATE_KEY, json, compressed)).toBe(
      encodeAgentStoredState(json, compressed)
    );
  });
  it("reads existing plain JSON snapshots", () => {
    expect(parseAgentStoredState(json)).toEqual(snapshot);
  });

  it("strictly decodes known encoder markers without changing metadata shape", () => {
    const source = {
      messagesByChatId: {
        "chat-1": [
          { content },
          { content, metadata: {} },
          { content, metadata: { custom: "keep" } },
        ],
      },
    };
    const encoded = encodeAgentStoredState(JSON.stringify(source), null, true);
    expect(
      parseAgentStoredState(encoded, { strictCompressionMarkers: true })
    ).toEqual(source);
    expect(
      parseAgentStoredState(json, { strictCompressionMarkers: true })
    ).toEqual(snapshot);
  });

  it.each([
    {
      name: "stale marker on plaintext",
      content: "Updated content",
      marker: { version: 1, value: LZString.compressToUTF16(content) },
    },
    {
      name: "unknown marker on plaintext",
      content: "Opaque content",
      marker: { version: 99, custom: "keep" },
    },
    {
      name: "unknown compressed marker fields",
      content: PLACEHOLDER,
      marker: {
        version: 1,
        value: LZString.compressToUTF16(content),
        custom: "keep",
      },
    },
  ])(
    "strictly rejects $name without changing ordinary hydration",
    ({ content: originalContent, marker }) => {
      const serialized = JSON.stringify({
        messagesByChatId: {
          "chat-1": [
            {
              content: originalContent,
              metadata: { custom: "keep", agentCompressedContent: marker },
            },
          ],
        },
      });
      expect(() =>
        parseAgentStoredState(serialized, { strictCompressionMarkers: true })
      ).toThrow("Invalid compressed agent message content.");
      expect(parseAgentStoredState(serialized)).toEqual({
        messagesByChatId: {
          "chat-1": [
            {
              content:
                originalContent === PLACEHOLDER ? content : originalContent,
              metadata: { custom: "keep" },
            },
          ],
        },
      });
    }
  );

  it.each([
    { version: 99, value: LZString.compressToUTF16(content) },
    { version: 1, value: "" },
    {
      version: 1,
      value: LZString.compressToUTF16(content),
      metadataAbsent: false,
    },
    null,
  ])("strictly rejects invalid v1 marker data: %j", (marker) => {
    const serialized = JSON.stringify({
      messagesByChatId: {
        "chat-1": [
          {
            content: PLACEHOLDER,
            metadata: { agentCompressedContent: marker },
          },
        ],
      },
    });
    expect(() =>
      parseAgentStoredState(serialized, { strictCompressionMarkers: true })
    ).toThrow("Invalid compressed agent message content.");
  });

  it.each([
    null,
    42,
    ["normal", "JSON"],
    { nested: { content, metadata: { unrelated: true } } },
    {
      messagesByChatId: {
        invalid: "not an array",
        "chat-1": [null, "not a message", { content, metadata: null }],
      },
    },
  ])("leaves unmarked JSON unchanged: %j", (state) => {
    const serialized = JSON.stringify(state);
    const storage = createCapacityStorage();

    const stored = writeAgentStoredState(
      storage,
      STATE_KEY,
      serialized,
      encodedJson()
    );

    expect(stored).toBe(serialized);
    expect(parseAgentStoredState(stored)).toEqual(state);
  });

  it("identifies compressed content in an otherwise normal JSON snapshot", () => {
    expect(isCompressedAgentState(null)).toBe(false);
    expect(isCompressedAgentState(json)).toBe(false);
    expect(isCompressedAgentState("invalid JSON")).toBe(false);
    expect(isCompressedAgentState(encodedJson())).toBe(true);
  });

  it("losslessly reads Unicode SQL and the full JSON snapshot envelope", () => {
    expect(parseAgentStoredState(encodedJson())).toEqual(snapshot);
    expect(JSON.parse(encodedJson()).chats).toEqual(snapshot.chats);
  });

  it.each(["QuotaExceededError", "NS_ERROR_DOM_QUOTA_REACHED"])(
    "compresses on %s when other origin data prevents a plain write",
    (quotaName) => {
      const capacity = 12_000;
      const storage = createCapacityStorage(capacity, quotaName);
      const unrelated = "x".repeat(3500);
      const previous = JSON.stringify({ version: 3, chats: [] });
      storage.setItem("unrelated-app", unrelated);
      storage.setItem(STATE_KEY, previous);
      vi.mocked(storage.setItem).mockClear();
      expect(2 * (STATE_KEY.length + json.length)).toBeLessThan(capacity);

      const stored = writeAgentStoredState(storage, STATE_KEY, json, previous);

      expect(stored).toBe(encodedJson());
      expect(stored).toBe(storage.getItem(STATE_KEY));
      expect(parseAgentStoredState(stored)).toEqual(snapshot);
      expect(JSON.parse(stored).pendingAskByChatId).toEqual(
        snapshot.pendingAskByChatId
      );
      expect(storage.getItem("unrelated-app")).toBe(unrelated);
      expect(storage.length).toBe(2);
      expect(storage.setItem).toHaveBeenCalledTimes(2);
      expect(storage.removeItem).not.toHaveBeenCalled();
      expect(storage.clear).not.toHaveBeenCalled();
    }
  );

  it("writes plain JSON without compression when capacity is available", () => {
    const storage = createCapacityStorage();

    expect(writeAgentStoredState(storage, STATE_KEY, json, null)).toBe(json);

    expect(storage.getItem(STATE_KEY)).toBe(json);
    expect(storage.setItem).toHaveBeenCalledExactlyOnceWith(STATE_KEY, json);
  });

  it("compresses subsequent updates without another plain storage attempt", () => {
    const storage = createCapacityStorage();
    const previous = encodedJson();
    storage.setItem(STATE_KEY, previous);
    vi.mocked(storage.setItem).mockClear();
    const next = JSON.stringify({ ...snapshot, currentChatId: null });

    const stored = writeAgentStoredState(storage, STATE_KEY, next, previous);

    expect(isCompressedAgentState(stored)).toBe(true);
    expect(parseAgentStoredState(stored)).toEqual({
      ...snapshot,
      currentChatId: null,
    });
    expect(storage.getItem(STATE_KEY)).toBe(stored);
    expect(storage.setItem).toHaveBeenCalledExactlyOnceWith(STATE_KEY, stored);
  });

  it("does not rewrite an unchanged compressed snapshot", () => {
    const storage = createCapacityStorage();
    const previous = encodedJson();
    storage.setItem(STATE_KEY, previous);
    vi.mocked(storage.setItem).mockClear();

    expect(writeAgentStoredState(storage, STATE_KEY, json, previous)).toBe(
      previous
    );

    expect(storage.setItem).not.toHaveBeenCalled();
    expect(storage.getItem(STATE_KEY)).toBe(previous);
  });

  it("reuses previous compressed bodies and encodes only new or changed content", () => {
    const storage = createCapacityStorage();
    const previous = encodedJson();
    const nextContent = "SELECT id FROM another_table;\n".repeat(100);
    const next = JSON.stringify({
      ...snapshot,
      messagesByChatId: {
        "chat-1": [
          snapshot.messagesByChatId["chat-1"][0],
          { id: "message-3", role: "tool", content: nextContent },
        ],
      },
    });
    const compress = vi.mocked(LZString.compressToUTF16);
    compress.mockClear();
    try {
      const stored = writeAgentStoredState(storage, STATE_KEY, next, previous);

      expect(compress).toHaveBeenCalledExactlyOnceWith(nextContent);
      expect(parseAgentStoredState(stored)).toEqual(JSON.parse(next));
      compress.mockClear();

      expect(writeAgentStoredState(storage, STATE_KEY, next, stored)).toBe(
        stored
      );
      expect(compress).not.toHaveBeenCalled();

      const changed = JSON.parse(next);
      const changedContent = `${content}\nSELECT id FROM changed_table;`;
      changed.messagesByChatId["chat-1"][0].content = changedContent;
      const updated = writeAgentStoredState(
        storage,
        STATE_KEY,
        JSON.stringify(changed),
        stored
      );

      expect(compress).toHaveBeenCalledExactlyOnceWith(changedContent);
      expect(parseAgentStoredState(updated)).toEqual(changed);
    } finally {
      compress.mockClear();
    }
  });

  it("restores absent and explicitly empty metadata without changing their shape", () => {
    const next = JSON.stringify({
      messagesByChatId: {
        "chat-1": [{ content }, { content, metadata: {} }],
      },
    });
    const storage = createCapacityStorage();

    const stored = writeAgentStoredState(
      storage,
      STATE_KEY,
      next,
      encodedJson()
    );

    expect(parseAgentStoredState(stored)).toEqual(JSON.parse(next));
    const raw = JSON.parse(stored).messagesByChatId["chat-1"];
    expect(raw[0].metadata.agentCompressedContent.metadataAbsent).toBe(true);
    expect(raw[1].metadata.agentCompressedContent).not.toHaveProperty(
      "metadataAbsent"
    );
  });

  it("does not compress pending message content or small messages", () => {
    const next = JSON.stringify({
      messagesByChatId: {
        "chat-1": [
          { content },
          { content, metadata: { pending: true } },
          { content: "small" },
        ],
      },
    });
    const storage = createCapacityStorage();

    const stored = writeAgentStoredState(
      storage,
      STATE_KEY,
      next,
      encodedJson()
    );

    expect(parseAgentStoredState(stored)).toEqual(JSON.parse(next));
    const raw = JSON.parse(stored).messagesByChatId["chat-1"];
    expect(raw[0].content).toBe(PLACEHOLDER);
    expect(raw[1]).toEqual({ content, metadata: { pending: true } });
    expect(raw[2]).toEqual({ content: "small" });
  });

  it("preserves an older writer's changed content instead of restoring stale data", () => {
    const raw = JSON.parse(encodedJson());
    const message = raw.messagesByChatId["chat-1"][0];
    message.content = "Updated content";

    const restored = parseAgentStoredState(JSON.stringify(raw));

    expect(restored).toEqual({
      ...snapshot,
      messagesByChatId: {
        "chat-1": [
          {
            ...snapshot.messagesByChatId["chat-1"][0],
            content: "Updated content",
          },
          snapshot.messagesByChatId["chat-1"][1],
        ],
      },
    });
  });

  it("retains metadata added by an older writer after compression", () => {
    const storage = createCapacityStorage();
    const next = JSON.stringify({
      messagesByChatId: { "chat-1": [{ content }] },
    });
    const stored = writeAgentStoredState(
      storage,
      STATE_KEY,
      next,
      encodedJson()
    );
    const raw = JSON.parse(stored);
    raw.messagesByChatId["chat-1"][0].metadata.runId = "run-2";

    expect(parseAgentStoredState(JSON.stringify(raw))).toEqual({
      messagesByChatId: {
        "chat-1": [{ content, metadata: { runId: "run-2" } }],
      },
    });
  });

  it("preserves the old snapshot and other data when compressed data cannot fit", () => {
    const previous = "{}";
    const unrelated = "x".repeat(100);
    const capacity =
      2 * (STATE_KEY.length + previous.length + "unrelated-app".length + 100);
    const storage = createCapacityStorage(capacity);
    storage.setItem(STATE_KEY, previous);
    storage.setItem("unrelated-app", unrelated);
    vi.mocked(storage.setItem).mockClear();

    expect(() =>
      writeAgentStoredState(storage, STATE_KEY, json, previous)
    ).toThrowError(expect.objectContaining({ name: "QuotaExceededError" }));

    expect(storage.getItem(STATE_KEY)).toBe(previous);
    expect(storage.getItem("unrelated-app")).toBe(unrelated);
    expect(storage.setItem).toHaveBeenCalledTimes(2);
    expect(storage.removeItem).not.toHaveBeenCalled();
    expect(storage.clear).not.toHaveBeenCalled();
  });

  it("does not retry a quota failure when no message can be made smaller", () => {
    const storage = createCapacityStorage();
    const error = new DOMException(
      "Storage quota exceeded",
      "QuotaExceededError"
    );
    vi.mocked(storage.setItem).mockImplementation(() => {
      throw error;
    });

    expect(() => writeAgentStoredState(storage, STATE_KEY, "{}", null)).toThrow(
      error
    );

    expect(storage.setItem).toHaveBeenCalledTimes(1);
  });

  it("does not retry non-quota storage failures", () => {
    const storage = createCapacityStorage();
    const previous = "{}";
    storage.setItem(STATE_KEY, previous);
    const error = new DOMException("Storage is unavailable", "SecurityError");
    vi.mocked(storage.setItem)
      .mockClear()
      .mockImplementation(() => {
        throw error;
      });

    expect(() =>
      writeAgentStoredState(storage, STATE_KEY, json, previous)
    ).toThrow(error);

    expect(storage.setItem).toHaveBeenCalledTimes(1);
    expect(storage.getItem(STATE_KEY)).toBe(previous);
    expect(storage.removeItem).not.toHaveBeenCalled();
  });

  it("preserves a compressed snapshot when its replacement cannot be stored", () => {
    const storage = createCapacityStorage();
    const previous = encodedJson();
    storage.setItem(STATE_KEY, previous);
    const error = new DOMException(
      "Storage quota exceeded",
      "QuotaExceededError"
    );
    vi.mocked(storage.setItem)
      .mockClear()
      .mockImplementation(() => {
        throw error;
      });
    const next = JSON.stringify({ ...snapshot, currentChatId: null });

    expect(() =>
      writeAgentStoredState(storage, STATE_KEY, next, previous)
    ).toThrow(error);

    expect(storage.setItem).toHaveBeenCalledTimes(1);
    expect(storage.getItem(STATE_KEY)).toBe(previous);
    expect(storage.removeItem).not.toHaveBeenCalled();
  });

  it.each([
    { version: 1, value: "" },
    { version: 1, value: "not compressed" },
    { version: 1, value: 123 },
    { version: 2, value: LZString.compressToUTF16(content) },
    {
      version: 1,
      value: LZString.compressToUTF16(content),
      metadataAbsent: "yes",
    },
  ])("rejects corrupt compressed message metadata: %j", (marker) => {
    const raw = JSON.parse(encodedJson());
    raw.messagesByChatId["chat-1"][0].metadata.agentCompressedContent = marker;

    expect(() => parseAgentStoredState(JSON.stringify(raw))).toThrow();
  });

  it("rejects invalid plain JSON", () => {
    expect(() => parseAgentStoredState("not JSON")).toThrow();
  });
});
