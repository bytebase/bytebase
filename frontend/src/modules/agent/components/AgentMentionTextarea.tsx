import {
  type CSSProperties,
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useState,
} from "react";
import { Textarea, type TextareaProps } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import {
  type AgentMentionRange,
  deleteResourceMention,
} from "../logic/resourceMention";

type AgentMentionTextareaProps = Omit<TextareaProps, "ref" | "value"> & {
  value: string;
  mention?: AgentMentionRange | null;
  mentionLabel?: string;
  onDeleteMention?: (result: { text: string; caret: number }) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
};

export function AgentMentionTextarea({
  value,
  mention,
  mentionLabel,
  onDeleteMention,
  textareaRef,
  className,
  onKeyDown,
  onScroll,
  ...props
}: AgentMentionTextareaProps) {
  const [mirrorStyle, setMirrorStyle] = useState<CSSProperties>({});
  const syncMirror = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea || !mention) return;
    const style = window.getComputedStyle(textarea);
    setMirrorStyle({
      position: "absolute",
      top: style.borderTopWidth,
      left: style.borderLeftWidth,
      width: textarea.clientWidth,
      boxSizing: "border-box",
      paddingTop: style.paddingTop,
      paddingRight: style.paddingRight,
      paddingBottom: style.paddingBottom,
      paddingLeft: style.paddingLeft,
      fontFamily: style.fontFamily,
      fontSize: style.fontSize,
      fontWeight: style.fontWeight,
      fontStyle: style.fontStyle,
      lineHeight: style.lineHeight,
      letterSpacing: style.letterSpacing,
      textAlign: style.textAlign as CSSProperties["textAlign"],
      tabSize: style.tabSize,
      whiteSpace: "pre-wrap",
      overflowWrap: "break-word",
      wordBreak: "break-word",
      transform: `translate(${-textarea.scrollLeft}px, ${-textarea.scrollTop}px)`,
    });
  }, [mention, textareaRef]);

  useLayoutEffect(syncMirror, [syncMirror, value]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea || !mention || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(syncMirror);
    observer.observe(textarea);
    return () => observer.disconnect();
  }, [mention, syncMirror, textareaRef]);

  return (
    <div className="relative w-full">
      {mention && (
        <div
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute inset-0 overflow-hidden text-main",
            props.disabled && "opacity-50"
          )}
        >
          <div style={mirrorStyle}>
            {value.slice(0, mention.start)}
            <span
              className="rounded-xs bg-accent/10 text-accent"
              data-agent-inline-mention
            >
              {value.slice(mention.start, mention.end)}
            </span>
            {value.slice(mention.end)}
          </div>
        </div>
      )}
      {/* Keep native text editing and selection; the mirror paints only the mention. */}
      <Textarea
        {...props}
        ref={textareaRef}
        value={value}
        className={cn(
          className,
          mention &&
            "relative bg-transparent text-transparent caret-main disabled:bg-transparent"
        )}
        onKeyDown={(event) => {
          if (
            !event.nativeEvent.isComposing &&
            (event.key === "Backspace" || event.key === "Delete") &&
            mention &&
            mentionLabel &&
            onDeleteMention
          ) {
            const result = deleteResourceMention(
              value,
              mention,
              mentionLabel,
              event.currentTarget.selectionStart,
              event.currentTarget.selectionEnd,
              event.key
            );
            if (result) {
              event.preventDefault();
              onDeleteMention(result);
              return;
            }
          }
          onKeyDown?.(event);
        }}
        onScroll={(event) => {
          syncMirror();
          onScroll?.(event);
        }}
      />
    </div>
  );
}
