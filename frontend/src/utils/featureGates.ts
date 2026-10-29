import { isDev } from "./util";

// SQL Review V2 is dark-launched: its settings and the standard rule review in
// the plan page's Checks row stay on the Vite dev server only. The review rule
// policy API stays live. On the server, the profile's SQLReviewV2 is the same
// switch, on only in a dev build.
//
// `isDev()` is a build-time constant, so a release bundle folds every call to
// `false` and drops the UI rather than hiding it. Un-gating flips this and the
// server switch together.
export const sqlReviewV2Enabled = (): boolean => isDev();

export const isSaaSOrDev = (isSaaSMode: boolean): boolean =>
  isSaaSMode || isDev();
