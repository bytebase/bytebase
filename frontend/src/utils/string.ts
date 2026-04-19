export const escapeMarkdown = (md: string): string => {
  return md.replaceAll(/[*_~\-#`[\]()\\]/g, (ch) => `\\${ch}`);
};

// Matches the Go `strings.TrimSpace` / `unicode.IsSpace` predicate so that
// the frontend's empty-title decision agrees with the backend. JS `.trim()`
// alone misses a handful of code points — notably `U+0085` (NEXT LINE),
// `U+00A0` (NBSP), ideographic spaces, and friends. Use this helper
// everywhere the frontend decides "is this title empty / is Create allowed /
// should the fallback fire".
const UNICODE_WHITESPACE =
  /^[\s\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF]+|[\s\u0085\u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF]+$/g;

export const normalizeTitle = (title: string): string =>
  title.replace(UNICODE_WHITESPACE, "");
