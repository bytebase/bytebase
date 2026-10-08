import { act } from "react";
import { vi } from "vitest";

/**
 * Advance fake timers a second at a time, committing between turns as a
 * browser does. A single long advance would commit once, at its end, and hide
 * whether a display re-rendered when it should have or only at the finish.
 */
export const advanceSeconds = (seconds: number) => {
  for (let second = 0; second < seconds; second++) {
    act(() => {
      vi.advanceTimersByTime(1_000);
    });
  }
};
