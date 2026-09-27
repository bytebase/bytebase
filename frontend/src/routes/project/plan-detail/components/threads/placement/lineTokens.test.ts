import { describe, expect, it } from "vitest";
import { tokenizeLines } from "./lineTokens";

describe("tokenizeLines", () => {
  it("always emits a final token", () => {
    expect(tokenizeLines("")).toEqual([""]);
    expect(tokenizeLines("A")).toEqual(["A"]);
    expect(tokenizeLines("A\n")).toEqual(["A\n", ""]);
  });

  it("treats CRLF, LF, and lone CR as line breaks and keeps terminators", () => {
    expect(tokenizeLines("A\r\nB\r")).toEqual(["A\r\n", "B\r", ""]);
    expect(tokenizeLines("A\rB\nC\r\nD")).toEqual(["A\r", "B\n", "C\r\n", "D"]);
    expect(tokenizeLines("\n\n")).toEqual(["\n", "\n", ""]);
    expect(tokenizeLines("\r\r\n")).toEqual(["\r", "\r\n", ""]);
  });

  it("never merges a CR with a later LF that is not adjacent", () => {
    expect(tokenizeLines("A\r \nB")).toEqual(["A\r", " \n", "B"]);
  });

  it("reproduces the input when concatenated", () => {
    for (const text of [
      "",
      "x",
      "a\nb",
      "a\r\nb\rc\n",
      "\r\n\r\n",
      "é\n漢字\r\n",
    ]) {
      expect(tokenizeLines(text).join("")).toBe(text);
    }
  });
});
