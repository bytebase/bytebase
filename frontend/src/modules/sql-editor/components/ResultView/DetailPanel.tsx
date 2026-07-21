import {
  parse as losslessParse,
  stringify as losslessStringify,
} from "lossless-json";
import {
  BracesIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  CopyIcon,
  SearchIcon,
  WrapTextIcon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import {
  STORAGE_KEY_SQL_EDITOR_DETAIL_FORMAT,
  STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP,
} from "@/utils/storage-keys";
import { BinaryFormatButton } from "./BinaryFormatButton";
import { getPlainValue } from "./cell-value";
import { useBinaryFormatContext, useSQLResultViewContext } from "./context";
import {
  DETAIL_SEARCH_ACTIVE_MATCH_SELECTOR,
  renderTextWithSearchMatches,
  searchMatchCountLabel,
} from "./detail-panel-search";
import { PrettyJSON } from "./PrettyJSON";
import type { ResultTableColumn, ResultTableRow } from "./types";

interface DetailPanelProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
}

function useLocalStorageBoolean(
  key: string,
  defaultValue: boolean
): [boolean, (next: boolean) => void] {
  const [value, setValue] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (raw === "true") return true;
      if (raw === "false") return false;
    } catch {
      // ignore
    }
    return defaultValue;
  });
  const update = useCallback(
    (next: boolean) => {
      setValue(next);
      try {
        localStorage.setItem(key, String(next));
      } catch {
        // ignore
      }
    },
    [key]
  );
  return [value, update];
}

const isEditableTarget = (target: EventTarget | null) => {
  if (!(target instanceof HTMLElement)) {
    return false;
  }
  const tagName = target.tagName.toLowerCase();
  return (
    tagName === "input" ||
    tagName === "textarea" ||
    target.isContentEditable ||
    target.closest("[contenteditable='true']") !== null
  );
};

