import { useAppStore } from "@/react/stores/app";
import type { SQLEditorConnection, SQLEditorTab } from "@/types";
import { isValidDatabaseName, isValidInstanceName } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import { emptySQLEditorConnection, extractDatabaseResourceName } from "@/utils";
import { getInstanceResource } from "@/utils/v1/database";
import { instanceV1AllowsCrossDatabaseQuery } from "@/utils/v1/instance";

/**
 * Resolves the live `Database` + `InstanceResource` for a SQL editor tab's
 * connection. Lives under `@/react/lib/` (not `@/utils/`) because the lookup
 * goes through the React app store — putting it under `@/utils/` would
 * create a static ESM cycle: `@/utils` → `@/react/stores/app` → ...slices...
 * → `@/types` → ... → back into `@/utils`. Hosting on the React side breaks
 * the import graph cleanly while keeping the migration off Pinia.
 */
export const getConnectionForSQLEditorTab = (tab?: SQLEditorTab) => {
  const target: {
    instance: InstanceResource | undefined;
    database: Database | undefined;
  } = {
    instance: undefined,
    database: undefined,
  };
  if (!tab) {
    return target;
  }
  const { connection } = tab;
  if (connection.database) {
    const database = useAppStore
      .getState()
      .getDatabaseByName(connection.database);
    target.database = database;
    target.instance = getInstanceResource(database);
  }
  return target;
};

export const isConnectedSQLEditorTab = (tab: SQLEditorTab): boolean => {
  const { instance, database } = getConnectionForSQLEditorTab(tab);
  if (!instance) {
    return false;
  }
  if (!isValidInstanceName(instance.name)) {
    return false;
  }

  if (instanceV1AllowsCrossDatabaseQuery(instance)) {
    // Connecting to instance directly.
    return true;
  }
  return !!database && isValidDatabaseName(database.name);
};

/**
 * Resolves the connection record for a worksheet by hydrating its database
 * through the React app store. Bails on the `unknownDatabase` fallback so a
 * draft tab whose database was deleted or no longer readable doesn't get
 * `instances/-1/databases/-1` written into its connection — downstream
 * `migrateDraftsFromCache` would attempt to create a worksheet against that
 * bogus target and drop the local draft on failure.
 */
export const extractWorksheetConnection = async (worksheet: {
  database: string;
}): Promise<SQLEditorConnection> => {
  const connection = emptySQLEditorConnection();
  if (worksheet.database) {
    try {
      const database = await useAppStore
        .getState()
        .getOrFetchDatabaseByName(worksheet.database);
      if (isValidDatabaseName(database.name)) {
        const { instance } = extractDatabaseResourceName(database.name);
        connection.instance = instance;
        connection.database = database.name;
      }
    } catch {
      // Skip — leave the connection empty so the tab opens disconnected
      // rather than against a corrupt target.
    }
  }
  return connection;
};
