import { sortBy, uniq } from "lodash-es";
import { useCallback, useState } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useCurrentUserV1 } from "@/store";
import { getDefaultPagination } from "@/utils";

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500];

export function getPageSizeOptions(): number[] {
  const defaultSize = getDefaultPagination();
  return sortBy(uniq([defaultSize, ...PAGE_SIZE_OPTIONS]));
}

export function useSessionPageSize(
  sessionKey: string
): [number, (size: number) => void] {
  const currentUser = useCurrentUserV1();
  const email = useVueState(() => currentUser.value.email);
  const storageKey = `bb.paged-table.${sessionKey}.${email}`;

  const [pageSize, setPageSize] = useState<number>(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored);
        const size = parsed?.pageSize;
        const options = getPageSizeOptions();
        if (typeof size === "number" && options.includes(size)) {
          return Math.max(options[0], size);
        }
      }
    } catch {
      // ignore
    }
    return getPageSizeOptions()[0];
  });

  const updatePageSize = useCallback(
    (size: number) => {
      setPageSize(size);
      try {
        localStorage.setItem(storageKey, JSON.stringify({ pageSize: size }));
      } catch {
        // ignore
      }
    },
    [storageKey]
  );

  return [pageSize, updatePageSize];
}
