/**
 * Top-level page shim so `ReactPageMount` can locate the
 * `<SchemaDiagram>` root via `mount.ts`'s `pageDirs` lookup (it scans
 * for `<page>.tsx` files, not directories).
 *
 * Vue callers mount this with `<ReactPageMount page="SchemaDiagramPage"
 * :page-props="{ database, databaseMetadata, editable, onEditTable,
 * onEditColumn }">` to render the React diagram inside a still-Vue
 * surface (e.g. `SchemaEditorLite/Panels/DatabaseEditor.vue`).
 */
export { SchemaDiagram as SchemaDiagramPage } from "./SchemaDiagram";
