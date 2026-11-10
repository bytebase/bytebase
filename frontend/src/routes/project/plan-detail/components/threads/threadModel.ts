// Pure helpers for comment threads and statement anchors (design doc: Plan
// Review Comment and Thread UI/UX). The store keeps a flat comment list; these
// group it into threads, derive anchor state from the plan, and decide what
// the statement editor shows.
import { create } from "@bufbuild/protobuf";
import type { TFunction } from "i18next";
import { isThreadReply, isThreadRoot } from "@/stores/app/issueComment";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  type IssueComment,
  IssueComment_ThreadState,
  ReviewRun_Type,
  type StatementAnchor,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { sheetNameOfSpec } from "@/utils/v1/issue/plan";
import { extractSheetUID } from "@/utils/v1/sheet";
import { tokenizeLines } from "./placement/lineTokens";
import {
  type LineRange,
  type Placement,
  trivialPlacement,
} from "./placement/place";

export type { LineRange };

export interface CommentThread {
  root: IssueComment;
  // Oldest first.
  replies: IssueComment[];
  resolved: boolean;
}

export type AnchorState = Placement["state"] | "PENDING";

// A thread placed in the statement editor: its root anchor is CURRENT for the
// displayed spec and sheet.
export interface EditorThread {
  thread: CommentThread;
  range: LineRange;
}

const createTimeOf = (comment: IssueComment): number =>
  getTimeForPbTimestampProtoEs(comment.createTime, 0);

export const compareByCreateTime = (a: IssueComment, b: IssueComment): number =>
  createTimeOf(a) - createTimeOf(b) || a.name.localeCompare(b.name);

// Sheet names embed the content hash: projects/{project}/sheets/{sha256}.
// Local drafts use negative ids and never anchor.
export const sheetSha256OfName = (sheetName: string): string | undefined => {
  const uid = extractSheetUID(sheetName).toLowerCase();
  return /^[0-9a-f]{64}$/.test(uid) ? uid : undefined;
};

export const sheetNameOfSha256 = (
  projectName: string,
  sheetSha256: string
): string => `${projectName}/sheets/${sheetSha256}`;

// The hash of the sheet a spec currently points at, the target every anchor
// on that spec is placed against.
export const targetSha256OfSpec = (
  spec: Plan_Spec | undefined
): string | undefined =>
  spec ? sheetSha256OfName(sheetNameOfSpec(spec)) : undefined;

export function groupThreads(comments: IssueComment[]): CommentThread[] {
  const repliesByRoot = new Map<string, IssueComment[]>();
  for (const comment of comments) {
    if (!isThreadReply(comment)) continue;
    const list = repliesByRoot.get(comment.root) ?? [];
    list.push(comment);
    repliesByRoot.set(comment.root, list);
  }
  return comments.filter(isThreadRoot).map((root) => ({
    root,
    replies: (repliesByRoot.get(root.name) ?? []).sort(compareByCreateTime),
    resolved: root.threadState === IssueComment_ThreadState.RESOLVED,
  }));
}

export const countUnresolvedThreads = (threads: CommentThread[]): number =>
  threads.filter((thread) => !thread.resolved).length;

// Unresolved threads placed on each spec's current sheet: the number the
// statement editor shows for that change. Specs without a placed thread
// have no entry.
export function countPlacedUnresolvedBySpec(
  threads: CommentThread[],
  specs: Plan_Spec[],
  placementsFor: (
    specId: string,
    sheetSha256: string
  ) => ReadonlyMap<string, Placement> | undefined,
  previewEndLineFor?: (specId: string) => number | undefined
): ReadonlyMap<string, number> {
  const counts = new Map<string, number>();
  for (const spec of specs) {
    const sheetSha256 = targetSha256OfSpec(spec);
    if (!sheetSha256) continue;
    const placed = selectUnresolvedEditorThreads(
      threads,
      { specId: spec.id, sheetSha256 },
      placementsFor(spec.id, sheetSha256)
    ).filter((entry) =>
      isEditorThreadVisible(entry, previewEndLineFor?.(spec.id))
    ).length;
    if (placed > 0) counts.set(spec.id, placed);
  }
  return counts;
}

