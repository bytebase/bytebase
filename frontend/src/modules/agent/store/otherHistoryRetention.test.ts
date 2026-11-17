import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { PersistedAgentState } from "./agent";
import {
  reclaimOtherAgentHistory,
  withAgentHistoryReclamationLocks,
} from "./otherHistoryRetention";
import * as stateStorage from "./stateStorage";

const member = "member@example.com";
const scopedKey = (workspace: string, email = member, version = 3) =>
  `bb-agent-state-v${version}.${workspace.length}:${workspace}.${email}`;
const currentKey = scopedKey("current");

const history = (ids = ["old", "new"], bytes = 100) => ({
  currentChatId: ids.at(-1) ?? null,
  chats: ids.map((id, index) => ({
    id,
    title: id,
    status: "idle",
    createdTs: index,
    updatedTs: index,
    totalTokensUsed: 0,
    archived: false,
  })),
  messagesByChatId: Object.fromEntries(
    ids.map((id) => [
      id,
      [
        {
          id: `message-${id}`,
          chatId: id,
          role: "user",
          content: "h".repeat(bytes),
        },
      ],
    ])
  ),
  draftsByChatId: {},
  pendingAskByChatId: {},
  queuedUserMessagesByChatId: {},
  activeChatIds: [],
});

const createStorage = () => {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    key: (index: number) => [...values.keys()][index] ?? null,
    getItem: vi.fn((key: string) => values.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      values.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      values.delete(key);
    }),
    clear: () => values.clear(),
  } satisfies Storage;
};

function fakeLocks() {
  const held: LockInfo[] = [];
  const pending: LockInfo[] = [];
  return {
    block(name: string, queued = false) {
      const lock: LockInfo = { name, mode: "exclusive" };
      (queued ? pending : held).push(lock);
    },
    query: vi.fn(async () => ({ held: [...held], pending: [...pending] })),
    request: vi.fn(
      async (
        name: string,
        options: LockOptions,
        action: (lock: Lock | null) => unknown
      ) => {
        if (
          options.ifAvailable &&
          [...held, ...pending].some((lock) => lock.name === name)
        )
          return action(null);
        const lock = { name, mode: options.mode ?? "exclusive" };
        held.push(lock);
        try {
          return await action(lock);
        } finally {
          held.splice(held.indexOf(lock), 1);
        }
      }
    ),
  };
}

let storage: ReturnType<typeof createStorage>;
let locks: ReturnType<typeof fakeLocks>;
const reclaim = (
  overrides: Partial<Parameters<typeof reclaimOtherAgentHistory>[1]> = {}
) =>
  reclaimOtherAgentHistory(storage, {
    currentKey,
    userEmail: member,
    mode: "automatic",
    isCurrent: () => true,
    ...overrides,
  });

beforeEach(() => {
  storage = createStorage();
  storage.setItem(currentKey, JSON.stringify(history(["current"])));
  locks = fakeLocks();
  vi.stubGlobal("navigator", { locks });
});
afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("withAgentHistoryReclamationLocks", () => {
  test("keeps scope retirement blocked by a newly opened chat during partial cleanup", async () => {
    locks.block(
      `bb-agent-chat-protection:${JSON.stringify(currentKey)}:"later"`
    );
    const action = vi.fn((ids: ReadonlySet<string>, ownsScope: boolean) => {
      expect([...ids]).toEqual(["old"]);
      expect(ownsScope).toBe(false);
      return 1;
    });

    expect(
      await withAgentHistoryReclamationLocks(
        currentKey,
        ["old"],
        () => true,
        action,
        { allowPartial: true }
      )
    ).toBe(1);
    expect(action).toHaveBeenCalledOnce();
  });

  test.each([
    { version: 2, kind: "open", pending: false },
    { version: 3, kind: "open", pending: false },
    { version: 2, kind: "running", pending: false },
    { version: 3, kind: "running", pending: false },
    { version: 2, kind: "open", pending: true },
    { version: 3, kind: "open", pending: true },
    { version: 2, kind: "running", pending: true },
    { version: 3, kind: "running", pending: true },
  ])(
    "preserves a v$version $kind claim with pending=$pending during partial reclamation",
    async ({ version, kind, pending }) => {
      const scope = scopedKey("current", member, version);
      locks.block(
        kind === "open"
          ? `bb-agent-chat-protection:${JSON.stringify(scope)}:"old"`
          : `bb-agent-run:${JSON.stringify([scope, "old"])}`,
        pending
      );
      const action = vi.fn((ids: ReadonlySet<string>, ownsScope: boolean) => {
        expect([...ids]).toEqual(["new"]);
        expect(ownsScope).toBe(false);
        return 1;
      });

      expect(
        await withAgentHistoryReclamationLocks(
          currentKey,
          ["old", "new"],
          () => true,
          action,
          { allowPartial: true }
        )
      ).toBe(1);
      expect(action).toHaveBeenCalledOnce();
    }
  );
});

