/**
 * Sanitize a user-supplied base filename so it is safe to embed inside a ZIP
 * entry name or used directly as the outer download filename.
 *
 * Threat model: a database name or batch label flows from server-controlled
 * data through the UI into the ZIP we hand the user. A ZIP entry name like
 * `../../etc/passwd` is honored by some unzip implementations; NUL bytes
 * confuse downstream tooling; backslashes are path separators on Windows.
 *
 * Rules (applied in order):
 *  1. Strip NUL bytes (always invalid in filenames).
 *  2. Replace forward and back slashes with `_` so no path components survive.
 *  3. Collapse parent-directory segments (`.`, `..`) since they would still be
 *     interpretable after step 2.
 *  4. Trim leading/trailing whitespace and dots (Windows hostility).
 *  5. Truncate to 200 characters — long enough for "<dbname>-YYYYMMDD-HHMMSS"
 *     style names, short enough to keep ZIP central directory entries small.
 *  6. Fall back to "download" if the result is empty.
 *
 * Note: this intentionally does NOT validate Windows-reserved device names
 * (CON, PRN, NUL, AUX, COM1..COM9, LPT1..LPT9). Adding `.csv`/`.zip` etc.
 * after sanitization makes them legal again on Windows, and the user-side
 * impact is minor compared to the cost of false positives.
 */
export const FALLBACK_BASENAME = "download";
export const MAX_BASENAME_LENGTH = 200;

export function sanitizeBasename(input: string): string {
  let s = input ?? "";
  // Drop NUL bytes outright.
  s = s.replaceAll("\0", "");
  // Replace path separators (forward and back) with `_`.
  s = s.replace(/[/\\]/g, "_");
  // Strip Unicode bidi-override and other format characters that can spoof
  // displayed filenames in archive viewers (e.g. `evil_<U+202E>gpj.exe`
  // appears as `evil_exe.jpg`). Only strip the known-dangerous set, not all
  // of category Cf — combining marks and joiners are legitimate in non-ASCII
  // identifiers.
  //   U+061C: Arabic Letter Mark (bidi control predates U+202x)
  //   U+200B-200F: zero-width / LRM/RLM
  //   U+202A-202E: bidi embed/override
  //   U+2066-2069: bidi isolates
  //   U+FEFF: BOM
  s = s.replace(/[\u061C\u200B-\u200F\u202A-\u202E\u2066-\u2069\uFEFF]/g, "");
  // Collapse `..` and `.` segments — after replacing slashes there are no
  // segments, but a literal "..foo" or trailing "." is still treated as a
  // parent-directory hint by some unzip tools. Replace runs of dots with a
  // single dot only inside the body (leading/trailing dots are trimmed
  // separately below).
  s = s.replace(/\.{2,}/g, ".");
  // Trim leading/trailing whitespace and dots.
  s = s.replace(/^[\s.]+|[\s.]+$/g, "");
  if (s.length > MAX_BASENAME_LENGTH) {
    s = s.slice(0, MAX_BASENAME_LENGTH);
    // If the cap split a surrogate pair, drop the trailing high surrogate
    // so we don't return an invalid JS string.
    const last = s.charCodeAt(s.length - 1);
    if (last >= 0xd800 && last <= 0xdbff) {
      s = s.slice(0, -1);
    }
  }
  if (s.length === 0) {
    return FALLBACK_BASENAME;
  }
  return s;
}

/**
 * De-duplicate a filename within a set of already-used names. If `name` is
 * unused, returns it as-is and reserves it. Otherwise appends `-1`, `-2`, ...
 * before the extension (or at the end if no extension) until unused.
 *
 * Intended for ZIP entry names where two database results may share the same
 * baseFilename and we must not silently produce two entries with identical
 * names — readers behavior is unspecified and at least one entry is lost.
 *
 * Comparison is case-insensitive AND Unicode-normalized (NFC) so that
 * `Prod.csv` / `prod.csv` collide on Windows extraction, and `é` (U+00E9)
 * collides with `é` (e + U+0301) on macOS HFS+. Stored set uses the folded
 * form; returned filename preserves the caller's original casing.
 */
export function uniqueFilename(name: string, taken: Set<string>): string {
  const fold = (s: string) => s.normalize("NFC").toLowerCase();
  if (!taken.has(fold(name))) {
    taken.add(fold(name));
    return name;
  }
  // For filenames, suffix `-N` BEFORE the extension (`prod.csv` → `prod-1.csv`).
  // For path-like inputs (containing `/`) the trailing segment isn't a file
  // extension — a database named `prod.app_v2` would otherwise become
  // `prod-1.app_v2`, which mis-labels the path. Suffix at the end instead.
  const dot = name.includes("/") ? -1 : name.lastIndexOf(".");
  const stem = dot > 0 ? name.slice(0, dot) : name;
  const ext = dot > 0 ? name.slice(dot) : "";
  for (let i = 1; ; i++) {
    const candidate = `${stem}-${i}${ext}`;
    if (!taken.has(fold(candidate))) {
      taken.add(fold(candidate));
      return candidate;
    }
  }
}
