import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { head } from "lodash-es";
import { CircleAlert, Eye, EyeOff, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DataExportButton,
  type DataExportRequest,
  type DownloadContent,
} from "@/react/components/DataExportButton";
import { DatabaseTableView } from "@/react/components/database";
import { EngineIconPath } from "@/react/components/instance/constants";
import { RequestExportButton } from "@/react/components/sql-editor/RequestExportButton";
import { useExportGrantBypass } from "@/react/components/sql-editor/useExportGrantBypass";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useSQLEditorQueryDataPolicy } from "@/react/hooks/useSQLEditorBridge";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useCurrentSQLEditorTab,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { ExportRequestSchema } from "@/types/proto-es/v1/sql_service_pb";
import type { SQLEditorDatabaseQueryContext } from "@/types/sqlEditor/tab";
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
  hexToRgb,
} from "@/utils";
import { TabContextMenu } from "./ContextMenu";
import { type CloseTabAction, resultTabEvents } from "./resultTabContext";

const MAX_EXPORT = 20;

type Props = {
  readonly selectedDatabase: Database | undefined;
  readonly onSelectedDatabaseChange: (db: Database | undefined) => void;
};

type BatchQueryItem = {
  database: Database;
  context: SQLEditorDatabaseQueryContext | undefined;
};

const isEmptyQueryItem = (item: BatchQueryItem) => {
  if (!item.context) return true;
  if (item.context.resultSet?.error) return false;
  if (item.context.status !== "DONE") return false;
  return (
    item.context.resultSet?.results.every(
      (result) => result.rows.length === 0
    ) ?? true
  );
};

const isDatabaseQueryFailed = (item: BatchQueryItem) =>
  Boolean(
    item.context?.resultSet?.error ||
      item.context?.resultSet?.results.find((result) => result.error)
  );

/**
 * Replaces `frontend/src/views/sql-editor/EditorPanel/ResultPanel/BatchQuerySelect.vue`.
 *
 * Renders the batch-query tab strip above the result panel: one tab per
 * queried database, with environment-tinted backgrounds, an empty-results
 * toggle, a batch-export drawer, and a right-click context menu (close /
 * close others / close to right / close all).
 */
