import { describe, expect, test } from "vitest";

import source from "./ProjectIssueDashboardPage.tsx?raw";

describe("project issue list creator cache", () => {
  test("fetches listed issue creators before rendering fallback names", () => {
    expect(source).toMatch(
      /const batchGetOrFetchUsers = useAppStore\(\s*\(state\) => state\.batchGetOrFetchUsers\s*\)/
    );
    expect(source).toMatch(
      /batchGetOrFetchUsers\(paged\.dataList\.map\(\(issue\) => issue\.creator\)\)/
    );
  });
});
