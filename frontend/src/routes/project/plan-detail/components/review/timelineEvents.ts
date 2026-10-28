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

// A run posts its findings in one batch, a microsecond apart.
const RUN_BATCH_GAP_MS = 1000;

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
  // only each reviewer's latest run is open.
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
  const runs = new Map<
    ReviewRun_Type,
    {
      at: number;
      latest: boolean;
      source: Extract<TimelineSource, { type: "review-run" }>;
    }
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
    if (input.keepFindingCard && review && isThreadRoot(comment)) {
      // A completed run supersedes every earlier result of its reviewer, so
      // the results still current are the latest run's, and a run's results
      // are never split between the two.
      const latest = !review.superseded;
      const time = getTimeForPbTimestampProtoEs(comment.createTime, 0);
      let run = runs.get(review.runType);
      if (run && run.latest === latest && time - run.at <= RUN_BATCH_GAP_MS) {
        run.source.findings.push(comment);
        run.at = time;
      } else {
        const id = `review-run:${comment.name}`;
        run = {
          at: time,
          latest,
          source: {
            type: "review-run",
            runType: review.runType,
            time: comment.createTime,
            findings: [comment],
            open: input.runToggles?.get(id) ?? latest,
          },
        };
        runs.set(review.runType, run);
        entries.push({ id, source: run.source, pinned: run.source.open });
      }
      if (!run.source.open && !input.keepFindingCard(comment)) continue;
      entries.push({
        id: comment.name,
        source: { type: "comment", comment },
        pinned: run.source.open,
      });
      continue;
    }
    entries.push({ id: comment.name, source: { type: "comment", comment } });
  }
  return entries;
}
