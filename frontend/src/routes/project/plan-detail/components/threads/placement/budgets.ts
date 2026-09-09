// Fixed limits for computing placements in the browser. Fetching is bounded
// before it starts and the diff is bounded while it runs; anything past a limit
// is UNAVAILABLE, never a partial answer.
//
// The values are provisional placeholders until page-load measurements on
// representative plans set them (design open question 3). Keep them fixed in
// code rather than configurable so results are reproducible.
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
  // Matches the backend preview cutoff (common.MaxSheetSize).
  maxBytesPerSheet: 2 * 1024 * 1024,
  maxTotalBytes: 8 * 1024 * 1024,
  maxLinesPerSheet: 50_000,
  maxWorkPerPair: 5_000_000,
  maxWorkPerRun: 20_000_000,
  workerDeadlineMs: 5_000,
});
