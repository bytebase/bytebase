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
  WrapTextIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type {
  MaskingReason,
  QueryResult,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  STORAGE_KEY_SQL_EDITOR_DETAIL_FORMAT,
  STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP,
} from "@/utils/storage-keys";
import { getInstanceResource } from "@/utils/v1/database";
import { BinaryFormatButton } from "./BinaryFormatButton";
import { getPlainValue, isLikelyJSON } from "./cell-value";
import { useBinaryFormatContext, useSQLResultViewContext } from "./context";
import {
  DETAIL_SEARCH_ACTIVE_MATCH_SELECTOR,
  renderRowFieldsWithSearchMatches,
  renderTextWithSearchMatches,
} from "./detail-panel-search";
import { getCellDisplayText, PlainCellValue } from "./PlainCellValue";
import { PrettyJSON } from "./PrettyJSON";
import { RowDataBlock } from "./RowDataBlock";
import { TextSearchControl } from "./TextSearchControl";
import type { ResultTableColumn, ResultTableRow } from "./types";

interface DetailPanelProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  database: Database;
  result: QueryResult;
  statement?: string;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  presentation?: "sheet" | "embedded";
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

export function DetailPanel({
  rows,
  columns,
  database,
  result,
  statement,
  getMaskingReason,
  presentation = "sheet",
}: DetailPanelProps) {
  const { t } = useTranslation();
  const { detail, disallowCopyingData, setDetail } = useSQLResultViewContext();
  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();
  const [copied, setCopied] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeMatchIndex, setActiveMatchIndex] = useState(0);
  const [formattedMatchCount, setFormattedMatchCount] = useState(0);
  const [highlightedContentVersion, setHighlightedContentVersion] = useState(0);
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

  const engine = getInstanceResource(database).engine;
  const isDocumentEngine =
    engine === Engine.COSMOSDB || engine === Engine.MONGODB;

  const detailContent = useMemo(() => {
    if (!detail) return undefined;
    const displayedRow = rows[detail.row];
    if (!displayedRow) return undefined;

    if (detail.view === "row") {
      if (isDocumentEngine) {
        const sourceValue = result.rows[displayedRow.key]?.values[0];
        return {
          kind: "document" as const,
          content: getPlainValue(
            sourceValue,
            result.columnTypeNames[0] ?? "",
            undefined
          ),
        };
      }

      const fields = columns.map((column, colIndex) => ({
        column,
        value: getPlainValue(
          displayedRow.item.values[colIndex],
          column.columnType,
          getBinaryFormat({ rowIndex: detail.row, colIndex })
        ),
      }));
      return {
        kind: "row" as const,
        fields,
        content: fields
          .map(
            ({ column, value }) =>
              `${column.name}: ${getCellDisplayText(value)}`
          )
          .join("\n"),
      };
    }

    const value = displayedRow.item.values[detail.col];
    const columnType = columns[detail.col]?.columnType ?? "";
    const binaryFormat = getBinaryFormat({
      rowIndex: detail.row,
      colIndex: detail.col,
    });
    return {
      kind: "cell" as const,
      value,
      binaryFormat,
      content: getPlainValue(value, columnType, binaryFormat),
    };
  }, [columns, detail, getBinaryFormat, isDocumentEngine, result, rows]);

  const content = detailContent?.content;

  const guessedIsJSON = isLikelyJSON(content);
  const showFormattedJSON =
    guessedIsJSON &&
    (detailContent?.kind === "document" ||
      (detailContent?.kind === "cell" && format));

  const searchResult = useMemo(() => {
    if (detailContent?.kind === "row") {
      return {
        kind: "row" as const,
        ...renderRowFieldsWithSearchMatches(
          detailContent.fields.map(({ column, value }) => ({
            columnName: column.name,
            value: getCellDisplayText(value),
          })),
          searchQuery,
          activeMatchIndex
        ),
      };
    }
    return {
      kind: "plain" as const,
      ...renderTextWithSearchMatches(
        content ?? "",
        searchQuery,
        activeMatchIndex
      ),
    };
  }, [activeMatchIndex, content, detailContent, searchQuery]);
  const matchCount = showFormattedJSON
    ? formattedMatchCount
    : searchResult.count;
  const rowSearchFields =
    searchResult.kind === "row" ? searchResult.fields : [];

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
    setActiveMatchIndex(0);
  }, [content, detailContent?.kind, format, searchQuery]);

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
    if (showFormattedJSON) {
      try {
        const obj = losslessParse(raw);
        return losslessStringify(obj, null, "  ") ?? raw;
      } catch {
        console.warn("[DetailPanel]", "failed to format JSON for copy");
        return raw;
      }
    }
    return raw;
  }, [content, showFormattedJSON]);

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

  const handleHighlightedContentChange = useCallback(() => {
    setHighlightedContentVersion((version) => version + 1);
  }, []);
  const selectedColumnName =
    detail?.view === "cell" ? columns[detail.col]?.name : undefined;

  const body = detail ? (
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
        "flex-1 min-h-0 flex flex-col gap-y-2",
        presentation === "sheet" ? "px-6 py-4" : "p-2",
        "text-main"
      )}
    >
      <div
        className={cn(
          "flex items-center justify-between gap-x-4",
          presentation === "embedded" && "flex-wrap gap-y-2"
        )}
      >
        <div className="flex items-center gap-x-2">
          <Tooltip content={t("sql-editor.previous-row")} delayDuration={500}>
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
          <TextSearchControl
            query={searchQuery}
            activeMatchIndex={activeMatchIndex}
            matchCount={matchCount}
            onQueryChange={setSearchQuery}
            onMove={moveSearchMatch}
            label={t("sql-editor.result-detail.search")}
            className={cn(
              presentation === "sheet" ? "w-80 flex-none" : "flex-1"
            )}
          />

          <div className="flex shrink-0 items-center gap-1">
            {guessedIsJSON && detailContent?.kind === "cell" && (
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

            {detailContent?.kind === "cell" &&
              detailContent.value?.kind?.case === "bytesValue" && (
                <BinaryFormatButton
                  format={detailContent.binaryFormat}
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
          showFormattedJSON && !wrap ? "whitespace-pre" : "whitespace-pre-wrap"
        )}
        onClick={stopSelectionClickPropagation}
      >
        {detailContent?.kind === "row" ? (
          <RowDataBlock
            columns={columns}
            database={database.name}
            statement={statement}
            getMaskingReason={getMaskingReason}
            renderColumnName={(_, columnIndex) =>
              rowSearchFields[columnIndex]?.columnName
            }
            renderValue={(_, columnIndex) => (
              <div className="px-2 py-1">
                <PlainCellValue
                  value={detailContent.fields[columnIndex]?.value}
                >
                  {rowSearchFields[columnIndex]?.value}
                </PlainCellValue>
              </div>
            )}
          />
        ) : showFormattedJSON ? (
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
              onMatchCountChange={setFormattedMatchCount}
              onHighlightedContentChange={handleHighlightedContentChange}
            />
          </>
        ) : content && content.length > 0 ? (
          <>{searchResult.kind === "plain" ? searchResult.nodes : null}</>
        ) : (
          <br style={{ minWidth: "1rem", display: "inline-flex" }} />
        )}
      </div>
    </div>
  ) : null;

  if (presentation === "embedded") {
    return <div className="h-full min-h-0 flex flex-col">{body}</div>;
  }

  return (
    <Sheet open={isOpen} onOpenChange={handleOpenChange}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>
            {detail?.view === "cell"
              ? t("sql-editor.result-detail.cell")
              : t("common.detail")}
          </SheetTitle>
          {selectedColumnName && (
            <SheetDescription className="flex min-w-0 items-center gap-x-1">
              <span className="shrink-0">{t("common.column")}:</span>
              <Tooltip content={selectedColumnName} delayDuration={500}>
                <span className="block max-w-[32rem] truncate font-mono text-control">
                  {selectedColumnName}
                </span>
              </Tooltip>
            </SheetDescription>
          )}
        </SheetHeader>
        {body}
      </SheetContent>
    </Sheet>
  );
}
