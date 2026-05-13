import { Check, Copy, PlayIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import { Tooltip } from "@/react/components/ui/tooltip";
import { findAncestor } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { useAIContext } from "../../context";

export type CodeBlockProps = {
  /** Fraction (0..1) of the parent `.message` width the snippet card occupies. */
  width: number;
};

type Props = CodeBlockProps & {
  code: string;
};

const MIN_WIDTH_PX = 8 * 16; // 8rem — same minimum as the Vue source.
const PADDING_PX = 8;

/**
 * React port of `plugins/ai/components/ChatView/Markdown/CodeBlock.vue`.
 *
 * Renders a SQL snippet inside a read-only `MonacoEditor` with three
 * actions:
 *   - Run: `aiContextEvents.emit("run-statement", { statement })`
 *     (the SQL editor host listens and executes the snippet)
 *   - Insert-at-caret: `sqlEditorEvents.emit("insert-at-caret", { content })`
 *   - Copy: writes to the clipboard with an ephemeral "Copied" indicator.
 *
 * Width is adaptive: measure the nearest `.message` ancestor (set by
 * `ChatView`) via ResizeObserver and clamp to `[MIN_WIDTH_PX, ...]`.
 * `width` is the fraction of the message bubble this card occupies —
 * `AIMessageView` passes `0.85`.
 */
export function CodeBlock({ code, width }: Props) {
  const { t } = useTranslation();
  const { events, setShowHistoryDialog } = useAIContext();

  const containerRef = useRef<HTMLDivElement | null>(null);
  // The message wrapper element is mounted by `ChatView` with the
  // `.message` class. We measure it (not our own container) so the
  // snippet can size relative to the bubble even when the bubble shrinks
  // due to a sibling layout change.
  const [messageWidth, setMessageWidth] = useState(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const messageEl = findAncestor(el, ".message");
    if (!(messageEl instanceof HTMLElement)) return;
    const update = () => setMessageWidth(messageEl.clientWidth);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(messageEl);
    return () => observer.disconnect();
  }, []);

  const computedWidth = Math.max(
    MIN_WIDTH_PX,
    messageWidth * width - PADDING_PX * 2
  );

  const handleExecute = () => {
    events.emit("run-statement", { statement: code });
    setShowHistoryDialog(false);
  };

  const handleInsertAtCaret = () => {
    sqlEditorEvents.emit("insert-at-caret", { content: code });
    setShowHistoryDialog(false);
  };

  return (
    <div
      ref={containerRef}
      className="flex flex-col overflow-x-hidden border border-gray-500 rounded-sm bg-white"
      style={{ width: `${computedWidth}px` }}
    >
      <div className="flex items-center justify-between px-1 pt-1">
        <div className="text-xs">SQL</div>
        <div className="flex items-center justify-end gap-2">
          <Tooltip content={t("common.run")} side="bottom">
            <button
              type="button"
              className="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              onClick={handleExecute}
              aria-label={t("common.run")}
            >
              <PlayIcon className="size-3.5" />
            </button>
          </Tooltip>
          <Tooltip
            content={t("plugin.ai.actions.insert-at-caret")}
            side="bottom"
          >
            <button
              type="button"
              className="inline-flex items-center justify-center hover:text-accent cursor-pointer"
              onClick={handleInsertAtCaret}
              aria-label={t("plugin.ai.actions.insert-at-caret")}
            >
              <InsertAtCaretIconCompact />
            </button>
          </Tooltip>
          <Tooltip content={t("common.copy")} side="bottom">
            <CopyButton content={code} />
          </Tooltip>
        </div>
      </div>

      <MonacoEditor
        content={code}
        readOnly
        autoFocus={false}
        autoHeight
        min={20}
        max={120}
        options={{
          fontSize: 12,
          lineHeight: 14,
          lineNumbers: "off",
          wordWrap: "on",
          scrollbar: {
            vertical: "hidden",
            horizontal: "hidden",
            useShadows: false,
            verticalScrollbarSize: 0,
            horizontalScrollbarSize: 0,
            alwaysConsumeMouseWheel: false,
          },
        }}
        className="h-auto"
      />
    </div>
  );
}

/**
 * Inline copy of `InsertAtCaretIcon` at a fixed 14px to match the other
 * action buttons. Avoids an extra import + size prop juggling.
 */
function InsertAtCaretIconCompact() {
  return (
    <span
      style={{ width: 14, height: 14 }}
      className="relative inline-block"
      aria-hidden="true"
    >
      <svg
        width="14"
        height="14"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <line x1="21" y1="6" x2="3" y2="6" />
        <line x1="15" y1="12" x2="3" y2="12" />
        <line x1="17" y1="18" x2="3" y2="18" />
      </svg>
    </span>
  );
}

/**
 * Inline 20-LOC copy primitive. No shared `CopyButton` exists in
 * `react/components/` yet; extracting one is out of scope for Stage 22.
 * Falls back silently when `navigator.clipboard` is missing (SSR / older
 * browsers) — same conservative behaviour as `TableSchemaViewer.tsx`.
 */
function CopyButton({ content }: { content: string }) {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const handle = window.setTimeout(() => setCopied(false), 1500);
    return () => window.clearTimeout(handle);
  }, [copied]);

  const handleClick = async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard) return;
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
    } catch {
      // ignore — same posture as TableSchemaViewer.tsx
    }
  };

  return (
    <button
      type="button"
      className="inline-flex items-center justify-center hover:text-accent cursor-pointer"
      onClick={handleClick}
    >
      {copied ? (
        <Check className="size-3.5 text-success" />
      ) : (
        <Copy className="size-3.5" />
      )}
    </button>
  );
}
