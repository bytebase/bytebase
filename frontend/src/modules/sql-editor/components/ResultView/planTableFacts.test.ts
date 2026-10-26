import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  DatabaseMetadataSchema,
  IndexMetadataSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { tableFactsLookup } from "./planTableFacts";

const index = (name: string, expressions: string[], flags = {}) =>
  create(IndexMetadataSchema, { name, expressions, ...flags });

const table = (
  name: string,
  rowCount: bigint,
  indexes = [index("salary_pkey", ["emp_no", "from_date"], { primary: true })]
) => create(TableMetadataSchema, { name, rowCount, indexes });

const metadata = (
  schemas: { name: string; tables: ReturnType<typeof table>[] }[]
) =>
  create(DatabaseMetadataSchema, {
    schemas: schemas.map((schema) =>
      create(SchemaMetadataSchema, { name: schema.name, tables: schema.tables })
    ),
  });

const oneSchema = () =>
  metadata([{ name: "public", tables: [table("salary", 9488n)] }]);

/** The lookup as the result pane builds it, for the connected database. */
const lookup = (data = oneSchema(), connected = "hr_test") =>
  tableFactsLookup(data, connected);

describe("tableFactsLookup", () => {
  test("answers with the table's size and its indexes", () => {
    expect(lookup()({ table: "salary" })).toEqual({
      rowCount: 9488,
      indexes: [
        {
          name: "salary_pkey",
          columns: "emp_no, from_date",
          primary: true,
          unique: false,
        },
      ],
    });
  });

  test("answers for a table with no indexes, which is the point", () => {
    const facts = lookup(
      metadata([{ name: "public", tables: [table("salary", 100n, [])] }])
    )({ table: "salary" });

    expect(facts).toEqual({ rowCount: 100, indexes: [] });
  });

  test("leaves out a row count nothing has counted", () => {
    expect(
      lookup(metadata([{ name: "public", tables: [table("salary", 0n, [])] }]))(
        {
          table: "salary",
        }
      )?.rowCount
    ).toBeUndefined();
  });

  test("says nothing about a relation it does not have", () => {
    expect(lookup()({ table: "orders" })).toBeUndefined();
    expect(lookup()({ table: "" })).toBeUndefined();
  });

  test("says nothing when two schemas both have that name", () => {
    // A plan that names no schema and a name both schemas use: the wrong
    // table's indexes would be worse than none.
    const ambiguous = metadata([
      { name: "public", tables: [table("salary", 9488n)] },
      { name: "archive", tables: [table("salary", 12n)] },
    ]);

    expect(lookup(ambiguous)({ table: "salary" })).toBeUndefined();
  });

  test("picks the schema the plan named, where it named one", () => {
    const ambiguous = metadata([
      { name: "public", tables: [table("salary", 9488n)] },
      { name: "archive", tables: [table("salary", 12n)] },
    ]);

    expect(
      lookup(ambiguous)({ table: "salary", schema: "archive" })?.rowCount
    ).toBe(12);
    expect(
      lookup(ambiguous)({ table: "salary", schema: "DBO" })
    ).toBeUndefined();
  });

  test("says nothing about a table in another database", () => {
    // SQL Server writes the database on every object it reads, so a plan for
    // `archive.dbo.salary` must not be answered from hr_test's catalog, where
    // an unrelated `salary` happens to live.
    expect(
      lookup()({ table: "salary", database: "archive", schema: "public" })
    ).toBeUndefined();
  });

  test("still answers for the connected database, whatever its case", () => {
    expect(lookup()({ table: "salary", database: "HR_TEST" })?.rowCount).toBe(
      9488
    );
  });

  test("carries a unique index as unique, not as primary", () => {
    const facts = lookup(
      metadata([
        {
          name: "public",
          tables: [
            table("salary", 5n, [
              index("salary_amount_key", ["amount"], { unique: true }),
            ]),
          ],
        },
      ])
    )({ table: "salary" });

    expect(facts?.indexes).toEqual([
      {
        name: "salary_amount_key",
        columns: "amount",
        unique: true,
        primary: false,
      },
    ]);
  });
});
