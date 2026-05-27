// @vitest-environment node

import { create } from "@bufbuild/protobuf";
import * as zipjs from "@zip.js/zip.js";
import { describe, expect, it } from "vitest";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import {
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { buildDownloadBlob } from "../index";
import type { DownloadGroup } from "../types";

const tinyResult = create(QueryResultSchema, {
  columnNames: ["id"],
  columnTypeNames: ["INT"],
  rows: [],
});

const oneStmtGroup = (
  overrides: Partial<DownloadGroup> = {}
): DownloadGroup => ({
  instanceId: "prod-instance",
  databaseName: "demo-db",
  engine: Engine.MYSQL,
  statements: [{ result: tinyResult, statement: "SELECT 1" }],
  ...overrides,
});

async function readZipEntries(blob: Blob, password?: string) {
  const reader = new zipjs.ZipReader(new zipjs.BlobReader(blob), { password });
  const entries = await reader.getEntries();
  await reader.close();
  return entries;
}

async function readEntryText(entry: zipjs.Entry): Promise<string> {
  if (!("getData" in entry) || !entry.getData) {
    throw new Error("expected file entry, got directory");
  }
  return entry.getData(new zipjs.TextWriter());
}

describe("buildDownloadBlob", () => {
  it("rejects empty groups", async () => {
    await expect(
      buildDownloadBlob({
        groups: [],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      })
    ).rejects.toMatchObject({ code: "SerializationFailed" });
  });

  it("rejects groups with zero statements", async () => {
    await expect(
      buildDownloadBlob({
        groups: [
          {
            instanceId: "i",
            databaseName: "d",
            engine: Engine.MYSQL,
            statements: [],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      })
    ).rejects.toMatchObject({ code: "SerializationFailed" });
  });

  it("rejects unsupported format", async () => {
    await expect(
      buildDownloadBlob({
        groups: [oneStmtGroup()],
        format: 99 as ExportFormat,
        baseFilename: "demo",
      })
    ).rejects.toMatchObject({ code: "UnsupportedFormat" });
  });

  describe("single statement (1 group, 1 statement)", () => {
    it("emits ZIP with <instance>/<db>/statement-1.sql and .result.csv", async () => {
      const out = await buildDownloadBlob({
        groups: [oneStmtGroup()],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      });
      expect(out.filename).toBe("demo.zip");
      expect(out.mimeType).toBe("application/zip");
      const entries = await readZipEntries(out.blob);
      expect(entries.map((e) => e.filename).sort()).toEqual([
        "prod-instance/demo-db/statement-1.result.csv",
        "prod-instance/demo-db/statement-1.sql",
      ]);
    });

    it("statement-1.sql contains the SQL text", async () => {
      const out = await buildDownloadBlob({
        groups: [
          oneStmtGroup({
            statements: [
              {
                result: tinyResult,
                statement: "SELECT * FROM users WHERE id = 1",
              },
            ],
          }),
        ],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      });
      const entries = await readZipEntries(out.blob);
      const sqlEntry = entries.find((e) => e.filename.endsWith(".sql"))!;
      const text = await readEntryText(sqlEntry);
      expect(text).toBe("SELECT * FROM users WHERE id = 1");
    });

    it.each([
      [ExportFormat.JSON, "result.json"],
      [ExportFormat.SQL, "result.sql"],
      [ExportFormat.XLSX, "result.xlsx"],
    ])("format %p produces statement-1.%s alongside statement-1.sql", async (format, ext) => {
      const out = await buildDownloadBlob({
        groups: [oneStmtGroup()],
        format,
        baseFilename: "demo",
      });
      const entries = await readZipEntries(out.blob);
      const filenames = entries.map((e) => e.filename).sort();
      expect(filenames).toContain(`prod-instance/demo-db/statement-1.sql`);
      expect(filenames).toContain(`prod-instance/demo-db/statement-1.${ext}`);
    });

    it("encrypts every entry under one password", async () => {
      const out = await buildDownloadBlob({
        groups: [oneStmtGroup()],
        format: ExportFormat.CSV,
        baseFilename: "demo",
        password: "p@ss",
      });
      const entries = await readZipEntries(out.blob, "p@ss");
      expect(entries.every((e) => e.encrypted)).toBe(true);
    });
  });

  describe("multi-statement (1 group, N statements)", () => {
    it("numbers statements 1..N inside the same instance/db dir", async () => {
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "prod",
            databaseName: "db1",
            engine: Engine.POSTGRES,
            statements: [
              { result: tinyResult, statement: "SELECT 1" },
              { result: tinyResult, statement: "SELECT 2" },
              { result: tinyResult, statement: "SELECT 3" },
            ],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "multi",
      });
      const entries = await readZipEntries(out.blob);
      expect(entries.map((e) => e.filename).sort()).toEqual([
        "prod/db1/statement-1.result.csv",
        "prod/db1/statement-1.sql",
        "prod/db1/statement-2.result.csv",
        "prod/db1/statement-2.sql",
        "prod/db1/statement-3.result.csv",
        "prod/db1/statement-3.sql",
      ]);
    });

    it("each statement-N.sql contains its own SQL text", async () => {
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "i",
            databaseName: "d",
            engine: Engine.MYSQL,
            statements: [
              { result: tinyResult, statement: "SELECT alpha" },
              { result: tinyResult, statement: "SELECT beta" },
            ],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "x",
      });
      const entries = await readZipEntries(out.blob);
      const sqls = entries
        .filter((e) => e.filename.endsWith(".sql"))
        .sort((a, b) => a.filename.localeCompare(b.filename));
      const t1 = await readEntryText(sqls[0]);
      const t2 = await readEntryText(sqls[1]);
      expect(t1).toBe("SELECT alpha");
      expect(t2).toBe("SELECT beta");
    });
  });

  describe("multi-group (batch — N instances/dbs)", () => {
    it("each group becomes its own <instance>/<db>/ subtree", async () => {
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "prod",
            databaseName: "db-a",
            engine: Engine.MYSQL,
            statements: [{ result: tinyResult, statement: "SELECT 1" }],
          },
          {
            instanceId: "stage",
            databaseName: "db-b",
            engine: Engine.POSTGRES,
            statements: [
              { result: tinyResult, statement: "SELECT 1" },
              { result: tinyResult, statement: "SELECT 2" },
            ],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "batch",
      });
      const entries = await readZipEntries(out.blob);
      expect(entries.map((e) => e.filename).sort()).toEqual([
        "prod/db-a/statement-1.result.csv",
        "prod/db-a/statement-1.sql",
        "stage/db-b/statement-1.result.csv",
        "stage/db-b/statement-1.sql",
        "stage/db-b/statement-2.result.csv",
        "stage/db-b/statement-2.sql",
      ]);
    });

    it("uses each group's engine for SQL serialization", async () => {
      const result = create(QueryResultSchema, {
        columnNames: ["id"],
        columnTypeNames: ["INT"],
        rows: [
          create(QueryRowSchema, {
            values: [
              create(RowValueSchema, {
                kind: { case: "int64Value", value: 42n },
              }),
            ],
          }),
        ],
      });
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "prod",
            databaseName: "my",
            engine: Engine.MYSQL,
            statements: [{ result, statement: "SELECT 42" }],
          },
          {
            instanceId: "prod",
            databaseName: "pg",
            engine: Engine.POSTGRES,
            statements: [{ result, statement: "SELECT 42" }],
          },
        ],
        format: ExportFormat.SQL,
        baseFilename: "x",
      });
      const entries = await readZipEntries(out.blob);
      const myEntry = entries.find(
        (e) => e.filename === "prod/my/statement-1.result.sql"
      )!;
      const pgEntry = entries.find(
        (e) => e.filename === "prod/pg/statement-1.result.sql"
      )!;
      const myText = await readEntryText(myEntry);
      const pgText = await readEntryText(pgEntry);
      expect(myText).toContain("`<table_name>`"); // MySQL backtick
      expect(pgText).toContain('"<table_name>"'); // Postgres double-quote
    });
  });

  describe("path sanitization", () => {
    it("sanitizes instanceId and databaseName as path segments", async () => {
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "../../etc",
            databaseName: "passwd\\x00",
            engine: Engine.MYSQL,
            statements: [{ result: tinyResult, statement: "SELECT 1" }],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      });
      const entries = await readZipEntries(out.blob);
      // No leading dot / parent-directory, no slash inside segments.
      for (const e of entries) {
        expect(e.filename).not.toMatch(/^\./);
        expect(e.filename).not.toContain("\\");
        expect(e.filename).not.toContain("\0");
        // exactly two `/` separators (the structure we constructed).
        const slashes = (e.filename.match(/\//g) ?? []).length;
        expect(slashes).toBe(2);
      }
    });

    it("disambiguates cross-group dirPrefixes that sanitize to the same path", async () => {
      // Two distinct (instanceId, databaseName) inputs that both sanitize
      // to "inst_A/db_1". Without dedup, zip.js rejects the second group
      // with `File already exists` and aborts the whole batch.
      const out = await buildDownloadBlob({
        groups: [
          {
            instanceId: "inst/A",
            databaseName: "db/1",
            engine: Engine.MYSQL,
            statements: [{ result: tinyResult, statement: "SELECT 1" }],
          },
          {
            instanceId: "inst\\A",
            databaseName: "db\\1",
            engine: Engine.MYSQL,
            statements: [{ result: tinyResult, statement: "SELECT 1" }],
          },
        ],
        format: ExportFormat.CSV,
        baseFilename: "demo",
      });
      const entries = await readZipEntries(out.blob);
      const filenames = entries.map((e) => e.filename).sort();
      // 2 groups × 2 entries (.sql + .result.csv) = 4 entries, all unique.
      expect(filenames).toHaveLength(4);
      expect(new Set(filenames).size).toBe(4);
      // First group keeps the candidate prefix; second gets disambiguated.
      expect(filenames).toContain("inst_A/db_1/statement-1.sql");
      expect(filenames.some((f) => f.startsWith("inst_A/db_1-"))).toBe(true);
    });
  });

  describe("caps", () => {
    it("rejects when total cells across all groups exceeds cap", async () => {
      const wide = create(QueryResultSchema, {
        columnNames: Array.from({ length: 100 }, (_, i) => `c${i}`),
        rows: Array.from({ length: 30_000 }, () =>
          create(QueryRowSchema, { values: [] })
        ),
      });
      // 2 groups × 30k rows × 100 cols = 6M cells > 5M cap.
      await expect(
        buildDownloadBlob({
          groups: [
            {
              instanceId: "i",
              databaseName: "a",
              engine: Engine.MYSQL,
              statements: [{ result: wide, statement: "SELECT" }],
            },
            {
              instanceId: "i",
              databaseName: "b",
              engine: Engine.MYSQL,
              statements: [{ result: wide, statement: "SELECT" }],
            },
          ],
          format: ExportFormat.CSV,
          baseFilename: "demo",
        })
      ).rejects.toMatchObject({ code: "ResultTooLarge" });
    });

    it("rejects when total estimated bytes across all groups exceeds cap", async () => {
      const huge = "x".repeat(25 * 1024 * 1024);
      const wideStrings = create(QueryResultSchema, {
        columnNames: Array.from({ length: 10 }, (_, i) => `c${i}`),
        rows: Array.from({ length: 10 }, () =>
          create(QueryRowSchema, {
            values: Array.from({ length: 10 }, () =>
              create(RowValueSchema, {
                kind: { case: "stringValue", value: huge },
              })
            ),
          })
        ),
      });
      await expect(
        buildDownloadBlob({
          groups: [
            {
              instanceId: "i",
              databaseName: "a",
              engine: Engine.MYSQL,
              statements: [{ result: wideStrings, statement: "SELECT" }],
            },
          ],
          format: ExportFormat.CSV,
          baseFilename: "demo",
        })
      ).rejects.toMatchObject({ code: "ResultTooLarge" });
    });
  });

  describe("SQL-engine guard", () => {
    it("propagates UnsupportedFormat from per-group engine", async () => {
      await expect(
        buildDownloadBlob({
          groups: [
            {
              instanceId: "i",
              databaseName: "d",
              engine: Engine.ENGINE_UNSPECIFIED,
              statements: [{ result: tinyResult, statement: "SELECT" }],
            },
          ],
          format: ExportFormat.SQL,
          baseFilename: "demo",
        })
      ).rejects.toMatchObject({ code: "UnsupportedFormat" });
    });
  });
});
