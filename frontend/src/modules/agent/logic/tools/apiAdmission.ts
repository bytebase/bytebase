import {
  type AgentDomContext,
  type CapturedDomMutation,
} from "../../dom/interactionRegistry";
import { fingerprintPreparedApiCall } from "../apiAttempt";
import { notSentApiResult, preflightRejectedApiResult } from "../callApiResult";
import { normalizeAgentDiagnosticCapability } from "../diagnosticCapability";
import type { AgentSqlExecutionGuard } from "../sqlExecution";
import type { AgentDiagnosticCapability, ToolExecutionResult } from "../types";
import { apiCallNeedsConfirmation } from "./apiMethodClass";
import { checkPreparedApiPermission } from "./apiPermissionPreflight";
import { dispatchPreparedApiCall, type PreparedApiCall } from "./callApi";
import { isDiagnosticApiCallInScope } from "./diagnosticApiScope";
import { prepareIssueDecisionPermission } from "./issueDecisionPermission";
import {
  createPendingApiCall,
  isCurrentDomApiCall,
  isSupportedDomApiCall,
  staleDomPermission,
} from "./pendingApiCall";
import { apiTargetsAnotherProject } from "./referencedProjectApiScope";

export interface AgentApiAdmissionInput {
  call: PreparedApiCall;
  toolCallId: string;
  chatId?: string;
  userName?: string;
  signal?: AbortSignal;
  referencedProject?: string;
  referencedIssue?: string;
  diagnosticRequested: boolean;
  sqlOrigin?: AgentSqlExecutionGuard;
  capability?: AgentDiagnosticCapability;
  priorAttemptIds?: (fingerprint: string) => string[];
  isScopeCurrent?: () => boolean;
  dom?: { origin: CapturedDomMutation; context: AgentDomContext };
}

const toolResult = (result: string): ToolExecutionResult => ({
  kind: "tool_result",
  result,
});

export async function admitPreparedAgentApiCall(
  input: AgentApiAdmissionInput
): Promise<ToolExecutionResult> {
  const { call, toolCallId, dom } = input;
  const capability = normalizeAgentDiagnosticCapability(input.capability);
  if (input.diagnosticRequested && !capability)
    return toolResult(
      notSentApiResult(
        "The diagnostic API scope could not be verified.",
        "preflight-rejected"
      )
    );
  const current = () =>
    !input.signal?.aborted &&
    input.isScopeCurrent?.() !== false &&
    input.sqlOrigin?.isCurrent() !== false &&
    (!dom || isCurrentDomApiCall(dom.origin, call, dom.context));
  if (
    input.sqlOrigin &&
    (input.diagnosticRequested ||
      capability ||
      dom ||
      call.path !== "/bytebase.v1.SQLService/Query" ||
      call.operationId !== "bytebase.v1.SQLService.Query")
  )
    return toolResult(
      notSentApiResult("Invalid SQL action request.", "preflight-rejected")
    );
  const stale = () =>
    toolResult(
      dom
        ? staleDomPermission
        : notSentApiResult(
            "Agent API request ownership changed before admission.",
            "admission-rejected"
          )
    );
  if (dom && (!current() || !isSupportedDomApiCall(dom.origin, call)))
    return toolResult(staleDomPermission);
  if (!current()) return stale();
  if (
    capability &&
    !(await isDiagnosticApiCallInScope(call, capability).catch(() => false))
  )
    return toolResult(
      notSentApiResult(
        `This API request is outside the referenced project (${capability.resource.project}) or its diagnostic scope could not be verified. Approval cannot override this boundary.`,
        "preflight-rejected"
      )
    );
  if (
    !input.diagnosticRequested &&
    input.referencedProject &&
    (await apiTargetsAnotherProject(call, input.referencedProject))
  )
    return toolResult(
      notSentApiResult(
        `This API request is outside the referenced project (${input.referencedProject}). Ask in a separate message to use another project.`,
        "preflight-rejected"
      )
    );
  if (!current()) return stale();
  const access = await checkPreparedApiPermission(call);
  if (!current()) return stale();
  if (access.kind === "denied")
    return toolResult(preflightRejectedApiResult(access.result));
  const issueDecision = await prepareIssueDecisionPermission(
    call,
    input.referencedIssue
  );
  if (!current()) return stale();
  if (issueDecision.kind === "error")
    return toolResult(preflightRejectedApiResult(issueDecision.result));
  if (
    !input.sqlOrigin &&
    !dom &&
    issueDecision.kind === "not-issue-decision" &&
    !apiCallNeedsConfirmation(call.operationId)
  ) {
    if (capability)
      return toolResult(
        await dispatchPreparedApiCall(call, {
          signal: input.signal,
          isScopeCurrent: input.isScopeCurrent,
          validateBeforeSend: () =>
            isDiagnosticApiCallInScope(call, capability),
        })
      );
    return toolResult(
      await (input.signal
        ? dispatchPreparedApiCall(call, { signal: input.signal })
        : dispatchPreparedApiCall(call))
    );
  }
  if (!input.chatId || !input.userName)
    return toolResult(
      notSentApiResult(
        "Cannot request API permission without a signed-in chat.",
        "permission-rejected"
      )
    );
  const fingerprint = input.priorAttemptIds
    ? await fingerprintPreparedApiCall(call).catch(() => undefined)
    : undefined;
  if (!current()) return stale();
  const { actionId, expiresAt } = createPendingApiCall({
    chatId: input.chatId,
    toolCallId,
    userName: input.userName,
    call,
    domOrigin: dom?.origin,
    domContext: dom?.context,
    sqlOrigin: input.sqlOrigin,
    priorAttemptIds: fingerprint
      ? input.priorAttemptIds?.(fingerprint)
      : undefined,
    isScopeCurrent: input.isScopeCurrent,
    ...(capability ? { capability } : {}),
    issueDecision:
      issueDecision.kind === "issue-decision"
        ? issueDecision.decision
        : undefined,
  });
  return {
    kind: "ask_user",
    ask: {
      kind: "confirm",
      toolCallId,
      prompt: call.operationId,
      apiPermission: { actionId, operationId: call.operationId, expiresAt },
    },
  };
}
