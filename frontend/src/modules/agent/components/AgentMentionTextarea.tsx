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
  mentions?: { range: AgentMentionRange; label: string }[];
  onDeleteMention?: (result: { text: string; caret: number }) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
};

export function AgentMentionTextarea({
  value,
  mentions = [],
  onDeleteMention,
  textareaRef,
  className,
  onKeyDown,
  onScroll,
  ...props
}: AgentMentionTextareaProps) {
  const [mirrorStyle, setMirrorStyle] = useState<CSSProperties>({});
  const orderedMentions = [...mentions].sort(
    (a, b) => a.range.start - b.range.start
  );
  const syncMirror = useCallback(() => {
    const textarea = textareaRef.current;
    if (!textarea || mentions.length === 0) return;
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
  }, [mentions.length, textareaRef]);

  useLayoutEffect(syncMirror, [syncMirror, value]);

  useEffect(() => {
    const textarea = textareaRef.current;
    if (
      !textarea ||
      mentions.length === 0 ||
      typeof ResizeObserver === "undefined"
    )
      return;
    const observer = new ResizeObserver(syncMirror);
    observer.observe(textarea);
    return () => observer.disconnect();
  }, [mentions.length, syncMirror, textareaRef]);

  return (
    <div className="relative w-full">
      {mentions.length > 0 && (
        <div
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute inset-0 overflow-hidden text-main",
            props.disabled && "opacity-50"
          )}
        >
          <div style={mirrorStyle}>
            {orderedMentions.map(({ range }, index) => (
              <span key={`${range.start}-${range.end}`}>
                {value.slice(
                  index === 0 ? 0 : orderedMentions[index - 1].range.end,
                  range.start
                )}
                <span
                  className="rounded-xs bg-accent/10 text-accent"
                  data-agent-inline-mention
                >
                  {value.slice(range.start, range.end)}
                </span>
              </span>
            ))}
            {value.slice(orderedMentions[orderedMentions.length - 1].range.end)}
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
          mentions.length > 0 &&
            "relative bg-transparent text-transparent caret-main disabled:bg-transparent"
        )}
        onKeyDown={(event) => {
          if (
            !event.nativeEvent.isComposing &&
            (event.key === "Backspace" || event.key === "Delete") &&
            onDeleteMention
          ) {
            const key = event.key as "Backspace" | "Delete";
            const result = mentions
              .map(({ range, label }) =>
                deleteResourceMention(
                  value,
                  range,
                  label,
                  event.currentTarget.selectionStart,
                  event.currentTarget.selectionEnd,
                  key
                )
              )
              .find((result) => result !== null);
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
