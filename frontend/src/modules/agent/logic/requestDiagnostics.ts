import {
  type AIChatMessage,
  AIChatMessageRole,
  type AIChatToolDefinition,
} from "@/types/proto-es/v1/ai_service_pb";

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
