import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { dirname, extname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");
const LOCALE_ROOTS = [
  resolve(ROOT, "src/locales"),
  resolve(ROOT, "src/react/locales"),
];

export function sortObjectKeys(value) {
  if (Array.isArray(value) || value === null || typeof value !== "object") {
    return value;
  }

  return Object.fromEntries(
    Object.keys(value)
      .sort()
      .map((key) => [key, sortObjectKeys(value[key])])
  );
}

function findJsonFiles(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...findJsonFiles(fullPath));
      continue;
    }
    if (entry.isFile() && extname(entry.name) === ".json") {
      files.push(fullPath);
    }
  }
  return files;
}

export function normalizeLocaleFile(filePath) {
  let current;
  try {
    current = readFileSync(filePath, "utf-8");
  } catch (error) {
    throw new Error(`Failed to read locale file ${filePath}: ${error.message}`);
  }

  let parsed;
  try {
    parsed = JSON.parse(current);
  } catch (error) {
    throw new Error(`Failed to parse locale file ${filePath}: ${error.message}`);
  }

  const normalized = `${JSON.stringify(sortObjectKeys(parsed), null, 2)}\n`;
  if (normalized === current) {
    return false;
  }

  try {
    writeFileSync(filePath, normalized);
  } catch (error) {
    throw new Error(`Failed to write locale file ${filePath}: ${error.message}`);
  }
  return true;
}

function main() {
  const files = LOCALE_ROOTS.flatMap((root) => findJsonFiles(root));
  const updated = [];
  for (const filePath of files) {
    if (normalizeLocaleFile(filePath)) {
      updated.push(filePath);
    }
  }

  const unchanged = files.length - updated.length;
  if (updated.length === 0) {
    console.log(`Locale sorter: no changes (${files.length} file(s) checked).`);
    return;
  }

  console.log(
    `Locale sorter: updated ${updated.length} file(s), left ${unchanged} unchanged.`
  );
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main();
}
