// The automated reviewers' conclusions and the gates the automatic rollout
// waits on, computed once for the Review section, the Deploy waiting list, the
// header, and the collapsed summaries. Mirrors the server's rollout creator:
// approval passed, plan checks passed, the rule review done for the current
// SQL, and no open thread.
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { isThreadRoot } from "@/stores/app/issueComment";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  type Issue,
  type IssueComment,
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ThreadState,
  type ReviewRun,
  ReviewRun_Status,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  type PlanCheckRun,
  PlanCheckRun_Result_Type,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

export type AutomatedReviewer =
  | "rule"
  | "ai"
  | "policy"
  | "ghost"
  | "report"
  | "checks";

export type PlanCheckReviewer = "policy" | "ghost" | "report";

// One conclusion per reviewer. "passed" is a run with no findings;
// "resolved" a run whose findings people have all resolved.
export type ReviewConclusion =
  | "loading"
  | "not-run"
  | "running"
  | "outdated"
  | "failed"
  | "open"
  | "resolved"
  | "passed";

// A finding's priority as its reviewer gave it: P0 says the SQL is wrong and
// must change, P1 that a person must accept a real risk, P2 is advice.
export type FindingPriority = "p0" | "p1" | "p2";

// Where a finding stands: open findings by priority, and the latest run's
// resolved findings, where a P0 resolved without a change stands apart. A
// run supersedes and resolves everything its reviewer posted before, so a
// resolved finding of the latest run was resolved by a person on the
// current SQL.
export type FindingSet =
  | FindingPriority
  | "overridden"
  | "resolved"
  | "earlier";

// What a count or the walker covers: one set, or every open finding.
export type FindingGroup = FindingSet | "open";

// The number of a reviewer's findings in each set.
export interface ReviewerFindings extends Record<FindingSet, number> {
  open: number;
}

export interface AutomatedReviewRow {
  reviewer: AutomatedReviewer;
  conclusion: ReviewConclusion;
  findings: ReviewerFindings;
  holdsRollout: boolean;
  // A failed run's error, or a load failure.
  error?: string;
  // When the run the conclusion comes from finished.
  time?: Timestamp;
  // Plan check rows: databases the check ran on, and warnings that pass.
  targets?: number;
  warnings?: number;
}

export type PlanGate =
  | { kind: "submit" }
  | { kind: "approval"; rejected: boolean; role?: string }
  | { kind: "review"; row: AutomatedReviewRow }
  | { kind: "discussion"; open: number };

export const findingPriority = (
  priority: IssueComment_ReviewMetadata_Priority
): FindingPriority =>
  priority === IssueComment_ReviewMetadata_Priority.P0
    ? "p0"
    : priority === IssueComment_ReviewMetadata_Priority.P2
      ? "p2"
      : "p1";

// The set a reviewer's finding belongs to; undefined for anything else.
export function findingSetOf(comment: IssueComment): FindingSet | undefined {
  const review = comment.reviewMetadata;
  if (!review || !isThreadRoot(comment)) return undefined;
  const priority = findingPriority(review.priority);
  if (comment.threadState === IssueComment_ThreadState.OPEN) return priority;
  if (review.superseded) return "earlier";
  return priority === "p0" ? "overridden" : "resolved";
}

export function inFindingGroup(
  comment: IssueComment,
  group: FindingGroup
): boolean {
  const set = findingSetOf(comment);
  return group === "open"
    ? set === "p0" || set === "p1" || set === "p2"
    : set === group;
}

