import { SHA256 } from "crypto-js";
import { isEqual } from "lodash-es";
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { useAppStore } from "@/stores/app";
import { useAgentStore } from "../store/agent";
import { getCurrentTurnQueryCalls } from "./queryFollowUps";
import {
  getHumanInputResourceReferenceMap,
  normalizeAgentResourceReferences,
} from "./resourceMention";
import { normalizeAgentResponseActions } from "./responseActions";
import { supportsAgentRunRecovery } from "./runOwnership";
import {
  type AgentSqlEditorSnapshot,
  getAgentSourceSqlEditorContext,
  normalizeAgentSqlActionProvenance,
} from "./sqlEditorContext";
import {
  type AgentSqlTargetState,
  getCurrentAgentSqlEditorContext,
  getRegisteredAgentSqlEditorHost,
} from "./sqlEditorHost";
import type { AgentChat, AgentMessage } from "./types";

export interface AgentSqlBlock {
  blockIndex: number;
  start: number;
  end: number;
  sql: string;
}

export interface AgentSqlActionSource {
  chatId: string;
  messageId: string;
  blockIndex: number;
  sourceHash: string;
  sql: string;
  snapshot: AgentSqlEditorSnapshot;
}

const markdownParser = unified().use(remarkParse).use(remarkGfm);
const maxSqlBytes = 100_000;
const encoder = new TextEncoder();
const placeholder =
  /\[\s*(?:redacted|omitted|placeholder)[^[\]]*\]|<\s*(?:table|column|database|sql|your[_ -]?\w+)[^<>]*>|\.{3}|\b(?:rest|remaining|other|original)\s+(?:of\s+(?:the\s+)?)?(?:sql|statements?|code)\s+(?:is\s+)?(?:unchanged|omitted)|\b(?:insert|replace|add)\s+(?:your\s+)?(?:sql|code)\s+here\b/i;

