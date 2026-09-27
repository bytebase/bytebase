import { useCallback, useSyncExternalStore } from "react";

/**
 * Subscribes to a CSS media query and returns whether it currently matches.
 *
 * Backed by `matchMedia`, so it re-renders only when the match state flips —
 * not on every resize event. SSR-safe: returns `false` when `window` is
 * unavailable.
 *
 * @param query - A media query string, e.g. `"(max-width: 639px)"`.
 *
 * @example
 * const isWide = useMediaQuery("(min-width: 1024px)");
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      if (typeof window === "undefined") return () => {};
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onStoreChange);
      return () => mql.removeEventListener("change", onStoreChange);
    },
    [query]
  );

  const getSnapshot = useCallback(
    () => typeof window !== "undefined" && window.matchMedia(query).matches,
    [query]
  );

  // Server snapshot: assume the query does not match (desktop-first default).
  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}
