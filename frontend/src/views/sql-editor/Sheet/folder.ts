import { sortBy, uniq } from "lodash-es";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { useDynamicLocalStorage } from "@/utils";
import type { SheetViewMode } from "./types";

export const useFolderByView = (viewMode: SheetViewMode) => {
  const me = useCurrentUserV1();
  const rootPath = computed(() => `/${viewMode}`);
  const localCacheKey = computed(
    () => `bb.sql-editor.worksheet-folder.${viewMode}.${me.value.name}`
  );
  const localCache = useDynamicLocalStorage<Set<string>>(
    localCacheKey,
    new Set()
  );

  const folders = computed(() => sortBy(uniq([...localCache.value])));

  // Must not end with "/" for non-root, for example, "xxx/" is not valid
  // Must starts with "/", like "/xxx"
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
    // if (parent === "/") {
    //   parentPrefix = parent;
    // }

    return (
      // ensure not self (specially for "/")
      // path !== parentPrefix &&
      // ensure is subfolder
      path.startsWith(parentPrefix) && dig
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

    for (const item of pendingUpdatePathes) {
      localCache.value.delete(item.old);
      localCache.value.add(item.new);
    }
  };

  const mergeFolders = (pathes: string[]) => {
    for (const folder of pathes) {
      const validPath = ensureFolderPath(folder);
      localCache.value.add(validPath);
    }
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
