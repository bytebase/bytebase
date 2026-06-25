/**
 * Supported gh-ost parameters for online schema migration.
 *
 * This list mirrors the backend allowlist and defaults in
 * `backend/component/ghost/config.go` (`knownKeys` + `defaultConfig`). Keep the
 * two in sync — an entry here that the backend does not accept is rejected at
 * run time.
 *
 * Configuration travels with the SQL statement as a single-line directive
 * (`-- gh-ost = {...}`); only values that differ from the backend default are
 * written, so the directive stays minimal.
 */

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

// Ordered most-commonly-tuned first; booleans grouped at the end.
export const GHOST_PARAMETERS: GhostParameter[] = [
  { key: "max-load", type: "string", default: "" },
  { key: "chunk-size", type: "int", default: "1000" },
  { key: "max-lag-millis", type: "int", default: "1500" },
  { key: "dml-batch-size", type: "int", default: "10" },
  { key: "nice-ratio", type: "float", default: "0" },
  { key: "cut-over-lock-timeout-seconds", type: "int", default: "10" },
  { key: "default-retries", type: "int", default: "60" },
  { key: "exponential-backoff-max-interval", type: "int", default: "64" },
  { key: "heartbeat-interval-millis", type: "int", default: "100" },
  { key: "throttle-control-replicas", type: "string", default: "" },
  { key: "allow-on-master", type: "bool", default: "true" },
  { key: "attempt-instant-ddl", type: "bool", default: "true" },
  { key: "switch-to-rbr", type: "bool", default: "false" },
  { key: "assume-rbr", type: "bool", default: "false" },
  { key: "assume-master-host", type: "bool", default: "false" },
];

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
