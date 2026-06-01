import { useEffect, useMemo } from "react";
import { useAppStore } from "@/react/stores/app";
import {
  catalogResourceName,
  emptyDatabaseCatalog,
} from "@/react/stores/app/databaseCatalog";
import { UNKNOWN_ID } from "@/types";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  extractDatabaseResourceName,
  getDatabaseProject,
  hasProjectPermissionV2,
} from "@/utils";

// React port of the legacy Pinia `useDatabaseCatalog` composable: fetches the
// catalog (gated by permission) and reactively returns the cached entry,
// falling back to a stable empty catalog so the selector never loops.
export function useDatabaseCatalog(
  database: string,
  skipCache: boolean
): DatabaseCatalog {
  useEffect(() => {
    if (
      extractDatabaseResourceName(database).databaseName === String(UNKNOWN_ID)
    ) {
      return;
    }
    void (async () => {
      const db = await useAppStore
        .getState()
        .getOrFetchDatabaseByName(database);
      if (
        hasProjectPermissionV2(
          getDatabaseProject(db),
          "bb.databaseCatalogs.get"
        )
      ) {
        await useAppStore
          .getState()
          .getOrFetchDatabaseCatalog({ database, skipCache });
      }
    })();
  }, [database, skipCache]);

  const cached = useAppStore(
    (state) => state.catalogsByName[catalogResourceName(database)]
  );
  return useMemo(
    () => cached ?? emptyDatabaseCatalog(database),
    [cached, database]
  );
}
