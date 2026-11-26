import { useAppStore } from "@/stores/app";
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

// SQL Review V2 runs on Bytebase Cloud and on the Vite dev server, unless that
// browser switched back to V1; a self-hosted release keeps V1. On the server,
// the profile's SQLReviewV2 is the same switch.
//
// Cloud is known once the server info loads, which happens before the app
// mounts for a signed-in user, so read the gate when rendering or navigating,
// never at module load.
export const sqlReviewV2Enabled = (): boolean =>
  isDev()
    ? devSqlReviewVersion() === "v2"
    : useAppStore.getState().isSaaSMode();

export const isSaaSOrDev = (isSaaSMode: boolean): boolean =>
  isSaaSMode || isDev();