// The state shown for an anchor in the timeline. A computed placement is
// authoritative. Before one exists, a hash match is CURRENT, a missing spec
// is UNAVAILABLE, and anything else is PENDING until the diff settles it.
export function resolveAnchorState(
  anchor: StatementAnchor,
  plan: Pick<Plan, "specs">,
  placement: Placement | undefined
): AnchorState {
  if (placement) return placement.state;
  const spec = plan.specs.find((candidate) => candidate.id === anchor.spec);
  return trivialPlacement(anchor, targetSha256OfSpec(spec))?.state ?? "PENDING";
}

// A truncated preview may end in the middle of a line, including the empty
// line after a trailing newline. Only preceding lines have complete content.
export const completePreviewLineCount = (
  statement: string,
  isTruncated: boolean
): number => {
  let completeLines = 0;
  for (let i = 0; i < statement.length; i++) {
    const code = statement.charCodeAt(i);
    if (code === 13) {
      if (statement.charCodeAt(i + 1) === 10) i++;
      completeLines++;
    } else if (code === 10) {
      completeLines++;
    }
  }
  return completeLines + (isTruncated ? 0 : 1);
};

export function buildWholeLineAnchor(input: {
  spec: string;
  sheetSha256: string;
  startLine: number;
  endLine: number;
}): StatementAnchor {
  const startLine = Math.min(input.startLine, input.endLine);
  const endLine = Math.max(input.startLine, input.endLine);
  return create(StatementAnchorSchema, {
    spec: input.spec,
    sheetSha256: input.sheetSha256,
    startPosition: create(PositionSchema, { line: startLine, column: 0 }),
    endPosition: create(PositionSchema, { line: endLine, column: 0 }),
  });
}

export const formatLineRange = (range: LineRange): string =>
  range.startLine === range.endLine
    ? `${range.startLine}`
    : `${range.startLine}–${range.endLine}`;

// "Line 2" or "Lines 2–4".
export const lineRangeLabel = (t: TFunction, range: LineRange): string =>
  t("plan.review.thread.anchor.line", {
    count: range.endLine - range.startLine + 1,
    range: formatLineRange(range),
  });

// The whole lines a Monaco selection covers. An end at column 1 of a later
// line covers nothing on that line.
export const selectionLineRange = (selection: {
  startLineNumber: number;
  endLineNumber: number;
  endColumn: number;
}): LineRange => ({
  startLine: selection.startLineNumber,
  endLine:
    selection.endColumn === 1 &&
    selection.endLineNumber > selection.startLineNumber
      ? selection.endLineNumber - 1
      : selection.endLineNumber,
});

const compareEditorThreads = (a: EditorThread, b: EditorThread): number =>
  a.range.startLine - b.range.startLine ||
  a.range.endLine - b.range.endLine ||
  compareByCreateTime(a.thread.root, b.thread.root);

// The statement shows only the latest standard rule review. A result a later
// run superseded stays in the timeline, unless someone reopened it, since an
// open thread holds the rollout.
const isSupersededRuleResult = (root: IssueComment): boolean =>
  root.reviewMetadata?.runType === ReviewRun_Type.RULE &&
  root.reviewMetadata.superseded &&
  root.threadState !== IssueComment_ThreadState.OPEN;

