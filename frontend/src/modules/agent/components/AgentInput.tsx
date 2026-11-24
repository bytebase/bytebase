import { AlertTriangle, ChevronRight, Send } from "lucide-react";
import {
  type RefObject,
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { router } from "@/app/router";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleTrigger } from "@/components/ui/collapsible";
import { FormField, FormLabel } from "@/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { projectNamePrefix } from "@/lib/resourceName";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import { useAppStore } from "@/stores/app";
import {
  getNameParentTokens,
  issueNamePrefix,
  planNamePrefix,
} from "@/stores/modules/v1/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { runAgentLoop } from "../logic/agentLoop";
import {
  getAgentAIConfigurationEnabled,
  isAgentAIConfigurationError,
} from "../logic/aiConfiguration";
import { priorUncertainApiAttemptIds } from "../logic/apiAttempt";
import { getAgentApiRecoveryGuidance } from "../logic/apiResponseGuidance";
import { parseCallApiResult } from "../logic/callApiResult";
import {
  type AgentMaskingPageResource,
  getCurrentAgentPageResource,
  getCurrentChatContext,
  isAgentMaskingPageResource,
} from "../logic/chatContext";
import { resolveDeniedQueryAccessDraft } from "../logic/deniedQueryAccess";
import { normalizeAgentDiagnosticCapability } from "../logic/diagnosticCapability";
import { captureAgentDomContext } from "../logic/domContext";
import { acceptAgentHandoff, collectAgentHandoffs } from "../logic/handoff";
import type { IssueDecisionImpact } from "../logic/issueDecisionImpact";
import {
  parsePreparedAccessGrantDraft,
  withDefaultAccessGrantReason,
  withPreparedAccessGrantReason,
} from "../logic/justInTime";
import {
  getLandingStarterIdForText,
  type LandingStarterId,
} from "../logic/landingStarters";
import { getMaskingResumeArgs } from "../logic/maskingResume";
import {
  getMaskingSectionsForPage,
  getMaskingSectionsForTurn,
} from "../logic/maskingSkillContext";
import {
  buildOutboundHistory,
  getHistoricalMaskingDatabase,
} from "../logic/outboundHistory";
import { buildSystemPrompt } from "../logic/prompt";
import {
  getCurrentTurnQueryCalls,
  getQueryMaskingSetupTarget,
  getQueryResponseActions,
} from "../logic/queryFollowUps";
import { resolveQueryMaskingEligibility } from "../logic/queryMaskingEligibility";
import {
  canAddResourceMention,
  formatResourceMention,
  getCurrentTurnResourceReferences,
  getDatabaseMentionLabel,
  getDraftResourceMentions,
  getMentionTriggerRange,
  getPlanMentionLabel,
  hasUnboundResourceMention,
  insertResourceMention,
  normalizeAgentResourceMentions,
  normalizeAgentResourceReference,
  normalizeAgentResourceReferences,
  removeMentionTrigger,
  updateDraftResourceMentions,
} from "../logic/resourceMention";
import { acceptAgentResponseActions } from "../logic/responseActions";
import {
  getAgentRunContinuation,
  getAgentRunStartup,
} from "../logic/runCheckpoint";
import { getAgentRunFeedback } from "../logic/runFeedback";
import {
  type AgentRunLease,
  hasAgentRunLease,
  isAgentRunLeasePending,
  subscribeAgentRunLeases,
  supportsAgentRunRecovery,
  withAgentRunLease,
} from "../logic/runOwnership";
import { isAgentSendShortcut } from "../logic/sendShortcut";
import {
  getMaskingSkillGuidance,
  type MaskingSkillSection,
} from "../logic/skills";
import { createToolExecutor, getToolDefinitions } from "../logic/tools";
import { apiCallNeedsConfirmation } from "../logic/tools/apiMethodClass";
import { extractDatabaseOverviewEvidence } from "../logic/tools/databaseOverview";
import { resolveInstanceNameFromPagePath } from "../logic/tools/instanceOverview";
import { extractIssueOverviewEvidence } from "../logic/tools/issueOverview";
import {
  getPendingApiCallPreview,
  resolvePendingApiCall,
} from "../logic/tools/pendingApiCall";
import { extractPlanOverviewEvidence } from "../logic/tools/planOverview";
import { createProjectOpenIssuesTool } from "../logic/tools/projectOpenIssues";
import { extractProjectOverviewEvidence } from "../logic/tools/projectOverview";
import { getRecentActivity } from "../logic/tools/recentActivity";
import { getEndpointPath } from "../logic/tools/searchApi";
import { createWorkspaceAttentionTool } from "../logic/tools/workspaceAttention";
import { extractWorkspaceOverviewEvidence } from "../logic/tools/workspaceOverview";
import type {
  AgentAnswerEvidence,
  AgentDraft,
  AgentEvidence,
  AgentHandoff,
  AgentMessage,
  AgentModelCallUsage,
  AgentQueryMaskingEligibility,
  AgentResourceReference,
  AgentRunBindings,
  AgentRunCheckpoint,
  AgentRunMode,
  Message,
  ToolExecutionResult,
} from "../logic/types";
import {
  type AgentState,
  getAgentStateKey,
  selectCurrentChat,
  selectCurrentChatRequiresAIConfiguration,
  selectCurrentPendingAsk,
  selectLoading,
  selectMessages,
  useAgentStore,
} from "../store/agent";
import { AgentActionGroup } from "./AgentActionGroup";
import { AgentChatbox, AgentChatboxFooter } from "./AgentChatbox";
import { AgentErrorDetails, AgentErrorDetailsPanel } from "./AgentErrorDetails";
import { AgentFollowUpQueueList } from "./AgentFollowUpQueueList";
import { AgentResourcePicker } from "./AgentResourcePicker";
import { AgentTokenBudgetStatus } from "./AgentTokenBudgetStatus";

// Module-level map — survives across renders.
const runTokens = new Map<string, number>();
const drainingFollowUps = new Set<string>();

interface ChatRunResult {
  accepted: boolean;
  succeeded: boolean;
}

interface FrozenPageContext {
  maskingPage?: AgentMaskingPageResource;
}

// Pending usage belongs to the live controller, even when the composer unmounts.
const pendingProviderAttempts = new WeakMap<AbortController, string>();
const pendingAttemptListeners = new Set<() => void>();
const subscribePendingAttempts = (listener: () => void) => {
  pendingAttemptListeners.add(listener);
  return () => pendingAttemptListeners.delete(listener);
};
const setPendingProviderAttempt = (
  controller: AbortController,
  attemptId?: string
) => {
  if (attemptId) pendingProviderAttempts.set(controller, attemptId);
  else pendingProviderAttempts.delete(controller);
  for (const listener of pendingAttemptListeners) listener();
};

const withChatRunSegment = async (
  chatId: string,
  action: (lease?: AgentRunLease) => Promise<void>,
  lease?: AgentRunLease
) => {
  const store = useAgentStore.getState();
  if (!store.canUseAgentExecution()) return;
  if (!supportsAgentRunRecovery()) return action();
  if (lease) return lease.run(() => action(lease));
  if (!store.workspaceName || !store.userEmail) return;
  const scopeKey = getAgentStateKey(store.workspaceName, store.userEmail);
  if (!scopeKey) return;
  await withAgentRunLease(scopeKey, chatId, async (owner) => {
    if (!isCurrentChatScope(store, chatId)) return;
    await store.recoverLegacyApiAttempts(chatId, owner);
    await action(owner);
  });
};
const maskingHandoffKinds = new Set<AgentHandoff["kind"]>([
  "open-global-masking",
  "open-semantic-types",
  "open-data-classification",
  "open-project-masking-exemptions",
  "open-database-catalog",
]);

const isCurrentChatScope = (
  scope: Pick<AgentState, "workspaceName" | "userEmail">,
  chatId: string
) => {
  const current = useAgentStore.getState();
  return (
    current.workspaceName === scope.workspaceName &&
    current.userEmail === scope.userEmail &&
    current.getChat(chatId) !== null
  );
};

const permissionDomContext = (
  scope: Pick<AgentState, "workspaceName" | "userEmail">,
  chatId: string,
  runId: string | null | undefined,
  actionId: string,
  controller?: AbortController
) => {
  const chat = useAgentStore.getState().getChat(chatId);
  if (!runId || !chat) return;
  const runMode = chat.runMode ?? "default";
  const project = chat.runCheckpoint?.bindings.context ?? chat.context;
  const scopeKey =
    scope.workspaceName &&
    scope.userEmail &&
    getAgentStateKey(scope.workspaceName, scope.userEmail);
  return captureAgentDomContext(router, {
    chatId,
    runId,
    runMode,
    projectName: project?.startsWith("projects/") ? project : undefined,
    signal: controller?.signal,
    isCurrent: () => {
      const store = useAgentStore.getState(),
        live = store.getChat(chatId);
      if (
        !isCurrentChatScope(scope, chatId) ||
        store.currentChatId !== chatId ||
        !live ||
        live.archived ||
        live.runId !== runId ||
        live.capability !== undefined ||
        (live.runMode ?? "default") !== runMode ||
        (live.runCheckpoint?.bindings.context ?? live.context) !== project
      )
        return false;
      if (!controller)
        return (
          live.status === "awaiting_user" &&
          store.getPendingAsk(chatId)?.apiPermission?.actionId === actionId
        );
      return (
        store.canUseAgentExecution() &&
        live.status === "running" &&
        store.getAbortController(chatId) === controller &&
        (!supportsAgentRunRecovery() ||
          (!!scopeKey && hasAgentRunLease(scopeKey, chatId)))
      );
    },
  });
};

const getRunUpdateError = (fallback: string, error?: unknown): string => {
  const failure = useAgentStore.getState().getRunPersistenceError() ?? error;
  const details =
    failure instanceof Error || failure instanceof DOMException
      ? `${failure.name}: ${failure.message}`
      : typeof failure === "string"
        ? failure
        : undefined;
  return details ? `${fallback}\n${details}` : fallback;
};

const formatPermissionTime = (expiresAt: number): string => {
  const seconds = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
  return `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, "0")}`;
};

const hasExpiredApiPermission = (expiresAt: number | undefined): boolean =>
  typeof expiresAt === "number" &&
  Number.isFinite(expiresAt) &&
  expiresAt > 0 &&
  expiresAt <= Date.now();

const isSuccessfulDatabaseSchemaRead = (
  args: Record<string, unknown>,
  result: ToolExecutionResult,
  databaseName: string
): boolean => {
  if (
    typeof args.operationId !== "string" ||
    getEndpointPath(args.operationId) !==
      "/bytebase.v1.DatabaseService/GetDatabaseSchema" ||
    typeof args.body !== "object" ||
    args.body === null ||
    Array.isArray(args.body) ||
    (args.body as Record<string, unknown>).name !== `${databaseName}/schema` ||
    result.kind !== "tool_result"
  ) {
    return false;
  }
  try {
    const response = JSON.parse(result.result) as {
      status?: unknown;
      error?: unknown;
      response?: unknown;
    };
    return (
      typeof response.status === "number" &&
      response.status >= 200 &&
      response.status < 300 &&
      !response.error &&
      typeof response.response === "object" &&
      response.response !== null &&
      !Array.isArray(response.response)
    );
  } catch {
    return false;
  }
};

const invalidatesPlanDraftEvidence = (
  name: string,
  args: Record<string, unknown>
): boolean =>
  (name === "call_api" &&
    (typeof args.operationId !== "string" ||
      apiCallNeedsConfirmation(args.operationId))) ||
  (name === "dom_action" && args.type !== "read" && args.type !== "scroll");

const getVerifiedDatabaseOverview = (
  result: string,
  database: string | undefined,
  project: string | undefined
): { name: string; project: string } | undefined => {
  if (!database || !project) return undefined;
  try {
    const overview = JSON.parse(result);
    if (
      overview?.project === project &&
      overview.database?.status === "available" &&
      overview.database.name === database &&
      overview.database.state === "ACTIVE"
    )
      return { name: database, project };
  } catch {
    // Only canonical successful overviews establish database verification.
  }
  return undefined;
};

const hasCurrentTurnDatabaseSchemaRead = (
  messages: AgentMessage[],
  databaseName: string
): boolean => {
  const lastUserIndex = messages.findLastIndex(
    (message) => message.role === "user"
  );
  if (lastUserIndex < 0) return false;

  let mutation = 0;
  let schemaRead = false;
  const calls = new Map<
    string,
    { name: string; args: Record<string, unknown>; mutation: number }
  >();
  for (const message of messages.slice(lastUserIndex + 1)) {
    if (message.role === "assistant") {
      for (const call of message.toolCalls ?? []) {
        let args: Record<string, unknown> = {};
        try {
          const parsed: unknown = JSON.parse(call.arguments);
          if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
            args = parsed as Record<string, unknown>;
          }
        } catch {
          // Malformed calls cannot establish schema-read evidence.
        }
        if (invalidatesPlanDraftEvidence(call.name, args)) {
          mutation++;
          schemaRead = false;
        }
        calls.set(call.id, { name: call.name, args, mutation });
      }
    } else if (message.role === "tool" && message.toolCallId) {
      const call = calls.get(message.toolCallId);
      if (
        call?.name === "call_api" &&
        call.mutation === mutation &&
        isSuccessfulDatabaseSchemaRead(
          call.args,
          { kind: "tool_result", result: message.content ?? "" },
          databaseName
        )
      ) {
        schemaRead = true;
      }
    }
  }
  return schemaRead;
};

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

const getCurrentPageSnapshot = () => ({
  path: router.currentRoute.value.fullPath,
  title: document.title,
});

interface RecentActivityEvent {
  actor: string;
  action: string;
  resource: string;
  occurredAt?: string;
}

interface RecentActivityResult {
  events: RecentActivityEvent[];
  partial: boolean;
  error?: string;
}

interface AttentionResult {
  issues: {
    name: string;
    title: string;
    relation: "review" | "created";
  }[];
  partial: boolean;
  error?: string;
}

const parseRecentActivityResult = (value: string): RecentActivityResult => {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.error === "string") {
      return { events: [], partial: false, error: parsed.error };
    }
    const events = Array.isArray(parsed.events)
      ? parsed.events.flatMap((event) => {
          if (
            typeof event !== "object" ||
            event === null ||
            typeof event.actor !== "string" ||
            typeof event.action !== "string" ||
            typeof event.resource !== "string"
          ) {
            return [];
          }
          return [
            {
              actor: event.actor,
              action: event.action,
              resource: event.resource,
              ...(typeof event.occurredAt === "string"
                ? { occurredAt: event.occurredAt }
                : {}),
            },
          ];
        })
      : [];
    return { events, partial: parsed.partial === true };
  } catch {
    return { events: [], partial: false, error: "invalid result" };
  }
};

const parseAttentionResult = (value: string): AttentionResult => {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (typeof parsed.error === "string") {
      return { issues: [], partial: false, error: parsed.error };
    }
    const issues = Array.isArray(parsed.issues)
      ? parsed.issues.flatMap((issue) => {
          if (
            typeof issue !== "object" ||
            issue === null ||
            typeof issue.name !== "string" ||
            typeof issue.title !== "string"
          ) {
            return [];
          }
          return [
            {
              name: issue.name,
              title: issue.title,
              relation:
                issue.relation === "created"
                  ? ("created" as const)
                  : ("review" as const),
            },
          ];
        })
      : [];
    return { issues, partial: parsed.partial === true };
  } catch {
    return { issues: [], partial: false, error: "invalid result" };
  }
};

