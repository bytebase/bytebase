import { useCallback, useRef, useState } from "react";

interface Page<T> {
  items: T[];
  nextPageToken?: string;
}

interface UsePaginatedSelectOptions<T> {
  fetchPage: (query: string, pageToken: string) => Promise<Page<T>>;
}

export function usePaginatedSelect<T extends { name: string }>({
  fetchPage,
}: UsePaginatedSelectOptions<T>) {
  const [items, setItems] = useState<T[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);
  const queryRef = useRef("");
  const requestGenerationRef = useRef(0);

  const search = useCallback(
    async (query: string) => {
      const generation = ++requestGenerationRef.current;
      queryRef.current = query;
      setNextPageToken("");
      setLoadingMore(false);
      try {
        const response = await fetchPage(query, "");
        if (generation !== requestGenerationRef.current) return;
        setItems(response.items);
        setNextPageToken(response.nextPageToken ?? "");
      } catch {
        // Keep the current options when a search request fails.
      }
    },
    [fetchPage]
  );

  const loadMore = useCallback(async () => {
    if (!nextPageToken || loadingMore) return;
    const generation = requestGenerationRef.current;
    setLoadingMore(true);
    try {
      const response = await fetchPage(queryRef.current, nextPageToken);
      if (generation !== requestGenerationRef.current) return;
      setItems((previous) => {
        const names = new Set(previous.map((item) => item.name));
        return [
          ...previous,
          ...response.items.filter((item) => !names.has(item.name)),
        ];
      });
      setNextPageToken(response.nextPageToken ?? "");
    } catch {
      // Keep the current page and token so the request can be retried.
    } finally {
      if (generation === requestGenerationRef.current) {
        setLoadingMore(false);
      }
    }
  }, [fetchPage, loadingMore, nextPageToken]);

  return {
    items,
    search,
    hasMore: Boolean(nextPageToken),
    loadingMore,
    loadMore,
  };
}
