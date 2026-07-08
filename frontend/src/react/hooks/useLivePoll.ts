import { useEffect } from "react";
import { useLatestRef } from "./useLatestRef";

/**
 * Calls `fn` every `intervalMs` while `enabled`, and only while the tab is
 * visible (a background tab does no work). `fn` may be async; a rejected tick
 * is swallowed so it never surfaces as an unhandled rejection. The interval is
 * cleared on unmount or when `enabled` goes false.
 *
 * `fn` is read through a ref, so passing a fresh closure each render does not
 * restart the interval — only `enabled`/`intervalMs` changes do.
 */
export function useLivePoll(
  enabled: boolean,
  intervalMs: number,
  fn: () => void | Promise<void>
): void {
  const fnRef = useLatestRef(fn);
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const timer = window.setInterval(() => {
      if (document.hidden) {
        return;
      }
      void Promise.resolve(fnRef.current()).catch(() => undefined);
    }, intervalMs);
    return () => window.clearInterval(timer);
  }, [enabled, intervalMs, fnRef]);
}
