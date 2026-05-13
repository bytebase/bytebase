import { CornerDownLeft } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import { keyboardShortcutStr } from "@/utils";
import { useAIContext } from "./context";

type Props = {
  readonly disabled?: boolean;
  readonly onEnter: (value: string) => void;
};

const LINE_HEIGHT_PX = 20;
const MIN_ROWS = 1;
const MAX_ROWS = 10;

/**
 * React port of `plugins/ai/components/PromptInput.vue`.
 *
 * Autosizing textarea (1-10 visible rows) bound to local state. Enter
 * submits; Shift+Enter inserts a newline. The trailing button is a
 * tooltipped ⏎ that submits when clicked.
 *
 * Naive UI's `NInput type="textarea" autosize` isn't available; we
 * hand-roll the autosize by measuring `scrollHeight` after each value
 * change and clamping to `[MIN_ROWS, MAX_ROWS] * lineHeight`. Adding a
 * runtime dep (`react-textarea-autosize`) just for this surface isn't
 * worth it — the hand-rolled version is ~10 lines and behaves
 * identically for plain text input.
 *
 * Reacts to two external triggers from `AIContext`:
 *   - `pendingPreInput`: a one-shot seed value (e.g. `OpenAIButton`
 *     "Ask AI about this query"). Cleared after consumption.
 *   - `new-conversation` event: re-focuses the textarea so the user can
 *     immediately type into a freshly-created conversation.
 */
export function PromptInput({ disabled = false, onEnter }: Props) {
  const { t } = useTranslation();
  const { pendingPreInput, setPendingPreInput, events } = useAIContext();

  const [value, setValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // Focus on mount + on `new-conversation` so the user can start typing
  // immediately.
  useEffect(() => {
    textareaRef.current?.focus();
  }, []);
  useEffect(() => {
    const off = events.on("new-conversation", () => {
      textareaRef.current?.focus();
    });
    return () => {
      off();
    };
  }, [events]);

  // Consume `pendingPreInput`: when the provider sets it, copy into
  // local state and clear the trigger. rAF mirrors the Vue version's
  // `flush: "post"` watch — defers to the next paint so any conversation
  // creation that triggered the seed has landed first.
  useEffect(() => {
    if (!pendingPreInput) return;
    const raf = requestAnimationFrame(() => {
      setValue(pendingPreInput);
      setPendingPreInput(undefined);
    });
    return () => cancelAnimationFrame(raf);
  }, [pendingPreInput, setPendingPreInput]);

  // Autosize: after every value change, set height to scrollHeight,
  // clamped to [MIN_ROWS, MAX_ROWS] lines.
  useLayoutEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    const minHeight = MIN_ROWS * LINE_HEIGHT_PX;
    const maxHeight = MAX_ROWS * LINE_HEIGHT_PX;
    const next = Math.min(maxHeight, Math.max(minHeight, el.scrollHeight));
    el.style.height = `${next}px`;
  }, [value]);

  const applyValue = (raw: string) => {
    setValue("");
    onEnter(raw);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key !== "Enter") return;
    if (e.shiftKey) return; // Shift+Enter → newline
    e.preventDefault();
    if (!value.trim()) return;
    applyValue(value);
  };

  const handleSubmitClick = () => {
    if (!value.trim()) return;
    applyValue(value);
  };

  const tooltipContent = useMemo(
    () => (
      <div className="text-xs flex flex-col gap-1">
        <p className="flex items-center gap-1">
          <span>{t("plugin.ai.send")}</span>
          <span>({keyboardShortcutStr("⏎")})</span>
        </p>
        <p className="flex items-center gap-1">
          <span>{t("plugin.ai.new-line")}</span>
          <span>({keyboardShortcutStr("shift+⏎")})</span>
        </p>
      </div>
    ),
    [t]
  );

  return (
    <div className="relative w-full">
      <textarea
        ref={textareaRef}
        value={value}
        disabled={disabled}
        placeholder={t("plugin.ai.text-to-sql-placeholder")}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={MIN_ROWS}
        className={cn(
          "w-full resize-none rounded-xs border border-control-border bg-background pl-2 pr-9 py-1 text-sm",
          "leading-5 text-main placeholder:text-control-placeholder",
          "focus:outline-none focus:border-accent focus:ring-0",
          "disabled:opacity-50 disabled:cursor-not-allowed"
        )}
      />
      <Tooltip content={tooltipContent} side="top">
        <Button
          variant="ghost"
          size="xs"
          className="absolute right-1 bottom-1 h-6 px-1.5 text-accent"
          disabled={!value || disabled}
          onClick={handleSubmitClick}
          aria-label={t("plugin.ai.send")}
        >
          <CornerDownLeft className="size-3.5" />
        </Button>
      </Tooltip>
    </div>
  );
}
