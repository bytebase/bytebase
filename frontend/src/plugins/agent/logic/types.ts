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

export type AgentThreadStatus = "idle" | "running" | "awaiting_user" | "error";

export interface AgentThreadSnapshot {
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
  threadId: string;
  createdTs: number;
  metadata?: AgentMessageMetadata;
}

export interface AgentThread {
  id: string;
  title: string;
  createdTs: number;
  updatedTs: number;
  status: AgentThreadStatus;
  page?: AgentThreadSnapshot;
  lastError?: string | null;
  interrupted?: boolean;
}

export type ToolExecutor = (
  name: string,
  args: Record<string, unknown>
) => Promise<string>;
