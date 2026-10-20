import { createContext, type ReactNode, useContext, useMemo } from "react";
import { useProjectByName } from "@/hooks/useProjectByName";
import type { PlanCheckSummary } from "@/lib/plan/check";
import {
  type ReviewRun,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { sqlReviewV2Enabled } from "@/utils/featureGates";
import { useUnresolvedThreadTotal } from "../components/threads/useUnresolvedThreadCounts";
import { PROJECT_NAME_PREFIX } from "../shell/constants";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { type RuleReviewRunState, useRuleReviewRun } from "./useRuleReviewRun";

// What SQL Review V2 changes on the plan page: the standard rule review is one
// of the checks, and every open thread holds the automatic rollout, as the
// server's rollout creator does.
export type SqlReviewV2State = {
  unresolvedThreads: number;
  // Undefined without permission to read the review run.
  ruleReview?: RuleReviewRunState;
};

const SqlReviewV2Context = createContext<SqlReviewV2State | undefined>(
  undefined
);

// Provides one rule review poll for the whole page. The value is undefined
// where SQL Review V2 is off or before the plan has an issue.
export function SqlReviewV2Provider({ children }: { children: ReactNode }) {
  const page = usePlanDetailContext();
  const project = useProjectByName(`${PROJECT_NAME_PREFIX}${page.projectId}`);
  const enabled = sqlReviewV2Enabled() && !!page.issue;
  const canReadReview =
    enabled && hasProjectPermissionV2(project, "bb.reviewRuns.get");
  const updateTime = page.plan.updateTime;
  // The server starts the rule review whenever the plan's SQL changes.
  const ruleReview = useRuleReviewRun({
    enabled: canReadReview,
    issueName: page.issue?.name,
    reloadKey: `${updateTime?.seconds ?? ""}:${updateTime?.nanos ?? ""}`,
  });
  const unresolvedThreads = useUnresolvedThreadTotal(page.issue?.name);
  const value = useMemo(
    () =>
      enabled
        ? {
            unresolvedThreads,
            ruleReview: canReadReview ? ruleReview : undefined,
          }
        : undefined,
    [canReadReview, enabled, ruleReview, unresolvedThreads]
  );
  return (
    <SqlReviewV2Context.Provider value={value}>
      {children}
    </SqlReviewV2Context.Provider>
  );
}

export const useSqlReviewV2 = (): SqlReviewV2State | undefined =>
  useContext(SqlReviewV2Context);

// The plan checks with the rule review counted among them: a queued or running
// review is a running check, and a failed one a failed check.
export function withRuleReview(
  checks: PlanCheckSummary,
  run: ReviewRun | null | undefined
): PlanCheckSummary {
  switch (run?.status) {
    case ReviewRun_Status.AVAILABLE:
    case ReviewRun_Status.RUNNING:
      return {
        ...checks,
        total: checks.total + 1,
        running: checks.running + 1,
      };
    case ReviewRun_Status.FAILED:
      return { ...checks, total: checks.total + 1, error: checks.error + 1 };
    default:
      return checks;
  }
}
