import { useCallback, useEffect, useRef, useState } from "react";

export type UseDelayedValueOptions = {
  delayBefore?: number;
  delayAfter?: number;
};

export type DelayedValueUpdate<T> = (
  value: T,
  direction: "before" | "after",
  overrideDelay?: number
) => void;

export type UseDelayedValueResult<T> = {
  value: T;
  update: DelayedValueUpdate<T>;
  cancel: () => void;
};

/**
 * React port of `src/composables/useDelayedValue.ts`.
 *
 * Schedules transitions to a value after a configurable delay; commits
 * immediately when delay is 0. The "before"/"after" direction lets the
 * caller use a longer open delay (e.g. 1000ms before showing a hover
 * panel) and a shorter close delay (e.g. 350ms before hiding it).
 *
 * The timer lives in a ref so it survives re-renders without resetting,
 * and is unconditionally cleared on every reschedule + on unmount —
 * so rapid update bursts can never leave a stale callback to fire
 * after the component has gone away.
 */
export function useDelayedValue<T>(
  initialValue: T,
  options: UseDelayedValueOptions = {}
): UseDelayedValueResult<T> {
  const { delayBefore = 0, delayAfter = 0 } = options;
  const [value, setValue] = useState<T>(initialValue);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancel = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const update = useCallback<DelayedValueUpdate<T>>(
    (next, direction, overrideDelay) => {
      const delay =
        overrideDelay ?? (direction === "before" ? delayBefore : delayAfter);
      cancel();
      if (delay > 0) {
        timerRef.current = setTimeout(() => {
          timerRef.current = null;
          setValue(next);
        }, delay);
      } else {
        setValue(next);
      }
    },
    [cancel, delayBefore, delayAfter]
  );

  useEffect(() => cancel, [cancel]);

  return { value, update, cancel };
}
