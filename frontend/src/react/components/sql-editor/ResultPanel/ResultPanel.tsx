import { CircleAlert, Loader2, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { watch } from "vue";
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
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
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
 * State read from Pinia via `useVueState`:
 * - `tabStore.currentTab.databaseQueryContexts.get(selectedDatabase.name)` —
 *   the array of query contexts for the active database. `{ deep: true }`
 *   so per-context status / resultSet flips re-render this tree without
 *   the Map reference changing.
 */
export function ResultPanel() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const [selectedDatabase, setSelectedDatabase] = useState<Database>();
  const [selectedTab, setSelectedTab] = useState<string>();

  // Force a React re-render whenever **anything** changes inside the
  // current tab's `databaseQueryContexts` Map — Map mutations
  // (`set` / `delete`), inner-array mutations (`unshift` / `splice`),
  // and per-item field flips (status PENDING → EXECUTING → DONE,
  // resultSet writes). Keying the watch on the Map itself (rather than
  // `Map.get(selectedDatabase.name)`) is critical: when this component
  // first mounts `selectedDatabase` is `undefined`, and a getter that
  // bailed out early would register **no** Vue reactive deps — leaving
  // the watch dead forever, since `selectedDatabase` becoming defined
  // doesn't fire any watcher. Tracking the Map directly with `deep:
  // true` keeps the watch live across all reactive transitions, so the
  // newly-created PENDING context reaches `<DatabaseQueryContext>` and
  // its auto-run effect can advance the status.
  const [, forceRender] = useState(0);
  const tabIdForWatch = useVueState(() => tabStore.currentTab?.id);
  useEffect(() => {
    const stop = watch(
      () => tabStore.currentTab?.databaseQueryContexts,
      () => forceRender((v) => v + 1),
      { deep: true, flush: "sync" }
    );
    return () => {
      stop();
    };
    // Re-arm the watch when the active tab changes — each tab has its
    // own `databaseQueryContexts` Map and Vue's auto-re-tracking only
    // fires when the previously-tracked deps change. Without this,
    // switching tabs would lose live updates from the new tab's Map.
  }, [tabStore, tabIdForWatch]);

  // Re-spread on every render so the iteration sees the latest array
  // (Pinia mutates in place; cached refs would hide additions). Items
  // are still the same Vue reactive proxies so `context.status` reads
  // the live value.
  const queryContexts: SQLEditorDatabaseQueryContext[] | undefined = (() => {
    const name = selectedDatabase?.name;
    if (!name) return undefined;
    const arr = tabStore.currentTab?.databaseQueryContexts?.get(name);
    return arr ? [...arr] : undefined;
  })();

  const isBatchQuery = useVueState(() => {
    const contexts = tabStore.currentTab?.databaseQueryContexts;
    return contexts ? contexts.size > 1 : false;
  });

  const batchModeDataSourceType = useVueState(() => {
    if (!tabStore.isInBatchMode) return null;
    return tabStore.currentTab?.batchQueryContext.dataSourceType ?? null;
  });

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
    const next = tabStore.removeDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextId: id,
    });
    if (selectedTab === id && next) setSelectedTab(next.id);
  };

  const closeOthers = (id: string) => {
    tabStore.batchRemoveDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextIds:
        queryContexts?.filter((c) => c.id !== id).map((c) => c.id) ?? [],
    });
    setSelectedTab(id);
  };

  const closeToTheRight = (id: string) => {
    const idx = queryContexts?.findIndex((c) => c.id === id) ?? -1;
    if (idx < 0) return;
    tabStore.batchRemoveDatabaseQueryContext({
      database: selectedDatabase?.name ?? "",
      contextIds: queryContexts?.slice(idx + 1).map((c) => c.id) ?? [],
    });
    setSelectedTab(id);
  };

  const closeAll = () => {
    tabStore.deleteDatabaseQueryContext(selectedDatabase?.name ?? "");
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
                        tabStore.currentTab?.batchQueryContext.dataSourceType
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
