import { describe, expect, it } from "vitest";

import { normalizeTitle } from "./string";

describe("normalizeTitle", () => {
  it("trims plain ASCII whitespace", () => {
    expect(normalizeTitle("  hello  ")).toBe("hello");
  });

  it("trims U+0085 (NEXT LINE)", () => {
    expect(normalizeTitle("\u0085")).toBe("");
    expect(normalizeTitle("\u0085hello\u0085")).toBe("hello");
  });

  it("trims U+00A0 (NBSP)", () => {
    expect(normalizeTitle("\u00A0\u00A0")).toBe("");
  });

  it("trims U+3000 (ideographic space)", () => {
    expect(normalizeTitle("\u3000hello\u3000")).toBe("hello");
  });

  it("preserves internal whitespace", () => {
    expect(normalizeTitle("  hello world  ")).toBe("hello world");
  });

  it("returns empty string for empty input", () => {
    expect(normalizeTitle("")).toBe("");
  });
});
