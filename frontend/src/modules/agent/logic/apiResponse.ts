import { Code, ConnectError } from "@connectrpc/connect";
import {
  codeToString,
  errorToJson,
} from "@connectrpc/connect/protocol-connect";
import { isQueryAccessGrantPermission } from "@/modules/sql-editor/model/queryAccessRequest";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import { PlanFeatureSchema } from "@/types/proto-es/v1/subscription_service_pb";
import { parseStringToResource } from "@/utils/v1/databaseResource";
import { isAgentApiEvidenceForEnvelope } from "./apiExecution";
import { parseAgentRpcError } from "./apiRpcError";

export type DeniedQueryAccessRequest = {
  sourceDatabase: string;
  targets: string[];
  resources: string[];
  statement: string;
  requiredPermissions: string[];
  schema?: string;
  container?: string;
};

type PermissionDetails = { requiredPermissions: string[]; resources: string[] };

export type AgentApiRecovery = (
  | { kind: "api-error"; code?: Code }
  | { kind: "query-access-denied"; request: DeniedQueryAccessRequest }
  | ({ kind: "permission-denied" } & PermissionDetails)
  | {
      kind: "feature-required";
      feature: string;
      requiredPlan?: "TEAM" | "ENTERPRISE";
    }
  | { kind: "instance-activation-required"; feature: string; instance: string }
) & { source?: "api-client" | "preflight" };

export const MAX_STATEMENT_LENGTH = 100_000;
const DATABASE_NAME =
  /^(?:projects\/[^/]+\/)?instances\/[^/]+\/databases\/[^/]+$/;
const RESOURCE_ID = "[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?";
const INSTANCE_NAME = new RegExp(
  `^(?:(?:projects/${RESOURCE_ID}/)?instances/)?${RESOURCE_ID}$`
);

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

export const databaseFromResource = (resource: unknown): string | undefined => {
  if (typeof resource !== "string") return undefined;
  const database = parseStringToResource(resource)?.databaseFullName;
  return database &&
    DATABASE_NAME.test(database) &&
    !database.split("/").includes("-")
    ? database
    : undefined;
};

function combinePermissionDetails(
  details: unknown[]
): PermissionDetails | undefined {
  const isString = (value: unknown): value is string =>
    typeof value === "string" && value.trim() !== "";
  const requiredPermissions = new Set<string>();
  const resources = new Set<string>();
  for (const detail of details) {
    if (!isRecord(detail)) return undefined;
    const { requiredPermissions: permissions, resources: deniedResources } =
      detail;
    if (
      !Array.isArray(permissions) ||
      !permissions.length ||
      !permissions.every(isString) ||
      !Array.isArray(deniedResources) ||
      !deniedResources.every(isString)
    ) {
      return undefined;
    }
    for (const permission of permissions) requiredPermissions.add(permission);
    for (const resource of deniedResources) resources.add(resource);
  }
  return requiredPermissions.size
    ? {
        requiredPermissions: [...requiredPermissions],
        resources: [...resources],
      }
    : undefined;
}

function queryAccessRecovery(
  args: unknown,
  permission: PermissionDetails
): Extract<AgentApiRecovery, { kind: "query-access-denied" }> | undefined {
  if (
    !isRecord(args) ||
    (args.operationId !== "SQLService/Query" &&
      args.operationId !== "bytebase.v1.SQLService.Query") ||
    !isRecord(args.body)
  ) {
    return undefined;
  }
  const { name, statement, schema, container } = args.body;
  const sourceDatabase = databaseFromResource(name);
  if (
    !sourceDatabase ||
    sourceDatabase !== name ||
    typeof statement !== "string" ||
    !statement.trim() ||
    statement.length > MAX_STATEMENT_LENGTH ||
    !permission.requiredPermissions.every(isQueryAccessGrantPermission) ||
    !permission.resources.length
  ) {
    return undefined;
  }
  const targets = new Set<string>();
  for (const resource of permission.resources) {
    const target = databaseFromResource(resource);
    if (!target) return undefined;
    targets.add(target);
  }
  return {
    kind: "query-access-denied",
    request: {
      ...permission,
      sourceDatabase,
      targets: [...targets],
      statement,
      ...(typeof schema === "string" && schema ? { schema } : {}),
      ...(typeof container === "string" && container ? { container } : {}),
    },
  };
}

