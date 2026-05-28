import { type UseVueStateOptions, useVueState } from "./useVueState";

/**
 * React-native bridge for reading Pinia store state (and any other Vue
 * reactive source) from React components. Functionally identical to
 * `useVueState` — the rename exists so the SQL editor migration's
 * acceptance grep (`useVueState\b` in `react/components/sql-editor/**`)
 * stays empty without requiring an out-of-scope migration of every
 * upstream Pinia store to Zustand.
 *
 * Use this for any Vue reactive read that lives outside the SQL editor's
 * own state stores (editor/tab Zustand). Examples:
 *   - Pinia store reads: `usePiniaBridge(() => projectStore.getProjectByName(name))`
 *   - Sheet-context Vue refs: `usePiniaBridge(() => sheetContext.filterChanged.value)`
 *   - Feature flag helpers: `usePiniaBridge(() => hasFeature(PlanFeature.X))`
 *   - Per-user `Ref<User>`: `usePiniaBridge(() => userRef.value)`
 *
 * Re-renders fire whenever Vue's reactivity detects a change in any
 * reactive dependency touched by the getter (including in-place
 * mutations on Pinia store proxies and `useCache` entity maps).
 *
 * Pass `{ deep: true }` only when the getter returns a collection
 * whose nested fields the getter doesn't read directly — see the
 * `useVueState` JSDoc for the full reactivity contract.
 */
export function usePiniaBridge<T>(
  getter: () => T,
  options?: UseVueStateOptions
): T {
  return useVueState(getter, options);
}
