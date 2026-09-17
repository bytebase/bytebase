import { useEffect, useReducer } from "react";

/**
 * A single clock shared by every time-varying display.
 *
 * Each subscriber declares the instant its rendering next changes, so the clock
 * holds one timeout, set for the earliest of those, and wakes only the
 * subscribers that are due. A woken subscriber is retired until it re-renders
 * and declares its next instant; one whose boundary fails to advance keeps its
 * last reading rather than being woken again.
 */
type Subscriber = {
  changesAtMs: number;
  wake: () => void;
};

const subscribers = new Set<Subscriber>();
let timer: ReturnType<typeof setTimeout> | undefined;
let armedForMs = Number.POSITIVE_INFINITY;
let lastTickMs = Number.NEGATIVE_INFINITY;

// Deadlines are wall-clock instants, but timers do not count time the machine
// sleeps and the wall clock can be adjusted, so the clock re-checks at least
// this often however far its earliest deadline is. It also keeps delays inside
// setTimeout's 32-bit range.
const RESYNC_MS = 60_000;
// A boundary that names a past instant on every render would otherwise re-arm
// at zero delay and spin.
const MIN_WAKE_GAP_MS = 250;

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
  const nowMs = Date.now();
  // Bounded above too, so a clock stepped backward cannot stretch the gap.
  const gapMs = Math.min(
    Math.max(lastTickMs + MIN_WAKE_GAP_MS - nowMs, 0),
    MIN_WAKE_GAP_MS
  );
  const delayMs = Math.min(Math.max(changesAtMs - nowMs, gapMs), RESYNC_MS);
  timer = setTimeout(tick, delayMs);
}

function tick(): void {
  timer = undefined;
  armedForMs = Number.POSITIVE_INFINITY;
  const nowMs = Date.now();
  lastTickMs = nowMs;
  let nextMs = Number.POSITIVE_INFINITY;
  for (const subscriber of subscribers) {
    if (subscriber.changesAtMs <= nowMs) {
      subscriber.changesAtMs = Number.POSITIVE_INFINITY;
      subscriber.wake();
    } else {
      nextMs = Math.min(nextMs, subscriber.changesAtMs);
    }
  }
  arm(nextMs);
}

/**
 * Re-renders the caller once `changesAtMs` arrives; `undefined` or `Infinity`
 * holds no subscription.
 *
 * Compute `changesAtMs` before the reading it schedules. Computed after, it can
 * see a change the reading just missed, name the change after that, and leave
 * the stale reading on screen.
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
      // early wake, which re-arms from those remaining.
      if (subscribers.size === 0) {
        disarm();
      }
    };
  }, [changesAtMs]);
}
