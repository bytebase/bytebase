import { Rows3Icon, Table2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PlainCellValue } from "@/components/PlainCellValue";
import { QueryResultVerticalBody } from "@/components/QueryResultVerticalBody";
import { QueryResultVerticalRow } from "@/components/QueryResultVerticalRow";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { getPlainValue } from "@/modules/sql-editor/components/ResultView/cell-value";
import { QueryResultTableGrid } from "@/modules/sql-editor/components/ResultView/QueryResultTableGrid";
import type {
  ResultTableColumn,
  ResultTableRow,
} from "@/modules/sql-editor/components/ResultView/types";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import type {
  AgentQueryResultPreview,
  AgentQueryResultSetPreview,
} from "../logic/queryResult";

type QueryResultLayout = "TABLE" | "VERTICAL";

const queryCellText = (value: RowValue | undefined, columnType?: string) => {
  if (!value) return undefined;
  if (columnType) return getPlainValue(value, columnType, undefined);
  const plain = extractSQLRowValuePlain(value);
  if (plain === null || plain === undefined) return plain;
  return String(plain);
};

function AgentQueryResultTable({
  resultSet,
}: {
  resultSet: AgentQueryResultSetPreview;
}) {
  const { t } = useTranslation();
  const columns = useMemo<ResultTableColumn[]>(
    () =>
      resultSet.columns.map((name, index) => ({
        id: String(index),
        name,
        columnType: resultSet.columnTypes[index] ?? "",
      })),
    [resultSet.columns, resultSet.columnTypes]
  );
  const rows = useMemo<ResultTableRow[]>(
    () => resultSet.rows.map((item, key) => ({ key, item })),
    [resultSet.rows]
  );
  const getRowCellContent = useCallback(
    (columnIndex: number) => {
      for (let i = 0; i < Math.min(3, rows.length); i++) {
        const plain = queryCellText(
          rows[i]?.item.values[columnIndex],
          columns[columnIndex]?.columnType
        );
        if (plain) return plain;
      }
      return undefined;
    },
    [rows, columns]
  );

  return (
    <>
      <QueryResultTableGrid
        rows={rows}
        columns={columns}
        getRowCellContent={getRowCellContent}
        renderHeader={(column) => (
          <span className="min-w-0 truncate select-none" title={column.name}>
            {column.name}
          </span>
        )}
        renderRowIndex={(rowIndex) => (
          <span className="textinfolabel pl-1 pr-1 truncate">
            {rowIndex + 1}
          </span>
        )}
        renderCell={(rowIndex, columnIndex) => {
          const value = rows[rowIndex]?.item.values[columnIndex];
          if (!value) return null;
          const displayValue = queryCellText(
            value,
            columns[columnIndex]?.columnType
          );
          return (
            <div className="relative flex h-full w-full items-center px-2 py-1 select-none">
              <div
                className="line-clamp-3 text-start font-mono whitespace-pre"
                title={displayValue ?? undefined}
              >
                <PlainCellValue value={displayValue} />
              </div>
            </div>
          );
        }}
        className="max-h-80"
      />
      {rows.length === 0 && resultSet.returnedRowCount === 0 && (
        <div className="px-2 py-3 text-center text-control-light">
          {t("common.no-data")}
        </div>
      )}
    </>
  );
}

function AgentQueryResultVertical({
  resultSet,
}: {
  resultSet: AgentQueryResultSetPreview;
}) {
  const { t } = useTranslation();
  const columns = useMemo(
    () => resultSet.columns.map((name, index) => ({ id: String(index), name })),
    [resultSet.columns]
  );

  return (
    <>
      <QueryResultVerticalBody
        rowCount={resultSet.rows.length}
        columnCount={columns.length}
        className="max-h-80"
        renderRow={(rowIndex) => (
          <QueryResultVerticalRow
            columns={columns}
            tooltipLayer="agent"
            renderValue={(_, columnIndex) => {
              const value = resultSet.rows[rowIndex]?.values[columnIndex];
              if (!value) return null;
              const displayValue = queryCellText(
                value,
                resultSet.columnTypes[columnIndex]
              );
              return (
                <div className="relative flex h-full w-full items-center px-2 py-1 select-none">
                  <div
                    className="line-clamp-3 text-start font-mono whitespace-pre"
                    title={displayValue ?? undefined}
                  >
                    <PlainCellValue value={displayValue} />
                  </div>
                </div>
              );
            }}
          />
        )}
      />
      {resultSet.rows.length === 0 && resultSet.returnedRowCount === 0 && (
        <div className="px-2 py-3 text-center text-control-light">
          {t("common.no-data")}
        </div>
      )}
    </>
  );
}

