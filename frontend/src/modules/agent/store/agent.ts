import { isEqual } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import { create } from "zustand";
import { immer } from "zustand/middleware/immer";
import { getResourceId, workspaceNamePrefix } from "@/lib/resourceName";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import { useAppStore } from "@/stores/app";
import {
  getNameParentTokens,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import {
  actionReceiptKey,
  createAccessGrantActionReceipt,
  mergeActionReceipts,
  normalizeActionReceipts,
} from "../logic/actionReceipt";
import {
  type AgentApiActionIdentity,
  type AgentApiAttempt,
  type AgentApiDispatchContext,
  digestAgentApiJson,
  getLegacyApiOrphans,
  mergeAgentApiAttempts,
  normalizeAgentApiAttempts,
  reconcileApiResultLinks,
  storedApiReceipts,
} from "../logic/apiAttempt";
import {
  agentApiExecutionEvidence,
  isAgentApiEvidenceForEnvelope,
} from "../logic/apiExecution";
import { parseCallApiResult } from "../logic/callApiResult";
import {
  createAgentChatProtection,
  getProtectedAgentChatIds,
  supportsAgentChatProtection,
  withAgentChatEvictionLease,
} from "../logic/chatProtection";
import {
  normalizeAgentDiagnosticCapability,
  sameAgentDiagnosticCapability,
} from "../logic/diagnosticCapability";
import type {
  AgentExpressionDraftMode,
  AgentExpressionDraftTarget,
} from "../logic/expressionDraft";
import { normalizeAgentHandoff } from "../logic/handoff";
import type { LandingStarterId } from "../logic/landingStarters";
import { omitAgentQueryRows } from "../logic/queryResultRetention";
import {
  formatResourceMention,
  getCurrentTurnResourceReferences,
  getMentionTriggerRange,
  insertResourceMention,
  isResourceMentionRange,
  normalizeAgentResourceMentions,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
} from "../logic/resourceMention";
import { normalizeAgentResponseActions } from "../logic/responseActions";
import {
  getAgentRunClarificationBindings,
  getAgentRunContinuation,
  getAgentRunStartup,
} from "../logic/runCheckpoint";
import {
  type AgentRunLease,
  hasAgentRunLease,
  supportsAgentRunRecovery,
  withAgentApiActionLease,
  withAgentPersistenceLock,
  withAgentRunLease,
  withAgentStorageBudgetLock,
} from "../logic/runOwnership";
import {
  type AgentSqlFixDraft,
  type AgentSqlFixSource,
  extractAgentSqlFixSQL,
  isCurrentAgentSqlFixSource,
} from "../logic/sqlFix";
import { isAccessGrantSourceDraft } from "../logic/submittedAccessGrant";
import {
  createRunBudget,
  increaseRunBudgetLimit,
  mergeModelAttempts,
  normalizeModelAttempt,
  normalizeRunBindings,
  normalizeRunBudget,
  normalizeRunCheckpoint,
  normalizeRunMaskingPage,
  projectChatUsage,
  projectRunBudget,
  reserveRunAttempt,
} from "../logic/tokenBudget";
import { normalizeModelCallUsage } from "../logic/tokenUsage";
import type {
  AgentActionObservation,
  AgentActionOrigin,
  AgentActionReceipt,
  AgentAskUserResponse,
  AgentAttemptReservation,
  AgentChat,
  AgentChatSnapshot,
  AgentChatStatus,
  AgentDiagnosticCapability,
  AgentDraft,
  AgentInterruptionReason,
  AgentMentionRange,
  AgentMessage,
  AgentMessageMetadata,
  AgentModelAttempt,
  AgentModelCallUsage,
  AgentPendingAsk,
  AgentResourceReference,
  AgentResponseAction,
  AgentRunBindings,
  AgentRunCheckpoint,
  AgentRunMode,
  AgentStoragePressure,
  Message,
  ToolCall,
} from "../logic/types";
import { getDefaultAgentWindowState } from "../window";
import {
  getAgentHistoryEvictionCandidates,
  protectsAgentChatFromDeletion,
  removeAgentHistoryChats,
} from "./historyRetention";
import {
  canDiscardStoredAgentHistory,
  canRetireLegacyAgentState,
  preservesLegacyApiDispatchEvidence,
} from "./legacyStateRetention";
import {
  reclaimOtherAgentHistory,
  withAgentHistoryReclamationLocks,
} from "./otherHistoryRetention";
import {
  type AgentPreferences,
  DEFAULT_AGENT_PREFERENCES,
  getAgentPreferencesKey,
  normalizeAgentPreferences,
} from "./preferences";
import { createAgentRunPersistence } from "./runPersistence";
import {
  encodeAgentStoredState,
  isAgentStorageQuotaError,
  parseAgentStoredState,
} from "./stateStorage";
import {
  AGENT_STORAGE_ADMISSION_BYTES,
  AGENT_STORAGE_CLEANUP_TARGET_BYTES,
  AGENT_STORAGE_CLEANUP_TRIGGER_BYTES,
  AGENT_STORAGE_LIMIT_BYTES,
  AgentStorageFullError,
  estimateAgentStorageBytes,
} from "./storageBudget";
import {
  omitAgentQuerySamples,
  retainAgentQuerySamples,
} from "./storageRetention";

export const AGENT_STATE_KEY = "bb-agent-state-v2";
export const AGENT_RUN_STATE_KEY = "bb-agent-state-v3";
export const AGENT_WINDOW_KEY = "bb-agent-window";
export const AGENT_WINDOW_LAYOUT_VERSION = 4;
const LEGACY_AGENT_STATE_KEY = `${AGENT_STATE_KEY}.legacy`;
const LEGACY_AGENT_OWNER_KEY = `${AGENT_STATE_KEY}-migration-owner`;

const getWorkspaceStorageKey = (key: string, workspaceName: string) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId ? `${key}.${workspaceId}` : null;
};

const getScopedAgentStateKey = (
  key: string,
  workspaceName: string,
  userEmail: string
) => {
  const workspaceId = getResourceId(workspaceName, workspaceNamePrefix);
  return workspaceId && userEmail
    ? `${key}.${workspaceId.length}:${workspaceId}.${userEmail}`
    : null;
};

export const getAgentStateKey = (workspaceName: string, userEmail: string) =>
  getScopedAgentStateKey(
    supportsAgentRunRecovery() ? AGENT_RUN_STATE_KEY : AGENT_STATE_KEY,
    workspaceName,
    userEmail
  );

export const getAgentWindowKey = (workspaceName: string) =>
  getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName) ?? AGENT_WINDOW_KEY;

export interface PersistedAgentState {
  currentChatId: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  followUpQueuesByChatId?: Record<string, AgentFollowUpQueue>;
  activeChatIds?: string[];
}

interface CreateChatOptions {
  page?: AgentChatSnapshot;
  context?: string;
  workspaceName?: string;
  select?: boolean;
  title?: string;
  archived?: boolean;
  capability?: AgentDiagnosticCapability;
}

interface AddMessageOptions extends Message {
  chatId?: string;
  metadata?: AgentMessage["metadata"];
}

export interface QueuedUserMessage {
  id: string;
  chatId: string;
  text: string;
  page: AgentChatSnapshot;
  maskingPage?: AgentRunBindings["maskingPage"];
  fromFollowUpQueue?: true;
  runMode?: AgentRunMode;
  starterId?: LandingStarterId;
  followUpSourceId?: string;
  resourceReference?: AgentResourceReference;
  resourceReferences?: AgentResourceReference[];
}

export interface AgentFollowUpQueue {
  messages: QueuedUserMessage[];
  paused: boolean;
}

interface RegisteredExpressionDraftTarget {
  apply: AgentExpressionDraftTarget;
  allowRaw: boolean;
  token: symbol;
}

export type QueueUserMessageInput = Omit<
  QueuedUserMessage,
  "id" | "fromFollowUpQueue"
>;

const DEFAULT_CHAT_STATUS: AgentChatStatus = "idle";
const DRAFT_PERSIST_DELAY_MS = 250;

const normalizeInterruptionReason = (
  value: unknown
): AgentInterruptionReason | undefined =>
  value === "user" || value === "unexpected" ? value : undefined;

const setLastUserInterruptionReason = (
  messages: AgentMessage[],
  reason?: AgentInterruptionReason
) => {
  const lastUserMessage = messages.findLast(
    (message) => message.role === "user"
  );
  if (!lastUserMessage) return;
  if (reason) {
    lastUserMessage.metadata = {
      ...lastUserMessage.metadata,
      interruptionReason: reason,
    };
  } else if (lastUserMessage.metadata?.interruptionReason) {
    const { interruptionReason: _, ...metadata } = lastUserMessage.metadata;
    lastUserMessage.metadata = Object.keys(metadata).length
      ? metadata
      : undefined;
  }
};

// --- Pure helper functions ---

const createChatRecord = (options: CreateChatOptions = {}): AgentChat => {
  const capability = normalizeAgentDiagnosticCapability(options.capability);
  if (options.capability !== undefined && !capability) {
    throw new Error("Invalid Agent diagnostic capability");
  }
  const now = Date.now();
  return {
    id: uuidv4(),
    title: options.title ?? "",
    context: options.context,
    workspaceName: options.workspaceName,
    createdTs: now,
    updatedTs: now,
    status: DEFAULT_CHAT_STATUS,
    totalTokensUsed: 0,
    page: options.page,
    archived: options.archived ?? false,
    lastError: null,
    requiresAIConfiguration: false,
    interrupted: false,
    runId: null,
    ...(capability ? { capability } : {}),
  };
};

const normalizeChatTitle = (title?: string) => {
  return (title ?? "").trim();
};

const getChatTitleFromMessage = (content?: string) => {
  const title = normalizeChatTitle(content).replace(/\s+/g, " ");
  if (!title) {
    return "";
  }
  return title.length <= 48 ? title : `${title.slice(0, 45)}...`;
};

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === "object" && value !== null && !Array.isArray(value);
};

const normalizeFollowUpSourceId = (value: unknown): string | undefined => {
  if (value === undefined) return undefined;
  // Dropping an invalid marker would silently restore latest-query fallback.
  return typeof value === "string" && value.trim() && value.length <= 1000
    ? value
    : "";
};

const normalizeDraft = (raw: unknown): AgentDraft | null => {
  if (!isRecord(raw) || typeof raw.text !== "string") return null;
  const followUpSourceId = normalizeFollowUpSourceId(raw.followUpSourceId);
  const provenance = followUpSourceId !== undefined ? { followUpSourceId } : {};
  if (raw.mentions !== undefined) {
    const mentions = normalizeAgentResourceMentions(raw.text, raw.mentions);
    return {
      text: raw.text,
      ...provenance,
      ...(mentions.length > 0 ? { mentions } : {}),
    };
  }
  const project = isRecord(raw.project) ? raw.project : {};
  const projectName =
    typeof project.name === "string" ? project.name.trim() : "";
  const projectTitle =
    typeof project.title === "string" ? project.title.trim() : "";
  const normalizedProject =
    projectTitle && getNameParentTokens(projectName, [projectNamePrefix]).length
      ? { name: projectName, title: projectTitle }
      : undefined;
  const normalizedResource = normalizeAgentResourceReference(raw.resource);
  const resource =
    normalizedResource &&
    (!normalizedProject ||
      normalizedProject.name === normalizedResource.project)
      ? normalizedResource
      : normalizedProject
        ? {
            kind: "project" as const,
            name: normalizedProject.name,
            project: normalizedProject.name,
            label: normalizedProject.title,
          }
        : undefined;
  if (!resource) return { text: raw.text, ...provenance };

  let text = raw.text;
  let mention: AgentMentionRange | null = null;
  if (isRecord(raw.mention)) {
    const { start, end } = raw.mention;
    if (typeof start === "number" && typeof end === "number") {
      const candidate = { start, end };
      if (isResourceMentionRange(text, candidate, resource.label)) {
        mention = candidate;
      }
    }
    if (!mention) return { text, ...provenance };
  } else if (raw.mention !== undefined) {
    return { text, ...provenance };
  } else {
    const token = formatResourceMention(resource.label);
    let start = text.lastIndexOf(token);
    while (start >= 0) {
      const candidate = { start, end: start + token.length };
      if (isResourceMentionRange(text, candidate, resource.label)) {
        mention = candidate;
        break;
      }
      if (start === 0) break;
      start = text.lastIndexOf(token, start - 1);
    }
    if (!mention) {
      const trigger = getMentionTriggerRange(text, text.length);
      const prefix = trigger
        ? text
        : text && !/\s$/u.test(text)
          ? `${text} @`
          : `${text}@`;
      const inserted = insertResourceMention(
        prefix,
        trigger ?? getMentionTriggerRange(prefix, prefix.length),
        resource.label
      );
      if (!inserted) return { text: raw.text, ...provenance };
      text = inserted.text;
      mention = inserted.mention;
    }
  }
  if (!mention) return { text, ...provenance };

  return {
    text,
    ...provenance,
    ...(normalizedProject ? { project: normalizedProject } : {}),
    resource,
    mention,
  };
};

const normalizeQueuedUserMessage = (
  raw: unknown,
  chatIds: Set<string>
): QueuedUserMessage | null => {
  if (
    !isRecord(raw) ||
    typeof raw.id !== "string" ||
    !raw.id ||
    typeof raw.chatId !== "string" ||
    !chatIds.has(raw.chatId) ||
    typeof raw.text !== "string" ||
    !raw.text.trim() ||
    !isRecord(raw.page) ||
    typeof raw.page.path !== "string" ||
    typeof raw.page.title !== "string"
  ) {
    return null;
  }
  const resourceReferences = normalizeAgentResourceReferences(
    raw.resourceReferences,
    raw.resourceReference
  );
  const maskingPage = normalizeRunMaskingPage(raw.maskingPage);
  return {
    id: raw.id,
    chatId: raw.chatId,
    text: raw.text,
    page: { path: raw.page.path, title: raw.page.title },
    ...(maskingPage ? { maskingPage } : {}),
    ...(raw.fromFollowUpQueue === true ? { fromFollowUpQueue: true } : {}),
    runMode:
      raw.runMode === "default" || raw.runMode === "read-only"
        ? raw.runMode
        : undefined,
    starterId:
      typeof raw.starterId === "string"
        ? (raw.starterId as LandingStarterId)
        : undefined,
    ...(raw.followUpSourceId !== undefined
      ? { followUpSourceId: normalizeFollowUpSourceId(raw.followUpSourceId) }
      : {}),
    ...(raw.resourceReferences === undefined && resourceReferences[0]
      ? { resourceReference: resourceReferences[0] }
      : {}),
    ...(raw.resourceReferences !== undefined && resourceReferences.length > 0
      ? { resourceReferences }
      : {}),
  };
};

const restorePromotedFollowUp = (
  state: Pick<
    PersistedAgentState,
    "queuedUserMessagesByChatId" | "followUpQueuesByChatId"
  >,
  message: QueuedUserMessage
) => {
  const { fromFollowUpQueue: _promoted, ...pending } = message;
  const queues = (state.followUpQueuesByChatId ??= {});
  const tail = queues[message.chatId]?.messages ?? [];
  queues[message.chatId] = {
    paused: true,
    messages: [pending, ...tail.filter(({ id }) => id !== message.id)],
  };
  delete state.queuedUserMessagesByChatId[message.chatId];
};

type DraftHandoffKind =
  | "draft-masking-rule"
  | "draft-masking-exemption"
  | "draft-semantic-type";

const normalizeDraftHandoff = (
  receipt: unknown,
  kind: DraftHandoffKind,
  role: unknown,
  actions: AgentResponseAction[]
): AgentMessageMetadata["maskingExemptionDraftHandoff"] =>
  role === "assistant" &&
  isRecord(receipt) &&
  typeof receipt.draftId === "string" &&
  (receipt.outcome === "opened" || receipt.outcome === "applied") &&
  (kind !== "draft-semantic-type" || receipt.outcome === "opened") &&
  actions.some(
    (action) => action.kind === kind && action.draftId === receipt.draftId
  )
    ? {
        draftId: receipt.draftId,
        outcome: receipt.outcome,
        ...(kind !== "draft-masking-rule" && receipt.created === true
          ? { created: true as const }
          : {}),
      }
    : undefined;

const actionReceiptMatchesChat = (
  receipt: AgentActionReceipt,
  chat: Pick<AgentChat, "workspaceName" | "context"> | undefined
) =>
  receipt.kind === "masking-rule" && chat?.workspaceName
    ? receipt.policy === `${chat.workspaceName}/policies/MASKING_RULE`
    : receipt.kind === "masking-exemption" &&
        chat?.context?.startsWith("projects/")
      ? receipt.policy === `${chat.context}/policies/MASKING_EXEMPTION`
      : true;

const normalizeMessage = (
  raw: unknown,
  chatId: string,
  fallbackTs: number,
  ownerEmail?: string,
  receiptOwner?: Pick<AgentChat, "workspaceName" | "context">
): AgentMessage => {
  const message = isRecord(raw) ? raw : {};
  const metadata = isRecord(message.metadata) ? message.metadata : null;
  const resourceReferences = metadata
    ? normalizeAgentResourceReferences(
        metadata.resourceReferences,
        metadata.resourceReference
      )
    : [];
  const actions = normalizeAgentResponseActions(metadata?.actions);
  const semanticTypeReceipt = normalizeDraftHandoff(
    metadata?.semanticTypeDraftHandoff,
    "draft-semantic-type",
    message.role,
    actions
  );
  const receipts =
    message.role === "assistant"
      ? normalizeActionReceipts(
          metadata?.actionReceipts,
          metadata as AgentMessageMetadata | undefined,
          ownerEmail
        )?.filter((receipt) => actionReceiptMatchesChat(receipt, receiptOwner))
      : undefined;
  return {
    id: typeof message.id === "string" ? message.id : uuidv4(),
    chatId:
      typeof message.chatId === "string"
        ? message.chatId
        : typeof message.threadId === "string"
          ? message.threadId
          : chatId,
    createdTs:
      typeof message.createdTs === "number" ? message.createdTs : fallbackTs,
    role:
      message.role === "assistant" ||
      message.role === "system" ||
      message.role === "tool"
        ? message.role
        : "user",
    content:
      typeof message.content === "string"
        ? metadata?.queryResultRetention &&
          isRecord(metadata.queryResultRetention) &&
          metadata.queryResultRetention.version === 1 &&
          metadata.queryResultRetention.level === "omitted"
          ? omitAgentQueryRows(message.content)
          : message.content
        : undefined,
    toolCalls: Array.isArray(message.toolCalls)
      ? (message.toolCalls as ToolCall[])
      : undefined,
    toolCallId:
      typeof message.toolCallId === "string" ? message.toolCallId : undefined,
    metadata: metadata
      ? {
          ...(metadata as AgentMessage["metadata"]),
          submittedAccessGrants: undefined,
          actionReceipts: receipts?.length ? receipts : undefined,
          followUpSourceId: normalizeFollowUpSourceId(
            metadata.followUpSourceId
          ),
          interruptionReason: normalizeInterruptionReason(
            metadata.interruptionReason
          ),
          handoff: normalizeAgentHandoff(metadata.handoff),
          actions: actions.length > 0 ? actions : undefined,
          maskingRuleDraftHandoff: normalizeDraftHandoff(
            metadata.maskingRuleDraftHandoff,
            "draft-masking-rule",
            message.role,
            actions
          ),
          maskingExemptionDraftHandoff: normalizeDraftHandoff(
            metadata.maskingExemptionDraftHandoff,
            "draft-masking-exemption",
            message.role,
            actions
          ),
          semanticTypeDraftHandoff:
            semanticTypeReceipt?.outcome === "opened"
              ? { ...semanticTypeReceipt, outcome: "opened" }
              : undefined,
          resourceReference:
            metadata.resourceReferences === undefined
              ? resourceReferences[0]
              : undefined,
          resourceReferences:
            metadata.resourceReferences !== undefined &&
            resourceReferences.length > 0
              ? resourceReferences
              : undefined,
        }
      : undefined,
  };
};

const normalizePendingAsk = (
  raw: unknown,
  context?: string,
  messages: readonly AgentMessage[] = []
): AgentPendingAsk | null => {
  const pendingAsk = isRecord(raw) ? raw : {};
  if (
    typeof pendingAsk.toolCallId !== "string" ||
    typeof pendingAsk.prompt !== "string"
  ) {
    return null;
  }

  // Saved runtime questions need an unambiguous origin, not just a matching option value.
  const legacyProjectInput = (() => {
    if (
      pendingAsk.kind !== "choose" ||
      context?.startsWith(projectNamePrefix)
    ) {
      return false;
    }
    const userIndex = messages.findLastIndex(
      (message) => message.role === "user"
    );
    if (userIndex < 0) return false;
    const turn = messages.slice(userIndex + 1);
    const calls = turn.flatMap((message) =>
      message.role === "assistant"
        ? (message.toolCalls ?? []).filter(
            (call) => isRecord(call) && call.id === pendingAsk.toolCallId
          )
        : []
    );
    if (
      calls.length !== 1 ||
      calls[0].name !== "ask_masking_rule_scope" ||
      typeof calls[0].arguments !== "string" ||
      turn.some(
        (message) =>
          message.role === "tool" &&
          message.toolCallId === pendingAsk.toolCallId
      )
    ) {
      return false;
    }
    try {
      const args: unknown = JSON.parse(calls[0].arguments);
      return (
        isRecord(args) &&
        Object.entries(args).every(
          ([key, value]) =>
            (key === "project" || key === "database") &&
            typeof value === "string"
        ) &&
        (args.project === undefined ||
          (typeof args.project === "string" && !args.project.trim()))
      );
    } catch {
      return false;
    }
  })();

  const options = Array.isArray(pendingAsk.options)
    ? pendingAsk.options
        .map((option) => {
          const candidate = isRecord(option) ? option : {};
          const label =
            typeof candidate.label === "string"
              ? candidate.label.trim()
              : typeof candidate.value === "string"
                ? candidate.value.trim()
                : "";
          const value =
            typeof candidate.value === "string" ? candidate.value.trim() : "";
          if (!label || !value) {
            return null;
          }
          const customInput =
            candidate.kind === "custom-input" ||
            (legacyProjectInput &&
              value === "choose-project" &&
              !Object.hasOwn(candidate, "kind"));
          if (
            (candidate.kind != null && !customInput) ||
            (candidate.placeholder != null &&
              (!customInput || typeof candidate.placeholder !== "string"))
          ) {
            return null;
          }
          return {
            label,
            value,
            description:
              typeof candidate.description === "string" &&
              candidate.description.trim()
                ? candidate.description.trim()
                : undefined,
            ...(customInput ? { kind: "custom-input" as const } : {}),
            ...(customInput &&
            typeof candidate.placeholder === "string" &&
            candidate.placeholder.trim()
              ? { placeholder: candidate.placeholder.trim() }
              : {}),
          };
        })
        .filter(
          (option): option is NonNullable<typeof option> => option !== null
        )
    : [];

  const kind =
    pendingAsk.kind === "confirm"
      ? "confirm"
      : pendingAsk.kind === "choose" && options.length > 0
        ? "choose"
        : "input";
  const defaultResource = normalizeAgentResourceReference(
    pendingAsk.defaultResource
  );
  const hasBoundDefault =
    kind === "input" &&
    defaultResource?.kind === "database" &&
    defaultResource.name === pendingAsk.defaultValue &&
    (!context?.startsWith(projectNamePrefix) ||
      defaultResource.project === context);
  const resourceProject =
    typeof pendingAsk.resourceProject === "string" &&
    /^projects\/(?!-$)[A-Za-z0-9_-]+$/.test(pendingAsk.resourceProject) &&
    (!context?.startsWith(projectNamePrefix) ||
      pendingAsk.resourceProject === context)
      ? pendingAsk.resourceProject
      : undefined;
  const apiPermission = isRecord(pendingAsk.apiPermission)
    ? pendingAsk.apiPermission
    : undefined;
  const expiresAt = apiPermission?.expiresAt;

  return {
    toolCallId: pendingAsk.toolCallId,
    prompt: pendingAsk.prompt,
    kind,
    ...(resourceProject ? { resourceProject } : {}),
    ...(kind === "confirm" &&
    typeof apiPermission?.actionId === "string" &&
    typeof apiPermission.operationId === "string"
      ? {
          apiPermission: {
            actionId: apiPermission.actionId,
            operationId: apiPermission.operationId,
            ...(typeof expiresAt === "number" &&
            Number.isFinite(expiresAt) &&
            expiresAt > 0
              ? { expiresAt }
              : {}),
          },
        }
      : {}),
    defaultValue:
      typeof pendingAsk.defaultValue === "string"
        ? pendingAsk.defaultValue
        : undefined,
    ...(hasBoundDefault ? { defaultResource } : {}),
    confirmLabel:
      typeof pendingAsk.confirmLabel === "string"
        ? pendingAsk.confirmLabel
        : undefined,
    cancelLabel:
      typeof pendingAsk.cancelLabel === "string"
        ? pendingAsk.cancelLabel
        : undefined,
    options: kind === "choose" ? options : undefined,
  };
};

