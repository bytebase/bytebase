// Maps the review timeline's event sources into a flat, oldest-first entry
// list (spec: "Activity timeline"). Whether an entry renders as a bordered
// card or a one-line system row is decided at render time by body presence,
// so no weight/tier is stored here. Plan check results are never entries.
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";

export type TimelineSource =
  | { type: "plan-created"; creator: string; time?: Timestamp }
  | { type: "ready-for-review"; creator: string; time?: Timestamp }
  | { type: "comment"; comment: IssueComment };

export interface TimelineEntry {
  id: string;
  source: TimelineSource;
}

export function buildTimelineEntries(input: {
  planCreator?: string;
  planCreateTime?: Timestamp;
  issueCreator?: string;
  issueCreateTime?: Timestamp;
  comments: IssueComment[];
}): TimelineEntry[] {
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
  if (input.issueCreator) {
    entries.push({
      id: "ready-for-review",
      source: {
        type: "ready-for-review",
        creator: input.issueCreator,
        time: input.issueCreateTime,
      },
    });
  }
  for (const comment of input.comments) {
    entries.push({ id: comment.name, source: { type: "comment", comment } });
  }
  return entries;
}
