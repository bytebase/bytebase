import { describe, expect, test } from "vitest";
import { getAgentErrorSummary } from "./errorDisplay";

describe("getAgentErrorSummary", () => {
  test("uses the provider's actionable message instead of its transport wrapper", () => {
    expect(
      getAgentErrorSummary(
        'OpenAI API error: {"error":{"message":"API key not valid. Please pass a valid API key."}}'
      )
    ).toBe("API key not valid. Please pass a valid API key.");
  });

  test("falls back to the first line and keeps long raw errors bounded", () => {
    expect(getAgentErrorSummary("Error: Request failed\nprovider trace")).toBe(
      "Request failed"
    );
    expect(getAgentErrorSummary(`Error: ${"x".repeat(300)}`)).toBe(
      `${"x".repeat(239)}…`
    );
    expect(getAgentErrorSummary(" \n ")).toBeUndefined();
  });
});
