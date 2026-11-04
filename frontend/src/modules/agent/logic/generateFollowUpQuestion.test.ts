import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { aiServiceClientConnect } from "@/api";
import {
  AIChatResponse_StopReason,
  AIChatResponseSchema,
  AIChatToolCallSchema,
} from "@/types/proto-es/v1/ai_service_pb";
import { generateFollowUpQuestion } from "./generateFollowUpQuestion";

vi.mock("@/api", () => ({
  aiServiceClientConnect: { chat: vi.fn() },
}));

const action = {
  kind: "transfer-project-databases" as const,
  project: "projects/sample",
};

const response = (content: string) =>
  create(AIChatResponseSchema, {
    content,
    toolCalls: [],
    stopReason: AIChatResponse_StopReason.TURN_COMPLETED,
    usage: { totalTokens: 7 },
  });

beforeEach(() => {
  vi.mocked(aiServiceClientConnect.chat).mockReset();
});

describe("generateFollowUpQuestion", () => {
  test("returns distinct validated questions from a single read-only AI call", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue(
      response(
        JSON.stringify([
          "What information do I need to connect a new instance?",
          "Which database can I transfer into this project?",
          "What information do I need to connect a new instance?",
          "How can I verify the transfer afterward?",
        ])
      )
    );

    const onUsage = vi.fn();
    const result = await generateFollowUpQuestion(
      "How do I connect a database?",
      "Transfer an existing database, or connect a new instance.",
      { action, existingQuestions: [] },
      new AbortController().signal,
      onUsage
    );

    expect(result).toEqual({
      actions: [
        {
          kind: "continue",
          prompt: "What information do I need to connect a new instance?",
        },
        {
          kind: "continue",
          prompt: "Which database can I transfer into this project?",
        },
        {
          kind: "continue",
          prompt: "How can I verify the transfer afterward?",
        },
      ],
      totalTokensUsed: 7,
    });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledOnce();
    expect(onUsage).toHaveBeenCalledExactlyOnceWith({
      purpose: "follow-up",
      toolNames: [],
      totalTokens: 7,
    });
    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0][0];
    expect(request.toolDefinitions).toEqual([]);
    expect(request.messages?.[1]?.content).toContain(
      "How do I connect a database?"
    );
    expect(request.messages?.[1]?.content).toContain(
      "Transfer an existing database"
    );
  });

  test("tops up existing questions without repeating them", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue(
      response(
        JSON.stringify([
          "What information do I need to connect a new instance?",
          "How can I verify the transfer?",
          "What changed after the transfer?",
        ])
      )
    );

    const result = await generateFollowUpQuestion(
      "How do I connect a database?",
      "Transfer an existing database.",
      {
        existingQuestions: [
          "What information do I need to connect a new instance?",
        ],
      },
      new AbortController().signal
    );

    expect(result.actions).toEqual([
      { kind: "continue", prompt: "How can I verify the transfer?" },
      { kind: "continue", prompt: "What changed after the transfer?" },
    ]);
    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0][0];
    expect(request.messages?.[1]?.content).toContain(
      "What information do I need to connect a new instance?"
    );
  });

  test.each([
    "NONE",
    "not JSON",
    JSON.stringify(["What about @other?"]),
    JSON.stringify(["What should I do?\nWhat else?"]),
    JSON.stringify(["This is not a question."]),
    JSON.stringify([`${"x".repeat(141)}?`]),
    JSON.stringify({ questions: ["What next?"] }),
  ])("omits an invalid generated answer: %s", async (content) => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue(response(content));

    const result = await generateFollowUpQuestion(
      "What next?",
      "Review the plan.",
      { action, existingQuestions: [] },
      new AbortController().signal
    );

    expect(result).toEqual({ actions: [], totalTokensUsed: 7 });
  });

  test("ignores incomplete or tool-calling responses", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      ...response('["What next?"]'),
      stopReason: AIChatResponse_StopReason.TOKEN_LIMIT,
    });
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValueOnce({
      ...response('["What next?"]'),
      toolCalls: [
        create(AIChatToolCallSchema, {
          id: "call-1",
          name: "call_api",
          arguments: "{}",
        }),
      ],
    });

    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        new AbortController().signal
      )
    ).toEqual({ actions: [], totalTokensUsed: 7 });
    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        new AbortController().signal
      )
    ).toEqual({ actions: [], totalTokensUsed: 7 });
  });

  test("keeps the original answer usable when generation fails or is stopped", async () => {
    const onUsage = vi.fn();
    vi.mocked(aiServiceClientConnect.chat).mockRejectedValue(
      new Error("Service unavailable")
    );
    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        new AbortController().signal,
        onUsage
      )
    ).toEqual({ actions: [], totalTokensUsed: 0 });

    const controller = new AbortController();
    controller.abort();
    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        controller.signal,
        onUsage
      )
    ).toEqual({ actions: [], totalTokensUsed: 0 });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledOnce();
    expect(onUsage).not.toHaveBeenCalled();
  });

  test.each([
    { usage: undefined, expected: {} },
    {
      usage: {
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        cachedInputTokens: 0,
      },
      expected: {
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        cachedInputTokens: 0,
      },
    },
  ])(
    "reports usage for invalid answers, preserving unknown and zero: $usage",
    async ({ usage, expected }) => {
      vi.mocked(aiServiceClientConnect.chat).mockResolvedValue({
        ...response("not JSON"),
        usage,
      } as never);
      const onUsage = vi.fn();

      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { existingQuestions: [] },
        new AbortController().signal,
        onUsage
      );

      expect(onUsage).toHaveBeenCalledExactlyOnceWith({
        purpose: "follow-up",
        toolNames: [],
        ...expected,
      });
    }
  );

  test("reports a response returned after cancellation without publishing questions", async () => {
    let resolveChat!: (
      result: Awaited<ReturnType<typeof aiServiceClientConnect.chat>>
    ) => void;
    vi.mocked(aiServiceClientConnect.chat).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveChat = resolve;
        })
    );
    const controller = new AbortController();
    const onUsage = vi.fn();
    const generation = generateFollowUpQuestion(
      "What next?",
      "Review the plan.",
      { existingQuestions: [] },
      controller.signal,
      onUsage
    );
    controller.abort();
    resolveChat({
      ...response('["How can I review the plan?"]'),
      usage: {
        totalTokens: 7,
        inputTokens: 5,
        outputTokens: 2,
        cachedInputTokens: 1,
      },
    } as never);

    expect(await generation).toEqual({ actions: [], totalTokensUsed: 7 });
    expect(onUsage).toHaveBeenCalledExactlyOnceWith({
      purpose: "follow-up",
      toolNames: [],
      totalTokens: 7,
      inputTokens: 5,
      outputTokens: 2,
      cachedInputTokens: 1,
    });
  });

  test("bounds the context sent for a follow-up", async () => {
    vi.mocked(aiServiceClientConnect.chat).mockResolvedValue(response("[]"));

    await generateFollowUpQuestion(
      "r".repeat(20_000),
      "a".repeat(100_000),
      { action, existingQuestions: [] },
      new AbortController().signal
    );

    const request = vi.mocked(aiServiceClientConnect.chat).mock.calls[0][0];
    const content = request.messages?.[1]?.content ?? "";
    expect(content).not.toBe("");
    expect(content.length).toBeLessThan(7_000);
  });
});
