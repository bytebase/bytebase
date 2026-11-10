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
import {
  measureAgentRequest,
  serializedAgentRequestBytes,
} from "./requestDiagnostics";
import { getModelCallUsage } from "./tokenUsage";
import type {
  AgentHandoff,
  AgentLoopOutcome,
  AgentModelCallUsage,
  Message,
  ToolCall,
  ToolDefinition,
  ToolExecutor,
} from "./types";

const MAX_PROVIDER_ATTEMPTS_PER_RUN = 12;
const MAX_TOTAL_TOKENS_PER_RUN = 60_000;
const MAX_SERIALIZED_REQUEST_BYTES = 256 * 1024;
const MAX_RETRIES_PER_REQUEST = 2;
const RETRY_BASE_DELAY_MS = 1000;
const COMPACTION_NOTE =
  "Earlier tool calls and results were omitted to fit the context limit. Re-fetch any state you need and verify it before repeating an action.";

class ProviderAttemptLimitError extends Error {}

async function callWithRetry<T>(
  fn: () => Promise<T>,
  signal: AbortSignal | undefined,
  reserveProviderAttempt: () => boolean
): Promise<T> {
  let lastError: Error | undefined;
  for (let attempt = 0; attempt <= MAX_RETRIES_PER_REQUEST; attempt++) {
    if (signal?.aborted) {
      throw new DOMException("Agent loop aborted", "AbortError");
    }
    if (!reserveProviderAttempt()) {
      if (lastError) throw lastError;
      throw new ProviderAttemptLimitError(
        "Agent provider attempt limit reached."
      );
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
      if (attempt < MAX_RETRIES_PER_REQUEST) {
        await new Promise((r) =>
          setTimeout(r, RETRY_BASE_DELAY_MS * (attempt + 1))
        );
      }
    }
  }
  throw lastError!;
}

function removeOldestCompletedToolCallGroup(messages: Message[]): boolean {
  for (let index = 0; index < messages.length; index++) {
    const message = messages[index];
    if (message?.role !== "assistant" || !message.toolCalls?.length) continue;

    const callIds = new Set(message.toolCalls.map((toolCall) => toolCall.id));
    const groupEnd = messages.findIndex(
      (candidate, candidateIndex) =>
        candidateIndex > index && candidate.role === "assistant"
    );
    const end = groupEnd < 0 ? messages.length : groupEnd;
    const completedIds = new Set<string>();
    for (const candidate of messages.slice(index + 1, end)) {
      if (
        candidate.role === "tool" &&
        candidate.toolCallId &&
        callIds.has(candidate.toolCallId)
      ) {
        completedIds.add(candidate.toolCallId);
      }
    }
    if (![...callIds].every((id) => completedIds.has(id))) continue;

    for (let current = end - 1; current >= index; current--) {
      const candidate = messages[current];
      if (
        current === index ||
        (candidate?.role === "tool" &&
          candidate.toolCallId &&
          callIds.has(candidate.toolCallId))
      ) {
        messages.splice(current, 1);
      }
    }
    return true;
  }
  return false;
}

function composeRequestMessages(
  messages: Message[],
  guidance: string,
  compacted: boolean
): Message[] {
  const systems = messages.filter((message) => message.role === "system");
  const content = [
    ...systems.map((message) => message.content),
    guidance,
    compacted ? COMPACTION_NOTE : "",
  ]
    .filter(Boolean)
    .join("\n\n");
  if (!systems.length && !content) return messages;
  return [
    { role: "system", content },
    ...messages.filter((message) => message.role !== "system"),
  ];
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
  onText?: (text: string, handoff?: AgentHandoff) => void;
  onError?: (error: Error) => void;
  onUsage?: (usage: AgentModelCallUsage) => void;
  onRequestDiagnostics?: (
    diagnostics: ReturnType<typeof measureAgentRequest>
  ) => void;
}

export interface AgentLoopOptions {
  allowPlainTextCompletion?: boolean;
  resolveRequestContext?: () => {
    tools: ToolDefinition[];
    systemGuidance: string;
  };
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
  const terminalToolNames = new Set([
    "done",
    "ask_user",
    "ask_masking_exemption_scope",
    "ask_masking_rule_scope",
    "ask_masking_semantic_type",
  ]);
  let totalTokensUsed = 0;
  let providerAttemptCount = 0;
  let contextCompacted = false;
  let awaitingTerminalCall = false;
  const apiAttempts = getCurrentTurnApiAttempts(messages);

  const completeWithApiError = (error: string): AgentLoopOutcome => {
    callbacks?.onText?.(error);
    return { kind: "completed", text: error, success: false, totalTokensUsed };
  };

  const completeWithRunLimit = (): AgentLoopOutcome => {
    const text =
      "I stopped before making another AI request because this run reached its usage limit. Earlier actions may already have completed; check their status before asking me to retry.";
    callbacks?.onText?.(text);
    return { kind: "completed", text, success: false, totalTokensUsed };
  };

