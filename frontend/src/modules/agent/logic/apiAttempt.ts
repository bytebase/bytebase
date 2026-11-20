import {
  type AgentApiExecutionEvidence,
  agentApiExecutionEvidence,
  isAgentApiEvidenceForEnvelope,
  isAgentApiEvidenceForOperation,
} from "./apiExecution";
import { getAgentApiActionLiveness } from "./runOwnership";
import {
  apiCallNeedsConfirmation,
  getApiMethodDescriptor,
} from "./tools/apiMethodClass";
import type { PreparedApiCall } from "./tools/callApi";
import type { AgentChat, AgentMessage, AgentPendingAsk } from "./types";

export interface AgentApiActionIdentity {
  scopeKey: string;
  userName: string;
  chatId: string;
  runId: string;
  rootUserMessageId: string;
  toolCallId: string;
  logicalActionId: string;
}

export interface AgentApiAttemptInput extends AgentApiActionIdentity {
  operationId: string;
  path: string;
  fingerprint: string;
}

export function priorUncertainApiAttemptIds(
  chats: readonly Pick<AgentChat, "apiAttempts">[],
  fingerprint: string
): string[] {
  if (!digest(fingerprint)) return [];
  return chats.flatMap((chat) =>
    (chat.apiAttempts ?? [])
      .filter(
        (a) =>
          a.fingerprint === fingerprint &&
          !a.fingerprintKind &&
          (a.phase === "dispatching" ||
            a.execution?.effects === "possible" ||
            a.execution?.effects === "partial")
      )
      .map((a) => a.id)
  );
}

export interface AgentApiAttempt extends AgentApiAttemptInput {
  version: 1;
  id: string;
  ordinal: number;
  startedAt: number;
  phase: "dispatching" | "settled";
  fingerprintKind?: "legacy-orphan";
  settledAt?: number;
  execution?: AgentApiExecutionEvidence;
  /** Inert metadata retained from previously saved conversations. */
  reviewedExternallyAt?: number;
  opaqueReceipts?: unknown[];
}

export type AgentApiAttemptAdmission =
  | { kind: "admitted"; attemptId: string }
  | { kind: "blocked"; reason: string };
export interface AgentApiDispatchContext extends AgentApiActionIdentity {
  isCurrent(): boolean;
  beginAgentApiAttempt(
    input: AgentApiAttemptInput
  ): Promise<AgentApiAttemptAdmission>;
  settleAgentApiAttempt(
    attemptId: string,
    evidence: AgentApiExecutionEvidence,
    toolResult: string
  ): Promise<boolean>;
}

const record = (x: unknown): x is Record<string, unknown> =>
  Boolean(x && typeof x === "object" && !Array.isArray(x));
const string = (x: unknown): x is string =>
  typeof x === "string" && x.length > 0;
const digest = (x: unknown): x is string =>
  typeof x === "string" && /^[0-9a-f]{64}$/.test(x);
const time = (x: unknown): x is number =>
  typeof x === "number" && Number.isSafeInteger(x) && x >= 0;
function canonical(x: unknown): string {
  if (Array.isArray(x)) return `[${x.map(canonical).join(",")}]`;
  if (record(x))
    return `{${Object.keys(x)
      .sort((a, b) => (a < b ? -1 : a > b ? 1 : 0))
      .filter((k) => x[k] !== undefined)
      .map((k) => `${JSON.stringify(k)}:${canonical(x[k])}`)
      .join(",")}}`;
  return JSON.stringify(x);
}

export async function digestAgentApiJson(value: unknown): Promise<string> {
  const buffer = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(canonical(value))
  );
  return Array.from(new Uint8Array(buffer), (b) =>
    b.toString(16).padStart(2, "0")
  ).join("");
}

export const fingerprintPreparedApiCall = ({
  operationId,
  path,
  body,
}: PreparedApiCall) =>
  digestAgentApiJson({ operationId, path, body: JSON.parse(body) });

