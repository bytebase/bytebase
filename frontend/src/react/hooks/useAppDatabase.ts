import { useEffect, useMemo } from "react";
import { useAppStore } from "@/react/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";

/**
 * Reactively reads a database from the Zustand app store, self-fetching when
 * the current route hasn't preloaded it (the SQL editor route does not mount
 * the dashboard shells that hydrate the app store). Returns the
 * Pinia-compatible `unknownDatabase` fallback so callers can read
 * `.project` / `.instanceResource` without null checks — mirroring the legacy
 * `useDatabaseV1Store().getDatabaseByName`.
 */
export const useAppDatabase = (name: string): Database => {
  const getDatabaseByName = useAppStore((s) => s.getDatabaseByName);
  const getOrFetchDatabaseByName = useAppStore(
    (s) => s.getOrFetchDatabaseByName
  );
  // Subscribe to the specific cache entry so the value recomputes once the
  // database resolves. Selecting the raw entry keeps the snapshot stable for
  // `useSyncExternalStore`.
  const cached = useAppStore((s) => s.databasesByName[name]);
  useEffect(() => {
    if (isValidDatabaseName(name)) {
      void getOrFetchDatabaseByName(name);
    }
  }, [getOrFetchDatabaseByName, name]);
  return useMemo(
    () => getDatabaseByName(name),
    [getDatabaseByName, name, cached]
  );
};
