import type {
  AutomatedReviewer,
  AutomatedReviewRow,
  FindingAction,
  FindingSet,
  ReviewerFindings,
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
  }
};

export const findingSetLabel = (t: T, set: FindingSet): string => {
  switch (set) {
    case "fix":
      return t("plan.findings.set.fix");
    case "confirm":
      return t("plan.findings.set.confirm");
    case "advice":
      return t("plan.findings.set.advice");
    case "overridden":
      return t("plan.findings.set.overridden");
    case "accepted":
      return t("plan.findings.set.accepted");
    case "dismissed":
      return t("plan.findings.set.dismissed");
    case "earlier":
      return t("plan.findings.set.earlier");
  }
};

const ACTIONS: readonly FindingAction[] = ["fix", "confirm", "advice"];

// The open findings by what they ask, largest stakes first: "1 must fix".
export const openActionParts = (
  t: T,
  findings: ReviewerFindings
): { action: FindingAction; label: string }[] =>
  ACTIONS.filter((action) => findings[action] > 0).map((action) => ({
    action,
    label:
      action === "fix"
        ? t("plan.findings.n-fix", { count: findings.fix })
        : action === "confirm"
          ? t("plan.findings.n-confirm", { count: findings.confirm })
          : t("plan.findings.n-advice", { count: findings.advice }),
  }));

// A held reviewer as a line of what the rollout waits on.
export const reviewGateLabel = (t: T, row: AutomatedReviewRow): string => {
  const reviewer = reviewerLabel(t, row.reviewer);
  switch (row.conclusion) {
    case "open":
      return t("plan.deploy-waiting.reviewer-findings", {
        reviewer,
        findings: openActionParts(t, row.findings)
          .map((part) => part.label)
          .join(t("plan.findings.separator")),
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
// about its findings: "Standard rules: 2 accepted, 1 resolved without a change".
export const reviewSummaryPart = (t: T, row: AutomatedReviewRow): string => {
  const reviewer = reviewerLabel(t, row.reviewer);
  const { findings } = row;
  let status: string;
  switch (row.conclusion) {
    case "open":
      status = openActionParts(t, findings)
        .map((part) => part.label)
        .join(t("plan.findings.separator"));
      break;
    case "resolved": {
      const accepted = findings.accepted.reduce(
        (sum, group) => sum + group.count,
        0
      );
      status = [
        accepted > 0 ? t("plan.findings.n-accepted", { count: accepted }) : "",
        findings.overridden > 0
          ? t("plan.findings.n-overridden", { count: findings.overridden })
          : "",
        findings.dismissed > 0
          ? t("plan.findings.n-dismissed", { count: findings.dismissed })
          : "",
      ]
        .filter(Boolean)
        .join(t("plan.findings.separator"));
      break;
    }
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
