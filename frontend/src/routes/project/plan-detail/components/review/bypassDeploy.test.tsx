import { act, renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  type Issue,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

const mocks = vi.hoisted(() => ({
  createRollout: vi.fn(),
  page: {} as Record<string, unknown>,
  sqlReview: undefined as unknown,
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count === undefined ? key : `${key}:${options.count}`,
  }),
}));

vi.mock("@/api", () => ({
  rolloutServiceClientConnect: { createRollout: mocks.createRollout },
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("../../hooks/useSqlReviewV2", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../hooks/useSqlReviewV2")>()),
  useSqlReviewV2: () => mocks.sqlReview,
}));

vi.mock("../../shell/focusPhase", () => ({ focusPlanPhase: vi.fn() }));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

import { useBypassDeploy } from "./bypassDeploy";

const issue = {
  approvalStatus: ApprovalStatus.APPROVED,
  status: IssueStatus.OPEN,
} as Issue;
const plan = {
  hasRollout: false,
  name: "projects/foo/plans/7",
  planCheckRunStatusCount: { SUCCESS: 1 },
} as unknown as Plan;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.createRollout.mockResolvedValue({});
  mocks.page = {
    expandPhase: vi.fn(),
    planCheckRuns: [],
    projectCanCreateRollout: true,
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    readonly: false,
    refreshState: vi.fn().mockResolvedValue(undefined),
  };
});

test("asks to acknowledge open threads before deploying past them", async () => {
  mocks.sqlReview = {
    unresolvedThreads: 2,
    ruleReview: {
      run: { status: ReviewRun_Status.DONE },
      stale: false,
      loadFailed: false,
    },
  };
  const { result } = renderHook(() => useBypassDeploy(issue, plan));
  expect(result.current.state.kind).toBe("approved-threads-open");
  expect(result.current.weight).toBe("link");
  expect(result.current.warnings).toContain(
    "plan.review.footer.confirm-unresolved-threads:2"
  );
  expect(result.current.confirmDisabled).toBe(true);

  act(() => result.current.triggerDeploy());
  expect(result.current.confirmOpen).toBe(true);
  expect(mocks.createRollout).not.toHaveBeenCalled();

  act(() => result.current.setAcknowledge(true));
  expect(result.current.confirmDisabled).toBe(false);
});

test("counts a failed rule review as a failed check", () => {
  mocks.sqlReview = {
    unresolvedThreads: 0,
    ruleReview: { run: { status: ReviewRun_Status.FAILED } },
  };
  const { result } = renderHook(() => useBypassDeploy(issue, plan));
  expect(result.current.state.kind).toBe("approved-checks-failed");
  expect(result.current.checks.error).toBe(1);
});

test.each([
  ["without permission to read it", undefined],
  ["while it loads", { run: undefined, stale: true, loadFailed: false }],
  [
    "when it fails to load",
    { run: { status: ReviewRun_Status.DONE }, stale: false, loadFailed: true },
  ],
  [
    "before it reads the plan's latest change",
    { run: { status: ReviewRun_Status.DONE }, stale: true, loadFailed: false },
  ],
])("asks to confirm deploying past a rule review it cannot read %s", (_, ruleReview) => {
  mocks.sqlReview = { unresolvedThreads: 0, ruleReview };
  const { result } = renderHook(() => useBypassDeploy(issue, plan));
  expect(result.current.state.kind).toBe("all-gates-passed");
  expect(result.current.warnings).toEqual([
    "plan.review.footer.confirm-rule-review-unknown",
  ]);

  act(() => result.current.triggerDeploy());
  expect(result.current.confirmOpen).toBe(true);
  expect(mocks.createRollout).not.toHaveBeenCalled();
});

test("deploys at once after the rule review is done", async () => {
  mocks.sqlReview = {
    unresolvedThreads: 0,
    ruleReview: {
      run: { status: ReviewRun_Status.DONE },
      stale: false,
      loadFailed: false,
    },
  };
  const { result } = renderHook(() => useBypassDeploy(issue, plan));
  expect(result.current.warnings).toEqual([]);

  await act(async () => {
    result.current.triggerDeploy();
  });
  expect(mocks.createRollout).toHaveBeenCalledTimes(1);
});

test("deploys at once where SQL Review V2 is off", async () => {
  mocks.sqlReview = undefined;
  const { result } = renderHook(() => useBypassDeploy(issue, plan));
  expect(result.current.state.kind).toBe("all-gates-passed");
  expect(result.current.warnings).toEqual([]);

  await act(async () => {
    result.current.triggerDeploy();
  });
  expect(mocks.createRollout).toHaveBeenCalledTimes(1);
});
