export type PagedDataCacheSnapshot<T> = {
  dataList: T[];
  hasMore: boolean;
  nextPageToken: string;
};

type StoredSnapshot<T> = PagedDataCacheSnapshot<T> & {
  cachedAt: number;
};

const MAX_ENTRIES = 20;
const MAX_AGE_MS = 5 * 60 * 1000;
const cache = new Map<string, StoredSnapshot<unknown>>();

// List arrays are copied so consumers cannot change membership/order in the
// stored view. Items remain shared under the app's immutable proto convention.
const cloneSnapshot = <T>(
  snapshot: PagedDataCacheSnapshot<T>
): PagedDataCacheSnapshot<T> => ({
  dataList: [...snapshot.dataList],
  hasMore: snapshot.hasMore,
  nextPageToken: snapshot.nextPageToken,
});

export function readPagedDataCache<T>(
  key: string | undefined
): PagedDataCacheSnapshot<T> | undefined {
  if (!key) return undefined;
  const stored = cache.get(key) as StoredSnapshot<T> | undefined;
  if (!stored) return undefined;
  if (Date.now() - stored.cachedAt > MAX_AGE_MS) {
    cache.delete(key);
    return undefined;
  }

  // Refresh Map insertion order for LRU eviction without extending expiry.
  cache.delete(key);
  cache.set(key, stored as StoredSnapshot<unknown>);
  return cloneSnapshot(stored);
}

export function writePagedDataCache<T>(
  key: string | undefined,
  snapshot: PagedDataCacheSnapshot<T>
): void {
  if (!key) return;
  cache.delete(key);
  cache.set(key, {
    ...cloneSnapshot(snapshot),
    cachedAt: Date.now(),
  } as StoredSnapshot<unknown>);

  while (cache.size > MAX_ENTRIES) {
    const oldestKey = cache.keys().next().value as string | undefined;
    if (!oldestKey) break;
    cache.delete(oldestKey);
  }
}

export function deletePagedDataCache(key: string | undefined): void {
  if (key) cache.delete(key);
}

export function clearPagedDataCache(): void {
  cache.clear();
}
