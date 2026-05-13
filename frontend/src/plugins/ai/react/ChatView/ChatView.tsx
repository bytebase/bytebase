import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import type { Conversation } from "../../types";
import { useAIContext } from "../context";
import { AIMessageView } from "./AIMessageView";
import { ChatViewProvider, type Mode } from "./context";
import { EmptyView } from "./EmptyView";
import { UserMessageView } from "./UserMessageView";

type Props = {
  readonly mode?: Mode;
  readonly conversation?: Conversation;
};

/**
 * React port of `plugins/ai/components/ChatView/ChatView.vue`.
 *
 * Scrollable message list. Auto-scrolls to the bottom whenever the
 * inner container's height changes (a new message arrives, an AI
 * response streams in, etc.) — same `useElementSize` trigger as the
 * Vue version, ported to a `ResizeObserver`.
 *
 * Two empty paths:
 *   - `mode="VIEW"` with a conversation that has no messages → `<EmptyView>`.
 *   - `mode="CHAT"` with no conversation at all → "select or create"
 *     prompt with a clickable Create. The `select-or-create` i18n
 *     string uses a `{create}` interpolation slot; we split manually
 *     because `react-i18next`'s `Trans` v17 wipes child slots on
 *     empty placeholder tags (see SelectionCopyTooltips for the same
 *     fix in Stage 20).
 */
export function ChatView({ mode = "CHAT", conversation }: Props) {
  const { t } = useTranslation();
  const { events } = useAIContext();

  const scrollerRef = useRef<HTMLDivElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    const scroller = scrollerRef.current;
    if (!container || !scroller) return;
    const scrollToBottom = () => {
      scroller.scrollTo(0, container.scrollHeight);
    };
    scrollToBottom();
    const observer = new ResizeObserver(scrollToBottom);
    observer.observe(container);
    return () => observer.disconnect();
  }, [conversation?.id, conversation?.messageList.length]);

  const chatViewValue = useMemo(() => ({ mode }), [mode]);

  // i18n: "Select or {create} a conversation to start." — split on the
  // placeholder so we can render the localized prefix/suffix around a
  // clickable button. Matches every locale's structure since they all
  // keep the same `{create}` placeholder.
  const selectOrCreateTemplate = t("plugin.ai.conversation.select-or-create");
  const selectOrCreateParts = selectOrCreateTemplate.split("{create}");

  return (
    <ChatViewProvider value={chatViewValue}>
      <div ref={scrollerRef} className="flex-1 overflow-y-auto">
        {conversation ? (
          conversation.messageList.length === 0 ? (
            mode === "VIEW" ? (
              <EmptyView />
            ) : null
          ) : (
            <div
              ref={containerRef}
              className="flex flex-col justify-end px-2 gap-y-4"
            >
              {conversation.messageList.map((message) => (
                <div
                  key={message.id}
                  className={`flex message ${
                    message.author === "AI" ? "justify-start" : "justify-end"
                  }`}
                >
                  {message.author === "USER" && (
                    <UserMessageView message={message} />
                  )}
                  {message.author === "AI" && (
                    <AIMessageView message={message} />
                  )}
                </div>
              ))}
            </div>
          )
        ) : mode === "CHAT" ? (
          <div className="w-full h-full flex flex-col justify-end items-center pb-8">
            <p className="text-sm text-control-light">
              {selectOrCreateParts[0]}
              <button
                type="button"
                className="text-accent underline hover:text-accent-hover cursor-pointer"
                onClick={() => events.emit("new-conversation", { input: "" })}
              >
                {t("common.create")}
              </button>
              {selectOrCreateParts[1] ?? ""}
            </p>
          </div>
        ) : null}
      </div>
    </ChatViewProvider>
  );
}
