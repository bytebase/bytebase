import { CheckCircle, ChevronDown, XCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EngineIconPath } from "@/components/InstanceForm/constants";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LabelsDisplay } from "@/react/components/LabelsDisplay";
import { Button } from "@/react/components/ui/button";
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
  const showProjectColumn = mode === "ALL";

  return (
    <>
      <table className="w-full text-sm">
        <thead>
          <tr className="bg-gray-50 border-b border-control-border">
            <th className="w-12 px-4 py-2">
              <input
                ref={headerCheckboxRef}
                type="checkbox"
                checked={allSelected}
                onChange={toggleSelectAll}
                className="rounded border-control-border"
              />
            </th>
            <th
              className="px-4 py-2 text-left font-medium min-w-[200px] cursor-pointer select-none"
              onClick={() => toggleSort("name")}
            >
              <div className="flex items-center gap-x-1">
                {t("common.name")}
                {renderSortIndicator("name")}
              </div>
            </th>
            <th className="px-4 py-2 text-left font-medium">
              {t("common.environment")}
            </th>
            {showProjectColumn && (
              <th
                className="px-4 py-2 text-left font-medium cursor-pointer select-none"
                onClick={() => toggleSort("project")}
              >
                <div className="flex items-center gap-x-1">
                  {t("common.project")}
                  {renderSortIndicator("project")}
                </div>
              </th>
            )}
            <th
              className="px-4 py-2 text-left font-medium cursor-pointer select-none"
              onClick={() => toggleSort("instance")}
            >
              <div className="flex items-center gap-x-1">
                {t("common.instance")}
                {renderSortIndicator("instance")}
              </div>
            </th>
            <th className="px-4 py-2 text-left font-medium hidden md:table-cell">
              {t("common.address")}
            </th>
            <th className="px-4 py-2 text-left font-medium min-w-[240px] hidden md:table-cell">
              {t("common.labels")}
            </th>
            <th className="px-4 py-2 text-left font-medium whitespace-nowrap">
              {t("common.status")}
            </th>
          </tr>
        </thead>
        <tbody>
          {loading && databases.length === 0 ? (
            <tr>
              <td
                colSpan={showProjectColumn ? 8 : 7}
                className="px-4 py-8 text-center text-control-placeholder"
              >
                <div className="flex items-center justify-center gap-x-2">
                  <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                  {t("common.loading")}
                </div>
              </td>
            </tr>
          ) : databases.length === 0 ? (
            <tr>
              <td
                colSpan={showProjectColumn ? 8 : 7}
                className="px-4 py-8 text-center text-control-placeholder"
              >
                {t("common.no-data")}
              </td>
            </tr>
          ) : (
            databases.map((db, i) => {
              const isSelected = selectedNames.has(db.name);
              const instanceResource = getInstanceResource(db);
              return (
                <tr
                  key={db.name}
                  className={cn(
                    "border-b last:border-b-0 cursor-pointer hover:bg-gray-50",
                    i % 2 === 1 && "bg-gray-50/50"
                  )}
                  onClick={(e) => handleRowClick(db, e)}
                >
                  <td className="w-12 px-4 py-2">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={() => toggleSelection(db.name)}
                      onClick={(e) => e.stopPropagation()}
                      className="rounded border-control-border"
                    />
                  </td>
                  <td className="px-4 py-2">
                    <div className="flex items-center gap-x-2">
                      <img
                        className="h-5 w-5"
                        src={EngineIconPath[instanceResource.engine]}
                        alt=""
                      />
                      <span className="truncate">
                        {extractDatabaseResourceName(db.name).databaseName}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-2">
                    <EnvironmentLabel
                      environmentName={getDatabaseEnvironment(db).name}
                    />
                  </td>
                  {showProjectColumn && (
                    <td className="px-4 py-2">
                      <span className="truncate">
                        {extractProjectResourceName(
                          getDatabaseProject(db).name
                        )}
                      </span>
                    </td>
                  )}
                  <td className="px-4 py-2">
                    <span className="truncate">{instanceResource.title}</span>
                  </td>
                  <td className="px-4 py-2 hidden md:table-cell">
                    <span className="truncate">
                      {hostPortOfInstanceV1(instanceResource)}
                    </span>
                  </td>
                  <td className="px-4 py-2 hidden md:table-cell">
                    <LabelsDisplay labels={db.labels} />
                  </td>
                  <td className="px-4 py-2">
                    {db.syncStatus === SyncStatus.FAILED ? (
                      <span
                        title={db.syncError || t("database.sync-status-failed")}
                      >
                        <XCircle className="w-4 h-4 text-error" />
                      </span>
                    ) : (
                      <CheckCircle className="w-4 h-4 text-success" />
                    )}
                  </td>
                </tr>
              );
            })
          )}
        </tbody>
      </table>

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
