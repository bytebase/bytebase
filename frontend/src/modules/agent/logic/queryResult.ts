import { fromJson } from "@bufbuild/protobuf";
import {
  QueryResponseSchema,
  type QueryRow,
} from "@/types/proto-es/v1/sql_service_pb";

const MAX_RESULT_SETS = 3;
const MAX_ROWS = 100;
const MAX_COLUMNS = 20;

export interface AgentQueryResultSetPreview {
  columns: string[];
  columnTypes: string[];
  rows: QueryRow[];
  returnedRowCount: number;
  displayedRowCount: number;
  returnedColumnCount: number;
  displayedColumnCount: number;
  rowsTruncated: boolean;
  columnsTruncated: boolean;
  error?: string;
}

export interface AgentQueryResultPreview {
  resultSets: AgentQueryResultSetPreview[];
  totalResultSets: number;
  resultSetsTruncated: boolean;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

export const parseAgentQueryResult = (
  raw: string
): AgentQueryResultPreview | undefined => {
  try {
    const apiResult: unknown = JSON.parse(raw);
    if (!isRecord(apiResult)) return undefined;
    const { status, response } = apiResult;
    if (
      typeof status !== "number" ||
      status < 200 ||
      status >= 300 ||
      !isRecord(response) ||
      !Array.isArray(response.results)
    ) {
      return undefined;
    }

    const displayableResults = response.results.filter((rawResult) => {
      if (!isRecord(rawResult)) throw new Error("Invalid query result");
      const columns = rawResult.columnNames ?? [];
      const rows = rawResult.rows ?? [];
      if (!Array.isArray(columns) || !Array.isArray(rows)) {
        throw new Error("Invalid query result rows or columns");
      }
      return (
        columns.length > 0 ||
        (typeof rawResult.error === "string" && rawResult.error.length > 0)
      );
    });
    const rawResults = displayableResults.slice(0, MAX_RESULT_SETS);
    const counts: { rows: number; columns: number }[] = [];
    const boundedResults = rawResults.map((rawResult) => {
      if (!isRecord(rawResult)) throw new Error("Invalid query result");
      const columns = rawResult.columnNames ?? [];
      const rows = rawResult.rows ?? [];
      if (!Array.isArray(columns) || !Array.isArray(rows)) {
        throw new Error("Invalid query result rows or columns");
      }
      counts.push({ rows: rows.length, columns: columns.length });
      return {
        columnNames: columns.slice(0, MAX_COLUMNS),
        columnTypeNames: Array.isArray(rawResult.columnTypeNames)
          ? rawResult.columnTypeNames.slice(0, MAX_COLUMNS)
          : [],
        rows: rows.slice(0, MAX_ROWS).map((row) => {
          if (!isRecord(row)) throw new Error("Invalid query row");
          const values = row.values ?? [];
          if (!Array.isArray(values)) throw new Error("Invalid query values");
          return { values: values.slice(0, MAX_COLUMNS) };
        }),
        ...(typeof rawResult.error === "string"
          ? { error: rawResult.error }
          : {}),
      };
    });
    const responsePreview = fromJson(QueryResponseSchema, {
      results: boundedResults,
    });

    return {
      resultSets: responsePreview.results.map((result, index) => {
        const count = counts[index];
        return {
          columns: result.columnNames,
          columnTypes: result.columnTypeNames,
          rows: result.rows,
          returnedRowCount: count.rows,
          displayedRowCount: result.rows.length,
          returnedColumnCount: count.columns,
          displayedColumnCount: result.columnNames.length,
          rowsTruncated: count.rows > MAX_ROWS,
          columnsTruncated: count.columns > MAX_COLUMNS,
          ...(result.error ? { error: result.error } : {}),
        };
      }),
      totalResultSets: displayableResults.length,
      resultSetsTruncated: displayableResults.length > MAX_RESULT_SETS,
    };
  } catch {
    return undefined;
  }
};