const normalizeChat = (raw: unknown): AgentChat => {
  const chat = isRecord(raw) ? raw : {};
  const now = Date.now();
  const status =
    chat.status === "running" ||
    chat.status === "awaiting_user" ||
    chat.status === "paused" ||
    chat.status === "error"
      ? chat.status
      : DEFAULT_CHAT_STATUS;
  return {
    id: typeof chat.id === "string" ? chat.id : uuidv4(),
    title: typeof chat.title === "string" ? chat.title : "",
    context:
      typeof chat.context === "string" && chat.context.trim()
        ? chat.context
        : undefined,
    workspaceName:
      typeof chat.workspaceName === "string" && chat.workspaceName.trim()
        ? chat.workspaceName
        : undefined,
    createdTs: typeof chat.createdTs === "number" ? chat.createdTs : now,
    updatedTs: typeof chat.updatedTs === "number" ? chat.updatedTs : now,
    status,
    totalTokensUsed:
      typeof chat.totalTokensUsed === "number" &&
      Number.isFinite(chat.totalTokensUsed) &&
      chat.totalTokensUsed >= 0
        ? chat.totalTokensUsed
        : 0,
    hasUnreportedUsage: chat.hasUnreportedUsage === true,
    legacyUsage:
      isRecord(chat.legacyUsage) &&
      typeof chat.legacyUsage.totalTokensUsed === "number" &&
      Number.isSafeInteger(chat.legacyUsage.totalTokensUsed) &&
      chat.legacyUsage.totalTokensUsed >= 0
        ? {
            totalTokensUsed: chat.legacyUsage.totalTokensUsed,
            hasUnreportedUsage: chat.legacyUsage.hasUnreportedUsage === true,
          }
        : undefined,
    modelAttempts: Array.isArray(chat.modelAttempts)
      ? mergeModelAttempts(chat.modelAttempts as AgentModelAttempt[])
      : undefined,
    apiAttempts: normalizeAgentApiAttempts(chat.apiAttempts, String(chat.id)),
    runBudget: normalizeRunBudget(chat.runBudget),
    runCheckpoint: normalizeRunCheckpoint(chat.runCheckpoint),
    lastRunUsage:
      isRecord(chat.lastRunUsage) &&
      typeof chat.lastRunUsage.runId === "string" &&
      Array.isArray(chat.lastRunUsage.calls)
        ? {
            runId: chat.lastRunUsage.runId,
            calls: chat.lastRunUsage.calls.flatMap((call) => {
              const usage = normalizeModelCallUsage(call);
              return usage ? [usage] : [];
            }),
          }
        : undefined,
    page:
      isRecord(chat.page) &&
      typeof chat.page.path === "string" &&
      typeof chat.page.title === "string"
        ? {
            path: chat.page.path,
            title: chat.page.title,
          }
        : undefined,
    archived: Boolean(chat.archived),
    lastError:
      typeof chat.lastError === "string"
        ? chat.lastError
        : chat.lastError === null
          ? null
          : null,
    requiresAIConfiguration: Boolean(chat.requiresAIConfiguration),
    interrupted: Boolean(chat.interrupted),
    interruptionReason: chat.interrupted
      ? (normalizeInterruptionReason(chat.interruptionReason) ?? "unexpected")
      : undefined,
    runId: typeof chat.runId === "string" ? chat.runId : null,
    runMode:
      chat.runMode === "read-only" || chat.runMode === "default"
        ? chat.runMode
        : undefined,
    ...(chat.capability !== undefined
      ? { capability: normalizeAgentDiagnosticCapability(chat.capability) }
      : {}),
  };
};

function applyChatAccounting(chat: AgentChat) {
  const attempts = mergeModelAttempts(chat.modelAttempts);
  chat.modelAttempts = attempts;
  chat.legacyUsage ??= {
    totalTokensUsed: chat.totalTokensUsed,
    hasUnreportedUsage: Boolean(
      chat.hasUnreportedUsage || chat.totalTokensUsed > 0
    ),
  };
  Object.assign(chat, projectChatUsage(chat.legacyUsage, attempts));
  if (chat.runBudget) {
    chat.runBudget = projectRunBudget(chat.runBudget, attempts);
    chat.hasUnreportedUsage ||= chat.runBudget.hasUnreportedUsage;
  }
  const runId = chat.lastRunUsage?.runId;
  if (runId && attempts.some((attempt) => attempt.runId === runId)) {
    chat.lastRunUsage = {
      runId,
      calls: attempts
        .filter(
          (attempt) =>
            attempt.runId === runId &&
            attempt.outcome === "response" &&
            attempt.usage
        )
        .sort((a, b) => a.ordinal - b.ordinal)
        .map((attempt) => ({
          ...attempt.usage!,
          attemptId: attempt.attemptId,
        })),
    };
  }
}

function isValidChatCheckpoint(
  chat: AgentChat,
  messages: AgentMessage[],
  checkpoint: AgentRunCheckpoint
) {
  const root = messages.findIndex(
    (message) =>
      message.id === checkpoint.rootUserMessageId && message.role === "user"
  );
  const through = messages.findIndex(
    (message) => message.id === checkpoint.throughMessageId
  );
  return (
    !chat.archived &&
    chat.runId === checkpoint.runId &&
    chat.runBudget?.runId === checkpoint.runId &&
    root >= 0 &&
    through >= root &&
    samePersistedValue(chat.capability, checkpoint.bindings.capability)
  );
}

const sortMessages = (messages: AgentMessage[]) => {
  messages.sort((a, b) => a.createdTs - b.createdTs);
  return messages;
};

const createEmptyPersistedState = (): PersistedAgentState => ({
  currentChatId: null,
  chats: [],
  messagesByChatId: {},
  pendingAskByChatId: {},
  draftsByChatId: {},
  queuedUserMessagesByChatId: {},
});

const normalizePersistedState = (
  raw: unknown,
  recoverInterruptedRuns = true,
  preserveActiveFollowUpOwners = false,
  scopeKey?: string,
  userEmail?: string
): PersistedAgentState => {
  if (!isRecord(raw)) {
    return createEmptyPersistedState();
  }

  const rawChats = Array.isArray(raw.chats)
    ? raw.chats
    : Array.isArray(raw.threads)
      ? raw.threads
      : [];
  const rawMessagesByChatId = isRecord(raw.messagesByChatId)
    ? raw.messagesByChatId
    : isRecord(raw.messagesByThreadId)
      ? raw.messagesByThreadId
      : {};
  const rawPendingAskByChatId = isRecord(raw.pendingAskByChatId)
    ? raw.pendingAskByChatId
    : isRecord(raw.pendingAskByThreadId)
      ? raw.pendingAskByThreadId
      : {};
  const rawDraftsByChatId = isRecord(raw.draftsByChatId)
    ? raw.draftsByChatId
    : {};

  // Invalid capabilities must not restore as unrestricted chats.
  if (
    rawChats.some(
      (c) =>
        isRecord(c) &&
        c.apiAttempts !== undefined &&
        (typeof c.id !== "string" ||
          !c.id ||
          (c.capability !== undefined &&
            !normalizeAgentDiagnosticCapability(c.capability)))
    )
  ) {
    throw new Error("Cannot restore the owner of an Agent API journal");
  }
  const chats = rawChats
    .filter(
      (chat) =>
        !isRecord(chat) ||
        chat.capability === undefined ||
        normalizeAgentDiagnosticCapability(chat.capability) !== undefined
    )
    .map((chat) => normalizeChat(chat));
  const ownerIds = new Set(chats.map((chat) => chat.id));
  for (const buckets of [raw.messagesByChatId, raw.messagesByThreadId]) {
    if (!isRecord(buckets)) continue;
    for (const [id, rows] of Object.entries(buckets)) {
      if (
        Array.isArray(rows) &&
        rows.some(
          (row) =>
            isRecord(row) &&
            isRecord(row.metadata) &&
            "apiAttemptId" in row.metadata &&
            (!ownerIds.has(id) ||
              buckets !== rawMessagesByChatId ||
              (row.chatId !== undefined && row.chatId !== id) ||
              (row.threadId !== undefined && row.threadId !== id))
        )
      )
        throw new Error("Cannot restore the owner of an Agent API result");
    }
  }
  const messagesByChatId: Record<string, AgentMessage[]> = {};
  const pendingAskByChatId: Record<string, AgentPendingAsk> = {};
  const draftsByChatId: Record<string, AgentDraft> = {};
  const rawFollowUpQueues = isRecord(raw.followUpQueuesByChatId)
    ? raw.followUpQueuesByChatId
    : {};
  const rawStartupMessages = isRecord(raw.queuedUserMessagesByChatId)
    ? raw.queuedUserMessagesByChatId
    : {};
  const rawActiveChatIds = Array.isArray(raw.activeChatIds)
    ? raw.activeChatIds.filter((id): id is string => typeof id === "string")
    : [];
  const activeFollowUpOwners = new Set<string>();

  for (const chat of chats) {
    const rawMessages: unknown[] = Array.isArray(rawMessagesByChatId[chat.id])
      ? (rawMessagesByChatId[chat.id] as unknown[])
      : [];
    const messages = sortMessages(
      rawMessages.map((message, index) =>
        normalizeMessage(
          message,
          chat.id,
          chat.createdTs + index,
          userEmail,
          chat
        )
      )
    );
    if (userEmail && scopeKey?.startsWith(AGENT_RUN_STATE_KEY + ".")) {
      const legacyKey = scopeKey.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY);
      for (const attempt of chat.apiAttempts ?? []) {
        if (
          attempt.scopeKey === legacyKey &&
          attempt.userName === `users/${userEmail}`
        )
          attempt.scopeKey = scopeKey;
      }
    }
    chat.apiAttempts = reconcileApiResultLinks(
      chat.id,
      scopeKey,
      chat.apiAttempts,
      messages
    );
    if (chat.legacyUsage || chat.modelAttempts || chat.runBudget)
      applyChatAccounting(chat);
    if (
      chat.runCheckpoint &&
      !isValidChatCheckpoint(chat, messages, chat.runCheckpoint)
    )
      chat.runCheckpoint = undefined;
    if (
      (chat.status === "paused" || chat.runBudget) &&
      (!chat.runCheckpoint || chat.runBudget?.runId !== chat.runId)
    ) {
      if (chat.runId || chat.status === "paused") {
        chat.status = DEFAULT_CHAT_STATUS;
        chat.interrupted = true;
        chat.interruptionReason = "unexpected";
      }
      chat.runCheckpoint = undefined;
    }

    const firstUserMessage = messages.find(
      (message) => message.role === "user"
    );
    const lastMessage = messages.at(-1);

    if (!chat.title) {
      chat.title = getChatTitleFromMessage(firstUserMessage?.content);
    }
    if (lastMessage) {
      chat.updatedTs = Math.max(chat.updatedTs, lastMessage.createdTs);
    }
    const followUps = rawFollowUpQueues[chat.id];
    const startup = rawStartupMessages[chat.id];
    if (
      preserveActiveFollowUpOwners &&
      !chat.archived &&
      ((isRecord(startup) && startup.fromFollowUpQueue === true) ||
        (isRecord(followUps) &&
          Array.isArray(followUps.messages) &&
          followUps.messages.length > 0 &&
          (chat.status === "running" ||
            chat.status === "awaiting_user" ||
            rawActiveChatIds.includes(chat.id))))
    )
      activeFollowUpOwners.add(chat.id);
    if (
      recoverInterruptedRuns &&
      chat.status === "running" &&
      !activeFollowUpOwners.has(chat.id)
    ) {
      chat.status = DEFAULT_CHAT_STATUS;
      chat.interrupted = true;
      chat.interruptionReason = "unexpected";
      chat.lastError = null;
      chat.requiresAIConfiguration = false;
      if (chat.runCheckpoint) chat.runCheckpoint.suspension = "interrupted";
    }
    if (chat.interrupted) {
      setLastUserInterruptionReason(
        messages,
        chat.interruptionReason ?? "unexpected"
      );
    }

    const pendingAsk = normalizePendingAsk(
      rawPendingAskByChatId[chat.id],
      getCurrentTurnResourceReferences(messages).at(-1)?.project ??
        chat.context,
      messages
    );
    if (chat.status === "awaiting_user") {
      if (pendingAsk) {
        pendingAskByChatId[chat.id] = pendingAsk;
      } else if (recoverInterruptedRuns) {
        chat.status = DEFAULT_CHAT_STATUS;
      }
    }

    messagesByChatId[chat.id] = messages;
    const draft = normalizeDraft(rawDraftsByChatId[chat.id]);
    if (
      draft &&
      (draft.text || draft.project || draft.resource || draft.mentions)
    ) {
      draftsByChatId[chat.id] = draft;
    }
  }

  const chatIds = new Set(chats.map((chat) => chat.id));
  const queuedUserMessagesByChatId: Record<string, QueuedUserMessage> = {};
  for (const [chatId, value] of Object.entries(rawStartupMessages)) {
    const message = normalizeQueuedUserMessage(value, chatIds);
    if (message?.chatId === chatId) {
      queuedUserMessagesByChatId[chatId] = message;
    }
  }
  const legacyQueuedMessage = normalizeQueuedUserMessage(
    raw.queuedUserMessage,
    chatIds
  );
  if (legacyQueuedMessage) {
    queuedUserMessagesByChatId[legacyQueuedMessage.chatId] ??=
      legacyQueuedMessage;
  }

  const followUpQueuesByChatId: Record<string, AgentFollowUpQueue> = {};
  if (Object.keys(rawFollowUpQueues).length) {
    for (const chat of chats) {
      const queue = rawFollowUpQueues[chat.id];
      if (chat.archived || !isRecord(queue) || !Array.isArray(queue.messages))
        continue;
      const seen = new Set(messagesByChatId[chat.id]?.map(({ id }) => id));
      const startup = queuedUserMessagesByChatId[chat.id];
      if (startup) seen.add(startup.id);
      const messages: QueuedUserMessage[] = [];
      for (const value of queue.messages) {
        const message = normalizeQueuedUserMessage(value, chatIds);
        if (message?.chatId !== chat.id || seen.has(message.id)) continue;
        seen.add(message.id);
        messages.push(message);
      }
      if (messages.length)
        followUpQueuesByChatId[chat.id] = {
          messages,
          paused:
            chat.archived ||
            (recoverInterruptedRuns && !activeFollowUpOwners.has(chat.id)) ||
            queue.paused !== false,
        };
    }
  }

  if (recoverInterruptedRuns && !preserveActiveFollowUpOwners) {
    for (const chat of chats) {
      const startup = queuedUserMessagesByChatId[chat.id];
      if (!startup?.fromFollowUpQueue) continue;
      const messages = messagesByChatId[chat.id];
      const recorded = messages.some(({ id }) => id === startup.id);
      if (!recorded && !chat.archived) {
        restorePromotedFollowUp(
          { queuedUserMessagesByChatId, followUpQueuesByChatId },
          startup
        );
      } else {
        delete queuedUserMessagesByChatId[chat.id];
        if (chat.status === "idle" && !chat.archived && !chat.interrupted) {
          chat.interrupted = true;
          chat.interruptionReason = "unexpected";
          setLastUserInterruptionReason(messages, "unexpected");
        }
      }
    }
  }

  const hasCurrentChatId = Object.prototype.hasOwnProperty.call(
    raw,
    "currentChatId"
  );
  const hasCurrentThreadId = Object.prototype.hasOwnProperty.call(
    raw,
    "currentThreadId"
  );
  const persistedCurrentChatId = hasCurrentChatId
    ? raw.currentChatId
    : hasCurrentThreadId
      ? raw.currentThreadId
      : undefined;
  const currentChatId =
    persistedCurrentChatId === null
      ? null
      : typeof persistedCurrentChatId === "string" &&
          chats.some((chat) => chat.id === persistedCurrentChatId)
        ? persistedCurrentChatId
        : (chats[0]?.id ?? null);

  return {
    currentChatId,
    chats,
    messagesByChatId,
    pendingAskByChatId,
    draftsByChatId,
    queuedUserMessagesByChatId,
    ...(Object.keys(followUpQueuesByChatId).length
      ? { followUpQueuesByChatId }
      : {}),
    activeChatIds: Array.isArray(raw.activeChatIds)
      ? rawActiveChatIds.filter(
          (id) =>
            chatIds.has(id) &&
            (!recoverInterruptedRuns || activeFollowUpOwners.has(id))
        )
      : undefined,
  };
};

// --- Zustand store types ---

export interface AgentState {
  recoverLegacyApiAttempts: (
    chatId: string,
    lease?: AgentRunLease
  ) => Promise<boolean>;
  getApiAttemptToolResult: (
    chatId: string,
    attemptId: string
  ) => AgentMessage | undefined;
  captureApiScope: () => () => boolean;
  createApiDispatchContext: (
    identity: Omit<AgentApiActionIdentity, "scopeKey" | "userName">
  ) => AgentApiDispatchContext | null;
  beginAgentApiAttempt: AgentApiDispatchContext["beginAgentApiAttempt"];
  settleAgentApiAttempt: AgentApiDispatchContext["settleAgentApiAttempt"];
  // Window state
  visible: boolean;
  size: { width: number; height: number };
  sidebarWidth: number;
  minimized: boolean;

  preferences: AgentPreferences;
  preferencesStorageError: boolean;
  updatePreferences: (patch: Partial<AgentPreferences>) => void;
  resetPreferences: () => void;
  syncPreferences: () => void;

  // Chat state
  workspaceName: string | null;
  userEmail: string | null;
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  landingDraft: AgentDraft;
  landingContext: string | undefined;
  currentChatId: string | null;
  abortControllersByChatId: Record<string, AbortController>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  followUpQueuesByChatId: Record<string, AgentFollowUpQueue>;
  followUpDrainChatIds: Record<string, boolean>;
  queuedUserMessageInFlightByChatId: Record<string, boolean>;
  remoteBusyChatIds: Record<string, boolean>;
  chatProtectionReady: boolean;
  refreshChatProtection: () => void;
  releaseChatProtection: () => void;
  storagePressure?: AgentStoragePressure;
  // Volatile owner identity; failed history loading must not disable other owners.
  initializationFailureIdentity?: string;
  expressionDraftTargetsByChatId: Record<
    string,
    RegisteredExpressionDraftTarget
  >;
  sqlFixSourcesByChatId: Record<string, AgentSqlFixSource>;
  sqlFixDraftsById: Record<string, AgentSqlFixDraft>;
  setSqlFixSource: (chatId: string, source: AgentSqlFixSource) => void;
  prepareSqlFixDraft: (
    chatId: string,
    answer: string
  ) => Extract<AgentResponseAction, { kind: "review-sql-fix" }> | undefined;

  // UI actions
  open: () => void;
  toggle: () => void;
  minimize: () => void;
  restore: () => void;
  setSize: (width: number, height: number) => void;
  setSidebarWidth: (width: number) => void;

  // Chat CRUD
  createChat: (options?: CreateChatOptions) => AgentChat;
  queueUserMessage: (message: QueueUserMessageInput) => void;
  enqueueFollowUp: (message: QueueUserMessageInput) => string | null;
  removeFollowUp: (chatId: string, messageId: string) => boolean;
  setFollowUpQueuePaused: (chatId: string, paused: boolean) => boolean;
  setFollowUpDrainActive: (chatId: string, active: boolean) => boolean;
  promoteFollowUp: (
    chatId: string,
    messageId: string
  ) => QueuedUserMessage | null;
  takeQueuedUserMessage: (chatId: string) => QueuedUserMessage | null;
  finishQueuedUserMessage: (chatId: string) => void;
  commitQueuedUserMessage: (
    chatId: string
  ) => { queued: QueuedUserMessage; message: AgentMessage } | null;
  getDraft: (chatId?: string | null) => AgentDraft;
  setDraft: (chatId: string, draft: AgentDraft) => void;
  clearDraft: (chatId: string) => void;
  setLandingDraft: (draft: AgentDraft, context?: string) => void;
  clearLandingDraft: (context?: string) => void;
  ensureCurrentChat: (
    page?: AgentChatSnapshot,
    context?: string,
    workspaceName?: string
  ) => AgentChat;
  selectChatContext: (
    context: string,
    page?: AgentChatSnapshot,
    workspaceName?: string
  ) => AgentChat | null;
  selectOrCreateChatForContext: (
    context: string,
    page?: AgentChatSnapshot,
    workspaceName?: string
  ) => AgentChat | null;
  setCurrentChat: (chatId: string) => void;
  clearCurrentChat: () => void;
  updateChatPage: (chatId: string, page: AgentChatSnapshot) => AgentChat | null;
  renameChat: (chatId: string, title: string) => AgentChat | null;
  archiveChat: (chatId: string) => AgentChat | null;
  archiveAllActiveChats: () => number;
  unarchiveChat: (chatId: string) => AgentChat | null;
  unarchiveAllArchivedChats: () => number;
  deleteChat: (chatId: string) => boolean;
  deleteAllArchivedChats: () => number;
  registerExpressionDraftTarget: (
    chatId: string,
    target: AgentExpressionDraftTarget,
    options?: { allowRaw?: boolean }
  ) => () => void;
  canApplyExpressionDraft: (
    chatId: string,
    mode: AgentExpressionDraftMode
  ) => boolean;
  applyExpressionDraft: (
    chatId: string,
    expression: string,
    mode: AgentExpressionDraftMode
  ) => Promise<boolean>;

  // Messages
  addMessage: (message: AddMessageOptions) => AgentMessage | null;
  removeMessagesByRunId: (
    chatId: string,
    runId?: string | null
  ) => AgentMessage[];
  appendToolCall: (
    chatId: string,
    messageId: string,
    toolCall: ToolCall
  ) => AgentMessage | null;
  appendContinueActions: (
    chatId: string,
    messageId: string,
    actions: Extract<AgentResponseAction, { kind: "continue" }>[],
    references: AgentResourceReference[]
  ) => AgentMessage | null;
  recordMaskingRuleDraftHandoff: (
    chatId: string,
    messageId: string,
    draftId: string,
    outcome: "opened" | "applied"
  ) => AgentMessage | null;
  recordMaskingExemptionDraftHandoff: AgentState["recordMaskingRuleDraftHandoff"];
  recordSemanticTypeDraftHandoff: (
    chatId: string,
    messageId: string,
    draftId: string,
    outcome: "opened" | "created"
  ) => AgentMessage | null;
  markMaskingExemptionDraftCreated: (
    chatId: string,
    messageId: string,
    draftId: string
  ) => AgentMessage | null;
  recordSubmittedAccessGrant: (
    chatId: string,
    messageId: string,
    grant: AccessGrant,
    origin: {
      workspaceName: string;
      userEmail: string;
      draft: AccessGrantDraft;
    }
  ) => AgentMessage | null;
  recordActionReceipt: (
    chatId: string,
    messageId: string,
    receipt: AgentActionReceipt,
    origin: AgentActionOrigin
  ) => AgentMessage | null;
  recordActionObservation: (
    chatId: string,
    messageId: string,
    receiptKey: string,
    observation: AgentActionObservation,
    origin: AgentActionOrigin
  ) => AgentMessage | null;

  // Status
  setChatStatus: (
    chatId: string,
    status: AgentChatStatus,
    options?: {
      interrupted?: boolean;
      interruptionReason?: AgentInterruptionReason;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
      page?: AgentChatSnapshot;
    }
  ) => AgentChat | null;
  startChatRun: (
    chatId: string,
    page?: AgentChatSnapshot,
    options?: { runId?: string; runMode?: AgentRunMode }
  ) => void;
  beginChatRun: (
    chatId: string,
    rootUserMessageId: string,
    bindings: AgentRunBindings
  ) => Promise<string | null>;
  continueChatRun: (chatId: string, runId: string) => Promise<boolean>;
  retryChatRunStartup: (chatId: string, runId: string) => Promise<boolean>;
  saveChatRunCheckpoint: (
    chatId: string,
    checkpoint: AgentRunCheckpoint
  ) => Promise<boolean>;
  reserveChatRunAttempt: (
    chatId: string,
    runId: string
  ) => Promise<AgentAttemptReservation>;
  recordChatModelAttempt: (
    chatId: string,
    attempt: AgentModelAttempt
  ) => Promise<boolean>;
  pauseChatRun: (
    chatId: string,
    checkpoint: AgentRunCheckpoint
  ) => Promise<boolean>;
  increaseRunTokenLimit: (chatId: string, limit: number) => Promise<boolean>;
  endPausedRun: (chatId: string) => Promise<boolean>;
  flushRunPersistence: () => Promise<boolean>;
  checkAgentStorageAdmission: () => Promise<boolean>;
  freeAgentStorage: (options?: {
    resetCurrentChat?: boolean;
  }) => Promise<boolean>;
  getRunPersistenceError: () => unknown;
  canUseAgentExecution: () => boolean;
  finishChatRun: (
    chatId: string,
    options?: {
      status?: Extract<AgentChatStatus, "idle" | "error">;
      lastError?: string | null;
      requiresAIConfiguration?: boolean;
    }
  ) => void;
  interruptChatRun: (
    chatId: string,
    page?: AgentChatSnapshot,
    reason?: AgentInterruptionReason
  ) => void;
  recoverChatRun: (chatId: string, runId: string | null) => Promise<boolean>;

  // Pending ask
  setPendingAsk: (
    chatId: string,
    pendingAsk: AgentPendingAsk
  ) => AgentPendingAsk;
  clearPendingAsk: (chatId?: string | null) => void;
  awaitUser: (chatId: string, pendingAsk: AgentPendingAsk) => AgentPendingAsk;
  recoverPendingAsk: (chatId: string, runId: string) => Promise<boolean>;
  cancelPendingAsk: (chatId?: string | null) => boolean;
  answerPendingAsk: (
    chatId: string,
    response: AgentAskUserResponse,
    metadata?: AgentMessage["metadata"]
  ) => AgentMessage | null;

