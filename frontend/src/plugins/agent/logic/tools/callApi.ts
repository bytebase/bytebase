import { getApiOperations } from "./searchApi";

export async function callApi(args: {
  operation_id: string;
  body?: Record<string, unknown>;
}): Promise<string> {
  const operations = getApiOperations();
  const operation = operations.find((op) => op.id === args.operation_id);
  if (!operation) {
    return JSON.stringify({
      error: `Unknown operation_id: ${args.operation_id}. Use search_api to find valid operations.`,
    });
  }

  try {
    // All Bytebase APIs use Connect protocol: POST with JSON body.
    const response = await fetch(`${window.location.origin}${operation.path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Connect-Protocol-Version": "1",
      },
      credentials: "include",
      body: JSON.stringify(args.body ?? {}),
    });

    const data = await response.json();

    if (!response.ok) {
      return JSON.stringify({
        error: `API returned ${response.status}`,
        details: data,
      });
    }

    return JSON.stringify(data);
  } catch (err) {
    return JSON.stringify({
      error: `Failed to call ${operation.id}: ${err instanceof Error ? err.message : String(err)}`,
    });
  }
}
