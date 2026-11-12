import { describe, expect, test } from "vitest";
import { buildOutboundHistory } from "./outboundHistory";
import {
  getAgentRunClarificationBindings,
  getAgentRunContinuation,
} from "./runCheckpoint";
import type {
  AgentChat,
  AgentMessage,
  AgentResourceReference,
  AgentRunCheckpoint,
} from "./types";

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

describe("getAgentRunClarificationBindings", () => {
  const resource: AgentResourceReference = {
    kind: "database",
    name: "instances/prod/databases/orders",
    project: "projects/example",
    label: "Orders",
  };
  const ask: AgentMessage = {
    ...message("ask-message", "assistant", ""),
    toolCalls: [
      {
        id: "ask-database",
        name: "ask_user",
        arguments: '{"kind":"input","prompt":"Which database?"}',
      },
    ],
    metadata: { runId: "run-1" },
  };
  const reply: AgentMessage = {
    ...message("reply-message", "tool", '{"kind":"input","answer":"@Orders"}'),
    toolCallId: "ask-database",
    metadata: { runId: "run-1", resourceReference: resource },
  };
  const awaiting: AgentChat = {
    ...chat,
    status: "awaiting_user",
    runCheckpoint: {
      ...checkpoint,
      suspension: "awaiting-user",
      throughMessageId: ask.id,
      bindings: {
        ...checkpoint.bindings,
        runMode: "default",
        instanceName: "instances/previous",
        maskingPage: { kind: "semantic-types" },
      },
    },
  };
  const transcript = [...messages, ask, reply];

  test("refines explicit clarification scope without rebinding the saved page", () => {
    expect(
      getAgentRunClarificationBindings(
        {
          ...awaiting,
          context: "projects/elsewhere",
          page: { path: "/projects/elsewhere", title: "Elsewhere" },
        },
        transcript
      )
    ).toEqual({
      page: { path: "/projects/example", title: "Example" },
      context: "projects/example",
      references: [resource],
      runMode: "read-only",
      instanceName: undefined,
      maskingPage: { kind: "semantic-types" },
    });
    expect(awaiting.runCheckpoint?.bindings.references).toEqual([]);
  });

  test("takes the latest normalized explicit clarification references", () => {
    const project: AgentResourceReference = {
      kind: "project",
      name: "projects/example",
      project: "projects/example",
      label: "Example",
    };
    const nextAsk: AgentMessage = {
      ...ask,
      id: "next-ask-message",
      toolCalls: [{ ...ask.toolCalls![0], id: "next-ask" }],
    };
    const nextReply: AgentMessage = {
      ...reply,
      id: "next-reply-message",
      toolCallId: "next-ask",
      metadata: {
        runId: "run-1",
        resourceReferences: [{ ...resource, label: " Orders " }, project],
      },
    };
    expect(
      getAgentRunClarificationBindings(awaiting, [
        ...transcript,
        nextAsk,
        nextReply,
      ])?.references
    ).toEqual([project, resource]);
  });

  test.each([
    { name: "missing", metadata: undefined },
    { name: "another run", metadata: { ...reply.metadata, runId: "run-2" } },
    { name: "no references", metadata: { runId: "run-1" } },
    {
      name: "invalid references",
      metadata: {
        runId: "run-1",
        resourceReference: { ...resource, project: "invalid" },
      },
    },
  ])("does not refine a reply with $name metadata", ({ metadata }) => {
    expect(
      getAgentRunClarificationBindings(awaiting, [
        ...messages,
        ask,
        { ...reply, metadata },
      ])
    ).toBeUndefined();
  });

  test.each([
    { name: "unmatched", toolCallId: "different-ask" },
    { name: "non-input", content: '{"kind":"choose","answer":"@Orders"}' },
    { name: "empty", content: '{"kind":"input","answer":" "}' },
    { name: "malformed", content: "invalid" },
  ])("does not refine a $name reply", ({ name: _name, ...change }) => {
    expect(
      getAgentRunClarificationBindings(awaiting, [
        ...messages,
        ask,
        { ...reply, ...change },
      ])
    ).toBeUndefined();
  });

  test("does not trust an input reply matched to a different tool", () => {
    expect(
      getAgentRunClarificationBindings(awaiting, [
        ...messages,
        { ...ask, toolCalls: [{ ...ask.toolCalls![0], name: "call_api" }] },
        reply,
      ])
    ).toBeUndefined();
  });

  test.each([reply.id, "after-reply"])(
    "does not reapply a reply at or before cursor %s",
    (throughMessageId) => {
      expect(
        getAgentRunClarificationBindings(
          {
            ...awaiting,
            runCheckpoint: {
              ...awaiting.runCheckpoint!,
              throughMessageId,
            },
          },
          [...transcript, message("after-reply", "assistant", "Checking")]
        )
      ).toBeUndefined();
    }
  );

  test("does not refine from a later user turn", () => {
    expect(
      getAgentRunClarificationBindings(awaiting, [
        ...messages,
        message("later-user", "user", "A new request"),
        ask,
        reply,
      ])
    ).toBeUndefined();
  });

  test("does not change bindings without a new explicit clarification mention", () => {
    expect(
      getAgentRunClarificationBindings(
        {
          ...awaiting,
          context: "projects/elsewhere",
          page: { path: "/projects/elsewhere", title: "Elsewhere" },
        },
        [...messages, ask]
      )
    ).toBeUndefined();
    expect(
      getAgentRunClarificationBindings(
        {
          ...awaiting,
          runCheckpoint: {
            ...awaiting.runCheckpoint!,
            bindings: {
              ...awaiting.runCheckpoint!.bindings,
              references: [resource],
            },
          },
        },
        transcript
      )
    ).toBeUndefined();
  });

  test.each([
    { runId: "other-run" },
    { runBudget: undefined },
    { runCheckpoint: undefined },
    {
      runCheckpoint: {
        ...awaiting.runCheckpoint!,
        startupComplete: false,
      },
    },
    {
      runCheckpoint: {
        ...awaiting.runCheckpoint!,
        suspension: "interrupted" as const,
      },
    },
  ])("requires a validated run awaiting clarification %j", (change) => {
    expect(
      getAgentRunClarificationBindings({ ...awaiting, ...change }, transcript)
    ).toBeUndefined();
  });

  test("does not refine diagnostic capability bindings", () => {
    const capability = {
      kind: "diagnostic",
      resource: {
        kind: "issue",
        name: "projects/example/issues/1",
        project: "projects/example",
        label: "Issue",
      },
    } as const;
    expect(
      getAgentRunClarificationBindings(
        {
          ...awaiting,
          capability,
          runCheckpoint: {
            ...awaiting.runCheckpoint!,
            bindings: { ...awaiting.runCheckpoint!.bindings, capability },
          },
        },
        transcript
      )
    ).toBeUndefined();
  });
});