describe("reclaimOtherAgentHistory", () => {
  test("compresses closed same-account workspace history without touching the current scope", async () => {
    const peer = scopedKey("another.workspace");
    const source = JSON.stringify(history(["old"], 1_100_000));
    storage.setItem(peer, source);
    const current = storage.getItem(currentKey);
    const freed = await reclaim();
    expect(freed).toBeGreaterThan(2_000_000);
    expect(freed).toBe(2 * (source.length - storage.getItem(peer)!.length));
    expect(stateStorage.parseAgentStoredState(storage.getItem(peer)!)).toEqual(
      JSON.parse(source)
    );
    expect(storage.getItem(currentKey)).toBe(current);
  });

  test("omits only reproducible Query rows while preserving result metadata", async () => {
    const peer = scopedKey("query-results");
    const source = history(["query"]) as unknown as PersistedAgentState;
    source.messagesByChatId.query = [
      {
        id: "query-call",
        chatId: "query",
        createdTs: 0,
        role: "assistant",
        toolCalls: [
          {
            id: "call",
            name: "call_api",
            arguments: JSON.stringify({ operationId: "SQLService/Query" }),
          },
        ],
      },
      {
        id: "query-result",
        chatId: "query",
        createdTs: 1,
        role: "tool",
        toolCallId: "call",
        content: JSON.stringify({
          status: 200,
          response: {
            custom: "keep",
            results: [
              {
                columnNames: ["value"],
                rows: [{ values: [{ stringValue: "h".repeat(1_100_000) }] }],
                statement: "SELECT value FROM example_table",
              },
              { error: "A later statement failed", rows: [] },
            ],
          },
        }),
      },
    ];
    storage.setItem(peer, JSON.stringify(source));
    expect(await reclaim()).toBeGreaterThan(0);
    const saved = stateStorage.parseAgentStoredState(
      storage.getItem(peer)!
    ) as PersistedAgentState;
    const result = JSON.parse(saved.messagesByChatId.query[1].content!);
    expect(result.response.custom).toBe("keep");
    expect(result.response.results[0]).toMatchObject({
      columnNames: ["value"],
      rows: [],
      statement: "SELECT value FROM example_table",
    });
    expect(result.response.results[1].error).toBe("A later statement failed");
  });

  test("evicts globally oldest completed chats after lossless compression cannot reach the target", async () => {
    vi.spyOn(stateStorage, "encodeAgentStoredState").mockImplementation(
      (json) => json
    );
    const younger = scopedKey("younger");
    const older = scopedKey("older");
    const youngState = history(["young"], 400_000);
    youngState.chats[0].updatedTs = 20;
    const oldState = history(["old", "middle"], 400_000);
    oldState.chats[0].updatedTs = 1;
    oldState.chats[1].updatedTs = 10;
    storage.setItem(younger, JSON.stringify(youngState));
    storage.setItem(older, JSON.stringify(oldState));
    expect(await reclaim()).toBeGreaterThan(0);
    expect(
      JSON.parse(storage.getItem(older)!).chats.map(
        (chat: { id: string }) => chat.id
      )
    ).not.toContain("old");
    expect(JSON.parse(storage.getItem(younger)!).chats).toHaveLength(1);
  });

  test("does not guess account identity from a matching email suffix or rewrite unknown data automatically", async () => {
    const foreign = scopedKey("foreign", `prefix.${member}`);
    const unknown = scopedKey("unknown");
    const malformed = scopedKey("malformed");
    const raw = JSON.stringify({
      ...history(["old"], 1_100_000),
      futureData: "keep",
    });
    storage.setItem(foreign, JSON.stringify(history(["foreign"], 1_100_000)));
    storage.setItem(unknown, raw);
    storage.setItem(malformed, "not JSON");
    const before = [foreign, unknown, malformed].map((key) =>
      storage.getItem(key)
    );
    expect(await reclaim()).toBe(0);
    expect(
      [foreign, unknown, malformed].map((key) => storage.getItem(key))
    ).toEqual(before);
  });

  test.each(["open", "busy", "pending open", "pending busy"] as const)(
    "keeps a peer protected by %s leases in both modes",
    async (condition) => {
      const peer = scopedKey("protected");
      const source = JSON.stringify(history(["old"], 1_100_000));
      storage.setItem(peer, source);
      const name = condition.includes("open")
        ? `bb-agent-chat-protection:${JSON.stringify(peer)}:"old"`
        : `bb-agent-run:${JSON.stringify([peer, "old"])}`;
      locks.block(name, condition.startsWith("pending"));
      expect(await reclaim()).toBe(0);
      expect(await reclaim({ mode: "confirmed" })).toBe(0);
      expect(storage.getItem(peer)).toBe(source);
    }
  );

  test("preserves raw paused work and saved drafts automatically but explicit consent can discard closed history", async () => {
    const peer = scopedKey("paused", "another@example.com");
    const ownPeer = scopedKey("paused-own");
    const source = JSON.stringify({
      ...history(["paused"], 1_100_000),
      chats: [
        {
          ...history(["paused"]).chats[0],
          status: "paused",
          runId: "stale",
          runCheckpoint: { runId: "stale" },
        },
      ],
      draftsByChatId: { paused: { text: "Saved draft" } },
      activeChatIds: ["paused"],
    });
    storage.setItem(peer, source);
    storage.setItem(ownPeer, source);
    expect(await reclaim()).toBe(0);
    expect(storage.getItem(ownPeer)).toBe(source);
    expect(await reclaim({ mode: "confirmed" })).toBe(
      [peer, ownPeer].reduce(
        (bytes, key) => bytes + 2 * (key.length + source.length),
        0
      )
    );
    expect(storage.getItem(peer)).toBeNull();
    expect(storage.getItem(ownPeer)).toBeNull();
  });

  test.each([
    { text: "", mention: { start: 0, end: 1 } },
    { text: "", mentions: "unrecognized" },
    { text: 0 },
    { text: "", futureField: false },
  ])("protects nonempty or unrecognized saved draft data %j", async (draft) => {
    const peer = scopedKey("saved-draft");
    const source = JSON.stringify({
      ...history(["old"], 1_100_000),
      draftsByChatId: { old: draft },
    });
    storage.setItem(peer, source);
    expect(await reclaim()).toBe(0);
    expect(storage.getItem(peer)).toBe(source);
  });

  test("confirmed cleanup clears opaque history across accounts but never preferences, window state, or the current key", async () => {
    const keys = [
      "bb-agent-state-v2",
      "bb-agent-state-v2.legacy",
      scopedKey("foreign", "other@example.com"),
      "bb-agent-state-v3.opaque-backup",
    ];
    for (const key of keys) storage.setItem(key, "opaque history");
    const unrelated = [
      "bb-agent-window",
      "bb-agent-preferences-v1.example",
      "bb-agent-ui-v1.example",
      "application-state",
      "bb-agent-state-v30.example",
    ];
    for (const key of unrelated) storage.setItem(key, "keep");
    expect(await reclaim({ mode: "confirmed" })).toBe(
      keys.reduce(
        (bytes, key) => bytes + 2 * (key.length + "opaque history".length),
        0
      )
    );
    for (const key of keys) expect(storage.getItem(key)).toBeNull();
    for (const key of unrelated) expect(storage.getItem(key)).toBe("keep");
    expect(storage.getItem(currentKey)).not.toBeNull();
  });

  test("preserves the current scope's only v2 source during other-history cleanup", async () => {
    const legacyKey = currentKey.replace("state-v3", "state-v2");
    storage.removeItem(currentKey);
    storage.setItem(legacyKey, "Only saved current history");

    expect(await reclaim({ mode: "confirmed" })).toBe(0);

    expect(storage.getItem(legacyKey)).toBe("Only saved current history");
    expect(storage.getItem(currentKey)).toBeNull();
  });

  test("does not remove unscoped opaque history while any Agent execution lease is active", async () => {
    storage.setItem("bb-agent-state-v2.legacy", "opaque history");
    locks.block(`bb-agent-run:${JSON.stringify([currentKey, "active"])}`);
    expect(await reclaim({ mode: "confirmed" })).toBe(0);
    expect(storage.getItem("bb-agent-state-v2.legacy")).toBe("opaque history");
  });

  test("does not confuse an unrelated current selection claim with an active legacy writer", async () => {
    storage.setItem("bb-agent-state-v2.legacy", "opaque history");
    locks.block(
      `bb-agent-chat-protection:${JSON.stringify(currentKey)}:"current"`
    );
    expect(await reclaim({ mode: "confirmed" })).toBeGreaterThan(0);
    expect(storage.getItem("bb-agent-state-v2.legacy")).toBeNull();
  });

  test("protects opaque scoped history when its version alias has a live tab", async () => {
    const peer = scopedKey("opaque");
    const legacy = peer.replace("state-v3", "state-v2");
    storage.setItem(peer, "opaque history");
    locks.block(
      `bb-agent-chat-protection:${JSON.stringify(legacy)}:"unknown-chat"`
    );
    expect(await reclaim({ mode: "confirmed" })).toBe(0);
    expect(storage.getItem(peer)).toBe("opaque history");
  });

  test("retires a verified v2 backup before trimming its peer and skips an unproven backup", async () => {
    const peer = scopedKey("duplicated");
    const legacy = peer.replace("state-v3", "state-v2");
    const source = JSON.stringify(history(["old"], 1_100_000));
    storage.setItem(
      peer,
      stateStorage.encodeAgentStoredState(source, null, true)
    );
    storage.setItem(legacy, source);
    expect(await reclaim()).toBeGreaterThan(0);
    expect(storage.getItem(legacy)).toBeNull();
    const unmatched = JSON.stringify({
      ...history(["old"], 1_100_000),
      futureData: "unique",
    });
    storage.setItem(legacy, unmatched);
    const before = storage.getItem(peer);
    expect(await reclaim()).toBe(0);
    expect(storage.getItem(legacy)).toBe(unmatched);
    expect(storage.getItem(peer)).toBe(before);
  });

  test("does not report reclaimed bytes after rejected writes or removals", async () => {
    const peer = scopedKey("failure");
    const source = JSON.stringify(history(["old"], 1_100_000));
    storage.setItem(peer, source);
    storage.setItem.mockImplementation(() => {
      throw new Error("Write failed");
    });
    expect(await reclaim()).toBe(0);
    expect(storage.getItem(peer)).toBe(source);
    storage.removeItem.mockImplementation(() => {
      throw new Error("Remove failed");
    });
    expect(await reclaim({ mode: "confirmed" })).toBe(0);
    expect(storage.getItem(peer)).toBe(source);
  });

  test("does not prune v3 while its verified v2 backup cannot be removed", async () => {
    const peer = scopedKey("backup-remove-failure");
    const legacy = peer.replace("state-v3", "state-v2");
    const source = JSON.stringify(history(["old"], 1_100_000));
    const encoded = stateStorage.encodeAgentStoredState(source, null, true);
    storage.setItem(peer, encoded);
    storage.setItem(legacy, source);
    storage.removeItem.mockImplementation(() => {
      throw new Error("Remove failed");
    });
    expect(await reclaim()).toBe(0);
    expect(storage.getItem(peer)).toBe(encoded);
    expect(storage.getItem(legacy)).toBe(source);
  });

  test("does not reclaim without protection APIs or a matching current account", async () => {
    const peer = scopedKey("other");
    const source = JSON.stringify(history(["old"], 1_100_000));
    storage.setItem(peer, source);
    expect(await reclaim({ userEmail: "other@example.com" })).toBe(0);
    vi.stubGlobal("navigator", {});
    expect(await reclaim()).toBe(0);
    expect(await reclaim({ mode: "confirmed" })).toBe(0);
    expect(storage.getItem(peer)).toBe(source);
  });

  test.each([
    ["automatic", "identity"],
    ["automatic", "source"],
    ["confirmed", "identity"],
    ["confirmed", "source"],
  ] as const)(
    "%s rechecks %s after asynchronous lock acquisition",
    async (mode, change) => {
      const peer = scopedKey("changed");
      const source = JSON.stringify(history(["old"], 1_100_000));
      storage.setItem(peer, source);
      let current = true;
      const request = locks.request.getMockImplementation()!;
      locks.request.mockImplementation(async (name, options, action) => {
        if (name === "bb-agent-storage-budget-v1") {
          if (change === "identity") current = false;
          else storage.setItem(peer, "Changed by another tab");
        }
        return request(name, options, action);
      });
      expect(await reclaim({ mode, isCurrent: () => current })).toBe(0);
      expect(storage.getItem(peer)).toBe(
        change === "source" ? "Changed by another tab" : source
      );
    }
  );

  test("force reclaims some capacity below the soft target for a native quota failure", async () => {
    const peer = scopedKey("small");
    const source = JSON.stringify(history(["old"], 10_000));
    storage.setItem(peer, source);
    expect(await reclaim()).toBe(0);
    expect(await reclaim({ force: true })).toBeGreaterThan(0);
  });
});
