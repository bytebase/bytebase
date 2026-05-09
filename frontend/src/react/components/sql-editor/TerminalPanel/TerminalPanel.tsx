import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { watch } from "vue";
import type { IStandaloneCodeEditor } from "@/components/MonacoEditor";
import { ConnectionHolder } from "@/react/components/sql-editor/ConnectionHolder";
import { EditorAction } from "@/react/components/sql-editor/EditorAction";
import { ResultView } from "@/react/components/sql-editor/ResultView";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useDatabaseV1Store,
  useSQLEditorTabStore,
  useWebTerminalStore,
} from "@/store";
import type { SQLEditorQueryParams, WebTerminalQueryItemV1 } from "@/types";
import { CompactSQLEditor } from "./CompactSQLEditor";
import { useHistory } from "./useHistory";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/TerminalPanel.vue`.
 *
 * Hosts the admin-mode terminal: a top action bar, then a vertically
 * scrolling stack of `<CompactSQLEditor>` + `<ResultView>` rows (one per
 * historical query). Tail row is editable; older rows are read-only.
 * When the underlying Pinia tab is disconnected we render
 * `<ConnectionHolder>` instead (mirrors the Vue `v-if`).
 *
 * State source:
 * - `webTerminalStore.getQueryStateByTab(currentTab).queryItemList` — the
 *   per-tab list of query items (statements + their result sets).
 * - `useHistory()` — the up/down arrow command-history Pinia composable.
 *
 * Auto-scroll: a ResizeObserver on the inner stack scrolls the outer
 * container to the bottom whenever the stack grows (replacing Vue's
 * `useElementSize` + watch).
 */
export function TerminalPanel() {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const webTerminalStore = useWebTerminalStore();
  const databaseStore = useDatabaseV1Store();

  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const currentTabId = useVueState(() => tabStore.currentTab?.id);

  // Force a React re-render on every mutation of the active tab's
  // `queryItemList` (push, shift, per-item field flips). Watching the
  // ref's `.value` keeps the watch alive even when no tab is loaded
  // initially — re-arming on `currentTabId` change picks up the new
  // tab's query list. A plain `useVueState` getter wouldn't work here
  // because the store mutates the array in place, so the getter would
  // keep returning the same reference and `useSyncExternalStore` would
  // skip the re-render.
  const [, forceRender] = useState(0);
  useEffect(() => {
    const tab = tabStore.currentTab;
    if (!tab) return;
    const qs = webTerminalStore.getQueryStateByTab(tab);
    const stop = watch(
      () => qs.queryItemList.value,
      () => forceRender((v) => v + 1),
      { deep: true, flush: "sync" }
    );
    return () => {
      stop();
    };
  }, [tabStore, webTerminalStore, currentTabId]);

  // Re-spread on every render so the JSX iteration sees the latest
  // array. Items are still the same Vue reactive proxies; mutating
  // `item.statement` etc. flows back to the store via Pinia. Operations
  // that need to mutate the **list shape** (e.g. clear-screen
  // `.shift()`) must reach the live `queryItemList.value` directly.
  const queryList: WebTerminalQueryItemV1[] = (() => {
    const tab = tabStore.currentTab;
    if (!tab) return [];
    const qs = webTerminalStore.getQueryStateByTab(tab);
    return [...qs.queryItemList.value];
  })();

  const expired = useVueState(() => {
    const tab = tabStore.currentTab;
    if (!tab) return false;
    const qs = webTerminalStore.getQueryStateByTab(tab);
    return qs.timer.expired.value;
  });

  // Pre-fetch any database referenced by an existing query item so the
  // ResultView can render `database.environment` etc. synchronously.
  useEffect(() => {
    void databaseStore.batchGetOrFetchDatabases(
      queryList.map((q) => q?.params?.connection.database ?? "")
    );
  }, [queryList, databaseStore]);

  const currentQuery = queryList[queryList.length - 1];
  const isEditableQueryItem = (item: WebTerminalQueryItemV1) =>
    item === currentQuery && item.status === "IDLE";

  const { move: moveHistory } = useHistory();

  const handleExecute = useCallback(
    (params: SQLEditorQueryParams) => {
      if (currentQuery?.status !== "IDLE") return;
      if (!params.statement) {
        console.warn("Empty query");
        return;
      }
      const tab = tabStore.currentTab;
      if (!tab) return;
      const qs = webTerminalStore.getQueryStateByTab(tab);
      void qs.controller.events.emit("query", params);
    },
    [currentQuery, tabStore, webTerminalStore]
  );

  const handleClearScreen = useCallback(() => {
    // Mutate the live Vue array, not our shallow copy — the store's
    // reactivity is tracked on the source and React re-renders via the
    // tick-bumped `queryList` snapshot.
    const tab = tabStore.currentTab;
    if (!tab) return;
    const list = webTerminalStore.getQueryStateByTab(tab).queryItemList.value;
    while (list.length > 1) {
      list.shift();
    }
  }, [tabStore, webTerminalStore]);

  const handleHistory = useCallback(
    (direction: "up" | "down", editor: IStandaloneCodeEditor) => {
      if (currentQuery?.status !== "IDLE") return;
      moveHistory(direction);
      requestAnimationFrame(() => {
        const model = editor.getModel();
        if (!model) return;
        const lineNumber = model.getLineCount();
        const column = model.getLineMaxColumn(lineNumber);
        editor.setPosition({ lineNumber, column });
      });
    },
    [currentQuery, moveHistory]
  );

  const handleCancelQuery = () => {
    const tab = tabStore.currentTab;
    if (!tab) return;
    webTerminalStore.getQueryStateByTab(tab).controller.abort();
  };

  // Auto-scroll the outer container to the bottom whenever the inner
  // stack resizes. ResizeObserver replaces `@vueuse/core`'s
  // `useElementSize` + `watch(queryListHeight, ...)`.
  const containerRef = useRef<HTMLDivElement | null>(null);
  const stackRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const stack = stackRef.current;
    if (!stack) return;
    const observer = new ResizeObserver(() => {
      const container = containerRef.current;
      if (container) {
        requestAnimationFrame(() => {
          container.scrollTo(0, container.scrollHeight);
        });
      }
    });
    observer.observe(stack);
    return () => observer.disconnect();
    // Re-bind when the active tab changes — the stack element may remount.
  }, [currentTabId]);

  // Build a stable handler list per-query for the editor onChange callback,
  // since the editor inside `<CompactSQLEditor>` re-registers actions on
  // identity change of these props.
  const handleChangeFor = useMemo(
    () =>
      new Map(
        queryList.map((q) => [
          q.id,
          (value: string) => {
            q.statement = value;
          },
        ])
      ),
    [queryList]
  );

  return (
    <div className="flex h-full w-full flex-col justify-start items-stretch overflow-hidden bg-dark-bg">
      <EditorAction />
      {!isDisconnected ? (
        <div
          ref={containerRef}
          className="w-full flex-1 overflow-y-auto bg-dark-bg"
        >
          <div ref={stackRef} className="w-full flex flex-col">
            {queryList.map((query) => {
              const editable = isEditableQueryItem(query);
              const database = query.params?.connection.database
                ? databaseStore.getDatabaseByName(
                    query.params.connection.database
                  )
                : undefined;
              return (
                <div key={query.id} className="relative">
                  <CompactSQLEditor
                    content={query.statement}
                    readonly={!editable}
                    onChange={handleChangeFor.get(query.id) ?? noop}
                    onExecute={handleExecute}
                    onHistory={handleHistory}
                    onClearScreen={handleClearScreen}
                  />
                  {query.params && query.resultSet && database && (
                    <div className="p-2 w-full flex-1 min-h-0">
                      <ResultView
                        executeParams={query.params}
                        resultSet={query.resultSet}
                        database={database}
                        loading={query.status === "RUNNING"}
                        dark
                      />
                    </div>
                  )}
                  {query.resultSet?.error && (
                    <div className="p-2 pb-1 text-md font-normal text-matrix-green-hover">
                      {t("sql-editor.connection-lost")}
                    </div>
                  )}
                  {query.status === "RUNNING" && (
                    <div className="absolute inset-0 bg-black/20 flex justify-center items-center gap-2">
                      <Loader2 className="size-5 animate-spin text-control-light" />
                      {query === currentQuery && expired && (
                        <button
                          type="button"
                          className="text-gray-400 cursor-pointer hover:underline text-sm select-none"
                          onClick={handleCancelQuery}
                        >
                          {t("common.cancel")}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      ) : (
        <div className="flex-1 flex flex-col min-h-0">
          <ConnectionHolder />
        </div>
      )}
    </div>
  );
}

const noop = () => {};
