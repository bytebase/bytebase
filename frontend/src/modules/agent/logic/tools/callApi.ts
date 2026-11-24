import { create, fromJson } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { refreshTokens } from "@/api/refreshToken";
import { getEffectiveQueryDataPolicy } from "@/lib/queryDataPolicy";
import { useAppStore } from "@/stores/app";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { QueryRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  type AgentApiDispatchContext,
  fingerprintPreparedApiCall,
} from "../apiAttempt";
import {
  type AgentApiTransportObservation,
  classifyAgentApiResponse,
  isProtectedAgentApiCall,
} from "../apiExecution";
import { handleAgentApiResponse } from "../apiResponse";
import { notSentApiResult } from "../callApiResult";
import {
  type AgentExecutionContext,
  isAgentExecutionContextCurrent,
} from "../executionContext";
import { boundAgentQueryResult } from "../queryResultRetention";
import { withAgentApiActionLease } from "../runOwnership";
import type { PropertyInfo, SchemaInfo } from "./gen/openapi-index";
import { getEndpointPath, getRequestSchema, getSchema } from "./searchApi";

const baseAddress =
  import.meta.env.BB_GRPC_LOCAL ||
  (typeof window === "undefined" ? "" : window.location.origin);
const REFRESH_PATH = "/bytebase.v1.AuthService/Refresh";
export const DEFAULT_AGENT_QUERY_LIMIT = 100;

export interface CallApiArgs {
  operationId: string;
  body?: Record<string, unknown>;
}

export interface PreparedApiCall {
  readonly operationId: string;
  readonly path: string;
  readonly body: string;
}

export type PrepareApiCallResult =
  | { kind: "prepared"; call: PreparedApiCall }
  | { kind: "error"; result: string };

interface ParsedApiResponse {
  status?: number;
  response?: unknown;
  error?: string;
}

interface ObservedApiResponse {
  parsed: ParsedApiResponse;
  observation: AgentApiTransportObservation;
}

const textEncoder = new TextEncoder();

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

const encodeUtf8ToBase64 = (value: string): string => {
  const bytes = textEncoder.encode(value);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
};

const getArrayItemSchema = (
  property: PropertyInfo
): PropertyInfo | undefined => {
  if (property.items) {
    return property.items;
  }

  if (property.type.startsWith("array<") && property.type.endsWith(">")) {
    return {
      name: "item",
      type: property.type.slice(6, -1),
    };
  }

  return undefined;
};

const coerceRequestValue = (
  value: unknown,
  property?: PropertyInfo,
  schema?: SchemaInfo
): unknown => {
  if (property?.type === "string" && property.format === "byte") {
    return typeof value === "string" ? encodeUtf8ToBase64(value) : value;
  }

  const resolvedSchema =
    schema ??
    (property && !property.type.startsWith("array<")
      ? getSchema(property.type)
      : undefined);

  if (Array.isArray(value)) {
    const itemSchema = property ? getArrayItemSchema(property) : undefined;
    return itemSchema
      ? value.map((item) => coerceRequestValue(item, itemSchema))
      : value;
  }

  if (!isPlainObject(value)) {
    return value;
  }

  if (property?.additionalProperties) {
    return Object.fromEntries(
      Object.entries(value).map(([key, itemValue]) => [
        key,
        coerceRequestValue(itemValue, property.additionalProperties),
      ])
    );
  }

  if (resolvedSchema?.type !== "object" || !resolvedSchema.properties) {
    return value;
  }

  const propertiesByName = new Map(
    resolvedSchema.properties.map((prop) => [prop.name, prop])
  );

  return Object.fromEntries(
    Object.entries(value).map(([key, itemValue]) => {
      const childProperty = propertiesByName.get(key);
      return [key, coerceRequestValue(itemValue, childProperty)];
    })
  );
};

const coerceRequestBody = (
  body: Record<string, unknown>,
  schema?: SchemaInfo
): unknown => coerceRequestValue(body, undefined, schema);

export const __testOnly = {
  coerceRequestBody,
};

