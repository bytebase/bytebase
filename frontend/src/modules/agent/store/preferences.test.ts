import { describe, expect, test } from "vitest";
import {
  DEFAULT_AGENT_PREFERENCES,
  getAgentPreferencesKey,
  normalizeAgentPreferences,
} from "./preferences";

describe("getAgentPreferencesKey", () => {
  test.each([
    ["", "member@example.com"],
    ["workspaces/", "member@example.com"],
    ["projects/example", "member@example.com"],
    ["workspaces/example", ""],
  ])("requires both a Workspace and member: %s, %s", (workspace, email) => {
    expect(getAgentPreferencesKey(workspace, email)).toBeNull();
  });

  test("uses a versioned key independent from chat storage", () => {
    expect(
      getAgentPreferencesKey("workspaces/example", "member@example.com")
    ).toBe("bb-agent-preferences-v1.7:example.member@example.com");
  });
});

describe("normalizeAgentPreferences", () => {
  test("defaults to 100000 tokens per run without overwriting a saved limit", () => {
    expect(DEFAULT_AGENT_PREFERENCES.tokenLimitPerRun).toBe(100_000);
    expect(normalizeAgentPreferences(undefined).tokenLimitPerRun).toBe(100_000);
    expect(
      normalizeAgentPreferences({ tokenLimitPerRun: 60_000 }).tokenLimitPerRun
    ).toBe(60_000);
  });

  test("keeps supported values and discards unknown fields", () => {
    expect(
      normalizeAgentPreferences({
        sendWithEnter: false,
        tokenLimitPerRun: Number.MAX_SAFE_INTEGER,
        openingBehavior: "continue",
        rows: 100,
      })
    ).toEqual({
      sendWithEnter: false,
      tokenLimitPerRun: Number.MAX_SAFE_INTEGER,
      openingBehavior: "continue",
    });
  });

  test.each([undefined, null, [], "invalid", 2, false])(
    "normalizes %s to independent defaults",
    (value) => {
      const normalized = normalizeAgentPreferences(value);
      expect(normalized).toEqual(DEFAULT_AGENT_PREFERENCES);
      expect(normalized).not.toBe(DEFAULT_AGENT_PREFERENCES);
    }
  );

  test.each([Number.NaN, Number.POSITIVE_INFINITY, Number.NEGATIVE_INFINITY])(
    "rejects non-finite token limit %s",
    (tokenLimitPerRun) => {
      expect(normalizeAgentPreferences({ tokenLimitPerRun })).toEqual(
        DEFAULT_AGENT_PREFERENCES
      );
    }
  );
});
