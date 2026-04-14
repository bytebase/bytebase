import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { AgentMessage, ToolCall } from "../logic/types";
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

export function AgentChat({ className }: AgentChatProps) {
  const { t } = useTranslation();
  const chatContainerRef = useRef<HTMLDivElement>(null);

  const messages = useAgentStore(selectMessages);
  const loading = useAgentStore(selectLoading);
  const error = useAgentStore(selectError);
  const showAIConfigurationRecovery = useAgentStore(
    selectCurrentChatRequiresAIConfiguration
  );
  const currentChatId = useAgentStore((s) => s.currentChatId);
  const allMessages = useAgentStore(selectMessages);
  const clearError = useAgentStore((s) => s.clearError);

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
    router.push({
      name: SETTING_ROUTE_WORKSPACE_GENERAL,
      hash: "#ai-assistant",
    });
  }

  function dismiss() {
    clearError(currentChatId);
  }

  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop =
        chatContainerRef.current.scrollHeight;
    }
  }, [currentChatId, messages.length]);

  return (
    <div
      ref={chatContainerRef}
      className={`flex flex-col overflow-y-auto gap-3 p-3 ${className ?? ""}`}
    >
      {displayMessages.map((msg) =>
        msg.role === "user" ? (
          <div key={msg.id} className="flex justify-end">
            <div className="max-w-[80%] rounded-lg bg-blue-50 px-3 py-2 text-sm">
              {msg.content}
            </div>
          </div>
        ) : (
          <div key={msg.id} className="flex flex-col gap-y-2">
            {msg.content && (
              <div className="max-w-[80%] rounded-lg bg-control-bg px-3 py-2 text-sm">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    p: ({ children }) => (
                      <p className="my-1 first:mt-0 last:mb-0">{children}</p>
                    ),
                    pre: ({ children }) => (
                      <pre className="my-1 overflow-x-auto rounded-xs bg-control-bg p-2 text-xs">
                        {children}
                      </pre>
                    ),
                    code: ({ children }) => (
                      <code className="break-all rounded bg-control-bg-hover px-1 text-xs">
                        {children}
                      </code>
                    ),
                    ul: ({ children }) => (
                      <ul className="my-1 list-disc pl-5">{children}</ul>
                    ),
                    ol: ({ children }) => (
                      <ol className="my-1 list-decimal pl-5">{children}</ol>
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
                        className="text-accent underline"
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
                      <table className="my-1 border-collapse text-xs">
                        {children}
                      </table>
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
              </div>
            )}
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
        )
      )}

      {loading && (
        <div className="flex items-center gap-x-2 text-sm text-control-placeholder">
          <span className="animate-pulse">&#9679;</span> {t("common.loading")}
        </div>
      )}

      {showAIConfigurationRecovery ? (
        <div className="max-w-[80%] rounded-lg border border-amber-200 bg-amber-50 px-3 py-3 text-sm text-amber-900">
          <div className="font-medium">
            {t("agent.ai-not-configured.title")}
          </div>
          <div className="mt-1">{t("agent.ai-not-configured.description")}</div>
          <div className="mt-3 flex flex-wrap gap-x-2 gap-y-2">
            {allowConfigure && (
              <Button variant="default" size="sm" onClick={goConfigure}>
                {t("agent.ai-not-configured.configure")}
              </Button>
            )}
            <Button variant="outline" size="sm" onClick={dismiss}>
              {t("common.dismiss")}
            </Button>
          </div>
          {!allowConfigure && (
            <div className="mt-1 text-amber-800">
              {t("agent.ai-not-configured.contact-admin")}
            </div>
          )}
        </div>
      ) : (
        error && (
          <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-error">
            {error}
          </div>
        )
      )}
    </div>
  );
}
