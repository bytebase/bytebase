import { create } from "@bufbuild/protobuf";
import { StructSchema, type Value, ValueSchema } from "@bufbuild/protobuf/wkt";
import ExcelJS from "exceljs";
import { describe, expect, it } from "vitest";
import {
  QueryResultSchema,
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { serializeXLSX } from "../formats/xlsx";

const rowOf = (...values: RowValue[]) => create(QueryRowSchema, { values });
const intRow = (n: bigint): RowValue =>
  create(RowValueSchema, { kind: { case: "int64Value", value: n } });
const strRow = (s: string): RowValue =>
  create(RowValueSchema, { kind: { case: "stringValue", value: s } });
const valueRow = (v: Value): RowValue =>
  create(RowValueSchema, { kind: { case: "valueValue", value: v } });

async function loadCells(bytes: Uint8Array): Promise<string[][]> {
  const book = new ExcelJS.Workbook();
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
      cells.push(cell.value == null ? "" : String(cell.value));
    });
    rows.push(cells);
  });
  return rows;
}

describe("serializeXLSX", () => {
  it("writes a header row plus N data rows with string-typed cells", async () => {
    const r = create(QueryResultSchema, {
      columnNames: ["id", "name"],
      rows: [
        rowOf(intRow(1n), strRow("Alice")),
        rowOf(intRow(2n), strRow("Bob")),
      ],
    });
    expect(await loadCells(await serializeXLSX(r))).toEqual([
      ["id", "name"],
      ["1", "Alice"],
      ["2", "Bob"],
    ]);
  });

  it("emits structpb cells as JSON-as-string in the cell text (Tier 2)", async () => {
    const struct = create(ValueSchema, {
      kind: {
        case: "structValue",
        value: create(StructSchema, {
          fields: {
            a: create(ValueSchema, { kind: { case: "numberValue", value: 1 } }),
            b: create(ValueSchema, {
              kind: { case: "stringValue", value: "x" },
            }),
          },
        }),
      },
    });
    const r = create(QueryResultSchema, {
      columnNames: ["v"],
      rows: [rowOf(valueRow(struct))],
    });
    expect(await loadCells(await serializeXLSX(r))).toEqual([
      ["v"],
      [`{"a":1,"b":"x"}`],
    ]);
  });

  it("throws ResultTooLarge when columns exceed 16384", async () => {
    const tooWide = create(QueryResultSchema, {
      columnNames: Array.from({ length: 16385 }, (_, i) => `c${i}`),
      rows: [],
    });
    await expect(serializeXLSX(tooWide)).rejects.toMatchObject({
      code: "ResultTooLarge",
      message: expect.stringContaining("16384"),
    });
  });

  it("throws ResultTooLarge when data rows exceed 1,048,575", async () => {
    // Sparse array — guard rejects on `rows.length` before iteration.
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