export function BatchQuerySelect({
  selectedDatabase,
  onSelectedDatabaseChange,
}: Props) {
  const { t } = useTranslation();
  const getDatabaseByName = useAppStore((s) => s.getDatabaseByName);
  const exportData = useAppStore((s) => s.exportData);
  const currentTab = useCurrentSQLEditorTab();
  const project = useSQLEditorEditorState((s) => s.project);
  const queryDataPolicy = useSQLEditorQueryDataPolicy(project);

  const [showEmpty, setShowEmpty] = useState(true);
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());

  // Subscribe to the current tab's `databaseQueryContexts` Map. Immer
  // produces a fresh Map (and inner arrays) on every mutation, so the
  // selector re-runs whenever `preExecute` adds a context or `runQuery`
  // flips a status / writes a resultSet.
  const databaseQueryContexts = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.databaseQueryContexts
  );

  const queriedDatabaseNames = useMemo(
    () => Array.from(databaseQueryContexts?.keys() || []),
    [databaseQueryContexts]
  );

  const contextsByDatabase = useMemo(() => {
    const map = new Map<string, SQLEditorDatabaseQueryContext | undefined>();
    if (!databaseQueryContexts) return map;
    for (const name of databaseQueryContexts.keys()) {
      map.set(name, head(databaseQueryContexts.get(name)));
    }
    return map;
  }, [databaseQueryContexts]);

  const items = useMemo<BatchQueryItem[]>(() => {
    return queriedDatabaseNames.map((name) => ({
      database: getDatabaseByName(name),
      context: contextsByDatabase.get(name),
    }));
  }, [queriedDatabaseNames, contextsByDatabase, getDatabaseByName]);

  const showEmptySwitch = useMemo(
    () => items.length > 1 && items.some((item) => isEmptyQueryItem(item)),
    [items]
  );

  const filteredItems = useMemo(() => {
    if (showEmpty || !showEmptySwitch) return items;
    return items.filter((item) => !isEmptyQueryItem(item));
  }, [items, showEmpty, showEmptySwitch]);

  // Auto-select a proper database when the items list changes (mirrors
  // the Vue `watch(filteredItems, ..., { immediate: true })`).
  useEffect(() => {
    if (
      !selectedDatabase ||
      !filteredItems.find((item) => item.database === selectedDatabase)
    ) {
      onSelectedDatabaseChange(head(filteredItems)?.database);
    }
  }, [filteredItems, selectedDatabase, onSelectedDatabaseChange]);

  const supportFormats = useMemo(
    () => [
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ],
    []
  );

  // The batch was run from a single user-typed statement against many
  // databases — pull it from any one context (they all carry the same
  // params.statement). Used as the request-export drawer seed when the
  // policy disables direct export.
  const batchStatement = useMemo(() => {
    for (const context of contextsByDatabase.values()) {
      if (context?.params.statement) return context.params.statement;
    }
    return "";
  }, [contextsByDatabase]);

  // Per-DB grant lookup. When the policy disables direct export, the
  // hook reports which of the queried DBs are covered by an active
  // export grant (matched) and which are not (unmatched). Both lists
  // feed the partial-coverage UX: Export operates on the matched set
  // only, Request Export pre-seeds the unmatched set.
  const policyAllowsExport = !queryDataPolicy?.disableExport;
  const {
    matchedDatabases,
    unmatchedDatabases,
    tooltip: exportTooltip,
  } = useExportGrantBypass({
    enabled: !policyAllowsExport,
    project,
    statement: batchStatement,
    targets: queriedDatabaseNames,
  });

  // When the policy allows export, every queried DB is authorized;
  // otherwise only the JIT-matched subset. The Export drawer's
  // `DatabaseTableView` is filtered to this list so users can't
  // physically pick an unauthorized DB — eliminating the partial-
  // failure toast storm.
  const exportableDatabaseNames = policyAllowsExport
    ? queriedDatabaseNames
    : matchedDatabases;
  const exportableDatabaseList = useMemo(
    () => exportableDatabaseNames.map((name) => getDatabaseByName(name)),
    [exportableDatabaseNames, getDatabaseByName]
  );

  const showExport = exportableDatabaseNames.length > 0;
  // Surface "Request Export" whenever the batch contains uncovered DBs.
  // Visible alongside Export in the partial-coverage case (e.g. 2 of 5
  // covered) so users can request grants for the missing ones without
  // losing the immediate export of the covered set.
  const showRequestExport =
    !policyAllowsExport && unmatchedDatabases.length > 0;
  // Partial-coverage flag: some matched, some not. Drives the Export
  // button label change from "Batch export" to "Partial batch export"
  // so the user knows up-front the action operates on a strict subset.
  const isPartialExport =
    showExport && !policyAllowsExport && unmatchedDatabases.length > 0;

  const handleCloseSingleResultView = (item: BatchQueryItem) => {
    const contexts = currentTab?.databaseQueryContexts?.get(item.database.name);
    if (!contexts) return;
    for (const context of contexts) {
      context.abortController?.abort();
    }
    getSQLEditorTabsState().deleteDatabaseQueryContext(item.database.name);
  };

  // Subscribe to context-menu close-tab events.
  useEffect(() => {
    const unsubscribe = resultTabEvents.on(
      "close-tab",
      ({ index, action }: { index: number; action: CloseTabAction }) => {
        const max = items.length - 1;
        switch (action) {
          case "CLOSE":
            handleCloseSingleResultView(items[index]);
            return;
          case "CLOSE_OTHERS":
            for (let i = max; i > index; i--) {
              handleCloseSingleResultView(items[i]);
            }
            for (let i = index - 1; i >= 0; i--) {
              handleCloseSingleResultView(items[i]);
            }
            return;
          case "CLOSE_TO_THE_RIGHT":
            for (let i = max; i > index; i--) {
              handleCloseSingleResultView(items[i]);
            }
            return;
          case "CLOSE_ALL":
            for (let i = max; i >= 0; i--) {
              handleCloseSingleResultView(items[i]);
            }
            return;
        }
      }
    );
    return () => {
      unsubscribe();
    };
    // `tab` reactivity is captured by `items`; including it explicitly
    // re-creates the listener on every reactive tick.
  }, [items]);

  const validateExport = () =>
    selectedDatabaseNames.size > 0 && selectedDatabaseNames.size <= MAX_EXPORT;

  const handleExport = ({ options, resolve }: DataExportRequest) => {
    void (async () => {
      // === Prod path: per-database backend Export RPC ===
      const contents: DownloadContent[] = [];
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      // Defensive intersect — the drawer already filters to authorized
      // DBs, but if a stale name lingers in `selectedDatabaseNames` we
      // skip it here rather than firing a doomed backend call that
      // would produce a "failed for db" toast.
      const exportableSet = new Set(exportableDatabaseNames);
      for (const databaseName of Array.from(selectedDatabaseNames)) {
        if (!exportableSet.has(databaseName)) continue;
        const database = getDatabaseByName(databaseName);
        const context = head(tab?.databaseQueryContexts?.get(databaseName));
        if (!context) continue;
        try {
          const content = await exportData(
            create(ExportRequestSchema, {
              name: databaseName,
              ...(context.params.connection.dataSourceId
                ? { dataSourceId: context.params.connection.dataSourceId }
                : {}),
              format: options.format,
              statement: context.params.statement,
              limit: options.limit,
              admin: tab?.mode === "ADMIN",
              password: options.password,
              schema: context.params.connection.schema,
            })
          );
          contents.push({
            content,
            filename: `${
              extractDatabaseResourceName(database.name).databaseName
            }.${dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss")}.zip`,
          });
        } catch (e) {
          useAppStore.getState().notify({
            module: "bytebase",
            style: "CRITICAL",
            title: t("sql-editor.batch-export.failed-for-db", {
              db: databaseName,
            }),
            description: String(e),
          });
        }
      }
      resolve(contents);
      setSelectedDatabaseNames(new Set());
    })();
  };

  if (queriedDatabaseNames.length <= 1) {
    return null;
  }

  return (
    <div className="w-full flex flex-row justify-start items-center p-2 pb-0 gap-2 shrink-0">
      {showEmptySwitch && (
        <Tooltip
          content={t("sql-editor.batch-query.show-or-hide-empty-query-results")}
          side="bottom"
        >
          <Button
            variant={showEmpty ? "default" : "ghost"}
            size="sm"
            className="h-7 px-1.5 mb-2"
            onClick={() => setShowEmpty(!showEmpty)}
            aria-label={t(
              "sql-editor.batch-query.show-or-hide-empty-query-results"
            )}
          >
            {showEmpty ? (
              <Eye className="size-4" />
            ) : (
              <EyeOff className="size-4" />
            )}
          </Button>
        </Tooltip>
      )}

      {/*
        Partial-coverage UX: when the policy disables export and only a
        subset of the queried DBs has an active grant, both buttons can
        appear — Export operates on the covered set, Request Export
        pre-seeds the uncovered set so the user can extend coverage
        without losing the immediate export.
      */}
      <div className="mb-2 flex flex-row gap-2">
        {showExport && (
          <DataExportButton
            size="sm"
            viewMode="DRAWER"
            supportFormats={supportFormats}
            supportPassword
            text={
              isPartialExport
                ? t("sql-editor.batch-export.partial")
                : t("sql-editor.batch-export.self")
            }
            // Surface the grant-bypass explanation when the export is
            // only available because of a JIT grant; otherwise keep the
            // long-standing "select at most N databases" hint.
            tooltip={
              exportTooltip ??
              t("sql-editor.batch-export.tooltip", { max: MAX_EXPORT })
            }
            validate={validateExport}
            maximumExportCount={queryDataPolicy.maximumResultRows}
            onExport={handleExport}
            formContent={
              <div className="w-full flex flex-col gap-y-2">
                <div>
                  <p className="text-sm font-medium text-control">
                    {t("database.select")}
                    <span className="text-error ml-0.5">*</span>
                  </p>
                  <span className="text-xs text-control-light">
                    {t("sql-editor.batch-export.tooltip", { max: MAX_EXPORT })}
                  </span>
                </div>
                {/*
                  Filtered to `exportableDatabaseList` so users can only
                  check authorized DBs. Unauthorized ones are routed to
                  the Request Export button next to this one.
                */}
                <DatabaseTableView
                  databases={exportableDatabaseList}
                  mode="PROJECT"
                  selectedNames={selectedDatabaseNames}
                  onSelectedNamesChange={setSelectedDatabaseNames}
                />
              </div>
            }
          />
        )}
        {showRequestExport && (
          <RequestExportButton
            statement={batchStatement}
            // Pre-seed only the uncovered DBs so approvers see the
            // minimum new blast radius (matched DBs already have a
            // grant — no need to duplicate).
            targets={unmatchedDatabases}
          />
        )}
      </div>

      <div className="overflow-x-auto pb-2 flex-1">
        <div className="flex flex-row justify-start items-center gap-2">
          {filteredItems.map((item) => (
            <TabContextMenu
              key={item.database.name}
              index={items.indexOf(item)}
            >
              <TabButton
                item={item}
                isSelected={selectedDatabase === item.database}
                onSelect={() => onSelectedDatabaseChange(item.database)}
                onClose={() => handleCloseSingleResultView(item)}
                isFailed={isDatabaseQueryFailed(item)}
                isEmpty={isEmptyQueryItem(item)}
              />
            </TabContextMenu>
          ))}
        </div>
      </div>
    </div>
  );
}

