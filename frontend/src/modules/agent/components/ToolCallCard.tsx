import { Rows3Icon, Table2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PlainCellValue } from "@/components/PlainCellValue";
import { QueryResultVerticalBody } from "@/components/QueryResultVerticalBody";
import { QueryResultVerticalRow } from "@/components/QueryResultVerticalRow";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { getPlainValue } from "@/modules/sql-editor/components/ResultView/cell-value";
import { QueryResultTableGrid } from "@/modules/sql-editor/components/ResultView/QueryResultTableGrid";
import type {
  ResultTableColumn,
  ResultTableRow,
} from "@/modules/sql-editor/components/ResultView/types";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { extractSQLRowValuePlain } from "@/utils/v1/sql";
import { parseCallApiResult } from "../logic/callApiResult";
import {
  type AgentQueryResultPreview,
  type AgentQueryResultSetPreview,
  parseAgentQueryResult,
} from "../logic/queryResult";
import type {
  AgentAskUserOption,
  AgentAskUserResponse,
  ToolCall,
} from "../logic/types";

interface Props {
  toolCall: ToolCall;
  result?: string;
}

const parseJson = (value?: string): unknown => {
  if (!value) return undefined;
  try {
    return JSON.parse(value);
  } catch {
    return value;
  }
};

const formatJson = (value?: string): string => {
  if (!value) return "";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
};

const isQueryOperation = (value: unknown): boolean =>
  value === "SQLService/Query" || value === "bytebase.v1.SQLService.Query";

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
      {rows.length === 0 && (
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
      {resultSet.rows.length === 0 && (
        <div className="px-2 py-3 text-center text-control-light">
          {t("common.no-data")}
        </div>
      )}
    </>
  );
}

