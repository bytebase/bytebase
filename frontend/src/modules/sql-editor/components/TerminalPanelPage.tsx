/**
 * Top-level page shim so `ReactPageMount` can locate `<TerminalPanel>`
 * via `mount.ts`'s `pageDirs` lookup (it scans for `<page>.tsx` files at
 * the registered roots, not subdirectories).
 *
 * Vue caller mounts this with `<ReactPageMount page="TerminalPanelPage">`.
 * Removed when `EditorPanel.vue` flips to React in a later slice of
 * Stage 21 and imports `TerminalPanel` directly.
 */
export { TerminalPanel as TerminalPanelPage } from "./TerminalPanel/TerminalPanel";
