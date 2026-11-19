import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
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
  hasReviewableSQL,
  loadedMatchedDatabaseCount,
} from "../utils/reviewRun";
import { useAIReviewOn } from "./useAIReviewOn";
import { usePlanReviewRun, useSqlReviewV2 } from "./useSqlReviewV2";

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
  // The review is off when the project applies no standard rule and gets no
  // AI review. Rules not known yet count as on.
  const aiReviewOn = useAIReviewOn();
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
    rows.push(...planCheckRows(page.planCheckRuns));
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

// The plan checks' rows: the SQL review advice and the gh-ost dry run by
// their results, and the run itself while it has none to show.
function planCheckRows(
  planCheckRuns: readonly PlanCheckRun[]
): AutomatedReviewRow[] {
  const rows: AutomatedReviewRow[] = [];
  for (const reviewer of ["policy", "ghost"] as const) {
    const row = planCheckRow(reviewer, planCheckRuns);
    if (row) rows.push(row);
  }
  const runRow = planCheckRunRow(planCheckRuns);
  if (runRow) rows.push(runRow);
  return rows;
}

// The AI review and the plan checks beside it on V1's page, once the page
// reads the AI review's run. Its run holds nothing, so there are no gates: its
// findings hold the rollout only as open threads, which V1's footer counts
// with every other thread. Undefined under SQL Review V2, where
// `usePlanReviewState` has the rows.
export function useAIReviewBesideV1():
  | Pick<PlanReviewState, "rows">
  | undefined {
  const page = usePlanDetailContext();
  const sqlReview = useSqlReviewV2();
  const review = usePlanReviewRun();
  const issue = page.issue;
  const comments = useAppStore((state) =>
    issue ? state.getIssueComments(issue.name) : NO_COMMENTS
  );
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
  return useMemo(() => {
    if (sqlReview || !review) return undefined;
    const rows: AutomatedReviewRow[] = [];
    if (reviewApplies(review.run, reviewsSQL)) {
      rows.push(
        reviewRow({
          run: review.run,
          stale: review.stale,
          loadFailed: review.loadFailed,
          findings: countFindings(comments),
          off: false,
        })
      );
    }
    rows.push(...planCheckRows(page.planCheckRuns));
    return { rows };
  }, [comments, page.planCheckRuns, review, reviewsSQL, sqlReview]);
}
