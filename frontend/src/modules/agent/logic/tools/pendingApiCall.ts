import type { AgentApiDispatchContext } from "../apiAttempt";
import { notSentApiResult, preflightRejectedApiResult } from "../callApiResult";
import type { AgentDiagnosticCapability } from "../types";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchApprovedApiCall, type PreparedApiCall } from "./callApi";
import {
  type IssueDecisionPermission,
  validateIssueDecisionPermission,
} from "./issueDecisionPermission";

const PERMISSION_TTL_MS = 5 * 60_000;

interface PendingApiCall {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  issueDecision?: IssueDecisionPermission;
  expiresAt: number;
  priorAttemptIds?: readonly string[];
  isScopeCurrent?: () => boolean;
}

const pendingApiCalls = new Map<string, PendingApiCall>();

export function createPendingApiCall({
  chatId,
  toolCallId,
  userName,
  call,
  issueDecision,
  priorAttemptIds,
  isScopeCurrent,
}: {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  issueDecision?: IssueDecisionPermission;
  priorAttemptIds?: readonly string[];
  isScopeCurrent?: () => boolean;
}): { actionId: string; expiresAt: number } {
  if (!userName) {
    throw new Error("A signed-in user is required to request permission.");
  }
  const now = Date.now();
  for (const [id, pending] of pendingApiCalls) {
    if (
      pending.expiresAt <= now ||
      (pending.chatId === chatId && pending.toolCallId === toolCallId)
    ) {
      pendingApiCalls.delete(id);
    }
  }

  let actionId: string;
  do {
    actionId = Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
      byte.toString(16).padStart(2, "0")
    ).join("");
  } while (pendingApiCalls.has(actionId));
  const expiresAt = now + PERMISSION_TTL_MS;
  pendingApiCalls.set(actionId, {
    chatId,
    toolCallId,
    userName,
    call: Object.freeze({ ...call }),
    issueDecision: issueDecision
      ? Object.freeze({ ...issueDecision })
      : undefined,
    expiresAt,
    isScopeCurrent,
    ...(priorAttemptIds?.length && { priorAttemptIds: [...priorAttemptIds] }),
  });
  return { actionId, expiresAt };
}

export function getPendingApiCallPreview({
  actionId,
  chatId,
  toolCallId,
  userName,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
}):
  | {
      operationId: string;
      body: string;
      priorAttemptIds?: readonly string[];
      expiresAt: number;
      issueDecision?: {
        issueName: string;
        issueTitle: string;
        impact: IssueDecisionPermission["impact"];
      };
    }
  | undefined {
  const pending = pendingApiCalls.get(actionId);
  if (
    !pending ||
    pending.expiresAt <= Date.now() ||
    pending.chatId !== chatId ||
    pending.toolCallId !== toolCallId ||
    pending.userName !== userName
  ) {
    return undefined;
  }
  return {
    operationId: pending.call.operationId,
    body: pending.call.body,
    ...(pending.priorAttemptIds && {
      priorAttemptIds: [...pending.priorAttemptIds],
    }),
    expiresAt: pending.expiresAt,
    ...(pending.issueDecision
      ? {
          issueDecision: {
            issueName: pending.issueDecision.issueName,
            issueTitle: pending.issueDecision.issueTitle,
            impact: pending.issueDecision.impact,
          },
        }
      : {}),
  };
}

export async function resolvePendingApiCall({
  actionId,
  chatId,
  toolCallId,
  userName,
  approved,
  signal,
  capability,
  context,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  approved: boolean;
  signal?: AbortSignal;
  capability?: AgentDiagnosticCapability;
  context?: AgentApiDispatchContext;
}): Promise<string> {
  const pending = pendingApiCalls.get(actionId);
  if (!pending) {
    return notSentApiResult(
      "Permission request not found or already used.",
      "permission-rejected"
    );
  }
  if (pending.expiresAt <= Date.now()) {
    pendingApiCalls.delete(actionId);
    return notSentApiResult(
      "Permission request expired.",
      "permission-rejected"
    );
  }
  if (
    pending.chatId !== chatId ||
    pending.toolCallId !== toolCallId ||
    pending.userName !== userName
  ) {
    return notSentApiResult(
      "Permission request does not match this user, chat, or action.",
      "permission-rejected"
    );
  }

  pendingApiCalls.delete(actionId);
  if (capability !== undefined) {
    return notSentApiResult(
      "API permission requests cannot execute in a diagnostic chat.",
      "permission-rejected"
    );
  }
  if (!approved) {
    return notSentApiResult(
      "Permission request denied.",
      "permission-rejected"
    );
  }
  if (signal?.aborted) {
    return notSentApiResult(
      "Request stopped before the API call was sent.",
      "canceled-before-dispatch"
    );
  }
  const access = await checkPreparedApiPermission(pending.call);
  if (signal?.aborted) {
    return notSentApiResult(
      "Request stopped before the API call was sent.",
      "canceled-before-dispatch"
    );
  }
  if (access.kind === "denied") {
    return preflightRejectedApiResult(access.result);
  }
  if (pending.issueDecision) {
    const valid = await validateIssueDecisionPermission(pending.issueDecision);
    if (signal?.aborted) {
      return notSentApiResult(
        "Request stopped before the API call was sent.",
        "canceled-before-dispatch"
      );
    }
    if (!valid) {
      return notSentApiResult(
        "The Issue changed or became unavailable since permission was requested. Refresh and retry the decision.",
        "preflight-rejected"
      );
    }
  }
  if (
    pending.isScopeCurrent?.() === false ||
    (context &&
      (context.userName !== userName ||
        context.chatId !== chatId ||
        context.toolCallId !== toolCallId ||
        context.logicalActionId !== actionId))
  )
    return notSentApiResult(
      "Permission has no matching execution context.",
      "admission-rejected"
    );
  return dispatchApprovedApiCall(pending.call, {
    context,
    signal,
    ...(pending.isScopeCurrent && { isScopeCurrent: pending.isScopeCurrent }),
  });
}
