import { describe, expect, test } from "vitest";
import { prepareSnippetInsertion } from "./prepareSnippetInsertion";

describe("prepareSnippetInsertion", () => {
  test("separates a snippet from SQL following the cursor", () => {
    expect(
      prepareSnippetInsertion("SELECT 2;", "", "SELECT 1;", "\n")
    ).toEqual({
      text: "SELECT 2;\n",
      cursorOffset: "SELECT 2;\n".length,
    });
  });

  test("places a snippet on its own lines when inserted mid-line", () => {
    expect(
      prepareSnippetInsertion("SELECT 2;", "SELECT ", "1;", "\n")
    ).toEqual({
      text: "\nSELECT 2;\n",
      cursorOffset: "\nSELECT 2;\n".length,
    });
  });

  test("uses an existing following newline without creating a blank line", () => {
    expect(
      prepareSnippetInsertion("SELECT 2;", "SELECT 1;\n", "\nSELECT 3;", "\n")
    ).toEqual({
      text: "SELECT 2;",
      cursorOffset: "SELECT 2;".length + 1,
    });
  });

  test("leaves the cursor on a new line when inserting at end of file", () => {
    expect(
      prepareSnippetInsertion("SELECT 2;", "SELECT 1;", "", "\n")
    ).toEqual({
      text: "\nSELECT 2;\n",
      cursorOffset: "\nSELECT 2;\n".length,
    });
  });

  test("normalizes snippet boundaries and respects the model line ending", () => {
    expect(
      prepareSnippetInsertion("\nSELECT 2;\r\n", "", "SELECT 1;", "\r\n")
    ).toEqual({
      text: "SELECT 2;\r\n",
      cursorOffset: "SELECT 2;\r\n".length,
    });
  });
});
