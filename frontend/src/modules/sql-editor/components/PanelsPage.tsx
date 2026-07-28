/**
 * Top-level page shim so `ReactPageMount` can locate `<Panels>` via
 * `mount.ts`'s `pageDirs` lookup (it scans for `<page>.tsx` files at
 * the registered roots, not subdirectories).
 *
 * Vue caller mounts this with `<ReactPageMount page="PanelsPage">`.
 * Removed when `EditorPanel.vue` flips to React in the next slice of
 * Stage 21 and imports `Panels` directly.
 */
export { Panels as PanelsPage } from "./Panels/Panels";
