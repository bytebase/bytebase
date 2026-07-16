import { useCallback, useEffect, useRef } from "react";
import { useLatestRef } from "@/react/hooks/useLatestRef";

const IDLE_MIN_INTERVAL_MS = 1000;
const IDLE_MAX_INTERVAL_MS = 16000;
const ACTIVE_MIN_INTERVAL_MS = 500;
const ACTIVE_MAX_INTERVAL_MS = 4000;
const GROWTH_FACTOR = 2;
const JITTER_MS = 250;

export type PlanPollingMode = "active" | "idle";

const minimumInterval = (mode: PlanPollingMode): number =>
  mode === "active" ? ACTIVE_MIN_INTERVAL_MS : IDLE_MIN_INTERVAL_MS;

const nextInterval = (intervalMs: number, mode: PlanPollingMode): number => {
  if (mode === "active") {
    return Math.min(intervalMs * GROWTH_FACTOR, ACTIVE_MAX_INTERVAL_MS);
  }
  return Math.min(
    Math.max(intervalMs * GROWTH_FACTOR, IDLE_MIN_INTERVAL_MS),
    IDLE_MAX_INTERVAL_MS
  );
};

const addJitter = (intervalMs: number): number => {
  // Keep the existing +/-250ms bound, but scale it down for the new 500ms
  // active floor so jitter never consumes half of that latency budget.
  const jitterMs = Math.min(JITTER_MS, intervalMs * 0.25);
  return intervalMs + Math.floor(Math.random() * (jitterMs * 2 + 1)) - jitterMs;
};

type PollingControl = {
  restart: () => void;
  setMode: (mode: PlanPollingMode) => void;
};

const NOOP_CONTROL: PollingControl = {
  restart: () => {},
  setMode: () => {},
};

/**
 * Polls the full plan snapshot without overlapping requests. The idle cadence
 * backs off from 1s to 16s; active mode backs off from 500ms to 4s. Every
 * scheduled wait gets independent jitter so clients do not synchronize, while
 * the jittered delay never feeds back into the next base interval.
 */
export const usePlanPolling = ({
  enabled,
  mode,
  refreshState,
  resetKey,
}: {
  enabled: boolean;
  mode: PlanPollingMode;
  refreshState: () => void | Promise<void>;
  resetKey?: string;
}): { restart: () => void } => {
  const refreshStateRef = useLatestRef(refreshState);
  const controlRef = useRef<PollingControl>(NOOP_CONTROL);
  const previousResetKeyRef = useRef(resetKey);

  useEffect(() => {
    if (!enabled) {
      controlRef.current = NOOP_CONTROL;
      return;
    }

    let canceled = false;
    let timer: number | undefined;
    let inFlight = false;
    let restartAfterFlight = false;
    let pollingMode = mode;
    let lastCompletedBaseIntervalMs = 0;

    const clearTimer = () => {
      if (timer !== undefined) {
        window.clearTimeout(timer);
        timer = undefined;
      }
    };

    const schedule = (baseIntervalMs: number) => {
      clearTimer();
      timer = window.setTimeout(async () => {
        timer = undefined;
        if (canceled || document.hidden) {
          return;
        }

        inFlight = true;
        try {
          await refreshStateRef.current();
        } catch {
          // A transient refresh failure must not stop the polling loop.
        } finally {
          inFlight = false;
          if (canceled) {
            return;
          }

          lastCompletedBaseIntervalMs = baseIntervalMs;
          if (restartAfterFlight) {
            restartAfterFlight = false;
            lastCompletedBaseIntervalMs = 0;
            schedule(minimumInterval(pollingMode));
            return;
          }

          schedule(nextInterval(baseIntervalMs, pollingMode));
        }
      }, addJitter(baseIntervalMs));
    };

    const restart = () => {
      lastCompletedBaseIntervalMs = 0;
      if (inFlight) {
        restartAfterFlight = true;
        return;
      }
      schedule(minimumInterval(pollingMode));
    };

    const setMode = (nextMode: PlanPollingMode) => {
      if (pollingMode === nextMode) {
        return;
      }
      pollingMode = nextMode;
      if (nextMode === "active") {
        restart();
        return;
      }
      if (!inFlight) {
        schedule(nextInterval(lastCompletedBaseIntervalMs, nextMode));
      }
    };

    const control = { restart, setMode };
    controlRef.current = control;

    const handleVisibilityChange = () => {
      if (!document.hidden) {
        restart();
      }
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("online", restart);
    schedule(minimumInterval(pollingMode));

    return () => {
      canceled = true;
      clearTimer();
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("online", restart);
      if (controlRef.current === control) {
        controlRef.current = NOOP_CONTROL;
      }
    };
  }, [enabled, refreshStateRef]);

  useEffect(() => {
    controlRef.current.setMode(mode);
  }, [mode]);

  useEffect(() => {
    if (previousResetKeyRef.current === resetKey) {
      return;
    }
    previousResetKeyRef.current = resetKey;
    controlRef.current.restart();
  }, [resetKey]);

  const restart = useCallback(() => {
    controlRef.current.restart();
  }, []);

  return { restart };
};
