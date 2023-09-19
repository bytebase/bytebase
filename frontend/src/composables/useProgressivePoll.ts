import { random } from "lodash-es";
import { onBeforeUnmount, reactive, unref } from "vue";
import { MaybeRef } from "@/types";
import { minmax } from "@/utils";

type ProgressivePollOptions = {
  interval: {
    min: number;
    max: number;
    growth: number;
    jitter: number;
  };
};

type ProgressivePollState = {
  timer: ReturnType<typeof setTimeout> | undefined;
};

// Poll with time interval increasing progressively
// e.g., 1s, 2s, 4s, 8s, 16s, 30s, 30s, 30s, 30s...
export const useProgressivePoll = (
  tick: () => void,
  options: MaybeRef<ProgressivePollOptions>
) => {
  const state = reactive<ProgressivePollState>({
    timer: undefined,
  });

  const stop = () => {
    if (!state.timer) return;
    clearTimeout(state.timer);
    state.timer = undefined;
  };

  const poll = (interval: number) => {
    stop();

    const { min, max, growth, jitter } = unref(options).interval;
    const int = minmax(interval + random(-jitter, jitter), min, max);

    state.timer = setTimeout(() => {
      tick();

      const next = Math.min(int * growth, max);
      poll(next);
    }, int);
  };

  const start = () => {
    poll(unref(options).interval.min);
  };

  onBeforeUnmount(stop);

  return { stop, start, restart: start };
};
