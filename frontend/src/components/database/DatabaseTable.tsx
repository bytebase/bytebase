import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { router } from "@/app/router";
import { useListScrollRestorationLoadMore } from "@/app/router/NavigationScrollRestoration";
import { PagedTableFooter } from "@/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/hooks/useSessionPageSize";
import type { DatabaseFilter } from "@/lib/databaseFilter";
import { useAppStore } from "@/stores/app";
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
  /** Row click selects instead of navigating to the database page. */
  selectOnRowClick?: boolean;
  onOpenDatabase?: () => void;
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
  onLoadingChange?: (loading: boolean) => void;
  refreshToken?: number;
  emptyPlaceholder?: React.ReactNode;
  selectionColumnIntroTarget?: string;
}

function databaseFilterKey(filter: DatabaseFilter | string | undefined) {
  if (!filter) return "";
  if (typeof filter === "string") return filter;
  return JSON.stringify({
    project: filter.project ?? "",
    instance: filter.instance ?? "",
    environment: filter.environment ?? "",
    query: filter.query ?? "",
    showDeleted: filter.showDeleted ?? false,
    excludeUnassigned: filter.excludeUnassigned ?? false,
    labels: filter.labels ?? [],
    engines: filter.engines ?? [],
    excludeEngines: filter.excludeEngines ?? [],
    table: filter.table ?? "",
  });
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
  selectOnRowClick = false,
  onOpenDatabase,
  selectedNames,
  onSelectedNamesChange,
  onDatabasesChange,
  onLoadingChange,
  refreshToken,
  emptyPlaceholder,
  selectionColumnIntroTarget,
}: DatabaseTableProps) {
  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.databases-table");
  const fetchIdRef = useRef(0);

  const [sort, setSort] = useState<DatabaseTableSort | null>(null);
  const orderBy = sort ? `${sort.key} ${sort.order}` : "";
  const filterRef = useRef(filter);
  filterRef.current = filter;
  const filterKey = useMemo(() => databaseFilterKey(filter), [filter]);

  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());

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
        const result = await useAppStore.getState().fetchDatabases({
          pageToken: token,
          pageSize,
          parent: parent || workspaceResourceName,
          filter: filterRef.current,
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
    [pageSize, filterKey, orderBy, parent, workspaceResourceName]
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
  useListScrollRestorationLoadMore({
    dataList: databases,
    hasMore: !selectOnRowClick && hasMore,
    isFetchingMore,
    isLoading: loading,
    loadMore,
  });

  // Notify caller whenever the visible-page database list changes.
  useEffect(() => {
    onDatabasesChange?.(databases);
  }, [databases, onDatabasesChange]);
  useEffect(() => {
    onLoadingChange?.(loading);
  }, [loading, onLoadingChange]);

  const handleRowClick = useCallback(
    (db: Database, e: React.MouseEvent) => {
      const url = router.resolve(autoDatabaseRoute(db)).fullPath;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        onOpenDatabase?.();
        router.push(url);
      }
    },
    [onOpenDatabase]
  );

  const pageSizeOptions = getPageSizeOptions();

  return (
    <>
      <DatabaseTableView
        databases={databases}
        mode={mode}
        loading={loading}
        emptyPlaceholder={
          !loading && databases.length === 0 && !hasMore
            ? emptyPlaceholder
            : undefined
        }
        selectedNames={selectedNames}
        onSelectedNamesChange={onSelectedNamesChange}
        sort={sort}
        onSortChange={setSort}
        onRowClick={selectOnRowClick ? undefined : handleRowClick}
        selectOnRowClick={selectOnRowClick}
        selectionColumnIntroTarget={selectionColumnIntroTarget}
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