const mergeOverviewEvidence = (
  current: AgentAnswerEvidence | undefined,
  next: AgentEvidence | undefined
): AgentAnswerEvidence | undefined => {
  if (!next || next.references.length === 0) return current;
  if (!current) return next;

  const groups = Array.isArray(current) ? current : [current];
  const matchingIndex = groups.findIndex(
    (group) =>
      group.project === next.project && group.workspace === next.workspace
  );
  if (matchingIndex < 0) return [...groups, next];

  const existing = groups[matchingIndex];
  const seen = new Set(
    existing.references.map(
      (reference) => `${reference.kind}:${reference.name}`
    )
  );
  const references = [...existing.references];
  for (const reference of next.references) {
    const key = `${reference.kind}:${reference.name}`;
    if (seen.has(key)) continue;
    seen.add(key);
    references.push(reference);
  }
  const merged = { ...existing, references };
  if (!Array.isArray(current)) return merged;
  return groups.map((group, index) =>
    index === matchingIndex ? merged : group
  );
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function AgentInput({
  retryActionRef,
  resendActionRef,
  focusActionRef,
  freeSpaceAndRetryActionRef,
}: {
  retryActionRef?: RefObject<(() => void) | null>;
  resendActionRef?: RefObject<((messageId: string) => void) | null>;
  focusActionRef?: RefObject<((atEnd?: boolean) => void) | null>;
  freeSpaceAndRetryActionRef?: RefObject<(() => Promise<boolean>) | null>;
}) {
  const { t, i18n } = useTranslation();
  // Zustand selectors
  const currentChat = useAgentStore(selectCurrentChat);
  const currentMessages = useAgentStore(selectMessages);
  const currentPendingAsk = useAgentStore(selectCurrentPendingAsk);
  const loading = useAgentStore(selectLoading);
  const executionAvailable = useAgentStore((state) =>
    state.canUseAgentExecution()
  );
  const remoteBusy = useAgentStore((state) =>
    Boolean(state.currentChatId && state.remoteBusyChatIds[state.currentChatId])
  );
  const isAIConfigurationBlocked = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const landingContext = useAgentStore((s) => s.landingContext);
  const workspaceName = useAgentStore((s) => s.workspaceName);
  const userEmail = useAgentStore((s) => s.userEmail);
  const localLeasePending = useSyncExternalStore(
    subscribeAgentRunLeases,
    () => {
      const key =
        workspaceName && userEmail
          ? getAgentStateKey(workspaceName, userEmail)
          : null;
      return Boolean(
        key && currentChatId && isAgentRunLeasePending(key, currentChatId)
      );
    },
    () => false
  );
  const storagePressure = useAgentStore((s) => s.storagePressure);
  const sendWithEnter = useAgentStore((s) => s.preferences.sendWithEnter);
  const currentDraft = useAgentStore((s) =>
    currentChatId ? s.draftsByChatId[currentChatId] : s.landingDraft
  );
  const queuedUserMessagesByChatId = useAgentStore(
    (s) => s.queuedUserMessagesByChatId
  );
  const followUpQueue = useAgentStore((state) =>
    state.currentChatId
      ? state.followUpQueuesByChatId[state.currentChatId]
      : undefined
  );
  const getOrFetchSettingByName = useAppStore((s) => s.getOrFetchSettingByName);
  const currentUserName = useAppStore((s) => s.currentUser?.name ?? "");
  const context = currentChat
    ? currentChat.context
    : (landingContext ?? getCurrentChatContext() ?? workspaceName ?? undefined);
  const projectContext = context?.startsWith(projectNamePrefix)
    ? context
    : undefined;
  const currentTurnProject = useAgentStore((s) =>
    currentChatId
      ? getCurrentTurnResourceReferences(
          s.messagesByChatId[currentChatId] ?? []
        ).at(-1)?.project
      : undefined
  );
  const askProject =
    currentPendingAsk?.resourceProject ?? currentTurnProject ?? projectContext;
  const contextProject = useAppStore((s) =>
    projectContext ? s.projectsByName[projectContext] : undefined
  );
  const workspace = useAppStore((s) => s.workspace);
  const getOrFetchProjectByName = useAppStore((s) => s.getOrFetchProjectByName);
  const aiEnabled = useAppStore((s) => s.serverInfo?.aiEnabled);

  const formatRecentActivity = useCallback(
    (result: string) => {
      const payload = parseRecentActivityResult(result);
      if (payload.error) {
        return t("agent.recent-activity.unavailable");
      }
      if (payload.events.length === 0) {
        return t("agent.recent-activity.empty");
      }

      const formatTime = (value: string | undefined) => {
        if (!value) return t("agent.recent-activity.unknown-time");
        const timestamp = Date.parse(value);
        if (Number.isNaN(timestamp)) {
          return t("agent.recent-activity.unknown-time");
        }
        return new Intl.DateTimeFormat(i18n.language, {
          month: "short",
          day: "numeric",
          hour: "numeric",
          minute: "2-digit",
        }).format(timestamp);
      };

      return [
        t("agent.recent-activity.summary"),
        ...payload.events.map(
          (event) =>
            `- ${t("agent.recent-activity.event", {
              actor: event.actor,
              action: event.action,
              resource: event.resource,
              time: formatTime(event.occurredAt),
            })}`
        ),
        ...(payload.partial ? [t("agent.recent-activity.more")] : []),
      ].join("\n");
    },
    [i18n.language, t]
  );

  const formatWorkspaceAttention = useCallback(
    (result: string) => {
      const payload = parseAttentionResult(result);
      if (payload.error) {
        return t("agent.workspace-attention.unavailable");
      }
      if (payload.issues.length === 0 && payload.partial) {
        return t("agent.workspace-attention.partial");
      }
      if (payload.issues.length === 0) {
        return t("agent.workspace-attention.empty");
      }
      return [
        t("agent.workspace-attention.summary"),
        ...payload.issues.map((issue) =>
          t(`agent.workspace-attention.issue-${issue.relation}`, {
            title: issue.title,
          })
        ),
        ...(payload.partial ? [t("agent.workspace-attention.more")] : []),
      ].join("\n");
    },
    [t]
  );

  const formatProjectAttention = useCallback(
    (result: string) => {
      const payload = parseAttentionResult(result);
      if (payload.error) {
        return t("agent.project-attention.unavailable");
      }
      if (payload.issues.length === 0 && payload.partial) {
        return t("agent.project-attention.partial");
      }
      if (payload.issues.length === 0) {
        return t("agent.project-attention.empty");
      }
      return [
        t("agent.project-attention.summary"),
        ...payload.issues.map((issue) =>
          t(`agent.project-attention.issue-${issue.relation}`, {
            title: issue.title,
          })
        ),
        ...(payload.partial ? [t("agent.project-attention.more")] : []),
      ].join("\n");
    },
    [t]
  );

  const [askDraft, setAskDraft] = useState<AgentDraft>({ text: "" });
  const [continuationFailure, setContinuationFailure] = useState<{
    chatId: string;
    runId: string | null;
    endFailed?: boolean;
  } | null>(null);
  const [endingRun, setEndingRun] = useState(false);
  const endingRunRef = useRef(false);
  const [choiceSelection, setChoiceSelection] = useState<{
    chatId: string;
    toolCallId: string;
    value: string;
    inputs: Map<string, AgentDraft>;
  } | null>(null);
  const choiceInputId = useId();
  const choiceValueRef = useRef<string | null>(null);
  const [, setPermissionTick] = useState(0);
  const [resourcePickerOpen, setResourcePickerOpen] = useState(false);
  const currentController = useAgentStore((state) =>
    state.currentChatId
      ? state.abortControllersByChatId[state.currentChatId]
      : undefined
  );
  const pendingAttemptId = useSyncExternalStore(
    subscribePendingAttempts,
    () =>
      currentController
        ? pendingProviderAttempts.get(currentController)
        : undefined,
    () => undefined
  );
  const currentContinuationFailure =
    continuationFailure?.chatId === currentChat?.id &&
    continuationFailure?.runId === (currentChat?.runId ?? null) &&
    !getAgentRunContinuation(currentChat, currentMessages) &&
    !getAgentRunStartup(currentChat, currentMessages)
      ? continuationFailure
      : null;
  const feedback = getAgentRunFeedback({
    executionAvailable,
    storageBlocked: Boolean(storagePressure),
    aiConfigurationRequired: isAIConfigurationBlocked || aiEnabled === false,
    recoverySupported: supportsAgentRunRecovery(),
    remoteBusy,
    continuationUnavailable: Boolean(currentContinuationFailure),
    chat: currentChat,
    messages: currentMessages,
    pendingAttemptId,
  });
  const mentionRangeRef = useRef<{ start: number; end: number } | null>(null);
  const selectedMentionCaretRef = useRef<number | null>(null);
  const previousChatIdRef = useRef<string | null>(null);
  const confirmationInFlightRef = useRef(false);
  const messageActionInFlightRef = useRef<symbol | null>(null);
  const storageRecoveryInFlightRef = useRef<symbol | null>(null);
  useEffect(() => {
    return () => {
      messageActionInFlightRef.current = null;
      storageRecoveryInFlightRef.current = null;
    };
  }, [currentChatId, workspaceName, userEmail]);

  const composerRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const handledAskFocusRef = useRef<string | null>(null);

  useEffect(() => {
    if (!focusActionRef) return;
    const focus = (atEnd = false) => {
      const textarea = textareaRef.current;
      if (!textarea) return;
      textarea.focus();
      if (atEnd) {
        textarea.setSelectionRange(
          textarea.value.length,
          textarea.value.length
        );
      }
    };
    focusActionRef.current = focus;
    return () => {
      if (focusActionRef.current === focus) focusActionRef.current = null;
    };
  }, [focusActionRef]);

  // Derived values
  const isCurrentChatRunning = loading;
  const chooseOptions =
    currentPendingAsk?.kind === "choose"
      ? (currentPendingAsk.options ?? [])
      : [];
  const isAwaitingConfirm = currentPendingAsk?.kind === "confirm";
  const apiPermission = currentPendingAsk?.apiPermission;
  const canCancelPendingRequest =
    currentChat?.status === "awaiting_user" &&
    Boolean(currentPendingAsk) &&
    !apiPermission &&
    !remoteBusy;
  const apiPermissionPreview =
    apiPermission && currentChat && currentUserName
      ? getPendingApiCallPreview({
          actionId: apiPermission.actionId,
          chatId: currentChat.id,
          toolCallId: currentPendingAsk.toolCallId,
          userName: currentUserName,
          domContext: permissionDomContext(
            { workspaceName, userEmail },
            currentChat.id,
            currentChat.runId,
            apiPermission.actionId
          ),
        })
      : undefined;
  const canRestorePendingAsk = Boolean(
    apiPermissionPreview ||
      (followUpQueue?.messages.length &&
        (!apiPermission || hasExpiredApiPermission(apiPermission.expiresAt)))
  );
  useEffect(() => {
    if (
      !executionAvailable ||
      !remoteBusy ||
      !currentChatId ||
      !currentChat ||
      currentController ||
      (currentChat.status === "awaiting_user" && !canRestorePendingAsk)
    )
      return;
    const chatId = currentChatId;
    const runId = currentChat.runId ?? null;
    const scope = { workspaceName, userEmail };
    let cancelled = false;
    let recovering = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const restore = async () => {
      if (cancelled || recovering) return;
      recovering = true;
      try {
        await withChatRunSegment(chatId, async () => {
          if (cancelled || !isCurrentChatScope(scope, chatId)) return;
          useAgentStore.getState().syncState();
          const store = useAgentStore.getState();
          const chat = store.getChat(chatId);
          if (
            store.currentChatId !== chatId ||
            !chat ||
            (chat.runId ?? null) !== runId ||
            !store.remoteBusyChatIds[chatId] ||
            store.getAbortController(chatId)
          )
            return;
          if (chat.status !== "awaiting_user") {
            await store.recoverChatRun(chatId, runId);
          } else if (chat.status === "awaiting_user") {
            const ask = store.getPendingAsk(chatId);
            if (
              ask &&
              (!ask.apiPermission ||
                hasExpiredApiPermission(ask.apiPermission.expiresAt) ||
                getPendingApiCallPreview({
                  actionId: ask.apiPermission.actionId,
                  chatId,
                  toolCallId: ask.toolCallId,
                  userName: currentUserName,
                  domContext: permissionDomContext(
                    scope,
                    chatId,
                    runId,
                    ask.apiPermission.actionId
                  ),
                }))
            ) {
              if (runId) await store.recoverPendingAsk(chatId, runId);
            }
          }
        });
      } catch {
        // Lease and persistence failures remain passive until a later attempt.
      } finally {
        recovering = false;
        if (
          !cancelled &&
          isCurrentChatScope(scope, chatId) &&
          useAgentStore.getState().remoteBusyChatIds[chatId]
        ) {
          timer = setTimeout(() => void restore(), 500);
        }
      }
    };
    const onFocus = () => {
      clearTimeout(timer);
      void restore();
    };
    void restore();
    window.addEventListener("focus", onFocus);
    return () => {
      cancelled = true;
      clearTimeout(timer);
      window.removeEventListener("focus", onFocus);
    };
  }, [
    apiPermissionPreview?.expiresAt,
    canRestorePendingAsk,
    currentChat?.runId,
    currentChat?.status,
    currentChatId,
    currentController,
    currentUserName,
    executionAvailable,
    followUpQueue?.messages.length,
    remoteBusy,
    userEmail,
    workspaceName,
  ]);
  useEffect(() => {
    if (apiPermissionPreview) {
      const timer = window.setInterval(
        () => setPermissionTick((tick) => tick + 1),
        1000
      );
      return () => window.clearInterval(timer);
    }
    const expiresAt = apiPermission?.expiresAt;
    if (
      !remoteBusy ||
      !followUpQueue?.messages.length ||
      !expiresAt ||
      hasExpiredApiPermission(expiresAt)
    )
      return;
    const timer = window.setTimeout(
      () => setPermissionTick((tick) => tick + 1),
      Math.min(2_147_483_647, Math.max(0, expiresAt - Date.now() + 1))
    );
    return () => window.clearTimeout(timer);
  }, [
    apiPermissionPreview?.expiresAt,
    apiPermission?.expiresAt,
    followUpQueue?.messages.length,
    remoteBusy,
  ]);
  const isAwaitingChoose =
    currentPendingAsk?.kind === "choose" && chooseOptions.length > 0;
  const selectedChoice =
    choiceSelection?.chatId === currentChatId &&
    choiceSelection?.toolCallId === currentPendingAsk?.toolCallId
      ? chooseOptions.find((option) => option.value === choiceSelection.value)
      : undefined;
  const isCustomChoice = selectedChoice?.kind === "custom-input";
  const composerDraft = isCustomChoice
    ? choiceSelection?.inputs.get(selectedChoice.value)
    : currentPendingAsk
      ? askDraft
      : currentDraft;
  const input = composerDraft?.text ?? "";
  const needsChoiceInput = isCustomChoice && !input.trim();
  useLayoutEffect(() => {
    if (selectedChoice?.kind === "custom-input") {
      textareaRef.current?.focus();
    }
  }, [selectedChoice?.kind, selectedChoice?.value]);
  const updateComposerDraft = useCallback(
    (update: (draft: AgentDraft) => AgentDraft) => {
      if (isCustomChoice) {
        setChoiceSelection((previous) => {
          if (
            previous?.chatId !== currentChatId ||
            previous.toolCallId !== currentPendingAsk?.toolCallId ||
            previous.value !== selectedChoice.value
          )
            return previous;
          return {
            ...previous,
            inputs: new Map(previous.inputs).set(
              previous.value,
              update(previous.inputs.get(previous.value) ?? { text: "" })
            ),
          };
        });
        return;
      }
      if (currentPendingAsk) {
        setAskDraft(update);
        return;
      }
      const store = useAgentStore.getState();
      if (store.currentChatId) {
        store.setDraft(
          store.currentChatId,
          update(store.getDraft(store.currentChatId))
        );
      } else {
        store.setLandingDraft(
          update(store.landingDraft),
          store.landingContext ?? getCurrentChatContext() ?? workspace?.name
        );
      }
    },
    [
      currentChatId,
      currentPendingAsk,
      isCustomChoice,
      selectedChoice,
      workspace?.name,
    ]
  );
  const setInput = useCallback(
    (value: string) => {
      updateComposerDraft((draft) => {
        const mentions = updateDraftResourceMentions(
          draft.text,
          value,
          getDraftResourceMentions(draft)
        );
        return { text: value, ...(mentions.length > 0 ? { mentions } : {}) };
      });
    },
    [updateComposerDraft]
  );
  const handleResourcePickerOpenChange = useCallback(
    (nextOpen: boolean) => {
      setResourcePickerOpen(nextOpen);
      if (nextOpen) return;
      const trigger = mentionRangeRef.current;
      mentionRangeRef.current = null;
      const removed = removeMentionTrigger(input, trigger);
      if (!removed) return;
      selectedMentionCaretRef.current = removed.caret;
      setInput(removed.text);
    },
    [input, setInput]
  );
  const isQueueSubmission =
    !currentPendingAsk &&
    (isCurrentChatRunning ||
      Boolean(currentController) ||
      Boolean(followUpQueue?.messages.length));
  const sendLabel = currentPendingAsk
    ? t("agent.reply")
    : isQueueSubmission
      ? t("agent.follow-up-queue.enqueue")
      : t("agent.send");
  const inputPlaceholder = isCustomChoice
    ? (selectedChoice.placeholder ?? t("agent.custom-input-placeholder"))
    : currentPendingAsk?.kind === "input"
      ? currentPendingAsk.prompt
      : currentPendingAsk?.kind === "choose"
        ? currentPendingAsk.prompt
        : isCurrentChatRunning && !remoteBusy
          ? t("agent.follow-up-queue.input-placeholder")
          : t("agent.input-placeholder");
  const confirmLabel =
    currentPendingAsk?.confirmLabel?.trim() || t("agent.confirm");
  const cancelLabel =
    currentPendingAsk?.cancelLabel?.trim() || t("agent.cancel");
  const draftMentions = composerDraft
    ? getDraftResourceMentions(composerDraft)
    : [];
  const selectedProjectMention = draftMentions.find(
    ({ resource }) => resource.kind === "project"
  );
  const hasAdditionalMention =
    draftMentions.length > 0 && hasUnboundResourceMention(input, draftMentions);
  const isSendDisabled =
    !executionAvailable ||
    currentChat?.archived ||
    currentChat?.status === "paused" ||
    currentChat?.runCheckpoint?.suspension === "token-limit" ||
    remoteBusy ||
    (isCurrentChatRunning && Boolean(currentPendingAsk)) ||
    (localLeasePending && Boolean(currentPendingAsk)) ||
    isAwaitingConfirm ||
    (isAwaitingChoose && !isCustomChoice) ||
    isAIConfigurationBlocked ||
    hasAdditionalMention ||
    !input.trim();
  const contextTitle = projectContext
    ? contextProject?.title || projectContext.slice(projectNamePrefix.length)
    : context === workspace?.name
      ? workspace?.title || context
      : context;
  const canMentionResource = Boolean(
    workspace?.name &&
      (!currentPendingAsk ||
        ((currentPendingAsk.kind === "input" || isCustomChoice) &&
          !apiPermission))
  );

  const selectContext = useCallback(
    (resource: string) => {
      const store = useAgentStore.getState();
      if (!store.currentChatId) {
        store.setLandingDraft(store.landingDraft, resource);
        setResourcePickerOpen(false);
        mentionRangeRef.current = null;
        return true;
      }
      const selected = store.selectOrCreateChatForContext(
        resource,
        getCurrentPageSnapshot(),
        workspace?.name
      );
      if (!selected) return false;
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      return true;
    },
    [workspace?.name]
  );

  const selectMentionResource = useCallback(
    (resource: AgentResourceReference) => {
      const store = useAgentStore.getState();
      if (
        currentPendingAsk &&
        (store.currentChatId !== currentChatId ||
          store.getPendingAsk(currentChatId)?.toolCallId !==
            currentPendingAsk.toolCallId ||
          (isCustomChoice &&
            (choiceValueRef.current !== selectedChoice.value ||
              (askProject && resource.project !== askProject))))
      ) {
        return;
      }
      const draft = currentPendingAsk
        ? (composerDraft ?? { text: "" })
        : store.currentChatId
          ? store.getDraft(store.currentChatId)
          : store.landingDraft;
      const mentions = getDraftResourceMentions(draft);
      if (!canAddResourceMention(mentions, resource)) return;
      const range = mentionRangeRef.current;
      const label = resource.label;
      const insertion = insertResourceMention(draft.text, range, label);
      if (!insertion) return;
      selectedMentionCaretRef.current = insertion.caret;
      const nextDraft: AgentDraft = {
        text: insertion.text,
        mentions: [
          ...updateDraftResourceMentions(draft.text, insertion.text, mentions),
          { resource, range: insertion.mention },
        ],
      };
      updateComposerDraft(() => nextDraft);
      mentionRangeRef.current = null;
    },
    [
      askProject,
      composerDraft,
      currentChatId,
      currentPendingAsk,
      isCustomChoice,
      selectedChoice,
      updateComposerDraft,
    ]
  );

  const selectMentionProject = useCallback(
    (project: Project) =>
      selectMentionResource({
        kind: "project",
        name: project.name,
        project: project.name,
        label: project.title || project.name,
      }),
    [selectMentionResource]
  );

  const selectMentionDatabase = useCallback(
    (database: Database) =>
      selectMentionResource({
        kind: "database",
        name: database.name,
        project: database.project,
        label: getDatabaseMentionLabel(database),
      }),
    [selectMentionResource]
  );

  const selectMentionIssue = useCallback(
    (issue: Issue) => {
      const [projectId] = getNameParentTokens(issue.name, [
        projectNamePrefix,
        issueNamePrefix,
      ]);
      if (!projectId) return;
      selectMentionResource({
        kind: "issue",
        name: issue.name,
        project: `${projectNamePrefix}${projectId}`,
        label: issue.title || issue.name,
      });
    },
    [selectMentionResource]
  );

  const selectMentionPlan = useCallback(
    (plan: Plan) => {
      const [projectId, planId] = getNameParentTokens(plan.name, [
        projectNamePrefix,
        planNamePrefix,
      ]);
      if (!projectId || !planId) return;
      selectMentionResource({
        kind: "plan",
        name: plan.name,
        project: `${projectNamePrefix}${projectId}`,
        label: getPlanMentionLabel(plan),
      });
    },
    [selectMentionResource]
  );

  useEffect(() => {
    if (projectContext && !contextProject) {
      void getOrFetchProjectByName(projectContext, true);
    }
  }, [contextProject, getOrFetchProjectByName, projectContext]);

  const blockWhenAIIsNotConfigured = useCallback(
    async (chatId: string) => {
      const scope = useAgentStore.getState();
      const setting = await getOrFetchSettingByName(
        Setting_SettingName.AI,
        true
      );
      if (!isCurrentChatScope(scope, chatId)) return true;
      if (getAgentAIConfigurationEnabled(setting) !== false) return false;

      useAgentStore.getState().setChatStatus(chatId, "error", {
        requiresAIConfiguration: true,
      });
      return true;
    },
    [getOrFetchSettingByName]
  );

  useEffect(() => {
    const chat = useAgentStore.getState().getChat(currentChatId);
    if (!chat) return;

    if (aiEnabled === false) {
      useAgentStore.getState().setChatStatus(chat.id, "error", {
        requiresAIConfiguration: true,
      });
      return;
    }

    if (aiEnabled === true && chat.requiresAIConfiguration) {
      useAgentStore.getState().clearError(chat.id);
    }
  }, [aiEnabled, currentChatId]);

  useEffect(() => {
    const chat = useAgentStore.getState().getChat(currentChatId);
    if (!chat) return;
    void blockWhenAIIsNotConfigured(chat.id);
  }, [blockWhenAIIsNotConfigured, currentChatId]);

  const defaultAskDraft = useMemo<AgentDraft>(() => {
    const text = currentPendingAsk?.defaultValue ?? "";
    const resource = normalizeAgentResourceReference(
      currentPendingAsk?.defaultResource
    );
    if (
      currentPendingAsk?.kind !== "input" ||
      resource?.kind !== "database" ||
      resource.name !== text ||
      (askProject && resource.project !== askProject)
    ) {
      return { text };
    }
    const insertion = insertResourceMention(
      "@",
      { start: 0, end: 1 },
      resource.label
    );
    return insertion
      ? {
          text: insertion.text,
          mentions: [{ resource, range: insertion.mention }],
        }
      : { text };
  }, [
    currentPendingAsk?.defaultValue,
    currentPendingAsk?.defaultResource,
    currentPendingAsk?.kind,
    askProject,
  ]);

  // Pending-ask answers are separate from the saved question draft.
  useEffect(() => {
    setChoiceSelection(null);
    choiceValueRef.current = null;
    setResourcePickerOpen(false);
    mentionRangeRef.current = null;
    selectedMentionCaretRef.current = null;
    if (
      currentPendingAsk?.kind === "input" ||
      currentPendingAsk?.kind === "choose"
    ) {
      setAskDraft(defaultAskDraft);
      return;
    }
    setAskDraft({ text: "" });
  }, [
    currentChatId,
    workspaceName,
    userEmail,
    currentPendingAsk?.toolCallId,
    currentPendingAsk?.kind,
    defaultAskDraft,
  ]);

  useLayoutEffect(() => {
    const askKey =
      currentChatId &&
      currentPendingAsk?.kind === "input" &&
      !currentPendingAsk.apiPermission
        ? `${workspaceName}:${userEmail}:${currentChatId}:${currentPendingAsk.toolCallId}`
        : null;
    if (!askKey) {
      handledAskFocusRef.current = null;
      return;
    }
    if (
      handledAskFocusRef.current === askKey ||
      input !== defaultAskDraft.text
    ) {
      return;
    }
    handledAskFocusRef.current = askKey;
    const store = useAgentStore.getState();
    const textarea = textareaRef.current;
    if (
      !store.visible ||
      store.minimized ||
      store.currentChatId !== currentChatId ||
      currentChat?.status !== "awaiting_user" ||
      !textarea ||
      textarea.disabled ||
      textarea.closest(
        '[hidden], [inert], [aria-hidden="true"], [data-base-ui-inert]'
      )
    ) {
      return;
    }
    textarea.focus();
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
  }, [
    currentChatId,
    currentChat?.status,
    currentPendingAsk,
    defaultAskDraft.text,
    input,
    workspaceName,
    userEmail,
  ]);

  useEffect(() => {
    if (
      previousChatIdRef.current &&
      previousChatIdRef.current !== currentChatId
    ) {
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      selectedMentionCaretRef.current = null;
    }
    previousChatIdRef.current = currentChatId;
  }, [currentChatId]);

  useLayoutEffect(() => {
    const caret = selectedMentionCaretRef.current;
    if (caret === null) return;
    selectedMentionCaretRef.current = null;
    textareaRef.current?.focus();
    textareaRef.current?.setSelectionRange(caret, caret);
  }, [input]);

  // ---------------------------------------------------------------------------
  // Agent loop execution
  // ---------------------------------------------------------------------------

  const buildChatHistory = useCallback(
    (chatId: string, systemPrompt: string): Message[] => {
      const chat = useAgentStore.getState().getChat(chatId);
      return buildOutboundHistory(
        systemPrompt,
        useAgentStore.getState().getMessages(chatId),
        chat?.context,
        chat?.apiAttempts?.length ? chat.apiAttempts : undefined,
        {
          includeActionReceipts: !chat?.capability,
          ownerEmail: useAgentStore.getState().userEmail ?? undefined,
        }
      );
    },
    []
  );

  const canContinueChatRun = useCallback((chatId: string) => {
    if (!supportsAgentRunRecovery()) return true;
    const store = useAgentStore.getState();
    const chat = store.getChat(chatId);
    if (!getAgentRunContinuation(chat, store.getMessages(chatId))) {
      if (chat) setContinuationFailure({ chatId, runId: chat.runId ?? null });
      return false;
    }
    setContinuationFailure(null);
    return true;
  }, []);

  const handleOutcome = useCallback(
    (chatId: string, outcome: Awaited<ReturnType<typeof runAgentLoop>>) => {
      const store = useAgentStore.getState();
      switch (outcome.kind) {
        case "completed":
        case "stopped":
          store.finishChatRun(chatId);
          return;
        case "awaiting_user":
          store.awaitUser(chatId, outcome.ask);
          return;
        case "paused":
          return;
        case "aborted":
          store.interruptChatRun(chatId, getCurrentPageSnapshot());
          return;
        case "error": {
          store.setChatStatus(chatId, "error", {
            lastError: outcome.error.message,
            requiresAIConfiguration: isAgentAIConfigurationError(outcome.error),
          });
          return;
        }
      }
    },
    []
  );

  const runChat = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string,
      runMode: AgentRunMode,
      skipReferencePreflight = false,
      resumeCheckpoint?: AgentRunCheckpoint,
      frozenPageContext?: FrozenPageContext
    ) => {
      const store = useAgentStore.getState();
      if (!store.getChat(chatId)) return;
      const managed = supportsAgentRunRecovery();
      const bindings = managed
        ? store.getChat(chatId)?.runCheckpoint?.bindings
        : undefined;
      page = bindings?.page ?? page;
      runMode = bindings?.runMode ?? runMode;
      const runContext = bindings
        ? bindings.context
        : store.getChat(chatId)?.context;
      const maxTotalTokens = store.preferences.tokenLimitPerRun;
      const rawCapability =
        bindings?.capability ?? store.getChat(chatId)?.capability;
      const capability = normalizeAgentDiagnosticCapability(rawCapability);
      const turnReferences = getCurrentTurnResourceReferences(
        store.getMessages(chatId)
      );
      const scopeMismatch =
        capability &&
        turnReferences.some((reference) =>
          reference.kind === "project"
            ? reference.name !== capability.resource.project
            : reference.kind !== capability.resource.kind ||
              reference.name !== capability.resource.name ||
              reference.project !== capability.resource.project
        );
      if ((rawCapability !== undefined && !capability) || scopeMismatch) {
        store.addMessage({
          chatId,
          role: "assistant",
          content: scopeMismatch
            ? t("agent.diagnostic-help.scope-mismatch")
            : t("agent.contextual-start.resource-unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
        return;
      }
      const references = bindings
        ? bindings.references
        : capability
          ? [capability.resource]
          : turnReferences;
      const projectReference = references.find(
        (reference) => reference.kind === "project"
      );
      const databaseReference = references.find(
        (reference) => reference.kind === "database"
      );
      const issueReference = references.find(
        (reference) => reference.kind === "issue"
      );
      const planReference = references.find(
        (reference) => reference.kind === "plan"
      );
      const referencedProject =
        projectReference?.name ??
        databaseReference?.project ??
        issueReference?.project ??
        planReference?.project;
      const referencedDatabase = databaseReference?.name;
      const referencedIssue = issueReference?.name;
      const referencedPlan = planReference?.name;
      const currentPageResource = capability
        ? undefined
        : frozenPageContext
          ? frozenPageContext.maskingPage
          : getCurrentAgentPageResource();
      const maskingPage = bindings
        ? bindings.maskingPage
        : isAgentMaskingPageResource(currentPageResource)
          ? currentPageResource
          : undefined;
      const instanceName = bindings
        ? bindings.instanceName
        : referencedProject
          ? undefined
          : resolveInstanceNameFromPagePath(page.path);
      const controller = new AbortController();
      let admitted = false;
      let completedSuccessfully = false;
      const recordUsage = (usage: AgentModelCallUsage) => {
        if (isCurrentChatScope(store, chatId)) {
          store.recordChatModelCallUsage(chatId, runId, usage);
        }
      };
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      const ownsInvocation = () =>
        !controller.signal.aborted &&
        runTokens.get(chatId) === runToken &&
        isCurrentChatScope(store, chatId) &&
        useAgentStore.getState().getChat(chatId)?.runId === runId &&
        useAgentStore.getState().getAbortController(chatId) === controller;
      const reportPersistenceFailure = (error?: unknown) => {
        const current = useAgentStore.getState().getChat(chatId);
        if (
          isCurrentChatScope(store, chatId) &&
          runTokens.get(chatId) === runToken &&
          current &&
          (managed
            ? current?.runBudget?.runId === runId
            : current?.runId === runId) &&
          !current.archived
        ) {
          useAgentStore.getState().setChatStatus(chatId, "error", {
            lastError: getRunUpdateError(
              t("agent.token-budget.action-failed"),
              error
            ),
          });
        }
      };
      const admitExecution = async () => {
        const allowed =
          ownsInvocation() &&
          (await useAgentStore.getState().checkAgentStorageAdmission()) &&
          ownsInvocation() &&
          !useAgentStore.getState().getChat(chatId)?.archived;
        if (allowed) admitted = true;
        return allowed;
      };

      const systemPrompt = buildSystemPrompt({
        ...page,
        context: runContext,
        referencedProject,
        referencedDatabase,
        referencedIssue,
        referencedPlan,
        instanceName,
        maskingPage,
        runMode,
        ...(capability ? { capability } : {}),
      });
      const messages = store.getMessages(chatId);
      const maskingResumeArgs = capability
        ? undefined
        : getMaskingResumeArgs(messages);
      const maskingSections = new Set<MaskingSkillSection>(
        capability
          ? []
          : [
              ...(resumeCheckpoint?.maskingSections ?? []),
              ...getMaskingSectionsForPage(maskingPage),
              ...getMaskingSectionsForTurn(messages),
            ]
      );
      const toolOptions = {
        runMode,
        referencedProject,
        instanceName,
        maskingPage,
        ...(capability ? { capability } : {}),
      };
      const resolveRequestContext = () => ({
        tools: getToolDefinitions({
          ...toolOptions,
          maskingSections: [...maskingSections],
        }),
        systemGuidance: [
          capability ? "" : getMaskingSkillGuidance([...maskingSections]),
          getAgentApiRecoveryGuidance(
            useAgentStore.getState().getMessages(chatId),
            { diagnostic: Boolean(capability) }
          ),
        ]
          .filter(Boolean)
          .join("\n\n"),
      });
      const tools = resolveRequestContext().tools;
      let pageToolPath =
        resumeCheckpoint?.pageToolPath ??
        bindings?.pageToolPath ??
        (frozenPageContext ? page.path : undefined);
      const executor = createToolExecutor(router, {
        chatId,
        runId,
        getDomContext: () =>
          captureAgentDomContext(
            router,
            {
              chatId,
              runId,
              runMode,
              signal: controller.signal,
              projectName:
                referencedProject ??
                (runContext?.startsWith("projects/") ? runContext : undefined),
              isCurrent: () => {
                const live = useAgentStore.getState(),
                  chat = live.getChat(chatId);
                return (
                  ownsInvocation() &&
                  live.canUseAgentExecution() &&
                  !!chat &&
                  !chat.archived &&
                  chat.capability === undefined &&
                  live.currentChatId === chatId &&
                  (chat.runMode ?? "default") === runMode &&
                  (chat.runCheckpoint?.bindings.context ?? chat.context) ===
                    runContext &&
                  (pageToolPath === undefined ||
                    router.currentRoute.value.fullPath === pageToolPath)
                );
              },
            },
            true
          ),
        priorAttemptIds: (fingerprint) =>
          priorUncertainApiAttemptIds(
            useAgentStore.getState().chats,
            fingerprint
          ),
        isApiScopeCurrent: store.captureApiScope(),
        signal: controller.signal,
        userName: useAppStore.getState().currentUser?.name,
        context: runContext,
        workspaceName: workspace?.name,
        referencedProject,
        pagePath: page.path,
        referencedDatabase,
        databaseChoiceProject: maskingResumeArgs?.project,
        historicalDatabase: getHistoricalMaskingDatabase(
          store.getMessages(chatId),
          store.getChat(chatId)?.context
        ),
        referencedIssue,
        referencedPlan,
        maskingPage,
        runMode,
        ...(capability ? { capability } : {}),
        onSkillLoaded: (load) => {
          if (!capability && load.name === "data-masking" && load.section) {
            maskingSections.add(load.section);
          }
        },
        canUseCurrentPage: () =>
          isCurrentChatScope(store, chatId) &&
          useAgentStore.getState().currentChatId === chatId &&
          (pageToolPath === undefined ||
            getCurrentPageSnapshot().path === pageToolPath),
        onNavigate: () => {
          if (pageToolPath !== undefined)
            pageToolPath = getCurrentPageSnapshot().path;
          useAgentStore
            .getState()
            .updateChatPage(chatId, getCurrentPageSnapshot());
        },
      });
      const toolNamesByCallId = new Map<string, string>();
      let overviewEvidence: AgentAnswerEvidence | undefined;
      let verifiedDatabase: { name: string; project: string } | undefined;
      let readDatabaseSchema = referencedDatabase
        ? hasCurrentTurnDatabaseSchemaRead(
            store.getMessages(chatId),
            referencedDatabase
          )
        : false;
      const handoffCandidates: AgentHandoff[] = [];
      const maskingDraftIds: string[] = [];
      const exemptionDraftIds: string[] = [];
      const semanticTypeDraftIds: string[] = [];
      let preparedAccessRequestDraft: AccessGrantDraft | undefined;
      const chatContext = store.getChat(chatId)?.context;
      const handoffScope = {
        workspaceName: chatContext?.startsWith("workspaces/")
          ? chatContext
          : workspace?.name,
        projectName:
          referencedProject ??
          (chatContext?.startsWith(projectNamePrefix)
            ? chatContext
            : undefined),
      };
      const observeHandoffs = (name: string, result: string) => {
        if (name === "prepare_access_grant_draft") {
          maskingDraftIds.length = 0;
          exemptionDraftIds.length = 0;
          semanticTypeDraftIds.length = 0;
          preparedAccessRequestDraft = parsePreparedAccessGrantDraft(result);
        }
        if (
          name === "prepare_masking_rule_draft" ||
          name === "prepare_masking_rule_drafts" ||
          name === "prepare_masking_exemption_draft" ||
          name === "prepare_semantic_type_draft"
        ) {
          preparedAccessRequestDraft = undefined;
          maskingDraftIds.length = 0;
          exemptionDraftIds.length = 0;
          semanticTypeDraftIds.length = 0;
          try {
            const prepared = JSON.parse(result);
            if (!prepared.error && typeof prepared.draftId === "string")
              (name === "prepare_semantic_type_draft"
                ? semanticTypeDraftIds
                : name === "prepare_masking_exemption_draft"
                  ? exemptionDraftIds
                  : maskingDraftIds
              ).push(prepared.draftId);
          } catch {
            // Only successfully prepared structured drafts can become actions.
          }
        }
        if (name === "get_masking_overview") {
          for (let index = handoffCandidates.length - 1; index >= 0; index--) {
            if (maskingHandoffKinds.has(handoffCandidates[index].kind)) {
              handoffCandidates.splice(index, 1);
            }
          }
        }
        handoffCandidates.push(
          ...collectAgentHandoffs(name, result, handoffScope)
        );
      };
      if (resumeCheckpoint) {
        const calls = new Map<
          string,
          { name: string; args: Record<string, unknown>; epoch: number }
        >();
        let epoch = 0;
        readDatabaseSchema = false;
        const invalidate = () => {
          epoch++;
          overviewEvidence = undefined;
          verifiedDatabase = undefined;
          readDatabaseSchema = false;
          handoffCandidates.length = 0;
          maskingDraftIds.length = 0;
          exemptionDraftIds.length = 0;
          semanticTypeDraftIds.length = 0;
          preparedAccessRequestDraft = undefined;
        };
        const runMessages = messages.slice(
          messages.findIndex(
            (message) => message.id === resumeCheckpoint.rootUserMessageId
          ) + 1
        );
        for (const message of runMessages) {
          for (const call of message.role === "assistant"
            ? (message.toolCalls ?? [])
            : []) {
            let args: Record<string, unknown> = {};
            try {
              const value: unknown = JSON.parse(call.arguments);
              if (value && typeof value === "object" && !Array.isArray(value))
                args = value as Record<string, unknown>;
            } catch {
              // Malformed calls cannot establish evidence.
            }
            if (invalidatesPlanDraftEvidence(call.name, args)) invalidate();
            calls.set(call.id, { name: call.name, args, epoch });
          }
          if (message.role !== "tool" || !message.toolCallId) continue;
          const call = calls.get(message.toolCallId);
          calls.delete(message.toolCallId);
          if (!call || call.epoch !== epoch) continue;
          const name = call?.name;
          if (!message.content) continue;
          if (
            name === "get_database_overview" &&
            call.args.databaseName === referencedDatabase
          ) {
            verifiedDatabase = getVerifiedDatabaseOverview(
              message.content,
              referencedDatabase,
              referencedProject
            );
            if (!verifiedDatabase) readDatabaseSchema = false;
          } else if (
            name === "call_api" &&
            verifiedDatabase &&
            isSuccessfulDatabaseSchemaRead(
              call.args,
              { kind: "tool_result", result: message.content },
              verifiedDatabase.name
            )
          ) {
            readDatabaseSchema = true;
          }
          if (name) observeHandoffs(name, message.content);
          const extract =
            name === "get_project_overview"
              ? extractProjectOverviewEvidence
              : name === "get_plan_overview"
                ? extractPlanOverviewEvidence
                : name === "get_issue_overview"
                  ? extractIssueOverviewEvidence
                  : name === "get_database_overview"
                    ? extractDatabaseOverviewEvidence
                    : name === "get_workspace_overview"
                      ? extractWorkspaceOverviewEvidence
                      : undefined;
          if (extract)
            overviewEvidence = mergeOverviewEvidence(
              overviewEvidence,
              extract(message.content)
            );
        }
      }

      try {
        const history = buildChatHistory(chatId, systemPrompt);
        if (maskingResumeArgs && !resumeCheckpoint) {
          if (!(await admitExecution())) {
            reportPersistenceFailure();
            return;
          }
          const toolCallId = uuidv4();
          const toolCall = {
            id: toolCallId,
            name: "get_masking_overview",
            arguments: JSON.stringify(maskingResumeArgs),
            origin: "runtime" as const,
          };
          let result: string;
          try {
            const refreshed = await executor(
              toolCall.name,
              { ...maskingResumeArgs },
              toolCallId
            );
            result =
              refreshed.kind === "tool_result"
                ? refreshed.result
                : JSON.stringify({
                    error:
                      "The masking overview could not be refreshed. Current configuration and permissions remain unverified.",
                  });
          } catch {
            result = JSON.stringify({
              error:
                "The masking overview could not be refreshed. Current configuration and permissions remain unverified.",
            });
          }
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !isCurrentChatScope(store, chatId)
          )
            return;
          observeHandoffs(toolCall.name, result);
          useAgentStore.getState().addMessage({
            chatId,
            role: "assistant",
            toolCalls: [toolCall],
            metadata: { route: page.path, runId },
          });
          useAgentStore.getState().addMessage({
            chatId,
            role: "tool",
            toolCallId,
            content: result,
            metadata: { route: page.path, runId },
          });
          history.push(
            { role: "assistant", toolCalls: [toolCall] },
            { role: "tool", toolCallId, content: result }
          );
        }
        const startupStart = history.length;
        let prefetchedProjectOverview: string | undefined;
        if (
          !skipReferencePreflight &&
          projectReference &&
          !databaseReference &&
          !issueReference &&
          !planReference &&
          referencedProject
        ) {
          if (!(await admitExecution())) {
            reportPersistenceFailure();
            return;
          }
          const preflightCallId = uuidv4();
          const result = await executor(
            "get_project_overview",
            {},
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !isCurrentChatScope(store, chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                error?: unknown;
              };
              valid = overview.project === referencedProject && !overview.error;
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.project-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          prefetchedProjectOverview = overviewResult;
          observeHandoffs("get_project_overview", overviewResult);
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_project_overview",
                  arguments: "{}",
                  origin: "runtime",
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        } else if (
          !skipReferencePreflight &&
          referencedPlan &&
          referencedProject
        ) {
          if (!(await admitExecution())) {
            reportPersistenceFailure();
            return;
          }
          const preflightCallId = uuidv4();
          const args = { planName: referencedPlan };
          const result = await executor(
            "get_plan_overview",
            args,
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !isCurrentChatScope(store, chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                plan?: { status?: unknown; name?: unknown; state?: unknown };
              };
              valid =
                overview.project === referencedProject &&
                overview.plan?.status === "available" &&
                overview.plan.name === referencedPlan &&
                overview.plan.state === "ACTIVE";
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.plan-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          overviewEvidence = mergeOverviewEvidence(
            overviewEvidence,
            extractPlanOverviewEvidence(overviewResult)
          );
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_plan_overview",
                  arguments: JSON.stringify(args),
                  origin: "runtime",
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        } else if (
          !skipReferencePreflight &&
          referencedIssue &&
          referencedProject
        ) {
          if (!(await admitExecution())) {
            reportPersistenceFailure();
            return;
          }
          const preflightCallId = uuidv4();
          const args = { issueName: referencedIssue };
          const result = await executor(
            "get_issue_overview",
            args,
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !isCurrentChatScope(store, chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          let valid = false;
          if (overviewResult) {
            try {
              const overview = JSON.parse(overviewResult) as {
                project?: unknown;
                issue?: { status?: unknown; name?: unknown };
              };
              valid =
                overview.project === referencedProject &&
                overview.issue?.status === "available" &&
                overview.issue.name === referencedIssue;
            } catch {
              valid = false;
            }
          }
          if (!valid || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.issue-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          overviewEvidence = mergeOverviewEvidence(
            overviewEvidence,
            extractIssueOverviewEvidence(overviewResult)
          );
          observeHandoffs("get_issue_overview", overviewResult);
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_issue_overview",
                  arguments: JSON.stringify(args),
                  origin: "runtime",
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        } else if (
          !skipReferencePreflight &&
          referencedDatabase &&
          referencedProject
        ) {
          if (!(await admitExecution())) {
            reportPersistenceFailure();
            return;
          }
          const preflightCallId = uuidv4();
          const args = { databaseName: referencedDatabase };
          const result = await executor(
            "get_database_overview",
            args,
            preflightCallId
          ).catch(() => undefined);
          if (
            controller.signal.aborted ||
            runToken !== runTokens.get(chatId) ||
            !isCurrentChatScope(store, chatId)
          ) {
            return;
          }
          const overviewResult =
            result?.kind === "tool_result" ? result.result : undefined;
          verifiedDatabase = overviewResult
            ? getVerifiedDatabaseOverview(
                overviewResult,
                referencedDatabase,
                referencedProject
              )
            : undefined;
          if (!verifiedDatabase || !overviewResult) {
            useAgentStore.getState().addMessage({
              chatId,
              role: "assistant",
              content: t("agent.database-reference-unavailable"),
              metadata: { route: page.path, runId },
            });
            useAgentStore.getState().finishChatRun(chatId);
            return;
          }
          overviewEvidence = mergeOverviewEvidence(
            overviewEvidence,
            extractDatabaseOverviewEvidence(overviewResult)
          );
          history.push(
            {
              role: "assistant",
              toolCalls: [
                {
                  id: preflightCallId,
                  name: "get_database_overview",
                  arguments: JSON.stringify(args),
                  origin: "runtime",
                },
              ],
            },
            {
              role: "tool",
              toolCallId: preflightCallId,
              content: overviewResult,
            }
          );
        }
        if (managed) {
          const startupPairs = history.slice(startupStart);
          for (const message of startupPairs) {
            useAgentStore.getState().addMessage({
              ...message,
              chatId,
              metadata: { route: page.path, runId, runPreflight: true },
            });
          }
        }
        let checkpointPhase: Pick<
          AgentRunCheckpoint,
          "phase" | "finalizationText"
        > = resumeCheckpoint ?? { phase: "working" };
        const saveCheckpoint = async (phase = checkpointPhase) => {
          if (!ownsInvocation()) return false;
          checkpointPhase = phase;
          const current = useAgentStore.getState();
          const previous = current.getChat(chatId)?.runCheckpoint;
          const throughMessageId = current.getMessages(chatId).at(-1)?.id;
          if (!previous || !throughMessageId) return false;
          return current.saveChatRunCheckpoint(chatId, {
            ...previous,
            ...(pageToolPath !== undefined ? { pageToolPath } : {}),
            ...phase,
            suspension: "interrupted",
            throughMessageId,
            maskingSections: [...maskingSections],
            startupComplete: true,
            referencePreflightPending: undefined,
          });
        };
        if (managed && !(await saveCheckpoint())) {
          useAgentStore.getState().setChatStatus(chatId, "error", {
            lastError: getRunUpdateError(t("agent.token-budget.action-failed")),
          });
          return;
        }
        let finalText: string | undefined;
        let finalHandoffProposal: AgentHandoff | undefined;
        const outcome = await runAgentLoop(
          history,
          tools,
          (name, args, toolCallId) => {
            if (invalidatesPlanDraftEvidence(name, args)) {
              handoffCandidates.length = 0;
              prefetchedProjectOverview = undefined;
              overviewEvidence = undefined;
              verifiedDatabase = undefined;
              readDatabaseSchema = false;
              maskingDraftIds.length = 0;
              exemptionDraftIds.length = 0;
              semanticTypeDraftIds.length = 0;
              preparedAccessRequestDraft = undefined;
            }
            const execution =
              name === "get_project_overview" && prefetchedProjectOverview
                ? Promise.resolve({
                    kind: "tool_result" as const,
                    result: prefetchedProjectOverview,
                  })
                : executor(name, args, toolCallId);
            return execution.then((result) => {
              if (
                name === "get_database_overview" &&
                args.databaseName === referencedDatabase &&
                result.kind === "tool_result"
              ) {
                verifiedDatabase = getVerifiedDatabaseOverview(
                  result.result,
                  referencedDatabase,
                  referencedProject
                );
                if (!verifiedDatabase) readDatabaseSchema = false;
              }
              if (
                name === "call_api" &&
                verifiedDatabase &&
                isSuccessfulDatabaseSchemaRead(
                  args,
                  result,
                  verifiedDatabase.name
                )
              ) {
                readDatabaseSchema = true;
              }
              return result;
            });
          },
          {
            ...(!managed ? { onUsage: recordUsage } : {}),
            onAssistantMessage: (message) => {
              if (managed && !ownsInvocation()) return;
              for (const toolCall of message.toolCalls ?? []) {
                toolNamesByCallId.set(toolCall.id, toolCall.name);
              }
              if (!message.content && !message.toolCalls?.length) return;
              useAgentStore.getState().addMessage({
                chatId,
                role: "assistant",
                content: message.content,
                toolCalls: message.toolCalls,
                metadata: { route: page.path, runId },
              });
            },
            onToolResult: async (toolCallId, result) => {
              if (
                managed &&
                (runTokens.get(chatId) !== runToken ||
                  !isCurrentChatScope(store, chatId) ||
                  (useAgentStore.getState().getChat(chatId)?.runId !== runId &&
                    !(
                      useAgentStore.getState().getChat(chatId)?.archived &&
                      useAgentStore.getState().getChat(chatId)?.runBudget
                        ?.runId === runId
                    )))
              )
                return;
              const current = useAgentStore.getState();
              const attempt =
                toolNamesByCallId.get(toolCallId) === "call_api"
                  ? current
                      .getChat(chatId)
                      ?.apiAttempts?.find(
                        (a) =>
                          a.runId === runId &&
                          a.toolCallId === toolCallId &&
                          parseCallApiResult(result, a.operationId).execution
                            ?.attemptId === a.id
                      )
                  : undefined;
              if (
                attempt &&
                current.getApiAttemptToolResult(chatId, attempt.id)
              ) {
                return;
              }
              if (managed && controller.signal.aborted) {
                useAgentStore.getState().addMessage({
                  chatId,
                  role: "tool",
                  toolCallId,
                  content: result,
                  metadata: { route: page.path, runId },
                });
                if (!(await useAgentStore.getState().flushRunPersistence()))
                  throw new Error("Agent tool result could not be persisted");
                return;
              }
              const toolName = toolNamesByCallId.get(toolCallId);
              if (toolName) observeHandoffs(toolName, result);
              if (toolName === "get_project_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractProjectOverviewEvidence(result)
                );
              } else if (toolName === "get_issue_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractIssueOverviewEvidence(result)
                );
              } else if (toolName === "get_plan_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractPlanOverviewEvidence(result)
                );
              } else if (toolName === "get_database_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractDatabaseOverviewEvidence(result)
                );
              } else if (toolName === "get_workspace_overview") {
                overviewEvidence = mergeOverviewEvidence(
                  overviewEvidence,
                  extractWorkspaceOverviewEvidence(result)
                );
              }
              useAgentStore.getState().addMessage({
                chatId,
                role: "tool",
                toolCallId,
                content: result,
                metadata: { route: page.path, runId },
              });
              if (
                managed &&
                !(await useAgentStore.getState().flushRunPersistence())
              )
                throw new Error("Agent tool result could not be persisted");
            },
            onText: (text, proposedHandoff) => {
              if (!text) return;
              finalText = text;
              finalHandoffProposal = proposedHandoff;
            },
          },
          controller.signal,
          {
            allowPlainTextCompletion: runMode === "read-only",
            maxTotalTokens,
            resolveRequestContext,
            ...(managed
              ? {
                  runId,
                  checkpoint: resumeCheckpoint,
                  reserveProviderAttempt: async () => {
                    const reservation = ownsInvocation()
                      ? await useAgentStore
                          .getState()
                          .reserveChatRunAttempt(chatId, runId)
                      : { kind: "superseded" as const };
                    if (reservation.kind === "allowed") {
                      admitted = true;
                      setPendingProviderAttempt(
                        controller,
                        reservation.attemptId
                      );
                    }
                    return reservation;
                  },
                  onAttemptSettled: async (attempt) => {
                    try {
                      return isCurrentChatScope(store, chatId)
                        ? await useAgentStore
                            .getState()
                            .recordChatModelAttempt(chatId, attempt)
                        : false;
                    } finally {
                      if (
                        pendingProviderAttempts.get(controller) ===
                        attempt.attemptId
                      )
                        setPendingProviderAttempt(controller);
                    }
                  },
                  onCheckpoint: saveCheckpoint,
                  beforeToolExecution: async () =>
                    ownsInvocation() &&
                    !useAgentStore.getState().getChat(chatId)?.archived &&
                    (await saveCheckpoint()) &&
                    (await admitExecution()),
                }
              : { onCheckpoint: admitExecution }),
          }
        );

        if (
          controller.signal.aborted ||
          runToken !== runTokens.get(chatId) ||
          !isCurrentChatScope(store, chatId)
        ) {
          return;
        }
        if (
          outcome.kind === "completed" &&
          resumeCheckpoint?.phase === "finalizing" &&
          typeof resumeCheckpoint.finalizationText === "string"
        )
          admitted = true;
        const queryCalls = getCurrentTurnQueryCalls(
          useAgentStore.getState().getMessages(chatId)
        );
        const accessRequestDraft = withDefaultAccessGrantReason(
          withPreparedAccessGrantReason(
            !capability &&
              outcome.kind === "completed" &&
              (!outcome.success ||
                (maskingDraftIds.length === 0 &&
                  exemptionDraftIds.length === 0 &&
                  semanticTypeDraftIds.length === 0))
              ? outcome.success && preparedAccessRequestDraft
                ? preparedAccessRequestDraft
                : queryCalls.length === 1 && queryCalls[0].result !== undefined
                  ? await resolveDeniedQueryAccessDraft(
                      queryCalls[0].args,
                      queryCalls[0].result
                    )
                      .then((draft) =>
                        preparedAccessRequestDraft ||
                        draft?.kind === "requestRole"
                          ? draft
                          : undefined
                      )
                      .catch(() => undefined)
                  : undefined
              : undefined,
            preparedAccessRequestDraft
          )
        );
        if (
          controller.signal.aborted ||
          runToken !== runTokens.get(chatId) ||
          !isCurrentChatScope(store, chatId)
        ) {
          return;
        }

        if (outcome.kind === "stopped") {
          useAgentStore.getState().addMessage({
            chatId,
            role: "assistant",
            content: t("agent.provider-attempt-limit"),
            metadata: { route: page.path, runId },
          });
        }

        if (finalText && outcome.kind === "completed") {
          let handoff = outcome.success
            ? acceptAgentHandoff(finalHandoffProposal, handoffCandidates)
            : undefined;
          const proposedActions =
            capability && Array.isArray(outcome.actions)
              ? outcome.actions.filter(
                  (action) =>
                    action !== null &&
                    typeof action === "object" &&
                    (action.kind === "open" || action.kind === "continue")
                )
              : outcome.actions;
          let actions = outcome.success
            ? acceptAgentResponseActions(
                proposedActions,
                handoffCandidates,
                !capability && verifiedDatabase && readDatabaseSchema
                  ? { verifiedDatabase, answer: finalText }
                  : undefined,
                maskingDraftIds,
                exemptionDraftIds,
                semanticTypeDraftIds
              )
            : [];
          if (outcome.success && accessRequestDraft) {
            actions = actions.filter((action) => action.kind === "continue");
            handoff = undefined;
          }
          const sqlFixAction =
            capability && outcome.success
              ? useAgentStore.getState().prepareSqlFixDraft(chatId, finalText)
              : undefined;
          if (sqlFixAction) {
            actions = [
              sqlFixAction,
              ...actions.filter((action) => action.kind === "continue"),
            ];
            handoff = undefined;
          }
          let queryMaskingEligibility: AgentQueryMaskingEligibility | undefined;
          if (outcome.success && !capability && !accessRequestDraft) {
            const turnMessages = useAgentStore.getState().getMessages(chatId);
            const maskingSetupTarget = getQueryMaskingSetupTarget(
              actions,
              turnMessages,
              references
            );
            if (maskingSetupTarget) {
              queryMaskingEligibility = await resolveQueryMaskingEligibility(
                store.workspaceName ?? "",
                maskingSetupTarget
              );
              if (
                !ownsInvocation() ||
                useAgentStore.getState().getChat(chatId)?.archived
              )
                return;
            }
            actions = getQueryResponseActions(
              actions,
              turnMessages,
              references,
              t,
              queryMaskingEligibility
            );
          }
          useAgentStore.getState().addMessage({
            chatId,
            role: "assistant",
            content: finalText,
            metadata: {
              route: page.path,
              runId,
              ...(overviewEvidence ? { evidence: overviewEvidence } : {}),
              ...(handoff ? { handoff } : {}),
              ...(actions.length > 0 ? { actions } : {}),
              ...(queryMaskingEligibility ? { queryMaskingEligibility } : {}),
              ...(accessRequestDraft ? { accessRequestDraft } : {}),
              ...(actions.some((action) => action.kind === "continue") &&
              references.length > 0
                ? { resourceReferences: references }
                : {}),
            },
          });
        }

        if (managed && outcome.kind === "paused") {
          const checkpoint = useAgentStore
            .getState()
            .getChat(chatId)?.runCheckpoint;
          if (
            !checkpoint ||
            !(await useAgentStore.getState().pauseChatRun(chatId, {
              ...checkpoint,
              phase: outcome.phase,
              finalizationText: outcome.finalizationText,
              suspension: "token-limit",
            }))
          )
            reportPersistenceFailure();
        } else {
          if (managed && outcome.kind === "awaiting_user") {
            if (!(await saveCheckpoint())) {
              reportPersistenceFailure();
              return;
            }
            const checkpoint = useAgentStore
              .getState()
              .getChat(chatId)?.runCheckpoint;
            if (
              !checkpoint ||
              !(await useAgentStore.getState().saveChatRunCheckpoint(chatId, {
                ...checkpoint,
                suspension: "awaiting-user",
              }))
            ) {
              reportPersistenceFailure();
              return;
            }
          }
          const succeeded =
            outcome.kind === "completed" && outcome.success && ownsInvocation();
          handleOutcome(chatId, outcome);
          if (
            managed &&
            !(await useAgentStore.getState().flushRunPersistence())
          ) {
            reportPersistenceFailure();
          } else {
            completedSuccessfully = succeeded;
          }
        }
      } catch (error) {
        if (!managed) throw error;
        reportPersistenceFailure(error);
      } finally {
        setPendingProviderAttempt(controller);
        if (
          useAgentStore.getState().getAbortController(chatId) === controller
        ) {
          useAgentStore.getState().setAbortController(chatId, null);
        }
      }
      return {
        accepted: admitted,
        succeeded: completedSuccessfully && !controller.signal.aborted,
      } satisfies ChatRunResult;
    },
    [buildChatHistory, handleOutcome, t, workspace?.name]
  );

  const runRecentActivityStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string,
      parent: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.recent-activity.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await getRecentActivity(parent);
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatRecentActivity(result),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
        return !parseRecentActivityResult(result).error;
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.recent-activity.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatRecentActivity, t]
  );

  const runWorkspaceAttentionStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.workspace-attention.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await createWorkspaceAttentionTool({
          workspaceName: store.getChat(chatId)?.context,
        })();
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatWorkspaceAttention(result),
          metadata: {
            route: page.path,
            runId,
            handoff: collectAgentHandoffs("get_workspace_attention", result, {
              workspaceName: store.getChat(chatId)?.context,
            })[0],
          },
        });
        store.finishChatRun(chatId);
        return !parseAttentionResult(result).error;
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.workspace-attention.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatWorkspaceAttention, t]
  );

  const runProjectAttentionStarter = useCallback(
    async (
      chatId: string,
      page: { path: string; title: string },
      runId: string
    ) => {
      const store = useAgentStore.getState();
      const controller = new AbortController();
      const runToken = (runTokens.get(chatId) ?? 0) + 1;
      runTokens.set(chatId, runToken);
      store.setAbortController(chatId, controller);
      store.addMessage({
        chatId,
        role: "assistant",
        content: t("agent.project-attention.checking"),
        metadata: { route: page.path, runId, pending: true },
      });

      try {
        const result = await createProjectOpenIssuesTool({
          projectName: store.getChat(chatId)?.context,
        })();
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: formatProjectAttention(result),
          metadata: {
            route: page.path,
            runId,
            handoff: collectAgentHandoffs("get_project_open_issues", result, {
              projectName: store.getChat(chatId)?.context,
            })[0],
          },
        });
        store.finishChatRun(chatId);
        return !parseAttentionResult(result).error;
      } catch {
        if (controller.signal.aborted || runToken !== runTokens.get(chatId)) {
          store.removeMessagesByRunId(chatId, runId);
          return;
        }
        store.removeMessagesByRunId(chatId, runId);
        store.addMessage({
          chatId,
          role: "assistant",
          content: t("agent.project-attention.unavailable"),
          metadata: { route: page.path, runId },
        });
        store.finishChatRun(chatId);
      } finally {
        if (store.getAbortController(chatId) === controller) {
          store.setAbortController(chatId, null);
        }
      }
    },
    [formatProjectAttention, t]
  );

  const executeChatRun = useCallback(
    async (
      chatId: string,
      currentPage: { path: string; title: string },
      runMode: AgentRunMode = "default",
      starterId?: LandingStarterId,
      skipReferencePreflight = false,
      intent: "new" | "continue" | "startup" = "new",
      lease?: AgentRunLease,
      frozenPageContext?: FrozenPageContext
    ) => {
      let accepted = false;
      let succeeded = false;
      await withChatRunSegment(
        chatId,
        async () => {
          let runId = uuidv4();
          const store = useAgentStore.getState();
          store.syncState();
          const currentStore = useAgentStore.getState();
          const chat = currentStore.getChat(chatId);
          if (
            !chat ||
            chat.archived ||
            (currentStore.remoteBusyChatIds[chatId] &&
              !supportsAgentRunRecovery())
          ) {
            return;
          }
          const references = getCurrentTurnResourceReferences(
            store.getMessages(chatId)
          );
          const hasResourceReference =
            references.some(
              (reference) =>
                reference.kind === "project" ||
                reference.kind === "database" ||
                reference.kind === "issue" ||
                reference.kind === "plan"
            ) || currentStore.getChat(chatId)?.capability !== undefined;
          let effectiveRunMode = hasResourceReference ? "read-only" : runMode;
          if (
            !hasResourceReference &&
            (starterId === "workspace-recent" ||
              starterId === "project-open-recent" ||
              starterId === "workspace-attention" ||
              starterId === "project-open-blockers") &&
            (!(await currentStore.checkAgentStorageAdmission()) ||
              !isCurrentChatScope(currentStore, chatId))
          )
            return;
          let resumeCheckpoint: AgentRunCheckpoint | undefined;
          if (supportsAgentRunRecovery()) {
            if (intent === "startup") {
              const startup = getAgentRunStartup(
                chat,
                currentStore.getMessages(chatId)
              );
              if (!startup) return;
              const restarted = await currentStore.retryChatRunStartup(
                chatId,
                startup.runId
              );
              const latest = useAgentStore.getState();
              if (
                !isCurrentChatScope(currentStore, chatId) ||
                latest.currentChatId !== chatId ||
                latest.getChat(chatId)?.runId !== startup.runId ||
                latest.getChat(chatId)?.archived
              )
                return;
              if (!restarted) {
                latest.setChatStatus(chatId, "error", {
                  lastError: getRunUpdateError(
                    t("agent.token-budget.action-failed")
                  ),
                });
                return;
              }
              runId = startup.runId;
              currentPage = startup.bindings.page;
              effectiveRunMode = startup.bindings.runMode;
              skipReferencePreflight = false;
            } else if (intent === "continue") {
              if (!canContinueChatRun(chatId)) return;
              const continuation = getAgentRunContinuation(
                chat,
                currentStore.getMessages(chatId)
              );
              if (!continuation) return;
              if (
                !(await currentStore.continueChatRun(
                  chatId,
                  continuation.checkpoint.runId
                ))
              ) {
                currentStore.setChatStatus(chatId, "error", {
                  lastError: getRunUpdateError(
                    t("agent.token-budget.action-failed")
                  ),
                });
                return;
              }
              resumeCheckpoint = useAgentStore
                .getState()
                .getChat(chatId)?.runCheckpoint;
              if (!resumeCheckpoint) return;
              runId = resumeCheckpoint.runId;
              currentPage = resumeCheckpoint.bindings.page;
            } else {
              const capability = normalizeAgentDiagnosticCapability(
                chat.capability
              );
              const pageResource = frozenPageContext
                ? frozenPageContext.maskingPage
                : getCurrentAgentPageResource();
              const bindings: AgentRunBindings = {
                page: currentPage,
                ...(frozenPageContext
                  ? { pageToolPath: currentPage.path }
                  : {}),
                context: chat.context,
                references: capability ? [capability.resource] : references,
                runMode: effectiveRunMode,
                ...(capability ? { capability } : {}),
                ...(isAgentMaskingPageResource(pageResource)
                  ? { maskingPage: pageResource }
                  : {}),
                ...(!hasResourceReference
                  ? {
                      instanceName: resolveInstanceNameFromPagePath(
                        currentPage.path
                      ),
                    }
                  : {}),
              };
              const rootMessage = currentStore
                .getMessages(chatId)
                .findLast((message) => message.role === "user");
              if (!rootMessage) return;
              const started = await currentStore.beginChatRun(
                chatId,
                rootMessage.id,
                bindings
              );
              if (!started) {
                currentStore.setChatStatus(chatId, "error", {
                  lastError: getRunUpdateError(
                    t("agent.token-budget.action-failed")
                  ),
                });
                return;
              }
              runId = started;
            }
          } else {
            store.updateChatPage(chatId, currentPage);
            store.startChatRun(chatId, currentPage, {
              runId,
              runMode: effectiveRunMode,
            });
          }
          // Stop can interrupt startup while its initial checkpoint is being saved.
          const startedChat = useAgentStore.getState().getChat(chatId);
          if (
            !isCurrentChatScope(currentStore, chatId) ||
            !startedChat ||
            startedChat.archived ||
            startedChat.status !== "running" ||
            startedChat.runId !== runId
          )
            return;
          if (
            !resumeCheckpoint &&
            !hasResourceReference &&
            starterId === "workspace-recent"
          ) {
            accepted = true;
            succeeded = Boolean(
              await runRecentActivityStarter(
                chatId,
                currentPage,
                runId,
                "projects/-"
              )
            );
            return;
          }
          if (
            !resumeCheckpoint &&
            !hasResourceReference &&
            starterId === "project-open-recent"
          ) {
            const projectName = store.getChat(chatId)?.context;
            if (projectName?.startsWith(projectNamePrefix)) {
              accepted = true;
              succeeded = Boolean(
                await runRecentActivityStarter(
                  chatId,
                  currentPage,
                  runId,
                  projectName
                )
              );
              return;
            }
          }
          if (
            !resumeCheckpoint &&
            !hasResourceReference &&
            starterId === "workspace-attention"
          ) {
            accepted = true;
            succeeded = Boolean(
              await runWorkspaceAttentionStarter(chatId, currentPage, runId)
            );
            return;
          }
          if (
            !resumeCheckpoint &&
            !hasResourceReference &&
            starterId === "project-open-blockers"
          ) {
            const projectName = store.getChat(chatId)?.context;
            if (projectName?.startsWith(projectNamePrefix)) {
              accepted = true;
              succeeded = Boolean(
                await runProjectAttentionStarter(chatId, currentPage, runId)
              );
              return;
            }
          }
          const result = await runChat(
            chatId,
            currentPage,
            runId,
            effectiveRunMode,
            resumeCheckpoint?.referencePreflightPending
              ? false
              : skipReferencePreflight || Boolean(resumeCheckpoint),
            resumeCheckpoint,
            frozenPageContext
          );
          accepted = result?.accepted ?? false;
          succeeded = result?.succeeded ?? false;
        },
        lease
      );
      if (succeeded && supportsAgentRunRecovery()) {
        succeeded = await useAgentStore.getState().flushRunPersistence();
      }
      return { accepted, succeeded } satisfies ChatRunResult;
    },
    [
      canContinueChatRun,
      runChat,
      runProjectAttentionStarter,
      runRecentActivityStarter,
      runWorkspaceAttentionStarter,
      t,
    ]
  );

  const drainFollowUps = useCallback(
    async (chatId: string, lease?: AgentRunLease, resume = false) => {
      const scope = useAgentStore.getState();
      const key = JSON.stringify([
        scope.workspaceName,
        scope.userEmail,
        chatId,
      ]);
      if (drainingFollowUps.has(key)) return;
      drainingFollowUps.add(key);
      try {
        await withChatRunSegment(
          chatId,
          async (owner) => {
            const store = useAgentStore.getState();
            const chat = store.getChat(chatId);
            if (
              !isCurrentChatScope(scope, chatId) ||
              !chat ||
              chat.archived ||
              store.remoteBusyChatIds[chatId] ||
              store.getAbortController(chatId) ||
              store.getPendingAsk(chatId) ||
              chat.status === "running" ||
              chat.status === "paused" ||
              chat.runCheckpoint?.suspension === "token-limit"
            )
              return;
            if (resume) {
              if (
                chat.runId &&
                supportsAgentRunRecovery() &&
                !(await store.endPausedRun(chatId))
              )
                return;
              if (!isCurrentChatScope(scope, chatId)) return;
              store.clearError(chatId);
              store.setFollowUpQueuePaused(chatId, false);
            }
            if (useAgentStore.getState().followUpQueuesByChatId[chatId]?.paused)
              return;
            if (
              !store.followUpQueuesByChatId[chatId]?.messages.length ||
              !store.setFollowUpDrainActive(chatId, true)
            )
              return;
            try {
              if (
                supportsAgentRunRecovery() &&
                !(await store.flushRunPersistence())
              ) {
                store.setFollowUpQueuePaused(chatId, true);
                return;
              }
              while (isCurrentChatScope(scope, chatId)) {
                const current = useAgentStore.getState();
                const queue = current.followUpQueuesByChatId[chatId];
                const head = queue?.messages[0];
                if (!head || queue.paused) return;
                if (
                  (await blockWhenAIIsNotConfigured(chatId)) ||
                  !(await current.checkAgentStorageAdmission()) ||
                  (supportsAgentRunRecovery() &&
                    !(await current.flushRunPersistence()))
                ) {
                  if (isCurrentChatScope(scope, chatId))
                    current.setFollowUpQueuePaused(chatId, true);
                  return;
                }
                if (!isCurrentChatScope(scope, chatId)) return;
                // Admission is asynchronous; deletion or Stop may have changed the head.
                const latest =
                  useAgentStore.getState().followUpQueuesByChatId[chatId];
                if (latest?.paused) return;
                if (latest?.messages[0]?.id !== head.id) continue;
                const promoted = current.promoteFollowUp(chatId, head.id);
                if (!promoted) return;
                if (!current.commitQueuedUserMessage(chatId)) {
                  current.setFollowUpQueuePaused(chatId, true);
                  return;
                }
                const result = await executeChatRun(
                  chatId,
                  promoted.page,
                  promoted.runMode ?? "default",
                  promoted.starterId,
                  false,
                  "new",
                  owner,
                  { maskingPage: promoted.maskingPage }
                );
                if (!isCurrentChatScope(scope, chatId)) return;
                if (!result.succeeded) {
                  const status = current.getChat(chatId)?.status;
                  if (status !== "awaiting_user" && status !== "paused")
                    current.setFollowUpQueuePaused(chatId, true);
                  return;
                }
              }
            } finally {
              if (
                isCurrentChatScope(scope, chatId) &&
                store.setFollowUpDrainActive(chatId, false) &&
                supportsAgentRunRecovery()
              ) {
                await store.flushRunPersistence();
              }
            }
          },
          lease
        );
      } catch {
        if (isCurrentChatScope(scope, chatId))
          scope.setFollowUpQueuePaused(chatId, true);
      } finally {
        drainingFollowUps.delete(key);
      }
    },
    [blockWhenAIIsNotConfigured, executeChatRun]
  );

  const startChatRun = useCallback(
    async (...args: Parameters<typeof executeChatRun>) => {
      const [
        chatId,
        page,
        mode,
        starter,
        skipPreflight,
        intent,
        lease,
        frozenPage,
      ] = args;
      const scope = useAgentStore.getState();
      let accepted = false;
      await withChatRunSegment(
        chatId,
        async (owner) => {
          const result = await executeChatRun(
            chatId,
            page,
            mode,
            starter,
            skipPreflight,
            intent,
            owner,
            frozenPage
          );
          accepted = result.accepted;
          if (!isCurrentChatScope(scope, chatId)) return;
          if (result.succeeded) {
            await drainFollowUps(chatId, owner);
          } else {
            const status = scope.getChat(chatId)?.status;
            if (status !== "awaiting_user" && status !== "paused")
              scope.setFollowUpQueuePaused(chatId, true);
          }
        },
        lease
      );
      return accepted;
    },
    [drainFollowUps, executeChatRun]
  );

  useEffect(() => {
    const store = useAgentStore.getState();
    for (const chatId of Object.keys(queuedUserMessagesByChatId)) {
      const message = store.takeQueuedUserMessage(chatId);
      if (!message) continue;

      void (async () => {
        const restoreQueuedText = () => {
          const draft = store.getDraft(message.chatId);
          if (!draft.text && !draft.project && !draft.resource) {
            const references = normalizeAgentResourceReferences(
              message.resourceReferences,
              message.resourceReference
            );
            let offset = 0;
            const mentions = references.map((resource) => {
              const start = offset;
              const end = start + formatResourceMention(resource.label).length;
              offset = end + 1;
              return { resource, range: { start, end } };
            });
            const hasReferencePrefix =
              normalizeAgentResourceMentions(message.text, mentions).length ===
              references.length;
            const prefix = references
              .map((resource) => formatResourceMention(resource.label))
              .join(" ");
            const text =
              references.length > 0 && !hasReferencePrefix
                ? `${prefix}${message.text ? ` ${message.text}` : " "}`
                : message.text;
            store.setDraft(message.chatId, {
              text,
              ...(message.followUpSourceId !== undefined
                ? { followUpSourceId: message.followUpSourceId }
                : {}),
              ...(references.length > 0
                ? message.resourceReferences
                  ? { mentions }
                  : {
                      resource: references[0],
                      mention: mentions[0]?.range,
                    }
                : {}),
            });
          }
        };
        if (store.getChat(message.chatId)?.archived) {
          restoreQueuedText();
          store.finishQueuedUserMessage(message.chatId);
          return;
        }
        let isBlocked: boolean;
        try {
          isBlocked = await blockWhenAIIsNotConfigured(message.chatId);
        } catch {
          if (!isCurrentChatScope(store, message.chatId)) return;
          if (
            store.queuedUserMessagesByChatId[message.chatId]?.id === message.id
          ) {
            restoreQueuedText();
            store.finishQueuedUserMessage(message.chatId);
          }
          return;
        }
        if (!isCurrentChatScope(store, message.chatId)) return;
        if (isBlocked || store.getChat(message.chatId)?.archived) {
          restoreQueuedText();
          store.finishQueuedUserMessage(message.chatId);
          return;
        }
        await withChatRunSegment(message.chatId, async (lease) => {
          if (
            supportsAgentRunRecovery() &&
            store.getChat(message.chatId)?.runId
          ) {
            restoreQueuedText();
            store.finishQueuedUserMessage(message.chatId);
            return;
          }
          const committed = store.commitQueuedUserMessage(message.chatId);
          if (!committed) return;
          store.clearError(message.chatId);
          await startChatRun(
            message.chatId,
            message.page,
            message.runMode ?? "default",
            message.starterId,
            false,
            "new",
            lease
          );
        });
      })();
    }
  }, [blockWhenAIIsNotConfigured, queuedUserMessagesByChatId, startChatRun]);

  const send = useCallback(async () => {
    const store = useAgentStore.getState();
    const chat = selectCurrentChat(store);
    if (
      chat &&
      !store.getPendingAsk(chat.id) &&
      !chat.archived &&
      chat.status !== "paused" &&
      chat.runCheckpoint?.suspension !== "token-limit" &&
      (store.isChatRunning(chat.id) ||
        store.getAbortController(chat.id) ||
        store.queuedUserMessagesByChatId[chat.id] ||
        store.followUpQueuesByChatId[chat.id]?.messages.length)
    ) {
      const draft = store.getDraft(chat.id);
      const text = draft.text.trim();
      const mentions = getDraftResourceMentions(draft);
      if (
        !text ||
        (mentions.length > 0 && hasUnboundResourceMention(draft.text, mentions))
      )
        return;
      const references = normalizeAgentResourceReferences(
        mentions.map(({ resource }) => resource)
      );
      const pageResource = getCurrentAgentPageResource();
      const queuedId = store.enqueueFollowUp({
        chatId: chat.id,
        text,
        page: getCurrentPageSnapshot(),
        runMode: "default",
        ...(draft.followUpSourceId !== undefined
          ? { followUpSourceId: draft.followUpSourceId }
          : {}),
        ...(isAgentMaskingPageResource(pageResource)
          ? { maskingPage: pageResource }
          : {}),
        ...(references.length ? { resourceReferences: references } : {}),
      });
      if (queuedId) {
        if (store.getDraft(chat.id) === draft) store.clearDraft(chat.id);
        setResourcePickerOpen(false);
        mentionRangeRef.current = null;
      }
      return;
    }
    if (
      chat?.archived ||
      chat?.status === "paused" ||
      chat?.runCheckpoint?.suspension === "token-limit" ||
      !store.canUseAgentExecution() ||
      store.isChatRunning(chat?.id)
    )
      return;

    const currentPage = getCurrentPageSnapshot();
    const landingDraft = chat ? undefined : store.landingDraft;
    if (landingDraft) {
      const mentions = getDraftResourceMentions(landingDraft);
      if (
        !landingDraft.text.trim() ||
        (mentions.length > 0 &&
          hasUnboundResourceMention(landingDraft.text, mentions))
      )
        return;
    }
    const sourceThread = store.ensureCurrentChat(
      currentPage,
      (landingDraft ? store.landingContext : undefined) ??
        getCurrentChatContext(),
      workspace?.name
    );
    const sourceChatId = sourceThread.id;
    if (landingDraft) {
      store.setDraft(sourceChatId, landingDraft);
      if (useAgentStore.getState().landingDraft === landingDraft)
        store.clearLandingDraft();
    }
    const submittedDraft = store.getDraft(sourceChatId);
    const pendingAskBeforeConfig = store.getPendingAsk(sourceChatId);
    if (await blockWhenAIIsNotConfigured(sourceChatId)) return;
    if (!isCurrentChatScope(store, sourceChatId)) return;
    const currentStore = useAgentStore.getState();
    if (
      currentStore.getChat(sourceChatId)?.archived ||
      currentStore.getChat(sourceChatId)?.status === "paused" ||
      currentStore.getChat(sourceChatId)?.runCheckpoint?.suspension ===
        "token-limit" ||
      currentStore.isChatRunning(sourceChatId)
    ) {
      return;
    }
    const pendingAsk = currentStore.getPendingAsk(sourceChatId);
    if (pendingAsk?.toolCallId !== pendingAskBeforeConfig?.toolCallId) return;
    if (
      pendingAsk &&
      currentStore.getChat(sourceChatId)?.status !== "awaiting_user"
    ) {
      return;
    }
    if (pendingAsk) {
      return withChatRunSegment(sourceChatId, async (lease) => {
        if (!canContinueChatRun(sourceChatId)) return;
        const answer = input.trim();
        if (!answer) return;
        let answered;
        if (pendingAsk.kind === "input") {
          const mentions = getDraftResourceMentions(askDraft);
          if (
            mentions.length > 0 &&
            hasUnboundResourceMention(askDraft.text, mentions)
          ) {
            return;
          }
          const references = normalizeAgentResourceReferences(
            mentions.map(({ resource }) => resource)
          );
          answered = currentStore.answerPendingAsk(
            sourceChatId,
            { kind: "input", answer },
            {
              route: currentPage.path,
              runId: currentStore.getChat(sourceChatId)?.runId ?? undefined,
              ...(references[0] ? { resourceReference: references[0] } : {}),
              ...(references.length > 1
                ? { resourceReferences: references }
                : {}),
            }
          );
        } else {
          return;
        }
        if (!answered) return;
        setAskDraft({ text: "" });
        await startChatRun(
          sourceChatId,
          currentPage,
          currentStore.getChat(sourceChatId)?.runMode ?? "default",
          undefined,
          false,
          "continue",
          lease
        );
        return;
      });
    }

    currentStore.clearError(sourceChatId);
    setContinuationFailure(null);

    const text = submittedDraft.text.trim();
    if (!text) return;
    const mentions = getDraftResourceMentions(submittedDraft);
    if (
      mentions.length > 0 &&
      hasUnboundResourceMention(submittedDraft.text, mentions)
    )
      return;
    const references = normalizeAgentResourceReferences(
      mentions.map(({ resource }) => resource)
    );
    const stagedReference = references[0];
    const thread =
      stagedReference &&
      !sourceThread.context &&
      currentStore.currentChatId === sourceChatId
        ? store.selectOrCreateChatForContext(
            stagedReference.project,
            currentPage,
            workspace?.name
          )
        : sourceThread;
    if (!thread) return;
    const chatId = thread.id;
    return withChatRunSegment(chatId, async (lease) => {
      const latest = useAgentStore.getState().getChat(chatId);
      if (
        !latest ||
        latest.archived ||
        latest.status === "paused" ||
        latest.runCheckpoint?.suspension === "token-limit" ||
        useAgentStore.getState().isChatRunning(chatId)
      )
        return;
      if (
        supportsAgentRunRecovery() &&
        latest.runId &&
        !(await store.endPausedRun(chatId))
      )
        return;
      const submittedMessage = store.addMessage({
        chatId,
        role: "user",
        content: text,
        metadata: {
          route: currentPage.path,
          ...(submittedDraft.followUpSourceId !== undefined
            ? { followUpSourceId: submittedDraft.followUpSourceId }
            : {}),
          ...(stagedReference ? { resourceReference: stagedReference } : {}),
          ...(references.length > 1 ? { resourceReferences: references } : {}),
        },
      });
      if (!submittedMessage) return;
      if (store.getDraft(sourceChatId) === submittedDraft) {
        store.clearDraft(sourceChatId);
      }
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      const starterId = getLandingStarterIdForText({
        context: thread.context,
        text,
        translate: t,
      });
      await startChatRun(
        chatId,
        currentPage,
        "default",
        starterId,
        false,
        "new",
        lease
      );
    });
  }, [
    askDraft,
    blockWhenAIIsNotConfigured,
    canContinueChatRun,
    input,
    startChatRun,
    t,
    workspace?.name,
  ]);

  const runMessageAction = useCallback(
    async (
      messageId?: string,
      storageRecovery?: "new" | "continue" | "startup"
    ) => {
      const store = useAgentStore.getState();
      const chat = selectCurrentChat(store);
      if (
        !chat ||
        chat.archived ||
        store.isChatRunning(chat.id) ||
        store.getAbortController(chat.id) ||
        chat.status === "awaiting_user" ||
        (!messageId &&
          chat.status !== "idle" &&
          chat.status !== "error" &&
          !(storageRecovery === "continue" && chat.status === "paused")) ||
        store.getPendingAsk(chat.id) ||
        store.queuedUserMessagesByChatId[chat.id] ||
        store.queuedUserMessageInFlightByChatId[chat.id] ||
        (!messageId &&
          !storageRecovery &&
          chat.runCheckpoint?.suspension === "token-limit") ||
        messageActionInFlightRef.current
      ) {
        return;
      }
      const messages = store.getMessages(chat.id);
      const message = messageId
        ? messages.find((candidate) => candidate.id === messageId)
        : messages.findLast((candidate) => candidate.role === "user");
      if (
        message?.role !== "user" ||
        !message.content?.trim() ||
        (!messageId &&
          !storageRecovery &&
          chat.status !== "error" &&
          !chat.interrupted)
      ) {
        return;
      }

      const messageText = message.content;
      const retryStartup =
        !messageId && Boolean(getAgentRunStartup(chat, messages));
      const retryNew =
        storageRecovery === "new" ||
        (!messageId && !chat.runId && !chat.runCheckpoint);
      if (!messageId && store.storagePressure) {
        if (!retryStartup && !retryNew) canContinueChatRun(chat.id);
        return;
      }
      const action = Symbol();
      messageActionInFlightRef.current = action;
      const currentPage = getCurrentPageSnapshot();
      try {
        if (await blockWhenAIIsNotConfigured(chat.id)) return;
        const current = useAgentStore.getState();
        const currentChat = current.getChat(chat.id);
        if (
          messageActionInFlightRef.current !== action ||
          !isCurrentChatScope(store, chat.id) ||
          current.currentChatId !== chat.id ||
          currentChat?.archived ||
          currentChat?.status !== chat.status ||
          currentChat.runId !== chat.runId ||
          currentChat.lastError !== chat.lastError ||
          currentChat.interrupted !== chat.interrupted ||
          current.getMessages(chat.id) !== messages ||
          current.isChatRunning(chat.id) ||
          current.getAbortController(chat.id) ||
          current.getPendingAsk(chat.id) ||
          current.queuedUserMessagesByChatId[chat.id] ||
          current.queuedUserMessageInFlightByChatId[chat.id] ||
          getCurrentPageSnapshot().path !== currentPage.path
        ) {
          return;
        }
        let accepted = false;
        await withChatRunSegment(chat.id, async (lease) => {
          if (
            !messageId &&
            !retryStartup &&
            !retryNew &&
            !canContinueChatRun(chat.id)
          )
            return;
          if (
            messageId &&
            supportsAgentRunRecovery() &&
            current.getChat(chat.id)?.runId &&
            !(await current.endPausedRun(chat.id))
          ) {
            if (useAgentStore.getState().storagePressure) {
              current.setChatStatus(chat.id, "error", {
                lastError: getRunUpdateError(
                  t("agent.token-budget.action-failed")
                ),
              });
            }
            return;
          }
          if (!retryStartup) current.clearError(chat.id);
          if (retryNew) setContinuationFailure(null);
          let starterId: LandingStarterId | undefined;
          if (messageId) {
            setContinuationFailure(null);
            const references = normalizeAgentResourceReferences(
              message.metadata?.resourceReferences,
              message.metadata?.resourceReference
            );
            const submittedMessage = current.addMessage({
              chatId: chat.id,
              role: "user",
              content: messageText,
              metadata: {
                route: currentPage.path,
                ...(message.metadata?.followUpSourceId !== undefined
                  ? { followUpSourceId: message.metadata.followUpSourceId }
                  : {}),
                ...(references[0] ? { resourceReference: references[0] } : {}),
                ...(references.length > 1
                  ? { resourceReferences: references }
                  : {}),
              },
            });
            if (!submittedMessage) return;
            starterId = getLandingStarterIdForText({
              context: chat.context,
              text: messageText,
              translate: t,
            });
          }
          accepted = await startChatRun(
            chat.id,
            currentPage,
            messageId ? "default" : (chat.runMode ?? "default"),
            starterId,
            false,
            messageId || retryNew
              ? "new"
              : retryStartup
                ? "startup"
                : "continue",
            lease
          );
          if (accepted && retryStartup) setContinuationFailure(null);
        });
        return accepted;
      } finally {
        if (messageActionInFlightRef.current === action) {
          messageActionInFlightRef.current = null;
        }
      }
    },
    [blockWhenAIIsNotConfigured, canContinueChatRun, startChatRun, t]
  );

  const freeSpaceAndRetry = useCallback(async () => {
    const scope = useAgentStore.getState();
    const chatId = scope.currentChatId;
    if (!chatId || storageRecoveryInFlightRef.current) return false;
    const action = Symbol();
    storageRecoveryInFlightRef.current = action;
    try {
      if (!(await scope.freeAgentStorage())) return false;
      const current = useAgentStore.getState();
      if (
        storageRecoveryInFlightRef.current !== action ||
        !isCurrentChatScope(scope, chatId) ||
        current.currentChatId !== chatId
      )
        return false;
      const chat = current.getChat(chatId);
      if (!chat || chat.archived) return false;
      if (current.getPendingAsk(chatId)) return true;
      const continuation = getAgentRunContinuation(
        chat,
        current.getMessages(chatId)
      );
      if (continuation) {
        return Boolean(await runMessageAction(undefined, "continue"));
      } else if (getAgentRunStartup(chat, current.getMessages(chatId))) {
        return Boolean(await runMessageAction(undefined, "startup"));
      } else if (!chat.runId && !chat.runCheckpoint) {
        return Boolean(await runMessageAction(undefined, "new"));
      }
      return false;
    } finally {
      if (storageRecoveryInFlightRef.current === action)
        storageRecoveryInFlightRef.current = null;
    }
  }, [runMessageAction]);

  useEffect(() => {
    if (!freeSpaceAndRetryActionRef) return;
    freeSpaceAndRetryActionRef.current = freeSpaceAndRetry;
    return () => {
      if (freeSpaceAndRetryActionRef.current === freeSpaceAndRetry)
        freeSpaceAndRetryActionRef.current = null;
    };
  }, [freeSpaceAndRetryActionRef, freeSpaceAndRetry]);

  useEffect(() => {
    if (!retryActionRef) return;
    const retry = () => void runMessageAction();
    retryActionRef.current = retry;
    return () => {
      if (retryActionRef.current === retry) retryActionRef.current = null;
    };
  }, [retryActionRef, runMessageAction]);

  useEffect(() => {
    if (!resendActionRef) return;
    const resend = (messageId: string) => void runMessageAction(messageId);
    resendActionRef.current = resend;
    return () => {
      if (resendActionRef.current === resend) resendActionRef.current = null;
    };
  }, [resendActionRef, runMessageAction]);

  const endCurrentRun = async () => {
    const scope = useAgentStore.getState();
    const chat = currentChat;
    if (
      endingRunRef.current ||
      !chat?.runId ||
      chat.runBudget?.runId !== chat.runId ||
      chat.archived ||
      scope.currentChatId !== chat.id ||
      scope.remoteBusyChatIds[chat.id] ||
      !scope.canUseAgentExecution()
    )
      return;
    endingRunRef.current = true;
    setEndingRun(true);
    try {
      let ended = false;
      await withChatRunSegment(chat.id, async () => {
        const store = useAgentStore.getState();
        if (
          !isCurrentChatScope(scope, chat.id) ||
          store.currentChatId !== chat.id ||
          store.getChat(chat.id)?.runId !== chat.runId ||
          store.remoteBusyChatIds[chat.id]
        )
          return;
        store.syncState();
        const latest = useAgentStore.getState();
        const current = latest.getChat(chat.id);
        if (
          !current ||
          !isCurrentChatScope(scope, chat.id) ||
          latest.currentChatId !== chat.id ||
          current.runId !== chat.runId ||
          current.runBudget?.runId !== chat.runId ||
          current.archived ||
          latest.remoteBusyChatIds[chat.id] ||
          !latest.canUseAgentExecution()
        )
          return;
        ended = await latest.endPausedRun(chat.id);
      });
      if (!ended) throw new Error("Agent run could not be ended");
      setContinuationFailure((failure) =>
        failure?.chatId === chat.id && failure.runId === chat.runId
          ? null
          : failure
      );
    } catch (error) {
      const latest = useAgentStore.getState();
      if (
        !getAgentRunContinuation(
          latest.getChat(chat.id),
          latest.getMessages(chat.id)
        )
      ) {
        setContinuationFailure({
          chatId: chat.id,
          runId: chat.runId,
          endFailed: true,
        });
      }
      throw error;
    } finally {
      endingRunRef.current = false;
      setEndingRun(false);
    }
  };

  const cancelPendingRequest = useCallback(() => {
    const chat = selectCurrentChat(useAgentStore.getState());
    if (!chat) return;
    void withChatRunSegment(chat.id, async () => {
      const store = useAgentStore.getState();
      if (!store.cancelPendingAsk(chat.id)) return;
      setContinuationFailure(null);
      setAskDraft({ text: "" });
      if (supportsAgentRunRecovery() && !(await store.flushRunPersistence())) {
        store.setChatStatus(chat.id, "error", {
          lastError: getRunUpdateError(t("agent.token-budget.action-failed")),
        });
      }
    });
  }, [t]);

  const submitConfirmation = useCallback(
    async (confirmed: boolean) => {
      const store = useAgentStore.getState();
      const pendingAsk = selectCurrentPendingAsk(store);
      const chat = selectCurrentChat(store);
      if (
        chat?.archived ||
        !pendingAsk ||
        pendingAsk.kind !== "confirm" ||
        confirmationInFlightRef.current ||
        store.isChatRunning(chat?.id)
      ) {
        return;
      }

      if (pendingAsk.apiPermission && chat) {
        const permission = pendingAsk.apiPermission;
        confirmationInFlightRef.current = true;
        let approvalController: AbortController | undefined;
        try {
          const chatId = chat.id;
          const currentPage = getCurrentPageSnapshot();
          const permissionArgs = {
            actionId: pendingAsk.apiPermission.actionId,
            chatId,
            toolCallId: pendingAsk.toolCallId,
            userName: useAppStore.getState().currentUser?.name ?? "",
          };
          const preview = () =>
            getPendingApiCallPreview({
              ...permissionArgs,
              domContext: permissionDomContext(
                store,
                chatId,
                chat.runId,
                permission.actionId
              ),
            });
          const initialPreview = preview();
          let requestAvailable = Boolean(initialPreview);
          const recheckPage = () => {
            if (initialPreview?.dom && requestAvailable)
              requestAvailable = Boolean(preview());
          };
          if (requestAvailable && (await blockWhenAIIsNotConfigured(chatId))) {
            return;
          }
          recheckPage();

          await withChatRunSegment(chatId, async (lease) => {
            const currentStore = useAgentStore.getState();
            const sameAsk = () =>
              isCurrentChatScope(store, chatId) &&
              !currentStore.getChat(chatId)?.archived &&
              currentStore.getChat(chatId)?.runId === chat.runId &&
              currentStore.getPendingAsk(chatId)?.apiPermission?.actionId ===
                permission.actionId &&
              !currentStore.isChatRunning(chatId);
            if (!sameAsk()) return;
            recheckPage();
            if (confirmed && requestAvailable && !canContinueChatRun(chatId))
              return;
            if (
              supportsAgentRunRecovery() &&
              (!(await currentStore.flushRunPersistence()) || !sameAsk())
            )
              return;
            recheckPage();
            if (
              confirmed &&
              requestAvailable &&
              (!(await currentStore.checkAgentStorageAdmission()) || !sameAsk())
            )
              return;

            recheckPage();
            const runMode = currentStore.getChat(chatId)?.runMode ?? "default";
            const capability = currentStore.getChat(chatId)?.capability;
            const active = currentStore.getChat(chatId);
            const root = currentStore
              .getMessages(chatId)
              .findLast((m) => m.role === "user");
            const context =
              root && active?.runId
                ? currentStore.createApiDispatchContext({
                    chatId,
                    runId: active.runId,
                    rootUserMessageId: root.id,
                    toolCallId: pendingAsk.toolCallId,
                    logicalActionId: permission.actionId,
                  })
                : null;
            currentStore.setChatStatus(chatId, "running");
            approvalController = new AbortController();
            currentStore.setAbortController(chatId, approvalController);
            const domContext =
              initialPreview?.dom &&
              permissionDomContext(
                store,
                chatId,
                chat.runId,
                permission.actionId,
                approvalController
              );
            const result = await resolvePendingApiCall({
              ...permissionArgs,
              ...(domContext ? { domContext } : {}),
              approved: confirmed && requestAvailable,
              signal: approvalController.signal,
              ...(context && { context }),
              ...(capability !== undefined ? { capability } : {}),
            });
            const currentChat = currentStore.getChat(chatId);
            if (
              !isCurrentChatScope(store, chatId) ||
              !currentChat ||
              (currentChat.runId !== chat.runId &&
                !(
                  currentChat.archived &&
                  currentChat.runBudget?.runId === chat.runId
                )) ||
              (!supportsAgentRunRecovery() &&
                (approvalController.signal.aborted ||
                  currentChat.status !== "running"))
            ) {
              return;
            }
            const evidence = parseCallApiResult(result, permission.operationId);
            const attemptId = evidence.execution?.attemptId;
            if (
              !attemptId ||
              !currentStore.getApiAttemptToolResult(chatId, attemptId)
            )
              currentStore.addMessage({
                chatId,
                role: "tool",
                toolCallId: pendingAsk.toolCallId,
                content: result,
                metadata: {
                  route: currentPage.path,
                  runId: chat.runId ?? undefined,
                },
              });
            if (supportsAgentRunRecovery()) {
              if (!(await currentStore.flushRunPersistence())) {
                currentStore.setChatStatus(chatId, "error", {
                  lastError: getRunUpdateError(
                    t("agent.token-budget.action-failed")
                  ),
                });
                return;
              }
              if (approvalController.signal.aborted || currentChat.archived)
                return;
            }
            if (
              !requestAvailable ||
              evidence.error === "Permission request expired." ||
              evidence.error === "Permission request not found or already used."
            ) {
              currentStore.addMessage({
                chatId,
                role: "assistant",
                content: t("agent.api-permission-expired"),
                metadata: {
                  route: currentPage.path,
                  runId: chat.runId ?? undefined,
                },
              });
              currentStore.setFollowUpQueuePaused(chatId, true);
              currentStore.finishChatRun(chatId);
              setContinuationFailure(null);
              if (supportsAgentRunRecovery())
                await currentStore.flushRunPersistence();
              return;
            }
            if (currentChat.archived) {
              currentStore.finishChatRun(chatId);
              return;
            }
            if (
              !confirmed &&
              supportsAgentRunRecovery() &&
              !getAgentRunContinuation(
                currentChat,
                currentStore.getMessages(chatId)
              )
            ) {
              currentStore.setFollowUpQueuePaused(chatId, true);
              currentStore.finishChatRun(chatId);
              setContinuationFailure(null);
              if (!(await currentStore.flushRunPersistence())) {
                currentStore.setChatStatus(chatId, "error", {
                  lastError: getRunUpdateError(
                    t("agent.token-budget.action-failed")
                  ),
                });
              }
              return;
            }
            // An approved write can make the selected resource unavailable.
            // Resume from its recorded result; the executor still checks further access.
            if (
              supportsAgentRunRecovery() &&
              !(await currentStore.flushRunPersistence())
            )
              return;
            await startChatRun(
              chatId,
              currentPage,
              runMode,
              undefined,
              true,
              "continue",
              lease
            );
          });
        } catch (error) {
          if (!supportsAgentRunRecovery()) throw error;
          if (
            isCurrentChatScope(store, chat.id) &&
            useAgentStore.getState().getChat(chat.id)?.runId === chat.runId
          ) {
            useAgentStore.getState().setChatStatus(chat.id, "error", {
              lastError: getRunUpdateError(
                t("agent.token-budget.action-failed"),
                error
              ),
            });
          }
        } finally {
          if (
            approvalController &&
            store.getAbortController(chat.id) === approvalController
          ) {
            store.setAbortController(chat.id, null);
          }
          confirmationInFlightRef.current = false;
        }
        return;
      }

      const currentPage = getCurrentPageSnapshot();
      const thread = store.ensureCurrentChat(currentPage);
      const chatId = thread.id;
      const answer = confirmed ? confirmLabel : cancelLabel;

      if (await blockWhenAIIsNotConfigured(chatId)) return;
      const currentStore = useAgentStore.getState();
      if (
        currentStore.getChat(chatId)?.archived ||
        currentStore.getChat(chatId)?.status !== "awaiting_user" ||
        currentStore.getPendingAsk(chatId)?.toolCallId !== pendingAsk.toolCallId
      ) {
        return;
      }
      await withChatRunSegment(chatId, async (lease) => {
        if (!canContinueChatRun(chatId)) return;
        currentStore.clearError(chatId);
        const answered = currentStore.answerPendingAsk(
          chatId,
          { kind: "confirm", answer, confirmed },
          {
            route: currentPage.path,
            runId: currentStore.getChat(chatId)?.runId ?? undefined,
          }
        );
        if (!answered) return;
        await startChatRun(
          chatId,
          currentPage,
          currentStore.getChat(chatId)?.runMode ?? "default",
          undefined,
          false,
          "continue",
          lease
        );
      });
    },
    [
      blockWhenAIIsNotConfigured,
      canContinueChatRun,
      confirmLabel,
      cancelLabel,
      startChatRun,
      t,
    ]
  );

  const submitChoice = useCallback(async () => {
    const store = useAgentStore.getState();
    const pendingAsk = selectCurrentPendingAsk(store);
    const chat = selectCurrentChat(store);
    if (
      !chat ||
      chat.archived ||
      !selectedChoice ||
      needsChoiceInput ||
      hasAdditionalMention ||
      !choiceSelection ||
      choiceSelection.chatId !== chat.id ||
      choiceSelection.toolCallId !== pendingAsk?.toolCallId ||
      !pendingAsk ||
      pendingAsk.kind !== "choose" ||
      store.isChatRunning(chat?.id)
    ) {
      return;
    }

    const currentPage = getCurrentPageSnapshot();
    const chatId = chat.id;

    if (await blockWhenAIIsNotConfigured(chatId)) return;
    const currentStore = useAgentStore.getState();
    if (
      !isCurrentChatScope(store, chatId) ||
      currentStore.currentChatId !== chatId ||
      currentStore.getChat(chatId)?.archived ||
      currentStore.getChat(chatId)?.status !== "awaiting_user" ||
      currentStore.getPendingAsk(chatId) !== pendingAsk
    ) {
      return;
    }
    await withChatRunSegment(chatId, async (lease) => {
      if (!canContinueChatRun(chatId)) return;
      currentStore.clearError(chatId);
      const references = isCustomChoice
        ? normalizeAgentResourceReferences(
            draftMentions.map(({ resource }) => resource)
          )
        : [];
      const answered = currentStore.answerPendingAsk(
        chatId,
        {
          kind: "choose",
          answer: selectedChoice.label,
          value: selectedChoice.value,
          ...(selectedChoice.kind === "custom-input"
            ? { input: input.trim() }
            : {}),
        },
        {
          route: currentPage.path,
          runId: currentStore.getChat(chatId)?.runId ?? undefined,
          ...(references[0] ? { resourceReference: references[0] } : {}),
          ...(references.length > 1 ? { resourceReferences: references } : {}),
        }
      );
      if (!answered) return;
      setChoiceSelection(null);
      await startChatRun(
        chatId,
        currentPage,
        currentStore.getChat(chatId)?.runMode ?? "default",
        undefined,
        false,
        "continue",
        lease
      );
    });
  }, [
    blockWhenAIIsNotConfigured,
    canContinueChatRun,
    choiceSelection,
    draftMentions,
    hasAdditionalMention,
    input,
    isCustomChoice,
    needsChoiceInput,
    selectedChoice,
    startChatRun,
  ]);

  const handleTextareaKeydown = useCallback(
    (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.nativeEvent.isComposing || event.nativeEvent.keyCode === 229)
        return;
      if (resourcePickerOpen && canMentionResource) {
        const range = getMentionTriggerRange(
          input,
          event.currentTarget.selectionStart
        );
        if (!range) {
          setResourcePickerOpen(false);
          mentionRangeRef.current = null;
        } else if (event.key === "Escape") {
          event.preventDefault();
          handleResourcePickerOpenChange(false);
        } else if (
          event.key === "Enter" &&
          !event.shiftKey &&
          !event.altKey &&
          !event.ctrlKey &&
          !event.metaKey
        ) {
          event.preventDefault();
        }
        if (range) return;
      }
      if (isAgentSendShortcut(event.nativeEvent, sendWithEnter)) {
        event.preventDefault();
        if (isCustomChoice) void submitChoice();
        else void send();
      }
    },
    [
      canMentionResource,
      handleResourcePickerOpenChange,
      input,
      isCustomChoice,
      resourcePickerOpen,
      send,
      sendWithEnter,
      submitChoice,
    ]
  );

  // ---------------------------------------------------------------------------
  // Render
  // ---------------------------------------------------------------------------

  const issueDecisionPermissionText = (impact: IssueDecisionImpact["kind"]) => {
    switch (impact) {
      case "reject":
        return t("agent.permission.issue-rejection");
      case "final-approval-with-rollout":
        return t("agent.permission.issue-approval-rollout");
      case "final-approval-with-access-grant":
        return t("agent.permission.issue-approval-access-grant");
      case "final-approval-with-role-grant":
        return t("agent.permission.issue-approval-role-grant");
      case "final-approval":
        return t("agent.permission.issue-approval-final");
      default:
        return t("agent.permission.issue-approval");
    }
  };

  const inputFooter = (
    <AgentChatboxFooter
      context={context}
      contextTitle={contextTitle}
      workspace={workspace}
      contextDisabled={
        loading ||
        Boolean(currentPendingAsk) ||
        currentChat?.status === "paused"
      }
      onSelectContext={selectContext}
      chat={currentChat}
      pendingAttemptId={pendingAttemptId}
    />
  );

  const chatbox = (
    <div data-agent-input-row>
      <AgentChatbox
        containerRef={composerRef}
        footer={inputFooter}
        processing={isCurrentChatRunning}
        placeholder={`${inputPlaceholder} · ${sendWithEnter ? t("agent.input-keyboard-hint") : t("agent.input-keyboard-hint-newline")}`}
        onSend={() => void (isCustomChoice ? submitChoice() : send())}
        sendDisabled={isSendDisabled}
        sendLabel={sendLabel}
        allowQueue={!currentPendingAsk && !remoteBusy}
        onStop={() => {
          const store = useAgentStore.getState();
          if (currentChat) store.setFollowUpQueuePaused(currentChat.id, true);
          store.cancel(currentChat?.id, "user");
        }}
        stopDisabled={remoteBusy}
        textareaProps={{
          textareaRef,
          id: isCustomChoice ? choiceInputId : undefined,
          value: input,
          mentions: draftMentions.map(({ resource, range }) => ({
            range,
            label: resource.label,
          })),
          onDeleteMention: ({ text, caret }) => {
            selectedMentionCaretRef.current = caret;
            mentionRangeRef.current = null;
            setResourcePickerOpen(false);
            setInput(text);
          },
          "aria-label": isCustomChoice
            ? t("agent.custom-input-label")
            : inputPlaceholder,
          disabled:
            remoteBusy ||
            (isCurrentChatRunning && Boolean(currentPendingAsk)) ||
            isAIConfigurationBlocked,
          onChange: (event) => {
            setInput(event.target.value);
            if (!canMentionResource) {
              mentionRangeRef.current = null;
              setResourcePickerOpen(false);
              return;
            }
            const range = getMentionTriggerRange(
              event.target.value,
              event.target.selectionStart
            );
            mentionRangeRef.current = range;
            setResourcePickerOpen(Boolean(range));
          },
          onKeyDown: handleTextareaKeydown,
        }}
      >
        {canMentionResource && workspace?.name ? (
          <AgentResourcePicker
            key={
              isCustomChoice
                ? `${currentPendingAsk?.toolCallId}:${selectedChoice.value}`
                : "composer"
            }
            open={resourcePickerOpen}
            onOpenChange={handleResourcePickerOpenChange}
            anchorRef={textareaRef}
            workspaceName={workspace.name}
            projectName={isCustomChoice ? askProject : projectContext}
            selectedProjectName={
              isCustomChoice && askProject
                ? askProject
                : selectedProjectMention?.resource.name
            }
            hasSelectedReference={
              draftMentions.length > 1 ||
              (draftMentions.length === 1 && !selectedProjectMention)
            }
            onSelectProject={selectMentionProject}
            onSelectDatabase={selectMentionDatabase}
            onSelectIssue={selectMentionIssue}
            onSelectPlan={selectMentionPlan}
          />
        ) : null}
      </AgentChatbox>
    </div>
  );

  return (
    <div
      ref={composerRef}
      className="flex max-h-full min-h-0 flex-col gap-3 border-t p-3"
    >
      <AgentFollowUpQueueList
        queue={followUpQueue}
        disabled={remoteBusy}
        resumeDisabled={
          !executionAvailable ||
          Boolean(storagePressure) ||
          isAIConfigurationBlocked ||
          isCurrentChatRunning ||
          Boolean(currentController) ||
          localLeasePending ||
          Boolean(currentPendingAsk) ||
          currentChat?.status === "paused"
        }
        onRemove={(messageId) => {
          if (currentChatId)
            useAgentStore.getState().removeFollowUp(currentChatId, messageId);
        }}
        onResume={() => {
          if (currentChatId)
            void drainFollowUps(currentChatId, undefined, true);
        }}
      />
      {feedback === "execution-unavailable" && (
        <Alert appearance="caption" variant="warning">
          <div className="min-w-0">
            {t("agent.token-budget.recovery-unavailable")}
            <AgentErrorDetails error={currentChat?.lastError} />
          </div>
        </Alert>
      )}
      {feedback === "continuation-unavailable" && currentChat && (
        <Alert
          appearance="caption"
          variant="warning"
          role="alert"
          showIcon={false}
        >
          <Collapsible className="min-w-0">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <div className="flex min-w-0 max-w-full items-start gap-1">
                <AlertTriangle
                  size={14}
                  className="shrink-0"
                  aria-hidden="true"
                />
                <span className="min-w-0 break-words">
                  {t("agent.token-budget.continuation-unavailable")}
                </span>
              </div>
              {currentChat.lastError && (
                <CollapsibleTrigger
                  render={<Button size="xs" appearance="secondary" />}
                  className="group w-auto gap-x-1.5"
                  data-agent-error-details
                >
                  <ChevronRight
                    size={14}
                    className="shrink-0 group-aria-expanded:rotate-90"
                    aria-hidden="true"
                  />
                  {t("error-page.error-details")}
                </CollapsibleTrigger>
              )}
              {currentContinuationFailure?.endFailed && (
                <span>{t("agent.token-budget.action-failed")}</span>
              )}
            </div>
            {currentChat.lastError && (
              <AgentErrorDetailsPanel error={currentChat.lastError} />
            )}
          </Collapsible>
        </Alert>
      )}
      {feedback === "token-budget" && (
        <AgentTokenBudgetStatus
          chat={currentChat}
          error={currentChat?.lastError}
          pendingAttemptId={pendingAttemptId}
          disabled={!executionAvailable || endingRun}
          remoteBusy={remoteBusy}
          canResume={Boolean(
            !storagePressure &&
              currentChat &&
              getAgentRunContinuation(
                currentChat,
                useAgentStore.getState().getMessages(currentChat.id)
              )
          )}
          onIncreaseLimit={async (limit) => {
            if (!currentChat) return false;
            const store = useAgentStore.getState();
            if (store.getAbortController(currentChat.id))
              return store.increaseRunTokenLimit(currentChat.id, limit);
            let increased = false;
            await withChatRunSegment(currentChat.id, async () => {
              increased = await useAgentStore
                .getState()
                .increaseRunTokenLimit(currentChat.id, limit);
            });
            return increased;
          }}
          onResume={async () => {
            if (
              !currentChat ||
              useAgentStore.getState().storagePressure ||
              (await blockWhenAIIsNotConfigured(currentChat.id))
            )
              return false;
            let resumed = false;
            await withChatRunSegment(currentChat.id, async (lease) => {
              const store = useAgentStore.getState();
              store.syncState();
              const latest = useAgentStore.getState();
              const chat = latest.getChat(currentChat.id);
              if (
                !chat ||
                chat.runId !== currentChat.runId ||
                latest.getAbortController(chat.id) ||
                chat.runCheckpoint?.suspension !== "token-limit" ||
                !getAgentRunContinuation(chat, latest.getMessages(chat.id))
              )
                return;
              resumed = true;
              await startChatRun(
                chat.id,
                chat.page ?? getCurrentPageSnapshot(),
                chat.runMode ?? "default",
                undefined,
                true,
                "continue",
                lease
              );
            });
            return resumed;
          }}
          onEndRun={endCurrentRun}
        />
      )}
      <div className="flex min-h-0 min-w-0 flex-col">
        {/* Pending ask banner */}
        {currentPendingAsk && (
          <div
            className="mb-3 min-h-0 overflow-y-auto rounded-xs bg-warning/10 px-3 py-2 text-xs text-warning"
            data-agent-pending-preview
          >
            {apiPermission ? (
              <>
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="font-medium">
                    {apiPermissionPreview
                      ? t("agent.api-permission-title")
                      : t("agent.api-permission-expired-title")}
                  </div>
                  {apiPermissionPreview && (
                    <span role="timer" className="tabular-nums">
                      {t("agent.api-permission-expires-in", {
                        time: formatPermissionTime(
                          apiPermissionPreview.expiresAt
                        ),
                      })}
                    </span>
                  )}
                </div>
                {apiPermissionPreview ? (
                  apiPermissionPreview.dom ? (
                    <div className="mt-2 flex min-w-0 flex-col gap-2 text-control">
                      <div className="font-medium">
                        {apiPermissionPreview.dom.kind === "database-labels"
                          ? t("agent.permission.database-labels")
                          : t("agent.permission.database-environment")}
                      </div>
                      <div className="min-w-0 font-mono wrap-anywhere">
                        {apiPermissionPreview.dom.resourceName}
                      </div>
                      {Boolean(
                        apiPermissionPreview.priorAttemptIds?.length
                      ) && (
                        <Alert variant="warning" appearance="caption">
                          {t("agent.api-recovery.repeat-risk")}
                        </Alert>
                      )}
                      <ul
                        className="flex min-w-0 flex-col gap-3"
                        aria-label={
                          apiPermissionPreview.dom.kind === "database-labels"
                            ? t("agent.permission.database-labels")
                            : t("agent.permission.database-environment")
                        }
                      >
                        {apiPermissionPreview.dom.changes.map((change) => (
                          <li key={change.field} className="min-w-0">
                            <div className="font-mono font-medium wrap-anywhere">
                              {apiPermissionPreview.dom?.kind ===
                              "database-environment"
                                ? t("common.environment")
                                : change.field}
                            </div>
                            <dl className="mt-1 grid min-w-0 grid-cols-2 gap-2">
                              {(["before", "after"] as const).map((side) => (
                                <div key={side} className="min-w-0">
                                  <dt className="text-control-light">
                                    {side === "before"
                                      ? t("agent.permission.before")
                                      : t("agent.permission.after")}
                                  </dt>
                                  <dd className="whitespace-pre-wrap font-mono wrap-anywhere">
                                    {change[side] === null
                                      ? t("agent.permission.not-set")
                                      : change[side] === ""
                                        ? t("common.empty")
                                        : change[side]}
                                  </dd>
                                </div>
                              ))}
                            </dl>
                          </li>
                        ))}
                      </ul>
                      <details className="min-w-0">
                        <summary className="cursor-pointer rounded-xs focus-visible:outline-2 focus-visible:outline-accent">
                          {t("agent.permission.exact-request")}
                        </summary>
                        <div className="mt-2 font-mono wrap-anywhere">
                          {apiPermissionPreview.operationId}
                        </div>
                        <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap font-mono wrap-anywhere">
                          {JSON.stringify(
                            JSON.parse(apiPermissionPreview.body),
                            null,
                            2
                          )}
                        </pre>
                      </details>
                    </div>
                  ) : (
                    <>
                      <div className="mt-1">
                        {t("agent.api-permission-request")}
                      </div>
                      <div className="mt-2 font-mono break-all">
                        {apiPermissionPreview.operationId}
                      </div>
                      {Boolean(
                        apiPermissionPreview.priorAttemptIds?.length
                      ) && (
                        <Alert
                          variant="warning"
                          appearance="caption"
                          className="mt-2"
                        >
                          {t("agent.api-recovery.repeat-risk")}
                        </Alert>
                      )}
                      {apiPermissionPreview.issueDecision && (
                        <div className="mt-2 border-l-2 border-warning pl-2">
                          <div className="font-medium">
                            {apiPermissionPreview.issueDecision.issueTitle}
                          </div>
                          <div className="font-mono break-all text-control-light">
                            {apiPermissionPreview.issueDecision.issueName}
                          </div>
                          <div className="mt-1">
                            {issueDecisionPermissionText(
                              apiPermissionPreview.issueDecision.impact
                            )}
                          </div>
                        </div>
                      )}
                      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all rounded-xs border bg-background p-2 font-mono text-control">
                        {JSON.stringify(
                          JSON.parse(apiPermissionPreview.body),
                          null,
                          2
                        )}
                      </pre>
                    </>
                  )
                ) : (
                  <div className="mt-1">
                    {t("agent.api-permission-expired")}
                  </div>
                )}
              </>
            ) : (
              <>
                <div className="font-medium">{currentPendingAsk.prompt}</div>
                <div className="mt-1">
                  {currentPendingAsk.kind === "confirm"
                    ? t("agent.pending-confirm-hint")
                    : currentPendingAsk.kind === "choose"
                      ? t("agent.pending-choose-hint")
                      : t("agent.pending-input-hint")}
                </div>
                {canCancelPendingRequest && (
                  <AgentActionGroup className="mt-2">
                    <Button
                      appearance="secondary"
                      size="sm"
                      className="rounded-xs border text-control hover:bg-control-bg"
                      onClick={cancelPendingRequest}
                    >
                      {t("agent.cancel-request")}
                    </Button>
                  </AgentActionGroup>
                )}
              </>
            )}
          </div>
        )}

        {isAwaitingConfirm && (
          <div className="mb-1 shrink-0">{inputFooter}</div>
        )}

        {/* Confirm buttons */}
        {isAwaitingConfirm ? (
          <AgentActionGroup
            className="shrink-0"
            data-agent-confirmation-actions
          >
            <Button
              appearance="secondary"
              size="sm"
              className="h-auto min-h-7 min-w-0 max-w-full rounded-xs border py-1 text-control whitespace-normal wrap-anywhere hover:bg-control-bg disabled:opacity-50"
              disabled={isCurrentChatRunning || localLeasePending}
              onClick={() => submitConfirmation(false)}
            >
              {apiPermission
                ? apiPermissionPreview
                  ? t("agent.api-permission-deny")
                  : t("agent.api-permission-dismiss")
                : cancelLabel}
            </Button>
            <Button
              appearance="secondary"
              size="sm"
              className="h-auto min-h-7 min-w-0 max-w-full rounded-xs bg-accent py-1 text-accent-text whitespace-normal wrap-anywhere hover:bg-accent-hover disabled:opacity-50"
              disabled={
                isCurrentChatRunning ||
                localLeasePending ||
                (apiPermission && !apiPermissionPreview)
              }
              onClick={() => submitConfirmation(true)}
            >
              {apiPermission ? t("agent.api-permission-allow") : confirmLabel}
            </Button>
          </AgentActionGroup>
        ) : isAwaitingChoose ? (
          /* Choose options */
          <div className="flex min-w-0 flex-col gap-3">
            <RadioGroup
              aria-label={currentPendingAsk.prompt}
              className="max-h-48 flex-col items-stretch gap-2 overflow-y-auto"
              value={selectedChoice?.value ?? null}
              disabled={isCurrentChatRunning || isAIConfigurationBlocked}
              onValueChange={(value) => {
                if (currentChatId && typeof value === "string") {
                  choiceValueRef.current = value;
                  setResourcePickerOpen(false);
                  mentionRangeRef.current = null;
                  selectedMentionCaretRef.current = null;
                  setChoiceSelection((previous) => ({
                    chatId: currentChatId,
                    toolCallId: currentPendingAsk.toolCallId,
                    value,
                    inputs:
                      previous?.chatId === currentChatId &&
                      previous.toolCallId === currentPendingAsk.toolCallId
                        ? previous.inputs
                        : new Map(),
                  }));
                }
              }}
            >
              {chooseOptions.map((option) => (
                <RadioGroupItem
                  key={option.value}
                  value={option.value}
                  className="items-start rounded-xs p-2 hover:bg-control-bg"
                  contentClassName="min-w-0 break-words"
                  radioClassName="mt-0.5"
                >
                  <div className="font-medium text-main">{option.label}</div>
                  {option.description && (
                    <div className="text-xs text-control-light">
                      {option.description}
                    </div>
                  )}
                </RadioGroupItem>
              ))}
            </RadioGroup>
            {isCustomChoice ? (
              <FormField>
                <FormLabel htmlFor={choiceInputId}>
                  {t("agent.custom-input-label")}
                </FormLabel>
                {chatbox}
              </FormField>
            ) : (
              <div className="flex min-w-0 items-center gap-2">
                {inputFooter}
                <Button
                  size="sm"
                  className="shrink-0"
                  disabled={
                    !selectedChoice ||
                    needsChoiceInput ||
                    isCurrentChatRunning ||
                    isAIConfigurationBlocked
                  }
                  onClick={() => void submitChoice()}
                >
                  <Send className="size-4" />
                  {t("agent.reply")}
                </Button>
              </div>
            )}
          </div>
        ) : (
          chatbox
        )}
      </div>
    </div>
  );
}
