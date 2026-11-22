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
import { useAIReviewOn } from "./useAIReviewOn";
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

// What holds the automatic rollout besides approval and the plan checks: the
// issue's open threads, and under SQL Review V2 the review's run of the
// current SQL. `reviewHolds` says whether the run holds; `review` is
// undefined beside V1's plan checks, and without permission to read it.
export type ReviewHold = Pick<
  SqlReviewV2State,
  "unresolvedThreads" | "threadsLoaded" | "review"
> & { reviewHolds: boolean };

type PlanReviewContextValue = {
  // Undefined where SQL Review V2 is off or before the plan has an issue.
  v2?: SqlReviewV2State;
  // Undefined before the plan has an issue.
  hold?: ReviewHold;
  // The issue's review run wherever the page shows it: under SQL Review V2,
  // and beside V1's plan checks, where the review is the AI review alone.
  // Undefined elsewhere and without permission to read it.
  review?: ReviewRunState;
};

const PlanReviewContext = createContext<PlanReviewContextValue>({});

// Provides one review run poll for the whole page. Beside V1's plan checks the
// run is the AI review's, from the draft onward as under SQL Review V2.
export function PlanReviewProvider({ children }: { children: ReactNode }) {
  const page = usePlanDetailContext();
  const projectName = `${PROJECT_NAME_PREFIX}${page.projectId}`;
  const project = useProjectByName(projectName);
  const aiReviewOn = useAIReviewOn();
  const v2 = sqlReviewV2Enabled();
  const enabled = v2 && !!page.issue;
  const aiBesideV1 = !v2 && aiReviewOn && !!page.issue;
  const canReadReview =
    (enabled || aiBesideV1) &&
    hasProjectPermissionV2(project, "bb.reviewRuns.get");
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
    () => ({
      v2: enabled
        ? {
            unresolvedThreads,
            threadsLoaded,
            review: canReadReview ? review : undefined,
            standardRules,
          }
        : undefined,
      review: canReadReview ? review : undefined,
      hold: enabled
        ? {
            unresolvedThreads,
            threadsLoaded,
            review: canReadReview ? review : undefined,
            reviewHolds: true,
          }
        : page.issue
          ? { unresolvedThreads, threadsLoaded, reviewHolds: false }
          : undefined,
    }),
    [
      canReadReview,
      enabled,
      page.issue,
      review,
      standardRules,
      threadsLoaded,
      unresolvedThreads,
    ]
  );
  return (
    <PlanReviewContext.Provider value={value}>
      {children}
    </PlanReviewContext.Provider>
  );
}

export const useSqlReviewV2 = (): SqlReviewV2State | undefined =>
  useContext(PlanReviewContext).v2;

export const usePlanReviewRun = (): ReviewRunState | undefined =>
  useContext(PlanReviewContext).review;

export const useReviewHold = (): ReviewHold | undefined =>
  useContext(PlanReviewContext).hold;

// Whether the plan checks show among the reviewers in the Review section
// instead of under the changes: under SQL Review V2, and beside the AI review
// once V1's page reads its run.
export const useChecksInReview = (): boolean =>
  sqlReviewV2Enabled() || usePlanReviewRun() !== undefined;

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
