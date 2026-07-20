import { describe, expect, test } from "vitest";
import {
  projectIssuesPagedDataCacheScope,
  projectPlansPagedDataCacheScope,
} from "./pagedDataCacheScope";

describe("project paged data cache scopes", () => {
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
});
