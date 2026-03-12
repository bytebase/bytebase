import { create } from "@bufbuild/protobuf";
import { aiServiceClientConnect } from "@/connect";
import type {
  AIChatMessage,
  AIChatToolDefinition,
} from "@/types/proto-es/v1/ai_service_pb";
import {
  AIChatMessageRole,
  AIChatMessageSchema,
  AIChatToolDefinitionSchema,
} from "@/types/proto-es/v1/ai_service_pb";
import type { Message, ToolCall, ToolDefinition, ToolExecutor } from "./types";

const MAX_ITERATIONS = 10;

const ROLE_MAP: Record<Message["role"], AIChatMessageRole> = {
  system: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_SYSTEM,
  user: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_USER,
  assistant: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_ASSISTANT,
  tool: AIChatMessageRole.AI_CHAT_MESSAGE_ROLE_TOOL,
};

function roleToProto(role: Message["role"]): AIChatMessageRole {
  return ROLE_MAP[role];
}

function messageToProto(msg: Message): AIChatMessage {
  const proto = create(AIChatMessageSchema);
  proto.role = roleToProto(msg.role);
  proto.content = msg.content;
  proto.toolCallId = msg.toolCallId;
  if (msg.toolCalls) {
    proto.toolCalls = msg.toolCalls.map((tc) => ({
      $typeName: "bytebase.v1.AIChatToolCall" as const,
      id: tc.id,
      name: tc.name,
      arguments: tc.arguments,
      metadata: tc.metadata,
    }));
  }
  return proto;
}

function toolDefToProto(tool: ToolDefinition): AIChatToolDefinition {
  const proto = create(AIChatToolDefinitionSchema);
  proto.name = tool.name;
  proto.description = tool.description;
  proto.parametersSchema = JSON.stringify(tool.parametersSchema);
  return proto;
}

export interface AgentCallbacks {
  onToolCall?: (toolCall: ToolCall) => void;
  onToolResult?: (toolCallId: string, result: string) => void;
  onText?: (text: string) => void;
  onError?: (error: Error) => void;
}

export async function runAgentLoop(
  messages: Message[],
  tools: ToolDefinition[],
  executeTool: ToolExecutor,
  callbacks?: AgentCallbacks,
  signal?: AbortSignal
): Promise<string> {
  const conversation: Message[] = [...messages];
  const protoTools = tools.map(toolDefToProto);

  for (let i = 0; i < MAX_ITERATIONS; i++) {
    if (signal?.aborted) {
      throw new DOMException("Agent loop aborted", "AbortError");
    }

    const protoMessages = conversation.map(messageToProto);

    const response = await aiServiceClientConnect.chat(
      { messages: protoMessages, toolDefinitions: protoTools },
      { signal }
    );

    if (response.toolCalls.length > 0) {
      const toolCalls: ToolCall[] = response.toolCalls.map((tc) => ({
        id: tc.id,
        name: tc.name,
        arguments: tc.arguments,
        metadata: tc.metadata,
      }));

      // Append assistant message with tool calls
      conversation.push({
        role: "assistant",
        content: response.content,
        toolCalls,
      });

      // Execute each tool call and append results
      for (const tc of toolCalls) {
        callbacks?.onToolCall?.(tc);

        let result: string;
        try {
          const args = JSON.parse(tc.arguments) as Record<string, unknown>;
          result = await executeTool(tc.name, args);
        } catch (err) {
          const error = err instanceof Error ? err : new Error(String(err));
          callbacks?.onError?.(error);
          result = `Error: ${error.message}`;
        }

        callbacks?.onToolResult?.(tc.id, result);

        conversation.push({
          role: "tool",
          content: result,
          toolCallId: tc.id,
        });
      }

      continue;
    }

    // Text-only response — return final answer
    const text = response.content ?? "";
    callbacks?.onText?.(text);
    return text;
  }

  throw new Error("Agent loop exceeded maximum iterations");
}
