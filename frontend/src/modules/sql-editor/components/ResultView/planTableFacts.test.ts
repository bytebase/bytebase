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

describe("tableFactsLookup", () => {
  test("answers with the table's size and its indexes", () => {
    expect(tableFactsLookup(oneSchema())("salary")).toEqual({
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
    const facts = tableFactsLookup(
      metadata([{ name: "public", tables: [table("salary", 100n, [])] }])
    )("salary");

    expect(facts).toEqual({ rowCount: 100, indexes: [] });
  });

  test("leaves out a row count nothing has counted", () => {
    expect(
      tableFactsLookup(
        metadata([{ name: "public", tables: [table("salary", 0n, [])] }])
      )("salary")?.rowCount
    ).toBeUndefined();
  });

  test("says nothing about a relation it does not have", () => {
    expect(tableFactsLookup(oneSchema())("orders")).toBeUndefined();
    expect(tableFactsLookup(oneSchema())("")).toBeUndefined();
  });

  test("says nothing when two schemas both have that name", () => {
    // A plan names the relation without its schema, and the wrong table's
    // indexes would be worse than none.
    const ambiguous = metadata([
      { name: "public", tables: [table("salary", 9488n)] },
      { name: "archive", tables: [table("salary", 12n)] },
    ]);

    expect(tableFactsLookup(ambiguous)("salary")).toBeUndefined();
  });

  test("carries a unique index as unique, not as primary", () => {
    const facts = tableFactsLookup(
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
    )("salary");

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