export function readAgentSqlBlocks(
  markdown: string
): (AgentSqlBlock & { actionable: boolean })[] {
  let blockIndex = 0;
  const blocks: (AgentSqlBlock & { actionable: boolean })[] = [];
  for (const node of markdownParser.parse(markdown).children) {
    if (node.type !== "code" || node.lang?.toLowerCase() !== "sql") continue;
    const start = node.position?.start.offset;
    const end = node.position?.end.offset;
    if (start === undefined || end === undefined) continue;
    const raw = markdown.slice(start, end);
    const opening = /^(`{3,}|~{3,})([^\r\n]*)(?:\r\n|\n|\r)/.exec(raw);
    if (!opening) continue;
    const delimiter = opening[1];
    const closing = new RegExp(
      `(?:\r\n|\n|\r) {0,3}${delimiter[0]}{${delimiter.length},}[ \t]*$`
    ).exec(raw);
    if (!closing) continue;
    // Index all complete top-level SQL fences before testing byte eligibility.
    const index = blockIndex++;
    if (node.position?.start.column !== 1 || closing.index < opening[0].length)
      continue;
    const sql = raw.slice(opening[0].length, closing.index);
    blocks.push({
      blockIndex: index,
      start,
      end,
      sql,
      actionable:
        /^[ \t]*sql[ \t]*$/i.test(opening[2]) &&
        sql.length <= maxSqlBytes &&
        Boolean(sql.trim()) &&
        sql.isWellFormed() &&
        !sql.includes("\0") &&
        encoder.encode(sql).byteLength <= maxSqlBytes &&
        !placeholder.test(sql),
    });
  }
  return blocks;
}

export function extractAgentSqlBlocks(markdown: string): AgentSqlBlock[] {
  return readAgentSqlBlocks(markdown).flatMap(({ actionable, ...block }) =>
    actionable ? [block] : []
  );
}

function hasCompatibleSqlTargets(
  messages: readonly AgentMessage[],
  message: AgentMessage,
  snapshot: AgentSqlEditorSnapshot
): boolean {
  const turn = messages.slice(
    0,
    messages.findIndex(({ id }) => id === message.id)
  );
  const userIndex = turn.findLastIndex(({ role }) => role === "user");
  const references = normalizeAgentResourceReferences(
    message.metadata?.resourceReferences,
    message.metadata?.resourceReference
  );
  if (
    (message.metadata?.resourceReferences !== undefined ||
      message.metadata?.resourceReference !== undefined) &&
    !references.length
  )
    return false;
  const turnReferences = [
    ...getHumanInputResourceReferenceMap(turn.slice(userIndex)).values(),
  ].flat();
  return (
    [...references, ...turnReferences].every(
      (reference) =>
        reference.project === snapshot.project &&
        (reference.kind !== "database" ||
          reference.name === snapshot.connection?.database)
    ) &&
    getCurrentTurnQueryCalls(turn).every(({ args }) => {
      const body = args.body;
      return Boolean(
        snapshot.connection &&
          body &&
          typeof body === "object" &&
          "name" in body &&
          body.name === snapshot.connection.database
      );
    })
  );
}

export function bindAgentSqlBlockActions(
  message: AgentMessage,
  messages: readonly AgentMessage[] = useAgentStore
    .getState()
    .getMessages(message.chatId)
): AgentSqlActionSource[] {
  const { metadata, content } = message;
  const provenance = normalizeAgentSqlActionProvenance(
    metadata?.sqlActionProvenance
  );
  if (
    !Array.isArray(metadata?.actions) ||
    !metadata.actions.some((action) => action?.kind === "sql-editor-block") ||
    !content?.isWellFormed() ||
    !provenance ||
    metadata?.runId !== provenance.runId ||
    provenance.runMode !== "default" ||
    provenance.diagnostic ||
    !isEqual(
      messages.find(({ id }) => id === message.id),
      message
    )
  )
    return [];
  const context = getAgentSourceSqlEditorContext(messages, message.id);
  if (context?.kind !== "ready") return [];
  const { snapshot } = context;
  if (
    snapshot.tab?.mode !== "SAVED_QUERY" ||
    !snapshot.editorTarget ||
    snapshot.batchActive ||
    !hasCompatibleSqlTargets(messages, message, snapshot)
  )
    return [];
  const blocks = extractAgentSqlBlocks(content);
  const validIndexes = new Set(blocks.map(({ blockIndex }) => blockIndex));
  const markers = normalizeAgentResponseActions(metadata.actions, (index) =>
    validIndexes.has(index)
  ).filter((action) => action.kind === "sql-editor-block");
  const markerIndexes = new Set(markers.map(({ blockIndex }) => blockIndex));
  const marked = blocks.filter(({ blockIndex }) =>
    markerIndexes.has(blockIndex)
  );
  if (!marked.length) return [];
  const sourceHash = SHA256(content).toString();
  return marked.map(({ blockIndex, sql }) => ({
    chatId: message.chatId,
    messageId: message.id,
    blockIndex,
    sourceHash,
    sql,
    snapshot,
  }));
}

function getIdleAgentSqlActionChat(source: AgentSqlActionSource) {
  const state = useAgentStore.getState();
  const { chatId } = source;
  const chat = state.getChat(chatId);
  if (
    !chat ||
    chat.id !== state.currentChatId ||
    chat.status !== "idle" ||
    !isAgentSqlActionChat(chat, source.snapshot, state.workspaceName) ||
    state.remoteBusyChatIds[chatId] ||
    state.abortControllersByChatId[chatId] ||
    state.queuedUserMessageInFlightByChatId[chatId] ||
    state.queuedUserMessagesByChatId[chatId] ||
    state.followUpQueuesByChatId[chatId]?.messages.length
  )
    return undefined;
  return chat;
}

export function getCurrentAgentSqlActionSource(
  source: AgentSqlActionSource
): AgentSqlActionSource | undefined {
  if (!getIdleAgentSqlActionChat(source)) return undefined;
  const { chatId, messageId, blockIndex } = source;
  const state = useAgentStore.getState();
  const messages = state.getMessages(chatId);
  const message = messages.find(({ id }) => id === messageId);
  const bound =
    message &&
    bindAgentSqlBlockActions(message, messages).find(
      (candidate) => candidate.blockIndex === blockIndex
    );
  return bound && isEqual(source, bound) ? bound : undefined;
}

export function isAgentSqlActionChat(
  chat: AgentChat,
  snapshot: AgentSqlEditorSnapshot,
  workspaceName: string | null | undefined
): boolean {
  const target = snapshot.editorTarget;
  return Boolean(
    target &&
      workspaceName === target.workspace &&
      !chat.archived &&
      chat.capability === undefined &&
      (!("legacySqlImport" in chat) || chat.legacySqlImport === undefined) &&
      (chat.workspaceName === undefined ||
        chat.workspaceName === target.workspace) &&
      (snapshot.project
        ? chat.context === snapshot.project
        : chat.context === undefined || chat.context === target.workspace)
  );
}

export function inspectAgentSqlAction(
  source: AgentSqlActionSource,
  mode: "insert" | "replace-selection" = "insert"
): AgentSqlTargetState {
  const target = source.snapshot.editorTarget;
  const host = getRegisteredAgentSqlEditorHost();
  if (!target || !host?.inspect || !host.applySql)
    return { status: "unavailable" };
  // Rendering receives an already bound source. The click path rebinds before editing.
  if (!getIdleAgentSqlActionChat(source)) return { status: "stale" };
  if (mode === "replace-selection") return host.inspect(target);
  const live = getCompatibleSqlSnapshot(source);
  return live?.editorTarget
    ? host.inspect(live.editorTarget)
    : { status: "stale" };
}

function getCompatibleSqlSnapshot(source: AgentSqlActionSource) {
  const original = source.snapshot;
  const capture = getCurrentAgentSqlEditorContext();
  const live = capture?.kind === "ready" ? capture.snapshot : undefined;
  const target = live?.editorTarget;
  const host = getRegisteredAgentSqlEditorHost();
  if (
    !original.editorTarget ||
    !live ||
    !target ||
    !host ||
    target.hostId !== host.id ||
    target.actor !== original.editorTarget.actor ||
    target.actor !== useAppStore.getState().currentUser?.name ||
    target.workspace !== original.editorTarget.workspace ||
    target.workspace !== useAgentStore.getState().workspaceName ||
    live.tab?.mode !== "SAVED_QUERY" ||
    live.project !== original.project ||
    (live.connection && live.connection.project !== live.project) ||
    (original.connection && !isEqual(original.connection, live.connection))
  )
    return undefined;
  return live;
}

export function isAgentSqlExecutionTargetCurrent(
  source: AgentSqlActionSource,
  snapshot: AgentSqlEditorSnapshot = source.snapshot
): boolean {
  const target = snapshot.editorTarget;
  const live = getCompatibleSqlSnapshot(source);
  if (!target || !live?.editorTarget) return false;
  const { selection: _selection, modelVersion: _version, ...binding } = target;
  const {
    selection: _liveSelection,
    modelVersion: _liveVersion,
    ...liveBinding
  } = live.editorTarget;
  // A Query does not edit Monaco. Its click-time destination and options stay
  // fixed through approval, independently of later text and cursor changes.
  return (
    !live.batchActive &&
    isEqual(liveBinding, binding) &&
    isEqual(live.tab, snapshot.tab) &&
    isEqual(live.connection, snapshot.connection) &&
    live.project === snapshot.project &&
    isEqual(live.queryOptions, snapshot.queryOptions) &&
    live.rowLimit === snapshot.rowLimit
  );
}

export function canBeginAgentSqlResponseAction(
  source: AgentSqlActionSource,
  captured?: AgentSqlEditorSnapshot
): boolean {
  const state = useAgentStore.getState();
  const chat = getIdleAgentSqlActionChat(source);
  const snapshot = captured ?? getCompatibleSqlSnapshot(source);
  if (!snapshot?.editorTarget || !source.snapshot.connection) return false;
  return Boolean(
    chat &&
      supportsAgentRunRecovery() &&
      state.canUseAgentExecution() &&
      !state.storagePressure &&
      !state.followUpDrainChatIds[chat.id] &&
      !state.getPendingAsk(chat.id) &&
      !chat.interrupted &&
      !chat.runId &&
      !chat.runCheckpoint &&
      snapshot.tab?.mode === "SAVED_QUERY" &&
      !snapshot.batchActive &&
      snapshot.project &&
      snapshot.connection?.project === snapshot.project &&
      snapshot.rowLimit !== undefined &&
      isAgentSqlExecutionTargetCurrent(source, snapshot)
  );
}

export function captureAgentSqlExecutionSnapshot(source: AgentSqlActionSource) {
  const snapshot = getCompatibleSqlSnapshot(source);
  return snapshot && canBeginAgentSqlResponseAction(source, snapshot)
    ? structuredClone(snapshot)
    : undefined;
}

export function applyAgentSqlAction(
  source: AgentSqlActionSource,
  mode: "insert" | "replace-selection"
): boolean {
  const current = getCurrentAgentSqlActionSource(source);
  const target =
    mode === "replace-selection"
      ? current?.snapshot.editorTarget
      : current && getCompatibleSqlSnapshot(current)?.editorTarget;
  const host = getRegisteredAgentSqlEditorHost();
  if (
    !current ||
    !target ||
    !host?.inspect ||
    host.inspect(target).status !== "ready" ||
    (mode === "replace-selection" &&
      target.selection.start === target.selection.end)
  )
    return false;
  return (
    host.applySql?.({ target, sql: current.sql, mode }).status === "applied"
  );
}
