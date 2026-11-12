import { Send, Square } from "lucide-react";
import { type ReactNode, type RefObject, useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { AgentChat } from "../logic/types";
import { AgentComposer } from "./AgentComposer";
import { AgentContextPicker } from "./AgentContextPicker";
import {
  AgentMentionTextarea,
  type AgentMentionTextareaProps,
} from "./AgentMentionTextarea";
import { AgentTokenUsage } from "./AgentTokenUsage";

interface AgentChatboxProps {
  containerRef?: RefObject<HTMLDivElement | null>;
  textareaProps: AgentMentionTextareaProps;
  placeholder: string;
  processing?: boolean;
  footer: ReactNode;
  onSend: () => void;
  sendDisabled: boolean;
  sendLabel?: string;
  onStop?: () => void;
  stopDisabled?: boolean;
  children?: ReactNode;
}

interface AgentChatboxFooterProps {
  context?: string;
  contextTitle?: string;
  workspace?: { name: string; title: string };
  contextDisabled: boolean;
  onSelectContext: (context: string) => boolean;
  chat: AgentChat | null;
  pendingAttemptId?: string;
  showEmptyUsage?: boolean;
}

export function AgentChatbox({
  containerRef,
  textareaProps,
  placeholder,
  processing = false,
  footer,
  onSend,
  sendDisabled,
  sendLabel,
  onStop,
  stopDisabled = false,
  children,
}: AgentChatboxProps) {
  const { t } = useTranslation();
  const { textareaRef, value } = textareaProps;
  const autoResize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;

    el.style.height = "auto";

    const styles = window.getComputedStyle(el);
    const minHeight =
      Number.parseFloat(styles.minHeight) || el.getBoundingClientRect().height;
    const borderHeight = el.offsetHeight - el.clientHeight;
    const desiredHeight = Math.max(minHeight, el.scrollHeight + borderHeight);
    const composer =
      containerRef?.current ?? el.closest<HTMLElement>("[data-agent-composer]");
    const parent = composer?.parentElement;
    // Flex containers can shrink while their textarea and footer still overflow.
    const composerChrome = composer
      ? Math.max(
          0,
          Math.max(
            composer.offsetHeight,
            composer.scrollHeight +
              composer.offsetHeight -
              composer.clientHeight
          ) - el.offsetHeight
        )
      : 0;
    const pixels = (value: string) => Number.parseFloat(value) || 0;
    const siblings = parent
      ? Array.from(parent.children).filter((child) => {
          if (child === composer) return false;
          const style = window.getComputedStyle(child);
          return (
            style.display !== "none" &&
            style.position !== "absolute" &&
            style.position !== "fixed"
          );
        })
      : [];
    const siblingSpace = siblings.reduce((height, child) => {
      const style = window.getComputedStyle(child);
      return (
        height +
        Math.max(
          pixels(style.minHeight),
          pixels(style.paddingTop) +
            pixels(style.paddingBottom) +
            pixels(style.borderTopWidth) +
            pixels(style.borderBottomWidth)
        ) +
        pixels(style.marginTop) +
        pixels(style.marginBottom)
      );
    }, 0);
    const parentStyle = parent ? window.getComputedStyle(parent) : null;
    const parentSpace = parentStyle
      ? pixels(parentStyle.paddingTop) +
        pixels(parentStyle.paddingBottom) +
        pixels(parentStyle.rowGap) * siblings.length
      : 0;
    const availableHeight = parent
      ? parent.clientHeight - composerChrome - siblingSpace - parentSpace
      : 0;
    const maxHeight =
      availableHeight > 0
        ? Math.max(minHeight, availableHeight)
        : desiredHeight;
    const nextHeight = Math.min(desiredHeight, maxHeight);

    el.style.height = `${nextHeight}px`;
    el.style.overflowY = desiredHeight > maxHeight ? "auto" : "hidden";
  }, [containerRef, textareaRef]);

  useEffect(() => {
    autoResize();
  }, [value, autoResize]);

  useEffect(() => {
    const composer =
      containerRef?.current ??
      textareaRef.current?.closest<HTMLElement>("[data-agent-composer]");
    const parent = composer?.parentElement;
    if (!composer || !parent || typeof ResizeObserver === "undefined") return;

    const observer = new ResizeObserver(autoResize);
    observer.observe(composer);
    observer.observe(parent);
    return () => observer.disconnect();
  }, [autoResize, containerRef, textareaRef]);

  return (
    <AgentComposer
      footer={footer}
      action={
        processing ? (
          <Button
            variant="destructive"
            size="sm"
            className="shrink-0 whitespace-nowrap"
            disabled={stopDisabled}
            onClick={onStop}
          >
            <Square className="size-4" />
            {t("agent.stop")}
          </Button>
        ) : (
          <Button
            size="sm"
            className="shrink-0 whitespace-nowrap"
            data-agent-send
            disabled={sendDisabled}
            onClick={onSend}
          >
            <Send className="size-4" />
            {sendLabel ?? t("agent.send")}
          </Button>
        )
      }
    >
      <div className="relative w-full">
        {!textareaProps.value && (
          <div
            className="pointer-events-none absolute inset-x-3 inset-y-0 z-10 flex items-center truncate text-sm leading-5 text-control-light"
            data-agent-input-placeholder
          >
            {processing ? t("agent.input-processing-placeholder") : placeholder}
          </div>
        )}
        <AgentMentionTextarea
          {...textareaProps}
          size="md"
          rows={1}
          placeholder={undefined}
          className={cn(
            "block h-9 min-h-9 w-full resize-none overflow-y-hidden border-0 focus:border-0 focus:ring-0 disabled:bg-transparent",
            textareaProps.className
          )}
        />
        {children}
      </div>
    </AgentComposer>
  );
}

export function AgentChatboxFooter({
  context,
  contextTitle,
  workspace,
  contextDisabled,
  onSelectContext,
  chat,
  pendingAttemptId,
  showEmptyUsage = false,
}: AgentChatboxFooterProps) {
  return (
    <div
      className="flex min-w-0 flex-1 flex-wrap items-center justify-between gap-2 text-xs text-control-light"
      data-agent-input-footer
    >
      <AgentContextPicker
        context={context}
        contextTitle={contextTitle}
        workspace={workspace}
        disabled={contextDisabled}
        onSelect={onSelectContext}
      />
      <AgentTokenUsage
        key={chat?.id}
        chat={chat}
        pendingAttemptId={pendingAttemptId}
        showEmpty={showEmptyUsage}
      />
    </div>
  );
}
