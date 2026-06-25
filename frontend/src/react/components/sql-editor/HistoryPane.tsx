import dayjs from "dayjs";
import { Copy, Link2, Loader2, X } from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Button } from "@/react/components/ui/button";
import { writeTextToClipboard } from "@/react/lib/clipboard";
import { cn } from "@/react/lib/utils";
import { router } from "@/react/router";
import { SQL_EDITOR_QUERY_HISTORY_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import type { QueryHistoryFilter } from "@/react/stores/sqlEditor";
import {
  selectQueryHistoryEntry,
  useSQLEditorStore,
} from "@/react/stores/sqlEditor";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { DEBOUNCE_SEARCH_DELAY, getDateForPbTimestampProtoEs } from "@/types";
import type { QueryHistory } from "@/types/proto-es/v1/sql_service_pb";
import { extractProjectResourceName, extractQueryHistoryUID } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { HistorySearchInput } from "./HistorySearchInput";

/**
 * React migration of frontend/src/views/sql-editor/AsidePanel/HistoryPane/HistoryPane.vue.
 * Displays the query history list with search, copy, and click-to-append features.
 */
export function HistoryPane() {
  const { t } = useTranslation();

  const fetchQueryHistoryList = useSQLEditorStore(
    (s) => s.fetchQueryHistoryList
  );
  const resetPageToken = useSQLEditorStore((s) => s.resetPageToken);
  const workspaceExternalURL = useAppStore((s) => s.serverInfo?.externalUrl);
  const linkedQueryHistory = useSQLEditorStore((s) => s.linkedQueryHistory);
  const linkedQueryHistoryTabId = useSQLEditorStore(
    (s) => s.linkedQueryHistoryTabId
  );
  const setLinkedQueryHistory = useSQLEditorStore(
    (s) => s.setLinkedQueryHistory
  );
  const currentTabId = useSQLEditorTabState((s) => s.currentTabId);

  const [searchText, setSearchText] = useState("");
  const [loading, setLoading] = useState(false);
  // Bumped when `query-executed` fires; forces React to re-read the
  // store cache (which `useExecuteSQL` / `webTerminal` have already
  // refreshed via `mergeLatest` by the time we get the event).
  const [, bumpRefresh] = useReducer((c: number) => c + 1, 0);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const currentTabDatabase = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.database
  );
  const project = useSQLEditorEditorState((s) => s.project);

  const historyQuery = useMemo<QueryHistoryFilter>(
    () => ({
      database: currentTabDatabase,
      project: project,
      statement: searchText,
    }),
    [currentTabDatabase, project, searchText]
  );

  const historyData = useSQLEditorStore(selectQueryHistoryEntry(historyQuery));

  const fetchHistory = useCallback(async () => {
    setLoading(true);
    try {
      await fetchQueryHistoryList(historyQuery);
    } finally {
      setLoading(false);
    }
  }, [fetchQueryHistoryList, historyQuery]);

  // Debounced fetch on query change — only when list is empty (initial load / filter change)
  useEffect(() => {
    if (historyData.queryHistories.length === 0) {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
      searchTimerRef.current = setTimeout(fetchHistory, DEBOUNCE_SEARCH_DELAY);
    }
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, [historyQuery]);

  // Force a re-render on every post-execute event. The store cache is
  // already up-to-date by the time the event fires (`useExecuteSQL` /
  // `webTerminal` chain the emit in `.finally` after `mergeLatest`
  // resolves). The bumped reducer state triggers a render, which
  // re-runs the Vue-bridge getter and reads the merged list —
  // preserving any pages the user had already loaded.
  useEffect(() => {
    sqlEditorEvents.on("query-executed", bumpRefresh);
    return () => {
      sqlEditorEvents.off("query-executed", bumpRefresh);
    };
  }, []);

  const onSearchUpdate = useCallback(
    (next: string) => {
      resetPageToken({ ...historyQuery, statement: next });
      setSearchText(next);
    },
    [resetPageToken, historyQuery]
  );

  const titleOfQueryHistory = (h: QueryHistory) =>
    dayjs(getDateForPbTimestampProtoEs(h.createTime)).format(
      "YYYY-MM-DD HH:mm:ss"
    );

  const handleHistoryClick = async (history: QueryHistory) => {
    const { statement } = history;
    const tabsState = getSQLEditorTabsState();
    if (tabsState.tabsById.get(tabsState.currentTabId)) {
      await sqlEditorEvents.emit("append-editor-content", {
        content: statement,
        select: true,
      });
    } else {
      tabsState.addTab(
        {
          title: `Query history at ${titleOfQueryHistory(history)}`,
          statement,
        },
        /* beside */ true
      );
    }
  };

  const handleCopy = async (statement: string) => {
    if (await writeTextToClipboard(statement)) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      // clipboard not available
    }
  };

  // Builds a shareable deep link to a query history. Opening it loads the
  // statement into a new draft tab — see `SQLEditorRouteShell`.
  const buildHistoryLink = (history: QueryHistory) => {
    const route = router.resolve({
      name: SQL_EDITOR_QUERY_HISTORY_MODULE,
      params: {
        project: extractProjectResourceName(history.name),
        queryHistory: extractQueryHistoryUID(history.name),
      },
    });
    return new URL(
      route.href,
      workspaceExternalURL || globalThis.location.origin
    ).href;
  };

  const handleCopyLink = async (history: QueryHistory) => {
    if (await writeTextToClipboard(buildHistoryLink(history))) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      // clipboard not available
    }
  };

  // Only surface the "Opened from link" section while its own draft tab is
  // active — switching to another tab hides it (it reappears on switching back).
  const showLinkedHistory =
    !!linkedQueryHistory && currentTabId === linkedQueryHistoryTabId;

  // When shown, the linked history must not also repeat in the recent list.
  const recentHistories = historyData.queryHistories.filter(
    (h) => !(showLinkedHistory && h.name === linkedQueryHistory?.name)
  );

  // Highlight each whitespace-separated search term independently, so matches
  // show even when the stored statement differs in whitespace/case from the
  // query — mirroring the server's normalized, case-insensitive search.
  const searchKeywords = useMemo(
    () => searchText.trim().split(/\s+/).filter(Boolean),
    [searchText]
  );

  const renderHistoryRow = (history: QueryHistory, highlighted = false) => (
    <div
      key={history.name}
      data-history-row
      className={cn(
        "w-full p-2 gap-y-1 flex flex-col justify-start items-start cursor-pointer",
        highlighted ? "bg-accent/10" : "border-b hover:bg-control-bg"
      )}
      onClick={() => {
        void handleHistoryClick(history);
      }}
    >
      <div className="w-full flex flex-row justify-between items-center">
        <div className="flex items-start gap-x-1">
          <span className="text-xs text-control-placeholder">
            {titleOfQueryHistory(history)}
          </span>
        </div>
        <div className="flex items-center gap-x-1">
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 hover:bg-control-bg-hover"
            data-copy-link-btn
            onClick={(e) => {
              e.stopPropagation();
              void handleCopyLink(history);
            }}
            aria-label={t("sql-editor.copy-history-link")}
          >
            <Link2 className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0 hover:bg-control-bg-hover"
            data-copy-btn
            onClick={(e) => {
              e.stopPropagation();
              void handleCopy(history.statement);
            }}
            aria-label={t("common.copy")}
          >
            <Copy className="size-3.5" />
          </Button>
        </div>
      </div>
      <HighlightLabelText
        text={history.statement}
        keyword={searchKeywords}
        className="max-w-full text-xs wrap-break-word font-mono line-clamp-3"
      />
    </div>
  );

  return (
    <div className="relative w-full h-full flex flex-col justify-start items-start">
      <div className="w-full px-1">
        <HistorySearchInput
          value={searchText}
          onChange={onSearchUpdate}
          placeholder={t("sql-editor.search-history-by-statement")}
        />
      </div>

      <div className="w-full flex flex-col justify-start items-start overflow-y-auto">
        {showLinkedHistory && linkedQueryHistory && (
          <section className="w-full" data-linked-history>
            <div className="w-full flex items-center justify-between px-2 py-1.5 bg-accent/20">
              <div className="flex items-center gap-x-1.5 text-accent">
                <Link2 className="size-3.5" />
                <span className="text-xs font-semibold uppercase tracking-wide">
                  {t("sql-editor.opened-from-link")}
                </span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="h-5 w-5 p-0 text-accent hover:bg-accent/10 hover:text-accent"
                data-dismiss-linked-history
                onClick={() => setLinkedQueryHistory(undefined)}
                aria-label={t("common.close")}
              >
                <X className="size-3.5" />
              </Button>
            </div>
            {renderHistoryRow(linkedQueryHistory, /* highlighted */ true)}
          </section>
        )}

        {showLinkedHistory && recentHistories.length > 0 && (
          <div className="w-full flex items-center px-2 py-1.5 bg-control-bg">
            <span className="text-xs font-semibold uppercase tracking-wide text-control-placeholder">
              {t("sql-editor.recent")}
            </span>
          </div>
        )}

        {recentHistories.map((history) => renderHistoryRow(history))}

        {historyData.nextPageToken && (
          <div className="w-full flex flex-col items-center my-2">
            <Button
              variant="ghost"
              size="sm"
              disabled={loading}
              onClick={() => void fetchHistory()}
            >
              {loading && <Loader2 className="size-4 mr-1 animate-spin" />}
              <span className="textinfolabel">{t("common.load-more")}</span>
            </Button>
          </div>
        )}
      </div>

      {recentHistories.length === 0 &&
        !showLinkedHistory &&
        (loading ? (
          <div className="absolute inset-0 flex items-center justify-center bg-background/75">
            <Loader2 className="size-6 animate-spin text-accent" />
          </div>
        ) : (
          <div className="w-full flex items-center justify-center py-8 textinfolabel">
            {t("sql-editor.no-history-found")}
          </div>
        ))}
    </div>
  );
}
