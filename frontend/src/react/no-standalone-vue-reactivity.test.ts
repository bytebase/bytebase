import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, test } from "vitest";

const repoRoot = process.cwd();
const srcRoot = join(repoRoot, "src");
const allowedVueImportPaths = new Set([
  "react/explain-visualizer/PostgresPlanView.tsx",
]);
const sourceFilePattern = /\.(?:ts|tsx|d\.ts)$/;
const bannedVueImportPattern = /from\s+["']vue["']/g;
const bannedVueUseImportPattern = /from\s+["']@vueuse\/[^"']+["']/g;

function listSourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      return listSourceFiles(path);
    }
    return sourceFilePattern.test(entry.name) ? [path] : [];
  });
}

describe("standalone Vue reactivity dependencies", () => {
  test("only the explain visualizer imports Vue runtime APIs", () => {
    const violations: string[] = [];
    for (const file of listSourceFiles(srcRoot)) {
      const source = readFileSync(file, "utf-8");
      const relativePath = relative(srcRoot, file);
      const importsVue = bannedVueImportPattern.test(source);
      const importsVueUse = bannedVueUseImportPattern.test(source);
      bannedVueImportPattern.lastIndex = 0;
      bannedVueUseImportPattern.lastIndex = 0;
      if (
        (importsVue || importsVueUse) &&
        !allowedVueImportPaths.has(relativePath)
      ) {
        violations.push(relativePath);
      }
    }
    expect(violations).toEqual([]);
  });

  test("does not depend on @vueuse/core", () => {
    const packageJson = JSON.parse(
      readFileSync(join(repoRoot, "package.json"), "utf-8")
    ) as {
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };

    expect(packageJson.dependencies?.["@vueuse/core"]).toBeUndefined();
    expect(packageJson.devDependencies?.["@vueuse/core"]).toBeUndefined();
  });
});
