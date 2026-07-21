import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useReducer,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  getPageSizeOptions,
  useSessionPageSize,
} from "@/hooks/useSessionPageSize";
import {
  deletePagedDataCache,
  type PagedDataCacheSnapshot,
  readPagedDataCache,
  writePagedDataCache,
} from "./pagedDataCache";

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
  cacheKey?: string;
  cacheWriteState: "clean" | "dirty";
  dataList: T[];
  status: "idle" | "loading" | "ready" | "loadingMore";
  hasMore: boolean;
};

type PagedDataAction<T> =
  | {
      type: "restore-cache";
      cacheKey?: string;
      snapshot?: PagedDataCacheSnapshot<T>;
    }
  | { type: "fetch-start"; mode: FetchMode; cacheKey?: string }
  | {
      type: "fetch-success";
      mode: FetchMode;
      cacheKey?: string;
      list: T[];
      hasMore: boolean;
    }
  | { type: "fetch-error" }
  | { type: "update-cache"; items: T[] }
  | { type: "remove-cache"; item: T }
  | { type: "cache-persisted"; cacheKey?: string };

const stateFromSnapshot = <T extends { name: string }>(
  cacheKey: string | undefined,
  snapshot?: PagedDataCacheSnapshot<T>
): PagedDataState<T> => ({
  cacheKey,
  cacheWriteState: "clean",
  dataList: snapshot?.dataList ?? [],
  status: snapshot ? "ready" : "idle",
  hasMore: snapshot?.hasMore ?? false,
});

const pagedDataReducer = <T extends { name: string }>(
  state: PagedDataState<T>,
  action: PagedDataAction<T>
): PagedDataState<T> => {
  switch (action.type) {
    case "restore-cache":
      return stateFromSnapshot(action.cacheKey, action.snapshot);
    case "fetch-start":
      return {
        ...state,
        cacheKey: action.cacheKey,
        status: action.mode === "refresh" ? "loading" : "loadingMore",
      };
    case "fetch-success":
      return {
        cacheKey: action.cacheKey,
        cacheWriteState: "dirty",
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
        cacheWriteState: "clean",
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
        cacheWriteState: "dirty",
        dataList,
      };
    }
    case "remove-cache":
      return {
        ...state,
        cacheWriteState: "dirty",
        dataList: state.dataList.filter(
          (data) => data.name !== action.item.name
        ),
      };
    case "cache-persisted":
      return action.cacheKey === state.cacheKey
        ? { ...state, cacheWriteState: "clean" }
        : state;
  }
};

