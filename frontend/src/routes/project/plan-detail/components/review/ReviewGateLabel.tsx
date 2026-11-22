import { Trans, useTranslation } from "react-i18next";
import { levelPriority } from "@/components/sql-review/LevelPriorityBadge";
import type { AutomatedReviewRow } from "../../utils/planGates";
import { isPlanCheckRow, reviewerLabel } from "../../utils/reviewLabels";
import { PriorityCounts } from "./PriorityCounts";

// A reviewer's open findings: the review's by priority, a plan check's as a
// count of its errors, which show as P0.
function OpenFindings({ row }: { row: AutomatedReviewRow }) {
  const { t } = useTranslation();
  return isPlanCheckRow(row) ? (
    t("plan.findings.n-priority", {
      count: row.findings.open,
      priority: levelPriority("error"),
    })
  ) : (
    <PriorityCounts findings={row.findings} />
  );
}

// A held reviewer as a line of what the rollout waits on.
export function ReviewGateLabel({ row }: { row: AutomatedReviewRow }) {
  const { t } = useTranslation();
  const reviewer = reviewerLabel(t, row.reviewer);
  switch (row.conclusion) {
    case "open":
      return (
        <Trans
          components={{ findings: <OpenFindings row={row} /> }}
          i18nKey="plan.deploy-waiting.reviewer-findings"
          t={t}
          values={{ reviewer }}
        />
      );
    case "failed":
      return t("plan.deploy-waiting.reviewer-failed", { reviewer });
    case "outdated":
      return t("plan.deploy-waiting.reviewer-outdated", { reviewer });
    case "not-run":
      return t("plan.deploy-waiting.reviewer-not-run", { reviewer });
    default:
      return t("plan.deploy-waiting.reviewer-running", { reviewer });
  }
}
