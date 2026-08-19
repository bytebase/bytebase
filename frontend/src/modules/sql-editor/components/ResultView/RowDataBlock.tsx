import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { MaskingReasonPopover } from "@/modules/sql-editor/components/MaskingReasonPopover";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import type { ResultTableColumn } from "./types";

interface RowDataBlockProps {
  columns: ResultTableColumn[];
  database: string;
  statement?: string;
  getMaskingReason?: (index: number) => MaskingReason | undefined;
  renderColumnName?: (column: ResultTableColumn, index: number) => ReactNode;
  renderValue: (column: ResultTableColumn, index: number) => ReactNode;
  actions?: ReactNode;
  className?: string;
}

export function RowDataBlock({
  columns,
  database,
  statement,
  getMaskingReason,
  renderColumnName = (column) => column.name,
  renderValue,
  actions,
  className,
}: RowDataBlockProps) {
  return (
    <div
      data-testid="row-data-block"
      className={cn("relative rounded bg-control-bg/40 px-3 py-2", className)}
    >
      {actions}
      {columns.map((column, columnIndex) => {
        const reason = getMaskingReason?.(columnIndex);
        return (
          <div
            key={`${column.id}-${columnIndex}`}
            className="flex items-start text-sm text-control-light"
          >
            <div className="flex min-w-28 items-center pt-1 text-left font-medium">
              <div className="flex items-center gap-x-1">
                {renderColumnName(column, columnIndex)}
                {reason && (
                  <MaskingReasonPopover
                    reason={reason}
                    statement={statement}
                    database={database}
                  />
                )}
              </div>
              {": "}
            </div>
            <div className="min-w-0 flex-1">
              {renderValue(column, columnIndex)}
            </div>
          </div>
        );
      })}
    </div>
  );
}
