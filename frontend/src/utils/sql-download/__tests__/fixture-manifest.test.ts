// fixture-manifest.test.ts

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { FIXTURES } from "./fixtures";

const here = dirname(fileURLToPath(import.meta.url));

describe("fixture id manifest", () => {
  it("TS FIXTURES.keys equals Go-emitted goldens/fixture_ids.txt", () => {
    const manifestPath = resolve(here, "goldens/fixture_ids.txt");
    let manifest: string;
    try {
      manifest = readFileSync(manifestPath, "utf8");
    } catch {
      throw new Error(
        `goldens/fixture_ids.txt is missing. Generate goldens first:\n` +
          `  go test ./backend/component/export -run TestDownloadGoldens -update`
      );
    }
    const goIds = manifest.trimEnd().split("\n").sort();
    const tsIds = Object.keys(FIXTURES).sort();
    expect(tsIds).toEqual(goIds);
    // If this fails, either:
    //   - Add the missing fixture in TS frontend/src/utils/sql-download/__tests__/fixtures.ts to match Go
    //   - OR remove it from Go backend/component/export/download_goldens_fixtures.go
    //   - Then regenerate: go test ./backend/component/export -run TestDownloadGoldens -update
  });
});
