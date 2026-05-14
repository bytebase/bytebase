import { useCallback, useEffect, useRef } from "react";
import { minmax } from "@/utils";
import { POLLER_INTERVAL } from "../constants";

export interface UsePollingParams {
  enabled: boolean;
  refreshState: () => Promise<void>;
}

export function usePolling({ enabled, refreshState }: UsePollingParams): void {
  const pollTimerRef = useRef<number | undefined>(undefined);

  const stopPolling = useCallback(() => {
    if (!pollTimerRef.current) {
      return;
    }
    window.clearTimeout(pollTimerRef.current);
    pollTimerRef.current = undefined;
  }, []);

  useEffect(() => {
    if (!enabled) {
      stopPolling();
      return;
    }

    let canceled = false;

    const poll = (interval: number) => {
      stopPolling();
      const nextInterval = minmax(
        interval +
          Math.floor(Math.random() * (POLLER_INTERVAL.jitter * 2 + 1)) -
          POLLER_INTERVAL.jitter,
        POLLER_INTERVAL.min,
        POLLER_INTERVAL.max
      );

      pollTimerRef.current = window.setTimeout(async () => {
        if (canceled) {
          return;
        }
        await refreshState().catch(() => undefined);
        if (canceled) {
          return;
        }
        poll(
          Math.min(nextInterval * POLLER_INTERVAL.growth, POLLER_INTERVAL.max)
        );
      }, nextInterval);
    };

    poll(POLLER_INTERVAL.min);

    return () => {
      canceled = true;
      stopPolling();
    };
  }, [enabled, refreshState, stopPolling]);
}
