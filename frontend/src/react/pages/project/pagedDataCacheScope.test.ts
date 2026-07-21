import { beforeEach, describe, expect, test } from "vitest";
import {
  clearPagedDataCache,
  readPagedDataCache,
  writePagedDataCache,
} from "@/react/hooks/pagedDataCache";
import {
  invalidateProjectPagedDataCacheIfChanged,
  projectIssuesPagedDataCacheScope,
  projectPlansPagedDataCacheScope,
} from "./pagedDataCacheScope";

describe("project paged data cache scopes", () => {
  beforeEach(clearPagedDataCache);

  test("keeps resource types and projects isolated", () => {
    expect(projectPlansPagedDataCacheScope("a")).not.toBe(
      projectIssuesPagedDataCacheScope("a")
    );
    expect(projectPlansPagedDataCacheScope("a")).not.toBe(
      projectPlansPagedDataCacheScope("b")
    );
  });

  test("returns a stable scope", () => {
    expect(projectPlansPagedDataCacheScope("a")).toBe(
      projectPlansPagedDataCacheScope("a")
    );
  });

  test("invalidates only the changed resource scope", () => {
    writePagedDataCache(
      "plans",
      { dataList: ["plan"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );
    writePagedDataCache(
      "issues",
      { dataList: ["issue"], hasMore: false, nextPageToken: "" },
      projectIssuesPagedDataCacheScope("a")
    );

    invalidateProjectPagedDataCacheIfChanged(
      "a",
      "plans",
      { name: "projects/a/plans/1", updateTime: { seconds: 1n, nanos: 0 } },
      { name: "projects/a/plans/1", updateTime: { seconds: 2n, nanos: 0 } }
    );

    expect(readPagedDataCache("plans")).toBeUndefined();
    expect(readPagedDataCache("issues")?.dataList).toEqual(["issue"]);
  });

  test("keeps the cache for initial or unchanged resources", () => {
    writePagedDataCache(
      "plans",
      { dataList: ["plan"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );

    invalidateProjectPagedDataCacheIfChanged("a", "plans", undefined, {
      name: "projects/a/plans/1",
      updateTime: { seconds: 1n, nanos: 0 },
    });
    invalidateProjectPagedDataCacheIfChanged(
      "a",
      "plans",
      { name: "projects/a/plans/1", updateTime: { seconds: 1n, nanos: 0 } },
      { name: "projects/a/plans/1", updateTime: { seconds: 1n, nanos: 0 } }
    );

    expect(readPagedDataCache("plans")?.dataList).toEqual(["plan"]);
  });

  test("keeps the cache when the detail route changes resources", () => {
    writePagedDataCache(
      "plans",
      { dataList: ["plan"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );

    invalidateProjectPagedDataCacheIfChanged(
      "a",
      "plans",
      { name: "projects/a/plans/1", updateTime: { seconds: 1n, nanos: 0 } },
      { name: "projects/b/plans/2", updateTime: { seconds: 2n, nanos: 0 } }
    );

    expect(readPagedDataCache("plans")?.dataList).toEqual(["plan"]);
  });
});
