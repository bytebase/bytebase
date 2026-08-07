import { describe, expect, test } from "vitest";
import { buildDatabaseFilter } from "./utils";

describe("buildDatabaseFilter", () => {
  test("escapes the database name query as a CEL string literal", () => {
    expect(buildDatabaseFilter({ query: 'Payroll "Q3"\\West\nArchive' })).toBe(
      'name.contains("payroll \\"q3\\"\\\\west\\narchive")'
    );
  });
});
