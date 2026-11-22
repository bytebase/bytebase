import { Play, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import type { QueuedUserMessage } from "../store/agent";
import { AgentTooltip } from "./ui/AgentTooltip";

interface AgentFollowUpQueueListProps {
  queue:
    | { messages: readonly QueuedUserMessage[]; paused: boolean }
    | undefined;
  disabled?: boolean;
  resumeDisabled?: boolean;
  onRemove: (messageId: string) => void;
  onResume: () => void;
}

export function AgentFollowUpQueueList({
  queue,
  disabled,
  resumeDisabled,
  onRemove,
  onResume,
}: AgentFollowUpQueueListProps) {
  const { t } = useTranslation();
  if (!queue?.messages.length) return null;

  const count = t("agent.follow-up-queue.queued-count", {
    count: queue.messages.length,
  });
  const resume = t("agent.follow-up-queue.resume");

  return (
    <div className="flex min-w-0 shrink-0 flex-col gap-1 px-1">
      <div className="flex min-h-6 items-center justify-between gap-2 px-2">
        <div
          role="status"
          className="flex flex-wrap items-center gap-2 text-xs text-control-light"
        >
          <span>{count}</span>
          {queue.paused && (
            <span className="text-control-light">
              {t("agent.follow-up-queue.paused")}
            </span>
          )}
        </div>
        {queue.paused && (
          <AgentTooltip content={resume}>
            <Button
              size="sm"
              appearance="secondary"
              className="shrink-0 text-control-light"
              aria-label={resume}
              disabled={disabled || resumeDisabled}
              onClick={onResume}
            >
              <Play className="size-4" aria-hidden="true" />
            </Button>
          </AgentTooltip>
        )}
      </div>
      <ol
        aria-label={count}
        className="flex max-h-24 flex-col gap-1 overflow-y-auto"
        role="list"
      >
        {queue.messages.map((message, index) => {
          const remove = t("agent.follow-up-queue.remove", {
            position: index + 1,
          });
          return (
            <li
              key={message.id}
              className="flex min-h-8 shrink-0 items-center gap-2 rounded-xs pl-2 transition-colors hover:bg-control-bg-hover focus-within:bg-control-bg-hover motion-reduce:transition-none"
            >
              <div className="grid min-w-0 flex-1 grid-cols-1">
                <AgentTooltip
                  content={
                    <span className="block max-h-48 overflow-y-auto wrap-anywhere whitespace-pre-wrap">
                      {message.text}
                    </span>
                  }
                >
                  <span
                    tabIndex={0}
                    className="min-w-0 truncate rounded-xs text-sm text-main focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-accent"
                  >
                    {message.text}
                  </span>
                </AgentTooltip>
              </div>
              <AgentTooltip content={remove}>
                <Button
                  size="sm"
                  appearance="secondary"
                  className="shrink-0 text-control-light hover:text-error focus-visible:text-error"
                  aria-label={remove}
                  disabled={disabled}
                  onClick={() => onRemove(message.id)}
                >
                  <Trash2 className="size-4" aria-hidden="true" />
                </Button>
              </AgentTooltip>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
