import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));

export function loadGolden(
  format: "csv" | "json" | "sql" | "xlsx",
  id: string
): Uint8Array {
  return new Uint8Array(readFileSync(resolve(here, "goldens", format, id)));
}

export function expectGolden(
  actual: Uint8Array,
  format: "csv" | "json" | "sql" | "xlsx",
  id: string
): void {
  const want = loadGolden(format, id);
  if (actual.byteLength !== want.byteLength || !uintArraysEqual(actual, want)) {
    const expected = new TextDecoder("utf-8", { fatal: false }).decode(want);
    const got = new TextDecoder("utf-8", { fatal: false }).decode(actual);
    throw new Error(
      `golden mismatch for goldens/${format}/${id}\n` +
        `  expected ${want.byteLength} bytes:\n${truncate(expected)}\n` +
        `  got      ${actual.byteLength} bytes:\n${truncate(got)}\n` +
        `If backend changed: regenerate with \`go test ./backend/component/export -run TestDownloadGoldens -update\` and commit.`
    );
  }
}

function uintArraysEqual(a: Uint8Array, b: Uint8Array): boolean {
  for (let i = 0; i < a.byteLength; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

function truncate(s: string): string {
  return s.length > 800 ? s.slice(0, 800) + "…" : s;
}
