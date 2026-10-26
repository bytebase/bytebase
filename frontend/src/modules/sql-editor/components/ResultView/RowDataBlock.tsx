import type { ReactNode } from "react";
import { QueryResultVerticalRow } from "@/components/QueryResultVerticalRow";
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
    <QueryResultVerticalRow
      data-testid="row-data-block"
      columns={columns}
      renderColumnName={renderColumnName}
      renderColumnAccessory={(_, columnIndex) => {
        const reason = getMaskingReason?.(columnIndex);
        return reason ? (
          <MaskingReasonPopover
            reason={reason}
            statement={statement}
            database={database}
          />
        ) : null;
      }}
      renderValue={renderValue}
      actions={actions}
      className={className}
    />
  );
}
