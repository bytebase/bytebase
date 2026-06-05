import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const __dirname = dirname(fileURLToPath(import.meta.url));
const srcDir = resolve(__dirname, "..");
const sourceFilePattern = /\.(?:ts|tsx|vue)$/;

function listSourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === "locales" || entry.name === "proto-es") {
        return [];
      }
      return listSourceFiles(path);
    }
    return sourceFilePattern.test(entry.name) ? [path] : [];
  });
}

describe("canonical locale resources", () => {
  test("keeps app locale JSON under src/locales only", () => {
    expect(existsSync(resolve(srcDir, "locales"))).toBe(true);
    expect(existsSync(resolve(srcDir, "react/locales"))).toBe(false);
  });

  test("loads React i18n resources from the canonical locale root", () => {
    const source = readFileSync(resolve(srcDir, "react/i18n.ts"), "utf-8");

    expect(source).toContain("@/locales/");
    expect(source).not.toContain("@/react/locales/");
  });

  test("uses the React i18n module as the single runtime entrypoint", () => {
    expect(existsSync(resolve(srcDir, "plugins/i18n.ts"))).toBe(false);

    const legacyEntrypoint = ["@", "plugins", "i18n"].join("/");
    const violations = listSourceFiles(srcDir).filter((file) =>
      readFileSync(file, "utf-8").includes(legacyEntrypoint)
    );
    expect(violations).toEqual([]);
  });

  test("checks i18n keys across shared non-React callers", () => {
    const source = readFileSync(
      resolve(srcDir, "../scripts/check-react-i18n.mjs"),
      "utf-8"
    );

    expect(source).toContain('const SOURCE_DIR = resolve(ROOT, "src")');
    expect(source).toContain("SOURCE_SCAN_DIRS");
    expect(source).not.toContain(
      'const REACT_DIR = resolve(ROOT, "src/react")'
    );
  });
});
