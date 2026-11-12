import { normalizeAgentDiagnosticCapability } from "./diagnosticCapability";
import { normalizeAgentResourceReferences } from "./resourceMention";
import { normalizeModelCallUsage } from "./tokenUsage";
import type {
  AgentAttemptReservation,
  AgentChat,
  AgentModelAttempt,
  AgentRunBindings,
  AgentRunBudget,
  AgentRunCheckpoint,
} from "./types";

const record = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);
const count = (value: unknown): value is number =>
  typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
export const isValidRunTokenLimit = (value: unknown): value is number =>
  count(value) && value > 0;

export function createRunBudget(runId: string, limit: number): AgentRunBudget {
  if (!runId || !isValidRunTokenLimit(limit))
    throw new Error("Invalid run budget");
  return {
    version: 1,
    runId,
    limit,
    totalTokensUsed: 0,
    hasUnreportedUsage: false,
    providerAttemptCount: 0,
    revision: 0,
  };
}

export function normalizeRunBudget(value: unknown): AgentRunBudget | undefined {
  if (
    !record(value) ||
    value.version !== 1 ||
    typeof value.runId !== "string" ||
    !value.runId ||
    !isValidRunTokenLimit(value.limit) ||
    !count(value.totalTokensUsed) ||
    !count(value.providerAttemptCount) ||
    value.providerAttemptCount > 12 ||
    !count(value.revision) ||
    typeof value.hasUnreportedUsage !== "boolean"
  )
    return undefined;
  return {
    version: 1,
    runId: value.runId,
    limit: value.limit,
    totalTokensUsed: value.totalTokensUsed,
    hasUnreportedUsage: value.hasUnreportedUsage,
    providerAttemptCount: value.providerAttemptCount,
    revision: value.revision,
  };
}

export const getRunBudgetState = (
  budget: AgentRunBudget
): "normal" | "warning" | "exhausted" =>
  budget.totalTokensUsed >= budget.limit
    ? "exhausted"
    : budget.totalTokensUsed >= budget.limit * 0.8
      ? "warning"
      : "normal";

export function increaseRunBudgetLimit(
  budget: AgentRunBudget,
  limit: number
): AgentRunBudget | undefined {
  return isValidRunTokenLimit(limit) &&
    limit > budget.limit &&
    limit > budget.totalTokensUsed
    ? { ...budget, limit, revision: budget.revision + 1 }
    : undefined;
}

export function reserveRunAttempt(budget: AgentRunBudget): {
  budget: AgentRunBudget;
  reservation: AgentAttemptReservation;
} {
  if (getRunBudgetState(budget) === "exhausted")
    return { budget, reservation: { kind: "token-limit" } };
  if (budget.providerAttemptCount >= 12)
    return { budget, reservation: { kind: "attempt-limit" } };
  const ordinal = budget.providerAttemptCount + 1;
  return {
    budget: {
      ...budget,
      providerAttemptCount: ordinal,
      revision: budget.revision + 1,
    },
    reservation: {
      kind: "allowed",
      attemptId: `${budget.runId}:${ordinal}`,
      ordinal,
    },
  };
}

export function normalizeModelAttempt(
  value: unknown
): AgentModelAttempt | undefined {
  if (
    !record(value) ||
    typeof value.runId !== "string" ||
    !value.runId ||
    !count(value.ordinal) ||
    value.ordinal < 1 ||
    value.ordinal > 12 ||
    value.attemptId !== `${value.runId}:${value.ordinal}` ||
    !["response", "error", "aborted"].includes(String(value.outcome))
  )
    return undefined;
  return {
    attemptId: value.attemptId as string,
    runId: value.runId,
    ordinal: value.ordinal,
    outcome: value.outcome as AgentModelAttempt["outcome"],
    ...(value.usage === undefined
      ? {}
      : { usage: normalizeModelCallUsage(value.usage) }),
  };
}

export function mergeModelAttempts(
  ...ledgers: (readonly AgentModelAttempt[] | undefined)[]
): AgentModelAttempt[] {
  const records = new Map<string, AgentModelAttempt>();
  for (const ledger of ledgers)
    for (const raw of ledger ?? []) {
      const attempt = normalizeModelAttempt(raw);
      if (!attempt) continue;
      const previous = records.get(attempt.attemptId);
      if (!previous) records.set(attempt.attemptId, attempt);
      else if (JSON.stringify(previous) !== JSON.stringify(attempt)) {
        // Conflicting terminal records retain a conservative known lower bound.
        const usage = [previous.usage, attempt.usage].sort(
          (a, b) =>
            (b?.totalTokens ?? -1) - (a?.totalTokens ?? -1) ||
            JSON.stringify(a ?? {}).localeCompare(JSON.stringify(b ?? {}))
        )[0];
        records.set(attempt.attemptId, {
          attemptId: attempt.attemptId,
          runId: attempt.runId,
          ordinal: attempt.ordinal,
          outcome: "error",
          ...(usage ? { usage } : {}),
        });
      }
    }
  return [...records.values()].sort((a, b) =>
    a.attemptId.localeCompare(b.attemptId)
  );
}

export function projectChatUsage(
  baseline: { totalTokensUsed: number; hasUnreportedUsage: boolean },
  attempts: readonly AgentModelAttempt[]
): { totalTokensUsed: number; hasUnreportedUsage: boolean } {
  let totalTokensUsed = baseline.totalTokensUsed;
  let hasUnreportedUsage = baseline.hasUnreportedUsage;
  for (const attempt of attempts) {
    totalTokensUsed += attempt.usage?.totalTokens ?? 0;
    hasUnreportedUsage ||=
      attempt.outcome !== "response" ||
      attempt.usage?.totalTokens === undefined;
  }
  return { totalTokensUsed, hasUnreportedUsage };
}