export const isAgentApiAttemptUncertain = (attempt: AgentApiAttempt) =>
  attempt.phase !== "settled" ||
  !attempt.execution ||
  attempt.execution.effects === "possible" ||
  attempt.execution.effects === "partial";

export async function getAgentApiAttemptStatus(attempt: AgentApiAttempt) {
  if (attempt.phase === "settled")
    return attempt.execution?.result ?? "unknown";
  const live = await getAgentApiActionLiveness(
    attempt.scopeKey,
    attempt.logicalActionId
  );
  return live === "live"
    ? "in-flight"
    : live === "absent"
      ? "unknown"
      : "unavailable";
}

const quarantine = (
  chatId: string,
  index: number,
  id?: string,
  opaqueReceipts?: unknown[]
): AgentApiAttempt => {
  const attemptId = id ?? `unreadable:${chatId}:${index}`;
  return {
    version: 1,
    ...(opaqueReceipts && { opaqueReceipts }),
    id: attemptId,
    logicalActionId: attemptId,
    ordinal: 1,
    scopeKey: "*",
    userName: "*",
    chatId,
    runId: "*",
    toolCallId: "*",
    rootUserMessageId: "*",
    operationId: "*",
    path: "*",
    fingerprint: "*",
    startedAt: 0,
    phase: "settled",
    settledAt: 0,
    execution: agentApiExecutionEvidence("journal-unreadable", { attemptId }),
  };
};

function normalize(
  raw: unknown,
  chatId: string,
  index: number
): AgentApiAttempt {
  const id = record(raw) && string(raw.id) ? raw.id : undefined;
  const invalid = () => quarantine(chatId, index, id, [raw]);
  if (
    record(raw) &&
    raw.version === 1 &&
    raw.chatId === chatId &&
    raw.fingerprint === "*" &&
    raw.phase === "settled" &&
    record(raw.execution) &&
    raw.execution.reason === "journal-unreadable" &&
    string(raw.id) &&
    raw.execution.attemptId === raw.id
  ) {
    return {
      ...quarantine(
        chatId,
        index,
        id,
        Array.isArray(raw.opaqueReceipts) ? raw.opaqueReceipts : undefined
      ),
      ...(time(raw.reviewedExternallyAt) && {
        reviewedExternallyAt: raw.reviewedExternallyAt,
      }),
    };
  }
  const legacy =
    record(raw) &&
    raw.fingerprintKind === "legacy-orphan" &&
    raw.phase === "settled" &&
    record(raw.execution) &&
    raw.execution.reason === "legacy-interrupted";
  if (
    !record(raw) ||
    raw.version !== 1 ||
    raw.chatId !== chatId ||
    !string(raw.id) ||
    !string(raw.logicalActionId) ||
    !string(raw.scopeKey) ||
    !string(raw.userName) ||
    !string(raw.runId) ||
    !string(raw.rootUserMessageId) ||
    !string(raw.toolCallId) ||
    !string(raw.operationId) ||
    !string(raw.path) ||
    !digest(raw.fingerprint) ||
    !time(raw.startedAt) ||
    !time(raw.ordinal) ||
    (legacy ? raw.ordinal !== 0 : raw.ordinal < 1)
  )
    return invalid();
  const identity = {
    version: 1 as const,
    id: raw.id,
    logicalActionId: raw.logicalActionId,
    ordinal: raw.ordinal,
    scopeKey: raw.scopeKey,
    userName: raw.userName,
    chatId,
    runId: raw.runId,
    rootUserMessageId: raw.rootUserMessageId,
    toolCallId: raw.toolCallId,
    operationId: raw.operationId,
    path: raw.path,
    fingerprint: raw.fingerprint,
    startedAt: raw.startedAt,
  };
  if (legacy) Object.assign(identity, { fingerprintKind: "legacy-orphan" });
  if (
    raw.phase === "dispatching" &&
    raw.execution === undefined &&
    raw.settledAt === undefined
  )
    return {
      ...identity,
      phase: "dispatching",
      ...(time(raw.reviewedExternallyAt) && {
        reviewedExternallyAt: raw.reviewedExternallyAt,
      }),
    };
  if (
    raw.phase !== "settled" ||
    !time(raw.settledAt) ||
    !record(raw.execution) ||
    raw.execution.attemptId !== raw.id ||
    !isAgentApiEvidenceForOperation(
      raw.execution,
      raw.execution.httpStatus,
      raw.operationId
    ) ||
    (raw.execution.result !== "unknown" &&
      raw.path !==
        `/${raw.operationId.slice(0, raw.operationId.lastIndexOf("."))}/${raw.operationId.split(".").at(-1)}`)
  )
    return invalid();
  const { reason, attemptId, httpStatus, connectCode } = raw.execution;
  return {
    ...identity,
    phase: "settled",
    settledAt: raw.settledAt,
    execution: agentApiExecutionEvidence(reason, {
      attemptId,
      ...(httpStatus !== undefined && { httpStatus }),
      ...(connectCode !== undefined && { connectCode }),
    }),
    ...(time(raw.reviewedExternallyAt) && {
      reviewedExternallyAt: raw.reviewedExternallyAt,
    }),
  };
}

