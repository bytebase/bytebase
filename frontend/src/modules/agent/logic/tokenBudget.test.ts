import { describe, expect, test } from "vitest";
import {
  createRunBudget,
  getRunBudgetState,
  getRunTokenBudget,
  increaseRunBudgetLimit,
  mergeModelAttempts,
  normalizeRunBudget,
  normalizeRunCheckpoint,
  projectChatUsage,
  projectRunBudget,
  reserveRunAttempt,
} from "./tokenBudget";
import type { AgentChat, AgentModelAttempt } from "./types";

const attempt = (ordinal: number, totalTokens?: number): AgentModelAttempt => ({
  attemptId: `run:${ordinal}`,
  runId: "run",
  ordinal,
  outcome: "response",
  usage: {
    purpose: "agent",
    toolNames: [],
    ...(totalTokens === undefined ? {} : { totalTokens }),
  },
});

describe("run token budget", () => {
  test("derives normal, warning and exhausted at exact boundaries", () => {
    const budget = createRunBudget("run", 100);
    expect(getRunBudgetState({ ...budget, totalTokensUsed: 79 })).toBe(
      "normal"
    );
    expect(getRunBudgetState({ ...budget, totalTokensUsed: 80 })).toBe(
      "warning"
    );
    expect(getRunBudgetState({ ...budget, totalTokensUsed: 100 })).toBe(
      "exhausted"
    );
  });
  test("requires safe positive limits greater than limit and consumption", () => {
    const budget = { ...createRunBudget("run", 100), totalTokensUsed: 110 };
    for (const limit of [
      0,
      -1,
      100,
      110,
      110.5,
      Infinity,
      Number.MAX_SAFE_INTEGER + 1,
    ])
      expect(increaseRunBudgetLimit(budget, limit)).toBeUndefined();
    expect(increaseRunBudgetLimit(budget, 111)).toMatchObject({
      limit: 111,
      revision: 1,
    });
  });
  test("reserves monotonically and retains the twelve-attempt safeguard", () => {
    let budget = createRunBudget("run", 100);
    for (let ordinal = 1; ordinal <= 12; ordinal++) {
      const result = reserveRunAttempt(budget);
      expect(result.reservation).toEqual({
        kind: "allowed",
        attemptId: `run:${ordinal}`,
        ordinal,
      });
      budget = result.budget;
    }
    expect(reserveRunAttempt(budget).reservation).toEqual({
      kind: "attempt-limit",
    });
    expect(
      reserveRunAttempt({ ...budget, totalTokensUsed: 100 }).reservation
    ).toEqual({ kind: "token-limit" });
  });
  test("deduplicates immutable settlements and marks conflicting IDs incomplete", () => {
    expect(
      mergeModelAttempts([attempt(1, 10)], [attempt(1, 10), attempt(2, 20)])
    ).toHaveLength(2);
    const conflict = mergeModelAttempts([attempt(1, 10)], [attempt(1, 30)]);
    expect(
      projectChatUsage(
        { totalTokensUsed: 0, hasUnreportedUsage: false },
        conflict
      ).totalTokensUsed
    ).toBe(30);
    expect(
      projectChatUsage(
        { totalTokensUsed: 0, hasUnreportedUsage: false },
        conflict
      ).hasUnreportedUsage
    ).toBe(true);
  });
  test("never turns an unsettled reservation into known zero consumption", () => {
    const budget = { ...createRunBudget("run", 100), providerAttemptCount: 1 };
    expect(projectRunBudget(budget, [])).toMatchObject({
      totalTokensUsed: 0,
      hasUnreportedUsage: true,
    });
    expect(projectRunBudget(budget, [attempt(1, 0)])).toMatchObject({
      totalTokensUsed: 0,
      hasUnreportedUsage: false,
    });
  });
  test("does not present an explicitly pending response as missing usage", () => {
    const chat: AgentChat = {
      id: "chat",
      title: "Budget",
      createdTs: 0,
      updatedTs: 0,
      status: "running",
      archived: false,
      totalTokensUsed: 800,
      runId: "run",
      runBudget: { ...createRunBudget("run", 1000), providerAttemptCount: 2 },
      modelAttempts: [attempt(1, 800)],
    };
    expect(getRunTokenBudget(chat, "run:2")).toMatchObject({
      hasUnreportedUsage: false,
      budget: {
        totalTokensUsed: 800,
        hasUnreportedUsage: true,
        providerAttemptCount: 2,
      },
    });
    expect(getRunTokenBudget(chat)?.hasUnreportedUsage).toBe(true);
  });
  test.each([
    { records: [] },
    { records: [attempt(1)] },
    { records: [{ ...attempt(1, 100), outcome: "error" as const }] },
    { records: [{ ...attempt(1), outcome: "aborted" as const }] },
  ])(
    "keeps earlier unknown consumption visible while a later response is pending: $records",
    ({ records }) => {
      const chat: AgentChat = {
        id: "chat",
        title: "Budget",
        createdTs: 0,
        updatedTs: 0,
        status: "running",
        archived: false,
        totalTokensUsed: 0,
        runId: "run",
        runBudget: { ...createRunBudget("run", 1000), providerAttemptCount: 2 },
        modelAttempts: records,
      };
      expect(getRunTokenBudget(chat, "run:2")?.hasUnreportedUsage).toBe(true);
    }
  );
  test.each([undefined, "another-run:1", "run:2"])(
    "does not hide unknown usage for a noncurrent pending ID %j",
    (pendingAttemptId) => {
      const chat: AgentChat = {
        id: "chat",
        title: "Budget",
        createdTs: 0,
        updatedTs: 0,
        status: "running",
        archived: false,
        totalTokensUsed: 0,
        runId: "run",
        runBudget: { ...createRunBudget("run", 1000), providerAttemptCount: 1 },
        modelAttempts: [],
      };
      expect(
        getRunTokenBudget(chat, pendingAttemptId)?.hasUnreportedUsage
      ).toBe(true);
    }
  );
  test.each(["idle", "paused", "error"] as const)(
    "retains recovered unknown usage in %s even with a stale pending ID",
    (status) => {
      const chat: AgentChat = {
        id: "chat",
        title: "Budget",
        createdTs: 0,
        updatedTs: 0,
        status,
        archived: false,
        totalTokensUsed: 0,
        runId: "run",
        runBudget: { ...createRunBudget("run", 1000), providerAttemptCount: 1 },
        modelAttempts: [],
      };
      expect(getRunTokenBudget(chat, "run:1")?.hasUnreportedUsage).toBe(true);
    }
  );
  test("does not treat a settled response with absent usage as pending", () => {
    const chat: AgentChat = {
      id: "chat",
      title: "Budget",
      createdTs: 0,
      updatedTs: 0,
      status: "running",
      archived: false,
      totalTokensUsed: 0,
      runId: "run",
      runBudget: { ...createRunBudget("run", 1000), providerAttemptCount: 1 },
      modelAttempts: [attempt(1)],
    };
    expect(getRunTokenBudget(chat, "run:1")?.hasUnreportedUsage).toBe(true);
  });
  test("rejects invalid budget versions, unsafe counters, and missing finalization text", () => {
    const budget = createRunBudget("run", 100);
    expect(normalizeRunBudget({ ...budget, version: 2 })).toBeUndefined();
    expect(
      normalizeRunBudget({ ...budget, providerAttemptCount: 13 })
    ).toBeUndefined();
    expect(normalizeRunBudget({ ...budget, revision: 0.5 })).toBeUndefined();
    expect(
      normalizeRunCheckpoint({
        version: 1,
        runId: "run",
        suspension: "interrupted",
        rootUserMessageId: "root",
        throughMessageId: "root",
        bindings: {
          page: { path: "/", title: "Home" },
          references: [],
          runMode: "default",
        },
        maskingSections: [],
        phase: "finalizing",
      })
    ).toBeUndefined();
  });
  test("projects legacy baseline plus unique records without double-counting cached input", () => {
    const record = attempt(1, 30);
    record.usage!.cachedInputTokens = 20;
    expect(
      projectChatUsage({ totalTokensUsed: 40, hasUnreportedUsage: true }, [
        record,
      ])
    ).toEqual({ totalTokensUsed: 70, hasUnreportedUsage: true });
  });
  test("keeps zero known but absent and failed settlements unknown", () => {
    const baseline = { totalTokensUsed: 0, hasUnreportedUsage: false };
    expect(projectChatUsage(baseline, [attempt(1, 0)]).hasUnreportedUsage).toBe(
      false
    );
    expect(projectChatUsage(baseline, [attempt(1)]).hasUnreportedUsage).toBe(
      true
    );
    expect(
      projectChatUsage(baseline, [
        { ...attempt(1), outcome: "aborted", usage: undefined },
      ]).hasUnreportedUsage
    ).toBe(true);
  });
});
