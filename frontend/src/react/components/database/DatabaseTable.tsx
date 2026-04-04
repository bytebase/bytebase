import { CheckCircle, ChevronDown, XCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LabelsDisplay } from "@/react/components/LabelsDisplay";
import { Button } from "@/react/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type { ColumnDef } from "@/react/hooks/useColumnWidths";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
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

export type DatabaseTableMode = "ALL" | "PROJECT";

export interface DatabaseTableProps {
  filter: DatabaseFilter;
  parent?: string;
  mode?: DatabaseTableMode;
  selectedNames: Set<string>;
  onSelectedNamesChange: (names: Set<string>) => void;
  refreshToken?: number;
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
  const showProjectColumn = mode === "ALL";

  const columns: ColumnDef[] = useMemo(() => {
    const cols: ColumnDef[] = [
      { key: "checkbox", defaultWidth: 48, minWidth: 48, resizable: false },
      { key: "name", defaultWidth: 200, minWidth: 120 },
      { key: "environment", defaultWidth: 140, minWidth: 80 },
    ];
    if (showProjectColumn) {
      cols.push({ key: "project", defaultWidth: 150, minWidth: 80 });
    }
    cols.push(
      { key: "instance", defaultWidth: 160, minWidth: 80 },
      { key: "address", defaultWidth: 160, minWidth: 80 },
      { key: "labels", defaultWidth: 240, minWidth: 100 },
      { key: "status", defaultWidth: 70, minWidth: 50, resizable: false }
    );
    return cols;
  }, [showProjectColumn]);

  const { widths, totalWidth, onResizeStart } = useColumnWidths(
    columns,
    `bb.database-table-widths.${mode}`
  );

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

  const toggleSelection = useCallback(
    (name: string) => {
      const next = new Set(selectedNames);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      onSelectedNamesChange(next);
    },
    [selectedNames, onSelectedNamesChange]
  );

