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
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import {
  pushNotification,
  useDatabaseV1Store,
  useSQLEditorStore,
  useSQLEditorTabStore,
  useSQLStore,
} from "@/store";
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
  const tabStore = useSQLEditorTabStore();
  const databaseStore = useDatabaseV1Store();
  const sqlStore = useSQLStore();
  const editorStore = useSQLEditorStore();

  const [showEmpty, setShowEmpty] = useState(true);
  const [selectedDatabaseNames, setSelectedDatabaseNames] = useState<
    Set<string>
  >(new Set());

  const queryDataPolicy = useVueState(() => editorStore.queryDataPolicy);

  // Read the Map's `.keys()` directly inside the Vue getter so Vue's
  // reactivity tracks the iteration. `useVueState(() => tabStore.currentTab)`
  // would only fire when the tab object reference changes — Map mutations
  // (which is how `useExecuteSQL.preExecute` adds new query contexts)
  // wouldn't trigger React re-renders.
  const queriedDatabaseNames = useVueState(
    () => Array.from(tabStore.currentTab?.databaseQueryContexts?.keys() || []),
    { deep: true }
  );

  // Track the contexts arrays themselves so the `items[].context` snapshot
  // and `isEmptyQueryItem` computations stay in sync as `runQuery` mutates
  // each context's `status` / `resultSet`.
  const contextsByDatabase = useVueState(
    () => {
      const map = new Map<string, SQLEditorDatabaseQueryContext | undefined>();
      const contexts = tabStore.currentTab?.databaseQueryContexts;
      if (!contexts) return map;
      for (const name of contexts.keys()) {
        map.set(name, head(contexts.get(name)));
      }
      return map;
    },
    { deep: true }
  );

  const items = useMemo<BatchQueryItem[]>(() => {
    return queriedDatabaseNames.map((name) => ({
      database: databaseStore.getDatabaseByName(name),
      context: contextsByDatabase.get(name),
    }));
  }, [queriedDatabaseNames, contextsByDatabase, databaseStore]);

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
    () =>
      queriedDatabaseNames.map((name) => databaseStore.getDatabaseByName(name)),
    [queriedDatabaseNames, databaseStore]
  );

  const handleCloseSingleResultView = (item: BatchQueryItem) => {
    const tab = tabStore.currentTab;
    const contexts = tab?.databaseQueryContexts?.get(item.database.name);
    if (!contexts) return;
    for (const context of contexts) {
      context.abortController?.abort();
    }
    tab?.databaseQueryContexts?.delete(item.database.name);
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
      const contents: DownloadContent[] = [];
      const tab = tabStore.currentTab;
      for (const databaseName of Array.from(selectedDatabaseNames)) {
        const database = databaseStore.getDatabaseByName(databaseName);
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
              admin: tabStore.currentTab?.mode === "ADMIN",
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
          supportFormats={[
            ExportFormat.CSV,
            ExportFormat.JSON,
            ExportFormat.SQL,
            ExportFormat.XLSX,
          ]}
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
