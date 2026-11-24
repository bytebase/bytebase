import { fromJsonString } from "@bufbuild/protobuf";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  type AgentDomContext,
  type AgentDomMutationPreview,
  type CapturedDomMutation,
  isCapturedDomMutationCurrent,
  matchesCapturedDomMutationCall,
  refreshCommittedDomMutation,
  validateCapturedDomMutation,
} from "../../dom/interactionRegistry";
import type { AgentApiDispatchContext } from "../apiAttempt";
import {
  notSentApiResult,
  parseCallApiResult,
  preflightRejectedApiResult,
} from "../callApiResult";
import {
  normalizeAgentDiagnosticCapability,
  sameAgentDiagnosticCapability,
} from "../diagnosticCapability";
import { isAgentExecutionContextCurrent } from "../executionContext";
import type { AgentDiagnosticCapability } from "../types";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchApprovedApiCall, type PreparedApiCall } from "./callApi";
import { isDiagnosticApiCallInScope } from "./diagnosticApiScope";
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
  capability?: AgentDiagnosticCapability;
  issueDecision?: IssueDecisionPermission;
  domOrigin?: CapturedDomMutation;
  expiresAt: number;
  priorAttemptIds?: readonly string[];
  isScopeCurrent?: () => boolean;
}

const pendingApiCalls = new Map<string, PendingApiCall>();

export const staleDomPermission = JSON.stringify({
  called: false,
  ...JSON.parse(
    notSentApiResult(
      "Not called: The page action changed, expired, or could not be verified. Inspect the page again.",
      "admission-rejected"
    )
  ),
});

export function isCurrentDomApiCall(
  origin: CapturedDomMutation,
  call: PreparedApiCall,
  context?: AgentDomContext
): boolean {
  return (
    !!context &&
    isAgentExecutionContextCurrent(origin, context.signal) &&
    matchesCapturedDomMutationCall(origin, call) &&
    isCapturedDomMutationCurrent(origin, context)
  );
}

export function isSupportedDomApiCall(
  origin: CapturedDomMutation,
  call: PreparedApiCall
): boolean {
  if (
    !origin.projectName ||
    !/^projects\/(?!-$)[^/]+$/.test(origin.projectName) ||
    !/^(?:projects\/(?!-\/)[^/]+\/)?instances\/(?!-\/)[^/]+\/databases\/(?!-$)[^/]+$/.test(
      origin.resourceName
    ) ||
    (origin.resourceName.startsWith("projects/") &&
      !origin.resourceName.startsWith(`${origin.projectName}/instances/`)) ||
    call.operationId !== "bytebase.v1.DatabaseService.UpdateDatabase" ||
    call.path !== "/bytebase.v1.DatabaseService/UpdateDatabase"
  )
    return false;
  try {
    const field =
      origin.preview.kind === "database-labels"
        ? "labels"
        : origin.preview.kind === "database-environment"
          ? "environment"
          : undefined;
    const request = fromJsonString(UpdateDatabaseRequestSchema, call.body);
    return (
      !!field &&
      request.database?.name === origin.resourceName &&
      request.updateMask?.paths.length === 1 &&
      request.updateMask.paths[0] === field
    );
  } catch {
    return false;
  }
}

