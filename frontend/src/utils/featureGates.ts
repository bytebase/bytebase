import { STORAGE_KEY_SQL_REVIEW_VERSION } from "./storage-keys";
import { isDev } from "./util";

export type SqlReviewVersion = "v1" | "v2";

// The SQL review this browser shows on the Vite dev server. The workspace SQL
// Review page switches it, together with the workspace standard rules the
// server reads to choose between V1 and V2.
export const devSqlReviewVersion = (): SqlReviewVersion =>
  localStorage.getItem(STORAGE_KEY_SQL_REVIEW_VERSION) === "v1" ? "v1" : "v2";

export const setDevSqlReviewVersion = (version: SqlReviewVersion): void => {
  localStorage.setItem(STORAGE_KEY_SQL_REVIEW_VERSION, version);
};

// SQL Review V2 is dark-launched: its settings and the plan page's SQL review
// row stay on the Vite dev server only, unless that browser switched back to
// V1. The review rule policy API stays live. On the server, the profile's
// SQLReviewV2 is the same switch, on only in a dev build.
//
// `isDev()` is a build-time constant, so a release bundle folds every call to
// `false` and drops the UI rather than hiding it. Un-gating flips this and the
// server switch together.
export const sqlReviewV2Enabled = (): boolean =>
  isDev() && devSqlReviewVersion() === "v2";

export const isSaaSOrDev = (isSaaSMode: boolean): boolean =>
  isSaaSMode || isDev();
