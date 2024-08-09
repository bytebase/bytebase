export type StoreCache = {
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
