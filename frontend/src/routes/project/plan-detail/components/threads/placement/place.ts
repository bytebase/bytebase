import type { StatementAnchor } from "@/types/proto-es/v1/issue_service_pb";
import { tokenizeLines } from "./lineTokens";
import { diffLines, type Hunk } from "./myersDiff";

// One-based, inclusive. The one range shape shared by anchors, placements,
// and the editor.
export interface LineRange {
  readonly startLine: number;
  readonly endLine: number;
}

// Where an anchored comment shows for the spec's current statement. CURRENT
// carries the range in the current sheet; the text there is byte for byte
// what the reviewer selected. OUTDATED means the selected lines changed.
// UNAVAILABLE means the inputs were missing or invalid, or a limit was hit.
export type Placement =
  | { readonly state: "CURRENT"; readonly range: LineRange }
  | { readonly state: "OUTDATED" }
  | { readonly state: "UNAVAILABLE" };

export const OUTDATED: Placement = Object.freeze({ state: "OUTDATED" });
export const UNAVAILABLE: Placement = Object.freeze({ state: "UNAVAILABLE" });

export const current = (startLine: number, endLine: number): Placement => ({
  state: "CURRENT",
  range: { startLine, endLine },
});

// An anchor on a saved/current pair. `id` names the line range, so identical
// ranges from different comments share one computation.
export interface PairAnchor extends LineRange {
  readonly id: string;
}

export const anchorId = (range: LineRange): string =>
  `${range.startLine}-${range.endLine}`;

// This version supports whole-line anchors only: both columns zero, one-based
// lines, and an inclusive end at or after the start.
export function wholeLineRange(
  anchor: Pick<StatementAnchor, "startPosition" | "endPosition">
): LineRange | undefined {
  const start = anchor.startPosition;
  const end = anchor.endPosition;
  if (!start || !end) return undefined;
  if (start.column !== 0 || end.column !== 0) return undefined;
  if (start.line < 1 || end.line < start.line) return undefined;
  return { startLine: start.line, endLine: end.line };
}

// The part of the placement rule that needs no diff, shared by the planner
// and by every consumer that shows a comment before a run has settled it:
// an unsupported anchor is UNAVAILABLE, no target (spec gone or without a
// statement) is UNAVAILABLE, and an anchor on the target sheet itself is
// CURRENT at its own lines. Undefined means the diff has to decide.
export function trivialPlacement(
  anchor: Pick<
    StatementAnchor,
    "startPosition" | "endPosition" | "sheetSha256"
  >,
  targetSha256: string | undefined
): Placement | undefined {
  const range = wholeLineRange(anchor);
  if (!range || !targetSha256) return UNAVAILABLE;
  if (targetSha256 === anchor.sheetSha256) {
    return current(range.startLine, range.endLine);
  }
  return undefined;
}

// Maps the selected saved lines through `hunks` (a complete saved-to-current
// diff). Hunks entirely above shift the selection by their net line count,
// hunks entirely below have no effect, and a hunk that touches a selected
// line or inserts between selected lines outdates the comment.
export function placeWithHunks(
  saved: readonly string[],
  currentLines: readonly string[],
  hunks: readonly Hunk[],
  range: LineRange
): Placement {
  const { startLine: first, endLine: last } = range;
  if (!(first >= 1 && first <= last && last <= saved.length)) {
    return UNAVAILABLE;
  }
  let delta = 0;
  for (const hunk of hunks) {
    const sEnd = hunk.sStart + hunk.sLen - 1;
    if (hunk.sLen === 0 && hunk.sStart <= first) {
      delta += hunk.cLen;
    } else if (hunk.sLen > 0 && sEnd < first) {
      delta += hunk.cLen - hunk.sLen;
    } else if (hunk.sStart > last) {
      continue;
    } else {
      return OUTDATED;
    }
  }
  const mappedFirst = first + delta;
  const mappedLast = last + delta;
  if (!(mappedFirst >= 1 && mappedLast <= currentLines.length)) {
    return UNAVAILABLE;
  }
  for (let i = 0; i <= last - first; i++) {
    if (saved[first - 1 + i] !== currentLines[mappedFirst - 1 + i]) {
      return UNAVAILABLE;
    }
  }
  return current(mappedFirst, mappedLast);
}

// A sheet's complete text with its editor lines, tokenized once however many
// pairs it takes part in.
export interface SheetText {
  readonly text: string;
  readonly lines: readonly string[];
}

export const tokenizeSheet = (text: string): SheetText => ({
  text,
  lines: tokenizeLines(text),
});

// A diffed saved/current pair, reusable for every anchor on that pair.
export interface DiffedPair {
  readonly saved: readonly string[];
  readonly current: readonly string[];
  // Undefined when the diff did not run to completion; `reason` says why.
  // A "budget" miss is retriable with more budget, a "lines" miss is not.
  readonly hunks?: readonly Hunk[];
  readonly reason?: "budget" | "lines";
  readonly work: number;
}

export interface DiffPairLimits {
  readonly maxLinesPerSheet: number;
  readonly maxWork: number;
}

// Diffs one pair once. Identical contents skip the diff. A sheet over the
// line cap or a diff over the work budget yields no hunks, and every anchor
// on the pair then resolves UNAVAILABLE.
export function diffPair(
  saved: SheetText,
  current: SheetText,
  limits: DiffPairLimits
): DiffedPair {
  const base = { saved: saved.lines, current: current.lines };
  if (
    saved.lines.length > limits.maxLinesPerSheet ||
    current.lines.length > limits.maxLinesPerSheet
  ) {
    return { ...base, reason: "lines", work: 0 };
  }
  if (saved.text === current.text) {
    return { ...base, hunks: [], work: 0 };
  }
  const result = diffLines(saved.lines, current.lines, limits.maxWork);
  return result.complete
    ? { ...base, hunks: result.hunks, work: result.work }
    : { ...base, reason: "budget", work: result.work };
}

export function placeOnPair(pair: DiffedPair, range: LineRange): Placement {
  if (!pair.hunks) return UNAVAILABLE;
  return placeWithHunks(pair.saved, pair.current, pair.hunks, range);
}
