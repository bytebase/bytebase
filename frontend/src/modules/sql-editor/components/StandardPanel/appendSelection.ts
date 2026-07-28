/**
 * Computes the Monaco selection range (1-based) covering the `appended` block
 * after it is joined to `oldStatement` via `appendStatement` semantics:
 *
 *   - empty old statement → `newStatement === appended`, starting at line 1
 *   - otherwise → `oldStatement + "\n\n" + appended`, so the appended block
 *     starts after the old statement plus the blank separator line.
 *
 * Returns the four args for `monaco.Selection`, selecting exactly the appended
 * block (including its first line — see PR #20554).
 */
export const computeAppendedSelection = (
  oldStatement: string,
  appended: string
): {
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
} => {
  const appendedLines = appended.split("\n");
  const startLineNumber = oldStatement
    ? oldStatement.split("\n").length + 2
    : 1;
  const endLineNumber = startLineNumber + appendedLines.length - 1;
  const endColumn = appendedLines[appendedLines.length - 1].length + 1;
  return { startLineNumber, startColumn: 1, endLineNumber, endColumn };
};
