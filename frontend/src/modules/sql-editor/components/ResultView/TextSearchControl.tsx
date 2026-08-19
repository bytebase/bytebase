import {
  ChevronDownIcon,
  ChevronUpIcon,
  SearchIcon,
  XIcon,
} from "lucide-react";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { searchMatchCountLabel } from "./detail-panel-search";

interface TextSearchControlProps {
  query: string;
  activeMatchIndex: number;
  matchCount: number;
  onQueryChange: (query: string) => void;
  onMove: (offset: number) => void;
  label?: string;
  className?: string;
}

export function TextSearchControl({
  query,
  activeMatchIndex,
  matchCount,
  onQueryChange,
  onMove,
  label,
  className,
}: TextSearchControlProps) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const searchActive = query.trim().length > 0;
  const searchLabel = label ?? t("common.search");

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "f") {
        event.preventDefault();
        event.stopPropagation();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, []);

  return (
    <div
      className={cn(
        "h-8 min-w-0 flex-1 flex items-center overflow-hidden rounded-xs",
        "border border-control-border bg-transparent text-main transition-colors",
        className
      )}
    >
      <SearchIcon className="ml-2.5 size-4 shrink-0 text-control-placeholder" />
      <Input
        ref={inputRef}
        size="sm"
        aria-label={searchLabel}
        className="h-7 min-w-0 flex-1 border-0 px-2 text-sm focus:ring-0"
        placeholder={searchLabel}
        value={query}
        autoComplete="off"
        onChange={(event) => onQueryChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key !== "Enter") {
            return;
          }
          event.preventDefault();
          onMove(event.shiftKey ? -1 : 1);
        }}
      />
      {searchActive && (
        <span className="min-w-10 text-center text-xs text-control-light">
          {searchMatchCountLabel(activeMatchIndex, matchCount)}
        </span>
      )}
      {searchActive && (
        <div className="ml-1 flex shrink-0 items-center">
          <Tooltip content={t("sql-editor.result-detail.previous-match")}>
            <Button
              size="sm"
              appearance="secondary"
              className="size-7 p-0"
              aria-label={t("sql-editor.result-detail.previous-match")}
              disabled={matchCount === 0}
              onClick={() => onMove(-1)}
            >
              <ChevronUpIcon className="size-4" />
            </Button>
          </Tooltip>
          <Tooltip content={t("sql-editor.result-detail.next-match")}>
            <Button
              size="sm"
              appearance="secondary"
              className="size-7 p-0"
              aria-label={t("sql-editor.result-detail.next-match")}
              disabled={matchCount === 0}
              onClick={() => onMove(1)}
            >
              <ChevronDownIcon className="size-4" />
            </Button>
          </Tooltip>
          <Tooltip content={t("common.close")}>
            <Button
              size="sm"
              appearance="secondary"
              className="size-7 border-l border-control-border p-0"
              aria-label={t("common.close")}
              onClick={() => {
                onQueryChange("");
                inputRef.current?.focus();
              }}
            >
              <XIcon className="size-4" />
            </Button>
          </Tooltip>
        </div>
      )}
    </div>
  );
}
