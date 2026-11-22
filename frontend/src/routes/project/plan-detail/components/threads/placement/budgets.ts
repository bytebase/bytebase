import { SHEET_PREVIEW_CHARACTER_LIMIT } from "@/utils/v1/sheet";

const MAX_SHEET_BYTES = 4 * SHEET_PREVIEW_CHARACTER_LIMIT;

// Fixed limits for computing placements in the browser. Fetching is bounded
// before it starts and the diff is bounded while it runs; anything past a limit
// is UNAVAILABLE, never a partial answer.
//
// The resource budgets are provisional until page-load measurements on
// representative plans set them. Keep them fixed so results are reproducible.
export interface PlacementBudgets {
  // Distinct sheets one run may download, and again the distinct sheets one
  // run may hand to the worker, cached or not.
  readonly maxSheets: number;
  // Upper bound on a single sheet's content size in bytes.
  readonly maxBytesPerSheet: number;
  // Upper bound on the bytes downloaded by one run, sources and targets, and
  // again on the bytes one run hands to the worker.
  readonly maxTotalBytes: number;
  // Upper bound on the lines of one tokenized sheet.
  readonly maxLinesPerSheet: number;
  // Diff frontier steps plus token comparisons for one saved/current pair.
  readonly maxWorkPerPair: number;
  // The same unit, summed over every pair in one run.
  readonly maxWorkPerRun: number;
  // Wall-clock limit for the worker; the worker is terminated when it expires.
  readonly workerDeadlineMs: number;
}

export const PLACEMENT_BUDGETS: PlacementBudgets = Object.freeze({
  maxSheets: 16,
  // UTF-8 uses at most four bytes per character. These separate byte caps
  // bound downloads and worker payloads without rejecting valid previews.
  maxBytesPerSheet: MAX_SHEET_BYTES,
  maxTotalBytes: 2 * MAX_SHEET_BYTES,
  maxLinesPerSheet: 50_000,
  maxWorkPerPair: 5_000_000,
  maxWorkPerRun: 20_000_000,
  workerDeadlineMs: 5_000,
});