  // Token
  incrementChatTotalTokens: (
    chatId: string,
    totalTokensUsed: number
  ) => AgentChat | null;
  recordChatModelCallUsage: (
    chatId: string,
    runId: string,
    usage: AgentModelCallUsage
  ) => void;

  // Abort
  setAbortController: (
    chatId: string,
    controller: AbortController | null
  ) => AbortController | null;
  getAbortController: (chatId?: string | null) => AbortController | null;
  cancel: (chatId?: string | null, reason?: AgentInterruptionReason) => void;

  // Error
  clearError: (chatId?: string | null) => void;

  // Persistence
  setWorkspaceName: (workspaceName?: string, userEmail?: string) => boolean;
  saveWindowState: () => void;
  loadWindowState: () => void;
  saveState: () => void;
  loadState: () => void;
  syncState: () => boolean;

  // Queries
  getChat: (chatId?: string | null) => AgentChat | null;
  getMessages: (chatId?: string | null) => AgentMessage[];
  getPendingAsk: (chatId?: string | null) => AgentPendingAsk | null;
  isChatRunning: (chatId?: string | null) => boolean;
  canSelectChat: (chatId?: string | null) => boolean;
  touchChat: (chatId: string) => AgentChat | null;
}

// --- Selector functions (derived values) ---

// Stable empty references — avoids creating new objects on every selector call,
// which would cause infinite re-render loops with useSyncExternalStore.
const EMPTY_MESSAGES: AgentMessage[] = [];
const EMPTY_CHATS: AgentChat[] = [];

// Memoised selector: only re-sorts when the chats array reference changes.
let _orderedChatsInput: AgentChat[] | null = null;
let _orderedChatsResult: AgentChat[] = EMPTY_CHATS;

export const selectOrderedChats = (state: AgentState): AgentChat[] => {
  if (state.chats !== _orderedChatsInput) {
    _orderedChatsInput = state.chats;
    _orderedChatsResult = [...state.chats].sort((a, b) => {
      if (b.updatedTs !== a.updatedTs) {
        return b.updatedTs - a.updatedTs;
      }
      return b.createdTs - a.createdTs;
    });
  }
  return _orderedChatsResult;
};

export const selectCurrentChat = (state: AgentState): AgentChat | null =>
  state.chats.find((chat) => chat.id === state.currentChatId) ?? null;

export const selectContinueChatForContext = (
  state: AgentState,
  context?: string,
  workspaceName?: string
): AgentChat | null => {
  if (!context || !workspaceName || state.workspaceName !== workspaceName) {
    return null;
  }
  return (
    selectOrderedChats(state).find(
      (chat) =>
        chat.context === context &&
        chat.workspaceName === workspaceName &&
        !chat.archived &&
        state.canSelectChat(chat.id)
    ) ?? null
  );
};

export const selectMessages = (state: AgentState): AgentMessage[] =>
  (state.currentChatId
    ? state.messagesByChatId[state.currentChatId]
    : undefined) ?? EMPTY_MESSAGES;

export const selectCurrentPendingAsk = (
  state: AgentState
): AgentPendingAsk | null =>
  (state.currentChatId
    ? state.pendingAskByChatId[state.currentChatId]
    : undefined) ?? null;

export const selectLoading = (state: AgentState): boolean =>
  selectCurrentChat(state)?.status === "running" ||
  Boolean(
    state.currentChatId && state.remoteBusyChatIds?.[state.currentChatId]
  );

export const selectError = (state: AgentState): string | null =>
  selectCurrentChat(state)?.lastError ?? null;

export const selectCurrentChatRequiresAIConfiguration = (
  state: AgentState
): boolean => selectCurrentChat(state)?.requiresAIConfiguration ?? false;

export const selectHasRunningChat = (state: AgentState): boolean =>
  state.chats.some((chat) => chat.status === "running");

// --- Store creation ---

const getSafeStorage = (): Storage | null => {
  try {
    const storage = globalThis.localStorage as Partial<Storage> | undefined;
    if (
      !storage ||
      typeof storage.getItem !== "function" ||
      typeof storage.setItem !== "function" ||
      typeof storage.removeItem !== "function"
    ) {
      return null;
    }
    // Browsers may expose Storage while denying access to its contents.
    storage.getItem(AGENT_STATE_KEY);
    return storage as Storage;
  } catch {
    return null;
  }
};

const loadAgentPreferences = (workspaceName: string, userEmail: string) => {
  const key = getAgentPreferencesKey(workspaceName, userEmail);
  try {
    const storage = getSafeStorage();
    if (!storage || !key)
      throw new Error("Agent preference storage unavailable");
    const serialized = storage.getItem(key);
    return {
      preferences: normalizeAgentPreferences(
        serialized ? JSON.parse(serialized) : null
      ),
      preferencesStorageError: false,
    };
  } catch {
    return {
      preferences: { ...DEFAULT_AGENT_PREFERENCES },
      preferencesStorageError: true,
    };
  }
};

const getInitialWindowState = () => {
  if (typeof window === "undefined") {
    return {
      size: { width: 480, height: 540 },
      sidebarWidth: 200,
    };
  }

  return getDefaultAgentWindowState(window.innerWidth, window.innerHeight);
};

const ensureCurrentChat = (state: PersistedAgentState): PersistedAgentState => {
  if (
    state.currentChatId === null
      ? state.chats.length > 0
      : state.chats.some((chat) => chat.id === state.currentChatId)
  ) {
    return state;
  }

  const defaultChat = createChatRecord();
  return {
    ...state,
    chats: [...state.chats, defaultChat],
    messagesByChatId: {
      ...state.messagesByChatId,
      [defaultChat.id]: [],
    },
    currentChatId: defaultChat.id,
  };
};

const readPersistedState = (
  saved: string | null | undefined,
  key: string,
  receiptScope = key,
  userEmail?: string
): PersistedAgentState | null => {
  if (saved == null) return null;
  return normalizePersistedState(
    parseAgentStoredState(saved),
    true,
    key.startsWith(`${AGENT_RUN_STATE_KEY}.`) && supportsAgentRunRecovery(),
    receiptScope,
    userEmail
  );
};

const serializePersistedState = (state: PersistedAgentState): string =>
  JSON.stringify({
    ...state,
    chats: state.chats.map((chat) =>
      chat.apiAttempts?.some((a) => a.opaqueReceipts)
        ? { ...chat, apiAttempts: chat.apiAttempts.flatMap(storedApiReceipts) }
        : chat
    ),
  });

const getPersistedFollowUpOwners = (state: PersistedAgentState) =>
  Object.fromEntries(
    state.chats
      .filter(
        (chat) =>
          !chat.archived &&
          (state.queuedUserMessagesByChatId[chat.id]?.fromFollowUpQueue ||
            (state.followUpQueuesByChatId?.[chat.id] &&
              (chat.status === "running" ||
                chat.status === "awaiting_user" ||
                state.activeChatIds?.includes(chat.id))))
      )
      .map((chat) => [chat.id, true])
  );

const selectPersistedChats = (
  state: PersistedAgentState,
  chats: AgentChat[]
): PersistedAgentState => {
  const ids = new Set(chats.map((chat) => chat.id));
  const followUpQueuesByChatId = Object.fromEntries(
    Object.entries(state.followUpQueuesByChatId ?? {}).filter(([id]) =>
      ids.has(id)
    )
  );
  return {
    currentChatId: ids.has(state.currentChatId ?? "")
      ? state.currentChatId
      : (chats[0]?.id ?? null),
    chats,
    messagesByChatId: Object.fromEntries(
      chats.map((chat) => [chat.id, state.messagesByChatId[chat.id] ?? []])
    ),
    pendingAskByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const ask = state.pendingAskByChatId[chat.id];
        return ask ? [[chat.id, ask]] : [];
      })
    ),
    draftsByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const draft = state.draftsByChatId[chat.id];
        return draft ? [[chat.id, draft]] : [];
      })
    ),
    queuedUserMessagesByChatId: Object.fromEntries(
      chats.flatMap((chat) => {
        const message = state.queuedUserMessagesByChatId[chat.id];
        return message ? [[chat.id, message]] : [];
      })
    ),
    ...(Object.keys(followUpQueuesByChatId).length
      ? { followUpQueuesByChatId }
      : {}),
  };
};

const mergePersistedStates = (
  existing: PersistedAgentState | null,
  incoming: PersistedAgentState
): PersistedAgentState => {
  if (!existing) return incoming;
  const byId = new Map(
    incoming.chats.map((chat) => [chat.id, { chat, state: incoming }])
  );
  for (const chat of existing.chats) {
    const previous = byId.get(chat.id);
    if (!previous || previous.chat.updatedTs <= chat.updatedTs) {
      byId.set(chat.id, { chat, state: existing });
    }
  }
  const chats = [...byId.values()].map(({ chat }) => chat);
  const merged: PersistedAgentState = {
    currentChatId:
      existing.currentChatId && byId.has(existing.currentChatId)
        ? existing.currentChatId
        : incoming.currentChatId,
    chats,
    messagesByChatId: {},
    pendingAskByChatId: {},
    draftsByChatId: {},
    queuedUserMessagesByChatId: {},
  };
  for (const [id, { state }] of byId) {
    merged.messagesByChatId[id] = state.messagesByChatId[id] ?? [];
    if (state.pendingAskByChatId[id]) {
      merged.pendingAskByChatId[id] = state.pendingAskByChatId[id];
    }
    if (state.draftsByChatId[id]) {
      merged.draftsByChatId[id] = state.draftsByChatId[id];
    }
    if (state.queuedUserMessagesByChatId[id]) {
      merged.queuedUserMessagesByChatId[id] =
        state.queuedUserMessagesByChatId[id];
    }
    const queue = state.followUpQueuesByChatId?.[id];
    if (queue) (merged.followUpQueuesByChatId ??= {})[id] = queue;
  }
  return merged;
};

const hasOnlyEmptyChat = (state: PersistedAgentState | null) => {
  if (!state || state.chats.length === 0) return true;
  if (
    state.chats.length !== 1 ||
    Object.keys(state.queuedUserMessagesByChatId).length > 0 ||
    Object.keys(state.followUpQueuesByChatId ?? {}).length > 0
  )
    return false;
  const chat = state.chats[0];
  return (
    !state.messagesByChatId[chat.id]?.length &&
    !state.pendingAskByChatId[chat.id] &&
    !state.draftsByChatId[chat.id]
  );
};

const isScopedAgentStateKey = (key: string) => {
  const suffix = key.slice(`${AGENT_STATE_KEY}.`.length);
  const separator = suffix.indexOf(":");
  if (separator < 1) return false;
  const workspaceIdLength = Number(suffix.slice(0, separator));
  if (!Number.isSafeInteger(workspaceIdLength) || workspaceIdLength < 1)
    return false;
  const remainder = suffix.slice(separator + 1);
  return (
    remainder[workspaceIdLength] === "." &&
    remainder.slice(workspaceIdLength + 1).includes("@")
  );
};

const migrateLegacyState = (
  storage: Storage | null,
  workspaceName: string,
  userEmail: string
): boolean => {
  if (
    !storage ||
    !getScopedAgentStateKey(AGENT_STATE_KEY, workspaceName, userEmail)
  )
    return true;
  try {
    const prefix = `${AGENT_STATE_KEY}.`;
    const keys = Array.from({ length: storage.length }, (_, index) =>
      storage.key(index)
    ).filter((key): key is string => Boolean(key));
    const sources = keys.flatMap((key) => {
      const globalSource =
        key === AGENT_STATE_KEY || key === LEGACY_AGENT_STATE_KEY;
      if (
        !globalSource &&
        (!key.startsWith(prefix) || isScopedAgentStateKey(key))
      ) {
        return [];
      }
      const saved = storage.getItem(key);
      if (!saved) return [];
      try {
        const raw = JSON.parse(saved);
        if (
          !isRecord(raw) ||
          (!Array.isArray(raw.chats) && !Array.isArray(raw.threads))
        ) {
          return [];
        }
        return [
          {
            key,
            workspaceName: globalSource
              ? null
              : `${workspaceNamePrefix}${key.slice(prefix.length)}`,
            state: normalizePersistedState(raw),
          },
        ];
      } catch {
        return [];
      }
    });
    if (sources.length === 0) return true;

    const owner = storage.getItem(LEGACY_AGENT_OWNER_KEY);
    if (owner !== null && owner !== userEmail) return true;
    if (owner === null) storage.setItem(LEGACY_AGENT_OWNER_KEY, userEmail);

    const incomingByWorkspace = new Map<string, PersistedAgentState>();
    const retainedSourceKeys = new Set<string>();
    for (const source of sources) {
      const chatsByWorkspace = new Map<string, AgentChat[]>();
      for (const chat of source.state.chats) {
        const chatWorkspaceName =
          source.workspaceName ??
          (chat.workspaceName?.startsWith(workspaceNamePrefix)
            ? chat.workspaceName
            : chat.context?.startsWith(workspaceNamePrefix)
              ? chat.context
              : workspaceName);
        const chats = chatsByWorkspace.get(chatWorkspaceName) ?? [];
        chats.push({ ...chat, workspaceName: chatWorkspaceName });
        chatsByWorkspace.set(chatWorkspaceName, chats);
      }
      for (const [name, chats] of chatsByWorkspace) {
        const v3Key = getScopedAgentStateKey(
          AGENT_RUN_STATE_KEY,
          name,
          userEmail
        );
        if (v3Key && storage.getItem(v3Key)) retainedSourceKeys.add(source.key);
        const incoming = selectPersistedChats(source.state, chats);
        incomingByWorkspace.set(
          name,
          mergePersistedStates(incomingByWorkspace.get(name) ?? null, incoming)
        );
      }
    }

    // Stage in v2 so v3 hydration and accounting use the serialized writer.
    for (const [name, incoming] of incomingByWorkspace) {
      const key = getScopedAgentStateKey(AGENT_STATE_KEY, name, userEmail);
      if (!key) throw new Error("Invalid legacy workspace");
      const existing = readPersistedState(storage.getItem(key), key);
      const merged = mergePersistedStates(existing, incoming);
      if (hasOnlyEmptyChat(existing)) {
        merged.currentChatId = incoming.currentChatId;
      }
      storage.setItem(key, serializePersistedState(merged));
    }
    for (const { key } of sources) {
      // An existing v3 cache takes precedence and does not import staged v2.
      if (!retainedSourceKeys.has(key)) storage.removeItem(key);
    }
    return true;
  } catch {
    // The owner marker prevents another account from claiming a partial migration.
    return false;
  }
};

const loadInitialState = (): {
  chats: AgentChat[];
  messagesByChatId: Record<string, AgentMessage[]>;
  pendingAskByChatId: Record<string, AgentPendingAsk>;
  draftsByChatId: Record<string, AgentDraft>;
  queuedUserMessagesByChatId: Record<string, QueuedUserMessage>;
  currentChatId: string | null;
} => {
  return ensureCurrentChat(createEmptyPersistedState());
};

const persistedSnapshot = (state: AgentState): PersistedAgentState => ({
  currentChatId: state.currentChatId,
  chats: state.chats,
  messagesByChatId: state.messagesByChatId,
  pendingAskByChatId: state.pendingAskByChatId,
  draftsByChatId: state.draftsByChatId,
  queuedUserMessagesByChatId: state.queuedUserMessagesByChatId,
  ...(Object.keys(state.followUpQueuesByChatId).length
    ? { followUpQueuesByChatId: state.followUpQueuesByChatId }
    : {}),
  activeChatIds: [
    ...new Set([
      ...Object.keys(state.abortControllersByChatId),
      ...Object.keys(state.followUpDrainChatIds),
    ]),
  ],
});

const samePersistedValue = (a: unknown, b: unknown) =>
  a === b || JSON.stringify(a) === JSON.stringify(b);

const currentScopeKey = (state: AgentState) =>
  state.workspaceName && state.userEmail
    ? getAgentStateKey(state.workspaceName, state.userEmail)
    : null;

const ownsApiScope = (state: AgentState, key: string) => {
  const { currentUser, workspace } = useAppStore.getState();
  return (
    currentScopeKey(state) === key &&
    !!workspace &&
    workspace.name === state.workspaceName &&
    currentUser?.workspace === workspace.name &&
    !!currentUser.email &&
    currentUser.email === state.userEmail
  );
};

let appApiScope = {};
let watchingAppApiScope = false;
function getAppApiScope() {
  if (!watchingAppApiScope) {
    // The app can switch ownership even when a running Agent chat prevents reloading its saved scope.
    useAppStore.subscribe((state, previous) => {
      if (
        state.workspace?.name !== previous.workspace?.name ||
        state.currentUser?.name !== previous.currentUser?.name ||
        state.currentUser?.email !== previous.currentUser?.email ||
        state.currentUser?.workspace !== previous.currentUser?.workspace
      )
        appApiScope = {};
    });
    watchingAppApiScope = true;
  }
  return appApiScope;
}

function insertApiToolResult(messages: AgentMessage[], result: AgentMessage) {
  const call = messages.findLastIndex(
    (m) =>
      m.role === "assistant" &&
      m.metadata?.runId === result.metadata?.runId &&
      m.toolCalls?.some((t) => t.id === result.toolCallId)
  );
  let index = call < 0 ? messages.length : call + 1;
  while (index < messages.length && messages[index].role === "tool") index++;
  messages.splice(
    index,
    0,
    index < messages.length && call >= 0
      ? { ...result, createdTs: messages[index - 1].createdTs }
      : result
  );
}
const canOwnRun = (state: AgentState, chatId: string) => {
  const key = currentScopeKey(state);
  if (!key) return false;
  if (supportsAgentRunRecovery()) return hasAgentRunLease(key, chatId);
  try {
    return !getSafeStorage()?.getItem(
      key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY)
    );
  } catch {
    return false;
  }
};

const draftShadowKey = (scopeKey: string) => `bb-agent-ui-v1.${scopeKey}`;
const saveDraftShadow = (state: AgentState) => {
  const key = currentScopeKey(state);
  if (!key || !supportsAgentRunRecovery()) return;
  try {
    sessionStorage.setItem(
      draftShadowKey(key),
      JSON.stringify({
        currentChatId: state.currentChatId,
        draftsByChatId: state.draftsByChatId,
      })
    );
  } catch {
    /* In-memory drafts remain available when browser storage is unavailable. */
  }
};
const restoreDraftShadow = (key: string, state: PersistedAgentState) => {
  if (!supportsAgentRunRecovery()) return state;
  try {
    const raw = JSON.parse(
      sessionStorage.getItem(draftShadowKey(key)) ?? "null"
    );
    if (!isRecord(raw)) return state;
    const ids = new Set(state.chats.map((chat) => chat.id));
    if (
      raw.currentChatId === null ||
      (typeof raw.currentChatId === "string" && ids.has(raw.currentChatId))
    )
      state.currentChatId = raw.currentChatId as string | null;
    if (isRecord(raw.draftsByChatId))
      for (const id of ids) {
        const draft = normalizeDraft(raw.draftsByChatId[id]);
        if (draft) state.draftsByChatId[id] = draft;
        else delete state.draftsByChatId[id];
      }
  } catch {
    /* Draft recovery never restores execution state. */
  }
  return state;
};

const mergeDraftHandoffs = (
  message: AgentMessage,
  other: AgentMessage | undefined
): AgentMessage => {
  if (!other || other.role !== "assistant" || other.chatId !== message.chatId)
    return message;
  let metadata = message.metadata;
  for (const [key, kind] of [
    ["maskingRuleDraftHandoff", "draft-masking-rule"],
    ["maskingExemptionDraftHandoff", "draft-masking-exemption"],
    ["semanticTypeDraftHandoff", "draft-semantic-type"],
  ] as const) {
    const incoming = normalizeDraftHandoff(
      other.metadata?.[key],
      kind,
      message.role,
      message.metadata?.actions ?? []
    );
    if (!incoming) continue;
    const current = normalizeDraftHandoff(
      metadata?.[key],
      kind,
      message.role,
      message.metadata?.actions ?? []
    );
    if (current && current.draftId !== incoming.draftId) continue;
    const receipt = current
      ? { ...current, ...(incoming.created ? { created: true as const } : {}) }
      : incoming;
    if (!samePersistedValue(metadata?.[key], receipt))
      metadata = { ...metadata, [key]: receipt };
  }
  if (message.role === "assistant") {
    const incoming = normalizeActionReceipts(
      other.metadata?.actionReceipts,
      other.metadata
    );
    const candidates = samePersistedValue(
      message.metadata?.accessRequestDraft,
      other.metadata?.accessRequestDraft
    )
      ? incoming
      : incoming?.filter((receipt) => receipt.kind !== "access-grant");
    const receipts = normalizeActionReceipts(
      mergeActionReceipts([
        ...(metadata?.actionReceipts ?? []),
        ...(candidates ?? []),
      ]),
      message.metadata
    );
    if (!samePersistedValue(metadata?.actionReceipts, receipts))
      metadata = { ...metadata, actionReceipts: receipts };
  }
  return metadata === message.metadata ? message : { ...message, metadata };
};

const chatExecution = (state: PersistedAgentState, chat: AgentChat) => {
  // Hydration must not look like remote progress just because it normalizes records.
  const messages = (state.messagesByChatId[chat.id] ?? []).map(
    (message, index) =>
      normalizeMessage(
        message,
        chat.id,
        chat.createdTs + index,
        undefined,
        chat
      )
  );
  const {
    title: _title,
    context: _context,
    workspaceName: _workspaceName,
    createdTs: _createdTs,
    updatedTs: _updatedTs,
    page: _page,
    archived: _archived,
    totalTokensUsed: _total,
    hasUnreportedUsage: _unreported,
    modelAttempts: _attempts,
    apiAttempts: _apiAttempts,
    legacyUsage: _legacy,
    lastRunUsage: _usage,
    runBudget: budget,
    ...execution
  } = normalizeChat(chat);
  return {
    ...execution,
    ...(!chat.runBudget && !chat.modelAttempts
      ? {
          totalTokensUsed: chat.totalTokensUsed,
          hasUnreportedUsage: chat.hasUnreportedUsage,
          lastRunUsage: chat.lastRunUsage,
        }
      : {}),
    runBudget: budget
      ? {
          ...budget,
          totalTokensUsed: 0,
          hasUnreportedUsage: false,
          limit: 0,
          revision: 0,
        }
      : undefined,
    messages: messages
      .filter((message) => !message.metadata?.apiAttemptId)
      .map((message) => {
        if (!message.metadata) return message;
        const {
          maskingRuleDraftHandoff: _ruleReceipt,
          maskingExemptionDraftHandoff: _exemptionReceipt,
          semanticTypeDraftHandoff: _typeReceipt,
          submittedAccessGrants: _accessGrantReceipts,
          actionReceipts: _actionReceipts,
          queryResultRetention: _queryRetention,
          ...metadata
        } = message.metadata;
        return { ...message, metadata };
      }),
    pendingAsk: normalizePendingAsk(
      state.pendingAskByChatId[chat.id],
      getCurrentTurnResourceReferences(messages).at(-1)?.project ??
        chat.context,
      messages
    ),
    queuedMessage: state.queuedUserMessagesByChatId[chat.id],
    active: state.activeChatIds?.includes(chat.id) ?? false,
  };
};

// Queue edits rebase independently of transcript ownership. Retain only local
// additions and removals so concurrent writes cannot restore an already sent item.
const rebaseFollowUpQueue = (
  base: AgentFollowUpQueue | undefined,
  local: AgentFollowUpQueue | undefined,
  remote: AgentFollowUpQueue | undefined,
  restoredHeadId?: string
): AgentFollowUpQueue | undefined => {
  if (samePersistedValue(base, local)) return remote;
  const previousIds = new Set(base?.messages.map(({ id }) => id));
  const localIds = new Set(local?.messages.map(({ id }) => id));
  const remoteIds = new Set(remote?.messages.map(({ id }) => id));
  const messages = (remote?.messages ?? []).filter(
    ({ id }) => !previousIds.has(id) || localIds.has(id)
  );
  for (const message of local?.messages ?? []) {
    if (!previousIds.has(message.id) && !remoteIds.has(message.id)) {
      if (message.id === restoredHeadId) messages.unshift(message);
      else messages.push(message);
    }
  }
  if (!messages.length) return undefined;
  let paused = remote?.paused ?? local?.paused ?? false;
  if (local && (base ? local.paused !== base.paused : local.paused))
    paused = local.paused;
  return { messages, paused };
};

