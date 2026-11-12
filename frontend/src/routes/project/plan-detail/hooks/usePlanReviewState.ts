import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { deriveSteps } from "../components/review/ReviewApprovalFlow";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  type AutomatedReviewRow,
  computePlanGates,
  countFindings,
  countOpenDiscussions,
  findingsRow,
  type PlanGate,
  planCheckRow,
  planCheckRunRow,
  resolvedByPeople,
  reviewApplies,
  reviewRow,
  statementReportFails,
} from "../utils/planGates";
import {
  aiReviewEnabled,
  hasReviewableSQL,
  loadedMatchedDatabaseCount,
} from "../utils/reviewRun";
import { useSqlReviewV2 } from "./useSqlReviewV2";

const NO_COMMENTS: IssueComment[] = [];

export type PlanReviewState = {
  rows: AutomatedReviewRow[];
  gates: PlanGate[];
  openDiscussions: number;
};

// The automated reviewers' rows and the rollout's gates under SQL Review V2;
// undefined where it is off or before the plan has an issue.
export function usePlanReviewState(): PlanReviewState | undefined {
  const page = usePlanDetailContext();
  const sqlReview = useSqlReviewV2();
  const issue = page.issue;
  const comments = useAppStore((state) =>
    issue ? state.getIssueComments(issue.name) : NO_COMMENTS
  );
  const review = sqlReview?.review;
  // The review is off when the project applies no standard rule and the
  // workspace has no AI review. Rules not known yet count as on.
  const aiReviewOn = useAppStore((state) => aiReviewEnabled(state.serverInfo));
  const off = sqlReview?.standardRules?.rules.length === 0 && !aiReviewOn;
  const dbGroupsByName = useAppStore((state) => state.dbGroupsByName);
  const dbGroupViewByName = useAppStore((state) => state.dbGroupViewByName);
  const reviewsSQL = useMemo(
    () =>
      hasReviewableSQL(
        page.plan,
        loadedMatchedDatabaseCount({ dbGroupsByName, dbGroupViewByName })
      ),
    [dbGroupViewByName, dbGroupsByName, page.plan]
  );
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
    const findings = countFindings(comments);
    if (review) {
      if (reviewApplies(review.run, reviewsSQL)) {
        rows.push(
          reviewRow({
            run: review.run,
            stale: review.stale,
            loadFailed: review.loadFailed,
            findings,
            off,
          })
        );
      }
    } else if (findings.open + resolvedByPeople(findings) > 0) {
      rows.push(findingsRow(findings));
    }
    for (const reviewer of ["policy", "ghost"] as const) {
      const row = planCheckRow(reviewer, page.planCheckRuns);
      if (row) rows.push(row);
    }
    const runRow = planCheckRunRow(page.planCheckRuns);
    if (runRow) rows.push(runRow);
    const openDiscussions = countOpenDiscussions(comments);
    return {
      rows,
      gates: computePlanGates({
        issue,
        currentApprovalRole,
        rows,
        statementError: statementReportFails(page.planCheckRuns),
        openDiscussions,
      }),
      openDiscussions,
    };
  }, [
    comments,
    currentApprovalRole,
    issue,
    off,
    page.planCheckRuns,
    review,
    reviewsSQL,
    sqlReview,
  ]);
}
