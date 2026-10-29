import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { aiServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import type {
  AIChatMessage,
  AIChatToolDefinition,
} from "@/types/proto-es/v1/ai_service_pb";
import {
  AIChatMessageRole,
  AIChatMessageSchema,
  AIChatToolDefinitionSchema,
} from "@/types/proto-es/v1/ai_service_pb";
import { parseCallApiResult } from "./callApiResult";
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
      if (signal?.aborted || lastError.name === "AbortError") throw lastError;
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

export interface AgentLoopOptions {
  allowPlainTextCompletion?: boolean;
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

function getCallApiOperationId(argumentsJson: string): string | undefined {
  try {
    const args: unknown = JSON.parse(argumentsJson);
    if (args && typeof args === "object" && !Array.isArray(args)) {
      const operationId = (args as Record<string, unknown>).operationId;
      return typeof operationId === "string" ? operationId : undefined;
    }
  } catch {
    return undefined;
  }
  return undefined;
}

interface ApiAttemptSummary {
  failedApiCallCount: number;
  lastApiError?: string;
}

function recordCallApiResult(
  summary: ApiAttemptSummary,
  toolCall: ToolCall,
  result: string
): void {
  const outcome = parseCallApiResult(
    result,
    getCallApiOperationId(toolCall.arguments)
  );
  if (outcome.error) {
    summary.failedApiCallCount++;
  }
  summary.lastApiError = outcome.error;
}

function getCurrentTurnApiAttempts(messages: Message[]): ApiAttemptSummary {
  const summary: ApiAttemptSummary = { failedApiCallCount: 0 };
  const userIndex = messages.findLastIndex(
    (message) => message.role === "user"
  );
  if (userIndex < 0) return summary;

  const toolCalls = new Map<string, ToolCall>();
  for (const message of messages.slice(userIndex + 1)) {
    if (message.role === "assistant") {
      for (const toolCall of message.toolCalls ?? []) {
        toolCalls.set(toolCall.id, toolCall);
      }
    } else if (message.role === "tool" && message.toolCallId) {
      const toolCall = toolCalls.get(message.toolCallId);
      if (toolCall?.name === "call_api") {
        recordCallApiResult(summary, toolCall, message.content ?? "");
      }
    }
  }
  return summary;
}

export async function runAgentLoop(
  messages: Message[],
  tools: ToolDefinition[],
  executeTool: ToolExecutor,
  callbacks?: AgentCallbacks,
  signal?: AbortSignal,
  options: AgentLoopOptions = {}
): Promise<AgentLoopOutcome> {
  const conversation: Message[] = [...messages];
  const protoTools = tools.map(toolDefToProto);
  const terminalTools = protoTools.filter(
    (tool) => tool.name === "done" || tool.name === "ask_user"
  );
  let totalTokensUsed = 0;
  let awaitingTerminalCall = false;
  const apiAttempts = getCurrentTurnApiAttempts(messages);

  const completeWithApiError = (error: string): AgentLoopOutcome => {
    callbacks?.onText?.(error);
    return { kind: "completed", text: error, success: false, totalTokensUsed };
  };

  try {
    for (let i = 0; i < MAX_ITERATIONS; i++) {
      if (signal?.aborted) {
        return { kind: "aborted", totalTokensUsed };
      }

      const protoMessages = conversation.map(messageToProto);

      const response = await callWithRetry(
        () =>
          aiServiceClientConnect.chat(
            {
              messages: protoMessages,
              toolDefinitions: awaitingTerminalCall
                ? terminalTools
                : protoTools,
            },
            {
              signal,
              contextValues: createContextValues().set(silentContextKey, true),
            }
          ),
        signal
      );

      if (signal?.aborted) {
        return { kind: "aborted", totalTokensUsed };
      }

      totalTokensUsed += response.usage?.totalTokens ?? 0;

      if (response.toolCalls.length > 0) {
        if (
          awaitingTerminalCall &&
          response.toolCalls.some(
            (call) => call.name !== "done" && call.name !== "ask_user"
          )
        ) {
          const error = new Error(
            "Agent called a non-terminal tool while finalizing its answer."
          );
          callbacks?.onError?.(error);
          return { kind: "error", error, totalTokensUsed };
        }
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
          if (signal?.aborted) {
            return { kind: "aborted", totalTokensUsed };
          }
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
            if (signal?.aborted) {
              return { kind: "aborted", totalTokensUsed };
            }
            const error = err instanceof Error ? err : new Error(String(err));
            callbacks?.onError?.(error);
            executionResult = {
              kind: "tool_result" as const,
              result: `Error: ${error.message}`,
            };
          }

          if (signal?.aborted) {
            return { kind: "aborted", totalTokensUsed };
          }

          if (executionResult.kind === "tool_result") {
            if (tc.name === "call_api") {
              recordCallApiResult(apiAttempts, tc, executionResult.result);
            }
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

          const { text, success } = executionResult;
          const doneResult = JSON.stringify({
            text,
            success,
            ...(success && apiAttempts.failedApiCallCount > 0
              ? { failedApiCallCount: apiAttempts.failedApiCallCount }
              : {}),
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
              text,
              success,
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

      const text = response.content?.trim() ?? "";
      if (options.allowPlainTextCompletion && text) {
        const apiError = apiAttempts.lastApiError;
        if (apiError) {
          return completeWithApiError(apiError);
        }
        callbacks?.onText?.(text);
        return {
          kind: "completed",
          text,
          success: true,
          totalTokensUsed,
        };
      }

      if (
        !awaitingTerminalCall &&
        terminalTools.some((tool) => tool.name === "done")
      ) {
        if (text) {
          conversation.push({ role: "assistant", content: response.content });
        }
        conversation.push({
          role: "user",
          content: text
            ? "Finalize your preceding answer by calling done({ text, success }), or call ask_user if input is required. Do not call any other tools or start another task."
            : "Your previous response was empty. Answer the user's request by calling done({ text, success }), or call ask_user if input is required. Do not call any other tools or start another task.",
        });
        awaitingTerminalCall = true;
        continue;
      }

      const apiError = apiAttempts.lastApiError;
      if (apiError && (awaitingTerminalCall || !text)) {
        return completeWithApiError(apiError);
      }

      const error = new Error(
        "Agent must finish with done({ text, success }) or ask_user(...), not plain assistant text."
      );
      callbacks?.onError?.(error);
      return {
        kind: "error",
        error,
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
    if (signal?.aborted || error.name === "AbortError") {
      return { kind: "aborted", totalTokensUsed };
    }
    return {
      kind: "error",
      error,
      totalTokensUsed,
    };
  }
}
