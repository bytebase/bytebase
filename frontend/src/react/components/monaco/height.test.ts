import { describe, expect, test } from "vitest";
import { clampEditorHeight } from "./height";

describe("clampEditorHeight", () => {
  test("clamps short content height to the minimum height", () => {
    expect(
      clampEditorHeight({
        contentHeight: 24,
        min: 120,
        max: 600,
      })
    ).toBe(120);
  });

  test("returns the measured content height when it is within bounds", () => {
    expect(
      clampEditorHeight({
        contentHeight: 144,
        min: 120,
        max: 600,
      })
    ).toBe(144);
  });

  test("clamps tall content height to the maximum height", () => {
    expect(
      clampEditorHeight({
        contentHeight: 900,
        min: 120,
        max: 600,
      })
    ).toBe(600);
  });
});
