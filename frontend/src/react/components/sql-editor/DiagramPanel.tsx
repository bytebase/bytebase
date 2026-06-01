import { SchemaDiagram } from "@/react/components/SchemaDiagram";
import { useAppDatabaseMetadata } from "@/react/hooks/useAppDatabaseMetadata";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/hooks/useSQLEditorBridge";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/DiagramPanel/DiagramPanel.vue`.
 *
 * Reads the current tab's database from the SQL Editor connection
 * store + its metadata from the app store, and forwards both
 * to `<SchemaDiagram>`. Mounted by `Panels.vue` via `ReactPageMount`.
 */
export function DiagramPanel() {
  const { database } = useConnectionOfCurrentSQLEditorTab();
  // Parent `Panels.tsx` drives the metadata fetch; we only subscribe.
  const databaseMetadata = useAppDatabaseMetadata(database.name, {
    autoFetch: false,
  });

  return (
    <div className="w-full h-full bb-react-schema-diagram">
      <SchemaDiagram database={database} databaseMetadata={databaseMetadata} />
    </div>
  );
}
