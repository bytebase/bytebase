import { type DescMethod, fromJsonString, isMessage } from "@bufbuild/protobuf";
import { Code } from "@connectrpc/connect";
import {
  codeFromString,
  codeToHttpStatus,
  parseContentType,
} from "@connectrpc/connect/protocol-connect";
import {
  ImportInstancesRequestSchema,
  type ImportInstancesResponse,
  ImportInstancesResponseSchema,
  InstanceImportFailure_CreationOutcome,
  InstanceService,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  QueryResponseSchema,
  SQLService,
} from "@/types/proto-es/v1/sql_service_pb";
import { protobufJsonRegistry } from "@/types/protobufJsonRegistry";
import { parseAgentRpcError } from "./apiRpcError";
import {
  apiCallNeedsConfirmation,
  getApiMethodDescriptor,
} from "./tools/apiMethodClass";
import type { PreparedApiCall } from "./tools/callApi";

export type AgentApiResult = "not-sent" | "succeeded" | "failed" | "unknown";
export type AgentApiEffects = "none" | "acknowledged" | "possible" | "partial";

const outcomes = {
  "preparation-failed": ["not-sent", "none"],
  "preflight-rejected": ["not-sent", "none"],
  "permission-rejected": ["not-sent", "none"],
  "admission-rejected": ["not-sent", "none"],
  "journal-conflict": ["unknown", "possible"],
  "journal-unreadable": ["unknown", "possible"],
  "legacy-interrupted": ["unknown", "possible"],
  "canceled-before-dispatch": ["not-sent", "none"],
  "fetch-failed": ["unknown", "possible"],
  "canceled-after-dispatch": ["unknown", "possible"],
  "timed-out": ["unknown", "possible"],
  "response-body-unreadable": ["unknown", "possible"],
  "unexpected-redirect": ["unknown", "possible"],
  "unverified-response": ["unknown", "possible"],
  "rpc-error": ["failed", "possible"],
  "rpc-rejected-before-effects": ["failed", "none"],
  "rpc-success": ["succeeded", "acknowledged"],
  "query-result-error": ["succeeded", "possible"],
  "import-none-created": ["succeeded", "none"],
  "import-partial": ["succeeded", "partial"],
  "import-outcome-unknown": ["succeeded", "possible"],
} as const satisfies Record<string, readonly [AgentApiResult, AgentApiEffects]>;

export type AgentApiExecutionReason = keyof typeof outcomes;
export type AgentApiNotSentReason = {
  [Reason in AgentApiExecutionReason]: (typeof outcomes)[Reason][0] extends "not-sent"
    ? Reason
    : never;
}[AgentApiExecutionReason];

interface ExecutionMetadata {
  attemptId?: string;
  httpStatus?: number;
  connectCode?: Code;
}

export interface AgentApiExecutionEvidence extends ExecutionMetadata {
  version: 1;
  result: AgentApiResult;
  effects: AgentApiEffects;
  reason: AgentApiExecutionReason;
}

export type AgentApiTransportObservation = { attemptId?: string } & (
  | {
      kind: "response";
      status: number;
      body: string;
      contentType: string | null;
    }
  | {
      kind: "response-body-unreadable" | "unexpected-redirect";
      status?: number;
    }
  | {
      kind:
        | "fetch-failed"
        | "canceled-after-dispatch"
        | "timed-out"
        | AgentApiNotSentReason;
    }
);

export function agentApiExecutionEvidence(
  reason: AgentApiExecutionReason,
  metadata?: ExecutionMetadata
): AgentApiExecutionEvidence {
  const [result, effects] = outcomes[reason];
  return { version: 1, result, effects, reason, ...metadata };
}

const isConnectCode = (value: unknown): value is Code =>
  typeof value === "number" && Object.hasOwn(Code, value);

export function isAgentApiExecutionEvidence(
  value: unknown
): value is AgentApiExecutionEvidence {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const {
    version,
    reason,
    result,
    effects,
    attemptId,
    httpStatus,
    connectCode,
  } = value as Partial<AgentApiExecutionEvidence>;
  if (
    version !== 1 ||
    typeof reason !== "string" ||
    !Object.hasOwn(outcomes, reason)
  )
    return false;
  const [expectedResult, expectedEffects] = outcomes[reason];
  return (
    result === expectedResult &&
    effects === expectedEffects &&
    (attemptId === undefined ||
      (typeof attemptId === "string" && attemptId.length > 0)) &&
    (httpStatus === undefined ||
      (Number.isInteger(httpStatus) &&
        httpStatus >= 100 &&
        httpStatus <= 599)) &&
    (connectCode === undefined || isConnectCode(connectCode))
  );
}

export function agentApiNeedsRecovery(
  operationId: string | undefined,
  execution?: AgentApiExecutionEvidence
): boolean {
  return (
    apiCallNeedsConfirmation(operationId ?? "") &&
    (!execution ||
      execution.effects === "possible" ||
      execution.effects === "partial")
  );
}

const getCallMethod = (call: Pick<PreparedApiCall, "operationId" | "path">) => {
  const method = getApiMethodDescriptor(call.operationId);
  return method && call.path === `/${method.parent.typeName}/${method.name}`
    ? method
    : undefined;
};

export const isProtectedAgentApiCall = (
  call: Pick<PreparedApiCall, "operationId" | "path">
) => !getCallMethod(call) || apiCallNeedsConfirmation(call.operationId);

