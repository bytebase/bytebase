import { describe, expect, test } from "vitest";

import source from "./ProjectDataExportPage.tsx?raw";

describe("data export issue list creator cache", () => {
  test("fetches listed issue creators before rendering fallback names", () => {
    expect(source).toMatch(
      /const batchGetOrFetchUsers = useAppStore\(\s*\(state\) => state\.batchGetOrFetchUsers\s*\)/
    );
    expect(source).toMatch(
      /batchGetOrFetchUsers\(paged\.dataList\.map\(\(issue\) => issue\.creator\)\)/
    );
  });
});
