import { ContainerChooser } from "./ContainerChooser";
import { DatabaseChooser } from "./DatabaseChooser";
import { SchemaChooser } from "./SchemaChooser";

/**
 * Replaces the <NButtonGroup> wrapping the 3 connection choosers in
 * frontend/src/views/sql-editor/EditorCommon/EditorAction.vue (lines 125-129).
 * Mounted via <ReactPageMount page="ChooserGroup" /> from the Vue parent.
 */
export function ChooserGroup() {
  return (
    <div className="flex items-center">
      <DatabaseChooser />
      <SchemaChooser />
      <ContainerChooser />
    </div>
  );
}
