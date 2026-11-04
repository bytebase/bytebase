import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { AIChatResponseSchema } from "@/types/proto-es/v1/ai_service_pb";
import { getModelCallUsage, normalizeModelCallUsage } from "./tokenUsage";

describe("getModelCallUsage", () => {
  test("retains provider counts without adding cached input to the total", () => {
    const response = {
      ...create(AIChatResponseSchema),
      toolCalls: [{ id: "call-1", name: "done", arguments: "{}" }],
      usage: {
        totalTokens: 12,
        inputTokens: 8,
        outputTokens: 4,
        cachedInputTokens: 5,
      },
    } as never;

    expect(getModelCallUsage("agent", response)).toEqual({
      purpose: "agent",
      toolNames: ["done"],
      totalTokens: 12,
      inputTokens: 8,
      outputTokens: 4,
      cachedInputTokens: 5,
    });
  });

  test("does not treat absent usage as a reported zero", () => {
    expect(
      getModelCallUsage("follow-up", create(AIChatResponseSchema))
    ).toEqual({ purpose: "follow-up", toolNames: [] });
    expect(
      getModelCallUsage(
        "follow-up",
        create(AIChatResponseSchema, { usage: { totalTokens: 0 } })
      )
    ).toEqual({ purpose: "follow-up", toolNames: [], totalTokens: 0 });
  });
});

describe("normalizeModelCallUsage", () => {
  test("preserves zero and drops unrecognized persisted fields", () => {
    expect(
      normalizeModelCallUsage({
        purpose: "agent",
        toolNames: ["get_page_state"],
        totalTokens: 0,
        inputTokens: 0,
        outputTokens: 0,
        cachedInputTokens: 0,
        unknown: "ignored",
      })
    ).toEqual({
      purpose: "agent",
      toolNames: ["get_page_state"],
      totalTokens: 0,
      inputTokens: 0,
      outputTokens: 0,
      cachedInputTokens: 0,
    });
  });

  test.each([
    null,
    [],
    {},
    { purpose: "unknown", toolNames: [] },
    { purpose: "agent", toolNames: "done" },
  ])("rejects malformed persisted usage: %j", (raw) => {
    expect(normalizeModelCallUsage(raw)).toBeUndefined();
  });

  test("drops malformed persisted tool names without losing valid metrics", () => {
    expect(
      normalizeModelCallUsage({
        purpose: "agent",
        toolNames: ["done", 7, null],
        totalTokens: 12,
      })
    ).toEqual({ purpose: "agent", toolNames: ["done"], totalTokens: 12 });
  });

  test("does not manufacture a total from partial or invalid metrics", () => {
    expect(
      normalizeModelCallUsage({
        purpose: "follow-up",
        toolNames: [],
        totalTokens: -1,
        inputTokens: 9,
        outputTokens: Number.NaN,
        cachedInputTokens: "3",
      })
    ).toEqual({ purpose: "follow-up", toolNames: [], inputTokens: 9 });
  });

  test.each([Number.POSITIVE_INFINITY, 1.5, -1, null])(
    "does not report an invalid token count as measured usage: %j",
    (totalTokens) => {
      expect(
        normalizeModelCallUsage({
          purpose: "agent",
          toolNames: [],
          totalTokens,
        })
      ).toEqual({ purpose: "agent", toolNames: [] });
    }
  );
});