  const completeWithRequestLimit = (): AgentLoopOutcome => {
    const text =
      "This request is too large to send safely, even after removing earlier tool results. Start a new chat or shorten the request.";
    callbacks?.onText?.(text);
    return { kind: "completed", text, success: false, totalTokensUsed };
  };

  try {
    for (;;) {
      if (signal?.aborted) {
        return { kind: "aborted", totalTokensUsed };
      }
      if (
        totalTokensUsed >= MAX_TOTAL_TOKENS_PER_RUN ||
        providerAttemptCount >= MAX_PROVIDER_ATTEMPTS_PER_RUN
      ) {
        return completeWithRunLimit();
      }

      const requestContext = options.resolveRequestContext?.();
      const protoTools = (requestContext?.tools ?? tools).map(toolDefToProto);
      const terminalTools = protoTools.filter((tool) =>
        terminalToolNames.has(tool.name)
      );
      const toolDefinitions = awaitingTerminalCall ? terminalTools : protoTools;
      const requestMessages = () =>
        composeRequestMessages(
          conversation,
          requestContext?.systemGuidance ?? "",
          contextCompacted
        ).map(messageToProto);
      let protoMessages = requestMessages();
      while (
        serializedAgentRequestBytes(protoMessages, toolDefinitions) >
          MAX_SERIALIZED_REQUEST_BYTES &&
        removeOldestCompletedToolCallGroup(conversation)
      ) {
        contextCompacted = true;
        protoMessages = requestMessages();
      }
      if (
        serializedAgentRequestBytes(protoMessages, toolDefinitions) >
        MAX_SERIALIZED_REQUEST_BYTES
      ) {
        return completeWithRequestLimit();
      }

      const response = await callWithRetry(
        () => {
          callbacks?.onRequestDiagnostics?.(
            measureAgentRequest(protoMessages, toolDefinitions)
          );
          return aiServiceClientConnect.chat(
            {
              messages: protoMessages,
              toolDefinitions,
            },
            {
              signal,
              contextValues: createContextValues().set(silentContextKey, true),
            }
          );
        },
        signal,
        () => {
          if (providerAttemptCount >= MAX_PROVIDER_ATTEMPTS_PER_RUN)
            return false;
          providerAttemptCount++;
          return true;
        }
      );

      totalTokensUsed += response.usage?.totalTokens ?? 0;
      callbacks?.onUsage?.(getModelCallUsage("agent", response));

      if (signal?.aborted) {
        return { kind: "aborted", totalTokensUsed };
      }

      if (response.toolCalls.length > 0) {
        if (
          awaitingTerminalCall &&
          response.toolCalls.some((call) => !terminalToolNames.has(call.name))
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
          const handoff = success ? executionResult.handoff : undefined;
          const actions = success ? executionResult.actions : undefined;
          const doneResult = JSON.stringify({
            text,
            success,
            ...(handoff ? { handoff } : {}),
            ...(actions?.length ? { actions } : {}),
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
              ...(handoff ? { handoff } : {}),
              ...(actions?.length ? { actions } : {}),
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
            if (terminalState.outcome.handoff) {
              callbacks?.onText?.(
                terminalState.outcome.text,
                terminalState.outcome.handoff
              );
            } else {
              callbacks?.onText?.(terminalState.outcome.text);
            }
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
            ? "Finalize your preceding answer by calling done({ text, success, actions }), using actions: [] when none apply, or call ask_user, ask_masking_exemption_scope, ask_masking_rule_scope, or ask_masking_semantic_type if input is required. Do not call any other tools or start another task."
            : "Your previous response was empty. Answer the user's request by calling done({ text, success, actions }), using actions: [] when none apply, or call ask_user, ask_masking_exemption_scope, ask_masking_rule_scope, or ask_masking_semantic_type if input is required. Do not call any other tools or start another task.",
        });
        awaitingTerminalCall = true;
        continue;
      }

      const apiError = apiAttempts.lastApiError;
      if (apiError && (awaitingTerminalCall || !text)) {
        return completeWithApiError(apiError);
      }

      const error = new Error(
        "Agent must finish with done({ text, success, actions }), ask_user(...), ask_masking_exemption_scope(...), ask_masking_rule_scope(...), or ask_masking_semantic_type(...), not plain assistant text."
      );
      callbacks?.onError?.(error);
      return {
        kind: "error",
        error,
        totalTokensUsed,
      };
    }
  } catch (err) {
    const error = err instanceof Error ? err : new Error(String(err));
    if (signal?.aborted || error.name === "AbortError") {
      return { kind: "aborted", totalTokensUsed };
    }
    if (error instanceof ProviderAttemptLimitError) {
      return completeWithRunLimit();
    }
    return {
      kind: "error",
      error,
      totalTokensUsed,
    };
  }
}
