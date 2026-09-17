import { useEffect, useReducer } from "react";

/**
 * A single clock shared by every time-varying display.
 *
 * Each subscriber declares the instant its rendering next changes, so the clock
 * holds one timeout, set for the earliest of those, and wakes only the
 * subscribers that are due. A woken subscriber re-renders and must declare a
 * strictly later instant; one that does not is left out of the next deadline,
 * so a boundary that fails to advance freezes its label rather than spinning
 * the clock. A display that never changes with time subscribes to nothing.
 */
type Subscriber = {
  changesAtMs: number;
  wake: () => void;
};

const subscribers = new Set<Subscriber>();
let timer: ReturnType<typeof setTimeout> | undefined;
let armedForMs = Number.POSITIVE_INFINITY;

// setTimeout keeps its delay in a signed 32-bit integer and fires at once when
// asked for more, so a far deadline is reached in steps of at most this.
const MAX_TIMER_DELAY_MS = 2 ** 31 - 1;

function disarm(): void {
  if (timer !== undefined) {
    clearTimeout(timer);
    timer = undefined;
  }
  armedForMs = Number.POSITIVE_INFINITY;
}

function arm(changesAtMs: number): void {
  disarm();
  if (!Number.isFinite(changesAtMs)) {
    return;
  }
  armedForMs = changesAtMs;
  const delayMs = Math.min(
    Math.max(changesAtMs - Date.now(), 0),
    MAX_TIMER_DELAY_MS
  );
  timer = setTimeout(tick, delayMs);
}

function tick(): void {
  timer = undefined;
  armedForMs = Number.POSITIVE_INFINITY;
  const nowMs = Date.now();
  // Woken subscribers still hold the instant that just passed until they
  // re-render, so the next deadline comes only from those still waiting; each
  // woken one re-arms the clock as it declares its next instant.
  let nextMs = Number.POSITIVE_INFINITY;
  for (const subscriber of subscribers) {
    if (subscriber.changesAtMs <= nowMs) {
      subscriber.wake();
    } else {
      nextMs = Math.min(nextMs, subscriber.changesAtMs);
    }
  }
  arm(nextMs);
}

/**
 * Re-renders the caller once `changesAtMs` arrives. Pass `undefined` for a
 * display that does not vary with time; it then holds no subscription.
 */
export function useNow(changesAtMs: number | undefined): void {
  // A counter rather than the time: setting a timestamp can bail out when two
  // wakes land in the same millisecond, and a missed render is a frozen label.
  const [, wake] = useReducer((count: number) => count + 1, 0);

  useEffect(() => {
    if (changesAtMs === undefined || !Number.isFinite(changesAtMs)) {
      return;
    }
    const subscriber: Subscriber = { changesAtMs, wake };
    subscribers.add(subscriber);
    if (changesAtMs < armedForMs) {
      arm(changesAtMs);
    }
    return () => {
      subscribers.delete(subscriber);
      // Leaving the timer armed for a departed subscriber costs at most one
      // early wake, which re-arms from those remaining; only an empty clock
      // needs no timer at all.
      if (subscribers.size === 0) {
        disarm();
      }
    };
  }, [changesAtMs]);
}
