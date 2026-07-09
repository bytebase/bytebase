import { useEffect } from "react";
import { useLatestRef } from "./useLatestRef";

/**
 * Runs `fn` on a loop while `enabled`, waiting `intervalMs` between the end of
 * one call and the start of the next — and only while the tab is visible (a
 * background tab does no work). `fn` may be async; a rejected tick is swallowed
 * so it never surfaces as an unhandled rejection or stops the loop. The loop is
 * stopped on unmount or when `enabled` goes false.
 *
 * Rescheduling AFTER each call settles (rather than a fixed interval) is
 * deliberate: a call slower than `intervalMs` must not overlap the next tick.
 * Callers guard responses with a start-of-fetch sequence, so overlapping calls
 * would let every response be invalidated by the next tick and leave the view
 * stuck loading/stale under sustained latency.
 *
 * `fn` is read through a ref, so passing a fresh closure each render does not
 * restart the loop — only `enabled`/`intervalMs` changes do.
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
