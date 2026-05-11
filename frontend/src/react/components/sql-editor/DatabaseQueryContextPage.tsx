/**
 * Top-level page shim so `ReactPageMount` can locate the
 * `<DatabaseQueryContext>` root via `mount.ts`'s `pageDirs` lookup
 * (it scans for `<page>.tsx` files at the registered roots, not
 * subdirectories).
 *
 * Vue caller mounts this with `<ReactPageMount page="DatabaseQueryContextPage"
 * :page-props="{ database, context }">`. Stage 21 deletes both this shim
 * and the Vue caller once `ResultPanel.vue` flips to React and imports
 * `DatabaseQueryContext` directly.
 */
export { DatabaseQueryContext as DatabaseQueryContextPage } from "./ResultPanel/DatabaseQueryContext";
