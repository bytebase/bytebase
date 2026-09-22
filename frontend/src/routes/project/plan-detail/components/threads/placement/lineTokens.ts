// Splits raw sheet content into editor lines the way the design's placement
// rule expects: CRLF, LF, and a lone CR each end a line, every token keeps its
// own terminator, and the final token is always emitted even when empty, so
// "A\n" has two lines like it does in the editor. Concatenating the tokens
// reproduces the input byte for byte. Token index + 1 is the editor line.
export function tokenizeLines(text: string): string[] {
  const tokens: string[] = [];
  let start = 0;
  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i);
    if (code === 10) {
      tokens.push(text.slice(start, i + 1));
      start = i + 1;
    } else if (code === 13) {
      if (text.charCodeAt(i + 1) === 10) i++;
      tokens.push(text.slice(start, i + 1));
      start = i + 1;
    }
  }
  tokens.push(text.slice(start));
  return tokens;
}
