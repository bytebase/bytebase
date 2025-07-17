import { reactive } from "vue";

const ATTR_SEPARATOR = "-";

const LIST_CACHE = new Map<
  // The cache name.
  string,
  Map<string, StoreListCache>
>();

type StoreListCache = {
  // The timestamp when the cache was last updated.
  timestamp: number;
  // The flag indicating whether the cache is currently being fetched.
  isFetching: boolean;
};

export const useListCache = (namespace: string) => {
  const listCache = (() => {
    const cache = LIST_CACHE.get(namespace);
    if (cache) {
      return cache;
    }
    const newCache = reactive(new Map<string, StoreListCache>());
    LIST_CACHE.set(namespace, newCache);
    return newCache;
  })();

  const getCacheKey = (
    // Other attributes to include in the cache key, e.g. "active".
    ...attrs: string[]
  ) => {
    let key = namespace;
    for (const attr of attrs.filter(Boolean)) {
      key += `${ATTR_SEPARATOR}${attr}`;
    }
    return key;
  };

  const getCache = (key: string) => {
    const keyParts = key.split(ATTR_SEPARATOR);
    let cacheKey = "";
    for (const part of keyParts) {
      cacheKey += `${part}`;
      if (listCache.has(cacheKey)) {
        return listCache.get(cacheKey);
      }
      cacheKey += ATTR_SEPARATOR;
    }
    return undefined;
  };

  const deleteCache = (attr: string) => {
    listCache.delete(getCacheKey(attr));
  };

  const clearCache = () => {
    LIST_CACHE.delete(namespace);
  };

  return {
    cacheMap: listCache,
    getCacheKey,
    getCache,
    clearCache,
    deleteCache,
  };
};
