import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import {
  type ReviewRun,
  ReviewRun_Status,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { RuleReviewRunState } from "../hooks/useRuleReviewRun";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count === undefined ? key : `${key}:${options.count}`,
  }),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

import { PlanDetailRuleReviewSummary } from "./PlanDetailRuleReviewSummary";

const reviewRun = (status: ReviewRun_Status, extra = {}) =>
  ({
    name: "projects/foo/issues/7/reviewRuns/rule",
    type: ReviewRun_Type.RULE,
    status,
    error: "",
    ...extra,
  }) as ReviewRun;

const renderSummary = (state: Partial<RuleReviewRunState>) =>
  render(
    <PlanDetailRuleReviewSummary
      state={{
        loadFailed: false,
        openFindings: 0,
        reload: vi.fn(),
        run: undefined,
        ...state,
      }}
    />
  );

test("renders nothing while loading", () => {
  const { container } = renderSummary({});
  expect(container).toBeEmptyDOMElement();
});

test("says the review never ran", () => {
  renderSummary({ run: null });
  expect(screen.getByText("plan.rule-review.not-run")).toBeVisible();
});

test.each([
  [ReviewRun_Status.AVAILABLE, "plan.rule-review.queued"],
  [ReviewRun_Status.RUNNING, "plan.rule-review.running"],
])("shows a run in progress", (status, label) => {
  renderSummary({ run: reviewRun(status) });
  expect(screen.getByText(label)).toBeVisible();
});

test("counts the open findings of a completed run", () => {
  renderSummary({
    openFindings: 3,
    run: reviewRun(ReviewRun_Status.DONE, {
      endTime: { seconds: 5n, nanos: 0 },
    }),
  });
  expect(screen.getByText("plan.rule-review.open-findings:3")).toBeVisible();
  expect(screen.getByTestId("humanize-ts")).toHaveTextContent("5000");
});

test("says a completed run found nothing", () => {
  renderSummary({ run: reviewRun(ReviewRun_Status.DONE) });
  expect(screen.getByText("plan.rule-review.no-open-findings")).toBeVisible();
});

test("shows why a run failed", () => {
  renderSummary({
    run: reviewRun(ReviewRun_Status.FAILED, {
      error: "1 of 1 review units failed",
    }),
  });
  expect(screen.getByText("plan.rule-review.failed")).toBeVisible();
  expect(screen.getByText("1 of 1 review units failed")).toBeVisible();
});

test("offers a retry when the status could not be read", () => {
  const reload = vi.fn();
  renderSummary({ loadFailed: true, reload });
  expect(screen.getByText("plan.rule-review.load-failed")).toBeVisible();
  fireEvent.click(
    screen.getByRole("button", { name: "sql-review.standard-rules.retry" })
  );
  expect(reload).toHaveBeenCalledTimes(1);
});
