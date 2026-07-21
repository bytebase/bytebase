/**
 * Top-level page shim so `ReactPageMount` can locate the `<ResultView>`
 * root via `mount.ts`'s `pageDirs` lookup (it scans for `<page>.tsx`
 * files at the registered roots, not subdirectories).
 *
 * Vue callers mount this with `<ReactPageMount page="ResultViewPage"
 * :page-props="{ executeParams, database, resultSet, loading, dark }">`
 * to render the React result-view stack inside the still-Vue
 * `DatabaseQueryContext.vue` and `TerminalPanel.vue` shells. Phase 21
 * flips those Vue hosts to React, after which this shim can be deleted.
 */
export { ResultView as ResultViewPage } from "./ResultView/ResultView";
