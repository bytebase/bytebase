import { afterEach, describe, expect, it, vi } from "vitest";
import * as queryResultRetention from "../logic/queryResultRetention";
import {
  boundAgentQueryResult,
  getAgentQuerySampleBytes,
} from "../logic/queryResultRetention";
import type { AgentMessage } from "../logic/types";
import type { PersistedAgentState } from "./agent";
import {
  omitAgentQuerySamples,
  retainAgentQuerySamples,
} from "./storageRetention";

const queryContent = (value = "sample") =>
  JSON.stringify({
    status: 200,
    response: {
      results: [
        {
          columnNames: ["value"],
          columnTypeNames: ["TEXT"],
          rows: [{ values: [{ stringValue: value }] }],
          statement: "SELECT value FROM example_table",
          maskingReasons: [{ reason: "policy" }],
        },
        { error: "Later statement failed", rows: [] },
      ],
      accessGrant: "projects/example/accessGrants/example",
      unknownProtectedField: { value: "preserve" },
    },
  });

const queryMessages = (
  id: string,
  createdTs: number,
  content = queryContent(),
  operationId = "SQLService/Query"
): AgentMessage[] => [
  {
    id: `assistant-${id}`,
    chatId: id,
    createdTs: createdTs - 1,
    role: "assistant",
    toolCalls: [
      {
        id: `call-${id}`,
        name: "call_api",
        arguments: JSON.stringify({
          operationId,
          body: {
            name: "instances/example/databases/example",
            statement: "SELECT value FROM example_table",
          },
        }),
      },
    ],
  },
  {
    id: `tool-${id}`,
    chatId: id,
    createdTs,
    role: "tool",
    toolCallId: `call-${id}`,
    content,
    metadata: { runId: `run-${id}` },
  },
];

const stateWith = (
  groups: Record<string, AgentMessage[]>
): PersistedAgentState => ({
  currentChatId: Object.keys(groups)[0] ?? null,
  chats: Object.keys(groups).map((id) => ({
    id,
    title: id,
    createdTs: 1,
    updatedTs: 2,
    status: "idle",
    archived: false,
    totalTokensUsed: 42,
  })),
  messagesByChatId: groups,
  pendingAskByChatId: {},
  draftsByChatId: { draft: { text: "member draft" } },
  queuedUserMessagesByChatId: {},
});

const resultMessage = (state: PersistedAgentState, id: string) =>
  state.messagesByChatId[id][1];

const parsedResult = (state: PersistedAgentState, id: string) =>
  JSON.parse(resultMessage(state, id).content!);

afterEach(() => vi.restoreAllMocks());

