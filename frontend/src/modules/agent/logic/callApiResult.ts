import {
  type AgentApiExecutionEvidence,
  type AgentApiNotSentReason,
  agentApiExecutionEvidence,
  agentApiNeedsRecovery,
  isAgentApiEvidenceForEnvelope,
} from "./apiExecution";

export function notSentApiResult(
  error: string,
  reason: AgentApiNotSentReason = "preparation-failed",
  attemptId?: string
): string {
  return JSON.stringify({
    error,
    agentExecution: agentApiExecutionEvidence(reason, { attemptId }),
  });
}

export function preflightRejectedApiResult(result: string): string {
  const {
    error,
    status: preflightStatus,
    response: preflight,
  } = JSON.parse(result);
  return JSON.stringify({
    error,
    preflightStatus,
    preflight,
    agentExecution: agentApiExecutionEvidence("preflight-rejected"),
  });
}

function compatibilityFields(
  { status, error, response }: Record<string, unknown>,
  operationId?: string
): { error?: string; succeeded?: boolean } {
  if (typeof error === "string" && error.trim()) {
    return { error: error.trim() };
  }
  if (typeof status === "number" && status >= 200 && status < 400) {
    if (
      (operationId === "SQLService/Query" ||
        operationId === "bytebase.v1.SQLService.Query") &&
      response &&
      typeof response === "object" &&
      !Array.isArray(response)
    ) {
      const results = (response as Record<string, unknown>).results;
      if (Array.isArray(results)) {
        for (const queryResult of results) {
          if (
            queryResult &&
            typeof queryResult === "object" &&
            !Array.isArray(queryResult)
          ) {
            const queryError = (queryResult as Record<string, unknown>).error;
            if (typeof queryError === "string" && queryError.trim()) {
              return { error: queryError.trim() };
            }
          }
        }
      }
    }
    return { succeeded: true };
  }
  if (typeof status === "number" && status >= 400) {
    return { error: `HTTP ${status}` };
  }
  return {};
}

export function parseCallApiResult(
  result: string,
  operationId?: string
): {
  error?: string;
  succeeded?: boolean;
  execution?: AgentApiExecutionEvidence;
  requiresRecovery: boolean;
  settlementNotSaved?: boolean;
} {
  try {
    const parsed: unknown = JSON.parse(result);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      const envelope = parsed as Record<string, unknown>;
      const { agentExecution } = envelope;
      const compatibility = compatibilityFields(envelope, operationId);
      if (
        !isAgentApiEvidenceForEnvelope(agentExecution, envelope, operationId)
      ) {
        return {
          ...(agentExecution === undefined || compatibility.error
            ? compatibility
            : {}),
          requiresRecovery: agentApiNeedsRecovery(operationId),
        };
      }
      const requiresRecovery = agentApiNeedsRecovery(
        operationId,
        agentExecution
      );
      const fields = {
        execution: agentExecution,
        requiresRecovery,
        ...(agentExecution.attemptId &&
          envelope.agentSettlementSaved === false && {
            settlementNotSaved: true,
          }),
      };
      if (compatibility.error) return { ...fields, error: compatibility.error };
      switch (agentExecution.reason) {
        case "rpc-success":
          return { ...fields, succeeded: true };
        case "query-result-error":
          return { ...fields, error: "Query reported a statement error." };
        case "import-none-created":
          return { ...fields, error: "No instances were imported." };
        case "import-partial":
          return {
            ...fields,
            error: "The request confirmed only some imports.",
          };
        case "import-outcome-unknown":
          return {
            ...fields,
            error: "Instance import outcomes could not be verified.",
          };
        default:
          return fields;
      }
    }
  } catch {
    // Legacy or invalid content is not dispatch evidence.
  }
  return { requiresRecovery: agentApiNeedsRecovery(operationId) };
}