export function normalizeAgentApiAttempts(
  raw: unknown,
  chatId: string
): AgentApiAttempt[] | undefined {
  if (raw === undefined) return undefined;
  return mergeAgentApiAttempts(
    (Array.isArray(raw) ? raw : [raw]).map((a, i) => normalize(a, chatId, i))
  );
}

// Unknown formats stay in their original persisted form.
export function storedApiReceipts(
  attempt: AgentApiAttempt
): readonly unknown[] {
  return attempt.opaqueReceipts?.length &&
    attempt.opaqueReceipts.every(
      (raw, i) =>
        normalize(raw, attempt.chatId, i).execution?.reason ===
        "journal-unreadable"
    )
    ? attempt.opaqueReceipts
    : [attempt];
}

export function reconcileApiResultLinks(
  chatId: string,
  scopeKey: string | undefined,
  attempts: AgentApiAttempt[] | undefined,
  messages: readonly AgentMessage[]
) {
  const links = new Map<string, AgentMessage[]>();
  for (const m of messages) {
    const id = m.metadata?.apiAttemptId;
    if (m.role === "tool" && string(id)) {
      const previous = links.get(id) ?? [];
      previous.push(m);
      links.set(id, previous);
    }
  }
  const result = (attempts ?? []).map((a, index) => {
    const rows = links.get(a.id) ?? [];
    links.delete(a.id);
    if (scopeKey && a.scopeKey !== "*" && a.scopeKey !== scopeKey)
      return quarantine(chatId, index, a.id, [a]);
    if (
      a.execution?.reason === "journal-conflict" ||
      a.execution?.reason === "journal-unreadable"
    )
      return a;
    if (a.phase === "dispatching" && !rows.length) return a;
    const row = rows[0];
    if (
      rows.length === 1 &&
      a.phase === "settled" &&
      row.toolCallId === a.toolCallId &&
      row.chatId === a.chatId &&
      row.metadata?.runId === a.runId
    ) {
      try {
        const body = JSON.parse(row.content ?? "");
        if (
          record(body) &&
          isAgentApiEvidenceForEnvelope(
            body.agentExecution,
            body,
            a.operationId
          ) &&
          canonical(body.agentExecution) === canonical(a.execution)
        )
          return a;
      } catch {
        /* An unreadable tool result cannot verify a settlement. */
      }
    }
    return quarantine(chatId, index, a.id, [a]);
  });
  for (const id of links.keys())
    result.push(quarantine(chatId, result.length, id));
  return result.length ? result : undefined;
}

