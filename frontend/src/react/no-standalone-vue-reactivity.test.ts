import { existsSync, readdirSync, readFileSync } from "node:fs";
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
const packageJsonPath = join(repoRoot, "package.json");

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
    const packageJson = JSON.parse(readFileSync(packageJsonPath, "utf-8")) as {
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };

    expect(packageJson.dependencies?.["@vueuse/core"]).toBeUndefined();
    expect(packageJson.devDependencies?.["@vueuse/core"]).toBeUndefined();
  });
});

describe("Vue build tooling", () => {
  test("does not configure Vue Vite plugins or generated component registries", () => {
    const viteConfig = readFileSync(join(repoRoot, "vite.config.ts"), "utf-8");

    expect(viteConfig).not.toMatch(/@vitejs\/plugin-vue\b/);
    expect(viteConfig).not.toMatch(/@vitejs\/plugin-vue-jsx\b/);
    expect(viteConfig).not.toMatch(/unplugin-vue-components/);
    expect(viteConfig).not.toMatch(/\bvue\(\)/);
    expect(viteConfig).not.toMatch(/\bvueJsx\(/);
    expect(viteConfig).not.toMatch(/\bComponents\(/);
    expect(viteConfig).toContain("explain-visualizer-vue");
    expect(existsSync(join(repoRoot, "components.d.ts"))).toBe(false);
  });

  test("does not keep ESLint configuration", () => {
    expect(existsSync(join(repoRoot, "eslint.config.mjs"))).toBe(false);
  });

  test("does not keep Vue type-check or build-only package entries", () => {
    const packageJson = JSON.parse(readFileSync(packageJsonPath, "utf-8")) as {
      scripts?: Record<string, string>;
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    const dependencyNames = Object.keys(packageJson.dependencies ?? {});
    const devDependencyNames = Object.keys(packageJson.devDependencies ?? {});

    expect(packageJson.scripts?.["type-check"]).not.toContain("vue-tsc");
    expect(dependencyNames.filter((name) => name.startsWith("@vue/"))).toEqual(
      []
    );
    expect(
      dependencyNames.filter((name) => name.startsWith("@vueuse/"))
    ).toEqual([]);
    expect(
      dependencyNames.filter((name) => name.startsWith("@intlify/"))
    ).toEqual([]);
    expect(
      devDependencyNames.filter(
        (name) =>
          name.startsWith("@vue/") ||
          name.startsWith("@vueuse/") ||
          name.startsWith("@intlify/") ||
          name.startsWith("@vitejs/plugin-vue") ||
          [
            "eslint-plugin-vue",
            "unplugin-vue-components",
            "vue-component-type-helpers",
            "vue-tsc",
          ].includes(name)
      )
    ).toEqual([]);
    expect(existsSync(join(repoRoot, "tsconfig.react.json"))).toBe(false);
    expect(existsSync(join(srcRoot, "react", "tsconfig.json"))).toBe(false);
  });
});

describe("frontend lint tooling", () => {
  test("uses Biome as the single frontend linter", () => {
    const packageJson = JSON.parse(readFileSync(packageJsonPath, "utf-8")) as {
      scripts?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    const scripts = Object.values(packageJson.scripts ?? {});

    expect(existsSync(join(repoRoot, "eslint.config.mjs"))).toBe(false);
    expect(scripts.some((script) => /\beslint\b/.test(script))).toBe(false);
    expect(packageJson.devDependencies?.["eslint"]).toBeUndefined();
    expect(packageJson.devDependencies?.["typescript-eslint"]).toBeUndefined();
  });

  test("keeps frontend compatibility policy checks outside ESLint", () => {
    const packageJson = JSON.parse(readFileSync(packageJsonPath, "utf-8")) as {
      scripts?: Record<string, string>;
    };

    expect(
      existsSync(join(repoRoot, "scripts", "check-no-crypto-randomuuid.mjs"))
    ).toBe(true);
    expect(packageJson.scripts?.["check"]).toContain(
      "node scripts/check-no-crypto-randomuuid.mjs"
    );
  });
});
