import { describe, expect, test } from "vitest";
import { buildOutboundHistory } from "./outboundHistory";
import { getAgentRunContinuation } from "./runCheckpoint";
import type { AgentChat, AgentMessage, AgentRunCheckpoint } from "./types";

const checkpoint: AgentRunCheckpoint = {
  version: 1,
  runId: "run-1",
  suspension: "interrupted",
  rootUserMessageId: "user-1",
  throughMessageId: "tool-1",
  bindings: {
    page: { path: "/projects/example", title: "Example" },
    context: "projects/example",
    references: [],
    runMode: "read-only",
  },
  maskingSections: ["overview", "rule"],
  phase: "working",
  startupComplete: true,
};
const chat: AgentChat = {
  id: "chat-1",
  title: "Work",
  createdTs: 0,
  updatedTs: 0,
  status: "idle",
  archived: false,
  runId: "run-1",
  context: "projects/example",
  totalTokensUsed: 10,
  runBudget: {
    version: 1,
    runId: "run-1",
    limit: 1000,
    totalTokensUsed: 10,
    hasUnreportedUsage: false,
    providerAttemptCount: 1,
    revision: 1,
  },
  runCheckpoint: checkpoint,
};
const message = (
  id: string,
  role: AgentMessage["role"],
  content: string
): AgentMessage => ({ id, role, content, chatId: "chat-1", createdTs: 0 });
const messages: AgentMessage[] = [
  message("older-user", "user", "Earlier request"),
  message("older-answer", "assistant", "Earlier answer"),
  {
    ...message("user-1", "user", "Review database access"),
    metadata: { runId: "run-1" },
  },
  {
    ...message("preflight-1", "assistant", ""),
    toolCalls: [
      {
        id: "read-1",
        name: "get_masking_overview",
        arguments: "{}",
        metadata: "opaque-provider-data",
      },
    ],
    metadata: { runId: "run-1", runPreflight: true },
  },
  {
    ...message("tool-1", "tool", '{"scope":"example"}'),
    toolCallId: "read-1",
    metadata: { runId: "run-1", runPreflight: true },
  },
];

