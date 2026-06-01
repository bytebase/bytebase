// In-memory store for SQL statement ranges, keyed by document URI.
//
// The backend pushes `$/textDocument/statementRanges` over the LSP socket when
// a document opens (`didOpen`) and on every edit (`didChange`). The active-
// statement highlight reads these ranges. They must outlive a single editor
// instance, because the SQL Editor remounts the editor on every tab switch
// (`key={filename}`) and `didOpen` fires only once per model per page session
// (models are cached and never disposed), so switching back to a previously
// visited tab gets no fresh push. A module-level map keyed by URI survives
// those remounts.
//
// This is intentionally NOT persisted to localStorage: a page reload creates a
// fresh model, which re-triggers `didOpen` and re-fetches the ranges from the
// backend. Keeping it in memory avoids unbounded storage growth and stale
// ranges that no longer match a worksheet's content.

export type StatementRange = {
  startLineNumber: number;
  endLineNumber: number;
  startColumn: number;
  endColumn: number;
};

const store = new Map<string, StatementRange[]>();

export const getStatementRanges = (uri: string): StatementRange[] =>
  store.get(uri) ?? [];

export const setStatementRanges = (
  uri: string,
  ranges: StatementRange[]
): void => {
  store.set(uri, ranges);
};
