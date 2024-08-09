export type StoreCache = {
  // The timestamp when the cache was last updated.
  timestamp: number;
  // The flag indicating whether the cache is currently being fetched.
  isFetching: boolean;
};

export const getResourceStoreCacheKey = (
  // The resource type.
  resource: "project" | "instance" | "database",
  // Other attributes to include in the cache key, e.g. "view", "deleted".
  ...attrs: string[]
) => {
  return `${resource}-${attrs.join("-")}`;
};