// Threads shown in the displayed statement, ordered by start line, then
// oldest first. A computed placement decides: CURRENT threads sit at their
// mapped range, which may differ from the anchor's when edits above shifted
// it; OUTDATED and UNAVAILABLE threads stay out of the editor. `placements`
// must have been computed against the displayed sheet; a thread without one
// falls back to the same no-diff rule the planner applies, so the two never
// disagree.
export function selectEditorThreads(
  threads: CommentThread[],
  target: { specId: string; sheetSha256: string },
  placements: ReadonlyMap<string, Placement> = new Map()
): EditorThread[] {
  const placed: EditorThread[] = [];
  for (const thread of threads) {
    const anchor = thread.root.statementAnchor;
    if (!anchor) continue;
    if (anchor.spec !== target.specId) continue;
    if (isSupersededRuleResult(thread.root)) continue;
    const placement =
      placements.get(thread.root.name) ??
      trivialPlacement(anchor, target.sheetSha256);
    if (placement?.state !== "CURRENT") continue;
    placed.push({ thread, range: placement.range });
  }
  return placed.sort(compareEditorThreads);
}

// Whether the statement editor shows a thread: anchored, placed CURRENT on
// its spec's current sheet, and inside the part of an oversized statement
// the editor loaded. Anything else can only be read in its Activity card.
export function isShownInEditor(
  root: IssueComment,
  specs: readonly Plan_Spec[],
  placementFor: (
    commentName: string,
    specId: string,
    sheetSha256: string | undefined
  ) => Placement | undefined,
  previewEndLineFor: (specId: string) => number | undefined
): boolean {
  const anchor = root.statementAnchor;
  if (!anchor) return false;
  const spec = specs.find((candidate) => candidate.id === anchor.spec);
  const sheetSha256 = targetSha256OfSpec(spec);
  const placement =
    placementFor(root.name, anchor.spec, sheetSha256) ??
    trivialPlacement(anchor, sheetSha256);
  if (placement?.state !== "CURRENT") return false;
  const previewEndLine = previewEndLineFor(anchor.spec);
  return (
    previewEndLine === undefined || placement.range.endLine <= previewEndLine
  );
}

// Under SQL Review V2, a folded run shows in Activity as one line, so a
// finding of it keeps its own card only where nothing else shows it: people
// replied to it, or the editor cannot show it. A result a later run superseded
// is history unless someone reopened it, since an open thread holds the
// rollout.
export function needsActivityCard(
  thread: CommentThread,
  shownInEditor: boolean
): boolean {
  if (thread.replies.length > 0) return true;
  if (thread.root.reviewMetadata?.superseded && thread.resolved) return false;
  return !shownInEditor;
}

export const isEditorThreadVisible = (
  entry: EditorThread,
  previewEndLine: number | undefined
): boolean =>
  previewEndLine === undefined || entry.range.endLine <= previewEndLine;

// The placed threads still open: what the editor walker visits and what the
// change tab counts.
export const selectUnresolvedEditorThreads = (
  ...args: Parameters<typeof selectEditorThreads>
): EditorThread[] =>
  selectEditorThreads(...args).filter((entry) => !entry.thread.resolved);

// Only the unresolved thread anchored on the earliest line expands by
// default; every other thread is a gutter marker.
export function defaultExpandedThread(
  editorThreads: EditorThread[]
): EditorThread | undefined {
  return editorThreads.find((entry) => !entry.thread.resolved);
}

// Markers sit on the last anchored line. Threads sharing that line combine
// into one marker with a count, listed oldest first.
export function groupMarkersByLine(
  editorThreads: EditorThread[]
): Map<number, EditorThread[]> {
  const byLine = new Map<number, EditorThread[]>();
  for (const entry of editorThreads) {
    const list = byLine.get(entry.range.endLine) ?? [];
    list.push(entry);
    byLine.set(entry.range.endLine, list);
  }
  for (const list of byLine.values()) {
    list.sort((a, b) => compareByCreateTime(a.thread.root, b.thread.root));
  }
  return byLine;
}

// The editor lines of a range, without terminators, on the same line model
// the placement diff uses.
export function excerptLines(statement: string, range: LineRange): string[] {
  return tokenizeLines(statement)
    .slice(range.startLine - 1, range.endLine)
    .map((line) => line.replace(/\r\n$|[\r\n]$/, ""));
}
