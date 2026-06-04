import { useSyncExternalStore } from "react";
import type { ReactRoute } from "@/react/router";
import { router } from "@/react/router";

// `router.afterEach` returns its own unregister fn — wraps it as a React
// `useSyncExternalStore` subscription so route changes trigger re-renders
// off the react-router data router.
const subscribe = (onChange: () => void): (() => void) =>
  router.afterEach(() => onChange());

// `router.currentRoute.value` is memoized at the router level — it returns a
// referentially stable object between actual route changes — so it satisfies
// `useSyncExternalStore`'s requirement that `getSnapshot` be cached (otherwise
// React re-renders forever: "The result of getSnapshot should be cached").
const getSnapshot = (): ReactRoute => router.currentRoute.value;

/**
 * Reactively reads the current route, re-rendering on navigation (URL
 * changes, param updates). Backed by the react-router data router via the
 * `@/react/router` shim.
 */
export const useReactiveRoute = (): ReactRoute => {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
};
