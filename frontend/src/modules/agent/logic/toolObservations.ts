import type { Message } from "./types";

export interface ToolObservation {
  name: string;
  args: Record<string, unknown>;
  result?: string;
}

export function getCurrentTurnToolCalls(
  messages: readonly Message[]
): ToolObservation[] {
  const lastUser = messages.findLastIndex((message) => message.role === "user");
  if (lastUser < 0) return [];
  const observations: ToolObservation[] = [];
  const pending = new Map<string, ToolObservation>();
  for (const message of messages.slice(lastUser + 1)) {
    if (message.role === "assistant") {
      pending.clear();
      const ids = new Set<string>();
      for (const call of message.toolCalls ?? []) {
        try {
          const args: unknown = JSON.parse(call.arguments);
          if (!args || typeof args !== "object" || Array.isArray(args))
            continue;
          const observation = {
            name: call.name,
            args: args as Record<string, unknown>,
          };
          observations.push(observation);
          if (ids.has(call.id)) pending.delete(call.id);
          else pending.set(call.id, observation);
          ids.add(call.id);
        } catch {
          // Malformed arguments cannot establish an operation for recovery.
        }
      }
    } else if (message.role === "tool" && message.toolCallId) {
      const call = pending.get(message.toolCallId);
      if (call && message.content !== undefined) {
        call.result = message.content;
        pending.delete(message.toolCallId);
      }
    }
  }
  return observations;
}
