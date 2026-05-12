import { useVirtualizer } from "@tanstack/react-virtual";
import { CheckIcon, CopyIcon } from "lucide-react";
import {
  forwardRef,
  useCallback,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { MaskingReasonPopover } from "@/react/components/sql-editor/MaskingReasonPopover";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import type { SearchParams } from "@/utils/v1/advanced-search/common";
import { getPlainValue } from "./cell-value";
import { useBinaryFormatContext } from "./context";
import { TableCell } from "./TableCell";
import type { ResultTableColumn, ResultTableRow } from "./types";

export interface VirtualDataBlockHandle {
  scrollTo(index: number): void;
}

export interface VirtualDataBlockProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  activeRowIndex: number;
  isSensitiveColumn: (index: number) => boolean;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  database: Database;
  statement?: string;
  search: SearchParams;
}

/**
 * Vertical "row blocks" view of the result set — each row renders as a
 * stacked card showing every column on its own line. Used when the user
 * toggles list mode in `SingleResultView`.
 */
export const VirtualDataBlock = forwardRef<
  VirtualDataBlockHandle,
  VirtualDataBlockProps
>(function VirtualDataBlock(
  {
    rows,
    columns,
    activeRowIndex,
    getMaskingReason,
    database,
    statement,
    search,
  },
  ref
) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const { getBinaryFormat } = useBinaryFormatContext();

  // Estimate based on column count: header + (per-column line ≈ 28px) +
  // padding. Real measurements come from `measureElement` once mounted.
  const estimateRowHeight = useCallback(
    () => 60 + columns.length * 28,
    [columns.length]
  );

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => containerRef.current,
    estimateSize: estimateRowHeight,
    measureElement: (el) => el.getBoundingClientRect().height,
    overscan: 4,
    // Each row is `position: absolute` (placed by `translateY(start)`),
    // so a margin / padding-bottom on the row div has no effect on the
    // gap to the next row. The virtualizer's `gap` option adds the
    // requested pixels into each row's `start` so cards visibly breathe.
    gap: 16,
  });

  useImperativeHandle(
    ref,
    () => ({
      scrollTo(index: number) {
        virtualizer.scrollToIndex(index, {
          align: "start",
          behavior: "smooth",
        });
      },
    }),
    [virtualizer]
  );

  const buildRowJSON = useCallback(
    (rowIndex: number): string => {
      const obj = columns.reduce(
        (acc, column, columnIndex) => {
          if (!column.name) return acc;
          const binaryFormat = getBinaryFormat({
            rowIndex,
            colIndex: columnIndex,
          });
          acc[column.name] = getPlainValue(
            rows[rowIndex].item.values[columnIndex],
            column.columnType,
            binaryFormat
          );
          return acc;
        },
        {} as Record<string, unknown>
      );
      return JSON.stringify(obj, null, 4);
    },
    [columns, rows, getBinaryFormat]
  );

  return (
    <div
      ref={containerRef}
      className="relative w-full flex-1 min-h-0 overflow-auto rounded-sm border"
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
          const rowIndex = virtualRow.index;
          const row = rows[rowIndex];
          if (!row) return null;
          const isActive = activeRowIndex === rowIndex;
          return (
            <div
              key={virtualRow.key}
              ref={virtualizer.measureElement}
              data-index={rowIndex}
              className="font-mono mx-2 absolute inset-x-0"
              style={{
                top: 0,
                transform: `translateY(${virtualRow.start}px)`,
              }}
            >
              <p className="font-bold text-control-light dark:text-gray-300 overflow-hidden whitespace-nowrap mb-1">
                ******************************** {rowIndex + 1}. row
                ********************************
              </p>
              <div
                className={cn(
                  "py-2 px-3 bg-control-bg dark:bg-gray-700 rounded relative",
                  isActive && "border-2 border-accent/20 bg-accent/10!"
                )}
              >
                <CopyJSONButton
                  getContent={() => buildRowJSON(rowIndex)}
                  label={t("common.copy")}
                />
                {columns.map((column, columnIndex) => {
                  const reason = getMaskingReason?.(columnIndex);
                  return (
                    <div
                      key={column.id}
                      className="flex items-start text-control-light dark:text-gray-300 text-sm"
                    >
                      <div className="min-w-28 text-left flex items-center font-medium pt-1">
                        <div className="flex items-center gap-x-1">
                          {column.name}
                          {reason && (
                            <MaskingReasonPopover
                              reason={reason}
                              statement={statement}
                              database={database.name}
                            />
                          )}
                        </div>
                        :
                      </div>
                      <div className="flex-1">
                        <TableCell
                          value={row.item.values[columnIndex]}
                          keyword={search.query}
                          scope={search.scopes.find(
                            (s) => s.id === columns[columnIndex]?.id
                          )}
                          rowIndex={rowIndex}
                          colIndex={columnIndex}
                          columnType={column.columnType}
                          allowSelect
                          database={database}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
});

function CopyJSONButton({
  getContent,
  label,
}: {
  getContent: () => string;
  label: string;
}) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(getContent());
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // ignore
    }
  };
  return (
    <div className="absolute right-2 top-2 z-10 opacity-70 hover:opacity-100">
      <Tooltip content={label}>
        {/*
         * In admin mode the row card is `dark:bg-gray-700`, and the
         * `ghost` variant's `text-control` icon nearly disappears
         * against it. Force a light icon + visible hover surface inside
         * the `.dark` parent so the action is reachable.
         */}
        <Button
          size="sm"
          variant="ghost"
          className="size-7 p-0 dark:text-gray-100 dark:hover:bg-gray-600"
          onClick={handleCopy}
        >
          {copied ? (
            <CheckIcon className="size-4" />
          ) : (
            <CopyIcon className="size-4" />
          )}
        </Button>
      </Tooltip>
    </div>
  );
}