const importPreCreationCodes = new Set([
  Code.InvalidArgument,
  Code.Unauthenticated,
  Code.PermissionDenied,
  Code.NotFound,
]);

const isImportPreCreationError = (method: DescMethod | undefined, code: Code) =>
  method === InstanceService.method.importInstances &&
  importPreCreationCodes.has(code);

export function isAgentApiEvidenceForOperation(
  evidence: unknown,
  status: unknown,
  operationId?: string
): evidence is AgentApiExecutionEvidence {
  if (!isAgentApiExecutionEvidence(evidence)) return false;
  if (evidence.httpStatus !== undefined && evidence.httpStatus !== status)
    return false;
  if (evidence.result === "not-sent")
    return status === undefined && evidence.connectCode === undefined;
  if (evidence.result === "unknown") return evidence.connectCode === undefined;

  const method = operationId ? getApiMethodDescriptor(operationId) : undefined;
  if (evidence.result === "failed") {
    const { connectCode } = evidence;
    if (connectCode === undefined || status !== codeToHttpStatus(connectCode))
      return false;
    return (
      evidence.reason === "rpc-error" ||
      isImportPreCreationError(method, connectCode)
    );
  }

  if (
    status !== 200 ||
    evidence.connectCode !== undefined ||
    (operationId !== undefined && method?.methodKind !== "unary")
  )
    return false;
  switch (evidence.reason) {
    case "query-result-error":
      return method === SQLService.method.query;
    case "import-none-created":
    case "import-partial":
    case "import-outcome-unknown":
      return method === InstanceService.method.importInstances;
    default:
      return true;
  }
}

export function isAgentApiEvidenceForEnvelope(
  evidence: unknown,
  { status, response }: Record<string, unknown>,
  operationId?: string
): evidence is AgentApiExecutionEvidence {
  return (
    isAgentApiEvidenceForOperation(evidence, status, operationId) &&
    (evidence.result !== "failed" ||
      Boolean(
        response &&
          typeof response === "object" &&
          "code" in response &&
          typeof response.code === "string" &&
          codeFromString(response.code) === evidence.connectCode
      ))
  );
}

function classifyImportResponse(
  body: string,
  response: ImportInstancesResponse
): AgentApiExecutionReason {
  const { parent, imports } = fromJsonString(
    ImportInstancesRequestSchema,
    body
  );
  if (!imports.length || imports.length !== response.results.length)
    return "unverified-response";

  let created = 0;
  let unknown = false;
  for (const [index, { outcome }] of response.results.entries()) {
    if (outcome.case === "imported") {
      const instanceId = imports[index].instanceId;
      const expected = `${parent ? `${parent}/` : ""}instances/${instanceId}`;
      if (!instanceId || outcome.value.name !== expected)
        return "unverified-response";
      created++;
    } else if (
      outcome.case === "failure" &&
      isConnectCode(outcome.value.status?.code)
    ) {
      const { creationOutcome } = outcome.value;
      if (creationOutcome === InstanceImportFailure_CreationOutcome.UNKNOWN) {
        unknown = true;
      } else if (
        creationOutcome !== InstanceImportFailure_CreationOutcome.NOT_CREATED
      ) {
        return "unverified-response";
      }
    } else {
      return "unverified-response";
    }
  }

  if (created === imports.length) return "rpc-success";
  if (created > 0) return "import-partial";
  return unknown ? "import-outcome-unknown" : "import-none-created";
}

export function classifyAgentApiResponse(
  call: PreparedApiCall,
  observation: AgentApiTransportObservation
): AgentApiExecutionEvidence {
  const metadata: ExecutionMetadata = {};
  if (observation.attemptId !== undefined)
    metadata.attemptId = observation.attemptId;
  if ("status" in observation && observation.status)
    metadata.httpStatus = observation.status;
  if (observation.kind !== "response")
    return agentApiExecutionEvidence(observation.kind, metadata);

  const { status, body, contentType } = observation;
  if (status >= 300 && status < 400)
    return agentApiExecutionEvidence("unexpected-redirect", metadata);
  const mediaType = parseContentType(contentType);
  if (!mediaType || mediaType.binary || mediaType.stream) {
    return agentApiExecutionEvidence("unverified-response", metadata);
  }
  const method = getCallMethod(call);
  if (status >= 400) {
    const code = parseAgentRpcError(body, status)?.code;
    if (code === undefined)
      return agentApiExecutionEvidence("unverified-response", metadata);
    metadata.connectCode = code;
    return agentApiExecutionEvidence(
      isImportPreCreationError(method, code)
        ? "rpc-rejected-before-effects"
        : "rpc-error",
      metadata
    );
  }

  if (status === 200 && method?.methodKind === "unary") {
    try {
      const decoded = fromJsonString(method.output, body, {
        registry: protobufJsonRegistry,
      });
      if (
        isMessage(decoded, QueryResponseSchema) &&
        decoded.results.some(
          (result) =>
            result.error !== "" || result.detailedError.case !== undefined
        )
      ) {
        // A statement denial doesn't establish that earlier statements had no effect.
        return agentApiExecutionEvidence("query-result-error", metadata);
      }
      return agentApiExecutionEvidence(
        isMessage(decoded, ImportInstancesResponseSchema)
          ? classifyImportResponse(call.body, decoded)
          : "rpc-success",
        metadata
      );
    } catch {
      // A schema mismatch is not evidence that the server failed to act.
    }
  }
  return agentApiExecutionEvidence("unverified-response", metadata);
}
