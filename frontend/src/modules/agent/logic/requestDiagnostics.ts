import {
  type AIChatMessage,
  AIChatMessageRole,
  AIChatResponse_StopReason,
  type AIChatToolDefinition,
} from "@/types/proto-es/v1/ai_service_pb";
import type {
  AgentModelAttempt,
  AgentModelCallUsage,
  AgentRunCheckpoint,
} from "./types";

export interface AgentProviderAttemptDiagnostics
  extends Omit<AgentModelAttempt, "usage"> {
  phase: AgentRunCheckpoint["phase"];
  roundTripMs: number;
  request: ReturnType<typeof measureAgentRequest>;
  usage?: Pick<
    AgentModelCallUsage,
    "totalTokens" | "inputTokens" | "outputTokens" | "cachedInputTokens"
  >;
  stopReason?: ReturnType<typeof normalizeAgentStopReason>;
  errorCategory?: "provider-error" | "aborted";
}

export function normalizeAgentStopReason(
  reason: AIChatResponse_StopReason | undefined
) {
  switch (reason) {
    case AIChatResponse_StopReason.TURN_COMPLETED:
      return "turn-completed";
    case AIChatResponse_StopReason.TOKEN_LIMIT:
      return "token-limit";
    case AIChatResponse_StopReason.BLOCKED:
      return "blocked";
    case AIChatResponse_StopReason.MALFORMED_TOOL_CALL:
      return "malformed-tool-call";
    case AIChatResponse_StopReason.OTHER:
      return "other";
    default:
      return "unknown";
  }
}

const jsonBytes = (value: unknown): number =>
  new TextEncoder().encode(JSON.stringify(value)).byteLength;

export const serializedAgentRequestBytes = (
  messages: AIChatMessage[],
  toolDefinitions: AIChatToolDefinition[]
): number => jsonBytes({ messages, toolDefinitions });

export const measureAgentRequest = (
  messages: AIChatMessage[],
  toolDefinitions: AIChatToolDefinition[]
) => ({
  serializedBytes: serializedAgentRequestBytes(messages, toolDefinitions),
  systemBytes: jsonBytes(
    messages.filter(
      (message) =>
        message.role === AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM
    )
  ),
  conversationBytes: jsonBytes(
    messages.filter(
      (message) =>
        message.role !== AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM
    )
  ),
  toolBytes: jsonBytes(toolDefinitions),
  messageCount: messages.length,
  toolCount: toolDefinitions.length,
});
