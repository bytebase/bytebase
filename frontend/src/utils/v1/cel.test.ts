import { describe, expect, test } from "vitest";
import { escapeCELStringLiteral } from "./cel";

describe("escapeCELStringLiteral", () => {
  test("escapes double quotes — regression for SQL identifiers", () => {
    // `statement.contains("...")` must stay valid CEL when the statement
    // contains double quotes (e.g. a quoted identifier).
    expect(escapeCELStringLiteral('SELECT * FROM "public".db LIMIT 1;')).toBe(
      'SELECT * FROM \\"public\\".db LIMIT 1;'
    );
  });

  test("escapes backslashes before quotes", () => {
    expect(escapeCELStringLiteral('a\\b"c')).toBe('a\\\\b\\"c');
  });

  test("escapes newlines, carriage returns, and tabs", () => {
    expect(escapeCELStringLiteral("a\nb\rc\td")).toBe("a\\nb\\rc\\td");
  });

  test("leaves plain text unchanged", () => {
    expect(escapeCELStringLiteral("select 1")).toBe("select 1");
  });
});
