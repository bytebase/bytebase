import { describe, expect, test } from "vitest";
import type { AgentApiAttempt } from "../logic/apiAttempt";
import { agentApiExecutionEvidence } from "../logic/apiExecution";
import {
  boundAgentQueryResult,
  omitAgentQueryRows,
} from "../logic/queryResultRetention";
import type { PersistedAgentState } from "./agent";
import {
  canDiscardStoredAgentHistory,
  canRetireLegacyAgentState,
  preservesLegacyApiDispatchEvidence,
} from "./legacyStateRetention";
import { omitAgentQuerySamples } from "./storageRetention";

interface StoredState {
  currentChatId: string | null;
  chats: Record<string, unknown>[];
  messagesByChatId: Record<string, Record<string, unknown>[]>;
  draftsByChatId: Record<string, Record<string, unknown>>;
  pendingAskByChatId: Record<string, unknown>;
  queuedUserMessagesByChatId: Record<string, unknown>;
  activeChatIds: string[];
  [key: string]: unknown;
}

const state = (): StoredState => ({
  currentChatId: "chat",
  chats: [
    {
      id: "chat",
      title: "Work",
      status: "idle",
      archived: false,
      updatedTs: 1,
      totalTokensUsed: 10,
      custom: { value: "keep" },
    },
  ],
  messagesByChatId: {
    chat: [
      {
        id: "user",
        chatId: "chat",
        role: "user",
        content: "Review access",
        metadata: { custom: "keep" },
      },
      {
        id: "answer",
        chatId: "chat",
        role: "assistant",
        content: "Review complete",
      },
    ],
  },
  draftsByChatId: { chat: { text: "Unsent question", mentions: [] } },
  pendingAskByChatId: {},
  queuedUserMessagesByChatId: {},
  activeChatIds: [],
});

const queryState = (): StoredState => ({
  ...state(),
  messagesByChatId: {
    chat: [
      {
        id: "query",
        chatId: "chat",
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
        id: "result",
        chatId: "chat",
        role: "tool",
        toolCallId: "call",
        content: JSON.stringify({
          status: 200,
          response: {
            results: [
              {
                columnNames: ["value"],
                rows: [
                  { values: [{ stringValue: "first" }] },
                  { values: [{ stringValue: "second" }] },
                ],
                statement: "SELECT value FROM example_table",
              },
            ],
            custom: { keep: true },
          },
        }),
        metadata: { custom: "keep" },
      },
    ],
  },
});

const withoutSamples = (value: StoredState): StoredState =>
  omitAgentQuerySamples(
    value as unknown as PersistedAgentState
  ) as unknown as StoredState;

const receiptState = (scopeKey: string): StoredState => {
  const value = state();
  const receipt: AgentApiAttempt = {
    version: 1,
    id: "attempt",
    logicalActionId: "action",
    ordinal: 1,
    scopeKey,
    userName: "users/member@example.com",
    chatId: "chat",
    runId: "run",
    rootUserMessageId: "user",
    toolCallId: "call",
    operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
    path: "/bytebase.v1.DatabaseService/UpdateDatabase",
    fingerprint: "a".repeat(64),
    startedAt: 1,
    phase: "settled",
    settledAt: 2,
    execution: agentApiExecutionEvidence("rpc-success", {
      attemptId: "attempt",
      httpStatus: 200,
    }),
  };
  value.chats[0].apiAttempts = [receipt];
  value.messagesByChatId.chat.push({
    id: "result",
    role: "tool",
    toolCallId: "call",
    chatId: "chat",
    content: JSON.stringify({ status: 200, agentExecution: receipt.execution }),
    metadata: { apiAttemptId: "attempt", runId: "run" },
  });
  return value;
};

describe("canDiscardStoredAgentHistory", () => {
  const key = "bb-agent-state-v3.4:peer.member@example.com";
  test.each([
    "missing result",
    "different result",
    "different scope",
    "unowned result",
  ])("preserves unreadable raw receipt evidence with %s", (damage) => {
    const source = receiptState(key);
    if (damage === "missing result") source.messagesByChatId.chat.pop();
    else if (damage === "different result")
      source.messagesByChatId.chat.at(-1)!.toolCallId = "different";
    else if (damage === "different scope")
      (source.chats[0].apiAttempts as AgentApiAttempt[])[0].scopeKey = "other";
    else delete source.chats[0].apiAttempts;
    expect(canDiscardStoredAgentHistory(source, key)).toBe(false);
  });
  test("accepts a recognized result with its implicit legacy chat owner", () => {
    const source = receiptState(key);
    delete source.messagesByChatId.chat.at(-1)!.chatId;
    expect(canDiscardStoredAgentHistory(source, key)).toBe(true);
  });
});

