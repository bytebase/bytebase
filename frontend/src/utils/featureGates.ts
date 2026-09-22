import { isDev } from "./util";

// Inline comment threads on plan detail are still stabilizing: the browser
// journeys in plan-detail-review.spec.ts do not run in CI, and re-anchoring a
// comment onto a newer revision is unverified against a saved sheet. Until
// that closes the feature stays on the Vite dev server only.
//
// `isDev()` is a build-time constant, so a release bundle folds every call to
// `false` and drops the thread UI rather than hiding it. Flipping this to
// `true` is the whole un-gate.
export const inlineThreadsEnabled = (): boolean => isDev();
