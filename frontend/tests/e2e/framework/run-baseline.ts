import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { TestInfo } from "@playwright/test";

// Returns the value `read` gave the first time this run asked for `key`.
//
// Shared state a suite changes and restores must be captured once per run, not
// once per attempt: a retry restarts the worker and re-runs beforeAll, which
// would otherwise snapshot the previous attempt's change as the baseline, and
// teardown would then pin the shared server there. The project output directory
// is cleared at the start of a run and survives its retries. Specs that change
// the same state share a key, so they restore the same baseline. A `read` that
// throws stores nothing.
export async function runBaseline<T>(
  testInfo: TestInfo,
  key: string,
  read: () => Promise<T>,
): Promise<T> {
  const path = join(testInfo.project.outputDir, `baseline-${key}.json`);
  if (existsSync(path)) {
    return JSON.parse(readFileSync(path, "utf8")) as T;
  }
  const value = await read();
  mkdirSync(testInfo.project.outputDir, { recursive: true });
  writeFileSync(path, JSON.stringify(value));
  return value;
}
