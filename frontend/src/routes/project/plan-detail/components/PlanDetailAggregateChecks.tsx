import { useMemo } from "react";
import { PlanCheckSection } from "@/components/plan-check/PlanCheckSection";
import { getPlanCheckSummaryWithFallback } from "@/lib/plan/check";
import { usePlanCheckActions } from "../hooks/usePlanCheckActions";
import { useSqlReviewV2 } from "../hooks/useSqlReviewV2";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { PlanDetailRuleReviewSummary } from "./PlanDetailRuleReviewSummary";
import { PlanTargetDisplay } from "./PlanTargetDisplay";

// Plan-wide checks rendered as a verdict-line footer at the bottom of the
// CHANGES phase card, so reviewers can scan aggregate check status across
// every spec without clicking into each tab. Drill-down details still live
// in the PlanCheckSection drawer. Create mode renders nothing here — there
// is no plan yet to aggregate against; per-spec PlanDetailDraftChecks
// covers that case. Under SQL Review V2 the standard rule review replaces the
// checks' SQL review, so its status joins the verdict line.
export function PlanDetailAggregateChecks() {
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, refreshChecks, runChecks } =
    usePlanCheckActions();
  const ruleReview = useSqlReviewV2()?.ruleReview;
  const summary = useMemo(
    () =>
      getPlanCheckSummaryWithFallback(
        page.planCheckRuns,
        page.plan.planCheckRunStatusCount
      ),
    [page.plan.planCheckRunStatusCount, page.planCheckRuns]
  );

  if (summary.total === 0 && !allowRunChecks) {
    return null;
  }

  // Running the checks also starts the rule review on the server.
  const runAllChecks = async () => {
    await runChecks();
    await ruleReview?.reload();
  };

  return (
    <div className="border-t px-4 py-4">
      <PlanCheckSection
        canRun={allowRunChecks}
        includeRunFailure
        isRunning={isRunningChecks}
        onRefreshOnOpen={refreshChecks}
        onRun={runAllChecks}
        planCheckRuns={page.planCheckRuns}
        renderTarget={(target) => (
          <PlanTargetDisplay showEnvironment target={target} />
        )}
        summaryOverride={summary}
        trailingSummary={
          ruleReview && <PlanDetailRuleReviewSummary state={ruleReview} />
        }
      />
    </div>
  );
}
