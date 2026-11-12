import { createContext, type ReactNode, useContext, useMemo } from "react";
import { useProjectByName } from "@/hooks/useProjectByName";
import type { PlanCheckSummary } from "@/lib/plan/check";
import {
  type ReviewRun,
  ReviewRun_State,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { sqlReviewV2Enabled } from "@/utils/featureGates";
import {
  useIssueThreadsLoaded,
  useUnresolvedThreadTotal,
} from "../components/threads/useUnresolvedThreadCounts";
import { PROJECT_NAME_PREFIX } from "../shell/constants";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  type EffectiveStandardRules,
  useEffectiveStandardRules,
} from "./useEffectiveStandardRules";
import { type ReviewRunState, useReviewRun } from "./useReviewRun";

// What SQL Review V2 changes on the plan page: the review is one of the
// checks, and every open thread holds the automatic rollout, as the server's
// rollout creator does.
export type SqlReviewV2State = {
  unresolvedThreads: number;
  // False until the issue's threads have loaded; `unresolvedThreads` reads
  // zero until then.
  threadsLoaded: boolean;
  // Undefined without permission to read the review run.
  review?: ReviewRunState;
  // The standard rules the review applies to the project; undefined until
  // known. An empty list means the rules are switched off.
  standardRules?: EffectiveStandardRules;
};

const SqlReviewV2Context = createContext<SqlReviewV2State | undefined>(
  undefined
);

// Provides one review run poll for the whole page. The value is undefined
// where SQL Review V2 is off or before the plan has an issue.
export function SqlReviewV2Provider({ children }: { children: ReactNode }) {
  const page = usePlanDetailContext();
  const projectName = `${PROJECT_NAME_PREFIX}${page.projectId}`;
  const project = useProjectByName(projectName);
  const enabled = sqlReviewV2Enabled() && !!page.issue;
  const canReadReview =
    enabled && hasProjectPermissionV2(project, "bb.reviewRuns.get");
  const updateTime = page.plan.updateTime;
  // The server starts the review whenever the plan's SQL changes.
  const review = useReviewRun({
    enabled: canReadReview,
    issueName: page.issue?.name,
    reloadKey: `${updateTime?.seconds ?? ""}:${updateTime?.nanos ?? ""}`,
  });
  const standardRules = useEffectiveStandardRules(projectName, enabled);
  const unresolvedThreads = useUnresolvedThreadTotal(page.issue?.name);
  const threadsLoaded = useIssueThreadsLoaded(page.issue?.name);
  const value = useMemo(
    () =>
      enabled
        ? {
            unresolvedThreads,
            threadsLoaded,
            review: canReadReview ? review : undefined,
            standardRules,
          }
        : undefined,
    [
      canReadReview,
      enabled,
      review,
      standardRules,
      threadsLoaded,
      unresolvedThreads,
    ]
  );
  return (
    <SqlReviewV2Context.Provider value={value}>
      {children}
    </SqlReviewV2Context.Provider>
  );
}

export const useSqlReviewV2 = (): SqlReviewV2State | undefined =>
  useContext(SqlReviewV2Context);

// The plan checks with the review run counted among them: a queued or running
// review is a running check, and a failed one a failed check.
export function withReviewRun(
  checks: PlanCheckSummary,
  run: ReviewRun | null | undefined
): PlanCheckSummary {
  switch (run?.state) {
    case ReviewRun_State.RUNNING:
      return {
        ...checks,
        total: checks.total + 1,
        running: checks.running + 1,
      };
    case ReviewRun_State.FAILED:
      return { ...checks, total: checks.total + 1, error: checks.error + 1 };
    default:
      return checks;
  }
}
