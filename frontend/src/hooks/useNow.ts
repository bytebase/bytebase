import { useEffect, useState } from "react";

/**
 * A single clock shared by every time-varying display, replacing the
 * per-component intervals that used to tick independently.
 *
 * Each subscriber declares the instant at which its own rendering could next
 * differ, so the clock holds one timeout — set to the earliest of those — and
 * wakes only the subscribers whose instant has arrived. A display that never
 * changes with time subscribes to nothing and costs nothing. This is the
 * scheduling behavior of GitHub's `relative-time` element.
 */
type Subscriber = {
  changesAtMs: number;
  notify: () => void;
};

const subscribers = new Set<Subscriber>();
let timer: ReturnType<typeof setTimeout> | undefined;

// A boundary computed as already past would otherwise spin the event loop.
const MIN_DELAY_MS = 16;

function reschedule(): void {
  if (timer !== undefined) {
    clearTimeout(timer);
    timer = undefined;
  }
  let earliest = Number.POSITIVE_INFINITY;
  for (const subscriber of subscribers) {
    earliest = Math.min(earliest, subscriber.changesAtMs);
  }
  if (!Number.isFinite(earliest)) {
    return;
  }
  timer = setTimeout(
    () => {
      timer = undefined;
      const nowMs = Date.now();
      for (const subscriber of [...subscribers]) {
        if (subscriber.changesAtMs <= nowMs) {
          subscriber.notify();
        }
      }
      reschedule();
    },
    Math.max(earliest - Date.now(), MIN_DELAY_MS)
  );
}

/**
 * Re-renders the caller once `changesAtMs` arrives, and returns the current
 * time. Pass `undefined` for a display that does not vary with time — it then
 * holds no subscription.
 */
export function useNow(changesAtMs: number | undefined): number {
  const [nowMs, setNowMs] = useState(() => Date.now());

  useEffect(() => {
    if (changesAtMs === undefined || !Number.isFinite(changesAtMs)) {
      return;
    }
    const subscriber: Subscriber = {
      changesAtMs,
      notify: () => setNowMs(Date.now()),
    };
    subscribers.add(subscriber);
    reschedule();
    return () => {
      subscribers.delete(subscriber);
      reschedule();
    };
  }, [changesAtMs]);

  return nowMs;
}