export function usePagedData<T extends { name: string }>({
  sessionKey,
  cacheKey,
  cacheScope,
  cacheRestoreToken,
  fetchList,
  enabled = true,
}: {
  sessionKey: string;
  cacheKey?: string;
  cacheScope?: string;
  cacheRestoreToken?: string;
  fetchList: (params: {
    pageSize: number;
    pageToken: string;
  }) => Promise<{ list: T[]; nextPageToken?: string }>;
  enabled?: boolean;
}): PagedDataResult<T> {
  const [pageSize, setPageSize] = useSessionPageSize(sessionKey);
  const pageSizeOptions = getPageSizeOptions();

  const resolvedCacheKey = cacheKey
    ? `${cacheKey}:page-size=${pageSize}`
    : undefined;
  const [initialCache] = useState(() => {
    const snapshot = cacheRestoreToken
      ? readPagedDataCache<T>(resolvedCacheKey)
      : undefined;
    return {
      key: resolvedCacheKey,
      cacheRestoreToken,
      snapshot,
    };
  });
  const [state, dispatch] = useReducer(
    pagedDataReducer<T>,
    initialCache,
    ({ key, snapshot }) => stateFromSnapshot(key, snapshot)
  );

  const abortRef = useRef<AbortController | null>(null);
  const fetchIdRef = useRef(0);
  const nextPageTokenRef = useRef(initialCache.snapshot?.nextPageToken ?? "");
  const activeCacheKeyRef = useRef(initialCache.key);
  const activeCacheRestoreTokenRef = useRef(initialCache.cacheRestoreToken);
  const consumedCacheRestoreTokenRef = useRef(initialCache.cacheRestoreToken);
  const skipNextFetchRef = useRef(Boolean(initialCache.snapshot));

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      const mode: FetchMode = isRefresh ? "refresh" : "append";
      const activeCacheKey = activeCacheKeyRef.current;
      dispatch({ type: "fetch-start", mode, cacheKey: activeCacheKey });

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await fetchList({ pageSize, pageToken: token });
        if (controller.signal.aborted || currentFetchId !== fetchIdRef.current)
          return;
        nextPageTokenRef.current = result.nextPageToken ?? "";
        dispatch({
          type: "fetch-success",
          mode,
          cacheKey: activeCacheKey,
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
    deletePagedDataCache(activeCacheKeyRef.current);
    doFetch(true);
  }, [doFetch]);

  const updateCache = useCallback((items: T[]) => {
    dispatch({ type: "update-cache", items });
  }, []);

  const removeCache = useCallback((item: T) => {
    dispatch({ type: "remove-cache", item });
  }, []);

  useLayoutEffect(() => {
    if (
      activeCacheKeyRef.current === resolvedCacheKey &&
      activeCacheRestoreTokenRef.current === cacheRestoreToken
    ) {
      return;
    }
    fetchIdRef.current++;
    abortRef.current?.abort();
    activeCacheKeyRef.current = resolvedCacheKey;
    activeCacheRestoreTokenRef.current = cacheRestoreToken;

    const shouldRestore =
      cacheRestoreToken !== undefined &&
      consumedCacheRestoreTokenRef.current !== cacheRestoreToken;
    consumedCacheRestoreTokenRef.current = cacheRestoreToken;
    const snapshot = shouldRestore
      ? readPagedDataCache<T>(resolvedCacheKey)
      : undefined;
    nextPageTokenRef.current = snapshot?.nextPageToken ?? "";
    skipNextFetchRef.current = Boolean(snapshot);
    dispatch({
      type: "restore-cache",
      cacheKey: resolvedCacheKey,
      snapshot,
    });
  }, [cacheRestoreToken, resolvedCacheKey]);

  useEffect(() => {
    if (
      !state.cacheKey ||
      state.cacheWriteState !== "dirty" ||
      state.status !== "ready" ||
      state.cacheKey !== resolvedCacheKey
    ) {
      return;
    }
    writePagedDataCache(
      state.cacheKey,
      {
        dataList: state.dataList,
        hasMore: state.hasMore,
        nextPageToken: nextPageTokenRef.current,
      },
      cacheScope
    );
    dispatch({ type: "cache-persisted", cacheKey: state.cacheKey });
  }, [cacheScope, resolvedCacheKey, state]);

  // Fetch on mount and when fetchList/pageSize changes (handles search text reactivity)
  const isFirstLoad = useRef(true);
  useEffect(() => {
    if (!enabled) return;
    if (skipNextFetchRef.current) {
      isFirstLoad.current = false;
      // Keep the guard through React Strict Mode's immediate effect replay.
      const timer = window.setTimeout(() => {
        skipNextFetchRef.current = false;
      });
      return () => window.clearTimeout(timer);
    }
    if (isFirstLoad.current) {
      isFirstLoad.current = false;
      doFetch(true);
      return;
    }
    fetchIdRef.current++;
    abortRef.current?.abort();
    dispatch({
      type: "fetch-start",
      mode: "refresh",
      cacheKey: activeCacheKeyRef.current,
    });
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [cacheRestoreToken, doFetch, enabled, resolvedCacheKey]);

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
        <Select
          value={String(pageSize)}
          onValueChange={(value) => onPageSizeChange(Number(value))}
        >
          <SelectTrigger size="sm" className="min-w-[5rem] text-control">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {pageSizeOptions.map((opt) => (
              <SelectItem key={opt} value={String(opt)}>
                {opt}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {hasMore && (
        <Button
          appearance="secondary"
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
