import { useEffect, useRef, useState } from "react";
import { watch } from "vue";

export interface UseVueStateOptions {
  /**
   * Force the watch to traverse nested reactive properties when
   * registering deps. Default `false`.
   *
   * Set this to `true` when the getter returns a reactive object/Map/
   * Set whose **nested fields** mutate in place (e.g. a collection
   * getter where consumers also read item fields). Without it, Vue's
   * watch only tracks the top-level reactive accesses the getter made
   * during evaluation.
   *
   * Note: this hook ALREADY surfaces in-place mutations of fields read
   * by the getter — those reads register Vue deps individually, and a
   * mutation to any of them triggers a re-render via the version
   * counter even when the getter's *return reference* is unchanged.
   * `deep: true` is only needed for collections whose contents the
   * getter doesn't read directly.
   */
  readonly deep?: boolean;
}

/**
 * Subscribe a React component to a Vue reactive getter.
 *
 * Re-renders whenever:
 *   1. Vue's `watch` fires (any reactive dep accessed by the getter
 *      changed — including in-place field mutations of objects the
 *      getter reads), OR
 *   2. The getter's closure variables (props, local state, etc.)
 *      change in a way that produces a different return value.
 *
 * Why a version counter rather than `useSyncExternalStore`'s snapshot
 * compare: Pinia stores typically mutate proxy objects in place via
 * `Object.assign(tab, payload)` or per-field writes. A getter that
 * returns the proxy itself (e.g. `() => tabStore.currentTab`) keeps
 * returning the same reference across mutations, and a snapshot
 * comparison via `Object.is` would skip the re-render even though Vue
 * detected a change. The version counter forces React to commit a new
 * render whenever Vue's watch callback fires, regardless of the
 * snapshot reference.
 *
 * @example
 *   // Reads a primitive — version bump and value change agree.
 *   const url = useVueState(() => actuatorStore.serverInfo?.externalUrl ?? "");
 *
 * @example
 *   // Reads a proxy object that mutates in place — the version bump
 *   // is what makes consumers see the new field values.
 *   const tab = useVueState(() => tabStore.currentTab);
 *   const isEmpty = !tab || tab.statement === "";
 *
 * @example
 *   // Deep-tracks nested collections the getter doesn't read directly.
 *   const queryList = useVueState(
 *     () => webTerminalStore.getQueryStateByTab(tab).queryItemList.value,
 *     { deep: true }
 *   );
 */
export function useVueState<T>(
  getter: () => T,
  options?: UseVueStateOptions
): T {
  // Always read the latest getter from the closure (props, local state,
  // etc. may have updated since mount).
  const getterRef = useRef(getter);
  getterRef.current = getter;

  // Pulling the deep flag through a ref keeps the watch wiring stable
  // across re-renders. Flipping it post-mount isn't supported — that
  // would require resubscribing.
  const deepRef = useRef(!!options?.deep);

  // Version bumps from the watch callback; React's `useState` setter
  // triggers a re-render even when the underlying Pinia object's
  // reference is unchanged (in-place mutation case).
  const [, setVersion] = useState(0);

  useEffect(() => {
    const stop = watch(
      () => getterRef.current(),
      () => setVersion((v) => v + 1),
      { flush: "sync", deep: deepRef.current }
    );
    return () => {
      stop();
    };
  }, []);

  // Read the live value at render time. The watch above guarantees a
  // re-render fires after every Vue mutation, so this read sees the
  // freshly-mutated state on the next render.
  return getter();
}
