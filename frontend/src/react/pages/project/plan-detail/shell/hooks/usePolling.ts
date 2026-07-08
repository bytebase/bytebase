import { useCallback, useEffect, useRef } from "react";
import { minmax } from "@/utils";

// Poll cadence: start at `min`, grow ×`growth` up to `max` while quiescent, with
// ±`jitter` ms of randomization to keep many open dashboards from resynchronizing.
const POLLER_INTERVAL = {
  min: 1000,
  max: 30000,
  growth: 2,
  jitter: 250,
} as const;

export interface UsePollingParams {
  enabled: boolean;
  refreshState: () => Promise<void>;
  // While true, hold the poll at the minimum (jittered) interval instead of
  // backing off — foreground freshness during transient work (a task moving
  // PENDING -> RUNNING -> DONE). Exponential backoff is for relieving a
  // struggling server after failures, not for keeping a live dashboard fresh;
  // its growing interval is exactly the dead zone that delays observing a
  // status change. Back off only once everything is quiescent.
  fast: boolean;
}

export interface UsePollingResult {
  // Reset the backoff and poll again from the minimum interval. Call this
  // after a user-triggered mutation (run/skip/cancel a task, etc.) so the UI
  // watches closely for the resulting status transition instead of waiting out
  // the current (possibly maxed-out) backoff interval.
  restart: () => void;
}

export function usePolling({
  enabled,
  refreshState,
  fast,
}: UsePollingParams): UsePollingResult {
  const pollTimerRef = useRef<number | undefined>(undefined);
  // `active` mirrors the effect's mounted+enabled lifetime so a late-resolving
  // refreshState (or a `restart` racing an unmount) cannot schedule a new poll.
  const activeRef = useRef(false);
  // Bumped by every scheduleNext. A tick whose refresh was in flight when a
  // restart() happened sees a newer epoch afterwards and must not reschedule —
  // otherwise its grown backoff would clobber the restart's minimum interval.
  const epochRef = useRef(0);
  const refreshStateRef = useRef(refreshState);
  refreshStateRef.current = refreshState;
  // Read the latest `fast` when scheduling the next tick, so a transition that
  // introduces (or clears) active work takes effect on the following tick.
  const fastRef = useRef(fast);
  fastRef.current = fast;

  const stopPolling = useCallback(() => {
    if (!pollTimerRef.current) {
      return;
    }
    window.clearTimeout(pollTimerRef.current);
    pollTimerRef.current = undefined;
  }, []);

  const scheduleNext = useCallback(
    (interval: number) => {
      stopPolling();
      epochRef.current += 1;
      const epoch = epochRef.current;
      const nextInterval = minmax(
        interval +
          Math.floor(Math.random() * (POLLER_INTERVAL.jitter * 2 + 1)) -
          POLLER_INTERVAL.jitter,
        POLLER_INTERVAL.min,
        POLLER_INTERVAL.max
      );

      pollTimerRef.current = window.setTimeout(async () => {
        if (!activeRef.current) {
          return;
        }
        await refreshStateRef.current().catch(() => undefined);
        if (!activeRef.current) {
          return;
        }
        if (epoch !== epochRef.current) {
          // A restart() landed while this tick's refresh was in flight and has
          // already scheduled the next poll at the minimum interval.
          return;
        }
        scheduleNext(
          // Active work: stay at the floor. Quiescent: back off toward the cap.
          fastRef.current
            ? POLLER_INTERVAL.min
            : Math.min(
                nextInterval * POLLER_INTERVAL.growth,
                POLLER_INTERVAL.max
              )
        );
      }, nextInterval);
    },
    [stopPolling]
  );

  const restart = useCallback(() => {
    if (!activeRef.current) {
      return;
    }
    scheduleNext(POLLER_INTERVAL.min);
  }, [scheduleNext]);

  useEffect(() => {
    if (!enabled) {
      activeRef.current = false;
      stopPolling();
      return;
    }

    activeRef.current = true;
    scheduleNext(POLLER_INTERVAL.min);

    return () => {
      activeRef.current = false;
      stopPolling();
    };
  }, [enabled, scheduleNext, stopPolling]);

  return { restart };
}
