import { beforeEach, describe, expect, test } from "vitest";
import {
  clearPagedDataCache,
  readPagedDataCache,
  writePagedDataCache,
} from "@/hooks/pagedDataCache";
import {
  invalidateProjectPagedDataCacheIfChanged,
  invalidateProjectPlansPagedDataCacheForIssues,
  projectIssuesPagedDataCacheScope,
  projectPlansPagedDataCacheScope,
} from "./projectPagedDataCache";

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

  test("invalidates Plan caches for every Plan-backed Issue project", () => {
    writePagedDataCache(
      "plans-a",
      { dataList: ["plan-a"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );
    writePagedDataCache(
      "plans-b",
      { dataList: ["plan-b"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("b")
    );
    writePagedDataCache(
      "plans-c",
      { dataList: ["plan-c"], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("c")
    );

    invalidateProjectPlansPagedDataCacheForIssues([
      { name: "projects/a/issues/1", plan: "projects/a/plans/1" },
      { name: "projects/a/issues/2", plan: "projects/a/plans/2" },
      { name: "projects/b/issues/1", plan: "projects/b/plans/1" },
      { name: "projects/c/issues/1", plan: "" },
    ]);

    expect(readPagedDataCache("plans-a")).toBeUndefined();
    expect(readPagedDataCache("plans-b")).toBeUndefined();
    expect(readPagedDataCache("plans-c")?.dataList).toEqual(["plan-c"]);
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