const parseResponse = async (
  response: Response
): Promise<ObservedApiResponse> => {
  const status = response.status;
  if (
    response.redirected ||
    response.type === "opaqueredirect" ||
    (status >= 300 && status < 400)
  ) {
    return {
      parsed: { status, response: null },
      observation: { kind: "unexpected-redirect", status },
    };
  }

  let respJSON: unknown = null;
  let observation: AgentApiTransportObservation;
  try {
    const body = await response.text();
    respJSON = JSON.parse(body);
    observation = {
      kind: "response",
      status,
      body,
      contentType: response.headers.get("Content-Type"),
    };
  } catch {
    observation = { kind: "response-body-unreadable", status };
  }

  const parsed: ParsedApiResponse = { status, response: respJSON };
  if (status >= 400) {
    parsed.error = `HTTP ${status}`;
    if (isPlainObject(respJSON)) {
      if (typeof respJSON.message === "string") parsed.error = respJSON.message;
      else if (typeof respJSON.code === "string") parsed.error = respJSON.code;
    }
  }

  return { parsed, observation };
};

const fetchApi = async ({
  path,
  body,
  signal,
}: {
  path: string;
  body: string;
  signal: AbortSignal;
}) => {
  const response = await fetch(`${baseAddress}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Connect-Protocol-Version": "1",
    },
    credentials: "include",
    redirect: "manual",
    body,
    signal,
  });
  return parseResponse(response);
};

const getQueryRequestLimit = async (
  body: Record<string, unknown>
): Promise<number> => {
  if (
    body.limit != null &&
    typeof body.limit !== "number" &&
    typeof body.limit !== "string"
  ) {
    throw new Error("Query limit must be a 32-bit integer.");
  }
  const limit =
    body.limit == null
      ? undefined
      : fromJson(QueryRequestSchema, { limit: body.limit }).limit;
  const requestedLimit = limit ?? DEFAULT_AGENT_QUERY_LIMIT;
  if (
    typeof body.name !== "string" ||
    !/^(?:projects\/[^/]+\/)?instances\/[^/]+\/databases\/[^/]+$/.test(
      body.name
    )
  ) {
    return requestedLimit;
  }

  const store = useAppStore.getState();
  if (!(await store.loadSubscription())) {
    throw new Error("Cannot determine the query policy subscription status.");
  }
  if (!useAppStore.getState().hasFeature(PlanFeature.FEATURE_QUERY_POLICY)) {
    return requestedLimit;
  }
  const workspace = store.workspaceResourceName();
  if (!workspace) throw new Error("Cannot determine the query workspace.");
  const database = await databaseServiceClientConnect.getDatabase(
    create(GetDatabaseRequestSchema, { name: body.name }),
    { contextValues: createContextValues().set(silentContextKey, true) }
  );
  if (
    database.name !== body.name ||
    !/^projects\/[^/]+$/.test(database.project) ||
    database.project === "projects/-"
  ) {
    throw new Error("Cannot determine the target database project.");
  }
  const policies = await Promise.all(
    [workspace, database.project].map((parentPath) =>
      store.fetchPolicyByParentAndType({
        parentPath,
        policyType: PolicyType.DATA_QUERY,
        refresh: true,
      })
    )
  );
  if (policies.some((policy) => policy === undefined)) {
    throw new Error("Cannot load the target database query policies.");
  }
  const [workspacePolicy, projectPolicy] = policies.map((policy) =>
    policy?.policy.case === "queryDataPolicy" ? policy.policy.value : undefined
  );
  const { maximumResultRows } = getEffectiveQueryDataPolicy(
    workspacePolicy,
    projectPolicy
  );
  if (maximumResultRows === Number.MAX_VALUE) return requestedLimit;
  return limit == null || limit <= 0
    ? maximumResultRows
    : Math.min(limit, maximumResultRows);
};

export async function prepareApiCall(
  args: CallApiArgs
): Promise<PrepareApiCallResult> {
  const operationId = args.operationId;
  if (typeof operationId !== "string" || !operationId) {
    return {
      kind: "error",
      result: notSentApiResult(
        "operationId is required. Use search_api to find valid operations."
      ),
    };
  }

  const path = getEndpointPath(operationId);
  if (!path) {
    return {
      kind: "error",
      result: notSentApiResult(
        `Unknown operation: ${operationId}. Use search_api to find valid operations.`
      ),
    };
  }

  const match = /^\/([^/]+)\/([^/]+)$/.exec(path);
  if (!match) {
    return {
      kind: "error",
      result: notSentApiResult(`Invalid endpoint: ${path}`),
    };
  }

  try {
    // Snapshot arguments before policy reads so the approved payload cannot drift.
    const body = JSON.parse(JSON.stringify(args.body ?? {}));
    const requestBody =
      path === "/bytebase.v1.SQLService/Query" && isPlainObject(body)
        ? { ...body, limit: await getQueryRequestLimit(body) }
        : body;
    return {
      kind: "prepared",
      call: Object.freeze({
        operationId: `${match[1]}.${match[2]}`,
        path,
        body: JSON.stringify(
          coerceRequestBody(requestBody, getRequestSchema(operationId))
        ),
      }),
    };
  } catch (err) {
    return {
      kind: "error",
      result: notSentApiResult(
        `Failed to prepare ${operationId}: ${err instanceof Error ? err.message : String(err)}`
      ),
    };
  }
}

interface ApiDispatchOptions {
  isScopeCurrent?: () => boolean;
  validateBeforeSend?: () => Promise<boolean>;
  signal?: AbortSignal;
  attemptId?: string;
  context?: AgentApiDispatchContext;
  dom?: { expectedContext: AgentExecutionContext; beforeSend: () => boolean };
}

async function tryRefresh(signal: AbortSignal): Promise<boolean> {
  let onAbort = () => {};
  const aborted = new Promise<false>((resolve) => {
    onAbort = () => resolve(false);
  });
  signal.addEventListener("abort", onAbort, { once: true });
  try {
    return await Promise.race([refreshTokens().then(() => true), aborted]);
  } catch {
    return false;
  } finally {
    signal.removeEventListener("abort", onAbort);
  }
}

export async function dispatchPreparedApiCall(
  call: PreparedApiCall,
  options: ApiDispatchOptions = {}
): Promise<string> {
  if (isProtectedAgentApiCall(call))
    return notSentApiResult(
      "A one-time API permission is required.",
      "admission-rejected"
    );
  return sendPreparedApiCall(Object.freeze({ ...call }), options);
}

// Only the resolver of a consumed, revalidated permission dispatches a protected call.
export async function dispatchApprovedApiCall(
  call: PreparedApiCall,
  options: ApiDispatchOptions = {}
): Promise<string> {
  const snapshot = Object.freeze({ ...call });
  const { context, signal } = options;
  const current = () =>
    options.isScopeCurrent?.() !== false && context?.isCurrent() !== false;
  const changed = () =>
    notSentApiResult(
      "Agent action ownership changed before dispatch.",
      "admission-rejected"
    );
  if (!current()) return changed();
  if (!context || signal?.aborted)
    return sendPreparedApiCall(snapshot, options);
  const fingerprint = await fingerprintPreparedApiCall(snapshot).catch(
    () => undefined
  );
  const dispatch = async () => {
    if (!current()) return changed();
    let attemptId: string | undefined;
    if (fingerprint && !signal?.aborted) {
      try {
        const {
          scopeKey,
          userName,
          chatId,
          runId,
          rootUserMessageId,
          toolCallId,
          logicalActionId,
        } = context;
        const receipt = await context.beginAgentApiAttempt({
          scopeKey,
          userName,
          chatId,
          runId,
          rootUserMessageId,
          toolCallId,
          logicalActionId,
          fingerprint,
          operationId: snapshot.operationId,
          path: snapshot.path,
        });
        if (receipt.kind === "blocked")
          return notSentApiResult(receipt.reason, "admission-rejected");
        attemptId = receipt.attemptId;
      } catch {
        // Lack of local bookkeeping does not invalidate the consumed permission.
      }
    }
    const result = current()
      ? await sendPreparedApiCall(snapshot, { ...options, attemptId })
      : notSentApiResult(
          "Agent action ownership changed before dispatch.",
          "admission-rejected",
          attemptId
        );
    if (!attemptId) return result;
    try {
      if (
        await context.settleAgentApiAttempt(
          attemptId,
          JSON.parse(result).agentExecution,
          result
        )
      )
        return result;
    } catch {
      // The observed result remains authoritative even if the receipt cannot be saved.
    }
    return JSON.stringify({
      ...JSON.parse(result),
      agentSettlementSaved: false,
    });
  };
  // The optional liveness lock must never cause a second POST, even if its promise rejects.
  let execution: Promise<string> | undefined;
  const once = () => (execution ??= dispatch());
  try {
    await withAgentApiActionLease(
      context.scopeKey,
      context.logicalActionId,
      once
    );
  } catch {
    // Some browsers do not provide usable Web Locks.
  }
  return once();
}

async function sendPreparedApiCall(
  call: PreparedApiCall,
  {
    signal,
    attemptId,
    context,
    isScopeCurrent,
    validateBeforeSend,
    dom,
  }: ApiDispatchOptions = {}
): Promise<string> {
  const checkContext = () => {
    if (signal?.aborted) {
      return notSentApiResult(
        "The request was canceled before dispatch.",
        "canceled-before-dispatch",
        attemptId
      );
    }
    if (isScopeCurrent?.() === false || context?.isCurrent() === false)
      return notSentApiResult(
        "Agent action ownership changed before dispatch.",
        "admission-rejected",
        attemptId
      );
    if (
      dom &&
      (!isAgentExecutionContextCurrent(dom.expectedContext, signal) ||
        !dom.beforeSend())
    )
      return JSON.stringify({
        called: false,
        ...JSON.parse(
          notSentApiResult(
            "Not called: The page action changed or expired.",
            "admission-rejected",
            attemptId
          )
        ),
      });
  };
  const validateScope = async () => {
    try {
      return (await validateBeforeSend?.()) === true;
    } catch {
      return false;
    }
  };
  let failure = checkContext();
  if (failure) return failure;
  if (validateBeforeSend) {
    const valid = await validateScope();
    failure = checkContext();
    if (failure) return failure;
    if (!valid)
      return notSentApiResult(
        "The API request scope could not be verified before dispatch.",
        "preflight-rejected",
        attemptId
      );
  }

  const controller = new AbortController();
  let abortKind: "canceled-after-dispatch" | "timed-out" | undefined;
  const abort = (kind: NonNullable<typeof abortKind>) => {
    if (controller.signal.aborted) return;
    abortKind = kind;
    controller.abort();
  };
  const onAbort = () => abort("canceled-after-dispatch");
  signal?.addEventListener("abort", onAbort, { once: true });
  const timeoutId = setTimeout(() => abort("timed-out"), 30_000);

  const send = () =>
    fetchApi({ path: call.path, body: call.body, signal: controller.signal });
  let result: ObservedApiResponse;

  try {
    result = await send();
    if (
      result.parsed.status === 401 &&
      call.path !== REFRESH_PATH &&
      !dom &&
      !isProtectedAgentApiCall(call) &&
      !controller.signal.aborted &&
      (await tryRefresh(controller.signal)) &&
      !controller.signal.aborted &&
      !checkContext()
    ) {
      const valid = !validateBeforeSend || (await validateScope());
      if (valid && !controller.signal.aborted && !checkContext())
        result = await send();
    }
  } catch (err) {
    result = {
      parsed: {
        error: `Failed to call ${call.operationId}: ${err instanceof Error ? err.message : String(err)}`,
      },
      observation: { kind: abortKind ?? "fetch-failed" },
    };
  } finally {
    clearTimeout(timeoutId);
    signal?.removeEventListener("abort", onAbort);
  }

  let body: unknown;
  try {
    body = JSON.parse(call.body);
  } catch {
    // Missing request context must not discard an observed API outcome.
  }
  const raw = handleAgentApiResponse(
    { operationId: call.operationId, body },
    JSON.stringify({
      ...result.parsed,
      agentExecution: classifyAgentApiResponse(call, {
        ...result.observation,
        attemptId,
      }),
    })
  );
  return call.operationId === "bytebase.v1.SQLService.Query" &&
    call.path === "/bytebase.v1.SQLService/Query"
    ? boundAgentQueryResult(raw)
    : raw;
}

export async function callApi(
  args: CallApiArgs,
  options?: ApiDispatchOptions
): Promise<string> {
  const result = await prepareApiCall(args);
  return result.kind === "error"
    ? result.result
    : dispatchPreparedApiCall(result.call, options);
}