export function projectRunBudget(
  budget: AgentRunBudget,
  attempts: readonly AgentModelAttempt[]
): AgentRunBudget {
  const matching = attempts.filter((attempt) => attempt.runId === budget.runId);
  const usage = projectChatUsage(
    { totalTokensUsed: 0, hasUnreportedUsage: false },
    matching
  );
  return {
    ...budget,
    ...usage,
    hasUnreportedUsage:
      usage.hasUnreportedUsage || matching.length < budget.providerAttemptCount,
  };
}

export function getRunTokenBudget(chat: AgentChat, pendingAttemptId?: string) {
  if (!chat.runBudget) return undefined;
  const attempts = chat.modelAttempts ?? [];
  const budget = projectRunBudget(chat.runBudget, attempts);
  const pending =
    chat.status === "running" &&
    chat.runId === budget.runId &&
    budget.providerAttemptCount > 0 &&
    pendingAttemptId === `${budget.runId}:${budget.providerAttemptCount}` &&
    !attempts.some((attempt) => attempt.attemptId === pendingAttemptId);
  // Only the explicitly pending local attempt is awaiting usage, not missing it.
  const presentation = pending
    ? projectRunBudget(
        { ...budget, providerAttemptCount: budget.providerAttemptCount - 1 },
        attempts
      )
    : budget;
  return {
    budget,
    state: getRunBudgetState(budget),
    hasUnreportedUsage: presentation.hasUnreportedUsage,
    failedAttemptCount: attempts.filter(
      (attempt) =>
        attempt.runId === budget.runId && attempt.outcome !== "response"
    ).length,
  };
}

export function normalizeRunBindings(
  value: unknown
): AgentRunBindings | undefined {
  if (
    !record(value) ||
    !record(value.page) ||
    typeof value.page.path !== "string" ||
    typeof value.page.title !== "string" ||
    !Array.isArray(value.references) ||
    (value.runMode !== "default" && value.runMode !== "read-only") ||
    (value.context !== undefined && typeof value.context !== "string") ||
    (value.instanceName !== undefined && typeof value.instanceName !== "string")
  )
    return undefined;
  const references = normalizeAgentResourceReferences(value.references);
  if (references.length !== value.references.length) return undefined;
  const capability = normalizeAgentDiagnosticCapability(value.capability);
  if (value.capability !== undefined && !capability) return undefined;
  let maskingPage: AgentRunBindings["maskingPage"];
  if (value.maskingPage !== undefined) {
    if (!record(value.maskingPage)) return undefined;
    const page = value.maskingPage;
    if (
      page.kind === "global-masking" ||
      page.kind === "semantic-types" ||
      page.kind === "data-classification"
    )
      maskingPage = { kind: page.kind };
    else if (
      page.kind === "masking-exemption" &&
      typeof page.project === "string" &&
      page.project
    )
      maskingPage = { kind: page.kind, project: page.project };
    else if (
      page.kind === "database-catalog" &&
      typeof page.project === "string" &&
      page.project &&
      Array.isArray(page.names) &&
      page.names.length > 0 &&
      page.names.every((name) => typeof name === "string" && name)
    )
      maskingPage = {
        kind: page.kind,
        project: page.project,
        names: page.names as string[],
      };
    else return undefined;
  }
  return {
    page: { path: value.page.path, title: value.page.title },
    references,
    runMode: value.runMode,
    ...(value.context === undefined
      ? {}
      : { context: value.context as string }),
    ...(value.instanceName === undefined
      ? {}
      : { instanceName: value.instanceName as string }),
    ...(capability ? { capability } : {}),
    ...(maskingPage ? { maskingPage } : {}),
  };
}

export function normalizeRunCheckpoint(
  value: unknown
): AgentRunCheckpoint | undefined {
  if (
    !record(value) ||
    value.version !== 1 ||
    typeof value.runId !== "string" ||
    !value.runId ||
    !["token-limit", "awaiting-user", "interrupted"].includes(
      String(value.suspension)
    ) ||
    typeof value.rootUserMessageId !== "string" ||
    !value.rootUserMessageId ||
    typeof value.throughMessageId !== "string" ||
    !value.throughMessageId ||
    !["working", "finalizing"].includes(String(value.phase)) ||
    !Array.isArray(value.maskingSections) ||
    !value.maskingSections.every((section) =>
      ["overview", "rule", "exemption", "semantic-type"].includes(section)
    ) ||
    (value.finalizationText !== undefined &&
      typeof value.finalizationText !== "string") ||
    (value.phase === "finalizing" && typeof value.finalizationText !== "string")
  )
    return undefined;
  const bindings = normalizeRunBindings(value.bindings);
  if (
    value.startupComplete !== undefined &&
    typeof value.startupComplete !== "boolean"
  )
    return undefined;
  return bindings
    ? {
        version: 1,
        runId: value.runId,
        suspension: value.suspension as AgentRunCheckpoint["suspension"],
        rootUserMessageId: value.rootUserMessageId,
        throughMessageId: value.throughMessageId,
        bindings,
        maskingSections: [
          ...new Set(value.maskingSections),
        ] as AgentRunCheckpoint["maskingSections"],
        phase: value.phase as AgentRunCheckpoint["phase"],
        ...(value.startupComplete === undefined
          ? {}
          : { startupComplete: value.startupComplete }),
        ...(value.finalizationText === undefined
          ? {}
          : { finalizationText: value.finalizationText as string }),
      }
    : undefined;
}
