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

    const result = await generateFollowUpQuestion(
      "How do I connect a database?",
      "Transfer an existing database, or connect a new instance.",
      { action, existingQuestions: [] },
      new AbortController().signal
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
    vi.mocked(aiServiceClientConnect.chat).mockRejectedValue(
      new Error("Service unavailable")
    );
    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        new AbortController().signal
      )
    ).toEqual({ actions: [], totalTokensUsed: 0 });

    const controller = new AbortController();
    controller.abort();
    expect(
      await generateFollowUpQuestion(
        "What next?",
        "Review the plan.",
        { action, existingQuestions: [] },
        controller.signal
      )
    ).toEqual({ actions: [], totalTokensUsed: 0 });
    expect(aiServiceClientConnect.chat).toHaveBeenCalledOnce();
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
