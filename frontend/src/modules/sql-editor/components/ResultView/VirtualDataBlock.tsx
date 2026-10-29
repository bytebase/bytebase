import { CheckIcon, CopyIcon } from "lucide-react";
import { forwardRef, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { QueryResultVerticalBody } from "@/components/QueryResultVerticalBody";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { writeTextToClipboard } from "@/lib/clipboard";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { MaskingReason } from "@/types/proto-es/v1/sql_service_pb";
import type { SearchParams } from "@/utils/v1/advanced-search/common";
import { getPlainValue } from "./cell-value";
import { useBinaryFormatContext, useSQLResultViewContext } from "./context";
import { ResultCopyContextMenu } from "./ResultCopyContextMenu";
import { RowDataBlock } from "./RowDataBlock";
import { TableCell } from "./TableCell";
import type { ResultTableColumn, ResultTableRow } from "./types";

export interface VirtualDataBlockHandle {
  scrollTo(index: number): void;
}

export interface VirtualDataBlockProps {
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  activeRowIndex: number;
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
  const { getBinaryFormat } = useBinaryFormatContext();
  const { setDetail } = useSQLResultViewContext();

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
    <ResultCopyContextMenu>
      <QueryResultVerticalBody
        ref={ref}
        rowCount={rows.length}
        columnCount={columns.length}
        renderRow={(rowIndex) => {
          const row = rows[rowIndex];
          if (!row) return null;
          const isActive = activeRowIndex === rowIndex;
          return (
            <RowDataBlock
              columns={columns}
              database={database.name}
              statement={statement}
              getMaskingReason={getMaskingReason}
              className={cn(
                isActive && "border-2 border-accent/20 bg-accent/10!"
              )}
              actions={
                <CopyJSONButton
                  getContent={() => buildRowJSON(rowIndex)}
                  label={t("common.copy")}
                />
              }
              renderValue={(column, columnIndex) => (
                <div
                  onDoubleClick={(event) => {
                    event.stopPropagation();
                    setDetail({
                      row: rowIndex,
                      col: columnIndex,
                      view: "cell",
                    });
                  }}
                >
                  <TableCell
                    value={row.item.values[columnIndex]}
                    keyword={search.query}
                    scope={search.scopes.find(
                      (scope) => scope.id === column.id
                    )}
                    rowIndex={rowIndex}
                    colIndex={columnIndex}
                    columnType={column.columnType}
                    allowSelect
                  />
                </div>
              )}
            />
          );
        }}
      />
    </ResultCopyContextMenu>
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
    if (await writeTextToClipboard(getContent())) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } else {
      // ignore
    }
  };
  return (
    <div className="absolute right-2 top-2 z-10 opacity-70 hover:opacity-100">
      <Tooltip content={label}>
        <Button
          size="sm"
          appearance="secondary"
          aria-label={label}
          className="w-7 p-0"
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
