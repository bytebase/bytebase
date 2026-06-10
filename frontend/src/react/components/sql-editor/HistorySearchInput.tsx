import { Search, X } from "lucide-react";
import { useLayoutEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";

// Tallest the box grows before it starts scrolling internally (px).
const MAX_HEIGHT = 120;

interface Props {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

/**
 * Query-history search box. Unlike the shared single-line `SearchInput`, this
 * uses an auto-growing `<textarea>` so a long query soft-wraps and the box
 * grows up to `MAX_HEIGHT` (then scrolls), plus a trailing clear button.
 * The auto-grow follows the codebase's `scrollHeight` pattern (see
 * `AgentInput` / `PromptInput`).
 */
export function HistorySearchInput({
  value,
  onChange,
  placeholder,
  className,
}: Props) {
  const { t } = useTranslation();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useLayoutEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    // Reset first so shrinking works, then clamp to the max.
    el.style.height = "auto";
    const next = Math.min(el.scrollHeight, MAX_HEIGHT);
    el.style.height = `${next}px`;
    el.style.overflowY = el.scrollHeight > MAX_HEIGHT ? "auto" : "hidden";
  }, [value]);

  return (
    <div
      className={cn(
        "relative flex w-full rounded-xs border border-control-border bg-transparent transition-colors",
        className
      )}
    >
      <Search className="absolute left-2.5 top-2 h-4 w-4 text-control-placeholder pointer-events-none" />
      <textarea
        ref={textareaRef}
        rows={1}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        // It's a search box, not a multi-line editor: long text soft-wraps,
        // but Enter must not insert a newline.
        onKeyDown={(e) => {
          if (e.key === "Enter") e.preventDefault();
        }}
        placeholder={placeholder ?? t("common.type-to-search")}
        className={cn(
          "w-full resize-none appearance-none border-0 bg-transparent text-sm leading-5 text-main",
          "py-1.5 pl-8 pr-8 outline-hidden focus:ring-0 focus:outline-hidden",
          "placeholder:text-control-placeholder"
        )}
      />
      {value && (
        <button
          type="button"
          data-clear-search
          aria-label={t("common.clear")}
          onClick={() => onChange("")}
          className="absolute right-1.5 top-1.5 flex h-5 w-5 items-center justify-center rounded-xs text-control-placeholder hover:bg-control-bg-hover hover:text-control"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}
