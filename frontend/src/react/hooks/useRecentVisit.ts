import { useCallback } from "react";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { storageKeyRecentVisit, workspaceCacheScope } from "@/utils";

const MAX_HISTORY = 10;

// Two URLs are "the same" visit when their paths match (ignoring querystring
// and hash) — usually just different tab-panes or filters on the same page.
function getPath(url: string): string {
  return url.replace(/[?#].*$/, "");
}

// React port of the Vue `@/router/useRecentVisit` composable. Maintains the
// per-user recent-visit list in localStorage under the same scoped key + JSON
// array format as the Vue version (`@vueuse` useStorage), so the two remain
// interoperable during the migration.
export function useRecentVisit() {
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());
  const currentUser = useAppStore((s) => s.currentUser);
  const storageKey = storageKeyRecentVisit(
    workspaceCacheScope(isSaaSMode, currentUser?.workspace ?? ""),
    currentUser?.email ?? ""
  );

  const read = useCallback((): string[] => {
    try {
      const raw = localStorage.getItem(storageKey);
      const parsed = raw ? JSON.parse(raw) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }, [storageKey]);

  const write = useCallback(
    (list: string[]) => {
      localStorage.setItem(storageKey, JSON.stringify(list));
    },
    [storageKey]
  );

  const remove = useCallback(
    (path: string) => {
      const list = read();
      const index = list.findIndex((item) => getPath(item) === getPath(path));
      if (index >= 0) {
        list.splice(index, 1);
        write(list);
      }
    },
    [read, write]
  );

  const record = useCallback(
    (path: string) => {
      const list = read();
      // Pull out an existing entry for the current page before re-inserting.
      const index = list.findIndex((item) => getPath(item) === getPath(path));
      if (index >= 0) {
        list.splice(index, 1);
      }
      // Current page is always first, so cap at MAX_HISTORY + 1.
      while (list.length > MAX_HISTORY + 1) {
        list.pop();
      }
      list.unshift(path);
      write(list);
    },
    [read, write]
  );

  const list = read();
  const lastVisit = list.length > 0 ? list[0] : undefined;
  const lastVisitProjectPath =
    list.find((visit) => visit.startsWith(`/${projectNamePrefix}`)) ?? "";

  return { remove, record, lastVisit, lastVisitProjectPath };
}
