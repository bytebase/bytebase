import { describe, expect, test } from "vitest";
import { computeAppendedSelection } from "./appendSelection";

describe("computeAppendedSelection", () => {
  test("empty tab — selects the appended block from line 1", () => {
    // Regression for PR #20554: the first line must be included.
    const appended = 'SELECT\n  *\nFROM "public".db\nLIMIT\n1;';
    expect(computeAppendedSelection("", appended)).toEqual({
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: 5,
      endColumn: 3, // "1;".length + 1
    });
  });

  test("single-line append into empty tab", () => {
    expect(computeAppendedSelection("", "SELECT 1;")).toEqual({
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: 1,
      endColumn: 10, // "SELECT 1;".length + 1
    });
  });

  test("append after an existing single-line statement", () => {
    // newStatement = "SELECT 1;\n\nSELECT 2;" → appended starts at line 3.
    expect(computeAppendedSelection("SELECT 1;", "SELECT 2;")).toEqual({
      startLineNumber: 3,
      startColumn: 1,
      endLineNumber: 3,
      endColumn: 10,
    });
  });

  test("append after a multi-line statement", () => {
    // old = 2 lines → blank separator on line 3 → appended starts line 4.
    const appended = "SELECT\n2;";
    expect(computeAppendedSelection("SELECT\n1;", appended)).toEqual({
      startLineNumber: 4,
      startColumn: 1,
      endLineNumber: 5,
      endColumn: 3,
    });
  });
});