type TabButtonProps = {
  item: BatchQueryItem;
  isSelected: boolean;
  onSelect: () => void;
  onClose: () => void;
  isFailed: boolean;
  isEmpty: boolean;
} & Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "onClick">;

function TabButton({
  item,
  isSelected,
  onSelect,
  onClose,
  isFailed,
  isEmpty,
  ref,
  className,
  style: styleProp,
  ...rest
}: TabButtonProps & { ref?: React.Ref<HTMLButtonElement> }) {
  const { t } = useTranslation();
  const environment = getDatabaseEnvironment(item.database);
  const colorRgb = hexToRgb(environment.color || "#4f46e5").join(", ");
  const style = {
    backgroundColor: `rgba(${colorRgb}, 0.1)`,
    borderTopColor: `rgb(${colorRgb})`,
    color: `rgb(${colorRgb})`,
    borderTop: isSelected ? "3px solid" : "",
    ...styleProp,
  };
  const instance = getInstanceResource(item.database);

  return (
    <button
      type="button"
      ref={ref}
      style={style}
      onClick={onSelect}
      className={cn(
        "inline-flex items-center gap-x-1 h-7 px-2 rounded-xs text-xs font-medium",
        "border border-control-border cursor-pointer whitespace-nowrap",
        className
      )}
      {...rest}
    >
      {EngineIconPath[instance.engine] && (
        <img
          src={EngineIconPath[instance.engine]}
          alt=""
          className="size-4 shrink-0"
        />
      )}
      <span className="truncate">
        {extractDatabaseResourceName(item.database.name).databaseName}
      </span>
      {isFailed && <CircleAlert className="ml-1 text-error size-4 shrink-0" />}
      {isEmpty && (
        <span className="text-control-placeholder italic ml-1">
          ({t("common.empty")})
        </span>
      )}
      <X
        className="ml-1 text-control-light hover:text-control size-4 shrink-0"
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
      />
    </button>
  );
}
