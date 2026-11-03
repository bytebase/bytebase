// Maps the review timeline's event sources into a flat, oldest-first entry
// list (spec: "Activity timeline"). Whether an entry renders as a bordered
// card or a one-line system row is decided at render time by body presence,
// so no weight/tier is stored here. Plan check results are never entries.
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { isThreadReply, isThreadRoot } from "@/stores/app/issueComment";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type {
  IssueComment,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";

export type TimelineSource =
  | { type: "plan-created"; creator: string; time?: Timestamp }
  | { type: "ready-for-review"; creator: string; time?: Timestamp }
  | { type: "comment"; comment: IssueComment }
  // One line for a reviewer's run, which opens to the findings it posted.
  | {
      type: "review-run";
      runType: ReviewRun_Type;
      time?: Timestamp;
      findings: IssueComment[];
      open: boolean;
    };

// A run posts its findings in one batch, a microsecond apart; two runs of
// one reviewer complete milliseconds apart at the least.
const RUN_BATCH_GAP_MS = 1;

// The run that posted a finding. A run that reports an earlier finding again
// keeps its thread instead of posting it, so one run's findings can be both
// current and superseded; the run stays current while any of them is.
interface ReviewRunBatch {
  id: string;
  current: boolean;
}

// Each review result's run, by the result's name.
function reviewRunBatches(
  comments: readonly IssueComment[]
): Map<string, ReviewRunBatch> {
  const batches = new Map<string, ReviewRunBatch>();
  const last = new Map<ReviewRun_Type, { at: number; batch: ReviewRunBatch }>();
  for (const comment of comments) {
    const review = comment.reviewMetadata;
    if (!review || !isThreadRoot(comment)) continue;
    const at = getTimeForPbTimestampProtoEs(comment.createTime, 0);
    const previous = last.get(review.runType);
    const batch =
      previous && at - previous.at <= RUN_BATCH_GAP_MS
        ? previous.batch
        : { id: `review-run:${comment.name}`, current: false };
    batch.current ||= !review.superseded;
    last.set(review.runType, { at, batch });
    batches.set(comment.name, batch);
  }
  return batches;
}

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
  // Under SQL Review V2, findings become one line per run. An open run shows
  // every finding's card; a folded one only those this keeps.
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
  const batches = reviewRunBatches(input.comments);
  const runs = new Map<
    ReviewRunBatch,
    Extract<TimelineSource, { type: "review-run" }>
  >();
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
    const review = comment.reviewMetadata;
    const batch = batches.get(comment.name);
    if (input.keepFindingCard && review && batch) {
      let run = runs.get(batch);
      if (!run) {
        run = {
          type: "review-run",
          runType: review.runType,
          time: comment.createTime,
          findings: [],
          open: input.runToggles?.get(batch.id) ?? batch.current,
        };
        runs.set(batch, run);
        entries.push({ id: batch.id, source: run, pinned: run.open });
      }
      run.findings.push(comment);
      if (!run.open && !input.keepFindingCard(comment)) continue;
      entries.push({
        id: comment.name,
        source: { type: "comment", comment },
        pinned: run.open,
      });
      continue;
    }
    entries.push({ id: comment.name, source: { type: "comment", comment } });
  }
  return entries;
}
