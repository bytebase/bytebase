import { useWindowSize } from "@vueuse/core";
import { computed } from "vue";
import { TailwindBreakpoints } from "@/utils";

export const useWideScreen = (breakpoint: number = TailwindBreakpoints.md) => {
  const { width } = useWindowSize();
  return computed(() => width.value >= breakpoint);
};