// Rebase only this tab's edits; an untouched snapshot must not undo another tab.
const rebasePersistedChanges = (
  base: PersistedAgentState,
  local: PersistedAgentState,
  remote: PersistedAgentState,
  locallyBusy: Set<string>,
  remoteBusyChatIds: Record<string, boolean>
): PersistedAgentState => {
  const baseChats = new Map(base.chats.map((chat) => [chat.id, chat]));
  const localChats = new Map(local.chats.map((chat) => [chat.id, chat]));
  const remoteChats = new Map(remote.chats.map((chat) => [chat.id, chat]));
  const result = createEmptyPersistedState();
  result.activeChatIds = [];
  for (const id of new Set([...remoteChats.keys(), ...localChats.keys()])) {
    const previous = baseChats.get(id);
    const current = localChats.get(id);
    const incoming = remoteChats.get(id);
    if (previous && (!current || (!incoming && !locallyBusy.has(id)))) continue;
    const useLocal =
      current &&
      !remoteBusyChatIds[id] &&
      (!incoming ||
        !previous ||
        locallyBusy.has(id) ||
        !samePersistedValue(
          chatExecution(base, previous),
          chatExecution(local, current)
        ));
    const source = useLocal ? local : remote;
    const sourceChat = useLocal ? current : incoming;
    if (!sourceChat) continue;
    const chat = { ...sourceChat };
    chat.apiAttempts = mergeAgentApiAttempts(
      incoming?.apiAttempts,
      current?.apiAttempts
    );
    if (
      current?.modelAttempts ||
      incoming?.modelAttempts ||
      current?.legacyUsage ||
      incoming?.legacyUsage
    ) {
      chat.legacyUsage = incoming?.legacyUsage ??
        current?.legacyUsage ?? {
          totalTokensUsed: sourceChat.totalTokensUsed,
          hasUnreportedUsage: Boolean(
            sourceChat.hasUnreportedUsage || sourceChat.totalTokensUsed > 0
          ),
        };
      if (current?.legacyUsage && incoming?.legacyUsage) {
        chat.legacyUsage = {
          totalTokensUsed: Math.max(
            current.legacyUsage.totalTokensUsed,
            incoming.legacyUsage.totalTokensUsed
          ),
          hasUnreportedUsage:
            current.legacyUsage.hasUnreportedUsage ||
            incoming.legacyUsage.hasUnreportedUsage,
        };
      }
      chat.modelAttempts = mergeModelAttempts(
        incoming?.modelAttempts,
        current?.modelAttempts
      );
      if (
        chat.runBudget &&
        current?.runBudget?.runId === chat.runBudget.runId &&
        incoming?.runBudget?.runId === chat.runBudget.runId
      ) {
        chat.runBudget = {
          ...chat.runBudget,
          limit: Math.max(current.runBudget.limit, incoming.runBudget.limit),
          revision: Math.max(
            current.runBudget.revision,
            incoming.runBudget.revision
          ),
        };
      }
      applyChatAccounting(chat);
    }
    if (current?.capability || incoming?.capability) {
      chat.capability = current?.capability ?? incoming?.capability;
    }
    if (current && incoming && previous) {
      for (const key of [
        "title",
        "context",
        "workspaceName",
        "page",
        "archived",
      ] as const) {
        if (!samePersistedValue(current[key], previous[key])) {
          Object.assign(chat, { [key]: current[key] });
        } else {
          Object.assign(chat, { [key]: incoming[key] });
        }
      }
      chat.updatedTs =
        current.updatedTs === previous.updatedTs
          ? incoming.updatedTs
          : incoming.updatedTs === previous.updatedTs
            ? current.updatedTs
            : Math.max(current.updatedTs, incoming.updatedTs);
    }
    result.chats.push(chat);
    // Native forms may finish in another tab while this tab owns execution.
    const otherMessages = new Map(
      (useLocal ? remote : local).messagesByChatId[id]?.map((message) => [
        message.id,
        message,
      ])
    );
    const previousMessages = new Map(
      (base.messagesByChatId[id] ?? []).map((message) => [message.id, message])
    );
    result.messagesByChatId[id] = (source.messagesByChatId[id] ?? []).map(
      (message) => {
        const merged = mergeDraftHandoffs(
          message,
          otherMessages.get(message.id)
        );
        const omitted = [
          message,
          otherMessages.get(message.id),
          previousMessages.get(message.id),
        ].some(
          (item) =>
            item?.metadata?.queryResultRetention?.version === 1 &&
            item.metadata.queryResultRetention.level === "omitted"
        );
        return omitted
          ? {
              ...merged,
              content: merged.content
                ? omitAgentQueryRows(merged.content)
                : merged.content,
              metadata: {
                ...merged.metadata,
                queryResultRetention: {
                  version: 1 as const,
                  level: "omitted" as const,
                },
              },
            }
          : merged;
      }
    );
    for (const other of otherMessages.values()) {
      const attemptId = other.metadata?.apiAttemptId;
      if (
        !attemptId ||
        !chat.apiAttempts?.some(
          (a) => a.id === attemptId && a.phase === "settled"
        ) ||
        result.messagesByChatId[id].some(
          (m) => m.metadata?.apiAttemptId === attemptId
        )
      )
        continue;
      insertApiToolResult(result.messagesByChatId[id], other);
    }
    if (source.activeChatIds?.includes(id)) result.activeChatIds.push(id);
    if (source.pendingAskByChatId[id])
      result.pendingAskByChatId[id] = source.pendingAskByChatId[id];
    if (source.queuedUserMessagesByChatId[id])
      result.queuedUserMessagesByChatId[id] =
        source.queuedUserMessagesByChatId[id];
    if (!chat.archived) {
      const remoteQueue = remote.followUpQueuesByChatId?.[id];
      const remoteStartup = remote.queuedUserMessagesByChatId[id];
      const queue = rebaseFollowUpQueue(
        base.followUpQueuesByChatId?.[id],
        local.followUpQueuesByChatId?.[id],
        remoteQueue,
        !local.queuedUserMessagesByChatId[id] &&
          remoteStartup?.fromFollowUpQueue
          ? remoteStartup.id
          : undefined
      );
      if (queue) {
        const claimedIds = new Set(
          result.messagesByChatId[id].map(({ id }) => id)
        );
        const startup = result.queuedUserMessagesByChatId[id];
        if (startup) claimedIds.add(startup.id);
        const messages = queue.messages.filter(({ id }) => !claimedIds.has(id));
        if (messages.length)
          (result.followUpQueuesByChatId ??= {})[id] = {
            ...queue,
            messages,
            paused: chat.archived || queue.paused,
          };
      }
    }
    const draft = samePersistedValue(
      local.draftsByChatId[id],
      base.draftsByChatId[id]
    )
      ? remote.draftsByChatId[id]
      : local.draftsByChatId[id];
    if (draft) result.draftsByChatId[id] = draft;
  }
  const selected =
    local.currentChatId !== base.currentChatId
      ? local.currentChatId
      : remote.currentChatId;
  result.currentChatId =
    selected && result.chats.some((chat) => chat.id === selected)
      ? selected
      : null;
  return result;
};

