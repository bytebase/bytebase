import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";

const repoRoot = process.cwd();
const srcRoot = join(repoRoot, "src");
const sourceFilePattern = /\.(?:cjs|cts|js|jsx|mjs|mts|ts|tsx)$/;
const ignoredDirectories = new Set(["dist", "node_modules", "proto-es"]);
const bannedPattern = /\bcrypto\s*\.\s*randomUUID\s*\(/;

function listSourceFiles(dir) {
  if (!existsSync(dir)) {
    return [];
  }

  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      if (ignoredDirectories.has(entry.name)) {
        return [];
      }
      return listSourceFiles(path);
    }
    return sourceFilePattern.test(entry.name) ? [path] : [];
  });
}

const violations = [];
for (const file of listSourceFiles(srcRoot)) {
  const source = readFileSync(file, "utf-8");
  if (bannedPattern.test(source)) {
    violations.push(relative(repoRoot, file));
  }
}

if (violations.length > 0) {
  console.error(
    "Use v4 from the uuid package instead of crypto.randomUUID() for compatibility:"
  );
  for (const violation of violations) {
    console.error(`  - ${violation}`);
  }
  process.exitCode = 1;
}
