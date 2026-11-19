import { sqlReviewV2Enabled } from "@/utils/featureGates";
import type {
  AutomatedReviewer,
  AutomatedReviewRow,
  FindingGroup,
} from "./planGates";

type T = (key: string, options?: Record<string, unknown>) => string;

// The review's name, which every review result carries as its author. Under
// SQL Review V2 it is one review, whichever part of it found the problem.
// Beside V1's plan checks it is the AI review alone, and the plan checks'
// SQL review keeps its own name.
export const reviewerName = (t: T): string =>
  sqlReviewV2Enabled()
    ? t("sql-review.reviewer")
    : t("sql-review.ai-review.self");

export const reviewerLabel = (t: T, reviewer: AutomatedReviewer): string => {
  switch (reviewer) {
    case "review":
      return reviewerName(t);
    case "policy":
      return sqlReviewV2Enabled()
        ? t("plan.automated-review.policy")
        : t("task.check-type.sql-review.self");
    case "ghost":
      return t("plan.automated-review.ghost");
    case "checks":
      return t("plan.automated-review.checks");
  }
};

export const findingGroupLabel = (t: T, group: FindingGroup): string => {
  switch (group) {
    case "p0":
    case "p1":
    case "p2":
      return group.toUpperCase();
    case "open":
      return t("plan.findings.set.open");
    case "overridden":
      return t("plan.findings.set.overridden");
    case "resolved":
      return t("plan.findings.set.resolved");
    case "earlier":
      return t("plan.findings.set.earlier");
  }
};

export const isPlanCheckRow = (row: AutomatedReviewRow): boolean =>
  row.reviewer !== "review";

// A part of the collapsed Review summary: text, or a reviewer whose open
// findings the page draws as it does where the rollout waits on them.
export type ReviewSummaryPart = string | AutomatedReviewRow;

// A reviewer's conclusion in a collapsed summary, saying what people decided
// about its findings: "SQL review: 2 resolved, 1 P0 resolved without a
// change".
export const reviewSummaryPart = (
  t: T,
  row: AutomatedReviewRow
): ReviewSummaryPart => {
  if (row.conclusion === "open") return row;
  const reviewer = reviewerLabel(t, row.reviewer);
  const { findings } = row;
  let status: string;
  switch (row.conclusion) {
    case "resolved":
      status = [
        findings.resolved > 0
          ? t("plan.findings.n-resolved", { count: findings.resolved })
          : "",
        findings.overridden > 0
          ? t("plan.findings.n-overridden", { count: findings.overridden })
          : "",
      ]
        .filter(Boolean)
        .join(t("plan.findings.separator"));
      break;
    case "passed":
      status = t("plan.automated-review.passed");
      break;
    case "failed":
      status = t("plan.automated-review.failed");
      break;
    case "outdated":
      status = t("plan.automated-review.out-of-date");
      break;
    case "not-run":
      status = t("plan.automated-review.not-run");
      break;
    case "off":
      status = t("plan.automated-review.off");
      break;
    default:
      status = t("plan.automated-review.running");
  }
  return t("plan.automated-review.summary-part", { reviewer, status });
};
