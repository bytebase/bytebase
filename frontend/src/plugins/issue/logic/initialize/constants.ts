// validateOnly: true doesn't support empty SQL
// so we use a fake sql to validate and then set it back to empty if needed
export const VALIDATE_ONLY_SQL = "/* YOUR_SQL_HERE */";

export const ESTABLISH_BASELINE_SQL =
  "/* Establish baseline using current schema */";
