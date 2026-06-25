/**
 * Supported gh-ost parameters for online schema migration.
 *
 * The parameter list lives in `flags.json` — the single source of truth the
 * Configure UI builds from. `TestFrontendFlagManifestInSync` in
 * `backend/component/ghost/manifest_test.go` fails CI if `flags.json` drifts from
 * the backend allowlist and defaults in `backend/component/ghost/config.go`
 * (`knownKeys` + `defaultConfig`), so a flag added, removed, or re-defaulted on
 * one side cannot silently diverge from the other.
 *
 * Configuration travels with the SQL statement as a single-line directive
 * (`-- gh-ost = {...}`); only values that differ from the backend default are
 * written, so the directive stays minimal.
 */

import flagManifest from "./flags.json";

export type GhostParameterType = "bool" | "int" | "float" | "string";

export interface GhostParameter {
  key: string;
  type: GhostParameterType;
  /**
   * The effective backend default, expressed as the string the directive would
   * carry. An empty string means "no default / unset" (e.g. `max-load`).
   */
  default: string;
}

// Built from flags.json — ordered most-commonly-tuned first, booleans last.
export const GHOST_PARAMETERS: GhostParameter[] = flagManifest.map(
  (flag): GhostParameter => ({
    key: flag.key,
    type: flag.type as GhostParameterType,
    default: flag.default,
  })
);

/**
 * Applies a single flag edit, keeping only overrides — values that differ from
 * the backend default. The raw control value is coerced to the directive's
 * string form (and dropped when it equals the default or is empty), mirroring
 * the backend, which only overrides a default when the key is present. Unknown
 * keys already in `flags` (e.g. hand-typed) are preserved.
 */
export function withFlag(
  flags: Record<string, string>,
  param: GhostParameter,
  raw: string | number | boolean | null | undefined
): Record<string, string> {
  const next = { ...flags };
  let value: string | undefined;
  if (raw === undefined || raw === null) {
    value = undefined;
  } else if (param.type === "bool") {
    value = raw ? "true" : "false";
  } else {
    const trimmed = String(raw).trim();
    value = trimmed === "" ? undefined : trimmed;
  }
  if (value === undefined || value === param.default) {
    delete next[param.key];
  } else {
    next[param.key] = value;
  }
  return next;
}