function queryResultRecovery(args: unknown, response: Record<string, unknown>) {
  if (
    !isRecord(args) ||
    (args.operationId !== "SQLService/Query" &&
      args.operationId !== "bytebase.v1.SQLService.Query")
  )
    return undefined;
  const { results } = response;
  if (!Array.isArray(results) || !results.length) return undefined;
  const details: unknown[] = [];
  for (const entry of results) {
    if (
      !isRecord(entry) ||
      typeof entry.error !== "string" ||
      !entry.error.trim() ||
      !isRecord(entry.permissionDenied) ||
      !Array.isArray(entry.permissionDenied.resources) ||
      !entry.permissionDenied.resources.length ||
      !entry.permissionDenied.resources.every((resource) =>
        databaseFromResource(resource)
      )
    ) {
      return undefined;
    }
    details.push(entry.permissionDenied);
  }
  const permission = combinePermissionDetails(details);
  return permission && permissionRecovery(args, permission);
}

const knownFeatures = new Set(
  PlanFeatureSchema.values
    .filter((feature) => feature.number !== 0)
    .map((feature) => feature.name)
);

function licenseRecovery(message: string): AgentApiRecovery | undefined {
  // Recognized backend license errors describe this failed call, not the
  // caller's current subscription.
  const paid =
    /^feature (FEATURE_[A-Z0-9_]+) is a (TEAM|ENTERPRISE) feature, please upgrade to access it$/.exec(
      message
    );
  if (paid?.[0] === message && knownFeatures.has(paid[1])) {
    return {
      kind: "feature-required",
      feature: paid[1],
      requiredPlan: paid[2] === "TEAM" ? "TEAM" : "ENTERPRISE",
    };
  }
  const activation =
    /^feature "(FEATURE_[A-Z0-9_]+)" requires an activated instance under the current license: (\S+)$/.exec(
      message
    );
  if (
    activation?.[0] === message &&
    knownFeatures.has(activation[1]) &&
    INSTANCE_NAME.test(activation[2])
  ) {
    return {
      kind: "instance-activation-required",
      feature: activation[1],
      instance: activation[2],
    };
  }
  if (
    message ===
    "SharePoint export requires the FEATURE_SHAREPOINT_EXPORT feature"
  ) {
    return { kind: "feature-required", feature: "FEATURE_SHAREPOINT_EXPORT" };
  }
  if (
    message ===
    "role request requires approval workflow feature (available in Enterprise plan)"
  ) {
    return {
      kind: "feature-required",
      feature: "FEATURE_REQUEST_ROLE_WORKFLOW",
      requiredPlan: "ENTERPRISE",
    };
  }
  return undefined;
}

