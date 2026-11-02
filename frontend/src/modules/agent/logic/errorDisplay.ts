const providerMessagePattern = /"message"\s*:\s*("(?:\\.|[^"\\])*")/g;
const maxSummaryLength = 240;

export function getAgentErrorSummary(error: string): string | undefined {
  for (const match of error.matchAll(providerMessagePattern)) {
    try {
      const message: unknown = JSON.parse(match[1]!);
      if (typeof message === "string" && message.trim()) {
        return truncateSummary(message.trim());
      }
    } catch {
      // Ignore malformed provider payloads and fall back to the original error.
    }
  }

  const firstLine = error
    .trim()
    .split(/\r?\n/, 1)[0]
    ?.replace(/^(?:Error|ConnectError):\s*/i, "");
  return firstLine ? truncateSummary(firstLine) : undefined;
}

function truncateSummary(summary: string): string {
  if (summary.length <= maxSummaryLength) return summary;
  return `${summary.slice(0, maxSummaryLength - 1).trimEnd()}…`;
}
