import {
  ArrowUpRight,
  Building2,
  ChevronRight,
  Folder,
  Loader2,
} from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  SETTING_ROUTE_WORKSPACE_GENERAL,
} from "@/app/router/handles";
import { HumanizeTs } from "@/components/HumanizeTs";
import { RouterLink } from "@/components/RouterLink";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Table } from "@/components/ui/table";
import {
  ExistingIssueCommentDraftError,
  IssueCommentDraftPermissionError,
  IssueCommentDraftStorageError,
  prepareIssueCommentDraftRoute,
  wasIssueCommentDraftDelivered,
} from "@/lib/issueCommentDraft";
import {
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { projectNamePrefix } from "@/lib/resourceName";
import {
  ACCESS_GRANT_DRAFT_QUERY_KEY,
  wasDeniedQueryRequestDraftDelivered,
} from "@/modules/sql-editor/model/accessGrantDraft";
import { useAppStore } from "@/stores/app";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { getAgentAIConfigurationEnabled } from "../logic/aiConfiguration";
import {
  type DeniedQueryRequestDraft,
  DeniedQueryRequestDraftStorageError,
  prepareAgentDeniedQueryRequestRoute,
} from "../logic/deniedQueryAccess";
import { getAgentErrorSummary } from "../logic/errorDisplay";
import { resolveAgentHandoffRoute } from "../logic/handoff";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  PlanDraftStorageError,
  prepareAgentPlanDraftRoute,
} from "../logic/planDraft";
import {
  formatResourceMention,
  isResourceMentionRange,
  normalizeAgentResourceReferences,
} from "../logic/resourceMention";
import { normalizeAgentResponseActions } from "../logic/responseActions";
import type {
  AgentAnswerEvidence,
  AgentEvidence,
  AgentEvidenceReference,
  AgentHandoff,
  AgentMessage,
  AgentResourceReference,
  AgentResponseAction,
  ToolCall,
} from "../logic/types";
import {
  selectCurrentChat,
  selectCurrentChatRequiresAIConfiguration,
  selectError,
  selectLoading,
  selectMessages,
  useAgentStore,
} from "../store/agent";
import { ToolCallCard } from "./ToolCallCard";

interface AgentChatProps {
  className?: string;
  contextLabel?: string;
  onRetry?: () => void;
  onSelectSuggestion?: (starterId: LandingStarterId) => void;
  onContinueAction?: (
    chatId: string,
    prompt: string,
    references: AgentResourceReference[]
  ) => void;
}

const EMPTY_CHAT_STARTERS: Record<"workspace" | "project", LandingStarterId[]> =
  {
    workspace: [
      "workspace-attention",
      "workspace-recent",
      "workspace-governance",
    ],
    project: [
      "project-open-blockers",
      "project-open-recent",
      "project-propose-change",
      "project-next",
    ],
  };

const ISSUE_NAME_PATTERN = /^projects\/([^/]+)\/issues\/([^/]+)$/;
const PLAN_NAME_PATTERN = /^projects\/([^/]+)\/plans\/([^/]+)$/;
const DATABASE_NAME_PATTERN =
  /^(?:projects\/([^/]+)\/)?instances\/([^/]+)\/databases\/([^/]+)$/;

type ResolvedEvidenceReference = {
  reference: AgentEvidenceReference;
  label: string;
  to:
    | {
        name: typeof PROJECT_V1_ROUTE_ISSUE_DETAIL;
        params: { projectId: string; issueId: string };
      }
    | {
        name: typeof PROJECT_V1_ROUTE_PLAN_DETAIL;
        params: { projectId: string; planId: string };
      }
    | {
        name: typeof PROJECT_V1_ROUTE_DATABASE_DETAIL;
        params: { projectId: string; instanceId: string; databaseName: string };
        query: { parent: string };
      };
};