describe("Agent query sample retention", () => {
  it("omits the oldest eligible sample across chats without reordering messages", () => {
    const state = stateWith({
      newest: queryMessages("newest", 30),
      oldest: queryMessages("oldest", 10),
      middle: queryMessages("middle", 20),
    });
    const before = structuredClone(state);
    const oneSample = getAgentQuerySampleBytes(queryContent())!;

    const retained = retainAgentQuerySamples(state, 2 * oneSample);

    expect(parsedResult(retained, "oldest").response.results[0].rows).toEqual(
      []
    );
    expect(
      resultMessage(retained, "oldest").metadata?.queryResultRetention
    ).toEqual({
      version: 1,
      level: "omitted",
    });
    expect(
      parsedResult(retained, "middle").response.results[0].rows
    ).toHaveLength(1);
    expect(
      parsedResult(retained, "newest").response.results[0].rows
    ).toHaveLength(1);
    expect(Object.keys(retained.messagesByChatId)).toEqual([
      "newest",
      "oldest",
      "middle",
    ]);
    expect(state).toEqual(before);
    expect(retained.chats).toBe(state.chats);
    expect(retained.draftsByChatId).toBe(state.draftsByChatId);
  });

  it("keeps an exact aggregate byte boundary and omits one byte beyond it", () => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    const bytes = getAgentQuerySampleBytes(queryContent())!;

    expect(
      parsedResult(retainAgentQuerySamples(state, bytes), "chat").response
        .results[0].rows
    ).toHaveLength(1);
    expect(
      parsedResult(retainAgentQuerySamples(state, bytes - 1), "chat").response
        .results[0].rows
    ).toEqual([]);
  });

  it("counts UTF-8 serialized row arrays across result sets", () => {
    const content = JSON.stringify({
      status: 200,
      response: {
        results: [
          { rows: [{ values: [{ stringValue: "\u90ae\u7bb1\ud83d\ude80" }] }] },
          { rows: [{ values: [{ stringValue: "\u00e9" }] }] },
        ],
      },
    });
    const state = stateWith({ chat: queryMessages("chat", 1, content) });
    const rows = JSON.parse(content).response.results;
    const bytes = rows.reduce(
      (sum: number, result: { rows: unknown[] }) =>
        sum + new TextEncoder().encode(JSON.stringify(result.rows)).length,
      0
    );

    expect(
      parsedResult(retainAgentQuerySamples(state, bytes), "chat").response
        .results[1].rows
    ).toHaveLength(1);
    expect(
      parsedResult(
        retainAgentQuerySamples(state, bytes - 1),
        "chat"
      ).response.results.map((result: { rows: unknown[] }) => result.rows)
    ).toEqual([[], []]);
  });

  it("bounds positively identified unmarked legacy Query results before retention", () => {
    const content = queryContent("x".repeat(70 * 1024));
    const state = stateWith({ chat: queryMessages("chat", 1, content) });

    const retained = retainAgentQuerySamples(state);

    expect(
      getAgentQuerySampleBytes(resultMessage(retained, "chat").content!)
    ).toBeLessThanOrEqual(64 * 1024);
    expect(parsedResult(retained, "chat").agentQueryPreview).toMatchObject({
      version: 1,
    });
    expect(resultMessage(retained, "chat").id).toBe("tool-chat");
    expect(retained.messagesByChatId.chat[0]).toEqual(
      state.messagesByChatId.chat[0]
    );
    expect(resultMessage(state, "chat").content).toBe(content);
  });

  it("uses a 512 KiB default across active and completed chats", () => {
    const groups = Object.fromEntries(
      Array.from({ length: 9 }, (_, index) => [
        `chat-${index}`,
        queryMessages(
          `chat-${index}`,
          index + 1,
          queryContent("x".repeat(60 * 1024))
        ),
      ])
    );
    const state = stateWith(groups);
    state.chats[0].status = "running";
    state.chats[0].runId = "active-run";

    const retained = retainAgentQuerySamples(state);
    const bytes = Object.keys(groups).reduce(
      (sum, id) =>
        sum + getAgentQuerySampleBytes(resultMessage(retained, id).content!)!,
      0
    );

    expect(bytes).toBeLessThanOrEqual(512 * 1024);
    expect(parsedResult(retained, "chat-0").response.results[0].rows).toEqual(
      []
    );
    expect(
      parsedResult(retained, "chat-1").response.results[0].rows
    ).toHaveLength(1);
    expect(retained.chats).toBe(state.chats);
    expect(state.chats[0]).toMatchObject({
      status: "running",
      runId: "active-run",
    });
  });

  it.each(["SQLService/Query", "bytebase.v1.SQLService.Query"])(
    "classifies only an exact supported Query operation: %s",
    (operation) => {
      const state = stateWith({
        chat: queryMessages("chat", 1, queryContent(), operation),
      });
      expect(
        parsedResult(omitAgentQuerySamples(state), "chat").response.results[0]
          .rows
      ).toEqual([]);
    }
  );

  it.each([
    "SQLService/QueryExtra",
    "/bytebase.v1.SQLService/Query",
    "SQLService/Export",
    "SQLService/Execute",
  ])("protects noncanonical or non-Query operation %s", (operation) => {
    const state = stateWith({
      chat: queryMessages("chat", 1, queryContent(), operation),
    });
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it.each([
    ["missing call", (messages: AgentMessage[]) => messages.slice(1)],
    [
      "call after result",
      (messages: AgentMessage[]) => [...messages].reverse(),
    ],
    [
      "malformed arguments",
      (messages: AgentMessage[]) => {
        messages[0].toolCalls![0].arguments = "not JSON";
        return messages;
      },
    ],
    [
      "wrong tool",
      (messages: AgentMessage[]) => {
        messages[0].toolCalls![0].name = "other_tool";
        return messages;
      },
    ],
    [
      "duplicate call",
      (messages: AgentMessage[]) => {
        messages[0].toolCalls!.push({ ...messages[0].toolCalls![0] });
        return messages;
      },
    ],
    [
      "duplicate non-Query call",
      (messages: AgentMessage[]) => {
        messages[0].toolCalls!.push({
          ...messages[0].toolCalls![0],
          name: "other_tool",
        });
        return messages;
      },
    ],
    [
      "pending result",
      (messages: AgentMessage[]) => {
        messages[1].metadata!.pending = true;
        return messages;
      },
    ],
  ] as const)("protects %s", (_, change) => {
    const state = stateWith({ chat: change(queryMessages("chat", 1)) });
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it("protects the current pending tool even without a message pending flag", () => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    state.pendingAskByChatId.chat = {
      toolCallId: "call-chat",
      prompt: "Confirm request",
      kind: "confirm",
    };
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it.each([
    "not JSON",
    JSON.stringify({
      status: 200,
      response: { results: [{ rows: "invalid" }] },
    }),
    JSON.stringify({
      status: 200,
      response: { results: [{ rows: [{ values: "invalid" }] }] },
    }),
    JSON.stringify({
      status: 403,
      error: { message: "Permission denied", details: [{ reason: "policy" }] },
    }),
  ])("protects malformed or non-Query envelope %s", (content) => {
    const state = stateWith({ chat: queryMessages("chat", 1, content) });
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it("protects an ambiguous ID even when its duplicate occurs after the result", () => {
    const messages = queryMessages("chat", 1);
    messages.push({
      id: "duplicate",
      chatId: "chat",
      createdTs: 3,
      role: "assistant",
      toolCalls: [{ ...messages[0].toolCalls![0] }],
    });
    const state = stateWith({ chat: messages });
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it.each([
    null,
    { version: 2, level: "omitted" },
    { version: 1, level: "unknown" },
  ])("protects unknown per-message retention metadata %s", (marker) => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    const metadata = resultMessage(state, "chat").metadata as unknown as Record<
      string,
      unknown
    >;
    metadata.queryResultRetention = marker;
    expect(omitAgentQuerySamples(state)).toEqual(state);
  });

  it("preserves IDs, arguments, all non-row data, metadata, and outcomes", () => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    const protectedMessage: AgentMessage = {
      id: "mutation",
      chatId: "chat",
      createdTs: 3,
      role: "tool",
      toolCallId: "mutation-call",
      content: "Write completed",
    };
    state.messagesByChatId.chat.push(protectedMessage);
    const original = parsedResult(state, "chat");

    const retained = omitAgentQuerySamples(state);
    const result = parsedResult(retained, "chat");

    expect(result.response).toEqual({
      ...original.response,
      results: original.response.results.map((result: { rows: unknown[] }) => ({
        ...result,
        rows: [],
      })),
    });
    expect(result.agentQueryPreview).toMatchObject({
      version: 1,
      level: "omitted",
    });
    expect(resultMessage(retained, "chat")).toMatchObject({
      id: "tool-chat",
      toolCallId: "call-chat",
      metadata: { runId: "run-chat" },
    });
    expect(retained.messagesByChatId.chat[0]).toBe(
      state.messagesByChatId.chat[0]
    );
    expect(retained.messagesByChatId.chat[2]).toBe(protectedMessage);
    expect(retained.pendingAskByChatId).toBe(state.pendingAskByChatId);
    expect(retained.queuedUserMessagesByChatId).toBe(
      state.queuedUserMessagesByChatId
    );
  });

  it("honors an omission marker even when stale rows fit the budget", () => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    resultMessage(state, "chat").metadata!.queryResultRetention = {
      version: 1,
      level: "omitted",
    };

    const retained = retainAgentQuerySamples(state, Number.POSITIVE_INFINITY);
    expect(parsedResult(retained, "chat").response.results[0].rows).toEqual([]);
    expect(retainAgentQuerySamples(retained)).toEqual(retained);
  });

  it("is idempotent for bounded snapshots and leaves histories without eligible samples alone", () => {
    const state = stateWith({
      chat: queryMessages("chat", 1, boundAgentQueryResult(queryContent())),
    });
    const retained = retainAgentQuerySamples(state);
    expect(retainAgentQuerySamples(retained)).toEqual(retained);

    const history = stateWith({
      chat: [
        {
          id: "user",
          chatId: "chat",
          role: "user",
          createdTs: 1,
          content: "Protected conversation",
        },
      ],
    });
    expect(omitAgentQuerySamples(history)).toBe(history);
  });

  it("preserves bounded sampled content and snapshot references without reserializing", () => {
    const content = JSON.stringify(
      JSON.parse(boundAgentQueryResult(queryContent())),
      null,
      2
    );
    const state = stateWith({ chat: queryMessages("chat", 1, content) });
    const message = resultMessage(state, "chat");
    const inspect = vi.spyOn(queryResultRetention, "getAgentQuerySampleInfo");
    const bound = vi.spyOn(queryResultRetention, "boundAgentQueryResult");

    const retained = retainAgentQuerySamples(state);

    expect(inspect).toHaveBeenCalledTimes(1);
    expect(bound).not.toHaveBeenCalled();
    expect(retained).toBe(state);
    expect(retained.messagesByChatId).toBe(state.messagesByChatId);
    expect(resultMessage(retained, "chat")).toBe(message);
    expect(message.content).toBe(content);
    expect(retainAgentQuerySamples(retained)).toBe(retained);
  });

  it("preserves snapshot references after samples have been irreversibly omitted", () => {
    const state = stateWith({ chat: queryMessages("chat", 1) });
    const omitted = omitAgentQuerySamples(state);
    const inspect = vi.spyOn(queryResultRetention, "getAgentQuerySampleInfo");
    const omit = vi.spyOn(queryResultRetention, "omitAgentQueryRows");

    expect(omitAgentQuerySamples(omitted)).toBe(omitted);
    expect(inspect).toHaveBeenCalledTimes(1);
    expect(omit).not.toHaveBeenCalled();
    expect(retainAgentQuerySamples(omitted)).toBe(omitted);
  });

  it("rechecks an oversized marked preview against the per-response bound", () => {
    const content = boundAgentQueryResult(
      queryContent("x".repeat(70 * 1024)),
      128 * 1024
    );
    const state = stateWith({ chat: queryMessages("chat", 1, content) });

    const retained = retainAgentQuerySamples(state);

    expect(getAgentQuerySampleBytes(content)).toBeGreaterThan(64 * 1024);
    expect(
      getAgentQuerySampleBytes(resultMessage(retained, "chat").content!)
    ).toBe(0);
    expect(parsedResult(retained, "chat").agentQueryPreview.results[0]).toEqual(
      {
        receivedRows: 1,
        retainedRows: 0,
      }
    );
    expect(resultMessage(state, "chat").content).toBe(content);
  });

  it("keeps malformed marked previews protected and reference-stable", () => {
    const value = JSON.parse(boundAgentQueryResult(queryContent()));
    value.agentQueryPreview.results[0].retainedRows = 0;
    const content = JSON.stringify(value);
    const state = stateWith({ chat: queryMessages("chat", 1, content) });

    expect(retainAgentQuerySamples(state)).toBe(state);
    expect(omitAgentQuerySamples(state)).toBe(state);
    expect(resultMessage(state, "chat").content).toBe(content);
  });
});
