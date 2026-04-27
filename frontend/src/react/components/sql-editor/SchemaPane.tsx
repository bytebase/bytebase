/**
 * Top-level mount-shim for the SchemaPane subtree. The mount registry
 * (`src/react/mount.ts`) discovers React pages via a non-recursive glob
 * (`./components/sql-editor/*.tsx`); the subtree itself lives one level
 * deeper at `./SchemaPane/SchemaPane.tsx` so its helpers (TreeNode/,
 * HoverPanel/, schemaTree.ts, …) co-locate. Re-exporting here keeps the
 * file structure clean and lets `<ReactPageMount page="SchemaPane" />`
 * find the component.
 */
export { SchemaPane } from "./SchemaPane/SchemaPane";
