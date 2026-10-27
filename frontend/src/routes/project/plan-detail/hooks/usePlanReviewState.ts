import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { deriveSteps } from "../components/review/ReviewApprovalFlow";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  type AutomatedReviewRow,
  aiReviewRow,
  computePlanGates,
  countFindings,
  countOpenDiscussions,
  type PlanGate,
  planCheckRow,
  ruleReviewRow,
} from "../utils/planGates";
import { useAIReviewRun } from "./useAIReviewRun";
import { useSqlReviewV2 } from "./useSqlReviewV2";

const NO_COMMENTS: IssueComment[] = [];

export type PlanReviewState = {
  rows: AutomatedReviewRow[];
  gates: PlanGate[];
  openDiscussions: number;
  ai: ReturnType<typeof useAIReviewRun>;
};

// The automated reviewers' rows and the rollout's gates under SQL Review V2;
// undefined where it is off or before the plan has an issue.
export function usePlanReviewState(): PlanReviewState | undefined {
  const page = usePlanDetailContext();
  const sqlReview = useSqlReviewV2();
  const ai = useAIReviewRun();
  const issue = page.issue;
  const comments = useAppStore((state) =>
    issue ? state.getIssueComments(issue.name) : NO_COMMENTS
  );
  const ruleReview = sqlReview?.ruleReview;
  const currentApprovalRole = useMemo(
    () =>
      issue
        ? deriveSteps(issue).find((step) => step.status === "current")?.role
        : undefined,
    [issue]
  );

  return useMemo(() => {
    if (!sqlReview || !issue) return undefined;
    const rows: AutomatedReviewRow[] = [];
    if (ruleReview) {
      rows.push(
        ruleReviewRow({
          run: ruleReview.run,
          stale: ruleReview.stale,
          loadFailed: ruleReview.loadFailed,
          findings: countFindings(comments, "rule"),
        })
      );
    }
    const aiFindings = countFindings(comments, "ai");
    if (ai.shown || aiFindings.open + aiFindings.resolved > 0) {
      rows.push(aiReviewRow({ run: ai.run, findings: aiFindings }));
    }
    for (const reviewer of ["policy", "ghost"] as const) {
      const row = planCheckRow(reviewer, page.planCheckRuns);
      if (row) rows.push(row);
    }
    const openDiscussions = countOpenDiscussions(comments);
    return {
      rows,
      gates: computePlanGates({
        issue,
        currentApprovalRole,
        rows,
        openDiscussions,
      }),
      openDiscussions,
      ai,
    };
  }, [
    ai,
    comments,
    currentApprovalRole,
    issue,
    page.planCheckRuns,
    ruleReview,
    sqlReview,
  ]);
}