export function AgentQueryResult({
  preview,
}: {
  preview: AgentQueryResultPreview;
}) {
  const { t } = useTranslation();
  const [layout, setLayout] = useState<QueryResultLayout>("TABLE");
  const hasTabularResult = preview.resultSets.some(
    (resultSet) => resultSet.columns.length > 0
  );

  return (
    <div
      className="flex min-w-0 flex-col gap-y-3 border-t border-block-border p-2"
      data-agent-query-result
    >
      {preview.resultSets.map((resultSet, resultIndex) => (
        <div key={resultIndex} className="min-w-0">
          <div className="mb-1 flex flex-wrap items-center justify-between gap-x-2 gap-y-1 text-control-light">
            <div className="flex flex-wrap items-center gap-x-2">
              <span className="font-medium text-control">
                {t("agent.result")}
                {preview.totalResultSets > 1 ? ` #${resultIndex + 1}` : ""}
              </span>
              {resultSet.previewLevel === "omitted" &&
              resultSet.returnedRowCount > 0
                ? t("agent.query-result.omitted-rows", {
                    received: resultSet.returnedRowCount,
                  })
                : resultSet.previewLevel === "sampled" &&
                    resultSet.retainedRowCount < resultSet.returnedRowCount
                  ? t("agent.query-result.sampled-rows", {
                      retained: resultSet.retainedRowCount,
                      received: resultSet.returnedRowCount,
                    })
                  : resultSet.rowsTruncated
                    ? t("agent.query-result.showing-rows", {
                        shown: resultSet.displayedRowCount,
                        total: resultSet.returnedRowCount,
                      })
                    : t("agent.query-result.returned-rows", {
                        count: resultSet.returnedRowCount,
                      })}
              {resultSet.previewLevel === "sampled" &&
                resultSet.retainedRowCount < resultSet.returnedRowCount &&
                resultSet.displayedRowCount < resultSet.retainedRowCount &&
                t("agent.query-result.showing-rows", {
                  shown: resultSet.displayedRowCount,
                  total: resultSet.returnedRowCount,
                })}
              {resultSet.columnsTruncated &&
                t("agent.query-result.showing-columns", {
                  shown: resultSet.displayedColumnCount,
                  total: resultSet.returnedColumnCount,
                })}
            </div>
            {resultIndex === 0 && hasTabularResult && (
              <SegmentedControl<QueryResultLayout>
                value={layout}
                options={[
                  {
                    value: "TABLE",
                    label: (
                      <span className="flex w-7 items-center justify-center py-1">
                        <Table2Icon className="size-4" aria-hidden />
                        <span className="sr-only">
                          {t("sql-editor.table-view")}
                        </span>
                      </span>
                    ),
                    tooltip: t("sql-editor.table-view"),
                  },
                  {
                    value: "VERTICAL",
                    label: (
                      <span className="flex w-7 items-center justify-center py-1">
                        <Rows3Icon className="size-4" aria-hidden />
                        <span className="sr-only">
                          {t("sql-editor.vertical-display")}
                        </span>
                      </span>
                    ),
                    tooltip: t("sql-editor.vertical-display"),
                  },
                ]}
                onValueChange={setLayout}
                ariaLabel={t("sql-editor.result-view-mode")}
                appearance="soft"
                size="xs"
              />
            )}
          </div>
          {resultSet.error ? (
            <div className="text-error" role="alert">
              {resultSet.error}
            </div>
          ) : resultSet.columns.length > 0 && layout === "TABLE" ? (
            <AgentQueryResultTable resultSet={resultSet} />
          ) : resultSet.columns.length > 0 ? (
            <AgentQueryResultVertical resultSet={resultSet} />
          ) : null}
        </div>
      ))}
      {preview.resultSetsTruncated &&
        t("agent.query-result.showing-results", {
          shown: preview.resultSets.length,
          total: preview.totalResultSets,
        })}
    </div>
  );
}
