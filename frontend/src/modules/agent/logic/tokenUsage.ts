import type { AIChatResponse } from "@/types/proto-es/v1/ai_service_pb";
import type { AgentModelCallUsage } from "./types";

const tokenCountFields = [
  "totalTokens",
  "inputTokens",
  "outputTokens",
  "cachedInputTokens",
] as const;

function normalizeTokenCounts(
  raw: unknown
): Pick<AgentModelCallUsage, (typeof tokenCountFields)[number]> {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
  const counts: Pick<AgentModelCallUsage, (typeof tokenCountFields)[number]> =
    {};
  for (const field of tokenCountFields) {
    const value = (raw as Record<string, unknown>)[field];
    if (
      typeof value === "number" &&
      Number.isSafeInteger(value) &&
      value >= 0
    ) {
      counts[field] = value;
    }
  }
  return counts;
}

export function getModelCallUsage(
  purpose: AgentModelCallUsage["purpose"],
  response: Pick<AIChatResponse, "toolCalls" | "usage">
): AgentModelCallUsage {
  return {
    purpose,
    toolNames: response.toolCalls.map((call) => call.name),
    ...normalizeTokenCounts(response.usage),
  };
}

export function normalizeModelCallUsage(
  raw: unknown
): AgentModelCallUsage | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const value = raw as Record<string, unknown>;
  if (
    (value.purpose !== "agent" && value.purpose !== "follow-up") ||
    !Array.isArray(value.toolNames)
  ) {
    return undefined;
  }
  return {
    purpose: value.purpose,
    toolNames: value.toolNames.filter(
      (name): name is string => typeof name === "string"
    ),
    ...normalizeTokenCounts(value),
  };
}
