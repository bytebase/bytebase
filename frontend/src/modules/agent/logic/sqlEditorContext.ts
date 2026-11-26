import type { GenEnum } from "@bufbuild/protobuf/codegenv2";
import { isEqual } from "lodash-es";
import { Engine, EngineSchema } from "@/types/proto-es/v1/common_pb";
import {
  type QueryOption,
  QueryOption_ExplainFormatSchema,
  QueryOption_RedisRunCommandsOnSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import type { SQLEditorTabMode } from "@/types/sqlEditor/tab";
import type { AgentChatSnapshot, AgentMessage, AgentRunMode } from "./types";

const MAX_ATTACHMENT_BYTES = 32_768;

export interface AgentSqlConnectionSnapshot {
  database: string;
  project: string;
  engine: Engine;
  querySchema?: string;
  container?: string;
  dataSourceId?: string;
}

export interface AgentSqlEditorTarget {
  actor: string;
  workspace: string;
  hostId: string;
  mountId: string;
  project?: string;
  tabId: string;
  tabMode: "SAVED_QUERY";
  savedQuery?: string;
  modelUri: string;
  modelVersion: number;
  connection?: AgentSqlConnectionSnapshot;
  selection: { start: number; end: number; caret: number };
}

type AgentSqlQueryOptions = Partial<
  Pick<QueryOption, "redisRunCommandsOn" | "explainFormat">
>;

export interface AgentSqlEditorSnapshot {
  version: 1;
  project?: string;
  tab?: { id: string; mode: SQLEditorTabMode };
  connection?: AgentSqlConnectionSnapshot;
  viewerObject?: { kind: string; name: string; schema?: string };
  batchActive?: boolean;
  rowLimit?: number;
  queryOptions?: AgentSqlQueryOptions;
  editorTarget?: AgentSqlEditorTarget;
  attachment?: { kind: "explain" | "find-problems" | "selection"; sql: string };
}

export type AgentSqlContextCapture =
  | { kind: "pending" }
  | { kind: "ready"; snapshot: AgentSqlEditorSnapshot };

// Only a valid legacy turn with no SQL on either message has no SQL context.
export function getAgentSourceSqlEditorContext(
  messages: readonly AgentMessage[],
  sourceMessageId: string
): AgentSqlContextCapture | undefined {
  const pending = { kind: "pending" } as const;
  if (
    typeof sourceMessageId !== "string" ||
    !sourceMessageId ||
    sourceMessageId.length > 1000
  )
    return pending;
  const index = messages.findIndex((message) => message.id === sourceMessageId);
  const answer = messages[index];
  if (
    !answer ||
    index !==
      messages.findLastIndex((message) => message.id === sourceMessageId) ||
    answer.role !== "assistant" ||
    !answer.content?.trim() ||
    answer.toolCalls?.length ||
    answer.metadata?.pending ||
    answer.metadata?.error?.trim() ||
    answer.metadata?.interruptionReason ||
    answer.metadata?.sqlEditorContextInvalid
  )
    return pending;
  const user = messages
    .slice(0, index)
    .findLast((message) => message.role === "user");
  if (
    !user ||
    user.chatId !== answer.chatId ||
    user.metadata?.interruptionReason ||
    user.metadata?.sqlEditorContextInvalid
  )
    return pending;
  if (
    user.metadata?.sqlEditor === undefined &&
    answer.metadata?.sqlEditor === undefined
  )
    return undefined;
  const snapshot = normalizeAgentSqlEditorSnapshot(user.metadata?.sqlEditor);
  return snapshot &&
    isEqual(
      snapshot,
      normalizeAgentSqlEditorSnapshot(answer.metadata?.sqlEditor)
    )
    ? { kind: "ready", snapshot }
    : pending;
}

export interface AgentSqlActionProvenance {
  version: 1;
  runId: string;
  runMode: AgentRunMode;
  diagnostic: boolean;
}

const record = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);
const text = (value: unknown): value is string =>
  typeof value === "string" && Boolean(value.trim()) && !value.includes("\0");
const count = (value: unknown): value is number =>
  typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
const resourceId = (value: string): boolean =>
  value.trim() === value &&
  !/\p{Cc}/u.test(value) &&
  value !== "-" &&
  value !== "." &&
  value !== "..";
