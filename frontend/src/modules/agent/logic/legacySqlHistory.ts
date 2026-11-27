import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { GetProjectRequestSchema } from "@/types/proto-es/v1/project_service_pb";
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

export function normalizeLegacySqlProject(value: unknown): string | undefined {
  return typeof value === "string" &&
    value.trim() === value &&
    /^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(value)
    ? value
    : undefined;
}

function legacyDatabaseName(source: NonNullable<AgentChat["legacySqlImport"]>) {
  const segment = (value: string) =>
    Boolean(value) &&
    value.trim() === value &&
    !/\p{Cc}/u.test(value) &&
    !["-", ".", ".."].includes(value);
  const name = source.database.includes("/")
    ? source.database
    : `${source.instance}/databases/${source.database}`;
  const match =
    /^(?:(projects\/[^/]+)\/)?instances\/([^/]+)\/databases\/([^/]+)$/.exec(
      name
    );
  if (
    !match ||
    (match[1] && !normalizeLegacySqlProject(match[1])) ||
    !segment(match[2]) ||
    !segment(match[3]) ||
    (source.instance &&
      source.instance !== name.slice(0, name.lastIndexOf("/databases/")))
  )
    return undefined;
  return { name, project: match[1] };
}

export async function resolveLegacySqlHistoryProjects(
  transcripts: readonly LegacySqlTranscript[],
  isCurrent: () => boolean
): Promise<Map<string, string>> {
  const databases = new Map<string, Promise<string | undefined>>();
  const projects = new Map<string, Promise<string | undefined>>();
  const contextValues = createContextValues().set(silentContextKey, true);
  const resolveProject = (name: string) => {
    let result = projects.get(name);
    if (!result) {
      result = (async () => {
        if (!isCurrent()) return undefined;
        try {
          const project = await projectServiceClientConnect.getProject(
            create(GetProjectRequestSchema, { name }),
            { contextValues }
          );
          return isCurrent() &&
            project.name === name &&
            project.state === State.ACTIVE
            ? name
            : undefined;
        } catch {
          return undefined;
        }
      })();
      projects.set(name, result);
    }
    return result;
  };
  const entries = await Promise.all(
    transcripts.map(async ({ chat }) => {
      const source = chat.legacySqlImport;
      const target = source && legacyDatabaseName(source);
      if (!isCurrent() || !target) return undefined;
      let result = databases.get(target.name);
      if (!result) {
        result = (async () => {
          try {
            const database = await databaseServiceClientConnect.getDatabase(
              create(GetDatabaseRequestSchema, { name: target.name }),
              { contextValues }
            );
            const project = normalizeLegacySqlProject(database.project);
            if (
              !isCurrent() ||
              database.name !== target.name ||
              database.state !== State.ACTIVE ||
              !project ||
              (target.project && target.project !== project)
            )
              return undefined;
            return resolveProject(project);
          } catch {
            return undefined;
          }
        })();
        databases.set(target.name, result);
      }
      const project = await result;
      return isCurrent() && project ? ([chat.id, project] as const) : undefined;
    })
  );
  return new Map(
    isCurrent() ? entries.filter((entry) => entry !== undefined) : []
  );
}

export async function importLegacySqlHistory(
  identity: LegacySqlIdentity
): Promise<LegacySqlHistoryImportResult> {
  const { useAgentStore } = await import("../store/agent");
  return useAgentStore.getState().importLegacySqlHistory(identity);
}
