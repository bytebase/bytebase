import { renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

const ISSUE = "projects/foo/issues/7";

const mocks = vi.hoisted(() => ({
  issueCommentsByIssue: {} as Record<string, unknown[]>,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      issueCommentsByIssue: Record<string, unknown[]>;
    }) => unknown
  ) => selector({ issueCommentsByIssue: mocks.issueCommentsByIssue }),
}));

import { useIssueThreadsLoaded } from "./useUnresolvedThreadCounts";

beforeEach(() => {
  mocks.issueCommentsByIssue = {};
});

test("reports the threads unloaded until the issue's comments load", () => {
  const { result, rerender } = renderHook(() => useIssueThreadsLoaded(ISSUE));
  expect(result.current).toBe(false);

  mocks.issueCommentsByIssue = { [ISSUE]: [] };
  rerender();
  expect(result.current).toBe(true);
});

test("reports the threads loaded before an issue exists", () => {
  const { result } = renderHook(() => useIssueThreadsLoaded(undefined));
  expect(result.current).toBe(true);
});