export function DetailPanel({ rows, columns }: DetailPanelProps) {
  const { t } = useTranslation();
  const { detail, disallowCopyingData, setDetail } = useSQLResultViewContext();
  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();
  const [copied, setCopied] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeMatchIndex, setActiveMatchIndex] = useState(0);
  const [matchCount, setMatchCount] = useState(0);
  const [highlightedContentVersion, setHighlightedContentVersion] = useState(0);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  const [format, setFormat] = useLocalStorageBoolean(
    STORAGE_KEY_SQL_EDITOR_DETAIL_FORMAT,
    false
  );
  const [wrap, setWrap] = useLocalStorageBoolean(
    STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP,
    true
  );

  const totalCount = rows.length;

  const rawValue = useMemo(() => {
    if (!detail) return undefined;
    return rows[detail.row]?.item.values[detail.col];
  }, [detail, rows]);

  const columnType = detail ? (columns[detail.col]?.columnType ?? "") : "";

  const binaryFormat = detail
    ? getBinaryFormat({ rowIndex: detail.row, colIndex: detail.col })
    : undefined;

  const isBinaryData = rawValue?.kind?.case === "bytesValue";

  const content = useMemo(
    () => getPlainValue(rawValue, columnType, binaryFormat),
    [rawValue, columnType, binaryFormat]
  );

  const guessedIsJSON = useMemo(() => {
    if (!content) return false;
    const trimmed = content.trim();
    return (
      (trimmed.startsWith("{") && trimmed.endsWith("}")) ||
      (trimmed.startsWith("[") && trimmed.endsWith("]"))
    );
  }, [content]);

  const move = useCallback(
    (offset: number) => {
      if (!detail) return;
      const target = detail.row + offset;
      if (target < 0 || target >= totalCount) return;
      setDetail({ ...detail, row: target });
    },
    [detail, totalCount, setDetail]
  );

  const moveSearchMatch = useCallback(
    (offset: number) => {
      if (matchCount === 0) {
        return;
      }
      setActiveMatchIndex((current) => {
        return (current + offset + matchCount) % matchCount;
      });
    },
    [matchCount]
  );

  // Replicates Vue's onKeyStroke("ArrowUp"/"ArrowDown") row navigation while
  // the panel is open.
  useEffect(() => {
    if (!detail) return;
    const handler = (e: KeyboardEvent) => {
      if (isEditableTarget(e.target)) {
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        e.stopPropagation();
        move(-1);
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        e.stopPropagation();
        move(1);
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [detail, move]);

  useEffect(() => {
    if (!detail) return;
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "f") {
        e.preventDefault();
        e.stopPropagation();
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [detail]);

  useEffect(() => {
    setActiveMatchIndex(0);
  }, [content, format, searchQuery]);

  useEffect(() => {
    if (matchCount === 0) {
      setActiveMatchIndex(0);
      return;
    }
    if (activeMatchIndex >= matchCount) {
      setActiveMatchIndex(matchCount - 1);
    }
  }, [activeMatchIndex, content, format, matchCount, searchQuery]);

  useEffect(() => {
    const activeMatch = contentRef.current?.querySelector(
      DETAIL_SEARCH_ACTIVE_MATCH_SELECTOR
    );
    if (activeMatch instanceof HTMLElement) {
      activeMatch.scrollIntoView?.({ block: "center", inline: "nearest" });
    }
  }, [
    activeMatchIndex,
    content,
    format,
    highlightedContentVersion,
    matchCount,
    searchQuery,
  ]);

  const copyContent = useMemo(() => {
    const raw = content ?? "";
    if (guessedIsJSON && format) {
      try {
        const obj = losslessParse(raw);
        return losslessStringify(obj, null, "  ") ?? raw;
      } catch {
        console.warn("[DetailPanel]", "failed to format JSON for copy");
        return raw;
      }
    }
    return raw;
  }, [content, guessedIsJSON, format]);

  const handleCopy = useCallback(async () => {
    if (await writeTextToClipboard(copyContent)) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } else {
      // ignore
    }
  }, [copyContent]);

  const isOpen = !!detail;
  const handleOpenChange = (next: boolean) => {
    if (!next) setDetail(undefined);
  };

  const stopSelectionClickPropagation = (event: React.MouseEvent) => {
    if (window.getSelection()?.toString()) {
      event.stopPropagation();
    }
  };

  const plainSearchResult = useMemo(
    () =>
      renderTextWithSearchMatches(content ?? "", searchQuery, activeMatchIndex),
    [activeMatchIndex, content, searchQuery]
  );

  const handleHighlightedContentChange = useCallback(() => {
    setHighlightedContentVersion((version) => version + 1);
  }, []);

  useEffect(() => {
    if (!(guessedIsJSON && format)) {
      setMatchCount(plainSearchResult.count);
    }
  }, [format, guessedIsJSON, plainSearchResult.count]);

  const handleSearchKeyDown = (
    event: React.KeyboardEvent<HTMLInputElement>
  ) => {
    if (event.key !== "Enter") {
      return;
    }
    event.preventDefault();
    moveSearchMatch(event.shiftKey ? -1 : 1);
  };

  const clearSearch = () => {
    setSearchQuery("");
    searchInputRef.current?.focus();
  };

  return (
    <Sheet open={isOpen} onOpenChange={handleOpenChange}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>{t("common.detail")}</SheetTitle>
        </SheetHeader>
        {detail && (
          <div
            className={cn(
              // `flex-1 min-h-0` instead of `h-full` so the wrapper absorbs
              // the space remaining after `SheetHeader` (not 100vh). The
              // `min-h-0` lets the inner `flex-1 overflow-auto` block
              // actually clip and scroll — without it, the scroll region
              // expands to fit its content and the bottom rows render past
              // the viewport.
              // Match `SheetBody`'s `px-6 py-4` so the toolbar buttons
              // and the content code block don't bleed to the sheet's
              // raw edges.
              "flex-1 min-h-0 flex flex-col gap-y-2 px-6 py-4",
              "text-main"
            )}
          >
            <div className="flex items-center justify-between gap-x-4">
              <div className="flex items-center gap-x-2">
                <Tooltip
                  content={t("sql-editor.previous-row")}
                  delayDuration={500}
                >
                  <Button
                    size="sm"
                    appearance="outline"
                    disabled={detail.row === 0}
                    onClick={() => move(-1)}
                  >
                    <ChevronUpIcon className="size-4" />
                  </Button>
                </Tooltip>
                <Tooltip content={t("sql-editor.next-row")} delayDuration={500}>
                  <Button
                    size="sm"
                    appearance="outline"
                    disabled={detail.row === totalCount - 1}
                    onClick={() => move(1)}
                  >
                    <ChevronDownIcon className="size-4" />
                  </Button>
                </Tooltip>
                <div className="text-xs text-control-light flex items-center gap-x-1">
                  <span>{detail.row + 1}</span>
                  <span>/</span>
                  <span>{totalCount}</span>
                  <span>{t("sql-editor.rows.self")}</span>
                </div>
              </div>

              <div className="flex min-w-0 items-center gap-x-2">
                <div
                  data-testid="detail-search-control"
                  className={cn(
                    "h-8 w-80 min-w-0 flex items-center overflow-hidden rounded-xs",
                    "border border-control-border bg-transparent text-main transition-colors"
                  )}
                >
                  <SearchIcon className="ml-2.5 size-4 shrink-0 text-control-placeholder" />
                  <Input
                    ref={searchInputRef}
                    size="sm"
                    aria-label={t("sql-editor.result-detail.search")}
                    className="h-7 min-w-0 flex-1 border-0 px-2 text-sm focus:ring-0"
                    placeholder={t("sql-editor.result-detail.search")}
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    onKeyDown={handleSearchKeyDown}
                  />
                  {searchQuery.trim() && (
                    <span className="min-w-10 text-center text-xs text-control-light">
                      {searchMatchCountLabel(activeMatchIndex, matchCount)}
                    </span>
                  )}
                  {searchQuery.trim() && (
                    <div className="ml-1 flex shrink-0 items-center">
                      <Tooltip
                        content={t("sql-editor.result-detail.previous-match")}
                      >
                        <Button
                          size="sm"
                          appearance="secondary"
                          className="size-7 p-0"
                          disabled={matchCount === 0}
                          onClick={() => moveSearchMatch(-1)}
                        >
                          <ChevronUpIcon className="size-4" />
                        </Button>
                      </Tooltip>
                      <Tooltip
                        content={t("sql-editor.result-detail.next-match")}
                      >
                        <Button
                          size="sm"
                          appearance="secondary"
                          className="size-7 p-0"
                          disabled={matchCount === 0}
                          onClick={() => moveSearchMatch(1)}
                        >
                          <ChevronDownIcon className="size-4" />
                        </Button>
                      </Tooltip>
                      <Tooltip content={t("common.close")}>
                        <Button
                          size="sm"
                          appearance="secondary"
                          className="size-7 border-l border-control-border p-0"
                          onClick={clearSearch}
                        >
                          <XIcon className="size-4" />
                        </Button>
                      </Tooltip>
                    </div>
                  )}
                </div>

                <div className="flex shrink-0 items-center gap-1">
                  {guessedIsJSON && (
                    <Tooltip content={t("sql-editor.format")}>
                      <Button
                        size="sm"
                        appearance={format ? "solid" : "outline"}
                        className="h-7 px-1.5"
                        onClick={() => setFormat(!format)}
                      >
                        <BracesIcon className="size-4" />
                      </Button>
                    </Tooltip>
                  )}

                  {isBinaryData && (
                    <BinaryFormatButton
                      format={binaryFormat}
                      onFormatChange={(next) =>
                        setBinaryFormat({
                          rowIndex: detail.row,
                          colIndex: detail.col,
                          format: next,
                        })
                      }
                    />
                  )}

                  {!disallowCopyingData && (
                    <Tooltip content={t("common.copy")}>
                      <Button
                        size="sm"
                        appearance="secondary"
                        className="size-7 p-0"
                        onClick={handleCopy}
                      >
                        {copied ? (
                          <CheckIcon className="size-4" />
                        ) : (
                          <CopyIcon className="size-4" />
                        )}
                      </Button>
                    </Tooltip>
                  )}
                </div>
              </div>
            </div>

            <div
              ref={contentRef}
              className={cn(
                "flex-1 overflow-auto text-sm font-mono border p-2 relative",
                disallowCopyingData ? "select-none" : "select-text",
                guessedIsJSON && format && !wrap
                  ? "whitespace-pre"
                  : "whitespace-pre-wrap"
              )}
              onClick={stopSelectionClickPropagation}
            >
              {guessedIsJSON && format ? (
                <>
                  <div className="absolute right-2 top-2 flex justify-end items-center gap-1">
                    <Tooltip content={t("common.text-wrap")}>
                      <Button
                        size="sm"
                        appearance={wrap ? "solid" : "outline"}
                        className="h-6 px-1"
                        onClick={() => setWrap(!wrap)}
                      >
                        <WrapTextIcon className="size-3" />
                      </Button>
                    </Tooltip>
                  </div>
                  <PrettyJSON
                    content={content ?? ""}
                    searchQuery={searchQuery}
                    activeMatchIndex={activeMatchIndex}
                    onMatchCountChange={setMatchCount}
                    onHighlightedContentChange={handleHighlightedContentChange}
                  />
                </>
              ) : content && content.length > 0 ? (
                <>{plainSearchResult.nodes}</>
              ) : (
                <br style={{ minWidth: "1rem", display: "inline-flex" }} />
              )}
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