export function mergeAgentApiAttempts(
  ...groups: (readonly AgentApiAttempt[] | undefined)[]
): AgentApiAttempt[] | undefined {
  const seen = new Map<string, AgentApiAttempt>();
  for (const group of groups)
    for (const next of group ?? []) {
      const previous = seen.get(next.id);
      if (!previous) {
        seen.set(next.id, next);
        continue;
      }
      const {
        phase: pPhase,
        execution: pEvidence,
        settledAt: pTime,
        reviewedExternallyAt: pReview,
        ...pIdentity
      } = previous;
      const {
        phase: nPhase,
        execution: nEvidence,
        settledAt: nTime,
        reviewedExternallyAt: nReview,
        ...nIdentity
      } = next;
      if (canonical(pIdentity) !== canonical(nIdentity)) {
        const opaque =
          previous.opaqueReceipts || next.opaqueReceipts
            ? [
                ...new Map(
                  [
                    ...(previous.opaqueReceipts ?? [previous]),
                    ...(next.opaqueReceipts ?? [next]),
                  ].map((entry) => [canonical(entry), entry])
                ).values(),
              ]
            : undefined;
        seen.set(next.id, quarantine(previous.chatId, 0, next.id, opaque));
        continue;
      }
      if (
        pPhase === "settled" &&
        nPhase === "settled" &&
        canonical(pEvidence) !== canonical(nEvidence)
      ) {
        seen.set(next.id, {
          ...pIdentity,
          phase: "settled",
          settledAt: Math.max(pTime ?? 0, nTime ?? 0),
          execution: agentApiExecutionEvidence("journal-conflict", {
            attemptId: next.id,
          }),
        });
        continue;
      }
      const winner = pPhase === "settled" ? previous : next;
      const reviewed = Math.max(pReview ?? -1, nReview ?? -1);
      seen.set(
        next.id,
        reviewed < 0 ? winner : { ...winner, reviewedExternallyAt: reviewed }
      );
    }
  return seen.size ? [...seen.values()] : undefined;
}

// Only an interrupted turn with a missing result is evidence of an old dispatch gap.
export function getLegacyApiOrphans(
  chat: Pick<AgentChat, "interrupted" | "status" | "runId" | "apiAttempts">,
  messages: readonly AgentMessage[],
  pendingAsk?: AgentPendingAsk | null
) {
  if (
    (!chat.interrupted && chat.status !== "running") ||
    pendingAsk ||
    chat.status === "awaiting_user"
  )
    return [];
  const root = messages.findLast((m) => m.role === "user");
  if (!root) return [];
  const turn = messages.slice(messages.indexOf(root) + 1);
  return turn.flatMap((m, index) =>
    m.role === "assistant"
      ? (m.toolCalls ?? []).flatMap((tool) => {
          if (
            tool.name !== "call_api" ||
            !tool.id ||
            turn
              .slice(index + 1)
              .some(
                (row) => row.role === "tool" && row.toolCallId === tool.id
              ) ||
            chat.apiAttempts?.some(
              (a) => a.toolCallId === tool.id && a.rootUserMessageId === root.id
            )
          )
            return [];
          let operation: unknown;
          try {
            operation = JSON.parse(tool.arguments).operationId;
          } catch {
            /* Unknown methods are protected. */
          }
          if (
            typeof operation === "string" &&
            !apiCallNeedsConfirmation(operation)
          )
            return [];
          const method =
            typeof operation === "string"
              ? getApiMethodDescriptor(operation)
              : undefined;
          return [
            {
              id: `legacy:${JSON.stringify([m.id, tool.id])}`,
              toolCallId: tool.id,
              rootUserMessageId: root.id,
              runId: m.metadata?.runId ?? chat.runId ?? `legacy:${root.id}`,
              startedAt: m.createdTs,
              operationId: method
                ? `${method.parent.typeName}.${method.name}`
                : "unknown",
              path: method
                ? `/${method.parent.typeName}/${method.name}`
                : "unknown",
            },
          ];
        })
      : []
  );
}
