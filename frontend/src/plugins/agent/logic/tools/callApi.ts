import { refreshTokens } from "@/connect/refreshToken";
import { getEndpointPath } from "./searchApi";

const baseAddress = import.meta.env.BB_GRPC_LOCAL || window.location.origin;
const REFRESH_PATH = "/bytebase.v1.AuthService/Refresh";

export interface CallApiArgs {
  operationId: string;
  body?: Record<string, unknown>;
}

interface ParsedApiResponse {
  status: number;
  response: unknown;
  error?: string;
}

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

export async function callApi(args: CallApiArgs): Promise<string> {
  if (!args.operationId) {
    return JSON.stringify({
      error:
        "operationId is required. Use search_api to find valid operations.",
    });
  }

  const path = getEndpointPath(args.operationId);
  if (!path) {
    return JSON.stringify({
      error: `Unknown operation: ${args.operationId}. Use search_api to find valid operations.`,
    });
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 30_000);
  const body = JSON.stringify(args.body ?? {});

  try {
    let result = await fetchApi({
      path,
      body,
      signal: controller.signal,
    });

    if (result.status === 401 && path !== REFRESH_PATH) {
      try {
        await refreshTokens();
        result = await fetchApi({
          path,
          body,
          signal: controller.signal,
        });
      } catch {
        // Preserve the original API error shape when refresh fails.
      }
    }

    return JSON.stringify(result);
  } catch (err) {
    return JSON.stringify({
      error: `Failed to call ${args.operationId}: ${err instanceof Error ? err.message : String(err)}`,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}