const resourceName = (value: unknown, collection: string): value is string => {
  if (!text(value)) return false;
  const [prefix, id, ...rest] = value.split("/");
  return prefix === collection && Boolean(id && resourceId(id)) && !rest.length;
};
const enumValue = <T extends number>(
  value: unknown,
  schema: GenEnum<T>
): value is T => schema.values.some((item) => item.number === value);

function normalizeConnection(
  value: unknown
): AgentSqlConnectionSnapshot | undefined {
  if (
    !record(value) ||
    !text(value.database) ||
    !resourceName(value.project, "projects") ||
    !enumValue(value.engine, EngineSchema) ||
    value.engine === Engine.ENGINE_UNSPECIFIED ||
    (value.querySchema !== undefined && !text(value.querySchema)) ||
    (value.container !== undefined && !text(value.container)) ||
    (value.dataSourceId !== undefined && !text(value.dataSourceId))
  )
    return undefined;
  const match =
    /^(?:(projects\/[^/]+)\/)?instances\/([^/]+)\/databases\/([^/]+)$/.exec(
      value.database
    );
  if (
    !match ||
    (match[1] !== undefined && match[1] !== value.project) ||
    !resourceId(match[2]) ||
    !resourceId(match[3])
  )
    return undefined;
  return {
    database: value.database,
    project: value.project,
    engine: value.engine,
    ...(value.querySchema === undefined
      ? {}
      : { querySchema: value.querySchema }),
    ...(value.container === undefined ? {} : { container: value.container }),
    ...(value.dataSourceId === undefined
      ? {}
      : { dataSourceId: value.dataSourceId }),
  };
}

function normalizeEditorTarget(
  value: unknown,
  project: AgentSqlEditorSnapshot["project"],
  tab: AgentSqlEditorSnapshot["tab"],
  connection: AgentSqlEditorSnapshot["connection"]
): AgentSqlEditorTarget | undefined {
  if (
    !record(value) ||
    !resourceName(value.actor, "users") ||
    !resourceName(value.workspace, "workspaces") ||
    !text(value.hostId) ||
    !text(value.mountId) ||
    value.project !== project ||
    tab?.mode !== "SAVED_QUERY" ||
    value.tabMode !== tab.mode ||
    (value.savedQuery !== undefined &&
      (!project ||
        typeof value.savedQuery !== "string" ||
        !value.savedQuery.startsWith(`${project}/`) ||
        !resourceName(
          value.savedQuery.slice(project.length + 1),
          "savedQueries"
        ))) ||
    value.tabId !== tab.id ||
    !text(value.modelUri) ||
    !count(value.modelVersion) ||
    value.modelVersion === 0 ||
    !record(value.selection) ||
    !count(value.selection.start) ||
    !count(value.selection.end) ||
    !count(value.selection.caret) ||
    value.selection.start > value.selection.end ||
    (value.selection.caret !== value.selection.start &&
      value.selection.caret !== value.selection.end)
  )
    return undefined;
  const target = normalizeConnection(value.connection);
  if (
    (value.connection !== undefined && !target) ||
    target?.database !== connection?.database ||
    target?.project !== connection?.project ||
    target?.engine !== connection?.engine ||
    target?.querySchema !== connection?.querySchema ||
    target?.container !== connection?.container ||
    target?.dataSourceId !== connection?.dataSourceId
  )
    return undefined;
  return {
    actor: value.actor,
    workspace: value.workspace,
    hostId: value.hostId,
    mountId: value.mountId,
    ...(project === undefined ? {} : { project }),
    tabId: tab.id,
    tabMode: tab.mode,
    ...(value.savedQuery === undefined ? {} : { savedQuery: value.savedQuery }),
    modelUri: value.modelUri,
    modelVersion: value.modelVersion,
    ...(target ? { connection: target } : {}),
    selection: {
      start: value.selection.start,
      end: value.selection.end,
      caret: value.selection.caret,
    },
  };
}

