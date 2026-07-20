import { afterEach, describe, expect, test, vi } from "vitest";
import {
  clearPagedDataCache,
  readPagedDataCache,
  writePagedDataCache,
} from "@/react/hooks/pagedDataCache";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { applyProjectDetailMutationResult } from "./applyProjectDetailMutationResult";
import {
  projectIssuesPagedDataCacheScope,
  projectPlansPagedDataCacheScope,
} from "./pagedDataCacheScope";

describe("applyProjectDetailMutationResult", () => {
  afterEach(() => {
    clearPagedDataCache();
  });

  test("invalidates affected list scopes before patching detail state", () => {
    writePagedDataCache(
      "plans",
      { dataList: [], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );
    writePagedDataCache(
      "issues",
      { dataList: [], hasMore: false, nextPageToken: "" },
      projectIssuesPagedDataCacheScope("a")
    );
    const patchState = vi.fn();
    const patch = {
      plan: { name: "projects/a/plans/1" } as Plan,
      issue: { name: "projects/a/issues/1" } as Issue,
    };

    applyProjectDetailMutationResult({ projectId: "a", patchState }, patch);

    expect(readPagedDataCache("plans")).toBeUndefined();
    expect(readPagedDataCache("issues")).toBeUndefined();
    expect(patchState).toHaveBeenCalledWith(patch);
  });

  test("keeps unrelated resource scopes cached", () => {
    writePagedDataCache(
      "plans",
      { dataList: [], hasMore: false, nextPageToken: "" },
      projectPlansPagedDataCacheScope("a")
    );
    writePagedDataCache(
      "issues",
      { dataList: [], hasMore: false, nextPageToken: "" },
      projectIssuesPagedDataCacheScope("a")
    );

    applyProjectDetailMutationResult(
      { projectId: "a", patchState: vi.fn() },
      { plan: { name: "projects/a/plans/1" } as Plan }
    );

    expect(readPagedDataCache("plans")).toBeUndefined();
    expect(readPagedDataCache("issues")).toBeDefined();
  });
});
