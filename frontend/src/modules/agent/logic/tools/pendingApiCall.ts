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
import { isAgentExecutionContextCurrent } from "../executionContext";
import type { AgentDiagnosticCapability } from "../types";
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
  domOrigin?: CapturedDomMutation;
  expiresAt: number;
}

const pendingApiCalls = new Map<string, PendingApiCall>();

export const staleDomPermission = JSON.stringify({
  called: false,
  error:
    "Not called: The page action changed, expired, or could not be verified. Inspect the page again.",
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
}: {
  chatId: string;
  toolCallId: string;
  userName: string;
  call: PreparedApiCall;
  issueDecision?: IssueDecisionPermission;
  domOrigin?: CapturedDomMutation;
  domContext?: AgentDomContext;
}): { actionId: string; expiresAt: number } {
  if (!userName) {
    throw new Error("A signed-in user is required to request permission.");
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
}: {
  actionId: string;
  chatId: string;
  toolCallId: string;
  userName: string;
  approved: boolean;
  signal?: AbortSignal;
  capability?: AgentDiagnosticCapability;
  domContext?: AgentDomContext;
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
  if (capability !== undefined) {
    return JSON.stringify({
      error: "API permission requests cannot execute in a diagnostic chat.",
    });
  }
  const origin = pending.domOrigin;
  const domCurrent = () =>
    Date.now() < pending.expiresAt &&
    !!origin &&
    isCurrentDomApiCall(origin, pending.call, domContext);
  if (!approved) {
    return origin && !domCurrent()
      ? staleDomPermission
      : JSON.stringify({ error: "Permission request denied." });
  }
  const checkContext = () => {
    if (signal?.aborted)
      return JSON.stringify({
        error: "Request stopped before the API call was sent.",
      });
    if (origin && !domCurrent()) return staleDomPermission;
  };
  let failure = checkContext();
  if (failure) return failure;
  const access = await checkPreparedApiPermission(pending.call);
  failure = checkContext();
  if (failure) return failure;
  if (access.kind === "denied") {
    return access.result;
  }
  if (pending.issueDecision) {
    const valid = await validateIssueDecisionPermission(pending.issueDecision);
    failure = checkContext();
    if (failure) return failure;
    if (!valid) {
      return JSON.stringify({
        error:
          "The Issue changed or became unavailable since permission was requested. Refresh and retry the decision.",
      });
    }
  }
  if (!origin) return dispatchPreparedApiCall(pending.call);
  if (!domContext || !(await validateCapturedDomMutation(origin, domContext)))
    return staleDomPermission;
  failure = checkContext();
  if (failure) return failure;
  const result = await dispatchPreparedApiCall(pending.call, signal, {
    expectedContext: origin,
    beforeSend: domCurrent,
  });
  const envelope = JSON.parse(result);
  if (
    typeof envelope?.status === "number" &&
    envelope.status >= 200 &&
    envelope.status < 300
  ) {
    try {
      await refreshCommittedDomMutation(origin, domContext);
    } catch {
      return JSON.stringify({ ...envelope, domRefresh: "failed" });
    }
  }
  return result;
}
