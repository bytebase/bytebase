import { CircleAlert, Loader2, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuTrigger,
} from "@/react/components/ui/context-menu";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { cn } from "@/react/lib/utils";
import {
  getSQLEditorTabsState,
  useCurrentSQLEditorTab,
  useIsInBatchMode,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import type { SQLEditorDatabaseQueryContext } from "@/types";
import { getDataSourceTypeI18n } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { formatAbsoluteDateTime, getInstanceResource } from "@/utils";
import { BatchQuerySelect } from "./BatchQuerySelect";
import { DatabaseQueryContext } from "./DatabaseQueryContext";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/ResultPanel/ResultPanel.vue`.
 *
 * Hosts the batch-query database selector at the top, then a card-style
 * tab strip of query contexts for the currently selected database, with
 * each tab rendering a `<DatabaseQueryContext>` (spinner / cancelled /
 * `<ResultView>`). Right-click on a tab opens a "close / close others /
 * close to the right / close all" menu.
 *
 * State the React component owns:
 * - `selectedDatabase`: which database the inner tabs are showing.
 * - `selectedTab`: which query-context id is the active inner tab.
 *
 * State read from the tab Zustand store:
 * - `tab.databaseQueryContexts.get(selectedDatabase.name)` — the array
 *   of query contexts for the active database. Immer produces fresh
 *   references on every mutation, so per-context status / resultSet
 *   flips re-render this tree.
 */
export function ResultPanel() {
  const { t } = useTranslation();
  const currentTab = useCurrentSQLEditorTab();
  const isInBatchMode = useIsInBatchMode();
  const [selectedDatabase, setSelectedDatabase] = useState<Database>();
  const [selectedTab, setSelectedTab] = useState<string>();

  // Subscribe to the current tab's `databaseQueryContexts` Map. The
  // store mutates this Map (and the inner arrays + entries) via immer,
  // so every mutation produces a new reference — Zustand re-runs this
  // selector on each change.
  const databaseQueryContexts = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.databaseQueryContexts
  );

  // Re-spread on every render so the iteration sees the latest array
  // (the store mutates via immer; cached refs would hide additions).
  const queryContexts: SQLEditorDatabaseQueryContext[] | undefined = (() => {
    const name = selectedDatabase?.name;
    if (!name) return undefined;
    const arr = databaseQueryContexts?.get(name);
    return arr ? [...arr] : undefined;
  })();

  const isBatchQuery = (databaseQueryContexts?.size ?? 0) > 1;

  const batchModeDataSourceType = isInBatchMode
    ? (currentTab?.batchQueryContext?.dataSourceType ?? null)
    : null;

  const hasMultipleContexts = (queryContexts?.length ?? 0) > 1;

  // Mirror Vue's `watch(queryContexts.[0]?.id, ..., { immediate: true })`:
  // when the head of the contexts list changes, switch the active tab to
  // it (newest run becomes selected).
  const headId = queryContexts?.[0]?.id;
  useEffect(() => {
    if (headId !== undefined) setSelectedTab(headId);
  }, [headId]);

  const tabName = (context: SQLEditorDatabaseQueryContext): string => {
    switch (context.status) {
      case "PENDING":
        return t("sql-editor.pending-query");
      case "EXECUTING":
        return t("sql-editor.executing-query");
      default:
        return formatAbsoluteDateTime(context.beginTimestampMS ?? Date.now());
    }
  };

  const dataSourcesById = useMemo(() => {
    if (!selectedDatabase) return new Map();
    const instance = getInstanceResource(selectedDatabase);
    return new Map((instance.dataSources ?? []).map((ds) => [ds.id, ds]));
  }, [selectedDatabase]);

  const dataSourceInContext = (context: SQLEditorDatabaseQueryContext) =>
    dataSourcesById.get(context.params.connection.dataSourceId);

  const isMatchedDataSource = (context: SQLEditorDatabaseQueryContext) => {
    const mode = batchModeDataSourceType;
    if (!mode) return true;
    const ds = dataSourceInContext(context);
    if (!ds) return true;
    return ds.type === mode;
  };

  const closeTab = (id: string) => {
    const next = getSQLEditorTabsState().removeDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextId: id,
    });
    if (selectedTab === id && next) setSelectedTab(next.id);
  };

  const closeOthers = (id: string) => {
    getSQLEditorTabsState().batchRemoveDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextIds:
        queryContexts?.filter((c) => c.id !== id).map((c) => c.id) ?? [],
    });
    setSelectedTab(id);
  };

  const closeToTheRight = (id: string) => {
    const idx = queryContexts?.findIndex((c) => c.id === id) ?? -1;
    if (idx < 0) return;
    getSQLEditorTabsState().batchRemoveDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextIds: queryContexts?.slice(idx + 1).map((c) => c.id) ?? [],
    });
    setSelectedTab(id);
  };

  const closeAll = () => {
    getSQLEditorTabsState().deleteDatabaseQueryContext(
      selectedDatabase?.name ?? ""
    );
    setSelectedTab(undefined);
  };

  return (
    <div className="relative w-full h-full flex flex-col justify-start items-start z-10 overflow-x-hidden">
      <div className="w-full shrink-0">
        <BatchQuerySelect
          selectedDatabase={selectedDatabase}
          onSelectedDatabaseChange={setSelectedDatabase}
        />
      </div>
      {selectedDatabase && queryContexts && queryContexts.length > 0 && (
        <Tabs
          value={selectedTab ?? ""}
          onValueChange={(v) => setSelectedTab(v as string)}
          className={cn(
            "flex-1 flex flex-col overflow-hidden px-2 min-h-0",
            isBatchQuery ? "pt-0" : "pt-2"
          )}
        >
          <TabsList className="shrink-0 gap-x-1 border-b border-control-border overflow-x-auto overflow-y-hidden">
            {queryContexts.map((context) => (
              // Order matters: `Tooltip` MUST wrap `TabContextMenu`, not the
              // other way around. `ContextMenuTrigger` uses Base UI's
              // `render` prop, which clones the given element and attaches
              // `onContextMenu` / `ref` to it — that only works when the
              // rendered element forwards unknown props onto its DOM
              // (`TabsTrigger` does, our `Tooltip` wrapper doesn't). With
              // Tooltip on the outside, the trigger renders `TabsTrigger`
              // directly and the right-click handler reaches the DOM.
              <Tooltip key={context.id} content={context.params.statement}>
                <TabContextMenu
                  onSelect={(action) => {
                    switch (action) {
                      case "CLOSE":
                        closeTab(context.id);
                        break;
                      case "CLOSE_OTHERS":
                        closeOthers(context.id);
                        break;
                      case "CLOSE_TO_THE_RIGHT":
                        closeToTheRight(context.id);
                        break;
                      case "CLOSE_ALL":
                        closeAll();
                        break;
                    }
                  }}
                >
                  <TabsTrigger
                    value={context.id}
                    className="flex items-center gap-x-2 px-3 py-1 shrink-0"
                  >
                    <span className="truncate">{tabName(context)}</span>
                    {context.resultSet?.error && (
                      <CircleAlert className="size-4 text-error shrink-0" />
                    )}
                    {context.status === "EXECUTING" && (
                      <Loader2 className="size-3 animate-spin shrink-0" />
                    )}
                    {hasMultipleContexts && (
                      <X
                        className="size-4 text-control-light hover:text-control shrink-0"
                        onClick={(e) => {
                          e.stopPropagation();
                          closeTab(context.id);
                        }}
                      />
                    )}
                  </TabsTrigger>
                </TabContextMenu>
              </Tooltip>
            ))}
          </TabsList>
          {queryContexts.map((context, i) => (
            <TabsPanel
              key={context.id}
              value={context.id}
              className="flex-1 flex flex-col min-h-0 overflow-hidden mt-2"
            >
              {i === 0 &&
                batchModeDataSourceType &&
                !isMatchedDataSource(context) && (
                  <Alert variant="warning" className="mb-2">
                    {t("sql-editor.batch-query.select-data-source.not-match", {
                      expect: getDataSourceTypeI18n(
                        currentTab?.batchQueryContext?.dataSourceType
                      ),
                      actual: getDataSourceTypeI18n(
                        dataSourceInContext(context)?.type
                      ),
                    })}
                  </Alert>
                )}
              <DatabaseQueryContext
                database={selectedDatabase}
                context={context}
              />
            </TabsPanel>
          ))}
        </Tabs>
      )}
    </div>
  );
}

type CloseAction =
  | "CLOSE"
  | "CLOSE_OTHERS"
  | "CLOSE_TO_THE_RIGHT"
  | "CLOSE_ALL";

const CLOSE_ACTIONS: readonly CloseAction[] = [
  "CLOSE",
  "CLOSE_OTHERS",
  "CLOSE_TO_THE_RIGHT",
  "CLOSE_ALL",
];

const CLOSE_ACTION_KEYS: Record<CloseAction, string> = {
  CLOSE: "sql-editor.tab.context-menu.actions.close",
  CLOSE_OTHERS: "sql-editor.tab.context-menu.actions.close-others",
  CLOSE_TO_THE_RIGHT: "sql-editor.tab.context-menu.actions.close-to-the-right",
  CLOSE_ALL: "sql-editor.tab.context-menu.actions.close-all",
};

/**
 * Right-click context menu wrapper for a single tab. Local to ResultPanel
 * so the close-tab handler is delivered directly via prop, sidestepping
 * the cross-component `resultTabEvents` channel that Stage 18's
 * `BatchQuerySelect` already owns for its database-strip tabs.
 */
function TabContextMenu({
  children,
  onSelect,
}: {
  children: React.ReactElement;
  onSelect: (action: CloseAction) => void;
}) {
  const { t } = useTranslation();
  return (
    <ContextMenu>
      <ContextMenuTrigger render={children} />
      <ContextMenuContent>
        {CLOSE_ACTIONS.map((action) => (
          <ContextMenuItem key={action} onClick={() => onSelect(action)}>
            {t(CLOSE_ACTION_KEYS[action])}
          </ContextMenuItem>
        ))}
      </ContextMenuContent>
    </ContextMenu>
  );
}
