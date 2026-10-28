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
  // One line for a reviewer's run, standing in for the findings it posted.
  | {
      type: "review-run";
      runType: ReviewRun_Type;
      time?: Timestamp;
      findings: IssueComment[];
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
}

export function buildTimelineEntries(input: {
  planCreator?: string;
  planCreateTime?: Timestamp;
  issueCreator?: string;
  issueCreateTime?: Timestamp;
  issueDraft?: boolean;
  comments: IssueComment[];
  // Under SQL Review V2, findings become one line per run; a finding keeps
  // its own card only where this says so.
  keepFindingCard?: (root: IssueComment) => boolean;
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
      const time = getTimeForPbTimestampProtoEs(comment.createTime, 0);
      const last = runs.get(review.runType);
      if (last && time - last.at <= RUN_BATCH_GAP_MS) {
        last.source.findings.push(comment);
        last.at = time;
      } else {
        const source = {
          type: "review-run" as const,
          runType: review.runType,
          time: comment.createTime,
          findings: [comment],
        };
        runs.set(review.runType, { at: time, source });
        entries.push({ id: `review-run:${comment.name}`, source });
      }
      if (!input.keepFindingCard(comment)) continue;
    }
    entries.push({ id: comment.name, source: { type: "comment", comment } });
  }
  return entries;
}
