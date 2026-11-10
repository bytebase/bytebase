import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type ReviewRun,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";

const mocks = vi.hoisted(() => ({
  gate: true,
  page: {} as Record<string, unknown>,
  permissions: new Set<string>(),
  ruleReview: {
    run: null,
    stale: false,
    loadFailed: false,
    openFindings: 0,
  } as unknown,
  threadsLoaded: true,
  unresolvedThreads: 2,
  useRuleReviewRun: vi.fn(),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/foo" }),
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions.has(permission),
}));

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.gate,
}));

vi.mock("../components/threads/useUnresolvedThreadCounts", () => ({
  useIssueThreadsLoaded: () => mocks.threadsLoaded,
  useUnresolvedThreadTotal: () => mocks.unresolvedThreads,
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./useRuleReviewRun", () => ({
  useRuleReviewRun: mocks.useRuleReviewRun,
}));

import {
  SqlReviewV2Provider,
  useSqlReviewV2,
  withRuleReview,
} from "./useSqlReviewV2";

const wrapper = ({ children }: { children: ReactNode }) => (
  <SqlReviewV2Provider>{children}</SqlReviewV2Provider>
);

beforeEach(() => {
  vi.clearAllMocks();
  mocks.gate = true;
  mocks.permissions = new Set(["bb.reviewRuns.get"]);
  mocks.threadsLoaded = true;
  mocks.page = {
    issue: { name: "projects/foo/issues/7" },
    plan: { updateTime: { seconds: 9n, nanos: 1 } },
    projectId: "foo",
  };
  mocks.useRuleReviewRun.mockReturnValue(mocks.ruleReview);
});

describe("SqlReviewV2Provider", () => {
  test("provides the open threads and the rule review", () => {
    const { result } = renderHook(() => useSqlReviewV2(), { wrapper });
    expect(result.current).toEqual({
      unresolvedThreads: 2,
      threadsLoaded: true,
      ruleReview: mocks.ruleReview,
    });
    expect(mocks.useRuleReviewRun).toHaveBeenCalledWith({
      enabled: true,
      issueName: "projects/foo/issues/7",
      reloadKey: "9:1",
    });
  });

  test("still counts threads without permission to read the review", () => {
    mocks.permissions.clear();
    const { result } = renderHook(() => useSqlReviewV2(), { wrapper });
    expect(result.current).toEqual({
      unresolvedThreads: 2,
      threadsLoaded: true,
    });
    expect(mocks.useRuleReviewRun).toHaveBeenCalledWith(
      expect.objectContaining({ enabled: false })
    );
  });

  test("says when the threads have not loaded", () => {
    mocks.threadsLoaded = false;
    const { result } = renderHook(() => useSqlReviewV2(), { wrapper });
    expect(result.current?.threadsLoaded).toBe(false);
  });

  test.each([
    [
      "SQL Review V2 is off",
      () => {
        mocks.gate = false;
      },
    ],
    [
      "the plan has no issue",
      () => {
        mocks.page.issue = undefined;
      },
    ],
  ])("provides nothing when %s", (_, arrange) => {
    arrange();
    const { result } = renderHook(() => useSqlReviewV2(), { wrapper });
    expect(result.current).toBeUndefined();
  });
});

describe("withRuleReview", () => {
  const checks = { error: 0, running: 0, success: 1, total: 1, warning: 0 };
  const run = (status: ReviewRun_Status) => ({ status }) as ReviewRun;

  test("counts a queued or running review as a running check", () => {
    for (const status of [ReviewRun_Status.AVAILABLE, ReviewRun_Status.RUNNING]) {
      expect(withRuleReview(checks, run(status))).toEqual({
        ...checks,
        running: 1,
        total: 2,
      });
    }
  });

  test("counts a failed review as a failed check", () => {
    expect(withRuleReview(checks, run(ReviewRun_Status.FAILED))).toEqual({
      ...checks,
      error: 1,
      total: 2,
    });
  });

  test("leaves the checks alone for a completed or absent review", () => {
    expect(withRuleReview(checks, run(ReviewRun_Status.DONE))).toBe(checks);
    expect(withRuleReview(checks, null)).toBe(checks);
    expect(withRuleReview(checks, undefined)).toBe(checks);
  });
});
