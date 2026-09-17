import { useEffect, useReducer } from "react";
import type { TimeReading } from "@/utils/datetime";

/**
 * A single clock shared by every time-varying display.
 *
 * Each subscriber declares the instant its rendering next changes, so the clock
 * holds one timeout, set for the earliest of those, and wakes only the
 * subscribers that are due. A woken subscriber normally re-renders and declares
 * its next instant; one that names the same instant again -- a boundary held
 * stale, say -- is re-checked after a resync interval rather than on every
 * neighbour's wake.
 */
type Subscriber = {
  changesAtMs: number;
  wake: () => void;
};

const subscribers = new Set<Subscriber>();
let timer: ReturnType<typeof setTimeout> | undefined;
let armedForMs = Number.POSITIVE_INFINITY;
let armedAtMs = Number.NEGATIVE_INFINITY;
let lastWakeMs = Number.NEGATIVE_INFINITY;

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
  const nowMs = Date.now();
  armedForMs = changesAtMs;
  armedAtMs = nowMs;
  // Bounded above too, so a clock stepped backward cannot stretch the gap.
  const gapMs = Math.min(
    Math.max(lastWakeMs + MIN_WAKE_GAP_MS - nowMs, 0),
    MIN_WAKE_GAP_MS
  );
  const delayMs = Math.min(Math.max(changesAtMs - nowMs, gapMs), RESYNC_MS);
  timer = setTimeout(tick, delayMs);
}

function tick(): void {
  const nowMs = Date.now();
  // A timer cannot fire before it was armed, so a wall clock reading earlier
  // than the arm means the clock stepped backward, and every boundary computed
  // on the later clock may now be wrong in either direction.
  const clockSteppedBack = nowMs < armedAtMs;
  let nextMs = Number.POSITIVE_INFINITY;
  for (const subscriber of subscribers) {
    if (clockSteppedBack || subscriber.changesAtMs <= nowMs) {
      subscriber.changesAtMs = nowMs + RESYNC_MS;
      subscriber.wake();
      lastWakeMs = nowMs;
    }
    nextMs = Math.min(nextMs, subscriber.changesAtMs);
  }
  arm(nextMs);
}

function useNow(changesAtMs: number | undefined): void {
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

/**
 * The current value of a time-varying reading, kept current: the caller
 * re-renders when the value next changes. An absent input has no value and
 * holds no subscription.
 *
 * This is the only way a display gets such a value, so the value always comes
 * with its own boundary and a subscription. The boundary is evaluated first:
 * evaluated after the value, it can see a change the value just missed, name
 * the change after that, and leave the stale value on screen.
 */
export function useTimeReading<Input, Value>(
  reading: TimeReading<Input, Value>,
  input: Input
): Value;
export function useTimeReading<Input, Value>(
  reading: TimeReading<Input, Value>,
  input: Input | undefined
): Value | undefined;
export function useTimeReading<Input, Value>(
  reading: TimeReading<Input, Value>,
  input: Input | undefined
): Value | undefined {
  useNow(input === undefined ? undefined : reading.nextChangeAt(input));
  return input === undefined ? undefined : reading.read(input);
}
