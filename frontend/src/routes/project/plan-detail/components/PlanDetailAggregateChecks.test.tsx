import { act, render } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  page: {} as Record<string, unknown>,
  runChecks: vi.fn(),
  section: {} as { onRun?: () => Promise<void> },
}));

vi.mock("@/components/plan-check/PlanCheckSection", () => ({
  PlanCheckSection: (props: { onRun: () => Promise<void> }) => {
    mocks.section = props;
    return <div />;
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

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

vi.mock("./PlanTargetDisplay", () => ({ PlanTargetDisplay: () => null }));

import { PlanDetailAggregateChecks } from "./PlanDetailAggregateChecks";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.page = { plan: { name: "projects/foo/plans/7" }, planCheckRuns: [] };
  mocks.runChecks.mockResolvedValue(undefined);
});

test("runs the plan checks from the footer", async () => {
  render(<PlanDetailAggregateChecks />);
  await act(async () => {
    await mocks.section.onRun?.();
  });
  expect(mocks.runChecks).toHaveBeenCalledTimes(1);
});
