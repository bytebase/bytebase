/**
 * Top-level page shim so `ReactPageMount` can locate the React
 * `<ResultPanel>` via `mount.ts`'s `pageDirs` lookup (it scans for
 * `<page>.tsx` files at the registered roots, not subdirectories).
 *
 * Vue caller mounts this with `<ReactPageMount page="ResultPanelPage">`.
 * The shim is removed when `StandardPanel.vue` flips to React in the
 * next slice of Stage 21 and imports `ResultPanel` directly.
 */
export { ResultPanel as ResultPanelPage } from "./ResultPanel/ResultPanel";
