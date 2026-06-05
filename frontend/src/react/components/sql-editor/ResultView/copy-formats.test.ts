import { create } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { RowValue } from "@/types/proto-es/v1/sql_service_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import type { CopyFormatterContext } from "./copy-formats";
import { formatAsCSV, formatAsSQL, formatAsText } from "./copy-formats";
import type { ResultTableColumn, ResultTableRow } from "./types";

// `copy-formats` → `@/utils/v1/sql` pulls in the heavy `@/types` barrel (and
// the React router). Mock it with a minimal real implementation.
vi.mock("@/types", () => ({
  getDateForPbTimestampProtoEs: (ts: { seconds: bigint; nanos: number }) =>
    new Date(Number(ts.seconds) * 1000 + Math.floor((ts.nanos ?? 0) / 1e6)),
}));

const rv = (kind: RowValue["kind"]): RowValue =>
  create(RowValueSchema, { kind });

const columns: ResultTableColumn[] = [
  { id: "name", name: "name", columnType: "TEXT" },
  { id: "age", name: "age", columnType: "INT" },
];

const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [
        rv({ case: "stringValue", value: "alice" }),
        rv({ case: "int32Value", value: 30 }),
      ],
    }),
  },
  {
    key: 1,
    item: create(QueryRowSchema, {
      values: [
        rv({ case: "stringValue", value: "bob" }),
        rv({ case: "int32Value", value: 25 }),
      ],
    }),
  },
];

// Display values for text/CSV. Row 1 col 0 holds a comma to exercise CSV
// quoting (the SQL path reads the raw RowValue, not this table).
const display: Record<string, string> = {
  "0,0": "alice",
  "0,1": "30",
  "1,0": "bob,jr",
  "1,1": "25",
};
const getFormattedValue = (r: number, c: number) => display[`${r},${c}`] ?? "";

const ctx = (over: Partial<CopyFormatterContext>): CopyFormatterContext => ({
  scope: "all",
  selection: { rows: [], columns: [] },
  rows,
  columns,
  engine: Engine.POSTGRES,
  schema: "public",
  getFormattedValue,
  ...over,
});

describe("formatAsText", () => {
  it("copies all rows as TSV with a leading index column", () => {
    expect(formatAsText(ctx({ scope: "all" }))).toBe(
      "index\tname\tage\n0\talice\t30\n1\tbob,jr\t25"
    );
  });

  it("copies a single selected cell as its plain value", () => {
    expect(
      formatAsText(
        ctx({ scope: "selected", selection: { rows: [0], columns: [1] } })
      )
    ).toBe("30");
  });

  it("copies a selected column for every row", () => {
    expect(
      formatAsText(
        ctx({ scope: "selected", selection: { rows: [], columns: [0] } })
      )
    ).toBe("name\nalice\nbob,jr");
  });
});

describe("formatAsCSV", () => {
  it("copies a header row plus RFC-4180-quoted rows", () => {
    expect(formatAsCSV(ctx({ scope: "all" }))).toBe(
      `name,age\nalice,30\n"bob,jr",25`
    );
  });

  it("copies only the selected rows", () => {
    expect(
      formatAsCSV(
        ctx({ scope: "selected", selection: { rows: [1], columns: [] } })
      )
    ).toBe(`name,age\n"bob,jr",25`);
  });
});

describe("formatAsSQL", () => {
  it("renders a batched INSERT from the raw row values", () => {
    expect(formatAsSQL(ctx({ scope: "all" }))).toBe(
      `INSERT INTO "public"."<table_name>" ("name", "age") VALUES\n` +
        `  ('alice', 30),\n` +
        `  ('bob', 25);`
    );
  });

  it("scopes to the selected rows when scope is selected", () => {
    expect(
      formatAsSQL(
        ctx({ scope: "selected", selection: { rows: [1], columns: [] } })
      )
    ).toBe(
      `INSERT INTO "public"."<table_name>" ("name", "age") VALUES\n` +
        `  ('bob', 25);`
    );
  });
});
