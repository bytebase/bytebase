import { computed, unref, watch, type MaybeRef } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { DEFAULT_PAGE_SIZE } from "../common";
import { useListCache } from "./cache";
import {
  instanceNamePrefix,
  projectNamePrefix,
  workspaceNamePrefix,
} from "./common";
import { useDatabaseV1Store } from "./database";

const formatDatabaseParent = (parent: string) => {
  if (
    parent.startsWith(workspaceNamePrefix) ||
    parent.startsWith(projectNamePrefix) ||
    parent.startsWith(instanceNamePrefix)
  ) {
    return parent;
  }
  // Otherwise, list all databases in the workspace.
  return `${workspaceNamePrefix}-`;
};

export const useDatabaseV1List = (
  // The parent name of databases to list.
  // * Leave empty or `workspaces/` to list all databases in workspace.
  // * `projects/{project}` to list all databases in the project.
  // * `instances/{instance}` to list all databases in the instance.
  parent?: MaybeRef<string>
) => {
  const listCache = useListCache("database");
  const store = useDatabaseV1Store();
  const formatParent = computed(() =>
    formatDatabaseParent(unref(parent) || "")
  );
  const cacheKey = computed(() => listCache.getCacheKey(formatParent.value));
  const cache = computed(() => listCache.getCache(cacheKey.value));

  watch(
    [() => cache.value],
    async () => {
      // Skip if request is already in progress or cache is available.
      if (cache.value?.isFetching || cache.value) {
        return;
      }

      listCache.cacheMap.set(cacheKey.value, {
        timestamp: Date.now(),
        isFetching: true,
      });
      const request = formatParent.value.startsWith(instanceNamePrefix)
        ? databaseServiceClient.listInstanceDatabases
        : databaseServiceClient.listDatabases;
      const { databases } = await request({
        parent: formatParent.value,
        pageSize: DEFAULT_PAGE_SIZE,
      });
      if (formatParent.value.startsWith(instanceNamePrefix)) {
        store.removeCacheByInstance(formatParent.value);
      }
      await store.upsertDatabaseMap(databases);
      listCache.cacheMap.set(cacheKey.value, {
        timestamp: Date.now(),
        isFetching: false,
      });
    },
    {
      deep: true,
      immediate: true,
    }
  );

  const databaseList = computed(() => {
    if (formatParent.value.startsWith(projectNamePrefix)) {
      return store.databaseListByProject(formatParent.value);
    } else if (formatParent.value.startsWith(instanceNamePrefix)) {
      return store.databaseListByInstance(formatParent.value);
    }
    // Otherwise, list all databases in the workspace.
    return store.databaseList;
  });

  return {
    databaseList,
    listCache,
    ready: computed(() => cache.value && !cache.value.isFetching),
  };
};
