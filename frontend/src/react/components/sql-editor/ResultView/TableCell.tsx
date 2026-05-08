import { ExpandIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import type { SearchScope } from "@/utils/v1/advanced-search/common";
import { getInstanceResource } from "@/utils/v1/database";
import { BinaryFormatButton } from "./BinaryFormatButton";
import type { BinaryFormat } from "./binary-format";
import { getPlainValue } from "./cell-value";
import {
  useBinaryFormatContext,
  useSelectionContext,
  useSQLResultViewContext,
} from "./context";

interface TableCellProps {
  value: RowValue;
  rowIndex: number;
  colIndex: number;
  allowSelect?: boolean;
  columnType: string;
  database: Database;
  scope?: SearchScope;
  keyword: string;
}

export function TableCell({
  value,
  rowIndex,
  colIndex,
  allowSelect: allowSelectProp,
  columnType,
  database,
  scope,
  keyword,
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

  // ResizeObserver replaces Vue's `useResizeObserver(cellRef, ...)`.
  useEffect(() => {
    const cell = cellRef.current;
    const wrapper = wrapperRef.current;
    if (!cell || !wrapper) return;
    const measure = () => {
      const verticalTruncated = wrapper.scrollHeight > wrapper.offsetHeight + 2;
      const horizontalTruncated = wrapper.scrollWidth > cell.offsetWidth + 2;
      setTruncated(verticalTruncated || horizontalTruncated);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(cell);
    return () => observer.disconnect();
  }, []);

  const clickable = useMemo(() => {
    if (truncated) return true;
    const eng = getInstanceResource(database).engine;
    if (eng === Engine.MONGODB || eng === Engine.ELASTICSEARCH) {
      const maybeJSON = String(value).trim();
      return (
        (maybeJSON.startsWith("{") && maybeJSON.endsWith("}")) ||
        (maybeJSON.startsWith("[") && maybeJSON.endsWith("]"))
      );
    }
    return false;
  }, [truncated, database, value]);

  const selected = useMemo(() => {
    if (!allowSelect) return false;
    const { columns, rows } = selectionState;
    if (columns.length === 1 && rows.length === 1) {
      return columns[0] === colIndex && rows[0] === rowIndex;
    }
    return columns.includes(colIndex) || rows.includes(rowIndex);
  }, [allowSelect, selectionState, colIndex, rowIndex]);

  const plainValue = useMemo(
    () => getPlainValue(value, columnType, binaryFormat),
    [value, columnType, binaryFormat]
  );

  const showDetail = () => {
    setDetail({ row: rowIndex, col: colIndex });
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

  let inner: React.ReactNode;
  if (plainValue === undefined) {
    inner = <span className="text-control-placeholder italic">UNSET</span>;
  } else if (plainValue === null) {
    inner = <span className="text-control-placeholder italic">NULL</span>;
  } else if (plainValue.length === 0) {
    inner = <br style={{ minWidth: "1rem", display: "inline-flex" }} />;
  } else {
    inner = <HighlightLabelText text={plainValue} keyword={activeKeyword} />;
  }

  return (
    <div
      ref={cellRef}
      onClick={handleClick}
      onDoubleClick={showDetail}
      className={cn(
        "px-2 py-1 flex items-center",
        allowSelect ? "cursor-pointer hover:bg-accent/10" : "select-none",
        selected && "bg-accent/20!"
      )}
    >
      <div
        ref={wrapperRef}
        className="font-mono text-start wrap-break-word line-clamp-3"
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
        {clickable && (
          <Button
            size="sm"
            variant="outline"
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
