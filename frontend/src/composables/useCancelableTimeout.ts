import { useTimestamp } from "@vueuse/core";
import { computed, ref, unref } from "vue";
import { MaybeRef } from "@/types";

export const useCancelableTimeout = (timeoutMS: MaybeRef<number>) => {
  const running = ref(false);
  const startTS = ref(0);
  const nowTS = useTimestamp();

  const elapsedMS = computed(() => {
    if (!running.value) return 0;
    return nowTS.value - startTS.value;
  });

  const expired = computed(() => {
    if (!running.value) return false;
    return elapsedMS.value > unref(timeoutMS);
  });

  const start = () => {
    startTS.value = Date.now();
    running.value = true;
  };

  const stop = () => {
    running.value = false;
  };

  return { start, stop, elapsedMS, expired };
};
