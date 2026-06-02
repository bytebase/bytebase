import { useEffect, useMemo } from "react";
import { useAppStore } from "@/react/stores/app";
import { isValidDatabaseName } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";

interface UseAppDatabaseMetadataOptions {
  // Pass-through to `getOrFetchDatabaseMetadata` for advanced fetch
  // semantics (filtered schemas/tables, limited rows). Most consumers
  // don't pass these.
  readonly filter?: string;
  readonly limit?: number;
  readonly skipCache?: boolean;
  readonly silent?: boolean;
  // Set to `false` to suppress the self-fetch (e.g. when the parent has
  // already pre-fetched). Defaults to `true`.
  readonly autoFetch?: boolean;
}

/**
 * Reactively reads a database's metadata from the Zustand app store,
 * self-fetching when the route hasn't preloaded it. Exposes the React
 * DB schema slice as a single hook with stable refs.
 */
export const useAppDatabaseMetadata = (
  database: string,
  options: UseAppDatabaseMetadataOptions = {}
): DatabaseMetadata => {
  const {
    filter = "",
    limit = 0,
    skipCache = false,
    silent = false,
    autoFetch = true,
  } = options;
  const getDatabaseMetadata = useAppStore((s) => s.getDatabaseMetadata);
  const getOrFetchDatabaseMetadata = useAppStore(
    (s) => s.getOrFetchDatabaseMetadata
  );
  // Subscribe to the specific cache entry so re-fetches with the same
  // filter/limit trigger re-renders. Selecting the raw entry keeps the
  // snapshot stable for `useSyncExternalStore`.
  const cacheKey = `${database}/metadata::${filter}::${limit}`;
  const cached = useAppStore((s) => s.metadataByName[cacheKey]);

  useEffect(() => {
    if (!autoFetch) return;
    if (!isValidDatabaseName(database)) return;
    void getOrFetchDatabaseMetadata({
      database,
      filter,
      limit,
      skipCache,
      silent,
    });
  }, [
    autoFetch,
    database,
    filter,
    limit,
    skipCache,
    silent,
    getOrFetchDatabaseMetadata,
  ]);

  // Prefer the subscribed cache entry so consumers passing `filter` /
  // `limit` see their fetched result. `getDatabaseMetadata` only reads
  // the unfiltered `::0` slot; using it as the primary return would
  // make filtered/limited fetches resolve into a cache slot the hook
  // never returns from. Fall back to `getDatabaseMetadata` for the
  // empty-cache case so the return type stays non-nullable.
  return useMemo(
    () => cached ?? getDatabaseMetadata(database),
    [getDatabaseMetadata, database, cached]
  );
};
