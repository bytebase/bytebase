import { useIntervalFn } from "@vueuse/core";
import { ref } from "vue";

export const useCurrentTimestamp = () => {
  // Update every 1 second instead of every frame
  const currentTsInMS = ref(Date.now());
  const { pause, resume } = useIntervalFn(() => {
    currentTsInMS.value = Date.now();
  }, 1000);

  return {
    currentTsInMS,
    pause,
    resume,
  };
};
