import { sortBy, uniq } from "lodash-es";
import { useCallback, useEffect, useRef, useState } from "react";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { getDefaultPagination } from "@/utils";

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500];

export function getPageSizeOptions(): number[] {
  const defaultSize = getDefaultPagination();
  return sortBy(uniq([defaultSize, ...PAGE_SIZE_OPTIONS]));
}

function readStoredPageSize(storageKey: string): number {
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
}

export function useSessionPageSize(
  sessionKey: string
): [number, (size: number) => void] {
  const email = useCurrentUser().email;
  const storageKey = `bb.paged-table.${sessionKey}.${email}`;

  const [pageSize, setPageSize] = useState<number>(() =>
    readStoredPageSize(storageKey)
  );

  // `email` is empty until the current user loads asynchronously, so the
  // initial read can hit the wrong key. Re-read the persisted size once the
  // key (email) resolves.
  const loadedKeyRef = useRef(storageKey);
  useEffect(() => {
    if (loadedKeyRef.current !== storageKey) {
      loadedKeyRef.current = storageKey;
      setPageSize(readStoredPageSize(storageKey));
    }
  }, [storageKey]);

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
