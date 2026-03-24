import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { aiServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type {
  AIChatMessage,
  AIChatToolDefinition,
} from "@/types/proto-es/v1/ai_service_pb";
import {
  AIChatMessageRole,
  AIChatMessageSchema,
  AIChatToolDefinitionSchema,
} from "@/types/proto-es/v1/ai_service_pb";
import type {
  AgentLoopOutcome,
  Message,
  ToolCall,
  ToolDefinition,
  ToolExecutor,
} from "./types";

const MAX_ITERATIONS = 1000;
const MAX_RETRIES = 2;
const RETRY_DELAY_MS = 1000;

async function callWithRetry<T>(
  fn: () => Promise<T>,
  signal?: AbortSignal
): Promise<T> {
  let lastError: Error | undefined;
  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    if (signal?.aborted) {
      throw new DOMException("Agent loop aborted", "AbortError");
    }
    try {
      return await fn();
    } catch (err) {
      lastError = err instanceof Error ? err : new Error(String(err));
      if (lastError.name === "AbortError") throw lastError;
      const msg = lastError.message.toLowerCase();
      if (
        msg.includes("400") ||
        msg.includes("401") ||
        msg.includes("403") ||
        msg.includes("404")
      ) {
        throw lastError;
      }
      if (attempt < MAX_RETRIES) {
        await new Promise((r) => setTimeout(r, RETRY_DELAY_MS * (attempt + 1)));
      }
    }
  }
  throw lastError!;
}

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
  onAssistantMessage?: (
    message: Pick<Message, "content" | "toolCalls">
  ) => void;
  onToolResult?: (toolCallId: string, result: string) => void;
  onText?: (text: string) => void;
  onError?: (error: Error) => void;
}

function createSkippedToolResult(blockedBy: {
  toolCallId: string;
  kind: "ask_user" | "done";
}): string {
  return JSON.stringify({
    skipped: true,
    blockedByToolCallId: blockedBy.toolCallId,
    blockedByKind: blockedBy.kind,
    reason: `Skipped because this assistant turn already emitted ${blockedBy.kind}`,
  });
}

export async function runAgentLoop(
  messages: Message[],
  tools: ToolDefinition[],
  executeTool: ToolExecutor,
  callbacks?: AgentCallbacks,
  signal?: AbortSignal
): Promise<AgentLoopOutcome> {
  const conversation: Message[] = [...messages];
  const protoTools = tools.map(toolDefToProto);
  let totalTokensUsed = 0;

  try {
    for (let i = 0; i < MAX_ITERATIONS; i++) {
      if (signal?.aborted) {
        return { kind: "aborted", totalTokensUsed };
      }

      const protoMessages = conversation.map(messageToProto);

      const response = await callWithRetry(
        () =>
          aiServiceClientConnect.chat(
            { messages: protoMessages, toolDefinitions: protoTools },
            {
              signal,
              contextValues: createContextValues().set(silentContextKey, true),
            }
          ),
        signal
      );

      totalTokensUsed += response.usage?.totalTokens ?? 0;

      if (response.toolCalls.length > 0) {
        const toolCalls: ToolCall[] = response.toolCalls.map((tc) => ({
          id: tc.id,
          name: tc.name,
          arguments: tc.arguments,
          metadata: tc.metadata,
        }));

        const assistantMessage: Message = {
          role: "assistant",
          content: response.content,
          toolCalls,
        };
        conversation.push(assistantMessage);
        callbacks?.onAssistantMessage?.(assistantMessage);

        let terminalState:
          | {
              outcome: AgentLoopOutcome;
              blockedBy: { toolCallId: string; kind: "ask_user" | "done" };
            }
          | undefined;

        for (const tc of toolCalls) {
          if (terminalState) {
            const skippedResult = createSkippedToolResult(
              terminalState.blockedBy
            );
            callbacks?.onToolResult?.(tc.id, skippedResult);
            conversation.push({
              role: "tool",
              content: skippedResult,
              toolCallId: tc.id,
            });
            continue;
          }

          let executionResult;
          try {
            const args = JSON.parse(tc.arguments) as Record<string, unknown>;
            executionResult = await executeTool(tc.name, args, tc.id);
          } catch (err) {
            const error = err instanceof Error ? err : new Error(String(err));
            callbacks?.onError?.(error);
            executionResult = {
              kind: "tool_result" as const,
              result: `Error: ${error.message}`,
            };
          }

          if (executionResult.kind === "tool_result") {
            callbacks?.onToolResult?.(tc.id, executionResult.result);
            conversation.push({
              role: "tool",
              content: executionResult.result,
              toolCallId: tc.id,
            });
            continue;
          }

          if (executionResult.kind === "ask_user") {
            terminalState = {
              outcome: {
                kind: "awaiting_user",
                ask: executionResult.ask,
                totalTokensUsed,
              },
              blockedBy: {
                toolCallId: tc.id,
                kind: "ask_user",
              },
            };
            continue;
          }

          const doneResult = JSON.stringify({
            text: executionResult.text,
            success: executionResult.success,
          });
          callbacks?.onToolResult?.(tc.id, doneResult);
          conversation.push({
            role: "tool",
            content: doneResult,
            toolCallId: tc.id,
          });
          terminalState = {
            outcome: {
              kind: "completed",
              text: executionResult.text,
              success: executionResult.success,
              explicit: true,
              totalTokensUsed,
            },
            blockedBy: {
              toolCallId: tc.id,
              kind: "done",
            },
          };
        }

        if (terminalState) {
          if (terminalState.outcome.kind === "completed") {
            callbacks?.onText?.(terminalState.outcome.text);
          }
          return terminalState.outcome;
        }

        continue;
      }

      const text = response.content ?? "";
      callbacks?.onText?.(text);
      return {
        kind: "completed",
        text,
        success: true,
        explicit: false,
        totalTokensUsed,
      };
    }

    return {
      kind: "error",
      error: new Error("Agent loop exceeded maximum iterations"),
      totalTokensUsed,
    };
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    if (error.name === "AbortError") {
      return { kind: "aborted", totalTokensUsed };
    }
    return {
      kind: "error",
      error,
      totalTokensUsed,
    };
  }
}
