import { useCallback, useSyncExternalStore } from "react";
import type { ReactRoute } from "@/react/router";
import { router } from "@/react/router";

// `router.afterEach` returns its own unregister fn — wraps it as a React
// `useSyncExternalStore` subscription so route changes trigger re-renders
// off the react-router data router.
const subscribe = (onChange: () => void): (() => void) =>
  router.afterEach(() => onChange());

const getSnapshot = (): ReactRoute => router.currentRoute.value;

/**
 * Reactively reads the current route, re-rendering on navigation (URL
 * changes, param updates). Backed by the react-router data router via the
 * `@/react/router` shim.
 */
export const useVueRoute = (): ReactRoute => {
  const subscribeFn = useCallback(subscribe, []);
  return useSyncExternalStore(subscribeFn, getSnapshot, getSnapshot);
};
