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

function prepareLocaleFile(filePath, read = readFileSync) {
  let current;
  try {
    current = read(filePath, "utf-8");
  } catch (error) {
    throw new Error(`Failed to read locale file ${filePath}: ${error.message}`);
  }

  let parsed;
  try {
    parsed = JSON.parse(current);
  } catch (error) {
    throw new Error(`Failed to parse locale file ${filePath}: ${error.message}`);
  }

  return {
    current,
    normalized: `${JSON.stringify(sortObjectKeys(parsed), null, 2)}\n`,
  };
}

export function normalizeLocaleFile(filePath) {
  const { current, normalized } = prepareLocaleFile(filePath);
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

export function sortLocaleFiles(files, io = {}) {
  const read = io.readFileSync ?? readFileSync;
  const write = io.writeFileSync ?? writeFileSync;
  const preparedFiles = files.map((filePath) => ({
    filePath,
    ...prepareLocaleFile(filePath, read),
  }));

  const updated = [];
  for (const file of preparedFiles) {
    if (file.normalized === file.current) {
      continue;
    }

    try {
      write(file.filePath, file.normalized);
    } catch (error) {
      throw new Error(
        `Failed to write locale file ${file.filePath}: ${error.message}`
      );
    }
    updated.push(file.filePath);
  }

  return updated;
}

function main() {
  const files = LOCALE_ROOTS.flatMap((root) => findJsonFiles(root));
  const updated = sortLocaleFiles(files);

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
