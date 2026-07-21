import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import { generateInsertStatementFromRows } from "@/utils/v1/sql";
import { isSingleCellSelected, type SelectionState } from "./selection-math";
import type { ResultTableColumn, ResultTableRow } from "./types";

export type CopyScope = "all" | "selected";

/**
 * Everything a copy formatter needs to render the clipboard payload. The
 * provider builds this once per copy and hands it to the chosen formatter.
 */
export interface CopyFormatterContext {
  scope: CopyScope;
  selection: SelectionState;
  rows: ResultTableRow[];
  columns: ResultTableColumn[];
  engine: Engine;
  // Connected schema, used to qualify the INSERT table name.
  schema?: string;
  // Display string for one cell, honoring binary-format overrides. Returns ""
  // for a missing cell.
  getFormattedValue: (rowIndex: number, colIndex: number) => string;
}

export type CopyFormatter = (ctx: CopyFormatterContext) => string;

// Placeholder table name for the generated INSERT: arbitrary result queries
// (joins, expressions) have no single source table, so the user fills this in.
const INSERT_TABLE_PLACEHOLDER = "<table_name>";

// Row indices for row-oriented formats (CSV / SQL): the selected rows
// (ascending) when scope is "selected" and any are selected, otherwise all.
const resolveRowIndices = (ctx: CopyFormatterContext): number[] => {
  if (ctx.scope === "selected" && ctx.selection.rows.length > 0) {
    return [...ctx.selection.rows].sort((a, b) => a - b);
  }
  return ctx.rows.map((_, i) => i);
};

const escapeTSVValue = (val: string): string =>
  /[\t\n"]/.test(val) ? `"${val.replaceAll('"', '""')}"` : val;

const escapeCSVValue = (val: string): string =>
  /[,\n\r"]/.test(val) ? `"${val.replaceAll('"', '""')}"` : val;

/**
 * Default "plain" copy: tab-separated values mirroring the in-grid selection —
 * a single cell, whole rows (with a leading `index` column), or whole columns.
 * Scope "all" copies every row.
 */
export const formatAsText: CopyFormatter = (ctx) => {
  const { columns, rows, getFormattedValue } = ctx;
  const state: SelectionState =
    ctx.scope === "all"
      ? { rows: rows.map((_, i) => i), columns: [] }
      : ctx.selection;

  if (isSingleCellSelected(state)) {
    return getFormattedValue(state.rows[0], state.columns[0]);
  }

  if (state.rows.length > 0) {
    const header = ["index", ...columns.map((c) => c.name)];
    const lines: string[] = [];
    for (const rowIndex of state.rows) {
      if (!rows[rowIndex]) continue;
      const cells = columns
        .map((_, colIdx) => escapeTSVValue(getFormattedValue(rowIndex, colIdx)))
        .join("\t");
      lines.push(`${rowIndex}\t${cells}`);
    }
    if (lines.length === 0) return "";
    return `${header.join("\t")}\n${lines.join("\n")}`;
  }

  if (state.columns.length > 0) {
    const header = state.columns.map((i) => columns[i]?.name ?? "");
    const lines: string[] = [];
    for (let rowIdx = 0; rowIdx < rows.length; rowIdx++) {
      const cells = state.columns.map((colIdx) =>
        escapeTSVValue(getFormattedValue(rowIdx, colIdx))
      );
      lines.push(cells.join("\t"));
    }
    if (lines.length === 0) return "";
    return `${header.join("\t")}\n${lines.join("\n")}`;
  }

  return "";
};

/**
 * CSV copy: a header row of column names plus one comma-separated line per row,
 * RFC-4180 quoted. Always all columns.
 */
export const formatAsCSV: CopyFormatter = (ctx) => {
  const { columns, rows, getFormattedValue } = ctx;
  const rowIndices = resolveRowIndices(ctx);
  const lines = [columns.map((c) => escapeCSVValue(c.name)).join(",")];
  for (const rowIndex of rowIndices) {
    if (!rows[rowIndex]) continue;
    lines.push(
      columns
        .map((_, colIdx) => escapeCSVValue(getFormattedValue(rowIndex, colIdx)))
        .join(",")
    );
  }
  return lines.join("\n");
};

/**
 * SQL copy: a single batched, engine-aware INSERT statement. Always all
 * columns. Returns "" when there are no rows to copy.
 */
export const formatAsSQL: CopyFormatter = (ctx) => {
  const { columns, rows, engine, schema } = ctx;
  const rowValues = resolveRowIndices(ctx)
    .map((i) => rows[i]?.item.values)
    .filter((values): values is RowValue[] => values !== undefined);
  if (rowValues.length === 0) return "";
  return generateInsertStatementFromRows({
    engine,
    schema,
    table: INSERT_TABLE_PLACEHOLDER,
    columns: columns.map((c) => c.name),
    rows: rowValues,
  });
};
