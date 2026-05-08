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
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import {
  STORAGE_KEY_SQL_EDITOR_DETAIL_FORMAT,
  STORAGE_KEY_SQL_EDITOR_DETAIL_LINE_WRAP,
} from "@/utils/storage-keys";
import { BinaryFormatButton } from "./BinaryFormatButton";
import { getPlainValue } from "./cell-value";
import { useBinaryFormatContext, useSQLResultViewContext } from "./context";
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

export function DetailPanel({ rows, columns }: DetailPanelProps) {
  const { t } = useTranslation();
  const { dark, detail, disallowCopyingData, setDetail } =
    useSQLResultViewContext();
  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();
  const [copied, setCopied] = useState(false);

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

  // Replicates Vue's onKeyStroke("ArrowUp"/"ArrowDown") row navigation while
  // the panel is open.
  useEffect(() => {
    if (!detail) return;
    const handler = (e: KeyboardEvent) => {
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
    try {
      await navigator.clipboard.writeText(copyContent);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // ignore
    }
  }, [copyContent]);

  const isOpen = !!detail;
  const handleOpenChange = (next: boolean) => {
    if (!next) setDetail(undefined);
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
              "h-full flex flex-col gap-y-2",
              dark ? "text-white" : "text-main"
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
                    variant="outline"
                    disabled={detail.row === 0}
                    onClick={() => move(-1)}
                  >
                    <ChevronUpIcon className="size-4" />
                  </Button>
                </Tooltip>
                <Tooltip content={t("sql-editor.next-row")} delayDuration={500}>
                  <Button
                    size="sm"
                    variant="outline"
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

              <div className="flex items-center gap-1">
                {guessedIsJSON && (
                  <Tooltip content={t("sql-editor.format")}>
                    <Button
                      size="sm"
                      variant={format ? "default" : "outline"}
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
                      variant="ghost"
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

            <div
              className={cn(
                "flex-1 overflow-auto text-sm font-mono border p-2 relative",
                disallowCopyingData && "select-none",
                guessedIsJSON && format && !wrap
                  ? "whitespace-pre"
                  : "whitespace-pre-wrap"
              )}
            >
              {guessedIsJSON && format ? (
                <>
                  <div className="absolute right-2 top-2 flex justify-end items-center gap-1">
                    <Tooltip content={t("common.text-wrap")}>
                      <Button
                        size="sm"
                        variant={wrap ? "default" : "outline"}
                        className="h-6 px-1"
                        onClick={() => setWrap(!wrap)}
                      >
                        <WrapTextIcon className="size-3" />
                      </Button>
                    </Tooltip>
                  </div>
                  <PrettyJSON content={content ?? ""} />
                </>
              ) : content && content.length > 0 ? (
                <>{content}</>
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
