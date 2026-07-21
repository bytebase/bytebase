import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, test } from "vitest";
import { isAgentAIConfigurationError } from "./aiConfiguration";

describe("isAgentAIConfigurationError", () => {
  test("returns true for the AI-not-enabled failed precondition", () => {
    const error = new ConnectError(
      "AI is not enabled",
      Code.FailedPrecondition
    );

    expect(error.rawMessage).toBe("AI is not enabled");
    expect(isAgentAIConfigurationError(error)).toBe(true);
  });

  test("returns false for other errors", () => {
    expect(
      isAgentAIConfigurationError(
        new ConnectError("AI is not enabled", Code.PermissionDenied)
      )
    ).toBe(false);
    expect(
      isAgentAIConfigurationError(
        new ConnectError("Other failure", Code.FailedPrecondition)
      )
    ).toBe(false);
    expect(isAgentAIConfigurationError(new Error("AI is not enabled"))).toBe(
      false
    );
  });
});
