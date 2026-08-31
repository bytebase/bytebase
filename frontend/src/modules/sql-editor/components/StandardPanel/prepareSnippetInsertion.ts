type PreparedSnippetInsertion = {
  text: string;
  /** Cursor offset relative to the start of the replaced range. */
  cursorOffset: number;
};

const followingLineBreakLength = (text: string): number => {
  if (text.startsWith("\r\n")) return 2;
  if (text.startsWith("\n") || text.startsWith("\r")) return 1;
  return 0;
};

export const prepareSnippetInsertion = (
  content: string,
  before: string,
  after: string,
  eol: string
): PreparedSnippetInsertion => {
  const snippet = content
    .replace(/\r\n?/g, "\n")
    .replace(/^\n+|\n+$/g, "")
    .replaceAll("\n", eol);
  const needsLeadingLineBreak =
    before.length > 0 && !before.endsWith("\n") && !before.endsWith("\r");
  const followingLineBreak = followingLineBreakLength(after);
  const needsTrailingLineBreak = followingLineBreak === 0;
  const text = `${needsLeadingLineBreak ? eol : ""}${snippet}${
    needsTrailingLineBreak ? eol : ""
  }`;

  return {
    text,
    cursorOffset: text.length + followingLineBreak,
  };
};
