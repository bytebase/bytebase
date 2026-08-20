import { ExpandIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import type { SearchScope } from "@/utils/v1/advanced-search/common";
import { BinaryFormatButton } from "./BinaryFormatButton";
import type { BinaryFormat } from "./binary-format";
import { getPlainValue, isLikelyJSON } from "./cell-value";
import {
  useBinaryFormatContext,
  useSelectionContext,
  useSQLResultViewContext,
} from "./context";
import { PlainCellValue } from "./PlainCellValue";

interface TableCellProps {
  value: RowValue;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
  columnType: string;
  scope?: SearchScope;
  keyword: string;
  showDetailAction?: boolean;
}

export function TableCell({
  value,
  rowIndex,
  colIndex,
  allowSelect: allowSelectProp,
  columnType,
  scope,
  keyword,
  showDetailAction = true,
}: TableCellProps) {
  const { setDetail } = useSQLResultViewContext();
  const { getBinaryFormat, setBinaryFormat } = useBinaryFormatContext();
  const {
    state: selectionState,
    disabled: selectionDisabled,
    toggleSelectCell,
    toggleSelectRow,
  } = useSelectionContext();

  const cellRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [truncated, setTruncated] = useState(false);

  const allowSelect = !!allowSelectProp && !selectionDisabled;
  const hasByteData = value.kind?.case === "bytesValue";

  const binaryFormat = getBinaryFormat({ rowIndex, colIndex });
  const plainValue = useMemo(
    () => getPlainValue(value, columnType, binaryFormat),
    [value, columnType, binaryFormat]
  );

  // ResizeObserver replaces Vue's `useResizeObserver(cellRef, ...)`.
  useEffect(() => {
    const cell = cellRef.current;
    const wrapper = wrapperRef.current;
    if (!cell || !wrapper) return;
    const measure = () => {
      const verticalTruncated = wrapper.scrollHeight > wrapper.offsetHeight + 2;
      const availableWidth = Math.min(cell.offsetWidth, wrapper.offsetWidth);
      const horizontalTruncated = wrapper.scrollWidth > availableWidth + 2;
      setTruncated(verticalTruncated || horizontalTruncated);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(cell);
    return () => observer.disconnect();
  }, []);

  const clickable = useMemo(
    () => truncated || isLikelyJSON(plainValue),
    [plainValue, truncated]
  );

  const selected = useMemo(() => {
    if (!allowSelect) return false;
    const { columns, rows } = selectionState;
    if (columns.length === 1 && rows.length === 1) {
      return columns[0] === colIndex && rows[0] === rowIndex;
    }
    return columns.includes(colIndex) || rows.includes(rowIndex);
  }, [allowSelect, selectionState, colIndex, rowIndex]);

  const showDetail = () => {
    setDetail({ row: rowIndex, col: colIndex, view: "cell" });
  };

  const handleClick = (e: React.MouseEvent) => {
    if (!allowSelect) return;
    if (window.getSelection()?.toString()) return;
    if (e.ctrlKey || e.metaKey) {
      toggleSelectRow(rowIndex);
    } else {
      toggleSelectCell(rowIndex, colIndex);
    }
    e.stopPropagation();
  };

  const activeKeyword = (scope?.value || keyword).trim();

  const inner = (
    <PlainCellValue value={plainValue}>
      {plainValue !== undefined && plainValue !== null ? (
        <HighlightLabelText text={plainValue} keyword={activeKeyword} />
      ) : undefined}
    </PlainCellValue>
  );

  return (
    <div
      ref={cellRef}
      onClick={handleClick}
      className={cn(
        "relative w-full h-full px-2 py-1 flex items-center",
        allowSelect ? "cursor-pointer hover:bg-accent/10" : "select-none",
        selected && "bg-accent/20!"
      )}
    >
      <div
        ref={wrapperRef}
        className={cn(
          "font-mono text-start whitespace-pre line-clamp-3",
          (hasByteData || (clickable && showDetailAction)) &&
            "max-w-[calc(100%-1.5rem)]",
          hasByteData &&
            clickable &&
            showDetailAction &&
            "max-w-[calc(100%-3.25rem)]"
        )}
      >
        {inner}
      </div>
      <div className="absolute right-1 top-1/2 -translate-y-[45%] flex items-center gap-1">
        {hasByteData && (
          <BinaryFormatButton
            format={binaryFormat}
            onFormatChange={(format: BinaryFormat) =>
              setBinaryFormat({ colIndex, rowIndex, format })
            }
          />
        )}
        {clickable && showDetailAction && (
          <Button
            size="sm"
            appearance="outline"
            className="size-6 p-0 rounded-full shadow opacity-90 hover:opacity-100"
            onClick={(e) => {
              e.stopPropagation();
              showDetail();
            }}
          >
            <ExpandIcon className="size-3" />
          </Button>
        )}
      </div>
    </div>
  );
}
