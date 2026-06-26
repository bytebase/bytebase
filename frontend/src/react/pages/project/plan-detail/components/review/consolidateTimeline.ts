// Consolidates runs of consecutive "similar" timeline entries (spec: BYT-9756).
// Repetitive system events — per-toggle label edits, repeated description/title
// edits — each land as their own comment and clutter the feed. A maximal run of
// >=2 consecutive entries that share a consolidation key collapses into a single
// representative row, tagged with the run length so it can show a "N similar
// activities" badge.
//
// To extend consolidation, add a ConsolidationKind to KINDS — the badge / fold /
// render pipeline downstream is kind-agnostic. A kind supplies:
//   - key:   groups adjacent entries (same actor + same repetitive action).
//   - merge: builds the one representative row from the run; defaults to keeping
//            the latest entry as-is, which is faithful for content-free rows
//            (labels, description) where collapsing loses nothing. Value-bearing
//            rows (a title rename shows old->new) need a real merge so the badge
//            summarizes the net change rather than just the last hop.
import { clone } from "@bufbuild/protobuf";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import {
  type IssueComment,
  IssueCommentSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { TimelineEntry } from "./timelineEvents";

interface ConsolidationKind {
  key: (entry: TimelineEntry) => string | null;
  merge?: (run: TimelineEntry[]) => TimelineEntry | null;
}

// A single issue-update comment edits exactly one field (the backend emits one
// comment per changed field), so classify it by which from/to pair is set —
// mirroring the renderer's precedence (title > description > status > labels).
type IssueField = "title" | "description" | "status" | "labels";

function classifyIssueUpdate(
  entry: TimelineEntry
): { field: IssueField; comment: IssueComment } | null {
  if (entry.source.type !== "comment") return null;
  const comment = entry.source.comment;
  if (getIssueCommentType(comment) !== IssueCommentType.ISSUE_UPDATE) {
    return null;
  }
  if (comment.event.case !== "issueUpdate") return null;
  const e = comment.event.value;
  if (e.fromTitle !== undefined || e.toTitle !== undefined) {
    return { field: "title", comment };
  }
  if (e.fromDescription !== undefined || e.toDescription !== undefined) {
    return { field: "description", comment };
  }
  if (e.fromStatus !== undefined || e.toStatus !== undefined) {
    return { field: "status", comment };
  }
  if (e.fromLabels.length > 0 || e.toLabels.length > 0) {
    return { field: "labels", comment };
  }
  return null;
}

// Labels and description render content-free sentences, so the latest row plus a
// count badge is a faithful summary. Keyed by field so the two never blend into
// a single ambiguous row.
const contentFreeFieldKind: ConsolidationKind = {
  key: (entry) => {
    const c = classifyIssueUpdate(entry);
    if (!c || (c.field !== "labels" && c.field !== "description")) return null;
    return `${c.field}:${c.comment.creator}`;
  },
};

// Title renames carry old/new values, so a plain "latest row" would show only
// the last hop (Y->Z) while claiming to stand for the whole run. Merge to the
// net change (first.from -> last.to); drop a run that nets to no change.
const titleRenameKind: ConsolidationKind = {
  key: (entry) => {
    const c = classifyIssueUpdate(entry);
    return c?.field === "title" ? `title:${c.comment.creator}` : null;
  },
  merge: (run) => {
    const first = classifyIssueUpdate(run[0]);
    const last = classifyIssueUpdate(run[run.length - 1]);
    if (!first || !last) return run[run.length - 1];
    const fromTitle =
      first.comment.event.case === "issueUpdate"
        ? first.comment.event.value.fromTitle
        : undefined;
    const toTitle =
      last.comment.event.case === "issueUpdate"
        ? last.comment.event.value.toTitle
        : undefined;
    if (fromTitle === toTitle) return null; // net no-op — nothing to show
    const comment = clone(IssueCommentSchema, last.comment);
    if (comment.event.case === "issueUpdate") {
      comment.event.value.fromTitle = fromTitle;
    }
    return { id: run[run.length - 1].id, source: { type: "comment", comment } };
  },
};

const KINDS: ConsolidationKind[] = [contentFreeFieldKind, titleRenameKind];

function classify(
  entry: TimelineEntry
): { key: string; kind: ConsolidationKind } | null {
  for (const kind of KINDS) {
    const key = kind.key(entry);
    if (key !== null) return { key, kind };
  }
  return null;
}

const latestMerge = (run: TimelineEntry[]): TimelineEntry =>
  run[run.length - 1];

export function consolidateConsecutive(
  entries: TimelineEntry[]
): TimelineEntry[] {
  const result: TimelineEntry[] = [];
  let i = 0;
  while (i < entries.length) {
    const classified = classify(entries[i]);
    if (classified === null) {
      result.push(entries[i]);
      i++;
      continue;
    }
    let j = i + 1;
    while (j < entries.length && classify(entries[j])?.key === classified.key) {
      j++;
    }
    const run = entries.slice(i, j);
    if (run.length === 1) {
      result.push(run[0]);
    } else {
      const rep = (classified.kind.merge ?? latestMerge)(run);
      if (rep !== null) {
        result.push({ ...rep, similarCount: run.length });
      }
    }
    i = j;
  }
  return result;
}
