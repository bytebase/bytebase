import { afterEach, describe, expect, test, vi } from "vitest";
import {
  applyAgentExpressionDraft,
  canApplyAgentExpressionDraft,
  extractAgentExpressionDraft,
  registerAgentExpressionTarget,
  subscribeToAgentExpressionTargets,
} from "./expressionDraft";

describe("agent expression drafts", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("offers only one bounded CEL block for an editor draft", () => {
    expect(
      extractAgentExpressionDraft(
        'Use both filters:\n```cel\nresource.environment == "prod"\n```'
      )
    ).toBe('resource.environment == "prod"');
    expect(
      extractAgentExpressionDraft(
        '```cel\nresource.environment == "prod"\n```\n```cel\ntrue\n```'
      )
    ).toBeUndefined();
    expect(
      extractAgentExpressionDraft("```sql\nSELECT 1\n```")
    ).toBeUndefined();
  });

  test("applies a draft only to the editor registered by that chat", async () => {
    const apply = vi.fn(async () => true);
    const unregister = registerAgentExpressionTarget("chat-1", apply);

    await expect(
      applyAgentExpressionDraft(
        "chat-1",
        'resource.environment == "prod"',
        "selector"
      )
    ).resolves.toBe(true);
    await expect(
      applyAgentExpressionDraft("chat-2", "true", "raw")
    ).resolves.toBe(false);
    expect(apply).toHaveBeenCalledOnce();
    expect(apply).toHaveBeenCalledWith(
      'resource.environment == "prod"',
      "selector"
    );

    unregister();
    await expect(
      applyAgentExpressionDraft("chat-1", "true", "raw")
    ).resolves.toBe(false);
  });

  test("does not offer raw CEL for editors that disable raw expressions", () => {
    const unregister = registerAgentExpressionTarget(
      "selector-chat",
      async () => true,
      { allowRaw: false }
    );

    expect(canApplyAgentExpressionDraft("selector-chat", "selector")).toBe(
      true
    );
    expect(canApplyAgentExpressionDraft("selector-chat", "raw")).toBe(false);

    unregister();
  });

  test("notifies Agent chat when a bound editor becomes unavailable", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeToAgentExpressionTargets(listener);
    const unregister = registerAgentExpressionTarget(
      "chat-1",
      async () => true
    );
    unregister();

    expect(listener).toHaveBeenCalledTimes(2);
    unsubscribe();
  });
});