  const toggleSelectAll = useCallback(() => {
    if (selectedNames.size === databases.length) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(databases.map((db) => db.name)));
    }
  }, [databases, selectedNames.size, onSelectedNamesChange]);

  const handleRowClick = useCallback((db: Database, e: React.MouseEvent) => {
    const url = router.resolve(autoDatabaseRoute(db)).fullPath;
    if (e.ctrlKey || e.metaKey) {
      window.open(url, "_blank");
    } else {
      router.push(url);
    }
  }, []);

  const renderSortIndicator = (columnKey: string) => {
    if (sortKey !== columnKey) {
      return <ChevronDown className="h-3 w-3 text-gray-300" />;
    }
    return (
      <ChevronDown
        className={cn(
          "h-3 w-3 text-accent transition-transform",
          sortOrder === "asc" && "rotate-180"
        )}
      />
    );
  };

  const allSelected =
    databases.length > 0 && selectedNames.size === databases.length;
  const someSelected =
    selectedNames.size > 0 && selectedNames.size < databases.length;
  const headerCheckboxRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (headerCheckboxRef.current) {
      headerCheckboxRef.current.indeterminate = someSelected;
    }
  }, [someSelected]);

  const pageSizeOptions = getPageSizeOptions();
  const colCount = columns.length;

  return (
    <>
      <div className="overflow-x-auto">
        <Table style={{ width: `${totalWidth}px` }}>
          <colgroup>
            {widths.map((w, i) => (
              <col key={columns[i].key} style={{ width: `${w}px` }} />
            ))}
          </colgroup>
          <TableHeader>
            <TableRow className="bg-gray-50 border-b border-control-border">
              <TableHead>
                <input
                  ref={headerCheckboxRef}
                  type="checkbox"
                  checked={allSelected}
                  onChange={toggleSelectAll}
                  className="rounded border-control-border"
                />
              </TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => toggleSort("name")}
                resizable
                onResizeStart={(e) => onResizeStart(1, e)}
              >
                <div className="flex items-center gap-x-1">
                  {t("common.name")}
                  {renderSortIndicator("name")}
                </div>
              </TableHead>
              <TableHead resizable onResizeStart={(e) => onResizeStart(2, e)}>
                {t("common.environment")}
              </TableHead>
              {showProjectColumn && (
                <TableHead
                  className="cursor-pointer select-none"
                  onClick={() => toggleSort("project")}
                  resizable
                  onResizeStart={(e) => onResizeStart(3, e)}
                >
                  <div className="flex items-center gap-x-1">
                    {t("common.project")}
                    {renderSortIndicator("project")}
                  </div>
                </TableHead>
              )}
              <TableHead
                className="cursor-pointer select-none"
                onClick={() => toggleSort("instance")}
                resizable
                onResizeStart={(e) =>
                  onResizeStart(showProjectColumn ? 4 : 3, e)
                }
              >
                <div className="flex items-center gap-x-1">
                  {t("common.instance")}
                  {renderSortIndicator("instance")}
                </div>
              </TableHead>
              <TableHead
                resizable
                onResizeStart={(e) =>
                  onResizeStart(showProjectColumn ? 5 : 4, e)
                }
              >
                {t("common.address")}
              </TableHead>
              <TableHead
                resizable
                onResizeStart={(e) =>
                  onResizeStart(showProjectColumn ? 6 : 5, e)
                }
              >
                {t("common.labels")}
              </TableHead>
              <TableHead>{t("common.status")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && databases.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={colCount}
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
                  colSpan={colCount}
                  className="py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </TableCell>
              </TableRow>
            ) : (
              databases.map((db, i) => {
                const isSelected = selectedNames.has(db.name);
                const instanceResource = getInstanceResource(db);
                return (
                  <TableRow
                    key={db.name}
                    className={cn(
                      "cursor-pointer hover:bg-gray-50",
                      i % 2 === 1 && "bg-gray-50/50"
                    )}
                    onClick={(e) => handleRowClick(db, e)}
                  >
                    <TableCell>
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleSelection(db.name)}
                        onClick={(e) => e.stopPropagation()}
                        className="rounded border-control-border"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-x-2">
                        <img
                          className="h-5 w-5 shrink-0"
                          src={EngineIconPath[instanceResource.engine]}
                          alt=""
                        />
                        <span className="truncate">
                          {extractDatabaseResourceName(db.name).databaseName}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <EnvironmentLabel
                        environmentName={getDatabaseEnvironment(db).name}
                      />
                    </TableCell>
                    {showProjectColumn && (
                      <TableCell>
                        <span className="truncate">
                          {extractProjectResourceName(
                            getDatabaseProject(db).name
                          )}
                        </span>
                      </TableCell>
                    )}
                    <TableCell>
                      <span className="truncate">{instanceResource.title}</span>
                    </TableCell>
                    <TableCell>
                      <span className="truncate">
                        {hostPortOfInstanceV1(instanceResource)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <LabelsDisplay labels={db.labels} />
                    </TableCell>
                    <TableCell>
                      {db.syncStatus === SyncStatus.FAILED ? (
                        <span
                          title={
                            db.syncError || t("database.sync-status-failed")
                          }
                        >
                          <XCircle className="w-4 h-4 text-error" />
                        </span>
                      ) : (
                        <CheckCircle className="w-4 h-4 text-success" />
                      )}
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-end gap-x-2 mx-4">
        <div className="flex items-center gap-x-2">
          <span className="text-sm text-control-light">
            {t("common.rows-per-page")}
          </span>
          <select
            className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1 min-w-[5rem]"
            value={pageSize}
            onChange={(e) => setPageSize(Number(e.target.value))}
          >
            {pageSizeOptions.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </div>
        {hasMore && (
          <Button
            variant="ghost"
            size="sm"
            disabled={isFetchingMore}
            onClick={loadMore}
          >
            <span className="text-sm text-control-light">
              {isFetchingMore ? t("common.loading") : t("common.load-more")}
            </span>
          </Button>
        )}
      </div>
    </>
  );
}