export const createAgentStore = () => {
  const initial = loadInitialState();
  const initialWindowState = getInitialWindowState();
  const lastSerializedByScope = new Map<string, string>();
  const snapshotsByScope = new Map<string, PersistedAgentState>();
  const storageSnapshotsByScope = new Map<string, PersistedAgentState>();
  const pendingStatesByScope = new Map<string, AgentState>();
  const writersByScope = new Map<
    string,
    ReturnType<typeof createAgentRunPersistence>
  >();
  const automaticCleanups = new Map<string, Promise<boolean>>();
  const omitQuerySamplesForScope = new Set<string>();
  let cleanupTimer: ReturnType<typeof setTimeout> | undefined;
  let protectionSuspended = false;
  const reclamationChatIds = new Set<string>();
  let reclaimingStorage = false;
  let selectionRevision = 0;
  let scopeRevision = 0;
  const getWriter = (key: string) => {
    let writer = writersByScope.get(key);
    if (!writer) {
      writer = createAgentRunPersistence(key, () =>
        withAgentStorageBudgetLock(() => {
          // Draft edits can arrive while the writer is waiting for its lock.
          const current = store.getState();
          const state =
            currentScopeKey(current) === key
              ? current
              : pendingStatesByScope.get(key);
          if (!state || !synchronizeState(state, true, true))
            throw new Error("Agent persistence unavailable");
        })
      );
      writersByScope.set(key, writer);
    }
    return writer;
  };
  let applyingSnapshot = false;
  let draftPersistTimer: ReturnType<typeof setTimeout> | null = null;
  const clearDraftPersistTimer = () => {
    if (draftPersistTimer !== null) {
      clearTimeout(draftPersistTimer);
      draftPersistTimer = null;
    }
  };

  const store = create<AgentState>()(
    immer((set, get) => {
      // Internal helpers that work on the plain (non-draft) state via get()
      const _getChat = (chatId?: string | null): AgentChat | null => {
        if (!chatId) return null;
        return get().chats.find((chat) => chat.id === chatId) ?? null;
      };

      const _hasRunningChat = (): boolean =>
        get().chats.some((chat) => chat.status === "running");

      const canEditFollowUpQueue = (chatId: string): AgentChat | null => {
        if (!synchronizeState(get(), false) || get().remoteBusyChatIds[chatId])
          return null;
        const chat = _getChat(chatId);
        return chat && !chat.archived ? chat : null;
      };

      const pauseFollowUpQueue = (state: AgentState, chatId: string) => {
        const queue = state.followUpQueuesByChatId[chatId];
        if (queue) queue.paused = true;
      };

      const clearSqlFixDrafts = (state: AgentState, chatId: string) => {
        delete state.sqlFixSourcesByChatId[chatId];
        for (const [id, draft] of Object.entries(state.sqlFixDraftsById)) {
          if (draft.chatId === chatId) delete state.sqlFixDraftsById[id];
        }
      };

      const canUseSqlFixSource = (
        chatId: string,
        source: AgentSqlFixSource
      ) => {
        const state = get();
        const chat = _getChat(chatId);
        return Boolean(
          chat?.capability &&
            !chat.archived &&
            !state.remoteBusyChatIds[chatId] &&
            chat.workspaceName === state.workspaceName &&
            isCurrentAgentSqlFixSource(source, chat.capability.resource) &&
            JSON.parse(source.identity)[0] === state.workspaceName
        );
      };

      const _touchChatInDraft = (
        chats: AgentChat[],
        chatId: string
      ): AgentChat | null => {
        const chat = chats.find((c) => c.id === chatId);
        if (!chat) return null;
        chat.updatedTs = Date.now();
        return chat;
      };

      const recordMaskingDraftHandoff = (
        kind: "draft-masking-rule" | "draft-masking-exemption",
        key: "maskingRuleDraftHandoff" | "maskingExemptionDraftHandoff",
        chatId: string,
        messageId: string,
        draftId: string,
        outcome: "opened" | "applied"
      ): AgentMessage | null => {
        get().syncState();
        const state = get();
        const message = state.messagesByChatId[chatId]?.find(
          (candidate) => candidate.id === messageId
        );
        if (
          !state.workspaceName ||
          !state.userEmail ||
          _getChat(chatId)?.workspaceName !== state.workspaceName ||
          message?.chatId !== chatId ||
          message.role !== "assistant" ||
          !message.metadata?.actions?.some(
            (action) => action.kind === kind && action.draftId === draftId
          ) ||
          (outcome !== "opened" && outcome !== "applied")
        )
          return null;
        const receipt = message.metadata[key];
        if (receipt) return receipt.draftId === draftId ? message : null;
        set((draft) => {
          const current = draft.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          if (current?.metadata) current.metadata[key] = { draftId, outcome };
        });
        return (
          get().messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          ) ?? null
        );
      };

      return {
        // Window state
        visible: false,
        size: initialWindowState.size,
        sidebarWidth: initialWindowState.sidebarWidth,
        minimized: false,

        preferences: { ...DEFAULT_AGENT_PREFERENCES },
        preferencesStorageError: false,
        updatePreferences: (patch) => {
          const { workspaceName, userEmail } = get();
          const key =
            workspaceName && userEmail
              ? getAgentPreferencesKey(workspaceName, userEmail)
              : null;
          if (!key) return;
          const preferences = normalizeAgentPreferences({
            ...get().preferences,
            ...patch,
          });
          let preferencesStorageError = false;
          try {
            const storage = getSafeStorage();
            if (!storage)
              throw new Error("Agent preference storage unavailable");
            storage.setItem(key, JSON.stringify(preferences));
          } catch {
            preferencesStorageError = true;
          }
          set((draft) => {
            draft.preferences = preferences;
            draft.preferencesStorageError = preferencesStorageError;
          });
        },
        resetPreferences: () =>
          get().updatePreferences(DEFAULT_AGENT_PREFERENCES),
        syncPreferences: () => {
          const { workspaceName, userEmail } = get();
          if (!workspaceName || !userEmail) return;
          set(loadAgentPreferences(workspaceName, userEmail));
        },

        // Chat state (from persisted/initial)
        workspaceName: null,
        userEmail: null,
        chats: initial.chats,
        messagesByChatId: initial.messagesByChatId,
        pendingAskByChatId: initial.pendingAskByChatId,
        draftsByChatId: initial.draftsByChatId,
        landingDraft: { text: "" },
        landingContext: undefined,
        currentChatId: initial.currentChatId,
        abortControllersByChatId: {},
        queuedUserMessagesByChatId: initial.queuedUserMessagesByChatId,
        followUpQueuesByChatId: {},
        followUpDrainChatIds: {},
        queuedUserMessageInFlightByChatId: {},
        remoteBusyChatIds: {},
        chatProtectionReady: true,
        refreshChatProtection: () => {
          protectionSuspended = false;
          refreshChatProtection();
        },
        releaseChatProtection: () => {
          protectionSuspended = true;
          refreshChatProtection();
        },
        expressionDraftTargetsByChatId: {},
        sqlFixSourcesByChatId: {},
        sqlFixDraftsById: {},

        // UI actions
        open: () =>
          set((state) => {
            state.visible = true;
            state.minimized = false;
          }),
        toggle: () => {
          if (!get().visible || get().minimized) {
            get().open();
            return;
          }
          set((state) => {
            state.visible = false;
          });
        },
        minimize: () =>
          set((state) => {
            state.minimized = true;
          }),
        restore: () =>
          set((state) => {
            state.minimized = false;
          }),
        setSize: (width, height) =>
          set((state) => {
            state.size = { width, height };
          }),
        setSidebarWidth: (width) =>
          set((state) => {
            state.sidebarWidth = width;
          }),

        // Queries
        getChat: (chatId) => _getChat(chatId),

        getMessages: (chatId) => {
          if (!chatId) return [];
          return get().messagesByChatId[chatId] ?? [];
        },

        getPendingAsk: (chatId) => {
          const id = chatId ?? get().currentChatId;
          if (!id) return null;
          return get().pendingAskByChatId[id] ?? null;
        },

        getDraft: (chatId) => {
          const id = chatId ?? get().currentChatId;
          return (id ? get().draftsByChatId[id] : undefined) ?? { text: "" };
        },

        setDraft: (chatId, draft) => {
          if (!_getChat(chatId)) return;
          const normalized = normalizeDraft(draft);
          if (!normalized) return;
          const { followUpSourceId: source, ...previous } =
            get().getDraft(chatId);
          const { followUpSourceId: nextSource, ...next } = normalized;
          if (source !== undefined) {
            const unchanged = JSON.stringify(previous) === JSON.stringify(next);
            if (unchanged && nextSource === undefined)
              normalized.followUpSourceId = source;
            if (!unchanged && nextSource === source)
              delete normalized.followUpSourceId;
          }
          set((state) => {
            if (
              normalized.text ||
              normalized.project ||
              normalized.resource ||
              normalized.mentions
            ) {
              state.draftsByChatId[chatId] = normalized;
            } else {
              delete state.draftsByChatId[chatId];
            }
          });
        },

        clearDraft: (chatId) =>
          set((state) => {
            delete state.draftsByChatId[chatId];
          }),

        setLandingDraft: (draft, context) => {
          const normalized = normalizeDraft(draft);
          if (!normalized) return;
          set((state) => {
            state.landingDraft = normalized;
            if (context !== undefined) state.landingContext = context;
          });
        },

        clearLandingDraft: (context) =>
          set((state) => {
            state.landingDraft = { text: "" };
            state.landingContext = context;
          }),

        getAbortController: (chatId) => {
          if (!chatId) return null;
          return get().abortControllersByChatId[chatId] ?? null;
        },

        isChatRunning: (chatId) =>
          _getChat(chatId)?.status === "running" ||
          Boolean(chatId && get().remoteBusyChatIds[chatId]),

        canSelectChat: (chatId) => {
          return Boolean(chatId && _getChat(chatId));
        },

        touchChat: (chatId) => {
          let result: AgentChat | null = null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.updatedTs = Date.now();
              result = { ...chat } as AgentChat;
            }
          });
          return result;
        },

        setAbortController: (chatId, controller) => {
          // AbortControllers are not serializable, store outside immer
          const controllers = get().abortControllersByChatId;
          if (controller) {
            set((state) => {
              state.abortControllersByChatId[chatId] = controller;
            });
            return controller;
          }
          if (controllers[chatId]) {
            set((state) => {
              delete state.abortControllersByChatId[chatId];
            });
          }
          return null;
        },

        // Chat CRUD
        createChat: (options = {}) => {
          const chat = createChatRecord(options);
          set((state) => {
            state.chats.push(chat);
            state.messagesByChatId[chat.id] = [];
            if (options.select ?? true) {
              state.currentChatId = chat.id;
            }
          });
          return chat;
        },

        queueUserMessage: (message) => {
          get().syncState();
          if (
            get().remoteBusyChatIds[message.chatId] ||
            _getChat(message.chatId)?.status !== "idle" ||
            _getChat(message.chatId)?.runCheckpoint?.suspension ===
              "token-limit" ||
            get().queuedUserMessagesByChatId[message.chatId]
          ) {
            return;
          }
          const {
            resourceReference,
            resourceReferences,
            followUpSourceId,
            ...rest
          } = message;
          const normalizedReferences = normalizeAgentResourceReferences(
            resourceReferences,
            resourceReference
          );
          set((state) => {
            state.queuedUserMessagesByChatId[message.chatId] = {
              ...rest,
              id: uuidv4(),
              ...(followUpSourceId !== undefined
                ? {
                    followUpSourceId:
                      normalizeFollowUpSourceId(followUpSourceId),
                  }
                : {}),
              ...(resourceReferences === undefined && normalizedReferences[0]
                ? { resourceReference: normalizedReferences[0] }
                : {}),
              ...(resourceReferences !== undefined &&
              normalizedReferences.length > 0
                ? { resourceReferences: normalizedReferences }
                : {}),
            };
            delete state.queuedUserMessageInFlightByChatId[message.chatId];
          });
        },

        enqueueFollowUp: (input) => {
          const { chatId } = input;
          const chat = canEditFollowUpQueue(chatId);
          if (
            !chat ||
            !get().canUseAgentExecution() ||
            get().storagePressure ||
            chat.requiresAIConfiguration
          )
            return null;
          const message = normalizeQueuedUserMessage(
            { ...input, id: uuidv4() },
            new Set([chatId])
          );
          if (!message) return null;
          set((state) => {
            const queue = (state.followUpQueuesByChatId[chatId] ??= {
              messages: [],
              paused: chat.status === "error" || Boolean(chat.interrupted),
            });
            queue.messages.push(message);
          });
          return message.id;
        },

        removeFollowUp: (chatId, messageId) => {
          if (!canEditFollowUpQueue(chatId)) return false;
          const index = get().followUpQueuesByChatId[
            chatId
          ]?.messages.findIndex(({ id }) => id === messageId);
          if (index === undefined || index < 0) return false;
          set((state) => {
            const queue = state.followUpQueuesByChatId[chatId];
            queue.messages.splice(index, 1);
            if (!queue.messages.length)
              delete state.followUpQueuesByChatId[chatId];
          });
          return true;
        },

        setFollowUpQueuePaused: (chatId, paused) => {
          if (!canEditFollowUpQueue(chatId)) return false;
          const queue = get().followUpQueuesByChatId[chatId];
          if (!queue) return false;
          if (queue.paused !== paused) {
            set((state) => {
              state.followUpQueuesByChatId[chatId].paused = paused;
            });
          }
          return true;
        },

        setFollowUpDrainActive: (chatId, active) => {
          if (active) {
            if (!canEditFollowUpQueue(chatId) || !canOwnRun(get(), chatId))
              return false;
          } else if (!get().followUpDrainChatIds[chatId]) {
            return false;
          }
          set((state) => {
            if (active) state.followUpDrainChatIds[chatId] = true;
            else delete state.followUpDrainChatIds[chatId];
          });
          return true;
        },

        promoteFollowUp: (chatId, messageId) => {
          const chat = canEditFollowUpQueue(chatId);
          const state = get();
          const queue = state.followUpQueuesByChatId[chatId];
          const head = queue?.messages[0];
          if (
            !chat ||
            chat.status !== "idle" ||
            chat.runCheckpoint ||
            chat.requiresAIConfiguration ||
            state.storagePressure ||
            state.pendingAskByChatId[chatId] ||
            state.abortControllersByChatId[chatId] ||
            state.queuedUserMessagesByChatId[chatId] ||
            state.queuedUserMessageInFlightByChatId[chatId] ||
            !canOwnRun(state, chatId) ||
            head?.id !== messageId ||
            queue.paused
          )
            return null;
          // The answer may not exist yet when the follow-up is enqueued.
          const source = state.getMessages(chatId).at(-1);
          const promoted =
            head.followUpSourceId === undefined &&
            !head.resourceReference &&
            !head.resourceReferences?.length &&
            source?.chatId === chatId &&
            source.role === "assistant" &&
            source.content?.trim() &&
            !source.toolCalls?.length &&
            !source.metadata?.pending &&
            !source.metadata?.error
              ? { ...head, followUpSourceId: source.id }
              : head;
          set((draft) => {
            const pending = draft.followUpQueuesByChatId[chatId];
            pending.messages.shift();
            if (!pending.messages.length)
              delete draft.followUpQueuesByChatId[chatId];
            draft.queuedUserMessagesByChatId[chatId] = {
              ...promoted,
              fromFollowUpQueue: true,
            };
            draft.queuedUserMessageInFlightByChatId[chatId] = true;
          });
          return promoted;
        },

        takeQueuedUserMessage: (chatId) => {
          const message = get().queuedUserMessagesByChatId[chatId];
          if (
            !message ||
            message.fromFollowUpQueue ||
            _getChat(chatId)?.runCheckpoint?.suspension === "token-limit" ||
            get().remoteBusyChatIds[chatId] ||
            get().queuedUserMessageInFlightByChatId[chatId]
          ) {
            return null;
          }
          set((state) => {
            state.queuedUserMessageInFlightByChatId[chatId] = true;
          });
          return message;
        },

        finishQueuedUserMessage: (chatId) =>
          set((state) => {
            delete state.queuedUserMessagesByChatId[chatId];
            delete state.queuedUserMessageInFlightByChatId[chatId];
            delete state.expressionDraftTargetsByChatId[chatId];
          }),

        commitQueuedUserMessage: (chatId) => {
          const queued = get().queuedUserMessagesByChatId[chatId];
          if (
            !queued ||
            get().remoteBusyChatIds[chatId] ||
            !_getChat(chatId) ||
            _getChat(chatId)?.runCheckpoint?.suspension === "token-limit"
          ) {
            return null;
          }

          let message: AgentMessage | null = null;
          set((state) => {
            const queued = state.queuedUserMessagesByChatId[chatId];
            if (!queued) return;
            const chatMessages = state.messagesByChatId[chatId] ?? [];
            state.messagesByChatId[chatId] = chatMessages;
            const existing = chatMessages.find(
              (candidate) => candidate.id === queued.id
            );
            if (existing) {
              if (existing.role !== "user") return;
              message = { ...existing } as AgentMessage;
            } else {
              const committed: AgentMessage = {
                id: queued.id,
                chatId,
                createdTs: Date.now(),
                role: "user",
                content: queued.text,
                metadata:
                  queued.resourceReferences ||
                  queued.resourceReference ||
                  queued.followUpSourceId !== undefined
                    ? {
                        ...(queued.resourceReferences
                          ? { resourceReferences: queued.resourceReferences }
                          : queued.resourceReference
                            ? { resourceReference: queued.resourceReference }
                            : {}),
                        ...(queued.followUpSourceId !== undefined
                          ? { followUpSourceId: queued.followUpSourceId }
                          : {}),
                      }
                    : undefined,
              };
              chatMessages.push(committed);
              message = committed;
              const chat = state.chats.find(
                (candidate) => candidate.id === chatId
              );
              if (chat) {
                if (!chat.title)
                  chat.title = getChatTitleFromMessage(queued.text);
                chat.lastError = null;
                chat.requiresAIConfiguration = false;
                chat.interrupted = false;
                chat.interruptionReason = undefined;
                chat.updatedTs = committed.createdTs;
              }
            }
            state.queuedUserMessageInFlightByChatId[chatId] = true;
          });
          return message ? { queued, message } : null;
        },

        ensureCurrentChat: (page, context, workspaceName) => {
          const existing = _getChat(get().currentChatId);
          if (existing) {
            const canBindContext =
              Boolean(context) &&
              !existing.context &&
              get().getMessages(existing.id).length === 0;
            if ((page && !existing.page) || canBindContext) {
              set((state) => {
                const chat = state.chats.find((c) => c.id === existing.id);
                if (chat) {
                  if (page && !chat.page) chat.page = page;
                  if (canBindContext && context) {
                    chat.context = context;
                    chat.workspaceName = workspaceName;
                  }
                  _touchChatInDraft(state.chats, chat.id);
                }
              });
            }
            return _getChat(existing.id)!;
          }
          return get().createChat({ page, context, workspaceName });
        },

        selectChatContext: (context, page, workspaceName) => {
          if (!context) return null;
          if (
            workspaceName &&
            get().workspaceName !== workspaceName &&
            !get().setWorkspaceName(workspaceName)
          ) {
            return null;
          }
          const current = _getChat(get().currentChatId);
          if (!current)
            return get().createChat({ page, context, workspaceName });
          if (
            current.context === context &&
            current.workspaceName === workspaceName
          ) {
            return current;
          }
          if (
            current.status === "running" ||
            get().getPendingAsk(current.id) ||
            get().queuedUserMessagesByChatId[current.id] ||
            get().followUpQueuesByChatId[current.id]
          ) {
            return get().createChat({ page, context, workspaceName });
          }
          set((state) => {
            const chat = state.chats.find((c) => c.id === current.id);
            if (chat) {
              chat.context = context;
              chat.workspaceName = workspaceName;
              _touchChatInDraft(state.chats, chat.id);
            }
          });
          return _getChat(current.id);
        },

        selectOrCreateChatForContext: (context, page, workspaceName) => {
          if (!context) {
            return null;
          }
          if (
            workspaceName &&
            get().workspaceName !== workspaceName &&
            !get().setWorkspaceName(workspaceName)
          ) {
            return null;
          }

          const isReusable = (chat: AgentChat) =>
            !chat.archived &&
            chat.capability === undefined &&
            chat.status === "idle" &&
            !get().getPendingAsk(chat.id) &&
            !get().queuedUserMessagesByChatId[chat.id] &&
            !get().followUpQueuesByChatId[chat.id];
          const current = _getChat(get().currentChatId);
          const matchingChat =
            current?.context === context &&
            current.workspaceName === workspaceName &&
            isReusable(current)
              ? current
              : [...get().chats]
                  .filter(
                    (chat) =>
                      chat.context === context &&
                      chat.workspaceName === workspaceName &&
                      isReusable(chat)
                  )
                  .sort(
                    (a, b) =>
                      b.updatedTs - a.updatedTs || b.createdTs - a.createdTs
                  )[0];

          if (matchingChat) {
            set((state) => {
              const chat = state.chats.find((c) => c.id === matchingChat.id);
              if (!chat) return;
              if (page) chat.page = page;
              state.currentChatId = chat.id;
              _touchChatInDraft(state.chats, chat.id);
            });
            return _getChat(matchingChat.id);
          }

          if (
            current &&
            !current.context &&
            isReusable(current) &&
            get().getMessages(current.id).length === 0
          ) {
            set((state) => {
              const chat = state.chats.find((c) => c.id === current.id);
              if (!chat) return;
              chat.context = context;
              chat.workspaceName = workspaceName;
              if (page) chat.page = page;
              _touchChatInDraft(state.chats, chat.id);
            });
            return _getChat(current.id);
          }

          return get().createChat({ context, page, workspaceName });
        },

        setCurrentChat: (chatId) => {
          if (get().canSelectChat(chatId)) {
            set((state) => {
              state.currentChatId = chatId;
            });
          }
        },

        clearCurrentChat: () => {
          set((state) => {
            state.currentChatId = null;
          });
        },

        updateChatPage: (chatId, page) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.page = page;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        renameChat: (chatId, title) => {
          const chat = _getChat(chatId);
          if (!chat) return null;
          const normalizedTitle = normalizeChatTitle(title);
          if (chat.title === normalizedTitle) return chat;
          set((state) => {
            const c = state.chats.find((c) => c.id === chatId);
            if (c) {
              c.title = normalizedTitle;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        archiveChat: (chatId) => {
          get().syncState();
          if (
            get().remoteBusyChatIds[chatId] &&
            get().followUpQueuesByChatId[chatId]
          )
            return null;
          if (!_getChat(chatId)) return null;
          if (
            _getChat(chatId)?.runBudget ||
            get().pendingAskByChatId[chatId]?.apiPermission
          )
            get().getAbortController(chatId)?.abort();
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.archived = true;
              delete state.followUpQueuesByChatId[chatId];
              clearSqlFixDrafts(state, chatId);
              if (
                chat.runBudget ||
                state.pendingAskByChatId[chatId]?.apiPermission
              ) {
                chat.runCheckpoint = undefined;
                chat.runId = null;
                delete state.pendingAskByChatId[chatId];
                delete state.queuedUserMessagesByChatId[chatId];
                delete state.queuedUserMessageInFlightByChatId[chatId];
                if (
                  chat.status === "running" ||
                  chat.status === "paused" ||
                  chat.status === "awaiting_user"
                ) {
                  chat.status = "idle";
                  chat.interrupted = true;
                }
              }
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        archiveAllActiveChats: () => {
          get().syncState();
          const canArchive = (state: AgentState, chat: AgentChat) =>
            !chat.archived &&
            !(
              state.remoteBusyChatIds[chat.id] &&
              state.followUpQueuesByChatId[chat.id]
            );
          let archivedCount = 0;
          for (const chat of get().chats)
            if (
              canArchive(get(), chat) &&
              (chat.runBudget ||
                get().pendingAskByChatId[chat.id]?.apiPermission)
            )
              get().getAbortController(chat.id)?.abort();
          set((state) => {
            const activeChats = state.chats.filter((chat) =>
              canArchive(state, chat)
            );
            archivedCount = activeChats.length;
            if (archivedCount === 0) return;

            for (const chat of activeChats) {
              chat.archived = true;
              delete state.followUpQueuesByChatId[chat.id];
              clearSqlFixDrafts(state, chat.id);
              if (
                chat.runBudget ||
                state.pendingAskByChatId[chat.id]?.apiPermission
              ) {
                chat.runCheckpoint = undefined;
                chat.runId = null;
                delete state.pendingAskByChatId[chat.id];
                delete state.queuedUserMessagesByChatId[chat.id];
                delete state.queuedUserMessageInFlightByChatId[chat.id];
                if (
                  chat.status === "running" ||
                  chat.status === "paused" ||
                  chat.status === "awaiting_user"
                ) {
                  chat.status = "idle";
                  chat.interrupted = true;
                }
              }
              _touchChatInDraft(state.chats, chat.id);
            }

            if (
              state.currentChatId &&
              activeChats.some((chat) => chat.id === state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return archivedCount;
        },

        unarchiveChat: (chatId) => {
          if (!_getChat(chatId)) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.archived = false;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        unarchiveAllArchivedChats: () => {
          let unarchivedCount = 0;
          set((state) => {
            const archivedChats = state.chats.filter((chat) => chat.archived);
            unarchivedCount = archivedChats.length;
            if (unarchivedCount === 0) return;

            for (const chat of archivedChats) {
              chat.archived = false;
              _touchChatInDraft(state.chats, chat.id);
            }

            if (
              state.currentChatId &&
              archivedChats.some((chat) => chat.id === state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return unarchivedCount;
        },

        deleteChat: (chatId) => {
          get().syncState();
          if (!_getChat(chatId) || get().remoteBusyChatIds[chatId])
            return false;
          get().cancel(chatId);
          set((state) => {
            state.chats = state.chats.filter((c) => c.id !== chatId);
            delete state.messagesByChatId[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.draftsByChatId[chatId];
            delete state.abortControllersByChatId[chatId];
            delete state.queuedUserMessagesByChatId[chatId];
            delete state.followUpQueuesByChatId[chatId];
            delete state.followUpDrainChatIds[chatId];
            delete state.queuedUserMessageInFlightByChatId[chatId];
            clearSqlFixDrafts(state, chatId);
            if (state.currentChatId === chatId) {
              // Select next available chat
              const ordered = [...state.chats].sort((a, b) => {
                if (b.updatedTs !== a.updatedTs)
                  return b.updatedTs - a.updatedTs;
                return b.createdTs - a.createdTs;
              });
              if (ordered.length > 0) {
                state.currentChatId = ordered[0].id;
              } else {
                // Create a new default chat
                const newChat = createChatRecord();
                state.chats.push(newChat);
                state.messagesByChatId[newChat.id] = [];
                state.currentChatId = newChat.id;
              }
            }
          });
          return true;
        },

        deleteAllArchivedChats: () => {
          get().syncState();
          const archivedChatIds = get()
            .chats.filter(
              (chat) => chat.archived && !get().remoteBusyChatIds[chat.id]
            )
            .map((chat) => chat.id);
          for (const chatId of archivedChatIds) {
            get().cancel(chatId);
          }

          let deletedCount = 0;
          set((state) => {
            const archivedChats = state.chats.filter(
              (chat) => chat.archived && !state.remoteBusyChatIds[chat.id]
            );
            deletedCount = archivedChats.length;
            if (deletedCount === 0) return;

            const deletedChatIds = new Set(
              archivedChats.map((chat) => chat.id)
            );
            for (const chatId of deletedChatIds) {
              delete state.messagesByChatId[chatId];
              delete state.pendingAskByChatId[chatId];
              delete state.draftsByChatId[chatId];
              delete state.abortControllersByChatId[chatId];
            }

            for (const chatId of deletedChatIds) {
              delete state.queuedUserMessagesByChatId[chatId];
              delete state.followUpQueuesByChatId[chatId];
              delete state.followUpDrainChatIds[chatId];
              delete state.queuedUserMessageInFlightByChatId[chatId];
              delete state.expressionDraftTargetsByChatId[chatId];
              clearSqlFixDrafts(state, chatId);
            }

            state.chats = state.chats.filter(
              (chat) => !deletedChatIds.has(chat.id)
            );
            if (
              state.currentChatId &&
              deletedChatIds.has(state.currentChatId)
            ) {
              state.currentChatId = null;
            }
          });
          return deletedCount;
        },

        setSqlFixSource: (chatId, source) => {
          if (
            get().currentChatId !== chatId ||
            !canUseSqlFixSource(chatId, source)
          )
            return;
          set((state) => {
            clearSqlFixDrafts(state, chatId);
            state.sqlFixSourcesByChatId[chatId] = {
              ...source,
              resource: { ...source.resource },
              targets: [...source.targets],
            };
          });
        },

        prepareSqlFixDraft: (chatId, answer) => {
          const source = get().sqlFixSourcesByChatId[chatId];
          if (!source || !canUseSqlFixSource(chatId, source)) return undefined;
          const sql = extractAgentSqlFixSQL(answer);
          if (sql === undefined || sql === source.sql) return undefined;
          const id = uuidv4();
          set((state) => {
            state.sqlFixDraftsById[id] = { id, chatId, source, sql };
          });
          return { kind: "review-sql-fix", draftId: id };
        },

        registerExpressionDraftTarget: (chatId, apply, options = {}) => {
          const registered: RegisteredExpressionDraftTarget = {
            apply,
            allowRaw: options.allowRaw ?? true,
            token: Symbol(),
          };
          set((state) => {
            state.expressionDraftTargetsByChatId[chatId] = registered;
          });
          return () =>
            set((state) => {
              if (
                state.expressionDraftTargetsByChatId[chatId]?.token ===
                registered.token
              ) {
                delete state.expressionDraftTargetsByChatId[chatId];
              }
            });
        },

        canApplyExpressionDraft: (chatId, mode) => {
          const target = get().expressionDraftTargetsByChatId[chatId];
          return Boolean(target && (mode === "selector" || target.allowRaw));
        },

        applyExpressionDraft: async (chatId, expression, mode) => {
          const target = get().expressionDraftTargetsByChatId[chatId];
          if (!target || (mode === "raw" && !target.allowRaw)) return false;
          return target.apply(expression, mode);
        },

        // Messages
        getApiAttemptToolResult: (chatId, id) =>
          get()
            .getMessages(chatId)
            .find((m) => m.role === "tool" && m.metadata?.apiAttemptId === id),

        addMessage: (message) => {
          if (message.role === "user") get().syncState();
          if (message.chatId && !_getChat(message.chatId)) return null;
          const chatId = message.chatId ?? get().ensureCurrentChat().id;
          if (
            message.role === "tool" &&
            message.toolCallId &&
            message.metadata?.runId
          ) {
            const attempt = _getChat(chatId)?.apiAttempts?.find(
              (a) =>
                a.phase === "settled" &&
                a.runId === message.metadata?.runId &&
                a.toolCallId === message.toolCallId &&
                parseCallApiResult(message.content ?? "", a.operationId)
                  .execution?.attemptId === a.id
            );
            const linked =
              attempt && get().getApiAttemptToolResult(chatId, attempt.id);
            if (linked) return linked;
          }
          if (get().remoteBusyChatIds[chatId]) return null;
          if (
            message.role === "user" &&
            (_getChat(chatId)?.status === "paused" ||
              _getChat(chatId)?.runCheckpoint?.suspension === "token-limit")
          )
            return null;
          const createdTs = Date.now();
          const agentMessage: AgentMessage = {
            ...message,
            id: uuidv4(),
            chatId,
            createdTs,
          };
          set((state) => {
            const chatMessages = state.messagesByChatId[chatId] ?? [];
            if (!state.messagesByChatId[chatId]) {
              state.messagesByChatId[chatId] = chatMessages;
            }
            chatMessages.push(agentMessage);
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              if (message.role === "user" && !chat.title) {
                chat.title = getChatTitleFromMessage(message.content);
              }
              chat.lastError = null;
              chat.requiresAIConfiguration = false;
              if (message.role === "user") {
                chat.interrupted = false;
                chat.interruptionReason = undefined;
              }
              chat.updatedTs = createdTs;
            }
          });
          return agentMessage;
        },

        removeMessagesByRunId: (chatId, runId) => {
          if (!runId || !_getChat(chatId)) return [];
          const existingMessages = get().messagesByChatId[chatId] ?? [];
          const removedMessages = existingMessages.filter(
            (m) => m.metadata?.runId === runId && !m.metadata.apiAttemptId
          );
          if (removedMessages.length === 0) return [];
          set((state) => {
            state.messagesByChatId[chatId] = (
              state.messagesByChatId[chatId] ?? []
            ).filter(
              (m) => m.metadata?.runId !== runId || !!m.metadata.apiAttemptId
            );
            _touchChatInDraft(state.chats, chatId);
          });
          return removedMessages;
        },

        appendToolCall: (chatId, messageId, toolCall) => {
          const messages = get().messagesByChatId[chatId] ?? [];
          const message = messages.find((m) => m.id === messageId);
          if (!message) return null;
          set((state) => {
            const msgs = state.messagesByChatId[chatId] ?? [];
            const msg = msgs.find((m) => m.id === messageId);
            if (msg) {
              msg.toolCalls = [...(msg.toolCalls ?? []), toolCall];
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return (
            get()
              .getMessages(chatId)
              .find((m) => m.id === messageId) ?? null
          );
        },

        appendContinueActions: (chatId, messageId, newActions, references) => {
          const messages = get().messagesByChatId[chatId];
          const message = messages?.at(-1);
          if (
            !_getChat(chatId) ||
            message?.id !== messageId ||
            message.role !== "assistant"
          ) {
            return null;
          }
          const existingActions = message.metadata?.actions ?? [];
          const actions = normalizeAgentResponseActions([
            ...existingActions,
            ...newActions,
          ]);
          if (actions.length <= existingActions.length) return null;
          set((state) => {
            const current = state.messagesByChatId[chatId]?.at(-1);
            if (current?.id !== messageId) return;
            current.metadata = {
              ...current.metadata,
              actions,
              ...(references.length > 0
                ? { resourceReferences: references }
                : {}),
            };
          });
          return get().messagesByChatId[chatId]?.at(-1) ?? null;
        },

        recordSubmittedAccessGrant: (chatId, messageId, grant, origin) => {
          get().syncState();
          const state = get();
          const key = currentScopeKey(state);
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          const receipt = createAccessGrantActionReceipt(grant);
          if (
            !key ||
            !ownsApiScope(state, key) ||
            state.workspaceName !== origin.workspaceName ||
            state.userEmail !== origin.userEmail ||
            _getChat(chatId)?.workspaceName !== origin.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !isAccessGrantSourceDraft(message.metadata?.accessRequestDraft) ||
            !samePersistedValue(
              message.metadata.accessRequestDraft,
              origin.draft
            ) ||
            !receipt
          )
            return null;
          return get().recordActionReceipt(chatId, messageId, receipt, origin);
        },

        recordActionReceipt: (chatId, messageId, receipt, origin) => {
          get().syncState();
          const state = get();
          const key = currentScopeKey(state);
          const chat = _getChat(chatId);
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          if (
            !key ||
            !ownsApiScope(state, key) ||
            state.workspaceName !== origin.workspaceName ||
            state.userEmail !== origin.userEmail ||
            chat?.workspaceName !== origin.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !actionReceiptMatchesChat(receipt, chat)
          )
            return null;
          const normalized = normalizeActionReceipts(
            [receipt],
            { ...message.metadata, submittedAccessGrants: undefined },
            origin.userEmail
          )?.[0];
          if (!normalized) return null;
          const existing = message.metadata?.actionReceipts ?? [];
          if (
            existing.some(
              (value) =>
                actionReceiptKey(value) === actionReceiptKey(normalized)
            )
          )
            return message;
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current)
              current.metadata = {
                ...current.metadata,
                actionReceipts: [...existing, normalized],
              };
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },

        recordActionObservation: (
          chatId,
          messageId,
          receiptKey,
          observation,
          origin
        ) => {
          get().syncState();
          const state = get();
          const key = currentScopeKey(state);
          const chat = _getChat(chatId);
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          if (
            !key ||
            !ownsApiScope(state, key) ||
            state.workspaceName !== origin.workspaceName ||
            state.userEmail !== origin.userEmail ||
            chat?.workspaceName !== origin.workspaceName ||
            chat.archived ||
            message?.chatId !== chatId ||
            message.role !== "assistant"
          )
            return null;
          const existing = message.metadata?.actionReceipts ?? [];
          const receipt = existing.find(
            (value) => actionReceiptKey(value) === receiptKey
          );
          if (!receipt) return null;
          const normalized = normalizeActionReceipts(
            [{ ...receipt, observation }],
            { ...message.metadata, submittedAccessGrants: undefined },
            origin.userEmail
          )?.[0];
          if (
            !normalized?.observation ||
            !isEqual(normalized.observation, observation)
          )
            return null;
          const updated = {
            ...normalized,
            observation: {
              ...normalized.observation,
              revision: (receipt.observation?.revision ?? 0) + 1,
            },
          };
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current?.metadata)
              current.metadata.actionReceipts = existing.map((value) =>
                actionReceiptKey(value) === receiptKey ? updated : value
              );
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },

        recordSemanticTypeDraftHandoff: (
          chatId,
          messageId,
          draftId,
          outcome
        ) => {
          get().syncState();
          const state = get();
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          const receipt = message?.metadata?.semanticTypeDraftHandoff;
          if (
            !state.workspaceName ||
            !state.userEmail ||
            _getChat(chatId)?.workspaceName !== state.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !message.metadata?.actions?.some(
              (action) =>
                action.kind === "draft-semantic-type" &&
                action.draftId === draftId
            ) ||
            (outcome !== "opened" && outcome !== "created") ||
            (receipt && receipt.draftId !== draftId) ||
            (outcome === "created" && receipt?.outcome !== "opened")
          )
            return null;
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current?.metadata)
              current.metadata.semanticTypeDraftHandoff = {
                draftId,
                outcome: "opened",
                ...(receipt?.created || outcome === "created"
                  ? { created: true }
                  : {}),
              };
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },
        recordMaskingRuleDraftHandoff: (chatId, messageId, draftId, outcome) =>
          recordMaskingDraftHandoff(
            "draft-masking-rule",
            "maskingRuleDraftHandoff",
            chatId,
            messageId,
            draftId,
            outcome
          ),
        recordMaskingExemptionDraftHandoff: (
          chatId,
          messageId,
          draftId,
          outcome
        ) =>
          recordMaskingDraftHandoff(
            "draft-masking-exemption",
            "maskingExemptionDraftHandoff",
            chatId,
            messageId,
            draftId,
            outcome
          ),
        markMaskingExemptionDraftCreated: (chatId, messageId, draftId) => {
          get().syncState();
          const state = get();
          const message = state.messagesByChatId[chatId]?.find(
            (candidate) => candidate.id === messageId
          );
          const receipt = message?.metadata?.maskingExemptionDraftHandoff;
          if (
            !state.workspaceName ||
            !state.userEmail ||
            _getChat(chatId)?.workspaceName !== state.workspaceName ||
            message?.chatId !== chatId ||
            message.role !== "assistant" ||
            !message.metadata?.actions?.some(
              (action) =>
                action.kind === "draft-masking-exemption" &&
                action.draftId === draftId
            ) ||
            receipt?.draftId !== draftId ||
            (receipt.outcome !== "opened" && receipt.outcome !== "applied")
          )
            return null;
          if (receipt.created) return message;
          set((draft) => {
            const current = draft.messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            );
            if (current?.metadata?.maskingExemptionDraftHandoff)
              current.metadata.maskingExemptionDraftHandoff.created = true;
          });
          return (
            get().messagesByChatId[chatId]?.find(
              (candidate) => candidate.id === messageId
            ) ?? null
          );
        },

        // Status
        setChatStatus: (chatId, status, options = {}) => {
          if (!_getChat(chatId) || get().remoteBusyChatIds[chatId]) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.status = status;
              chat.interrupted = options.interrupted ?? false;
              chat.interruptionReason = chat.interrupted
                ? (options.interruptionReason ?? "unexpected")
                : undefined;
              chat.lastError = options.lastError ?? null;
              chat.requiresAIConfiguration =
                options.requiresAIConfiguration ?? false;
              if (status === "error" || options.interrupted)
                pauseFollowUpQueue(state, chatId);
              if (options.page) {
                chat.page = options.page;
              }
              if (status !== "awaiting_user") {
                delete state.pendingAskByChatId[chatId];
              }
              if (
                status !== "running" &&
                status !== "awaiting_user" &&
                state.queuedUserMessagesByChatId[chatId] &&
                state.messagesByChatId[chatId]?.some(
                  (message) =>
                    message.id ===
                      state.queuedUserMessagesByChatId[chatId]?.id &&
                    message.role === "user"
                )
              ) {
                delete state.queuedUserMessagesByChatId[chatId];
                delete state.queuedUserMessageInFlightByChatId[chatId];
              }
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        startChatRun: (chatId, page, options = {}) => {
          if (
            (currentScopeKey(get()) && !get().canUseAgentExecution()) ||
            _getChat(chatId)?.runCheckpoint ||
            (_getChat(chatId)?.runId && _getChat(chatId)?.runBudget) ||
            _getChat(chatId)?.status === "paused"
          )
            return;
          get().setChatStatus(chatId, "running", { page });
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.runId = options.runId ?? chat.runId ?? null;
              chat.runMode = options.runMode;
              if (chat.runId && chat.lastRunUsage?.runId !== chat.runId) {
                chat.lastRunUsage = { runId: chat.runId, calls: [] };
              }
              setLastUserInterruptionReason(
                state.messagesByChatId[chatId] ?? []
              );
            }
          });
        },

        beginChatRun: async (chatId, rootUserMessageId, rawBindings) => {
          get().syncState();
          const chat = _getChat(chatId);
          const bindings = normalizeRunBindings(rawBindings);
          if (
            !chat ||
            chat.archived ||
            chat.runCheckpoint ||
            (chat.runId && chat.runBudget) ||
            chat.status === "paused" ||
            (get().remoteBusyChatIds[chatId] && !supportsAgentRunRecovery()) ||
            !bindings ||
            !samePersistedValue(chat.capability, bindings.capability) ||
            !get()
              .getMessages(chatId)
              .some(
                (message) =>
                  message.id === rootUserMessageId && message.role === "user"
              ) ||
            !canOwnRun(get(), chatId)
          )
            return null;
          const scopeKey = currentScopeKey(get());
          const previousMessages = get().getMessages(chatId);
          const runId = uuidv4();
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.legacyUsage ??= {
              totalTokensUsed: current.totalTokensUsed,
              hasUnreportedUsage: Boolean(
                current.hasUnreportedUsage || current.totalTokensUsed > 0
              ),
            };
            current.legacyUsage.hasUnreportedUsage ||= Boolean(
              current.runBudget?.hasUnreportedUsage
            );
            current.modelAttempts ??= [];
            current.runId = runId;
            current.runMode = bindings.runMode;
            current.runBudget = createRunBudget(
              runId,
              state.preferences.tokenLimitPerRun
            );
            current.runCheckpoint = {
              version: 1,
              runId,
              suspension: "interrupted",
              rootUserMessageId,
              throughMessageId: rootUserMessageId,
              bindings,
              maskingSections: [],
              phase: "working",
              startupComplete: false,
            };
            current.lastRunUsage = { runId, calls: [] };
            current.status = "running";
            delete state.remoteBusyChatIds[chatId];
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            current.requiresAIConfiguration = false;
            delete state.pendingAskByChatId[chatId];
            const root = state.messagesByChatId[chatId]?.find(
              (message) => message.id === rootUserMessageId
            );
            if (root) root.metadata = { ...root.metadata, runId };
            setLastUserInterruptionReason(state.messagesByChatId[chatId] ?? []);
            _touchChatInDraft(state.chats, chatId);
          });
          const initial = _getChat(chatId)!;
          const initialMessages = get().getMessages(chatId);
          if (await get().flushRunPersistence()) return runId;
          if (currentScopeKey(get()) !== scopeKey) return null;
          // Execution cannot start before this initial checkpoint is durable.
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (
              !current ||
              current.runId !== runId ||
              current.status !== "running" ||
              current.archived ||
              state.abortControllersByChatId[chatId] ||
              state.remoteBusyChatIds[chatId] ||
              state.pendingAskByChatId[chatId] ||
              state.queuedUserMessagesByChatId[chatId] ||
              state.queuedUserMessageInFlightByChatId[chatId] ||
              current.runBudget?.providerAttemptCount !== 0 ||
              current.modelAttempts?.some(
                (attempt) => attempt.runId === runId
              ) ||
              !samePersistedValue(
                current.runCheckpoint,
                initial.runCheckpoint
              ) ||
              !samePersistedValue(current.runBudget, initial.runBudget) ||
              !samePersistedValue(current.lastRunUsage, initial.lastRunUsage) ||
              !samePersistedValue(
                state.messagesByChatId[chatId],
                initialMessages
              )
            )
              return;
            current.status = "error";
            current.runId = null;
            current.runMode = undefined;
            current.runCheckpoint = undefined;
            current.runBudget = chat.runBudget;
            current.lastRunUsage = chat.lastRunUsage;
            state.messagesByChatId[chatId] = previousMessages;
          });
          return null;
        },

        retryChatRunStartup: async (chatId, runId) => {
          get().syncState();
          const scopeKey = currentScopeKey(get());
          const canRetry = () => {
            const state = get();
            const chat = _getChat(chatId);
            return Boolean(
              supportsAgentRunRecovery() &&
                currentScopeKey(state) === scopeKey &&
                chat?.runId === runId &&
                canOwnRun(state, chatId) &&
                !state.abortControllersByChatId[chatId] &&
                !state.pendingAskByChatId[chatId] &&
                !state.queuedUserMessagesByChatId[chatId] &&
                !state.queuedUserMessageInFlightByChatId[chatId] &&
                getAgentRunStartup(chat, state.getMessages(chatId))
            );
          };
          if (!canRetry()) return false;
          set((state) => {
            const chat = state.chats.find((chat) => chat.id === chatId)!;
            chat.status = "running";
            chat.lastError = null;
            chat.requiresAIConfiguration = false;
            chat.interrupted = false;
            chat.interruptionReason = undefined;
            delete state.remoteBusyChatIds[chatId];
          });
          return (
            (await get().flushRunPersistence()) &&
            canRetry() &&
            _getChat(chatId)?.status === "running"
          );
        },

        continueChatRun: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          const pendingAsk = get().getPendingAsk(chatId);
          if (
            pendingAsk &&
            !get()
              .getMessages(chatId)
              .some(
                (message) =>
                  message.role === "tool" &&
                  message.toolCallId === pendingAsk.toolCallId &&
                  message.metadata?.runId === runId
              )
          )
            return false;
          if (
            !chat ||
            chat.runId !== runId ||
            !chat.runCheckpoint ||
            chat.runCheckpoint.startupComplete !== true ||
            !isValidChatCheckpoint(
              chat,
              get().getMessages(chatId),
              chat.runCheckpoint
            ) ||
            (get().remoteBusyChatIds[chatId] && !supportsAgentRunRecovery()) ||
            !canOwnRun(get(), chatId)
          )
            return false;
          const clarifiedBindings = getAgentRunClarificationBindings(
            chat,
            get().getMessages(chatId)
          );
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.status = "running";
            delete state.remoteBusyChatIds[chatId];
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            current.requiresAIConfiguration = false;
            delete state.pendingAskByChatId[chatId];
            setLastUserInterruptionReason(state.messagesByChatId[chatId] ?? []);
            if (clarifiedBindings) {
              current.runCheckpoint!.bindings = clarifiedBindings;
              current.runCheckpoint!.referencePreflightPending = true;
              current.runMode = clarifiedBindings.runMode;
            }
            current.runCheckpoint!.suspension = "interrupted";
          });
          return get().flushRunPersistence();
        },

        saveChatRunCheckpoint: async (chatId, rawCheckpoint) => {
          const checkpoint = normalizeRunCheckpoint(rawCheckpoint);
          const chat = _getChat(chatId);
          if (
            !chat ||
            !checkpoint ||
            !canOwnRun(get(), chatId) ||
            !isValidChatCheckpoint(
              chat,
              get().getMessages(chatId),
              checkpoint
            ) ||
            !chat.runCheckpoint ||
            !samePersistedValue(
              chat.runCheckpoint.bindings,
              checkpoint.bindings
            ) ||
            chat.runCheckpoint.rootUserMessageId !==
              checkpoint.rootUserMessageId ||
            (chat.runCheckpoint.phase === "finalizing" &&
              checkpoint.phase !== "finalizing") ||
            (chat.runCheckpoint.startupComplete && !checkpoint.startupComplete)
          )
            return false;
          set((state) => {
            state.chats.find(
              (candidate) => candidate.id === chatId
            )!.runCheckpoint = checkpoint;
          });
          return get().flushRunPersistence();
        },

        reserveChatRunAttempt: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          if (
            !chat ||
            chat.archived ||
            chat.runId !== runId ||
            chat.status !== "running" ||
            !chat.runBudget ||
            !chat.runCheckpoint ||
            !canOwnRun(get(), chatId)
          )
            return { kind: "superseded" };
          if (!(await get().checkAgentStorageAdmission()))
            return { kind: "storage-unavailable" };
          const latest = _getChat(chatId);
          if (
            !latest?.runBudget ||
            latest.runId !== runId ||
            !canOwnRun(get(), chatId)
          )
            return { kind: "superseded" };
          const result = reserveRunAttempt(latest.runBudget);
          if (result.reservation.kind !== "allowed") return result.reservation;
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.runBudget = projectRunBudget(
              result.budget,
              current.modelAttempts ?? []
            );
            current.hasUnreportedUsage = true;
          });
          return (await get().flushRunPersistence())
            ? result.reservation
            : { kind: "storage-unavailable" };
        },

        recordChatModelAttempt: async (chatId, rawAttempt) => {
          const attempt = normalizeModelAttempt(rawAttempt);
          const chat = _getChat(chatId);
          if (
            !chat ||
            !attempt ||
            !get()
              .getMessages(chatId)
              .some((message) => message.metadata?.runId === attempt.runId)
          )
            return false;
          const existing = chat.modelAttempts?.find(
            (record) => record.attemptId === attempt.attemptId
          );
          if (
            chat.runBudget?.runId === attempt.runId &&
            attempt.ordinal > chat.runBudget.providerAttemptCount
          )
            return false;
          if (!existing) {
            set((state) => {
              const current = state.chats.find(
                (candidate) => candidate.id === chatId
              )!;
              current.modelAttempts = mergeModelAttempts(
                current.modelAttempts,
                [attempt]
              );
              applyChatAccounting(current);
            });
          }
          if (!(await get().flushRunPersistence())) {
            throw new Error("Agent attempt could not be persisted");
          }
          return true;
        },

        pauseChatRun: async (chatId, checkpoint) => {
          if (
            !(await get().saveChatRunCheckpoint(chatId, {
              ...checkpoint,
              suspension: "token-limit",
            }))
          )
            return false;
          set((state) => {
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (chat?.runId === checkpoint.runId) {
              chat.status = "paused";
              chat.interrupted = false;
            }
          });
          return get().flushRunPersistence();
        },

        increaseRunTokenLimit: async (chatId, limit) => {
          get().syncState();
          const chat = _getChat(chatId);
          const recoverySupported = supportsAgentRunRecovery();
          const ownsRun = recoverySupported && canOwnRun(get(), chatId);
          const budget =
            chat?.runBudget && increaseRunBudgetLimit(chat.runBudget, limit);
          if (
            !chat ||
            chat.archived ||
            !chat.runCheckpoint ||
            !budget ||
            (recoverySupported && !ownsRun) ||
            (get().remoteBusyChatIds[chatId] && !ownsRun)
          )
            return false;
          set((state) => {
            if (ownsRun) delete state.remoteBusyChatIds[chatId];
            state.chats.find(
              (candidate) => candidate.id === chatId
            )!.runBudget = budget;
          });
          return get().flushRunPersistence();
        },

        recoverLegacyApiAttempts: async (chatId, lease) => {
          const scopeKey = currentScopeKey(get());
          const candidates = () => {
            const chat = _getChat(chatId);
            return chat
              ? getLegacyApiOrphans(
                  chat,
                  get().getMessages(chatId),
                  get().getPendingAsk(chatId)
                )
              : [];
          };
          if (!candidates().length) return true;
          if (
            !scopeKey ||
            !ownsApiScope(get(), scopeKey) ||
            !supportsAgentRunRecovery()
          )
            return false;
          const recover = async () => {
            if (
              !hasAgentRunLease(scopeKey, chatId) ||
              !ownsApiScope(get(), scopeKey) ||
              get().getAbortController(chatId) ||
              !synchronizeState(get(), false)
            )
              return false;
            for (const orphan of candidates()) {
              // The old transcript does not contain the prepared bytes. This digest identifies the orphan, not an equivalent request.
              const fingerprint = await digestAgentApiJson([
                "legacy-orphan",
                scopeKey,
                chatId,
                orphan.id,
              ]);
              const result = await withAgentApiActionLease(
                scopeKey,
                fingerprint,
                () =>
                  withAgentPersistenceLock(scopeKey, () =>
                    withAgentStorageBudgetLock(() => {
                      if (
                        !ownsApiScope(get(), scopeKey) ||
                        !synchronizeState(get(), false)
                      )
                        return false;
                      if (!candidates().some((a) => a.id === orphan.id))
                        return true;
                      const now = Date.now();
                      const execution = agentApiExecutionEvidence(
                        "legacy-interrupted",
                        { attemptId: orphan.id }
                      );
                      const attempt: AgentApiAttempt = {
                        ...orphan,
                        version: 1,
                        ordinal: 0,
                        userName: useAppStore.getState().currentUser!.name,
                        chatId,
                        scopeKey,
                        fingerprint,
                        fingerprintKind: "legacy-orphan",
                        logicalActionId: orphan.id,
                        phase: "settled",
                        settledAt: now,
                        execution,
                      };
                      const state = get();
                      const messages = [...state.getMessages(chatId)];
                      insertApiToolResult(messages, {
                        id: `api-result:${orphan.id}`,
                        role: "tool",
                        chatId,
                        toolCallId: orphan.toolCallId,
                        content: JSON.stringify({ agentExecution: execution }),
                        createdTs: now,
                        metadata: {
                          apiAttemptId: orphan.id,
                          runId: orphan.runId,
                        },
                      });
                      return synchronizeState(
                        {
                          ...state,
                          chats: state.chats.map((c) =>
                            c.id === chatId
                              ? {
                                  ...c,
                                  apiAttempts: mergeAgentApiAttempts(
                                    c.apiAttempts,
                                    [attempt]
                                  ),
                                }
                              : c
                          ),
                          messagesByChatId: {
                            ...state.messagesByChatId,
                            [chatId]: messages,
                          },
                        },
                        true,
                        true
                      );
                    })
                  )
              );
              if (!result.value) return false;
            }
            return true;
          };
          try {
            return lease
              ? await lease.run(recover)
              : !!(await withAgentRunLease(scopeKey, chatId, recover)).value;
          } catch {
            return false;
          }
        },

        captureApiScope: () => {
          const revision = scopeRevision;
          const scope = getAppApiScope();
          return () => revision === scopeRevision && scope === getAppApiScope();
        },

        createApiDispatchContext: (identity) => {
          const scopeKey = currentScopeKey(get());
          if (!scopeKey || !ownsApiScope(get(), scopeKey)) return null;
          const userName = useAppStore.getState().currentUser?.name;
          if (!userName) return null;
          const chat = _getChat(identity.chatId);
          if (!chat) return null;
          const capability = normalizeAgentDiagnosticCapability(
            chat.capability
          );
          if (chat.capability !== undefined && !capability) return null;
          const context = { ...identity, scopeKey, userName };
          const revision = scopeRevision;
          const appScope = getAppApiScope();
          const sameScope = () =>
            scopeRevision === revision &&
            getAppApiScope() === appScope &&
            ownsApiScope(get(), scopeKey) &&
            useAppStore.getState().currentUser?.name === userName;
          return Object.freeze({
            ...context,
            isCurrent: () => {
              const chat = _getChat(identity.chatId);
              return (
                sameScope() &&
                !!chat &&
                !chat.archived &&
                sameAgentDiagnosticCapability(capability, chat.capability) &&
                chat.status === "running" &&
                chat.runId === identity.runId
              );
            },
            beginAgentApiAttempt: (input) =>
              sameScope()
                ? get().beginAgentApiAttempt(input)
                : Promise.resolve({
                    kind: "blocked",
                    reason: "Agent action ownership changed before dispatch.",
                  }),
            settleAgentApiAttempt: (id: string, evidence, toolResult) =>
              sameScope()
                ? get().settleAgentApiAttempt(id, evidence, toolResult)
                : Promise.resolve(false),
          } satisfies AgentApiDispatchContext);
        },

        beginAgentApiAttempt: async (input) => {
          const { scopeKey, userName, chatId, logicalActionId } = input;
          const chat = _getChat(chatId);
          if (
            !chat ||
            !ownsApiScope(get(), scopeKey) ||
            useAppStore.getState().currentUser?.name !== userName ||
            chat.apiAttempts?.some((a) => a.logicalActionId === logicalActionId)
          ) {
            return {
              kind: "blocked",
              reason:
                "The approved action was already used or its owner changed.",
            };
          }
          const marker: AgentApiAttempt = {
            ...input,
            version: 1,
            id: uuidv4(),
            ordinal: 1,
            phase: "dispatching",
            startedAt: Date.now(),
          };
          set((state) => {
            const owner = state.chats.find((c) => c.id === chatId)!;
            owner.apiAttempts = mergeAgentApiAttempts(owner.apiAttempts, [
              marker,
            ]);
          });
          // The receipt is best-effort; the pending permission has already been consumed.
          await get()
            .flushRunPersistence()
            .catch(() => false);
          return { kind: "admitted", attemptId: marker.id };
        },

        settleAgentApiAttempt: async (id, evidence, toolResult) => {
          const state = get();
          const chat = state.chats.find((c) =>
            c.apiAttempts?.some((a) => a.id === id)
          );
          const attempt = chat?.apiAttempts?.find((a) => a.id === id);
          if (
            !chat ||
            !attempt ||
            !ownsApiScope(state, attempt.scopeKey) ||
            useAppStore.getState().currentUser?.name !== attempt.userName ||
            evidence.attemptId !== id
          )
            return false;
          let envelope;
          try {
            envelope = JSON.parse(toolResult);
          } catch {
            return false;
          }
          if (
            !isRecord(envelope) ||
            !isAgentApiEvidenceForEnvelope(
              evidence,
              envelope,
              attempt.operationId
            ) ||
            !samePersistedValue(envelope.agentExecution, evidence)
          )
            return false;
          const now = Date.now();
          set((draft) => {
            const current = draft.chats.find((c) => c.id === chat.id)!;
            current.apiAttempts = mergeAgentApiAttempts(current.apiAttempts, [
              {
                ...attempt,
                phase: "settled",
                execution: evidence,
                settledAt: now,
              },
            ]);
            const messages = (draft.messagesByChatId[chat.id] ??= []);
            if (!messages.some((m) => m.metadata?.apiAttemptId === id))
              insertApiToolResult(messages, {
                id: `api-result:${id}`,
                role: "tool",
                chatId: chat.id,
                createdTs: now,
                toolCallId: attempt.toolCallId,
                content: toolResult,
                metadata: { apiAttemptId: id, runId: attempt.runId },
              });
          });
          return get().flushRunPersistence();
        },

        endPausedRun: async (chatId) => {
          get().syncState();
          const chat = _getChat(chatId);
          const recoverySupported = supportsAgentRunRecovery();
          const ownsRun = recoverySupported && canOwnRun(get(), chatId);
          if (
            !chat ||
            (!chat.runCheckpoint && !(chat.runId && chat.runBudget)) ||
            (recoverySupported && !ownsRun) ||
            (chat.status === "running" &&
              (!ownsRun || get().getAbortController(chatId))) ||
            (get().remoteBusyChatIds[chatId] && !ownsRun)
          )
            return false;
          set((state) => {
            const current = state.chats.find(
              (candidate) => candidate.id === chatId
            )!;
            current.status = "idle";
            current.runId = null;
            current.runMode = undefined;
            current.runCheckpoint = undefined;
            current.interrupted = false;
            current.interruptionReason = undefined;
            current.lastError = null;
            if (ownsRun) delete state.remoteBusyChatIds[chatId];
            delete state.pendingAskByChatId[chatId];
            delete state.queuedUserMessagesByChatId[chatId];
            delete state.queuedUserMessageInFlightByChatId[chatId];
            pauseFollowUpQueue(state, chatId);
          });
          return get().flushRunPersistence();
        },

        flushRunPersistence: async () => {
          const state = get();
          const key =
            state.workspaceName && state.userEmail
              ? getAgentStateKey(state.workspaceName, state.userEmail)
              : null;
          if (!key) return false;
          if (!getSafeStorage()) {
            updateStoragePressure(
              key,
              new Error("Browser storage is unavailable"),
              "saving"
            );
            return false;
          }
          if (!supportsAgentRunRecovery()) return synchronizeState(state, true);
          pendingStatesByScope.set(key, state);
          const writer = getWriter(key);
          writer.schedule();
          let saved = await writer.flush();
          if (!saved) updateStoragePressure(key, writer.getError(), "saving");
          if (
            currentScopeKey(get()) === key &&
            (saved || writer.getError() instanceof AgentStorageFullError)
          ) {
            const cleaned = await automaticallyRetainHistory(key, !saved);
            if (!saved && cleaned) {
              pendingStatesByScope.set(key, get());
              writer.schedule();
              saved = await writer.flush();
              if (!saved)
                updateStoragePressure(key, writer.getError(), "saving");
            }
          }
          return saved;
        },

        checkAgentStorageAdmission: async () => {
          const key = currentScopeKey(get());
          if (!key || !(await get().flushRunPersistence())) return false;
          const check = () => {
            if (currentScopeKey(get()) !== key) return false;
            try {
              const storage = getSafeStorage();
              if (!storage) throw new Error("Browser storage is unavailable");
              const bytes = estimateAgentStorageBytes(storage);
              if (bytes > AGENT_STORAGE_ADMISSION_BYTES) {
                updateStoragePressure(
                  key,
                  new AgentStorageFullError(
                    "budget",
                    bytes,
                    AGENT_STORAGE_ADMISSION_BYTES
                  ),
                  "admission"
                );
                return false;
              }
              clearStoragePressure(key);
              return true;
            } catch (error) {
              updateStoragePressure(key, error, "admission");
              return false;
            }
          };
          try {
            return supportsAgentRunRecovery()
              ? await withAgentStorageBudgetLock(check)
              : check();
          } catch (error) {
            updateStoragePressure(key, error, "admission");
            return false;
          }
        },

        freeAgentStorage: (options) => freeAgentStorage(options),

        getRunPersistenceError: () => {
          const key = currentScopeKey(get());
          const error = key ? writersByScope.get(key)?.getError() : undefined;
          return error instanceof AgentStorageFullError && error.cause
            ? error.cause
            : error;
        },

        canUseAgentExecution: () => {
          const key = currentScopeKey(get());
          if (!key) return false;
          if (supportsAgentRunRecovery()) return true;
          try {
            return !getSafeStorage()?.getItem(
              key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY)
            );
          } catch {
            return false;
          }
        },

        finishChatRun: (chatId, options = {}) => {
          get().setChatStatus(chatId, options.status ?? DEFAULT_CHAT_STATUS, {
            lastError: options.lastError ?? null,
            requiresAIConfiguration: options.requiresAIConfiguration ?? false,
            interrupted: false,
          });
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.runId = null;
              chat.runMode = undefined;
              chat.runCheckpoint = undefined;
            }
          });
        },

        interruptChatRun: (chatId, page, reason) => {
          const interruptionReason =
            reason === "user" || _getChat(chatId)?.interruptionReason === "user"
              ? "user"
              : "unexpected";
          get().setChatStatus(chatId, DEFAULT_CHAT_STATUS, {
            interrupted: true,
            interruptionReason,
            page,
          });
          set((state) => {
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (chat?.runCheckpoint)
              chat.runCheckpoint.suspension = "interrupted";
            setLastUserInterruptionReason(
              state.messagesByChatId[chatId] ?? [],
              interruptionReason
            );
          });
        },

        recoverChatRun: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          const state = get();
          const queued = state.queuedUserMessagesByChatId[chatId];
          const key = currentScopeKey(state);
          const saved = key ? storageSnapshotsByScope.get(key) : undefined;
          const checkpoint = saved?.chats.find(
            ({ id }) => id === chatId
          )?.runCheckpoint;
          const recorded = saved?.messagesByChatId[chatId]?.find(
            ({ id }) => id === queued?.id
          );
          // The startup receipt no longer owns execution once its root is durable.
          const committedStartup = Boolean(
            queued &&
              checkpoint?.runId === runId &&
              checkpoint.rootUserMessageId === queued.id &&
              recorded?.role === "user"
          );
          // A follow-up may have been promoted before its checkpoint was written.
          const abandonedPromotion = Boolean(
            queued?.fromFollowUpQueue &&
              saved?.queuedUserMessagesByChatId[chatId]?.id === queued.id &&
              !checkpoint &&
              runId === null &&
              (!recorded || recorded.role === "user")
          );
          if (
            !chat ||
            chat.archived ||
            (chat.runId ?? null) !== runId ||
            chat.status === "awaiting_user" ||
            !get().remoteBusyChatIds[chatId] ||
            get().getAbortController(chatId) ||
            state.followUpDrainChatIds[chatId] ||
            ((queued || state.queuedUserMessageInFlightByChatId[chatId]) &&
              !committedStartup &&
              !abandonedPromotion) ||
            !supportsAgentRunRecovery() ||
            !canOwnRun(get(), chatId)
          )
            return false;
          set((state) => {
            delete state.remoteBusyChatIds[chatId];
            if (committedStartup || abandonedPromotion) {
              if (abandonedPromotion && !recorded && queued)
                restorePromotedFollowUp(state, queued);
              else delete state.queuedUserMessagesByChatId[chatId];
              delete state.queuedUserMessageInFlightByChatId[chatId];
            }
            pauseFollowUpQueue(state, chatId);
          });
          if (chat.status === "running" || (abandonedPromotion && recorded)) {
            if (chat.runCheckpoint?.suspension === "token-limit") {
              get().setChatStatus(chatId, "paused");
            } else {
              get().interruptChatRun(chatId);
              get().clearPendingAsk(chatId);
            }
          }
          if (await get().flushRunPersistence()) return true;
          const current = _getChat(chatId);
          if (current && (current.runId ?? null) === runId) {
            set((state) => {
              state.remoteBusyChatIds[chatId] = true;
            });
          }
          return false;
        },

        // Pending ask
        setPendingAsk: (chatId, pendingAsk) => {
          set((state) => {
            state.pendingAskByChatId[chatId] = pendingAsk;
            if (
              state.chats.find((chat) => chat.id === chatId)?.status ===
                "awaiting_user" &&
              state.queuedUserMessagesByChatId[chatId] &&
              state.messagesByChatId[chatId]?.some(
                (message) =>
                  message.id === state.queuedUserMessagesByChatId[chatId]?.id &&
                  message.role === "user"
              )
            ) {
              delete state.queuedUserMessagesByChatId[chatId];
              delete state.queuedUserMessageInFlightByChatId[chatId];
            }
            _touchChatInDraft(state.chats, chatId);
          });
          return pendingAsk;
        },

        clearPendingAsk: (chatId) => {
          const id = chatId ?? get().currentChatId;
          if (!id || !get().pendingAskByChatId[id]) return;
          set((state) => {
            delete state.pendingAskByChatId[id];
            _touchChatInDraft(state.chats, id);
          });
        },

        awaitUser: (chatId, pendingAsk) => {
          get().setChatStatus(chatId, "awaiting_user");
          return get().setPendingAsk(chatId, pendingAsk);
        },

        recoverPendingAsk: async (chatId, runId) => {
          get().syncState();
          const chat = _getChat(chatId);
          if (
            !chat ||
            chat.runId !== runId ||
            chat.status !== "awaiting_user" ||
            !get().getPendingAsk(chatId) ||
            get().getAbortController(chatId) ||
            !supportsAgentRunRecovery() ||
            !canOwnRun(get(), chatId) ||
            !getAgentRunContinuation(chat, get().getMessages(chatId))
          )
            return false;
          set((state) => {
            delete state.remoteBusyChatIds[chatId];
            pauseFollowUpQueue(state, chatId);
          });
          if (await get().flushRunPersistence()) return true;
          if (_getChat(chatId)?.runId === runId) {
            set((state) => {
              state.remoteBusyChatIds[chatId] = true;
            });
          }
          return false;
        },

        cancelPendingAsk: (chatId) => {
          get().syncState();
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          const pendingAsk = id ? get().pendingAskByChatId[id] : undefined;
          if (
            !id ||
            get().remoteBusyChatIds[id] ||
            chat?.status !== "awaiting_user" ||
            !pendingAsk ||
            pendingAsk.apiPermission
          ) {
            return false;
          }
          const toolMessage: AgentMessage = {
            id: uuidv4(),
            chatId: id,
            createdTs: Date.now(),
            role: "tool",
            toolCallId: pendingAsk.toolCallId,
            content: JSON.stringify({ kind: "cancelled", reason: "user" }),
            metadata: chat.runId ? { runId: chat.runId } : undefined,
          };
          set((state) => {
            const currentChat = state.chats.find(
              (candidate) => candidate.id === id
            );
            if (!currentChat) return;
            delete state.pendingAskByChatId[id];
            pauseFollowUpQueue(state, id);
            state.messagesByChatId[id] ??= [];
            state.messagesByChatId[id].push(toolMessage);
            currentChat.status = DEFAULT_CHAT_STATUS;
            currentChat.runId = null;
            currentChat.runMode = undefined;
            currentChat.runCheckpoint = undefined;
            currentChat.interrupted = false;
            currentChat.interruptionReason = undefined;
            currentChat.lastError = null;
            setLastUserInterruptionReason(state.messagesByChatId[id] ?? []);
            _touchChatInDraft(state.chats, id);
          });
          return true;
        },

        answerPendingAsk: (chatId, response, metadata) => {
          get().syncState();
          if (get().remoteBusyChatIds[chatId]) return null;
          const pendingAsk = get().getPendingAsk(chatId);
          if (!pendingAsk || pendingAsk.kind !== response.kind) return null;
          let normalizedResponse = response;
          if (response.kind === "choose") {
            const option = pendingAsk.options?.find(
              (candidate) => candidate.value === response.value
            );
            if (!option) return null;
            if (option.kind === "custom-input") {
              if (typeof response.input !== "string" || !response.input.trim())
                return null;
              const input = response.input.trim();
              normalizedResponse = {
                ...response,
                answer: `${option.label}: ${input}`,
                input,
              };
            } else if (response.input !== undefined) {
              return null;
            }
          }
          const toolMessage = get().addMessage({
            chatId,
            role: "tool",
            toolCallId: pendingAsk.toolCallId,
            content: JSON.stringify(normalizedResponse),
            metadata,
          });
          get().clearPendingAsk(chatId);
          return toolMessage;
        },

        // Token
        incrementChatTotalTokens: (chatId, totalTokensUsed) => {
          if (!_getChat(chatId) || totalTokensUsed <= 0) return null;
          set((state) => {
            const chat = state.chats.find((c) => c.id === chatId);
            if (chat) {
              chat.totalTokensUsed += totalTokensUsed;
              if (chat.legacyUsage)
                chat.legacyUsage.totalTokensUsed += totalTokensUsed;
              _touchChatInDraft(state.chats, chatId);
            }
          });
          return _getChat(chatId);
        },

        recordChatModelCallUsage: (chatId, runId, rawUsage) => {
          const usage = normalizeModelCallUsage(rawUsage);
          if (!usage || !_getChat(chatId)) return;
          set((state) => {
            const chat = state.chats.find(
              (candidate) => candidate.id === chatId
            );
            if (!chat) return;
            chat.totalTokensUsed += usage.totalTokens ?? 0;
            if (usage.totalTokens === undefined) chat.hasUnreportedUsage = true;
            if (chat.legacyUsage) {
              chat.legacyUsage.totalTokensUsed += usage.totalTokens ?? 0;
              chat.legacyUsage.hasUnreportedUsage ||=
                usage.totalTokens === undefined;
            }
            if (chat.lastRunUsage?.runId === runId) {
              chat.lastRunUsage.calls.push(usage);
            }
            _touchChatInDraft(state.chats, chatId);
          });
        },

        // Abort / cancel
        cancel: (chatId, reason) => {
          get().syncState();
          const id = chatId ?? get().currentChatId;
          if (!id || get().remoteBusyChatIds[id]) return;
          set((state) => pauseFollowUpQueue(state, id));
          get().getAbortController(id)?.abort();
          get().setAbortController(id, null);
          if (get().isChatRunning(id)) {
            get().interruptChatRun(id, undefined, reason);
          }
        },

        // Error
        clearError: (chatId) => {
          get().syncState();
          const id = chatId ?? get().currentChatId;
          const chat = _getChat(id);
          if (!chat || get().remoteBusyChatIds[chat.id]) return;
          set((state) => {
            const c = state.chats.find((c) => c.id === id);
            if (c) {
              c.lastError = null;
              c.requiresAIConfiguration = false;
              c.interrupted = false;
              c.interruptionReason = undefined;
              if (!c.runCheckpoint) c.runId = null;
              if (c.status === "error") {
                c.status = DEFAULT_CHAT_STATUS;
              }
              _touchChatInDraft(state.chats, c.id);
            }
          });
        },

        // Persistence
        setWorkspaceName: (workspaceName, userEmail) => {
          const nextUserEmail = userEmail ?? get().userEmail ?? "";
          const stateKey = workspaceName
            ? getAgentStateKey(workspaceName, nextUserEmail)
            : null;
          const identityChanged = get().userEmail !== nextUserEmail;
          if (!workspaceName || !stateKey) {
            if (!get().workspaceName && !get().userEmail) return false;
            get().saveState();
            for (const controller of Object.values(
              get().abortControllersByChatId
            )) {
              controller.abort();
            }
            const empty = ensureCurrentChat(createEmptyPersistedState());
            set((draft) => {
              draft.workspaceName = null;
              draft.userEmail = null;
              draft.chats = empty.chats;
              draft.messagesByChatId = empty.messagesByChatId;
              draft.pendingAskByChatId = empty.pendingAskByChatId;
              draft.draftsByChatId = empty.draftsByChatId;
              draft.landingDraft = { text: "" };
              draft.landingContext = undefined;
              draft.currentChatId = empty.currentChatId;
              draft.abortControllersByChatId = {};
              draft.queuedUserMessagesByChatId = {};
              draft.followUpQueuesByChatId = {};
              draft.followUpDrainChatIds = {};
              draft.queuedUserMessageInFlightByChatId = {};
              draft.remoteBusyChatIds = {};
              draft.expressionDraftTargetsByChatId = {};
              draft.sqlFixSourcesByChatId = {};
              draft.sqlFixDraftsById = {};
              draft.preferences = { ...DEFAULT_AGENT_PREFERENCES };
              draft.preferencesStorageError = false;
              draft.storagePressure = undefined;
            });
            return false;
          }
          if (
            get().workspaceName === workspaceName &&
            get().userEmail === nextUserEmail
          ) {
            return true;
          }
          if (
            !identityChanged &&
            (_hasRunningChat() ||
              Object.keys(get().queuedUserMessagesByChatId).length > 0)
          ) {
            return false;
          }

          get().saveState();
          if (identityChanged) {
            for (const controller of Object.values(
              get().abortControllersByChatId
            )) {
              controller.abort();
            }
          }
          const storage = getSafeStorage();
          if (
            !migrateLegacyState(storage, workspaceName, nextUserEmail) &&
            supportsAgentRunRecovery() &&
            !storage?.getItem(stateKey)
          ) {
            if (identityChanged) get().setWorkspaceName(undefined, "");
            return false;
          }
          const oldKey = stateKey.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY);
          const v3Key = stateKey.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY);
          const existingV3 = storage?.getItem(v3Key);
          const serialized = storage?.getItem(stateKey);
          const sourceKey = supportsAgentRunRecovery()
            ? serialized != null
              ? stateKey
              : oldKey
            : existingV3 != null
              ? v3Key
              : stateKey;
          let loadedState: PersistedAgentState | null = null;
          const source =
            sourceKey === stateKey ? serialized : storage?.getItem(sourceKey);
          try {
            loadedState = readPersistedState(
              source,
              sourceKey,
              supportsAgentRunRecovery() ? stateKey : sourceKey,
              nextUserEmail
            );
          } catch {
            // No current cache can replace an unreadable migration source.
            if (sourceKey !== stateKey || !supportsAgentRunRecovery()) {
              if (identityChanged) get().setWorkspaceName(undefined, "");
              return false;
            }
          }
          if (sourceKey === oldKey && oldKey !== stateKey) {
            if (source != null) lastSerializedByScope.set(oldKey, source);
            else lastSerializedByScope.delete(oldKey);
          }
          const persistedState = restoreDraftShadow(
            stateKey,
            ensureCurrentChat(loadedState ?? createEmptyPersistedState())
          );
          if (supportsAgentRunRecovery() && !serialized) {
            for (const chat of persistedState.chats) {
              chat.legacyUsage ??= {
                totalTokensUsed: chat.totalTokensUsed,
                hasUnreportedUsage: Boolean(
                  chat.hasUnreportedUsage ||
                    chat.totalTokensUsed > 0 ||
                    (persistedState.messagesByChatId[chat.id]?.length ?? 0) > 0
                ),
              };
              chat.modelAttempts ??= [];
              applyChatAccounting(chat);
            }
          }
          const preferencesState = loadAgentPreferences(
            workspaceName,
            nextUserEmail
          );
          snapshotsByScope.set(
            stateKey,
            serialized && loadedState
              ? persistedState
              : createEmptyPersistedState()
          );
          storageSnapshotsByScope.delete(stateKey);
          lastSerializedByScope.delete(stateKey);
          // Unreadable history must be reparsed, not cached as an empty baseline.
          if (!serialized || loadedState) {
            storageSnapshotsByScope.set(
              stateKey,
              serialized ? persistedState : createEmptyPersistedState()
            );
            if (serialized) lastSerializedByScope.set(stateKey, serialized);
          }
          set((draft) => {
            draft.workspaceName = workspaceName;
            draft.storagePressure = undefined;
            draft.userEmail = nextUserEmail;
            draft.chats = persistedState.chats;
            draft.messagesByChatId = persistedState.messagesByChatId;
            draft.pendingAskByChatId = persistedState.pendingAskByChatId;
            draft.draftsByChatId = persistedState.draftsByChatId;
            draft.landingDraft = { text: "" };
            draft.landingContext = undefined;
            draft.currentChatId = persistedState.currentChatId;
            draft.abortControllersByChatId = {};
            draft.queuedUserMessagesByChatId =
              persistedState.queuedUserMessagesByChatId;
            draft.followUpQueuesByChatId =
              persistedState.followUpQueuesByChatId ?? {};
            draft.followUpDrainChatIds = {};
            draft.queuedUserMessageInFlightByChatId = {};
            draft.remoteBusyChatIds = supportsAgentRunRecovery()
              ? getPersistedFollowUpOwners(persistedState)
              : {};
            draft.expressionDraftTargetsByChatId = {};
            draft.sqlFixSourcesByChatId = {};
            draft.sqlFixDraftsById = {};
            draft.preferences = preferencesState.preferences;
            draft.preferencesStorageError =
              preferencesState.preferencesStorageError;
          });
          get().loadWindowState();
          for (const chat of persistedState.chats)
            void get().recoverLegacyApiAttempts(chat.id);
          return true;
        },

        saveWindowState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName)
            : null;
          if (!storage || !key) return;
          const state = get();
          storage.setItem(
            key,
            JSON.stringify({
              layoutVersion: AGENT_WINDOW_LAYOUT_VERSION,
              size: state.size,
              sidebarWidth: state.sidebarWidth,
            })
          );
        },

        loadWindowState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const key = workspaceName
            ? getWorkspaceStorageKey(AGENT_WINDOW_KEY, workspaceName)
            : null;
          if (!key) return;
          const saved =
            storage?.getItem(key) ?? storage?.getItem(AGENT_WINDOW_KEY);
          if (!saved) {
            const defaultWindowState = getInitialWindowState();
            set((draft) => {
              draft.size = defaultWindowState.size;
              draft.sidebarWidth = defaultWindowState.sidebarWidth;
            });
            return;
          }
          try {
            const state = JSON.parse(saved) as {
              layoutVersion?: number;
              position?: unknown;
              size?: { width?: number; height?: number };
              sidebarWidth?: number;
            };
            const isCurrentLayout =
              state.layoutVersion === AGENT_WINDOW_LAYOUT_VERSION;
            const hasLegacyPosition = state.position !== undefined;
            const defaultWindowState = getInitialWindowState();
            set((draft) => {
              if (
                isCurrentLayout &&
                typeof state.size?.width === "number" &&
                typeof state.size?.height === "number"
              ) {
                draft.size = {
                  width: state.size.width,
                  height: state.size.height,
                };
              } else if (!isCurrentLayout) {
                draft.size = defaultWindowState.size;
              }
              if (typeof state.sidebarWidth === "number") {
                draft.sidebarWidth = state.sidebarWidth;
              }
            });
            if (
              !isCurrentLayout ||
              !storage?.getItem(key) ||
              hasLegacyPosition
            ) {
              get().saveWindowState();
            }
          } catch {
            storage?.removeItem(key);
          }
        },

        saveState: () => {
          clearDraftPersistTimer();
          synchronizeState(get(), true);
        },

        syncState: () => synchronizeState(get(), false),

        loadState: () => {
          const storage = getSafeStorage();
          const workspaceName = get().workspaceName;
          const userEmail = get().userEmail;
          const key =
            workspaceName && userEmail
              ? getAgentStateKey(workspaceName, userEmail)
              : null;
          if (!key) return;
          const serialized = storage?.getItem(key);
          if (
            serialized == null &&
            supportsAgentRunRecovery() &&
            storage?.getItem(
              key.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY)
            ) != null
          )
            return;
          let loadedState: PersistedAgentState | null;
          try {
            loadedState = readPersistedState(
              serialized,
              key,
              key,
              userEmail ?? undefined
            );
          } catch {
            return;
          }
          const state = ensureCurrentChat(
            loadedState ?? createEmptyPersistedState()
          );
          snapshotsByScope.set(key, state);
          storageSnapshotsByScope.delete(key);
          lastSerializedByScope.delete(key);
          if (!serialized || loadedState) {
            storageSnapshotsByScope.set(key, state);
            if (serialized) lastSerializedByScope.set(key, serialized);
          }
          set((draft) => {
            draft.chats = state.chats;
            draft.messagesByChatId = state.messagesByChatId;
            draft.pendingAskByChatId = state.pendingAskByChatId;
            draft.draftsByChatId = state.draftsByChatId;
            draft.currentChatId = state.currentChatId;
            draft.queuedUserMessagesByChatId = state.queuedUserMessagesByChatId;
            draft.followUpQueuesByChatId = state.followUpQueuesByChatId ?? {};
            draft.followUpDrainChatIds = {};
            draft.queuedUserMessageInFlightByChatId = {};
            draft.remoteBusyChatIds = supportsAgentRunRecovery()
              ? getPersistedFollowUpOwners(state)
              : {};
          });
          for (const chat of state.chats)
            void get().recoverLegacyApiAttempts(chat.id);
        },
      };
    })
  );

  const clearStoragePressure = (key: string) => {
    if (
      currentScopeKey(store.getState()) === key &&
      store.getState().storagePressure
    ) {
      store.setState((draft) => {
        draft.storagePressure = undefined;
      });
    }
  };

  const updateStoragePressure = (
    key: string,
    error: unknown,
    phase: AgentStoragePressure["phase"]
  ) => {
    if (currentScopeKey(store.getState()) !== key) return;
    let bytes = error instanceof AgentStorageFullError ? error.bytes : 0;
    let otherScopeBytes = 0;
    try {
      const storage = getSafeStorage();
      if (storage) {
        bytes ||= estimateAgentStorageBytes(storage);
        const value = storage.getItem(key);
        otherScopeBytes =
          estimateAgentStorageBytes(storage) -
          (value === null ? 0 : 2 * (key.length + value.length));
      }
    } catch {
      /* Storage may also be unreadable. Keep the original failure. */
    }
    store.setState((draft) => {
      draft.storagePressure = {
        reason:
          error instanceof AgentStorageFullError
            ? error.reason
            : isAgentStorageQuotaError(error)
              ? "quota"
              : "unavailable",
        phase,
        bytes,
        limitBytes:
          error instanceof AgentStorageFullError
            ? error.limitBytes
            : AGENT_STORAGE_LIMIT_BYTES,
        otherScopeBytes: Math.max(0, otherScopeBytes),
      };
    });
  };

  const isCleanupProtected = (state: AgentState, chatId: string): boolean => {
    const chat = state.chats.find((chat) => chat.id === chatId);
    if (!chat) return true;
    const draft = state.draftsByChatId[chatId];
    if (
      state.currentChatId === chatId &&
      draft &&
      (draft.text || draft.project || draft.resource || draft.mentions?.length)
    )
      return true;
    if (
      chat.runId ||
      chat.runCheckpoint ||
      chat.status === "running" ||
      chat.status === "paused" ||
      chat.status === "awaiting_user" ||
      state.abortControllersByChatId[chatId] ||
      state.pendingAskByChatId[chatId] ||
      state.queuedUserMessagesByChatId[chatId] ||
      state.followUpQueuesByChatId[chatId]?.messages.length ||
      state.followUpDrainChatIds[chatId] ||
      state.queuedUserMessageInFlightByChatId[chatId] ||
      state.remoteBusyChatIds[chatId]
    )
      return true;
    const key = currentScopeKey(state);
    const saved = new Map(
      (key
        ? (storageSnapshotsByScope.get(key)?.messagesByChatId[chatId] ?? [])
        : []
      ).map((message) => [message.id, message])
    );
    return (state.messagesByChatId[chatId] ?? []).some(
      (message) =>
        message.role === "tool" &&
        !samePersistedValue(message, saved.get(message.id))
    );
  };

  const getLocallyProtectedChats = (state: AgentState) => {
    const ids = new Set(Object.keys(state.expressionDraftTargetsByChatId));
    for (const draft of Object.values(state.sqlFixDraftsById))
      ids.add(draft.chatId);
    const key = currentScopeKey(state);
    const saved = key ? storageSnapshotsByScope.get(key) : undefined;
    for (const chat of state.chats) {
      if (
        isCleanupProtected(state, chat.id) ||
        !samePersistedValue(
          state.messagesByChatId[chat.id] ?? [],
          saved?.messagesByChatId[chat.id] ?? []
        )
      )
        ids.add(chat.id);
    }
    return ids;
  };

  const chatProtection = createAgentChatProtection(() => {
    const state = store.getState();
    const key = currentScopeKey(state);
    if (
      !protectionSuspended &&
      !state.chatProtectionReady &&
      key &&
      state.currentChatId &&
      chatProtection.isReady(key, state.currentChatId)
    )
      state.syncState();
    refreshChatProtection();
  });
  const refreshChatProtection = () => {
    const state = store.getState();
    const key = currentScopeKey(state);
    const protectedIds = new Set<string>();
    if (key) {
      for (const id of getLocallyProtectedChats(state)) protectedIds.add(id);
      if (!protectionSuspended && state.currentChatId)
        protectedIds.add(state.currentChatId);
      for (const [id, draft] of Object.entries(state.draftsByChatId)) {
        if (
          draft.text ||
          draft.project ||
          draft.resource ||
          draft.mentions?.length
        )
          protectedIds.add(id);
      }
    }
    for (const id of reclamationChatIds) protectedIds.delete(id);
    chatProtection.update(key, [...protectedIds]);
    const ready =
      !key ||
      !state.currentChatId ||
      (!protectionSuspended &&
        chatProtection.isReady(key, state.currentChatId));
    if (store.getState().chatProtectionReady !== ready)
      store.setState({ chatProtectionReady: ready });
  };

  const scheduleAutomaticCleanup = (key: string) => {
    if (
      !supportsAgentChatProtection() ||
      reclaimingStorage ||
      automaticCleanups.has(key) ||
      cleanupTimer !== undefined
    )
      return;
    // Cleanup acquires execution leases before persistence, outside the writer.
    cleanupTimer = setTimeout(() => {
      cleanupTimer = undefined;
      const pressure = store.getState().storagePressure;
      void automaticallyRetainHistory(
        key,
        pressure?.phase === "saving" &&
          (pressure.reason === "budget" || pressure.reason === "quota")
      );
    }, 0);
  };

  const commitAutomaticCleanup = async (
    key: string,
    ids: string[],
    quotaFailure: boolean
  ): Promise<{ saved: boolean; blocked?: string }> => {
    const ordered = [...ids].sort((a, b) => a.localeCompare(b));
    const withLeases = async (
      index: number,
      execution: boolean
    ): Promise<{ saved: boolean; blocked?: string }> => {
      if (index < ordered.length) {
        const id = ordered[index];
        const result = execution
          ? await withAgentRunLease(key, id, () => withLeases(index + 1, true))
          : await withAgentChatEvictionLease(key, id, () =>
              withLeases(index + 1, false)
            );
        return result.acquired && result.value
          ? result.value
          : { saved: false, blocked: id };
      }
      if (!execution) return withLeases(0, true);
      return withAgentPersistenceLock(key, () =>
        withAgentStorageBudgetLock(() => {
          if (currentScopeKey(store.getState()) !== key)
            return { saved: false };
          store.getState().syncState();
          const state = store.getState();
          if (quotaFailure) {
            try {
              if (synchronizeState(state, true, true)) quotaFailure = false;
            } catch (error) {
              if (!(error instanceof AgentStorageFullError)) throw error;
              quotaFailure = error.reason === "quota";
            }
          }
          const allowed = new Set(
            getAgentHistoryEvictionCandidates(
              persistedSnapshot(state),
              getLocallyProtectedChats(state)
            )
          );
          const blocked = ordered.find((id) => !allowed.has(id));
          if (blocked) return { saved: false, blocked };
          const storage = getSafeStorage();
          if (!storage) return { saved: false };
          const snapshot = omitAgentQuerySamples(persistedSnapshot(state));
          const previous = storage.getItem(key);
          const bytesAfter = (count: number) =>
            estimateAgentStorageBytes(storage, {
              key,
              value: encodeAgentStoredState(
                JSON.stringify(
                  removeAgentHistoryChats(
                    snapshot,
                    new Set(ids.slice(0, count))
                  )
                ),
                previous,
                true
              ),
            });
          if (
            !quotaFailure &&
            bytesAfter(0) <= AGENT_STORAGE_CLEANUP_TARGET_BYTES
          )
            return { saved: true };
          if (bytesAfter(ids.length) > AGENT_STORAGE_ADMISSION_BYTES)
            return { saved: false };
          let count = ids.length;
          if (!quotaFailure) {
            let low = 1;
            while (low < count) {
              const middle = Math.floor((low + count) / 2);
              if (bytesAfter(middle) <= AGENT_STORAGE_CLEANUP_TARGET_BYTES)
                count = middle;
              else low = middle + 1;
            }
          }
          const retained = removeAgentHistoryChats(
            snapshot,
            new Set(ids.slice(0, count))
          );
          const candidate = {
            ...state,
            ...retained,
            followUpQueuesByChatId: retained.followUpQueuesByChatId ?? {},
          };
          const saved = synchronizeState(candidate, true, true);
          if (saved) pendingStatesByScope.set(key, store.getState());
          return { saved };
        })
      );
    };
    return withLeases(0, false);
  };

  const automaticallyRetainHistory = (
    key: string,
    failedSave = false
  ): Promise<boolean> => {
    const pending = automaticCleanups.get(key);
    if (pending) return pending;
    if (reclaimingStorage) return Promise.resolve(false);
    if (!supportsAgentChatProtection()) return Promise.resolve(false);
    const cleanup = Promise.resolve()
      .then(async () => {
        if (currentScopeKey(store.getState()) !== key) return false;
        const storage = getSafeStorage();
        if (!storage) return false;
        let quotaFailure =
          failedSave && store.getState().storagePressure?.reason === "quota";
        if (
          !failedSave &&
          estimateAgentStorageBytes(storage) <
            AGENT_STORAGE_CLEANUP_TRIGGER_BYTES
        )
          return false;
        omitQuerySamplesForScope.add(key);
        try {
          // Discard reproducible Query previews before any completed conversation.
          for (;;) {
            try {
              await withAgentPersistenceLock(key, () =>
                withAgentStorageBudgetLock(() => {
                  if (currentScopeKey(store.getState()) !== key) return false;
                  return synchronizeState(store.getState(), true, true);
                })
              );
              quotaFailure = false;
              break;
            } catch (error) {
              if (!(error instanceof AgentStorageFullError)) return false;
              quotaFailure = error.reason === "quota";
              const freed = await reclaimOtherAgentHistory(storage, {
                currentKey: key,
                userEmail: store.getState().userEmail!,
                mode: "automatic",
                isCurrent: () => currentScopeKey(store.getState()) === key,
                force: true,
              });
              if (!freed) break;
            }
          }
          await reclaimOtherAgentHistory(storage, {
            currentKey: key,
            userEmail: store.getState().userEmail!,
            mode: "automatic",
            isCurrent: () => currentScopeKey(store.getState()) === key,
          });
          const blocked = new Set<string>();
          let minimumCount = quotaFailure ? 1 : 0;
          for (;;) {
            if (currentScopeKey(store.getState()) !== key) return false;
            store.getState().syncState();
            const state = store.getState();
            const protectedIds = await getProtectedAgentChatIds(key);
            for (const id of getLocallyProtectedChats(state))
              protectedIds.add(id);
            for (const id of blocked) protectedIds.add(id);
            const snapshot = omitAgentQuerySamples(persistedSnapshot(state));
            const ids = getAgentHistoryEvictionCandidates(
              snapshot,
              protectedIds
            );
            const previous = storage.getItem(key);
            const bytesAfter = (count: number) => {
              const candidate = removeAgentHistoryChats(
                snapshot,
                new Set(ids.slice(0, count))
              );
              return estimateAgentStorageBytes(storage, {
                key,
                value: encodeAgentStoredState(
                  serializePersistedState(candidate),
                  previous,
                  true
                ),
              });
            };
            if (
              !quotaFailure &&
              bytesAfter(0) <= AGENT_STORAGE_CLEANUP_TARGET_BYTES
            )
              return true;
            if (
              !ids.length ||
              bytesAfter(ids.length) > AGENT_STORAGE_ADMISSION_BYTES
            )
              return !failedSave;
            let low = Math.min(minimumCount, ids.length);
            let high = ids.length;
            while (low < high) {
              const middle = Math.floor((low + high) / 2);
              if (bytesAfter(middle) <= AGENT_STORAGE_CLEANUP_TARGET_BYTES)
                high = middle;
              else low = middle + 1;
            }
            const count = Math.max(1, low);
            try {
              const result = await commitAutomaticCleanup(
                key,
                ids.slice(0, count),
                quotaFailure
              );
              if (result.saved) return true;
              if (!result.blocked) return false;
              blocked.add(result.blocked);
              minimumCount = quotaFailure ? 1 : 0;
            } catch (error) {
              if (
                !(error instanceof AgentStorageFullError) ||
                count >= ids.length
              )
                return false;
              quotaFailure = error.reason === "quota";
              minimumCount = count + 1;
            }
          }
        } finally {
          omitQuerySamplesForScope.delete(key);
        }
      })
      .catch(() => false)
      .finally(() => automaticCleanups.delete(key));
    automaticCleanups.set(key, cleanup);
    return cleanup;
  };

  const freeAgentStorage = async (
    options: { resetCurrentChat?: boolean } = {}
  ): Promise<boolean> => {
    const initial = store.getState();
    const key = currentScopeKey(initial);
    const chatId = initial.currentChatId;
    const revision = selectionRevision;
    const storage = getSafeStorage();
    if (reclaimingStorage || !key || !storage || !supportsAgentChatProtection())
      return false;
    const canReset = (state: AgentState, id: string) => {
      const chat = state.chats.find((chat) => chat.id === id);
      return !!chat && !protectsAgentChatFromDeletion(state, chat);
    };
    if (options.resetCurrentChat && chatId && !canReset(initial, chatId))
      return false;
    const isCurrent = () => {
      const state = store.getState();
      return (
        selectionRevision === revision &&
        currentScopeKey(state) === key &&
        state.currentChatId === chatId
      );
    };
    const isExecuting = (
      state: AgentState,
      id: string,
      ownsReclamationLease = false
    ) =>
      Boolean(
        state.abortControllersByChatId[id] ||
          state.followUpDrainChatIds[id] ||
          state.queuedUserMessagesByChatId[id] ||
          state.queuedUserMessageInFlightByChatId[id] ||
          (!ownsReclamationLease && hasAgentRunLease(key, id))
      );
    const consent = persistedSnapshot(initial);
    const consentChats = new Map(consent.chats.map((chat) => [chat.id, chat]));
    const unchanged = (state: AgentState, id: string) => {
      const chat = state.chats.find((chat) => chat.id === id);
      const original = consentChats.get(id);
      return Boolean(
        chat &&
          original &&
          samePersistedValue(chat, original) &&
          samePersistedValue(
            state.messagesByChatId[id],
            consent.messagesByChatId[id]
          ) &&
          samePersistedValue(
            state.draftsByChatId[id],
            consent.draftsByChatId[id]
          ) &&
          samePersistedValue(
            state.pendingAskByChatId[id],
            consent.pendingAskByChatId[id]
          ) &&
          samePersistedValue(
            state.queuedUserMessagesByChatId[id],
            consent.queuedUserMessagesByChatId[id]
          ) &&
          samePersistedValue(
            state.followUpQueuesByChatId[id],
            consent.followUpQueuesByChatId?.[id]
          )
      );
    };
    reclaimingStorage = true;
    try {
      const legacyKey = key.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY);
      const legacyValue = storage.getItem(legacyKey);
      await automaticCleanups.get(key);
      await store.getState().flushRunPersistence();
      if (!isCurrent()) return false;
      await reclaimOtherAgentHistory(storage, {
        currentKey: key,
        userEmail: initial.userEmail!,
        mode: "confirmed",
        isCurrent,
      });
      if (!isCurrent()) return false;
      const state = store.getState();
      const ids = state.chats.map((chat) => chat.id);
      if (options.resetCurrentChat && chatId && isExecuting(state, chatId))
        return false;
      if (options.resetCurrentChat && chatId && !unchanged(state, chatId))
        return false;
      if (legacyValue) {
        try {
          const raw = parseAgentStoredState(legacyValue);
          const chats = isRecord(raw) ? (raw.chats ?? raw.threads) : undefined;
          if (Array.isArray(chats))
            for (const chat of chats)
              if (
                isRecord(chat) &&
                typeof chat.id === "string" &&
                !ids.includes(chat.id)
              )
                ids.push(chat.id);
        } catch {
          // Opaque backups still require the scope-level live-lock check.
        }
      }
      for (const id of ids) reclamationChatIds.add(id);
      refreshChatProtection();
      let saved = false;
      await withAgentHistoryReclamationLocks(
        key,
        ids,
        isCurrent,
        (lockedIds, ownsScope) => {
          store.getState().syncState();
          const current = store.getState();
          if (!isCurrent() || storage.getItem(legacyKey) !== legacyValue)
            return 0;
          if (
            options.resetCurrentChat &&
            chatId &&
            (!lockedIds.has(chatId) ||
              isExecuting(current, chatId, true) ||
              !unchanged(current, chatId))
          )
            return 0;
          const removed = new Set(
            current.chats
              .filter(
                (chat) =>
                  chat.id !== chatId &&
                  lockedIds.has(chat.id) &&
                  !isExecuting(current, chat.id, true) &&
                  !current.followUpQueuesByChatId[chat.id]?.messages.length &&
                  unchanged(current, chat.id)
              )
              .map((chat) => chat.id)
          );
          let retained = omitAgentQuerySamples(
            removeAgentHistoryChats(persistedSnapshot(current), removed)
          );
          if (options.resetCurrentChat && chatId) {
            const chat = current.chats.find((item) => item.id === chatId);
            if (!chat || !canReset(current, chatId)) return 0;
            const messages = current.messagesByChatId[chatId] ?? [];
            const root =
              messages.find(
                (message) =>
                  message.role === "user" &&
                  message.id === chat.runCheckpoint?.rootUserMessageId
              ) ?? messages.findLast((message) => message.role === "user");
            const reset: AgentChat = {
              ...chat,
              status: "error",
              runId: null,
              runMode: undefined,
              runCheckpoint: undefined,
              runBudget: undefined,
              modelAttempts: undefined,
              apiAttempts: undefined,
              legacyUsage: undefined,
              lastRunUsage: undefined,
              totalTokensUsed: 0,
              hasUnreportedUsage: false,
              interrupted: false,
              interruptionReason: undefined,
              lastError: null,
            };
            retained = {
              ...retained,
              chats: retained.chats.map((item) =>
                item.id === chatId ? reset : item
              ),
              messagesByChatId: {
                ...retained.messagesByChatId,
                [chatId]: root ? [root] : [],
              },
              pendingAskByChatId: { ...retained.pendingAskByChatId },
              queuedUserMessagesByChatId: {
                ...retained.queuedUserMessagesByChatId,
              },
              followUpQueuesByChatId: { ...retained.followUpQueuesByChatId },
              activeChatIds: retained.activeChatIds?.filter(
                (id) => id !== chatId
              ),
            };
            delete retained.pendingAskByChatId[chatId];
            delete retained.queuedUserMessagesByChatId[chatId];
            delete retained.followUpQueuesByChatId?.[chatId];
          }
          const candidate = {
            ...current,
            ...retained,
            followUpQueuesByChatId: retained.followUpQueuesByChatId ?? {},
            remoteBusyChatIds: Object.fromEntries(
              Object.entries(current.remoteBusyChatIds).filter(
                ([id]) => !lockedIds.has(id)
              )
            ),
          };
          saved = synchronizeState(candidate, true, true, {
            discardLegacy: ownsScope,
            resetChatId: options.resetCurrentChat
              ? (chatId ?? undefined)
              : undefined,
          });
          if (saved) pendingStatesByScope.set(key, store.getState());
          return saved ? 1 : 0;
        },
        { allowPartial: true }
      );
      if (!isCurrent() || (!saved && options.resetCurrentChat)) return false;
      return await store.getState().checkAgentStorageAdmission();
    } catch (error) {
      updateStoragePressure(key, error, "saving");
      return false;
    } finally {
      reclaimingStorage = false;
      reclamationChatIds.clear();
      refreshChatProtection();
    }
  };

  const writeRetainedState = (
    storage: Storage,
    key: string,
    state: PersistedAgentState,
    previous: string | null,
    userEmail: string,
    discardLegacy = false,
    resetChatId?: string
  ) => {
    const legacyKey = key.startsWith(`${AGENT_RUN_STATE_KEY}.`)
      ? key.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY)
      : null;
    const legacyValue = legacyKey ? storage.getItem(legacyKey) : null;
    const legacyBytes =
      legacyKey && legacyValue
        ? 2 * (legacyKey.length + legacyValue.length)
        : 0;
    let legacyState: unknown;
    let legacyParsed = false;
    let sourceDiscardable = false;
    const canRetireLegacySource = (candidate: string) => {
      if (!legacyKey || !legacyValue) return false;
      try {
        if (storage.getItem(legacyKey) !== legacyValue) return false;
        if (!legacyParsed) {
          legacyParsed = true;
          legacyState = parseAgentStoredState(legacyValue, {
            strictCompressionMarkers: true,
          });
          sourceDiscardable = canDiscardStoredAgentHistory(
            legacyState,
            legacyKey
          );
        }
        const saved = parseAgentStoredState(candidate, {
          strictCompressionMarkers: true,
        });
        return (
          sourceDiscardable &&
          preservesLegacyApiDispatchEvidence(legacyState, saved, resetChatId) &&
          (discardLegacy ||
            canRetireLegacyAgentState(legacyState, saved, {
              scopeKey: key,
              userEmail,
            }))
        );
      } catch {
        return false;
      }
    };
    const retireLegacyState = (durable: string | null) => {
      if (!legacyKey || !durable) return false;
      try {
        if (
          storage.getItem(key) !== durable ||
          !canRetireLegacySource(durable) ||
          storage.getItem(legacyKey) !== legacyValue ||
          storage.getItem(key) !== durable
        )
          return false;
        storage.removeItem(legacyKey);
        return storage.getItem(legacyKey) === null;
      } catch {
        return false;
      }
    };
    let retained = omitQuerySamplesForScope.has(key)
      ? omitAgentQuerySamples(state)
      : retainAgentQuerySamples(state);
    const encode = (compress = false) =>
      key.startsWith(`${AGENT_RUN_STATE_KEY}.`)
        ? encodeAgentStoredState(
            serializePersistedState(retained),
            previous,
            compress
          )
        : serializePersistedState(retained);
    let serialized = encode(omitQuerySamplesForScope.has(key));
    let bytes = estimateAgentStorageBytes(storage, { key, value: serialized });
    const oldBytes = estimateAgentStorageBytes(storage);
    if (bytes > AGENT_STORAGE_ADMISSION_BYTES) {
      retained = omitAgentQuerySamples(retained);
      serialized = encode(true);
      bytes = estimateAgentStorageBytes(storage, { key, value: serialized });
    }
    const enforceBudget = () => {
      // Keep the source until the replacement is durable, but budget only one verified copy.
      const budgetBytes =
        legacyBytes > 0 &&
        bytes > AGENT_STORAGE_LIMIT_BYTES &&
        canRetireLegacySource(serialized)
          ? bytes - legacyBytes
          : bytes;
      // Existing large histories must permit incremental, strictly shrinking cleanup.
      if (budgetBytes > AGENT_STORAGE_LIMIT_BYTES && bytes >= oldBytes) {
        throw new AgentStorageFullError("budget", bytes);
      }
    };
    enforceBudget();
    try {
      if (serialized !== previous) storage.setItem(key, serialized);
    } catch (error) {
      if (!isAgentStorageQuotaError(error)) throw error;
      const retiredLegacy = retireLegacyState(previous);
      retained = omitAgentQuerySamples(retained);
      const reduced = encode(true);
      if (!retiredLegacy && reduced.length >= serialized.length) {
        throw new AgentStorageFullError(
          "quota",
          bytes,
          AGENT_STORAGE_LIMIT_BYTES,
          error
        );
      }
      serialized = reduced;
      bytes = estimateAgentStorageBytes(storage, { key, value: serialized });
      enforceBudget();
      try {
        storage.setItem(key, serialized);
      } catch (retryError) {
        if (isAgentStorageQuotaError(retryError))
          throw new AgentStorageFullError(
            "quota",
            bytes,
            AGENT_STORAGE_LIMIT_BYTES,
            retryError
          );
        throw retryError;
      }
    }
    if (discardLegacy || bytes >= AGENT_STORAGE_CLEANUP_TRIGGER_BYTES)
      retireLegacyState(serialized);
    bytes = estimateAgentStorageBytes(storage);
    return { retained, serialized, bytes };
  };

  const synchronizeState = (
    state: AgentState,
    persist: boolean,
    underLock = false,
    reclamation?: { resetChatId?: string; discardLegacy?: boolean }
  ): boolean => {
    const storage = getSafeStorage();
    const workspaceName = state.workspaceName;
    const userEmail = state.userEmail;
    const key =
      workspaceName && userEmail
        ? getAgentStateKey(workspaceName, userEmail)
        : null;
    if (!storage || !key || !userEmail) {
      if (key && persist)
        updateStoragePressure(
          key,
          new Error("Browser storage is unavailable"),
          "saving"
        );
      return false;
    }
    if (persist && supportsAgentRunRecovery() && !underLock) {
      pendingStatesByScope.set(key, state);
      getWriter(key).schedule();
      return true;
    }
    let saved: string | null;
    try {
      if (
        !supportsAgentRunRecovery() &&
        storage.getItem(key.replace(AGENT_STATE_KEY, AGENT_RUN_STATE_KEY))
      )
        return false;
      saved = storage.getItem(key);
      if (saved === null && supportsAgentRunRecovery()) {
        const legacyKey = key.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY);
        // Only publish the first V3 snapshot from the same V2 bytes we hydrated.
        if (
          storage.getItem(legacyKey) !==
          (lastSerializedByScope.get(legacyKey) ?? null)
        )
          throw new Error(
            "Agent history changed before migration. Reload to reconcile the saved history."
          );
      }
    } catch (error) {
      if (persist) updateStoragePressure(key, error, "saving");
      if (underLock) throw error;
      return false;
    }
    let remote = storageSnapshotsByScope.get(key);
    if (!remote || saved !== lastSerializedByScope.get(key)) {
      try {
        remote =
          saved !== null
            ? normalizePersistedState(
                parseAgentStoredState(saved),
                false,
                false,
                key,
                userEmail ?? undefined
              )
            : createEmptyPersistedState();
      } catch (error) {
        if (persist) updateStoragePressure(key, error, "saving");
        if (underLock) throw error;
        return false;
      }
    }
    const local = persistedSnapshot(state);
    const base = snapshotsByScope.get(key) ?? createEmptyPersistedState();
    const baseChats = new Map(base.chats.map((chat) => [chat.id, chat]));
    const remoteChats = new Map(remote.chats.map((chat) => [chat.id, chat]));
    const locallyBusy = new Set(
      state.chats
        .filter((chat) => {
          if (
            state.remoteBusyChatIds[chat.id] ||
            !(
              state.abortControllersByChatId[chat.id] ||
              state.followUpDrainChatIds[chat.id] ||
              chat.status === "running" ||
              chat.status === "awaiting_user" ||
              state.pendingAskByChatId[chat.id] ||
              state.queuedUserMessagesByChatId[chat.id]
            )
          )
            return false;
          if (
            state.abortControllersByChatId[chat.id] ||
            state.followUpDrainChatIds[chat.id]
          )
            return true;
          const previous = baseChats.get(chat.id);
          if (!previous) return true;
          const previousExecution = chatExecution(base, previous);
          if (
            !samePersistedValue(previousExecution, chatExecution(local, chat))
          )
            return true;
          // Restored pending work must not override progress made by its owner.
          const incoming = remoteChats.get(chat.id);
          return Boolean(
            incoming &&
              samePersistedValue(
                previousExecution,
                chatExecution(remote, incoming)
              )
          );
        })
        .map((chat) => chat.id)
    );
    let merged = rebasePersistedChanges(
      base,
      local,
      remote,
      locallyBusy,
      state.remoteBusyChatIds
    );
    if (reclamation?.resetChatId) {
      const id = reclamation.resetChatId;
      const chat = local.chats.find((chat) => chat.id === id);
      if (!chat) return false;
      merged = {
        ...merged,
        chats: merged.chats.map((item) => (item.id === id ? chat : item)),
        messagesByChatId: {
          ...merged.messagesByChatId,
          [id]: local.messagesByChatId[id] ?? [],
        },
        pendingAskByChatId: { ...merged.pendingAskByChatId },
        queuedUserMessagesByChatId: { ...merged.queuedUserMessagesByChatId },
        activeChatIds: merged.activeChatIds?.filter((chatId) => chatId !== id),
      };
      delete merged.pendingAskByChatId[id];
      delete merged.queuedUserMessagesByChatId[id];
    }
    let serialized = saved;
    if (persist) {
      try {
        const written = writeRetainedState(
          storage,
          key,
          merged,
          saved,
          userEmail,
          Boolean(reclamation?.discardLegacy),
          reclamation?.resetChatId
        );
        merged = written.retained;
        serialized = written.serialized;
        if (written.bytes >= AGENT_STORAGE_CLEANUP_TRIGGER_BYTES)
          scheduleAutomaticCleanup(key);
        if (
          state.storagePressure?.phase !== "admission" ||
          written.bytes <= AGENT_STORAGE_ADMISSION_BYTES
        )
          clearStoragePressure(key);
      } catch (error) {
        updateStoragePressure(key, error, "saving");
        if (error instanceof AgentStorageFullError)
          scheduleAutomaticCleanup(key);
        if (underLock) throw error;
        return false;
      }
    }
    if (serialized) {
      lastSerializedByScope.set(key, serialized);
      if (key.startsWith(`${AGENT_RUN_STATE_KEY}.`))
        lastSerializedByScope.delete(
          key.replace(AGENT_RUN_STATE_KEY, AGENT_STATE_KEY)
        );
    }
    storageSnapshotsByScope.set(key, persist ? merged : remote);
    if (currentScopeKey(store.getState()) !== key) {
      snapshotsByScope.set(key, merged);
      return true;
    }

    const ids = new Set(merged.chats.map((chat) => chat.id));
    const currentChatId =
      state.currentChatId && !ids.has(state.currentChatId)
        ? (merged.chats.find((chat) => !chat.archived)?.id ??
          merged.chats[0]?.id ??
          null)
        : state.currentChatId;
    const draftsByChatId = Object.fromEntries(
      Object.entries(state.draftsByChatId).filter(([id]) => ids.has(id))
    );
    const remoteBusyChatIds = Object.fromEntries(
      merged.chats
        .filter(
          (chat) =>
            !locallyBusy.has(chat.id) &&
            (chat.status === "running" ||
              chat.status === "awaiting_user" ||
              merged.activeChatIds?.includes(chat.id) ||
              merged.queuedUserMessagesByChatId[chat.id])
        )
        .map((chat) => [chat.id, true])
    );
    applyingSnapshot = true;
    try {
      store.setState((draft) => {
        if (!samePersistedValue(draft.chats, merged.chats))
          draft.chats = merged.chats;
        if (
          !samePersistedValue(draft.messagesByChatId, merged.messagesByChatId)
        )
          draft.messagesByChatId = merged.messagesByChatId;
        if (
          !samePersistedValue(
            draft.pendingAskByChatId,
            merged.pendingAskByChatId
          )
        )
          draft.pendingAskByChatId = merged.pendingAskByChatId;
        if (
          !samePersistedValue(
            draft.queuedUserMessagesByChatId,
            merged.queuedUserMessagesByChatId
          )
        )
          draft.queuedUserMessagesByChatId = merged.queuedUserMessagesByChatId;
        if (
          !samePersistedValue(
            draft.followUpQueuesByChatId,
            merged.followUpQueuesByChatId ?? {}
          )
        )
          draft.followUpQueuesByChatId = merged.followUpQueuesByChatId ?? {};
        if (!samePersistedValue(draft.draftsByChatId, draftsByChatId))
          draft.draftsByChatId = draftsByChatId;
        if (!samePersistedValue(draft.remoteBusyChatIds, remoteBusyChatIds))
          draft.remoteBusyChatIds = remoteBusyChatIds;
        draft.currentChatId = currentChatId;
      });
      if (!persist && supportsAgentRunRecovery()) {
        snapshotsByScope.set(key, remote);
        // A scheduled write must use the rebased state with the new baseline.
        if (pendingStatesByScope.has(key))
          pendingStatesByScope.set(key, store.getState());
      } else
        snapshotsByScope.set(key, {
          ...persistedSnapshot(store.getState()),
          followUpQueuesByChatId: (persist ? merged : remote)
            .followUpQueuesByChatId,
          draftsByChatId: persist
            ? store.getState().draftsByChatId
            : base.draftsByChatId,
        });
    } finally {
      applyingSnapshot = false;
    }
    refreshChatProtection();
    return true;
  };

  store.subscribe((state, previousState) => {
    if (
      state.workspaceName !== previousState.workspaceName ||
      state.userEmail !== previousState.userEmail
    ) {
      scopeRevision++;
    }
    if (
      state.workspaceName !== previousState.workspaceName ||
      state.userEmail !== previousState.userEmail ||
      state.currentChatId !== previousState.currentChatId
    )
      selectionRevision++;
    if (
      state.workspaceName !== previousState.workspaceName ||
      state.userEmail !== previousState.userEmail ||
      state.currentChatId !== previousState.currentChatId ||
      state.chats !== previousState.chats ||
      state.messagesByChatId !== previousState.messagesByChatId ||
      state.draftsByChatId !== previousState.draftsByChatId ||
      state.pendingAskByChatId !== previousState.pendingAskByChatId ||
      state.abortControllersByChatId !==
        previousState.abortControllersByChatId ||
      state.queuedUserMessagesByChatId !==
        previousState.queuedUserMessagesByChatId ||
      state.followUpQueuesByChatId !== previousState.followUpQueuesByChatId ||
      state.followUpDrainChatIds !== previousState.followUpDrainChatIds ||
      state.queuedUserMessageInFlightByChatId !==
        previousState.queuedUserMessageInFlightByChatId ||
      state.remoteBusyChatIds !== previousState.remoteBusyChatIds ||
      state.sqlFixDraftsById !== previousState.sqlFixDraftsById ||
      state.expressionDraftTargetsByChatId !==
        previousState.expressionDraftTargetsByChatId
    )
      refreshChatProtection();
    if (applyingSnapshot) return;
    const nonDraftStateChanged =
      state.workspaceName !== previousState.workspaceName ||
      state.userEmail !== previousState.userEmail ||
      state.currentChatId !== previousState.currentChatId ||
      state.chats !== previousState.chats ||
      state.messagesByChatId !== previousState.messagesByChatId ||
      state.pendingAskByChatId !== previousState.pendingAskByChatId ||
      state.abortControllersByChatId !==
        previousState.abortControllersByChatId ||
      state.queuedUserMessagesByChatId !==
        previousState.queuedUserMessagesByChatId ||
      state.followUpQueuesByChatId !== previousState.followUpQueuesByChatId ||
      state.followUpDrainChatIds !== previousState.followUpDrainChatIds;
    const draftChanged = state.draftsByChatId !== previousState.draftsByChatId;
    if (!nonDraftStateChanged && !draftChanged) return;
    saveDraftShadow(state);

    clearDraftPersistTimer();
    if (!nonDraftStateChanged && draftChanged) {
      draftPersistTimer = setTimeout(() => {
        draftPersistTimer = null;
        synchronizeState(store.getState(), true);
      }, DRAFT_PERSIST_DELAY_MS);
    } else {
      synchronizeState(state, true);
    }
  });

  refreshChatProtection();

  return store;
};

