import { CheckCircle, XCircle } from "lucide-react";
import { memo, useCallback, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { EnvironmentBadge } from "@/react/components/EnvironmentLabel";
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
import { useEnvironmentList, usePlanFeature } from "@/react/hooks/useAppState";
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
import { cn } from "@/react/lib/utils";
import type { Environment } from "@/types";
import {
  isValidEnvironmentName,
  nullEnvironment,
  unknownEnvironment,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
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

  // Lift environment + feature lookups to the table level so the per-row
  // env cell doesn't subscribe to Zustand or run a `.find()` per render.
  // Without this, a 500-row table registers 500 redundant subscriptions
  // to `environmentList` + `hasFeature` and re-runs `loadSubscription`
  // 500 times. The Map below collapses lookups to O(1) per row.
  const environmentList = useEnvironmentList();
  const environmentByName = useMemo(() => {
    const map = new Map<string, Environment>();
    for (const env of environmentList) {
      map.set(env.name, env);
    }
    return map;
  }, [environmentList]);
  const hasEnvTierFeature = usePlanFeature(
    PlanFeature.FEATURE_ENVIRONMENT_TIERS
  );
  const resolveEnvironment = useCallback(
    (name: string | undefined): Environment => {
      if (!name) return nullEnvironment();
      const env = environmentByName.get(name);
      if (env) return env;
      if (!isValidEnvironmentName(name)) return unknownEnvironment();
      const id = name.replace(/^environments\//, "");
      return { ...unknownEnvironment(), id, name, title: id };
    },
    [environmentByName]
  );

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

  // Refs so per-row handlers stay referentially stable across selection
  // changes — without this, every selection toggle re-creates the closure
  // and `React.memo` on `DatabaseRowView` is defeated for every row.
  const selectedNamesRef = useRef(selectedNames);
  selectedNamesRef.current = selectedNames;
  const onSelectedNamesChangeRef = useRef(onSelectedNamesChange);
  onSelectedNamesChangeRef.current = onSelectedNamesChange;
  const databasesRef = useRef(databases);
  databasesRef.current = databases;

  const toggleSelection = useCallback((name: string) => {
    const current = selectedNamesRef.current;
    const cb = onSelectedNamesChangeRef.current;
    if (!current || !cb) return;
    const next = new Set(current);
    if (next.has(name)) next.delete(name);
    else next.add(name);
    cb(next);
  }, []);

  const toggleSelectAll = useCallback(() => {
    const current = selectedNamesRef.current;
    const cb = onSelectedNamesChangeRef.current;
    const dbs = databasesRef.current;
    if (!current || !cb) return;
    if (current.size === dbs.length) {
      cb(new Set());
    } else {
      cb(new Set(dbs.map((db) => db.name)));
    }
  }, []);

  const allSelected =
    databases.length > 0 && (selectedNames?.size ?? 0) === databases.length;
  const someSelected =
    (selectedNames?.size ?? 0) > 0 &&
    (selectedNames?.size ?? 0) < databases.length;

  // `columns` is intentionally NOT a function of `selectedNames` /
  // `databases` so it stays referentially stable across selection toggles
  // and pagination. The select column used to live here and dragged
  // `selectedNames` into the deps, which invalidated `columns` on every
  // toggle and forced every row to re-render. The leading checkbox is now
  // rendered separately in the row + header so columns can stay stable.
  const columns = useMemo<DatabaseColumn[]>(() => {
    const cols: DatabaseColumn[] = [];
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
        <EnvironmentBadge
          environment={resolveEnvironment(db.effectiveEnvironment)}
          hasEnvTierFeature={hasEnvTierFeature}
        />
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
  }, [showProjectColumn, t, resolveEnvironment, hasEnvTierFeature]);

  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  const totalColumnCount = columns.length + (showSelection ? 1 : 0);

  return (
    <div className="overflow-x-auto border rounded-sm">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {showSelection && <col style={{ width: "48px" }} />}
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {showSelection && (
              <TableHead
                className="cursor-pointer"
                onClick={(e) => {
                  e.stopPropagation();
                  toggleSelectAll();
                }}
              >
                <Checkbox
                  checked={someSelected ? "indeterminate" : allSelected}
                  onCheckedChange={toggleSelectAll}
                  onClick={(e) => e.stopPropagation()}
                />
              </TableHead>
            )}
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
                colSpan={totalColumnCount}
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
                colSpan={totalColumnCount}
                className="py-8 text-center text-control-placeholder"
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          ) : (
            databases.map((db) => (
              <DatabaseRowView
                key={db.name}
                database={db}
                columns={columns}
                showSelection={showSelection}
                selected={selectedNames?.has(db.name) ?? false}
                onToggleSelection={toggleSelection}
                onRowClick={onRowClick}
              />
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

interface DatabaseRowViewProps {
  database: Database;
  columns: DatabaseColumn[];
  showSelection: boolean;
  /** Primitive boolean — keeps memo equality cheap. Only flips for the
   *  row whose selection changed, so other rows skip re-rendering even
   *  when the parent re-runs (e.g. selection toggles, pagination loads). */
  selected: boolean;
  /** Stable per-render thanks to `useCallback` upstream. Wrapped via
   *  closure inside the row so the Checkbox can call it with the row's
   *  `database.name` without forcing the row to depend on `selectedNames`. */
  onToggleSelection: (name: string) => void;
  onRowClick?: (db: Database, e: React.MouseEvent) => void;
}

/**
 * Memoized per-row renderer. Keeps large database tables (hundreds of
 * rows) from re-running every cell's per-row helpers (`getInstanceResource`,
 * `getDatabaseEnvironment`, `getDatabaseProject`, etc.) on every parent
 * re-render. Only props on this list trigger a re-render of a given row.
 */
const DatabaseRowView = memo(function DatabaseRowView({
  database,
  columns,
  showSelection,
  selected,
  onToggleSelection,
  onRowClick,
}: DatabaseRowViewProps) {
  return (
    <TableRow
      className={onRowClick ? "cursor-pointer" : undefined}
      onClick={onRowClick ? (e) => onRowClick(database, e) : undefined}
    >
      {showSelection && (
        <TableCell
          className="overflow-hidden cursor-pointer"
          onClick={(e) => {
            e.stopPropagation();
            onToggleSelection(database.name);
          }}
        >
          <Checkbox
            checked={selected}
            onCheckedChange={() => onToggleSelection(database.name)}
            onClick={(e) => e.stopPropagation()}
          />
        </TableCell>
      )}
      {columns.map((col) => (
        <TableCell
          key={col.key}
          className={cn(
            "overflow-hidden",
            col.cellClassName,
            col.onCellClick && "cursor-pointer"
          )}
          onClick={
            col.onCellClick ? (e) => col.onCellClick!(database, e) : undefined
          }
        >
          {col.render(database)}
        </TableCell>
      ))}
    </TableRow>
  );
});
