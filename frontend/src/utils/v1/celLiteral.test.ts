import { describe, expect, test } from "vitest";
import { celMapField, celString, celStringList } from "./celLiteral";

describe("celString", () => {
  test("escapes double quotes — regression for SQL identifiers", () => {
    // `statement.contains(...)` must stay valid CEL when the statement
    // contains double quotes (e.g. a quoted identifier).
    expect(celString('SELECT * FROM "public".db LIMIT 1;')).toBe(
      '"SELECT * FROM \\"public\\".db LIMIT 1;"'
    );
  });

  test("escapes backslashes before quotes", () => {
    expect(celString('a\\b"c')).toBe('"a\\\\b\\"c"');
  });

  test("escapes newlines, carriage returns, and tabs", () => {
    expect(celString("a\nb\rc\td")).toBe('"a\\nb\\rc\\td"');
  });

  test("quotes plain text", () => {
    expect(celString("select 1")).toBe('"select 1"');
  });

  test("closes the literal it opens", () => {
    // The escape is only worth anything if the quotes come from the same
    // place — a caller writing its own quotes is the bug this prevents.
    expect(celString('a"b')).toMatch(/^".*"$/);
  });
});

describe("celStringList", () => {
  test("renders a CEL list literal", () => {
    expect(celStringList(["a", "b"])).toBe('["a", "b"]');
  });

  test("escapes each element", () => {
    expect(celStringList(['a"b'])).toBe('["a\\"b"]');
  });

  test("renders an empty list", () => {
    expect(celStringList([])).toBe("[]");
  });
});

describe("celMapField", () => {
  test("uses index syntax so non-identifier keys survive", () => {
    // Label keys allow dashes; `labels.cost-center` parses as subtraction and
    // the backend rejects it with `unsupport variable ""`.
    expect(celMapField("labels", "cost-center")).toBe('labels["cost-center"]');
  });

  test("escapes the key", () => {
    expect(celMapField("labels", 'a"b')).toBe('labels["a\\"b"]');
  });
});
