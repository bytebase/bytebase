import { Code } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import { type AgentApiRecovery, getAgentApiRecovery } from "./apiResponse";
import { parseCallApiResult } from "./callApiResult";
import { queryAccessRecoveryInstruction } from "./skills/query";
import {
  getCurrentTurnToolCalls,
  type ToolObservation,
} from "./toolObservations";
import type { Message } from "./types";

export const apiResponseSafetyInstruction =
  "If call_api returns an error or HTTP status 400 or higher, a Query result contains permissionDenied, or an API tool reports error or uncertainty, failure does not prove rollback. Distinguish not sent, rejected, partial and unknown effects. Do not claim the query or change succeeded or repeat the original request automatically. Prior errors are not current restrictions; use current tools and normal permissions within scope. Without safe recovery, use done({ text, success: false, actions: [] }).";

export const apiResponseFallbackInstruction =
  "Explain the observed failure or uncertainty using available evidence, separating known facts from unknown causes and effects. Only describe a request as sent or completed when execution evidence supports it. Preserve confirmed successful parts and stop work that still depends on an unresolved failed operation. Suggest one evidence-supported next step within the current chat's scope; if none is available, explain the limitation rather than inventing a recovery action. Never retry the original request automatically or infer a subscription, permission, or configuration cause from an unclassified error.";

type RecoveryInstruction = readonly [explanation: string, nextStep?: string];

const diagnosticFallbackInstruction =
  "This is diagnostic evidence only. Explain the observed failure or uncertainty using available evidence within the original diagnostic scope. Separate known facts from unknown causes and effects, and preserve confirmed successful parts. State missing evidence or ask for clarifying diagnostic input. Recovery guidance is not authorization: do not automatically retry, request access, change configuration, navigate, or prepare drafts. Protected API requests require the runtime's exact one-time approval within the original diagnostic project; approval never overrides scope or backend authorization. Direct SQL execution and export are unavailable because their effective database targets cannot be verified. Only submit mutations when the member explicitly asks, and never infer consent from a failure or earlier approval.";

const instructions = (
  subject: string
): Record<
  Exclude<AgentApiRecovery["kind"], "api-error">,
  RecoveryInstruction
> => ({
  "feature-required": [
    `${subject} reported an explicit paid-feature requirement. Explain the returned feature and required plan only when provided. This does not establish the current plan, license expiration, or trial eligibility.`,
    "Suggest asking the workspace subscription administrator to review the license or plan; do not promise an upgrade destination. Stop questions and actions that depend on this unavailable feature. Never retry the request automatically or imply that failure proves rollback.",
  ],
  "instance-activation-required": [
    `${subject} reported that this feature requires an activated instance under the current license. Explain the returned instance and feature. This is not evidence that another subscription plan must be purchased.`,
    "Suggest asking the workspace subscription administrator to review instance activation. Never retry or activate an instance automatically, and do not infer that failure proves rollback.",
  ],
  "permission-denied": [
    `${subject} reported a structured permission denial. Explain its exact required permissions and resource scope. A permission denial is not a paid-feature restriction.`,
    "Suggest requesting the reported permission from the resource administrator, not an unnecessarily broad role. Do not offer query Access Grant recovery for metadata or other permissions. Never retry, grant permissions, or submit a request automatically, and do not infer that failure proves rollback.",
  ],
  "query-access-denied": [
    `${subject} reported a structured query access denial.`,
    queryAccessRecoveryInstruction,
  ],
});

const codeInstructions = (
  subject: string
): Partial<Record<Code, RecoveryInstruction>> => ({
  [Code.InvalidArgument]: [
    `${subject} reported invalid input. Explain the reported field or value only when available.`,
    "Propose a correction and ask only for genuinely missing information. Do not invent field details or submit a corrected request without its normal approval.",
  ],
  [Code.OutOfRange]: [
    `${subject} reported a value outside its supported range. Explain only the returned bound or constraint.`,
    "Do not guess a permitted value or resubmit automatically.",
  ],
  [Code.FailedPrecondition]: [
    `${subject} reported an unmet prerequisite. Explain the condition only when provided.`,
    "Do not guess which setting to change or automatically enable a feature.",
  ],
  [Code.Unauthenticated]: [
    `${subject} could not authenticate this request. Explain that its session could not be verified.`,
    "The session may need to be restored through the existing sign-in flow. Do not handle credentials in chat, change the existing authentication behavior, or automatically repeat a protected request.",
  ],
  [Code.PermissionDenied]: [
    `${subject} reported an access denial without verified permission or entitlement details. Explain that the exact cause is unverified.`,
    "Do not invent a missing permission, broad administrator role, paid-feature restriction, or Access Grant recovery.",
  ],
  [Code.NotFound]: [
    `${subject} could not find the requested resource in the accessible scope. This does not prove deletion or absence.`,
    "Suggest verifying its exact identity or refreshing permitted resource state; do not broaden scope to find it.",
  ],
  [Code.AlreadyExists]: [
    `${subject} reported a resource conflict.`,
    "Suggest a permitted read to refresh current state before proposing another attempt. Do not silently overwrite, invent a replacement resource, or repeat the request.",
  ],
  [Code.Aborted]: [
    `${subject} reported a resource conflict or aborted operation.`,
    "Suggest a permitted read to refresh current state before proposing another attempt. Do not assume rollback, overwrite newer state, or repeat the request.",
  ],
  [Code.ResourceExhausted]: [
    `${subject} reported a resource limit. Explain the reported limit only when identified; otherwise say which limit applies is unknown. Do not assume a rate limit, quota, storage shortage, or subscription upgrade without evidence.`,
  ],
  [Code.Unavailable]: [
    `${subject} reported an availability failure and may be temporarily unavailable. Describe only the returned evidence.`,
    "Verify current resource state within scope before proposing another potentially mutating attempt; availability recovery does not authorize replay.",
  ],
  [Code.DeadlineExceeded]: [
    `${subject} reported a deadline exceeded. A timeout does not establish whether work completed.`,
    "Suggest permitted read-only verification before another potentially mutating attempt; do not repeat it automatically.",
  ],
  [Code.Canceled]: [
    `${subject} reported that the operation was canceled. Do not infer who canceled it or whether effects were rolled back without evidence.`,
    "Stop dependent work and do not retry it automatically.",
  ],
  [Code.Unimplemented]: [
    `${subject} reported that the operation is unsupported. Explain that limitation.`,
    "Do not infer a paid-feature restriction or invent another write path to bypass it.",
  ],
});

