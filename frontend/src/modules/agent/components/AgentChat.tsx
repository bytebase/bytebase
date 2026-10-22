import { ChevronRight } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
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
  AI_ASSISTANT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import { useAppStore } from "@/stores/app";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { getAgentAIConfigurationEnabled } from "../logic/aiConfiguration";
import type {
  AgentEvidence,
  AgentEvidenceReference,
  AgentMessage,
  ToolCall,
} from "../logic/types";
import {
  selectCurrentChatRequiresAIConfiguration,
  selectError,
  selectLoading,
  selectMessages,
  useAgentStore,
} from "../store/agent";
import { ToolCallCard } from "./ToolCallCard";

interface AgentChatProps {
  className?: string;
}

const ISSUE_NAME_PATTERN = /^projects\/([^/]+)\/issues\/([^/]+)$/;
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
        name: typeof PROJECT_V1_ROUTE_DATABASE_DETAIL;
        params: { projectId: string; instanceId: string; databaseName: string };
        query: { parent: string };
      };
};

function resolveEvidenceReferences(
  evidence: AgentEvidence | undefined
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

function EvidenceReferences({ evidence }: { evidence?: AgentEvidence }) {
  const { t } = useTranslation();
  const references = resolveEvidenceReferences(evidence);
  const labels = {
    issue: t("agent.evidence.issue"),
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

export function AgentChat({ className }: AgentChatProps) {
  const { t } = useTranslation();
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const [
    isAIConfigurationRecoveryDismissed,
    setAIConfigurationRecoveryDismissed,
  ] = useState(false);

  const messages = useAgentStore(selectMessages);
  const loading = useAgentStore(selectLoading);
  const error = useAgentStore(selectError);
  const chatRequiresAIConfiguration = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);
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
          message.role === "user" || message.role === "assistant"
      ),
    [messages]
  );

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
      chatContainerRef.current.scrollTop =
        chatContainerRef.current.scrollHeight;
    }
  }, [currentChatId, messages.length]);

  useEffect(() => {
    setAIConfigurationRecoveryDismissed(false);
  }, [aiConfigurationEnabled, currentChatId]);

  return (
    <div
      ref={chatContainerRef}
      className={`flex flex-col overflow-y-auto gap-3 p-3 ${className ?? ""}`}
    >
      {displayMessages.map((msg) => {
        const isPending = msg.metadata?.pending === true;
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
            <HumanizeTs tsMs={msg.createdTs} tooltip={false} />
          </time>
        );

        return (
          <div key={msg.id}>
            {msg.role === "user" ? (
              <div className="flex flex-col items-end">
                <div className="w-fit max-w-full rounded-sm bg-info/10 px-3 py-2 text-sm">
                  {msg.content}
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
                    <div className="w-fit max-w-full rounded-sm bg-control-bg px-3 py-2 text-sm">
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
          <div className="rounded-sm bg-error/10 px-3 py-2 text-sm text-error">
            {error}
          </div>
        )
      )}
    </div>
  );
}
