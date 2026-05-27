// @vitest-environment node

import { describe, expect, it } from "vitest";
import {
  FALLBACK_BASENAME,
  MAX_BASENAME_LENGTH,
  sanitizeBasename,
  uniqueFilename,
} from "../filename";

describe("sanitizeBasename (B5 — ZIP filename path traversal)", () => {
  it("returns plain ASCII names unchanged", () => {
    expect(sanitizeBasename("prod-db-2026-05-09")).toBe("prod-db-2026-05-09");
  });

  it("replaces forward slashes with underscores", () => {
    expect(sanitizeBasename("a/b")).toBe("a_b");
  });

  it("replaces backslashes with underscores", () => {
    expect(sanitizeBasename("a\\b")).toBe("a_b");
  });

  it("strips NUL bytes", () => {
    expect(sanitizeBasename("a\x00b")).toBe("ab");
  });

  it("collapses parent-directory hints embedded in the name", () => {
    // `..` and `.` segments inside a basename can still be re-interpreted
    // by some ZIP readers after we strip the slashes.
    expect(sanitizeBasename("..")).toBe(FALLBACK_BASENAME);
    // `../foo` -> slash becomes `_` -> `.._foo` -> dot-run collapses to `._foo`
    // -> leading dots/whitespace trimmed -> `_foo`. Underscore is safe.
    expect(sanitizeBasename("../foo")).toBe("_foo");
    // `foo/..` -> `foo_..` -> `foo_.` -> trimmed trailing dot -> `foo_`.
    expect(sanitizeBasename("foo/..")).toBe("foo_");
  });

  it("trims leading and trailing dots and whitespace", () => {
    expect(sanitizeBasename("  prod  ")).toBe("prod");
    expect(sanitizeBasename("...prod")).toBe("prod");
    expect(sanitizeBasename("prod...")).toBe("prod");
  });

  it("truncates names longer than the cap", () => {
    const long = "a".repeat(MAX_BASENAME_LENGTH + 50);
    const out = sanitizeBasename(long);
    expect(out.length).toBe(MAX_BASENAME_LENGTH);
  });

  it("falls back to 'download' when input is empty or only stripped chars", () => {
    expect(sanitizeBasename("")).toBe(FALLBACK_BASENAME);
    expect(sanitizeBasename("\x00\x00")).toBe(FALLBACK_BASENAME);
    expect(sanitizeBasename("///")).toBe("___"); // slashes become underscores, not empty
    expect(sanitizeBasename("....")).toBe(FALLBACK_BASENAME);
    expect(sanitizeBasename("   ")).toBe(FALLBACK_BASENAME);
  });

  it("preserves non-ASCII printable Unicode (databases may carry it)", () => {
    expect(sanitizeBasename("生产-db")).toBe("生产-db");
  });

  it("strips bidi-override / format chars that can spoof archive viewers", () => {
    // U+202E RIGHT-TO-LEFT OVERRIDE could rearrange display: `evil_<RLO>gpj.exe` -> `evil_exe.jpg`.
    expect(sanitizeBasename("evil_‮gpj.exe")).toBe("evil_gpj.exe");
    // BOM
    expect(sanitizeBasename("﻿prod")).toBe("prod");
    // LRM/RLM
    expect(sanitizeBasename("a‎b")).toBe("ab");
    // Bidi isolates
    expect(sanitizeBasename("⁦prod⁩")).toBe("prod");
    // U+061C ARABIC LETTER MARK (bidi control older than U+202x)
    expect(sanitizeBasename("a؜b")).toBe("ab");
  });

  it("does not split a surrogate pair when truncating to the length cap", () => {
    // 199 ASCII chars + 1 emoji (surrogate pair = 2 UTF-16 units) → s.length=201.
    // Slice(0, 200) cuts inside the pair; the high surrogate would be invalid.
    const input = "a".repeat(199) + "😀";
    const out = sanitizeBasename(input);
    expect(out.length).toBe(199); // emoji dropped entirely
    // No lone surrogate left at the tail.
    const last = out.charCodeAt(out.length - 1);
    expect(last >= 0xd800 && last <= 0xdfff).toBe(false);
  });
});

describe("uniqueFilename (B10 — batch ZIP duplicate filenames)", () => {
  it("returns the name unchanged when not yet taken", () => {
    const taken = new Set<string>();
    expect(uniqueFilename("prod.csv", taken)).toBe("prod.csv");
    expect(taken.has("prod.csv")).toBe(true);
  });

  it("appends -1 on first collision, -2 on second", () => {
    const taken = new Set<string>(["prod.csv"]);
    expect(uniqueFilename("prod.csv", taken)).toBe("prod-1.csv");
    expect(uniqueFilename("prod.csv", taken)).toBe("prod-2.csv");
  });

  it("skips past pre-existing suffixed siblings rather than colliding with them", () => {
    // If a customer's batch already had "db" and "db-1" as distinct
    // databases, a third group named "db" must become "db-2" rather than
    // double-collide with the existing literal "db-1" entry.
    const taken = new Set<string>(["db", "db-1"]);
    expect(uniqueFilename("db", taken)).toBe("db-2");
  });

  it("appends suffix at end if no extension", () => {
    const taken = new Set<string>(["batch"]);
    expect(uniqueFilename("batch", taken)).toBe("batch-1");
  });

  it("respects the LAST dot as extension separator", () => {
    const taken = new Set<string>(["a.b.csv"]);
    expect(uniqueFilename("a.b.csv", taken)).toBe("a.b-1.csv");
  });

  it("treats case-only-different names as colliding (Windows extraction)", () => {
    const taken = new Set<string>();
    expect(uniqueFilename("Prod.csv", taken)).toBe("Prod.csv");
    // Same name in different case must dedup.
    expect(uniqueFilename("prod.csv", taken)).toBe("prod-1.csv");
    expect(uniqueFilename("PROD.csv", taken)).toBe("PROD-2.csv");
  });

  it("treats NFC and NFD forms of the same string as colliding", () => {
    const taken = new Set<string>();
    const nfc = "café.csv"; // single code point é (U+00E9)
    const nfd = "café.csv"; // e + combining acute
    expect(uniqueFilename(nfc, taken)).toBe(nfc);
    expect(uniqueFilename(nfd, taken)).toBe("café-1.csv");
  });
  it("appends -N at the end (not before a dot) for path-like inputs", () => {
    // dirPrefix dedup in buildDownloadBlob passes "inst/db" paths. When the
    // database segment legitimately contains a dot (e.g. "prod.app_v2"),
    // treating the dot as an extension boundary would yield "prod-1.app_v2"
    // and mis-label the path. The "/" in the input signals path-mode.
    const taken = new Set<string>(["inst/prod.app_v2"]);
    expect(uniqueFilename("inst/prod.app_v2", taken)).toBe(
      "inst/prod.app_v2-1"
    );
  });
});
