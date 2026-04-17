import { useCallback, useEffect, useReducer, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/react/hooks/useSessionPageSize";

// ============================================================
// usePagedData hook
// ============================================================

export type PagedDataResult<T> = {
  dataList: T[];
  isLoading: boolean;
  isFetchingMore: boolean;
  hasMore: boolean;
  loadMore: () => void;
  refresh: () => void;
  updateCache: (items: T[]) => void;
  removeCache: (item: T) => void;
  pageSize: number;
  pageSizeOptions: number[];
  onPageSizeChange: (size: number) => void;
};

type FetchMode = "refresh" | "append";

type PagedDataState<T> = {
  dataList: T[];
  status: "idle" | "loading" | "ready" | "loadingMore";
  hasMore: boolean;
};

type PagedDataAction<T> =
  | { type: "fetch-start"; mode: FetchMode }
  | {
      type: "fetch-success";
      mode: FetchMode;
      list: T[];
      hasMore: boolean;
    }
  | { type: "fetch-error" }
  | { type: "update-cache"; items: T[] }
  | { type: "remove-cache"; item: T };

const initialPagedDataState = <
  T extends { name: string },
>(): PagedDataState<T> => ({
  dataList: [],
  status: "idle",
  hasMore: false,
});

const pagedDataReducer = <T extends { name: string }>(
  state: PagedDataState<T>,
  action: PagedDataAction<T>
): PagedDataState<T> => {
  switch (action.type) {
    case "fetch-start":
      return {
        ...state,
        status: action.mode === "refresh" ? "loading" : "loadingMore",
      };
    case "fetch-success":
      return {
        dataList:
          action.mode === "refresh"
            ? action.list
            : [...state.dataList, ...action.list],
        status: "ready",
        hasMore: action.hasMore,
      };
    case "fetch-error":
      return {
        ...state,
        status: "ready",
      };
    case "update-cache": {
      const dataList = [...state.dataList];
      for (const item of action.items) {
        const index = dataList.findIndex((data) => data.name === item.name);
        if (index >= 0) {
          dataList[index] = item;
        } else {
          dataList.push(item);
        }
      }
      return {
        ...state,
        dataList,
      };
    }
    case "remove-cache":
      return {
        ...state,
        dataList: state.dataList.filter(
          (data) => data.name !== action.item.name
        ),
      };
  }
};

export function usePagedData<T extends { name: string }>({
  sessionKey,
  fetchList,
  enabled = true,
}: {
  sessionKey: string;
  fetchList: (params: {
    pageSize: number;
    pageToken: string;
  }) => Promise<{ list: T[]; nextPageToken?: string }>;
  enabled?: boolean;
}): PagedDataResult<T> {
  const [pageSize, setPageSize] = useSessionPageSize(sessionKey);
  const pageSizeOptions = getPageSizeOptions();

  const [state, dispatch] = useReducer(
    pagedDataReducer<T>,
    undefined,
    initialPagedDataState<T>
  );

  const abortRef = useRef<AbortController | null>(null);
  const fetchIdRef = useRef(0);
  const nextPageTokenRef = useRef("");

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      const mode: FetchMode = isRefresh ? "refresh" : "append";
      dispatch({ type: "fetch-start", mode });

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await fetchList({ pageSize, pageToken: token });
        if (controller.signal.aborted || currentFetchId !== fetchIdRef.current)
          return;
        nextPageTokenRef.current = result.nextPageToken ?? "";
        dispatch({
          type: "fetch-success",
          mode,
          list: result.list,
          hasMore: Boolean(result.nextPageToken),
        });
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
        if (currentFetchId === fetchIdRef.current) {
          dispatch({ type: "fetch-error" });
        }
      }
    },
    [fetchList, pageSize]
  );

  const loadMore = useCallback(() => {
    if (state.hasMore && state.status === "ready") {
      doFetch(false);
    }
  }, [doFetch, state.hasMore, state.status]);

  const refresh = useCallback(() => {
    doFetch(true);
  }, [doFetch]);

  const updateCache = useCallback((items: T[]) => {
    dispatch({ type: "update-cache", items });
  }, []);

  const removeCache = useCallback((item: T) => {
    dispatch({ type: "remove-cache", item });
  }, []);

  // Fetch on mount and when fetchList/pageSize changes (handles search text reactivity)
  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (!enabled) return;
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      doFetch(true);
      return;
    }
    fetchIdRef.current++;
    abortRef.current?.abort();
    dispatch({ type: "fetch-start", mode: "refresh" });
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch, enabled]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return {
    dataList: state.dataList,
    isLoading:
      enabled && (state.status === "idle" || state.status === "loading"),
    isFetchingMore: state.status === "loadingMore",
    hasMore: state.hasMore,
    loadMore,
    refresh,
    updateCache,
    removeCache,
    pageSize,
    pageSizeOptions,
    onPageSizeChange: setPageSize,
  };
}

// ============================================================
// PagedTableFooter
// ============================================================

export function PagedTableFooter({
  pageSize,
  pageSizeOptions,
  onPageSizeChange,
  hasMore,
  isFetchingMore,
  onLoadMore,
}: {
  pageSize: number;
  pageSizeOptions: number[];
  onPageSizeChange: (size: number) => void;
  hasMore: boolean;
  isFetchingMore: boolean;
  onLoadMore: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-end gap-x-2">
      <div className="flex items-center gap-x-2">
        <span className="text-sm text-control-light">
          {t("common.rows-per-page")}
        </span>
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          className="border border-control-border rounded-sm text-sm pl-2 pr-6 py-1 min-w-[5rem]"
        >
          {pageSizeOptions.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
      </div>
      {hasMore && (
        <Button
          variant="ghost"
          size="sm"
          disabled={isFetchingMore}
          onClick={onLoadMore}
        >
          <span className="text-sm text-control-light">
            {isFetchingMore ? t("common.loading") : t("common.load-more")}
          </span>
        </Button>
      )}
    </div>
  );
}
