import { act, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  reload: vi.fn(),
  runChecks: vi.fn(),
  section: {} as { onRun?: () => Promise<void>; trailingSummary?: ReactNode },
  sqlReview: undefined as unknown,
}));

vi.mock("@/components/plan-check/PlanCheckSection", () => ({
  PlanCheckSection: (props: {
    onRun: () => Promise<void>;
    trailingSummary?: ReactNode;
  }) => {
    mocks.section = props;
    return <div>{props.trailingSummary}</div>;
  },
}));

vi.mock("@/lib/plan/check", () => ({
  getPlanCheckSummaryWithFallback: () => ({ total: 1 }),
}));

vi.mock("../hooks/usePlanCheckActions", () => ({
  usePlanCheckActions: () => ({
    allowRunChecks: true,
    isRunningChecks: false,
    refreshChecks: vi.fn(),
    runChecks: mocks.runChecks,
  }),
}));

vi.mock("../hooks/useSqlReviewV2", () => ({
  useSqlReviewV2: () => mocks.sqlReview,
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./PlanDetailRuleReviewSummary", () => ({
  PlanDetailRuleReviewSummary: () => <span>rule review summary</span>,
}));

vi.mock("./PlanTargetDisplay", () => ({ PlanTargetDisplay: () => null }));

import { PlanDetailAggregateChecks } from "./PlanDetailAggregateChecks";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = { plan: { name: "projects/foo/plans/7" }, planCheckRuns: [] };
  mocks.runChecks.mockResolvedValue(undefined);
  mocks.reload.mockResolvedValue(undefined);
  mocks.sqlReview = {
    unresolvedThreads: 0,
    ruleReview: {
      loadFailed: false,
      openFindings: 0,
      reload: mocks.reload,
      run: null,
    },
  };
});

test("joins the rule review to the checks under SQL Review V2", async () => {
  const { getByText } = render(<PlanDetailAggregateChecks />);
  expect(getByText("rule review summary")).toBeInTheDocument();

  await act(async () => {
    await mocks.section.onRun?.();
  });
  expect(mocks.runChecks).toHaveBeenCalledTimes(1);
  expect(mocks.reload).toHaveBeenCalledTimes(1);
});

test.each([
  ["SQL Review V2 is off", undefined],
  ["the review cannot be read", { unresolvedThreads: 0 }],
])("leaves the rule review out when %s", async (_, sqlReview) => {
  mocks.sqlReview = sqlReview;
  const { queryByText } = render(<PlanDetailAggregateChecks />);
  expect(queryByText("rule review summary")).not.toBeInTheDocument();

  await act(async () => {
    await mocks.section.onRun?.();
  });
  expect(mocks.runChecks).toHaveBeenCalledTimes(1);
  expect(mocks.reload).not.toHaveBeenCalled();
});