function hasObservedError(
  raw: string,
  parsedError: string | undefined
): boolean {
  try {
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== "object" || Array.isArray(value))
      return false;
    const { error, status } = value as Record<string, unknown>;
    return (
      (typeof error === "string" && error.trim() !== "") ||
      (typeof status === "number" &&
        Number.isInteger(status) &&
        status >= 400 &&
        status <= 599) ||
      (status === 200 && parsedError !== undefined)
    );
  } catch {
    return false;
  }
}

const operation = (call: ToolObservation) =>
  typeof call.args.operationId === "string"
    ? call.args.operationId.replace(/^bytebase\.v1\./, "").replace("/", ".")
    : call.name;

function sameRequest(left: ToolObservation, right: ToolObservation): boolean {
  return (
    left.name === right.name &&
    operation(left) === operation(right) &&
    isEqual(left.args.body ?? left.args, right.args.body ?? right.args)
  );
}

export function getAgentApiRecoveryGuidance(
  messages: readonly Message[],
  { diagnostic = false }: { diagnostic?: boolean } = {}
): string {
  const calls = getCurrentTurnToolCalls(messages).flatMap((call) => {
    if (call.result === undefined) return [];
    return [
      {
        call,
        raw: call.result,
        recovery: getAgentApiRecovery(call.args, call.result),
        result: parseCallApiResult(
          call.result,
          typeof call.args.operationId === "string"
            ? call.args.operationId
            : undefined
        ),
      },
    ];
  });
  const guidance = new Set<string>();
  const addInstruction = (instruction: RecoveryInstruction | undefined) => {
    if (instruction)
      guidance.add(
        diagnostic ? instruction[0] : instruction.filter(Boolean).join(" ")
      );
  };
  let failed = false;
  for (const [
    index,
    { call, raw, recovery: observedRecovery, result },
  ] of calls.entries()) {
    const execution = result.execution;
    if (call.name !== "call_api" && !observedRecovery && !execution) continue;
    const resolved = calls
      .slice(index + 1)
      .some(
        (later) =>
          sameRequest(call, later.call) &&
          later.result.succeeded === true &&
          later.result.execution?.effects !== "possible" &&
          later.result.execution?.effects !== "partial"
      );
    if (
      resolved &&
      execution?.effects !== "possible" &&
      execution?.effects !== "partial"
    )
      continue;
    const recovery = resolved ? undefined : observedRecovery;
    const executionIssue =
      execution &&
      (execution.result !== "succeeded" ||
        execution.effects === "partial" ||
        execution.reason === "query-result-error" ||
        execution.reason === "import-none-created" ||
        execution.reason === "import-outcome-unknown");
    if (!recovery && !hasObservedError(raw, result.error) && !executionIssue)
      continue;
    failed = true;
    const subject =
      recovery?.source === "api-client"
        ? "The API client"
        : recovery?.source === "preflight"
          ? "The pre-dispatch permission check"
          : "The API";
    if (recovery?.source === "api-client")
      addInstruction([
        "This client error does not establish an HTTP response, request dispatch, completion, or absence of effects; its code may originate in the server, transport, or client.",
      ]);
    if (recovery?.kind === "api-error") {
      const instruction =
        recovery.code !== undefined
          ? codeInstructions(subject)[recovery.code]
          : undefined;
      addInstruction(instruction);
    } else if (recovery) {
      addInstruction(
        instructions(subject)[
          diagnostic && recovery.kind === "query-access-denied"
            ? "permission-denied"
            : recovery.kind
        ]
      );
    }
    if (execution?.result === "unknown")
      addInstruction([
        "The operation's outcome could not be verified. Do not report a definitive failure or success, or imply there were no effects.",
        "Suggest safe read-only verification within the existing scope before another potentially mutating attempt.",
      ]);
    else if (execution?.result === "not-sent")
      addInstruction([
        "The target was not called for the attempt marked not-sent. This does not resolve any earlier attempt's effects. Explain the reported preparation, admission, permission, or cancellation blocker without claiming an API response or assuming the member declined.",
        "Stop; a new attempt requires its normal authorization.",
      ]);
    else if (execution?.effects === "partial")
      addInstruction([
        "The operation partially completed. Identify only confirmed successful and failed parts, and keep unresolved parts explicit. Never repeat the whole batch or assume failed parts had no effects.",
      ]);
    else if (execution?.reason === "query-result-error")
      addInstruction([
        "HTTP 200 acknowledges the Query RPC, not every statement. Explain the reported statement error and any verified successful results; earlier statements may already have had effects. Do not repeat the SQL or claim rollback.",
      ]);
    else if (execution?.effects === "possible")
      addInstruction([
        "This attempt's effects have not been ruled out. A later successful attempt does not verify or undo those effects.",
        "Use permitted read-only verification within scope before another potentially mutating attempt.",
      ]);
  }
  return failed
    ? [
        diagnostic
          ? diagnosticFallbackInstruction
          : apiResponseFallbackInstruction,
        ...guidance,
      ].join("\n\n")
    : "";
}