function resolveScopedEvidenceReferences(
  evidence: AgentEvidence
): ResolvedEvidenceReference[] {
  if (!evidence || !Array.isArray(evidence.references)) {
    return [];
  }

  const projectMatch =
    typeof evidence.project === "string"
      ? evidence.project.match(/^projects\/([^/]+)$/)
      : undefined;
  const isWorkspaceScoped =
    typeof evidence.workspace === "string" &&
    /^workspaces\/[^/]+$/.test(evidence.workspace);
  if ((projectMatch ? 1 : 0) + (isWorkspaceScoped ? 1 : 0) !== 1) {
    return [];
  }

  const projectId = projectMatch?.[1];
  const resolved: ResolvedEvidenceReference[] = [];
  for (const reference of evidence.references) {
    if (!reference || typeof reference.name !== "string") {
      continue;
    }
    if (reference.kind === "issue") {
      const match = reference.name.match(ISSUE_NAME_PATTERN);
      if (
        !match ||
        (projectId && match[1] !== projectId) ||
        typeof reference.title !== "string" ||
        !reference.title
      ) {
        continue;
      }
      resolved.push({
        reference,
        label: reference.title,
        to: {
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: { projectId: match[1], issueId: match[2] },
        },
      });
      continue;
    }

    if (reference.kind === "plan") {
      if (
        !projectId ||
        typeof reference.title !== "string" ||
        !reference.title
      ) {
        continue;
      }
      const match = reference.name.match(PLAN_NAME_PATTERN);
      if (!match || match[1] !== projectId) continue;
      resolved.push({
        reference,
        label: reference.title,
        to: {
          name: PROJECT_V1_ROUTE_PLAN_DETAIL,
          params: { projectId: match[1], planId: match[2] },
        },
      });
      continue;
    }

    if (reference.kind !== "database" || !projectId) continue;

    const match = reference.name.match(DATABASE_NAME_PATTERN);
    const referenceProjectId = match?.[1];
    if (!match || (referenceProjectId && referenceProjectId !== projectId)) {
      continue;
    }
    const parent = referenceProjectId
      ? `projects/${referenceProjectId}/instances/${match[2]}`
      : `instances/${match[2]}`;
    resolved.push({
      reference,
      label: match[3],
      to: {
        name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
        params: {
          projectId,
          instanceId: match[2],
          databaseName: match[3],
        },
        query: { parent },
      },
    });
  }
  return resolved;
}

function resolveEvidenceReferences(
  evidence: AgentAnswerEvidence | undefined
): ResolvedEvidenceReference[] {
  const groups = Array.isArray(evidence)
    ? evidence
    : evidence
      ? [evidence]
      : [];
  const seen = new Set<string>();
  return groups.flatMap((group) =>
    resolveScopedEvidenceReferences(group).filter(({ reference }) => {
      const key = `${reference.kind}:${reference.name}`;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    })
  );
}

function EvidenceReferences({ evidence }: { evidence?: AgentAnswerEvidence }) {
  const { t } = useTranslation();
  const references = resolveEvidenceReferences(evidence);
  const labels = {
    issue: t("agent.evidence.issue"),
    plan: t("agent.evidence.plan"),
    database: t("agent.evidence.database"),
  };
  if (references.length === 0) return null;

  return (
    <Collapsible className="mt-2">
      <CollapsibleTrigger className="w-fit text-xs text-control-light hover:text-control">
        <ChevronRight className="size-3" />
        {t("agent.evidence.references")}
      </CollapsibleTrigger>
      <CollapsiblePanel className="pt-1">
        <ul className="flex flex-col gap-y-1 pl-5 text-xs">
          {references.map(({ reference, label, to }) => (
            <li key={`${reference.kind}:${reference.name}`}>
              <span className="text-control-light">
                {labels[reference.kind]}:{" "}
              </span>
              <RouterLink className="normal-link" to={to}>
                {label}
              </RouterLink>
            </li>
          ))}
        </ul>
      </CollapsiblePanel>
    </Collapsible>
  );
}

