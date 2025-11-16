import { sortBy } from "lodash-es";
import { type ComputedRef, computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { useDynamicLocalStorage } from "@/utils";
import type { SheetViewMode } from "./types";

export const useFolderByView = (
  viewMode: SheetViewMode,
  project: ComputedRef<string>
) => {
  const me = useCurrentUserV1();

  const rootPath = computed(() => `/${viewMode}`);
  const localCacheKey = computed(
    () =>
      `bb.sql-editor.${project.value}.worksheet-folder.${viewMode}.${me.value.name}`
  );
  const localCache = useDynamicLocalStorage<Set<string>>(
    localCacheKey,
    new Set([rootPath.value])
  );
  localCache.value.add(rootPath.value);

  const folders = computed(() => sortBy([...localCache.value]));

  // A valid folder path should be like "/xxx"
  // It must not end with "/", for example, "xxx/" is not valid
  // It must starts with "/", like "/xxx"
  const ensureFolderPath = (path: string) => {
    let p = path;
    while (p.startsWith("/")) {
      p = p.slice(1);
    }
    while (p.endsWith("/")) {
      p = p.slice(0, -1);
    }
    if (!p) {
      return rootPath.value;
    }
    if (!p.startsWith("/")) {
      p = `/${p}`;
    }
    return p;
  };

  const addFolder = (path: string) => {
    const newPath = ensureFolderPath(path);
    localCache.value.add(newPath);
    return newPath;
  };

  const isSubFolder = ({
    parent,
    path,
    dig,
  }: {
    parent: string;
    path: string;
    dig: boolean;
  }) => {
    const parentPrefix = `${parent}/`;
    return (
      // ensure not self
      path !== parentPrefix &&
        // ensure is subfolder
        path.startsWith(parentPrefix) &&
        dig
        ? true
        : !path.replace(parentPrefix, "").includes("/")
    );
  };

  const removeFolder = (path: string) => {
    localCache.value = new Set(
      [...localCache.value].filter(
        (value) =>
          !(
            value == path ||
            isSubFolder({ parent: path, path: value, dig: true })
          )
      )
    );
  };

  const moveFolder = (from: string, to: string) => {
    const fromPath = ensureFolderPath(from);
    const toPath = ensureFolderPath(to);

    const pendingUpdatePathes = [];
    for (const path of localCache.value.values()) {
      if (
        path === fromPath ||
        isSubFolder({ parent: fromPath, path, dig: true })
      ) {
        pendingUpdatePathes.push({
          old: path,
          new: path.replace(fromPath, toPath),
        });
      }
    }

    const newSet = new Set(localCache.value);
    for (const item of pendingUpdatePathes) {
      newSet.delete(item.old);
      newSet.add(item.new);
    }
    localCache.value = newSet;
  };

  const mergeFolders = (pathes: Set<string>) => {
    const newSet = new Set(localCache.value);
    for (const folder of pathes.values()) {
      const validPath = ensureFolderPath(folder);
      newSet.add(validPath);
    }
    localCache.value = newSet;
  };

  const listSubFolders = (parent: string) => {
    return folders.value.filter((path) => {
      return isSubFolder({ parent, path, dig: false });
    });
  };

  return {
    rootPath,
    folders,
    listSubFolders,
    ensureFolderPath,
    addFolder,
    removeFolder,
    moveFolder,
    mergeFolders,
    isSubFolder,
  };
};
