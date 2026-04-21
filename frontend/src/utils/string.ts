export const escapeMarkdown = (md: string): string => {
  return md.replaceAll(/[*_~\-#`[\]()\\]/g, (ch) => `\\${ch}`);
};

// Matches the Go `strings.TrimSpace` / `unicode.IsSpace` predicate so that
// the frontend's empty-title decision agrees with the backend.
//
// We intentionally do NOT use JS `\s` here: ECMAScript's `\s` matches
// `U+FEFF` (ZERO WIDTH NO-BREAK SPACE / BOM), but Go's `unicode.IsSpace`
// does NOT — so `\s` alone would trim BOM on the frontend while the
// backend preserved it, re-introducing the very asymmetry this helper
// exists to close.
//
// The character class below enumerates exactly the code points
// `unicode.IsSpace` treats as whitespace: ASCII whitespace
// (`\t`, `\n`, `\v`, `\f`, `\r`, space), `U+0085` (NEXT LINE),
// `U+00A0` (NBSP), `U+1680` (OGHAM SPACE MARK), `U+2000`–`U+200A`,
// `U+2028` (LINE SEPARATOR), `U+2029` (PARAGRAPH SEPARATOR),
// `U+202F` (NARROW NO-BREAK SPACE), `U+205F` (MEDIUM MATHEMATICAL SPACE),
// and `U+3000` (IDEOGRAPHIC SPACE). No `U+FEFF` — see negative test in
// `string.test.ts` that locks this invariant.
//
// Use this helper everywhere the frontend decides "is this title empty /
// is Create allowed / should the fallback fire".
const UNICODE_WHITESPACE_CLASS =
  "\\t\\n\\v\\f\\r \\u0085\\u00A0\\u1680\\u2000-\\u200A\\u2028\\u2029\\u202F\\u205F\\u3000";
const UNICODE_WHITESPACE = new RegExp(
  `^[${UNICODE_WHITESPACE_CLASS}]+|[${UNICODE_WHITESPACE_CLASS}]+$`,
  "g"
);

export const normalizeTitle = (title: string): string =>
  title.replace(UNICODE_WHITESPACE, "");