function HandoffLink({ handoff }: { handoff?: AgentHandoff }) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const to = resolveAgentHandoffRoute(handoff);
  if (!to || !handoff) return null;
  const labelKey =
    handoff.kind === "transfer-project-databases"
      ? "project.home.transfer-databases"
      : handoff.kind === "connect-project-instance"
        ? "project.home.connect"
        : handoff.kind === "create-project-database"
          ? "project.home.create-database"
          : `agent.handoff.${handoff.kind}`;
  if (
    handoff.kind === "transfer-project-databases" ||
    handoff.kind === "connect-project-instance" ||
    handoff.kind === "create-project-database"
  ) {
    const openSetup = async () => {
      if (loading) return;
      setLoading(true);
      try {
        const targetPath = router.resolve(to).fullPath;
        await router.push(to);
        const currentPath = router.currentRoute.value.fullPath;
        if (
          (currentPath === targetPath ||
            sameRoutePathname(currentPath, targetPath)) &&
          useAgentStore.getState().visible
        ) {
          useAgentStore.getState().minimize();
        }
      } catch {
        useAppStore.getState().notify({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      } finally {
        setLoading(false);
      }
    };
    return (
      <Button
        appearance="link"
        className="h-auto px-0 py-0"
        data-agent-handoff
        data-agent-action-open
        disabled={loading}
        onClick={() => void openSetup()}
        size="sm"
      >
        {t(labelKey)}
        <ArrowUpRight className="size-3.5" aria-hidden="true" />
      </Button>
    );
  }
  return (
    <RouterLink
      data-agent-handoff
      data-agent-action-open
      to={to}
      className="inline-flex items-center gap-1 text-sm text-accent hover:underline"
    >
      {t(labelKey)}
      <ArrowUpRight className="size-3.5" aria-hidden="true" />
    </RouterLink>
  );
}

function reachedIssueCommentDraftRoute(
  currentPath: string,
  targetPath: string
): boolean {
  if (currentPath === targetPath) return true;
  if (!targetPath.startsWith("/") || !currentPath.startsWith("/")) return false;
  const target = new URL(targetPath, "https://bytebase.invalid");
  const current = new URL(currentPath, "https://bytebase.invalid");
  const targetIssue = /^\/projects\/([^/]+)\/issues\/[^/]+$/.exec(
    target.pathname
  );
  const draftKey = target.searchParams.get("agentCommentDraft");
  return Boolean(
    targetIssue &&
      draftKey &&
      current.searchParams.get("agentCommentDraft") === draftKey &&
      current.pathname.startsWith(`/projects/${targetIssue[1]}/plans/`)
  );
}

function sameRoutePathname(currentPath: string, targetPath: string): boolean {
  if (!currentPath.startsWith("/") || !targetPath.startsWith("/")) return false;
  return (
    new URL(currentPath, "https://bytebase.invalid").pathname ===
    new URL(targetPath, "https://bytebase.invalid").pathname
  );
}

async function waitForDeliveredAccessDraft(key: string): Promise<boolean> {
  for (let attempt = 0; attempt < 60; attempt++) {
    if (wasDeniedQueryRequestDraftDelivered(key)) return true;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  return false;
}

async function waitForDeliveredIssueCommentDraft(
  key: string
): Promise<boolean> {
  for (let attempt = 0; attempt < 60; attempt++) {
    if (wasIssueCommentDraftDelivered(key)) return true;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  return false;
}

function PlanDraftButton({
  action,
  answer,
}: {
  action: Extract<AgentResponseAction, { kind: "draft-plan" }>;
  answer: string;
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const openDraft = async () => {
    if (loading) return;
    setLoading(true);
    try {
      const route = await prepareAgentPlanDraftRoute(
        action,
        answer,
        t("agent.plan-draft.default-title")
      );
      const targetPath = router.resolve(route).fullPath;
      await router.push(route);
      if (
        router.currentRoute.value.fullPath === targetPath &&
        useAgentStore.getState().visible
      ) {
        useAgentStore.getState().minimize();
      }
    } catch (error) {
      const title =
        error instanceof PlanDraftStorageError
          ? t("plan.initial-sql-storage-failed")
          : t("agent.plan-draft.unavailable");
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Button
      appearance="link"
      size="sm"
      className="h-auto px-0 py-0"
      data-agent-action-draft-plan
      disabled={loading}
      onClick={() => void openDraft()}
    >
      {loading ? (
        <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
      ) : (
        <ArrowUpRight className="size-3.5" aria-hidden="true" />
      )}
      {t("agent.plan-draft.open")}
    </Button>
  );
}

function AccessRequestDraftButton({
  draft,
}: {
  draft: DeniedQueryRequestDraft;
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const openDraft = async () => {
    if (loading) return;
    setLoading(true);
    try {
      const route = await prepareAgentDeniedQueryRequestRoute(draft);
      const targetPath = router.resolve(route).fullPath;
      await router.push(route);
      const key =
        typeof route === "string"
          ? undefined
          : route.query?.[ACCESS_GRANT_DRAFT_QUERY_KEY];
      const delivered =
        typeof key === "string" &&
        sameRoutePathname(router.currentRoute.value.fullPath, targetPath) &&
        (await waitForDeliveredAccessDraft(key));
      if (!delivered) {
        useAppStore.getState().notify({
          module: "bytebase",
          style: "CRITICAL",
          title: t("agent.access-grant-draft.unavailable"),
        });
      } else if (useAgentStore.getState().visible) {
        useAgentStore.getState().minimize();
      }
    } catch (error) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title:
          error instanceof DeniedQueryRequestDraftStorageError
            ? t("agent.access-grant-draft.storage-failed")
            : t("agent.access-grant-draft.unavailable"),
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Button
      appearance="link"
      size="sm"
      className="h-auto px-0 py-0"
      data-agent-action-request-access
      disabled={loading}
      onClick={() => void openDraft()}
    >
      {loading ? (
        <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
      ) : (
        <ArrowUpRight className="size-3.5" aria-hidden="true" />
      )}
      {t("agent.access-grant-draft.open")}
    </Button>
  );
}

function IssueCommentDraftButton({
  action,
}: {
  action: Extract<AgentResponseAction, { kind: "draft-issue-comment" }>;
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const openDraft = async () => {
    if (loading) return;
    setLoading(true);
    try {
      const route = prepareIssueCommentDraftRoute(action.issue, action.comment);
      const targetPath = router.resolve(route).fullPath;
      await router.push(route);
      const draftKey = route.query?.agentCommentDraft;
      if (
        typeof draftKey !== "string" ||
        !reachedIssueCommentDraftRoute(
          router.currentRoute.value.fullPath,
          targetPath
        )
      ) {
        return;
      }
      if (!(await waitForDeliveredIssueCommentDraft(draftKey))) {
        useAppStore.getState().notify({
          module: "bytebase",
          style: "CRITICAL",
          title: t("agent.issue-comment-draft.unavailable"),
        });
        return;
      }
      if (
        reachedIssueCommentDraftRoute(
          router.currentRoute.value.fullPath,
          targetPath
        ) &&
        useAgentStore.getState().visible
      ) {
        useAgentStore.getState().minimize();
      }
    } catch (error) {
      const title =
        error instanceof ExistingIssueCommentDraftError
          ? t("agent.issue-comment-draft.existing")
          : error instanceof IssueCommentDraftPermissionError
            ? t("agent.issue-comment-draft.permission-denied")
            : error instanceof IssueCommentDraftStorageError
              ? t("agent.issue-comment-draft.storage-failed")
              : t("agent.issue-comment-draft.unavailable");
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Button
        appearance="link"
        size="sm"
        className="h-auto px-0 py-0"
        data-agent-action-draft-issue-comment
        disabled={loading}
        onClick={() => void openDraft()}
      >
        {loading ? (
          <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
        ) : (
          <ArrowUpRight className="size-3.5" aria-hidden="true" />
        )}
        {t("agent.issue-comment-draft.open")}
      </Button>
      <details className="mt-1 text-xs text-control-light">
        <summary className="cursor-pointer">
          {t("agent.issue-comment-draft.preview")}
        </summary>
        <p className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap text-main">
          {action.comment}
        </p>
      </details>
    </div>
  );
}

function renderBoundMentions(
  content: string,
  references: AgentResourceReference[]
): ReactNode {
  const labels = new Map<string, { label: string; count: number }>();
  for (const reference of references) {
    const mention = formatResourceMention(reference.label);
    const existing = labels.get(mention);
    labels.set(mention, {
      label: reference.label,
      count: (existing?.count ?? 0) + 1,
    });
  }

  const ranges: { start: number; end: number }[] = [];
  for (const [mention, { label, count }] of labels) {
    const matches: { start: number; end: number }[] = [];
    let from = 0;
    while (from < content.length) {
      const start = content.indexOf(mention, from);
      if (start < 0) break;
      const range = { start, end: start + mention.length };
      if (isResourceMentionRange(content, range, label)) matches.push(range);
      from = range.end;
    }
    if (matches.length === count) ranges.push(...matches);
  }

  ranges.sort((a, b) => a.start - b.start);
  if (ranges.length === 0) return content;

  const parts: ReactNode[] = [];
  let from = 0;
  for (const range of ranges) {
    if (range.start < from) continue;
    parts.push(content.slice(from, range.start));
    parts.push(
      <span
        key={range.start}
        className="font-medium text-accent"
        data-agent-message-mention
      >
        {content.slice(range.start, range.end)}
      </span>
    );
    from = range.end;
  }
  parts.push(content.slice(from));
  return parts;
}

export function AgentChat({
  className,
  contextLabel,
  onRetry,
  onSelectSuggestion,
  onContinueAction,
}: AgentChatProps) {
  const { t } = useTranslation();
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const [
    isAIConfigurationRecoveryDismissed,
    setAIConfigurationRecoveryDismissed,
  ] = useState(false);

  const messages = useAgentStore(selectMessages);
  const currentChat = useAgentStore(selectCurrentChat);
  const loading = useAgentStore(selectLoading);
  const error = useAgentStore(selectError);
  const chatRequiresAIConfiguration = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const queuedUserMessage = useAgentStore((s) =>
    s.currentChatId ? s.queuedUserMessagesByChatId[s.currentChatId] : undefined
  );
  const latestContinuePrompts = messages
    .at(-1)
    ?.metadata?.actions?.filter((action) => action.kind === "continue")
    .map((action) => action.prompt)
    .join("\0");
  const previousFollowUp = useRef<{
    chatId: string | null;
    prompts?: string;
  }>({ chatId: null });
  const draftText = useAgentStore((s) =>
    currentChatId ? s.draftsByChatId[currentChatId]?.text : undefined
  );
  const aiConfigurationEnabled = useAppStore((s) =>
    getAgentAIConfigurationEnabled(s.getSettingByName(Setting_SettingName.AI))
  );
  const allMessages = useAgentStore(selectMessages);
  const clearError = useAgentStore((s) => s.clearError);

  const showAIConfigurationRecovery =
    chatRequiresAIConfiguration ||
    (aiConfigurationEnabled === false && !isAIConfigurationRecoveryDismissed);

  const allowConfigure = hasWorkspacePermissionV2("bb.settings.set");

  const displayMessages = useMemo(
    () =>
      messages.filter(
        (message): message is AgentMessage =>
          (message.role === "user" || message.role === "assistant") &&
          !(
            message.role === "assistant" &&
            error &&
            message.metadata?.error === error
          )
      ),
    [error, messages]
  );
  const emptyChatScope = currentChat?.context?.startsWith(projectNamePrefix)
    ? "project"
    : "workspace";
  const showEmptyChat =
    currentChat?.status === "idle" &&
    !currentChat.archived &&
    messages.length === 0 &&
    !loading &&
    !error &&
    !showAIConfigurationRecovery;
  // An interruption belongs after the turn's final visible message, including tool calls.
  const interruptionByLastMessageId = useMemo(() => {
    const interruptions = new Map<
      string,
      { reason: "user" | "unexpected"; userMessageId: string }
    >();
    let userMessage: AgentMessage | undefined;
    let lastMessage: AgentMessage | undefined;
    const addInterruption = () => {
      const reason = userMessage?.metadata?.interruptionReason;
      if (
        userMessage &&
        lastMessage &&
        (reason === "user" || reason === "unexpected")
      ) {
        interruptions.set(lastMessage.id, {
          reason,
          userMessageId: userMessage.id,
        });
      }
    };
    for (const message of displayMessages) {
      if (message.role === "user") {
        addInterruption();
        userMessage = message;
      }
      lastMessage = message;
    }
    addInterruption();
    return interruptions;
  }, [displayMessages]);
  const latestUserMessageId = displayMessages.findLast(
    (message) => message.role === "user"
  )?.id;

  function getToolResult(
    messageId: string,
    toolCallId: string,
    duplicateIndex = 0
  ): string | undefined {
    const fullIndex = allMessages.findIndex(
      (message) => message.id === messageId
    );
    if (fullIndex < 0) return undefined;

    let matchingResultIndex = 0;
    for (let index = fullIndex + 1; index < allMessages.length; index++) {
      const message = allMessages[index];
      if (message.role === "tool" && message.toolCallId === toolCallId) {
        if (matchingResultIndex === duplicateIndex) {
          return message.content;
        }
        matchingResultIndex++;
      }
      if (
        message.role === "assistant" &&
        message.content &&
        !message.toolCalls?.length
      ) {
        break;
      }
    }
    return undefined;
  }

  function getToolCallDuplicateIndex(
    toolCalls: ToolCall[],
    currentToolCallIndex: number
  ): number {
    const currentToolCall = toolCalls[currentToolCallIndex];
    let duplicateIndex = 0;
    for (let index = 0; index < currentToolCallIndex; index++) {
      if (toolCalls[index]?.id === currentToolCall.id) {
        duplicateIndex++;
      }
    }
    return duplicateIndex;
  }

  function goConfigure() {
    clearError(currentChatId);
    useAgentStore.getState().toggle();
  }

  function dismiss() {
    setAIConfigurationRecoveryDismissed(true);
    clearError(currentChatId);
  }

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = showEmptyChat
        ? 0
        : chatContainerRef.current.scrollHeight;
    }
  }, [currentChatId, currentChat?.interrupted, messages.length, showEmptyChat]);

  useEffect(() => {
    const previous = previousFollowUp.current;
    previousFollowUp.current = {
      chatId: currentChatId,
      prompts: latestContinuePrompts,
    };
    if (
      previous.chatId !== currentChatId ||
      !latestContinuePrompts ||
      previous.prompts === latestContinuePrompts
    ) {
      return;
    }
    const container = chatContainerRef.current;
    if (
      container &&
      container.scrollHeight - container.scrollTop - container.clientHeight <=
        160
    ) {
      container.scrollTop = container.scrollHeight;
    }
  }, [currentChatId, latestContinuePrompts]);

  useEffect(() => {
    setAIConfigurationRecoveryDismissed(false);
  }, [aiConfigurationEnabled, currentChatId]);

  return (
    <div
      ref={chatContainerRef}
      className={`flex flex-col overflow-y-auto gap-3 p-3 ${className ?? ""}`}
    >
      {showEmptyChat && (
        <div className="m-auto w-full max-w-xl py-2" data-agent-empty-chat>
          <div className="flex min-w-0 items-center gap-2 text-sm font-medium text-control-light">
            {emptyChatScope === "project" ? (
              <Folder className="size-4 shrink-0" aria-hidden="true" />
            ) : (
              <Building2 className="size-4 shrink-0" aria-hidden="true" />
            )}
            <span className="min-w-0 truncate">
              {contextLabel ??
                currentChat.context ??
                t("agent.context-workspace")}
            </span>
          </div>
          <h2 className="mt-3 text-xl font-semibold text-main">
            {emptyChatScope === "project"
              ? t("agent.empty-chat.project-heading")
              : t("agent.empty-chat.workspace-heading")}
          </h2>
          {!draftText?.trim() && (
            <>
              <div className="mt-4 text-xs font-medium text-control-light">
                {t("agent.empty-chat.suggestions")}
              </div>
              <div className="mt-2 divide-y divide-block-border border-y border-block-border">
                {EMPTY_CHAT_STARTERS[emptyChatScope].map((starterId) => (
                  <Button
                    key={starterId}
                    appearance="secondary"
                    size="md"
                    className="h-auto w-full min-w-0 justify-between rounded-xs px-3 py-2 text-left whitespace-normal hover:text-accent"
                    onClick={() => onSelectSuggestion?.(starterId)}
                    data-agent-empty-suggestion
                    data-starter-id={starterId}
                  >
                    <span className="min-w-0 flex-1 break-words">
                      {t(`agent.landing-starters.${starterId}`)}
                    </span>
                    <ArrowUpRight className="shrink-0" aria-hidden="true" />
                  </Button>
                ))}
              </div>
            </>
          )}
        </div>
      )}
      {displayMessages.map((msg) => {
        const interruption = interruptionByLastMessageId.get(msg.id);
        const canRetry =
          interruption?.reason === "unexpected" &&
          interruption.userMessageId === latestUserMessageId &&
          currentChat?.interrupted &&
          currentChat.interruptionReason === "unexpected" &&
          !loading &&
          onRetry;
        const isPending = msg.metadata?.pending === true;
        const actions = normalizeAgentResponseActions(msg.metadata?.actions);
        const openAction = actions.find((action) => action.kind === "open");
        const planDraftAction = actions.find(
          (action) => action.kind === "draft-plan"
        );
        const issueCommentDraftAction = actions.find(
          (action) => action.kind === "draft-issue-comment"
        );
        const boundReferences = normalizeAgentResourceReferences(
          msg.metadata?.resourceReferences,
          msg.metadata?.resourceReference
        );
        const continueActions = actions.filter(
          (action) => action.kind === "continue"
        );
        const references = msg.role === "user" ? boundReferences : [];
        const content = msg.content ?? "";
        const projectReference = references.find(
          (reference) => reference.kind === "project"
        );
        const showTimestamp =
          msg.role === "user" ||
          (msg.role === "assistant" &&
            !isPending &&
            !!msg.content &&
            !msg.toolCalls?.length);
        const timestamp = (
          <time
            className="mt-1 text-xs text-control-light"
            dateTime={new Date(msg.createdTs).toISOString()}
            data-agent-message-timestamp
          >
            <HumanizeTs tsMs={msg.createdTs} mode="calendar" tooltip={false} />
          </time>
        );

        return (
          <div key={msg.id}>
            {msg.role === "user" ? (
              <div className="flex flex-col items-end">
                <div className="flex w-fit max-w-full flex-col gap-y-1 rounded-sm bg-info/10 px-3 py-2 text-sm">
                  <span>{renderBoundMentions(content, references)}</span>
                  {projectReference &&
                    !content.includes(
                      formatResourceMention(projectReference.label)
                    ) && (
                      <span
                        className="text-xs text-control-light"
                        data-agent-resource-reference
                      >
                        {t("common.project")}: {projectReference.label}
                      </span>
                    )}
                </div>
                {showTimestamp && timestamp}
              </div>
            ) : (
              <div className="flex flex-col">
                <div className="flex flex-col gap-y-2">
                  {isPending && msg.content ? (
                    <div
                      className="flex items-center gap-x-2 text-sm text-control-placeholder"
                      data-agent-pending-message
                    >
                      <span className="animate-pulse">&#9679;</span>
                      {msg.content}
                    </div>
                  ) : msg.content ? (
                    <div
                      className="w-fit max-w-full rounded-sm bg-control-bg px-3 py-2 text-sm"
                      data-agent-answer
                    >
                      <ReactMarkdown
                        remarkPlugins={[remarkGfm]}
                        components={{
                          p: ({ children }) => (
                            <p className="my-1 first:mt-0 last:mb-0">
                              {children}
                            </p>
                          ),
                          pre: ({ children }) => (
                            <pre className="my-1 overflow-x-auto rounded-xs bg-control-bg p-2 text-xs">
                              {children}
                            </pre>
                          ),
                          code: ({ children }) => (
                            <code className="break-all rounded-xs bg-control-bg-hover px-1 text-xs">
                              {children}
                            </code>
                          ),
                          ul: ({ children }) => (
                            <ul className="my-1 list-disc pl-5">{children}</ul>
                          ),
                          ol: ({ children }) => (
                            <ol className="my-1 list-decimal pl-5">
                              {children}
                            </ol>
                          ),
                          li: ({ children }) => (
                            <li className="my-0.5">{children}</li>
                          ),
                          h1: ({ children }) => (
                            <h1 className="my-1 font-semibold">{children}</h1>
                          ),
                          h2: ({ children }) => (
                            <h2 className="my-1 font-semibold">{children}</h2>
                          ),
                          h3: ({ children }) => (
                            <h3 className="my-1 font-semibold">{children}</h3>
                          ),
                          a: ({ children, href }) => (
                            <a
                              className="break-all text-accent underline"
                              href={href}
                              target="_blank"
                              rel="noopener noreferrer"
                            >
                              {children}
                            </a>
                          ),
                          blockquote: ({ children }) => (
                            <blockquote className="my-1 border-l-2 border-control-border pl-2 text-control-light">
                              {children}
                            </blockquote>
                          ),
                          table: ({ children }) => (
                            <Table className="my-1 border-collapse text-xs">
                              {children}
                            </Table>
                          ),
                          th: ({ children }) => (
                            <th className="border border-control-border px-2 py-1">
                              {children}
                            </th>
                          ),
                          td: ({ children }) => (
                            <td className="border border-control-border px-2 py-1">
                              {children}
                            </td>
                          ),
                        }}
                      >
                        {msg.content}
                      </ReactMarkdown>
                      <EvidenceReferences evidence={msg.metadata?.evidence} />
                      {!msg.metadata?.pending &&
                        !msg.metadata?.error &&
                        (openAction ||
                          planDraftAction ||
                          issueCommentDraftAction ||
                          msg.metadata?.accessRequestDraft ||
                          msg.metadata?.handoff) && (
                          <div className="mt-2">
                            {planDraftAction ? (
                              <PlanDraftButton
                                action={planDraftAction}
                                answer={msg.content}
                              />
                            ) : issueCommentDraftAction ? (
                              <IssueCommentDraftButton
                                action={issueCommentDraftAction}
                              />
                            ) : msg.metadata?.accessRequestDraft ? (
                              <AccessRequestDraftButton
                                draft={msg.metadata.accessRequestDraft}
                              />
                            ) : (
                              <HandoffLink
                                handoff={
                                  openAction?.target ?? msg.metadata?.handoff
                                }
                              />
                            )}
                          </div>
                        )}
                    </div>
                  ) : null}
                  {msg.toolCalls?.map((toolCall, index, toolCalls) => (
                    <ToolCallCard
                      key={`${msg.id}:${toolCall.id}:${index}`}
                      toolCall={toolCall}
                      result={getToolResult(
                        msg.id,
                        toolCall.id,
                        getToolCallDuplicateIndex(toolCalls, index)
                      )}
                    />
                  ))}
                </div>
                {showTimestamp && timestamp}
                {msg.content &&
                  !msg.metadata?.pending &&
                  !msg.metadata?.error &&
                  continueActions.length > 0 && (
                    <div
                      className="mt-3 flex max-w-full flex-col items-start gap-2"
                      role="group"
                      aria-label={t("agent.follow-up.ask-next")}
                      data-agent-follow-up
                    >
                      <div className="text-xs font-medium text-control-light">
                        {t("agent.follow-up.ask-next")}
                      </div>
                      {continueActions.map((action) => (
                        <Button
                          key={action.prompt}
                          type="button"
                          appearance="secondary"
                          size="md"
                          className="h-auto max-w-full justify-start bg-control-bg px-3 py-2 text-left whitespace-normal text-main hover:bg-control-bg-hover"
                          data-agent-action-continue
                          disabled={
                            !onContinueAction ||
                            currentChat?.status !== "idle" ||
                            currentChat.archived ||
                            Boolean(draftText?.trim()) ||
                            Boolean(queuedUserMessage)
                          }
                          onClick={() =>
                            onContinueAction?.(
                              msg.chatId,
                              action.prompt,
                              boundReferences
                            )
                          }
                        >
                          <span
                            className="min-w-0 break-words"
                            data-agent-follow-up-question
                          >
                            {action.prompt}
                          </span>
                        </Button>
                      ))}
                    </div>
                  )}
              </div>
            )}
            {interruption && (
              <div
                role="status"
                data-agent-interruption-status
                className="mt-3 rounded-sm border border-block-border bg-control-bg px-2 py-1.5 text-xs"
              >
                <div className="flex items-center justify-between gap-2">
                  <span
                    className={`font-medium ${
                      interruption.reason === "user"
                        ? "text-warning"
                        : "text-error"
                    }`}
                  >
                    {interruption.reason === "user"
                      ? t("agent.request-stopped-by-user")
                      : t("agent.interrupted")}
                  </span>
                  {canRetry && (
                    <Button appearance="outline" size="xs" onClick={onRetry}>
                      {t("agent.retry-last-chat-turn")}
                    </Button>
                  )}
                </div>
                <div className="mt-0.5 text-control-light">
                  {interruption.reason === "user"
                    ? t("agent.request-stopped-retry-hint")
                    : t("agent.interrupted-retry-hint")}
                </div>
              </div>
            )}
          </div>
        );
      })}

      {loading &&
        !displayMessages.some((message) => message.metadata?.pending) && (
          <div className="flex items-center gap-x-2 text-sm text-control-placeholder">
            <span className="animate-pulse">&#9679;</span> {t("common.loading")}
          </div>
        )}

      {showAIConfigurationRecovery ? (
        <div className="max-w-[80%] rounded-sm border border-warning/20 bg-warning/10 px-3 py-3 text-sm text-warning">
          <div className="font-medium">
            {t("agent.ai-not-configured.title")}
          </div>
          <div className="mt-1">{t("agent.ai-not-configured.description")}</div>
          <div className="mt-3 flex flex-wrap gap-x-2 gap-y-2">
            {allowConfigure && (
              <RouterLink
                to={{
                  name: SETTING_ROUTE_WORKSPACE_GENERAL,
                  hash: "#ai-assistant",
                  query: {
                    [PRODUCT_INTRO_QUERY_KEY]: AI_ASSISTANT_PRODUCT_INTRO,
                  },
                }}
                className={buttonVariants({ size: "sm" })}
                onClick={goConfigure}
              >
                {t("agent.ai-not-configured.configure")}
              </RouterLink>
            )}
            <Button appearance="outline" size="sm" onClick={dismiss}>
              {t("common.dismiss")}
            </Button>
          </div>
          {!allowConfigure && (
            <div className="mt-1 text-warning">
              {t("agent.ai-not-configured.contact-admin")}
            </div>
          )}
        </div>
      ) : (
        error && (
          <div
            className="rounded-sm bg-error/10 px-3 py-2 text-sm text-error"
            role="alert"
          >
            <div data-agent-error-summary>
              {getAgentErrorSummary(error) ?? t("agent.request-failed")}
            </div>
            <Collapsible className="mt-1">
              <CollapsibleTrigger
                className="w-fit text-xs underline decoration-dotted underline-offset-2"
                data-agent-error-details
              >
                <ChevronRight className="size-3" />
                {t("error-page.error-details")}
              </CollapsibleTrigger>
              <CollapsiblePanel className="pt-1">
                <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-xs">
                  {error}
                </pre>
              </CollapsiblePanel>
            </Collapsible>
          </div>
        )
      )}
    </div>
  );
}
