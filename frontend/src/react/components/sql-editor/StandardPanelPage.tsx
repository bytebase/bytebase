/**
 * Top-level page shim so `ReactPageMount` can locate `<StandardPanel>`
 * via `mount.ts`'s `pageDirs` lookup (it scans for `<page>.tsx` files
 * at the registered roots, not subdirectories).
 *
 * Vue caller mounts this with `<ReactPageMount page="StandardPanelPage">`.
 * Removed when `EditorPanel.vue` flips to React in a later slice of
 * Stage 21 and imports `StandardPanel` directly.
 */
export { StandardPanel as StandardPanelPage } from "./StandardPanel/StandardPanel";
