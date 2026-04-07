import { describe, expect, test } from "vitest";
import { getReadonlyMonacoHeight } from "./height";

describe("getReadonlyMonacoHeight", () => {
  test("clamps empty content to the minimum height", () => {
    expect(
      getReadonlyMonacoHeight("", {
        minHeight: 120,
        maxHeight: 600,
        lineHeight: 24,
        padding: 16,
      })
    ).toBe(120);
  });

  test("uses content lines before clamping to the maximum height", () => {
    expect(
      getReadonlyMonacoHeight("select 1;\nselect 2;\nselect 3;", {
        minHeight: 40,
        maxHeight: 200,
        lineHeight: 24,
        padding: 16,
      })
    ).toBe(88);
  });

  test("clamps tall content to the maximum height", () => {
    expect(
      getReadonlyMonacoHeight(
        Array.from({ length: 20 }, () => "select 1;").join("\n"),
        {
          minHeight: 40,
          maxHeight: 180,
          lineHeight: 24,
          padding: 16,
        }
      )
    ).toBe(180);
  });
});