export function createPendingApiCall({
  chatId,
  toolCallId,
  userName,
  call,
  issueDecision,
  domOrigin,
  domContext,
  priorAttemptIds,
  isScopeCurrent,
  capability,
}: {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  issueDecision?: IssueDecisionPermission;
  domOrigin?: CapturedDomMutation;
  domContext?: AgentDomContext;
  priorAttemptIds?: readonly string[];
  isScopeCurrent?: () => boolean;
  capability?: AgentDiagnosticCapability;
}): { actionId: string; expiresAt: number } {
  if (!userName) {
    throw new Error("A signed-in user is required to request permission.");
  }
  const diagnostic = normalizeAgentDiagnosticCapability(capability);
  if (capability !== undefined && (!diagnostic || domOrigin)) {
    throw new Error("The diagnostic API scope could not be verified.");
  }
  const now = Date.now();
  if (
    domOrigin &&
    (domOrigin.chatId !== chatId ||
      domOrigin.actor !== userName ||
      !isSupportedDomApiCall(domOrigin, call) ||
      !isCurrentDomApiCall(domOrigin, call, domContext))
  ) {
    throw new Error("The prepared page action could not be verified.");
  }
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
    domOrigin,
    call: Object.freeze({ ...call }),
    capability: diagnostic
      ? Object.freeze({
          ...diagnostic,
          resource: Object.freeze({ ...diagnostic.resource }),
        })
      : undefined,
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
  domContext,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  domContext?: AgentDomContext;
}):
  | {
      operationId: string;
      body: string;
      priorAttemptIds?: readonly string[];
      expiresAt: number;
      dom?: AgentDomMutationPreview & { resourceName: string };
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
    pending.userName !== userName ||
    (pending.domOrigin &&
      !isCurrentDomApiCall(pending.domOrigin, pending.call, domContext))
  ) {
    return undefined;
  }
  return {
    operationId: pending.call.operationId,
    body: pending.call.body,
    ...(pending.domOrigin
      ? {
          dom: {
            ...pending.domOrigin.preview,
            resourceName: pending.domOrigin.resourceName,
          },
        }
      : {}),
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
  domContext,
  context,
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  approved: boolean;
  signal?: AbortSignal;
  capability?: AgentDiagnosticCapability;
  domContext?: AgentDomContext;
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
  if (!sameAgentDiagnosticCapability(pending.capability, capability)) {
    return notSentApiResult(
      "Permission request does not match this diagnostic scope.",
      "permission-rejected"
    );
  }
  const origin = pending.domOrigin;
  const diagnostic = pending.capability;
  const domCurrent = () =>
    Date.now() < pending.expiresAt &&
    !!origin &&
    isCurrentDomApiCall(origin, pending.call, domContext);
  const contextCurrent = () =>
    pending.isScopeCurrent?.() !== false &&
    (!pending.capability || !!context) &&
    (!context ||
      (context.userName === userName &&
        context.chatId === chatId &&
        context.toolCallId === toolCallId &&
        context.logicalActionId === actionId &&
        context.isCurrent()));
  if (!approved) {
    return origin && !domCurrent()
      ? staleDomPermission
      : notSentApiResult("Permission request denied.", "permission-rejected");
  }
  const checkContext = () => {
    if (signal?.aborted)
      return notSentApiResult(
        "Request stopped before the API call was sent.",
        "canceled-before-dispatch"
      );
    if (origin && !domCurrent()) return staleDomPermission;
    if (Date.now() >= pending.expiresAt)
      return notSentApiResult(
        "Permission request expired.",
        "permission-rejected"
      );
    if (!sameAgentDiagnosticCapability(pending.capability, capability))
      return notSentApiResult(
        "Permission request does not match this diagnostic scope.",
        "permission-rejected"
      );
    if (!contextCurrent())
      return notSentApiResult(
        "Permission has no matching execution context.",
        "admission-rejected"
      );
  };
  let failure = checkContext();
  if (failure) return failure;
  if (diagnostic) {
    const inScope = await isDiagnosticApiCallInScope(
      pending.call,
      diagnostic
    ).catch(() => false);
    failure = checkContext();
    if (failure) return failure;
    if (!inScope)
      return notSentApiResult(
        "The approved API call is outside this diagnostic scope.",
        "preflight-rejected"
      );
  }
  const access = await checkPreparedApiPermission(pending.call);
  failure = checkContext();
  if (failure) return failure;
  if (access.kind === "denied") {
    return preflightRejectedApiResult(access.result);
  }
  if (pending.issueDecision) {
    const valid = await validateIssueDecisionPermission(pending.issueDecision);
    failure = checkContext();
    if (failure) return failure;
    if (!valid) {
      return notSentApiResult(
        "The Issue changed or became unavailable since permission was requested. Refresh and retry the decision.",
        "preflight-rejected"
      );
    }
  }
  if (origin) {
    if (!domContext || !(await validateCapturedDomMutation(origin, domContext)))
      return staleDomPermission;
    failure = checkContext();
    if (failure) return failure;
  }
  const result = await dispatchApprovedApiCall(pending.call, {
    context,
    signal,
    isScopeCurrent: () =>
      (!!origin || Date.now() < pending.expiresAt) &&
      sameAgentDiagnosticCapability(pending.capability, capability) &&
      contextCurrent(),
    ...(diagnostic && {
      validateBeforeSend: () =>
        isDiagnosticApiCallInScope(pending.call, diagnostic),
    }),
    ...(origin && { dom: { expectedContext: origin, beforeSend: domCurrent } }),
  });
  if (
    origin &&
    domContext &&
    parseCallApiResult(result, pending.call.operationId).execution?.reason ===
      "rpc-success"
  ) {
    try {
      await refreshCommittedDomMutation(origin, domContext);
    } catch {
      return JSON.stringify({ ...JSON.parse(result), domRefresh: "failed" });
    }
  }
  return result;
}
