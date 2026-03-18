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

export type ToolExecutor = (
  name: string,
  args: Record<string, unknown>
) => Promise<string>;
