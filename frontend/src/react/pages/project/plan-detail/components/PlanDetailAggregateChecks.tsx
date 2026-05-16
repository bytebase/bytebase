import { useMemo } from "react";
import { PlanCheckSection } from "@/react/components/plan-check/PlanCheckSection";
import { usePlanCheckActions } from "../hooks/usePlanCheckActions";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { getPlanCheckSummaryWithFallback } from "../utils/planCheck";

// Plan-wide checks rendered as a verdict-line footer at the bottom of the
// CHANGES phase card, so reviewers can scan aggregate check status across
// every spec without clicking into each tab. Drill-down details still live
// in the PlanCheckSection drawer. Create mode renders nothing here — there
// is no plan yet to aggregate against; per-spec PlanDetailDraftChecks
// covers that case.
export function PlanDetailAggregateChecks() {
  const page = usePlanDetailContext();
  const { allowRunChecks, isRunningChecks, refreshChecks, runChecks } =
    usePlanCheckActions();
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

  return (
    <div className="border-t px-4 py-4">
      <PlanCheckSection
        canRun={allowRunChecks}
        includeRunFailure
        isRunning={isRunningChecks}
        onRefreshOnOpen={refreshChecks}
        onRun={runChecks}
        planCheckRuns={page.planCheckRuns}
        summaryOverride={summary}
      />
    </div>
  );
}
