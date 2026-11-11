import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type ReviewRun,
  ReviewRun_State,
} from "@/types/proto-es/v1/issue_service_pb";

const mocks = vi.hoisted(() => ({
  gate: true,
  page: {} as Record<string, unknown>,
  permissions: new Set<string>(),
  review: {
    run: null,
    stale: false,
    loadFailed: false,
  } as unknown,
  standardRules: { rules: [], source: "workspace" } as unknown,
  threadsLoaded: true,
  unresolvedThreads: 2,
  useEffectiveStandardRules: vi.fn(),
  useReviewRun: vi.fn(),
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

vi.mock("./useEffectiveStandardRules", () => ({
  useEffectiveStandardRules: mocks.useEffectiveStandardRules,
}));

vi.mock("./useReviewRun", () => ({
  useReviewRun: mocks.useReviewRun,
}));

import {
  SqlReviewV2Provider,
  useSqlReviewV2,
  withReviewRun,
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
  mocks.useReviewRun.mockReturnValue(mocks.review);
  mocks.useEffectiveStandardRules.mockReturnValue(mocks.standardRules);
});

describe("SqlReviewV2Provider", () => {
  test("provides the open threads, the review run, and the rules", () => {
    const { result } = renderHook(() => useSqlReviewV2(), { wrapper });
    expect(result.current).toEqual({
      unresolvedThreads: 2,
      threadsLoaded: true,
      review: mocks.review,
      standardRules: mocks.standardRules,
    });
    expect(mocks.useEffectiveStandardRules).toHaveBeenCalledWith(
      "projects/foo",
      true
    );
    expect(mocks.useReviewRun).toHaveBeenCalledWith({
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
      standardRules: mocks.standardRules,
    });
    expect(mocks.useReviewRun).toHaveBeenCalledWith(
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
    expect(mocks.useEffectiveStandardRules).toHaveBeenCalledWith(
      "projects/foo",
      false
    );
  });
});

describe("withReviewRun", () => {
  const checks = { error: 0, running: 0, success: 1, total: 1, warning: 0 };
  const run = (state: ReviewRun_State) => ({ state }) as ReviewRun;

  test("counts a queued or running review as a running check", () => {
    expect(withReviewRun(checks, run(ReviewRun_State.RUNNING))).toEqual({
      ...checks,
      running: 1,
      total: 2,
    });
  });

  test("counts a failed review as a failed check", () => {
    expect(withReviewRun(checks, run(ReviewRun_State.FAILED))).toEqual({
      ...checks,
      error: 1,
      total: 2,
    });
  });

  test("leaves the checks alone for a completed or absent review", () => {
    expect(withReviewRun(checks, run(ReviewRun_State.SUCCEEDED))).toBe(checks);
    expect(withReviewRun(checks, null)).toBe(checks);
    expect(withReviewRun(checks, undefined)).toBe(checks);
  });
});