// An AI finding names its rule in its text; its first line is the title.
export const findingTitle = (comment: IssueComment): string =>
  comment.comment
    .split("\n", 1)[0]
    .replace(/^#+\s*/, "")
    .trim();

export const emptyFindings = (): ReviewerFindings => ({
  open: 0,
  p0: 0,
  p1: 0,
  p2: 0,
  overridden: 0,
  resolved: 0,
  earlier: 0,
});

// The latest run's findings people resolved, whatever their priority: what a
// rerun reopens.
export const resolvedByPeople = (findings: ReviewerFindings): number =>
  findings.resolved + findings.overridden;

const runTypeOf: Record<"rule" | "ai", ReviewRun_Type> = {
  rule: ReviewRun_Type.RULE,
  ai: ReviewRun_Type.AI,
};

// Open findings count whatever run posted them, since any open thread holds
// the rollout; resolved ones count for the latest run only.
export function countFindings(
  comments: readonly IssueComment[],
  reviewer: "rule" | "ai"
): ReviewerFindings {
  const runType = runTypeOf[reviewer];
  const counts = emptyFindings();
  for (const comment of comments) {
    if (comment.reviewMetadata?.runType !== runType) continue;
    const set = findingSetOf(comment);
    if (!set) continue;
    counts[set]++;
    if (set === "p0" || set === "p1" || set === "p2") counts.open++;
  }
  return counts;
}

// Open threads people started, as opposed to findings a reviewer posted.
export function countOpenDiscussions(comments: readonly IssueComment[]) {
  return comments.filter(
    (comment) =>
      isThreadRoot(comment) &&
      !comment.reviewMetadata &&
      comment.threadState === IssueComment_ThreadState.OPEN
  ).length;
}

const findingsConclusion = (findings: ReviewerFindings): ReviewConclusion =>
  findings.open > 0
    ? "open"
    : resolvedByPeople(findings) > 0
      ? "resolved"
      : "passed";

const runConclusion = (
  run: ReviewRun,
  findings: ReviewerFindings
): ReviewConclusion => {
  if (run.outdated) return "outdated";
  switch (run.status) {
    case ReviewRun_Status.AVAILABLE:
    case ReviewRun_Status.RUNNING:
      return "running";
    case ReviewRun_Status.FAILED:
      return "failed";
    case ReviewRun_Status.DONE:
      return findingsConclusion(findings);
    default:
      return "loading";
  }
};

// Every state but a finished current run holds the rollout: the server starts
// a missing or outdated run before it creates the rollout, and waits for it.
export function ruleReviewRow(input: {
  run: ReviewRun | null | undefined;
  stale: boolean;
  loadFailed: boolean;
  findings: ReviewerFindings;
}): AutomatedReviewRow {
  const { findings, run } = input;
  let conclusion: ReviewConclusion;
  if (input.loadFailed) {
    conclusion = "failed";
  } else if (run === undefined) {
    conclusion = "loading";
  } else if (run === null) {
    conclusion = "not-run";
  } else {
    conclusion = runConclusion(run, findings);
    // A completion the page has not re-read since the SQL changed, or whose
    // findings are still being listed, is not a result yet.
    if (input.stale && conclusion !== "outdated" && conclusion !== "failed") {
      conclusion = "running";
    }
  }
  return {
    reviewer: "rule",
    conclusion,
    findings,
    holdsRollout: conclusion !== "passed" && conclusion !== "resolved",
    error: input.loadFailed ? undefined : (run?.error ?? undefined),
    time: run?.endTime,
  };
}

// A plan with no SQL to review never gets a rule review run, and the server
// holds nothing for one; a run that exists still counts, since the server
// restarts a stale run whatever the plan holds.
export const ruleReviewApplies = (
  run: ReviewRun | null | undefined,
  reviewsSQL: boolean
): boolean => run !== null || reviewsSQL;

// The AI review holds the rollout only through the findings it posts. The
// page reads its run only while a review can still start, as before a
// rollout; otherwise the findings alone conclude.
export function aiReviewRow(input: {
  run: ReviewRun | null | undefined;
  runFetched: boolean;
  findings: ReviewerFindings;
}): AutomatedReviewRow {
  const { findings } = input;
  const run = input.runFetched ? input.run : null;
  const conclusion =
    run === undefined
      ? "loading"
      : run === null
        ? findings.open + resolvedByPeople(findings) > 0
          ? findingsConclusion(findings)
          : "not-run"
        : runConclusion(run, findings);
  return {
    reviewer: "ai",
    conclusion,
    findings,
    holdsRollout: findings.open > 0,
    error: run?.error || undefined,
    time: run?.endTime,
  };
}

const PLAN_CHECK_TYPE: Record<PlanCheckReviewer, PlanCheckRun_Result_Type> = {
  policy: PlanCheckRun_Result_Type.STATEMENT_ADVISE,
  ghost: PlanCheckRun_Result_Type.GHOST_SYNC,
  report: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
};

// A plan check type's row, from its results: the V1 SQL review on engines the
// standard rules do not cover, the gh-ost dry run, or the statement report.
// The server creates no rollout while any result is an ERROR. A report that
// passes describes the statement rather than judging it, so it has no row.
// Undefined without a result to show.
export function planCheckRow(
  reviewer: PlanCheckReviewer,
  runs: readonly PlanCheckRun[]
): AutomatedReviewRow | undefined {
  const type = PLAN_CHECK_TYPE[reviewer];
  const results = runs.flatMap((run) =>
    run.results.filter((result) => result.type === type)
  );
  const errors = results.filter(
    (result) => result.status === Advice_Level.ERROR
  ).length;
  const warnings = results.filter(
    (result) => result.status === Advice_Level.WARNING
  ).length;
  if (results.length === 0) return undefined;
  if (reviewer === "report" && errors === 0 && warnings === 0) return undefined;
  const conclusion: ReviewConclusion = errors > 0 ? "open" : "passed";
  return {
    reviewer,
    conclusion,
    findings: { ...emptyFindings(), open: errors, p0: errors },
    holdsRollout: errors > 0,
    time: runs[0]?.createTime,
    targets: new Set(results.map((result) => result.target)).size,
    warnings,
  };
}

// The plan check run while it has no results to show: queued, running, or
// stopped before producing any, as when its targets cannot be derived. The
// server creates the rollout only once the current run is DONE.
export function planCheckRunRow(
  runs: readonly PlanCheckRun[]
): AutomatedReviewRow | undefined {
  const run = runs.find(
    (candidate) =>
      candidate.status !== PlanCheckRun_Status.DONE &&
      candidate.results.length === 0
  );
  if (!run) return undefined;
  const stopped =
    run.status === PlanCheckRun_Status.FAILED ||
    run.status === PlanCheckRun_Status.CANCELED;
  return {
    reviewer: "checks",
    conclusion: stopped ? "failed" : "running",
    findings: emptyFindings(),
    holdsRollout: true,
    error: run.error || undefined,
    time: run.createTime,
  };
}

// A P0 resolved without a change does not hold the rollout, but an approver
// should look at it.
export const rowNeedsAttention = (row: AutomatedReviewRow): boolean =>
  row.conclusion === "open" ||
  row.conclusion === "failed" ||
  row.conclusion === "outdated" ||
  row.conclusion === "not-run" ||
  row.findings.overridden > 0;

const approvalPassed = (
  issue: Pick<Issue, "approvalStatus" | "approvalTemplate">
) =>
  issue.approvalStatus === ApprovalStatus.APPROVED ||
  issue.approvalStatus === ApprovalStatus.SKIPPED ||
  issue.approvalTemplate?.flow?.roles.length === 0;

// What the automatic rollout still waits on, in the order a reader acts on
// it. Empty once every gate passes.
export function computePlanGates(input: {
  issue: Pick<
    Issue,
    "draft" | "status" | "approvalStatus" | "approvalTemplate"
  >;
  currentApprovalRole?: string;
  rows: readonly AutomatedReviewRow[];
  openDiscussions: number;
}): PlanGate[] {
  const { issue } = input;
  if (issue.status !== IssueStatus.OPEN) return [];
  const gates: PlanGate[] = [];
  if (issue.draft) {
    gates.push({ kind: "submit" });
  } else if (!approvalPassed(issue)) {
    gates.push({
      kind: "approval",
      rejected: issue.approvalStatus === ApprovalStatus.REJECTED,
      role: input.currentApprovalRole,
    });
  }
  for (const row of input.rows) {
    if (row.holdsRollout) gates.push({ kind: "review", row });
  }
  if (input.openDiscussions > 0) {
    gates.push({ kind: "discussion", open: input.openDiscussions });
  }
  return gates;
}
