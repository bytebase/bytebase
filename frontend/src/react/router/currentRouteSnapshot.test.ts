import { describe, expect, it } from "vitest";
import { router } from "./index";

// Guardrail for the "getSnapshot should be cached" infinite-loop class.
// `router.currentRoute.value` backs the `useVueRoute` `useSyncExternalStore`
// snapshot. If it returns a fresh object on every read, any external-store
// consumer re-renders forever (this crashed the SQL Editor: GutterBar →
// useVueRoute → "Maximum update depth exceeded"). The snapshot is memoized in
// `currentRouteSnapshot` to keep a stable reference between actual route
// changes; this test fails if that memoization regresses.
describe("router.currentRoute snapshot stability", () => {
  it("returns a referentially stable object across reads without navigation", () => {
    const a = router.currentRoute.value;
    const b = router.currentRoute.value;
    expect(a).toBe(b);
  });
});
