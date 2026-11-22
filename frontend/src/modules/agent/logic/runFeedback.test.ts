import { describe, expect, test } from "vitest";
import { getAgentRunFeedback } from "./runFeedback";
import { createRunBudget } from "./tokenBudget";
import type { AgentChat, AgentMessage } from "./types";

const messages: AgentMessage[] = [
  {
    id: "user",
    chatId: "chat",
    role: "user",
    content: "Review database access",
    createdTs: 0,
    metadata: { runId: "run" },
  },
];

const makeChat = (
  overrides: Partial<AgentChat> = {},
  tokens = 0
): AgentChat => ({
  id: "chat",
  title: "Work",
  createdTs: 0,
  updatedTs: 0,
  status: "running",
  archived: false,
  totalTokensUsed: tokens,
  runId: "run",
  runBudget: { ...createRunBudget("run", 100), providerAttemptCount: 1 },
  runCheckpoint: {
    version: 1,
    runId: "run",
    suspension: "interrupted",
    rootUserMessageId: "user",
    throughMessageId: "user",
    bindings: {
      page: { path: "/projects/example", title: "Example" },
      context: "projects/example",
      references: [],
      runMode: "read-only",
    },
    maskingSections: [],
    phase: "working",
    startupComplete: true,
  },
  modelAttempts: [
    {
      attemptId: "run:1",
      runId: "run",
      ordinal: 1,
      outcome: "response",
      usage: { purpose: "agent", toolNames: [], totalTokens: tokens },
    },
  ],
  ...overrides,
});

const feedback = (
  overrides: Partial<Parameters<typeof getAgentRunFeedback>[0]> = {}
) =>
  getAgentRunFeedback({
    executionAvailable: true,
    aiConfigurationRequired: false,
    recoverySupported: true,
    remoteBusy: false,
    chat: makeChat(),
    messages,
    ...overrides,
  });

