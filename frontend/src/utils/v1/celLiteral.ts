// Pure CEL string-literal helpers. Deliberately dependency-free: every filter
// builder needs them, and importing this must not pull the RPC clients in
// `./cel` into a caller's module graph.

// Escapes a value for embedding inside a double-quoted CEL string literal.
// Backslash must be escaped first. Prefer `celString`, which also writes the
// surrounding quotes — a call site that forgets them looks identical to one
// that never escaped at all.
const escapeCELStringLiteral = (value: string): string =>
  value
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")
    .replace(/\t/g, "\\t");

// Renders a value as a complete CEL string literal, quotes included:
// `statement.contains(${celString(keyword)})`. Never interpolate a value into
// `"..."` by hand — free text holding a double quote (a SQL identifier like
// `"public"`, anything typed into a search box) closes the literal early, and
// the backend rejects the whole filter with InvalidArgument.
export const celString = (value: string): string =>
  `"${escapeCELStringLiteral(value)}"`;

// Renders values as a CEL list literal, e.g. `["a", "b"]`, for `in` operands.
export const celStringList = (values: readonly string[]): string =>
  `[${values.map(celString).join(", ")}]`;

// Renders a map field selector. CEL only parses `labels.foo` when the key is
// an identifier, so a key holding anything else — a dash, which label keys
// allow — must go through index syntax to reach the backend at all.
export const celMapField = (field: string, key: string): string =>
  `${field}[${celString(key)}]`;
