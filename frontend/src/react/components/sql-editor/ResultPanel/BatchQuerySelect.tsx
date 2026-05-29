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
import { pushNotification, useSQLStore } from "@/store";
import { isValidDatabaseName } from "@/types";
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
import { buildDownloadBlob, type DownloadGroup } from "@/utils/sql-download";
import { SQL_ENGINE_QUOTES } from "@/utils/sql-download/engines";
import { downloadErrorMessage } from "@/utils/sql-download/error-messages";
import { isDev } from "@/utils/util";
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
  const sqlStore = useSQLStore();
  const currentTab = useCurrentSQLEditorTab();
  const project = useSQLEditorEditorState((s) => s.project);
  const queryDataPolicy = useSQLEditorQueryDataPolicy(project);

  const [showEmpty, setShowEmpty] = useState(true);
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());

  // Editor's per-execution row limit. Used by the dev-path export to
  // infer probable truncation: `r.rows.length === resultRowsLimit` →
  // query capped, likely more rows available.
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);

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

  const databaseList = useMemo(
    () => queriedDatabaseNames.map((name) => getDatabaseByName(name)),
    [queriedDatabaseNames, getDatabaseByName]
  );

  // Drop SQL from the export drawer when isDev() is active and any
  // user-SELECTED database's engine can't be rendered as INSERTs. Gating
  // on the full queried-database set was too conservative (a tab with one
  // MongoDB + one MySQL would hide SQL even when the user selected only
  // MySQL). Empty selection shows all formats so the user sees the SQL
  // option before picking incompatible databases.
  const supportFormats = useMemo(() => {
    const all = [
      ExportFormat.CSV,
      ExportFormat.JSON,
      ExportFormat.SQL,
      ExportFormat.XLSX,
    ];
    if (!isDev()) return all;
    if (selectedDatabaseNames.size === 0) return all;
    const allSupportSql = Array.from(selectedDatabaseNames).every((name) => {
      const db = getDatabaseByName(name);
      if (!isValidDatabaseName(db.name)) return true; // skipped at export time
      return SQL_ENGINE_QUOTES.has(getInstanceResource(db).engine);
    });
    return allSupportSql
      ? all
      : [ExportFormat.CSV, ExportFormat.JSON, ExportFormat.XLSX];
  }, [selectedDatabaseNames, getDatabaseByName]);

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

  const handleExport = ({ options, resolve, reject }: DataExportRequest) => {
    void (async () => {
      // === Dev path: client-side ZIP via buildDownloadBlob ===
      if (isDev()) {
        try {
          // Stale format guard: drop the export if the user-selected format
          // is no longer in the (selection-aware) supportFormats list.
          if (!supportFormats.includes(options.format)) {
            reject(
              "The selected format is not supported for the current database selection. Pick a different format."
            );
            return;
          }
          const tab = currentTab;
          const groups: DownloadGroup[] = [];
          const limit = options.limit > 0 ? options.limit : Infinity;
          for (const databaseName of Array.from(selectedDatabaseNames)) {
            const database = getDatabaseByName(databaseName);
            // Skip stub-database returns with a WARN toast so the user
            // knows their selection became stale (tab torn down mid-export).
            if (!isValidDatabaseName(database.name)) {
              pushNotification({
                module: "bytebase",
                style: "WARN",
                title: t("sql-editor.batch-export.failed-for-db", {
                  db: databaseName,
                }),
                description: t(
                  "sql-editor.batch-export.database-no-longer-available"
                ),
              });
              continue;
            }
            const context = head(tab?.databaseQueryContexts?.get(databaseName));
            const resultSet = context?.resultSet;
            const resultError =
              resultSet?.error ||
              resultSet?.results.find((r) => r.error)?.error;
            if (resultError) {
              pushNotification({
                module: "bytebase",
                style: "CRITICAL",
                title: t("sql-editor.batch-export.failed-for-db", {
                  db: databaseName,
                }),
                description: resultError,
              });
              continue;
            }
            const results = resultSet?.results ?? [];
            if (results.length === 0) continue;
            // Truncation inference using execution-time limit.
            const executedLimit = context?.params.limit ?? resultRowsLimit;
            if (
              options.limit > 0 &&
              results.some(
                (r) =>
                  r.rows.length === executedLimit &&
                  options.limit > r.rows.length
              )
            ) {
              pushNotification({
                module: "bytebase",
                style: "WARN",
                title: t("sql-editor.batch-export.failed-for-db", {
                  db: databaseName,
                }),
                description: t(
                  "sql-editor.batch-export.export-limit-exceeds-executed",
                  {
                    exportLimit: options.limit,
                    executedLimit,
                  }
                ),
              });
            }
            const { databaseName: dbSeg, instanceName } =
              extractDatabaseResourceName(database.name);
            groups.push({
              instanceId: instanceName,
              databaseName: dbSeg,
              engine: getInstanceResource(database).engine,
              statements: results.map((r) => ({
                result:
                  r.rows.length > limit
                    ? { ...r, rows: r.rows.slice(0, limit) }
                    : r,
                statement: r.statement || context?.params.statement || "",
              })),
            });
          }
          if (groups.length === 0) {
            reject(t("sql-editor.batch-export.no-results"));
            return;
          }
          const out = await buildDownloadBlob({
            groups,
            format: options.format,
            baseFilename: `batch-export.${dayjs().format(
              "YYYY-MM-DDTHH-mm-ss"
            )}`,
            password: options.password,
          });
          const content = new Uint8Array(await out.blob.arrayBuffer());
          resolve([{ content, filename: out.filename }]);
          setSelectedDatabaseNames(new Set());
        } catch (e) {
          reject(downloadErrorMessage(e, t));
        }
        return;
      }

      // === Prod path: per-database backend Export RPC ===
      const contents: DownloadContent[] = [];
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      for (const databaseName of Array.from(selectedDatabaseNames)) {
        const database = getDatabaseByName(databaseName);
        const context = head(tab?.databaseQueryContexts?.get(databaseName));
        if (!context) continue;
        try {
          const content = await sqlStore.exportData(
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
          pushNotification({
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

      <div className="mb-2">
        <DataExportButton
          size="sm"
          viewMode="DRAWER"
          supportFormats={supportFormats}
          supportPassword
          text={t("sql-editor.batch-export.self")}
          tooltip={t("sql-editor.batch-export.tooltip", { max: MAX_EXPORT })}
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
              <DatabaseTableView
                databases={databaseList}
                mode="PROJECT"
                selectedNames={selectedDatabaseNames}
                onSelectedNamesChange={setSelectedDatabaseNames}
              />
            </div>
          }
        />
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
