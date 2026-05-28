import { SchemaDiagram } from "@/react/components/SchemaDiagram";
import { usePiniaBridge } from "@/react/hooks/usePiniaBridge";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";
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
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseMetadata = usePiniaBridge(
    () => dbSchemaStore.getDatabaseMetadata(database.name),
    { deep: true }
  );

  return (
    <div className="w-full h-full bb-react-schema-diagram">
      <SchemaDiagram database={database} databaseMetadata={databaseMetadata} />
    </div>
  );
}
