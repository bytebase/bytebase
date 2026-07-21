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

  test("keeps presets and rows in one bordered list surface", () => {
    expect(source.indexOf("<IssueListPanel")).toBeGreaterThan(
      source.indexOf("<ProjectPageContent>")
    );
    expect(source).toContain('<ProjectPageFooter className="px-2">');
  });
});
