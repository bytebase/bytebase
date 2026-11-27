import { storageKeyAiConversations } from "@/utils/storage-keys";
import type { AgentChat, AgentMessage } from "./types";

export type LegacySqlIdentity = { workspace: string; email: string };
export type LegacySqlTranscript = { chat: AgentChat; messages: AgentMessage[] };
export type LegacySqlHistoryImportResult =
  | { status: "imported" | "already-imported"; unverifiedTargets: string[] }
  | {
      status: "identity-unavailable" | "invalid-source" | "storage-unavailable";
      fallback: LegacySqlTranscript[];
    };
type Message = {
  id: string;
  created_ts: number;
  content: string;
  error: string;
  author: "USER" | "AI";
  status: "DONE" | "FAILED" | "LOADING";
};
type Conversation = {
  id: string;
  created_ts: number;
  name: string;
  instance: string;
  database: string;
  messageList: Message[];
};
export type LegacySqlHistoryReadResult =
  | { status: "ready"; conversations: Conversation[] }
  | {
      status: "identity-unavailable" | "storage-unavailable" | "invalid-source";
    };

const object = (v: unknown): v is Record<string, unknown> =>
  v !== null && typeof v === "object" && !Array.isArray(v);

const isTimestamp = (v: unknown): v is number =>
  typeof v === "number" && Number.isFinite(new Date(v).getTime());

const message = (v: unknown): v is Message =>
  object(v) &&
  typeof v.id === "string" &&
  !!v.id &&
  isTimestamp(v.created_ts) &&
  typeof v.content === "string" &&
  typeof v.error === "string" &&
  (v.author === "USER" || v.author === "AI") &&
  (v.status === "DONE" || v.status === "FAILED" || v.status === "LOADING");

const conversation = (v: unknown): v is Conversation =>
  object(v) &&
  typeof v.id === "string" &&
  !!v.id &&
  isTimestamp(v.created_ts) &&
  typeof v.name === "string" &&
  typeof v.instance === "string" &&
  typeof v.database === "string" &&
  Array.isArray(v.messageList) &&
  v.messageList.every(message) &&
  new Set(v.messageList.map((m) => m.id)).size === v.messageList.length;

export function readLegacySqlHistory(
  identity: LegacySqlIdentity
): LegacySqlHistoryReadResult {
  if (!identity.workspace || !identity.email)
    return { status: "identity-unavailable" };
  let raw: string | null;
  try {
    raw = localStorage.getItem(
      storageKeyAiConversations(identity.workspace, identity.email)
    );
  } catch {
    return { status: "storage-unavailable" };
  }
  if (raw === null) return { status: "ready", conversations: [] };
  const limit = 1024 * 1024;
  if (raw.length > limit || new TextEncoder().encode(raw).length > limit)
    return { status: "invalid-source" };
  try {
    const data: unknown = JSON.parse(raw);
    if (
      object(data) &&
      data.version === 1 &&
      Array.isArray(data.conversations) &&
      data.conversations.length <= 20 &&
      data.conversations.every(conversation) &&
      new Set(data.conversations.map((c) => c.id)).size ===
        data.conversations.length
    ) {
      return { status: "ready", conversations: data.conversations };
    }
  } catch {
    // Invalid source is retained for recovery.
  }
  return { status: "invalid-source" };
}

export function legacySqlTranscript(
  identity: LegacySqlIdentity,
  source: Conversation
): LegacySqlTranscript {
  const id =
    "legacy-sql:" +
    JSON.stringify([identity.workspace, identity.email, source.id]);
  return {
    chat: {
      id,
      title: source.name,
      workspaceName: identity.workspace,
      createdTs: source.created_ts,
      updatedTs: source.messageList.reduce(
        (ts, m) => Math.max(ts, m.created_ts),
        source.created_ts
      ),
      archived: true,
      status: "idle",
      totalTokensUsed: 0,
      legacySqlImport: {
        version: 1,
        sourceId: source.id,
        readOnly: true,
        instance: source.instance,
        database: source.database,
      },
    },
    messages: source.messageList.map((m) => ({
      id: m.id,
      chatId: id,
      role: m.author === "USER" ? "user" : "assistant",
      content: m.content,
      createdTs: m.created_ts,
      metadata: {
        legacySqlStatus: m.status,
        ...(m.error ? { error: m.error } : {}),
      },
    })),
  };
}

export async function importLegacySqlHistory(
  identity: LegacySqlIdentity
): Promise<LegacySqlHistoryImportResult> {
  const { useAgentStore } = await import("../store/agent");
  return useAgentStore.getState().importLegacySqlHistory(identity);
}
