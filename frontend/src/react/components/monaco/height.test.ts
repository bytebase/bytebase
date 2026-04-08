import { describe, expect, test } from "vitest";
import { clampEditorHeight } from "./height";

describe("clampEditorHeight", () => {
  test("clamps short content to the minimum height", () => {
    expect(
      clampEditorHeight({
        lineCount: 1,
        lineHeight: 24,
        min: 120,
        max: 600,
      })
    ).toBe(120);
  });

  test("returns the computed height when it is within bounds", () => {
    expect(
      clampEditorHeight({
        lineCount: 6,
        lineHeight: 24,
        min: 120,
        max: 600,
      })
    ).toBe(144);
  });

  test("clamps tall content to the maximum height", () => {
    expect(
      clampEditorHeight({
        lineCount: 40,
        lineHeight: 24,
        min: 120,
        max: 600,
      })
    ).toBe(600);
  });
});
