import { SchemaDiagram } from "@/react/components/SchemaDiagram";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/DiagramPanel/DiagramPanel.vue`.
 *
 * Reads the current tab's database from the SQL Editor connection
 * store + its metadata from `useDBSchemaV1Store`, and forwards both
 * to `<SchemaDiagram>`. Mounted by `Panels.vue` via `ReactPageMount`.
 */
export function DiagramPanel() {
  const dbSchemaStore = useDBSchemaV1Store();
  const { database: databaseRef } = useConnectionOfCurrentSQLEditorTab();
  const database = useVueState(() => databaseRef.value);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(database.name),
    { deep: true }
  );

  return (
    <div className="w-full h-full bb-react-schema-diagram">
      <SchemaDiagram database={database} databaseMetadata={databaseMetadata} />
    </div>
  );
}
