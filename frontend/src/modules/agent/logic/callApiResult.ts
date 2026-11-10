export function parseCallApiResult(
  result: string,
  operationId?: string
): { error?: string; succeeded?: boolean } {
  try {
    const parsed: unknown = JSON.parse(result);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return {};
    }
    const { status, error, response } = parsed as Record<string, unknown>;
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
  } catch {
    return {};
  }
}
