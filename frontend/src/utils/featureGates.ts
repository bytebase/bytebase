import { isDev } from "./util";

// SQL Review V2 is dark-launched: nothing triggers review runs yet and their
// results do not render, so its settings stay on the Vite dev server only. The
// review rule policy API stays live.
//
// `isDev()` is a build-time constant, so a release bundle folds every call to
// `false` and drops the UI rather than hiding it. Flipping this to `true` is
// the whole un-gate.
export const sqlReviewV2Enabled = (): boolean => isDev();

// Landing experience work can be exercised locally before it is made
// available to self-hosted production deployments.
export const isSaaSOrDev = (isSaaSMode: boolean): boolean =>
  isSaaSMode || isDev();
