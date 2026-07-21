import { describe, expect, test } from "vitest";

// Scan the entire application source tree. The only remaining Vue runtime is
// encapsulated by apps/explain-visualizer and does not expose .vue modules.
const sources = import.meta.glob("../**/*.{ts,tsx}", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const allowedVueImports = new Set<string>();

const vueImportPattern = /from\s+["']([^"']+\.vue)["']/g;

// Retired implementation prefixes must not be recreated.
const bannedReactToVueModulePrefixes = [
  "@/components/MonacoEditor",
  "@/components/Plan",
  "@/components/SQLReview",
  "@/components/ProjectMember",
  "@/components/ColumnDataTable",
  "@/components/SensitiveData",
  "@/components/DatabaseGroup",
];

describe("application source must not import .vue files", () => {
  test("no application TypeScript file imports a .vue file", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      // Don't scan this guard itself (it contains .vue strings as test data
      // in the allowlist above).
      if (file.endsWith("/no-vue-imports.test.ts")) continue;
      // Don't scan the sibling guard (same reason — it has .vue strings as
      // banned-import test data).
      if (file.endsWith("/legacy-boundaries.test.ts")) continue;

      let match: RegExpExecArray | null;
      vueImportPattern.lastIndex = 0;
      while ((match = vueImportPattern.exec(source)) !== null) {
        const importPath = match[1];
        if (!allowedVueImports.has(importPath)) {
          violations.push(`${file}: ${importPath}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });

  test("does not recreate retired implementation module prefixes", () => {
    const importPattern = /from\s+["']([^"']+)["']/g;
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      if (file.endsWith("/no-vue-imports.test.ts")) continue;
      if (file.endsWith("/legacy-boundaries.test.ts")) continue;

      let match: RegExpExecArray | null;
      importPattern.lastIndex = 0;
      while ((match = importPattern.exec(source)) !== null) {
        const importPath = match[1];
        for (const prefix of bannedReactToVueModulePrefixes) {
          if (importPath === prefix || importPath.startsWith(`${prefix}/`)) {
            violations.push(`${file}: ${importPath}`);
          }
        }
      }
    }
    expect(violations).toEqual([]);
  });
});
