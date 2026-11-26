import { describe, expect, test } from "vitest";
import { chatNeedsAttention } from "./chatAttention";
import type {
  AgentChat,
  AgentMessage,
  AgentPendingAsk,
  AgentRunBudget,
  AgentRunCheckpoint,
} from "./types";

const chat: AgentChat = {
  id: "chat-1",
  title: "Example",
  createdTs: 0,
  updatedTs: 0,
  status: "idle",
  archived: false,
  totalTokensUsed: 1000,
};
const awaiting: AgentChat = { ...chat, status: "awaiting_user" };
const ask: AgentPendingAsk = {
  toolCallId: "ask-1",
  kind: "input",
  prompt: "Which database?",
};

const runBudget: AgentRunBudget = {
  version: 1,
  runId: "run-1",
  limit: 1000,
  totalTokensUsed: 1000,
  hasUnreportedUsage: false,
  providerAttemptCount: 1,
  revision: 1,
};
const runCheckpoint: AgentRunCheckpoint = {
  version: 1,
  runId: "run-1",
  suspension: "token-limit",
  rootUserMessageId: "user-1",
  throughMessageId: "user-1",
  bindings: {
    page: { path: "/projects/example", title: "Example" },
    context: "projects/example",
    references: [],
    runMode: "read-only",
  },
  maskingSections: [],
  phase: "working",
  startupComplete: true,
};
const tokenLimited: AgentChat = {
  ...chat,
  status: "paused",
  runId: "run-1",
  runBudget,
  runCheckpoint,
};
const messages: AgentMessage[] = [
  {
    id: "user-1",
    chatId: chat.id,
    role: "user",
    content: "Review database access",
    createdTs: 0,
    metadata: { runId: "run-1" },
  },
];

describe("chatNeedsAttention", () => {
  test.each(["input", "confirm", "choose"] as const)(
    "marks an awaiting chat with a pending %s ask",
    (kind) => {
      expect(chatNeedsAttention(awaiting, [], { ...ask, kind })).toBe(true);
    }
  );

  test("keeps an expired permission request visible until dismissal", () => {
    const expiredAsk: AgentPendingAsk = {
      ...ask,
      kind: "confirm",
      apiPermission: {
        actionId: "action-1",
        operationId: "UpdateDatabase",
        expiresAt: 1,
      },
    };
    expect(chatNeedsAttention(awaiting, [], expiredAsk)).toBe(true);
    expect(chatNeedsAttention(awaiting, [], undefined)).toBe(false);
  });

  test("ignores waiting status without an ask or a stale ask after the chat is no longer waiting", () => {
    expect(chatNeedsAttention(awaiting, [], null)).toBe(false);
    expect(chatNeedsAttention(chat, [], ask)).toBe(false);
  });

  test("marks a valid token-limit continuation", () => {
    expect(chatNeedsAttention(tokenLimited, messages, undefined)).toBe(true);
  });

  test("keeps a token-limit continuation visible when status and budget have changed", () => {
    expect(
      chatNeedsAttention(
        { ...tokenLimited, status: "running" },
        messages,
        undefined
      )
    ).toBe(true);
    expect(
      chatNeedsAttention(
        { ...tokenLimited, runBudget: { ...runBudget, limit: 2000 } },
        messages,
        undefined
      )
    ).toBe(true);
  });

  test.each([
    { runId: "other-run" },
    { runBudget: undefined },
    { runCheckpoint: undefined },
    { runCheckpoint: { ...runCheckpoint, throughMessageId: "missing" } },
    { runCheckpoint: { ...runCheckpoint, suspension: "interrupted" as const } },
  ])("ignores invalid or non-token-limit run state %j", (change) => {
    expect(
      chatNeedsAttention({ ...tokenLimited, ...change }, messages, undefined)
    ).toBe(false);
  });

  test("ignores archived or missing chats even with a pending ask or checkpoint", () => {
    expect(chatNeedsAttention({ ...awaiting, archived: true }, [], ask)).toBe(
      false
    );
    expect(
      chatNeedsAttention(
        { ...tokenLimited, archived: true },
        messages,
        undefined
      )
    ).toBe(false);
    expect(chatNeedsAttention(undefined, [], ask)).toBe(false);
  });

  test.each([
    { status: "paused", interruptionReason: "user" },
    { status: "error", lastError: "Try again later" },
    { status: "idle" },
  ] satisfies Partial<AgentChat>[])(
    "does not infer attention from transcript prose or status %j",
    (change) => {
      const suggestion: AgentMessage = {
        id: "assistant-1",
        chatId: chat.id,
        role: "assistant",
        content: "Would you like me to continue?",
        createdTs: 0,
      };
      expect(
        chatNeedsAttention({ ...chat, ...change }, [suggestion], undefined)
      ).toBe(false);
    }
  );
});
