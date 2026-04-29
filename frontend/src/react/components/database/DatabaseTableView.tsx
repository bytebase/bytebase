import { CheckCircle, XCircle } from "lucide-react";
import { useCallback, useEffect, useRef } from "react";
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
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
  getDatabaseEnvironment,
  getDatabaseProject,
  getInstanceResource,
  hostPortOfInstanceV1,
} from "@/utils";
import { extractReleaseUID } from "@/utils/v1/release";

export type DatabaseTableMode = "ALL" | "PROJECT";

export type DatabaseTableSort = {
  key: "name" | "project" | "instance";
  order: "asc" | "desc";
};

interface DatabaseTableViewProps {
  databases: Database[];
  mode?: DatabaseTableMode;
  loading?: boolean;
  /** Selection — pass both to enable the checkbox column. */
  selectedNames?: Set<string>;
  onSelectedNamesChange?: (names: Set<string>) => void;
  /** Sort state + handler — pass both to enable sortable headers. */
  sort?: DatabaseTableSort | null;
  onSortChange?: (sort: DatabaseTableSort | null) => void;
  /** Row click handler. Not wired by default — pure UI doesn't navigate. */
  onRowClick?: (db: Database, e: React.MouseEvent) => void;
}

/**
 * Pure presentational table — given a list of databases, renders the
 * standard SQL Editor / settings database table layout (engine icon,
 * environment, project/release, instance, address, labels, sync status).
 *
 * No data fetching, no pagination footer, no router navigation. Callers
 * that need those compose this view inside their own wrapper:
 *
 *  - `DatabaseTable.tsx` — server-fetch wrapper used by settings/project
 *    pages. Owns paging + filter + sort, threads them through this view.
 *  - SQL Editor `BatchQuerySelect.tsx` — already has the database list,
 *    renders this view directly with a Set-based selection model.
 */
export function DatabaseTableView({
  databases,
  mode = "ALL",
  loading = false,
  selectedNames,
  onSelectedNamesChange,
  sort,
  onSortChange,
  onRowClick,
}: DatabaseTableViewProps) {
  const { t } = useTranslation();

  const showSelection = !!selectedNames && !!onSelectedNamesChange;
  const showProjectColumn = mode === "ALL";
  const sortable = !!onSortChange;

  const toggleSort = useCallback(
    (key: DatabaseTableSort["key"]) => {
      if (!onSortChange) return;
      if (sort?.key === key) {
        if (sort.order === "asc") {
          onSortChange({ key, order: "desc" });
        } else {
          onSortChange(null);
        }
      } else {
        onSortChange({ key, order: "asc" });
      }
    },
    [sort, onSortChange]
  );

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

  return (
    <div className="border rounded-sm">
      <div className="overflow-x-auto">
        <Table className="min-w-[800px]">
          <TableHeader>
            <TableRow className="bg-gray-50">
              {showSelection && (
                <TableHead className="w-12">
                  <input
                    ref={headerCheckboxRef}
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    className="rounded-xs border-control-border"
                  />
                </TableHead>
              )}
              <TableHead
                sortable={sortable}
                sortActive={sort?.key === "name"}
                sortDir={sort?.order ?? "asc"}
                onSort={sortable ? () => toggleSort("name") : undefined}
              >
                {t("common.name")}
              </TableHead>
              <TableHead>{t("common.environment")}</TableHead>
              {!showProjectColumn && (
                <TableHead>{t("common.release")}</TableHead>
              )}
              {showProjectColumn && (
                <TableHead
                  sortable={sortable}
                  sortActive={sort?.key === "project"}
                  sortDir={sort?.order ?? "asc"}
                  onSort={sortable ? () => toggleSort("project") : undefined}
                >
                  {t("common.project")}
                </TableHead>
              )}
              <TableHead
                sortable={sortable}
                sortActive={sort?.key === "instance"}
                sortDir={sort?.order ?? "asc"}
                onSort={sortable ? () => toggleSort("instance") : undefined}
              >
                {t("common.instance")}
              </TableHead>
              <TableHead className="hidden md:table-cell">
                {t("common.address")}
              </TableHead>
              <TableHead className="hidden md:table-cell">
                {t("common.labels")}
              </TableHead>
              <TableHead className="whitespace-nowrap">
                {t("common.status")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading && databases.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={8}
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
                  colSpan={8}
                  className="py-8 text-center text-control-placeholder"
                >
                  {t("common.no-data")}
                </TableCell>
              </TableRow>
            ) : (
              databases.map((db) => {
                const isSelected = selectedNames?.has(db.name) ?? false;
                const instanceResource = getInstanceResource(db);
                return (
                  <TableRow
                    key={db.name}
                    className={onRowClick ? "cursor-pointer" : undefined}
                    onClick={onRowClick ? (e) => onRowClick(db, e) : undefined}
                  >
                    {showSelection && (
                      <TableCell className="w-12">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleSelection(db.name)}
                          onClick={(e) => e.stopPropagation()}
                          className="rounded-xs border-control-border"
                        />
                      </TableCell>
                    )}
                    <TableCell>
                      <div className="flex items-center gap-x-2">
                        <EngineIcon
                          engine={instanceResource.engine}
                          className="h-5 w-5"
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
                    {!showProjectColumn && (
                      <TableCell>
                        <span className="truncate">
                          {db.release ? extractReleaseUID(db.release) : "-"}
                        </span>
                      </TableCell>
                    )}
                    {showProjectColumn && (
                      <TableCell>
                        <span className="truncate">
                          {extractProjectResourceName(
                            getDatabaseProject(db).name
                          )}
                        </span>
                      </TableCell>
                    )}
                    <TableCell className="max-w-[200px]">
                      <span className="block truncate">
                        {instanceResource.title}
                      </span>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      <span className="truncate">
                        {hostPortOfInstanceV1(instanceResource)}
                      </span>
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
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
    </div>
  );
}
