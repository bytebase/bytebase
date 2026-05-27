import type { QueryResult } from "@/types/proto-es/v1/sql_service_pb";
import { downloadError } from "../types";
import { xlsxValueFromRowValue } from "../value";

// Excel / ECMA-376 hard limit: column XFD (16,384). ExcelJS's col-cache
// throws "Out of bounds. Excel supports columns from 1 to 16384" once
// getCell(_, 16385) is invoked.
const EXCEL_MAX_COLUMN = 16384;
// Excel's hard worksheet row limit (2^20). The serializer writes a header in
// row 1 and data in rows 2..N+1, so max data rows = EXCEL_MAX_ROWS - 1.
// Writing past this produces a workbook Excel cannot reliably open.
const EXCEL_MAX_ROWS = 1_048_576;

export async function serializeXLSX(result: QueryResult): Promise<Uint8Array> {
  // Column and row overflows are "result too large for THIS format" — the
  // user can still download CSV/JSON/SQL of the same data, or use Export
  // for a server-streamed flow. ResultTooLarge keeps the message verbatim
  // through downloadErrorMessage so the actual counts reach the user.
  if (result.columnNames.length > EXCEL_MAX_COLUMN) {
    throw downloadError(
      "ResultTooLarge",
      `XLSX cannot exceed ${EXCEL_MAX_COLUMN} columns; got ${result.columnNames.length}. Use Export or a different format.`,
      undefined,
      {
        key: "sql-editor.download-too-large-xlsx-columns",
        params: {
          limit: EXCEL_MAX_COLUMN,
          count: result.columnNames.length,
        },
      }
    );
  }
  if (result.rows.length > EXCEL_MAX_ROWS - 1) {
    throw downloadError(
      "ResultTooLarge",
      `XLSX cannot exceed ${EXCEL_MAX_ROWS - 1} data rows; got ${result.rows.length}. Use Export or a different format.`,
      undefined,
      {
        key: "sql-editor.download-too-large-xlsx-rows",
        params: {
          limit: (EXCEL_MAX_ROWS - 1).toLocaleString(),
          count: result.rows.length.toLocaleString(),
        },
      }
    );
  }
  // ExcelJS ships as CJS. Vite's dep-prebundle exposes only the default
  // export, while Node native ESM exposes both `default` and the named
  // members. Destructure `default` to work in both — without this, the
  // prod Vite build hits `new undefined()` and every XLSX download fails
  // while vitest (which uses Node's resolver, not Vite's prebundle) passes.
  const { default: ExcelJS } = await import("exceljs");
  const book = new ExcelJS.Workbook();
  const sheet = book.addWorksheet("Sheet1");

  // Headers in row 1
  for (let j = 0; j < result.columnNames.length; j++) {
    sheet.getCell(1, j + 1).value = result.columnNames[j];
  }
  // Rows from row 2; every cell stored as text (parity with backend).
  for (let i = 0; i < result.rows.length; i++) {
    const row = result.rows[i];
    for (let j = 0; j < row.values.length; j++) {
      sheet.getCell(i + 2, j + 1).value = xlsxValueFromRowValue(row.values[j]);
    }
  }
  const buf = await book.xlsx.writeBuffer();
  return new Uint8Array(buf as ArrayBuffer);
}
