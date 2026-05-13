import dayjs from "dayjs";
import { Copy, Loader2 } from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { SearchInput } from "@/react/components/ui/search-input";
import { useVueState } from "@/react/hooks/useVueState";
import type { QueryHistoryFilter } from "@/store";
import {
  pushNotification,
  useSQLEditorQueryHistoryStore,
  useSQLEditorStore,
  useSQLEditorTabStore,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY, getDateForPbTimestampProtoEs } from "@/types";
import type { QueryHistory } from "@/types/proto-es/v1/sql_service_pb";
import { getHighlightHTMLByKeyWords } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";

/**
 * React migration of frontend/src/views/sql-editor/AsidePanel/HistoryPane/HistoryPane.vue.
 * Displays the query history list with search, copy, and click-to-append features.
 */
export function HistoryPane() {
  const { t } = useTranslation();

  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorStore();
  const queryHistoryStore = useSQLEditorQueryHistoryStore();

  const [searchText, setSearchText] = useState("");
  const [loading, setLoading] = useState(false);
  // Bumped when `query-executed` fires; forces React to re-read the
  // store cache (which `useExecuteSQL` / `webTerminal` have already
  // refreshed via `mergeLatest` by the time we get the event).
  const [, bumpRefresh] = useReducer((c: number) => c + 1, 0);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const currentTabDatabase = useVueState(
    () => tabStore.currentTab?.connection.database
  );
  const project = useVueState(() => editorStore.project);

  const historyQuery = useMemo<QueryHistoryFilter>(
    () => ({
      database: currentTabDatabase,
      project: project,
      statement: searchText,
    }),
    [currentTabDatabase, project, searchText]
  );

  const historyData = useVueState(() =>
    queryHistoryStore.getQueryHistoryList(historyQuery)
  );

  const fetchHistory = useCallback(async () => {
    setLoading(true);
    try {
      await queryHistoryStore.fetchQueryHistoryList(historyQuery);
    } finally {
      setLoading(false);
    }
  }, [queryHistoryStore, historyQuery]);

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
  // re-runs the `useVueState` getter and reads the merged list —
  // preserving any pages the user had already loaded.
  useEffect(() => {
    sqlEditorEvents.on("query-executed", bumpRefresh);
    return () => {
      sqlEditorEvents.off("query-executed", bumpRefresh);
    };
  }, []);

  const onSearchUpdate = useCallback(
    (next: string) => {
      queryHistoryStore.resetPageToken({ ...historyQuery, statement: next });
      setSearchText(next);
    },
    [queryHistoryStore, historyQuery]
  );

  const titleOfQueryHistory = (h: QueryHistory) =>
    dayjs(getDateForPbTimestampProtoEs(h.createTime)).format(
      "YYYY-MM-DD HH:mm:ss"
    );

  const handleHistoryClick = async (history: QueryHistory) => {
    const { statement } = history;
    if (tabStore.currentTab) {
      await sqlEditorEvents.emit("append-editor-content", {
        content: statement,
        select: true,
      });
    } else {
      tabStore.addTab(
        {
          title: `Query history at ${titleOfQueryHistory(history)}`,
          statement,
        },
        /* beside */ true
      );
    }
  };

  const handleCopy = async (statement: string) => {
    try {
      await navigator.clipboard.writeText(statement);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } catch {
      // clipboard not available
    }
  };

  return (
    <div className="relative w-full h-full flex flex-col justify-start items-start">
      <div className="w-full px-1">
        <SearchInput
          value={searchText}
          onChange={(e) => onSearchUpdate(e.target.value)}
          placeholder={t("sql-editor.search-history-by-statement")}
          className="h-8"
        />
      </div>

      <div className="w-full flex flex-col justify-start items-start overflow-y-auto">
        {historyData.queryHistories.map((history) => (
          <div
            key={history.name}
            data-history-row
            className="w-full p-2 gap-y-1 border-b flex flex-col justify-start items-start cursor-pointer hover:bg-control-bg"
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
              <Button
                variant="ghost"
                size="sm"
                className="h-6 w-6 p-0"
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
            <p
              className="max-w-full text-xs wrap-break-word font-mono line-clamp-3"
              dangerouslySetInnerHTML={{
                __html: getHighlightHTMLByKeyWords(
                  history.statement,
                  searchText
                ),
              }}
            />
          </div>
        ))}

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

      {historyData.queryHistories.length === 0 &&
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
