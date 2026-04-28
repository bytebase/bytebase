import { CheckCircle, XCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LabelsDisplay } from "@/react/components/LabelsDisplay";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { PagedTableFooter } from "@/react/hooks/usePagedData";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { useActuatorV1Store, useDatabaseV1Store } from "@/store";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";
import {
  autoDatabaseRoute,
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDatabaseEnvironment,
  getDatabaseProject,
  getInstanceResource,
  hostPortOfInstanceV1,
} from "@/utils";
import { extractReleaseUID } from "@/utils/v1/release";

export type DatabaseTableMode = "ALL" | "PROJECT";

export interface DatabaseTableProps {
  filter: DatabaseFilter;
  parent?: string;
  mode?: DatabaseTableMode;
  selectedNames?: Set<string>;
  onSelectedNamesChange?: (names: Set<string>) => void;
  refreshToken?: number;
}

interface DatabaseColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  sortKey?: string;
  cellClassName?: string;
  render: (database: Database) => React.ReactNode;
}

export function DatabaseTable({
  filter,
  parent,
  mode = "ALL",
  selectedNames,
  onSelectedNamesChange,
  refreshToken,
}: DatabaseTableProps) {
  const { t } = useTranslation();
  const databaseStore = useDatabaseV1Store();
  const actuatorStore = useActuatorV1Store();

  const [databases, setDatabases] = useState<Database[]>([]);
  const [loading, setLoading] = useState(true);
  const nextPageTokenRef = useRef("");
  const [hasMore, setHasMore] = useState(false);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [pageSize, setPageSize] = useSessionPageSize("bb.databases-table");
  const fetchIdRef = useRef(0);

  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");

  const orderBy = sortKey ? `${sortKey} ${sortOrder}` : "";

  const toggleSort = useCallback(
    (key: string) => {
      if (sortKey === key) {
        if (sortOrder === "asc") setSortOrder("desc");
        else {
          setSortKey(null);
          setSortOrder("asc");
        }
      } else {
        setSortKey(key);
        setSortOrder("asc");
      }
    },
    [sortKey, sortOrder]
  );

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

  const showSelection = !!selectedNames && !!onSelectedNamesChange;

  const toggleSelection = useCallback(
    (name: string) => {
      if (!selectedNames || !onSelectedNamesChange) return;
      const next = new Set(selectedNames);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      onSelectedNamesChange(next);
    },
    [selectedNames, onSelectedNamesChange]
  );

  const toggleSelectAll = useCallback(() => {
    if (!selectedNames || !onSelectedNamesChange) return;
    if (selectedNames.size === databases.length) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(databases.map((db) => db.name)));
    }
  }, [databases, selectedNames, onSelectedNamesChange]);

  const handleRowClick = useCallback((db: Database, e: React.MouseEvent) => {
    const url = router.resolve(autoDatabaseRoute(db)).fullPath;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }, []);

  const allSelected =
    databases.length > 0 && (selectedNames?.size ?? 0) === databases.length;
  const someSelected =
    (selectedNames?.size ?? 0) > 0 &&
    (selectedNames?.size ?? 0) < databases.length;
  const headerCheckboxRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (headerCheckboxRef.current) {
      headerCheckboxRef.current.indeterminate = someSelected;
    }
  }, [someSelected]);

  const pageSizeOptions = getPageSizeOptions();
  const showProjectColumn = mode === "ALL";

  const columns: DatabaseColumn[] = [];
  if (showSelection) {
    columns.push({
      key: "select",
      title: (
        <input
          ref={headerCheckboxRef}
          type="checkbox"
          checked={allSelected}
          onChange={toggleSelectAll}
          className="rounded-xs border-control-border"
        />
      ),
      defaultWidth: 48,
      render: (db) => (
        <input
          type="checkbox"
          checked={selectedNames?.has(db.name) ?? false}
          onChange={() => toggleSelection(db.name)}
          onClick={(e) => e.stopPropagation()}
          className="rounded-xs border-control-border"
        />
      ),
    });
  }
  columns.push({
    key: "name",
    title: t("common.name"),
    defaultWidth: 280,
    minWidth: 160,
    resizable: true,
    sortable: true,
    sortKey: "name",
    render: (db) => {
      const instanceResource = getInstanceResource(db);
      return (
        <div className="flex items-center gap-x-2">
          <EngineIcon engine={instanceResource.engine} className="h-5 w-5" />
          <span className="truncate">
            {extractDatabaseResourceName(db.name).databaseName}
          </span>
        </div>
      );
    },
  });
  columns.push({
    key: "environment",
    title: t("common.environment"),
    defaultWidth: 200,
    minWidth: 120,
    resizable: true,
    render: (db) => (
      <EnvironmentLabel environmentName={getDatabaseEnvironment(db).name} />
    ),
  });
  if (showProjectColumn) {
    columns.push({
      key: "project",
      title: t("common.project"),
      defaultWidth: 200,
      minWidth: 120,
      resizable: true,
      sortable: true,
      sortKey: "project",
      render: (db) => (
        <span className="truncate">
          {extractProjectResourceName(getDatabaseProject(db).name)}
        </span>
      ),
    });
  } else {
    columns.push({
      key: "release",
      title: t("common.release"),
      defaultWidth: 140,
      minWidth: 80,
      resizable: true,
      render: (db) => (
        <span className="truncate">
          {db.release ? extractReleaseUID(db.release) : "-"}
        </span>
      ),
    });
  }
  columns.push({
    key: "instance",
    title: t("common.instance"),
    defaultWidth: 240,
    minWidth: 120,
    resizable: true,
    sortable: true,
    sortKey: "instance",
    render: (db) => (
      <span className="block truncate">{getInstanceResource(db).title}</span>
    ),
  });
  columns.push({
    key: "address",
    title: t("common.address"),
    defaultWidth: 240,
    minWidth: 150,
    resizable: true,
    render: (db) => (
      <span className="truncate">
        {hostPortOfInstanceV1(getInstanceResource(db))}
      </span>
    ),
  });
  columns.push({
    key: "labels",
    title: t("common.labels"),
    defaultWidth: 240,
    minWidth: 150,
    resizable: true,
    render: (db) => <LabelsDisplay labels={db.labels} />,
  });
  columns.push({
    key: "status",
    title: t("common.status"),
    defaultWidth: 80,
    cellClassName: "whitespace-nowrap",
    render: (db) =>
      db.syncStatus === SyncStatus.FAILED ? (
        <span title={db.syncError || t("database.sync-status-failed")}>
          <XCircle className="w-4 h-4 text-error" />
        </span>
      ) : (
        <CheckCircle className="w-4 h-4 text-success" />
      ),
  });

  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  return (
    <>
      <div className="border rounded-sm">
        <div className="overflow-x-auto">
          <Table className="table-fixed" style={{ width: `${totalWidth}px` }}>
            <colgroup>
              {widths.map((w, i) => (
                <col key={columns[i].key} style={{ width: `${w}px` }} />
              ))}
            </colgroup>
            <TableHeader>
              <TableRow className="bg-gray-50">
                {columns.map((col, colIdx) => (
                  <TableHead
                    key={col.key}
                    sortable={col.sortable}
                    sortActive={
                      col.sortable && sortKey === (col.sortKey ?? col.key)
                    }
                    sortDir={sortOrder}
                    onSort={
                      col.sortable
                        ? () => toggleSort(col.sortKey ?? col.key)
                        : undefined
                    }
                    resizable={col.resizable}
                    onResizeStart={
                      col.resizable
                        ? (e) => onResizeStart(colIdx, e)
                        : undefined
                    }
                  >
                    {col.title}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && databases.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="py-8 text-center text-control-placeholder"
                  >
                    <div className="flex items-center justify-center gap-x-2">
                      <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                      {t("common.loading")}
                    </div>
                  </TableCell>
                </TableRow>
              ) : databases.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columns.length}
                    className="py-8 text-center text-control-placeholder"
                  >
                    {t("common.no-data")}
                  </TableCell>
                </TableRow>
              ) : (
                databases.map((db) => (
                  <TableRow
                    key={db.name}
                    className="cursor-pointer"
                    onClick={(e) => handleRowClick(db, e)}
                  >
                    {columns.map((col) => (
                      <TableCell
                        key={col.key}
                        className={cn("overflow-hidden", col.cellClassName)}
                      >
                        {col.render(db)}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="mx-2">
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
