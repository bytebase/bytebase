import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, test } from "vitest";

const repoRoot = process.cwd();
const srcRoot = join(repoRoot, "src");
// Tests are excluded: they spell out expected filter strings literally, which
// is exactly what an assertion should do.
const sourceFilePattern = /(?<!\.test)\.(?:ts|tsx)$/;
const generatedRoot = join(srcRoot, "types", "proto-es");

// A value interpolated straight into a double-quoted CEL string literal breaks
// the filter the moment it holds a `"` — a SQL identifier, anything typed into
// a search box — and the backend rejects the whole expression with
// InvalidArgument. `celString` / `celStringList` / `celMapField` in
// `src/utils/v1/celLiteral.ts` own the quotes so the escape cannot be forgotten.
const rawInterpolationRules = [
  {
    what: "a CEL string argument — use `name.contains(${celString(x)})`",
    pattern: /\.(?:contains|matches|startsWith|endsWith)\("\$\{/,
  },
  {
    what: "a CEL comparison operand — use `state == ${celString(x)}`",
    pattern: /[a-z][a-zA-Z0-9_.]*\s*(?:==|!=|>=|<=)\s*"\$\{/,
  },
  {
    what: "a CEL list element — use `engine in ${celStringList(xs)}`",
    pattern: /\bin\s*\[\$\{[^\n]*"\$\{/,
  },
];

function listSourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (path === generatedRoot) {
      return [];
    }
    if (entry.isDirectory()) {
      return listSourceFiles(path);
    }
    return sourceFilePattern.test(entry.name) ? [path] : [];
  });
}

describe("CEL filter string literals", () => {
  test("no filter builder interpolates a raw value into a CEL literal", () => {
    const violations: string[] = [];
    for (const file of listSourceFiles(srcRoot)) {
      const lines = readFileSync(file, "utf-8").split("\n");
      lines.forEach((line, index) => {
        for (const rule of rawInterpolationRules) {
          if (rule.pattern.test(line)) {
            violations.push(
              `${relative(srcRoot, file)}:${index + 1} — ${rule.what}`
            );
          }
        }
      });
    }
    expect(violations).toEqual([]);
  });
});
