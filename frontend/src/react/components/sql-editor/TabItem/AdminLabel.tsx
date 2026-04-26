import { ChevronRight } from "lucide-react";
import { useEffect } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types/const";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";
import { isValidDatabaseName } from "@/types/v1/database";
import { isValidInstanceName } from "@/types/v1/instance";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
} from "@/utils";

type Props = {
  readonly tab: SQLEditorTab;
};

/**
 * Replaces frontend/src/views/sql-editor/TabList/TabItem/AdminLabel.vue.
 * Breadcrumb shown in the tab title when a tab is in ADMIN mode:
 * `environment > instance > database`. Hides the environment segment when
 * the database has no environment (unknown).
 */
export function AdminLabel({ tab }: Props) {
  const dbName = tab.connection.database;
  const databaseStore = useDatabaseV1Store();

  // Mirror Vue's `useDatabaseV1ByName`: fire a fetch on mount and every time
  // the database resource name changes.
  useEffect(() => {
    if (!dbName) return;
    void databaseStore.getOrFetchDatabaseByName(dbName);
  }, [databaseStore, dbName]);

  const database = useVueState(() => databaseStore.getDatabaseByName(dbName));
  const instance = getInstanceResource(database);
  const environment = getDatabaseEnvironment(database);
  const { databaseName } = extractDatabaseResourceName(database.name);
  const hideEnvironment = environment?.id === String(UNKNOWN_ID);

  return (
    <label className="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
      {!hideEnvironment && (
        <>
          <span>{environment.title}</span>
          <ChevronRight className="shrink-0 h-4 w-4 opacity-70" />
        </>
      )}
      {isValidInstanceName(instance.name) && (
        <>
          <span>{instance.title}</span>
          <ChevronRight className="shrink-0 h-4 w-4 opacity-70" />
        </>
      )}
      {isValidDatabaseName(database.name) && <span>{databaseName}</span>}
    </label>
  );
}
