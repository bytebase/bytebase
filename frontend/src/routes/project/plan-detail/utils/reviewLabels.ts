import type {
  AutomatedReviewer,
  AutomatedReviewRow,
  FindingGroup,
  FindingPriority,
} from "./planGates";

type T = (key: string, options?: Record<string, unknown>) => string;

export const reviewerLabel = (t: T, reviewer: AutomatedReviewer): string => {
  switch (reviewer) {
    case "rule":
      return t("sql-review.standard-rules.self");
    case "ai":
      return t("sql-review.ai-review.self");
    case "policy":
      return t("plan.automated-review.policy");
    case "ghost":
      return t("plan.automated-review.ghost");
    case "report":
      return t("plan.automated-review.report");
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
  row.reviewer !== "rule" && row.reviewer !== "ai";

const PRIORITIES: readonly FindingPriority[] = ["p0", "p1", "p2"];

// A reviewer's open findings, P0 first: "1 P0, 2 P1". A plan check reports
// errors rather than priorities.
export const openFindingsLabel = (t: T, row: AutomatedReviewRow): string =>
  isPlanCheckRow(row)
    ? t("plan.automated-review.n-errors", { count: row.findings.open })
    : PRIORITIES.filter((priority) => row.findings[priority] > 0)
        .map((priority) =>
          t("plan.findings.n-priority", {
            count: row.findings[priority],
            priority: priority.toUpperCase(),
          })
        )
        .join(t("plan.findings.separator"));

// A held reviewer as a line of what the rollout waits on.
export const reviewGateLabel = (t: T, row: AutomatedReviewRow): string => {
  const reviewer = reviewerLabel(t, row.reviewer);
  switch (row.conclusion) {
    case "open":
      return t("plan.deploy-waiting.reviewer-findings", {
        reviewer,
        findings: openFindingsLabel(t, row),
      });
    case "failed":
      return t("plan.deploy-waiting.reviewer-failed", { reviewer });
    case "outdated":
      return t("plan.deploy-waiting.reviewer-outdated", { reviewer });
    case "not-run":
      return t("plan.deploy-waiting.reviewer-not-run", { reviewer });
    default:
      return t("plan.deploy-waiting.reviewer-running", { reviewer });
  }
};

// A reviewer's conclusion in a collapsed summary, saying what people decided
// about its findings: "Standard rules: 2 resolved, 1 P0 resolved without a
// change".
export const reviewSummaryPart = (t: T, row: AutomatedReviewRow): string => {
  const reviewer = reviewerLabel(t, row.reviewer);
  const { findings } = row;
  let status: string;
  switch (row.conclusion) {
    case "open":
      status = openFindingsLabel(t, row);
      break;
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
    default:
      status = t("plan.automated-review.running");
  }
  return t("plan.automated-review.summary-part", { reviewer, status });
};