function parseEnvelope(result: string): Record<string, unknown> | undefined {
  try {
    const parsed: unknown = JSON.parse(result);
    return isRecord(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

function permissionRecovery(
  args: unknown,
  details: PermissionDetails,
  completeScope = true
): AgentApiRecovery {
  return (
    (completeScope ? queryAccessRecovery(args, details) : undefined) ?? {
      kind: "permission-denied",
      ...details,
    }
  );
}

function classifyApiRecovery(
  args: unknown,
  envelope: Record<string, unknown>
): AgentApiRecovery | undefined {
  const { status, response, agentExecution, preflightStatus, preflight } =
    envelope;
  if (
    agentExecution !== undefined &&
    !isAgentApiEvidenceForEnvelope(
      agentExecution,
      envelope,
      isRecord(args) && typeof args.operationId === "string"
        ? args.operationId
        : undefined
    )
  )
    return undefined;
  const reason = isRecord(agentExecution) ? agentExecution.reason : undefined;

  if (
    status === undefined &&
    response === undefined &&
    agentExecution === undefined &&
    isRecord(envelope.apiError) &&
    typeof envelope.error === "string" &&
    envelope.error.trim()
  ) {
    const { apiError } = envelope;
    if (apiError.source === "client" && apiError.code === undefined)
      return { kind: "api-error", source: "api-client" };
    if (apiError.source !== "connect") return undefined;
    const rpcError = parseAgentRpcError(JSON.stringify(apiError));
    return rpcError
      ? { ...rpcErrorRecovery(args, apiError, rpcError), source: "api-client" }
      : undefined;
  }

  if (
    status === undefined &&
    response === undefined &&
    reason === "preflight-rejected" &&
    preflightStatus === 403 &&
    isRecord(preflight) &&
    preflight.code === "permission_denied"
  ) {
    const details = combinePermissionDetails([preflight]);
    return (
      details && {
        ...permissionRecovery(args, details),
        source: "preflight",
      }
    );
  }

  if (!isRecord(response)) return undefined;
  if (
    status === 200 &&
    (agentExecution === undefined || reason === "query-result-error")
  ) {
    return queryResultRecovery(args, response);
  }
  // A parsed envelope cannot recheck duplicate JSON fields from the wire body.
  if (
    typeof status !== "number" ||
    status < 400 ||
    status > 599 ||
    (agentExecution !== undefined &&
      reason !== "rpc-error" &&
      reason !== "rpc-rejected-before-effects")
  ) {
    return undefined;
  }
  const rpcError = parseAgentRpcError(JSON.stringify(response), status);
  if (!rpcError) return undefined;
  return rpcErrorRecovery(args, response, rpcError);
}

function rpcErrorRecovery(
  args: unknown,
  response: Record<string, unknown>,
  rpcError: ConnectError
): AgentApiRecovery {
  if (rpcError.code !== Code.PermissionDenied)
    return { kind: "api-error", code: rpcError.code };

  const details = rpcError.details.length
    ? rpcError.findDetails(PermissionDeniedDetailSchema)
    : [response];
  const completeDetails =
    rpcError.details.length === 0 ||
    rpcError.details.filter(
      (detail) =>
        "type" in detail &&
        detail.type === PermissionDeniedDetailSchema.typeName
    ).length === details.length;
  const permission = combinePermissionDetails(details);
  return permission
    ? permissionRecovery(
        args,
        permission,
        completeDetails &&
          details.every(
            (detail) =>
              isRecord(detail) &&
              Array.isArray(detail.resources) &&
              detail.resources.length > 0
          )
      )
    : (licenseRecovery(rpcError.rawMessage) ?? {
        kind: "api-error",
        code: rpcError.code,
      });
}

export function getAgentApiRecovery(
  args: unknown,
  result: string
): AgentApiRecovery | undefined {
  const envelope = parseEnvelope(result);
  return envelope && classifyApiRecovery(args, envelope);
}

export function extractDeniedQueryAccessRequest(
  args: unknown,
  result: string
): DeniedQueryAccessRequest | undefined {
  const recovery = getAgentApiRecovery(args, result);
  return recovery?.kind === "query-access-denied"
    ? recovery.request
    : undefined;
}

export function handleAgentApiResponse(args: unknown, result: string): string {
  const envelope = parseEnvelope(result);
  const recovery = envelope && classifyApiRecovery(args, envelope);
  if (!recovery) return result;
  const apiRecovery =
    recovery.kind === "query-access-denied"
      ? {
          kind: recovery.kind,
          requiredPermissions: recovery.request.requiredPermissions,
          resources: recovery.request.resources,
          ...(recovery.source ? { source: recovery.source } : {}),
        }
      : recovery;
  return JSON.stringify({ ...envelope, apiRecovery });
}

export function agentApiErrorResult(
  operationId: string | undefined,
  error: unknown,
  fallback: string
): string {
  const args = operationId ? { operationId } : {};
  if (!(error instanceof ConnectError))
    return handleAgentApiResponse(
      args,
      JSON.stringify({ error: fallback, apiError: { source: "client" } })
    );
  const response = errorToJson(error, undefined);
  const recovery = rpcErrorRecovery(args, response, error);
  const permission =
    recovery.kind === "permission-denied"
      ? recovery
      : recovery.kind === "query-access-denied"
        ? recovery.request
        : undefined;
  return handleAgentApiResponse(
    args,
    JSON.stringify({
      error:
        permission || recovery.kind === "api-error"
          ? fallback
          : error.rawMessage,
      apiError: {
        source: "connect",
        code: codeToString(error.code),
        ...(permission
          ? {
              requiredPermissions: permission.requiredPermissions,
              resources: permission.resources,
            }
          : recovery.kind !== "api-error"
            ? { message: error.rawMessage }
            : {}),
      },
    })
  );
}
