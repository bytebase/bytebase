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
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

export type AutomatedReviewer = "rule" | "ai" | "policy" | "ghost";

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

// What a finding asks of people, from the priority its reviewer gave it: P0
// says the change is wrong and must change, P1 that a person must accept a
// real risk, P2 is advice.
export type FindingAction = "fix" | "confirm" | "advice";

// Where a finding stands. Open findings by what they ask; the latest run's
// resolved findings by what resolving them meant: a P0 resolved without a
// change is overridden, a P1 accepted, a P2 dismissed. A run supersedes and
// resolves everything its reviewer posted before, so a resolved finding of
// the latest run was resolved by a person on the current SQL.
export type FindingSet =
  | FindingAction
  | "overridden"
  | "accepted"
  | "dismissed"
  | "earlier";

// The accepted findings of one rule, or for the AI review, of one title.
export interface AcceptedGroup {
  key: string;
  ruleType: ReviewRuleType;
  title: string;
  count: number;
}

export interface ReviewerFindings {
  open: number;
  fix: number;
  confirm: number;
  advice: number;
  // The latest run's findings people resolved.
  resolved: number;
  overridden: number;
  accepted: AcceptedGroup[];
  dismissed: number;
  // Resolved findings of earlier runs.
  earlier: number;
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

export const findingAction = (
  priority: IssueComment_ReviewMetadata_Priority
): FindingAction =>
  priority === IssueComment_ReviewMetadata_Priority.P0
    ? "fix"
    : priority === IssueComment_ReviewMetadata_Priority.P2
      ? "advice"
      : "confirm";

const RESOLVED_SET: Record<FindingAction, FindingSet> = {
  fix: "overridden",
  confirm: "accepted",
  advice: "dismissed",
};

// The set a reviewer's finding belongs to; undefined for anything else.
export function findingSetOf(comment: IssueComment): FindingSet | undefined {
  const review = comment.reviewMetadata;
  if (!review || !isThreadRoot(comment)) return undefined;
  const action = findingAction(review.priority);
  if (comment.threadState === IssueComment_ThreadState.OPEN) return action;
  return review.superseded ? "earlier" : RESOLVED_SET[action];
}

// An AI finding names its rule in its text; its first line is the title.
export const findingTitle = (comment: IssueComment): string =>
  comment.comment
    .split("\n", 1)[0]
    .replace(/^#+\s*/, "")
    .trim();

export const emptyFindings = (): ReviewerFindings => ({
  open: 0,
  fix: 0,
  confirm: 0,
  advice: 0,
  resolved: 0,
  overridden: 0,
  accepted: [],
  dismissed: 0,
  earlier: 0,
});

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
  const accepted = new Map<string, AcceptedGroup>();
  for (const comment of comments) {
    if (comment.reviewMetadata?.runType !== runType) continue;
    const set = findingSetOf(comment);
    switch (set) {
      case "fix":
      case "confirm":
      case "advice":
        counts.open++;
        counts[set]++;
        break;
      case "overridden":
      case "dismissed":
        counts.resolved++;
        counts[set]++;
        break;
      case "accepted": {
        counts.resolved++;
        const ruleType =
          comment.reviewMetadata?.ruleType ??
          ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED;
        const byRule = ruleType !== ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED;
        const title = byRule ? "" : findingTitle(comment);
        const key = byRule ? `rule:${ruleType}` : `title:${title}`;
        const group = accepted.get(key);
        if (group) group.count++;
        else accepted.set(key, { key, ruleType, title, count: 1 });
        break;
      }
      case "earlier":
        counts.earlier++;
        break;
    }
  }
  counts.accepted = [...accepted.values()].sort(
    (a, b) => b.count - a.count || a.key.localeCompare(b.key)
  );
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
  findings.open > 0 ? "open" : findings.resolved > 0 ? "resolved" : "passed";

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

// The AI review holds the rollout only through the findings it posts.
export function aiReviewRow(input: {
  run: ReviewRun | null | undefined;
  findings: ReviewerFindings;
}): AutomatedReviewRow {
  const { findings, run } = input;
  const conclusion =
    run === undefined
      ? "loading"
      : run === null
        ? findings.open + findings.resolved > 0
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

// A plan check type's row: the V1 SQL review on engines the standard rules
// do not cover, or the gh-ost dry run. Undefined when the plan's checks have
// no result of that type.
export function planCheckRow(
  reviewer: "policy" | "ghost",
  runs: readonly PlanCheckRun[]
): AutomatedReviewRow | undefined {
  const type =
    reviewer === "policy"
      ? PlanCheckRun_Result_Type.STATEMENT_ADVISE
      : PlanCheckRun_Result_Type.GHOST_SYNC;
  const results = runs.flatMap((run) =>
    run.results.filter((result) => result.type === type)
  );
  const running = runs.some(
    (run) => run.status === PlanCheckRun_Status.RUNNING
  );
  if (results.length === 0 && !running) return undefined;
  const errors = results.filter(
    (result) => result.status === Advice_Level.ERROR
  ).length;
  const warnings = results.filter(
    (result) => result.status === Advice_Level.WARNING
  ).length;
  const failedRun = runs.find(
    (run) => run.status === PlanCheckRun_Status.FAILED
  );
  const conclusion: ReviewConclusion = running
    ? "running"
    : failedRun
      ? "failed"
      : errors > 0
        ? "open"
        : "passed";
  return {
    reviewer,
    conclusion,
    findings: { ...emptyFindings(), open: errors, fix: errors },
    holdsRollout: conclusion !== "passed",
    error: failedRun?.error || undefined,
    time: runs[0]?.createTime,
    targets: new Set(results.map((result) => result.target)).size,
    warnings,
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
