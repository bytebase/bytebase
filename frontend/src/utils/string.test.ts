// @vitest-environment node
import { describe, expect, it } from "vitest";

import { formatList, normalizeTitle } from "./string";

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

  it("does NOT trim U+FEFF (BOM) — Go unicode.IsSpace leaves it intact", () => {
    // Go's `unicode.IsSpace` does not classify U+FEFF as whitespace, so the
    // frontend must not trim it either. If someone "helpfully" re-adds
    // U+FEFF to the regex, this test fails — preventing a frontend/backend
    // asymmetry in the opposite direction.
    expect(normalizeTitle("\uFEFF")).toBe("\uFEFF");
  });
});

describe("formatList", () => {
  it.each(["en-US", "zh-CN", "ja-JP", "es-ES", "vi-VN"])(
    "keeps the names apart in %s",
    (language) => {
      const joined = formatList(["Staging", "Prod"], language);
      expect(joined).toContain("Staging");
      expect(joined).toContain("Prod");
      expect(joined).not.toBe("StagingProd");
    }
  );

  it("returns a single name unchanged", () => {
    expect(formatList(["Prod"], "en-US")).toBe("Prod");
  });

  it("falls back to commas where the browser has no Intl.ListFormat", () => {
    const descriptor = Object.getOwnPropertyDescriptor(Intl, "ListFormat");
    Reflect.deleteProperty(Intl, "ListFormat");
    try {
      expect(formatList(["Staging", "Prod"], "en-US")).toBe("Staging, Prod");
    } finally {
      if (descriptor) {
        Object.defineProperty(Intl, "ListFormat", descriptor);
      }
    }
  });
});
