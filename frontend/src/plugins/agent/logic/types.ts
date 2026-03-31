export interface ToolDefinition {
  name: string;
  description: string;
  parametersSchema: Record<string, unknown>;
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: string; // JSON-encoded
  metadata?: string; // Opaque provider-specific data (e.g., Gemini thought_signature)
}

export interface Message {
  role: "system" | "user" | "assistant" | "tool";
  content?: string;
  toolCalls?: ToolCall[];
  toolCallId?: string;
}

export type AgentChatStatus = "idle" | "running" | "awaiting_user" | "error";

export interface AgentChatSnapshot {
  path: string;
  title: string;
}

export interface AgentMessageMetadata {
  route?: string;
  pending?: boolean;
  hidden?: boolean;
  error?: string;
  runId?: string;
}

export interface AgentMessage extends Message {
  id: string;
  chatId: string;
  createdTs: number;
  metadata?: AgentMessageMetadata;
}

export interface AgentChat {
  id: string;
  title: string;
  createdTs: number;
  updatedTs: number;
  status: AgentChatStatus;
  totalTokensUsed: number;
  page?: AgentChatSnapshot;
  archived: boolean;
  lastError?: string | null;
  requiresAIConfiguration?: boolean;
  interrupted?: boolean;
  runId?: string | null;
}

export type AgentAskUserKind = "input" | "confirm" | "choose";

export interface AgentAskUserOption {
  label: string;
  value: string;
  description?: string;
}

export interface AgentPendingAsk {
  toolCallId: string;
  prompt: string;
  kind: AgentAskUserKind;
  defaultValue?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  options?: AgentAskUserOption[];
}

export type AgentAskUserResponse =
  | {
      kind: "input";
      answer: string;
    }
  | {
      kind: "confirm";
      answer: string;
      confirmed: boolean;
    }
  | {
      kind: "choose";
      answer: string;
      value: string;
    };

export type ToolExecutionResult =
  | {
      kind: "tool_result";
      result: string;
    }
  | {
      kind: "done";
      text: string;
      success: boolean;
    }
  | {
      kind: "ask_user";
      ask: AgentPendingAsk;
    };

interface AgentLoopUsage {
  totalTokensUsed?: number;
}

export type AgentLoopOutcome =
  | ({
      kind: "completed";
      text: string;
      success: boolean;
    } & AgentLoopUsage)
  | ({
      kind: "awaiting_user";
      ask: AgentPendingAsk;
    } & AgentLoopUsage)
  | ({
      kind: "aborted";
    } & AgentLoopUsage)
  | ({
      kind: "error";
      error: Error;
    } & AgentLoopUsage);

export type ToolExecutor = (
  name: string,
  args: Record<string, unknown>,
  toolCallId: string
) => Promise<ToolExecutionResult>;
