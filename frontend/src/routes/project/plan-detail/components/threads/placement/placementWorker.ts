import type { PlacementBudgets } from "./budgets";
import {
  diffPair,
  type PairAnchor,
  type Placement,
  placeOnPair,
  type SheetText,
  tokenizeSheet,
  UNAVAILABLE,
} from "./place";

// The message contract between the page and the placement worker. Sheets
// travel once, by name; pairs refer to them, so a target shared by several
// saved revisions is cloned and tokenized a single time.
export interface PlacementRequestPair {
  readonly key: string;
  readonly saved: string;
  readonly current: string;
  readonly anchors: readonly PairAnchor[];
}

export interface PlacementRequest {
  // Identifies the run; the page drops responses from earlier runs.
  readonly generation: number;
  // Complete sheet text by sheet name.
  readonly sheets: Readonly<Record<string, string>>;
  readonly pairs: readonly PlacementRequestPair[];
  readonly limits: Pick<
    PlacementBudgets,
    "maxLinesPerSheet" | "maxWorkPerPair" | "maxWorkPerRun"
  >;
}

export interface PlacementResponse {
  readonly generation: number;
  // Placement by pair key, then anchor id.
  readonly results: Record<string, Record<string, Placement>>;
  // Pairs whose diff ran out of budget. Their UNAVAILABLE results are not
  // definitive: a later run with budget to spare may place them.
  readonly incomplete: string[];
  // Total diff work spent.
  readonly work: number;
}

// Diffs each pair once and places every anchor on it. The per-run work budget
// is shared in request order: once it is spent, remaining pairs are
// UNAVAILABLE rather than partially computed.
export function runPlacementBatch(
  request: PlacementRequest
): PlacementResponse {
  const results: Record<string, Record<string, Placement>> = {};
  const incomplete: string[] = [];
  const sheets = new Map<string, SheetText>();
  const sheetOf = (name: string): SheetText | undefined => {
    const text = request.sheets[name];
    if (text === undefined) return undefined;
    let sheet = sheets.get(name);
    if (!sheet) {
      sheet = tokenizeSheet(text);
      sheets.set(name, sheet);
    }
    return sheet;
  };
  let remaining = request.limits.maxWorkPerRun;
  let work = 0;
  for (const pair of request.pairs) {
    const placed: Record<string, Placement> = {};
    results[pair.key] = placed;
    const settleUnavailable = () => {
      for (const anchor of pair.anchors) placed[anchor.id] = UNAVAILABLE;
    };
    if (remaining <= 0) {
      settleUnavailable();
      incomplete.push(pair.key);
      continue;
    }
    const saved = sheetOf(pair.saved);
    const current = sheetOf(pair.current);
    if (!saved || !current) {
      settleUnavailable();
      continue;
    }
    const diffed = diffPair(saved, current, {
      maxLinesPerSheet: request.limits.maxLinesPerSheet,
      maxWork: Math.min(request.limits.maxWorkPerPair, remaining),
    });
    remaining -= diffed.work;
    work += diffed.work;
    if (diffed.reason === "budget") incomplete.push(pair.key);
    for (const anchor of pair.anchors) {
      placed[anchor.id] = placeOnPair(diffed, anchor);
    }
  }
  return { generation: request.generation, results, incomplete, work };
}
