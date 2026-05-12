import { describe, expect, test } from "vitest";

// Every file under frontend/src/react/ that is *.ts or *.tsx (the React layer).
// .vue files are skipped: by definition .vue is Vue-side and is allowed to import
// other .vue files.
const sources = import.meta.glob("./**/*.{ts,tsx}", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// Mount-bridge Vue files that React code is permitted to import until Phase B
// retires the Vue app shell. Adding new entries here requires explicit review.
const allowedVueImports = new Set([
  "@/components/SessionExpiredSurfaceMount.vue",
  "@/components/AgentWindowMount.vue",
]);

const vueImportPattern = /from\s+["']([^"']+\.vue)["']/g;

describe("React layer must not import .vue files", () => {
  test("no .tsx or .ts file under frontend/src/react/ imports a .vue file", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      // Don't scan this guard itself (it contains .vue strings as test data
      // in the allowlist above).
      if (file.endsWith("/no-react-to-vue-imports.test.ts")) continue;
      // Don't scan the sibling guard (same reason — it has .vue strings as
      // banned-import test data).
      if (file.endsWith("/no-legacy-vue-deps.test.ts")) continue;

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
});
