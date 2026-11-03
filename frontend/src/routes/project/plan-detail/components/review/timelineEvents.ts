// Maps the review timeline's event sources into a flat, oldest-first entry
// list (spec: "Activity timeline"). Whether an entry renders as a bordered
// card or a one-line system row is decided at render time by body presence,
// so no weight/tier is stored here. Plan check results are never entries.
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { isThreadReply, isThreadRoot } from "@/stores/app/issueComment";
import type {
  IssueComment,
  IssueComment_ReviewRunCompletion,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";

export type TimelineSource =
  | { type: "plan-created"; creator: string; time?: Timestamp }
  | { type: "ready-for-review"; creator: string; time?: Timestamp }
  | { type: "comment"; comment: IssueComment }
  // One line for a reviewer's completed run, which opens to the findings it
  // posted.
  | {
      type: "review-run";
      completion: IssueComment_ReviewRunCompletion;
      time?: Timestamp;
      findings: IssueComment[];
      open: boolean;
    };

const runKey = (runType: ReviewRun_Type, attempt: bigint) =>
  `${runType}/${attempt}`;

export interface TimelineEntry {
  id: string;
  source: TimelineSource;
  // Set by consolidateConsecutive when this row stands in for a run of N>=2
  // consecutive similar activities (e.g. repeated label edits). Drives the
  // "(N similar activities)" badge. Absent means a standalone row.
  similarCount?: number;
  // The line and cards of an open run, which the long-history fold leaves in
  // place.
  pinned?: boolean;
}

export function buildTimelineEntries(input: {
  planCreator?: string;
  planCreateTime?: Timestamp;
  issueCreator?: string;
  issueCreateTime?: Timestamp;
  issueDraft?: boolean;
  comments: IssueComment[];
  // Under SQL Review V2, findings become one line per completed run. An open
  // run shows every finding's card; a folded one only those this keeps.
  keepFindingCard?: (root: IssueComment) => boolean;
  // Runs someone opened or folded, by entry id. The rest follow the default:
  // only the runs with a finding still current are open.
  runToggles?: ReadonlyMap<string, boolean>;
}): TimelineEntry[] {
  const reviewSubmission = input.comments.find(
    (comment) => comment.event.case === "reviewSubmission"
  );
  const entries: TimelineEntry[] = [];
  if (input.planCreator) {
    entries.push({
      id: "plan-created",
      source: {
        type: "plan-created",
        creator: input.planCreator,
        time: input.planCreateTime,
      },
    });
  }
  if (input.issueCreator && !input.issueDraft && !reviewSubmission) {
    entries.push({
      id: "ready-for-review",
      source: {
        type: "ready-for-review",
        creator: input.issueCreator,
        time: input.issueCreateTime,
      },
    });
  }
  // A run's findings follow its completion. A finding a later run reports
  // again keeps its thread, so stays with the run that posted it.
  const findingsByRun = new Map<string, IssueComment[]>();
  const completedRuns = new Set<string>();
  for (const comment of input.comments) {
    const review = comment.reviewMetadata;
    if (review && isThreadRoot(comment)) {
      const key = runKey(review.runType, review.attempt);
      const findings = findingsByRun.get(key);
      if (findings) findings.push(comment);
      else findingsByRun.set(key, [comment]);
    } else if (comment.event.case === "reviewRunCompletion") {
      const { runType, attempt } = comment.event.value;
      completedRuns.add(runKey(runType, attempt));
    }
  }
  for (const comment of input.comments) {
    // Replies render inside their thread's entry, never as their own row.
    if (isThreadReply(comment)) continue;
    if (comment.event.case === "reviewSubmission") {
      if (comment === reviewSubmission) {
        entries.push({
          id: comment.name,
          source: {
            type: "ready-for-review",
            creator: comment.creator,
            time: comment.createTime,
          },
        });
      }
      continue;
    }
    if (comment.event.case === "reviewRunCompletion") {
      if (!input.keepFindingCard) continue;
      const completion = comment.event.value;
      const findings =
        findingsByRun.get(runKey(completion.runType, completion.attempt)) ?? [];
      const id = `review-run:${comment.name}`;
      const open =
        input.runToggles?.get(id) ??
        findings.some((finding) => !finding.reviewMetadata?.superseded);
      entries.push({
        id,
        source: {
          type: "review-run",
          completion,
          time: comment.createTime,
          findings,
          open,
        },
        pinned: open,
      });
      for (const finding of findings) {
        if (!open && !input.keepFindingCard(finding)) continue;
        entries.push({
          id: finding.name,
          source: { type: "comment", comment: finding },
          pinned: open,
        });
      }
      continue;
    }
    const review = comment.reviewMetadata;
    if (
      input.keepFindingCard &&
      review &&
      completedRuns.has(runKey(review.runType, review.attempt))
    ) {
      continue;
    }
    entries.push({ id: comment.name, source: { type: "comment", comment } });
  }
  return entries;
}
