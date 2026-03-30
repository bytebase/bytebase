import { useCallback, useRef, useSyncExternalStore } from "react";
import { watch } from "vue";

/**
 * Subscribe a React component to a Vue reactive getter.
 * Re-renders whenever the getter's tracked dependencies change.
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

  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      const stop = watch(
        getter,
        (newVal) => {
          snapshotRef.current = newVal;
          onStoreChange();
        },
        { flush: "sync" }
      );
      // Initialize with current value
      snapshotRef.current = getter();
      return stop;
    },
    // getter is typically an inline arrow — we intentionally omit it from deps
    // to keep subscribe stable. The watch() inside tracks Vue deps automatically.
    []
  );

  const getSnapshot = useCallback(() => snapshotRef.current, []);

  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
