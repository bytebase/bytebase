import { useCallback, useRef, useSyncExternalStore } from "react";
import { watch } from "vue";

export interface UseVueStateOptions {
  /**
   * Track deep mutations of returned reactive objects.
   *
   * Default `false` (shallow) — Vue's `watch` only fires when properties
   * read by the getter change reference. Pinia stores typically mutate
   * fields in place via `Object.assign(tab, payload)`, so a getter like
   * `() => [...tabStore.openTabList]` tracks the array + each item
   * reference but NOT each item's nested fields. Set `deep: true` for
   * collection getters where consumers care about field-level changes
   * (e.g. tab connection / status updates that don't reseat the tab
   * reference).
   */
  readonly deep?: boolean;
}

/**
 * Subscribe a React component to a Vue reactive getter.
 * Re-renders whenever the getter's tracked dependencies change,
 * AND whenever the getter's closure variables (e.g. props) change.
 *
 * @param getter — A function that reads Vue reactive state (Pinia store, ref, computed, etc.)
 * @param options — Optional `{ deep }` flag, see `UseVueStateOptions`.
 * @returns The current value of the getter, kept in sync with Vue reactivity.
 *
 * @example
 * const externalUrl = useVueState(() => useActuatorV1Store().serverInfo?.externalUrl ?? "");
 */
export function useVueState<T>(
  getter: () => T,
  options?: UseVueStateOptions
): T {
  // Cache the latest snapshot so getSnapshot returns a stable reference
  // between renders when the value hasn't changed.
  const snapshotRef = useRef<T>(getter());

  // Always point to the latest getter so the Vue watch evaluates
  // up-to-date closure variables (props, local state, etc.).
  const getterRef = useRef(getter);
  getterRef.current = getter;

  // Capture `deep` once per mount; flipping it post-mount would require
  // tearing down + re-subscribing the watch and isn't a real use case.
  const deepRef = useRef(!!options?.deep);

  const subscribe = useCallback((onStoreChange: () => void) => {
    const stop = watch(
      () => getterRef.current(),
      (newVal) => {
        snapshotRef.current = newVal;
        onStoreChange();
      },
      { flush: "sync", deep: deepRef.current }
    );
    // Initialize with current value
    snapshotRef.current = getterRef.current();
    return stop;
  }, []);

  // Re-evaluate getter on every render to catch closure-driven changes
  // (e.g. a prop like environmentName changing from "prod" to "test").
  // The Vue watch only fires for Vue reactive dep changes — it cannot
  // detect plain JS closure variable changes, so we sync here.
  const currentValue = getter();
  if (!Object.is(snapshotRef.current, currentValue)) {
    snapshotRef.current = currentValue;
  }

  const getSnapshot = useCallback(() => snapshotRef.current, []);

  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
