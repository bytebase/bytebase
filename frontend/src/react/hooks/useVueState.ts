import { useSyncExternalStore } from "react";
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
  // useSyncExternalStore requires a subscribe function and a getSnapshot function.
  // Vue's watch() provides the subscription; the getter IS the snapshot.
  const subscribe = (onStoreChange: () => void) => {
    const stop = watch(getter, onStoreChange, { flush: "sync" });
    return stop;
  };
  return useSyncExternalStore(subscribe, getter, getter);
}
