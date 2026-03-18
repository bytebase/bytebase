import { getEndpointPath } from "./searchApi";

const baseAddress = import.meta.env.BB_GRPC_LOCAL || window.location.origin;

export interface CallApiArgs {
  operationId: string;
  body?: Record<string, unknown>;
}

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

  try {
    const response = await fetch(`${baseAddress}${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
      },
      credentials: "include",
      body: JSON.stringify(args.body ?? {}),
      signal: controller.signal,
    });

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
      return JSON.stringify({
        status: response.status,
        error,
        response: respJSON,
      });
    }

    return JSON.stringify({
      status: response.status,
      response: respJSON,
    });
  } catch (err) {
    return JSON.stringify({
      error: `Failed to call ${args.operationId}: ${err instanceof Error ? err.message : String(err)}`,
    });
  } finally {
    clearTimeout(timeoutId);
  }
}