// Singleton for production use.
// The return value of create() IS a React hook — callable as
// useAgentStore(selector) to subscribe React components to state slices.
// Also callable outside React via useAgentStore.getState().
export const useAgentStore = createAgentStore();

if (typeof window !== "undefined") {
  window.addEventListener("storage", (event) => {
    const { workspaceName, userEmail } = useAgentStore.getState();
    if (
      !workspaceName ||
      !userEmail ||
      (event.storageArea && event.storageArea !== getSafeStorage())
    )
      return;
    if (
      event.key === null ||
      event.key === getAgentStateKey(workspaceName, userEmail)
    ) {
      useAgentStore.getState().syncState();
    }
    if (
      event.key === null ||
      event.key === getAgentPreferencesKey(workspaceName, userEmail)
    ) {
      useAgentStore.getState().syncPreferences();
    }
  });
  window.addEventListener("focus", () => {
    useAgentStore.getState().refreshChatProtection();
    useAgentStore.getState().syncState();
  });
  window.addEventListener("pageshow", () =>
    useAgentStore.getState().refreshChatProtection()
  );
  window.addEventListener("pagehide", () => {
    useAgentStore.getState().syncState();
    const state = useAgentStore.getState();
    for (const chat of state.chats) {
      if (state.remoteBusyChatIds[chat.id]) continue;
      const controller = state.abortControllersByChatId[chat.id];
      const queued = state.queuedUserMessagesByChatId[chat.id];
      if (state.followUpQueuesByChatId[chat.id])
        state.setFollowUpQueuePaused(chat.id, true);
      if (state.followUpDrainChatIds[chat.id])
        state.setFollowUpDrainActive(chat.id, false);
      controller?.abort();
      if (
        queued?.fromFollowUpQueue &&
        !state.messagesByChatId[chat.id]?.some(({ id }) => id === queued.id)
      ) {
        useAgentStore.setState((draft) => {
          restorePromotedFollowUp(draft, queued);
          delete draft.queuedUserMessageInFlightByChatId[chat.id];
        });
        continue;
      }
      // Keep the question for an explicit retry, never replay an approval on exit.
      if (queued) state.commitQueuedUserMessage(chat.id);
      if (
        chat.status === "running" ||
        chat.status === "awaiting_user" ||
        queued
      )
        state.interruptChatRun(chat.id);
      if (queued) state.finishQueuedUserMessage(chat.id);
      if (controller) state.setAbortController(chat.id, null);
    }
    useAgentStore.getState().saveState();
    useAgentStore.getState().releaseChatProtection();
  });
}
