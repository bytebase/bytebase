import { refreshTokens } from "@/api/refreshToken";
import type { PropertyInfo, SchemaInfo } from "./gen/openapi-index";
import { getEndpointPath, getRequestSchema, getSchema } from "./searchApi";

const baseAddress =
  import.meta.env.BB_GRPC_LOCAL ||
  (typeof window === "undefined" ? "" : window.location.origin);
const REFRESH_PATH = "/bytebase.v1.AuthService/Refresh";

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
  status: number;
  response: unknown;
  error?: string;
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
): Promise<ParsedApiResponse> => {
  let respJSON: unknown = null;
  try {
    respJSON = await response.json();
  } catch {
    // Not valid JSON — leave as null.
  }

  if (response.status >= 400) {
    let error = `HTTP ${response.status}`;
    if (respJSON && typeof respJSON === "object") {
      const errMap = respJSON as Record<string, unknown>;
      if (typeof errMap.message === "string") {
        error = errMap.message;
      } else if (typeof errMap.code === "string") {
        error = errMap.code;
      }
    }
    return {
      status: response.status,
      error,
      response: respJSON,
    };
  }

  return {
    status: response.status,
    response: respJSON,
  };
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
    body,
    signal,
  });
  return parseResponse(response);
};

export function prepareApiCall(args: CallApiArgs): PrepareApiCallResult {
  if (typeof args.operationId !== "string" || !args.operationId) {
    return {
      kind: "error",
      result: JSON.stringify({
        error:
          "operationId is required. Use search_api to find valid operations.",
      }),
    };
  }

  const path = getEndpointPath(args.operationId);
  if (!path) {
    return {
      kind: "error",
      result: JSON.stringify({
        error: `Unknown operation: ${args.operationId}. Use search_api to find valid operations.`,
      }),
    };
  }

  const match = /^\/([^/]+)\/([^/]+)$/.exec(path);
  if (!match) {
    return {
      kind: "error",
      result: JSON.stringify({ error: `Invalid endpoint: ${path}` }),
    };
  }

  try {
    return {
      kind: "prepared",
      call: Object.freeze({
        operationId: `${match[1]}.${match[2]}`,
        path,
        body: JSON.stringify(
          coerceRequestBody(args.body ?? {}, getRequestSchema(args.operationId))
        ),
      }),
    };
  } catch (err) {
    return {
      kind: "error",
      result: JSON.stringify({
        error: `Failed to prepare ${args.operationId}: ${err instanceof Error ? err.message : String(err)}`,
      }),
    };
  }
}

export async function dispatchPreparedApiCall(
  call: PreparedApiCall
): Promise<string> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 30_000);

  try {
    let result = await fetchApi({
      path: call.path,
      body: call.body,
      signal: controller.signal,
    });

    if (result.status === 401 && call.path !== REFRESH_PATH) {
      try {
        await refreshTokens();
        result = await fetchApi({
          path: call.path,
          body: call.body,
          signal: controller.signal,
        });
      } catch {
        // Preserve the original API error shape when refresh fails.
      }
    }

    return JSON.stringify(result);
  } catch (err) {
    return JSON.stringify({
      error: `Failed to call ${call.operationId}: ${err instanceof Error ? err.message : String(err)}`,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function callApi(args: CallApiArgs): Promise<string> {
  const result = prepareApiCall(args);
  return result.kind === "error"
    ? result.result
    : dispatchPreparedApiCall(result.call);
}
