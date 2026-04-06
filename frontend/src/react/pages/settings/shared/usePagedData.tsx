import { useCallback, useEffect, useRef, useState } from "react";
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

  const [dataList, setDataList] = useState<T[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [hasFetched, setHasFetched] = useState(false);
  const [hasMore, setHasMore] = useState(false);

  const abortRef = useRef<AbortController | null>(null);
  const fetchIdRef = useRef(0);
  const nextPageTokenRef = useRef("");

  const doFetch = useCallback(
    async (isRefresh: boolean) => {
      const currentFetchId = ++fetchIdRef.current;
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;

      if (isRefresh) {
        setIsLoading(true);
      } else {
        setIsFetchingMore(true);
      }

      try {
        const token = isRefresh ? "" : nextPageTokenRef.current;
        const result = await fetchList({ pageSize, pageToken: token });
        if (controller.signal.aborted || currentFetchId !== fetchIdRef.current)
          return;
        setDataList((prev) =>
          isRefresh ? result.list : [...prev, ...result.list]
        );
        nextPageTokenRef.current = result.nextPageToken ?? "";
        setHasMore(Boolean(result.nextPageToken));
      } catch (e) {
        if (e instanceof Error && e.name === "AbortError") return;
        console.error(e);
      } finally {
        if (currentFetchId === fetchIdRef.current) {
          setIsLoading(false);
          setIsFetchingMore(false);
          setHasFetched(true);
        }
      }
    },
    [fetchList, pageSize]
  );

  const loadMore = useCallback(() => {
    if (hasMore && !isFetchingMore) {
      doFetch(false);
    }
  }, [hasMore, isFetchingMore, doFetch]);

  const refresh = useCallback(() => {
    doFetch(true);
  }, [doFetch]);

  const updateCache = useCallback((items: T[]) => {
    setDataList((prev) => {
      const next = [...prev];
      for (const item of items) {
        const idx = next.findIndex((d) => d.name === item.name);
        if (idx >= 0) {
          next[idx] = item;
        } else {
          next.push(item);
        }
      }
      return next;
    });
  }, []);

  const removeCache = useCallback((item: T) => {
    setDataList((prev) => prev.filter((d) => d.name !== item.name));
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
    const timer = setTimeout(() => doFetch(true), 300);
    return () => clearTimeout(timer);
  }, [doFetch, enabled]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return {
    dataList,
    isLoading: enabled && (isLoading || !hasFetched),
    isFetchingMore,
    hasMore,
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