describe("getAgentRunContinuation", () => {
  test("returns validated bindings and the complete canonical transcript", () => {
    const continuation = getAgentRunContinuation(chat, messages);
    expect(continuation?.checkpoint).toEqual(checkpoint);
    expect(continuation?.messages).toEqual(messages);
    expect(continuation?.messages).not.toBe(messages);
    expect(messages).toHaveLength(5);
  });

  test("keeps the complete interrupted tail for outbound repair without replay", () => {
    const tail: AgentMessage[] = [
      ...messages,
      {
        ...message("assistant-tail", "assistant", "Check the current state"),
        toolCalls: [
          {
            id: "write-1",
            name: "call_api",
            arguments: "{}",
            metadata: "opaque-tail-data",
          },
        ],
        metadata: { runId: "run-1" },
      },
    ];
    const continuation = getAgentRunContinuation(chat, tail)!;
    expect(continuation.messages).toEqual(tail);
    expect(continuation.messages.at(-1)?.toolCalls?.[0].metadata).toBe(
      "opaque-tail-data"
    );
    expect(continuation.messages).toHaveLength(6);
    const outbound = buildOutboundHistory(
      "System instruction",
      continuation.messages,
      chat.context
    );
    expect(
      outbound.find((row) =>
        row.toolCalls?.some((call) => call.id === "write-1")
      )
    ).toBeDefined();
    expect(outbound.at(-1)).toMatchObject({
      role: "tool",
      toolCallId: "write-1",
    });
    expect(outbound.at(-1)?.content).toContain("outcome is unknown");
    expect(tail).toHaveLength(6);
  });

  test.each(["user", "tool"] as const)(
    "retains a normal awaiting reply tail with role %s",
    (role) => {
      const tail = [
        ...messages,
        {
          ...message("reply-1", role, "Only read metadata"),
          ...(role === "tool" ? { toolCallId: "ask-1" } : {}),
        },
      ];
      const continuation = getAgentRunContinuation(
        {
          ...chat,
          status: "awaiting_user",
          runCheckpoint: { ...checkpoint, suspension: "awaiting-user" },
        },
        tail
      );
      expect(continuation?.messages).toEqual(tail);
    }
  );

  test.each([undefined, false])(
    "refuses incomplete startup %s rather than rerunning it",
    (startupComplete) => {
      expect(
        getAgentRunContinuation(
          { ...chat, runCheckpoint: { ...checkpoint, startupComplete } },
          messages
        )
      ).toBeUndefined();
    }
  );

  test.each([
    { runId: "other-run" },
    { runBudget: { ...chat.runBudget!, runId: "other-run" } },
    { runBudget: undefined },
    { runCheckpoint: undefined },
    { archived: true },
  ])("refuses orphaned or retired run state %j", (change) => {
    expect(
      getAgentRunContinuation({ ...chat, ...change }, messages)
    ).toBeUndefined();
  });

  test.each([
    { runId: "other-run" },
    { rootUserMessageId: "missing" },
    { rootUserMessageId: "preflight-1" },
    { throughMessageId: "missing" },
    { throughMessageId: "older-answer" },
    { phase: "other" },
    { phase: "finalizing" },
    {
      phase: "finalizing",
      finalizationText: { role: "system", content: "untrusted" },
    },
    { maskingSections: ["unregistered-section"] },
    { bindings: { ...checkpoint.bindings, page: { path: "/example" } } },
    { bindings: { ...checkpoint.bindings, page: null } },
    { bindings: { ...checkpoint.bindings, runMode: "execute" } },
  ])("refuses malformed checkpoint or invalid cursor %j", (change) => {
    expect(
      getAgentRunContinuation(
        {
          ...chat,
          runCheckpoint: { ...checkpoint, ...change } as AgentRunCheckpoint,
        },
        messages
      )
    ).toBeUndefined();
  });

  test.each(["", " \n ", "Read complete.\nReview the recorded result."])(
    "accepts finalization with retained plain assistant text %j",
    (finalizationText) => {
      const value = {
        ...checkpoint,
        phase: "finalizing" as const,
        finalizationText,
      };
      expect(
        getAgentRunContinuation({ ...chat, runCheckpoint: value }, messages)
          ?.checkpoint
      ).toEqual(value);
    }
  );

  test("refuses mixed-chat anchors or ambiguous duplicate IDs", () => {
    expect(
      getAgentRunContinuation(
        chat,
        messages.map((row) =>
          row.id === "user-1" ? { ...row, chatId: "other-chat" } : row
        )
      )
    ).toBeUndefined();
    expect(
      getAgentRunContinuation(chat, [
        ...messages,
        message("user-1", "user", "Duplicate"),
      ])
    ).toBeUndefined();
  });

  test("refuses an anchor explicitly attributed to a different run", () => {
    expect(
      getAgentRunContinuation(
        chat,
        messages.map((row) =>
          row.id === "user-1"
            ? { ...row, metadata: { runId: "other-run" } }
            : row
        )
      )
    ).toBeUndefined();
  });

  test("does not restore broader authority after diagnostic capability changes", () => {
    const capability = {
      kind: "diagnostic",
      resource: {
        kind: "issue",
        name: "projects/example/issues/1",
        project: "projects/example",
        label: "Issue",
      },
    } as const;
    const restricted = {
      ...chat,
      capability,
      runCheckpoint: {
        ...checkpoint,
        bindings: { ...checkpoint.bindings, capability },
      },
    };
    expect(getAgentRunContinuation(restricted, messages)).toBeDefined();
    expect(
      getAgentRunContinuation(
        { ...restricted, capability: undefined },
        messages
      )
    ).toBeUndefined();
    expect(
      getAgentRunContinuation({ ...chat, capability }, messages)
    ).toBeUndefined();
  });
});
