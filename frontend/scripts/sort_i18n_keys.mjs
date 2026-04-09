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

export function checkLocaleFiles(files, io = {}) {
  const read = io.readFileSync ?? readFileSync;
  const preparedFiles = files.map((filePath) => ({
    filePath,
    ...prepareLocaleFile(filePath, read),
  }));

  return preparedFiles
    .filter((file) => file.normalized !== file.current)
    .map((file) => file.filePath);
}

export function sortLocaleFiles(files, io = {}) {
  const read = io.readFileSync ?? readFileSync;
  const write = io.writeFileSync ?? writeFileSync;
  const preparedFiles = files.map((filePath) => ({
    filePath,
    ...prepareLocaleFile(filePath, read),
  }));

  const updated = [];
  const originalContents = new Map();
  for (const file of preparedFiles) {
    if (file.normalized === file.current) {
      continue;
    }

    originalContents.set(file.filePath, file.current);
    try {
      write(file.filePath, file.normalized);
    } catch (error) {
      const writeError = error instanceof Error ? error : new Error(String(error));
      const rollbackErrors = [];
      const rollbackTargets = [...updated].reverse();
      rollbackTargets.push(file.filePath);
      for (const updatedFile of rollbackTargets) {
        try {
          write(updatedFile, originalContents.get(updatedFile));
        } catch (rollbackError) {
          rollbackErrors.push(
            `Failed to restore locale file ${updatedFile}: ${
              rollbackError instanceof Error ? rollbackError.message : String(rollbackError)
            }`
          );
        }
      }

      if (rollbackErrors.length > 0) {
        throw new Error(
          `Failed to write locale file ${file.filePath}: ${writeError.message}. ` +
            `Rollback also failed: ${rollbackErrors.join("; ")}`
        );
      }

      throw new Error(
        `Failed to write locale file ${file.filePath}: ${writeError.message}. ` +
          `Rolled back ${updated.length} earlier file(s) and restored the current file.`
      );
    }
    updated.push(file.filePath);
  }

  return updated;
}

function main() {
  const checkOnly = process.argv.includes("--check");
  const files = LOCALE_ROOTS.flatMap((root) => findJsonFiles(root));
  const updated = checkOnly ? checkLocaleFiles(files) : sortLocaleFiles(files);

  if (checkOnly) {
    if (updated.length === 0) {
      console.log(`Locale sorter: all ${files.length} file(s) are normalized.`);
      return;
    }

    console.error(
      `Locale sorter: ${updated.length} file(s) need sorting:\n` +
        updated.map((filePath) => `  - ${filePath}`).join("\n")
    );
    process.exitCode = 1;
    return;
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
