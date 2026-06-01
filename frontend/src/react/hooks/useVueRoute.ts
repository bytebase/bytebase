import { useCallback, useSyncExternalStore } from "react";
import type { RouteLocationNormalizedLoaded } from "vue-router";
import { router } from "@/router";

// `router.afterEach` returns its own unregister fn — wraps it as a React
// `useSyncExternalStore` subscription so route changes trigger re-renders
// without dragging Vue's reactivity (`useVueState` / `usePiniaBridge`)
// into the consumer.
const subscribe = (onChange: () => void): (() => void) =>
  router.afterEach(() => onChange());

const getSnapshot = (): RouteLocationNormalizedLoaded =>
  router.currentRoute.value;

/**
 * Reactively reads the current Vue Router route. Use when a React
 * component needs to re-render on navigation (URL changes, param updates)
 * without subscribing to Vue's reactive system.
 */
export const useVueRoute = (): RouteLocationNormalizedLoaded => {
  const subscribeFn = useCallback(subscribe, []);
  return useSyncExternalStore(subscribeFn, getSnapshot, getSnapshot);
};