describe("getAgentRunFeedback", () => {
  test.each([
    { executionAvailable: false, expected: "execution-unavailable" },
    { storageBlocked: true, expected: "storage" },
    { aiConfigurationRequired: true, expected: "ai-configuration" },
    { continuationUnavailable: true, expected: "continuation-unavailable" },
  ] as const)(
    "prioritizes $expected over lower feedback",
    ({ expected, ...overrides }) => {
      const lowerPriority = {
        executionAvailable: true,
        storageBlocked: false,
        aiConfigurationRequired: false,
        continuationUnavailable: false,
      };
      if (expected === "execution-unavailable")
        lowerPriority.storageBlocked = true;
      if (expected === "execution-unavailable" || expected === "storage")
        lowerPriority.aiConfigurationRequired = true;
      if (expected !== "continuation-unavailable")
        lowerPriority.continuationUnavailable = true;
      const chat = makeChat(
        { status: "error", lastError: "Request failed" },
        100
      );
      chat.runCheckpoint = {
        ...chat.runCheckpoint!,
        suspension: "token-limit",
      };
      expect(
        feedback({
          ...lowerPriority,
          ...overrides,
          chat,
        })
      ).toBe(expected);
    }
  );

  test("shows storage feedback only while saving is blocked", () => {
    expect(feedback({ storageBlocked: false })).toBeUndefined();
    expect(feedback({ storageBlocked: true })).toBe("storage");
  });

  test("shows unavailable continuation for an error with no valid checkpoint", () => {
    expect(
      feedback({
        chat: makeChat({
          status: "error",
          lastError: "Request failed",
          runCheckpoint: undefined,
        }),
      })
    ).toBe("continuation-unavailable");
    expect(
      feedback({ chat: makeChat({ status: "error" }), messages: [] })
    ).toBe("continuation-unavailable");
  });

  test("keeps an ordinary recoverable network error as the primary feedback", () => {
    expect(
      feedback({
        chat: makeChat({ status: "error", lastError: "Network error" }, 100),
      })
    ).toBe("error");
  });

  test("keeps a failed request without an active run as ordinary error feedback", () => {
    expect(
      feedback({
        chat: makeChat({
          status: "error",
          runId: null,
          runCheckpoint: undefined,
          lastError: "Storage admission failed",
        }),
      })
    ).toBe("error");
    expect(
      feedback({
        continuationUnavailable: true,
        chat: makeChat({
          status: "error",
          runId: null,
          runCheckpoint: undefined,
          lastError: "Storage admission failed",
        }),
      })
    ).toBe("error");
  });

  test("keeps proven zero-progress startup failure actionable without claiming continuation is lost", () => {
    const chat = makeChat({
      status: "error",
      lastError: "Storage admission failed",
      runBudget: createRunBudget("run", 100),
      modelAttempts: [],
    });
    chat.runCheckpoint = { ...chat.runCheckpoint!, startupComplete: false };
    expect(feedback({ chat })).toBe("error");
    expect(feedback({ chat, continuationUnavailable: true })).toBe("error");
  });

  test.each([
    { recoverySupported: false },
    {
      chat: makeChat({
        status: "error",
        archived: true,
        runCheckpoint: undefined,
        lastError: "Request failed",
      }),
    },
  ])(
    "does not infer unavailable continuation outside recovery",
    (overrides) => {
      expect(
        feedback({
          chat: makeChat({
            status: "error",
            runCheckpoint: undefined,
            lastError: "Request failed",
          }),
          ...overrides,
        })
      ).toBe("error");
    }
  );

  test("prioritizes a token pause over a generic error", () => {
    const chat = makeChat(
      { status: "error", lastError: "Request failed" },
      100
    );
    chat.runCheckpoint = { ...chat.runCheckpoint!, suspension: "token-limit" };
    expect(feedback({ chat })).toBe("token-budget");
  });

  test.each(["running", "awaiting_user"] as const)(
    "shows threshold feedback but not missing usage alone while %s",
    (status) => {
      expect(feedback({ chat: makeChat({ status }, 80) })).toBe("token-budget");
      expect(feedback({ chat: makeChat({ status }, 100) })).toBe(
        "token-budget"
      );
      expect(feedback({ chat: makeChat({ status, modelAttempts: [] }) })).toBe(
        undefined
      );
      for (const tokens of [24, 79, 80, 100]) {
        const chat = makeChat({ status }, tokens);
        chat.runBudget = { ...chat.runBudget!, providerAttemptCount: 2 };
        expect(feedback({ chat })).toBe(
          tokens >= 80 ? "token-budget" : undefined
        );
      }
    }
  );

  test("keeps actual token-pause recovery after the limit is increased", () => {
    const chat = makeChat({ status: "paused" }, 24);
    chat.runCheckpoint = { ...chat.runCheckpoint!, suspension: "token-limit" };
    expect(feedback({ chat })).toBe("token-budget");
  });

  test("does not mistake an in-flight provider attempt for unreported usage", () => {
    expect(
      feedback({
        chat: makeChat({ modelAttempts: [] }),
        pendingAttemptId: "run:1",
      })
    ).toBeUndefined();
  });

  test("shows no current warning for a completed run or normal active usage", () => {
    expect(feedback({ chat: null })).toBeUndefined();
    expect(feedback()).toBeUndefined();
    expect(feedback({ chat: makeChat({}, 79) })).toBeUndefined();
    expect(
      feedback({ chat: makeChat({ status: "idle" }, 100) })
    ).toBeUndefined();
    expect(
      feedback({ chat: makeChat({ status: "idle", modelAttempts: [] }) })
    ).toBeUndefined();
  });

  test("suppresses continuation and token feedback while another tab owns the run", () => {
    expect(
      feedback({ remoteBusy: true, continuationUnavailable: true })
    ).toBeUndefined();
    expect(
      feedback({
        remoteBusy: true,
        chat: makeChat({ status: "error", runCheckpoint: undefined }),
      })
    ).toBeUndefined();
    expect(
      feedback({
        remoteBusy: true,
        chat: makeChat({ status: "error", lastError: "Network error" }),
      })
    ).toBeUndefined();
    const chat = makeChat({ status: "paused" }, 100);
    chat.runCheckpoint = { ...chat.runCheckpoint!, suspension: "token-limit" };
    expect(feedback({ remoteBusy: true, chat })).toBeUndefined();
    expect(
      feedback({ remoteBusy: true, chat: makeChat({}, 80) })
    ).toBeUndefined();
    expect(feedback({ remoteBusy: true, storageBlocked: true })).toBe(
      "storage"
    );
    expect(feedback({ remoteBusy: true, aiConfigurationRequired: true })).toBe(
      "ai-configuration"
    );
  });
});
