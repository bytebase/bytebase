import { useCallback, useRef, useSyncExternalStore } from "react";
import { watch } from "vue";

/**
 * Subscribe a React component to a Vue reactive getter.
 * Re-renders whenever the getter's tracked dependencies change,
 * AND whenever the getter's closure variables (e.g. props) change.
 *
 * @param getter — A function that reads Vue reactive state (Pinia store, ref, computed, etc.)
 * @returns The current value of the getter, kept in sync with Vue reactivity.
 *
 * @example
 * const externalUrl = useVueState(() => useActuatorV1Store().serverInfo?.externalUrl ?? "");
 */
export function useVueState<T>(getter: () => T): T {
  // Cache the latest snapshot so getSnapshot returns a stable reference
  // between renders when the value hasn't changed.
  const snapshotRef = useRef<T>(getter());

  // Always point to the latest getter so the Vue watch evaluates
  // up-to-date closure variables (props, local state, etc.).
  const getterRef = useRef(getter);
  getterRef.current = getter;

  const subscribe = useCallback((onStoreChange: () => void) => {
    const stop = watch(
      () => getterRef.current(),
      (newVal) => {
        snapshotRef.current = newVal;
        onStoreChange();
      },
      { flush: "sync" }
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