describe("canRetireLegacyAgentState", () => {
  const migration = {
    scopeKey: "bb-agent-state-v3.4:peer.member@example.com",
    userEmail: "member@example.com",
  };
  const legacyKey = "bb-agent-state-v2.4:peer.member@example.com";
  test("retires the same owner's remapped V2 receipt after V3 preserves it and appends another receipt", () => {
    const source = receiptState(legacyKey);
    const saved = structuredClone(source);
    const receipts = saved.chats[0].apiAttempts as AgentApiAttempt[];
    receipts[0].scopeKey = migration.scopeKey;
    receipts.push({
      ...receipts[0],
      id: "later",
      logicalActionId: "later-action",
      toolCallId: "later-tool",
      execution: agentApiExecutionEvidence("rpc-success", {
        attemptId: "later",
        httpStatus: 200,
      }),
    });
    saved.messagesByChatId.chat.push({
      id: "later-result",
      role: "tool",
      chatId: "chat",
      toolCallId: "later-tool",
      content: JSON.stringify({
        status: 200,
        agentExecution: receipts[1].execution,
      }),
      metadata: { apiAttemptId: "later", runId: "run" },
    });
    expect(canRetireLegacyAgentState(source, saved, migration)).toBe(true);
    expect((source.chats[0].apiAttempts as AgentApiAttempt[])[0].scopeKey).toBe(
      legacyKey
    );
  });
  test.each([
    "different member",
    "different workspace",
    "different version",
    "changed evidence",
    "changed source field",
  ])("does not retire a remapped receipt with %s", (damage) => {
    const source = receiptState(legacyKey);
    const saved = structuredClone(source);
    const retained = (saved.chats[0].apiAttempts as AgentApiAttempt[])[0];
    retained.scopeKey = migration.scopeKey;
    if (damage === "different member")
      (source.chats[0].apiAttempts as AgentApiAttempt[])[0].userName =
        "users/other@example.com";
    else if (damage === "different workspace")
      retained.scopeKey = "bb-agent-state-v3.5:other.member@example.com";
    else if (damage === "different version")
      retained.scopeKey = "bb-agent-state-v4.4:peer.member@example.com";
    else if (damage === "changed evidence")
      retained.execution = agentApiExecutionEvidence("fetch-failed", {
        attemptId: "attempt",
      });
    else retained.fingerprint = "b".repeat(64);
    expect(
      canRetireLegacyAgentState(
        source,
        saved,
        damage === "different version"
          ? { ...migration, scopeKey: legacyKey }
          : migration
      )
    ).toBe(false);
  });
  test("accepts an exact recognized copy without modifying either state", () => {
    const source = state();
    const saved = structuredClone(source);
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
    expect(source).toEqual(state());
    expect(saved).toEqual(state());
  });

  test("retires a legacy dispatch gap only after the saved state preserves its evidence", () => {
    const source = state();
    source.chats[0].interrupted = true;
    source.messagesByChatId.chat.push({
      id: "request",
      chatId: "chat",
      role: "assistant",
      toolCalls: [
        {
          id: "call",
          name: "call_api",
          arguments: '{"operationId":"SQLService/Query"}',
        },
      ],
    });
    const saved = structuredClone(source);
    saved.chats[0].interrupted = false;
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    expect(preservesLegacyApiDispatchEvidence(source, saved)).toBe(false);
    saved.messagesByChatId.chat.push({
      id: "next",
      chatId: "chat",
      role: "user",
      content: "Another task",
    });
    expect(preservesLegacyApiDispatchEvidence(source, saved)).toBe(false);
    saved.messagesByChatId.chat.pop();
    expect(preservesLegacyApiDispatchEvidence(source, saved, "chat")).toBe(
      true
    );
    const removed = state();
    removed.currentChatId = null;
    removed.chats = [];
    removed.messagesByChatId = {};
    removed.draftsByChatId = {};
    expect(preservesLegacyApiDispatchEvidence(source, removed)).toBe(true);

    saved.chats[0].interrupted = true;
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
    saved.chats[0].interrupted = false;
    saved.messagesByChatId.chat.push({
      id: "result",
      chatId: "chat",
      role: "tool",
      toolCallId: "call",
      content: '{"status":200}',
    });
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
    saved.messagesByChatId.chat.pop();
    saved.chats[0].apiAttempts = [
      {
        version: 1,
        id: "attempt",
        chatId: "chat",
        logicalActionId: "action",
        ordinal: 1,
        scopeKey: "scope",
        userName: "users/member@example.com",
        runId: "run",
        rootUserMessageId: "user",
        toolCallId: "call",
        operationId: "bytebase.v1.SQLService.Query",
        path: "/bytebase.v1.SQLService/Query",
        fingerprint: "a".repeat(64),
        phase: "dispatching",
        startedAt: 1,
      },
    ];
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
  });

  test("accepts added normalized defaults and appended messages or chats", () => {
    const source = state();
    const saved = structuredClone(source);
    saved.currentChatId = "new";
    saved.chats[0] = {
      ...saved.chats[0],
      createdTs: 0,
      updatedTs: 2,
      totalTokensUsed: 20,
      status: "error",
      lastError: "Later failure",
      interrupted: true,
    };
    saved.messagesByChatId.chat[0].createdTs = 0;
    saved.messagesByChatId.chat.push({
      id: "later",
      chatId: "chat",
      role: "user",
      content: "Another question",
    });
    saved.chats.push({ id: "new", status: "idle" });
    saved.messagesByChatId.new = [];
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
  });

  test("accepts increased legacy usage uncertainty without discarding known totals", () => {
    const source = state();
    source.chats[0].hasUnreportedUsage = false;
    const saved = structuredClone(source);
    saved.chats[0].hasUnreportedUsage = true;
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
    expect(canRetireLegacyAgentState(saved, source)).toBe(false);
  });

  test("accepts shared Query row omission and sampling without changing the original states", () => {
    const source = queryState();
    const saved = withoutSamples(source);
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
    const sampled = structuredClone(source);
    sampled.messagesByChatId.chat[1].content = boundAgentQueryResult(
      String(source.messagesByChatId.chat[1].content),
      40
    );
    expect(canRetireLegacyAgentState(source, sampled)).toBe(true);
    expect(queryState()).toEqual(source);
    expect(withoutSamples(source)).toEqual(saved);
  });

  test("accepts Query row omission when the raw legacy envelope has no pending-ask map", () => {
    const source: Record<string, unknown> = queryState();
    const saved: Record<string, unknown> = withoutSamples(queryState());
    delete source.pendingAskByChatId;
    delete saved.pendingAskByChatId;
    expect(canRetireLegacyAgentState(source, saved)).toBe(true);
  });

  test("rejects malformed tool calls instead of using Query omission to prove preservation", () => {
    const source = queryState();
    source.messagesByChatId.chat[0].toolCalls = [null];
    const saved = structuredClone(source);
    saved.messagesByChatId.chat[1].content = omitAgentQueryRows(
      String(source.messagesByChatId.chat[1].content)
    );
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
  });

  test.each(["statement", "custom"])(
    "rejects Query %s changes outside reproducible rows",
    (field) => {
      const source = queryState();
      const saved = withoutSamples(source);
      const content = JSON.parse(
        String(saved.messagesByChatId.chat[1].content)
      );
      if (field === "statement")
        content.response.results[0].statement = "SELECT another_value";
      else content.response.custom = { keep: false };
      saved.messagesByChatId.chat[1].content = JSON.stringify(content);
      expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    }
  );

  test("does not erase unknown Query preview or retention marker fields during proof", () => {
    const source = queryState();
    const content = JSON.parse(
      boundAgentQueryResult(String(source.messagesByChatId.chat[1].content), 40)
    );
    content.agentQueryPreview.custom = "keep";
    source.messagesByChatId.chat[1].content = JSON.stringify(content);
    expect(canRetireLegacyAgentState(source, withoutSamples(source))).toBe(
      false
    );
    const marked = queryState();
    marked.messagesByChatId.chat[1].metadata = {
      queryResultRetention: { version: 1, level: "omitted", custom: "keep" },
    };
    expect(canRetireLegacyAgentState(marked, withoutSamples(marked))).toBe(
      false
    );
  });

  test.each(["another operation", "duplicate call"])(
    "rejects row omission without a unique Query call: %s",
    (reason) => {
      const source = queryState();
      source.messagesByChatId.chat[0].toolCalls =
        reason === "another operation"
          ? [
              {
                id: "call",
                name: "call_api",
                arguments: JSON.stringify({ operationId: "SQLService/Export" }),
              },
            ]
          : [
              {
                id: "call",
                name: "call_api",
                arguments: JSON.stringify({ operationId: "SQLService/Query" }),
              },
              {
                id: "call",
                name: "call_api",
                arguments: JSON.stringify({ operationId: "SQLService/Query" }),
              },
            ];
      const saved = structuredClone(source);
      saved.messagesByChatId.chat[1].content = omitAgentQueryRows(
        String(source.messagesByChatId.chat[1].content)
      );
      expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    }
  );

  test.each(["running", "paused", "awaiting_user"])(
    "rejects raw source status %s even when the saved copy is idle",
    (status) => {
      const source = state();
      source.chats[0].status = status;
      expect(canRetireLegacyAgentState(source, state())).toBe(false);
    }
  );

  test.each([{ runId: "run" }, { runCheckpoint: { runId: "run" } }])(
    "rejects source execution state %j",
    (execution) => {
      const source = state();
      Object.assign(source.chats[0], execution);
      expect(canRetireLegacyAgentState(source, structuredClone(source))).toBe(
        false
      );
    }
  );

  test.each([
    "activeChatIds",
    "pendingAskByChatId",
    "queuedUserMessagesByChatId",
  ])("rejects source work in %s", (key) => {
    const source = state();
    source[key] =
      key === "activeChatIds" ? ["chat"] : { chat: { toolCallId: "pending" } };
    expect(canRetireLegacyAgentState(source, structuredClone(source))).toBe(
      false
    );
  });

  test("rejects pending source messages", () => {
    const source = state();
    source.messagesByChatId.chat[1].metadata = { pending: true };
    expect(canRetireLegacyAgentState(source, structuredClone(source))).toBe(
      false
    );
  });

  test.each(["content", "metadata"])(
    "rejects changed or missing source message %s",
    (field) => {
      const source = state();
      const saved = structuredClone(source);
      saved.messagesByChatId.chat[0][field] =
        field === "content" ? "Changed request" : {};
      expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    }
  );

  test("rejects missing or reordered source messages", () => {
    const source = state();
    const saved = structuredClone(source);
    saved.messagesByChatId.chat.reverse();
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    saved.messagesByChatId.chat = [saved.messagesByChatId.chat[0]];
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
  });

  test("rejects source-only chats, messages, and drafts", () => {
    const source = state();
    const saved = structuredClone(source);
    saved.chats = [];
    saved.messagesByChatId = {};
    saved.draftsByChatId = {};
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    const missingMessage = structuredClone(source);
    missingMessage.messagesByChatId.chat.pop();
    expect(canRetireLegacyAgentState(source, missingMessage)).toBe(false);
    const missingDraft = structuredClone(source);
    missingDraft.draftsByChatId = {};
    expect(canRetireLegacyAgentState(source, missingDraft)).toBe(false);
  });

  test("rejects changed drafts and unknown nested fields", () => {
    const source = state();
    const saved = structuredClone(source);
    saved.draftsByChatId.chat.text = "Replacement draft";
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
    saved.draftsByChatId = structuredClone(source.draftsByChatId);
    saved.chats[0].custom = { value: "changed" };
    expect(canRetireLegacyAgentState(source, saved)).toBe(false);
  });

  test.each(["updatedTs", "totalTokensUsed"])(
    "requires finite, nondecreasing %s",
    (field) => {
      const source = state();
      const saved = structuredClone(source);
      for (const value of [0, NaN, Infinity, "20"])
        expect(
          canRetireLegacyAgentState(source, {
            ...saved,
            chats: [{ ...saved.chats[0], [field]: value }],
          })
        ).toBe(false);
    }
  );

  test.each([
    null,
    [],
    {},
    { threads: [] },
    { chats: [], messagesByChatId: [] },
    { chats: "invalid", messagesByChatId: {} },
  ])("rejects unrecognized envelopes %j", (value) => {
    expect(canRetireLegacyAgentState(value, state())).toBe(false);
    expect(canRetireLegacyAgentState(state(), value)).toBe(false);
  });

  test("rejects unknown source top-level fields even if the saved copy contains them", () => {
    const source = { ...state(), customBackup: { content: "keep" } };
    expect(canRetireLegacyAgentState(source, structuredClone(source))).toBe(
      false
    );
  });

  test("does not discard unrecognized values stored under transient field names", () => {
    const source = state();
    source.chats[0].lastError = { custom: "keep" };
    expect(canRetireLegacyAgentState(source, state())).toBe(false);
  });

  test("rejects malformed maps, duplicate identities, and orphaned source records", () => {
    for (const patch of [
      { draftsByChatId: [] },
      { activeChatIds: {} },
      { chats: [...state().chats, ...state().chats] },
      {
        messagesByChatId: {
          chat: [
            ...state().messagesByChatId.chat,
            state().messagesByChatId.chat[0],
          ],
        },
      },
      {
        messagesByChatId: {
          orphan: [{ id: "lost", role: "user", content: "keep" }],
        },
      },
      { draftsByChatId: { orphan: { text: "keep" } } },
    ]) {
      const malformed = { ...state(), ...patch };
      expect(
        canRetireLegacyAgentState(malformed, structuredClone(malformed))
      ).toBe(false);
    }
  });
});
