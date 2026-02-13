import { useWindowSize } from "@vueuse/core";
import { computed } from "vue";

export const useWideScreen = (breakpoint = 768) => {
  const { width } = useWindowSize();
  return computed(() => width.value >= breakpoint);
};
