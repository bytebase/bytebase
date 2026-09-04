import { describe, expect, test } from "vitest";
import {
  buildAccessGrantFilter,
  isAccessGrantFilterWithinCELLimit,
  MAX_CEL_FILTER_CODE_POINTS,
} from "./accessGrant";

describe("buildAccessGrantFilter", () => {
  test("empty filter returns empty string", () => {
    expect(buildAccessGrantFilter(undefined)).toBe("");
    expect(buildAccessGrantFilter({})).toBe("");
  });

  test("unmask=true emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ unmask: true })).toBe("unmask == true");
  });

  test("unmask=false emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ unmask: false })).toBe("unmask == false");
  });

  test("export=true emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ export: true })).toBe("export == true");
  });

  test("export=false emits literal CEL boolean", () => {
    expect(buildAccessGrantFilter({ export: false })).toBe("export == false");
  });

  test("unmask and export combine with &&", () => {
    expect(buildAccessGrantFilter({ unmask: true, export: true })).toBe(
      "unmask == true && export == true"
    );
  });

  test("statuses emit display-status filter", () => {
    expect(
      buildAccessGrantFilter({
        status: ["ACTIVE", "PENDING"],
      })
    ).toBe('(status == "ACTIVE" || status == "PENDING")');
  });

  test("unmask combines with other filters in order", () => {
    expect(
      buildAccessGrantFilter({
        creator: "users/dev@example.com",
        unmask: true,
        export: false,
      })
    ).toBe(
      'creator == "users/dev@example.com" && unmask == true && export == false'
    );
  });

  // Sanity check: filter values use the literal token `true`/`false` so the
  // server-side filter parser treats them as CEL booleans, not strings. This
  // matches the backend's `store.GetListAccessGrantFilter` switch which
  // expects `value.(bool)` for these two fields. Quoting them would silently
  // change the SQL emitted (or trigger a "must be a boolean" error).
  test("never quotes the boolean literal", () => {
    expect(buildAccessGrantFilter({ unmask: true })).not.toContain('"true"');
    expect(buildAccessGrantFilter({ export: false })).not.toContain('"false"');
  });

  test("absent unmask/export fields produce no emission", () => {
    // Only target, no unmask/export.
    expect(buildAccessGrantFilter({ target: "instances/i/databases/d" })).toBe(
      'target == "instances/i/databases/d"'
    );
  });

  test("statementExact emits exact CEL equality", () => {
    expect(buildAccessGrantFilter({ statementExact: "SELECT * FROM t" })).toBe(
      `query == "SELECT * FROM t"`
    );
  });

  test("statementExact trims boundary whitespace", () => {
    expect(buildAccessGrantFilter({ statementExact: "\n  SELECT 1\n" })).toBe(
      `query == "SELECT 1"`
    );
  });

  test("statementExact escapes embedded quotes and newlines safely", () => {
    expect(
      buildAccessGrantFilter({
        statementExact: `SELECT 'foo "bar"' FROM t\nWHERE x = 1`,
      })
    ).toBe(`query == "SELECT 'foo \\"bar\\"' FROM t\\nWHERE x = 1"`);
  });

  test("statement and statementExact emit different predicates", () => {
    expect(buildAccessGrantFilter({ statement: "SELECT 1" })).toBe(
      `query.contains("SELECT 1")`
    );
    expect(buildAccessGrantFilter({ statementExact: "SELECT 1" })).toBe(
      `query == "SELECT 1"`
    );
  });

  test("empty-string statementExact still emits exact predicate", () => {
    expect(buildAccessGrantFilter({ statementExact: "" })).toBe(`query == ""`);
  });

  test("does not submit a filter exceeding the CEL code point limit", () => {
    expect(
      isAccessGrantFilterWithinCELLimit({
        statementExact: "x".repeat(MAX_CEL_FILTER_CODE_POINTS),
      })
    ).toBe(false);
  });
});