function QueryResultPreview({ preview }: { preview: AgentQueryResultPreview }) {
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
              {resultSet.rowsTruncated
                ? t("agent.query-result.showing-rows", {
                    shown: resultSet.displayedRowCount,
                    total: resultSet.returnedRowCount,
                  })
                : t("agent.query-result.returned-rows", {
                    count: resultSet.returnedRowCount,
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

const parseAskUserOption = (value: unknown): AgentAskUserOption | null => {
  if (!value || typeof value !== "object") return null;

  const option = value as Record<string, unknown>;
  const label =
    typeof option.label === "string"
      ? option.label.trim()
      : typeof option.value === "string"
        ? option.value.trim()
        : "";
  const optionValue =
    typeof option.value === "string" ? option.value.trim() : "";

  if (!label || !optionValue) return null;

  return {
    label,
    value: optionValue,
    description:
      typeof option.description === "string" && option.description.trim()
        ? option.description.trim()
        : undefined,
  };
};

export function ToolCallCard({ toolCall, result }: Props) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const resultText = result ?? "";
  const parsedArguments = useMemo(
    () => parseJson(toolCall.arguments),
    [toolCall.arguments]
  );
  const parsedResult = useMemo(() => parseJson(resultText), [resultText]);
  const hasErrorResult =
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    typeof (parsedResult as Record<string, unknown>).error === "string";
  const operationId =
    typeof parsedArguments === "object" &&
    parsedArguments !== null &&
    !Array.isArray(parsedArguments) &&
    typeof (parsedArguments as Record<string, unknown>).operationId === "string"
      ? ((parsedArguments as Record<string, unknown>).operationId as string)
      : undefined;
  const callApiError =
    toolCall.name === "call_api"
      ? parseCallApiResult(resultText, operationId).error
      : undefined;
  const queryPreview = useMemo(() => {
    if (
      toolCall.name !== "call_api" ||
      typeof parsedArguments !== "object" ||
      parsedArguments === null ||
      !isQueryOperation(
        (parsedArguments as Record<string, unknown>).operationId
      )
    ) {
      return undefined;
    }
    const preview = parseAgentQueryResult(resultText);
    return preview?.resultSets.some(
      (resultSet) => resultSet.columns.length > 0 || resultSet.error
    )
      ? preview
      : undefined;
  }, [parsedArguments, resultText, toolCall.name]);
  const isAskUser = toolCall.name === "ask_user";
  const isDone = toolCall.name === "done";
  const isCanceledAsk =
    isAskUser &&
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    (parsedResult as Record<string, unknown>).kind === "cancelled";

  const askPrompt = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "prompt" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).prompt === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).prompt as string;
    }
    return "";
  }, [parsedArguments]);

  const askKind = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "kind" in parsedArguments
    ) {
      const kind = (parsedArguments as Record<string, unknown>).kind;
      if (kind === "confirm") return "confirm" as const;
      if (kind === "choose") return "choose" as const;
    }
    return "input" as const;
  }, [parsedArguments]);

  const askDefaultValue = useMemo(() => {
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "defaultValue" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).defaultValue ===
        "string"
    ) {
      return (parsedArguments as Record<string, unknown>)
        .defaultValue as string;
    }
    return "";
  }, [parsedArguments]);

  const askOptions = useMemo<AgentAskUserOption[]>(() => {
    if (
      typeof parsedArguments !== "object" ||
      !parsedArguments ||
      !("options" in parsedArguments) ||
      !Array.isArray((parsedArguments as Record<string, unknown>).options)
    ) {
      return [];
    }
    return ((parsedArguments as Record<string, unknown>).options as unknown[])
      .map((option) => parseAskUserOption(option))
      .filter((option): option is AgentAskUserOption => !!option);
  }, [parsedArguments]);

  const askResponse = useMemo<AgentAskUserResponse | null>(() => {
    if (
      typeof parsedResult !== "object" ||
      !parsedResult ||
      !("kind" in parsedResult) ||
      typeof (parsedResult as Record<string, unknown>).kind !== "string" ||
      !("answer" in parsedResult) ||
      typeof (parsedResult as Record<string, unknown>).answer !== "string"
    ) {
      return null;
    }

    const response = parsedResult as Record<string, unknown>;
    if (response.kind === "confirm") {
      if (typeof response.confirmed !== "boolean") return null;
      return response as unknown as AgentAskUserResponse;
    }
    if (response.kind === "choose") {
      if (typeof response.value !== "string") return null;
      return response as unknown as AgentAskUserResponse;
    }
    if (response.kind === "input") {
      return response as unknown as AgentAskUserResponse;
    }
    return null;
  }, [parsedResult]);

  const doneText = useMemo(() => {
    if (
      typeof parsedResult === "object" &&
      parsedResult !== null &&
      !Array.isArray(parsedResult) &&
      typeof (parsedResult as Record<string, unknown>).text === "string"
    ) {
      return (parsedResult as Record<string, unknown>).text as string;
    }
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "text" in parsedArguments &&
      typeof (parsedArguments as Record<string, unknown>).text === "string"
    ) {
      return (parsedArguments as Record<string, unknown>).text as string;
    }
    return "";
  }, [parsedArguments, parsedResult]);

  const doneSuccess = useMemo(() => {
    if (
      typeof parsedResult === "object" &&
      parsedResult !== null &&
      !Array.isArray(parsedResult) &&
      typeof (parsedResult as Record<string, unknown>).success === "boolean"
    ) {
      return (parsedResult as Record<string, unknown>).success as boolean;
    }
    if (
      typeof parsedArguments === "object" &&
      parsedArguments &&
      "success" in parsedArguments
    ) {
      return (parsedArguments as Record<string, unknown>).success !== false;
    }
    return true;
  }, [parsedArguments, parsedResult]);
  const doneHasApiWarnings =
    doneSuccess &&
    typeof parsedResult === "object" &&
    parsedResult !== null &&
    !Array.isArray(parsedResult) &&
    typeof (parsedResult as Record<string, unknown>).failedApiCallCount ===
      "number" &&
    ((parsedResult as Record<string, unknown>).failedApiCallCount as number) >
      0;
  const renderStatus = () => {
    if (hasErrorResult || callApiError) {
      return <span className="text-error">{t("agent.tool-failed")}</span>;
    }
    if (isAskUser) {
      if (isCanceledAsk) {
        return (
          <span className="text-control-light">
            {t("agent.tool-request-canceled")}
          </span>
        );
      }
      return resultText ? (
        <span className="text-success">
          {t("agent.tool-response-submitted")}
        </span>
      ) : (
        <span className="text-warning">{t("agent.tool-ask-user")}</span>
      );
    }
    if (isDone) {
      if (doneHasApiWarnings) {
        return (
          <span className="text-warning">
            {t("agent.tool-completed-with-warnings")}
          </span>
        );
      }
      return (
        <span className={doneSuccess ? "text-success" : "text-error"}>
          {doneSuccess ? t("agent.tool-completed") : t("agent.tool-failed")}
        </span>
      );
    }
    return resultText ? (
      <span className="text-success">&#10003;</span>
    ) : (
      <span className="animate-pulse text-control-placeholder">&#9679;</span>
    );
  };

  const renderExpandedContent = () => {
    if (isAskUser) {
      return (
        <>
          <div className="text-control-light">
            {askKind === "confirm"
              ? t("agent.tool-ask-user-confirm")
              : askKind === "choose"
                ? t("agent.tool-ask-user-choose")
                : t("agent.tool-ask-user-input")}
          </div>
          <div className="text-control-light">{t("agent.tool-prompt")}</div>
          <pre className="whitespace-pre-wrap break-all text-control">
            {askPrompt}
          </pre>
          {askDefaultValue && (
            <>
              <div className="text-control-light">
                {t("agent.tool-default-value")}
              </div>
              <pre className="whitespace-pre-wrap break-all text-control">
                {askDefaultValue}
              </pre>
            </>
          )}
          {askKind === "choose" && askOptions.length > 0 && (
            <>
              <div className="text-control-light">
                {t("agent.tool-options")}
              </div>
              <div className="flex flex-col gap-1">
                {askOptions.map((option) => (
                  <div
                    key={option.value}
                    className="rounded-sm border border-block-border bg-background px-2 py-1"
                  >
                    <div className="font-medium text-control">
                      {option.label}
                    </div>
                    <div className="text-control-light">{option.value}</div>
                    {option.description && (
                      <div className="text-control-light">
                        {option.description}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}
          {askResponse && (
            <>
              <div className="text-control-light">{t("agent.tool-answer")}</div>
              <pre className="whitespace-pre-wrap break-all text-control">
                {askResponse.answer}
              </pre>
              {askResponse.kind === "choose" && (
                <>
                  <div className="text-control-light">
                    {t("agent.tool-value")}
                  </div>
                  <pre className="whitespace-pre-wrap break-all text-control">
                    {askResponse.value}
                  </pre>
                </>
              )}
            </>
          )}
        </>
      );
    }

    if (isDone) {
      return (
        <>
          <div className="text-control-light">{t("agent.result")}</div>
          <pre className="whitespace-pre-wrap break-all text-control">
            {doneText}
          </pre>
        </>
      );
    }

    return (
      <>
        <div className="text-control-light">{t("agent.args")}</div>
        <pre className="whitespace-pre-wrap break-all text-control">
          {formatJson(toolCall.arguments)}
        </pre>
        {resultText && (
          <>
            <div className="text-control-light">{t("agent.result")}</div>
            <pre className="max-h-32 overflow-y-auto whitespace-pre-wrap break-all text-control">
              {formatJson(resultText)}
            </pre>
          </>
        )}
      </>
    );
  };

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setExpanded}
      className="rounded-sm border border-block-border bg-control-bg text-xs"
    >
      <CollapsibleTrigger className="px-2 py-1.5">
        <span className="font-mono text-control-light">{toolCall.name}</span>
        {renderStatus()}
        <span className="ml-auto text-control-placeholder">
          {expanded ? "\u25BE" : "\u25B8"}
        </span>
      </CollapsibleTrigger>

      {queryPreview && <QueryResultPreview preview={queryPreview} />}

      <CollapsiblePanel className="flex flex-col gap-1 border-t border-block-border px-2 py-1.5">
        {renderExpandedContent()}
      </CollapsiblePanel>
    </Collapsible>
  );
}
