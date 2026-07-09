import { useEffect } from "react";
import { useLatestRef } from "./useLatestRef";

/**
 * Polls `fn` while `enabled`, waiting `intervalMs` between the end of one call
 * and the start of the next — so a call slower than the interval can't overlap
 * the next tick — and only while the tab is visible (a background tab does no
 * work). `fn` may be async; a rejected tick is swallowed so it never surfaces as
 * an unhandled rejection or stops the loop. The loop stops on unmount or when
 * `enabled` goes false, and reschedules when `intervalMs` changes.
 *
 * `fn` is read through a ref, so passing a fresh closure each render does not
 * restart the loop — only `enabled`/`intervalMs` changes do. To poll again right
 * away after a user action, do an immediate fetch in the action handler; the
 * next tick follows `intervalMs` from there.
 */
export function usePolling(
  enabled: boolean,
  intervalMs: number,
  fn: () => void | Promise<void>
): void {
  const fnRef = useLatestRef(fn);
  useEffect(() => {
    if (!enabled) {
      return;
    }
    let canceled = false;
    let timer: number | undefined;
    const tick = async () => {
      if (!document.hidden) {
        try {
          await fnRef.current();
        } catch {
          // A rejected tick is swallowed — it must not surface as an unhandled
          // rejection or break the loop.
        }
      }
      if (!canceled) {
        timer = window.setTimeout(tick, intervalMs);
      }
    };
    timer = window.setTimeout(tick, intervalMs);
    return () => {
      canceled = true;
      if (timer !== undefined) {
        window.clearTimeout(timer);
      }
    };
  }, [enabled, intervalMs, fnRef]);
}
