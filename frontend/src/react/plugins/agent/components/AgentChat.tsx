import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { AgentMessage } from "../logic/types";
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
    toolCallId: string
  ): string | undefined {
    const fullIndex = allMessages.findIndex(
      (message) => message.id === messageId
    );
    if (fullIndex < 0) return undefined;

    for (let index = fullIndex + 1; index < allMessages.length; index++) {
      const message = allMessages[index];
      if (message.role === "tool" && message.toolCallId === toolCallId) {
        return message.content;
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
      className={`overflow-y-auto space-y-3 p-3 ${className ?? ""}`}
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
              <div className="max-w-[80%] rounded-lg bg-gray-50 px-3 py-2 text-sm">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    p: ({ children }) => (
                      <p className="my-1 first:mt-0 last:mb-0">{children}</p>
                    ),
                    pre: ({ children }) => (
                      <pre className="my-1 overflow-x-auto rounded bg-gray-100 p-2 text-xs">
                        {children}
                      </pre>
                    ),
                    code: ({ children }) => (
                      <code className="rounded bg-gray-200 px-1 text-xs">
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
                        className="text-blue-600 underline"
                        href={href}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {children}
                      </a>
                    ),
                    blockquote: ({ children }) => (
                      <blockquote className="my-1 border-l-2 border-gray-300 pl-2 text-gray-600">
                        {children}
                      </blockquote>
                    ),
                    table: ({ children }) => (
                      <table className="my-1 border-collapse text-xs">
                        {children}
                      </table>
                    ),
                    th: ({ children }) => (
                      <th className="border border-gray-300 px-2 py-1">
                        {children}
                      </th>
                    ),
                    td: ({ children }) => (
                      <td className="border border-gray-300 px-2 py-1">
                        {children}
                      </td>
                    ),
                  }}
                >
                  {msg.content}
                </ReactMarkdown>
              </div>
            )}
            {msg.toolCalls?.map((toolCall) => (
              <ToolCallCard
                key={toolCall.id}
                toolCall={toolCall}
                result={getToolResult(msg.id, toolCall.id)}
              />
            ))}
          </div>
        )
      )}

      {loading && (
        <div className="flex items-center gap-x-2 text-sm text-gray-400">
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
          <div className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-600">
            {error}
          </div>
        )
      )}
    </div>
  );
}
