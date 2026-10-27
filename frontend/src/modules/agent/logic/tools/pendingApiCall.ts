import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchPreparedApiCall, type PreparedApiCall } from "./callApi";
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
}

const pendingApiCalls = new Map<string, PendingApiCall>();

export function createPendingApiCall({
  chatId,
  toolCallId,
  userName,
  call,
  issueDecision,
}: {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  issueDecision?: IssueDecisionPermission;
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
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  approved: boolean;
  signal?: AbortSignal;
}): Promise<string> {
  const pending = pendingApiCalls.get(actionId);
  if (!pending) {
    return JSON.stringify({
      error: "Permission request not found or already used.",
    });
  }
  if (pending.expiresAt <= Date.now()) {
    pendingApiCalls.delete(actionId);
    return JSON.stringify({ error: "Permission request expired." });
  }
  if (
    pending.chatId !== chatId ||
    pending.toolCallId !== toolCallId ||
    pending.userName !== userName
  ) {
    return JSON.stringify({
      error: "Permission request does not match this user, chat, or action.",
    });
  }

  pendingApiCalls.delete(actionId);
  if (!approved) {
    return JSON.stringify({ error: "Permission request denied." });
  }
  if (signal?.aborted) {
    return JSON.stringify({
      error: "Request stopped before the API call was sent.",
    });
  }
  if (pending.issueDecision) {
    const valid = await validateIssueDecisionPermission(pending.issueDecision);
    if (signal?.aborted) {
      return JSON.stringify({
        error: "Request stopped before the API call was sent.",
      });
    }
    if (!valid) {
      return JSON.stringify({
        error:
          "The Issue changed or became unavailable since permission was requested. Refresh and retry the decision.",
      });
    }
  }
  const access = await checkPreparedApiPermission(pending.call);
  if (signal?.aborted) {
    return JSON.stringify({
      error: "Request stopped before the API call was sent.",
    });
  }
  if (access.kind === "denied") {
    return access.result;
  }
  return dispatchPreparedApiCall(pending.call);
}
