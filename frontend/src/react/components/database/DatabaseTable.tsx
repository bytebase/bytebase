import { useCallback, useEffect, useRef, useState } from "react";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useActuatorV1Store, useDatabaseV1Store } from "@/store";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { autoDatabaseRoute } from "@/utils";
import {
  type DatabaseTableMode,
  type DatabaseTableSort,
  DatabaseTableView,
} from "./DatabaseTableView";

export type { DatabaseTableMode } from "./DatabaseTableView";

export interface DatabaseTableProps {
  filter: DatabaseFilter;
  parent?: string;
  mode?: DatabaseTableMode;
  selectedNames?: Set<string>;
  onSelectedNamesChange?: (names: Set<string>) => void;
  /**
   * Fires when the visible-page database list changes. Lets a caller
   * compute "all selected on this page" for an external selection toolbar.
   *
   * Pass a stable callback reference (e.g. `setVisibleDatabases` directly,
   * not `(d) => setVisibleDatabases(d)` inline) — the effect that fires
   * this depends on the function identity, so an inline arrow recreates
   * it every render and re-fires the effect on every parent re-render.
   */
  onDatabasesChange?: (databases: Database[]) => void;
  refreshToken?: number;
}

/**
 * Server-fetching wrapper around `DatabaseTableView`. Owns paging,
 * filter, sort, and the workspace-resource scope; renders the pure view
 * for layout. Used by settings/project pages that don't already have a
 * pre-fetched database list — callers that do should compose
 * `DatabaseTableView` directly.
 */
export function DatabaseTable({
  filter,
  parent,
  mode = "ALL",
  selectedNames,
  onSelectedNamesChange,
  onDatabasesChange,
  refreshToken,
}: DatabaseTableProps) {
  const databaseStore = useDatabaseV1Store();
  const actuatorStore = useActuatorV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.databases-table");
  const fetchIdRef = useRef(0);

  const [sort, setSort] = useState<DatabaseTableSort | null>(null);
  const orderBy = sort ? `${sort.key} ${sort.order}` : "";

  const workspaceResourceName = useVueState(
    () => actuatorStore.workspaceResourceName
  );

  const fetchDatabases = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;

      if (isRefresh) {
        setLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await databaseStore.fetchDatabases({
          pageToken: token,
          pageSize,
          parent: parent || workspaceResourceName,
          filter,
          orderBy,
          skipCacheRemoval: !isRefresh,
        });

        if (currentFetchId !== fetchIdRef.current) return;

        if (isRefresh) {
          setDatabases(result.databases);
        } else {
          setDatabases((prev) => [...prev, ...result.databases]);
        }
        nextPageTokenRef.current = result.nextPageToken ?? "";
        setHasMore(Boolean(result.nextPageToken));
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setLoading(false);
          setIsFetchingMore(false);
        }
      }
    },
    [pageSize, filter, orderBy, databaseStore, parent, workspaceResourceName]
  );

  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      fetchDatabases(true);
      return;
    }
    const timer = setTimeout(() => fetchDatabases(true), 300);
    return () => clearTimeout(timer);
  }, [fetchDatabases]);

  // Support external refresh trigger
  const prevRefreshToken = useRef(refreshToken);
  useEffect(() => {
    if (
      refreshToken !== undefined &&
      refreshToken !== prevRefreshToken.current
    ) {
      prevRefreshToken.current = refreshToken;
      fetchDatabases(true);
    }
  }, [refreshToken, fetchDatabases]);

  const loadMore = useCallback(() => {
    if (nextPageTokenRef.current && !isFetchingMore) {
      fetchDatabases(false);
    }
  }, [isFetchingMore, fetchDatabases]);

  // Notify caller whenever the visible-page database list changes.
  useEffect(() => {
    onDatabasesChange?.(databases);
  }, [databases, onDatabasesChange]);

  const handleRowClick = useCallback((db: Database, e: React.MouseEvent) => {
    const url = router.resolve(autoDatabaseRoute(db)).fullPath;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }, []);

  const pageSizeOptions = getPageSizeOptions();

  return (
    <>
      <DatabaseTableView
        databases={databases}
        mode={mode}
        loading={loading}
        selectedNames={selectedNames}
        onSelectedNamesChange={onSelectedNamesChange}
        sort={sort}
        onSortChange={setSort}
        onRowClick={handleRowClick}
      />
      <div className="mt-4 mx-2">
        <PagedTableFooter
          pageSize={pageSize}
          pageSizeOptions={pageSizeOptions}
          onPageSizeChange={setPageSize}
          hasMore={hasMore}
          isFetchingMore={isFetchingMore}
          onLoadMore={loadMore}
        />
      </div>
    </>
  );
}
