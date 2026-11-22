// A driver records the statement it ran, which for a row-limited query is not
// always what the user typed: the parsers either fit a LIMIT onto the statement
// or, where they cannot, wrap the whole of it in a `WITH result AS (…)` query.
const RESULT_LIMIT_WRAPPER =
  /^WITH\s+result\s+AS\s*\(([\s\S]*)\)\s*SELECT\s+(?:TOP\s+\d+\s+)?\*\s+FROM\s+result(?:\s+LIMIT\s+\d+)?\s*;?$/i;

const TRAILING_LIMIT = /\s+LIMIT\s+\d+\s*;?\s*$/i;

/**
 * Recovers the text the user typed from the statement a driver ran, so it can
 * be found in the editor. A statement no rewrite touched comes back unchanged.
 */
export function statementAsTyped(executed: string): string {
  const trimmed = executed.trim();
  const wrapped = trimmed.match(RESULT_LIMIT_WRAPPER);
  // The wrapper holds the statement verbatim, so its own LIMIT is the only one
  // to drop; a LIMIT inside it is the user's.
  if (wrapped) return wrapped[1].trim();
  return trimmed.replace(TRAILING_LIMIT, "").trim();
}
