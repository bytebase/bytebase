// xlsx.test.ts

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ExcelJS from "exceljs";
import { describe, expect, it } from "vitest";
import { serializeXLSX } from "../formats/xlsx";
import { FIXTURES } from "./fixtures";

const here = dirname(fileURLToPath(import.meta.url));

/** Normalize a cell string for comparison:
 *  - Replace U+FFFD (replacement char, used by Go's excelize for invalid XML
 *    control chars like NUL/ESC) with empty string — ExcelJS strips those
 *    chars entirely when writing, so both sides map to the stripped form.
 *  - Normalize CRLF → LF: Go's excelize preserves \r\n, ExcelJS strips \r. */
function normCell(s: string): string {
  return s.replace(/�/g, "").replace(/\r\n/g, "\n").replace(/\r/g, "");
}

async function loadXlsx(bytes: Uint8Array): Promise<string[][]> {
  const book = new ExcelJS.Workbook();
  // Copy bytes into a fresh ArrayBuffer to avoid Buffer pool offset issues.
  const ab = bytes.buffer.slice(
    bytes.byteOffset,
    bytes.byteOffset + bytes.byteLength
  );
  await book.xlsx.load(ab as ArrayBuffer);
  const sheet = book.getWorksheet("Sheet1")!;
  const rows: string[][] = [];
  sheet.eachRow({ includeEmpty: true }, (row) => {
    const cells: string[] = [];
    row.eachCell({ includeEmpty: true }, (cell) => {
      cells.push(cell.value == null ? "" : normCell(String(cell.value)));
    });
    rows.push(cells);
  });
  return rows;
}

describe("serializeXLSX cell parity", () => {
  for (const id of Object.keys(FIXTURES)) {
    it(id, async () => {
      const goldenBytes = new Uint8Array(
        readFileSync(resolve(here, "goldens/xlsx", `${id}.xlsx`))
      );
      const ourBytes = await serializeXLSX(FIXTURES[id]);
      const goldenCells = await loadXlsx(goldenBytes);
      const ourCells = await loadXlsx(ourBytes);
      expect(ourCells).toEqual(goldenCells);
    });
  }
});

describe("serializeXLSX column-count guard", () => {
  // Excel's hard worksheet column limit (XFD = 16,384 per ECMA-376).
  // Beyond this, ExcelJS's col-cache itself throws.
  it("throws ResultTooLarge when columns exceed 16384", async () => {
    const { create } = await import("@bufbuild/protobuf");
    const { QueryResultSchema } = await import(
      "@/types/proto-es/v1/sql_service_pb"
    );
    const tooWide = create(QueryResultSchema, {
      columnNames: Array.from({ length: 16385 }, (_, i) => `c${i}`),
      rows: [],
    });
    await expect(serializeXLSX(tooWide)).rejects.toMatchObject({
      code: "ResultTooLarge",
      message: expect.stringContaining("16384"),
    });
  });
});

describe("serializeXLSX row-count guard", () => {
  // Excel's hard worksheet limit is 1,048,576 rows. The serializer writes a
  // header in row 1 and data in rows 2..N+1, so max data rows = 1,048,575.
  // The 5M-cell cap doesn't catch tall-narrow shapes (e.g. 1.1M × 1).
  it("throws ResultTooLarge when data rows exceed 1,048,575", async () => {
    const { create } = await import("@bufbuild/protobuf");
    const { QueryResultSchema } = await import(
      "@/types/proto-es/v1/sql_service_pb"
    );
    // Sparse array — the guard checks only `rows.length` and throws before
    // iteration, so the slots can stay unallocated. This keeps memory bounded
    // (only the array spine, not 1M+ row protos).
    const sparseRows = new Array(1_048_576) as never[];
    const tooTall = create(QueryResultSchema, {
      columnNames: ["a"],
      rows: sparseRows,
    });
    await expect(serializeXLSX(tooTall)).rejects.toMatchObject({
      code: "ResultTooLarge",
      message: expect.stringMatching(/1,?048,?575/),
    });
  });
});