export function normalizeAgentSqlEditorSnapshot(
  value: unknown
): AgentSqlEditorSnapshot | undefined {
  if (
    !record(value) ||
    value.version !== 1 ||
    (value.project !== undefined && !resourceName(value.project, "projects"))
  )
    return undefined;
  const snapshot: AgentSqlEditorSnapshot = { version: 1 };
  if (value.project !== undefined) snapshot.project = value.project;
  if (value.tab !== undefined) {
    const tab = value.tab;
    if (
      !record(tab) ||
      !text(tab.id) ||
      (tab.mode !== "SAVED_QUERY" &&
        tab.mode !== "ADMIN" &&
        tab.mode !== "DATA_EXPLORER")
    )
      return undefined;
    snapshot.tab = { id: tab.id, mode: tab.mode };
  }
  if (value.connection !== undefined) {
    const connection = normalizeConnection(value.connection);
    if (
      !connection ||
      !snapshot.project ||
      snapshot.project !== connection.project
    )
      return undefined;
    snapshot.connection = connection;
  }
  if (value.viewerObject !== undefined) {
    const object = value.viewerObject;
    if (
      !snapshot.connection ||
      !record(object) ||
      !text(object.kind) ||
      !text(object.name) ||
      (object.schema !== undefined && !text(object.schema))
    )
      return undefined;
    snapshot.viewerObject = {
      kind: object.kind,
      name: object.name,
      ...(object.schema === undefined ? {} : { schema: object.schema }),
    };
  }
  if (value.rowLimit !== undefined) {
    if (!count(value.rowLimit) || value.rowLimit === 0) return undefined;
    snapshot.rowLimit = value.rowLimit;
  }
  if (value.batchActive !== undefined) {
    if (typeof value.batchActive !== "boolean") return undefined;
    snapshot.batchActive = value.batchActive;
  }
  if (value.queryOptions !== undefined) {
    const options = value.queryOptions;
    if (
      !record(options) ||
      (options.redisRunCommandsOn !== undefined &&
        !enumValue(
          options.redisRunCommandsOn,
          QueryOption_RedisRunCommandsOnSchema
        )) ||
      (options.explainFormat !== undefined &&
        !enumValue(options.explainFormat, QueryOption_ExplainFormatSchema))
    )
      return undefined;
    snapshot.queryOptions = {
      ...(options.redisRunCommandsOn === undefined
        ? {}
        : { redisRunCommandsOn: options.redisRunCommandsOn }),
      ...(options.explainFormat === undefined
        ? {}
        : { explainFormat: options.explainFormat }),
    };
  }
  if (value.editorTarget !== undefined) {
    const target = normalizeEditorTarget(
      value.editorTarget,
      snapshot.project,
      snapshot.tab,
      snapshot.connection
    );
    if (!target) return undefined;
    snapshot.editorTarget = target;
  }
  if (value.attachment !== undefined) {
    const attachment = value.attachment;
    if (
      !record(attachment) ||
      (attachment.kind !== "explain" &&
        attachment.kind !== "find-problems" &&
        attachment.kind !== "selection") ||
      !text(attachment.sql) ||
      attachment.sql.length > MAX_ATTACHMENT_BYTES ||
      !attachment.sql.isWellFormed() ||
      new TextEncoder().encode(attachment.sql).length > MAX_ATTACHMENT_BYTES
    )
      return undefined;
    snapshot.attachment = { kind: attachment.kind, sql: attachment.sql };
  }
  return snapshot;
}

export function normalizeAgentChatSnapshot(
  value: unknown
): AgentChatSnapshot | undefined {
  if (
    !record(value) ||
    typeof value.path !== "string" ||
    typeof value.title !== "string"
  )
    return undefined;
  const sqlEditor = normalizeAgentSqlEditorSnapshot(value.sqlEditor);
  if (value.sqlEditor !== undefined && !sqlEditor) return undefined;
  return {
    path: value.path,
    title: value.title,
    ...(sqlEditor ? { sqlEditor } : {}),
  };
}

export function normalizeAgentSqlActionProvenance(
  value: unknown
): AgentSqlActionProvenance | undefined {
  if (
    !record(value) ||
    value.version !== 1 ||
    !text(value.runId) ||
    (value.runMode !== "default" && value.runMode !== "read-only") ||
    typeof value.diagnostic !== "boolean"
  )
    return undefined;
  return {
    version: 1,
    runId: value.runId,
    runMode: value.runMode,
    diagnostic: value.diagnostic,
  };
}
