import { CheckCircle, XCircle } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentLabel } from "@/react/components/EnvironmentLabel";
import { LabelsDisplay } from "@/react/components/LabelsDisplay";
import { Checkbox } from "@/react/components/ui/checkbox";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { cn } from "@/react/lib/utils";
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

interface DatabaseColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  sortable?: boolean;
  sortKey?: DatabaseTableSort["key"];
  cellClassName?: string;
  render: (database: Database) => React.ReactNode;
  onCellClick?: (database: Database, e: React.MouseEvent) => void;
  onHeaderClick?: (e: React.MouseEvent) => void;
}

/**
 * Pure presentational table — given a list of databases, renders the
 * standard SQL Editor / settings database table layout with resizable
 * columns (engine icon, environment, project/release, instance,
 * address, labels, sync status).
 *
 * No data fetching, no pagination footer, no router navigation. Callers
 * that need those compose this view inside their own wrapper:
 *
 *  - `DatabaseTable.tsx` — server-fetch wrapper used by settings/project
 *    pages. Owns paging + filter + sort, threads them through this view.
 *  - SQL Editor `BatchQuerySelect.tsx` — already has the database list,
 *    renders this view directly with a Set-based selection model.
 *  - `TransferProjectSheet.tsx` — read-only summary of selected
 *    databases inside a sheet.
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

  const toggleSort = (key: DatabaseTableSort["key"]) => {
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
  };

  const toggleSelection = (name: string) => {
    if (!selectedNames || !onSelectedNamesChange) return;
    const next = new Set(selectedNames);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    onSelectedNamesChange(next);
  };

  const toggleSelectAll = () => {
    if (!selectedNames || !onSelectedNamesChange) return;
    if (selectedNames.size === databases.length) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(databases.map((db) => db.name)));
    }
  };

  const allSelected =
    databases.length > 0 && (selectedNames?.size ?? 0) === databases.length;
  const someSelected =
    (selectedNames?.size ?? 0) > 0 &&
    (selectedNames?.size ?? 0) < databases.length;
  const columns = useMemo<DatabaseColumn[]>(() => {
    const cols: DatabaseColumn[] = [];
    if (showSelection) {
      cols.push({
        key: "select",
        title: (
          <Checkbox
            checked={someSelected ? "indeterminate" : allSelected}
            onCheckedChange={toggleSelectAll}
            onClick={(e) => e.stopPropagation()}
          />
        ),
        defaultWidth: 48,
        onCellClick: (db, e) => {
          e.stopPropagation();
          toggleSelection(db.name);
        },
        onHeaderClick: (e) => {
          e.stopPropagation();
          toggleSelectAll();
        },
        render: (db) => (
          <Checkbox
            checked={selectedNames?.has(db.name) ?? false}
            onCheckedChange={() => toggleSelection(db.name)}
            onClick={(e) => e.stopPropagation()}
          />
        ),
      });
    }
    cols.push({
      key: "name",
      title: t("common.name"),
      defaultWidth: 280,
      minWidth: 160,
      resizable: true,
      sortable: true,
      sortKey: "name",
      render: (db) => {
        const instanceResource = getInstanceResource(db);
        const databaseName = extractDatabaseResourceName(db.name).databaseName;
        return (
          <div className="flex items-center gap-x-2 min-w-0">
            <EngineIcon engine={instanceResource.engine} className="h-5 w-5" />
            <EllipsisText text={databaseName} className="min-w-0 flex-1" />
          </div>
        );
      },
    });
    cols.push({
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
      cols.push({
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
      cols.push({
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
    cols.push({
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
    cols.push({
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
    cols.push({
      key: "labels",
      title: t("common.labels"),
      defaultWidth: 240,
      minWidth: 150,
      resizable: true,
      render: (db) => <LabelsDisplay labels={db.labels} />,
    });
    cols.push({
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
    return cols;
    // `databases` and `onSelectedNamesChange` belong here because the
    // header checkbox's `toggleSelectAll` closes over both. Pagination /
    // filtering can replace `databases` while `allSelected` stays false —
    // without listing it the memoized handler keeps the previous page's
    // names and "select all" selects the wrong rows.
  }, [
    showSelection,
    showProjectColumn,
    allSelected,
    selectedNames,
    onSelectedNamesChange,
    databases,
    t,
  ]);

  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  return (
    <div className="overflow-x-auto border-y border-block-border">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow>
            {columns.map((col, colIdx) => {
              const colSortKey = col.sortKey;
              const sortActive = Boolean(
                col.sortable && colSortKey && sort?.key === colSortKey
              );
              return (
                <TableHead
                  key={col.key}
                  sortable={col.sortable && sortable}
                  sortActive={sortActive}
                  sortDir={sort?.order ?? "asc"}
                  onSort={
                    col.sortable && sortable && colSortKey
                      ? () => toggleSort(colSortKey)
                      : undefined
                  }
                  resizable={col.resizable}
                  onResizeStart={
                    col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
                  }
                  className={cn(col.onHeaderClick && "cursor-pointer")}
                  onClick={col.onHeaderClick}
                >
                  {col.title}
                </TableHead>
              );
            })}
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
                className={onRowClick ? "cursor-pointer" : undefined}
                onClick={onRowClick ? (e) => onRowClick(db, e) : undefined}
              >
                {columns.map((col) => (
                  <TableCell
                    key={col.key}
                    className={cn(
                      "overflow-hidden",
                      col.cellClassName,
                      col.onCellClick && "cursor-pointer"
                    )}
                    onClick={
                      col.onCellClick
                        ? (e) => col.onCellClick!(db, e)
                        : undefined
                    }
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
  );
}
